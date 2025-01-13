package logging

import (
	"encoding/json"
	"fmt"
	glog "log"
	"time"

	"hero-server/redis"
	"hero-server/utils"

	"github.com/google/uuid"
	"github.com/tidwall/gjson"
)

type Action byte

const (
	ACTION_LOGIN Action = iota
	ACTION_SELECT_SERVER
	ACTION_CREATE_CHARACTER
	ACTION_DELETE_CHARACTER
	ACTION_START_GAME
	ACTION_UPGRADE_ITEM
	ACTION_PRODUCTION
	ACTION_ADVANCED_FUSION
	ACTION_DISMANTLE
	ACTION_EXTRACTION
	ACTION_TRADE
	ACTION_BUY_SALE_ITEM
	ACTION_BUY_CONS_ITEM
	ACTION_CLAIM_CONS_ITEM
	ACTION_CREATE_ITEM
	ACTION_CREATE_GOLD
	ACTION_UPGRADE_GM_ITEM
	ACTION_ADD_NCASH
	ACTION_ADD_EXP
	ACTION_EXP_RATE
	ACTION_DROP_RATE
)

var (
	Logger = &LoggerController{}
	logs   = utils.NewMap()
)

type (
	Log struct {
		Action        Action    `json:"action,omitempty"`
		CharacterID   string    `json:"character_id,omitempty"`
		Date          time.Time `json:"date,omitempty"`
		ID            string    `json:"id,omitempty"`
		Message       string    `json:"message,omitempty"`
		UserID        string    `json:"user_id,omitempty"`
		CharacterName string    `json:"character_name,omitempty"`
	}

	LoggerController struct{}
)

func (l *LoggerController) Log(action Action, characterID int, message, userID, chrName string) {

	log := &Log{
		Action:        action,
		Date:          time.Now(),
		ID:            uuid.New().String(),
		Message:       message,
		UserID:        userID,
		CharacterName: chrName,
	}
	if characterID > 0 {
		log.CharacterID = fmt.Sprintf("%d", characterID)
	}

	data, err := json.Marshal(log)
	if err != nil {
		glog.Println("Log error:", err)
		return
	}

	logs.Add(log.ID, data)
}

func (l *LoggerController) StartLogging() {

	logData := logs.PopValues()
	for _, data := range logData {
		log := data.([]byte)
		id := gjson.Get(string(log), "id").String()

		err := redis.Set(id, log)
		if err != nil {
			glog.Println(err)
		}
	}

	time.AfterFunc(time.Second*10, func() {
		l.StartLogging()
	})
}
