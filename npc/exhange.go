package npc

import (
	"hero-server/database"
	"hero-server/utils"
)

type BuyItemHandler struct {
}

type SellItemHandler struct {
}

var ()

func (h *BuyItemHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {

	c := s.Character
	if c == nil {
		return nil, nil
	}

	if s.Character.TradeID != "" {
		return nil, nil
	}

	itemID := utils.BytesToInt(data[6:10], true)
	quantity := utils.BytesToInt(data[10:12], true)
	slotID := int16(utils.BytesToInt(data[16:18], true))

	npcID := int(utils.BytesToInt(data[18:22], true))
	shopID, ok := shops[npcID]
	if !ok {
		shopID = 25
	}

	shop, ok := database.Shops[shopID]
	if !ok {
		return nil, nil
	}

	canPurchase := shop.IsPurchasable(int(itemID))
	if !canPurchase {
		return nil, nil
	}

	slots, err := c.InventorySlots()
	if err != nil {
		return nil, err
	}

	info := database.Items[itemID]
	cost := uint64(info.BuyPrice) * uint64(quantity)

	if npcID == 20160 {
		var slot int16
		var coin *database.InventorySlot
		var err error

		if info.ID == 100080299 {
			slot, coin, err = c.FindItemInInventory(nil, 100080300)
			if err != nil {
				return nil, err
			} else if slot == -1 {
				return nil, nil
			}
		} else if info.ID == 100080300 {
			slot, coin, err = c.FindItemInInventory(nil, 100080180)
			if err != nil {
				return nil, err
			} else if slot == -1 {
				return nil, nil
			}
		} else {
			slot, coin, err = c.FindItemInInventory(nil, 100080299)
			if err != nil {
				return nil, err
			} else if slot == -1 {
				return nil, nil
			}
		}

		var item *database.InventorySlot

		if slots[slotID].ItemID == 0 && cost <= uint64(coin.Quantity) && quantity > 0 {
			resp := c.DecrementItem(slot, uint(cost))
			if info.TimerType > 0 {
				item = &database.InventorySlot{ItemID: itemID, Quantity: uint(info.Timer)}
			} else {
				item = &database.InventorySlot{ItemID: itemID, Quantity: uint(quantity)}
			}

			data, _, err := c.AddItem(item, slotID, false)
			if err != nil {
				return nil, err
			} else if data == nil {
				return nil, nil
			}

			resp.Concat(*data)

			return *resp, nil
		}
	} else if npcID == 20173 {
		slot, coin, err := c.FindItemInInventory(nil, 17502306)
		if err != nil {
			return nil, err
		} else if slot == -1 {
			return nil, nil
		}

		var item *database.InventorySlot

		if slots[slotID].ItemID == 0 && cost <= uint64(coin.Quantity) && quantity > 0 {
			resp := c.DecrementItem(slot, uint(cost))
			if info.TimerType > 0 {
				item = &database.InventorySlot{ItemID: itemID, Quantity: uint(info.Timer)}
			} else {
				item = &database.InventorySlot{ItemID: itemID, Quantity: uint(quantity)}
			}

			data, _, err := c.AddItem(item, slotID, false)
			if err != nil {
				return nil, err
			} else if data == nil {
				return nil, nil
			}

			resp.Concat(*data)

			return *resp, nil
		}
	} else if npcID == 20317 {
		slot, coin, err := c.FindItemInInventory(nil, 18500423)
		if err != nil {
			return nil, err
		} else if slot == -1 {
			return nil, nil
		}

		if info.ID == 18500427 {
			slot, coin, err = c.FindItemInInventory(nil, 100080180)
			if err != nil {
				return nil, err
			} else if slot == -1 {
				return nil, nil
			}
		}
		/*
			if info.ID == 99004213 || info.ID == 99004113 || info.ID == 99009011 || info.ID == 99004013 || info.ID == 99004203 || info.ID == 99004103 || info.ID == 99009001 || info.ID == 99004003 {
				slot, coin, err = c.FindItemInInventory(nil, 18500300)
				if err != nil {
					return nil, err
				} else if slot == -1 {
					return nil, nil
				}
			}
		*/

		var item *database.InventorySlot
		if slots[slotID].ItemID == 0 && cost <= uint64(coin.Quantity) && quantity > 0 {
			resp := c.DecrementItem(slot, uint(cost))
			if info.TimerType > 0 {
				item = &database.InventorySlot{ItemID: itemID, Quantity: uint(info.Timer)}
			} else {
				item = &database.InventorySlot{ItemID: itemID, Quantity: uint(quantity)}
			}

			data, _, err := c.AddItem(item, slotID, false)
			if err != nil {
				return nil, err
			} else if data == nil {
				return nil, nil
			}

			resp.Concat(*data)

			return *resp, nil
		}
	} else if npcID == 20051 {
		slot, coin, err := c.FindItemInInventory(nil, 18500300)
		if err != nil {
			return nil, err
		} else if slot == -1 {
			return nil, nil
		}

		var item *database.InventorySlot

		if slots[slotID].ItemID == 0 && cost <= uint64(coin.Quantity) && quantity > 0 {
			resp := c.DecrementItem(slot, uint(cost))
			if info.TimerType > 0 {
				item = &database.InventorySlot{ItemID: itemID, Quantity: uint(info.Timer)}
			} else {
				item = &database.InventorySlot{ItemID: itemID, Quantity: uint(quantity)}
			}

			data, _, err := c.AddItem(item, slotID, false)
			if err != nil {
				return nil, err
			} else if data == nil {
				return nil, nil
			}

			resp.Concat(*data)

			return *resp, nil
		}
	} else if npcID == 20293 || npcID == 20295 {
		slot, coin, err := c.FindItemInInventory(nil, 99002478)
		if err != nil {
			return nil, err
		} else if slot == -1 {
			return nil, nil
		}

		var item *database.InventorySlot

		if slots[slotID].ItemID == 0 && cost <= uint64(coin.Quantity) && quantity > 0 {
			resp := c.DecrementItem(slot, uint(cost))
			if info.TimerType > 0 {
				item = &database.InventorySlot{ItemID: itemID, Quantity: uint(info.Timer)}
			} else {
				item = &database.InventorySlot{ItemID: itemID, Quantity: uint(quantity)}
			}

			data, _, err := c.AddItem(item, slotID, false)
			if err != nil {
				return nil, err
			} else if data == nil {
				return nil, nil
			}

			resp.Concat(*data)

			return *resp, nil
		}
	} else if npcID == 23714 {
		slot, coin, err := c.FindItemInInventory(nil, 15710384)
		if err != nil {
			return nil, err
		} else if slot == -1 {
			return nil, nil
		}

		var item *database.InventorySlot

		if slots[slotID].ItemID == 0 && cost <= uint64(coin.Quantity) && quantity > 0 {
			resp := c.DecrementItem(slot, uint(cost))
			if info.TimerType > 0 {
				item = &database.InventorySlot{ItemID: itemID, Quantity: uint(info.Timer)}
			} else {
				item = &database.InventorySlot{ItemID: itemID, Quantity: uint(quantity)}
			}

			data, _, err := c.AddItem(item, slotID, false)
			if err != nil {
				return nil, err
			} else if data == nil {
				return nil, nil
			}

			resp.Concat(*data)

			return *resp, nil
		}
	} else if npcID == 23747 {
		slot, coin, err := c.FindItemInInventory(nil, 18500609)
		if err != nil {
			return nil, err
		} else if slot == -1 {
			return nil, nil
		}

		var item *database.InventorySlot

		if slots[slotID].ItemID == 0 && cost <= uint64(coin.Quantity) && quantity > 0 {
			resp := c.DecrementItem(slot, uint(cost))
			if info.TimerType > 0 {
				item = &database.InventorySlot{ItemID: itemID, Quantity: uint(info.Timer)}
			} else {
				item = &database.InventorySlot{ItemID: itemID, Quantity: uint(quantity)}
			}

			data, _, err := c.AddItem(item, slotID, false)
			if err != nil {
				return nil, err
			} else if data == nil {
				return nil, nil
			}

			resp.Concat(*data)

			return *resp, nil
		}
	} else if npcID == 20088 {
		slot, coin, err := c.FindItemInInventory(nil, 18500851)
		if err != nil {
			return nil, err
		} else if slot == -1 {
			return nil, nil
		}

		var item *database.InventorySlot

		if slots[slotID].ItemID == 0 && cost <= uint64(coin.Quantity) && quantity > 0 {
			resp := c.DecrementItem(slot, uint(cost))
			if info.TimerType > 0 {
				item = &database.InventorySlot{ItemID: itemID, Quantity: uint(info.Timer)}
			} else {
				item = &database.InventorySlot{ItemID: itemID, Quantity: uint(quantity)}
			}

			data, _, err := c.AddItem(item, slotID, false)
			if err != nil {
				return nil, err
			} else if data == nil {
				return nil, nil
			}

			resp.Concat(*data)

			return *resp, nil
		}
	} else if npcID == 20169 {
		slot, coin, err := c.FindItemInInventory(nil, 18500934)
		if err != nil {
			return nil, err
		} else if slot == -1 {
			return nil, nil
		}

		var item *database.InventorySlot

		if slots[slotID].ItemID == 0 && cost <= uint64(coin.Quantity) && quantity > 0 {
			resp := c.DecrementItem(slot, uint(cost))
			if info.TimerType > 0 {
				item = &database.InventorySlot{ItemID: itemID, Quantity: uint(info.Timer)}
			} else {
				item = &database.InventorySlot{ItemID: itemID, Quantity: uint(quantity)}
			}

			data, _, err := c.AddItem(item, slotID, false)
			if err != nil {
				return nil, err
			} else if data == nil {
				return nil, nil
			}

			resp.Concat(*data)

			return *resp, nil
		}
	} else {
		if slots[slotID].ItemID == 0 && cost <= c.Gold && quantity > 0 { // slot is empty, player can afford and quantity is positive
			percent := 0
			waterSpirit, _ := database.FindBuffByID(19000019, c.ID)
			if waterSpirit != nil {
				percent = 10
			}

			if percent > 0 {
				cost = cost - (cost*uint64(percent))/100
			}

			c.LootGold(-cost)
			item := &database.InventorySlot{ItemID: itemID, Quantity: uint(quantity)}

			if item.ItemID == 100080002 {
				item.Quantity = 3
			}

			if info.GetType() == database.PET_TYPE {
				petInfo := database.Pets[item.ItemID]
				expInfo := database.PetExps[petInfo.Level-1]

				item.Pet = &database.PetSlot{
					Fullness: 100, Loyalty: 100,
					Exp:   uint64(expInfo.ReqExpEvo1),
					HP:    petInfo.BaseHP,
					Level: byte(petInfo.Level),
					Name:  petInfo.Name,
					CHI:   petInfo.BaseChi}
			}

			resp, _, err := c.AddItem(item, slotID, false)
			if err != nil {
				return nil, err
			} else if resp == nil {
				return nil, nil
			}

			return *resp, nil
		}
	}

	return nil, nil
}

func (h *SellItemHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {

	c := s.Character
	if c == nil {
		return nil, nil
	}

	slots, err := c.InventorySlots()
	if err != nil {
		return nil, err
	}

	c.Looting.Lock()
	defer c.Looting.Unlock()

	itemID := utils.BytesToInt(data[6:10], true)
	quantity := int(utils.BytesToInt(data[10:12], true))
	slotID := int16(utils.BytesToInt(data[12:14], true))

	item := database.Items[itemID]
	slot := slots[slotID]

	if !item.Tradable {
		return nil, nil
	}

	multiplier := 0
	if slot.ItemID == itemID && quantity > 0 && uint(quantity) <= slot.Quantity {
		upgs := slot.GetUpgrades()
		for i := uint8(0); i < slot.Plus; i++ {
			upg := upgs[i]
			if code, ok := database.HaxCodes[int(upg)]; ok {
				multiplier += code.SaleMultiplier
			}
		}

		multiplier /= 1000
		if multiplier == 0 {
			multiplier = 1
		}

		unitPrice := uint64(item.SellPrice) * uint64(multiplier)
		if slot.Plus > 0 {
			unitPrice *= uint64(slot.Plus)
		}

		return c.SellItem(int(itemID), int(slotID), int(quantity), unitPrice)
	}

	return nil, nil
}
