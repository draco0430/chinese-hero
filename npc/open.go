package npc

import (
	"encoding/binary"
	"fmt"
	"strings"
	"time"

	"hero-server/database"
	"hero-server/dungeon"
	"hero-server/messaging"
	"hero-server/nats"
	"hero-server/utils"

	"github.com/thoas/go-funk"
	"github.com/tidwall/gjson"
)

type OpenHandler struct {
}

type PressButtonHandler struct {
}

var (
	shops = map[int]int{20002: 7, 20003: 2, 20004: 4, 20005: 1, 20009: 8, 20010: 10, 20011: 10, 20013: 25, 20160: 340,
		20024: 6, 20025: 6, 20026: 11, 20033: 21, 20034: 22, 20035: 23, 20036: 24, 20044: 21, 20047: 21, 20082: 21,
		20083: 21, 20084: 21, 20085: 23, 20086: 22, 20087: 21, 20094: 103, 20095: 100, 20105: 21, 20133: 21,
		20146: 21, 20151: 6, 20173: 342, 20211: 25, 20239: 21, 20415: 21, 20015: 340, 20202: 341, 20206: 11, 20203: 11,
		20316: 11, 20323: 11, 20337: 11, 20413: 25, 20379: 25, 20204: 344, 20317: 343, 20051: 345, 20293: 346, 20295: 346,
		23714: 347, 23741: 348, 23747: 349, 20088: 350, 20169: 351,
	}

	COMPOSITION_MENU          = utils.Packet{0xAA, 0x55, 0x03, 0x00, 0x57, 0x0F, 0x01, 0x55, 0xAA}
	OPEN_SHOP                 = utils.Packet{0xAA, 0x55, 0x07, 0x00, 0x57, 0x03, 0x01, 0x55, 0xAA}
	NPC_MENU                  = utils.Packet{0xAA, 0x55, 0x00, 0x00, 0x57, 0x02, 0x55, 0xAA}
	STRENGTHEN_MENU           = utils.Packet{0xAA, 0x55, 0x03, 0x00, 0x57, 0x08, 0x01, 0x55, 0xAA}
	JOB_PROMOTED              = utils.Packet{0xAA, 0x55, 0x03, 0x00, 0x57, 0x09, 0x00, 0x55, 0xAA}
	NOT_ENOUGH_LEVEL          = utils.Packet{0xAA, 0x55, 0x0B, 0x00, 0x57, 0x02, 0x38, 0x42, 0x0F, 0x00, 0x00, 0x55, 0xAA}
	INVALID_CLASS             = utils.Packet{0xAA, 0x55, 0x0B, 0x00, 0x57, 0x02, 0x49, 0x2F, 0x00, 0x00, 0x00, 0x55, 0xAA}
	GUILD_MENU                = utils.Packet{0xAA, 0x55, 0x02, 0x00, 0x57, 0x0D, 0x55, 0xAA}
	DISMANTLE_MENU            = utils.Packet{0xAA, 0x55, 0x03, 0x00, 0x57, 0x16, 0x01, 0x55, 0xAA}
	EXTRACTION_MENU           = utils.Packet{0xAA, 0x55, 0x03, 0x00, 0x57, 0x17, 0x01, 0x55, 0xAA}
	ADV_FUSION_MENU           = utils.Packet{0xAA, 0x55, 0x03, 0x00, 0x57, 0x32, 0x01, 0x55, 0xAA}
	TACTICAL_SPACE            = utils.Packet{0xAA, 0x55, 0x03, 0x00, 0x50, 0x01, 0x01, 0x55, 0xAA}
	CREATE_SOCKET_MENU        = utils.Packet{0xAA, 0x55, 0x03, 0x00, 0x57, 0x39, 0x01, 0x55, 0xAA}
	UPGRADE_SOCKET_MENU       = utils.Packet{0xAA, 0x55, 0x03, 0x00, 0x57, 0x3A, 0x01, 0x55, 0xAA}
	CONSIGNMENT_MENU          = utils.Packet{0xAA, 0x55, 0x03, 0x00, 0x57, 0x42, 0x01, 0x55, 0xAA}
	APPEARANCE_MENU           = utils.Packet{0xAA, 0x55, 0x03, 0x00, 0x57, 0x41, 0x01, 0x55, 0xAA}
	APPEARANCE_REMOVE_MENU    = utils.Packet{0xAA, 0x55, 0x03, 0x00, 0x57, 0x43, 0x01, 0x55, 0xAA}
	ENHANCEMENT_TRANSFER_MENU = utils.Packet{0xAA, 0x55, 0x03, 0x00, 0x57, 0x38, 0x01, 0x55, 0xAA}
)

func (h *OpenHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {

	c := s.Character
	if c == nil {
		return nil, nil
	}

	u := s.User
	if u == nil {
		return nil, nil
	}

	id := uint16(utils.BytesToInt(data[6:10], true))
	pos, ok := database.GetFromRegister(1, c.Map, id).(*database.NpcPosition)
	if !ok {
		return nil, nil
	}

	npc := database.NPCs[pos.NPCID]

	if npc.ID == 20147 { // Ice Palace Mistress Lord
		coordinate := &utils.Location{X: 163, Y: 350}
		return c.Teleport(coordinate), nil

	} else if npc.ID == 20055 { // Mysterious Tombstone
		coordinate := &utils.Location{X: 365, Y: 477}
		return c.Teleport(coordinate), nil

	} else if npc.ID == 20056 { // Mysterious Tombstone (R)
		coordinate := &utils.Location{X: 70, Y: 450}
		return c.Teleport(coordinate), nil

	} else if npc.ID == 22357 { // 2nd FL Entrance
		return c.ChangeMap(237, nil)

	} else if npc.ID == 22358 { // 3rd FL Entrance
		return c.ChangeMap(239, nil)
	} else if npc.ID == 20270 { // Hulma stone
		coordinate := &utils.Location{X: 113, Y: 227}
		return c.Teleport(coordinate), nil
	} else if npc.ID == 20322 { // Yuchun
		if c.Level > 159 {
			return c.ChangeMap(30, nil)
		}
	} else if npc.ID == 18600081 {
		// Shao
		x := 255.0
		y := 219.0
		tmpData := c.Teleport(database.ConvertPointToLocation(fmt.Sprintf("%.1f,%.1f", x, y)))
		return tmpData, nil
	} else if npc.ID == 18600082 {
		// Zhuang
		x := 251.0
		y := 299.0
		tmpData := c.Teleport(database.ConvertPointToLocation(fmt.Sprintf("%.1f,%.1f", x, y)))
		return tmpData, nil
	}

	/*

		else if npc.ID == 22351 { // Golden Castle Teleport Tombstone
			return c.ChangeMap(236, nil)

		}
	*/

	npcScript := database.NPCScripts[npc.ID]
	if npcScript == nil {
		return nil, nil
	}

	script := string(npcScript.Script)
	textID := gjson.Get(script, "text").Int()
	actions := []int{}

	for _, action := range gjson.Get(script, "actions").Array() {
		actions = append(actions, int(action.Int()))
	}

	resp := NPC_MENU
	resp.Insert(utils.IntToBytes(uint64(npc.ID), 4, true), 6)        // npc id
	resp.Insert(utils.IntToBytes(uint64(textID), 4, true), 10)       // text id
	resp.Insert(utils.IntToBytes(uint64(len(actions)), 1, true), 14) // action length

	index, length := 15, int16(11)
	for i, action := range actions {
		resp.Insert(utils.IntToBytes(uint64(action), 4, true), index) // action
		index += 4

		resp.Insert(utils.IntToBytes(uint64(npc.ID), 2, true), index) // npc id
		index += 2

		resp.Insert(utils.IntToBytes(uint64(i+1), 2, true), index) // action index
		index += 2

		length += 8
	}

	resp.SetLength(length)
	return resp, nil
}

func (h *PressButtonHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {

	c := s.Character
	if c == nil {
		return nil, nil
	}

	npcID := int(utils.BytesToInt(data[6:8], true))
	index := int(utils.BytesToInt(data[8:10], true))
	indexes := []int{index & 7, (index & 56) / 8, (index & 448) / 64, (index & 3584) / 512, (index & 28672) / 4096}
	indexes = funk.FilterInt(indexes, func(i int) bool {
		return i > 0
	})

	npcScript := database.NPCScripts[npcID]
	if npcScript == nil {
		return nil, nil
	}

	script := string(npcScript.Script)
	key := strings.Trim(strings.Join(strings.Fields(fmt.Sprint(indexes)), "."), "[]")

	script = gjson.Get(script, key).String()
	if script != "" {
		textID := int(gjson.Get(script, fmt.Sprintf("text")).Int())
		actions := []int{}

		for _, action := range gjson.Get(script, "actions").Array() {
			actions = append(actions, int(action.Int()))
		}

		resp := GetNPCMenu(npcID, textID, index, actions)
		return resp, nil
	} else { // Action button
		key := strings.Trim(strings.Join(strings.Fields(fmt.Sprint(indexes[:len(indexes)-1])), "."), "[]")
		script = string(npcScript.Script)
		if key != "" {
			script = gjson.Get(script, key).String()
		}

		actions := gjson.Get(script, "actions").Array()
		actIndex := indexes[len(indexes)-1] - 1
		actID := actions[actIndex].Int()

		resp := utils.Packet{}

		var err error
		book1, book2, job := 0, 0, 0
		switch actID {
		case 1: // Exchange
			shopNo := shops[npcID]
			resp = OPEN_SHOP
			resp.Insert(utils.IntToBytes(uint64(shopNo), 4, true), 7) // shop id

		case 2: // Compositon
			resp = COMPOSITION_MENU

		case 4: // Strengthen
			resp = STRENGTHEN_MENU

		case 6: // Deposit
			resp = c.BankItems()

		case 13: // Accept
			switch npcID {
			case 20006: // Hunter trainer
				book1, job = 16210003, 13
				resp, err = firstJobPromotion(c, book1, job, npcID)
				if err != nil {
					return nil, err
				}
			case 20020: // Warrior trainer
				book1, job = 16210001, 11
				resp, err = firstJobPromotion(c, book1, job, npcID)
				if err != nil {
					return nil, err
				}
			case 20021: // Physician trainer
				book1, job = 16210002, 12
				resp, err = firstJobPromotion(c, book1, job, npcID)
				if err != nil {
					return nil, err
				}
			case 20022: // Assassin trainer
				book1, job = 16210004, 14
				resp, err = firstJobPromotion(c, book1, job, npcID)
				if err != nil {
					return nil, err
				}
			case 20057: //HERO BATTLE MANAGER
				switch index {
				case 11: //THE GREAT WAR
					if database.CanJoinWar {
						if database.DivineWar {
							if c.Level > 100 {
								if database.DivineWar {
									if c.Faction == 1 {
										x := 75.0
										y := 45.0
										c.IsinWar = true
										database.OrderCharacters[c.ID] = c
										data, _ := c.ChangeMap(230, database.ConvertPointToLocation(fmt.Sprintf("%.1f,%.1f", x, y)))
										c.Socket.Write(data)
									} else {
										x := 81.0
										y := 475.0
										c.IsinWar = true
										database.ShaoCharacters[c.ID] = c
										data, _ := c.ChangeMap(230, database.ConvertPointToLocation(fmt.Sprintf("%.1f,%.1f", x, y)))
										c.Socket.Write(data)
									}
								}
							} else {
								resp.Concat(messaging.InfoMessage(fmt.Sprintf("You are not Divine")))
							}
						} else {
							if c.Level >= 50 && c.Level <= 100 {
								if c.Faction == 1 {
									x := 75.0
									y := 45.0
									c.IsinWar = true
									database.OrderCharacters[c.ID] = c
									data, _ := c.ChangeMap(230, database.ConvertPointToLocation(fmt.Sprintf("%.1f,%.1f", x, y)))
									c.Socket.Write(data)
								} else {
									x := 81.0
									y := 475.0
									c.IsinWar = true
									database.ShaoCharacters[c.ID] = c
									data, _ := c.ChangeMap(230, database.ConvertPointToLocation(fmt.Sprintf("%.1f,%.1f", x, y)))
									c.Socket.Write(data)
								}
							} else {
								resp.Concat(messaging.InfoMessage(fmt.Sprintf("You are not non Divine")))
							}
						}
					}
				case 10: //FACTION WAR
					//database.AddMemberToFactionWar(c)
					//resp, _ = c.ChangeMap(255)
				case 9: //FLAG KINGDOM
				}
			case 20415: // RDL tavern
				resp, _ = c.ChangeMap(254, nil)
			case 20267:
				coordinate := &utils.Location{X: 325, Y: 171}
				return c.Teleport(coordinate), nil

			}

		case 64: // Create Guild
			if c.GuildID == -1 {
				resp = GUILD_MENU
			}

		case 77: // Move to Souther Plains
			resp, _ = c.ChangeMap(7, nil)

		case 78: // Move to Dragon Castle
			resp, _ = c.ChangeMap(1, nil)

		case 86: // Move to Spirit Spire
			resp, _ = c.ChangeMap(5, nil)

		case 103: // Move to Highlands
			resp, _ = c.ChangeMap(2, nil)

		case 104: // Move to Venom Swamp
			resp, _ = c.ChangeMap(3, nil)
		case 106: // Move to Silent Valley
			resp, _ = c.ChangeMap(11, nil)
		case 148: // Become a Champion
			book1, book2, job = 16100039, 16100200, 21
			resp, err = secondJobPromotion(c, book1, book2, 11, job, npcID)
			if err != nil {
				return nil, err
			}
		case 149: // Become a Musa
			book1, book2, job = 16100040, 16100200, 22
			resp, err = secondJobPromotion(c, book1, book2, 11, job, npcID)
			if err != nil {
				return nil, err
			}
		case 151: // Become a Surgeon
			book1, book2, job = 16100041, 16100200, 23
			resp, err = secondJobPromotion(c, book1, book2, 12, job, npcID)
			if err != nil {
				return nil, err
			}
		case 152: // Become a Combat Medic
			book1, book2, job = 16100042, 16100200, 24
			resp, err = secondJobPromotion(c, book1, book2, 12, job, npcID)
			if err != nil {
				return nil, err
			}
		case 154: // Become a Slayer
			book1, book2, job = 16100043, 16100200, 27
			resp, err = secondJobPromotion(c, book1, book2, 14, job, npcID)
			if err != nil {
				return nil, err
			}
		case 155: // Become a Shinobi
			book1, book2, job = 16100044, 16100200, 28
			resp, err = secondJobPromotion(c, book1, book2, 14, job, npcID)
			if err != nil {
				return nil, err
			}
		case 157: // Become a Tracker
			book1, book2, job = 16100045, 16100200, 25
			resp, err = secondJobPromotion(c, book1, book2, 13, job, npcID)
			if err != nil {
				return nil, err
			}
		case 158: // Become a Ranger
			book1, book2, job = 16100046, 16100200, 26
			resp, err = secondJobPromotion(c, book1, book2, 13, job, npcID)
			if err != nil {
				return nil, err
			}

		case 194: // Dismantle
			resp = DISMANTLE_MENU

		case 116:
			database.AddMemberToFactionWar(c)
			resp, _ = c.ChangeMap(255, nil)

		case 195: // Extraction
			resp = EXTRACTION_MENU
		case 207: // Bounty
			fmt.Println("Bounty isteği geldi.")

		case 524: // Exit Paid Zone
			if maps, ok := database.DKMaps[c.Map]; ok {
				resp, err = c.ChangeMap(maps[0], nil)
				if err != nil {
					return nil, err
				}
			}

		case 525: // Enter Paid Zone
			f := func(item *database.InventorySlot) bool {
				return item.Activated
			}
			_, item, err := c.FindItemInInventory(f, 15700040, 15710087, 17200452)
			if err != nil {
				return nil, err
			} else if item == nil { // You don't have ticket
				resp := GetNPCMenu(npcID, 999993, 0, nil)
				return resp, nil
			}

			if maps, ok := database.DKMaps[c.Map]; ok {
				resp, err = c.ChangeMap(maps[1], nil)
				if err != nil {
					return nil, err
				}
			}

		case 559: // Advanced Fusion
			resp = ADV_FUSION_MENU

		case 631: // Tactical Space
			resp = TACTICAL_SPACE

		case 706:
			return ENHANCEMENT_TRANSFER_MENU, nil

		case 732: // Flexible Castle Entry
			f := func(item *database.InventorySlot) bool {
				return item.Activated
			}
			_, item, err := c.FindItemInInventory(f, 15710087, 17200452)
			if err != nil {
				return nil, err
			} else if item == nil { // You don't have ticket
				resp := GetNPCMenu(npcID, 999993, 0, nil)
				return resp, nil
			}

			if maps, ok := database.DKMaps[c.Map]; ok {
				resp, err = c.ChangeMap(maps[2], nil)
				if err != nil {
					return nil, err
				}
			}

		case 737: // Create Socket
			resp = CREATE_SOCKET_MENU

		case 738: // Upgrade Socket
			resp = UPGRADE_SOCKET_MENU

		case 208: //APPEARANCE CHANGE
			resp = APPEARANCE_MENU

		case 985: //APPEARANCE CHANGE
			resp = APPEARANCE_REMOVE_MENU

		case 970: // Consignment
			resp = CONSIGNMENT_MENU

		/*
			case 3306:
				if s.User.GetDailyAid() {
					itemData, _, err := c.AddItem(&database.InventorySlot{ItemID: 18500032, Quantity: 1}, -1, true)
					if err != nil {
						return nil, err
					}
					resp.Concat(*itemData)
					resp.Concat(c.GetGold())
				}
		*/
		case 3307:

			//c.Socket.Write(messaging.InfoMessage("New check-in event coming soon."))
			//return nil, nil

			freeSlotID, err := c.FindFreeSlot()
			if freeSlotID == 0 || err != nil {
				c.Socket.Write(messaging.InfoMessage("Your inventory is full."))
				return nil, nil
			}

			dayCount, ok := s.User.DailyCheck()
			if ok {
				tmpItem := &database.InventorySlot{}
				//tmpItem.ItemID =
				//tmpItem.Quantity =
				if dayCount == 0 {
					tmpItem.ItemID = 15710613
					tmpItem.Quantity = 1

				}
				if dayCount == 1 {
					tmpItem.ItemID = 17502658
					tmpItem.Quantity = 1
				}
				if dayCount == 2 {
					tmpItem.ItemID = 13370178
					tmpItem.Quantity = 1
				}
				if dayCount == 3 {
					tmpItem.ItemID = 13000173
					tmpItem.Quantity = 1
				}
				if dayCount == 4 {
					tmpItem.ItemID = 15710569
					tmpItem.Quantity = 1
				}
				if dayCount == 5 {
					tmpItem.ItemID = 15710018
					tmpItem.Quantity = 300
				}
				if dayCount == 6 {
					tmpItem.ItemID = 13370165
					tmpItem.Quantity = 1
				}
				if dayCount == 7 {
					tmpItem.ItemID = 15710002
					tmpItem.Quantity = 1
				}
				if dayCount == 8 {
					tmpItem.ItemID = 13000173
					tmpItem.Quantity = 1
				}
				if dayCount == 9 {
					tmpItem.ItemID = 15710269
					tmpItem.Quantity = 1440
				}
				if dayCount == 10 {
					tmpItem.ItemID = 15710273
					tmpItem.Quantity = 1440
				}
				if dayCount == 11 {
					tmpItem.ItemID = 99002838
					tmpItem.Quantity = 1
				}
				if dayCount == 12 {
					tmpItem.ItemID = 99002838
					tmpItem.Quantity = 1
				}
				if dayCount == 13 {
					tmpItem.ItemID = 99002838
					tmpItem.Quantity = 1
				}
				if dayCount == 14 {
					tmpItem.ItemID = 13000134
					tmpItem.Quantity = 1
				}
				if dayCount == 15 {
					tmpItem.ItemID = 17200185
					tmpItem.Quantity = 1
				}
				if dayCount == 16 {
					tmpItem.ItemID = 17200186
					tmpItem.Quantity = 1
				}
				if dayCount == 17 {
					tmpItem.ItemID = 13370163
					tmpItem.Quantity = 1
				}
				if dayCount == 18 {
					tmpItem.ItemID = 17300006
					tmpItem.Quantity = 500
				}
				if dayCount == 19 {
					tmpItem.ItemID = 17300004
					tmpItem.Quantity = 500
				}
				if dayCount == 20 {
					tmpItem.ItemID = 13000173
					tmpItem.Quantity = 1
				}
				if dayCount == 21 {
					tmpItem.ItemID = 99002838
					tmpItem.Quantity = 1
				}
				if dayCount == 22 {
					tmpItem.ItemID = 15710018
					tmpItem.Quantity = 300
				}
				if dayCount == 23 {
					tmpItem.ItemID = 100080299
					tmpItem.Quantity = 5
				}
				if dayCount == 24 {
					tmpItem.ItemID = 13000173
					tmpItem.Quantity = 1
				}
				if dayCount == 25 {
					tmpItem.ItemID = 15710239
					tmpItem.Quantity = 1
				}
				if dayCount == 26 {
					tmpItem.ItemID = 15710096
					tmpItem.Quantity = 1
				}
				if dayCount == 27 {
					tmpItem.ItemID = 15710001
					tmpItem.Quantity = 1
				}
				itemData, _, err := c.AddItem(tmpItem, freeSlotID, false)
				if err != nil {
					fmt.Println(err)
					return nil, err
				}
				resp.Concat(*itemData)
				resp.Concat(c.GetGold())
			} else {
				c.Socket.Write(messaging.InfoMessage(c.Name + ", you already checked in for today."))
				return nil, nil
			}

		case 1337: // Move to new map
			if c.Level >= 85 {
				resp, _ = c.ChangeMap(42, nil)
			}

		case 1338: // Move to coin map
			return nil, nil
		/*
			if c.Level >= 80 {
				resp, _ = c.ChangeMap(44, nil)
			}
		*/
		case 1339: // Move to dragon box map
			return nil, nil
		/*
			if c.Level >= 80 {
				resp, _ = c.ChangeMap(48, nil)
			}
		*/
		case 1340:

			// DİVİNE

			//resp.Concat(messaging.InfoMessage(fmt.Sprintf("Divine is not available yet."))) //NOTICE TO NO SELECTED CLASS
			//return nil, nil

			if c.Exp >= 233332051410 && c.Level == 100 {
				if c.Class != 0 {
					buff, err := database.FindBuffByID(70006, c.ID)
					if err == nil && buff != nil {
						buff.Delete()
					}

					buff, err = database.FindBuffByID(70007, c.ID)
					if err == nil && buff != nil {
						buff.Delete()
					}

					buff, err = database.FindBuffByID(70008, c.ID)
					if err == nil && buff != nil {
						buff.Delete()
					}
					_, rogueSword, err := c.FindItemInInventory(nil, 13370222)
					if err != nil {
						return nil, err
					} else if rogueSword == nil || rogueSword.Quantity < 1 {
						resp := messaging.InfoMessage("You're missing Rogue Sword")
						return resp, nil
					}
					_, shortSword, err := c.FindItemInInventory(nil, 13370223)
					if err != nil {
						return nil, err
					} else if shortSword == nil || shortSword.Quantity < 2000 {
						resp := messaging.InfoMessage("You're missing Short Sword")
						return resp, nil
					}
					_, longBlade, err := c.FindItemInInventory(nil, 13370224)
					if err != nil {
						return nil, err
					} else if longBlade == nil || longBlade.Quantity < 2000 {
						resp := messaging.InfoMessage("You're missing Short Sword")
						return resp, nil
					}

					_, shamanMask, err := c.FindItemInInventory(nil, 13370225)
					if err != nil {
						return nil, err
					} else if shamanMask == nil || shamanMask.Quantity < 2000 {
						resp := messaging.InfoMessage("You're missing Shaman Mask")
						return resp, nil
					}
					_, axemanAxe, err := c.FindItemInInventory(nil, 13370226)
					if err != nil {
						return nil, err
					} else if axemanAxe == nil || axemanAxe.Quantity < 2000 {
						resp := messaging.InfoMessage("You're missing Axeman's Axe")
						return resp, nil
					}
					rogueSword.Delete()
					shortSword.Delete()
					longBlade.Delete()
					shamanMask.Delete()
					axemanAxe.Delete()
					/*
						//charmenu := utils.Packet{0xAA, 0x55, 0x03, 0x00, 0x09, 0x09, 0x00, 0x55, 0xAA} //Select Character
						ATARAXIA := utils.Packet{0xAA, 0x55, 0x03, 0x00, 0x57, 0x21, 0xE3, 0x55, 0xAA, 0xaa, 0x55, 0x0b, 0x00, 0x75, 0x00, 0x01, 0x00, 0x80, 0xa1, 0x43, 0x00, 0x00, 0x3d, 0x43, 0x55, 0xaa}
						resp := ATARAXIA
						resp[6] = byte(c.Type + 10) // character type
						KIIRAS := utils.Packet{0xaa, 0x55, 0x54, 0x00, 0x71, 0x14, 0x51, 0x55, 0xAA}
						kiirasuzenet := "At this very moment, I ascend through Bushido Online to claim my place among the legendary masters of Strong HERO."
						kiirasresp := KIIRAS
						index := 6
						kiirasresp[index] = byte(len(c.Name) + len(kiirasuzenet))
						index++
						kiirasresp.Insert([]byte("["+c.Name+"]"), index) // character name
						index += len(c.Name) + 2
						kiirasresp.Insert([]byte(kiirasuzenet), index) // character name
						kiirasresp.SetLength(int16(binary.Size(kiirasresp) - 6))

						p := nats.CastPacket{CastNear: false, Data: kiirasresp}
						p.Cast()

						resp.Concat(kiirasresp)
						c.Socket.Write(resp)
						c.Level = 100
						c.Type += 10
						c.Update()
						c.AddExp(1)
						c.Level = 101
						c.Socket.Skills.Delete()
						c.Socket.Skills.Create(c)
						s.Skills.Delete()
						s.Skills.Create(c)
						s.Skills.SkillPoints = 6800
						s.Skills.Update()
						c.Update()
						s.User.Update()
						resp, _ = divineJobPromotion(c, npcID)
						statData, _ := c.GetStats()
						resp.Concat(statData)
						s.Conn.Write(resp)
						//database.MakeAnnouncement("At this moment I mark my name on list of Top master in Strong HERO. - " + c.Name)
						/*time.AfterFunc(time.Duration(60*time.Second), func() {
							CharacterSelect := utils.Packet{0xAA, 0x55, 0x04, 0x00, 0x01, 0x05, 0x0A, 0x00, 0x55, 0xAA}
							CHARACTER_MENU := utils.Packet{0xAA, 0x55, 0x03, 0x00, 0x09, 0x09, 0x00, 0x55, 0xAA}
							resp := CHARACTER_MENU
							resp.Concat(CharacterSelect)
							s.Conn.Write(resp)
						})*/
				} else {
					resp.Concat(messaging.InfoMessage(fmt.Sprintf("You don't have class."))) //NOTICE TO NO SELECTED CLASS
				}
			}

		case 1341: // Warlord
			if c.Class == 0 {
				c.Class = 31
				book1 := 100031001
				book2 := 100030013
				jobName := "Warlord"
				c.Update()
				resp = JOB_PROMOTED
				resp[6] = byte(c.Class)

				r, _, err := c.AddItem(&database.InventorySlot{ItemID: int64(book1), Quantity: 1}, -1, false)
				if err != nil {
					return resp, err
				} else if r == nil {
					return nil, nil
				}

				resp.Concat(*r)

				r, _, err = c.AddItem(&database.InventorySlot{ItemID: int64(book2), Quantity: 1}, -1, false)
				if err != nil {
					return resp, err
				} else if r == nil {
					return nil, nil
				}

				resp.Concat(*r)

				resp.Concat(messaging.InfoMessage(fmt.Sprintf("Promoted as a %s.", jobName))) //NOTICE TO PROMOTE
			} else {
				return nil, nil
			}
		case 1342: // BeastLord
			if c.Class == 0 {
				c.Class = 33
				book1 := 100031002
				book2 := 100030014
				jobName := "BeastLord"
				c.Update()
				resp = JOB_PROMOTED
				resp[6] = byte(c.Class)

				r, _, err := c.AddItem(&database.InventorySlot{ItemID: int64(book1), Quantity: 1}, -1, false)
				if err != nil {
					return resp, err
				} else if r == nil {
					return nil, nil
				}

				resp.Concat(*r)

				r, _, err = c.AddItem(&database.InventorySlot{ItemID: int64(book2), Quantity: 1}, -1, false)
				if err != nil {
					return resp, err
				} else if r == nil {
					return nil, nil
				}

				resp.Concat(*r)

				resp.Concat(messaging.InfoMessage(fmt.Sprintf("Promoted as a %s.", jobName))) //NOTICE TO PROMOTE
			} else {
				return nil, nil
			}
		case 1343: // HolyHand
			if c.Class == 0 {
				c.Class = 32
				book1 := 100031003
				book2 := 100030015
				jobName := "HolyHand"
				c.Update()
				resp = JOB_PROMOTED
				resp[6] = byte(c.Class)

				r, _, err := c.AddItem(&database.InventorySlot{ItemID: int64(book1), Quantity: 1}, -1, false)
				if err != nil {
					return resp, err
				} else if r == nil {
					return nil, nil
				}

				resp.Concat(*r)

				r, _, err = c.AddItem(&database.InventorySlot{ItemID: int64(book2), Quantity: 1}, -1, false)
				if err != nil {
					return resp, err
				} else if r == nil {
					return nil, nil
				}

				resp.Concat(*r)

				resp.Concat(messaging.InfoMessage(fmt.Sprintf("Promoted as a %s.", jobName))) //NOTICE TO PROMOTE
			} else {
				return nil, nil
			}
		case 1344: // ShadowLord
			if c.Class == 0 {
				c.Class = 34
				book1 := 100031004
				book2 := 100030016
				jobName := "ShadowLord"
				c.Update()
				resp = JOB_PROMOTED
				resp[6] = byte(c.Class)

				r, _, err := c.AddItem(&database.InventorySlot{ItemID: int64(book1), Quantity: 1}, -1, false)
				if err != nil {
					return resp, err
				} else if r == nil {
					return nil, nil
				}

				resp.Concat(*r)

				r, _, err = c.AddItem(&database.InventorySlot{ItemID: int64(book2), Quantity: 1}, -1, false)
				if err != nil {
					return resp, err
				} else if r == nil {
					return nil, nil
				}

				resp.Concat(*r)

				resp.Concat(messaging.InfoMessage(fmt.Sprintf("Promoted as a %s.", jobName))) //NOTICE TO PROMOTE
			} else {
				return nil, nil
			}
		case 1345: // Move To Non Divine Sawang
			if c.Level >= 75 && c.Level < 101 {
				resp, _ = c.ChangeMap(100, nil)
			}
		case 1346: // Divine Sawang
			if c.Level >= 150 {
				resp, _ = c.ChangeMap(101, nil)

			} else {
				resp := messaging.InfoMessage("Hey " + c.Name + ", your level is not 150!")
				return resp, nil
			}
		case 1347: // Divine Sawang 2
			if c.Level >= 185 {
				resp, _ = c.ChangeMap(102, nil)
			}
		case 1348: // Dream Valley
			if c.Level >= 140 {
				resp, _ = c.ChangeMap(30, nil)
			}
		case 1349: // Market Arena
			if database.CanJoinLastMan && !database.LastmanStarted {
				if c.Level >= 60 {
					c.IsinLastMan = true
					database.LastManMutex.Lock()
					database.LastManCharacters[c.ID] = c
					database.LastManMutex.Unlock()
					coordinate := &utils.Location{X: 137, Y: 365}
					resp, _ = c.ChangeMap(254, coordinate)
				}
			}
		case 1350: // Ancient Relic
			if c.Level > 70 {
				goldSandSlot, goldSand, err := c.FindItemInInventory(nil, 1030)
				if err != nil {
					return nil, err
				}

				stoneLiquidSlot, stoneLiquid, err := c.FindItemInInventory(nil, 1029)
				if err != nil {
					return nil, err
				}

				if goldSand == nil || stoneLiquid == nil {
					return nil, nil
				}

				removeStone, _ := c.RemoveItem(stoneLiquidSlot)
				removeGoldSand, _ := c.RemoveItem(goldSandSlot)

				if c.Type == 52 || c.Type == 53 || c.Type == 56 {
					r, _, err := c.AddItem(&database.InventorySlot{ItemID: 99032003, Quantity: 5000}, -1, false)
					if err != nil {
						return nil, nil
					} else if r == nil {
						return nil, nil
					}

					resp.Concat(removeStone)
					resp.Concat(removeGoldSand)
					resp.Concat(*r)
					resp.Concat(c.GetGold())

				}

				if c.Type == 54 || c.Type == 57 || c.Type == 59 {
					r, _, err := c.AddItem(&database.InventorySlot{ItemID: 99042003, Quantity: 5000}, -1, false)
					if err != nil {
						return nil, nil
					} else if r == nil {
						return nil, nil
					}

					resp.Concat(removeStone)
					resp.Concat(removeGoldSand)
					resp.Concat(*r)
					resp.Concat(c.GetGold())
				}
			}
		case 1351: // Red dragon boots
			if c.Level > 80 {
				goldSandSlot, goldSand, err := c.FindItemInInventory(nil, 1030)
				if err != nil {
					return nil, err
				}

				stoneLiquidSlot, stoneLiquid, err := c.FindItemInInventory(nil, 1029)
				if err != nil {
					return nil, err
				}

				if goldSand == nil || stoneLiquid == nil {
					return nil, nil
				}

				removeStone, _ := c.RemoveItem(stoneLiquidSlot)
				removeGoldSand, _ := c.RemoveItem(goldSandSlot)

				if c.Type == 52 || c.Type == 53 || c.Type == 56 {
					r, _, err := c.AddItem(&database.InventorySlot{ItemID: 99033003, Quantity: 5000}, -1, false)
					if err != nil {
						return nil, nil
					} else if r == nil {
						return nil, nil
					}

					resp.Concat(removeStone)
					resp.Concat(removeGoldSand)
					resp.Concat(*r)
					resp.Concat(c.GetGold())

				}

				if c.Type == 54 || c.Type == 57 || c.Type == 59 {
					r, _, err := c.AddItem(&database.InventorySlot{ItemID: 99043003, Quantity: 5000}, -1, false)
					if err != nil {
						return nil, nil
					} else if r == nil {
						return nil, nil
					}

					resp.Concat(removeStone)
					resp.Concat(removeGoldSand)
					resp.Concat(*r)
					resp.Concat(c.GetGold())
				}
			}
		case 1352: // Move to Endless
			if c.Faction == 1 {
				resp, _ = c.ChangeMap(250, &utils.Location{X: 255, Y: 133})
			}

			if c.Faction == 2 {
				resp, _ = c.ChangeMap(250, &utils.Location{X: 255, Y: 377})
			}
		case 1353: // Go Reborner
			//resp := messaging.InfoMessage("Not available yet.")
			//return resp, nil

			if c.Exp >= 233332051410 && c.Level == 100 {
				if c.RebornLevel == 0 {
					if c.Class != 0 {
						/*
							_, poleArm, err := c.FindItemInInventoryByPlus(nil, 3, 10410006)
							if err != nil {
								return nil, err
							} else if poleArm == nil {
								resp := messaging.InfoMessage("You don't have Golden Pole Arm")
								return resp, nil
							}

							_, preciousJade, err := c.FindItemInInventory(nil, 18500892)
							if err != nil {
								return nil, err
							} else if preciousJade == nil {
								resp := messaging.InfoMessage("You don't have Fraction of Precious Jade(9x)")
								return resp, nil
							}

							if preciousJade.Quantity < 9 {
								resp := messaging.InfoMessage("You don't have Fraction of Precious Jade(9x)")
								return resp, nil
							}

							_, ironOre, err := c.FindItemInInventory(nil, 1000)
							if err != nil {
								return nil, err
							} else if ironOre == nil {
								resp := messaging.InfoMessage("You are missing out Iron Ore(98x).")
								return resp, nil
							}

							if ironOre.Quantity < 98 {
								resp := messaging.InfoMessage("You are missing out Iron Ore(98x).")
								return resp, nil
							}

							_, castIron, err := c.FindItemInInventory(nil, 1001)
							if err != nil {
								return nil, err
							} else if castIron == nil {
								resp := messaging.InfoMessage("You are missing out Cast Iron(19x).")
								return resp, nil
							}

							if castIron.Quantity < 19 {
								resp := messaging.InfoMessage("You are missing out Cast Iron(19x).")
								return resp, nil
							}

							_, steel, err := c.FindItemInInventory(nil, 1002)
							if err != nil {
								return nil, err
							} else if steel == nil {
								resp := messaging.InfoMessage("You are missing out Stell(4x).")
								return resp, nil
							}

							if steel.Quantity < 4 {
								resp := messaging.InfoMessage("You are missing out Stell(4x).")
								return resp, nil
							}

							_, violetIron, err := c.FindItemInInventory(nil, 1003)
							if err != nil {
								return nil, err
							} else if violetIron == nil {
								resp := messaging.InfoMessage("You are missing out Violet Iron(764x).")
								return resp, nil
							}

							if violetIron.Quantity < 764 {
								resp := messaging.InfoMessage("You are missing out Violet Iron(764x).")
								return resp, nil
							}

							_, toxicIron, err := c.FindItemInInventory(nil, 1004)
							if err != nil {
								return nil, err
							} else if toxicIron == nil {
								resp := messaging.InfoMessage("You are missing out Faded Toxic Iron(812x).")
								return resp, nil
							}

							if toxicIron.Quantity < 812 {
								resp := messaging.InfoMessage("You are missing out Faded Toxic Iron(812x).")
								return resp, nil
							}

							_, blackCoal, err := c.FindItemInInventory(nil, 1008)
							if err != nil {
								return nil, err
							} else if blackCoal == nil {
								resp := messaging.InfoMessage("You are missing out Black Coal(91x).")
								return resp, nil
							}

							if blackCoal.Quantity < 91 {
								resp := messaging.InfoMessage("You are missing out Black Coal(91x).")
								return resp, nil
							}

							_, chromaticPowder, err := c.FindItemInInventory(nil, 1007)
							if err != nil {
								return nil, err
							} else if chromaticPowder == nil {
								resp := messaging.InfoMessage("You are missing out Chromatic Powder(8x).")
								return resp, nil
							}

							if chromaticPowder.Quantity < 8 {
								resp := messaging.InfoMessage("You are missing out Chromatic Powder(8x).")
								return resp, nil
							}

							_, chromaticGem, err := c.FindItemInInventory(nil, 1006)
							if err != nil {
								return nil, err
							} else if chromaticGem == nil {
								resp := messaging.InfoMessage("You are missing out Chromatic Gem(1x).")
								return resp, nil
							}

							if chromaticGem.Quantity < 1 {
								resp := messaging.InfoMessage("You are missing out Chromatic Gem(1x).")
								return resp, nil
							}

							_, naturalGem, err := c.FindItemInInventory(nil, 1005)
							if err != nil {
								return nil, err
							} else if naturalGem == nil {
								resp := messaging.InfoMessage("You are missing out Natural Gem(94x).")
								return resp, nil
							}

							if naturalGem.Quantity < 94 {
								resp := messaging.InfoMessage("You are missing out Natural Gem(94x).")
								return resp, nil
							}

							_, silkCloath, err := c.FindItemInInventory(nil, 1012)
							if err != nil {
								return nil, err
							} else if silkCloath == nil {
								resp := messaging.InfoMessage("You are missing out Silk Cloth(89x).")
								return resp, nil
							}

							if silkCloath.Quantity < 89 {
								resp := messaging.InfoMessage("You are missing out Silk Cloth(89x).")
								return resp, nil
							}

							_, silk, err := c.FindItemInInventory(nil, 1013)
							if err != nil {
								return nil, err
							} else if silk == nil {
								resp := messaging.InfoMessage("You are missing out Silk(12x).")
								return resp, nil
							}

							if silk.Quantity < 12 {
								resp := messaging.InfoMessage("You are missing out Silk(12x).")
								return resp, nil
							}

							_, tigerLeather, err := c.FindItemInInventory(nil, 1014)
							if err != nil {
								return nil, err
							} else if tigerLeather == nil {
								resp := messaging.InfoMessage("You are missing out Tiger Leather(96x).")
								return resp, nil
							}

							if tigerLeather.Quantity < 96 {
								resp := messaging.InfoMessage("You are missing out Tiger Leather(96x).")
								return resp, nil
							}

							_, ironThread, err := c.FindItemInInventory(nil, 1015)
							if err != nil {
								return nil, err
							} else if ironThread == nil {
								resp := messaging.InfoMessage("You are missing out Iron Thread(23x).")
								return resp, nil
							}

							if ironThread.Quantity < 23 {
								resp := messaging.InfoMessage("You are missing out Iron Thread(23x).")
								return resp, nil
							}

							_, fdEmbroidery, err := c.FindItemInInventory(nil, 1016)
							if err != nil {
								return nil, err
							} else if fdEmbroidery == nil {
								resp := messaging.InfoMessage("You are missing out Five Dragon Embroidery(6x).")
								return resp, nil
							}

							if fdEmbroidery.Quantity < 6 {
								resp := messaging.InfoMessage("You are missing out Five Dragon Embroidery(6x).")
								return resp, nil
							}

							_, wEmbroidery, err := c.FindItemInInventory(nil, 1017)
							if err != nil {
								return nil, err
							} else if wEmbroidery == nil {
								resp := messaging.InfoMessage("You are missing out White Embroidery(11x).")
								return resp, nil
							}

							if wEmbroidery.Quantity < 11 {
								resp := messaging.InfoMessage("You are missing out White Embroidery(11x).")
								return resp, nil
							}

							_, dye, err := c.FindItemInInventory(nil, 1018)
							if err != nil {
								return nil, err
							} else if dye == nil {
								resp := messaging.InfoMessage("You are missing out Dye(92x).")
								return resp, nil
							}

							if dye.Quantity < 92 {
								resp := messaging.InfoMessage("You are missing out Dye(92x).")
								return resp, nil
							}

							_, softeninOil, err := c.FindItemInInventory(nil, 1019)
							if err != nil {
								return nil, err
							} else if softeninOil == nil {
								resp := messaging.InfoMessage("You are missing out Softening Oil(87x).")
								return resp, nil
							}

							if softeninOil.Quantity < 87 {
								resp := messaging.InfoMessage("You are missing out Softening Oil(87x).")
								return resp, nil
							}

							_, stonePowder, err := c.FindItemInInventory(nil, 1020)
							if err != nil {
								return nil, err
							} else if stonePowder == nil {
								resp := messaging.InfoMessage("You are missing out Stone Powder(1x).")
								return resp, nil
							}

							if stonePowder.Quantity < 1 {
								resp := messaging.InfoMessage("You are missing out Stone Powder(1x).")
								return resp, nil
							}

							_, dilutedPowder, err := c.FindItemInInventory(nil, 1021)
							if err != nil {
								return nil, err
							} else if dilutedPowder == nil {
								resp := messaging.InfoMessage("You are missing out Diluted Powder(14x).")
								return resp, nil
							}

							if dilutedPowder.Quantity < 14 {
								resp := messaging.InfoMessage("You are missing out Diluted Powder(14x).")
								return resp, nil
							}

							_, gemPowder, err := c.FindItemInInventory(nil, 1022)
							if err != nil {
								return nil, err
							} else if gemPowder == nil {
								resp := messaging.InfoMessage("You are missing out Gem Powder(9x).")
								return resp, nil
							}

							if gemPowder.Quantity < 9 {
								resp := messaging.InfoMessage("You are missing out Gem Powder(9x).")
								return resp, nil
							}

							_, tussah, err := c.FindItemInInventory(nil, 1023)
							if err != nil {
								return nil, err
							} else if tussah == nil {
								resp := messaging.InfoMessage("You are missing out Tussah(113x).")
								return resp, nil
							}

							if tussah.Quantity < 113 {
								resp := messaging.InfoMessage("You are missing out Tussah(113x).")
								return resp, nil
							}

							_, denseIron, err := c.FindItemInInventory(nil, 1024)
							if err != nil {
								return nil, err
							} else if denseIron == nil {
								resp := messaging.InfoMessage("You are missing out Dense Iron(184x).")
								return resp, nil
							}

							if denseIron.Quantity < 184 {
								resp := messaging.InfoMessage("You are missing out Dense Iron(184x).")
								return resp, nil
							}

							_, fireProffIron, err := c.FindItemInInventory(nil, 1032)
							if err != nil {
								return nil, err
							} else if fireProffIron == nil {
								resp := messaging.InfoMessage("You are missing out Fireproof Iron(100x).")
								return resp, nil
							}

							if fireProffIron.Quantity < 100 {
								resp := messaging.InfoMessage("You are missing out Fireproof Iron(100x).")
								return resp, nil
							}

							_, delicateSilk, err := c.FindItemInInventory(nil, 1033)
							if err != nil {
								return nil, err
							} else if delicateSilk == nil {
								resp := messaging.InfoMessage("You are missing out Delicate Silk(1x).")
								return resp, nil
							}

							if delicateSilk.Quantity < 1 {
								resp := messaging.InfoMessage("You are missing out Delicate Silk(1x).")
								return resp, nil
							}

							_, volcanicRock, err := c.FindItemInInventory(nil, 1034)
							if err != nil {
								return nil, err
							} else if volcanicRock == nil {
								resp := messaging.InfoMessage("You are missing out Volcanic Rock(90x).")
								return resp, nil
							}

							if volcanicRock.Quantity < 90 {
								resp := messaging.InfoMessage("You are missing out Volcanic Rock(90x).")
								return resp, nil
							}

							_, redCottonPlant, err := c.FindItemInInventory(nil, 1035)
							if err != nil {
								return nil, err
							} else if redCottonPlant == nil {
								resp := messaging.InfoMessage("You are missing out Red Cotton Plant(85x).")
								return resp, nil
							}

							if redCottonPlant.Quantity < 85 {
								resp := messaging.InfoMessage("You are missing out Red Cotton Plant(85x).")
								return resp, nil
							}

							_, opallCrystallBall, err := c.FindItemInInventory(nil, 231)
							if err != nil {
								return nil, err
							} else if opallCrystallBall == nil {
								resp := messaging.InfoMessage("You are missing out Opal Crystal Ball(20x).")
								return resp, nil
							}

							if opallCrystallBall.Quantity < 20 {
								resp := messaging.InfoMessage("You are missing out Opal Crystal Ball(20x).")
								return resp, nil
							}

							_, garnetCrystallBall, err := c.FindItemInInventory(nil, 232)
							if err != nil {
								return nil, err
							} else if garnetCrystallBall == nil {
								resp := messaging.InfoMessage("You are missing out Garnet Crystal Ball(20x).")
								return resp, nil
							}

							if garnetCrystallBall.Quantity < 20 {
								resp := messaging.InfoMessage("You are missing out Garnet Crystal Ball(20x).")
								return resp, nil
							}

							_, citrineCrystallBall, err := c.FindItemInInventory(nil, 233)
							if err != nil {
								return nil, err
							} else if citrineCrystallBall == nil {
								resp := messaging.InfoMessage("You are missing out Citrine Crystal Ball(20x).")
								return resp, nil
							}

							if citrineCrystallBall.Quantity < 20 {
								resp := messaging.InfoMessage("You are missing out Citrine Crystal Ball(20x).")
								return resp, nil
							}

							_, jadeiteCrystallBall, err := c.FindItemInInventory(nil, 233)
							if err != nil {
								return nil, err
							} else if jadeiteCrystallBall == nil {
								resp := messaging.InfoMessage("You are missing out Emerald Jadeite Crystal Ball(20x).")
								return resp, nil
							}

							if jadeiteCrystallBall.Quantity < 20 {
								resp := messaging.InfoMessage("You are missing out Emerald Jadeite Crystal Ball(20x).")
								return resp, nil
							}

							_, divineSwordArts, err := c.FindItemInInventory(nil, 16100101)
							if err != nil {
								return nil, err
							} else if divineSwordArts == nil {
								resp := messaging.InfoMessage("You are missing out New: Divine Sword Arts(1x).")
								return resp, nil
							}

							if divineSwordArts.Quantity < 1 {
								resp := messaging.InfoMessage("You are missing out New: Divine Sword Arts(1x).")
								return resp, nil
							}

							_, divineBowArts, err := c.FindItemInInventory(nil, 16100103)
							if err != nil {
								return nil, err
							} else if divineBowArts == nil {
								resp := messaging.InfoMessage("You are missing out New: Divine Bow Arts(1x).")
								return resp, nil
							}

							if divineBowArts.Quantity < 1 {
								resp := messaging.InfoMessage("You are missing out New: Divine Bow Arts(1x).")
								return resp, nil
							}

							_, divineThrowingArts, err := c.FindItemInInventory(nil, 16100105)
							if err != nil {
								return nil, err
							} else if divineThrowingArts == nil {
								resp := messaging.InfoMessage("You are missing out New: Divine Throwing Arts(1x).")
								return resp, nil
							}

							if divineThrowingArts.Quantity < 1 {
								resp := messaging.InfoMessage("You are missing out New: Divine Throwing Arts(1x).")
								return resp, nil
							}

							_, divineRodArts, err := c.FindItemInInventory(nil, 16100107)
							if err != nil {
								return nil, err
							} else if divineRodArts == nil {
								resp := messaging.InfoMessage("You are missing out New: Divine Rod Arts(1x).")
								return resp, nil
							}

							if divineRodArts.Quantity < 1 {
								resp := messaging.InfoMessage("You are missing out New: Divine Rod Arts(1x).")
								return resp, nil
							}

							_, divineAxeArts, err := c.FindItemInInventory(nil, 16100109)
							if err != nil {
								return nil, err
							} else if divineAxeArts == nil {
								resp := messaging.InfoMessage("You are missing out New: Divine Axe Arts(1x).")
								return resp, nil
							}

							if divineAxeArts.Quantity < 1 {
								resp := messaging.InfoMessage("You are missing out New: Divine Axe Arts(1x).")
								return resp, nil
							}

							_, divineFistArts, err := c.FindItemInInventory(nil, 16100111)
							if err != nil {
								return nil, err
							} else if divineFistArts == nil {
								resp := messaging.InfoMessage("You are missing out New: Divine Fist Arts(1x).")
								return resp, nil
							}

							if divineFistArts.Quantity < 1 {
								resp := messaging.InfoMessage("You are missing out New: Divine Fist Arts(1x).")
								return resp, nil
							}

							_, hornOfRedDragon, err := c.FindItemInInventory(nil, 90000190)
							if err != nil {
								return nil, err
							} else if hornOfRedDragon == nil {
								resp := messaging.InfoMessage("You are missing out Horn of Red Dragon(1x).")
								return resp, nil
							}

							if hornOfRedDragon.Quantity < 1 {
								resp := messaging.InfoMessage("You are missing out Horn of Red Dragon(1x).")
								return resp, nil
							}

							if poleArm == nil || preciousJade == nil || ironOre == nil || castIron == nil || steel == nil || violetIron == nil || toxicIron == nil || blackCoal == nil || chromaticPowder == nil || chromaticGem == nil || naturalGem == nil || silkCloath == nil || silk == nil || tigerLeather == nil || ironThread == nil || fdEmbroidery == nil || wEmbroidery == nil || dye == nil || softeninOil == nil || stonePowder == nil || dilutedPowder == nil || gemPowder == nil || tussah == nil || denseIron == nil || fireProffIron == nil || delicateSilk == nil || volcanicRock == nil || redCottonPlant == nil || opallCrystallBall == nil || garnetCrystallBall == nil || citrineCrystallBall == nil || jadeiteCrystallBall == nil || divineSwordArts == nil || divineBowArts == nil || divineThrowingArts == nil || divineRodArts == nil || divineAxeArts == nil || divineFistArts == nil || hornOfRedDragon == nil {
								resp := messaging.InfoMessage("You are missing out items.")
								return resp, nil
							}

							poleArm.Delete()
							divineSwordArts.Delete()
							divineBowArts.Delete()
							divineThrowingArts.Delete()
							divineRodArts.Delete()
							divineAxeArts.Delete()
							divineFistArts.Delete()
							hornOfRedDragon.Delete()

							c.DecrementItem(preciousJade.SlotID, 9)
							c.DecrementItem(ironOre.SlotID, 98)
							c.DecrementItem(castIron.SlotID, 19)
							c.DecrementItem(steel.SlotID, 4)
							c.DecrementItem(violetIron.SlotID, 764)
							c.DecrementItem(toxicIron.SlotID, 812)
							c.DecrementItem(blackCoal.SlotID, 91)
							c.DecrementItem(chromaticPowder.SlotID, 8)
							c.DecrementItem(chromaticGem.SlotID, 1)
							c.DecrementItem(naturalGem.SlotID, 94)
							c.DecrementItem(silkCloath.SlotID, 89)
							c.DecrementItem(silk.SlotID, 37)
							c.DecrementItem(tigerLeather.SlotID, 96)
							c.DecrementItem(ironThread.SlotID, 23)
							c.DecrementItem(fdEmbroidery.SlotID, 6)
							c.DecrementItem(wEmbroidery.SlotID, 11)
							c.DecrementItem(dye.SlotID, 92)
							c.DecrementItem(softeninOil.SlotID, 87)
							c.DecrementItem(stonePowder.SlotID, 1)
							c.DecrementItem(dilutedPowder.SlotID, 14)
							c.DecrementItem(gemPowder.SlotID, 9)
							c.DecrementItem(tussah.SlotID, 113)
							c.DecrementItem(denseIron.SlotID, 184)
							c.DecrementItem(fireProffIron.SlotID, 100)
							c.DecrementItem(delicateSilk.SlotID, 1)
							c.DecrementItem(volcanicRock.SlotID, 90)
							c.DecrementItem(redCottonPlant.SlotID, 85)
							c.DecrementItem(opallCrystallBall.SlotID, 20)
							c.DecrementItem(garnetCrystallBall.SlotID, 20)
							c.DecrementItem(citrineCrystallBall.SlotID, 20)
							c.DecrementItem(jadeiteCrystallBall.SlotID, 20)
						*/

						KIIRAS := utils.Packet{0xaa, 0x55, 0x54, 0x00, 0x71, 0x14, 0x51, 0x55, 0xAA}
						kiirasuzenet := "has reborned and is ready for new adventures and endless possibilities."
						kiirasresp := KIIRAS
						index := 6
						kiirasresp[index] = byte(len(c.Name) + len(kiirasuzenet))
						index++
						kiirasresp.Insert([]byte("["+c.Name+"]"), index) // character name
						index += len(c.Name) + 2
						kiirasresp.Insert([]byte(kiirasuzenet), index) // character name
						kiirasresp.SetLength(int16(binary.Size(kiirasresp) - 6))

						p := nats.CastPacket{CastNear: false, Data: kiirasresp}
						p.Cast()

						resp.Concat(kiirasresp)

						ATARAXIA := utils.Packet{0xAA, 0x55, 0x03, 0x00, 0x57, 0x21, 0xE3, 0x55, 0xAA, 0xaa, 0x55, 0x0b, 0x00, 0x75, 0x00, 0x01, 0x00, 0x80, 0xa1, 0x43, 0x00, 0x00, 0x3d, 0x43, 0x55, 0xaa}
						tmpAxataxia := ATARAXIA
						tmpAxataxia[6] = byte(c.Type) // character type
						resp.Concat(tmpAxataxia)

						c.Socket.Write(resp)
						c.Level = 1
						c.RebornLevel = 1
						c.Exp = 0
						c.Class = 0
						c.AddExp(1)
						c.Update()
						c.Socket.Stats.Reset()
						c.Socket.Stats.Create(c)
						c.Socket.Stats.Update()
						c.Socket.Skills.Delete()
						c.Socket.Skills.Create(c)
						c.Socket.Skills.Update()
						c.Update()
						c.Socket.Stats.Update()
						s.Skills.Delete()
						s.Skills.Create(c)
						s.Skills.SkillPoints = 0
						s.Skills.Update()
						c.Update()
						s.User.Update()
						//resp, _ = divineJobPromotion(c, npcID)
						statData, _ := c.GetStats()
						buff := &database.Buff{ID: 70006, CharacterID: c.ID, Name: "Reborn St 1", EXPMultiplier: -10, DropMultiplier: 5, StartedAt: c.Epoch, Duration: (2592000 * 5)}
						err = buff.Create()
						if err != nil {
							fmt.Println(err)
							return nil, err
						}
						tmpResp, _, _ := c.AddItem(&database.InventorySlot{ItemID: 18500891, Quantity: 1}, -1, false)
						resp.Concat(*tmpResp)
						resp.Concat(statData)
						s.Conn.Write(resp)
						database.MakeAnnouncement(c.Name + " has reborned and is ready for new adventures and endless possibilities.")
						time.AfterFunc(time.Duration(3*time.Second), func() {
							CharacterSelect := utils.Packet{0xAA, 0x55, 0x04, 0x00, 0x01, 0x05, 0x0A, 0x00, 0x55, 0xAA}
							CHARACTER_MENU := utils.Packet{0xAA, 0x55, 0x03, 0x00, 0x09, 0x09, 0x00, 0x55, 0xAA}
							resp := CHARACTER_MENU
							resp.Concat(CharacterSelect)
							s.Conn.Write(resp)
						})
					} else {
						resp.Concat(messaging.InfoMessage(fmt.Sprintf("You don't have class."))) //NOTICE TO NO SELECTED CLASS
					}
				} else {
					resp.Concat(messaging.InfoMessage(fmt.Sprintf("You are already at Reborn stage."))) //NOTICE TO NO SELECTED CLASS
				}
			}

		case 1354: // +1 Socket
			slotID, item, err := c.FindItemInInventoryByPlus(nil, 1, 235)
			if err != nil {
				return nil, err
			} else if slotID == -1 { // no same item found => find free slot
				return nil, err
			}

			if item.Quantity%2 == 0 {
				tmpData, _, err := c.AddItem(&database.InventorySlot{ItemID: 235, Quantity: item.Quantity / 2, Plus: 2, UpgradeArr: "{235,235,0,0,0,0,0,0,0,0,0,0,0,0,0}"}, -1, true)
				if err != nil {
					return nil, err
				} else if tmpData == nil {
					return nil, nil
				}

				resp.Concat(*tmpData)
				resp.Concat(*c.DecrementItem(slotID, item.Quantity))
			}
		case 1355: // +2 Socket
			slotID, item, err := c.FindItemInInventoryByPlus(nil, 2, 235)
			if err != nil {
				return nil, err
			} else if slotID == -1 { // no same item found => find free slot
				return nil, err
			}

			if item.Quantity%2 == 0 {
				tmpData, _, err := c.AddItem(&database.InventorySlot{ItemID: 235, Quantity: item.Quantity / 2, Plus: 3, UpgradeArr: "{235,235,235,0,0,0,0,0,0,0,0,0,0,0,0}"}, -1, true)
				if err != nil {
					return nil, err
				} else if tmpData == nil {
					return nil, nil
				}

				resp.Concat(*tmpData)
				resp.Concat(*c.DecrementItem(slotID, item.Quantity))
			}
		case 1356: // +3 Socket
			slotID, item, err := c.FindItemInInventoryByPlus(nil, 3, 235)
			if err != nil {
				return nil, err
			} else if slotID == -1 { // no same item found => find free slot
				return nil, err
			}

			if item.Quantity%2 == 0 {
				tmpData, _, err := c.AddItem(&database.InventorySlot{ItemID: 235, Quantity: item.Quantity / 2, Plus: 4, UpgradeArr: "{235,235,235,235,0,0,0,0,0,0,0,0,0,0,0}"}, -1, true)
				if err != nil {
					return nil, err
				} else if tmpData == nil {
					return nil, nil
				}

				resp.Concat(*tmpData)
				resp.Concat(*c.DecrementItem(slotID, item.Quantity))
			}
		case 1357: // +4 Socket
			slotID, item, err := c.FindItemInInventoryByPlus(nil, 4, 235)
			if err != nil {
				return nil, err
			} else if slotID == -1 { // no same item found => find free slot
				return nil, err
			}

			if item.Quantity%2 == 0 {
				tmpData, _, err := c.AddItem(&database.InventorySlot{ItemID: 235, Quantity: item.Quantity / 2, Plus: 5, UpgradeArr: "{235,235,235,235,235,0,0,0,0,0,0,0,0,0,0}"}, -1, true)
				if err != nil {
					return nil, err
				} else if tmpData == nil {
					return nil, nil
				}

				resp.Concat(*tmpData)
				resp.Concat(*c.DecrementItem(slotID, item.Quantity))
			}
		case 1358: // Go to Reborn Island
			resp := messaging.InfoMessage("Not available yet.")
			return resp, nil
		/*
			if c.Level > 74 {
				r, _ := c.ChangeMap(100, &utils.Location{X: 249, Y: 281})
				resp.Concat(r)
			} else {
				resp.Concat(messaging.InfoMessage(fmt.Sprintf("You are not at 75 Level")))
			}
		*/
		case 1359: // Go Reborner 2
			return nil, nil
			/*
				if c.Exp >= 233332051410 && c.Level == 100 {
					if c.RebornLevel == 1 {
						if c.Class != 0 {
							st1Buff, err := database.FindBuffByID(70006, c.ID) // check for temple buff
							if err != nil && st1Buff == nil {
								return nil, nil
							}
							st1Buff.Delete()
							c.Socket.Write(resp)
							c.Level = 1
							c.RebornLevel = 2
							// TODO karakterin exp ve drobunu ayarla karakter login olduğunda reborn levele göre versin
							c.Exp = 0
							c.Class = 0
							c.AddExp(1)
							c.Update()
							c.Socket.Stats.Reset()
							c.Socket.Stats.Create(c)
							c.Socket.Stats.Update()
							c.Socket.Skills.Delete()
							c.Socket.Skills.Create(c)
							c.Socket.Skills.Update()
							c.Update()
							s.Skills.Delete()
							s.Skills.Create(c)
							s.Skills.SkillPoints = 0
							s.Skills.Update()
							c.Update()
							s.User.Update()
							//resp, _ = divineJobPromotion(c, npcID)
							statData, _ := c.GetStats()
							buff := &database.Buff{ID: 70007, CharacterID: c.ID, Name: "Reborn St 2", EXPMultiplier: -20, DropMultiplier: 10, StartedAt: c.Epoch, Duration: (2592000 * 5)}
							err = buff.Create()
							if err != nil {
								fmt.Println(err)
								return nil, err
							}
							resp.Concat(statData)
							s.Conn.Write(resp)
							database.MakeAnnouncement(c.Name + " has reborned and is ready for new adventures and endless possibilities.")
							time.AfterFunc(time.Duration(2*time.Second), func() {
								CharacterSelect := utils.Packet{0xAA, 0x55, 0x04, 0x00, 0x01, 0x05, 0x0A, 0x00, 0x55, 0xAA}
								CHARACTER_MENU := utils.Packet{0xAA, 0x55, 0x03, 0x00, 0x09, 0x09, 0x00, 0x55, 0xAA}
								resp := CHARACTER_MENU
								resp.Concat(CharacterSelect)
								s.Conn.Write(resp)
							})
						} else {
							resp.Concat(messaging.InfoMessage(fmt.Sprintf("You don't have class."))) //NOTICE TO NO SELECTED CLASS
						}
					} else {
						resp.Concat(messaging.InfoMessage(fmt.Sprintf("You are not Reborn stage 1"))) //NOTICE TO NO SELECTED CLASS
					}
				}
			*/
		case 1360: // Go reborner 3
			return nil, nil
		/*
			if c.Exp >= 233332051410 && c.Level == 100 {
				if c.RebornLevel == 2 {
					if c.Class != 0 {
						st2Buff, err := database.FindBuffByID(70007, c.ID)
						if err != nil && st2Buff == nil {
							return nil, nil
						}
						st2Buff.Delete()
						c.Socket.Write(resp)
						c.Level = 1
						c.RebornLevel = 3
						// TODO karakterin exp ve drobunu ayarla karakter login olduğunda reborn levele göre versin
						c.Exp = 0
						c.Class = 0
						c.AddExp(1)
						c.Update()
						c.Socket.Stats.Reset()
						c.Socket.Stats.Create(c)
						c.Socket.Stats.Update()
						c.Socket.Skills.Delete()
						c.Socket.Skills.Create(c)
						c.Socket.Skills.Update()
						c.Update()
						s.Skills.Delete()
						s.Skills.Create(c)
						s.Skills.SkillPoints = 0
						s.Skills.Update()
						c.Update()
						s.User.Update()
						//resp, _ = divineJobPromotion(c, npcID)
						statData, _ := c.GetStats()
						buff := &database.Buff{ID: 70008, CharacterID: c.ID, Name: "Reborn St 3", EXPMultiplier: -30, DropMultiplier: 15, StartedAt: c.Epoch, Duration: (2592000 * 5)}
						err = buff.Create()
						if err != nil {
							fmt.Println(err)
							return nil, err
						}
						resp.Concat(statData)
						s.Conn.Write(resp)
						database.MakeAnnouncement(c.Name + " has reborned and is ready for new adventures and endless possibilities.")
						time.AfterFunc(time.Duration(2*time.Second), func() {
							CharacterSelect := utils.Packet{0xAA, 0x55, 0x04, 0x00, 0x01, 0x05, 0x0A, 0x00, 0x55, 0xAA}
							CHARACTER_MENU := utils.Packet{0xAA, 0x55, 0x03, 0x00, 0x09, 0x09, 0x00, 0x55, 0xAA}
							resp := CHARACTER_MENU
							resp.Concat(CharacterSelect)
							s.Conn.Write(resp)
						})
					} else {
						resp.Concat(messaging.InfoMessage(fmt.Sprintf("You don't have class."))) //NOTICE TO NO SELECTED CLASS
					}
				} else {
					resp.Concat(messaging.InfoMessage(fmt.Sprintf("You are not Reborn stage 2"))) //NOTICE TO NO SELECTED CLASS
				}
			}
		*/
		case 1361: // Go to wedding hall
			return nil, nil
		case 1362: // Go to normal mountain
			if c.Level > 144 {
				r, _ := c.ChangeMap(72, nil)
				resp.Concat(r)
			} else {
				resp.Concat(messaging.InfoMessage(fmt.Sprintf("You are not at 145 Level")))
			}
		case 1363: // go to master mountain
			if c.Level > 149 {
				r, _ := c.ChangeMap(71, nil)
				resp.Concat(r)
			} else {
				resp.Concat(messaging.InfoMessage(fmt.Sprintf("You are not at 150 Level")))
			}
		case 1364: // go to nighmare mountain
			if c.Level > 154 {
				r, _ := c.ChangeMap(70, nil)
				resp.Concat(r)
			} else {
				resp.Concat(messaging.InfoMessage(fmt.Sprintf("You are not at 155 Level")))
			}
		case 1365: // go to nighmare mountain
			if c.Level > 159 {
				r, _ := c.ChangeMap(73, nil)
				resp.Concat(r)
			} else {
				resp.Concat(messaging.InfoMessage(fmt.Sprintf("You are not at 155 Level")))
			}
		case 1366:
			if database.GuildVarActive {
				guild, err := database.FindGuildByID(c.GuildID)
				if err != nil {
					fmt.Println(err)
					return nil, nil
				}
				if guild == nil {
					resp.Concat(messaging.InfoMessage(fmt.Sprintf("You have not a Guild !")))
				} else {
					if c.Level >= 50 {
						r, _ := c.ChangeMap(74, nil)
						resp.Concat(r)
					} else {
						resp.Concat(messaging.InfoMessage(fmt.Sprintf("Min Level 50 !")))
					}
				}
			} else {
				resp.Concat(messaging.InfoMessage(fmt.Sprintf("Guild War not active !")))
			}
		case 1367:
			if c.Exp >= 544951059310 && c.Level == 200 {
				_, blackDemon, err := c.FindItemInInventory(nil, 18500572)
				if err != nil {
					return nil, err
				} else if blackDemon == nil || blackDemon.Quantity < 1 {
					resp := messaging.InfoMessage("You don't have Blood BlackDemon Sword")
					return resp, nil
				}

				dlRs, _ := c.RemoveItem(blackDemon.SlotID)
				if err != nil {
					return nil, err
				}
				resp.Concat(dlRs)
				r, _ := c.ChangeMap(33, nil)
				resp.Concat(r)
			} else {
				resp.Concat(messaging.InfoMessage(fmt.Sprintf("You don't meet the requirements")))
			}
		case 1368:
			if c.Exp >= 544951059310 && c.Level == 200 {
				_, poisonousLiver, err := c.FindItemInInventory(nil, 18500574)
				if err != nil {
					return nil, err
				} else if poisonousLiver == nil || poisonousLiver.Quantity < 20 {
					resp := messaging.InfoMessage("You don't have Poisonous Centipede Liver (20x)")
					return resp, nil
				}

				_, monkeyLiquor, err := c.FindItemInInventory(nil, 18500576)
				if err != nil {
					return nil, err
				} else if monkeyLiquor == nil || monkeyLiquor.Quantity < 20 {
					resp := messaging.InfoMessage("You don't have Monkey's Liquor (20x)")
					return resp, nil
				}

				_, poisonousTail, err := c.FindItemInInventory(nil, 18500575)
				if err != nil {
					return nil, err
				} else if poisonousTail == nil || poisonousTail.Quantity < 20 {
					resp := messaging.InfoMessage("You don't have Poisonous Scorpion's Tail (20x)")
					return resp, nil
				}

				_, smallSkin, err := c.FindItemInInventory(nil, 18500573)
				if err != nil {
					return nil, err
				} else if smallSkin == nil || smallSkin.Quantity < 20 {
					resp := messaging.InfoMessage("You don't have Small Centipede Skin (20x)")
					return resp, nil
				}

				_, liver, err := c.FindItemInInventory(nil, 18500571)
				if err != nil {
					return nil, err
				} else if liver == nil || liver.Quantity < 1 {
					resp := messaging.InfoMessage("You don't have Wyrm's Liver")
					return resp, nil
				}

				poisonousLiver.Delete()
				monkeyLiquor.Delete()
				poisonousTail.Delete()
				smallSkin.Delete()
				liver.Delete()

				c.GoDarkness()
				resp.Concat(c.GetGold())

			} else {
				resp.Concat(messaging.InfoMessage(fmt.Sprintf("You don't meet the requirements")))
			}
		case 1369:
			book1, book2, job = 100030020, 100030021, 41
			book3 := 100032001
			jobName := "God of War"
			resp, err = darknessJobPromotion(c, book1, book2, book3, job, npcID, jobName)
			if err != nil {
				return nil, err
			}
		case 1370:
			book1, book2, job = 100030022, 100030023, 42
			book3 := 100032002
			jobName := "God of Death"
			resp, err = darknessJobPromotion(c, book1, book2, book3, job, npcID, jobName)
			if err != nil {
				return nil, err
			}
		case 1371:
			book1, book2, job = 100030024, 100030025, 43
			book3 := 100032003
			jobName := "God of Blade"
			resp, err = darknessJobPromotion(c, book1, book2, book3, job, npcID, jobName)
			if err != nil {
				return nil, err
			}
		case 1800:
			buff, err := database.FindBuffByID(1337, c.ID)
			if err != nil {
				return nil, nil
			}
			if buff != nil {
				resp.Concat(messaging.InfoMessage(fmt.Sprintf(c.Name + "You already have the Christmas Blessing.")))
				return nil, nil
			}

			buffinfo := database.BuffInfections[1337]
			buff = &database.Buff{ID: 1337, CharacterID: c.ID, Name: buffinfo.Name, BagExpansion: false, StartedAt: c.Epoch, Duration: int64(720) * 60, EXPMultiplier: 20, DropMultiplier: 5}
			err = buff.Create()
			if err != nil {
				resp.Concat(messaging.InfoMessage(fmt.Sprintf("Some error :(")))
				return nil, nil
			}

			resp.Concat(messaging.InfoMessage(fmt.Sprintf("Merry Christmas Dear " + c.Name)))
		case 1373:
			_, letterB, err := c.FindItemInInventory(nil, 18500731)
			if err != nil {
				return nil, err
			} else if letterB == nil {
				resp := messaging.InfoMessage("Missing letters.")
				return resp, nil
			}

			_, letterU, err := c.FindItemInInventory(nil, 18500732)
			if err != nil {
				return nil, err
			} else if letterU == nil {
				resp := messaging.InfoMessage("Missing letters.")
				return resp, nil
			}

			_, letterS, err := c.FindItemInInventory(nil, 18500733)
			if err != nil {
				return nil, err
			} else if letterS == nil {
				resp := messaging.InfoMessage("Missing letters.")
				return resp, nil
			}

			_, letterH, err := c.FindItemInInventory(nil, 18500734)
			if err != nil {
				return nil, err
			} else if letterH == nil {
				resp := messaging.InfoMessage("Missing letters.")
				return resp, nil
			}

			_, letterI, err := c.FindItemInInventory(nil, 18500735)
			if err != nil {
				return nil, err
			} else if letterI == nil {
				resp := messaging.InfoMessage("Missing letters.")
				return resp, nil
			}

			_, letterD, err := c.FindItemInInventory(nil, 18500736)
			if err != nil {
				return nil, err
			} else if letterD == nil {
				resp := messaging.InfoMessage("Missing letters.")
				return resp, nil
			}

			_, letterO, err := c.FindItemInInventory(nil, 18500737)
			if err != nil {
				return nil, err
			} else if letterO == nil {
				resp := messaging.InfoMessage("Missing letters.")
				return resp, nil
			}

			if letterB == nil || letterU == nil || letterS == nil || letterH == nil || letterI == nil || letterD == nil || letterO == nil {
				resp := messaging.InfoMessage("Missing letters.")
				return resp, nil
			} else {

				resp.Concat(*c.DecrementItem(letterB.SlotID, 1))
				resp.Concat(*c.DecrementItem(letterU.SlotID, 1))
				resp.Concat(*c.DecrementItem(letterS.SlotID, 1))
				resp.Concat(*c.DecrementItem(letterH.SlotID, 1))
				resp.Concat(*c.DecrementItem(letterI.SlotID, 1))
				resp.Concat(*c.DecrementItem(letterD.SlotID, 1))
				resp.Concat(*c.DecrementItem(letterO.SlotID, 1))

				tmpResp, _, _ := c.AddItem(&database.InventorySlot{ItemID: 18500738, Quantity: 1}, -1, false)
				resp.Concat(*tmpResp)
				return resp, nil
			}
		case 1374:
			r, _ := c.ChangeMap(75, nil)
			resp.Concat(r)
		case 1375:
			_, bookOfUniverse, err := c.FindItemInInventory(nil, 90000149, 90000127)
			if err != nil {
				return nil, err
			} else if bookOfUniverse == nil {
				resp := messaging.InfoMessage("You don't have the required item.")
				return resp, nil
			}

			resp := utils.Packet{}
			resp.Concat(*c.DecrementItem(bookOfUniverse.SlotID, 1))

			seed := utils.RandInt(0, 1000)
			if seed > 850 {
				resp.Concat(GetNPCMenu(npcID, 133713, 0, nil))
				tmpResp, _, _ := c.AddItem(&database.InventorySlot{ItemID: 18500851, Quantity: 1}, -1, false)
				resp.Concat(*tmpResp)
			} else {
				resp.Concat(messaging.InfoMessage("Unfortunately your document's did not get accepted, please try again later."))
				tmpResp, _, _ := c.AddItem(&database.InventorySlot{ItemID: 90000146, Quantity: 1}, -1, false)
				resp.Concat(*tmpResp)
				tmpResp, _, _ = c.AddItem(&database.InventorySlot{ItemID: 90000147, Quantity: 1}, -1, false)
				resp.Concat(*tmpResp)
				tmpResp, _, _ = c.AddItem(&database.InventorySlot{ItemID: 90000148, Quantity: 1}, -1, false)
				resp.Concat(*tmpResp)
			}
			return resp, nil
		case 1383:
			_, loveItem, err := c.FindItemInInventory(nil, 18500852)
			if err != nil {
				return nil, err
			} else if loveItem == nil {
				resp := messaging.InfoMessage("Missing item.")
				return resp, nil
			}

			if loveItem == nil {
				resp := messaging.InfoMessage("Missing item.")
				return resp, nil
			} else {
				resp.Concat(*c.DecrementItem(loveItem.SlotID, 1))

				tmpResp, _, _ := c.AddItem(&database.InventorySlot{ItemID: 30003005, Quantity: 1}, -1, false)
				resp.Concat(*tmpResp)
				tmpResp, _, _ = c.AddItem(&database.InventorySlot{ItemID: 30003105, Quantity: 1}, -1, false)
				resp.Concat(*tmpResp)
				tmpResp, _, _ = c.AddItem(&database.InventorySlot{ItemID: 30003205, Quantity: 1}, -1, false)
				resp.Concat(*tmpResp)
				tmpResp, _, _ = c.AddItem(&database.InventorySlot{ItemID: 30003305, Quantity: 1}, -1, false)
				resp.Concat(*tmpResp)
				return resp, nil
			}
		case 1384:
			_, cryptic1Item, err := c.FindItemInInventory(nil, 18500853)
			if err != nil {
				return nil, err
			} else if cryptic1Item == nil {
				resp := messaging.InfoMessage("Missing item.")
				return resp, nil
			}

			_, cryptic2Item, err := c.FindItemInInventory(nil, 18500854)
			if err != nil {
				return nil, err
			} else if cryptic2Item == nil {
				resp := messaging.InfoMessage("Missing item.")
				return resp, nil
			}

			_, cryptic3Item, err := c.FindItemInInventory(nil, 18500855)
			if err != nil {
				return nil, err
			} else if cryptic3Item == nil {
				resp := messaging.InfoMessage("Missing item.")
				return resp, nil
			}

			if cryptic1Item == nil || cryptic2Item == nil || cryptic3Item == nil {
				resp := messaging.InfoMessage("Missing item.")
				return resp, nil
			} else {
				resp.Concat(*c.DecrementItem(cryptic1Item.SlotID, 1))
				resp.Concat(*c.DecrementItem(cryptic2Item.SlotID, 1))
				resp.Concat(*c.DecrementItem(cryptic3Item.SlotID, 1))

				if c.Type == 53 || c.Type == 56 {
					tmpResp, _, _ := c.AddItem(&database.InventorySlot{ItemID: 99050001, Quantity: 7200}, -1, false)
					resp.Concat(*tmpResp)
					tmpResp, _, _ = c.AddItem(&database.InventorySlot{ItemID: 99032002, Quantity: 7200}, -1, false)
					resp.Concat(*tmpResp)
					tmpResp, _, _ = c.AddItem(&database.InventorySlot{ItemID: 99033002, Quantity: 7200}, -1, false)
					resp.Concat(*tmpResp)
				} else if c.Type == 54 || c.Type == 57 || c.Type == 59 {
					tmpResp, _, _ := c.AddItem(&database.InventorySlot{ItemID: 99051001, Quantity: 7200}, -1, false)
					resp.Concat(*tmpResp)
					tmpResp, _, _ = c.AddItem(&database.InventorySlot{ItemID: 99042002, Quantity: 7200}, -1, false)
					resp.Concat(*tmpResp)
					tmpResp, _, _ = c.AddItem(&database.InventorySlot{ItemID: 99043002, Quantity: 7200}, -1, false)
					resp.Concat(*tmpResp)
				}

				return resp, nil
			}
		case 1385:
			_, item1, err := c.FindItemInInventory(nil, 18500856)
			if err != nil {
				return nil, err
			} else if item1 == nil {
				resp := messaging.InfoMessage("Missing item.")
				return resp, nil
			}

			_, item2, err := c.FindItemInInventory(nil, 18500857)
			if err != nil {
				return nil, err
			} else if item2 == nil {
				resp := messaging.InfoMessage("Missing item.")
				return resp, nil
			}

			if item1 == nil || item2 == nil {
				resp := messaging.InfoMessage("Missing item.")
				return resp, nil
			} else {
				resp.Concat(*c.DecrementItem(item1.SlotID, 1))
				resp.Concat(*c.DecrementItem(item2.SlotID, 1))

				if c.Type == 53 {
					tmpResp, _, _ := c.AddItem(&database.InventorySlot{ItemID: 99003201, Quantity: 1}, -1, false)
					resp.Concat(*tmpResp)
					tmpResp, _, _ = c.AddItem(&database.InventorySlot{ItemID: 99003101, Quantity: 1}, -1, false)
					resp.Concat(*tmpResp)
					tmpResp, _, _ = c.AddItem(&database.InventorySlot{ItemID: 99003301, Quantity: 1}, -1, false)
					resp.Concat(*tmpResp)
				} else if c.Type == 54 {
					tmpResp, _, _ := c.AddItem(&database.InventorySlot{ItemID: 99003201, Quantity: 1}, -1, false)
					resp.Concat(*tmpResp)
					tmpResp, _, _ = c.AddItem(&database.InventorySlot{ItemID: 99003101, Quantity: 1}, -1, false)
					resp.Concat(*tmpResp)
					tmpResp, _, _ = c.AddItem(&database.InventorySlot{ItemID: 99003401, Quantity: 1}, -1, false)
					resp.Concat(*tmpResp)
				} else if c.Type == 56 {
					tmpResp, _, _ := c.AddItem(&database.InventorySlot{ItemID: 99003601, Quantity: 1}, -1, false)
					resp.Concat(*tmpResp)
					tmpResp, _, _ = c.AddItem(&database.InventorySlot{ItemID: 99003801, Quantity: 1}, -1, false)
					resp.Concat(*tmpResp)
					tmpResp, _, _ = c.AddItem(&database.InventorySlot{ItemID: 99003501, Quantity: 1}, -1, false)
					resp.Concat(*tmpResp)
				} else if c.Type == 57 {
					tmpResp, _, _ := c.AddItem(&database.InventorySlot{ItemID: 99003601, Quantity: 1}, -1, false)
					resp.Concat(*tmpResp)
					tmpResp, _, _ = c.AddItem(&database.InventorySlot{ItemID: 99003701, Quantity: 1}, -1, false)
					resp.Concat(*tmpResp)
					tmpResp, _, _ = c.AddItem(&database.InventorySlot{ItemID: 99003501, Quantity: 1}, -1, false)
					resp.Concat(*tmpResp)
				} else if c.Type == 59 {
					tmpResp, _, _ := c.AddItem(&database.InventorySlot{ItemID: 99002101, Quantity: 1}, -1, false)
					resp.Concat(*tmpResp)
					tmpResp, _, _ = c.AddItem(&database.InventorySlot{ItemID: 99002301, Quantity: 1}, -1, false)
					resp.Concat(*tmpResp)
				}

				return resp, nil
			}
		case 1386: // PP Exchange
			slotID, item, err := c.FindItemInInventoryByPlus(nil, 1, 242)
			if err != nil {
				return nil, err
			} else if slotID == -1 { // no same item found => find free slot
				return nil, err
			}

			if item.Quantity%2 == 0 {
				tmpData, _, err := c.AddItem(&database.InventorySlot{ItemID: 242, Quantity: item.Quantity / 2, Plus: 2, UpgradeArr: "{242,242,0,0,0,0,0,0,0,0,0,0,0,0,0}"}, -1, true)
				if err != nil {
					return nil, err
				} else if tmpData == nil {
					return nil, nil
				}

				resp.Concat(*tmpData)
				resp.Concat(*c.DecrementItem(slotID, item.Quantity))
			}
		case 1387:
			slotID, item, err := c.FindItemInInventoryByPlus(nil, 2, 242)
			if err != nil {
				return nil, err
			} else if slotID == -1 { // no same item found => find free slot
				return nil, err
			}

			if item.Quantity%2 == 0 {
				tmpData, _, err := c.AddItem(&database.InventorySlot{ItemID: 242, Quantity: item.Quantity / 2, Plus: 3, UpgradeArr: "{242,242,242,0,0,0,0,0,0,0,0,0,0,0,0}"}, -1, true)
				if err != nil {
					return nil, err
				} else if tmpData == nil {
					return nil, nil
				}

				resp.Concat(*tmpData)
				resp.Concat(*c.DecrementItem(slotID, item.Quantity))
			}
		case 1388:
			slotID, item, err := c.FindItemInInventoryByPlus(nil, 3, 242)
			if err != nil {
				return nil, err
			} else if slotID == -1 { // no same item found => find free slot
				return nil, err
			}

			if item.Quantity%2 == 0 {
				tmpData, _, err := c.AddItem(&database.InventorySlot{ItemID: 242, Quantity: item.Quantity / 2, Plus: 4, UpgradeArr: "{242,242,242,242,0,0,0,0,0,0,0,0,0,0,0}"}, -1, true)
				if err != nil {
					return nil, err
				} else if tmpData == nil {
					return nil, nil
				}

				resp.Concat(*tmpData)
				resp.Concat(*c.DecrementItem(slotID, item.Quantity))
			}
		case 1389:
			slotID, item, err := c.FindItemInInventoryByPlus(nil, 4, 242)
			if err != nil {
				return nil, err
			} else if slotID == -1 { // no same item found => find free slot
				return nil, err
			}

			if item.Quantity%2 == 0 {
				tmpData, _, err := c.AddItem(&database.InventorySlot{ItemID: 242, Quantity: item.Quantity / 2, Plus: 5, UpgradeArr: "{242,242,242,242,242,0,0,0,0,0,0,0,0,0,0}"}, -1, true)
				if err != nil {
					return nil, err
				} else if tmpData == nil {
					return nil, nil
				}

				resp.Concat(*tmpData)
				resp.Concat(*c.DecrementItem(slotID, item.Quantity))
			}
		case 1390:
			if c.Level < 50 {
				resp := messaging.InfoMessage("You are not at 50 Level.")
				return resp, nil
			}

			if database.GoldenBasinArea.FactionID == 0 {
				if database.CanJoinGoldenBasin {
					if c.Faction == 1 {
						x := 75.0
						y := 431.0
						data, _ := c.ChangeMap(76, database.ConvertPointToLocation(fmt.Sprintf("%.1f,%.1f", x, y)))
						resp.Concat(data)
					}

					if c.Faction == 2 {
						x := 430.0
						y := 75.0
						data, _ := c.ChangeMap(76, database.ConvertPointToLocation(fmt.Sprintf("%.1f,%.1f", x, y)))
						resp.Concat(data)
					}
				} else {
					resp := messaging.InfoMessage("Golden Basin war not started yet or already finished.")
					return resp, nil
				}
			} else {
				if c.Faction != database.GoldenBasinArea.FactionID {
					resp := messaging.InfoMessage("Golden Basin is currently on the enemy side.")
					return resp, nil
				} else {
					if c.Faction == 1 {
						x := 75.0
						y := 431.0
						data, _ := c.ChangeMap(76, database.ConvertPointToLocation(fmt.Sprintf("%.1f,%.1f", x, y)))
						resp.Concat(data)
					}

					if c.Faction == 2 {
						x := 430.0
						y := 75.0
						data, _ := c.ChangeMap(76, database.ConvertPointToLocation(fmt.Sprintf("%.1f,%.1f", x, y)))
						resp.Concat(data)
					}
				}
			}
		case 1391:
			return c.ChangeMap(236, nil)
		case 1392:
			_, item, err := c.FindItemInInventory(nil, 15700040, 15710087, 17200452)
			if err != nil {
				return nil, err
			} else if item == nil { // You don't have ticket
				resp := GetNPCMenu(npcID, 999993, 0, nil)
				return resp, nil
			}

			if item.Activated {
				if c.RebornLevel == 0 {
					if c.Level < 85 {
						return nil, nil
					}
				} else {
					if c.Level < 95 {
						return nil, nil
					}
				}
				resp, _ := c.ChangeMap(78, nil)
				return resp, nil
			}
		case 1393:
			resp := messaging.InfoMessage("Will lead to new Reborn map and when u click it u can put some info like: Soon stronger player can continiue his journey from here.")
			return resp, nil
		case 1394:
			buff, err := database.FindBuffByID(70024, c.ID)
			if err != nil || buff != nil {
				return nil, nil
			}

			buffinfo := database.BuffInfections[70024]
			buff = &database.Buff{ID: 70024, CharacterID: c.ID, Name: buffinfo.Name, BagExpansion: false, StartedAt: c.Epoch, Duration: int64(1440) * 60}
			err = buff.Create()
			if err != nil {
				fmt.Println("1-. ", err)
				return nil, nil
			}

			tmpResp, _, _ := c.AddItem(&database.InventorySlot{ItemID: 17502113, Quantity: 120}, -1, false)
			resp.Concat(*tmpResp)
			tmpResp, _, _ = c.AddItem(&database.InventorySlot{ItemID: 17502445, Quantity: 120}, -1, false)
			resp.Concat(*tmpResp)

			resp.Concat(messaging.InfoMessage("We are so happy with you."))
			return resp, nil
		case 1395:
			resp = OPEN_SHOP
			resp.Insert(utils.IntToBytes(uint64(351), 4, true), 7) // shop id
			return resp, nil
		case 3087:
			if c.Map == 17 {
				if c.PartyID == "" {
					resp = GetNPCMenu(npcID, 133702, 0, nil)
				} else {
					party := database.FindParty(c)
					if party.Leader.ID != c.ID {
						resp = GetNPCMenu(npcID, 133703, 0, nil)
						return resp, nil
					}
					if len(party.Members) > 3 {
						msg := messaging.InfoMessage("Maximum 3 players can entry at once.")
						party.Leader.Socket.Write(msg)
						return nil, nil
					}

					if dungeon.IsDungeonClosed {
						msg := messaging.InfoMessage("All dungeons are full at this moment, come back later.")
						party.Leader.Socket.Write(msg)
						return nil, nil
					}

					if c.Level < 70 {
						resp = GetNPCMenu(npcID, 133705, 0, nil)
						return resp, nil
					}

					database.CountYingYangMobs(243)
					database.YingYangMobsMutex.Lock()
					yingYangMobs := database.YingYangMobsCounter[243]
					database.YingYangMobsMutex.Unlock()

					if yingYangMobs.BlackBandits < 50 || yingYangMobs.Rogues < 50 || yingYangMobs.Ghosts < 50 || yingYangMobs.Animals < 50 || yingYangMobs.BeastMaster != 1 || yingYangMobs.Paechun != 1 {
						msg := messaging.InfoMessage("Dungeon is not ready at this moment, come back later.")
						party.Leader.Socket.Write(msg)
						return nil, nil
					}

					slot, _, err := c.FindItemInInventory(nil, 99002475)
					if err != nil {
						resp = GetNPCMenu(npcID, 133704, 0, nil)
						return resp, nil
					} else if slot == -1 {
						resp = GetNPCMenu(npcID, 133704, 0, nil)
						return resp, nil
					}

					for _, member := range party.Members {
						if member.Character.Level < 70 { //
							resp = GetNPCMenu(npcID, 133705, 0, nil)
							return resp, nil
						}

						tmpSlot, _, err := member.FindItemInInventory(nil, 99002475)
						if err != nil {
							resp = GetNPCMenu(npcID, 133704, 0, nil)
							return resp, nil
						} else if tmpSlot == -1 {
							resp = GetNPCMenu(npcID, 133704, 0, nil)
							return resp, nil
						}

						decResp := member.DecrementItem(tmpSlot, uint(1))
						//member.Socket.Conn.Write(*decResp)
						member.Socket.Write(*decResp)
					}

					decResp := c.DecrementItem(slot, uint(1))
					c.Socket.Write(*decResp)
					go dungeon.StartYingYang(party)
				}
			} /*else if c.Map == 24 {
				if c.PartyID == "" {
					resp = GetNPCMenu(npcID, 133702, 0, nil)
				} else {
					party := database.FindParty(c)
					if party.Leader.ID != c.ID {
						resp = GetNPCMenu(npcID, 133703, 0, nil)
						return resp, nil
					}
					if len(party.Members) > 3 {
						msg := messaging.InfoMessage("Maximum 3 players can entry at once.")
						party.Leader.Socket.Write(msg)
						return nil, nil
					}

					if c.YingYangTicketsLeft <= 0 {
						resp = GetNPCMenu(npcID, 133704, 0, nil)
						return resp, nil
					} else if c.Level < 101 || c.Level > 200 {
						resp = GetNPCMenu(npcID, 133705, 0, nil)
						return resp, nil
					}

					for _, member := range party.Members {

						if member.Character.YingYangTicketsLeft <= 0 {
							resp = GetNPCMenu(npcID, 133704, 0, nil)
							return resp, nil
						} else if member.Character.Level < 101 || member.Character.Level > 200 {
							resp = GetNPCMenu(npcID, 133705, 0, nil)
							return resp, nil
						}

						member.Character.YingYangTicketsLeft--
						member.Character.Update()
					}
					c.YingYangTicketsLeft--
					c.Update()
					go dungeon.StartDivineYingYang(party)
				}
			} */
		case 3088:
			if c.YingYangTicketsLeft {
				itemData, _, err := c.AddItem(&database.InventorySlot{ItemID: 99002475, Quantity: 3}, -1, false)
				if err != nil {
					return nil, err
				}

				c.YingYangTicketsLeft = false
				c.Update()

				return *itemData, nil
			}
			return nil, nil
		case 3103:
			shopNo := shops[npcID]
			resp = OPEN_SHOP
			resp.Insert(utils.IntToBytes(uint64(shopNo), 4, true), 7) // shop id
		case 197101: // Move to Marketplace
			resp, _ = c.ChangeMap(254, nil)
		case 197102:
			x := 321.0
			y := 339.0
			resp, _ = c.ChangeMap(254, database.ConvertPointToLocation(fmt.Sprintf("%.1f,%.1f", x, y)))

		case 1801:
			if c.Exp == 233332051410 && c.Level == 100 {
				r, _ := c.ChangeMap(43, nil)
				resp.Concat(r)
			} else {
				resp.Concat(messaging.InfoMessage(fmt.Sprintf("You don't meet the requirements")))
			}
		}

		return resp, nil
	}

}

func GetNPCMenu(npcID, textID, index int, actions []int) []byte {
	resp := NPC_MENU
	resp.Insert(utils.IntToBytes(uint64(npcID), 4, true), 6)         // npc id
	resp.Insert(utils.IntToBytes(uint64(textID), 4, true), 10)       // text id
	resp.Insert(utils.IntToBytes(uint64(len(actions)), 1, true), 14) // action length

	counter, length := 15, int16(11)
	for i, action := range actions {
		resp.Insert(utils.IntToBytes(uint64(action), 4, true), counter) // action
		counter += 4

		resp.Insert(utils.IntToBytes(uint64(npcID), 2, true), counter) // npc id
		counter += 2

		actIndex := int(index) + (i+1)<<(len(actions)*3)
		resp.Insert(utils.IntToBytes(uint64(actIndex), 2, true), counter) // action index
		counter += 2

		length += 8
	}

	resp.SetLength(length)
	return resp
}

func firstJobPromotion(c *database.Character, book, job, npcID int) (utils.Packet, error) {
	resp := utils.Packet{}
	if c.Class == 0 && c.Level >= 10 {
		c.Class = job
		resp = JOB_PROMOTED
		resp[6] = byte(job)

		r, _, err := c.AddItem(&database.InventorySlot{ItemID: int64(book), Quantity: 1}, -1, false)
		if err != nil {
			return resp, err
		} else if r == nil {
			return nil, nil
		}

		resp.Concat(*r)

	} else if c.Level < 10 {
		resp = NOT_ENOUGH_LEVEL
		resp.Insert(utils.IntToBytes(uint64(npcID), 4, true), 6) // npc id
	} else {
		resp = INVALID_CLASS
		resp.Insert(utils.IntToBytes(uint64(npcID), 4, true), 6) // npc id
	}

	return resp, nil
}

func secondJobPromotion(c *database.Character, book1, book2, preJob, job, npcID int) (utils.Packet, error) {
	resp := utils.Packet{}
	if c.Class == preJob && c.Level >= 50 {
		c.Class = job
		resp = JOB_PROMOTED
		resp[6] = byte(job)

		r, _, err := c.AddItem(&database.InventorySlot{ItemID: int64(book1), Quantity: 1}, -1, false)
		if err != nil {
			return resp, err
		} else if r == nil {
			return nil, nil
		}

		resp.Concat(*r)

		r, _, err = c.AddItem(&database.InventorySlot{ItemID: int64(book2), Quantity: 1}, -1, false)
		if err != nil {
			return resp, err
		} else if r == nil {
			return nil, nil
		}

		resp.Concat(*r)

	} else if c.Level < 50 {
		resp := NOT_ENOUGH_LEVEL
		resp.Insert(utils.IntToBytes(uint64(npcID), 4, true), 6) // npc id
	} else {
		resp = INVALID_CLASS
		resp.Insert(utils.IntToBytes(uint64(npcID), 4, true), 6) // npc id
	}

	return resp, nil
}

func divineJobPromotion(c *database.Character, npcID int) (utils.Packet, error) {
	resp := utils.Packet{}
	if c.Class != 0 {
		jobName := ""
		book1 := 0
		book2 := 0
		book3 := 0
		if c.Class == 21 || c.Class == 22 { //WARLORD
			c.Class = 31
			book1 = 100031001
			book2 = 100030013
			book3 = 16100300
			jobName = "Warlord"
		} else if c.Class == 23 || c.Class == 24 { //Holy Hand
			c.Class = 32
			book1 = 100031003
			book2 = 100030015
			book3 = 16100300
			jobName = "HolyHand"
		} else if c.Class == 25 || c.Class == 26 { //BeastLord
			c.Class = 33
			book1 = 100031002
			book2 = 100030014
			book3 = 16100300
			jobName = "BeastLord"
		} else if c.Class == 27 || c.Class == 28 { //ShadowLord
			c.Class = 34
			book1 = 100031004
			book2 = 100030016
			book3 = 16100300
			jobName = "ShadowLord"
		}
		c.Update()
		resp = JOB_PROMOTED
		resp[6] = byte(c.Class)

		r, _, err := c.AddItem(&database.InventorySlot{ItemID: int64(book1), Quantity: 1}, -1, false)
		if err != nil {
			return resp, err
		} else if r == nil {
			return nil, nil
		}

		resp.Concat(*r)

		r, _, err = c.AddItem(&database.InventorySlot{ItemID: int64(book2), Quantity: 1}, -1, false)
		if err != nil {
			return resp, err
		} else if r == nil {
			return nil, nil
		}

		resp.Concat(*r)

		r, _, err = c.AddItem(&database.InventorySlot{ItemID: int64(book3), Quantity: 1}, -1, false)
		if err != nil {
			return resp, err
		} else if r == nil {
			return nil, nil
		}

		resp.Concat(*r)

		resp.Concat(messaging.InfoMessage(fmt.Sprintf("Promoted as a %s.", jobName))) //NOTICE TO PROMOTE
		//r = c.ChangeMap(1, nil)
	} else {
		resp = INVALID_CLASS
		resp.Insert(utils.IntToBytes(uint64(npcID), 4, true), 6) // npc id
	}
	return resp, nil
}

func darknessJobPromotion(c *database.Character, book1, book2, book3, job, npcID int, jobName string) (utils.Packet, error) {
	resp := utils.Packet{}
	if c.Class == 40 {
		c.Class = job
		c.Update()
		resp = JOB_PROMOTED
		resp[6] = byte(c.Class)

		r, _, err := c.AddItem(&database.InventorySlot{ItemID: int64(book1), Quantity: 1}, -1, false)
		if err != nil {
			return resp, err
		} else if r == nil {
			return nil, nil
		}

		resp.Concat(*r)

		r, _, err = c.AddItem(&database.InventorySlot{ItemID: int64(book2), Quantity: 1}, -1, false)
		if err != nil {
			return resp, err
		} else if r == nil {
			return nil, nil
		}

		resp.Concat(*r)

		r, _, err = c.AddItem(&database.InventorySlot{ItemID: int64(book3), Quantity: 1}, -1, false)
		if err != nil {
			return resp, err
		} else if r == nil {
			return nil, nil
		}

		resp.Concat(*r)
		c.Update()
		resp.Concat(messaging.InfoMessage(fmt.Sprintf("Promoted as a %s.", jobName))) //NOTICE TO PROMOTE
		//r = c.ChangeMap(1, nil)
	} else {
		resp = INVALID_CLASS
		resp.Insert(utils.IntToBytes(uint64(npcID), 4, true), 6) // npc id
	}
	return resp, nil
}
