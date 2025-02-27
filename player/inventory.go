package player

import (
	"fmt"
	"log"
	"math"
	"sort"
	"strconv"
	"time"

	"hero-server/database"
	"hero-server/logging"
	"hero-server/messaging"
	"hero-server/nats"
	"hero-server/utils"

	"github.com/thoas/go-funk"
)

type (
	GetGoldHandler struct {
		gold uint64
	}

	GetInventoryHandler             struct{}
	ReplaceItemHandler              struct{}
	SwitchWeaponHandler             struct{}
	SwapItemsHandler                struct{}
	RemoveItemHandler               struct{}
	DestroyItemHandler              struct{}
	CombineItemsHandler             struct{}
	ArrangeInventoryHandler         struct{}
	ArrangeBankHandler              struct{}
	DepositHandler                  struct{}
	WithdrawHandler                 struct{}
	OpenHTMenuHandler               struct{}
	CloseHTMenuHandler              struct{}
	BuyHTItemHandler                struct{}
	ReplaceHTItemHandler            struct{}
	DressUpHandler                  struct{}
	SplitItemHandler                struct{}
	HolyWaterUpgradeHandler         struct{}
	UseConsumableHandler            struct{}
	OpenBoxHandler                  struct{}
	OpenBoxHandler2                 struct{}
	ActivateTimeLimitedItemHandler  struct{}
	ActivateTimeLimitedItemHandler2 struct{}
	ToggleMountPetHandler           struct{}
	TogglePetHandler                struct{}
	PetCombatModeHandler            struct{}
	EnchantBookHandler              struct{}
	FireworkHandler                 struct{}
)

var (
	GET_GOLD       = utils.Packet{0xAA, 0x55, 0x0A, 0x00, 0x57, 0x0B, 0x55, 0xAA}
	ITEMS_COMBINED = utils.Packet{0xAA, 0x55, 0x10, 0x00, 0x59, 0x06, 0x0A, 0x00, 0x00, 0x00, 0x55, 0xAA}
	ARRANGE_ITEM   = utils.Packet{0xAA, 0x55, 0x32, 0x00, 0x78, 0x02, 0x00, 0xA2, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x55, 0xAA}

	ARRANGE_BANK_ITEM = utils.Packet{0xAA, 0x55, 0x2F, 0x00, 0x80, 0x02, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x55, 0xAA}

	CLOSE_HT_MENU = utils.Packet{0xAA, 0x55, 0x04, 0x00, 0x64, 0x02, 0x0A, 0x00, 0x55, 0xAA}
	OPEN_HT_MENU  = utils.Packet{0xAA, 0x55, 0x04, 0x00, 0x64, 0x01, 0x0A, 0x00, 0x55, 0xAA}
	GET_CASH      = utils.Packet{0xAA, 0x55, 0x0C, 0x00, 0x64, 0x03, 0x0A, 0x00, 0x55, 0xAA}
	BUY_HT_ITEM   = utils.Packet{0xAA, 0x55, 0x38, 0x00, 0x64, 0x04, 0x0A, 0x00, 0x07, 0xA2, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x55, 0xAA}
	REPLACE_HT_ITEM = utils.Packet{0xAA, 0x55, 0x0A, 0x00, 0x59, 0x40, 0x0A, 0x00, 0x55, 0xAA}
	HT_VISIBILITY   = utils.Packet{0xAA, 0x55, 0x06, 0x00, 0x59, 0x11, 0x0A, 0x00, 0x01, 0x00, 0x55, 0xAA}

	PET_COMBAT      = utils.Packet{0xAA, 0x55, 0x06, 0x00, 0x51, 0x05, 0x0a, 0x00, 0x00, 0x55, 0xAA}
	htShopQuantites = map[int64]uint{17100004: 40, 17100005: 40, 15900001: 50}
)

func (ggh *GetGoldHandler) Handle(s *database.Socket) ([]byte, error) {

	if s.User == nil || s.Character == nil {
		s.Conn.Close()
		return nil, nil
	}

	resp := GET_GOLD
	resp.Insert(utils.IntToBytes(uint64(s.Character.Gold), 8, true), 6)
	return resp, nil
}

func (gih *GetInventoryHandler) Handle(s *database.Socket) ([]byte, error) {

	if s.User == nil || s.Character == nil {
		s.Conn.Close()
		return nil, nil
	}

	inventory, err := s.Character.InventorySlots()
	if err != nil {
		return nil, err
	}

	resp := utils.Packet{}
	for i := 0; i < len(inventory); i++ {

		slot := inventory[i]
		resp.Concat(slot.GetData(int16(i)))
	}

	if inventory[0x0A].ItemID > 0 { // pet
		resp.Concat(database.SHOW_PET_BUTTON)
	}

	if s.Character.DoesInventoryExpanded() {
		resp.Concat(database.BAG_EXPANDED)
	}

	return resp, nil
}

func (h *ReplaceItemHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {

	if s.User == nil || s.Character == nil {
		s.Conn.Close()
		return nil, nil
	}

	itemID := int(utils.BytesToInt(data[6:10], true))
	where := int16(utils.BytesToInt(data[10:12], true))
	to := int16(utils.BytesToInt(data[12:14], true))

	char, _ := database.FindCharacterByID(s.Character.ID)
	slots := char.GetAllEquipedSlots()
	useItem, _ := utils.Contains(slots, int(to))
	inventory, err := char.InventorySlots()
	if err != nil {
		return nil, err
	}
	itemInfo := inventory[where]
	info := database.Items[itemInfo.ItemID]
	isWeapon := false

	if useItem {
		if info.Slot == 3 || info.Slot == 4 {
			if int(to) == 4 || int(to) == 3 {
				isWeapon = true
			}
		}
		if int(to) != info.Slot && isWeapon == false {
			log.Printf("Warning CHEATER: %s Slot: %d", s.Character.Name, to)
			return nil, nil
		}
	}

	resp, err := s.Character.ReplaceItem(itemID, where, to)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (h *SwitchWeaponHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {

	if s.User == nil || s.Character == nil {
		s.Conn.Close()
		return nil, nil
	}

	slotID := data[6]
	s.Character.WeaponSlot = int(slotID)

	itemsData, err := s.Character.ShowItems()
	if err != nil {
		return nil, err
	}

	p := nats.CastPacket{CastNear: true, CharacterID: s.Character.ID, Type: nats.SHOW_ITEMS, Data: itemsData}
	if err = p.Cast(); err != nil {
		return nil, err
	}

	gsh := &GetStatsHandler{}
	statData, err := gsh.Handle(s)
	if err != nil {
		return nil, err
	}

	resp := utils.Packet{}
	resp.Concat(itemsData)
	resp.Concat(statData)

	return resp, nil
}

func (h *SwapItemsHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {

	if s.User == nil || s.Character == nil {
		s.Conn.Close()
		return nil, nil
	}

	index := 11
	where := int16(utils.BytesToInt(data[index:index+2], true))

	index += 2
	to := int16(utils.BytesToInt(data[index:index+2], true))

	resp, err := s.Character.SwapItems(where, to)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (h *RemoveItemHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {

	if s.User == nil || s.Character == nil {
		s.Conn.Close()
		return nil, nil
	}

	index := 11
	slotID := int16(utils.BytesToInt(data[index:index+2], true)) // slot

	resp, err := s.Character.RemoveItem(slotID)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (h *DestroyItemHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {

	if s.User == nil || s.Character == nil {
		s.Conn.Close()
		return nil, nil
	}

	index := 10
	slotID := int16(utils.BytesToInt(data[index:index+2], true)) // slot

	resp, err := s.Character.RemoveItem(slotID)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (h *CombineItemsHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {

	if s.User == nil || s.Character == nil {
		s.Conn.Close()
		return nil, nil
	}

	c := s.Character
	if c == nil {
		return nil, nil
	}

	where := int16(utils.BytesToInt(data[6:8], true))
	to := int16(utils.BytesToInt(data[8:10], true))
	itemID, qty, err := c.CombineItems(where, to)
	if err != nil {
		resp, err := s.Character.SwapItems(where, to)
		if err != nil {
			fmt.Println(err)
			return nil, err
		}

		return resp, nil
	}

	resp := ITEMS_COMBINED
	resp.Insert(utils.IntToBytes(uint64(itemID), 4, true), 8) // item id
	resp.Insert(utils.IntToBytes(uint64(where), 2, true), 12) // where slot
	resp.Insert(utils.IntToBytes(uint64(to), 2, true), 16)    // to slot
	resp.Insert(utils.IntToBytes(uint64(qty), 2, true), 18)   // item quantity

	return resp, nil
}

func (h *ArrangeInventoryHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {

	if s.User == nil || s.Character == nil {
		s.Conn.Close()
		return nil, nil
	}

	if s.Character.TradeID != "" || database.FindSale(s.Character.PseudoID) != nil {
		return nil, nil
	}

	if s.Character.InventoryArrange {
		s.Character.Socket.Write(messaging.InfoMessage("You need to wait 3 more seconds to arrange."))
		return nil, nil
	}
	s.Character.InventoryArrange = true

	slots, err := s.Character.InventorySlots()
	if err != nil {
		return nil, err
	}

	newSlots := make([]database.InventorySlot, 56)
	for i := 0; i < 56; i++ {
		slotID := i + 0x0B
		newSlots[i] = *slots[slotID]
		if newSlots[i].ItemID == 0 {
			newSlots[i].ItemID = math.MaxInt64
		}
		newSlots[i].RFU = int64(slotID)
	}

	sort.Slice(newSlots, func(i, j int) bool {
		return newSlots[i].ItemID < newSlots[j].ItemID
	})

	resp := utils.Packet{}
	for i := 0; i < 56; i++ { // first page
		slot := &newSlots[i]
		r, r2 := ARRANGE_ITEM, utils.Packet{}

		if slot.ItemID == math.MaxInt64 {
			slot.ItemID = 0
		}

		slot.SlotID = int16(i + 0x0B)
		r.Insert(utils.IntToBytes(uint64(slot.ItemID), 4, true), 6) // item id

		info := database.Items[slot.ItemID]
		if info != nil && slot.Activated { // using state
			if info.TimerType == 1 {
				r[10] = 3
			} else if info.TimerType == 3 {
				r[10] = 5
				r2 = database.GREEN_ITEM_COUNT
				r2.Insert(utils.IntToBytes(uint64(slot.SlotID), 2, true), 8)    // slot id
				r2.Insert(utils.IntToBytes(uint64(slot.Quantity), 4, true), 10) // item quantity
			}
		} else {
			r[10] = 0
		}

		if slot.ItemID == 0 {
			r[11] = 0
		} else if slot.Plus > 0 {
			r[11] = 0xA2
		}

		r.Insert(utils.IntToBytes(uint64(slot.Quantity), 2, true), 12) // item quantity
		r.Insert(utils.IntToBytes(uint64(slot.SlotID), 2, true), 14)   // slot id
		r.Insert(slot.GetUpgrades(), 16)                               // slot upgrades
		r[31] = byte(slot.SocketCount)                                 // socket count
		r.Insert(slot.GetSockets(), 32)                                // slot sockets

		r.Overwrite(utils.IntToBytes(uint64(slot.Appearance), 4, true), 46) //16 volt

		if i == 55 {
			r[50] = 1
		}

		r.Insert(utils.IntToBytes(uint64(slot.RFU.(int64)), 2, true), 52) // pre slot id

		if info != nil && info.GetType() == database.PET_TYPE {
			r2.Concat(slot.GetData(int16(slot.SlotID)))
		}

		slot.Update()
		resp.Concat(r)
		resp.Concat(r2)
		resp.Concat(slot.GetData(slot.SlotID))
	}

	for i := 0; i < 56; i++ {
		slotID := i + 0x0B
		newSlots[i].RFU = 0
		*slots[slotID] = newSlots[i]
	}

	newSlots = make([]database.InventorySlot, 56)
	for i := 0; i < 56; i++ {
		slotID := i + 0x0155
		newSlots[i] = *slots[slotID]
		if newSlots[i].ItemID == 0 {
			newSlots[i].ItemID = math.MaxInt64
		}
		newSlots[i].RFU = int64(slotID)
	}

	sort.Slice(newSlots, func(i, j int) bool {
		return newSlots[i].ItemID < newSlots[j].ItemID
	})

	for i := 0; i < 56; i++ { // second page
		slot := &newSlots[i]
		r, r2 := ARRANGE_ITEM, utils.Packet{}

		if slot.ItemID == math.MaxInt64 {
			slot.ItemID = 0
		}

		slot.SlotID = int16(i + 0x0155)
		r.Insert(utils.IntToBytes(uint64(slot.ItemID), 4, true), 6) // item id

		info := database.Items[slot.ItemID]
		if info != nil && slot.Activated { // using state
			if info.TimerType == 1 {
				r[10] = 3
			} else if info.TimerType == 3 {
				r[10] = 5
				r2 = database.GREEN_ITEM_COUNT
				r2.Insert(utils.IntToBytes(uint64(slot.SlotID), 2, true), 8)    // slot id
				r2.Insert(utils.IntToBytes(uint64(slot.Quantity), 4, true), 10) // item quantity
			}
		} else {
			r[10] = 0
		}

		if slot.ItemID == 0 {
			r[11] = 0
		} else if slot.Plus > 0 {
			r[11] = 0xA2
		}

		r.Insert(utils.IntToBytes(uint64(slot.Quantity), 2, true), 12) // item quantity
		r.Insert(utils.IntToBytes(uint64(slot.SlotID), 2, true), 14)   // slot id
		r.Insert(slot.GetUpgrades(), 16)                               // slot upgrades
		r[31] = byte(slot.SocketCount)                                 // socket count
		r.Insert(slot.GetSockets(), 32)                                // slot sockets

		if i == 55 {
			r[50] = 1
		}

		r[51] = 1
		r.Insert(utils.IntToBytes(uint64(slot.RFU.(int64)), 2, true), 52) // pre slot id

		if info != nil && info.GetType() == database.PET_TYPE {
			r2.Concat(slot.GetData(int16(slot.SlotID)))
		}

		slot.Update()
		resp.Concat(r)
		resp.Concat(r2)
		resp.Concat(slot.GetData(slot.SlotID))
	}

	for i := 0; i < 56; i++ {
		slotID := i + 0x0155
		newSlots[i].RFU = 0
		*slots[slotID] = newSlots[i]
	}

	time.AfterFunc(time.Second*3, func() {
		if s.Character != nil {
			s.Character.InventoryArrange = false
		}
	})

	return resp, nil
}

func (h *ArrangeBankHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {

	if s.User == nil || s.Character == nil {
		s.OnClose()
		return nil, nil
	}

	if s.Character.TradeID != "" {
		return nil, nil
	}

	if s.Character.BankInventoryArrange {
		s.Character.Socket.Write(messaging.InfoMessage("You need to wait 3 more seconds to arrange."))
		return nil, nil
	}
	s.Character.BankInventoryArrange = true

	user := s.User
	if user == nil {
		return nil, nil
	}

	slots, err := s.Character.InventorySlots()
	if err != nil {
		return nil, err
	}

	slots = slots[0x43:0x133]
	resp := utils.Packet{}
	for page := 0; page < 4; page++ {

		newSlots := make([]database.InventorySlot, 60)
		for i := 0; i < 60; i++ {
			index := page*60 + i
			newSlots[i] = *slots[index]
			if newSlots[i].ItemID == 0 {
				newSlots[i].ItemID = math.MaxInt64
			}
			newSlots[i].RFU = int64(index + 0x43)
		}

		sort.Slice(newSlots, func(i, j int) bool {
			return newSlots[i].ItemID < newSlots[j].ItemID
		})

		for i := 0; i < 60; i++ {
			slot := &newSlots[i]
			r, r2 := ARRANGE_BANK_ITEM, utils.Packet{}

			if slot.ItemID == math.MaxInt64 {
				slot.ItemID = 0
			}

			slot.SlotID = int16(page*60 + i + 0x43)
			r.Insert(utils.IntToBytes(uint64(slot.ItemID), 4, true), 6) // item id

			info := database.Items[slot.ItemID]
			if info != nil && slot.Activated { // using state
				if info.TimerType == 1 {
					r[10] = 3
				} else if info.TimerType == 3 {
					r[10] = 5
					r2 = database.GREEN_ITEM_COUNT
					r2.Insert(utils.IntToBytes(uint64(slot.SlotID), 2, true), 8)    // slot id
					r2.Insert(utils.IntToBytes(uint64(slot.Quantity), 4, true), 10) // item quantity
				}
			} else {
				r[10] = 0
			}

			if slot.ItemID == 0 {
				r[11] = 0
			} else if slot.Plus > 0 || slot.SocketCount > 0 {
				r[11] = 0xA2
			}

			r.Insert(utils.IntToBytes(uint64(slot.Quantity), 2, true), 12) // item quantity
			r.Insert(utils.IntToBytes(uint64(slot.SlotID), 2, true), 14)   // slot id

			if slot.Plus > 0 || slot.SocketCount > 0 {
				r.Insert(slot.GetUpgrades(), 16) // slot upgrades
				r[31] = byte(slot.SocketCount)   // socket count
				r.Insert(slot.GetSockets(), 32)  // slot sockets
				r.SetLength(0x4D)
			}

			if i == 60 {
				r[47] = 1
			}
			r[48] = byte(page)

			r.Insert(utils.IntToBytes(uint64(slot.RFU.(int64)), 2, true), 49) // pre slot id

			if info != nil && info.GetType() == database.PET_TYPE {
				r2.Concat(slot.GetData(int16(slot.SlotID)))
			}

			slot.Update()
			resp.Concat(r)
			resp.Concat(r2)
			resp.Concat(slot.GetData(slot.SlotID))
		}

		for i := 0; i < 60; i++ {
			slotID := page*60 + i
			newSlots[i].RFU = 0
			*slots[slotID] = newSlots[i]
		}
	}

	user.Update()
	time.AfterFunc(time.Second*3, func() {
		if s.Character != nil {
			s.Character.BankInventoryArrange = false
		}
	})
	return resp, nil
}

func (h *DepositHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {

	if s.User == nil || s.Character == nil {
		s.Conn.Close()
		return nil, nil
	}

	u := s.User
	if u == nil {
		s.Conn.Close()
		return nil, nil
	}

	c := s.Character
	if c == nil {
		s.Conn.Close()
		return nil, nil
	}

	if !c.IsOnline || !c.IsActive {
		s.Conn.Close()
		return nil, nil
	}

	if c.TradeID == "" {
		gold := uint64(utils.BytesToInt(data[6:14], true))
		if c.Gold >= gold {
			c.LootGold(-gold)
			u.BankGold += gold

			go logging.AddLogFile(3, c.Socket.User.ID+" idli kullanici ("+c.Name+") karakterinden ("+strconv.Itoa(int(gold))+") GOLD bankaya yatırdı. (BANK)")

			go u.Update()
			return c.GetGold(), nil
		}
	}

	return nil, nil
}

func (h *WithdrawHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {

	if s.User == nil || s.Character == nil {
		s.Conn.Close()
		return nil, nil
	}

	u := s.User
	if u == nil {
		s.Conn.Close()
		return nil, nil
	}

	c := s.Character
	if c == nil {
		s.Conn.Close()
		return nil, nil
	}

	if !c.IsOnline || !c.IsActive {
		s.Conn.Close()
		return nil, nil
	}

	if c.TradeID == "" {
		gold := uint64(utils.BytesToInt(data[6:14], true))
		if u.BankGold >= gold {
			c.LootGold(gold)
			u.BankGold -= gold

			go logging.AddLogFile(3, c.Socket.User.ID+" idli kullanici bankadan ("+c.Name+") isimli karaktere ("+strconv.Itoa(int(gold))+") GOLD çekti (BANK)")

			go u.Update()
			return c.GetGold(), nil
		}
	}

	return nil, nil
}

func (h *OpenHTMenuHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {

	if s.User == nil || s.Character == nil {
		s.Conn.Close()
		return nil, nil
	}

	u := s.User
	if u == nil {
		return nil, nil
	}

	resp := OPEN_HT_MENU
	r := GET_CASH
	r.Insert(utils.IntToBytes(u.NCash, 8, true), 8) // user nCash

	resp.Concat(r)
	return resp, nil
}

func (h *CloseHTMenuHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {

	if s.User == nil || s.Character == nil {
		s.Conn.Close()
		return nil, nil
	}

	return CLOSE_HT_MENU, nil
}

func (h *BuyHTItemHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {

	if s.User == nil || s.Character == nil {
		s.Conn.Close()
		return nil, nil
	}

	itemID := int(utils.BytesToInt(data[6:10], true))
	slotID := utils.BytesToInt(data[12:14], true)

	if item, ok := database.HTItems[itemID]; ok && item.IsActive && s.User.NCash >= uint64(item.Cash) {
		s.User.NCash -= uint64(item.Cash)
		itemCash := item.Cash
		info := database.Items[int64(itemID)]
		quantity := uint(1)
		if info.Timer > 0 && info.TimerType > 0 {
			quantity = uint(info.Timer)
		} else if qty, ok := htShopQuantites[info.ID]; ok {
			quantity = qty
		}

		item := &database.InventorySlot{ItemID: int64(itemID), Quantity: quantity}
		if info.GetType() == database.PET_TYPE {
			petInfo := database.Pets[int64(itemID)]
			petExpInfo := database.PetExps[int16(petInfo.Level)]

			targetExps := []int{petExpInfo.ReqExpEvo1, petExpInfo.ReqExpEvo2, petExpInfo.ReqExpEvo3, petExpInfo.ReqExpHt}
			item.Pet = &database.PetSlot{
				Fullness: 100, Loyalty: 100, PseudoID: 0,
				Exp:   uint64(targetExps[petInfo.Evolution-1]),
				HP:    petInfo.BaseHP,
				Level: byte(petInfo.Level),
				Name:  petInfo.Name,
				CHI:   petInfo.BaseChi,
			}
		}

		r, _, err := s.Character.AddItem(item, int16(slotID), false)
		if err != nil {
			return nil, err
		} else if r == nil {
			return nil, nil
		}

		resp := BUY_HT_ITEM
		resp.Insert(utils.IntToBytes(uint64(itemID), 4, true), 8)    // item id
		resp.Insert(utils.IntToBytes(uint64(quantity), 2, true), 14) // item quantity
		resp.Insert(utils.IntToBytes(uint64(slotID), 2, true), 16)   // slot id
		resp.Insert(utils.IntToBytes(s.User.NCash, 8, true), 52)     // user nCash

		resp.Concat(*r)

		go s.User.Update()
		go logging.AddLogFile(7, s.User.ID+" idli kullanici ("+s.Character.Name+") isimli karakteriyle HT Shopdan belirtilen itemi satın aldı "+info.Name+" Cash Karşılığı: "+strconv.Itoa(itemCash))
		return resp, nil
	}

	return nil, nil
}

func (h *ReplaceHTItemHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {

	if s.User == nil || s.Character == nil {
		s.Conn.Close()
		return nil, nil
	}

	slots, err := s.Character.InventorySlots()
	if err != nil {
		return nil, err
	}

	itemID := int(utils.BytesToInt(data[6:10], true))
	where := int16(utils.BytesToInt(data[10:12], true))
	to := int16(utils.BytesToInt(data[12:14], true))

	resp := REPLACE_HT_ITEM
	resp.Insert(utils.IntToBytes(uint64(itemID), 4, true), 8) // item id
	resp.Insert(utils.IntToBytes(uint64(where), 2, true), 12) // where slot id

	quantity := slots[where].Quantity

	r := database.ITEM_SLOT
	r.Insert(utils.IntToBytes(uint64(itemID), 4, true), 6)    // item id
	r.Insert(utils.IntToBytes(uint64(quantity), 2, true), 12) // item quantity
	r.Insert(utils.IntToBytes(uint64(where), 2, true), 14)    // where slot id
	resp.Concat(r)

	r, err = s.Character.ReplaceItem(itemID, where, to)
	if err != nil {
		return nil, err
	}

	resp.Concat(r)
	resp.Concat(slots[to].GetData(to))
	return resp, nil
}

func (h *DressUpHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {

	if s.User == nil || s.Character == nil {
		s.Conn.Close()
		return nil, nil
	}

	isHT := data[6] == 1
	if isHT {
		s.Character.HTVisibility = int(data[7])
		resp := HT_VISIBILITY
		resp[9] = data[7]

		itemsData, err := s.Character.ShowItems()
		if err != nil {
			return nil, err
		}

		p := nats.CastPacket{CastNear: true, CharacterID: s.Character.ID, Type: nats.SHOW_ITEMS, Data: itemsData}
		if err = p.Cast(); err != nil {
			return nil, err
		}

		resp.Concat(itemsData)

		return resp, nil
	}

	return nil, nil
}

func (h *SplitItemHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {

	if s.User == nil || s.Character == nil {
		s.Conn.Close()
		return nil, nil
	}

	where := uint16(utils.BytesToInt(data[6:8], true))
	to := uint16(utils.BytesToInt(data[8:10], true))
	quantity := uint16(utils.BytesToInt(data[10:12], true))

	return s.Character.SplitItem(where, to, quantity)
}

func (h *HolyWaterUpgradeHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {
	if s.User == nil || s.Character == nil {
		s.Conn.Close()
		return nil, nil
	}

	slots, err := s.Character.InventorySlots()
	if err != nil {
		return nil, err
	}

	itemSlot := int16(utils.BytesToInt(data[6:8], true))
	item := slots[itemSlot]
	if itemSlot == 0 || item.ItemID == 0 {
		return nil, nil
	}

	holyWaterSlot := int16(utils.BytesToInt(data[8:10], true))
	holyWater := slots[holyWaterSlot]
	if holyWaterSlot == 0 || holyWater.ItemID == 0 {
		return nil, nil
	}

	return s.Character.HolyWaterUpgrade(item, holyWater, itemSlot, holyWaterSlot)
}

func (h *UseConsumableHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {

	if s.User == nil || s.Character == nil {
		s.Conn.Close()
		return nil, nil
	}

	slots, err := s.Character.InventorySlots()
	if err != nil {
		return nil, err
	}

	itemID := int64(utils.BytesToInt(data[6:10], true))

	slotID := int16(utils.BytesToInt(data[10:12], true))

	item := slots[slotID]
	if item == nil || item.ItemID != itemID {
		return nil, nil
	}

	return s.Character.UseConsumable(item, slotID)
}

func (h *OpenBoxHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {

	if s.User == nil || s.Character == nil {
		s.Conn.Close()
		return nil, nil
	}

	slots, err := s.Character.InventorySlots()
	if err != nil {
		return nil, err
	}

	slotID := int16(utils.BytesToInt(data[6:8], true))

	item := slots[slotID]
	if item == nil {
		return nil, nil
	}

	return s.Character.UseConsumable(item, slotID)
}

func (h *OpenBoxHandler2) Handle(s *database.Socket, data []byte) ([]byte, error) {

	if s.User == nil || s.Character == nil {
		s.Conn.Close()
		return nil, nil
	}

	slots, err := s.Character.InventorySlots()
	if err != nil {
		return nil, err
	}

	itemID := utils.BytesToInt(data[6:10], true)
	slotID := int16(utils.BytesToInt(data[10:12], true))

	item := slots[slotID]
	if item == nil || item.ItemID != itemID {
		return nil, nil
	}

	return s.Character.UseConsumable(item, slotID)
}

func (h *ActivateTimeLimitedItemHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {

	if s.User == nil || s.Character == nil {
		s.Conn.Close()
		return nil, nil
	}

	slots, err := s.Character.InventorySlots()
	if err != nil {
		return nil, err
	}

	slotID := int16(utils.BytesToInt(data[6:8], true))
	item := slots[slotID]
	if item == nil {
		return nil, nil
	}

	info := database.Items[item.ItemID]
	if info == nil || info.Timer == 0 {
		return nil, nil
	}

	hasSameBuff := len(funk.Filter(slots, func(slot *database.InventorySlot) bool {

		// Birden fazla aynı türden itemi kapatma
		if item.ItemID == 15710267 || item.ItemID == 15710268 || item.ItemID == 15710269 || item.ItemID == 15710270 {
			if slot.Activated && slot.ItemID == 15710267 || slot.Activated && slot.ItemID == 15710268 || slot.Activated && slot.ItemID == 15710269 || slot.Activated && slot.ItemID == 15710270 {
				return true
			}
		}

		if item.ItemID == 15710271 || item.ItemID == 15710272 || item.ItemID == 15710273 || item.ItemID == 15710274 || item.ItemID == 18500104 {
			if slot.Activated && slot.ItemID == 15710271 || slot.Activated && slot.ItemID == 15710272 || slot.Activated && slot.ItemID == 15710273 || slot.Activated && slot.ItemID == 15710274 || slot.Activated && slot.ItemID == 18500104 {
				return true
			}
		}

		return slot.Activated && slot.ItemID == item.ItemID
	}).([]*database.InventorySlot)) > 0

	if hasSameBuff {
		return []byte{0xAA, 0x55, 0x04, 0x00, 0x59, 0x04, 0xFA, 0x03, 0x55, 0xAA}, nil
	}

	resp := utils.Packet{}
	item.Activated = !item.Activated
	item.InUse = !item.InUse
	resp.Concat(item.GetData(slotID))

	item.Update()
	statsData, _ := s.Character.GetStats()
	resp.Concat(statsData)
	return resp, nil
}

func (h *ActivateTimeLimitedItemHandler2) Handle(s *database.Socket, data []byte) ([]byte, error) {

	if s.User == nil || s.Character == nil {
		s.Conn.Close()
		return nil, nil
	}

	slots, err := s.Character.InventorySlots()
	if err != nil {
		return nil, err
	}

	where := int16(utils.BytesToInt(data[6:8], true))
	itemID := utils.BytesToInt(data[8:12], true)
	to := int16(utils.BytesToInt(data[12:14], true))

	item := slots[where]
	if item == nil || item.ItemID != itemID {
		return nil, nil
	}

	info := database.Items[item.ItemID]
	if info == nil || info.Timer == 0 {
		return nil, nil
	}

	hasSameBuff := len(funk.Filter(slots, func(slot *database.InventorySlot) bool {
		return slot.Activated && slot.ItemID == item.ItemID
	}).([]*database.InventorySlot)) > 0

	if hasSameBuff {
		return []byte{0xAA, 0x55, 0x04, 0x00, 0x59, 0x04, 0xFA, 0x03, 0x55, 0xAA}, nil
	}

	resp := utils.Packet{}
	item.Activated = !item.Activated
	item.InUse = !item.InUse
	//s.Conn.Write(item.GetData(where))
	s.Write(item.GetData(where))

	var itemData utils.Packet
	if slots[to].ItemID == 0 {
		itemData, err = s.Character.ReplaceItem(int(itemID), where, to)
	} else {
		itemData, err = s.Character.SwapItems(where, to)
	}

	if err != nil {
		return nil, err
	}
	resp.Concat(itemData)

	item.Update()
	statsData, _ := s.Character.GetStats()
	resp.Concat(statsData)
	return resp, nil
}

func (h *ToggleMountPetHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {

	if s.User == nil || s.Character == nil {
		s.Conn.Close()
		return nil, nil
	}

	return s.Character.TogglePet(), nil
}

func (h *TogglePetHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {
	if s.User == nil || s.Character == nil {
		s.Conn.Close()
		return nil, nil
	}

	return s.Character.TogglePet(), nil
}

func (h *PetCombatModeHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {
	if s.User == nil || s.Character == nil {
		s.Conn.Close()
		return nil, nil
	}
	CombatMode := utils.BytesToInt(data[7:8], true)
	slots, err := s.Character.InventorySlots()
	if err != nil {
		log.Println(err)
		return nil, nil
	}
	petSlot := slots[0x0A]
	pet := petSlot.Pet
	if pet == nil || petSlot.ItemID == 0 || !pet.IsOnline {
		return nil, nil
	}
	pet.PetCombatMode = int16(CombatMode)
	resp := PET_COMBAT
	resp.Insert(utils.IntToBytes(uint64(CombatMode), 1, true), 9)
	return resp, nil
}

func (h *EnchantBookHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {

	if s.User == nil || s.Character == nil {
		s.Conn.Close()
		return nil, nil
	}

	slots, err := s.Character.InventorySlots()
	if err != nil {
		return nil, err
	}

	bookID := int64(utils.BytesToInt(data[6:10], true))

	ReqItemsAmount := int16(utils.BytesToInt(data[10:11], true))
	matsSlotsIds := []int16{}
	matsIds := []int64{}

	index := 11
	for i := 0; i < int(ReqItemsAmount); i++ {
		slotid := int(utils.BytesToInt(data[index:index+2], true))
		index += 2
		matsSlotsIds = append(matsSlotsIds, int16(slotid))
		matsIds = append(matsIds, int64(slots[slotid].ItemID))
	}

	resp := utils.Packet{}
	prodData, err := s.Character.Enchant(bookID, matsSlotsIds, matsIds)
	if err != nil {
		return nil, err
	}
	resp.Concat(prodData)
	return resp, nil
}

func (h *FireworkHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {

	slotID, item, err := s.Character.FindItemInInventory(nil,
		3000040,
		13000241,
		16210019,
		16210020,
		17300008,
		17300009,
		17504859,
		17507515,
		17509032,
		17509049,
		92001033,
		92001065,
		92001110,
		99002335,
	)
	if err != nil || item == nil {
		return nil, err
	}
	if slotID == -1 {
		return nil, nil
	}

	resp := utils.Packet{0xAA, 0x55, 0x9B, 0x00, 0x72, 0x09, 0x55, 0xAA}
	coordinate := database.ConvertPointToLocation(s.Character.Coordinate)

	//itemID := utils.BytesToInt(data[6:10], true)
	//slotID := utils.BytesToInt(data[10:12], true)

	rotation := funk.RandomInt(0, 360)

	index := 6
	resp.Insert(utils.IntToBytes(uint64(rotation), 2, true), index) // coordinate-x
	index += 2
	resp.Insert(utils.FloatToBytes(coordinate.X, 4, true), index) // coordinate-x
	index += 4
	resp.Insert(utils.FloatToBytes(coordinate.Y, 4, true), index) // coordinate-y
	index += 4

	data = data[6 : len(data)-2]
	resp.Insert(data, index)
	resp.SetLength(int16(len(resp)) - 6)

	r := s.Character.DecrementItem(slotID, 1)
	s.Write(*r)

	p := nats.CastPacket{CastNear: true, CharacterID: s.Character.ID, Data: resp}
	if err := p.Cast(); err == nil {
		s.Write(resp)
	}

	return resp, nil
}
