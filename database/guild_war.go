package database

import (
	"database/sql"
	"fmt"
	"time"

	null "gopkg.in/guregu/null.v3"
)

var (
	GuildVarActive = false
	GuildWarAreas  = make(map[int]*GuildWar)
	GuildWarMobs   = []int{18600047, 18600048, 18600049, 18600050, 18600051}
	GuildWarBuffs  = []int{70009, 70010, 70011, 70012, 70013}
)

type GuildWar struct {
	AreaID     int       `db:"id"`
	ClanID     int       `db:"clanid"`
	ExpiresAt  null.Time `db:"expires_at" json:"expires_at"`
	TempleName string    `db:"name" json:"name"`
}

func (b *GuildWar) Update() error {
	_, err := db.Update(b)
	return err
}

func getGuildAreas() error {
	var areas []*GuildWar
	query := `select * from data.guild_war`

	if _, err := db.Select(&areas, query); err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		return fmt.Errorf("getGuildAreas: %s", err.Error())
	}

	for _, cr := range areas {
		GuildWarAreas[cr.AreaID] = cr
	}

	return nil
}

func LogoutGuildWarBuffDelete(char *Character) {
	for _, fivebuff := range GuildWarBuffs {
		buff, err := FindBuffByID(fivebuff, char.ID)
		if err != nil {
			continue
		}
		if buff == nil {
			continue
		}

		if buff.ID == 70009 {
			char.ExpMultiplier -= 0.15
		}
		if buff.ID == 70010 {
			char.DropMultiplier -= 0.05
		}
		if buff.ID == 70011 {
			char.DropMultiplier -= 0.03
			char.ExpMultiplier -= 0.15
		}
		if buff.ID == 70012 {
			char.DropMultiplier -= 0.03
			char.ExpMultiplier -= 0.15
		}
		if buff.ID == 70013 {
			char.DropMultiplier -= 0.03
			char.ExpMultiplier -= 0.15
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

func AddGuildWarBuffWhenLogin(char *Character) error {
	if char.GuildID > 0 {
		guild, err := FindGuildByID(char.GuildID)
		if err != nil {
			return err
		}
		for _, clans := range GuildWarAreas {
			if clans.ClanID == guild.ID {
				buffID := GuildWarBuffs[clans.AreaID-1]
				currentTime := time.Now()
				diff := clans.ExpiresAt.Time.Sub(currentTime)
				if diff < 0 {
					clans.ClanID = 0
					clans.Update()
					continue
				}

				if buffID == 70009 {
					infection := BuffInfections[70009]
					buff := &Buff{ID: int(70009), CharacterID: char.ID, Name: infection.Name, EXPMultiplier: 15, StartedAt: char.Epoch, Duration: int64(diff.Seconds()), CanExpire: true}
					err = buff.Create()
					if err != nil {
						continue
					}
					char.ExpMultiplier += 0.15 // char.ExpMultiplier += 0.2
				}

				if buffID == 70010 {
					infection := BuffInfections[70010]
					buff := &Buff{ID: int(70010), CharacterID: char.ID, Name: infection.Name, DropMultiplier: 5, StartedAt: char.Epoch, Duration: int64(diff.Seconds()), CanExpire: true}
					err = buff.Create()
					if err != nil {
						continue
					}

					char.DropMultiplier += 0.05 // char.DropMultiplier += 0.2
				}

				if buffID == 70011 {
					infection := BuffInfections[70011]
					buff := &Buff{ID: int(70011), CharacterID: char.ID, Name: infection.Name, EXPMultiplier: 10, DropMultiplier: 3, StartedAt: char.Epoch, Duration: int64(diff.Seconds()), CanExpire: true}
					err = buff.Create()
					if err != nil {
						fmt.Println(err)
						continue
					}

					char.ExpMultiplier += 0.15
					char.DropMultiplier += 0.03
				}

				if buffID == 70012 {
					infection := BuffInfections[70012]
					buff := &Buff{ID: int(70012), CharacterID: char.ID, Name: infection.Name, EXPMultiplier: 15, DropMultiplier: 3, StartedAt: char.Epoch, Duration: int64(diff.Seconds()), CanExpire: true}
					err = buff.Create()
					if err != nil {
						continue
					}

					char.ExpMultiplier += 0.15
					char.DropMultiplier += 0.03
				}

				if buffID == 70013 {
					infection := BuffInfections[70013]
					buff := &Buff{ID: int(70013), CharacterID: char.ID, Name: infection.Name, EXPMultiplier: 15, DropMultiplier: 3, StartedAt: char.Epoch, Duration: int64(diff.Seconds()), CanExpire: true}
					err = buff.Create()
					if err != nil {
						continue
					}
					char.ExpMultiplier += 0.15
					char.DropMultiplier += 0.03
				}

				char.Update()
			}
		}
	}
	return nil
}
