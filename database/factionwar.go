package database

import (
	"fmt"
	"time"

	"hero-server/utils"
)

var (
	//ANNOUNCEMENT     = utils.Packet{0xAA, 0x55, 0x00, 0x00, 0x71, 0x06, 0x00, 0x55, 0xAA}
	FACTION_WAR_START = utils.Packet{
		0xAA, 0x55, 0x23, 0x00, 0x65, 0x01, 0x0a, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x55, 0xaa}
	FACTION_WAR_UPDATE = utils.Packet{
		0xAA, 0x55, 0x23, 0x00, 0x65, 0x03, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x55, 0xaa}
	zhuangFactionWarMembersList []*Character
	shaoFactionWarMembersList   []*Character
	zhuangFactionWarPoints      int
	shaoFactionWarPoints        int
	timingFactionWar            int
	isFactionWarEntranceActive  bool
	isFactionWarStarted         bool
	minLevel                    int64
	maxLevel                    int64
)

func PrepareFactionWar(min, max int64) {
	go startFactionWarCounting(600)
	minLevel = min
	maxLevel = max
}

func startFactionWarCounting(cd int) {
	isFactionWarEntranceActive = true
	if cd >= 120 {

		checkMembersInMap()
		msg := fmt.Sprintf("Faction war level %d-%d will start in %d minutes. Enter faction war at Hero Battle Manager", minLevel, maxLevel, cd/60)
		makeAnnouncement(msg)
		time.AfterFunc(time.Second*60, func() {
			startFactionWarCounting(cd - 60)
		})
	} else if cd > 0 {
		checkMembersInMap()
		msg := fmt.Sprintf("Faction war level %d-%d will start in %d seconds.", minLevel, maxLevel, cd)
		makeAnnouncement(msg)
		time.AfterFunc(time.Second*10, func() {
			startFactionWarCounting(cd - 10)
		})
	}
	if cd <= 0 {
		StartFactionWar()
		isFactionWarEntranceActive = false
	}
}
func StartFactionWar() {

	checkMembersInMap()

	resp := FACTION_WAR_START
	timingFactionWar = 600
	isFactionWarStarted = true

	resp.Overwrite(utils.IntToBytes(uint64(len(zhuangFactionWarMembersList)), 4, true), 8) //Zhuang numbers
	resp.Overwrite(utils.IntToBytes(uint64(zhuangFactionWarPoints), 4, true), 12)          //Zhuang points
	resp.Overwrite(utils.IntToBytes(uint64(len(shaoFactionWarMembersList)), 4, true), 22)  //Shao number
	resp.Overwrite(utils.IntToBytes(uint64(shaoFactionWarPoints), 4, true), 26)            //Shao points
	resp.Overwrite(utils.IntToBytes(uint64(timingFactionWar), 4, true), 35)                //Time

	for _, c := range zhuangFactionWarMembersList {
		c.Socket.Write(resp)
	}
	for _, c := range shaoFactionWarMembersList {
		c.Socket.Write(resp)
	}
	updateFactionWarBar()
}
func updateFactionWarBar() {

	if timingFactionWar <= 0 {
		return
	}

	checkMembersInMap()

	for _, c := range zhuangFactionWarMembersList {
		if c == nil {
			return
		}
		resp := FACTION_WAR_UPDATE
		resp.Overwrite(utils.IntToBytes(uint64(len(zhuangFactionWarMembersList)), 4, true), 7) //Zhuang numbers
		resp.Overwrite(utils.IntToBytes(uint64(zhuangFactionWarPoints), 4, true), 11)          //Zhuang points
		resp.Overwrite(utils.IntToBytes(uint64(len(shaoFactionWarMembersList)), 4, true), 21)  //Shao number///////////////
		resp.Overwrite(utils.IntToBytes(uint64(shaoFactionWarPoints), 4, true), 25)            //Shao points
		resp.Overwrite(utils.IntToBytes(uint64(timingFactionWar), 4, true), 34)                //Time
		c.Socket.Write(resp)
	}
	for _, c := range shaoFactionWarMembersList {
		if c == nil {
			return
		}
		resp := FACTION_WAR_UPDATE
		resp.Overwrite(utils.IntToBytes(uint64(len(zhuangFactionWarMembersList)), 4, true), 7) //Zhuang numbers
		resp.Overwrite(utils.IntToBytes(uint64(zhuangFactionWarPoints), 4, true), 11)          //Zhuang points
		resp.Overwrite(utils.IntToBytes(uint64(len(shaoFactionWarMembersList)), 4, true), 21)  //Shao number///////////////
		resp.Overwrite(utils.IntToBytes(uint64(shaoFactionWarPoints), 4, true), 25)            //Shao points
		resp.Overwrite(utils.IntToBytes(uint64(timingFactionWar), 4, true), 34)                //Time
		c.Socket.Write(resp)
	}

	//AddPointsToFactionWarFaction(len(zhuangFactionWarMembersList), 1)
	//AddPointsToFactionWarFaction(len(shaoFactionWarMembersList), 2)

	timingFactionWar--
	if timingFactionWar <= 0 {
		finishFactionWar()
		return
	}
	time.AfterFunc(time.Second*2, func() {
		updateFactionWarBar()
	})
}
func AddPointsToFactionWarFaction(points int, faction int) {
	if faction == 1 {
		zhuangFactionWarPoints += points
		return
	}
	shaoFactionWarPoints += points
}
func IsFactionWarEntranceActive() bool {
	return isFactionWarEntranceActive
}
func IsFactionWarStarted() bool {
	return isFactionWarStarted
}
func AddMemberToFactionWar(char *Character) {
	if !isFactionWarEntranceActive {
		return
	}
	if char.Level < int(minLevel) || char.Level > int(maxLevel) {
		return
	}
	checkMembersInMap()

	coordinate := &utils.Location{X: 315, Y: 447}
	data, _ := char.ChangeMap(255, coordinate)
	if char.Faction == 2 {
		coordinate := &utils.Location{X: 211, Y: 73}
		data, _ = char.ChangeMap(255, coordinate)
	}
	//char.IsinWar = true
	char.Socket.Write(data)
}
func finishFactionWar() {
	isFactionWarStarted = false
	isFactionWarEntranceActive = false

	if zhuangFactionWarPoints > shaoFactionWarPoints { //zhuang won
		msg := fmt.Sprintf("Zhuang faction won the faction war!")
		makeAnnouncement(msg)

		for _, c := range zhuangFactionWarMembersList { //give item to all zhuangs
			if c == nil {
				return
			}
			c.IsinWar = false
			item := &InventorySlot{ItemID: 99009117, Quantity: uint(1)}
			r, _, err := c.AddItem(item, -1, false)
			if err == nil {
				c.Socket.Write(*r)
			}

			coin := &InventorySlot{ItemID: 18500095, Quantity: uint(100)}
			rc, _, err := c.AddItem(coin, -1, false)
			if err == nil {
				c.Socket.Write(*rc)
			}

			data, _ := c.ChangeMap(1, nil)
			c.Socket.Write(data)
		}
		for _, c := range shaoFactionWarMembersList { //give item to all shaos
			if c == nil {
				return
			}
			c.IsinWar = false
			item := &InventorySlot{ItemID: 99009118, Quantity: uint(1)}
			r, _, err := c.AddItem(item, -1, false)
			if err == nil {
				c.Socket.Write(*r)
			}

			coin := &InventorySlot{ItemID: 18500095, Quantity: uint(50)}
			rc, _, err := c.AddItem(coin, -1, false)
			if err == nil {
				c.Socket.Write(*rc)
			}

			data, _ := c.ChangeMap(1, nil)
			c.Socket.Write(data)
		}

	} else { // shao won
		msg := fmt.Sprintf("Shao faction won the faction war!")
		makeAnnouncement(msg)
		for _, c := range zhuangFactionWarMembersList { //give item to all zhuangs
			if c == nil {
				return
			}
			item := &InventorySlot{ItemID: 99009118, Quantity: uint(1)}
			r, _, err := c.AddItem(item, -1, false)
			if err == nil {
				c.Socket.Write(*r)
			}

			coin := &InventorySlot{ItemID: 18500095, Quantity: uint(50)}
			rc, _, err := c.AddItem(coin, -1, false)
			if err == nil {
				c.Socket.Write(*rc)
			}

			data, _ := c.ChangeMap(1, nil)
			c.Socket.Write(data)
		}
		for _, c := range shaoFactionWarMembersList { //give item to all shaos
			if c == nil {
				return
			}
			item := &InventorySlot{ItemID: 99009117, Quantity: uint(1)}
			r, _, err := c.AddItem(item, -1, false)
			if err == nil {
				c.Socket.Write(*r)
			}

			coin := &InventorySlot{ItemID: 18500095, Quantity: uint(100)}
			rc, _, err := c.AddItem(coin, -1, false)
			if err == nil {
				c.Socket.Write(*rc)
			}

			data, _ := c.ChangeMap(1, nil)
			c.Socket.Write(data)
		}
	}
	zhuangFactionWarPoints = 0
	shaoFactionWarPoints = 0
}
func checkMembersInMap() {
	zhuangFactionWarMembersList = nil
	shaoFactionWarMembersList = nil
	for _, member := range FindCharactersInMap(255) {
		if member.Faction == 1 {
			zhuangFactionWarMembersList = append(zhuangFactionWarMembersList, member)
		}
		if member.Faction == 2 {
			shaoFactionWarMembersList = append(shaoFactionWarMembersList, member)
		}
	}
}
