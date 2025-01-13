package auth

import (
	"fmt"

	"hero-server/config"
	"hero-server/database"
	"hero-server/logging"
	"hero-server/utils"
)

type SelectServerHandler struct {
	ip     string
	server int
}

var (
	SELECTED_SERVER = utils.Packet{0xAA, 0x55, 0x00, 0x00, 0x00, 0x05, 0x01, 0x00, 0x55, 0xAA}
)

func (ssh *SelectServerHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {

	ssh.ip = (*s).ClientAddr
	ssh.server = int(data[8]) + 1
	return ssh.selectServer(s)
}

func (ssh *SelectServerHandler) selectServer(s *database.Socket) ([]byte, error) {
	resp := SELECTED_SERVER

	serverIP := config.Default.Server.IP
	length := int16(len(serverIP) + 8)
	resp.SetLength(length)

	resp[7] = byte(len(serverIP)) // server ip length
	resp.Insert([]byte(serverIP), 8)

	index := len(serverIP) + 8
	port := config.Default.Server.Port
	resp.Insert(utils.IntToBytes(uint64(port), 4, true), index)

	logger.Log(logging.ACTION_SELECT_SERVER, 0, fmt.Sprintf("Server selected: %d", ssh.server), s.User.ID, "Server Select")

	s.User.ConnectingTo = ssh.server
	s.User.ConnectingIP = s.ClientAddr
	return resp, nil
}
