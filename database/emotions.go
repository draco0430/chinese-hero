package database

import (
	"database/sql"
	"fmt"
)

type Emotion struct {
	ID          int    `db:"id"`
	Cmd         string `db:"cmd"`
	Type        int    `db:"emotion_type"`
	AnimationID int    `db:"animation_id"`
}

var (
	Emotions = make(map[int]*Emotion)
)

/*

func GetEmotions() error {
	log.Print("Reading Emotions...")
	f, err := excelize.OpenFile("data/tb_EMotion.xlsx")
	if err != nil {
		return err
	}
	defer f.Close()

	// Get all the rows in the Sheet1.
	rows, err := f.GetRows("Sheet1")
	if err != nil {
		return err
	}
	for index, row := range rows {
		if index == 0 {
			continue
		}
		Emotions[utils.StringToInt(row[1])] = &Emotion{
			ID:          utils.StringToInt(row[1]),
			Cmd:         row[2],
			Type:        utils.StringToInt(row[5]),
			AnimationID: utils.StringToInt(row[6]),
		}
	}
	return nil
}
*/

func getAllEmotions() error {
	var emotions []*Emotion
	query := `select * from data.emotions`

	if _, err := db.Select(&emotions, query); err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		return fmt.Errorf("getAllEmotions: %s", err.Error())
	}

	for _, d := range emotions {
		Emotions[d.ID] = d
	}

	return nil
}

func GetAllEmotions() error {
	return getAllEmotions()
}
