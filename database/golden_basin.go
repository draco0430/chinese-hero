package database

import (
	"database/sql"
	"fmt"
	"time"

	null "gopkg.in/guregu/null.v3"
)

var (
	GoldenBasinArea    *GoldenBasin
	CanJoinGoldenBasin = false
)

type GoldenBasin struct {
	ID        int       `db:"id"`
	FactionID int       `db:"faction_id"`
	ExpiresAt null.Time `db:"expires_at" json:"expires_at"`
}

func (g *GoldenBasin) Update() error {
	_, err := db.Update(g)
	return err
}

func getGoldenBasin() error {
	var goldenBasin []*GoldenBasin
	query := `select * from data.golden_basin;`

	if _, err := db.Select(&goldenBasin, query); err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		return fmt.Errorf("getGoldenBasin: %s", err.Error())
	}

	GoldenBasinArea = goldenBasin[0]

	return nil
}

func startGoldenBasinTimer(prepareWarStart int) {
	//min, sec := secondsToMinutes(prepareWarStart)
	//msg := fmt.Sprintf("%d mins %d secs after the Golden Basin War will start.", min, sec)
	//makeAnnouncement(msg)
	if prepareWarStart > 0 {
		time.AfterFunc(time.Second*10, func() {
			startGoldenBasinTimer(prepareWarStart - 10)
		})
	} else {
		CanJoinGoldenBasin = true
		//msg2 := "Please join from Faction District !"
		//makeAnnouncement(msg2)
	}
}

func StartGoldenBasinWar() {
	GoldenBasinArea.FactionID = 0
	GoldenBasinArea.Update()
	startGoldenBasinTimer(300)
}
