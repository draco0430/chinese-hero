package database

import (
	"database/sql"
	"fmt"
	"log"
	"strconv"
	"strings"

	gorp "gopkg.in/gorp.v1"
)

var (
	Relics = make(map[int]*Relic)
)

type Relic struct {
	ID            int    `db:"id"`
	Count         int    `db:"count"`
	Limit         int    `db:"limit"`
	Tradable      bool   `db:"tradable"`
	RequiredItems string `db:"required_items"`
}

func (e *Relic) Create() error {
	return db.Insert(e)
}

func (e *Relic) CreateWithTransaction(tr *gorp.Transaction) error {
	return tr.Insert(e)
}

func (e *Relic) Delete() error {
	_, err := db.Delete(e)
	return err
}

func (e *Relic) Update() error {
	_, err := db.Update(e)
	if err != nil {
		log.Println(err)
	}
	return err
}

func (e *Relic) GetRequiredItems() []int {
	probs := strings.Trim(e.RequiredItems, "{}")
	sProbs := strings.Split(probs, ",")

	var arr []int
	for _, sProb := range sProbs {
		d, _ := strconv.Atoi(sProb)
		arr = append(arr, d)
	}
	return arr
}

func getRelics() error {
	var relics []*Relic
	query := `select * from hops.relics`

	if _, err := db.Select(&relics, query); err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		return fmt.Errorf("getRelics: %s", err.Error())
	}

	for _, r := range relics {
		Relics[r.ID] = r
	}

	return nil
}
