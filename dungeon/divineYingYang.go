package dungeon

import (
	"time"

	"hero-server/database"
	"hero-server/messaging"
	"hero-server/utils"
)

var (
	IsDivineDungeonClosed = false
)

func StartDivineYingYang(party *database.Party) {

	server := 1
	if !IsDivineDungeonClosed {
		IsDivineDungeonClosed = true
	} else {
		msg := messaging.InfoMessage("All dungeons are full at this moment, come back later. ")
		party.Leader.Socket.Write(msg)
		return
	}
	party.Leader.Socket.User.ConnectedServer = server
	//party.Leader.Socket.User.SelectedServerID = server
	data, _ := party.Leader.ChangeMap(215, nil)
	party.Leader.Socket.Write(data)
	for _, member := range party.Members {

		member.Character.Socket.User.ConnectedServer = server
		//member.Character.Socket.User.SelectedServerID = server
		member.IsDungeon = true
		data, _ := member.Character.ChangeMap(215, nil)
		member.Character.Socket.Write(data)
		go StartTimerDivineYingYang(member.Character.Socket, 900)
	}

	go StartTimerDivineYingYang(party.Leader.Socket, 1800)
	go SetDivineDungeonOpenAfterTime(server)
	go database.CountYingYangMobs(party.Leader.Map)

}
func SetDivineDungeonOpenAfterTime(server int) {
	time.Sleep(time.Minute * 30)
	IsDivineDungeonClosed = false

}
func StartTimerDivineYingYang(s *database.Socket, seconds int) {
	resp := DUNGEON_TIMER
	resp.Overwrite(utils.IntToBytes(uint64(seconds), 4, true), 6)
	s.Write(DUNGEON_TIMER)

	time.AfterFunc(time.Minute*30, func() {
		if s.Character.Map == 243 && s.Character.IsOnline {
			resp := utils.Packet{}
			resp.Concat(messaging.InfoMessage("Your time has ended. Come again when you are stronger. Teleporting to safe zone."))
			coordinate := &utils.Location{X: 513, Y: 467}
			data, _ := s.Character.ChangeMap(24, coordinate)
			resp.Concat(data)
			s.Write(resp)
			s.Character.IsDungeon = false
		}
	})
}
