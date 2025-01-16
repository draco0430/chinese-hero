package database

import (
	"log"
	"sync"

	"hero-server/utils"

	"github.com/xuri/excelize/v2"
	gorp "gopkg.in/gorp.v1"
)

var (
	Items         = make(map[int64]*Item)
	ItemsMutex    = sync.RWMutex{}
	STRRates      = []int{400, 350, 300, 250, 200, 175, 150, 125, 100, 75, 50, 40, 30, 20, 15}
	socketOrePlus = map[int64]byte{17402319: 1, 17402320: 2, 17402321: 3, 17402322: 4, 17402323: 5}
	haxBoxes      = []int64{92000002, 92000003, 92000004, 92000005, 92000006, 92000007, 92000008, 92000009, 92000010}
)

func GetItemInfo(id int64) (*Item, bool) {
	ItemsMutex.RLock()
	defer ItemsMutex.RUnlock()
	item, ok := Items[id]
	return item, ok
}
func SetItem(item *Item) {
	ItemsMutex.Lock()
	defer ItemsMutex.Unlock()
	Items[item.ID] = item
}

const (
	WEAPON_TYPE = iota
	ARMOR_TYPE
	HT_ARMOR_TYPE
	ACC_TYPE
	PENDENT_TYPE
	QUEST_TYPE
	PET_ITEM_TYPE
	SKILL_BOOK_TYPE
	PASSIVE_SKILL_BOOK_TYPE
	POTION_TYPE
	PET_TYPE
	PET_POTION_TYPE
	CHARM_OF_RETURN_TYPE
	FORTUNE_BOX_TYPE
	MARBLE_TYPE
	WRAPPER_BOX_TYPE
	ESOTERIC_POTION_TYPE
	NPC_SUMMONER_TYPE
	FIRE_SPIRIT
	WATER_SPIRIT
	HOLY_WATER_TYPE
	FILLER_POTION_TYPE
	SCALE_TYPE
	BAG_EXPANSION_TYPE
	MOVEMENT_SCROLL_TYPE
	SOCKET_TYPE
	INGREDIENTS_TYPE
	DEAD_SPIRIT_INCENSE_TYPE
	AFFLICTION_TYPE
	RESET_ART_TYPE
	RESET_ARTS_TYPE
	FORM_TYPE
	UNKNOWN_TYPE
	TRANSFORMATION_TYPE
)

type Item struct {
	ID              int64   `db:"id"`
	Name            string  `db:"name"`
	UIF             string  `db:"uif"`
	Type            int16   `db:"type"`
	ItemPair        int64   `db:"itempair"`
	HtType          int16   `db:"ht_type"`
	TimerType       int16   `db:"timer_type"`
	Timer           int     `db:"timer"`
	MinUpgradeLevel int16   `db:"min_upgrade_level"`
	BuyPrice        int64   `db:"buy_price"`
	SellPrice       int64   `db:"sell_price"`
	Slot            int     `db:"slot"`
	CharacterType   int     `db:"character_type"`
	MinLevel        int     `db:"min_level"`
	MaxLevel        int     `db:"max_level"`
	BaseDef1        int     `db:"base_def1"`
	BaseDef2        int     `db:"base_def2"`
	BaseDef3        int     `db:"base_def3"`
	BaseMinAtk      int     `db:"base_min_atk"`
	BaseMaxAtk      int     `db:"base_max_atk"`
	STR             int     `db:"str"`
	DEX             int     `db:"dex"`
	INT             int     `db:"int"`
	Wind            int     `db:"wind"`
	Water           int     `db:"water"`
	Fire            int     `db:"fire"`
	MaxHp           int     `db:"max_hp"`
	MaxChi          int     `db:"max_chi"`
	RunningSpeed    float64 `db:"running_speed"`
	MinAtk          int     `db:"min_atk"`
	MaxAtk          int     `db:"max_atk"`
	AtkRate         int     `db:"atk_rate"`
	MinArtsAtk      int     `db:"min_arts_atk"`
	MaxArtsAtk      int     `db:"max_arts_atk"`
	ArtsAtkRate     int     `db:"arts_atk_rate"`
	Def             int     `db:"def"`
	DefRate         int     `db:"def_rate"`
	ArtsDef         int     `db:"arts_def"`
	ArtsDefRate     int     `db:"arts_def_rate"`
	Accuracy        int     `db:"accuracy"`
	Dodge           int     `db:"dodge"`
	HpRecovery      int     `db:"hp_recovery"`
	ChiRecovery     int     `db:"chi_recovery"`
	ExpRate         float64 `db:"exp_rate"`
	DropRate        float64 `db:"drop_rate"`
	Tradable        int     `db:"tradable"`
	HolyWaterUpg1   int     `db:"holy_water_upg1"`
	HolyWaterUpg2   int     `db:"holy_water_upg2"`
	HolyWaterUpg3   int     `db:"holy_water_upg3"`
	HolyWaterRate1  int     `db:"holy_water_rate1"`
	HolyWaterRate2  int     `db:"holy_water_rate2"`
	HolyWaterRate3  int     `db:"holy_water_rate3"`
	PoisonATK       int     `db:"poison_attack"`
	PoisonDEF       int     `db:"poison_defense"`
	ParaATK         int     `db:"para_attack"`
	ParaDEF         int     `db:"para_defense"`
	ConfusionATK    int     `db:"confusion_attack"`
	ConfusionDEF    int     `db:"confusion_defense"`
	PoisonTime      int     `db:"poison_time"`
	ParaTime        int     `db:"para_time"`
	ConfusionTime   int     `db:"confusion_time"`

	NPCID int `db:"npc_id"`
}

func (item *Item) Create() error {
	return db.Insert(item)
}

func (item *Item) CreateWithTransaction(tr *gorp.Transaction) error {
	return tr.Insert(item)
}

func (item *Item) Delete() error {
	_, err := db.Delete(item)
	return err
}

func (item *Item) Update() error {
	_, err := db.Update(item)
	return err
}

func (item *Item) GetType() int {
	if item.Type == 51 {
		return FIRE_SPIRIT
	} else if item.Type == 52 {
		return WATER_SPIRIT
	} else if item.Type == 59 {
		return BAG_EXPANSION_TYPE
	} else if item.Type == 64 {
		return MARBLE_TYPE
	} else if (item.Type >= 70 && item.Type <= 71) || (item.Type >= 99 && item.Type <= 108) {
		return WEAPON_TYPE
	} else if item.Type == 80 {
		return SOCKET_TYPE
	} else if item.Type == 81 {
		return HOLY_WATER_TYPE
	} else if item.Type == 110 {
		return AFFLICTION_TYPE
	} else if item.Type == 111 {
		return RESET_ART_TYPE
	} else if item.Type == 112 {
		return RESET_ARTS_TYPE
	} else if item.Type == 113 {
		return TRANSFORMATION_TYPE
	} else if item.Type == 115 {
		return INGREDIENTS_TYPE
	} else if item.Type >= 121 && item.Type <= 124 && item.HtType == 0 {
		return ARMOR_TYPE
	} else if ((item.Type >= 121 && item.Type <= 124) || item.Type == 175) && item.HtType > 0 {
		return HT_ARMOR_TYPE
	} else if item.Type >= 131 && item.Type <= 134 {
		return ACC_TYPE
	} else if item.Type >= 135 && item.Type <= 137 {
		return PET_ITEM_TYPE
	} else if item.Type == 147 {
		return FILLER_POTION_TYPE
	} else if item.Type == 151 {
		return POTION_TYPE
	} else if item.Type == 152 {
		return CHARM_OF_RETURN_TYPE
	} else if item.Type == 153 {
		return MOVEMENT_SCROLL_TYPE
	} else if item.Type == 161 {
		return SKILL_BOOK_TYPE
	} else if item.Type == 162 {
		return PASSIVE_SKILL_BOOK_TYPE
	} else if item.Type == 166 {
		return SCALE_TYPE
	} else if item.Type == 168 || item.Type == 213 {
		return WRAPPER_BOX_TYPE
	} else if item.Type == 174 {
		return FORM_TYPE
	} else if item.Type == 191 {
		return PENDENT_TYPE
	} else if item.Type == 202 {
		return QUEST_TYPE
	} else if item.Type == 203 {
		return FORTUNE_BOX_TYPE
	} else if item.Type == 221 {
		return PET_TYPE
	} else if item.Type == 222 {
		return PET_POTION_TYPE
	} else if item.Type == 223 {
		return DEAD_SPIRIT_INCENSE_TYPE
	} else if item.Type == 233 {
		return NPC_SUMMONER_TYPE
	} else if item.Type == 150 {
		return ESOTERIC_POTION_TYPE
	}
	return UNKNOWN_TYPE
}

func getAllItems() error {
	log.Print("Reading Items table...")

	f, err := excelize.OpenFile("data/tb_ItemTable_Normal.xlsx")
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
		item := &Item{
			ID:              int64(utils.StringToInt(row[1])),
			Name:            row[2],
			UIF:             row[6],
			ItemPair:        int64(utils.StringToInt(row[18])),
			Type:            int16(utils.StringToInt(row[21])),
			HtType:          int16(utils.StringToInt(row[22])),
			TimerType:       int16(utils.StringToInt(row[26])),
			Timer:           utils.StringToInt(row[27]),
			MinUpgradeLevel: int16(utils.StringToInt(row[29])),
			BuyPrice:        int64(utils.StringToInt(row[38])),
			SellPrice:       int64(utils.StringToInt(row[39])),
			Slot:            utils.StringToInt(row[40]),
			CharacterType:   utils.StringToInt(row[42]),
			MinLevel:        utils.StringToInt(row[46]),
			MaxLevel:        utils.StringToInt(row[47]),
			BaseDef1:        utils.StringToInt(row[55]),
			BaseDef2:        utils.StringToInt(row[56]),
			BaseDef3:        utils.StringToInt(row[57]),
			BaseMinAtk:      utils.StringToInt(row[58]),
			BaseMaxAtk:      utils.StringToInt(row[59]),
			STR:             utils.StringToInt(row[63]),
			DEX:             utils.StringToInt(row[64]),
			INT:             utils.StringToInt(row[65]),
			Wind:            utils.StringToInt(row[66]),
			Water:           utils.StringToInt(row[67]),
			Fire:            utils.StringToInt(row[68]),

			PoisonATK:     utils.StringToInt(row[69]),
			PoisonDEF:     utils.StringToInt(row[70]),
			PoisonTime:    utils.StringToInt(row[72]),
			ConfusionATK:  utils.StringToInt(row[73]),
			ConfusionDEF:  utils.StringToInt(row[74]),
			ConfusionTime: utils.StringToInt(row[76]),
			ParaATK:       utils.StringToInt(row[77]),
			ParaDEF:       utils.StringToInt(row[78]),
			ParaTime:      utils.StringToInt(row[80]),
			MaxHp:         utils.StringToInt(row[83]),

			MaxChi:         utils.StringToInt(row[85]),
			RunningSpeed:   utils.StringToFloat64(row[87]),
			MinAtk:         utils.StringToInt(row[90]),
			MaxAtk:         utils.StringToInt(row[91]),
			AtkRate:        utils.StringToInt(row[92]),
			MinArtsAtk:     utils.StringToInt(row[93]),
			MaxArtsAtk:     utils.StringToInt(row[94]),
			ArtsAtkRate:    utils.StringToInt(row[95]),
			Def:            utils.StringToInt(row[96]),
			DefRate:        utils.StringToInt(row[97]),
			ArtsDef:        utils.StringToInt(row[99]),
			ArtsDefRate:    utils.StringToInt(row[100]),
			Accuracy:       utils.StringToInt(row[103]),
			Dodge:          utils.StringToInt(row[104]),
			HpRecovery:     utils.StringToInt(row[105]),
			ChiRecovery:    utils.StringToInt(row[106]),
			ExpRate:        utils.StringToFloat64(row[113]),
			DropRate:       utils.StringToFloat64(row[114]),
			Tradable:       utils.StringToInt(row[119]),
			HolyWaterUpg1:  utils.StringToInt(row[125]),
			HolyWaterUpg2:  utils.StringToInt(row[126]),
			HolyWaterUpg3:  utils.StringToInt(row[127]),
			HolyWaterRate1: utils.StringToInt(row[128]),
			HolyWaterRate2: utils.StringToInt(row[129]),
			HolyWaterRate3: utils.StringToInt(row[130]),
			NPCID:          utils.StringToInt(row[29]),
		}
		SetItem(item)
	}
	return nil
}

// Determines if a weapon item can use an action with specified type
func (item *Item) CanUse(t byte) bool {
	if item.Type == int16(t) || t == 0 {
		return true
	} else if (item.Type == 70 || item.Type == 71) && (t == 70 || t == 71) {
		return true
	} else if (item.Type == 102 || item.Type == 103) && (t == 102 || t == 103) {
		return true
	}

	return false
}
