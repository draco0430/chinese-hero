package database

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"

	gorp "gopkg.in/gorp.v1"
)

var (
	NPCs   map[int]*NPC
	bosses = []int{9999991, 9999992, 9999993, 9999994, 9999995, 9999996, 9999997, 9999998, 41941, 41171, 41371, 41381, 41671, 41851, 41852, 41853, 42451, 42452, 42561, 42562} /*40951, 41171, 41371, 41381, 41671, 41851, 41852, 41853, 41941, 42451, 42452, 42561, 42562,
	420108, 430108*/
)

type NPC struct {
	ID           int    `db:"id"`
	Name         string `db:"name"`
	Level        int16  `db:"level"`
	Exp          int64  `db:"exp"`
	DivineExp    int64  `db:"divine_exp"`
	DarknessExp  int64  `db:"darkness_exp"`
	GoldDrop     int    `db:"gold_drop"`
	DEF          int    `db:"def"`
	MaxHp        int    `db:"max_hp"`
	MinATK       int    `db:"min_atk"`
	MaxATK       int    `db:"max_atk"`
	MinArtsATK   int    `db:"min_arts_atk"`
	MaxArtsATK   int    `db:"max_arts_atk"`
	ArtsDEF      int    `db:"arts_def"`
	DropID       int    `db:"drop_id"`
	SkillID      string `db:"skill_id"`
	WalkingSpeed int    `db:"walking_speed"`
	RunningSpeed int    `db:"running_speed"`
}

func (e *NPC) Create() error {
	return db.Insert(e)
}

func (e *NPC) CreateWithTransaction(tr *gorp.Transaction) error {
	return tr.Insert(e)
}

func (e *NPC) Update() error {
	_, err := db.Update(e)
	return err
}

func (e *NPC) Delete() error {
	_, err := db.Delete(e)
	return err
}

func (e *NPC) GetSkills() []int {
	probs := strings.Trim(e.SkillID, "{}")
	sProbs := strings.Split(probs, ",")

	var arr []int
	for _, sProb := range sProbs {
		d, _ := strconv.Atoi(sProb)
		arr = append(arr, d)
	}
	return arr
}

func GetAllNPCs() (map[int]*NPC, error) {

	var arr []*NPC
	query := `select * from "data".npc_table`

	if _, err := db.Select(&arr, query); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("GetAllNPCs: %s", err.Error())
	}

	npcMap := make(map[int]*NPC)
	for _, npc := range arr {
		npcMap[npc.ID] = npc
	}

	return npcMap, nil
}

func FindNPCByID(id int) (*NPC, error) {

	npc := &NPC{}
	query := `select * from "data".npc_table where "id" = $1`

	if err := db.SelectOne(&npc, query, id); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("FindNPCByID: %s", err.Error())
	}

	return npc, nil
}
