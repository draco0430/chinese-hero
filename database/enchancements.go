package database

import (
	"database/sql"
	"fmt"
)

var (
	Enhancements = make(map[int]*Enhancement)
)

type Enhancement struct {
	ID        int   `db:"id"`
	BookID    int   `db:"bookid"`
	Material1 int64 `db:"material1"`
	Material2 int64 `db:"material2"`
	Material3 int64 `db:"material3"`
	Amount1   int16 `db:"amount1"`
	Amount2   int16 `db:"amount2"`
	Amount3   int16 `db:"amount3"`
	Rate      int32 `db:"rate"`
	Result    int   `db:"result"`
}

func getEnhancements() error {
	var enchant []*Enhancement
	query := `select * from data.enchant`

	if _, err := db.Select(&enchant, query); err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		return fmt.Errorf("getEnhancements: %s", err.Error())
	}

	for _, e := range enchant {
		Enhancements[e.BookID] = e
	}

	return nil
}
