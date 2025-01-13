package dungeon

import (
	"time"

	"hero-server/database"
	"hero-server/messaging"
	"hero-server/nats"
	"hero-server/utils"
)

var (
	DUNGEON_TIMER   = utils.Packet{0xAA, 0x55, 0x06, 0x00, 0xC0, 0x17, 0x09, 0x07, 0x00, 0x00, 0x55, 0xAA}
	IsDungeonClosed = false
	YY_TIME_LIMIT   = 30
	ANNOUNCEMENT    = utils.Packet{0xAA, 0x55, 0x00, 0x00, 0x71, 0x06, 0x00, 0x55, 0xAA}
)

func StartYingYang(party *database.Party) {

	server := 1
	if !IsDungeonClosed {
		IsDungeonClosed = true
	}
	party.Leader.Socket.User.ConnectedServer = server
	party.Leader.IsDungeon = true
	party.Leader.Update()
	data, _ := party.Leader.ChangeMap(243, nil)
	party.Leader.Socket.Write(data)
	for _, member := range party.Members {
		member.Character.Socket.User.ConnectedServer = server
		member.IsDungeon = true
		member.Update()
		data, _ := member.Character.ChangeMap(243, nil)
		member.Character.Socket.Write(data)
		go StartTimerYingYang(member.Character.Socket, YY_TIME_LIMIT)
	}

	go StartTimerYingYang(party.Leader.Socket, YY_TIME_LIMIT)
	go SetDungeonOpenAfterTime(server)
	//go database.CountYingYangMobs(party.Leader.Map)
	go CheckYingYang(party)
}

func SetDungeonOpenAfterTime(server int) {
	time.Sleep(time.Minute * time.Duration(YY_TIME_LIMIT))

	IsDungeonClosed = false
}

func StartTimerYingYang(s *database.Socket, minutes int) {
	resp := DUNGEON_TIMER
	resp.Overwrite(utils.IntToBytes(uint64(minutes*60), 4, true), 6)
	s.Write(DUNGEON_TIMER)

	time.AfterFunc(time.Minute*time.Duration(YY_TIME_LIMIT), func() {
		if s.Character.Map == 243 && s.Character.IsOnline {
			s.Character.IsDungeon = false
			s.Character.Update()
			resp := utils.Packet{}
			resp.Concat(messaging.InfoMessage("Your time has ended. Come again when you are stronger. Teleporting to safe zone."))
			coordinate := &utils.Location{X: 37, Y: 453}
			data, _ := s.Character.ChangeMap(17, coordinate)
			resp.Concat(data)
			s.Write(resp)
		}
	})
}

func CheckYingYang(party *database.Party) {
	for range time.Tick(1 * time.Second) {
		if party == nil {
			return
		}

		if party.Leader.Map != 243 {
			party.Leader.IsDungeon = false
			party.Leader.Update()
			for _, member := range party.Members {
				member.IsDungeon = false
				member.Update()
				if member.Map == 243 {
					resp := utils.Packet{}
					coordinate := &utils.Location{X: 37, Y: 453}
					data, _ := member.ChangeMap(17, coordinate)
					resp.Concat(data)
					member.Socket.Write(resp)
				}
			}
			return
		}

		database.YingYangMobsMutex.Lock()
		counter := database.YingYangMobsCounter[party.Leader.Map]
		database.YingYangMobsMutex.Unlock()
		if counter == nil {
			return
		}

		if counter.BeastMaster > 0 || counter.Paechun > 0 {
			continue
		} else {
			announceMsg := party.Leader.Name
			for _, member := range party.Members {
				member.IsDungeon = false
				member.Update()
				announceMsg = announceMsg + ", " + member.Name
				member.Socket.Write(messaging.InfoMessage("You finished the dungeon successfully !"))

				gomap, _ := member.ChangeMap(17, &utils.Location{X: 37, Y: 453})
				member.Socket.Write(gomap)
			}

			party.Leader.IsDungeon = false
			party.Leader.Update()

			gomap, _ := party.Leader.ChangeMap(17, &utils.Location{X: 37, Y: 453})
			party.Leader.Socket.Write(gomap)

			//IsDungeonClosed = false
			makeAnnouncement("Ying-Yang dungeon got pwned by:" + announceMsg)
			return
		}
	}
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
