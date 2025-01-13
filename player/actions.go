package player

import (
	"encoding/binary"
	"hero-server/database"
	"hero-server/messaging"
	"hero-server/nats"
	"hero-server/utils"
	"log"
)

type (
	BattleModeHandler        struct{}
	MeditationHandler        struct{}
	TargetSelectionHandler   struct{}
	TravelToCastleHandler    struct{}
	OpenTacticalSpaceHandler struct{}
	TacticalSpaceTPHandler   struct{}
	InTacticalSpaceTPHandler struct{}
	OpenLotHandler           struct{}
	EnterGateHandler         struct{}
	SendPvPRequestHandler    struct{}
	RespondPvPRequestHandler struct{}
	TransferItemTypeHandler  struct{}
	ClothImproveChest        struct{}
	CharmOfIdentity          struct{}
	TravelToFiveClanArea     struct{}
	EnhancementTransfer      struct{}
	ChangePetName            struct{}
	//TransferSoulHandler      struct{}
)

var (
	FreeLotQuantities = map[int]int{10820001: 1, 10600033: 10, 10600036: 10, 17500346: 1, 221: 1, 222: 1, 223: 1, 17500023: 1, 99009121: 1, 92000013: 1}
	PaidLotQuantities = map[int]int{92000001: 1, 92000011: 1, 10820001: 1, 17500346: 1, 10601023: 10, 10601024: 10, 92000012: 1, 17500023: 1, 99009121: 1, 18500033: 1, 18500034: 1, 253: 1, 240: 1, 241: 1, 10002: 1}

	BATTLE_MODE         = utils.Packet{0xAA, 0x55, 0x04, 0x00, 0x43, 0x00, 0x55, 0xAA}
	MEDITATION_MODE     = utils.Packet{0xAA, 0x55, 0x05, 0x00, 0x82, 0x05, 0x00, 0x55, 0xAA}
	TACTICAL_SPACE_MENU = utils.Packet{0xAA, 0x55, 0x03, 0x00, 0x50, 0x01, 0x01, 0x55, 0xAA, 0xAA, 0x55, 0x05, 0x00, 0x28, 0xFF, 0x00, 0x00, 0x00, 0x55, 0xAA}
	TACTICAL_SPACE_TP   = utils.Packet{0xAA, 0x55, 0x07, 0x00, 0x01, 0xB9, 0x0A, 0x00, 0x00, 0x00, 0x01, 0x55, 0xAA}
	OPEN_LOT            = utils.Packet{0xAA, 0x55, 0x0C, 0x00, 0xA2, 0x01, 0x32, 0x00, 0x00, 0x00, 0x00, 0x01, 0x55, 0xAA}
	SELECTION_CHANGED   = utils.Packet{0xAA, 0x55, 0x09, 0x00, 0xCF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x55, 0xAA}
	PVP_REQUEST         = utils.Packet{0xAA, 0x55, 0x04, 0x00, 0x2A, 0x01, 0x55, 0xAA}
	PVP_STARTED         = utils.Packet{0xAA, 0x55, 0x0A, 0x00, 0x2A, 0x02, 0x55, 0xAA}

	CLANCASTLE_MAP = utils.Packet{0xaa, 0x55, 0x62, 0x00, 0xbb, 0x03, 0x05, 0x55, 0xAA}
	CANNOT_MOVE    = utils.Packet{0xaa, 0x55, 0x04, 0x00, 0xbb, 0x02, 0x00, 0x00, 0x55, 0xaa}
)

func (h *BattleModeHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {

	if s.User == nil || s.Character == nil {
		s.OnClose()
		return nil, nil
	}

	battleMode := data[5]

	resp := BATTLE_MODE
	resp.Insert(utils.IntToBytes(uint64(s.Character.PseudoID), 2, true), 5) // character pseudo id
	resp[7] = battleMode

	p := nats.CastPacket{CastNear: true, CharacterID: s.Character.ID, Type: nats.BATTLE_MODE, Data: resp}
	if err := p.Cast(); err != nil {
		return nil, err
	}

	return resp, nil
}

func (h *MeditationHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {

	if s.User == nil || s.Character == nil {
		s.OnClose()
		return nil, nil
	}

	meditationMode := data[6] == 1
	s.Character.Meditating = meditationMode

	resp := MEDITATION_MODE
	resp.Insert(utils.IntToBytes(uint64(s.Character.PseudoID), 2, true), 6) // character pseudo id
	resp[8] = data[6]

	p := nats.CastPacket{CastNear: true, CharacterID: s.Character.ID, Type: nats.MEDITATION_MODE, Data: resp}
	if err := p.Cast(); err != nil {
		return nil, err
	}

	return resp, nil
}

func (h *TargetSelectionHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {

	if s.User == nil || s.Character == nil {
		s.OnClose()
		return nil, nil
	}

	id := int(utils.BytesToInt(data[5:7], true))

	char := database.FindCharacterByPseudoID(s.User.ConnectedServer, uint16(id))
	if char != nil {
		if char.Socket.Stats.HP <= 0 {
			return nil, nil
		}
	}

	s.Character.Selection = id

	resp := SELECTION_CHANGED
	resp.Insert(utils.IntToBytes(uint64(s.Character.Selection), 2, true), 5)
	return resp, nil
}

func (h *TravelToCastleHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {

	if s.User == nil || s.Character == nil {
		s.OnClose()
		return nil, nil
	}

	if s.Stats.HP <= 0 {
		return nil, nil
	}

	if s.Character.Level < 50 {
		return nil, nil
	}

	if s.Character.Map == 233 {
		resp := CLANCASTLE_MAP
		index := 7
		length := 3
		if database.FiveClans[1].ClanID != 0 {
			//FLAME, WATERFALL, SKY GARDEN, FOREST,UNDERGROUND
			resp.Insert([]byte{0x01, 0xdf, 0x04, 0x00, 0x00}, index)
			index += 5
			length += 5
			area, _ := database.FindGuildByID(database.FiveClans[1].ClanID) //FLAME WOLF TEMPLE
			resp.Insert([]byte{byte(len(area.Name))}, index)                // Guild name length
			index++
			resp.Insert([]byte(area.Name), index) // Guild name
			index += len(area.Name)
			length += 1 + len(area.Name)
		}
		if database.FiveClans[2].ClanID != 0 {
			resp.Insert([]byte{0x02, 0xeb, 0x00, 0x00, 0x00}, index)
			index += 5
			length += 5
			area, _ := database.FindGuildByID(database.FiveClans[2].ClanID) //OCEAN ARMY
			resp.Insert([]byte{byte(len(area.Name))}, index)                // Guild name length
			index++
			resp.Insert([]byte(area.Name), index) // Guild name
			index += len(area.Name)
			length += 1 + len(area.Name)
		}
		if database.FiveClans[3].ClanID != 0 {
			resp.Insert([]byte{0x03, 0x5d, 0x06, 0x00, 0x00}, index)
			index += 5
			length += 5
			area, _ := database.FindGuildByID(database.FiveClans[3].ClanID) //LIGHTNING HILL
			resp.Insert([]byte{byte(len(area.Name))}, index)                // Guild name length
			index++
			resp.Insert([]byte(area.Name), index) // Guild name
			index += len(area.Name)
			length += 1 + len(area.Name)
		}
		if database.FiveClans[4].ClanID != 0 {
			resp.Insert([]byte{0x04, 0xf0, 0x06, 0x00, 0x00}, index)
			index += 5
			length += 5
			area, _ := database.FindGuildByID(database.FiveClans[4].ClanID) //SOUTHERN WOOD TEMPLE
			resp.Insert([]byte{byte(len(area.Name))}, index)                // Guild name length
			index++
			resp.Insert([]byte(area.Name), index) // Guild name
			index += len(area.Name)
			length += 1 + len(area.Name)
		}
		if database.FiveClans[5].ClanID != 0 {
			resp.Insert([]byte{0x05, 0xd7, 0x05, 0x00, 0x00}, index)
			index += 5
			length += 5
			area, _ := database.FindGuildByID(database.FiveClans[5].ClanID) //WESTERN LAND TEMPLE
			resp.Insert([]byte{byte(len(area.Name))}, index)                // Guild name length
			index++
			resp.Insert([]byte(area.Name), index) // Guild name
			index += len(area.Name)
			length += 1 + len(area.Name)
		}

		resp.SetLength(int16(binary.Size(resp) - 6))
		return resp, nil
	}

	return s.Character.ChangeMap(233, nil)
}

func (h *TravelToFiveClanArea) Handle(s *database.Socket, data []byte) ([]byte, error) {

	if s.User == nil || s.Character == nil {
		s.Conn.Close()
		return nil, nil
	}

	if s.Stats.HP <= 0 {
		return nil, nil
	}

	areaID := int16(data[7])
	switch areaID {
	case 0:
		x := "508,564"
		coord := s.Character.Teleport(database.ConvertPointToLocation(x))
		//s.Conn.Write(coord)
		s.Write(coord)
		//s.Write(coord)
		/*
			case 1: //FLAME WOLF TEMPLE
				if s.Character.GuildID == database.FiveClans[1].ClanID {
					x := "243,777"
					coord := s.Character.Teleport(database.ConvertPointToLocation(x))
					s.Conn.Write(coord)
				} else {
					s.Conn.Write(CANNOT_MOVE)
				}
			case 2: //OCEAN ARMY
				if s.Character.GuildID == database.FiveClans[2].ClanID {
					x := "131,433"
					coord := s.Character.Teleport(database.ConvertPointToLocation(x))
					s.Conn.Write(coord)
				} else {
					s.Conn.Write(CANNOT_MOVE)
				}
			case 3: //LIGHTNING HILL
				if s.Character.GuildID == database.FiveClans[3].ClanID {
					x := "615,171"
					coord := s.Character.Teleport(database.ConvertPointToLocation(x))
					s.Conn.Write(coord)
				} else {
					s.Conn.Write(CANNOT_MOVE)
				}
			case 4: //SOUTHERN WOOD TEMPLE
				if s.Character.GuildID == database.FiveClans[4].ClanID {
					x := "863,425"
					coord := s.Character.Teleport(database.ConvertPointToLocation(x))
					s.Conn.Write(coord)
				} else {
					s.Conn.Write(CANNOT_MOVE)
				}
			case 5: //WESTERN LAND TEMPLE
				if s.Character.GuildID == database.FiveClans[5].ClanID {
					x := "689,867"
					coord := s.Character.Teleport(database.ConvertPointToLocation(x))
					s.Conn.Write(coord)
				} else {
					s.Conn.Write(CANNOT_MOVE)
				}
		*/
	}

	return nil, nil
}

func (h *OpenTacticalSpaceHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {

	if s.User == nil || s.Character == nil {
		s.OnClose()
		return nil, nil
	}
	return TACTICAL_SPACE_MENU, nil
}

func (h *TacticalSpaceTPHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {

	if s.User == nil || s.Character == nil {
		s.OnClose()
		return nil, nil
	}
	mapID := int16(data[6])
	return s.Character.ChangeMap(mapID, nil)
}

func (h *InTacticalSpaceTPHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {

	if s.User == nil || s.Character == nil {
		s.OnClose()
		return nil, nil
	}
	resp := TACTICAL_SPACE_TP
	resp[8] = data[6]
	return resp, nil
}

func (h *OpenLotHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {

	if s.User == nil || s.Character == nil {
		s.OnClose()
		return nil, nil
	}

	if !s.Character.HasLot {
		return nil, nil
	}

	s.Character.HasLot = false
	paid := data[5] == 1
	dropID := 1185

	if paid && s.Character.Gold >= 150000 {
		dropID = 1186
		s.Character.Gold -= 150000
		go s.Character.GetGold()
	}

	drop, ok := database.Drops[dropID]
	if drop == nil {
		return nil, nil
	}

	resp := OPEN_LOT
	itemID := 0
	for ok {
		index := 0
		seed := int(utils.RandInt(0, 1000))
		items := drop.GetItems()
		probabilities := drop.GetProbabilities()

		for _, prob := range probabilities {
			// 783
			if float64(seed) > float64(prob) {
				index++
				continue
			}
			break
		}

		if index >= len(items) {
			break
		}

		itemID = items[index]
		drop, ok = database.Drops[itemID]
	}

	if itemID == 10002 {
		s.User.NCash += 150
		go s.User.Update()

	} else {
		quantity := 1
		if paid {
			if q, ok := PaidLotQuantities[itemID]; ok {
				quantity = q
			}
		} else {
			if q, ok := FreeLotQuantities[itemID]; ok {
				quantity = q
			}
		}

		info := database.Items[int64(itemID)]
		if info == nil {
			return nil, nil
		}
		if info.Timer > 0 {
			quantity = info.Timer
		}

		item := &database.InventorySlot{ItemID: int64(itemID), Quantity: uint(quantity)}
		r, _, err := s.Character.AddItem(item, -1, false)
		if err != nil {
			return nil, err
		} else if r == nil {
			return nil, nil
		}

		resp.Concat(*r)
	}

	resp.Insert(utils.IntToBytes(uint64(itemID), 4, true), 11) // item id
	return resp, nil
}

func (h *EnterGateHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {

	if s.User == nil || s.Character == nil {
		s.OnClose()
		return nil, nil
	}

	if len(data) < 9 {
		return s.Character.ChangeMap(int16(s.Character.Map), nil)
	}

	gateID := int(utils.BytesToInt(data[5:9], true))
	if gateID == 0 {
		return nil, nil
	}

	database.GatesMutex.Lock()
	gate, ok := database.Gates[gateID]
	database.GatesMutex.Unlock()
	if gate == nil { // 01.12.2023 // BUG FIX
		return s.Character.ChangeMap(int16(s.Character.Map), nil)
	}
	if !ok {
		return s.Character.ChangeMap(int16(s.Character.Map), nil)
	}

	if gate.TargetMap == 14 || gate.TargetMap == 15 {
		if s.Character.Faction == 2 {
			gate.TargetMap = 15
		}

		if s.Character.Faction == 1 {
			gate.TargetMap = 14
		}
	}

	coordinate := database.ConvertPointToLocation(gate.Point)
	return s.Character.ChangeMap(int16(gate.TargetMap), coordinate)
}

func (h *SendPvPRequestHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {

	if s.User == nil || s.Character == nil {
		s.OnClose()
		return nil, nil
	}

	pseudoID := uint16(utils.BytesToInt(data[6:8], true))
	opponent := database.FindCharacterByPseudoID(s.User.ConnectedServer, pseudoID)
	if opponent == nil {
		return nil, nil

	} else if opponent.DuelID > 0 {
		resp := messaging.SystemMessage(messaging.ALREADY_IN_PVP)
		return resp, nil
	}

	resp := PVP_REQUEST
	resp.Insert(utils.IntToBytes(uint64(s.Character.PseudoID), 2, true), 6) // sender pseudo id

	database.GetSocket(opponent.UserID).Write(resp)
	return nil, nil
}

func (h *RespondPvPRequestHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {

	if s.User == nil || s.Character == nil {
		s.OnClose()
		return nil, nil
	}

	pseudoID := uint16(utils.BytesToInt(data[6:8], true))
	accepted := data[8] == 1

	opponent := database.FindCharacterByPseudoID(s.User.ConnectedServer, pseudoID)
	if opponent == nil {
		return nil, nil
	}

	if !accepted {
		resp := messaging.SystemMessage(messaging.PVP_REQUEST_REJECTED)
		s.Write(resp)
		database.GetSocket(opponent.UserID).Write(resp)

	} else if opponent.DuelID > 0 {
		resp := messaging.SystemMessage(messaging.ALREADY_IN_PVP)
		return resp, nil

	} else { // start pvp
		mC := database.ConvertPointToLocation(s.Character.Coordinate)
		oC := database.ConvertPointToLocation(opponent.Coordinate)
		fC := utils.Location{X: (mC.X + oC.X) / 2, Y: (mC.Y + oC.Y) / 2}

		s.Character.DuelID = opponent.ID
		opponent.DuelID = s.Character.ID

		resp := PVP_STARTED
		resp.Insert(utils.FloatToBytes(fC.X, 4, true), 6)  // flag-X
		resp.Insert(utils.FloatToBytes(fC.Y, 4, true), 10) // flag-Y

		//p := nats.CastPacket{CastNear: true, CharacterID: s.Character.ID, Type: nats.PVP_START, Data: resp}
		//p.Cast()

		s.Character.Socket.Write(resp)
		opponent.Socket.Write(resp)

		go s.Character.StartPvP(3)
		go opponent.StartPvP(3)
	}

	return nil, nil
}

func (h *ChangePetName) Handle(s *database.Socket, data []byte) ([]byte, error) {

	length := int(data[12])

	name := string(data[12 : length+13])

	if length < 4 {
		return messaging.SystemMessage(messaging.INVALID_NAME), nil
	}
	slotIDitem, _, _ := s.Character.FindItemInInventory(nil, 17300186)
	if slotIDitem == -1 {
		return messaging.SystemMessage(messaging.INVALID_NAME), nil
	}
	rr, _ := s.Character.RemoveItem(slotIDitem)
	s.Write(rr)

	slots, _ := s.Character.InventorySlots()
	petSlot := slots[0x0A]
	pet := petSlot.Pet

	pet.Name = name
	petSlot.Update()

	resp := petSlot.GetData(petSlot.SlotID)

	return resp, nil

}

/*
func (h *TransferSoulHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {

	//pseudoID := uint16(utils.BytesToInt(data[6:8], true))
	resp := utils.Packet{0xAA, 0x55, 0x06, 0x00, 0xA5, 0x01, 0x0A, 0x00, 0x55, 0xAA}
	resp.Insert(data[6:8], 8)
	resp.Print()
	return resp, nil
}
*/

func (h *TransferItemTypeHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {

	if s.User == nil || s.Character == nil {
		s.OnClose()
		return nil, nil
	}

	//pseudoID := uint16(utils.BytesToInt(data[6:8], true))
	resp := utils.Packet{0xAA, 0x55, 0x06, 0x00, 0x60, 0x01, 0x0A, 0x00, 0x55, 0xAA}
	resp.Insert(data[6:8], 8)
	fslot, _, err := s.Character.FindItemInInventory(nil, 15710007)
	if err != nil {
		return nil, err
	} else if fslot == -1 {
		return nil, nil
	}
	slot := utils.BytesToInt(data[6:8], true)
	invSlots, err := s.Character.InventorySlots()
	if err != nil {
		log.Println(err)
		return nil, err
	}
	item := invSlots[slot]
	info := database.Items[item.ItemID]
	if info.ItemPair == 0 {
		return nil, nil
	} else {
		freeslot, err := s.Character.FindFreeSlot()
		if err != nil {
			return nil, err
		} else if freeslot == -1 { // no free slot
			return nil, nil
		}
		s.Character.Socket.Write(*s.Character.DecrementItem(int16(fslot), 1))
		item.ItemID = info.ItemPair
		item.Update()

		resp.Concat(item.GetData(item.SlotID))
	}

	return resp, nil
}

func (h *ClothImproveChest) Handle(s *database.Socket, data []byte) ([]byte, error) {
	if s.User == nil || s.Character == nil {
		s.OnClose()
		return nil, nil
	}
	resp := utils.Packet{}
	c := s.Character
	slots, err := c.InventorySlots()
	if err != nil {
		return nil, err
	}

	itemSlot := int16(utils.BytesToInt(data[6:8], true))
	if itemSlot == 0 {
		return nil, nil
	} else if slots[itemSlot].ItemID == 0 {
		return nil, nil
	}

	slot, chest, err := c.FindItemInInventory(nil, 99002838, 18500407)
	if chest == nil || err != nil {
		return nil, nil
	}
	if slot == -1 {
		return nil, nil
	}

	rc, err := makeMaster(c, slots[itemSlot].ItemID, itemSlot)
	if err != nil {
		return nil, nil
	}
	resp.Concat(rc)

	/*
		seed := utils.RandInt(0, 1000)
		if seed > 910 {
			rc, err := makeMaster(c, slots[itemSlot].ItemID, itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(rc)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	*/

	rr, _ := s.Character.RemoveItem(slot)
	resp.Concat(rr)

	return resp, nil
}

func (h *CharmOfIdentity) Handle(s *database.Socket, data []byte) ([]byte, error) {

	if s.User == nil || s.Character == nil {
		s.OnClose()
		return nil, nil
	}

	length := utils.BytesToInt(data[6:7], true)
	name := string(data[7 : 7+length])

	ok, err := database.IsValidUsername(name)
	if err != nil {
		return nil, err
	} else if !ok || length < 4 {
		return messaging.SystemMessage(messaging.INVALID_NAME), nil
	}
	slotIDitem, _, _ := s.Character.FindItemInInventory(nil, 15710005)
	rr, _ := s.Character.RemoveItem(slotIDitem)
	s.Write(rr)
	return s.Character.ChangeName(name)
}

func (h *EnhancementTransfer) Handle(s *database.Socket, data []byte) ([]byte, error) {

	if s.User == nil || s.Character == nil {
		s.OnClose()
		return nil, nil
	}

	resp := utils.Packet{0xAA, 0x55, 0x03, 0x00, 0x57, 0x06, 0x01, 0x55, 0xAA}

	firstWeapSlot := utils.BytesToInt(data[6:8], true)
	secWeapSlot := utils.BytesToInt(data[8:10], true)

	fslot, _, err := s.Character.FindItemInInventory(nil, 13370163)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	if fslot == -1 {
		rip := utils.Packet{0xAA, 0x55, 0x05, 0x00, 0x54, 0x04, 0x11, 0x10, 0x00, 0x55, 0xAA}
		resp.Concat(rip)
		return resp, err
	}

	invSlots, err := s.Character.InventorySlots()
	if err != nil {
		return nil, err
	}

	firstWeap := invSlots[firstWeapSlot]
	secWeap := invSlots[secWeapSlot]
	/*
		firstweaponiteminfo, ok := database.GetItemInfo(firstWeap.ItemID)
		secWeapItemInfo, ok := database.GetItemInfo(secWeap.ItemID)
		if !ok {
			return nil, nil
		}
	*/

	if firstWeap.Plus != 0 /*&& firstweaponiteminfo.Slot == secWeapItemInfo.Slot */ {

		secWeap.Plus = firstWeap.Plus
		secWeap.UpgradeArr = firstWeap.UpgradeArr
		secWeap.SocketCount = firstWeap.SocketCount
		secWeap.SocketArr = firstWeap.SocketArr

		secWeap.Update()

		resp.Concat(secWeap.GetData(secWeap.SlotID))
		s.Write(resp)
		rr, _ := s.Character.RemoveItem(firstWeap.SlotID)
		s.Write(rr)
		rr, _ = s.Character.RemoveItem(fslot)
		s.Write(rr)
		rip := utils.Packet{0xAA, 0x55, 0x05, 0x00, 0x54, 0x04, 0x10, 0x10, 0x00, 0x55, 0xAA}
		resp.Concat(rip)

	} else {
		rip := utils.Packet{0xAA, 0x55, 0x05, 0x00, 0x54, 0x04, 0x11, 0x10, 0x00, 0x55, 0xAA}
		resp.Concat(rip)
	}
	return resp, nil
}

func makeMaster(c *database.Character, itemID int64, itemSlot int16) ([]byte, error) {
	if c == nil {
		return nil, nil
	}
	/*

			seed := utils.RandInt(0, 1000)
		if seed > 910 {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	*/

	resp := utils.Packet{}
	switch itemID {
	case 40000007: // Master Samurai
		seed := utils.RandInt(0, 1000)
		if seed > 910 {
			master := &database.InventorySlot{ItemID: 99100077, Quantity: 1}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 40000008: // Master Samurai
		seed := utils.RandInt(0, 1000)
		if seed > 910 {
			master := &database.InventorySlot{ItemID: 99100079, Quantity: 1}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 40000009: // Master Samurai
		seed := utils.RandInt(0, 1000)
		if seed > 910 {
			master := &database.InventorySlot{ItemID: 99100081, Quantity: 1}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 30012021: // Master Royal Circlet
		seed := utils.RandInt(0, 1000)
		if seed > 910 {
			master := &database.InventorySlot{ItemID: 99100047, Quantity: 1}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 30012121: // Master Royal Coat
		seed := utils.RandInt(0, 1000)
		if seed > 910 {
			master := &database.InventorySlot{ItemID: 99100049, Quantity: 1}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 30012221: // Master Royal Boots
		seed := utils.RandInt(0, 1000)
		if seed > 910 {
			master := &database.InventorySlot{ItemID: 99100051, Quantity: 1}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 30022021: // Master Golden sky Helm
		seed := utils.RandInt(0, 1000)
		if seed > 910 {
			master := &database.InventorySlot{ItemID: 99100053, Quantity: 1}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 30022121: // Master Golden Armor
		seed := utils.RandInt(0, 1000)
		if seed > 910 {
			master := &database.InventorySlot{ItemID: 99100055, Quantity: 1}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 30022221: // Master Golden Boots
		seed := utils.RandInt(0, 1000)
		if seed > 910 {
			master := &database.InventorySlot{ItemID: 99100057, Quantity: 1}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 30002029: // Master Corean Helm
		seed := utils.RandInt(0, 1000)
		if seed > 910 {
			master := &database.InventorySlot{ItemID: 99100041, Quantity: 1}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 30002127: // Master Corean Armor
		seed := utils.RandInt(0, 1000)
		if seed > 910 {
			master := &database.InventorySlot{ItemID: 99100043, Quantity: 1}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 30002227: // Master Corean Boots
		seed := utils.RandInt(0, 1000)
		if seed > 910 {
			master := &database.InventorySlot{ItemID: 99100045, Quantity: 1}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 31102228: // Gold Imperial Helm
		seed := utils.RandInt(0, 1000)
		if seed > 910 {
			master := &database.InventorySlot{ItemID: 99100296, Quantity: 1}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 31102230: // Gold Imperial Armor
		seed := utils.RandInt(0, 1000)
		if seed > 910 {
			master := &database.InventorySlot{ItemID: 99100298, Quantity: 1}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 31102232: // Gold Imperial Boots
		seed := utils.RandInt(0, 1000)
		if seed > 910 {
			master := &database.InventorySlot{ItemID: 99100300, Quantity: 1}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 30042021: // Lee soon Helm
		seed := utils.RandInt(0, 1000)
		if seed > 910 {
			master := &database.InventorySlot{ItemID: 99100065, Quantity: 1}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 30042121: // Lee soon Armor
		seed := utils.RandInt(0, 1000)
		if seed > 910 {
			master := &database.InventorySlot{ItemID: 99100067, Quantity: 1}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 30042221: // Lee soon Boots
		seed := utils.RandInt(0, 1000)
		if seed > 910 {
			master := &database.InventorySlot{ItemID: 99100069, Quantity: 1}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 30032021: // Eastern Sage Helm
		seed := utils.RandInt(0, 1000)
		if seed > 910 {
			master := &database.InventorySlot{ItemID: 99100059, Quantity: 1}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 30032121: // Eastern Sage Armor
		seed := utils.RandInt(0, 1000)
		if seed > 910 {
			master := &database.InventorySlot{ItemID: 99100061, Quantity: 1}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 30032221: // Eastern Sage Boots
		seed := utils.RandInt(0, 1000)
		if seed > 910 {
			master := &database.InventorySlot{ItemID: 99100063, Quantity: 1}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 30052021: // Ancient Kings Helm
		seed := utils.RandInt(0, 1000)
		if seed > 910 {
			master := &database.InventorySlot{ItemID: 99100071, Quantity: 1}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 30052121: // Ancient Kings Armor
		seed := utils.RandInt(0, 1000)
		if seed > 910 {
			master := &database.InventorySlot{ItemID: 99100073, Quantity: 1}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 30052221: // Ancient Kings Boots
		seed := utils.RandInt(0, 1000)
		if seed > 910 {
			master := &database.InventorySlot{ItemID: 99100075, Quantity: 1}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 30098149: // Siu Mask
		seed := utils.RandInt(0, 1000)
		if seed > 910 {
			master := &database.InventorySlot{ItemID: 30085046, Quantity: 1}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 18500050: // Beach Swim Helm
		seed := utils.RandInt(0, 1000)
		if seed > 910 {
			master := &database.InventorySlot{ItemID: 18500054, Quantity: 1}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 18500051: // Beach Swim Armor
		seed := utils.RandInt(0, 1000)
		if seed > 910 {
			master := &database.InventorySlot{ItemID: 18500055, Quantity: 1}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 18500052: // Beach Swim Boots
		seed := utils.RandInt(0, 1000)
		if seed > 910 {
			master := &database.InventorySlot{ItemID: 18500056, Quantity: 1}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 18500053: // Beach Swim Mask
		seed := utils.RandInt(0, 1000)
		if seed > 910 {
			master := &database.InventorySlot{ItemID: 18500057, Quantity: 1}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 13370169: // Carnival Helm
		seed := utils.RandInt(0, 1000)
		if seed > 910 {
			master := &database.InventorySlot{ItemID: 13370216, Quantity: 1}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 13370170: // Carnival Armor
		seed := utils.RandInt(0, 1000)
		if seed > 910 {
			master := &database.InventorySlot{ItemID: 13370217, Quantity: 1}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 13370171: // Carnival boots
		seed := utils.RandInt(0, 1000)
		if seed > 910 {
			master := &database.InventorySlot{ItemID: 13370218, Quantity: 1}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 13370166: // Dark Sky Helm
		seed := utils.RandInt(0, 1000)
		if seed > 910 {
			master := &database.InventorySlot{ItemID: 13370213, Quantity: 1}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 13370167: // Dark Sky Armor
		seed := utils.RandInt(0, 1000)
		if seed > 910 {
			master := &database.InventorySlot{ItemID: 13370214, Quantity: 1}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 13370168: // Dark Sky Boots
		seed := utils.RandInt(0, 1000)
		if seed > 910 {
			master := &database.InventorySlot{ItemID: 13370215, Quantity: 1}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 13370172: // Pirate Hat
		seed := utils.RandInt(0, 1000)
		if seed > 910 {
			master := &database.InventorySlot{ItemID: 13370219, Quantity: 1}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 13370173: // Pirate Armor
		seed := utils.RandInt(0, 1000)
		if seed > 910 {
			master := &database.InventorySlot{ItemID: 13370220, Quantity: 1}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 13370174: // Pirate Boots
		seed := utils.RandInt(0, 1000)
		if seed > 910 {
			master := &database.InventorySlot{ItemID: 13370221, Quantity: 1}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 15710730: // Frostbite Guard Helm(M)_Low
		seed := utils.RandInt(0, 1000)
		if seed > 750 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500331, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 15710731: // Frostbite Guard Helm(F)_Low
		seed := utils.RandInt(0, 1000)
		if seed > 750 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500333, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 15710732: // Frostbite Guard Boots(M)_Low
		seed := utils.RandInt(0, 1000)
		if seed > 750 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500335, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 15710733: // Frostbite Guard Boots(F)_Low
		seed := utils.RandInt(0, 1000)
		if seed > 750 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500337, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 15710734: // Frostbite Guard Armor (M)_Low
		seed := utils.RandInt(0, 1000)
		if seed > 750 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500327, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 15710735: // Frostbite Guard Armor (F)_Low
		seed := utils.RandInt(0, 1000)
		if seed > 750 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500329, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 15710736: // Frostbite Guard Mask(M)_Low
		seed := utils.RandInt(0, 1000)
		if seed > 750 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500339, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 15710737: // Frostbite Guard Mask(F)_Low
		seed := utils.RandInt(0, 1000)
		if seed > 750 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500341, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 18500327: // Frostbite Guard Armor (M)_Medium
		seed := utils.RandInt(0, 1000)
		if seed > 800 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500328, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 18500329: // Frostbite Guard Armor(F)_Medium
		seed := utils.RandInt(0, 1000)
		if seed > 800 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500330, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 18500331: // Frostbite Guard Helm(M)_Medium
		seed := utils.RandInt(0, 1000)
		if seed > 800 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500332, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 18500333: // Frostbite Guard Helm(F)_Medium
		seed := utils.RandInt(0, 1000)
		if seed > 800 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500334, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 18500335: // Frostbite Guard Boots(M)_Medium
		seed := utils.RandInt(0, 1000)
		if seed > 800 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500336, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 18500337: // Frostbite Guard Boots(F)_Medium
		seed := utils.RandInt(0, 1000)
		if seed > 800 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500338, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 18500339: // Frostbite Guard Mask(M)_Medium
		seed := utils.RandInt(0, 1000)
		if seed > 800 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500340, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 18500341: // Frostbite Guard Mask(F)_Medium
		seed := utils.RandInt(0, 1000)
		if seed > 800 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500342, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 18500328: // --
		seed := utils.RandInt(0, 1000)
		if seed > 820 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500388, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 18500332: // --
		seed := utils.RandInt(0, 1000)
		if seed > 820 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500392, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 18500336: // --
		seed := utils.RandInt(0, 1000)
		if seed > 820 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500396, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 18500340: // --
		seed := utils.RandInt(0, 1000)
		if seed > 820 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500400, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 18500330: // --
		seed := utils.RandInt(0, 1000)
		if seed > 820 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500389, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 18500334: // --
		seed := utils.RandInt(0, 1000)
		if seed > 820 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500393, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 18500338: // --
		seed := utils.RandInt(0, 1000)
		if seed > 820 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500397, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 18500342: // --
		seed := utils.RandInt(0, 1000)
		if seed > 820 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500401, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 18500388: // --
		seed := utils.RandInt(0, 1000)
		if seed > 820 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500390, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 18500392: // --
		seed := utils.RandInt(0, 1000)
		if seed > 820 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500394, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 18500396: // --
		seed := utils.RandInt(0, 1000)
		if seed > 820 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500398, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 18500400: // --
		seed := utils.RandInt(0, 1000)
		if seed > 820 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500402, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 18500389: // --
		seed := utils.RandInt(0, 1000)
		if seed > 820 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500391, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 18500393: // --
		seed := utils.RandInt(0, 1000)
		if seed > 820 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500395, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 18500397: // --
		seed := utils.RandInt(0, 1000)
		if seed > 820 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500399, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 18500401: // --
		seed := utils.RandInt(0, 1000)
		if seed > 820 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500403, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 18500148: // --
		seed := utils.RandInt(0, 1000)
		if seed > 820 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500347, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 18500149: // --
		seed := utils.RandInt(0, 1000)
		if seed > 820 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500348, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 18500150: // --
		seed := utils.RandInt(0, 1000)
		if seed > 820 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500349, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 18500151: // --
		seed := utils.RandInt(0, 1000)
		if seed > 820 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500350, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 18500152: // --
		seed := utils.RandInt(0, 1000)
		if seed > 820 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500351, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 18500153: // --
		seed := utils.RandInt(0, 1000)
		if seed > 820 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500352, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 18500154: // --
		seed := utils.RandInt(0, 1000)
		if seed > 820 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500353, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 18500155: // --
		seed := utils.RandInt(0, 1000)
		if seed > 820 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500354, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 18500156: // --
		seed := utils.RandInt(0, 1000)
		if seed > 820 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500355, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 18500157: // --
		seed := utils.RandInt(0, 1000)
		if seed > 820 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500356, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 18500158: // --
		seed := utils.RandInt(0, 1000)
		if seed > 820 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500357, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 18500159: // --
		seed := utils.RandInt(0, 1000)
		if seed > 820 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500358, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 18500347: // --
		seed := utils.RandInt(0, 1000)
		if seed > 820 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500359, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 18500348: // --
		seed := utils.RandInt(0, 1000)
		if seed > 820 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500360, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 18500349: // --
		seed := utils.RandInt(0, 1000)
		if seed > 820 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500361, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 18500350: // --
		seed := utils.RandInt(0, 1000)
		if seed > 820 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500362, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 18500351: // --
		seed := utils.RandInt(0, 1000)
		if seed > 820 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500363, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 18500352: // --
		seed := utils.RandInt(0, 1000)
		if seed > 820 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500364, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 18500353: // --
		seed := utils.RandInt(0, 1000)
		if seed > 820 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500365, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 18500354: // --
		seed := utils.RandInt(0, 1000)
		if seed > 820 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500366, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 18500355: // --
		seed := utils.RandInt(0, 1000)
		if seed > 820 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500367, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 18500356: // --
		seed := utils.RandInt(0, 1000)
		if seed > 820 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500368, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 18500357: // --
		seed := utils.RandInt(0, 1000)
		if seed > 820 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500369, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 18500358: // --
		seed := utils.RandInt(0, 1000)
		if seed > 820 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500370, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 18500359: // --
		seed := utils.RandInt(0, 1000)
		if seed > 820 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500371, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 18500360: // --
		seed := utils.RandInt(0, 1000)
		if seed > 820 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500372, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 18500361: // --
		seed := utils.RandInt(0, 1000)
		if seed > 820 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500373, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 18500362: // --
		seed := utils.RandInt(0, 1000)
		if seed > 820 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500374, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 18500363: // --
		seed := utils.RandInt(0, 1000)
		if seed > 820 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500375, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 18500364: // --
		seed := utils.RandInt(0, 1000)
		if seed > 820 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500376, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 18500365: // --
		seed := utils.RandInt(0, 1000)
		if seed > 820 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500377, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 18500366: // --
		seed := utils.RandInt(0, 1000)
		if seed > 820 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500378, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 18500367: // --
		seed := utils.RandInt(0, 1000)
		if seed > 820 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500379, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 18500368: // --
		seed := utils.RandInt(0, 1000)
		if seed > 820 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500380, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 18500369: // --
		seed := utils.RandInt(0, 1000)
		if seed > 820 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500381, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 18500370: // --
		seed := utils.RandInt(0, 1000)
		if seed > 820 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500382, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 18500323: // --
		seed := utils.RandInt(0, 1000)
		if seed > 820 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500383, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 18500383: // --
		seed := utils.RandInt(0, 1000)
		if seed > 820 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500384, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 18500384: // --
		seed := utils.RandInt(0, 1000)
		if seed > 820 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500385, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 18500177: // Auralı armor
		seed := utils.RandInt(0, 1000)
		if seed > 0 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500462, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 18500180: // Auralı armor
		seed := utils.RandInt(0, 1000)
		if seed > 0 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500463, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 18500183: // Auralı armor
		seed := utils.RandInt(0, 1000)
		if seed > 0 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500470, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 18500186: // Auralı armor
		seed := utils.RandInt(0, 1000)
		if seed > 0 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500464, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 18500189: // Auralı armor
		seed := utils.RandInt(0, 1000)
		if seed > 0 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500465, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 18500192: // Auralı armor
		seed := utils.RandInt(0, 1000)
		if seed > 0 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500466, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 18500195: // Auralı armor
		seed := utils.RandInt(0, 1000)
		if seed > 0 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500467, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 18500198: // Auralı armor
		seed := utils.RandInt(0, 1000)
		if seed > 0 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500468, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 18500201: // Auralı armor
		seed := utils.RandInt(0, 1000)
		if seed > 0 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500469, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 18500484: // Auralı armor
		seed := utils.RandInt(0, 1000)
		if seed > 0 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500471, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 18500474: // Auralı armor
		seed := utils.RandInt(0, 1000)
		if seed > 0 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500479, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 18500490: // Auralı armor
		seed := utils.RandInt(0, 1000)
		if seed > 0 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500472, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 99100041: // elite
		seed := utils.RandInt(0, 1000)
		if seed > 900 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 99100095, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 99100095: // elite
		seed := utils.RandInt(0, 1000)
		if seed > 900 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500182, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 99100043: // elite
		seed := utils.RandInt(0, 1000)
		if seed > 900 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 99100097, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 99100097: // elite
		seed := utils.RandInt(0, 1000)
		if seed > 900 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500183, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 99100045: // elite
		seed := utils.RandInt(0, 1000)
		if seed > 900 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 99100099, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 99100099: // elite
		seed := utils.RandInt(0, 1000)
		if seed > 900 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500184, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 99100047: // elite
		seed := utils.RandInt(0, 1000)
		if seed > 900 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 99100101, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 99100101: // elite
		seed := utils.RandInt(0, 1000)
		if seed > 900 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500176, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 99100049: // elite
		seed := utils.RandInt(0, 1000)
		if seed > 900 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 99100103, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 99100103: // elite
		seed := utils.RandInt(0, 1000)
		if seed > 900 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500177, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 99100051: // elite
		seed := utils.RandInt(0, 1000)
		if seed > 900 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 99100105, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 99100105: // elite
		seed := utils.RandInt(0, 1000)
		if seed > 900 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500178, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 99100053: // elite
		seed := utils.RandInt(0, 1000)
		if seed > 900 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 99100107, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 99100107: // elite
		seed := utils.RandInt(0, 1000)
		if seed > 900 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500179, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 99100055: // elite
		seed := utils.RandInt(0, 1000)
		if seed > 900 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 99100109, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 99100109: // elite
		seed := utils.RandInt(0, 1000)
		if seed > 900 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500180, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 99100057: // elite
		seed := utils.RandInt(0, 1000)
		if seed > 900 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 99100111, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 99100111: // elite
		seed := utils.RandInt(0, 1000)
		if seed > 900 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500181, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 99100077: // elite
		seed := utils.RandInt(0, 1000)
		if seed > 900 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 99100131, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 99100131: // elite
		seed := utils.RandInt(0, 1000)
		if seed > 900 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500185, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 99100079: // elite
		seed := utils.RandInt(0, 1000)
		if seed > 900 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 99100133, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 99100133: // elite
		seed := utils.RandInt(0, 1000)
		if seed > 900 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500186, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 99100081: // elite
		seed := utils.RandInt(0, 1000)
		if seed > 900 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 99100135, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 99100135: // elite
		seed := utils.RandInt(0, 1000)
		if seed > 900 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500187, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 99100065: // elite
		seed := utils.RandInt(0, 1000)
		if seed > 900 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 99100119, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 99100119: // elite
		seed := utils.RandInt(0, 1000)
		if seed > 900 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500188, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 99100067: // elite
		seed := utils.RandInt(0, 1000)
		if seed > 900 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 99100121, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 99100121: // elite
		seed := utils.RandInt(0, 1000)
		if seed > 900 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500189, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 99100069: // elite
		seed := utils.RandInt(0, 1000)
		if seed > 900 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 99100123, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 99100123: // elite
		seed := utils.RandInt(0, 1000)
		if seed > 900 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500190, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 99100059: // elite
		seed := utils.RandInt(0, 1000)
		if seed > 900 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 99100113, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 99100113: // elite
		seed := utils.RandInt(0, 1000)
		if seed > 900 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500191, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 99100061: // elite
		seed := utils.RandInt(0, 1000)
		if seed > 900 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 99100115, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 99100115: // elite
		seed := utils.RandInt(0, 1000)
		if seed > 900 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500192, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 99100063: // elite
		seed := utils.RandInt(0, 1000)
		if seed > 900 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 99100117, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 99100117: // elite
		seed := utils.RandInt(0, 1000)
		if seed > 900 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500193, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 99100071: // elite
		seed := utils.RandInt(0, 1000)
		if seed > 900 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 99100125, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 99100125: // elite
		seed := utils.RandInt(0, 1000)
		if seed > 900 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500194, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 99100073: // elite
		seed := utils.RandInt(0, 1000)
		if seed > 900 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 99100127, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 99100127: // elite
		seed := utils.RandInt(0, 1000)
		if seed > 900 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500195, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 99100075: // elite
		seed := utils.RandInt(0, 1000)
		if seed > 900 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 99100129, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 99100129: // elite
		seed := utils.RandInt(0, 1000)
		if seed > 900 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500196, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 99100296: // elite
		seed := utils.RandInt(0, 1000)
		if seed > 900 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 99100302, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 99100302: // elite
		seed := utils.RandInt(0, 1000)
		if seed > 900 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500197, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 99100298: // elite
		seed := utils.RandInt(0, 1000)
		if seed > 900 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 99100304, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 99100304: // elite
		seed := utils.RandInt(0, 1000)
		if seed > 900 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500198, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 99100300: // elite
		seed := utils.RandInt(0, 1000)
		if seed > 900 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 99100306, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 99100306: // elite
		seed := utils.RandInt(0, 1000)
		if seed > 900 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500199, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 18500054: // elite
		seed := utils.RandInt(0, 1000)
		if seed > 900 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500058, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 18500058: // elite
		seed := utils.RandInt(0, 1000)
		if seed > 900 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500200, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 18500055: // elite
		seed := utils.RandInt(0, 1000)
		if seed > 900 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500059, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 18500059: // elite
		seed := utils.RandInt(0, 1000)
		if seed > 900 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500201, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 18500056: // elite
		seed := utils.RandInt(0, 1000)
		if seed > 900 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500060, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 18500060: // elite
		seed := utils.RandInt(0, 1000)
		if seed > 900 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500202, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 18500320: // elite
		seed := utils.RandInt(0, 1000)
		if seed > 900 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500480, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 18500480: // elite
		seed := utils.RandInt(0, 1000)
		if seed > 900 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500483, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 18500321: // elite
		seed := utils.RandInt(0, 1000)
		if seed > 900 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500481, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 18500481: // elite
		seed := utils.RandInt(0, 1000)
		if seed > 900 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500484, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 18500322: // elite
		seed := utils.RandInt(0, 1000)
		if seed > 900 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500482, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 18500482: // elite
		seed := utils.RandInt(0, 1000)
		if seed > 900 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500485, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 18500324: // elite
		seed := utils.RandInt(0, 1000)
		if seed > 900 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500486, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 18500486: // elite
		seed := utils.RandInt(0, 1000)
		if seed > 900 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500489, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 18500325: // elite
		seed := utils.RandInt(0, 1000)
		if seed > 900 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500487, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 18500487: // elite
		seed := utils.RandInt(0, 1000)
		if seed > 900 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500490, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 18500326: // elite
		seed := utils.RandInt(0, 1000)
		if seed > 900 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500488, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 18500488: // elite
		seed := utils.RandInt(0, 1000)
		if seed > 900 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500491, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 18500317: // elite
		seed := utils.RandInt(0, 1000)
		if seed > 900 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500476, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 18500476: // elite
		seed := utils.RandInt(0, 1000)
		if seed > 900 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500473, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 18500318: // elite
		seed := utils.RandInt(0, 1000)
		if seed > 900 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500477, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 18500477: // elite
		seed := utils.RandInt(0, 1000)
		if seed > 900 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500474, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 18500319: // elite
		seed := utils.RandInt(0, 1000)
		if seed > 900 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500478, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 18500478: // elite
		seed := utils.RandInt(0, 1000)
		if seed > 900 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500475, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 30085046: // elite
		seed := utils.RandInt(0, 1000)
		if seed > 900 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500204, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 18500204: // Combat to Elite
		seed := utils.RandInt(0, 1000)
		if seed > 980 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500583, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}

	case 18500462: //
		seed := utils.RandInt(0, 1000)
		if seed > 920 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500599, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 18500463: //
		seed := utils.RandInt(0, 1000)
		if seed > 920 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500599, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 18500464: //
		seed := utils.RandInt(0, 1000)
		if seed > 920 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500599, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 18500465: //
		seed := utils.RandInt(0, 1000)
		if seed > 920 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500599, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 18500466: //
		seed := utils.RandInt(0, 1000)
		if seed > 920 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500599, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 18500467: //
		seed := utils.RandInt(0, 1000)
		if seed > 920 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500599, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 18500468: //
		seed := utils.RandInt(0, 1000)
		if seed > 920 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500599, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 18500469: //
		seed := utils.RandInt(0, 1000)
		if seed > 920 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500599, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 18500470: //
		seed := utils.RandInt(0, 1000)
		if seed > 920 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500599, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 18500471: //
		seed := utils.RandInt(0, 1000)
		if seed > 920 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500599, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 18500472: //
		seed := utils.RandInt(0, 1000)
		if seed > 920 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500599, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 18500479: //
		seed := utils.RandInt(0, 1000)
		if seed > 920 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500599, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 18500176: //
		seed := utils.RandInt(0, 1000)
		if seed > 920 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500598, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 18500179: //
		seed := utils.RandInt(0, 1000)
		if seed > 920 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500598, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 18500182: //
		seed := utils.RandInt(0, 1000)
		if seed > 920 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500598, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 18500185: //
		seed := utils.RandInt(0, 1000)
		if seed > 920 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500598, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 18500188: //
		seed := utils.RandInt(0, 1000)
		if seed > 920 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500598, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 18500191: //
		seed := utils.RandInt(0, 1000)
		if seed > 920 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500598, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 18500194: //
		seed := utils.RandInt(0, 1000)
		if seed > 920 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500598, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 18500197: //
		seed := utils.RandInt(0, 1000)
		if seed > 920 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500598, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 18500200: //
		seed := utils.RandInt(0, 1000)
		if seed > 920 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500598, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 18500473: //
		seed := utils.RandInt(0, 1000)
		if seed > 920 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500598, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 18500483: //
		seed := utils.RandInt(0, 1000)
		if seed > 920 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500598, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 18500489: //
		seed := utils.RandInt(0, 1000)
		if seed > 920 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500598, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 18500598: //
		seed := utils.RandInt(0, 1000)
		if seed > 920 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 15813006, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 18500599: //
		seed := utils.RandInt(0, 1000)
		if seed > 920 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 15813010, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 18500600: //
		seed := utils.RandInt(0, 1000)
		if seed > 920 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 15813008, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 18500178: //
		seed := utils.RandInt(0, 1000)
		if seed > 920 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500600, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 18500181: //
		seed := utils.RandInt(0, 1000)
		if seed > 920 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500600, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 18500184: //
		seed := utils.RandInt(0, 1000)
		if seed > 920 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500600, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 18500187: //
		seed := utils.RandInt(0, 1000)
		if seed > 920 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500600, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 18500190: //
		seed := utils.RandInt(0, 1000)
		if seed > 920 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500600, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 18500193: //
		seed := utils.RandInt(0, 1000)
		if seed > 920 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500600, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 18500196: //
		seed := utils.RandInt(0, 1000)
		if seed > 920 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500600, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 18500199: //
		seed := utils.RandInt(0, 1000)
		if seed > 920 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500600, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 18500202: //
		seed := utils.RandInt(0, 1000)
		if seed > 920 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500600, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 18500475: //
		seed := utils.RandInt(0, 1000)
		if seed > 920 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500600, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 18500485: //
		seed := utils.RandInt(0, 1000)
		if seed > 920 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500600, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 18500491: //
		seed := utils.RandInt(0, 1000)
		if seed > 920 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500600, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 18500583: //
		seed := utils.RandInt(0, 1000)
		if seed > 950 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 18500604, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	case 18500604: //
		seed := utils.RandInt(0, 1000)
		if seed > 975 {
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			master := &database.InventorySlot{ItemID: 15710341, Quantity: 1, Plus: slots[itemSlot].Plus, UpgradeArr: slots[itemSlot].UpgradeArr, SocketCount: slots[itemSlot].SocketCount, SocketArr: slots[itemSlot].SocketArr}
			rc, _, err := c.AddItem(master, -1, false)
			if err != nil {
				return nil, nil
			}
			resp.Concat(*rc)
			r, err := c.RemoveItem(itemSlot)
			if err != nil {
				return nil, nil
			}
			resp.Concat(r)
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
		} else {
			resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
		}
	}

	return resp, nil
}
