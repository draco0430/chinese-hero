package auth

import (
	"fmt"
	"sort"
	"strings"

	"hero-server/database"
	"hero-server/utils"
)

type ListCharactersHandler struct {
	username string
}

var (
	CHARACTER_LIST = utils.Packet{0xAA, 0x55, 0x00, 0x00, 0x01, 0x02, 0x0A, 0x00, 0x01, 0x01, 0x00, 0x55, 0xAA}
	NO_CHARACTERS  = utils.Packet{0xAA, 0x55, 0x07, 0x00, 0x01, 0x02, 0x0A, 0x00, 0x00, 0x00, 0x00, 0x55, 0xAA}
)

func (lch *ListCharactersHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {

	length := int(data[6])
	lch.username = string(data[7 : length+7])

	loginsMutex.RLock()
	val, have := logins[lch.username]
	loginsMutex.RUnlock()
	if !have {
		fmt.Println("Böyle bir username hiç giriş yapılmadı IP: ", s.Conn.RemoteAddr().String(), " user: ", lch.username)
		s.OnClose()
		return nil, nil
	}

	parsedIP := strings.Split(s.Conn.RemoteAddr().String(), ":")
	if parsedIP[0] != val {
		fmt.Println("Bu IP bu usera hiç girmemiş IP: ", s.Conn.RemoteAddr().String(), " user: ", lch.username)
		s.OnClose()
		return nil, nil
	}

	return lch.listCharacters(s)
}

func (lch *ListCharactersHandler) listCharacters(s *database.Socket) ([]byte, error) {
	user := findUser(lch.username)
	if user == nil /*|| user.ConnectingIP == ""*/ {
		return nil, nil
	}

	s.User = user
	s.ClientAddr = s.User.ConnectingIP
	s.User.ConnectedIP = s.ClientAddr
	s.User.ConnectedServer = s.User.ConnectingTo
	s.User.ConnectingTo = 0
	s.User.ConnectingIP = ""
	s.Add(s.User.ID)
	go s.User.Update()
	return lch.showCharacterMenu(s)
}

func (lch *ListCharactersHandler) showCharacterMenu(s *database.Socket) ([]byte, error) {
	characters, err := database.FindCharactersByUserID(s.User.ID)
	if err != nil {
		return nil, err
	}

	if len(characters) == 0 {
		return NO_CHARACTERS, nil
	}

	sort.Slice(characters, func(i, j int) bool {
		if characters[i].CreatedAt.Time.Sub(characters[j].CreatedAt.Time) < 0 {
			return true
		}
		return false
	})

	resp := CHARACTER_LIST

	length := 7
	resp[9] = byte(characters[0].Faction)
	resp[10] = byte(len(characters))

	index := 11
	for i, c := range characters {
		length += len(c.Name) + 269
		resp.Insert([]byte{byte(i)}, index) // character index
		index += 1

		id := uint64(c.ID)
		resp.Insert(utils.IntToBytes(id, 4, true), index) // character id
		index += 4

		resp.Insert([]byte{byte(len(c.Name))}, index) // character name length
		index += 1

		resp.Insert([]byte(c.Name), index) // character name
		index += len(c.Name)

		resp.Insert([]byte{byte(c.Type), byte(c.Class)}, index) // character type-class
		index += 2

		resp.Insert(utils.IntToBytes(uint64(c.Level), 4, true), index) //character level
		index += 4

		resp.Insert([]byte{0x3E}, index)
		index += 1

		resp.Insert([]byte{byte(c.WeaponSlot)}, index) // character weapon slot
		index += 1

		resp.Insert([]byte{0x00, 0x00, 0x4B, 0xFF, 0xE6, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}, index)
		index += 13

		slots := c.GetAppearingItemSlots()
		inventory, err := c.InventorySlots()
		if err != nil {
			return nil, err
		}

		for i, s := range slots {
			slot := inventory[s]
			itemID := slot.ItemID
			if slot.Appearance != 0 {
				itemID = slot.Appearance
			} else if slot.Appearance == 0 && itemID == 0 {

			}

			resp.Insert(utils.IntToBytes(uint64(itemID), 4, true), index) // item id
			index += 4

			resp.Insert([]byte{0x00, 0x00, 0x00}, index)
			index += 3

			resp.Insert(utils.IntToBytes(uint64(i), 2, true), index) // item slot
			index += 2

			resp.Insert([]byte{byte(slot.Plus), 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}, index) // item plus
			index += 13
		}
	}

	resp.SetLength(int16(length))
	return resp, nil
}

func findUser(username string) *database.User {

	all := database.AllUsers()
	for _, u := range all {
		if u.Username == username {
			return u
		}
	}

	return nil
}
