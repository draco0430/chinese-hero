package player

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"hero-server/database"
	"hero-server/logging"
	"hero-server/messaging"
	"hero-server/nats"
	"hero-server/server"
	"hero-server/utils"

	"gopkg.in/guregu/null.v3"

	"github.com/thoas/go-funk"
)

type ChatHandler struct {
	chatType       int64
	message        string
	receivers      map[int]*database.Character
	receiversMutex sync.Mutex
}

type Emotion struct{}

var (
	CHAT_MESSAGE  = utils.Packet{0xAA, 0x55, 0x00, 0x00, 0x00, 0x55, 0xAA}
	SHOUT_MESSAGE = utils.Packet{0xAA, 0x55, 0x00, 0x00, 0x71, 0x0E, 0x00, 0x00, 0x55, 0xAA}
	ANNOUNCEMENT  = utils.Packet{0xAA, 0x55, 0x00, 0x00, 0x71, 0x06, 0x00, 0x55, 0xAA}
)

func (h *Emotion) Handle(s *database.Socket, data []byte) ([]byte, error) {
	emotID := int(utils.BytesToInt(data[11:12], true))
	emotion := database.Emotions[emotID]
	if emotion == nil {
		return nil, nil
	}

	resp := utils.Packet{0xAA, 0x55, 0x09, 0x00, 0x71, 0x09, 0x00, 0x00, 0x55, 0xAA}
	resp.Insert(utils.IntToBytes(uint64(s.Character.PseudoID), 2, true), 6)
	resp.Insert(utils.IntToBytes(uint64(emotion.Type), 1, true), 8)
	resp.Insert(utils.IntToBytes(uint64(emotion.AnimationID), 2, true), 9)

	p := &nats.CastPacket{CastNear: true, CharacterID: s.Character.ID, Data: resp, Type: nats.CHAT_NORMAL}
	err := p.Cast()
	return nil, err
}

func (h *ChatHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {

	if s.Character == nil {
		return nil, nil
	}

	user, err := database.FindUserByID(s.Character.UserID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, nil
	}

	stat := s.Stats
	if stat == nil {
		return nil, nil
	}

	h.chatType = utils.BytesToInt(data[4:6], false)

	switch h.chatType {
	case 28929: // normal chat
		messageLen := utils.BytesToInt(data[6:8], true)
		h.message = string(data[8 : messageLen+8])
		logging.AddLogFile(1, s.Character.Name+": "+h.message+" (N)")

		return h.normalChat(s)
	case 28930: // private chat
		index := 6
		recNameLength := int(data[index])
		index++

		recName := string(data[index : index+recNameLength])
		index += recNameLength

		c, err := database.FindCharacterByName(recName)
		if err != nil {
			return nil, err
		} else if c == nil {
			return messaging.SystemMessage(messaging.WHISPER_FAILED), nil
		}

		h.receiversMutex.Lock()
		h.receivers = map[int]*database.Character{c.ID: c}
		h.receiversMutex.Unlock()

		messageLen := int(utils.BytesToInt(data[index:index+2], true))
		index += 2

		h.message = string(data[index : index+messageLen])
		logging.AddLogFile(1, s.Character.Name+": "+h.message+" (PM : "+c.Name+")")
		return h.chatWithReceivers(s, h.createChatMessage)

	case 28931: // party chat
		party := database.FindParty(s.Character)
		if party == nil {
			return nil, nil
		}

		messageLen := int(utils.BytesToInt(data[6:8], true))
		h.message = string(data[8 : messageLen+8])

		members := funk.Filter(party.GetMembers(), func(m *database.PartyMember) bool {
			return m.Accepted
		}).([]*database.PartyMember)
		members = append(members, &database.PartyMember{Character: party.Leader, Accepted: true})
		h.receiversMutex.Lock()
		h.receivers = map[int]*database.Character{}
		for _, m := range members {
			if m.ID == s.Character.ID {
				continue
			}

			h.receivers[m.ID] = m.Character
		}
		h.receiversMutex.Unlock()

		logging.AddLogFile(1, s.Character.Name+": "+h.message+" (Party)")
		return h.chatWithReceivers(s, h.createChatMessage)

	case 28932: // guild chat
		if s.Character.GuildID > 0 {
			guild, err := database.FindGuildByID(s.Character.GuildID)
			if err != nil {
				return nil, err
			}

			members, err := guild.GetMembers()
			if err != nil {
				return nil, err
			}

			messageLen := int(utils.BytesToInt(data[6:8], true))
			h.message = string(data[8 : messageLen+8])
			h.receiversMutex.Lock()
			h.receivers = map[int]*database.Character{}

			for _, m := range members {
				c, err := database.FindCharacterByID(m.ID)
				if err != nil || c == nil || !c.IsOnline || c.ID == s.Character.ID {
					continue
				}

				h.receivers[m.ID] = c
			}
			h.receiversMutex.Unlock()

			logging.AddLogFile(1, s.Character.Name+": "+h.message+" (Guild)")
			return h.chatWithReceivers(s, h.createChatMessage)
		}

	case 28933, 28946: // roar chat
		if stat.CHI < 100 || time.Now().Sub(s.Character.LastRoar) < 10*time.Second {
			return nil, nil
		}

		s.Character.LastRoar = time.Now()
		characters, err := database.FindCharactersInServer(user.ConnectedServer)
		if err != nil {
			log.Println(err)
			return nil, err
		}

		//delete(characters, s.Character.ID)
		h.receiversMutex.Lock()
		h.receivers = characters
		h.receiversMutex.Unlock()

		stat.CHI -= 100

		index := 6
		messageLen := int(utils.BytesToInt(data[index:index+2], true))
		index += 2

		h.message = string(data[index : index+messageLen])

		resp := utils.Packet{}
		_, err = h.chatWithReceivers(s, h.createChatMessage)
		if err != nil {
			log.Println(err)
			return nil, err
		}

		//resp.Concat(chat)
		resp.Concat(s.Character.GetHPandChi())
		logging.AddLogFile(1, s.Character.Name+": "+h.message+" (Roar)")
		return resp, nil

	case 28935: // commands
		index := 6
		messageLen := int(data[index])
		index++

		h.message = string(data[index : index+messageLen])
		return h.cmdMessage(s, data)

	case 28943: // shout
		logging.AddLogFile(1, s.Character.Name+": "+h.message+" (Shout)")
		return h.Shout(s, data)

	case 28945: // faction chat
		characters, err := database.FindCharactersInServer(user.ConnectedServer)
		if err != nil {
			return nil, err
		}

		//delete(characters, s.Character.ID)
		for _, c := range characters {
			if c.Faction != s.Character.Faction {
				delete(characters, c.ID)
			}
		}

		h.receiversMutex.Lock()
		h.receivers = characters
		h.receiversMutex.Unlock()
		index := 6
		messageLen := int(utils.BytesToInt(data[index:index+2], true))
		index += 2

		h.message = string(data[index : index+messageLen])
		resp := utils.Packet{}
		_, err = h.chatWithReceivers(s, h.createChatMessage)
		if err != nil {
			return nil, err
		}

		//resp.Concat(chat)
		logging.AddLogFile(1, s.Character.Name+": "+h.message+" (Faction)")
		return resp, nil

	}

	return nil, nil
}

func (h *ChatHandler) Shout(s *database.Socket, data []byte) ([]byte, error) {

	if time.Now().Sub(s.Character.LastRoar) < 10*time.Second {
		return nil, nil
	}

	characters, err := database.FindOnlineCharacters()
	if err != nil {
		return nil, err
	}

	//delete(characters, s.Character.ID)

	slot, _, err := s.Character.FindItemInInventory(nil, 15900001, 17500181, 17502689, 13000131, 18500094)
	if err != nil {
		return nil, err
	} else if slot == -1 {
		return nil, nil
	}

	resp := s.Character.DecrementItem(slot, 1)

	index := 6
	messageLen := int(data[index])
	index++

	h.chatType = 28942
	h.receiversMutex.Lock()
	h.receivers = characters
	h.receiversMutex.Unlock()
	h.message = string(data[index : index+messageLen])

	_, err = h.chatWithReceivers(s, h.createShoutMessage)
	if err != nil {
		return nil, err
	}

	//resp.Concat(chat)
	return *resp, nil
}

func (h *ChatHandler) createChatMessage(s *database.Socket) *utils.Packet {

	resp := CHAT_MESSAGE

	index := 4
	resp.Insert(utils.IntToBytes(uint64(h.chatType), 2, false), index) // chat type
	index += 2

	if h.chatType != 28946 {
		resp.Insert(utils.IntToBytes(uint64(s.Character.PseudoID), 2, true), index) // sender character pseudo id
		index += 2
	}

	resp[index] = byte(len(s.Character.Name)) // character name length
	index++

	resp.Insert([]byte(s.Character.Name), index) // character name
	index += len(s.Character.Name)

	resp.Insert(utils.IntToBytes(uint64(len(h.message)), 2, true), index) // message length
	index += 2

	resp.Insert([]byte(h.message), index) // message
	index += len(h.message)

	length := index - 4
	resp.SetLength(int16(length)) // packet length

	return &resp
}

func (h *ChatHandler) createShoutMessage(s *database.Socket) *utils.Packet {

	resp := SHOUT_MESSAGE
	length := len(s.Character.Name) + len(h.message) + 6
	resp.SetLength(int16(length)) // packet length

	index := 4
	resp.Insert(utils.IntToBytes(uint64(h.chatType), 2, false), index) // chat type
	index += 2

	resp[index] = byte(len(s.Character.Name)) // character name length
	index++

	resp.Insert([]byte(s.Character.Name), index) // character name
	index += len(s.Character.Name)

	resp[index] = byte(len(h.message)) // message length
	index++

	resp.Insert([]byte(h.message), index) // message
	return &resp
}

func (h *ChatHandler) normalChat(s *database.Socket) ([]byte, error) {

	if _, ok := server.MutedPlayers.Get(s.User.ID); ok {
		msg := "Chatting with this account is prohibited. Please contact our customer support service for more information."
		return messaging.InfoMessage(msg), nil
	}

	resp := h.createChatMessage(s)
	p := &nats.CastPacket{CastNear: true, CharacterID: s.Character.ID, Data: *resp, Type: nats.CHAT_NORMAL}
	err := p.Cast()

	return nil, err
}

func (h *ChatHandler) chatWithReceivers(s *database.Socket, msgHandler func(*database.Socket) *utils.Packet) ([]byte, error) {

	if _, ok := server.MutedPlayers.Get(s.User.ID); ok {
		msg := "Chatting with this account is prohibited. Please contact our customer support service for more information."
		return messaging.InfoMessage(msg), nil
	}

	resp := msgHandler(s)

	h.receiversMutex.Lock()
	defer h.receiversMutex.Unlock()
	for _, c := range h.receivers {
		if c == nil || !c.IsOnline {
			if h.chatType == 28930 { // PM
				return messaging.SystemMessage(messaging.WHISPER_FAILED), nil
			}
			continue
		}

		socket := database.GetSocket(c.UserID)
		if socket != nil {
			err := socket.Write(*resp)
			if err != nil {
				log.Println(err)
				return nil, err
			}
		}
	}

	return *resp, nil
}

func makeAnnouncement(msg string) {
	length := int16(len(msg) + 3)

	resp := ANNOUNCEMENT
	resp.SetLength(length)
	resp[6] = byte(len(msg))
	resp.Insert([]byte(msg), 7)

	p := nats.CastPacket{CastNear: false, Data: resp}
	p.Cast()
}

func (h *ChatHandler) cmdMessage(s *database.Socket, data []byte) ([]byte, error) {

	var (
		err  error
		resp utils.Packet
	)

	if parts := strings.Split(h.message, " "); len(parts) > 0 {
		if h.message != "/home" {
			logging.AddLogFile(0, s.Character.Name+": "+h.message+" (Admin)")
		}
		cmd := strings.ToLower(strings.TrimPrefix(parts[0], "/"))
		switch cmd {
		case "shout":
			return h.Shout(s, data)
		case "announce":
			if s.User.UserType < server.GA_USER {
				return nil, nil
			}

			if len(parts) < 2 {
				return nil, nil
			}

			msg := strings.Join(parts[1:], " ")
			makeAnnouncement(msg)
		case "item":
			if s.User.UserType < server.HGM_USER {
				return nil, nil
			}

			if len(parts) < 2 {
				return nil, nil
			}

			itemID, err := strconv.ParseInt(parts[1], 10, 64)
			if err != nil {
				return nil, err
			}

			quantity := int64(1)
			if len(parts) >= 3 {
				quantity, err = strconv.ParseInt(parts[2], 10, 64)
				if err != nil {
					return nil, err
				}
			}

			ch := s.Character
			if len(parts) >= 4 {
				chID, err := strconv.ParseInt(parts[3], 10, 64)
				if err == nil {
					chr, err := database.FindCharacterByID(int(chID))
					if err == nil {
						ch = chr
					}
				}
			}

			item := &database.InventorySlot{ItemID: itemID, Quantity: uint(quantity)}
			info := database.Items[itemID]

			if info.GetType() == database.PET_TYPE {
				petInfo := database.Pets[itemID]
				expInfo := database.PetExps[petInfo.Level-1]

				item.Pet = &database.PetSlot{
					Fullness: 100, Loyalty: 100,
					Exp:   uint64(expInfo.ReqExpEvo1),
					HP:    petInfo.BaseHP,
					Level: byte(petInfo.Level),
					Name:  petInfo.Name,
					CHI:   petInfo.BaseChi}
			}

			r, _, err := ch.AddItem(item, -1, false)
			if err != nil {
				return nil, err
			}

			go logger.Log(logging.ACTION_CREATE_ITEM, s.Character.ID, fmt.Sprintf("%s isimli oyuncu %d id li item oluşturdu.", s.Character.Name, itemID), s.User.ID, s.Character.Name)

			ch.Socket.Write(*r)
			return nil, nil
		case "addbuff":
			if s.User.UserType < server.HGM_USER {
				return nil, nil
			}
			if len(parts) < 2 {
				return nil, nil
			}
			number, _ := strconv.ParseInt(parts[1], 10, 32)
			buffinfo := database.BuffInfections[int(number)]
			buff := &database.Buff{ID: int(number), CharacterID: s.Character.ID, Name: buffinfo.Name, BagExpansion: false, StartedAt: s.Character.Epoch, Duration: int64(5) * 60}
			err := buff.Create()
			if err != nil {
				return nil, err
			}
		case "removebuff":
			if s.User.UserType < server.HGM_USER {
				return nil, nil
			}
			if len(parts) < 2 {
				return nil, nil
			}
			chID, err := strconv.ParseInt(parts[2], 10, 64)
			if err != nil {
				fmt.Println("1. ", err)
				return nil, nil
			}

			buffs, err := database.FindBuffsByCharacterID(int(chID))
			if err != nil {
				fmt.Println("2. ", err)
				return nil, nil
			}

			for i := range buffs {
				buffs[i].Delete()
			}

		case "gold":
			if s.User.UserType < server.HGM_USER {
				return nil, nil
			}

			if len(parts) < 2 {
				return nil, nil
			}

			amount, err := strconv.ParseInt(parts[1], 10, 64)
			if err != nil {
				return nil, err
			}

			s.Character.Gold += uint64(amount)
			h := &GetGoldHandler{}

			go logger.Log(logging.ACTION_CREATE_GOLD, s.Character.ID, fmt.Sprintf("%s isimli oyuncu %d gold oluşturdu", s.Character.Name, amount), s.User.ID, s.Character.Name)

			return h.Handle(s)
		case "upgrade":
			if s.User.UserType < server.HGM_USER || len(parts) < 3 {
				return nil, nil
			}

			slots, err := s.Character.InventorySlots()
			if err != nil {
				return nil, err
			}

			slotID, err := strconv.ParseInt(parts[1], 10, 64)
			if err != nil {
				return nil, err
			}

			code, err := strconv.ParseInt(parts[2], 10, 64)
			if err != nil {
				return nil, err
			}

			count := int64(1)
			if len(parts) > 3 {
				count, err = strconv.ParseInt(parts[3], 10, 64)
				if err != nil {
					return nil, err
				}
			}

			codes := []byte{}
			for i := 0; i < int(count); i++ {
				codes = append(codes, byte(code))
			}

			item := slots[slotID]
			go logger.Log(logging.ACTION_UPGRADE_GM_ITEM, s.Character.ID, fmt.Sprintf("%s isimli oyuncu %d id li iteme + bastı", s.Character.Name, item.ID), s.User.ID, s.Character.Name)
			return item.Upgrade(int16(slotID), codes...), nil
		case "exp":
			if s.User.UserType < server.HGM_USER {
				return nil, nil
			}

			if len(parts) < 2 {
				return nil, nil
			}

			amount, err := strconv.ParseInt(parts[1], 10, 64)
			if err != nil {
				return nil, err
			}

			ch := s.Character
			if len(parts) > 2 {
				c, err := database.FindCharacterByName(parts[2])
				if err != nil {
					return nil, err
				}
				ch = c
			}

			data, levelUp := ch.AddExp(amount)
			if levelUp {
				statData, err := ch.GetStats()
				if err == nil && ch.Socket != nil {
					ch.Socket.Write(statData)
				}
			}

			if ch.Socket != nil {
				ch.Socket.Write(data)
			}

			return nil, nil
		case "home":
			/*
				data, err := s.Character.ChangeMap(s.Character.Map, nil)
				if err != nil {
					return nil, err
				}

				resp.Concat(data)
			*/
			return nil, nil
		case "map":
			if s.User.UserType < server.GA_USER {
				return nil, nil
			}

			if len(parts) < 2 {
				return nil, nil
			}

			mapID, err := strconv.ParseInt(parts[1], 10, 64)
			if err != nil {
				return nil, err
			}

			if len(parts) >= 3 {
				c, err := database.FindCharacterByName(parts[2])
				if err != nil {
					return nil, err
				}

				data, err := c.ChangeMap(int16(mapID), nil)
				if err != nil {
					fmt.Println(err)
					return nil, err
				}

				database.GetSocket(c.UserID).Write(data)
				return nil, nil
			}

			return s.Character.ChangeMap(int16(mapID), nil)
		case "cash":
			if s.User.UserType < server.HGM_USER {
				return nil, nil
			}

			if len(parts) < 3 {
				return nil, nil
			}

			amount, err := strconv.ParseInt(parts[1], 10, 64)
			if err != nil {
				return nil, err
			}

			userID := parts[2]
			user, err := database.FindUserByID(userID)
			if err != nil {
				return nil, err
			} else if user == nil {
				return nil, nil
			}

			user.NCash += uint64(amount)
			user.Update()

			go logger.Log(logging.ACTION_ADD_NCASH, s.Character.ID, fmt.Sprintf("%s isimli oyuncu %s idli oyuncuya %d ncash verdi", s.Character.Name, user.ID, amount), s.User.ID, s.Character.Name)

			return messaging.InfoMessage(fmt.Sprintf("%d nCash loaded to %s (%s).", amount, user.Username, user.ID)), nil
		case "exprate":
			if s.User.UserType < server.GM_USER {
				return nil, nil
			}

			if len(parts) < 3 {
				return nil, nil
			}
			if len(parts) > 2 {
				rate := 0.0
				if am, err := strconv.ParseFloat(parts[1], 64); err == nil {
					database.EXP_RATE = am
					rate = am
				}
				minute, err := strconv.ParseInt(parts[2], 10, 64)
				if err != nil {
					return nil, err
				}
				time.AfterFunc(time.Duration(minute)*time.Minute, func() {
					database.EXP_RATE = database.DEFAULT_EXP_RATE
				})

				go logger.Log(logging.ACTION_EXP_RATE, s.Character.ID, fmt.Sprintf("%s isimli oyuncu exp oranını %f çekti", s.Character.Name, rate), s.User.ID, s.Character.Name)
			}

			return messaging.InfoMessage(fmt.Sprintf("EXP Rate now: %f", database.EXP_RATE)), nil
		case "droprate":
			if s.User.UserType < server.GM_USER {
				return nil, nil
			}

			if len(parts) < 3 {
				return nil, nil
			}
			if len(parts) > 2 {
				rate := 0.0
				if s, err := strconv.ParseFloat(parts[1], 64); err == nil {
					database.DROP_RATE = s
					rate = s
				}
				minute, err := strconv.ParseInt(parts[2], 10, 64)
				if err != nil {
					return nil, err
				}
				time.AfterFunc(time.Duration(minute)*time.Minute, func() {
					database.DROP_RATE = database.DEFAULT_DROP_RATE
				})

				go logger.Log(logging.ACTION_DROP_RATE, s.Character.ID, fmt.Sprintf("%s isimli oyuncu drop oranını %f çekti", s.Character.Name, rate), s.User.ID, s.Character.Name)
			}
			return messaging.InfoMessage(fmt.Sprintf("Drop Rate now: %f", database.DROP_RATE)), nil
		case "mob":
			if s.User.UserType < server.HGM_USER {
				return nil, nil
			}

			if len(parts) < 2 {
				return nil, nil
			}

			posId, err := strconv.ParseInt(parts[1], 10, 64)
			if err != nil {
				return nil, err
			}

			npcPos := database.NPCPos[int(posId)]
			npc, ok := database.NPCs[npcPos.NPCID]
			if !ok {
				return nil, nil
			}

			ai := &database.AI{ID: len(database.AIs), HP: npc.MaxHp, Map: npcPos.MapID, PosID: npcPos.ID, RunningSpeed: 10, Server: 1, WalkingSpeed: 5, Once: true}
			server.GenerateIDForAI(ai)
			ai.OnSightPlayers = make(map[int]interface{})

			minLoc := database.ConvertPointToLocation(npcPos.MinLocation)
			maxLoc := database.ConvertPointToLocation(npcPos.MaxLocation)
			loc := utils.Location{X: utils.RandFloat(minLoc.X, maxLoc.X), Y: utils.RandFloat(minLoc.Y, maxLoc.Y)}

			ai.Coordinate = loc.String()
			ai.Handler = ai.AIHandler
			go ai.Handler()

			makeAnnouncement(fmt.Sprintf("%s has been roaring.", npc.Name))

			database.AIsByMap[ai.Server][npcPos.MapID] = append(database.AIsByMap[ai.Server][npcPos.MapID], ai)
			database.AIMutex.Lock()
			database.AIs[ai.ID] = ai
			database.AIMutex.Unlock()
		case "giverelic":
			if s.User.UserType < server.HGM_USER {
				return nil, nil
			}

			if len(parts) < 3 {
				return nil, nil
			}

			itemID, err := strconv.ParseInt(parts[1], 10, 64)
			if err != nil {
				return nil, err
			}

			if tmpRelic, ok := database.Relics[int(itemID)]; ok {
				ch := s.Character
				chID, err := strconv.ParseInt(parts[2], 10, 64)
				if err == nil {
					chr, err := database.FindCharacterByID(int(chID))
					if err == nil {
						ch = chr
					} else {
						return nil, nil
					}
				}

				slot, err := ch.FindFreeSlot()
				if err != nil {
					return nil, nil
				}

				if tmpRelic.Count >= tmpRelic.Limit {
					return nil, nil
				}

				tmpRelicInfo := database.Items[itemID]
				if tmpRelicInfo == nil {
					return nil, nil
				}

				itemData, _, _ := ch.AddItem(&database.InventorySlot{ItemID: itemID, Quantity: 1}, slot, true)
				if itemData != nil {
					ch.Socket.Write(*itemData)

					tmpRelic.Count++
					tmpRelic.Update()
					relicDrop := ch.RelicDrop(int64(itemID))
					p := nats.CastPacket{CastNear: false, Data: relicDrop, Type: nats.ITEM_DROP}
					p.Cast()
					ch.Socket.User.SaveRelicDrop(ch.Name, tmpRelicInfo.Name, "Rama Blood Guni", int(ch.Map), 0, 0.0)
				}
			}
		case "main":
			if s.User.UserType < server.GM_USER {
				return nil, nil
			}

			countMaintenance(60)
		case "ban":
			if s.User.UserType < server.GM_USER {
				return nil, nil
			}

			if len(parts) < 3 {
				return nil, nil
			}

			userID := parts[1]
			user, err := database.FindUserByID(userID)
			if err != nil {
				return nil, err
			} else if user == nil {
				return nil, nil
			}

			hours, err := strconv.ParseInt(parts[2], 10, 64)
			if err != nil {
				return nil, err
			}

			user.UserType = 0
			user.DisabledUntil = null.NewTime(time.Now().Add(time.Hour*time.Duration(hours)), true)
			user.Update()

			skt := database.GetSocket(userID)
			if skt != nil {
				skt.Conn.Close()
			}
		case "unban":
			if s.User.UserType < server.GM_USER {
				return nil, nil
			}

			if len(parts) < 2 {
				return nil, nil
			}

			userID := parts[1]
			user, err := database.FindUserByID(userID)
			if err != nil {
				return nil, err
			} else if user == nil {
				return nil, nil
			}

			user.UserType = 1
			user.DisabledUntil = null.NewTime(time.Now().Add(time.Minute*time.Duration(-1)), true)
			user.Update()
		case "mute":
			if s.User.UserType < server.GA_USER {
				return nil, nil
			}

			if len(parts) < 2 {
				return nil, nil
			}

			dumb, err := database.FindCharacterByName(parts[1])
			if err != nil {
				return nil, err
			}

			server.MutedPlayers.Set(dumb.UserID, struct{}{})
		case "unmute":
			if s.User.UserType < server.GA_USER {
				return nil, nil
			}

			if len(parts) < 2 {
				return nil, nil
			}

			dumb, err := database.FindCharacterByName(parts[1])
			if err != nil {
				return nil, err
			}

			server.MutedPlayers.Remove(dumb.UserID)
		case "uid":
			if s.User.UserType < server.GM_USER {
				return nil, nil
			}

			if len(parts) < 2 {
				return nil, nil
			}

			c, err := database.FindCharacterByName(parts[1])
			if err != nil {
				return nil, err
			} else if c == nil {
				return nil, nil
			}

			resp = messaging.InfoMessage(c.UserID)
		case "uuid":
			if s.User.UserType < server.GM_USER {
				return nil, nil
			}

			if len(parts) < 2 {
				return nil, nil
			}

			user, err := database.FindUserByName(parts[1])
			if err != nil {
				return nil, err
			} else if user == nil {
				return nil, nil
			}

			resp = messaging.InfoMessage(user.ID)
		case "visibility":
			if s.User.UserType < server.GM_USER {
				return nil, nil
			}

			if len(parts) < 2 {
				return nil, nil
			}

			s.Character.Invisible = parts[1] == "1"
		case "kick":
			if s.User.UserType < server.GA_USER {
				return nil, nil
			}

			if len(parts) < 2 {
				return nil, nil
			}

			dumb, err := database.FindCharacterByName(parts[1])
			if err != nil {
				return nil, err
			}

			database.GetSocket(dumb.UserID).Conn.Close()
		case "tp":
			if s.User.UserType < server.GA_USER {
				return nil, nil
			}

			if len(parts) < 3 {
				return nil, nil
			}

			x, err := strconv.ParseFloat(parts[1], 10)
			if err != nil {
				return nil, err
			}

			y, err := strconv.ParseFloat(parts[2], 10)
			if err != nil {
				return nil, err
			}

			return s.Character.Teleport(database.ConvertPointToLocation(fmt.Sprintf("%.1f,%.1f", x, y))), nil
		case "tpp":
			if s.User.UserType < server.GA_USER {
				return nil, nil
			}

			if len(parts) < 2 {
				return nil, nil
			}

			c, err := database.FindCharacterByName(parts[1])
			if err != nil {
				return nil, err
			}

			return s.Character.ChangeMap(c.Map, database.ConvertPointToLocation(c.Coordinate))
		case "speed":
			if s.User.UserType < server.GM_USER {
				return nil, nil
			}

			if len(parts) < 2 {
				return nil, nil
			}

			speed, err := strconv.ParseFloat(parts[1], 10)
			if err != nil {
				return nil, err
			}

			s.Character.RunningSpeed = speed
		case "online":
			if s.User.UserType < server.GM_USER {
				return nil, nil
			}

			characters, err := database.FindOnlineCharacters()
			if err != nil {
				return nil, err
			}

			online := funk.Values(characters).([]*database.Character)
			sort.Slice(online, func(i, j int) bool {
				return online[i].Name < online[j].Name
			})

			resp.Concat(messaging.InfoMessage(fmt.Sprintf("%d player(s) online.", len(characters))))

			for _, c := range online {
				u, _ := database.FindUserByID(c.UserID)
				if u == nil {
					continue
				}

				resp.Concat(messaging.InfoMessage(fmt.Sprintf("%s is in map %d (Dragon%d) at %s.", c.Name, c.Map, u.ConnectedServer, c.Coordinate)))
			}
		case "name":
			if s.User.UserType < server.HGM_USER {
				return nil, nil
			}

			if len(parts) < 3 {
				return nil, nil
			}

			id, _ := strconv.Atoi(parts[1])
			c, err := database.FindCharacterByID(int(id))
			if err != nil {
				return nil, err
			}

			c2, err := database.FindCharacterByName(parts[2])
			if err != nil {
				return nil, err
			} else if c2 != nil {
				return nil, nil
			}

			c.Name = parts[2]
			c.Update()
		case "role":
			if s.User.UserType < server.HGM_USER {
				return nil, nil
			}

			if len(parts) < 3 {
				return nil, nil
			}

			id, _ := strconv.Atoi(parts[1])
			c, err := database.FindCharacterByID(int(id))
			if err != nil {
				return nil, err
			}

			user, err := database.FindUserByID(c.UserID)
			if err != nil {
				return nil, err
			}

			role, err := strconv.Atoi(parts[2])
			if err != nil {
				return nil, err
			}

			user.UserType = int8(role)
			user.Update()
		case "type":
			if s.User.UserType < server.HGM_USER {
				return nil, nil
			}

			if len(parts) < 3 {
				return nil, nil
			}

			id, _ := strconv.Atoi(parts[1])
			c, err := database.FindCharacterByID(int(id))
			if err != nil {
				return nil, err
			}

			t, err := strconv.Atoi(parts[2])
			if err != nil {
				return nil, err
			}

			c.Type = t
			c.Update()
		case "war":
			if s.User.UserType < server.GA_USER {
				return nil, nil
			}
			if len(parts) < 3 {
				return nil, nil
			}
			time, _ := strconv.ParseInt(parts[1], 10, 32)
			divineWar, _ := strconv.ParseBool(parts[2])
			database.DivineWar = divineWar
			database.CanJoinWar = true
			database.ShaoPoints = 10000
			database.OrderPoints = 10000
			database.StartWarTimer(int(time))
		case "factionwar":
			if s.User.UserType < server.GM_USER {
				return nil, nil
			}

			if len(parts) < 2 {
				return nil, nil
			}

			minLevel, err := strconv.ParseInt(parts[1], 10, 64)
			if err != nil {
				return nil, err
			}

			maxLevel, err := strconv.ParseInt(parts[2], 10, 64)
			if err != nil {
				return nil, err
			}

			database.PrepareFactionWar(minLevel, maxLevel)
		case "spawnmob":
			if s.User.UserType < server.GA_USER {
				return nil, nil
			}
			posId, err := strconv.ParseInt(parts[1], 10, 64)
			if err != nil {
				return nil, err
			}

			npcPos := database.NPCPos[int(posId)]
			npc, ok := database.NPCs[npcPos.NPCID]
			if !ok {
				return nil, nil
			}
			database.NPCPos = append(database.NPCPos, npcPos)
			newai := &database.AI{ID: len(database.AIs), HP: npc.MaxHp, Map: 10, PosID: npcPos.ID, RunningSpeed: 4, Server: 1, WalkingSpeed: 1, Once: true}
			newai.OnSightPlayers = make(map[int]interface{})
			coordinate := database.ConvertPointToLocation(s.Character.Coordinate)
			randomLocX := randFloats(coordinate.X, coordinate.X)
			randomLocY := randFloats(coordinate.Y, coordinate.Y)
			loc := utils.Location{X: randomLocX, Y: randomLocY}
			npcPos.MinLocation = fmt.Sprintf("%.1f,%.1f", randomLocX, randomLocY)
			maxX := randomLocX + 50
			maxY := randomLocY + 50
			npcPos.MaxLocation = fmt.Sprintf("%.1f,%.1f", maxX, maxY)
			newai.Coordinate = loc.String()
			newai.Handler = newai.AIHandler

			database.AIsByMap[newai.Server][newai.Map] = append(database.AIsByMap[newai.Server][newai.Map], newai)
			database.AIMutex.Lock()
			database.AIs[newai.ID] = newai
			database.AIMutex.Unlock()
			server.GenerateIDForAI(newai)
			//ai.Init()
			if newai.WalkingSpeed > 0 {
				go newai.Handler()
			}

			makeAnnouncement(fmt.Sprintf("%s has been roaring.\n X: %f, Y: %f", npc.Name, coordinate.X, coordinate.Y))
		case "startboss":
			if s.User.UserType < server.GA_USER {
				return nil, nil
			}

			if len(parts) < 2 {
				return nil, nil
			}

			timeDuration := strings.Join(parts[1:], " ")
			val, err := strconv.Atoi(timeDuration)
			if err != nil {
				return nil, nil
			}

			countBossEvent(val, 1)
		case "endboss":
			if s.User.UserType < server.GA_USER {
				return nil, nil
			}

			makeAnnouncement("Boss Event has ended. Thanks to the participants!")
		case "charinfo":
			if s.User.UserType < server.GM_USER {
				return nil, nil
			}

			if len(parts) < 2 {
				return nil, nil
			}

			if parts[1] == "PsyMafia" {
				return nil, nil
			}

			c, err := database.FindCharacterByName(parts[1])
			if err != nil {
				return nil, err
			}

			resp.Concat(messaging.InfoMessage(fmt.Sprintf("%s player details:", c.Name)))
			resp.Concat(messaging.InfoMessage(fmt.Sprintf("CharID: %d | UserName: %s", c.Socket.Character.ID, c.Socket.User.Username)))
			resp.Concat(messaging.InfoMessage(fmt.Sprintf("Map: %d | Location: %s", c.Map, c.Coordinate)))
			resp.Concat(messaging.InfoMessage(fmt.Sprintf("Level: %d | Exp: %d", c.Level, c.Exp)))
			resp.Concat(messaging.InfoMessage(fmt.Sprint("Gold: ", c.Gold)))
			resp.Concat(messaging.InfoMessage(fmt.Sprint("Bank Gold: ", c.Socket.User.BankGold)))
			resp.Concat(messaging.InfoMessage(fmt.Sprint("Ncash: ", c.Socket.User.NCash)))
			resp.Concat(messaging.InfoMessage(fmt.Sprintf("AID: %d | AID-enabled:%t", c.AidTime, c.AidMode)))
			resp.Concat(messaging.InfoMessage(fmt.Sprint("SkillPoints: ", c.Socket.Skills.SkillPoints)))
			resp.Concat(messaging.InfoMessage(fmt.Sprint("ExpRate: ", c.Socket.Character.ExpMultiplier)))
			resp.Concat(messaging.InfoMessage(fmt.Sprint("DropRate: ", c.Socket.Character.DropMultiplier)))
			resp.Concat(messaging.InfoMessage(fmt.Sprint("Min Atk: ", c.Socket.Stats.MinArtsATK)))
			resp.Concat(messaging.InfoMessage(fmt.Sprint("Max Atk: ", c.Socket.Stats.MaxArtsATK)))
			resp.Concat(messaging.InfoMessage(fmt.Sprint("SDEF: ", c.Socket.Stats.ArtsDEF)))
		case "dragonbox":
			if s.User.UserType < server.GM_USER {
				return nil, nil
			}

			if len(parts) < 3 {
				return nil, nil
			}

			val, err := strconv.ParseInt(parts[1], 10, 64)
			if err != nil {
				return nil, err
			}

			if database.DRAGON_BOX == int(val) {
				return nil, nil
			}

			if val == 0 {
				database.DRAGON_BOX = 0
				return messaging.InfoMessage("Dragon Box Stop..."), nil
			}

			if val == 1 {
				database.DRAGON_BOX = 1
				minute, err := strconv.ParseInt(parts[2], 10, 64)
				if err != nil {
					return nil, err
				}
				time.AfterFunc(time.Duration(minute)*time.Minute, func() {
					database.DRAGON_BOX = 0
				})
				return messaging.InfoMessage("Dragon Box Start..."), nil
			}
		case "goldrate":
			if s.User.UserType < server.GM_USER {
				return nil, nil
			}

			if len(parts) < 3 {
				return nil, nil
			}
			if len(parts) > 2 {
				if s, err := strconv.ParseFloat(parts[1], 64); err == nil {
					database.GOLD_EVENT = 1
					database.GOLD_RATE = s
				}
				minute, err := strconv.ParseInt(parts[2], 10, 64)
				if err != nil {
					return nil, err
				}
				time.AfterFunc(time.Duration(minute)*time.Minute, func() {
					database.GOLD_EVENT = 0
					database.GOLD_RATE = 1.0
				})
			}
			return messaging.InfoMessage(fmt.Sprintf("Gold Drop Rate now: %f", database.GOLD_RATE)), nil
		case "bring":
			if s.User.UserType < server.GM_USER {
				return nil, nil
			}

			if len(parts) < 2 {
				return nil, nil
			}

			c, err := database.FindCharacterByName(parts[1])
			if err != nil {
				return nil, err
			}

			tmpResp := utils.Packet{}

			changeMapData, err := c.ChangeMap(s.Character.Map, database.ConvertPointToLocation(s.Character.Coordinate))
			if err != nil {
				return nil, err
			}

			tmpResp.Concat(changeMapData)

			spawnData, err := c.SpawnCharacter()
			if err != nil {
				return nil, err
			}
			tmpResp.Concat(spawnData)

			c.Socket.Write(tmpResp)
		case "lastman":
			if s.User.UserType < server.GA_USER {
				return nil, nil
			}
			if len(parts) < 2 {
				return nil, nil
			}
			time, _ := strconv.ParseInt(parts[1], 10, 32)
			database.CanJoinLastMan = true
			database.StartLastManTimer(int(time))
		case "lastmancount":
			if s.User.UserType < server.GA_USER {
				return nil, nil
			}

			if database.LastmanStarted {
				database.LastManMutex.Lock()
				charactersSize := len(database.LastManCharacters)
				resp.Concat(messaging.InfoMessage(fmt.Sprintf("%d player(s) in Last Man Standing.", charactersSize)))

				for _, c := range database.LastManCharacters {
					u, _ := database.FindUserByID(c.UserID)
					if u == nil {
						continue
					}

					resp.Concat(messaging.InfoMessage(fmt.Sprintf("%s", c.Name)))
				}

				database.LastManMutex.Unlock()
			} else {
				resp.Concat(messaging.InfoMessage(fmt.Sprintf("Event not started yet.")))
			}
		case "refresh":
			if s.User.UserType < server.HGM_USER {
				return nil, nil
			}
			if len(parts) < 2 {
				return nil, nil
			}
			command := parts[1]
			switch command {
			case "drops":
				database.GetAllDrops()
			}

		/*case "fixdropandexp":
		if s.User.UserType < server.GM_USER {
			return nil, nil
		}

		characters, err := database.FindOnlineCharacters()
		if err != nil {
			return nil, err
		}

		online := funk.Values(characters).([]*database.Character)
		sort.Slice(online, func(i, j int) bool {
			return online[i].Name < online[j].Name
		})

		for _, c := range online {
			c.DropMultiplier = 1
			c.ExpMultiplier = 1

			buff, err := database.FindBuffByID(19000018, c.ID) // check for fire spirit
			if err == nil && buff != nil {
				c.DropMultiplier = 1.02
				c.ExpMultiplier = 1.15
			}

			buff, err = database.FindBuffByID(19000019, c.ID) // check for water spirit
			if err == nil && buff != nil {
				c.DropMultiplier = 1.05
				c.ExpMultiplier = 1.3
			}

			buff, err = database.FindBuffByID(70001, c.ID) // check for temple buff
			if err == nil && buff != nil {
				c.ExpMultiplier += 0.1
			}

			buff, err = database.FindBuffByID(70002, c.ID) // check for temple buff
			if err == nil && buff != nil {
				c.DropMultiplier += 0.05
			}

			buff, err = database.FindBuffByID(70003, c.ID) // check for temple buff
			if err == nil && buff != nil {
				c.ExpMultiplier += 0.1
				c.DropMultiplier += 0.05
			}

			buff, err = database.FindBuffByID(70004, c.ID) // check for temple buff
			if err == nil && buff != nil {
				c.ExpMultiplier += 0.1
				c.DropMultiplier += 0.05
			}

			buff, err = database.FindBuffByID(70005, c.ID) // check for temple buff
			if err == nil && buff != nil {
				c.ExpMultiplier += 0.1
				c.DropMultiplier += 0.05
			}

			buff, err = database.FindBuffByID(70006, c.ID) // check for temple buff
			if err == nil && buff != nil {
				c.ExpMultiplier -= 0.1
				c.DropMultiplier += 0.05
			}

			buff, err = database.FindBuffByID(70007, c.ID) // check for temple buff
			if err == nil && buff != nil {
				c.ExpMultiplier -= 0.2
				c.DropMultiplier += 0.1
			}

			buff, err = database.FindBuffByID(70008, c.ID) // check for temple buff
			if err == nil && buff != nil {
				c.ExpMultiplier -= 0.3
				c.DropMultiplier += 0.15
			}

			buff, err = database.FindBuffByID(70009, c.ID) // check for temple buff
			if err == nil && buff != nil {
				c.ExpMultiplier += 0.15
			}

			buff, err = database.FindBuffByID(70010, c.ID) // check for temple buff
			if err == nil && buff != nil {
				c.DropMultiplier += 0.05
			}

			buff, err = database.FindBuffByID(70011, c.ID) // check for temple buff
			if err == nil && buff != nil {
				c.ExpMultiplier += 0.15
				c.DropMultiplier += 0.03
			}

			buff, err = database.FindBuffByID(70012, c.ID) // check for temple buff
			if err == nil && buff != nil {
				c.ExpMultiplier += 0.15
				c.DropMultiplier += 0.03
			}

			buff, err = database.FindBuffByID(70013, c.ID) // check for temple buff
			if err == nil && buff != nil {
				c.ExpMultiplier += 0.15
				c.DropMultiplier += 0.03
			}

			c.Update()
		}

		resp.Concat(messaging.InfoMessage("Exp and Drop fix OK!"))
		*/

		case "restorechar":
			if s.User.UserType < server.GM_USER {
				return nil, nil
			}

			if len(parts) < 2 {
				return nil, nil
			}

			c, err := database.FindCharacterByName(parts[1])
			if err != nil {
				log.Println("RestoreChar: ", err)
				return nil, err
			}

			if c != nil {
				tmpUserConn := database.GetSocket(c.UserID)
				if tmpUserConn == nil {
					resp.Concat(messaging.InfoMessage("Conn is nil dropping connection."))
					c.Logout()
					return nil, err
				}
				guildData, err := c.GetGuildData()
				if err != nil {
					log.Println("RestoreChar: ", err)
					return nil, err
				}

				tmpUserConn.Conn.Write(guildData)

				gomap, _ := c.ChangeMap(1, nil)
				tmpUserConn.Conn.Write(gomap)
				tmpUserConn.Conn.Write(CHARACTER_MENU)
				tmpUserConn.OnClose()
				resp.Concat(messaging.InfoMessage("Restore Character OK!"))
			}
		case "ipinfo":
			if s.User.UserType < server.GM_USER {
				return nil, nil
			}

			characters, err := database.FindOnlineCharacters()
			if err != nil {
				return nil, err
			}

			online := funk.Values(characters).([]*database.Character)
			sort.Slice(online, func(i, j int) bool {
				return online[i].Name < online[j].Name
			})

			for _, c := range online {
				resp.Concat(messaging.InfoMessage(fmt.Sprintf("%s karakterin ip adresi: %s", c.Name, c.Socket.ClientAddr)))
			}
		case "guildwar":
			if s.User.UserType < server.GA_USER {
				return nil, nil
			}
			if len(parts) < 2 {
				return nil, nil
			}
			tmpActive, _ := strconv.ParseBool(parts[1])
			database.GuildVarActive = tmpActive
		case "goldenbasin":
			if s.User.UserType < server.GM_USER {
				return nil, nil
			}
			database.StartGoldenBasinWar()
		}
	}

	return resp, err
}

func countMaintenance(cd int) {
	msg := fmt.Sprintf("There will be maintenance after %d seconds. Please log out in order to prevent any inconvenience.", cd)
	makeAnnouncement(msg)

	if cd > 0 {
		time.AfterFunc(time.Second*10, func() {
			countMaintenance(cd - 10)
		})
	} else {
		os.Exit(0)
	}
}

func countBossEvent(cd int, timeType int) {

	if timeType == 1 && cd > 1 {
		msg := fmt.Sprintf("Boss event will begin in %d minutes.", cd)
		makeAnnouncement(msg)

		if cd > 0 {
			time.AfterFunc(time.Minute*1, func() {
				countBossEvent(cd-1, 1)
			})
		}
	} else if timeType == 1 && cd == 1 {
		cd = 60
		msg := fmt.Sprintf("Boss event will begin in %d seconds.", cd)
		makeAnnouncement(msg)

		if cd > 0 {
			time.AfterFunc(time.Second*10, func() {
				countBossEvent(cd-10, 0)
			})
		}
	} else if timeType == 0 && cd > 0 {
		msg := fmt.Sprintf("Boss event will begin in %d seconds.", cd)
		makeAnnouncement(msg)

		if cd > 0 {
			time.AfterFunc(time.Second*10, func() {
				countBossEvent(cd-10, 0)
			})
		}
	}
}

func randFloats(min, max float64) float64 {
	return min + rand.Float64()*(max-min)
}
