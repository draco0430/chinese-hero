package database

import (
	"fmt"
	"sync"
	"time"
)

var (
	LastManCharacters = make(map[int]*Character)
	LastManMutex      sync.Mutex

	CanJoinLastMan = false
	LastmanStarted = false
)

func StartLastManTimer(prepareWarStart int) {
	min, sec := secondsToMinutes(prepareWarStart)
	msg := fmt.Sprintf("Last Man Standing is starting in %d minutes %d seconds", min, sec)
	msg2 := fmt.Sprintf("Please join from Master Bak on Marketplace.")
	makeAnnouncement(msg)
	makeAnnouncement(msg2)
	if prepareWarStart > 0 {
		time.AfterFunc(time.Second*10, func() {
			StartLastManTimer(prepareWarStart - 10)
		})
	} else {
		StartLastManWar()
	}
}

func StartLastManWar() {
	CanJoinLastMan = false
	LastmanStarted = true
	makeAnnouncement("Last Man Standing has started! GEAR UP!")
	LastManRunning()
}

func LastManRunning() {
	for range time.Tick(10 * time.Second) {
		LastManMutex.Lock()
		totalChr := len(LastManCharacters)
		LastManMutex.Unlock()

		if totalChr > 1 {

		} else {
			CanJoinLastMan = false
			LastmanStarted = false

			winner := ""

			LastManMutex.Lock()
			for k := range LastManCharacters {
				winner = LastManCharacters[k].Name
				delete(LastManCharacters, LastManCharacters[k].ID)
			}
			LastManMutex.Unlock()

			msg := fmt.Sprintf("Last Man Standing Winner : %s", winner)
			makeAnnouncement(msg)
			return
		}
	}
}
