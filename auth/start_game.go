package auth

import (
	"encoding/binary"
	"log"
	"time"

	dbg "runtime/debug"

	"hero-server/database"
	"hero-server/logging"
	"hero-server/npc"
	"hero-server/player"
	"hero-server/utils"

	"github.com/thoas/go-funk"
)

type StartGameHandler struct {
}

var (
	GAME_STARTED = utils.Packet{0xAA, 0x55, 0xE6, 0x00, 0x17, 0xE1, 0x00, 0xF3, 0x0C, 0x1F, 0xF1, 0x0C, 0x08, 0x12, 0x00, 0x00, 0x01,
		0x00, 0x0C, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0x00, 0x00, 0x01, 0x07, 0x01, 0x02, 0x03, 0x04, 0x00, 0x00, 0x00, 0x00, 0x00, 0x0D, 0x20, 0x01, 0x00,
		0x0C, 0x20, 0x01, 0x00, 0x08, 0x20, 0x01, 0xE0, 0x03, 0x00, 0x00, 0x04, 0xE0, 0x03, 0x0C, 0x60, 0x00, 0x00, 0x64, 0x60, 0x05, 0x06, 0x00, 0x00, 0x10,
		0x0E, 0x00, 0x00, 0x51, 0x20, 0x07, 0x00, 0xCA, 0x20, 0x03, 0x00, 0x24, 0x20, 0x03, 0x00, 0x48, 0x20, 0x03, 0x60, 0x00, 0x01, 0x03, 0x01, 0x20, 0x00,
		0x60, 0x09, 0x60, 0x00, 0x40, 0x74, 0xC0, 0x00, 0x03, 0x74, 0x3B, 0xA4, 0x0B, 0x40, 0x0B, 0x13, 0x01, 0x32, 0x30, 0x31, 0x38, 0x2D, 0x30, 0x34, 0x2D,
		0x33, 0x30, 0x20, 0x30, 0x39, 0x3A, 0x31, 0x37, 0x3A, 0x34, 0x34, 0x40, 0x17, 0xE0, 0x1D, 0x00, 0x09, 0x02, 0x01, 0x00, 0x01, 0x00, 0x01, 0x00, 0xA1,
		0x01, 0x00, 0x60, 0x4C, 0x03, 0x00, 0xC0, 0x75, 0x06, 0x60, 0x0D, 0x00, 0x0C, 0xE0, 0x1D, 0x3E, 0xE0, 0xFF, 0x00, 0xE0, 0xFF, 0x00, 0xE0, 0xFF, 0x00,
		0xE0, 0xFF, 0x00, 0xE0, 0xFF, 0x00, 0xE0, 0xFF, 0x00, 0xE0, 0xFF, 0x00, 0xE0, 0x55, 0x00, 0x00, 0x03, 0xE0, 0x55, 0x5E, 0xE0, 0xFF, 0x00, 0xE0, 0xFF,
		0x00, 0xE0, 0xFF, 0x00, 0xE0, 0xB8, 0x00, 0x01, 0x00, 0x00, 0x55, 0xAA}

	GAME_STARTED2 = utils.Packet{0xAA, 0x55, 0xDE, 0x01, 0x17, 0xD9, 0x01, 0x9C, 0x10, 0x1F, 0x9A, 0x10, 0x08, 0x30, 0x02, 0xFC, 0x0F, 0x02, 0x00, 0x0F, 0x54,
		0x69, 0x6D, 0x65, 0x32, 0x52, 0x65, 0x76, 0x6F, 0x6C, 0x75, 0x74, 0x69, 0x6F, 0x6E, 0x35, 0x02, 0x00, 0x00, 0xAE, 0xBA, 0xAA, 0x11, 0x43, 0x93, 0xA3,
		0x52, 0x43, 0x00, 0x00, 0x80, 0x40, 0x05, 0x00, 0x00, 0x00, 0x10, 0x27, 0x00, 0x00, 0x2F, 0x20, 0x07, 0x00, 0x0C, 0xA0, 0x01, 0x00, 0x06, 0x20, 0x01,
		0xE0, 0x03, 0x00, 0x00, 0x14, 0x20, 0x0C, 0x00, 0x01, 0x20, 0x03, 0x03, 0x00, 0x00, 0xB2, 0x08, 0x20, 0x04, 0x20, 0x00, 0x01, 0x04, 0x10, 0x20, 0x04,
		0x20, 0x00, 0x05, 0x20, 0x1C, 0x00, 0x00, 0x3A, 0x02, 0x80, 0x03, 0x00, 0x9E, 0x20, 0x0F, 0x40, 0x03, 0x00, 0x90, 0x40, 0x2A, 0x01, 0x03, 0x01, 0x20,
		0x00, 0x20, 0x10, 0xA0, 0x00, 0x04, 0x02, 0x00, 0x05, 0x27, 0x0A, 0xA0, 0x0B, 0xE0, 0x3D, 0x00, 0x0A, 0x16, 0x00, 0x90, 0x5A, 0xF6, 0x05, 0x00, 0xA1,
		0x01, 0x00, 0x03, 0xE0, 0x1A, 0x50, 0x00, 0x91, 0xA0, 0x2B, 0x00, 0x04, 0xE0, 0x1A, 0x2B, 0x0A, 0xD8, 0x51, 0x9D, 0x00, 0x00, 0xA2, 0x01, 0x00, 0x0B,
		0x00, 0x42, 0xE0, 0x18, 0x2D, 0x08, 0xC1, 0x75, 0x06, 0x01, 0x03, 0xA1, 0xD9, 0x01, 0x0C, 0xE0, 0x18, 0x29, 0x02, 0x00, 0x00, 0xC2, 0x20, 0x2B, 0x04,
		0x00, 0xA1, 0x58, 0x02, 0x0D, 0x20, 0x0B, 0xE0, 0x17, 0x00, 0x08, 0xA9, 0x08, 0x0B, 0x01, 0x00, 0xA6, 0x01, 0x00, 0x0E, 0xE0, 0x17, 0x28, 0x20, 0x00,
		0x00, 0xC3, 0x40, 0x57, 0x20, 0xAF, 0x03, 0x0F, 0x00, 0xA0, 0x86, 0xE1, 0x02, 0x72, 0xE0, 0x0C, 0x00, 0x07, 0x41, 0xC2, 0xA1, 0x00, 0x00, 0xA1, 0x03,
		0x00, 0xA1, 0xBE, 0xE0, 0x14, 0x00, 0x02, 0xF9, 0x68, 0xC6, 0x20, 0x2B, 0x02, 0x01, 0x00, 0x11, 0xE0, 0x14, 0x25, 0x80, 0x00, 0x01, 0x62, 0xBE, 0x40,
		0x57, 0x02, 0x05, 0x00, 0x12, 0x80, 0x0E, 0xE0, 0x14, 0x00, 0x02, 0x98, 0x44, 0x9A, 0x60, 0x57, 0x00, 0x13, 0xE0, 0x14, 0x25, 0x80, 0x00, 0x02, 0x58,
		0x65, 0xB9, 0x60, 0x2B, 0x42, 0x81, 0xE0, 0x17, 0x00, 0x00, 0x96, 0xA0, 0x57, 0x00, 0x15, 0xE0, 0x17, 0x28, 0x20, 0x00, 0x00, 0xD5, 0x41, 0xE3, 0x20,
		0xDB, 0x00, 0x16, 0x20, 0x0B, 0xE0, 0x17, 0x00, 0x00, 0xD7, 0x40, 0x2B, 0x21, 0x5F, 0x03, 0x17, 0x00, 0x47, 0x51, 0xE0, 0x17, 0x2B, 0x02, 0x56, 0x6C,
		0xA3, 0x62, 0x3B, 0x02, 0x18, 0x00, 0x60, 0xE0, 0x17, 0x2A, 0x01, 0x00, 0x66, 0x61, 0x33, 0x02, 0x01, 0x00, 0x19, 0xE0, 0x18, 0x29, 0x04, 0x00, 0x00,
		0x35, 0xCB, 0x9B, 0x61, 0x07, 0x00, 0x1A, 0x20, 0x0B, 0xE0, 0x17, 0x00, 0x00, 0x64, 0x60, 0x57, 0x02, 0x02, 0x00, 0x1B, 0xE0, 0x17, 0x28, 0x20, 0x00,
		0x00, 0x95, 0xA1, 0x33, 0x23, 0xC6, 0xE0, 0x18, 0x00, 0x00, 0x56, 0xA1, 0x8B, 0x00, 0x1D, 0xE0, 0x18, 0x29, 0x04, 0x00, 0x00, 0x37, 0xF9, 0xBD, 0x60,
		0xAF, 0x00, 0x1E, 0x20, 0x0B, 0xE0, 0xFF, 0x00, 0xE0, 0xFF, 0x00, 0xE0, 0xFF, 0x00, 0xE0, 0xFF, 0x00, 0xE0, 0x16, 0x00, 0x00, 0x03, 0xE7, 0x1B, 0xDF,
		0xE0, 0xAB, 0x00, 0xE0, 0x00, 0xD7, 0xE0, 0x00, 0x08, 0xE0, 0xFF, 0x00, 0xE0, 0x17, 0x00, 0xE1, 0xFF, 0x30, 0xE0, 0xFF, 0x00, 0xE0, 0xFF, 0x00, 0xE0,
		0xFF, 0x00, 0xE0, 0xFF, 0x00, 0xE0, 0x8C, 0x00, 0x01, 0x00, 0x00, 0x55, 0xAA}

	CHARACTER_GONE = utils.Packet{0xAA, 0x55, 0x05, 0x00, 0x21, 0x02, 0x00, 0x55, 0xAA}

	MOB_DISAPPEARED = utils.Packet{0xAA, 0x55, 0x10, 0x00, 0x31, 0x02, 0x09, 0x00, 0x0A, 0x00, 0x55, 0xAA}

	NPC_APPEARED = utils.Packet{0xAA, 0x55, 0x5D, 0x00, 0x31, 0x01, 0x00, 0x00, 0x00, 0x00, 0x12, 0x47, 0x69, 0x6E, 0x73, 0x65,
		0x6E, 0x67, 0x20, 0x44, 0x69, 0x67, 0x67, 0x65, 0x72, 0x20, 0x44, 0x6F, 0x6E, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x00, 0x00, 0x00, 0x01, 0x01, 0x00, 0x00, 0xA0, 0x41, 0x00, 0x00, 0xA0, 0x41, 0x00, 0x00, 0x00, 0x00, 0xFF, 0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x64, 0x00, 0x55, 0xAA}

	AID_ITEM_HANDLE = utils.Packet{0xaa, 0x55, 0x6b, 0x00, 0xa3, 0x03, 0x01, 0x32, 0x30, 0x33, 0x30, 0x2d, 0x31, 0x32, 0x2d, 0x31, 0x38, 0x20, 0x31, 0x30, 0x3a, 0x33, 0x38, 0x3a, 0x30, 0x35, 0x00, 0x01, 0x32, 0x30, 0x34, 0x30, 0x2d, 0x31, 0x32, 0x2d, 0x31, 0x38, 0x20, 0x31, 0x30, 0x3a, 0x33, 0x38, 0x3a, 0x35, 0x30, 0x00, 0x01, 0x33, 0x30, 0x32, 0x30, 0x2d, 0x30, 0x36, 0x2d, 0x31, 0x38, 0x20, 0x31, 0x30, 0x3a, 0x33, 0x38, 0x3a, 0x35, 0x30, 0x00, 0x01, 0x33, 0x30, 0x32, 0x30, 0x2d, 0x30, 0x36, 0x2d, 0x31, 0x38, 0x20, 0x31, 0x30, 0x3a, 0x33, 0x38, 0x3a, 0x35, 0x30, 0x00, 0x01, 0x33, 0x30, 0x32, 0x30, 0x2d, 0x30, 0x36, 0x2d, 0x31, 0x38, 0x20, 0x31, 0x30, 0x3a, 0x33, 0x38, 0x3a, 0x35, 0x30, 0x00, 0x55, 0xaa}

	NPC_DISAPPEARED = MOB_DISAPPEARED
)

func (sgh *StartGameHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {
	if s.Character == nil { //
		return nil, nil
	}

	if s.Character.IsOnline {
		return nil, nil
	}

	return sgh.startGame(s)
}

func (csh *StartGameHandler) startGame(s *database.Socket) ([]byte, error) {

	if database.CheckCharacter(s.Character.ID, s.User.ID) {
		s.OnClose()
		return nil, nil
	}

	// API Security Check Start
	/*
		parsedIP := strings.Split(s.Conn.RemoteAddr().String(), ":")
		if parsedIP[0] == "" {
			s.Conn.Close()
			return nil, nil
		}

		if !security.CheckPlayer(parsedIP[0]) {
			s.Conn.Close()
			return nil, nil
		}
	*/
	// API Security Check Finish

	if s.Character != nil {
		s.Character.IsActive = false
	}
	if s.Stats != nil && s.Stats.HP <= 0 {
		s.Stats.HP = s.Stats.MaxHP / 10
	}
	// TODO
	//s.Stats.HPRecoveryRate = 0
	//s.Stats.CHIRecoveryRate = 0

	if s.Character.ID == 0 || s.Character.ID < 0 {
		s.OnClose()
		return nil, nil
	}

	if s.Character.Map == 230 || s.Character.Map == 255 || s.Character.Map == 74 {
		//gomap, _ := s.Character.ChangeMap(1, nil)
		//s.Conn.Write(gomap)

		s.Character.Map = 1
		s.Character.Coordinate = database.ConvertPointToCoordinate(324, 189)
	}

	if s.Character.Map == 76 {
		if s.Character.Faction != database.GoldenBasinArea.FactionID {
			s.Character.Map = 1
			s.Character.Coordinate = database.ConvertPointToCoordinate(324, 189)
		}
	}

	if s.Character.Map == 233 {
		if s.Character.Morphed {
			s.Character.HandleLimitedItems()
		}
		//gomap, _ := s.Character.ChangeMap(233, nil)
		//s.Conn.Write(gomap)

		s.Character.Map = 233
		s.Character.Coordinate = database.ConvertPointToCoordinate(508, 564)

	}

	if s.Character.Map == 243 {
		//gomap, _ := s.Character.ChangeMap(1, nil)
		//s.Conn.Write(gomap)
		s.Character.Map = 17
		s.Character.Coordinate = database.ConvertPointToCoordinate(37, 453)
	}

	sale := database.FindSale(s.Character.PseudoID)
	if sale != nil {
		sale.Delete()
	}

	trade := database.FindTrade(s.Character)
	if trade != nil {
		trade.Delete()
	}

	//s.Character.PartyMode = 33
	s.Character.IsinWar = false
	s.Character.IsinLastMan = false
	s.Character.HasLot = false
	s.Character.IsOnline = true
	s.Character.Respawning = false
	s.Character.SetInventorySlots(nil)
	s.Character.OnSight.Drops = make(map[int]interface{})
	s.Character.OnSight.NPCs = make(map[int]interface{})
	s.Character.OnSight.Mobs = make(map[int]interface{})
	s.Character.OnSight.Pets = make(map[int]interface{})
	s.Character.OnSight.Players = make(map[int]interface{})
	s.Character.UsedConsumables.Items = make(map[int64]bool)
	if s.Character.Injury > database.MAX_INJURY {
		s.Character.Injury = database.MAX_INJURY
	}
	s.Character.ExploreWorld = func() {
		for {
			if s.Character.ExploreWorld == nil {
				break
			} else if s.Character.IsActive {
				exploreWorld(s)
			}

			time.Sleep(time.Second)
		}
	}

	s.Character.HandlerCB = s.Character.Handler
	coordinate := database.ConvertPointToLocation(s.Character.Coordinate)
	mapData, err := s.Character.ChangeMap(s.Character.Map, coordinate, true)
	if err != nil {
		return nil, err
	}

	resp := GAME_STARTED
	resp.Insert(utils.IntToBytes(uint64(s.Character.PseudoID), 2, true), 13) // pseudo id
	resp.Insert(utils.IntToBytes(uint64(s.Character.ID), 4, true), 15)       // character id

	index := 20
	resp.Insert([]byte(s.Character.Name), index) // character name
	index += len(s.Character.Name)

	for i := len(s.Character.Name); i < 18; i++ {
		resp.Insert([]byte{0x00}, index)
		index++
	}

	resp[index] = byte(s.Character.Type) // character type
	index += 1

	resp[index] = byte(s.Character.Faction) // character faction
	index += 1

	resp[index] = 4
	index += 1

	resp[index] = byte(s.Character.Map - 1) // character map
	index += 2

	resp.Insert(utils.FloatToBytes(coordinate.X, 4, true), index) // character coordinate-x
	index += 4

	resp.Insert(utils.FloatToBytes(coordinate.Y, 4, true), index) // character coordinate-y
	index += 4
	index += 10

	resp.Overwrite(utils.IntToBytes(uint64(s.Character.Socket.Stats.Honor), 4, true), index)
	index += 4

	s.Write(resp)
	resp = utils.Packet{}
	//resp = utils.Packet{0xAA, 0x55, 0xDA, 0x01, 0x17, 0xD5, 0x01, 0x98, 0x10, 0x1B, 0x9A, 0x10, 0x08, 0x30, 0x02, 0xFC, 0x0F, 0x02, 0x00, 0x0B, 0x32, 0x52, 0x65, 0x76, 0x6F, 0x6C, 0x75, 0x74, 0x69, 0x6F, 0x6E, 0x39, 0x02, 0x00, 0x00, 0xAE, 0xBA, 0xAA, 0x11, 0x43, 0x93, 0xA3, 0x52, 0x43, 0x00, 0x00, 0x80, 0x40, 0x05, 0x00, 0x00, 0x00, 0x10, 0x27, 0x00, 0x00, 0x2F, 0x20, 0x07, 0x00, 0x0C, 0xA0, 0x01, 0x00, 0x06, 0x20, 0x01, 0xE0, 0x03, 0x00, 0x00, 0x14, 0x20, 0x0C, 0x00, 0x01, 0x20, 0x03, 0x03, 0x00, 0x00, 0xB2, 0x08, 0x20, 0x04, 0x20, 0x00, 0x01, 0x04, 0x10, 0x20, 0x04, 0x20, 0x00, 0x05, 0x20, 0x1C, 0x00, 0x00, 0x3A, 0x02, 0x80, 0x03, 0x00, 0x9E, 0x20, 0x0F, 0x40, 0x03, 0x00, 0x90, 0x40, 0x2A, 0x01, 0x03, 0x01, 0x20, 0x00, 0x20, 0x10, 0xA0, 0x00, 0x04, 0x02, 0x00, 0x05, 0x27, 0x0A, 0xA0, 0x0B, 0xE0, 0x3D, 0x00, 0x0A, 0x16, 0x00, 0x90, 0x5A, 0xF6, 0x05, 0x00, 0xA1, 0x01, 0x00, 0x03, 0xE0, 0x1A, 0x50, 0x00, 0x91, 0xA0, 0x2B, 0x00, 0x04, 0xE0, 0x1A, 0x2B, 0x0A, 0xD8, 0x51, 0x9D, 0x00, 0x00, 0xA2, 0x01, 0x00, 0x0B, 0x00, 0x42, 0xE0, 0x18, 0x2D, 0x08, 0xC1, 0x75, 0x06, 0x01, 0x03, 0xA1, 0xD9, 0x01, 0x0C, 0xE0, 0x18, 0x29, 0x02, 0x00, 0x00, 0xC2, 0x20, 0x2B, 0x04, 0x00, 0xA1, 0x58, 0x02, 0x0D, 0x20, 0x0B, 0xE0, 0x17, 0x00, 0x08, 0xA9, 0x08, 0x0B, 0x01, 0x00, 0xA6, 0x01, 0x00, 0x0E, 0xE0, 0x17, 0x28, 0x20, 0x00, 0x00, 0xC3, 0x40, 0x57, 0x20, 0xAF, 0x03, 0x0F, 0x00, 0xA0, 0x86, 0xE1, 0x02, 0x72, 0xE0, 0x0C, 0x00, 0x07, 0x41, 0xC2, 0xA1, 0x00, 0x00, 0xA1, 0x03, 0x00, 0xA1, 0xBE, 0xE0, 0x14, 0x00, 0x02, 0xF9, 0x68, 0xC6, 0x20, 0x2B, 0x02, 0x01, 0x00, 0x11, 0xE0, 0x14, 0x25, 0x80, 0x00, 0x01, 0x62, 0xBE, 0x40, 0x57, 0x02, 0x05, 0x00, 0x12, 0x80, 0x0E, 0xE0, 0x14, 0x00, 0x02, 0x98, 0x44, 0x9A, 0x60, 0x57, 0x00, 0x13, 0xE0, 0x14, 0x25, 0x80, 0x00, 0x02, 0x58, 0x65, 0xB9, 0x60, 0x2B, 0x42, 0x81, 0xE0, 0x17, 0x00, 0x00, 0x96, 0xA0, 0x57, 0x00, 0x15, 0xE0, 0x17, 0x28, 0x20, 0x00, 0x00, 0xD5, 0x41, 0xE3, 0x20, 0xDB, 0x00, 0x16, 0x20, 0x0B, 0xE0, 0x17, 0x00, 0x00, 0xD7, 0x40, 0x2B, 0x21, 0x5F, 0x03, 0x17, 0x00, 0x47, 0x51, 0xE0, 0x17, 0x2B, 0x02, 0x56, 0x6C, 0xA3, 0x62, 0x3B, 0x02, 0x18, 0x00, 0x60, 0xE0, 0x17, 0x2A, 0x01, 0x00, 0x66, 0x61, 0x33, 0x02, 0x01, 0x00, 0x19, 0xE0, 0x18, 0x29, 0x04, 0x00, 0x00, 0x35, 0xCB, 0x9B, 0x61, 0x07, 0x00, 0x1A, 0x20, 0x0B, 0xE0, 0x17, 0x00, 0x00, 0x64, 0x60, 0x57, 0x02, 0x02, 0x00, 0x1B, 0xE0, 0x17, 0x28, 0x20, 0x00, 0x00, 0x95, 0xA1, 0x33, 0x23, 0xC6, 0xE0, 0x18, 0x00, 0x00, 0x56, 0xA1, 0x8B, 0x00, 0x1D, 0xE0, 0x18, 0x29, 0x04, 0x00, 0x00, 0x37, 0xF9, 0xBD, 0x60, 0xAF, 0x00, 0x1E, 0x20, 0x0B, 0xE0, 0xFF, 0x00, 0xE0, 0xFF, 0x00, 0xE0, 0xFF, 0x00, 0xE0, 0xFF, 0x00, 0xE0, 0x16, 0x00, 0x00, 0x03, 0xE7, 0x1B, 0xDF, 0xE0, 0xAB, 0x00, 0xE0, 0x00, 0xD7, 0xE0, 0x00, 0x08, 0xE0, 0xFF, 0x00, 0xE0, 0x17, 0x00, 0xE1, 0xFF, 0x30, 0xE0, 0xFF, 0x00, 0xE0, 0xFF, 0x00, 0xE0, 0xFF, 0x00, 0xE0, 0xFF, 0x00, 0xE0, 0x8C, 0x00, 0x01, 0x00, 0x00, 0x55, 0xAA}
	ggh := &player.GetGoldHandler{}
	gold, _ := ggh.Handle(s)
	s.Write(gold)

	gih := &player.GetInventoryHandler{}
	inventory, err := gih.Handle(s)
	if err != nil {
		return nil, err
	}

	s.Write(inventory)
	s.Write(s.Character.GetPetStats())
	s.Write(mapData)

	//HONOR_RANKS      = []int{0, 1, 2, 14, 30, 50, 4}

	tmpHonor := s.Character.Socket.Stats.Honor

	if tmpHonor >= 15000 {
		s.Character.HonorRank = 1 // Bushido
	} else if tmpHonor >= 14000 {
		s.Character.HonorRank = 2 // God of Death
	} else if tmpHonor >= 13000 {
		s.Character.HonorRank = 4 // Grandmaster
	} else if tmpHonor >= 12000 {
		s.Character.HonorRank = 14 // Invincible
	} else if tmpHonor >= 11000 {
		s.Character.HonorRank = 30 // Unkown
	} else if tmpHonor > 9950 {
		s.Character.HonorRank = 0 // Nameless
	} else if tmpHonor <= 9950 {
		s.Character.HonorRank = 50 // Noob
	}

	honorresp := database.CHANGE_RANK
	honorresp.Insert(utils.IntToBytes(uint64(s.Character.PseudoID), 2, true), 6)
	honorresp.Insert(utils.IntToBytes(uint64(s.Character.HonorRank), 4, true), 8)
	s.Write(honorresp)

	spawnData, err := s.Character.SpawnCharacter()
	if err != nil {
		return nil, err
	}
	s.Write(spawnData)

	gsh := &player.GetStatsHandler{}
	statData, err := gsh.Handle(s)
	if err != nil {
		return nil, err
	}

	s.Write(statData)
	s.Write(s.User.GetTime())

	skillsData, err := s.Skills.GetSkillsData()
	if err != nil {
		return nil, err
	}

	s.Write(skillsData)
	s.Write(s.Character.GetGold())

	r := player.HT_VISIBILITY
	r[9] = byte(s.Character.HTVisibility)
	s.Write(r)

	r = npc.JOB_PROMOTED
	r[6] = byte(s.Character.Class)
	s.Write(r)

	guildData, err := s.Character.GetGuildData()
	if err != nil {
		return nil, err
	}

	s.Write(guildData)

	slotData := utils.Packet{}
	slotData.Concat(s.Character.Slotbar)
	s.Write(slotData)

	if s.Character.GuildID > 0 {
		guild, err := database.FindGuildByID(s.Character.GuildID)
		if err != nil {
			return nil, err
		} else if guild != nil {
			database.AddFiveBuffWhenLogin(s.Character)
			//database.AddGuildWarBuffWhenLogin(s.Character)
			//guild.AddGuildTierBuff(s.Character)
			guild.InformMembers(s.Character)
		}
	}

	//s.Character.FixDropAndExp()

	if s.User.ConnectedServer == 0 {
		/*gomap, _ := */ s.Character.ChangeMap(1, nil)
		//s.Conn.Write(gomap)
		s.OnClose()
		return nil, nil
	}

	s.Write(AID_ITEM_HANDLE)
	time.AfterFunc(time.Second*1, func() {
		if s.Character.ExploreWorld != nil {
			go s.Character.ExploreWorld()
		}

		if s.Character.HandlerCB != nil {
			go s.Character.HandlerCB()
		}
	})

	go s.Character.ActivityStatus(30)

	logger.Log(logging.ACTION_START_GAME, s.Character.ID, "Started the game", s.User.ID, s.Character.Name)
	return nil, nil
}

func exploreWorld(s *database.Socket) {

	defer func() {
		if err := recover(); err != nil {
			log.Println("Erroru incele ------------")
			log.Println(s.Character.Name)
			log.Println(s.Character.Map)
			log.Println(s.User.ConnectedServer)
			log.Println(err)
			log.Printf("%+v", string(dbg.Stack()))

		}
	}()

	explorePlayers(s)
	exploreMobs(s)
	exploreNPCs(s)
	exploreDrops(s)
	explorePets(s)
}

func explorePlayers(s *database.Socket) {
	c := s.Character
	if c == nil {
		return
	}

	resp := utils.Packet{}
	characters, err := c.GetNearbyCharacters()
	if err != nil {
		log.Println(err)
		return
	}

	for _, character := range characters {
		if character.IsMounting {
			c.OnSight.PlayerMutex.Lock()
			delete(c.OnSight.Players, character.ID)
			c.OnSight.PlayerMutex.Unlock()
		}
		c.OnSight.PlayerMutex.RLock()
		_, ok := c.OnSight.Players[character.ID]
		c.OnSight.PlayerMutex.RUnlock()

		if !ok {
			c.OnSight.PlayerMutex.Lock()
			c.OnSight.Players[character.ID] = character.PseudoID
			c.OnSight.PlayerMutex.Unlock()

			data, err := character.SpawnCharacter()
			if err != nil {
				log.Println(err)
				return
			}

			resp = data
			resp.Concat(character.GetHPandChi())
			//s.Conn.Write(resp)
			s.Write(resp)
		}
	}

	ids := funk.Map(characters, func(c *database.Character) int {
		return c.ID
	}).([]int)

	c.OnSight.PlayerMutex.RLock()
	losers := utils.SliceDiff(utils.Keys(c.OnSight.Players), ids)
	c.OnSight.PlayerMutex.RUnlock()

	for _, id := range losers {

		loser, err := database.FindCharacterByID(id)
		if err != nil {
			log.Println(err)
			return
		}

		c.OnSight.PlayerMutex.RLock()
		pseudoID := c.OnSight.Players[loser.ID].(uint16)
		c.OnSight.PlayerMutex.RUnlock()

		d := CHARACTER_GONE
		d.Insert(utils.IntToBytes(uint64(pseudoID), 2, true), 6)
		//s.Conn.Write(d)
		s.Write(d)

		c.OnSight.PlayerMutex.Lock()
		delete(c.OnSight.Players, id)
		c.OnSight.PlayerMutex.Unlock()

	}
}

func Tester(s []uint16, e uint16) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func remove(slice []int, s int) []int {
	pos := 0
	for _, i := range slice {
		if i == s {
			copy(slice[i:], slice[i+1:])
			//slice[len(slice)-1] = nil // or the zero value of T
			slice = slice[:len(slice)-1]
			continue
		}
		pos++
	}
	return slice
}

func exploreMobs(s *database.Socket) {
	c := s.Character
	if c == nil {
		return
	}

	ids, err := c.GetNearbyAIIDs()
	if err != nil {
		log.Println(err)
		return
	}

	for _, id := range ids {
		database.AIMutex.Lock()
		mob := database.AIs[id]
		database.AIMutex.Unlock()

		/*
			if c.IsinWar {
				isStone := Tester(database.WarStonesIDs, mob.PseudoID)
				if isStone {
					delete(c.OnSight.Mobs, id)
				}
			}
		*/

		c.OnSight.MobMutex.RLock()
		_, ok := c.OnSight.Mobs[id]
		c.OnSight.MobMutex.RUnlock()

		if mob.IsDead && ok {
			c.OnSight.MobMutex.Lock()
			delete(c.OnSight.Mobs, id)
			c.OnSight.MobMutex.Unlock()

			mob.PlayersMutex.Lock()
			delete(mob.OnSightPlayers, c.ID)
			mob.PlayersMutex.Unlock()

		} else if !mob.IsDead && !ok {
			c.OnSight.MobMutex.Lock()
			c.OnSight.Mobs[id] = struct{}{}
			c.OnSight.MobMutex.Unlock()

			mob.PlayersMutex.Lock()
			mob.OnSightPlayers[c.ID] = struct{}{}
			mob.PlayersMutex.Unlock()

			npcID := uint64(database.NPCPos[mob.PosID].NPCID)
			npc := database.NPCs[int(npcID)]
			coordinate := database.ConvertPointToLocation(mob.Coordinate)

			r := database.MOB_APPEARED
			if (mob.Faction != 0 && mob.Faction == c.Faction) || mob.Faction == 3 { //faction 3 = neutral
				r.Overwrite(utils.IntToBytes(uint64(1), 4, true), 6)
				npc.Level = 1
			} else {
				npc2 := database.NPCs[int(npcID)]
				npc.Level = npc2.Level
				r.Overwrite([]byte{0xFF, 0xFF, 0xFF, 0xFF}, 6)
			}
			r.Insert(utils.IntToBytes(uint64(mob.PseudoID), 2, true), 6) // mob pseudo id
			r.Insert(utils.IntToBytes(npcID, 4, true), 8)                // mob npc id
			r.Insert(utils.IntToBytes(uint64(npc.Level), 4, true), 12)   // mob level
			index := 20
			r.Insert(utils.IntToBytes(uint64(len(npc.Name)), 1, true), index)
			index++
			//npcCurrentHPHalf := (mob.HP / 2) / 10
			npcMaxHPHalf := (npc.MaxHp / 2) / 10
			r.Insert([]byte(npc.Name), index) // mob name
			index += len(npc.Name)
			r.Insert(utils.IntToBytes(uint64(mob.HP), 4, true), index) // mob hp
			index += 4
			r.Insert(utils.IntToBytes(uint64(npcMaxHPHalf), 4, true), index) // mob half hp
			index += 4
			r.Insert(utils.IntToBytes(uint64(npc.MaxHp), 4, true), index) // mob max hp
			index += 4
			r.Insert(utils.IntToBytes(uint64(npcMaxHPHalf), 4, true), index) // mob half max hp
			index += 6
			r.Insert(utils.FloatToBytes(coordinate.X, 4, true), index) // coordinate-x
			index += 4
			r.Insert(utils.FloatToBytes(coordinate.Y, 4, true), index) // coordinate-y
			index += 8
			r.Insert(utils.FloatToBytes(coordinate.X, 4, true), index) // coordinate-x
			index += 4
			r.Insert(utils.FloatToBytes(coordinate.Y, 4, true), index) // coordinate-y
			index += 4
			r.SetLength(int16(index + 16))
			if c.IsinWar {
				isStone := Tester(database.WarStonesIDs, mob.PseudoID)
				if isStone {
					if c.Faction == 1 {
						if ok, _ := utils.Contains(database.WarStones[int(mob.PseudoID)].NearbyZuhangV, c.ID); !ok {
							database.WarStones[int(mob.PseudoID)].NearbyZuhangV = append(database.WarStones[int(mob.PseudoID)].NearbyZuhangV, c.ID)
						}
					} else {
						if ok, _ := utils.Contains(database.WarStones[int(mob.PseudoID)].NearbyShaoV, c.ID); !ok {
							database.WarStones[int(mob.PseudoID)].NearbyShaoV = append(database.WarStones[int(mob.PseudoID)].NearbyShaoV, c.ID)
						}
					}
					if c.Socket.Stats.HP <= 0 {
						if c.Faction == 1 {
							if ok, _ := utils.Contains(database.WarStones[int(mob.PseudoID)].NearbyZuhangV, c.ID); ok {
								database.WarStones[int(mob.PseudoID)].RemoveZuhang(c.ID)
							}
						} else {
							if ok, _ := utils.Contains(database.WarStones[int(mob.PseudoID)].NearbyShaoV, c.ID); ok {
								database.WarStones[int(mob.PseudoID)].RemoveShao(c.ID)
							}
						}
					}
					resp := database.STONE_APPEARED
					resp.Insert(utils.IntToBytes(uint64(mob.PseudoID), 2, true), 6) // mob pseudo id
					resp.Insert(utils.IntToBytes(npcID, 4, true), 8)                // mob npc id
					resp.Insert(utils.IntToBytes(uint64(npc.Level), 4, true), 12)   // mob level
					resp.Insert(utils.IntToBytes(uint64(mob.HP), 8, true), 33)      // mob hp
					resp.Insert(utils.IntToBytes(uint64(npc.MaxHp), 8, true), 41)   // mob max hp
					resp.Insert(utils.FloatToBytes(coordinate.X, 4, true), 51)      // coordinate-x
					resp.Insert(utils.FloatToBytes(coordinate.Y, 4, true), 55)      // coordinate-y
					resp.Insert(utils.FloatToBytes(coordinate.X, 4, true), 63)      // coordinate-x
					resp.Insert(utils.FloatToBytes(coordinate.Y, 4, true), 67)      // coordinate-y
					resp.Overwrite(utils.IntToBytes(uint64(database.WarStones[int(mob.PseudoID)].ConquereValue), 1, false), 37)
					resp.Overwrite([]byte{0xc8}, 45)
					//s.Conn.Write(resp)
					s.Write(resp)
					continue
				}
			}
			//s.Conn.Write(r)
			s.Write(r)
			//resp.Concat(r)
		}
	}

	c.OnSight.MobMutex.RLock()
	losers := utils.SliceDiff(utils.Keys(c.OnSight.Mobs), ids)
	c.OnSight.MobMutex.RUnlock()
	//losers = append(losers, utils.SliceDiff(ids, utils.Keys(c.OnSight.Mobs))...)

	for _, id := range losers {
		database.AIMutex.Lock()
		loser := database.AIs[id]
		database.AIMutex.Unlock()
		coordinate := database.ConvertPointToLocation(loser.Coordinate)

		if c.IsinWar {
			isStone := Tester(database.WarStonesIDs, loser.PseudoID)
			if isStone {
				if c.Faction == 1 {
					if ok, _ := utils.Contains(database.WarStones[int(loser.PseudoID)].NearbyZuhangV, c.ID); ok {
						database.WarStones[int(loser.PseudoID)].RemoveZuhang(c.ID)
					}
				} else {
					if ok, _ := utils.Contains(database.WarStones[int(loser.PseudoID)].NearbyShaoV, c.ID); ok {
						database.WarStones[int(loser.PseudoID)].RemoveShao(c.ID)
					}
				}
			}

		}

		r := MOB_DISAPPEARED
		r.Insert(utils.IntToBytes(uint64(loser.PseudoID), 2, true), 6) // mob pseudo id
		r.Insert(utils.FloatToBytes(coordinate.X, 4, true), 12)        // coordinate-x
		r.Insert(utils.FloatToBytes(coordinate.Y, 4, true), 16)        // coordinate-y

		//s.Conn.Write(r)
		s.Write(r)
		//resp.Concat(r)

		c.OnSight.MobMutex.Lock()
		delete(c.OnSight.Mobs, loser.ID)
		c.OnSight.MobMutex.Unlock()

		loser.PlayersMutex.Lock()
		delete(loser.OnSightPlayers, c.ID)
		loser.PlayersMutex.Unlock()
	}
}

func explorePets(s *database.Socket) {
	c := s.Character
	if c == nil {
		return
	}

	characters, err := c.GetNearbyCharacters()
	if err != nil {
		log.Println(err)
		return
	}

	characters = append(characters, c)
	petSlots := make(map[int]*database.InventorySlot)
	petIDs := []int{}

	characters = funk.Filter(characters, func(ch *database.Character) bool {
		slots, err := ch.InventorySlots()
		if err != nil {
			return false
		}

		petSlot := slots[0x0A]
		if petSlot.Pet == nil || !petSlot.Pet.IsOnline {
			return false
		}

		petIDs = append(petIDs, petSlot.Pet.PseudoID)
		petSlots[ch.ID] = petSlot
		return true
	}).([]*database.Character)

	resp := utils.Packet{}
	for _, character := range characters {

		petSlot := petSlots[character.ID]
		pet := petSlot.Pet

		c.OnSight.PetsMutex.RLock()
		_, ok := c.OnSight.Pets[pet.PseudoID]
		c.OnSight.PetsMutex.RUnlock()

		if pet.HP <= 0 {

			c.OnSight.PetsMutex.Lock()
			delete(c.OnSight.Pets, pet.PseudoID)
			c.OnSight.PetsMutex.Unlock()

		} else if !ok {

			c.OnSight.PetsMutex.Lock()
			c.OnSight.Pets[pet.PseudoID] = struct{}{}
			c.OnSight.PetsMutex.Unlock()

			r := database.PET_APPEARED
			r.Insert(utils.IntToBytes(uint64(pet.PseudoID), 2, true), 6)   // pet pseudo id
			r.Insert(utils.IntToBytes(uint64(petSlot.ItemID), 4, true), 8) // pet npc id
			r.Insert(utils.IntToBytes(uint64(pet.Level), 4, true), 12)     // pet level
			//r.Overwrite(utils.IntToBytes(1, 4, true), 16)                  //Pets to neutral
			//r.Insert([]byte{0x09, 0x57, 0x69, 0x6C, 0x64, 0x20, 0x42, 0x6F, 0x61, 0x72}, 20)
			//r.Insert(utils.IntToBytes(uint64(len(pet.Name)), 1, true), 20)
			//	index++
			index := 0
			if pet.Name != "" {
				r.Insert(utils.IntToBytes(uint64(len(character.Name+"|"+pet.Name)), 1, true), 20)
				r.Insert([]byte(character.Name+"|"+pet.Name), 21) // pet name
				index = len(character.Name+"|"+pet.Name) + 21
			} else {
				r.Insert(utils.IntToBytes(uint64(len(character.Name)), 1, true), 20)
				r.Insert([]byte(character.Name), 21) // pet name
				index = len(character.Name) + 21
			}
			r.Insert(utils.IntToBytes(uint64(pet.HP), 4, true), index)        // pet hp
			r.Insert(utils.IntToBytes(uint64(pet.CHI), 4, true), index+4)     // pet chi
			r.Insert(utils.IntToBytes(uint64(pet.MaxHP), 4, true), index+8)   // pet max hp
			r.Insert(utils.IntToBytes(uint64(pet.MaxCHI), 4, true), index+12) // pet max chi
			r.Insert(utils.IntToBytes(3, 2, true), index+16)                  //
			r.Insert(utils.FloatToBytes(pet.Coordinate.X, 4, true), index+18) // coordinate-x
			r.Insert(utils.FloatToBytes(pet.Coordinate.Y, 4, true), index+22) // coordinate-y
			r.Insert(utils.FloatToBytes(12, 4, true), index+26)               // z?
			r.Insert(utils.FloatToBytes(pet.Coordinate.X, 4, true), index+30) // coordinate-x
			r.Insert(utils.FloatToBytes(pet.Coordinate.Y, 4, true), index+34) // coordinate-y
			r.Insert(utils.FloatToBytes(12, 4, true), index+38)               // z?
			r.Insert([]byte{0x00, 0x00, 0x00, 0x00}, index+42)
			//r = append(r[:index+42], r[index+50:]...)
			//r.Overwrite(utils.Packet{0x00, 0x00, 0x00, 0x00, 0xFF, 0xFF, 0x00, 0xE8, 0x03, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}, index+42)

			r.SetLength(int16(binary.Size(r) - 6))

			resp.Concat(r)
		}
	}

	c.OnSight.PetsMutex.RLock()
	losers := utils.SliceDiff(utils.Keys(c.OnSight.Pets), petIDs)
	c.OnSight.PetsMutex.RUnlock()

	for _, id := range losers {
		/*/
		loser, ok := database.GetFromRegister(c.Socket.User.ConnectedServer, c.Map, uint16(id)).(*database.PetSlot)
		if !ok {
			continue
		}
		*/

		r := MOB_DISAPPEARED
		r.Insert(utils.IntToBytes(uint64(id), 2, true), 6) // mob pseudo id
		r.Insert(utils.FloatToBytes(0, 4, true), 12)       // coordinate-x
		r.Insert(utils.FloatToBytes(0, 4, true), 16)       // coordinate-y

		resp.Concat(r)
		c.OnSight.PetsMutex.Lock()
		delete(c.OnSight.Pets, id)
		c.OnSight.PetsMutex.Unlock()
	}

	/*
		_, err = s.Conn.Write(resp)
		if err != nil {
			log.Println(err)
			return
		}
	*/

	s.Write(resp)
}

func exploreNPCs(s *database.Socket) {
	c := s.Character
	if c == nil {
		return
	}

	ids, err := c.GetNearbyNPCIDs()
	if err != nil {
		log.Println(err)
		return
	}

	npcPosIds := []int{}
	resp := utils.Packet{}
	for _, id := range ids {
		npcPos := database.NPCPos[id]
		npc := database.NPCs[npcPos.NPCID]
		npcPosIds = append(npcPosIds, npcPos.ID)

		c.OnSight.NpcMutex.RLock()
		_, ok := c.OnSight.NPCs[npcPos.ID]
		c.OnSight.NpcMutex.RUnlock()

		if !ok {
			c.OnSight.NpcMutex.Lock()
			c.OnSight.NPCs[npcPos.ID] = struct{}{}
			c.OnSight.NpcMutex.Unlock()

			minLocation := database.ConvertPointToLocation(npcPos.MinLocation)
			maxLocation := database.ConvertPointToLocation(npcPos.MaxLocation)
			coordinate := &utils.Location{X: (minLocation.X + maxLocation.X) / 2, Y: (minLocation.Y + maxLocation.Y) / 2}

			r := NPC_APPEARED
			r.Insert(utils.IntToBytes(uint64(npcPos.PseudoID), 2, true), 6) // npc pseudo id
			r.Insert(utils.IntToBytes(uint64(npc.ID), 4, true), 8)          // npc id
			r.Insert(utils.IntToBytes(uint64(npc.Level), 4, true), 12)      // npc level
			r.Insert(utils.IntToBytes(uint64(npc.MaxHp), 4, true), 39)      // npc hp
			r.Insert(utils.IntToBytes(uint64(npc.MaxHp), 4, true), 47)      // npc hp
			r.Insert(utils.FloatToBytes(coordinate.X, 4, true), 57)         // coordinate-x
			r.Insert(utils.FloatToBytes(coordinate.Y, 4, true), 61)         // coordinate-y
			r.Insert(utils.FloatToBytes(coordinate.X, 4, true), 69)         // coordinate-x
			r.Insert(utils.FloatToBytes(coordinate.Y, 4, true), 73)         // coordinate-y

			resp.Concat(r)
		}
	}

	c.OnSight.NpcMutex.RLock()
	losers := utils.SliceDiff(utils.Keys(c.OnSight.NPCs), npcPosIds)
	c.OnSight.NpcMutex.RUnlock()

	for _, id := range losers {
		loserPos := funk.Filter(database.NPCPos, func(pos *database.NpcPosition) bool {
			return pos.ID == id
		}).([]*database.NpcPosition)[0]

		if loserPos == nil {
			continue
		}

		minLocation := database.ConvertPointToLocation(loserPos.MinLocation)
		maxLocation := database.ConvertPointToLocation(loserPos.MaxLocation)
		coordinate := &utils.Location{X: (minLocation.X + maxLocation.X) / 2, Y: (minLocation.Y + maxLocation.Y) / 2}

		r := NPC_DISAPPEARED
		r.Insert(utils.IntToBytes(uint64(loserPos.PseudoID), 2, true), 6) // mob pseudo id
		r.Insert(utils.FloatToBytes(coordinate.X, 4, true), 12)           // coordinate-x
		r.Insert(utils.FloatToBytes(coordinate.Y, 4, true), 16)           // coordinate-y

		resp.Concat(r)
		c.OnSight.NpcMutex.Lock()
		delete(c.OnSight.NPCs, loserPos.ID)
		c.OnSight.NpcMutex.Unlock()
	}

	/*
		_, err = s.Conn.Write(resp)
		if err != nil {
			log.Println(err)
			return
		}
	*/
	s.Write(resp)
}

func exploreDrops(s *database.Socket) {
	c := s.Character
	if c == nil {
		return
	}

	ids, err := c.GetNearbyDrops()
	if err != nil {
		log.Println(err)
		return
	}

	func() {
		for _, id := range ids {

			drop := database.GetDrop(s.User.ConnectedServer, c.Map, uint16(id))
			if drop == nil {
				continue
			}

			c.OnSight.DropsMutex.RLock()
			_, ok := c.OnSight.Drops[id]
			c.OnSight.DropsMutex.RUnlock()

			claimer := drop.Claimer
			if claimer == nil {
				claimer = s.Character
			}

			if !ok {
				c.OnSight.DropsMutex.Lock()
				c.OnSight.Drops[id] = struct{}{}
				c.OnSight.DropsMutex.Unlock()

				r := database.ITEM_DROPPED
				r.Insert(utils.IntToBytes(uint64(id), 2, true), 6) // drop id

				r.Insert(utils.FloatToBytes(drop.Location.X, 4, true), 10) // drop coordinate-x
				r.Insert(utils.FloatToBytes(drop.Location.Y, 4, true), 18) // drop coordinate-y

				r.Insert(utils.IntToBytes(uint64(drop.Item.ItemID), 4, true), 22) // item id
				if drop.Item.Plus > 0 {
					r[27] = 0xA2
					r.Insert(drop.Item.GetUpgrades(), 32)                             // item upgrades
					r.Insert([]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}, 47) // item sockets
					r.Insert(utils.IntToBytes(uint64(claimer.PseudoID), 2, true), 66) // claimer id
					r.SetLength(0x42)
				} else {
					r[27] = 0xA1
					r.Insert(utils.IntToBytes(uint64(claimer.PseudoID), 2, true), 36) // claimer id
					r.SetLength(0x24)
				}

				//s.Conn.Write(r)
				s.Write(r)

			}
		}
	}()

	c.OnSight.DropsMutex.RLock()
	losers := utils.SliceDiff(utils.Keys(c.OnSight.Drops), ids)
	c.OnSight.DropsMutex.RUnlock()

	func() {
		for _, id := range losers {

			r := database.DROP_DISAPPEARED
			r.Insert(utils.IntToBytes(uint64(id), 2, true), 6) //drop id

			//s.Conn.Write(r)
			s.Write(r)

			c.OnSight.DropsMutex.Lock()
			delete(c.OnSight.Drops, id)
			c.OnSight.DropsMutex.Unlock()
		}
	}()
}
