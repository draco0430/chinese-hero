package database

import (
	"database/sql"
	"fmt"
	"time"

	null "gopkg.in/guregu/null.v3"
)

var (
	FiveClans     = make(map[int]*FiveClan)
	FiveclanMobs  = []int{423308, 423310, 423312, 423314, 423316}
	FiveclanBuffs = []int{70001, 70002, 70003, 70004, 70005}
)

type FiveClan struct {
	AreaID     int       `db:"id"`
	ClanID     int       `db:"clanid"`
	ExpiresAt  null.Time `db:"expires_at" json:"expires_at"`
	TempleName string    `db:"name" json:"name"`
}

func (b *FiveClan) Update() error {
	_, err := db.Update(b)
	return err
}

func getFiveAreas() error {
	var areas []*FiveClan
	query := `select * from data.fiveclan_war`

	if _, err := db.Select(&areas, query); err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		return fmt.Errorf("getFiveAreas: %s", err.Error())
	}

	for _, cr := range areas {
		FiveClans[cr.AreaID] = cr
	}

	return nil
}

func LogoutFiveBuffDelete(char *Character) {
	for _, fivebuff := range FiveclanBuffs {
		buff, err := FindBuffByID(fivebuff, char.ID)
		if err != nil {
			continue
		}
		if buff == nil {
			continue
		}

		if buff.ID == 70004 {
			char.ExpMultiplier -= 0.06
			char.DropMultiplier -= 0.02
		}
		if buff.ID == 70003 {
			char.ExpMultiplier -= 0.06
			char.DropMultiplier -= 0.02
		}
		if buff.ID == 70002 {
			char.DropMultiplier -= 0.02
			char.ExpMultiplier -= 0.06
		}
		if buff.ID == 70001 {
			char.ExpMultiplier -= 0.06
			char.DropMultiplier -= 0.02
		}
		if buff.ID == 70005 {
			char.ExpMultiplier -= 0.06
			char.DropMultiplier -= 0.02
		}
		if char.ExpMultiplier < 1 {
			char.ExpMultiplier = 1
		}
		if char.DropMultiplier < 1 {
			char.DropMultiplier = 1
		}
		buff.Delete()
	}
}

func AddFiveBuffWhenLogin(char *Character) error {
	if char.GuildID > 0 {
		guild, err := FindGuildByID(char.GuildID)
		if err != nil {
			return err
		}
		for _, clans := range FiveClans {
			if clans.ClanID == guild.ID {
				buffID := FiveclanBuffs[clans.AreaID-1]
				currentTime := time.Now()
				diff := clans.ExpiresAt.Time.Sub(currentTime)
				if diff < 0 {
					clans.ClanID = 0
					clans.Update()
					continue
				}

				if buffID == 70001 {
					haveBuff, _ := FindBuffByID(70001, char.ID)
					if haveBuff == nil {
						infection := BuffInfections[70001]
						buff := &Buff{ID: int(70001), CharacterID: char.ID, Name: infection.Name, EXPMultiplier: 6, DropMultiplier: 2, StartedAt: char.Epoch, Duration: int64(diff.Seconds()), CanExpire: true}
						err = buff.Create()
						if err != nil {
							continue
						}
					} else {
						haveBuff.Duration = int64(diff.Seconds())
						haveBuff.Update()
					}

					char.ExpMultiplier += 0.06 // char.ExpMultiplier += 0.2
					char.DropMultiplier += 0.02
				}

				if buffID == 70002 {
					haveBuff, _ := FindBuffByID(70002, char.ID)
					if haveBuff == nil {
						infection := BuffInfections[70002]
						buff := &Buff{ID: int(70002), CharacterID: char.ID, Name: infection.Name, EXPMultiplier: 6, DropMultiplier: 2, StartedAt: char.Epoch, Duration: int64(diff.Seconds()), CanExpire: true}
						err = buff.Create()
						if err != nil {
							continue
						}
					} else {
						haveBuff.Duration = int64(diff.Seconds())
						haveBuff.Update()
					}

					char.ExpMultiplier += 0.06
					char.DropMultiplier += 0.02 // char.DropMultiplier += 0.2
				}

				if buffID == 70003 {
					haveBuff, _ := FindBuffByID(70003, char.ID)
					if haveBuff == nil {
						infection := BuffInfections[70003]
						buff := &Buff{ID: int(70003), CharacterID: char.ID, Name: infection.Name, EXPMultiplier: 6, DropMultiplier: 2, StartedAt: char.Epoch, Duration: int64(diff.Seconds()), CanExpire: true}
						err = buff.Create()
						if err != nil {
							fmt.Println(err)
							continue
						}
					} else {
						haveBuff.Duration = int64(diff.Seconds())
						haveBuff.Update()
					}

					char.ExpMultiplier += 0.06
					char.DropMultiplier += 0.02
				}

				if buffID == 70004 {
					haveBuff, _ := FindBuffByID(70004, char.ID)
					if haveBuff == nil {
						infection := BuffInfections[70004]
						buff := &Buff{ID: int(70004), CharacterID: char.ID, Name: infection.Name, EXPMultiplier: 6, DropMultiplier: 2, StartedAt: char.Epoch, Duration: int64(diff.Seconds()), CanExpire: true}
						err = buff.Create()
						if err != nil {
							continue
						}
					} else {
						haveBuff.Duration = int64(diff.Seconds())
						haveBuff.Update()
					}

					char.ExpMultiplier += 0.06
					char.DropMultiplier += 0.02
				}

				if buffID == 70005 {
					haveBuff, _ := FindBuffByID(70005, char.ID)
					if haveBuff == nil {
						infection := BuffInfections[70005]
						buff := &Buff{ID: int(70005), CharacterID: char.ID, Name: infection.Name, EXPMultiplier: 6, DropMultiplier: 2, StartedAt: char.Epoch, Duration: int64(diff.Seconds()), CanExpire: true}
						err = buff.Create()
						if err != nil {
							continue
						}
					} else {
						haveBuff.Duration = int64(diff.Seconds())
						haveBuff.Update()
					}

					char.ExpMultiplier += 0.06
					char.DropMultiplier += 0.02
				}

				char.Update()
			}
		}
	}
	return nil
}
