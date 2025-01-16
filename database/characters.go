package database

import (
	"bytes"
	"database/sql"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"math"
	"math/rand"
	"regexp"
	dbg "runtime/debug"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"hero-server/logging"
	"hero-server/messaging"
	"hero-server/nats"
	"hero-server/utils"

	"github.com/thoas/go-funk"
	gorp "gopkg.in/gorp.v1"
	null "gopkg.in/guregu/null.v3"
)

const (
	MONK                  = 0x34
	MALE_BLADE            = 0x35
	FEMALE_BLADE          = 0x36
	AXE                   = 0x38
	FEMALE_ROD            = 0x39
	DUAL_BLADE            = 0x3B
	DIVINE_MONK           = 0x3E
	DIVINE_MALE_BLADE     = 0x3F
	DIVINE_FEMALE_BLADE   = 0x40
	DIVINE_AXE            = 0x42
	DIVINE_FEMALE_ROD     = 0x43
	DIVINE_DUAL_BLADE     = 0x45
	REBORN_SM             = 0x50
	DARKNESS_BEAST_KING   = 0x46
	DARKNESS_EMPRESS      = 0x47
	DARKNESS_MONK         = 0x48
	DARKNESS_MALE_BLADE   = 0x49
	DARKNESS_FEMALE_BLADE = 0x4A
	DARKNESS_AXE          = 0x4C
	DARKNESS_FEMALE_ROD   = 0x4D
	DARKNESS_DUAL_BLADE   = 0x4F
)

var (
	characters      = make(map[int]*Character)
	characterMutex  sync.RWMutex
	GenerateID      func(*Character) error
	GeneratePetID   func(*Character, *PetSlot)
	challengerGuild = &Guild{}
	enemyGuild      = &Guild{}

	DEAL_DAMAGE           = utils.Packet{0xAA, 0x55, 0x1C, 0x00, 0x16, 0xFF, 0xFF, 0xFF, 0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x55, 0xAA}
	PVP_DEAL_SKILL_DAMAGE = utils.Packet{0xAA, 0x55, 0x28, 0x00, 0x16, 0x42, 0x75, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xB8, 0x3B, 0x11, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x10, 0x27, 0x00, 0x00, 0x55, 0xAA}
	BAG_EXPANDED          = utils.Packet{0xAA, 0x55, 0x17, 0x00, 0xA3, 0x02, 0x01, 0x32, 0x30, 0x32, 0x30, 0x2D, 0x30, 0x33, 0x2D, 0x31, 0x37, 0x20, 0x31, 0x31, 0x3A, 0x32, 0x32, 0x3A, 0x30, 0x31, 0x00, 0x55, 0xAA}
	BANK_ITEMS            = utils.Packet{0xAA, 0x55, 0x00, 0x00, 0x57, 0x05, 0x01, 0x02, 0x55, 0xAA}
	CHARACTER_DIED        = utils.Packet{0xAA, 0x55, 0x02, 0x00, 0x12, 0x01, 0x55, 0xAA}

	CHARACTER_SPAWNED = utils.Packet{0xAA, 0x55, 0x00, 0x00, 0x21, 0x01, 0xD7, 0xEF, 0xE6, 0x00, 0x03, 0x01, 0x00, 0x00, 0x00, 0x00, 0xC9, 0x00, 0x00, 0x00,
		0x49, 0x2A, 0xFE, 0x00, 0x20, 0x1C, 0x00, 0x00, 0x02, 0xD2, 0x7E, 0x7F, 0xBF, 0xCD, 0x1A, 0x86, 0x3D, 0x33, 0x33, 0x6B, 0x41, 0xFF, 0xFF, 0x10, 0x27,
		0x00, 0x00, 0x02, 0x00, 0x00, 0x00, 0xC4, 0x0E, 0x00, 0x00, 0xC8, 0xBB, 0x30, 0x00, 0x00, 0x03, 0xF3, 0x03, 0x01, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00,
		0x00, 0x10, 0x27, 0x00, 0x00, 0x49, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x64, 0xFF, 0xFF, 0xFF, 0xFF, 0x00, 0x00, 0x00, 0x55, 0xAA}

	/*
		CHARACTER_SPAWNED = utils.Packet{0xAA, 0x55, 0x00, 0x00, 0x21, 0x01, 0xD7, 0xEF, 0xE6, 0x00, 0x03, 0x01, 0x00, 0x00, 0x00, 0x00, 0xC9, 0x00, 0x00, 0x00,
			0x49, 0x2A, 0xFE, 0x00, 0x20, 0x1C, 0x00, 0x00, 0x02, 0xD2, 0x7E, 0x7F, 0xBF, 0xCD, 0x1A, 0x86, 0x3D, 0x33, 0x33, 0x6B, 0x41, 0xFF, 0xFF, 0x10, 0x27,
			0x00, 0x00, 0x02, 0x00, 0x00, 0x00, 0xC4, 0x0E, 0x00, 0x00, 0xC8, 0xBB, 0x30, 0x00, 0x00, 0x03, 0xF3, 0x03, 0x01, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00,
			0x00, 0x10, 0x27, 0x00, 0x00, 0x49, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x64, 0xFF, 0xFF, 0xFF, 0xFF, 0x00, 0x00, 0x00, 0x55, 0xAA}
	*/

	EXP_SKILL_PT_CHANGED = utils.Packet{0xAA, 0x55, 0x0D, 0x00, 0x13, 0x55, 0xAA}

	HP_CHI = utils.Packet{0xAA, 0x55, 0x28, 0x00, 0x16, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x10, 0x27, 0x00, 0x00, 0x55, 0xAA}

	/*
		HP_CHI = utils.Packet{0xAA, 0x55, 0x28, 0x00, 0x16, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x10, 0x27, 0x00, 0x00, 0x55, 0xAA}
	*/

	RESPAWN_COUNTER = utils.Packet{0xAA, 0x55, 0x05, 0x00, 0x12, 0x02, 0x01, 0x00, 0x00, 0x55, 0xAA}
	SHOW_ITEMS      = utils.Packet{0xAA, 0x55, 0x00, 0x00, 0x59, 0x05, 0x0A, 0x00, 0x00, 0x00, 0x55, 0xAA}
	MEDITATION_MODE = utils.Packet{0xAA, 0x55, 0x05, 0x00, 0x82, 0x05, 0x00, 0x55, 0xAA}

	TELEPORT_PLAYER  = utils.Packet{0xAA, 0x55, 0x09, 0x00, 0x24, 0x55, 0xAA}
	ITEM_COUNT       = utils.Packet{0xAA, 0x55, 0x0C, 0x00, 0x59, 0x04, 0x0A, 0x00, 0x55, 0xAA}
	GREEN_ITEM_COUNT = utils.Packet{0xAA, 0x55, 0x0A, 0x00, 0x59, 0x19, 0x0A, 0x00, 0x55, 0xAA}
	ITEM_EXPIRED     = utils.Packet{0xAA, 0x55, 0x06, 0x00, 0x69, 0x03, 0x55, 0xAA}
	ITEM_ADDED       = utils.Packet{0xAA, 0x55, 0x1E, 0x00, 0x58, 0x01, 0x0A, 0x00, 0x00, 0xA1, 0x00, 0x00, 0x00, 0x00, 0x20, 0x1C, 0x00, 0x00, 0x55, 0xAA}
	//ITEM_ADDED  = utils.Packet{0xaa, 0x55, 0x2e, 0x00, 0x57, 0x0a, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0x83, 0x02, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x55, 0xaa}
	ITEM_LOOTED = utils.Packet{0xAA, 0x55, 0x33, 0x00, 0x59, 0x01, 0x0A, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x21, 0x11, 0x55, 0xAA}

	PTS_CHANGED = utils.Packet{0xAA, 0x55, 0x06, 0x00, 0xA2, 0x04, 0x55, 0xAA}
	GOLD_LOOTED = utils.Packet{0xAA, 0x55, 0x0D, 0x00, 0x59, 0x01, 0x0A, 0x00, 0x02, 0x55, 0xAA}
	GET_GOLD    = utils.Packet{0xAA, 0x55, 0x12, 0x00, 0x63, 0x01, 0x55, 0xAA}

	MAP_CHANGED = utils.Packet{0xAA, 0x55, 0x02, 0x00, 0x2B, 0x01, 0x55, 0xAA, 0xAA, 0x55, 0x0E, 0x00, 0x73, 0x00, 0x00, 0x00, 0x7A, 0x44, 0x55, 0xAA,
		0xAA, 0x55, 0x07, 0x00, 0x01, 0xB9, 0x0A, 0x00, 0x00, 0x01, 0x00, 0x55, 0xAA, 0xAA, 0x55, 0x09, 0x00, 0x24, 0x55, 0xAA,
		0xAA, 0x55, 0x03, 0x00, 0xA6, 0x00, 0x00, 0x55, 0xAA, 0xAA, 0x55, 0x02, 0x00, 0xAD, 0x01, 0x55, 0xAA}

	ITEM_REMOVED = utils.Packet{0xAA, 0x55, 0x0B, 0x00, 0x59, 0x02, 0x0A, 0x00, 0x01, 0x55, 0xAA}
	SELL_ITEM    = utils.Packet{0xAA, 0x55, 0x16, 0x00, 0x58, 0x02, 0x0A, 0x00, 0x20, 0x1C, 0x00, 0x00, 0x55, 0xAA}

	GET_STATS = utils.Packet{0xAA, 0x55, 0xDE, 0x00, 0x14, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x66, 0x66, 0xC6, 0x40, 0xF3,
		0x03, 0x00, 0x00, 0x00, 0x40, 0x05, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x0C, 0x00, 0x00, 0x00, 0x20, 0x00, 0x00, 0x00, 0x00, 0xFF, 0xFF, 0x00, 0x30, 0x30, 0x31, 0x2D, 0x30, 0x31, 0x2D, 0x30,
		0x31, 0x20, 0x30, 0x30, 0x3A, 0x30, 0x30, 0x3A, 0x30, 0x30, 0x20, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x03, 0x00, 0x00, 0x00, 0x00, 0x64, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x64, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x80, 0x3F, 0x10, 0x27, 0x80, 0x3F, 0x55, 0xAA}

	ITEM_REPLACEMENT   = utils.Packet{0xAA, 0x55, 0x0C, 0x00, 0x59, 0x03, 0x0A, 0x00, 0x55, 0xAA}
	ITEM_SWAP          = utils.Packet{0xAA, 0x55, 0x15, 0x00, 0x59, 0x07, 0x0A, 0x00, 0x00, 0x55, 0xAA}
	HT_UPG_FAILED      = utils.Packet{0xAA, 0x55, 0x31, 0x00, 0x54, 0x02, 0xA7, 0x0F, 0x01, 0x00, 0xA3, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x55, 0xAA}
	UPG_FAILED         = utils.Packet{0xAA, 0x55, 0x09, 0x00, 0x54, 0x02, 0xA2, 0x0F, 0x00, 0x55, 0xAA}
	PRODUCTION_SUCCESS = utils.Packet{0xAA, 0x55, 0x05, 0x00, 0x54, 0x04, 0x08, 0x10, 0x01, 0x55, 0xAA}
	PRODUCTION_FAILED  = utils.Packet{0xAA, 0x55, 0x05, 0x00, 0x54, 0x04, 0x09, 0x10, 0x00, 0x55, 0xAA}
	FUSION_SUCCESS     = utils.Packet{0xAA, 0x55, 0x05, 0x00, 0x54, 0x09, 0x08, 0x10, 0x01, 0x55, 0xAA}
	FUSION_FAILED      = utils.Packet{0xAA, 0x55, 0x05, 0x00, 0x54, 0x09, 0x09, 0x10, 0x00, 0x55, 0xAA}
	DISMANTLE_SUCCESS  = utils.Packet{0xAA, 0x55, 0x00, 0x00, 0x54, 0x05, 0x68, 0x10, 0x01, 0x00, 0x55, 0xAA}
	EXTRACTION_SUCCESS = utils.Packet{0xAA, 0x55, 0xB7, 0x00, 0x54, 0x06, 0xCC, 0x10, 0x01, 0x00, 0xA2, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x55, 0xAA}
	HOLYWATER_FAILED   = utils.Packet{0xAA, 0x55, 0x05, 0x00, 0x54, 0x10, 0x32, 0x11, 0x00, 0x55, 0xAA}
	HOLYWATER_SUCCESS  = utils.Packet{0xAA, 0x55, 0x05, 0x00, 0x54, 0x10, 0x31, 0x11, 0x01, 0x55, 0xAA}
	ITEM_REGISTERED    = utils.Packet{0xAA, 0x55, 0x43, 0x00, 0x3D, 0x01, 0x0A, 0x00, 0x00, 0x80, 0x1A, 0x06, 0x00, 0x00, 0x00, 0x00, 0x00, 0x0D, 0x00,
		0x00, 0x00, 0x63, 0x99, 0xEA, 0x00, 0x00, 0xA1, 0x00, 0x00, 0x00, 0x00, 0x55, 0xAA}
	CLAIM_MENU              = utils.Packet{0xAA, 0x55, 0x00, 0x00, 0x3D, 0x03, 0x0A, 0x00, 0x55, 0xAA}
	CONSIGMENT_ITEM_BOUGHT  = utils.Packet{0xAA, 0x55, 0x08, 0x00, 0x3D, 0x02, 0x0A, 0x00, 0x55, 0xAA}
	CONSIGMENT_ITEM_SOLD    = utils.Packet{0xAA, 0x55, 0x02, 0x00, 0x3F, 0x00, 0x55, 0xAA}
	CONSIGMENT_ITEM_CLAIMED = utils.Packet{0xAA, 0x55, 0x0A, 0x00, 0x3D, 0x04, 0x0A, 0x00, 0x01, 0x00, 0x55, 0xAA}
	SKILL_UPGRADED          = utils.Packet{0xAA, 0x55, 0x0B, 0x00, 0x81, 0x02, 0x0A, 0x00, 0x00, 0x00, 0x00, 0x55, 0xAA}
	SKILL_DOWNGRADED        = utils.Packet{0xAA, 0x55, 0x0E, 0x00, 0x81, 0x03, 0x0A, 0x00, 0x00, 0x00, 0x00, 0x55, 0xAA}
	SKILL_REMOVED           = utils.Packet{0xAA, 0x55, 0x09, 0x00, 0x81, 0x06, 0x0A, 0x00, 0x00, 0x55, 0xAA}
	PASSIVE_SKILL_UGRADED   = utils.Packet{0xAA, 0x55, 0x06, 0x00, 0x82, 0x02, 0x0A, 0x00, 0x00, 0x00, 0x55, 0xAA}
	PASSIVE_SKILL_REMOVED   = utils.Packet{0xAA, 0x55, 0x09, 0x00, 0x82, 0x04, 0x0A, 0x00, 0x00, 0x55, 0xAA}
	SKILL_CASTED            = utils.Packet{0xAA, 0x55, 0x1D, 0x00, 0x42, 0x0A, 0x00, 0x00, 0x00, 0x01, 0x01, 0x55, 0xAA}
	TRADE_CANCELLED         = utils.Packet{0xAA, 0x55, 0x06, 0x00, 0x53, 0x03, 0xD5, 0x07, 0x7E, 0x02, 0x55, 0xAA}
	SKILL_BOOK_EXISTS       = utils.Packet{0xAA, 0x55, 0x04, 0x00, 0x59, 0x04, 0xFA, 0x03, 0x55, 0xAA}
	INVALID_CHARACTER_TYPE  = utils.Packet{0xAA, 0x55, 0x04, 0x00, 0x59, 0x04, 0xF2, 0x03, 0x55, 0xAA}
	NO_SLOTS_FOR_SKILL_BOOK = utils.Packet{0xAA, 0x55, 0x04, 0x00, 0x59, 0x04, 0xF3, 0x03, 0x55, 0xAA}
	OPEN_SALE               = utils.Packet{0xAA, 0x55, 0x04, 0x00, 0x55, 0x01, 0x0A, 0x00, 0x55, 0xAA}
	GET_SALE_ITEMS          = utils.Packet{0xAA, 0x55, 0x00, 0x00, 0x55, 0x03, 0x0A, 0x00, 0x00, 0x55, 0xAA}
	CLOSE_SALE              = utils.Packet{0xAA, 0x55, 0x04, 0x00, 0x55, 0x02, 0x0A, 0x00, 0x55, 0xAA}
	BOUGHT_SALE_ITEM        = utils.Packet{0xAA, 0x55, 0x39, 0x00, 0x53, 0x10, 0x0A, 0x00, 0x01, 0x00, 0xA2, 0x00, 0x00, 0x00, 0x00, 0x55, 0xAA}
	SOLD_SALE_ITEM          = utils.Packet{0xAA, 0x55, 0x10, 0x00, 0x55, 0x07, 0x0A, 0x00, 0x55, 0xAA}
	BUFF_INFECTION          = utils.Packet{0xAA, 0x55, 0x0C, 0x00, 0x4D, 0x02, 0x0A, 0x01, 0x55, 0xAA}
	BUFF_EXPIRED            = utils.Packet{0xAA, 0x55, 0x06, 0x00, 0x4D, 0x03, 0x55, 0xAA}

	ENCHANT_SUCCESS = utils.Packet{0xAA, 0x55, 0x05, 0x00, 0x54, 0x07, 0x08, 0x10, 0x01, 0x55, 0xAA}
	ENCHANT_FAILED  = utils.Packet{0xAA, 0x55, 0x05, 0x00, 0x54, 0x07, 0x09, 0x10, 0x00, 0x55, 0xAA}
	ENCHANT_ERROR   = utils.Packet{0xAA, 0x55, 0x04, 0x00, 0x54, 0x07, 0x07, 0x10, 0x55, 0xAA}

	SPLIT_ITEM = utils.Packet{0xAA, 0x55, 0x5C, 0x00, 0x59, 0x09, 0x0A, 0x00, 0x00, 0xA1, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xA1, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x55, 0xAA}

	RELIC_DROP       = utils.Packet{0xAA, 0x55, 0x00, 0x00, 0x71, 0x10, 0x00, 0x55, 0xAA}
	PVP_FINISHED     = utils.Packet{0xAA, 0x55, 0x02, 0x00, 0x2A, 0x05, 0x55, 0xAA}
	FORM_ACTIVATED   = utils.Packet{0xAA, 0x55, 0x05, 0x00, 0x37, 0x55, 0xAA}
	FORM_DEACTIVATED = utils.Packet{0xAA, 0x55, 0x01, 0x00, 0x38, 0x55, 0xAA}
	CHANGE_RANK      = utils.Packet{0xAA, 0x55, 0x09, 0x00, 0x2F, 0xF1, 0x36, 0x55, 0xAA}
	PET_APPEARED     = utils.Packet{0xaa, 0x55, 0x53, 0x00, 0x31, 0x01, 0x00, 0x00, 0x00, 0x00, 0xff, 0xff, 0x00, 0xe8, 0x03, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x64, 0x00, 0x55, 0xaa}
	HONOR_RANKS      = []int{0, 1, 2, 14, 30, 50, 4}
)

type Target struct {
	Damage int  `db:"-" json:"damage"`
	AI     *AI  `db:"-" json:"ai"`
	Skill  bool `default:"false db:"-" json:"skill"`
	//Critical  bool `default:"false db:"-" json:"critical"`
	//Reflected bool `default:"false db:"-" json:"reflected"`
}

type AidSettings struct {
	PetFood1ItemID  int64 `db:"-" json:"petfood1"`
	PetFood1Percent uint  `db:"-" json:"petfood1percent"`
	PetChiItemID    int64 `db:"-" json:"petchi"`
	PetChiPercent   uint  `db:"-" json:"petchipercent"`
}

type PlayerTarget struct {
	Damage int        `db:"-" json:"damage"`
	Enemy  *Character `db:"-" json:"ai"`
	Skill  bool       `default:"false db:"-" json:"skill"`
	//Critical  bool       `default:"false db:"-" json:"critical"`
	//Reflected bool       `default:"false db:"-" json:"reflected"`
}

type Character struct {
	ID                       int       `db:"id" json:"id"`
	UserID                   string    `db:"user_id" json:"user_id"`
	Name                     string    `db:"name" json:"name"`
	Epoch                    int64     `db:"epoch" json:"epoch"`
	Type                     int       `db:"type" json:"type"`
	Faction                  int       `db:"faction" json:"faction"`
	Height                   int       `db:"height" json:"height"`
	Level                    int       `db:"level" json:"level"`
	Class                    int       `db:"class" json:"class"`
	IsOnline                 bool      `db:"is_online" json:"is_online"`
	IsActive                 bool      `db:"is_active" json:"is_active"`
	Gold                     uint64    `db:"gold" json:"gold"`
	Coordinate               string    `db:"coordinate" json:"coordinate"`
	Map                      int16     `db:"map" json:"map"`
	Exp                      int64     `db:"exp" json:"exp"`
	HTVisibility             int       `db:"ht_visibility" json:"ht_visibility"`
	WeaponSlot               int       `db:"weapon_slot" json:"weapon_slot"`
	RunningSpeed             float64   `db:"running_speed" json:"running_speed"`
	GuildID                  int       `db:"guild_id" json:"guild_id"`
	ClanGoldDonation         uint64    `db:"clan_gold_donation" json:"-"`
	ExpMultiplier            float64   `db:"exp_multiplier" json:"exp_multiplier"`
	DropMultiplier           float64   `db:"drop_multiplier" json:"drop_multiplier"`
	Slotbar                  []byte    `db:"slotbar" json:"slotbar"`
	CreatedAt                null.Time `db:"created_at" json:"created_at"`
	AdditionalExpMultiplier  float64   `db:"additional_exp_multiplier" json:"additional_exp_multiplier"`
	AdditionalDropMultiplier float64   `db:"additional_drop_multiplier" json:"additional_drop_multiplier"`
	AidMode                  bool      `db:"aid_mode" json:"aid_mode"`
	AidTime                  uint32    `db:"aid_time" json:"aid_time"`
	Injury                   float64   `db:"injury" json:"injury"`
	HonorRank                int64     `db:"rank" json:"rank"`
	YingYangTicketsLeft      bool      `db:"ying_yang_tickets" json:"ying_yang_tickets"`
	RebornLevel              int       `db:"reborn_level" json:"reborn_level"`

	Poisoned  bool `db:"-"`
	Paralised bool `db:"-"`
	Confused  bool `db:"-"`

	AddingExp              sync.Mutex `db:"-" json:"-"`
	AddingGold             sync.Mutex `db:"-" json:"-"`
	Looting                sync.Mutex `db:"-" json:"-"`
	AdditionalRunningSpeed float64    `db:"-" json:"-"`
	InvMutex               sync.Mutex `db:"-"`
	Socket                 *Socket    `db:"-" json:"-"`
	ExploreWorld           func()     `db:"-" json:"-"`
	HasLot                 bool       `db:"-" json:"-"`
	LastRoar               time.Time  `db:"-" json:"-"`
	Meditating             bool       `db:"-"`
	MovementToken          int64      `db:"-" json:"-"`
	//MovementTokenMutex     sync.Mutex `db:"-" json:"-"`
	PseudoID uint16 `db:"-" json:"pseudo_id"`
	PTS      int    `db:"-" json:"pts"`
	OnSight  struct {
		Drops       map[int]interface{} `db:"-" json:"drops"`
		DropsMutex  sync.RWMutex
		Mobs        map[int]interface{} `db:"-" json:"mobs"`
		MobMutex    sync.RWMutex        `db:"-"`
		NPCs        map[int]interface{} `db:"-" json:"npcs"`
		NpcMutex    sync.RWMutex        `db:"-"`
		Pets        map[int]interface{} `db:"-" json:"pets"`
		PetsMutex   sync.RWMutex        `db:"-"`
		Players     map[int]interface{} `db:"-" json:"players"`
		PlayerMutex sync.RWMutex        `db:"-"`
	} `db:"-" json:"on_sight"`
	PartyID           string          `db:"-"`
	Selection         int             `db:"-" json:"selection"`
	Targets           []*Target       `db:"-" json:"target"`
	TamingAI          *AI             `db:"-" json:"-"`
	PlayerTargets     []*PlayerTarget `db:"-" json:"player_targets"`
	TradeID           string          `db:"-" json:"trade_id"`
	Invisible         bool            `db:"-" json:"-"`
	DetectionMode     bool            `db:"-" json:"-"`
	VisitedSaleID     uint16          `db:"-" json:"-"`
	DuelID            int             `db:"-" json:"-"`
	DuelStarted       bool            `db:"-" json:"-"`
	Respawning        bool            `db:"-" json:"-"`
	SkillHistory      utils.SMap      `db:"-" json:"-"`
	Morphed           bool            `db:"-" json:"-"`
	MorphedNPCID      int             `db:"-" json:"-"`
	IsMounting        bool            `db:"-" json:"-"`
	PlayerAidSettings *AidSettings    `db:"-" json:"-"`

	IsAcceptedWar   bool    `db:"-" json:"-"`
	IsinWar         bool    `db:"-" json:"-"`
	IsinLastMan     bool    `db:"-" json:"-"`
	WarKillCount    int     `db:"-" json:"-"`
	InjuryCount     float64 `db:"-"`
	WarContribution int     `db:"-" json:"-"`

	UsedPotion bool `db:"-" json:"-"`
	UsedConsig bool `db:"-" json:"-"`
	IsAttacked bool `db:"-" json:"-"`
	//UsedConsumable   bool `db:"-" json:"-"`
	Loot      bool `db:"-"`
	CanSwap   bool `db:"-"`
	PartyMode int  `db:"-" json:"-"`
	//Floating  bool `db:"-" json:"-"`
	//DealPlayerAttack bool `db:"-" json:"-"`
	IsDungeon    bool  `db:"-"`
	DungeonLevel int16 `db:"-"`
	//UpdateMutex  sync.Mutex `db:"-" json:"-"`
	//UpdateChan chan struct{} `db:"-" json:"-"`
	SaleActive      bool  `db:"-"`
	SaleActiveEpoch int64 `db:"-"`

	HandlerCB    func() `db:"-"`
	PetHandlerCB func() `db:"-"`

	inventory []*InventorySlot `db:"-" json:"-"`

	UsedConsumables struct {
		Items     map[int64]bool `db:"-" json:"-"`
		ItemMutex sync.RWMutex   `db:"-" json:"-"`
	} `db:"-" json:"-"`

	InventoryArrange     bool `db:"-" json:"-"`
	BankInventoryArrange bool `db:"-" json:"-"`
}

func FindCharactersInMap(mapid int16) map[int]*Character {

	characterMutex.RLock()
	allChars := funk.Values(characters).([]*Character)
	characterMutex.RUnlock()

	allChars = funk.Filter(allChars, func(c *Character) bool {
		if c.Socket == nil {
			return false
		}

		return c.Map == mapid && c.IsOnline
	}).([]*Character)

	candidates := make(map[int]*Character)
	for _, c := range allChars {
		candidates[c.ID] = c
	}

	return candidates
}

func (t *Character) PreInsert(s gorp.SqlExecutor) error {
	now := time.Now().UTC()
	t.CreatedAt = null.TimeFrom(now)
	return nil
}

func (t *Character) SetCoordinate(coordinate *utils.Location) {
	t.Coordinate = fmt.Sprintf("(%.1f,%.1f)", coordinate.X, coordinate.Y)
}

func (t *Character) FixDropAndExp() {
	t.DropMultiplier = 1
	t.ExpMultiplier = 1

	buff, err := FindBuffByID(19000018, t.ID) // check for fire spirit
	if err == nil && buff != nil {
		t.DropMultiplier = 1.05
		t.ExpMultiplier = 1.30
	}

	buff, err = FindBuffByID(19000019, t.ID) // check for water spirit
	if err == nil && buff != nil {
		t.DropMultiplier = 1.05
		t.ExpMultiplier = 1.60
	}

	buff, err = FindBuffByID(70001, t.ID) // Flame Wolf (Exp)
	if err == nil && buff != nil {
		t.ExpMultiplier += 0.1
		t.DropMultiplier += 0.03
	}

	buff, err = FindBuffByID(70002, t.ID) // Ocean Army  (Drop)
	if err == nil && buff != nil {
		t.DropMultiplier += 0.1
		t.ExpMultiplier += 0.03
	}

	buff, err = FindBuffByID(70003, t.ID) // Lightning Hill (Drop & Exp)
	if err == nil && buff != nil {
		t.ExpMultiplier += 0.1
		t.DropMultiplier += 0.03
	}

	buff, err = FindBuffByID(70004, t.ID) // Southern Wood (Drop & Exp)
	if err == nil && buff != nil {
		t.ExpMultiplier += 0.1
		t.DropMultiplier += 0.03
	}

	buff, err = FindBuffByID(70005, t.ID) // Western Land (Drop & Exp)
	if err == nil && buff != nil {
		t.ExpMultiplier += 0.1
		t.DropMultiplier += 0.03
	}

	buff, err = FindBuffByID(70006, t.ID) // Reborn St 1
	if err == nil && buff != nil {
		t.ExpMultiplier -= 0.1
		t.DropMultiplier += 0.05
	}

	buff, err = FindBuffByID(70007, t.ID) // Reborn St 2
	if err == nil && buff != nil {
		t.ExpMultiplier -= 0.2
		t.DropMultiplier += 0.1
	}

	buff, err = FindBuffByID(70008, t.ID) // Reborn St 3
	if err == nil && buff != nil {
		t.ExpMultiplier -= 0.3
		t.DropMultiplier += 0.15
	}

	buff, err = FindBuffByID(70009, t.ID) // Whispering Woods
	if err == nil && buff != nil {
		t.ExpMultiplier += 0.15
	}

	buff, err = FindBuffByID(70010, t.ID) // Tranquil Lake
	if err == nil && buff != nil {
		t.DropMultiplier += 0.05
	}

	buff, err = FindBuffByID(70011, t.ID) // Silent Moon
	if err == nil && buff != nil {
		t.ExpMultiplier += 0.15
		t.DropMultiplier += 0.03
	}

	buff, err = FindBuffByID(70012, t.ID) // Serene Sky
	if err == nil && buff != nil {
		t.ExpMultiplier += 0.15
		t.DropMultiplier += 0.03
	}

	buff, err = FindBuffByID(70013, t.ID) // Calm Seas
	if err == nil && buff != nil {
		t.ExpMultiplier += 0.15
		t.DropMultiplier += 0.03
	}

	buff, err = FindBuffByID(1337, t.ID) // Sara Exp
	if err == nil && buff != nil {
		t.ExpMultiplier += 0.20
		t.DropMultiplier += 0.05
	}

	/*buff, err = FindBuffByID(70015, t.ID) // Guild Tier 1
	if err == nil && buff != nil {
		t.ExpMultiplier += 0.03
		t.DropMultiplier += 0.01
	}

	buff, err = FindBuffByID(70016, t.ID) // Guild Tier 2
	if err == nil && buff != nil {
		t.ExpMultiplier += 0.06
		t.DropMultiplier += 0.02
	}

	buff, err = FindBuffByID(70017, t.ID) // Guild Tier 3
	if err == nil && buff != nil {
		t.ExpMultiplier += 0.09
		t.DropMultiplier += 0.03
	}

	buff, err = FindBuffByID(70018, t.ID) // Guild Tier 4
	if err == nil && buff != nil {
		t.ExpMultiplier += 0.12
		t.DropMultiplier += 0.04
	}

	buff, err = FindBuffByID(70019, t.ID) // Guild Tier 5
	if err == nil && buff != nil {
		t.ExpMultiplier += 0.15
		t.DropMultiplier += 0.05
	}

	buff, err = FindBuffByID(70020, t.ID) // Great War Won Buff
	if err == nil && buff != nil {
		t.ExpMultiplier += 0.30
	}

	buff, err = FindBuffByID(70021, t.ID) // Great War Loss Buff
	if err == nil && buff != nil {
		t.ExpMultiplier += 0.15
	}*/

	t.Update()
}

/*func (t *Character) RemoveGuildTierBuffs(buffID int) {
	tierBuffID := 0
	switch buffID {
	case 1:
		tierBuffID = 70015
	case 2:
		tierBuffID = 70016
	case 3:
		tierBuffID = 70017
	case 4:
		tierBuffID = 70018
	case 5:
		tierBuffID = 70019
	}

	buff, err := FindBuffByID(tierBuffID, t.ID)
	if err == nil && buff != nil {
		buff.Delete()
	}
}*/

func (t *Character) Create() error {
	return db.Insert(t)
}

func (t *Character) CreateWithTransaction(tr *gorp.Transaction) error {
	return tr.Insert(t)
}

func (t *Character) PreUpdate(s gorp.SqlExecutor) error {
	if int64(t.Gold) < 0 {
		t.Gold = 0
	}
	return nil
}

func (t *Character) Update() error {
	_, err := db.Update(t)
	if err != nil {
		log.Println(err)
	}
	return err
}

func (t *Character) Delete() error {
	characterMutex.Lock()
	defer characterMutex.Unlock()
	delete(characters, t.ID)
	_, err := db.Delete(t)
	return err
}

func (t *Character) InventorySlots() ([]*InventorySlot, error) {

	if len(t.inventory) > 0 {
		return t.inventory, nil
	}

	inventory := make([]*InventorySlot, 450)

	for i := range inventory {
		inventory[i] = NewSlot()
	}

	slots, err := FindInventorySlotsByCharacterID(t.ID)
	if err != nil {
		return nil, err
	}

	bankSlots, err := FindBankSlotsByUserID(t.UserID)
	if err != nil {
		return nil, err
	}

	for _, s := range slots {
		inventory[s.SlotID] = s
	}

	for _, s := range bankSlots {
		inventory[s.SlotID] = s
	}

	t.inventory = inventory
	return inventory, nil
}

func (t *Character) SetInventorySlots(slots []*InventorySlot) { // FIX HERE
	t.inventory = slots
}

func (t *Character) CopyInventorySlots() []*InventorySlot {
	slots := []*InventorySlot{}
	for _, s := range t.inventory {
		copySlot := *s
		slots = append(slots, &copySlot)
	}

	return slots
}

func RefreshAIDs() error {

	characterMutex.RLock()
	allChars := funk.Values(characters).([]*Character)
	characterMutex.RUnlock()
	for _, c := range allChars {
		if c.AidTime < 7200 {
			query := `update hops.characters SET aid_time = 7200 where id = $1;`
			_, err := db.Exec(query, c.ID)
			if err != nil {
				return err
			}
			c.Update()
		}
	}

	return nil
}

func RefreshYingYangKeys() error {
	query := `update hops.characters SET ying_yang_tickets = true;`
	_, err := db.Exec(query)
	if err != nil {
		return err
	}

	characterMutex.RLock()
	allChars := funk.Values(characters).([]*Character)
	characterMutex.RUnlock()
	for _, c := range allChars {
		c.YingYangTicketsLeft = true
		c.Update()
	}

	return err
}

func FindCharactersByUserID(userID string) ([]*Character, error) {

	charMap := make(map[int]*Character)
	for _, c := range characters {
		if c.UserID == userID {
			charMap[c.ID] = c
		}
	}

	var arr []*Character
	query := `select * from hops.characters where user_id = $1`

	if _, err := db.Select(&arr, query, userID); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("FindCharactersByUserID: %s", err.Error())
	}

	characterMutex.Lock()
	defer characterMutex.Unlock()

	var chars []*Character
	for _, c := range arr {
		char, ok := charMap[c.ID]
		if ok {
			chars = append(chars, char)
		} else {
			characters[c.ID] = c
			chars = append(chars, c)
		}
	}

	return chars, nil
}

func IsValidUsername(name string) (bool, error) {

	var (
		count int64
		err   error
		query string
	)

	re := regexp.MustCompile(`^[\p{L}\p{N}]{4,18}$`)
	if !re.MatchString(name) {
		return false, nil
	}

	query = `select count(*) from hops.characters where lower(name) = $1`

	if count, err = db.SelectInt(query, strings.ToLower(name)); err != nil {
		return false, fmt.Errorf("IsValidUsername: %s", err.Error())
	}

	return count == 0, nil
}

func FindCharacterByName(name string) (*Character, error) {

	for _, c := range characters {
		if c.Name == name {
			return c, nil
		}
	}

	character := &Character{}
	query := `select * from hops.characters where name = $1`

	if err := db.SelectOne(&character, query, name); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("FindCharacterByName: %s", err.Error())
	}

	characterMutex.Lock()
	defer characterMutex.Unlock()
	characters[character.ID] = character

	return character, nil
}

func CheckCharacter(chrID int, loginID string) bool {

	tmpInfo := struct {
		UserID    string `db:"id"`
		UserType  int    `db:"user_type"`
		ChrUserID string `db:"user_id"`
	}{}

	query := "select u.id, u.user_type, c.user_id from hops.characters as c left join hops.users as u on u.id = c.user_id where c.id = $1;"

	if err := db.SelectOne(&tmpInfo, query, chrID); err != nil {
		fmt.Println(err)
		return true
	}

	if tmpInfo.UserID != loginID {
		return true
	}

	if tmpInfo.UserType == 0 {
		return true
	}

	return false
}

func FindCharacterByID(id int) (*Character, error) {
	if id <= 0 {
		return nil, nil
	}

	characterMutex.RLock()
	c, ok := characters[id]
	characterMutex.RUnlock()
	if ok {
		return c, nil
	}

	character := &Character{}
	query := `select * from hops.characters where id = $1`

	if err := db.SelectOne(&character, query, id); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("FindCharacterByID: %s", err.Error())
	}

	characterMutex.Lock()
	defer characterMutex.Unlock()
	characters[character.ID] = character
	return character, nil
}

func (c *Character) GetAppearingItemSlots() []int {

	helmSlot := 0
	if c.HTVisibility&0x01 != 0 {
		helmSlot = 0x0133
	}

	maskSlot := 1
	if c.HTVisibility&0x02 != 0 {
		maskSlot = 0x0134
	}

	armorSlot := 2
	if c.HTVisibility&0x04 != 0 {
		armorSlot = 0x0135
	}

	bootsSlot := 9
	if c.HTVisibility&0x10 != 0 {
		bootsSlot = 0x0136
	}

	armorSlot2 := 2
	if c.HTVisibility&0x08 != 0 {
		armorSlot2 = 0x0137
	}

	if armorSlot2 != 2 {
		armorSlot = armorSlot2
	}

	return []int{helmSlot, maskSlot, armorSlot, 3, 4, 5, 6, 7, 8, bootsSlot, 10}
}

func (c *Character) GetEquipedItemSlots() []int {
	return []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 307, 309, 310, 312, 313, 314, 315}
}

func (c *Character) GetAllEquipedSlots() []int {
	return []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 307, 309, 310, 312, 313, 314, 315, 317, 318, 319}
}

func (c *Character) Logout() {
	c.IsOnline = false
	c.IsActive = false
	c.OnSight.Drops = map[int]interface{}{}
	c.OnSight.Mobs = map[int]interface{}{}
	c.OnSight.NPCs = map[int]interface{}{}
	c.OnSight.Pets = map[int]interface{}{}
	c.OnSight.Players = map[int]interface{}{}
	c.UsedConsumables.Items = map[int64]bool{}
	c.ExploreWorld = nil
	c.HandlerCB = nil
	c.PetHandlerCB = nil
	c.Socket.CharacterSelected = false
	c.PTS = 0
	c.TradeID = ""
	c.LeaveParty()
	c.EndPvP()

	sale := FindSale(c.PseudoID)
	if sale != nil {
		sale.Delete()
		c.SaleActive = false
		c.SaleActiveEpoch = 0
	}

	if trade := FindTrade(c); trade != nil {
		c.CancelTrade()
	}

	if c.GuildID > 0 {
		guild, err := FindGuildByID(c.GuildID)
		if err == nil && guild != nil {
			guild.InformMembers(c)
		}
	}

	if c.IsinWar {
		c.IsinWar = false
		if c.Faction == 1 {
			delete(OrderCharacters, c.ID)
		} else {
			delete(ShaoCharacters, c.ID)
		}
		c.Map = 1
	}

	if c.IsinLastMan {
		c.IsinLastMan = false
		LastManMutex.Lock()
		delete(LastManCharacters, c.ID)
		LastManMutex.Unlock()
		c.Map = 254
	}

	LogoutFiveBuffDelete(c)
	LogoutGuildWarBuffDelete(c)

	c.Update()
	c.Socket.User.Update()

	RemoveFromRegister(c)
	RemovePetFromRegister(c)
	DeleteCharacterFromCache(c.ID)
	DeleteStatFromCache(c.ID)
}

func (c *Character) EndPvP() {
	if c.DuelID > 0 {
		op, _ := FindCharacterByID(c.DuelID)
		if op != nil {
			op.Socket.Write(PVP_FINISHED)
			op.DuelID = 0
			op.DuelStarted = false
		}
		c.DuelID = 0
		c.DuelStarted = false
		c.Socket.Write(PVP_FINISHED)
	}
}

func DeleteCharacterFromCache(id int) {
	characterMutex.Lock()
	delete(characters, id)
	characterMutex.Unlock()
}

func CheckCharacterIsHave(id int) bool {

	return characters[id] != nil
}

func (c *Character) GetNearbyCharacters() ([]*Character, error) {

	var (
		distance = float64(50)
	)

	u, err := FindUserByID(c.UserID)
	if err != nil {
		return nil, err
	}

	myCoordinate := ConvertPointToLocation(c.Coordinate)

	characterMutex.RLock()
	allChars := funk.Values(characters)
	characterMutex.RUnlock()

	characters := funk.Filter(allChars, func(character *Character) bool {

		user, err := FindUserByID(character.UserID)
		if err != nil || user == nil {
			return false
		}

		characterCoordinate := ConvertPointToLocation(character.Coordinate)
		//fmt.Println(characters[character.ID].Name)

		return character.IsOnline && user.ConnectedServer == u.ConnectedServer && character.Map == c.Map &&
			(!character.Invisible || c.DetectionMode) && utils.CalculateDistance(characterCoordinate, myCoordinate) <= distance
	}).([]*Character)

	return characters, nil
}

func (c *Character) GetNearbyAIIDs() ([]int, error) {

	var (
		distance = 64.0
		ids      []int
	)

	if c.IsinWar {
		distance = 25.0
	}

	user, err := FindUserByID(c.UserID)
	if err != nil {
		return nil, err
	} else if user == nil {
		return nil, nil
	}

	candidates := AIsByMap[user.ConnectedServer][c.Map]
	filtered := funk.Filter(candidates, func(ai *AI) bool {

		characterCoordinate := ConvertPointToLocation(c.Coordinate)
		aiCoordinate := ConvertPointToLocation(ai.Coordinate)

		return utils.CalculateDistance(characterCoordinate, aiCoordinate) <= distance
	})

	for _, ai := range filtered.([]*AI) {
		ids = append(ids, ai.ID)
	}

	return ids, nil
}

func (c *Character) GetNearbyNPCIDs() ([]int, error) {

	var (
		distance = 50.0
		ids      []int
	)

	user, err := FindUserByID(c.UserID)
	if err != nil {
		return nil, err
	} else if user == nil {
		return nil, nil
	}

	filtered := funk.Filter(NPCPos, func(pos *NpcPosition) bool {

		characterCoordinate := ConvertPointToLocation(c.Coordinate)
		minLocation := ConvertPointToLocation(pos.MinLocation)
		maxLocation := ConvertPointToLocation(pos.MaxLocation)

		npcCoordinate := &utils.Location{X: (minLocation.X + maxLocation.X) / 2, Y: (minLocation.Y + maxLocation.Y) / 2}

		return c.Map == pos.MapID && utils.CalculateDistance(characterCoordinate, npcCoordinate) <= distance && pos.IsNPC // Unutma && !pos.Attackable
	})

	for _, pos := range filtered.([]*NpcPosition) {
		ids = append(ids, pos.ID)
	}

	return ids, nil
}

func (c *Character) GetNearbyDrops() ([]int, error) {

	var (
		distance = 50.0
		ids      []int
	)

	user, err := FindUserByID(c.UserID)
	if err != nil {
		return nil, err
	} else if user == nil {
		return nil, nil
	}

	allDrops := GetDropsInMap(user.ConnectedServer, c.Map)
	filtered := funk.Filter(allDrops, func(drop *Drop) bool {

		characterCoordinate := ConvertPointToLocation(c.Coordinate)

		return utils.CalculateDistance(characterCoordinate, &drop.Location) <= distance
	})

	for _, d := range filtered.([]*Drop) {
		ids = append(ids, d.ID)
	}

	return ids, nil
}

func (c *Character) SpawnCharacter() ([]byte, error) {

	if c == nil {
		return nil, nil
	}

	resp := CHARACTER_SPAWNED
	resp.Insert(utils.IntToBytes(uint64(c.PseudoID), 2, true), 6) // character pseudo id
	if c.IsActive {
		resp[12] = 3
	} else {
		resp[12] = 4
	}

	if c.DuelID > 0 {
		resp.Overwrite(utils.IntToBytes(500, 2, true), 13) // duel state
	}

	resp[17] = byte(len(c.Name))    // character name length
	resp.Insert([]byte(c.Name), 18) // character name
	resp.Overwrite(utils.IntToBytes(uint64(c.Level), 4, true), 18+len(c.Name))

	index := len(c.Name) + 18 + 4
	resp[index] = byte(c.Type) // character type
	index += 1

	index += 8

	coordinate := ConvertPointToLocation(c.Coordinate)
	resp.Insert(utils.FloatToBytes(coordinate.X, 4, true), index) // coordinate-x
	index += 4

	resp.Insert(utils.FloatToBytes(coordinate.Y, 4, true), index) // coordinate-y
	index += 8

	resp.Insert(utils.FloatToBytes(coordinate.X, 4, true), index) // coordinate-x
	index += 4

	resp.Insert(utils.FloatToBytes(coordinate.Y, 4, true), index) // coordinate-y
	index += 4                                                    // 58

	//resp.Overwrite(utils.IntToBytes(uint64(c.Socket.Stats.Honor), 4, true), 61+len(c.Name))

	index += 18

	if stats[c.ID] != nil {
		resp.Overwrite(utils.IntToBytes(uint64(c.Socket.Stats.HP), 4, true), index) // hp
	}

	index += 9

	if c.Morphed {
		resp.Overwrite(utils.IntToBytes(uint64(c.MorphedNPCID), 2, true), 85+len(c.Name)) //
	}

	resp.Overwrite(utils.IntToBytes(uint64(c.HonorRank), 4, true), 89+len(c.Name))

	resp[index] = byte(c.WeaponSlot) // weapon slot
	index += 16

	resp.Insert(utils.IntToBytes(uint64(c.GuildID), 4, true), index) // guild id
	index += 8

	resp[index] = byte(c.Faction) // character faction
	index += 10

	items, err := c.ShowItems()
	if err != nil {
		return nil, err
	}

	itemsData := items[11 : len(items)-2]
	sale := FindSale(c.PseudoID)
	if sale != nil {
		itemsData = []byte{0x05, 0xAA, 0x45, 0xF1, 0x00, 0x00, 0x00, 0xA1, 0x00, 0x00, 0x00, 0x00, 0xB4, 0x6C, 0xF1, 0x00, 0x01, 0x00, 0xA1, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x09, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x0A, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	}

	resp.Insert(itemsData, index)
	index += len(itemsData)

	length := int16(len(itemsData) + len(c.Name) + 111)

	if sale != nil {
		resp.Insert([]byte{0x02}, index) // sale indicator
		index++

		resp.Insert([]byte{byte(len(sale.Name))}, index) // sale name length
		index++

		resp.Insert([]byte(sale.Name), index) // sale name
		index += len(sale.Name)

		resp.Insert([]byte{0x00}, index)
		index++
		length += int16(len(sale.Name) + 3)
	}

	//	resp.Concat([]byte{0xAA, 0x55, 0x43})

	//resp.Concat([]byte{0xAA, 0x55, 0x0D, 0x00, 0x01, 0xB5, 0x0A, 0x00, 0x02, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x55, 0xAA})

	resp.SetLength(length)
	resp.Concat(items) // FIX => workaround for weapon slot,

	if c.GuildID > 0 {
		guild, err := FindGuildByID(c.GuildID)
		if err == nil && guild != nil {
			resp.Concat(guild.GetInfo())
		}
	}

	return resp, nil
}

func (c *Character) ShowItems() ([]byte, error) {

	if c == nil {
		return nil, nil
	}

	slots := c.GetAppearingItemSlots()
	inventory, err := c.InventorySlots()
	if err != nil {
		return nil, err
	}

	helm := inventory[slots[0]]
	mask := inventory[slots[1]]
	armor := inventory[slots[2]]
	weapon1 := inventory[slots[3]]
	weapon2 := inventory[slots[4]]
	boots := inventory[slots[9]]
	pet := inventory[slots[10]].Pet

	count := byte(4)
	if weapon1.ItemID > 0 {
		count++
	}
	if weapon2.ItemID > 0 {
		count++
	}
	if pet != nil && pet.IsOnline {
		count++
	}

	weapon1ID := int64(0)
	if weapon1.Appearance != 0 {
		weapon1ID = weapon1.Appearance
	}
	weapon2ID := int64(0)
	if weapon2.Appearance != 0 {
		weapon2ID = weapon2.Appearance
	}
	helmID := int64(0)
	if slots[0] == 0 && helm.Appearance != 0 {
		helmID = helm.Appearance
	}
	maskID := int64(0)
	if slots[1] == 1 && mask.Appearance != 0 {
		maskID = mask.Appearance
	}
	armorID := int64(0)
	if slots[2] == 2 && armor.Appearance != 0 {
		armorID = armor.Appearance
	}
	bootsID := int64(0)
	if slots[9] == 9 && boots.Appearance != 0 {
		bootsID = boots.Appearance
	}

	resp := SHOW_ITEMS
	resp.Insert(utils.IntToBytes(uint64(c.PseudoID), 2, true), 8) // character pseudo id
	resp[10] = byte(c.WeaponSlot)                                 // character weapon slot
	resp[11] = count

	index := 12
	resp.Insert(utils.IntToBytes(uint64(helm.ItemID), 4, true), index) // helm id
	index += 4

	resp.Insert(utils.IntToBytes(uint64(slots[0]), 2, true), index) // helm slot
	resp.Insert([]byte{0xA2}, index+2)
	index += 3

	resp.Insert(utils.IntToBytes(uint64(helm.Plus), 1, true), index) // helm plus
	resp.Insert(utils.IntToBytes(uint64(helmID), 4, true), index+1)  // Kinézet
	index += 5

	resp.Insert(utils.IntToBytes(uint64(mask.ItemID), 4, true), index) // mask id
	index += 4

	resp.Insert(utils.IntToBytes(uint64(slots[1]), 2, true), index) // mask slot
	resp.Insert([]byte{0xA2}, index+2)
	index += 3

	resp.Insert(utils.IntToBytes(uint64(mask.Plus), 1, true), index) // mask plus
	resp.Insert(utils.IntToBytes(uint64(maskID), 4, true), index+1)  // Kinézet
	index += 5

	resp.Insert(utils.IntToBytes(uint64(armor.ItemID), 4, true), index) // armor id
	index += 4

	resp.Insert(utils.IntToBytes(uint64(slots[2]), 2, true), index) // armor slot
	resp.Insert([]byte{0xA2}, index+2)
	index += 3

	resp.Insert(utils.IntToBytes(uint64(armor.Plus), 1, true), index) // armor plus
	resp.Insert(utils.IntToBytes(uint64(armorID), 4, true), index+1)  // Kinézet
	index += 5

	if weapon1.ItemID > 0 {
		resp.Insert(utils.IntToBytes(uint64(weapon1.ItemID), 4, true), index) // weapon1 id
		index += 4

		resp.Insert([]byte{0x03, 0x00}, index) // weapon1 slot
		resp.Insert([]byte{0xA2}, index+2)
		index += 3

		resp.Insert(utils.IntToBytes(uint64(weapon1.Plus), 1, true), index) // weapon1 plus
		resp.Insert(utils.IntToBytes(uint64(weapon1ID), 4, true), index+1)  // Kinézet
		index += 5
	}

	if weapon2.ItemID > 0 {
		resp.Insert(utils.IntToBytes(uint64(weapon2.ItemID), 4, true), index) // weapon2 id
		index += 4

		resp.Insert([]byte{0x04, 0x00}, index) // weapon2 slot
		resp.Insert([]byte{0xA2}, index+2)
		index += 3

		resp.Insert(utils.IntToBytes(uint64(weapon2.Plus), 1, true), index) // weapon2 plus
		resp.Insert(utils.IntToBytes(uint64(weapon2ID), 4, true), index+1)  // Kinézet
		index += 5
	}

	resp.Insert(utils.IntToBytes(uint64(boots.ItemID), 4, true), index) // boots id
	index += 4

	resp.Insert(utils.IntToBytes(uint64(slots[9]), 2, true), index) // boots slot
	resp.Insert([]byte{0xA2}, index+2)
	index += 3

	resp.Insert(utils.IntToBytes(uint64(boots.Plus), 1, true), index) // boots plus
	resp.Insert(utils.IntToBytes(uint64(bootsID), 4, true), index+1)  // Kinézet
	index += 5

	if pet != nil && pet.IsOnline {
		resp.Insert(utils.IntToBytes(uint64(inventory[10].ItemID), 4, true), index) // pet id
		index += 4

		resp.Insert(utils.IntToBytes(uint64(slots[10]), 2, true), index) // pet slot
		resp.Insert([]byte{pet.Level}, index+2)
		index += 3

		resp.Insert(utils.IntToBytes(4, 1, true), index) // pet plus ?
		resp.Insert([]byte{0x00, 0x00, 0x00, 0x00}, index+1)
		index += 5
	}

	//resp.SetLength(int16(count*12) + 8) // packet length
	//return resp, nil

	resp.SetLength(int16(binary.Size(resp) - 6))
	return resp, nil
}

func FindOnlineCharacterByUserID(userID string) (*Character, error) {

	var id int
	query := `select id from hops.characters where user_id = $1 and is_online = true`

	if err := db.SelectOne(&id, query, userID); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("FindOnlineCharacterByUserID: %s", err.Error())
	}

	return FindCharacterByID(id)
}

func FindCharactersInServer(server int) (map[int]*Character, error) {

	characterMutex.RLock()
	allChars := funk.Values(characters).([]*Character)
	characterMutex.RUnlock()

	allChars = funk.Filter(allChars, func(c *Character) bool {
		if c.Socket == nil {
			return false
		}
		user := c.Socket.User
		if user == nil {
			return false
		}

		return user.ConnectedServer == server && c.IsOnline
	}).([]*Character)

	candidates := make(map[int]*Character)
	for _, c := range allChars {
		candidates[c.ID] = c
	}

	return candidates, nil
}

func FindOnlineCharacters() (map[int]*Character, error) {

	characters := make(map[int]*Character)
	users := AllUsers()
	users = funk.Filter(users, func(u *User) bool {
		return u.ConnectedIP != "" && u.ConnectedServer > 0
	}).([]*User)

	for _, u := range users {
		c, _ := FindOnlineCharacterByUserID(u.ID)
		if c == nil {
			continue
		}

		characters[c.ID] = c
	}

	return characters, nil
}

func (c *Character) FindItemInInventoryByPlus(callback func(*InventorySlot) bool, plus uint8, itemIDs ...int64) (int16, *InventorySlot, error) {

	slots, err := c.InventorySlots()
	if err != nil {
		return -1, nil, err
	}

	for index, slot := range slots {
		if ok, _ := utils.Contains(itemIDs, slot.ItemID); ok {
			if index >= 0x43 && index <= 0x132 {
				continue
			}

			if slot.Plus != plus {
				continue
			}

			if callback == nil || callback(slot) {
				return int16(index), slot, nil
			}
		}
	}

	return -1, nil, nil
}

func (c *Character) FindItemInInventory(callback func(*InventorySlot) bool, itemIDs ...int64) (int16, *InventorySlot, error) {

	slots, err := c.InventorySlots()
	if err != nil {
		return -1, nil, err
	}

	for index, slot := range slots {
		if ok, _ := utils.Contains(itemIDs, slot.ItemID); ok {
			if index >= 0x43 && index <= 0x132 {
				continue
			}

			if callback == nil || callback(slot) {
				return int16(index), slot, nil
			}
		}
	}

	return -1, nil, nil
}

func (c *Character) DecrementItem(slotID int16, amount uint) *utils.Packet {

	if c == nil || c.Socket.User == nil {
		//c.Socket.Conn.Close()
		c.Socket.OnClose()
		return nil
	}

	slots, err := c.InventorySlots()
	if err != nil {
		return nil
	}

	slot := slots[slotID]
	if slot == nil || slot.ItemID == 0 || slot.Quantity < amount {
		return nil
	}

	slot.Quantity -= amount

	info := Items[slot.ItemID]
	resp := utils.Packet{}

	if info.TimerType == 3 {
		resp = GREEN_ITEM_COUNT
		resp.Insert(utils.IntToBytes(uint64(slotID), 2, true), 8)         // slot id
		resp.Insert(utils.IntToBytes(uint64(slot.Quantity), 4, true), 10) // item quantity
	} else {
		resp = ITEM_COUNT
		resp.Insert(utils.IntToBytes(uint64(slot.ItemID), 4, true), 8)    // item id
		resp.Insert(utils.IntToBytes(uint64(slotID), 2, true), 12)        // slot id
		resp.Insert(utils.IntToBytes(uint64(slot.Quantity), 2, true), 14) // item quantity
	}

	if slot.Quantity == 0 {
		if slot.ItemID != 0 {
			go logging.AddLogFile(5, c.Socket.User.ID+" idli kullanici ("+c.Name+") isimli karakterine ait ("+info.Name+") iteminin süresi doldu veya sildi.")
			c.UsedConsumables.ItemMutex.Lock()
			delete(c.UsedConsumables.Items, slot.ItemID)
			c.UsedConsumables.ItemMutex.Unlock()
		}
		err = slot.Delete()
		if err != nil {
			log.Print(err)
		}
		*slot = *NewSlot()
	} else {
		err = slot.Update()
		if err != nil {
			log.Print(err)
		}
	}

	return &resp
}

func (c *Character) FindFreeSlot() (int16, error) {

	slotID := 11
	slots, err := c.InventorySlots()
	if err != nil {
		return -1, err
	}

	for ; slotID <= 66; slotID++ {
		slot := slots[slotID]
		if slot.ItemID == 0 {
			return int16(slotID), nil
		}
	}

	if c.DoesInventoryExpanded() {
		slotID = 341
		for ; slotID <= 396; slotID++ {
			slot := slots[slotID]
			if slot.ItemID == 0 {
				return int16(slotID), nil
			}
		}
	}

	return -1, nil
}

func (c *Character) FindFreeSlots(count int) ([]int16, error) {

	var slotIDs []int16
	slots, err := c.InventorySlots()
	if err != nil {
		return nil, err
	}

	for slotID := int16(11); slotID <= 66; slotID++ {
		slot := slots[slotID]
		if slot.ItemID == 0 {
			slotIDs = append(slotIDs, slotID)
		}
		if len(slotIDs) == count {
			return slotIDs, nil
		}
	}

	if c.DoesInventoryExpanded() {
		for slotID := int16(341); slotID <= 396; slotID++ {
			slot := slots[slotID]
			if slot.ItemID == 0 {
				slotIDs = append(slotIDs, slotID)
			}
			if len(slotIDs) == count {
				return slotIDs, nil
			}
		}
	}

	return nil, fmt.Errorf("not enough inventory space")
}

func (c *Character) DoesInventoryExpanded() bool {
	buffs, err := FindBuffsByCharacterID(c.ID)
	if err != nil || len(buffs) == 0 {
		return false
	}

	buffs = funk.Filter(buffs, func(b *Buff) bool {
		return b.BagExpansion
	}).([]*Buff)

	return len(buffs) > 0
}

func (c *Character) AddItem(itemToAdd *InventorySlot, slotID int16, lootingDrop bool) (*utils.Packet, int16, error) {
	var (
		item *InventorySlot
		err  error
	)

	if itemToAdd == nil {
		return nil, -1, nil
	}

	itemToAdd.CharacterID = null.IntFrom(int64(c.ID))
	itemToAdd.UserID = null.StringFrom(c.UserID)

	i := Items[itemToAdd.ItemID]
	stackable := FindStackableByUIF(i.UIF)

	slots, err := c.InventorySlots()
	if err != nil {
		return nil, -1, err
	}

	stacking := false
	resp := utils.Packet{}
	if slotID == -1 {
		if stackable != nil { // stackable item
			if itemToAdd.ItemID == 235 || itemToAdd.ItemID == 242 || itemToAdd.ItemID == 254 {
				slotID, item, err = c.FindItemInInventoryByPlus(nil, itemToAdd.Plus, itemToAdd.ItemID)
			} else {
				slotID, item, err = c.FindItemInInventory(nil, itemToAdd.ItemID)
			}

			if err != nil {
				return nil, -1, err
			} else if slotID == -1 { // no same item found => find free slot
				slotID, err = c.FindFreeSlot()
				if err != nil {
					return nil, -1, err
				} else if slotID == -1 { // no free slot
					return nil, -1, nil
				}
				stacking = false
			} else if item.ItemID != itemToAdd.ItemID { // slot is not available
				return nil, -1, nil
			} else if item != nil { // can be stacked
				itemToAdd.Quantity += item.Quantity
				stacking = true
			}
		} else { // not stackable item => find free slot
			slotID, err = c.FindFreeSlot()
			if err != nil {
				return nil, -1, err
			} else if slotID == -1 {
				return nil, -1, nil
			}
		}
	}

	itemToAdd.SlotID = slotID
	slot := slots[slotID]
	id := slot.ID
	*slot = *itemToAdd
	slot.ID = id

	if !stacking && stackable == nil {
		//for j := 0; j < int(itemToAdd.Quantity); j++ {

		if lootingDrop {
			r := ITEM_LOOTED
			r.Insert(utils.IntToBytes(uint64(itemToAdd.ItemID), 4, true), 9) // item id
			r[14] = 0xA1
			if itemToAdd.Plus > 0 || itemToAdd.SocketCount > 0 {
				r[14] = 0xA2
			}

			r.Insert(utils.IntToBytes(uint64(slot.Quantity), 2, true), 15) // item count
			r.Insert(utils.IntToBytes(uint64(slotID), 2, true), 17)        // slot id
			r.Insert(itemToAdd.GetUpgrades(), 19)                          // item upgrades
			resp.Concat(r)
		} else {
			r := ITEM_ADDED
			r.Insert(utils.IntToBytes(uint64(itemToAdd.ItemID), 4, true), 8) // item id
			r.Insert(utils.IntToBytes(uint64(slot.Quantity), 2, true), 14)   // item quantity
			r.Insert(utils.IntToBytes(uint64(slotID), 2, true), 16)          // slot id
			r.Insert(utils.IntToBytes(uint64(c.Gold), 8, true), 22)          // gold
			resp.Concat(r)
		}
		/*
			slotID, err = c.FindFreeSlot()
			if err != nil || slotID == -1 {
				break
			}
			slot = slots[slotID]
		*/
		//}
	} else {

		if lootingDrop {
			r := ITEM_LOOTED
			r.Insert(utils.IntToBytes(uint64(itemToAdd.ItemID), 4, true), 9) // item id
			r[14] = 0xA1
			r.Insert(utils.IntToBytes(uint64(slot.Quantity), 2, true), 15) // item count
			r.Insert(utils.IntToBytes(uint64(slotID), 2, true), 17)        // slot id
			r.Insert(itemToAdd.GetUpgrades(), 19)                          // item upgrades
			resp.Concat(r)

		} else if stacking {
			resp = ITEM_COUNT
			resp.Insert(utils.IntToBytes(uint64(itemToAdd.ItemID), 4, true), 8)    // item id
			resp.Insert(utils.IntToBytes(uint64(slotID), 2, true), 12)             // slot id
			resp.Insert(utils.IntToBytes(uint64(itemToAdd.Quantity), 2, true), 14) // item quantity

		} else if !stacking {
			slot := slots[slotID]
			slot.ItemID = itemToAdd.ItemID
			slot.Quantity = itemToAdd.Quantity
			slot.Plus = itemToAdd.Plus
			slot.UpgradeArr = itemToAdd.UpgradeArr

			resp = ITEM_ADDED
			resp.Insert(utils.IntToBytes(uint64(itemToAdd.ItemID), 4, true), 8)    // item id
			resp.Insert(utils.IntToBytes(uint64(itemToAdd.Quantity), 2, true), 14) // item quantity
			resp.Insert(utils.IntToBytes(uint64(slotID), 2, true), 16)             // slot id
			resp.Insert(utils.IntToBytes(uint64(c.Gold), 8, true), 22)             // gold
		}
	}

	if slot.ID > 0 {
		err = slot.Update()
	} else {
		err = slot.Insert()
	}

	if err != nil {
		*slot = *NewSlot()
		resp = utils.Packet{}
		resp.Concat(slot.GetData(slotID))
		return &resp, -1, nil
	}

	InventoryItems.Add(slot.ID, slot)
	resp.Concat(slot.GetData(slotID))
	return &resp, slotID, nil
}

func (c *Character) ReplaceItem(itemID int, where, to int16) ([]byte, error) {
	sale := FindSale(c.PseudoID)
	if sale != nil {
		return nil, fmt.Errorf("cannot replace item on sale")
	} else if c.TradeID != "" {
		return nil, fmt.Errorf("cannot replace item on trade")
	}

	c.InvMutex.Lock()
	defer c.InvMutex.Unlock()

	invSlots, err := c.InventorySlots()
	if err != nil {
		return nil, err
	}

	whereItem := invSlots[where]
	if whereItem.ItemID == 0 {
		return nil, nil
	}

	toItem := invSlots[to]
	whereInfoItem := Items[whereItem.ItemID]
	toInfoItem := Items[toItem.ItemID]
	slots := c.GetAllEquipedSlots()
	useItem, _ := utils.Contains(slots, int(to))
	isWeapon := false
	if useItem {
		if !c.CanUse(whereInfoItem.CharacterType) {
			return nil, errors.New("Cheat Warning")
		}
		if whereInfoItem.MinLevel > c.Level || (whereInfoItem.MaxLevel > 0 && whereInfoItem.MaxLevel < c.Level) {
			resp := utils.Packet{0xAA, 0x55, 0x04, 0x00, 0x59, 0x04, 0xF0, 0x03, 0x55, 0xAA} // inappropriate level
			return resp, nil
		}
		if whereInfoItem.Slot == 3 || whereInfoItem.Slot == 4 {
			if int(to) == 4 || int(to) == 3 {
				isWeapon = true
			}
		}
		if int(to) != whereInfoItem.Slot && isWeapon == false {
			return nil, errors.New("Cheat Warning")
		}
	}

	if (where >= 317 && where <= 319) && (to >= 317 && to <= 319) || where == 10 && to == 10 {
		if whereInfoItem.Slot != toInfoItem.Slot {
			return nil, errors.New("Cheat Warning")
		}
	}
	if (where >= 0x0043 && where <= 0x132) && (to >= 0x0043 && to <= 0x132) && toItem.ItemID == 0 { // From: Bank, To: Bank
		whereItem.SlotID = to
		*toItem = *whereItem
		*whereItem = *NewSlot()

	} else if (where >= 0x0043 && where <= 0x132) && (to < 0x0043 || to > 0x132) && toItem.ItemID == 0 { // From: Bank, To: Inventory
		whereItem.SlotID = to
		whereItem.CharacterID = null.IntFrom(int64(c.ID))
		*toItem = *whereItem
		*whereItem = *NewSlot()

	} else if (to >= 0x0043 && to <= 0x132) && (where < 0x0043 || where > 0x132) && toItem.ItemID == 0 &&
		!whereItem.Activated && !whereItem.InUse && whereInfoItem.Tradable != 2 { // From: Inventory, To: Bank
		whereItem.SlotID = to
		whereItem.CharacterID = null.IntFromPtr(nil)
		*toItem = *whereItem
		*whereItem = *NewSlot()

	} else if ((to < 0x0043 || to > 0x132) && (where < 0x0043 || where > 0x132)) && toItem.ItemID == 0 { // From: Inventory, To: Inventory
		whereItem.SlotID = to
		*toItem = *whereItem
		*whereItem = *NewSlot()

	} else {
		return nil, nil
	}

	toItem.Update()
	InventoryItems.Add(toItem.ID, toItem)

	resp := ITEM_REPLACEMENT
	resp.Insert(utils.IntToBytes(uint64(itemID), 4, true), 8) // item id
	resp.Insert(utils.IntToBytes(uint64(where), 2, true), 12) // where slot id
	resp.Insert(utils.IntToBytes(uint64(to), 2, true), 14)    // to slot id

	whereAffects, toAffects := DoesSlotAffectStats(where), DoesSlotAffectStats(to)

	info := Items[int64(itemID)]
	if whereAffects {
		if info != nil && info.Timer > 0 {
			toItem.InUse = false
		}
	}
	if toAffects {
		if info != nil && info.Timer > 0 {
			toItem.InUse = true
		}
	}

	if whereAffects || toAffects {
		statData, err := c.GetStats()
		if err != nil {
			return nil, err
		}

		resp.Concat(statData)

		itemsData, err := c.ShowItems()
		if err != nil {
			return nil, err
		}

		p := nats.CastPacket{CastNear: true, CharacterID: c.ID, Type: nats.SHOW_ITEMS, Data: itemsData}
		if err = p.Cast(); err != nil {
			return nil, err
		}

		resp.Concat(itemsData)
	}

	if to == 0x0A {
		resp.Concat(invSlots[to].GetPetStats(c))
		resp.Concat(SHOW_PET_BUTTON)
	} else if where == 0x0A {
		resp.Concat(DISMISS_PET)
	}

	if (where >= 317 && where <= 319) || (to >= 317 && to <= 319) {
		resp.Concat(c.GetPetStats())
	}

	return resp, nil
}

func (c *Character) SwapItems(where, to int16) ([]byte, error) {
	sale := FindSale(c.PseudoID)
	if sale != nil {
		return nil, fmt.Errorf("cannot swap items on sale")
	} else if c.TradeID != "" {
		return nil, fmt.Errorf("cannot swap item on trade")
	}

	if c.CanSwap {
		return nil, fmt.Errorf("Cannot swap")
	}
	c.CanSwap = true
	c.InvMutex.Lock()

	invSlots, err := c.InventorySlots()
	if err != nil {
		return nil, err
	}

	whereItem := invSlots[where]
	toItem := invSlots[to]

	if whereItem.ItemID == 0 || toItem.ItemID == 0 {
		return nil, nil
	}

	whereInfoItem := Items[whereItem.ItemID]
	toInfoItem := Items[toItem.ItemID]
	slots := c.GetAllEquipedSlots()
	useItem, _ := utils.Contains(slots, int(to))
	useItem2, _ := utils.Contains(slots, int(where))
	isWeapon := false
	if useItem || useItem2 {
		if !c.CanUse(toInfoItem.CharacterType) {
			return nil, errors.New("Cheat Warning")
		}
		if (where >= 317 && where <= 319) || (to >= 317 && to <= 319) || to == 10 || where == 10 {
			if whereInfoItem.Slot != toInfoItem.Slot {
				return nil, errors.New("Cheat Warning")
			}
		}
		if whereInfoItem.MinLevel > c.Level || toInfoItem.MinLevel > c.Level || (toInfoItem.MaxLevel > 0 && toInfoItem.MaxLevel < c.Level) || (whereInfoItem.MaxLevel > 0 && whereInfoItem.MaxLevel < c.Level) {
			resp := utils.Packet{0xAA, 0x55, 0x04, 0x00, 0x59, 0x04, 0xF0, 0x03, 0x55, 0xAA} // inappropriate level
			return resp, nil
		}
		if whereInfoItem.Slot == 3 || whereInfoItem.Slot == 4 || toInfoItem.Slot == 3 || toInfoItem.Slot == 4 {
			if int(to) == 4 || int(to) == 3 {
				isWeapon = true
			}
		}
		if int(to) != whereInfoItem.Slot && isWeapon == false {
			return nil, errors.New("Cheat Warning")
		}
	}

	if (where >= 0x0043 && where <= 0x132) && (to >= 0x0043 && to <= 0x132) { // From: Bank, To: Bank
		temp := *toItem
		*toItem = *whereItem
		*whereItem = temp
		toItem.SlotID = to
		whereItem.SlotID = where

	} else if (where >= 0x0043 && where <= 0x132) && (to < 0x0043 || to > 0x132) &&
		!toItem.Activated && !toItem.InUse { // From: Bank, To: Inventory

		temp := *toItem
		*toItem = *whereItem
		*whereItem = temp
		toItem.SlotID = to
		whereItem.SlotID = where

	} else if (to >= 0x0043 && to <= 0x132) && (where < 0x0043 || where > 0x132) &&
		!whereItem.Activated && !whereItem.InUse { // From: Inventory, To: Bank

		temp := *toItem
		*toItem = *whereItem
		*whereItem = temp
		toItem.SlotID = to
		whereItem.SlotID = where

	} else if (to < 0x0043 || to > 0x132) && (where < 0x0043 || where > 0x132) { // From: Inventory, To: Inventory
		if whereItem.ItemID == toItem.ItemID && whereInfoItem.Tradable != 2 && toInfoItem.Tradable != 2 {
			if whereItem.Activated || whereItem.InUse || toItem.Activated || toItem.InUse {
				return nil, nil
			}
		}

		temp := *toItem
		*toItem = *whereItem
		*whereItem = temp
		toItem.SlotID = to
		whereItem.SlotID = where

	} else {
		return nil, nil
	}

	whereItem.Update()
	toItem.Update()
	InventoryItems.Add(whereItem.ID, whereItem)
	InventoryItems.Add(toItem.ID, toItem)

	resp := ITEM_SWAP
	resp.Insert(utils.IntToBytes(uint64(where), 4, true), 9)  // where slot
	resp.Insert(utils.IntToBytes(uint64(where), 2, true), 13) // where slot
	resp.Insert(utils.IntToBytes(uint64(to), 2, true), 15)    // to slot
	resp.Insert(utils.IntToBytes(uint64(to), 4, true), 17)    // to slot
	resp.Insert(utils.IntToBytes(uint64(to), 2, true), 21)    // to slot
	resp.Insert(utils.IntToBytes(uint64(where), 2, true), 23) // where slot

	whereAffects, toAffects := DoesSlotAffectStats(where), DoesSlotAffectStats(to)

	if whereAffects {
		item := whereItem // new item
		info := Items[int64(item.ItemID)]
		if info != nil && info.Timer > 0 {
			item.InUse = true
		}

		item = toItem // old item
		info = Items[int64(item.ItemID)]
		if info != nil && info.Timer > 0 {
			item.InUse = false
		}
	}

	if toAffects {
		item := whereItem // old item
		info := Items[int64(item.ItemID)]
		if info != nil && info.Timer > 0 {
			item.InUse = false
		}

		item = toItem // new item
		info = Items[int64(item.ItemID)]
		if info != nil && info.Timer > 0 {
			item.InUse = true
		}
	}

	if whereAffects || toAffects {

		statData, err := c.GetStats()
		if err != nil {
			return nil, err
		}

		resp.Concat(statData)

		itemsData, err := c.ShowItems()
		if err != nil {
			return nil, err
		}

		p := nats.CastPacket{CastNear: true, CharacterID: c.ID, Data: itemsData, Type: nats.SHOW_ITEMS}
		if err = p.Cast(); err != nil {
			return nil, err
		}

		resp.Concat(itemsData)
	}

	if to == 0x0A {
		resp.Concat(invSlots[to].GetPetStats(c))
		resp.Concat(SHOW_PET_BUTTON)
	}

	if (where >= 317 && where <= 319) || (to >= 317 && to <= 319) {
		resp.Concat(c.GetPetStats())
	}

	time.AfterFunc(time.Millisecond*200, func() {
		c.CanSwap = false
		c.InvMutex.Unlock()
	})

	return resp, nil
}

func (c *Character) SplitItem(where, to, quantity uint16) ([]byte, error) {
	sale := FindSale(c.PseudoID)
	if sale != nil {
		return nil, fmt.Errorf("cannot split item on sale")
	} else if c.TradeID != "" {
		return nil, fmt.Errorf("cannot split item on trade")
	}

	c.InvMutex.Lock()
	defer c.InvMutex.Unlock()

	slots, err := c.InventorySlots()
	if err != nil {
		return nil, err
	}

	whereItem := slots[where]
	toItem := slots[to]

	if toItem.ItemID != 0 {
		return nil, nil
	}

	if whereItem.Activated || whereItem.InUse {
		return nil, nil
	}

	info := Items[whereItem.ItemID]

	if info.Timer > 0 {
		return nil, nil
	}

	if quantity > 0 {

		if whereItem.Quantity >= uint(quantity) {
			*toItem = *whereItem
			toItem.SlotID = int16(to)
			toItem.Quantity = uint(quantity)
			c.DecrementItem(int16(where), uint(quantity))

		} else {
			return nil, nil
		}

		toItem.Insert()
		InventoryItems.Add(toItem.ID, toItem)

		resp := SPLIT_ITEM
		resp.Insert(utils.IntToBytes(uint64(toItem.ItemID), 4, true), 8)       // item id
		resp.Insert(utils.IntToBytes(uint64(whereItem.Quantity), 2, true), 14) // remaining quantity
		resp.Insert(utils.IntToBytes(uint64(where), 2, true), 16)              // where slot id

		resp.Insert(utils.IntToBytes(uint64(toItem.ItemID), 4, true), 52) // item id
		resp.Insert(utils.IntToBytes(uint64(quantity), 2, true), 58)      // new quantity
		resp.Insert(utils.IntToBytes(uint64(to), 2, true), 60)            // to slot id
		resp.Concat(toItem.GetData(int16(to)))
		return resp, nil
	}

	return nil, nil
}

func (c *Character) GetHPandChi() []byte {
	hpChi := HP_CHI
	if c.Socket == nil {
		return nil
	}
	stat := c.Socket.Stats

	hpChi.Insert(utils.IntToBytes(uint64(c.PseudoID), 2, true), 5)
	hpChi.Insert(utils.IntToBytes(uint64(c.PseudoID), 2, true), 7)
	hpChi.Insert(utils.IntToBytes(uint64(stat.HP), 4, true), 9)
	hpChi.Insert(utils.IntToBytes(uint64(stat.CHI), 4, true), 13)

	count := 0
	buffs, _ := FindBuffsByCharacterID(c.ID)
	for _, buff := range buffs {

		_, ok := BuffInfections[buff.ID]
		if !ok {
			continue
		}
		if buff.ID == 10100 || buff.ID == 90098 {

			continue
		}

		hpChi.Insert(utils.IntToBytes(uint64(buff.ID), 4, true), 22)
		hpChi.Insert(utils.IntToBytes(uint64(buff.SkillPlus), 1, false), 26)
		hpChi.Insert([]byte{0x01}, 27)
		count++
	}

	if c.AidMode {
		hpChi.Insert(utils.IntToBytes(11121, 4, true), 22)
		hpChi.Insert([]byte{0x00, 0x00}, 26)
		count++
	}

	hpChi[21] = byte(count) // buff count

	injuryNumbers := c.CalculateInjury()
	injury1 := fmt.Sprintf("%x", injuryNumbers[1]) //0.7
	injury0 := fmt.Sprintf("%x", injuryNumbers[0]) //0.1
	injury3 := fmt.Sprintf("%x", injuryNumbers[3]) //17.48
	injury2 := fmt.Sprintf("%x", injuryNumbers[2]) //1.09
	injuryByte1 := string(injury0 + injury1)
	data, err := hex.DecodeString(injuryByte1)
	if err != nil {
		panic(err)
	}
	injuryByte2 := string(injury3 + injury2)
	data2, err := hex.DecodeString(injuryByte2)
	if err != nil {
		panic(err)
	}

	hpChi.Overwrite(data, len(hpChi)-18)
	hpChi.Overwrite(data2, len(hpChi)-16)

	hpChi.SetLength(int16(0x28 + count*6))

	// hpChi[19] = byte(0x02)
	// hpChi[21] = byte(count) // buff count
	// index += 5
	// //hpChi[index] = byte(15)
	// //hpChi.SetLength(int16(binary.Size(hpChi) - 6))
	// hpChi.SetLength(int16(0x28 + count*6))
	return hpChi
}

func (c *Character) Handler() {
	defer func() {
		if err := recover(); err != nil {
			log.Println(err)
			log.Printf("handler error: %+v", string(dbg.Stack()))
			c.HandlerCB = nil
			//c.Socket.Conn.Close()
			c.Socket.OnClose() // 01.12.2023 // BUG FIX
		}
	}()

	st := c.Socket.Stats
	c.Epoch++

	if st.HP > 0 && c.Epoch%2 == 0 {
		//hp, chi := st.HP, st.CHI
		hp, chi, injury := st.HP, st.CHI, c.Injury
		if st.HP += st.HPRecoveryRate; st.HP > st.MaxHP {
			st.HP = st.MaxHP
		}

		if st.CHI += st.CHIRecoveryRate; st.CHI > st.MaxCHI {
			st.CHI = st.MaxCHI
		}

		// if c.Meditating {
		// 	if st.HP += st.HPRecoveryRate; st.HP > st.MaxHP {
		// 		st.HP = st.MaxHP
		// 	}

		// 	if st.CHI += st.CHIRecoveryRate; st.CHI > st.MaxCHI {
		// 		st.CHI = st.MaxCHI
		// 	}
		// }
		if c.Meditating {
			if c.Injury > 0 {
				c.Injury--
				if c.Injury < 0 {
					c.Injury = 0
				} else if c.Injury <= 70 {
					statData, err := c.GetStats()
					if err == nil {
						c.Socket.Write(statData)
					}
				}
			}
			hprecoveramount := st.MaxHP * st.HPRecoveryRate / 1000
			if st.HP += hprecoveramount; st.HP > st.MaxHP {
				st.HP = st.MaxHP
			}

			if st.CHI += st.CHIRecoveryRate; st.CHI > st.MaxCHI {
				st.CHI = st.MaxCHI
			}
		}

		if hp != st.HP || chi != st.CHI || injury != c.Injury {
			c.Socket.Write(c.GetHPandChi()) // hp-chi packet
		}

	} else if st.HP > 0 && c.Epoch%5 == 0 {
		hp, chi, injury := st.HP, st.CHI, c.Injury
		hprecoveramount := st.MaxHP * st.HPRecoveryRate / 1000
		if st.HP += hprecoveramount; st.HP > st.MaxHP {
			st.HP = st.MaxHP
		}

		if st.CHI += st.CHIRecoveryRate; st.CHI > st.MaxCHI {
			st.CHI = st.MaxCHI
		}
		if hp != st.HP || chi != st.CHI || injury != c.Injury {
			c.Socket.Write(c.GetHPandChi()) // hp-chi packet
		}
	} else if st.HP <= 0 && !c.Respawning { // dead
		if c.Injury < MAX_INJURY {
			c.Injury += 20
			if c.Injury >= MAX_INJURY {
				c.Injury = MAX_INJURY
			}
			if c.Injury >= 70 {
				statData, err := c.GetStats()
				if err == nil {
					c.Socket.Write(statData)
				}
			}
		}
		c.Respawning = true
		st.HP = 0
		c.Socket.Write(c.GetHPandChi())
		c.Socket.Write(CHARACTER_DIED)
		go c.RespawnCounter(10)

		if c.DuelID > 0 { // lost pvp
			opponent, _ := FindCharacterByID(c.DuelID)

			c.DuelID = 0
			c.DuelStarted = false
			c.Socket.Write(PVP_FINISHED)

			opponent.DuelID = 0
			opponent.DuelStarted = false
			opponent.Socket.Write(PVP_FINISHED)

			//info := fmt.Sprintf("[%s] has defeated [%s]", opponent.Name, c.Name)
			//r := messaging.InfoMessage(info)

			//p := nats.CastPacket{CastNear: true, CharacterID: c.ID, Data: r, Type: nats.PVP_FINISHED}
			//p.Cast()
		}

		if c.Map == 255 {
			if c.Faction == 1 {
				AddPointsToFactionWarFaction(5, 2)
			}

			if c.Faction == 2 {
				AddPointsToFactionWarFaction(5, 1)
			}

		}

		c.Targets = []*Target{}
		c.PlayerTargets = []*PlayerTarget{}
		c.Selection = 0
		if c.Class == 34 {
			buff, _ := FindBuffByID(50, c.ID)
			if buff != nil {
				c.Invisible = false
				buff.Delete()
				r := BUFF_EXPIRED
				r.Insert(utils.IntToBytes(uint64(buff.ID), 4, true), 6) // buff infection id
				c.Socket.Write(r)
			}

			buff, _ = FindBuffByID(105, c.ID)
			if buff != nil {
				c.DetectionMode = false
				buff.Delete()
				r := BUFF_EXPIRED
				r.Insert(utils.IntToBytes(uint64(buff.ID), 4, true), 6) // buff infection id
				c.Socket.Write(r)
			}
		}
	}

	if c.SaleActive {
		c.SaleActiveEpoch++
		if c.SaleActiveEpoch%3600 == 0 {
			tmpData, _, _ := c.AddItem(&InventorySlot{ItemID: 18500934, Quantity: 1}, -1, false)
			c.Socket.Write(*tmpData)
			c.SaleActiveEpoch = 0
		}
	}

	if c.AidTime <= 0 && c.AidMode {

		c.AidTime = 0
		c.AidMode = false
		c.Socket.Write(c.AidStatus())

		tpData, _ := c.ChangeMap(c.Map, nil)
		c.Socket.Write(tpData)
	}

	if c.AidMode && !c.HasAidBuff() {
		c.AidTime--
		if c.AidTime%60 == 0 {
			stData, _ := c.GetStats()
			c.Socket.Write(stData)
		}
	}

	if !c.AidMode && c.Epoch%2 == 0 && c.AidTime < 6600 {
		c.AidTime++
		if c.AidTime%60 == 6600 {
			stData, _ := c.GetStats()
			c.Socket.Write(stData)
		}
	}

	if c.PartyID != "" {
		c.UpdatePartyStatus()
	}

	c.HandleBuffs()
	c.HandleLimitedItems()

	c.Update()
	st.Update()
	time.AfterFunc(time.Second, func() {
		if c.HandlerCB != nil {
			c.HandlerCB()
		}
	})
}

func (c *Character) PetHandler() {

	defer func() {
		if err := recover(); err != nil {
			log.Println(err)
			log.Printf("%+v", string(dbg.Stack()))
		}
	}()

	{
		slots, err := c.InventorySlots()
		if err != nil {
			log.Println(err)
			goto OUT
		}

		petSlot := slots[0x0A]
		pet := petSlot.Pet
		if pet == nil || petSlot.ItemID == 0 || !pet.IsOnline {
			return
		}

		petInfo, ok := Pets[petSlot.ItemID]
		if !ok {
			return
		}

		if pet.HP <= 0 {
			resp := utils.Packet{}
			resp.Concat(c.GetPetStats())
			resp.Concat(DISMISS_PET)
			c.IsMounting = false
			c.Socket.Write(resp)

			pet.IsOnline = false
			return
		}

		//pet.Fullness = byte(100)
		//pet.Loyalty = byte(100)

		if c.AidMode {
			if c.PlayerAidSettings.PetFood1ItemID != 0 && pet.IsOnline {
				slotID, item, err := c.FindItemInInventory(nil, c.PlayerAidSettings.PetFood1ItemID)
				if err != nil {
					log.Println(err)
					return
				}
				if slotID == -1 || item == nil {
					return
				}
				percent := float32(c.PlayerAidSettings.PetFood1Percent) / float32(100)
				minPetHP := float32(pet.MaxHP) * percent
				if float32(pet.HP) <= minPetHP {
					petresp, err := c.UseConsumable(item, slotID)
					if err == nil {
						c.Socket.Write(petresp)
					} else {
						return
					}
				}

			}
			if c.PlayerAidSettings.PetChiPercent != 0 && pet.IsOnline {
				slotID, item, err := c.FindItemInInventory(nil, c.PlayerAidSettings.PetChiItemID)
				if err != nil {
					log.Println(err)
					return
				}
				if slotID == -1 || item == nil {
					return
				}
				percent := float32(c.PlayerAidSettings.PetChiPercent) / float32(100)
				minPetChi := float32(pet.MaxCHI) * percent
				if float32(pet.CHI) <= minPetChi {
					petresp, err := c.UseConsumable(item, slotID)
					if err == nil {
						c.Socket.Write(petresp)
					} else {
						return
					}
				}
			}
		}

		if petInfo.Combat && pet.Target == 0 { // && pet.Loyalty >= 10
			if pet.PetOwner.DuelID > 0 {
				pet.Target = c.DuelID
			} else {
				pet.Target, err = pet.FindTargetMobID(c) // 75% chance to trigger
				if err != nil {
					log.Println("AIHandler error:", err)
				}
			}
		}

		if pet.Target > 0 {
			pet.IsMoving = false
		}

		if c.Epoch%60 == 0 {
			if pet.Loyalty > 100 {
				pet.Loyalty = 100
			}
			pet.Loyalty++
			pet.Fullness++
			/*
				if pet.Fullness > 1 {
					pet.Fullness--
				}
			*/
			/*
				if pet.Fullness < 25 && pet.Loyalty > 1 {
					pet.Loyalty--
				} else if pet.Fullness >= 25 && pet.Loyalty < 100 {
					pet.Loyalty++
				}
			*/
		}

		cPetLevel := int(pet.Level)
		if c.Epoch%20 == 0 {
			if pet.HP < pet.MaxHP {
				pet.HP = int(math.Min(float64(pet.HP+cPetLevel*3), float64(pet.MaxHP)))
			}
			if pet.CHI < pet.MaxCHI {
				pet.CHI = int(math.Min(float64(pet.CHI+cPetLevel*2), float64(pet.CHI)))
			}
			pet.RefreshStats = true
		}

		if pet.RefreshStats {
			pet.RefreshStats = false
			c.Socket.Write(c.GetPetStats())
		}

		if !petInfo.Combat {
			pet.Target = 0
			goto OUT
		}

		if pet.IsMoving || pet.Casting {
			goto OUT
		}

		/*
			if pet.Loyalty < 10 {
				pet.Target = 0
			}
		*/

	BEGIN:

		ownerPos := ConvertPointToLocation(c.Coordinate)
		ownerdistance := utils.CalculateDistance(ownerPos, &pet.Coordinate)

		if int(ownerdistance) > 20 {
			pet.Target = 0
			pet.IsMoving = true
		}

		/*
			if int(ownerdistance) > 15 && c.Selection > 0 {
				pet.Target = c.Selection
			} else if int(ownerdistance) > 20 {
				pet.Target = 0
			} else if int(ownerdistance) > 30 {
				pet.IsOnline = false
				time.Sleep(time.Microsecond * 100)
				pet.IsOnline = true
			}
		*/

		if petInfo.Combat && pet.Target == 0 {
			pet.Target, err = pet.FindTargetMobID(c) // 75% chance to trigger
			if err != nil {
				log.Println("AIHandler error:", err)
			}
		}

		if pet.Target == 0 { // Idle mode

			ownerPos := ConvertPointToLocation(c.Coordinate)
			distance := utils.CalculateDistance(ownerPos, &pet.Coordinate)

			if distance > 10 { // Pet is so far from his owner
				pet.IsMoving = true
				targetX := utils.RandFloat(ownerPos.X-5, ownerPos.X+5)
				targetY := utils.RandFloat(ownerPos.Y-5, ownerPos.Y+5)

				target := utils.Location{X: targetX, Y: targetY}
				pet.TargetLocation = target
				speed := float64(10.0)

				token := pet.MovementToken
				for token == pet.MovementToken {
					pet.MovementToken = utils.RandInt(1, math.MaxInt64)
				}

				go pet.MovementHandler(pet.MovementToken, &pet.Coordinate, &target, speed)
			}
		} else { // Target mode
			target := GetFromRegister(c.Socket.User.ConnectedServer, c.Map, uint16(pet.Target))
			if _, ok := target.(*AI); ok { // attacked to ai
				mob, ok := GetFromRegister(c.Socket.User.ConnectedServer, c.Map, uint16(pet.Target)).(*AI)
				if !ok || mob == nil {
					pet.Target = 0
					goto OUT

				} else if mob.HP <= 0 {
					pet.Target = 0
					time.Sleep(time.Second)
					goto BEGIN
				}

				aiCoordinate := ConvertPointToLocation(mob.Coordinate)
				distance := utils.CalculateDistance(&pet.Coordinate, aiCoordinate)

				if distance <= 3 && pet.LastHit%2 == 0 { // attack
					seed := utils.RandInt(1, 1000)
					r := utils.Packet{}
					skillID := petInfo.GetSkills()
					skill, ok := SkillInfos[skillID]
					if seed < 500 && ok && pet.CHI >= skill.BaseChi && skillID != 0 {
						r.Concat(pet.CastSkill(c))
					} else {
						r.Concat(pet.Attack(c))
					}

					p := nats.CastPacket{CastNear: true, PetID: pet.PseudoID, Data: r, Type: nats.MOB_ATTACK}
					p.Cast()
					pet.LastHit++

				} else if distance > 3 && distance <= 50 { // chase
					pet.IsMoving = true
					target := GeneratePoint(aiCoordinate)
					pet.TargetLocation = target
					speed := float64(10.0)

					token := pet.MovementToken
					for token == pet.MovementToken {
						pet.MovementToken = utils.RandInt(1, math.MaxInt64)
					}

					go pet.MovementHandler(pet.MovementToken, &pet.Coordinate, &target, speed)
					pet.LastHit = 0

				} else {
					pet.LastHit++
				}
			} else { // FIX => attacked to player
				mob, err := FindCharacterByID(pet.Target)
				if err != nil {
					goto OUT
				}

				if pet.PetOwner.DuelID == 0 && !pet.PetOwner.DuelStarted {
					pet.Target = 0
					pet.IsMoving = true
					goto BEGIN
				}

				if mob.Socket.Stats.HP <= 0 || !c.CanAttack(mob) {
					pet.Target = 0
					time.Sleep(time.Second)
					goto BEGIN
				}
				aiCoordinate := ConvertPointToLocation(mob.Coordinate)
				distance := utils.CalculateDistance(&pet.Coordinate, aiCoordinate)

				if distance <= 3 && pet.LastHit%2 == 0 { // attack
					seed := utils.RandInt(1, 1000)
					r := utils.Packet{}
					skillID := petInfo.GetSkills()
					skill, ok := SkillInfos[skillID]
					if seed < 500 && ok && pet.CHI >= skill.BaseChi && skillID != 0 {
						r.Concat(pet.CastSkill(c))
					} else {
						r.Concat(pet.PlayerAttack(c))
					}

					p := nats.CastPacket{CastNear: true, PetID: pet.PseudoID, Data: r, Type: nats.MOB_ATTACK}
					p.Cast()
					pet.LastHit++

				} else if distance > 3 && distance <= 50 { // chase
					pet.IsMoving = true
					target := GeneratePoint(aiCoordinate)
					pet.TargetLocation = target
					speed := float64(10.0)

					token := pet.MovementToken
					for token == pet.MovementToken {
						pet.MovementToken = utils.RandInt(1, math.MaxInt64)
					}

					go pet.MovementHandler(pet.MovementToken, &pet.Coordinate, &target, speed)
					pet.LastHit = 0

				} else {
					pet.LastHit++
				}
			}
			petSlot.Update()
		}
	}

OUT:
	time.AfterFunc(time.Second, func() {
		if c.PetHandlerCB != nil {
			c.PetHandlerCB()
		}
	})
}

func (c *Character) HandleBuffs() {
	buffs, err := FindBuffsByCharacterID(c.ID)
	if err != nil || len(buffs) == 0 {
		return
	}

	stat := c.Socket.Stats

	if buff := buffs[0]; buff.StartedAt+buff.Duration <= c.Epoch { // buff expired
		stat.MinATK -= buff.ATK
		stat.MaxATK -= buff.ATK
		stat.ATKRate -= buff.ATKRate
		stat.Accuracy -= buff.Accuracy
		stat.MinArtsATK -= buff.ArtsATK
		stat.MaxArtsATK -= buff.ArtsATK
		stat.ArtsATKRate -= buff.ArtsATKRate
		stat.ArtsDEF -= buff.ArtsDEF
		stat.ArtsDEFRate -= buff.ArtsDEFRate
		stat.CHIRecoveryRate -= buff.CHIRecoveryRate
		stat.ConfusionDEF -= buff.ConfusionDEF
		stat.DEF -= buff.DEF
		stat.DefRate -= buff.DEFRate
		stat.DEXBuff -= buff.DEX
		stat.Dodge -= buff.Dodge
		stat.HPRecoveryRate -= buff.HPRecoveryRate
		stat.INTBuff -= buff.INT
		stat.MaxCHI -= buff.MaxCHI
		stat.MaxHP -= buff.MaxHP
		stat.ParalysisDEF -= buff.ParalysisDEF
		stat.PoisonDEF -= buff.PoisonDEF
		stat.STRBuff -= buff.STR
		c.ExpMultiplier -= float64(buff.EXPMultiplier) / 100
		c.DropMultiplier -= float64(buff.DropMultiplier) / 100
		c.RunningSpeed -= buff.RunningSpeed

		stat.PoisonATK -= buff.PoisonDamage
		stat.PoisonDEF -= buff.PoisonDEF
		stat.ParalysisATK -= buff.PoisonDamage
		stat.ParalysisDEF -= buff.ParalysisDEF
		stat.ConfusionATK -= buff.ConfusionDamage
		stat.ConfusionDEF -= buff.ConfusionDEF

		if c.RunningSpeed <= 5.6 {
			c.RunningSpeed = 5.6
		}

		if stat.HPRecoveryRate < 0 {
			stat.HPRecoveryRate = 0
		}

		if c.ExpMultiplier < 1 {
			c.ExpMultiplier = 1
		}
		if c.DropMultiplier < 1 {
			c.DropMultiplier = 1
		}

		c.Update()

		data, _ := c.GetStats()

		r := BUFF_EXPIRED
		r.Insert(utils.IntToBytes(uint64(buff.ID), 4, true), 6) // buff infection id
		r.Concat(data)

		c.Socket.Write(r)
		buff.Delete()

		p := &nats.CastPacket{CastNear: true, CharacterID: c.ID, Data: c.GetHPandChi()}
		p.Cast()

		if buff.ID == 241 || buff.ID == 244 || buff.ID == 138 || buff.ID == 139 { // invisibility
			c.Invisible = false
			if c.DuelID > 0 {
				opponent, _ := FindCharacterByID(c.DuelID)
				sock := opponent.Socket
				if sock != nil {
					time.AfterFunc(time.Second*1, func() {
						sock.Write(opponent.OnDuelStarted())
					})
				}
			}

		} else if buff.ID == 142 || buff.ID == 242 || buff.ID == 245 { // detection arts
			c.DetectionMode = false
		}

		if len(buffs) == 1 {
			buffs = []*Buff{}
		} else {
			buffs = buffs[1:]
		}
	}

	for _, buff := range buffs {
		mapping := map[int]int{19000018: 10100, 19000019: 10098}
		id := buff.ID
		if d, ok := mapping[buff.ID]; ok {
			id = d
		}

		infection, ok := BuffInfections[id]
		if !ok {
			continue
		}

		remainingTime := buff.StartedAt + buff.Duration - c.Epoch
		data := BUFF_INFECTION
		data.Insert(utils.IntToBytes(uint64(infection.ID), 4, true), 6)   // infection id
		data.Insert(utils.IntToBytes(uint64(remainingTime), 4, true), 11) // buff remaining time

		c.Socket.Write(data)
	}
}

func (c *Character) HandleLimitedItems() {

	invSlots, err := c.InventorySlots()
	if err != nil {
		return
	}

	slotIDs := []int16{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0133, 0x0134, 0x0135, 0x0136, 0x0137, 0x0138, 0x0139, 0x013A, 0x013B}

	for _, slotID := range slotIDs {
		slot := invSlots[slotID]
		item := Items[slot.ItemID]
		if item != nil && (item.TimerType == 1 || item.TimerType == 3) { // time limited item
			if c.Epoch%60 == 0 {
				data := c.DecrementItem(slotID, 1)
				c.Socket.Write(*data)
			}
			if slot.Quantity == 0 {
				/*
					data := ITEM_EXPIRED
					data.Insert(utils.IntToBytes(uint64(item.ID), 4, true), 6)

					removeData, _ := c.RemoveItem(slotID)
					data.Concat(removeData)

					statData, _ := c.GetStats()
					data.Concat(statData)
					c.Socket.Write(data)
				*/

				data := ITEM_EXPIRED
				data.Insert(utils.IntToBytes(uint64(item.ID), 4, true), 6)

				c.RemoveItem(slotID)
				data.Concat(slot.GetData(slotID))

				statData, _ := c.GetStats()
				data.Concat(statData)
				c.Socket.Write(data)
			}
		}
	}

	starts, ends := []int16{0x0B, 0x0155}, []int16{0x043, 0x018D}
	for j := 0; j < 2; j++ {
		start, end := starts[j], ends[j]
		for slotID := start; slotID <= end; slotID++ {
			slot := invSlots[slotID]
			item := Items[slot.ItemID]
			if slot.Activated {
				if c.Epoch%60 == 0 {
					data := c.DecrementItem(slotID, 1)
					c.Socket.Write(*data)
				}
				if slot.Quantity == 0 { // item expired
					data := ITEM_EXPIRED
					data.Insert(utils.IntToBytes(uint64(item.ID), 4, true), 6)

					c.RemoveItem(slotID)
					data.Concat(slot.GetData(slotID))

					statData, _ := c.GetStats()
					data.Concat(statData)
					c.Socket.Write(data)

					/*
						if slot.ItemID == 100080008 || slot.ItemID == 100080002 { // eyeball of divine
							fmt.Println("Burası 2 ?")
							c.UsedConsumables.ItemMutex.Lock()
							delete(c.UsedConsumables.Items, 100080002)
							c.UsedConsumables.ItemMutex.Unlock()
							c.DetectionMode = false
							statData, _ := c.GetStats()
							c.Socket.Write(statData)

							p := &nats.CastPacket{CastNear: true, CharacterID: c.ID, Data: c.GetHPandChi()}
							p.Cast()
						}
					*/

					if item.GetType() == FORM_TYPE {
						c.Morphed = false
						c.MorphedNPCID = 0
						c.Socket.Write(FORM_DEACTIVATED)
					}
				} else { // item not expired
					if slot.ItemID == 100080008 || slot.ItemID == 100080002 && !c.DetectionMode { // eyeball of divine
						c.DetectionMode = true
						statData, _ := c.GetStats()
						c.Socket.Write(statData)

						p := &nats.CastPacket{CastNear: true, CharacterID: c.ID, Data: c.GetHPandChi()}
						p.Cast()
					} else if item.GetType() == FORM_TYPE && !c.Morphed {
						c.Morphed = true
						c.MorphedNPCID = item.NPCID
						r := FORM_ACTIVATED
						r.Insert(utils.IntToBytes(uint64(c.MorphedNPCID), 4, true), 5) // form npc id
						c.Socket.Write(r)

					} else if item.GetType() == FORM_TYPE && c.Morphed {
						if c.Map == 233 || c.Map == 74 {
							slot.Activated = false
							slot.InUse = false
							c.Morphed = false
							c.MorphedNPCID = 0

							item.Update()
							c.Socket.Write(slot.GetData(slot.SlotID))

							statData, _ := c.GetStats()
							c.Socket.Write(statData)
							c.Socket.Write(FORM_DEACTIVATED)
							p := &nats.CastPacket{CastNear: true, CharacterID: c.ID, Data: c.GetHPandChi()}
							p.Cast()
						}
					}
				}
			}
		}
	}
}

func (c *Character) makeCharacterMorphed(npcID uint64, activateState bool) []byte {

	resp := FORM_ACTIVATED
	resp.Insert(utils.IntToBytes(uint64(npcID), 4, true), 5) // form npc id
	c.Socket.Write(resp)

	return resp
}

func (c *Character) RespawnCounter(seconds byte) {

	resp := RESPAWN_COUNTER
	resp[7] = seconds
	c.Socket.Write(resp)

	if seconds > 0 {
		time.AfterFunc(time.Second, func() {
			c.RespawnCounter(seconds - 1)
		})
	}
}

func (c *Character) Teleport(coordinate *utils.Location) []byte {

	c.SetCoordinate(coordinate)

	resp := TELEPORT_PLAYER
	resp.Insert(utils.FloatToBytes(coordinate.X, 4, true), 5) // coordinate-x
	resp.Insert(utils.FloatToBytes(coordinate.Y, 4, true), 9) // coordinate-x

	return resp
}

func (c *Character) ActivityStatus(remainingTime int) {
	/*
		c.GoldMutex.Lock()
		c.Gold++
		c.GoldMutex.Unlock()
	*/
	var msg string
	if c.IsActive || remainingTime == 0 {
		msg = "Your character has been activated."
		c.IsActive = true

		data, err := c.SpawnCharacter()
		if err != nil {
			return
		}

		p := &nats.CastPacket{CastNear: true, CharacterID: c.ID, Data: data, Type: nats.PLAYER_SPAWN}
		if err = p.Cast(); err != nil {
			return
		}

	} else {
		msg = fmt.Sprintf("Your character will be activated %d seconds later.", remainingTime)

		if c.IsOnline {
			time.AfterFunc(time.Second, func() {
				if !c.IsActive {
					c.ActivityStatus(remainingTime - 1)
				}
			})
		}
	}

	info := messaging.InfoMessage(msg)
	if c.Socket != nil {
		c.Socket.Write(info)
	}
}

func (c *Character) ItemEffects(st *Stat, start, end int16) error {

	slots, err := c.InventorySlots()
	if err != nil {
		return err
	}

	indexes := []int16{}

	for i := start; i <= end; i++ {
		slot := slots[i]
		if start == 0x0B || start == 0x155 {
			if slot != nil && slot.Activated && slot.InUse {
				indexes = append(indexes, i)
			}
		} else {
			indexes = append(indexes, i)
		}
	}

	additionalDropMultiplier, additionalExpMultiplier, additionalRunningSpeed := float64(0), float64(0), float64(0)
	for _, i := range indexes {
		if (i == 3 && c.WeaponSlot == 4) || (i == 4 && c.WeaponSlot == 3) {
			continue
		}

		item := slots[i]

		if item.ItemID != 0 {

			info := Items[item.ItemID]
			slotId := i
			if slotId == 4 {
				slotId = 3
			}

			if (info == nil || slotId != int16(info.Slot) || c.Level < info.MinLevel || (info.MaxLevel > 0 && c.Level > info.MaxLevel)) &&
				!(start == 0x0B || start == 0x155) {
				continue
			}

			ids := []int64{item.ItemID}

			for _, u := range item.GetUpgrades() {
				if u == 0 {
					break
				}
				ids = append(ids, int64(u))
			}

			for _, s := range item.GetSockets() {
				if s == 0 {
					break
				}
				ids = append(ids, int64(s))
			}

			for _, id := range ids {
				item := Items[id]
				if item == nil {
					continue
				}

				st.STRBuff += item.STR
				st.DEXBuff += item.DEX
				st.INTBuff += item.INT
				st.WindBuff += item.Wind
				st.WaterBuff += item.Water
				st.FireBuff += item.Fire

				st.DEF += item.Def + ((item.BaseDef1 + item.BaseDef2 + item.BaseDef3) / 3)
				st.DefRate += item.DefRate

				st.ArtsDEF += item.ArtsDef
				st.ArtsDEFRate += item.ArtsDefRate

				st.MaxHP += item.MaxHp
				st.MaxCHI += item.MaxChi

				st.Accuracy += item.Accuracy
				st.Dodge += item.Dodge

				st.MinATK += item.BaseMinAtk + item.MinAtk
				st.MaxATK += item.BaseMaxAtk + item.MaxAtk
				st.ATKRate += item.AtkRate

				st.MinArtsATK += item.MinArtsAtk
				st.MaxArtsATK += item.MaxArtsAtk
				st.ArtsATKRate += item.ArtsAtkRate
				additionalExpMultiplier += item.ExpRate
				additionalDropMultiplier += item.DropRate
				additionalRunningSpeed += item.RunningSpeed

				st.PoisonATK += item.PoisonATK
				st.PoisonDEF += item.PoisonDEF
				st.ParalysisATK += item.ParaATK
				st.ParalysisDEF += item.ParaDEF
				st.ConfusionATK += item.ConfusionATK
				st.ConfusionDEF += item.ConfusionDEF
			}
		}
	}

	c.AdditionalExpMultiplier += additionalExpMultiplier
	c.AdditionalDropMultiplier += additionalDropMultiplier
	c.AdditionalRunningSpeed += additionalRunningSpeed
	return nil
}

func (c *Character) GetExpAndSkillPts() []byte {

	resp := EXP_SKILL_PT_CHANGED
	resp.Insert(utils.IntToBytes(uint64(c.Exp), 8, true), 5)                        // character exp
	resp.Insert(utils.IntToBytes(uint64(c.Socket.Skills.SkillPoints), 4, true), 13) // character skill points
	return resp
}

func (c *Character) GetPTS() []byte {

	resp := PTS_CHANGED
	resp.Insert(utils.IntToBytes(uint64(c.PTS), 4, true), 6) // character pts
	return resp
}

func (c *Character) LootGold(amount uint64) []byte {

	c.AddingGold.Lock()
	defer c.AddingGold.Unlock()

	c.Gold += amount
	resp := GOLD_LOOTED
	resp.Insert(utils.IntToBytes(uint64(c.Gold), 8, true), 9) // character gold

	return resp
}

func (c *Character) AddExp(amount int64) ([]byte, bool) {

	if c == nil {
		return nil, false
	}

	if !c.IsOnline || !c.IsActive {
		return nil, false
	}

	c.AddingExp.Lock()
	defer c.AddingExp.Unlock()

	expMultipler := c.ExpMultiplier + c.AdditionalExpMultiplier
	var exp int64

	if c.Socket.User.ConnectedServer == 6 {
		newRate := (expMultipler * EXP_RATE) * 1.15
		exp = c.Exp + int64(float64(amount)*(newRate))
	} else if c.Socket.User.ConnectedServer == 9 {
		newRate := (expMultipler * EXP_RATE) * 1.50
		exp = c.Exp + int64(float64(amount)*(newRate))
	} else {
		exp = c.Exp + int64(float64(amount)*(expMultipler*EXP_RATE))
	}

	spIndex := utils.SearchUInt64(SkillPoints, uint64(c.Exp))

	canLevelUp := true
	if exp > 233332051410 && c.Level <= 100 {
		exp = 233332051410
	}
	if exp > 544951059310 && c.Level <= 200 {
		exp = 544951059310
	}
	c.Exp = exp
	spIndex2 := utils.SearchUInt64(SkillPoints, uint64(c.Exp))

	//resp := c.GetExpAndSkillPts()

	st := c.Socket.Stats
	if st == nil {
		return nil, false
	}

	levelUp := false
	level := int16(c.Level)
	targetExp := EXPs[level].Exp
	skPts, sp := 0, 0
	np := 0                                             //nature pts
	for exp >= targetExp && level < 299 && canLevelUp { // Levelling up && level < 100
		if c.Type <= 59 && level >= 100 {
			level = 100
			canLevelUp = false
		} else if c.Type <= 69 && level >= 200 {
			level = 200
			canLevelUp = false
		} else {
			level++
			st.HP = st.MaxHP
			skPts += EXPs[int16(level)].SkillPoints

			if level <= 100 {
				sp += int(level/10) + 4
			} else {
				if level > 100 && level <= 115 {
					sp += 3
				} else if level > 115 && level <= 130 {
					sp += 4
				} else if level > 130 && level <= 145 {
					sp += 5
				} else if level > 145 && level <= 160 {
					sp += 6
				} else if level > 160 && level <= 175 {
					sp += 7
				} else if level > 175 && level <= 190 {
					sp += 8
				} else if level > 190 && level <= 201 {
					sp += 9
				} else if level > 201 && level <= 210 {
					sp += 10
				} else if level > 210 && level <= 220 {
					sp += 11
				} else if level > 220 && level <= 240 {
					sp += 12
				} else if level > 240 {
					sp += 13
				}
			}

			targetExp = EXPs[level].Exp
			levelUp = true
		}

		if level >= 101 && level < 201 { //divine nature stats
			np += 4
			//skPts = spIndex2 - spIndex
		}
	}
	c.Level = int(level)
	resp := EXP_SKILL_PT_CHANGED
	if level < 101 { //divine nature stats
		skPts = spIndex2 - spIndex
	}
	c.Socket.Skills.SkillPoints += skPts
	if levelUp {
		//LOAD QUESTS

		//LEVEL ALINCA LEVEL UP BUFFU

		//buffinfo := BuffInfections[266]
		//buff := Buff{ID: int(266), CharacterID: c.ID, Name: buffinfo.Name, BagExpansion: false, StartedAt: c.Epoch, Duration: int64(2) * 2}
		//buff.Create()

		st.StatPoints += sp
		st.NaturePoints += np
		resp.Insert(utils.IntToBytes(uint64(exp), 8, true), 5)                          // character exp
		resp.Insert(utils.IntToBytes(uint64(c.Socket.Skills.SkillPoints), 4, true), 13) // character skill points

		if c.GuildID > 0 {
			guild, err := FindGuildByID(c.GuildID)
			if err == nil && guild != nil {
				guild.InformMembers(c)
			}

		}

		resp.Concat(messaging.SystemMessage(messaging.LEVEL_UP))
		resp.Concat(messaging.SystemMessage(messaging.LEVEL_UP_SP))
		resp.Concat(messaging.InfoMessage(c.GetLevelText()))

		spawnData, err := c.SpawnCharacter()
		if err == nil {
			c.Socket.Write(spawnData)
			c.Update()
			//resp.Concat(spawnData)
		}
	} else {
		resp.Insert(utils.IntToBytes(uint64(exp), 8, true), 5)                          // character exp
		resp.Insert(utils.IntToBytes(uint64(c.Socket.Skills.SkillPoints), 4, true), 13) // character skill points
	}

	go c.Socket.Skills.Update()
	go c.Update()
	return resp, levelUp
}

func (c *Character) AddPlayerExp(amount int64) ([]byte, bool) {

	c.AddingExp.Lock()
	defer c.AddingExp.Unlock()

	exp := c.Exp + int64(float64(amount))
	spIndex := utils.SearchUInt64(SkillPoints, uint64(c.Exp))
	canLevelUp := true
	if exp > 233332051410 && c.Level <= 100 {
		exp = 233332051410
	}
	if exp > 544951059310 && c.Level <= 200 {
		exp = 544951059310
	}
	c.Exp = exp
	spIndex2 := utils.SearchUInt64(SkillPoints, uint64(c.Exp))

	//resp := c.GetExpAndSkillPts()

	st := c.Socket.Stats
	if st == nil {
		return nil, false
	}

	levelUp := false
	level := int16(c.Level)
	targetExp := EXPs[level].Exp
	skPts, sp := 0, 0
	np := 0                                             //nature pts
	for exp >= targetExp && level < 299 && canLevelUp { // Levelling up && level < 100
		if c.Type <= 59 && level >= 100 {
			level = 100
			canLevelUp = false
		} else if c.Type <= 69 && level >= 200 {
			level = 200
			canLevelUp = false
		} else {
			level++
			st.HP = st.MaxHP

			sp += int(level/10) + 4

			if level > 100 && level <= 115 {
				sp = 3
			} else if level > 115 && level <= 130 {
				sp = 4
			} else if level > 130 && level <= 145 {
				sp = 5
			} else if level > 145 && level <= 160 {
				sp = 6
			} else if level > 160 && level <= 175 {
				sp = 7
			} else if level > 175 && level <= 190 {
				sp = 8
			} else if level > 190 && level <= 201 {
				sp = 9
			} else if level > 201 && level <= 210 {
				sp = 10
			} else if level > 210 && level <= 220 {
				sp = 11
			} else if level > 220 && level <= 240 {
				sp = 12
			} else if level > 240 {
				sp = 13
			}

			targetExp = EXPs[level].Exp
			levelUp = true
		}
		if level >= 101 && level < 201 { //divine nature stats
			np += 4
			skPts = spIndex2 - spIndex
		}
	}
	c.Level = int(level)
	resp := EXP_SKILL_PT_CHANGED
	if level < 101 { //divine nature stats
		skPts = spIndex2 - spIndex
	}
	skPts = spIndex2 - spIndex
	c.Socket.Skills.SkillPoints += skPts
	if levelUp {
		st.StatPoints += sp
		st.NaturePoints += np
		resp.Insert(utils.IntToBytes(uint64(exp), 8, true), 5)                          // character exp
		resp.Insert(utils.IntToBytes(uint64(c.Socket.Skills.SkillPoints), 4, true), 13) // character skill points

		if c.GuildID > 0 {
			guild, err := FindGuildByID(c.GuildID)
			if err == nil && guild != nil {
				guild.InformMembers(c)
			}
		}

		resp.Concat(messaging.SystemMessage(messaging.LEVEL_UP))
		resp.Concat(messaging.SystemMessage(messaging.LEVEL_UP_SP))
		resp.Concat(messaging.InfoMessage(c.GetLevelText()))

		spawnData, err := c.SpawnCharacter()
		if err == nil {
			c.Socket.Write(spawnData)
			c.Update()
			//resp.Concat(spawnData)
		}
	} else {
		resp.Insert(utils.IntToBytes(uint64(exp), 8, true), 5)                          // character exp
		resp.Insert(utils.IntToBytes(uint64(c.Socket.Skills.SkillPoints), 4, true), 13) // character skill points
	}
	go c.Socket.Skills.Update()
	return resp, levelUp
}

func (c *Character) LosePlayerExp(percent int) (int64, error) {
	level := int16(c.Level)
	expminus := int64(0)
	if level >= 10 {
		oldExp := EXPs[level-1].Exp
		resp := EXP_SKILL_PT_CHANGED
		if oldExp <= c.Exp {
			per := float64(percent) / 200
			expLose := float64(c.Exp) * float64(1-per)
			if int64(expLose) >= oldExp {
				exp := c.Exp - int64(expLose)
				expminus = int64(float64(exp) * float64(1-0.30))
				c.Exp = int64(expLose)
			} else {
				exp := c.Exp - oldExp
				expminus = int64(float64(exp) * float64(1-0.30))
				c.Exp = oldExp
			}
		}
		resp.Insert(utils.IntToBytes(uint64(c.Exp), 8, true), 5)                        // character exp
		resp.Insert(utils.IntToBytes(uint64(c.Socket.Skills.SkillPoints), 4, true), 13) // character skill points
		go c.Socket.Skills.Update()
		c.Socket.Write(resp)
	}
	return expminus, nil
}

func (c *Character) CombineItems(where, to int16) (int64, int16, error) {
	invSlots, err := c.InventorySlots()
	if err != nil {
		return 0, 0, err
	}

	c.InvMutex.Lock()
	defer c.InvMutex.Unlock()

	whereItem := invSlots[where]
	toItem := invSlots[to]

	if toItem.ItemID != whereItem.ItemID {
		return 0, 0, nil
	}

	whereInfoItem := Items[whereItem.ItemID]
	toInfoItem := Items[toItem.ItemID]
	slots := c.GetAllEquipedSlots()
	useItem, _ := utils.Contains(slots, int(to))
	isWeapon := false
	if useItem {
		if !c.CanUse(whereInfoItem.CharacterType) {
			return 0, 0, nil
		}
		if whereInfoItem.MinLevel > c.Level || (whereInfoItem.MaxLevel > 0 && whereInfoItem.MaxLevel < c.Level) {
			return 0, 0, nil
		}
		if whereInfoItem.Slot == 3 || whereInfoItem.Slot == 4 {
			if int(to) == 4 || int(to) == 3 {
				isWeapon = true
			}
		}
		if int(to) != whereInfoItem.Slot && !isWeapon {
			return 0, 0, nil
		}
	}

	if (where >= 317 && where <= 319) && (to >= 317 && to <= 319) || where == 10 && to == 10 {
		if whereInfoItem.Slot != toInfoItem.Slot {
			return 0, 0, nil
		}
	}

	if toItem.ItemID == whereItem.ItemID {
		info := Items[toItem.ItemID]
		stackable := FindStackableByUIF(info.UIF)

		if stackable != nil {
			if whereItem.Plus == toItem.Plus {
				toItem.Quantity += whereItem.Quantity
				go toItem.Update()
				whereItem.Delete()
				*whereItem = *NewSlot()
				return toItem.ItemID, int16(toItem.Quantity), nil
			}
		}
	}

	return 0, 0, errors.New("error")
}

func (c *Character) BankItems() []byte {

	bankSlots, err := c.InventorySlots()
	if err != nil {
		return nil
	}

	bankSlots = bankSlots[0x43:0x133]
	resp := BANK_ITEMS

	index, length := 8, int16(4)
	for i, slot := range bankSlots {
		if slot.ItemID == 0 {
			continue
		}

		resp.Insert(utils.IntToBytes(uint64(slot.ItemID), 4, true), index) // item id
		index += 4

		resp.Insert([]byte{0x00, 0xA1, 0x01, 0x00}, index)
		index += 4

		resp.Insert(utils.IntToBytes(uint64(i+0x43), 2, true), index) // slot id
		index += 2

		resp.Insert([]byte{0x00, 0x00, 0x00, 0x00}, index)
		index += 4
		length += 14
	}

	resp.SetLength(length)
	return resp
}

func (c *Character) GetGold() []byte {

	user, err := FindUserByID(c.UserID)
	if err != nil || user == nil {
		return nil
	}

	resp := GET_GOLD
	resp.Insert(utils.IntToBytes(uint64(c.Gold), 8, true), 6)         // gold
	resp.Insert(utils.IntToBytes(uint64(user.BankGold), 8, true), 14) // bank gold

	return resp
}

func (c *Character) ChangeMap(mapID int16, coordinate *utils.Location, args ...interface{}) ([]byte, error) {
	if !funk.Contains(unlockedMaps, mapID) {
		return nil, nil
	}

	resp, r := MAP_CHANGED, utils.Packet{}

	if c.AidMode {
		c.AidMode = false
		r.Concat(c.AidStatus())
	}

	/*
		if mapID == 31 || mapID == 32 {
			buff, _ := FindBuffByID(93, c.ID)
			if buff != nil {
				buff.Duration = int64(5) * 60
				buff.Update()
			} else {
				buffinfo := BuffInfections[93]
				buff = &Buff{ID: int(93), CharacterID: c.ID, Name: buffinfo.Name, BagExpansion: false, StartedAt: c.Epoch, Duration: int64(5) * 60}
				buff.Create()
			}
		}

	*/

	c.Targets = []*Target{}
	c.PlayerTargets = []*PlayerTarget{}
	c.Selection = 0

	/*
		if mapID == 233 {
			buff, _ := FindBuffByID(70022, c.ID) // 70022
			if buff != nil {
				buff.Duration = int64(60) * 60
				buff.Update()
			} else {
				buffinfo := BuffInfections[70022]
				buff = &Buff{ID: int(70022), CharacterID: c.ID, Name: buffinfo.Name, ArtsATK: -2000, RunningSpeed: -10, BagExpansion: false, StartedAt: c.Epoch, Duration: int64(60) * 60}
				buff.Create()
			}
		} else {
			if c.Map == 233 {
				buff, _ := FindBuffByID(70022, c.ID)
				if buff != nil {
					buff.Duration = 0
					buff.Delete()
				}
			}
		}

		c.HandleBuffs()

	*/

	c.Map = mapID
	c.EndPvP()
	if c.Map == 233 || c.Map == 74 {
		if c.Morphed {
			c.HandleLimitedItems()
		}
	}

	if coordinate == nil { // if no coordinate then teleport home
		d := SavePoints[uint8(mapID)]
		if d == nil {
			d = &SavePoint{Point: "(100.0,100.0)"}
		}
		coordinate = ConvertPointToLocation(d.Point)
	}
	/*
		if !c.IsinWar {
			if coordinate == nil { // if no coordinate then teleport home
				d := SavePoints[uint8(mapID)]
				if d == nil {
					d = &SavePoint{Point: "(100.0,100.0)"}
				}
				coordinate = ConvertPointToLocation(d.Point)
			}
		}  else {
			if c.Faction == 1 && c.Map != 230 {
				delete(OrderCharacters, c.ID)
				c.IsinWar = false
			} else if c.Faction == 2 && c.Map != 230 {
				delete(ShaoCharacters, c.ID)
				c.IsinWar = false
			}
		}
	*/

	if c.IsinLastMan {
		if c.Map != 254 {
			c.IsinLastMan = false
			LastManMutex.Lock()
			delete(LastManCharacters, c.ID)
			LastManMutex.Unlock()
		}
	}

	if funk.Contains(sharedMaps, mapID) { // shared map
		c.Socket.User.ConnectedServer = 1
	}

	if c.GuildID > 0 {
		guild, err := FindGuildByID(c.GuildID)
		if err == nil && guild != nil {
			guild.InformMembers(c)
		}
	}

	consItems, _ := FindConsignmentItemsBySellerID(c.ID)
	consItems = (funk.Filter(consItems, func(item *ConsignmentItem) bool {
		return item.IsSold
	}).([]*ConsignmentItem))
	if len(consItems) > 0 {
		r.Concat(CONSIGMENT_ITEM_SOLD)
	}

	slots, err := c.InventorySlots()
	if err == nil {
		pet := slots[0x0A].Pet
		if pet != nil && pet.IsOnline {
			pet.IsOnline = false
			r.Concat(DISMISS_PET)
			showpet, _ := c.ShowItems()
			resp.Concat(showpet)
			c.IsMounting = false
		}
	}

	RemovePetFromRegister(c)
	//RemoveFromRegister(c)
	//GenerateID(c)

	c.SetCoordinate(coordinate)

	if len(args) == 0 { // not logging in
		c.OnSight.DropsMutex.Lock()
		c.OnSight.Drops = map[int]interface{}{}
		c.OnSight.DropsMutex.Unlock()

		c.OnSight.MobMutex.Lock()
		c.OnSight.Mobs = map[int]interface{}{}
		c.OnSight.MobMutex.Unlock()

		c.OnSight.NpcMutex.Lock()
		c.OnSight.NPCs = map[int]interface{}{}
		c.OnSight.NpcMutex.Unlock()

		c.OnSight.PetsMutex.Lock()
		c.OnSight.Pets = map[int]interface{}{}
		c.OnSight.PetsMutex.Unlock()
	}

	resp[13] = byte(mapID)                                     // map id
	resp.Insert(utils.FloatToBytes(coordinate.X, 4, true), 14) // coordinate-x
	resp.Insert(utils.FloatToBytes(coordinate.Y, 4, true), 18) // coordinate-y
	resp[36] = byte(mapID)                                     // map id
	resp.Insert(utils.FloatToBytes(coordinate.X, 4, true), 46) // coordinate-x
	resp.Insert(utils.FloatToBytes(coordinate.Y, 4, true), 50) // coordinate-y
	resp[61] = byte(mapID)                                     // map id

	spawnData, _ := c.SpawnCharacter()
	r.Concat(spawnData)
	resp.Concat(r)
	resp.Concat(c.Socket.User.GetTime())
	resp.Concat(c.GetHPandChi())
	data, err := c.GetStats()
	if err == nil {
		resp.Concat(data)
	}
	return resp, nil
}

func DoesSlotAffectStats(slotNo int16) bool {
	return slotNo < 0x0B || (slotNo >= 0x0133 && slotNo <= 0x013B) || (slotNo >= 0x18D && slotNo <= 0x192)
}

func (c *Character) RemoveItem(slotID int16) ([]byte, error) {

	slots, err := c.InventorySlots()
	if err != nil {
		return nil, err
	}

	item := slots[slotID]

	resp := ITEM_REMOVED
	resp.Insert(utils.IntToBytes(uint64(item.ItemID), 4, true), 9) // item id
	resp.Insert(utils.IntToBytes(uint64(slotID), 2, true), 13)     // slot id

	affects, activated := DoesSlotAffectStats(slotID), item.Activated
	if affects || activated {
		item.Activated = false
		item.InUse = false

		statData, err := c.GetStats()
		if err != nil {
			return nil, err
		}

		resp.Concat(statData)
	}

	info := Items[item.ItemID]
	if item.ItemID != 0 {
		go logging.AddLogFile(5, c.Socket.User.ID+" idli kullanici ("+c.Name+") isimli karakterine ait ("+info.Name+") iteminin süresi doldu veya sildi.")
	}

	/*
		c.UsedConsumables.ItemMutex.Lock()
		delete(c.UsedConsumables.Items, item.ItemID)
		c.UsedConsumables.ItemMutex.Unlock()
	*/

	if activated {
		/*
			if item.ItemID == 100080008 || item.ItemID == 100080002 { // eyeball of divine
				fmt.Println("Öldük aq")
				c.DetectionMode = false
				statData, _ := c.GetStats()
				c.Socket.Write(statData)

				p := &nats.CastPacket{CastNear: true, CharacterID: c.ID, Data: c.GetHPandChi()}
				p.Cast()
			}
		*/

		if info != nil && info.GetType() == FORM_TYPE {
			c.Morphed = false
			c.MorphedNPCID = 0
			resp.Concat(FORM_DEACTIVATED)
		}

		data := ITEM_EXPIRED
		data.Insert(utils.IntToBytes(uint64(item.ItemID), 4, true), 6)
		resp.Concat(data)
	}

	if affects {
		itemsData, err := c.ShowItems()
		if err != nil {
			return nil, err
		}

		p := nats.CastPacket{CastNear: true, CharacterID: c.ID, Data: itemsData, Type: nats.SHOW_ITEMS}
		if err = p.Cast(); err != nil {
			return nil, err
		}

		resp.Concat(itemsData)
	}

	consItem, _ := FindConsignmentItemByID(item.ID)
	if consItem == nil { // item is not in consigment table
		err = item.Delete()
		if err != nil {
			return nil, err
		}

	} else { // seller did not claim the consigment item
		newItem := NewSlot()
		*newItem = *item
		newItem.UserID = null.StringFromPtr(nil)
		newItem.CharacterID = null.IntFromPtr(nil)
		newItem.Update()
		InventoryItems.Add(consItem.ID, newItem)
	}

	*item = *NewSlot()
	return resp, nil
}

func (c *Character) SellItem(itemID, slot, quantity int, unitPrice uint64) ([]byte, error) {

	sellPrice := unitPrice * uint64(quantity)
	percent := 0

	waterSpirit, _ := FindBuffByID(19000019, c.ID)
	if waterSpirit != nil {
		percent = 10
	}

	if percent > 0 {
		sellPrice = sellPrice + (sellPrice*uint64(percent))/100
	}

	c.LootGold(sellPrice)
	_, err := c.RemoveItem(int16(slot))
	if err != nil {
		return nil, err
	}

	resp := SELL_ITEM
	resp.Insert(utils.IntToBytes(uint64(itemID), 4, true), 8)  // item id
	resp.Insert(utils.IntToBytes(uint64(slot), 2, true), 12)   // slot id
	resp.Insert(utils.IntToBytes(uint64(c.Gold), 8, true), 14) // character gold

	return resp, nil
}

func (c *Character) GetStats() ([]byte, error) {

	if c == nil {
		log.Println("c is nil")
		return nil, nil

	} else if c.Socket == nil {
		log.Println("socket is nil")
		return nil, nil
	}

	st := c.Socket.Stats
	if st == nil {
		return nil, nil
	}

	err := st.Calculate()
	if err != nil {
		return nil, err
	}

	resp := GET_STATS

	index := 5
	resp.Insert(utils.IntToBytes(uint64(c.Level), 4, true), index) // character level
	index += 4

	duelState := 0
	if c.DuelID > 0 && c.DuelStarted {
		duelState = 500
	}

	resp.Insert(utils.IntToBytes(uint64(duelState), 2, true), index) // duel state
	index += 4

	resp.Insert(utils.IntToBytes(uint64(st.StatPoints), 2, true), index) // stat points
	index += 2

	resp.Insert(utils.IntToBytes(uint64(st.NaturePoints), 2, true), index)
	index += 2

	resp.Insert(utils.IntToBytes(uint64(c.Socket.Skills.SkillPoints), 4, true), index) // character skill points
	index += 6

	resp.Insert(utils.IntToBytes(uint64(c.Exp), 8, true), index) // character experience
	index += 8

	resp.Insert(utils.IntToBytes(uint64(c.AidTime), 4, true), index) // remaining aid
	index += 4
	index++

	targetExp := EXPs[int16(c.Level)].Exp
	resp.Insert(utils.IntToBytes(uint64(targetExp), 8, true), index) // character target experience
	index += 8

	resp.Insert(utils.IntToBytes(uint64(st.STR), 2, true), index) // character str
	index += 2

	resp.Insert(utils.IntToBytes(uint64(st.STR+st.STRBuff), 2, true), index) // character str buff
	index += 2

	resp.Insert(utils.IntToBytes(uint64(st.DEX), 2, true), index) // character dex
	index += 2

	resp.Insert(utils.IntToBytes(uint64(st.DEX+st.DEXBuff), 2, true), index) // character dex buff
	index += 2

	resp.Insert(utils.IntToBytes(uint64(st.INT), 2, true), index) // character int
	index += 2

	resp.Insert(utils.IntToBytes(uint64(st.INT+st.INTBuff), 2, true), index) // character int buff
	index += 2

	resp.Insert(utils.IntToBytes(uint64(st.Wind), 2, true), index) // character wind
	index += 2

	resp.Insert(utils.IntToBytes(uint64(st.Wind+st.WindBuff), 2, true), index) // character wind buff
	index += 2

	resp.Insert(utils.IntToBytes(uint64(st.Water), 2, true), index) // character water
	index += 2

	resp.Insert(utils.IntToBytes(uint64(st.Water+st.WaterBuff), 2, true), index) // character water buff
	index += 2

	resp.Insert(utils.IntToBytes(uint64(st.Fire), 2, true), index) // character fire
	index += 2

	resp.Insert(utils.IntToBytes(uint64(st.Fire+st.FireBuff), 2, true), index) // character fire buff
	index += 7

	resp.Insert(utils.FloatToBytes(c.RunningSpeed+c.AdditionalRunningSpeed, 4, true), index) // character running speed
	index += 10

	resp.Insert(utils.IntToBytes(uint64(st.MaxHP), 4, true), index) // character max hp
	index += 4

	resp.Insert(utils.IntToBytes(uint64(st.MaxCHI), 4, true), index) // character max chi
	index += 4

	resp.Insert(utils.IntToBytes(uint64(st.MinATK), 2, true), index) // character min atk
	index += 2

	resp.Insert(utils.IntToBytes(uint64(st.MaxATK), 2, true), index) // character max atk
	index += 2

	resp.Insert(utils.IntToBytes(uint64(st.DEF), 4, true), index) // character def
	index += 4
	resp.Insert(utils.IntToBytes(uint64(st.DEF), 4, true), index) // character def
	index += 4
	resp.Insert(utils.IntToBytes(uint64(st.DEF), 4, true), index) // character def
	index += 4

	resp.Insert(utils.IntToBytes(uint64(st.MinArtsATK), 4, true), index) // character min arts atk
	index += 4

	resp.Insert(utils.IntToBytes(uint64(st.MaxArtsATK), 4, true), index) // character max arts atk
	index += 4

	resp.Insert(utils.IntToBytes(uint64(st.ArtsDEF), 4, true), index) // character arts def
	index += 4

	resp.Insert(utils.IntToBytes(uint64(st.Accuracy), 2, true), index) // character accuracy
	index += 2

	resp.Insert(utils.IntToBytes(uint64(st.Dodge), 2, true), index) // character dodge
	index += 2

	index += 2
	resp.Insert(utils.IntToBytes(uint64(st.PoisonATK), 2, true), index) // character PoisonDamage
	index += 2
	resp.Insert(utils.IntToBytes(uint64(st.PoisonDEF), 2, true), index) // character PoisonDef
	index += 2
	index++
	resp.Insert(utils.IntToBytes(uint64(st.ConfusionATK), 2, true), index) // character ParaATK
	index += 2
	resp.Insert(utils.IntToBytes(uint64(st.ConfusionDEF), 2, true), index) // character ParaDEF
	index += 2
	index++
	resp.Insert(utils.IntToBytes(uint64(st.ParalysisATK), 2, true), index) // character ConfusionATK
	index += 2
	resp.Insert(utils.IntToBytes(uint64(st.ParalysisDEF), 2, true), index) // character ConfusionDef
	index += 3

	resp.SetLength(int16(binary.Size(resp) - 6))

	resp.Concat(c.GetHPandChi()) // hp and chi
	return resp, nil
}

func (c *Character) BSUpgrade(slotID int64, stones []*InventorySlot, luck, protection *InventorySlot, stoneSlots []int64, luckSlot, protectionSlot int64) ([]byte, error) {

	slots, err := c.InventorySlots()
	if err != nil {
		return nil, err
	}

	item := slots[slotID]

	if item.Plus >= 12 { // cannot be upgraded more
		resp := utils.Packet{0xAA, 0x55, 0x31, 0x00, 0x54, 0x02, 0xA6, 0x0F, 0x01, 0x00, 0xA3, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x55, 0xAA}
		resp.Insert(utils.IntToBytes(uint64(item.ItemID), 4, true), 9) // item id
		resp.Insert(utils.IntToBytes(uint64(slotID), 2, true), 17)     // slot id
		resp.Insert(item.GetUpgrades(), 19)                            // item upgrades
		resp[34] = byte(item.SocketCount)                              // socket count
		resp.Insert(item.GetSockets(), 35)                             // item sockets

		return resp, nil
	}

	info := Items[item.ItemID]

	cost := (info.BuyPrice / 10) * int64(item.Plus+1) * int64(math.Pow(2, float64(len(stones)-1)))

	if uint64(cost) > c.Gold {
		resp := messaging.SystemMessage(messaging.INSUFFICIENT_GOLD)
		return resp, nil

	} else if len(stones) == 0 {
		resp := messaging.SystemMessage(messaging.INCORRECT_GEM_QTY)
		return resp, nil
	}

	stone := stones[0]
	stoneInfo := Items[stone.ItemID]

	if int16(item.Plus) < stoneInfo.MinUpgradeLevel || stoneInfo.ID > 255 {
		resp := messaging.SystemMessage(messaging.INCORRECT_GEM)
		return resp, nil
	}

	itemType := info.GetType()
	typeMatch := (stoneInfo.Type == 190 && itemType == PET_ITEM_TYPE) || (stoneInfo.Type == 191 && itemType == HT_ARMOR_TYPE) ||
		(stoneInfo.Type == 192 && itemType == ACC_TYPE) || (stoneInfo.Type == 194 && itemType == WEAPON_TYPE) || (stoneInfo.Type == 195 && itemType == ARMOR_TYPE)

	if !typeMatch {

		resp := utils.Packet{0xAA, 0x55, 0x09, 0x00, 0x54, 0x02, 0xA4, 0x0F, 0x00, 0x55, 0xAA}
		resp.Insert(utils.IntToBytes(uint64(item.ItemID), 4, true), 9) // item id
		return resp, nil
	}

	rate := float64(STRRates[item.Plus] * len(stones))
	plus := item.Plus + 1

	if stone.Plus > 0 { // Precious Pendent or Ghost Dagger or Dragon Scale
		for i := 0; i < len(stones); i++ {
			for j := i; j < len(stones); j++ {
				if stones[i].Plus != stones[j].Plus { // mismatch stone plus
					resp := utils.Packet{0xAA, 0x55, 0x09, 0x00, 0x54, 0x02, 0xA4, 0x0F, 0x00, 0x55, 0xAA}
					resp.Insert(utils.IntToBytes(uint64(item.ItemID), 4, true), 9) // item id
					return resp, nil
				}
			}
		}

		plus = item.Plus + stone.Plus
		if plus > 15 {
			plus = 15
		}

		rate = float64(STRRates[plus-1] * len(stones))
	}

	if luck != nil {
		luckInfo := Items[luck.ItemID]
		if luckInfo.Type == 164 { // charm of luck
			var k float64
			if luck.ItemID == 13000034 {
				k = float64(luckInfo.SellPrice) / 140
			} else if luck.ItemID == 13000033 {
				k = float64(luckInfo.SellPrice) / 120
			} else {
				k = float64(luckInfo.SellPrice) / 100
			}

			rate += rate * k / float64(len(stones))
		} else if luckInfo.Type == 219 { // bagua
			if byte(luckInfo.SellPrice) != item.Plus { // item plus not matching with bagua
				resp := utils.Packet{0xAA, 0x55, 0x04, 0x00, 0x54, 0x02, 0xB6, 0x0F, 0x55, 0xAA}
				return resp, nil

			} else if len(stones) < 3 {
				resp := messaging.SystemMessage(messaging.INCORRECT_GEM_QTY)
				return resp, nil
			}

			rate = 1000
			bagRates := []int{luckInfo.HolyWaterUpg3, luckInfo.HolyWaterRate1, luckInfo.HolyWaterRate2, luckInfo.HolyWaterRate3}
			seed := utils.RandInt(0, 100)

			for i := 0; i < len(bagRates); i++ {
				if int(seed) > bagRates[i] {
					plus++
				}
			}
		} else if luckInfo.Type == 240 {
			if itemType != WEAPON_TYPE {
				resp := utils.Packet{0xAA, 0x55, 0x09, 0x00, 0x54, 0x02, 0xA4, 0x0F, 0x00, 0x55, 0xAA}
				resp.Insert(utils.IntToBytes(uint64(item.ItemID), 4, true), 9) // item id
				return resp, nil
			}
			k := float64(luckInfo.SellPrice) / 100
			rate += rate * k / float64(len(stones))
		} else if luckInfo.Type == 241 {
			if itemType != ARMOR_TYPE {
				resp := utils.Packet{0xAA, 0x55, 0x09, 0x00, 0x54, 0x02, 0xA4, 0x0F, 0x00, 0x55, 0xAA}
				resp.Insert(utils.IntToBytes(uint64(item.ItemID), 4, true), 9) // item id
				return resp, nil
			}
			k := float64(luckInfo.SellPrice) / 100
			rate += rate * k / float64(len(stones))
		} else if luckInfo.Type == 242 {
			if itemType != ACC_TYPE {
				resp := utils.Packet{0xAA, 0x55, 0x09, 0x00, 0x54, 0x02, 0xA4, 0x0F, 0x00, 0x55, 0xAA}
				resp.Insert(utils.IntToBytes(uint64(item.ItemID), 4, true), 9) // item id
				return resp, nil
			}
			k := float64(luckInfo.SellPrice) / 100
			rate += rate * k / float64(len(stones))
		}
	}

	protectionInfo := &Item{}
	if protection != nil {
		protectionInfo = Items[protection.ItemID]
	}

	resp := utils.Packet{}
	c.LootGold(-uint64(cost))
	resp.Concat(c.GetGold())

	seed := int(utils.RandInt(0, 1000))
	if float64(seed) < rate { // upgrade successful
		var codes []byte
		for i := item.Plus; i < plus; i++ {
			codes = append(codes, byte(stone.ItemID))
		}

		before := item.GetUpgrades()
		resp.Concat(item.Upgrade(int16(slotID), codes...))
		logger.Log(logging.ACTION_UPGRADE_ITEM, c.ID, fmt.Sprintf("Item (%d) upgraded: %s -> %s", item.ID, before, item.GetUpgrades()), c.UserID, c.Name)
		/*if item.Plus > 7 {
			if itemType != HT_ARMOR_TYPE && itemType != PET_ITEM_TYPE {
				makeAnnouncement(c.Name + " has upgraded his " + info.Name + " to +" + strconv.Itoa(int(item.Plus)) + " successfully")
			}
		}*/
		if luck != nil {
			luckInfo := Items[luck.ItemID]
			go logging.AddLogFile(4, c.Socket.User.ID+" idli kullanici ("+c.Name+") isimli karakterine ait ("+info.Name+") iteme başarılı şekilde artı bastı : +"+strconv.Itoa(int(item.Plus))+" Kullanılan luck: "+luckInfo.Name)
		} else {
			go logging.AddLogFile(4, c.Socket.User.ID+" idli kullanici ("+c.Name+") isimli karakterine ait ("+info.Name+") iteme başarılı şekilde artı bastı : +"+strconv.Itoa(int(item.Plus)))
		}

	} else if itemType == HT_ARMOR_TYPE || itemType == PET_ITEM_TYPE ||
		(protection != nil && protectionInfo.GetType() == SCALE_TYPE) { // ht or pet item failed or got protection

		if protectionInfo.GetType() == SCALE_TYPE { // if scale
			if item.Plus < uint8(protectionInfo.SellPrice) {
				item.Plus = 0
			} else {
				item.Plus -= uint8(protectionInfo.SellPrice)
			}
		} else {
			if item.Plus < stone.Plus {
				item.Plus = 0
			} else {
				item.Plus -= stone.Plus
			}
		}

		upgs := item.GetUpgrades()
		for i := int(item.Plus); i < len(upgs); i++ {
			item.SetUpgrade(i, 0)
		}

		r := HT_UPG_FAILED
		r.Insert(utils.IntToBytes(uint64(item.ItemID), 4, true), 9) // item id
		r.Insert(utils.IntToBytes(uint64(slotID), 2, true), 17)     // slot id
		r.Insert(item.GetUpgrades(), 19)                            // item upgrades
		r[34] = byte(item.SocketCount)                              // socket count
		r.Insert(item.GetSockets(), 35)                             // item sockets

		resp.Concat(r)
		logger.Log(logging.ACTION_UPGRADE_ITEM, c.ID, fmt.Sprintf("Item (%d) upgrade failed but not vanished", item.ID), c.UserID, c.Name)

	} else { // casual item failed so destroy it
		r := UPG_FAILED
		r.Insert(utils.IntToBytes(uint64(item.ItemID), 4, true), 9) // item id
		resp.Concat(r)

		itemsData, err := c.RemoveItem(int16(slotID))
		if err != nil {
			return nil, err
		}

		resp.Concat(itemsData)
		if luck != nil {
			luckInfo := Items[luck.ItemID]
			go logging.AddLogFile(4, c.Socket.User.ID+" idli kullanici ("+c.Name+") isimli karakterine ait ("+info.Name+") iteme artı basarken yaktı. Kullanılan taş sayısı:  "+fmt.Sprintf("%d", len(stones))+" Kullanılan luck: "+luckInfo.Name)
		} else {
			go logging.AddLogFile(4, c.Socket.User.ID+" idli kullanici ("+c.Name+") isimli karakterine ait ("+info.Name+") iteme artı basarken yaktı. Kullanılan taş sayısı:  "+fmt.Sprintf("%d", len(stones)))
		}
		logger.Log(logging.ACTION_UPGRADE_ITEM, c.ID, fmt.Sprintf("Item (%d) upgrade failed and destroyed", item.ID), c.UserID, c.Name)
	}

	for _, slot := range stoneSlots {
		resp.Concat(*c.DecrementItem(int16(slot), 1))
	}

	if luck != nil {
		resp.Concat(*c.DecrementItem(int16(luckSlot), 1))
	}

	if protection != nil {
		resp.Concat(*c.DecrementItem(int16(protectionSlot), 1))
	}

	err = item.Update()
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (c *Character) BSProduction(book *InventorySlot, materials []*InventorySlot, special *InventorySlot, prodSlot int16, bookSlot, specialSlot int16, materialSlots []int16, materialCounts []uint) ([]byte, error) {

	production := Productions[int(book.ItemID)]
	prodMaterials, err := production.GetMaterials()
	if err != nil {
		return nil, err
	}

	canProduce := true

	for i := 0; i < len(materials); i++ {
		if materials[i].Quantity < uint(prodMaterials[i].Count) || int(materials[i].ItemID) != prodMaterials[i].ID {
			canProduce = false
			break
		}
	}

	if prodMaterials[2].ID > 0 && (special.Quantity < uint(prodMaterials[2].Count) || int(special.ItemID) != prodMaterials[2].ID) {
		canProduce = false
	}

	cost := uint64(production.Cost)
	if cost > c.Gold || !canProduce {
		resp := utils.Packet{0xAA, 0x55, 0x04, 0x00, 0x54, 0x04, 0x07, 0x10, 0x55, 0xAA}
		return resp, nil
	}

	c.LootGold(-cost)
	luckRate := float64(1)
	if special != nil {
		specialInfo := Items[special.ItemID]
		luckRate = float64(specialInfo.SellPrice+100) / 100
	}

	resp := &utils.Packet{}
	seed := int(utils.RandInt(0, 1000))
	if float64(seed) < float64(production.Probability)*luckRate { // Success
		tmpProduction := &InventorySlot{ItemID: int64(production.Production), Quantity: 1}
		if production.Production == 99043003 || production.Production == 99033003 {
			tmpProduction.Quantity = 5000
		}
		resp, _, err = c.AddItem(tmpProduction, prodSlot, false)
		if err != nil {
			return nil, err
		} else if resp == nil {
			return nil, nil
		}

		resp.Concat(PRODUCTION_SUCCESS)
		logger.Log(logging.ACTION_PRODUCTION, c.ID, fmt.Sprintf("Production (%d) success", book.ItemID), c.UserID, c.Name)

	} else { // Failed
		resp.Concat(PRODUCTION_FAILED)
		resp.Concat(c.GetGold())
		logger.Log(logging.ACTION_PRODUCTION, c.ID, fmt.Sprintf("Production (%d) failed", book.ItemID), c.UserID, c.Name)
	}

	resp.Concat(*c.DecrementItem(int16(bookSlot), 1))

	for i := 0; i < len(materialSlots); i++ {
		resp.Concat(*c.DecrementItem(int16(materialSlots[i]), uint(materialCounts[i])))
	}

	if special != nil {
		resp.Concat(*c.DecrementItem(int16(specialSlot), 1))
	}

	return *resp, nil
}

func (c *Character) AdvancedFusion(items []*InventorySlot, special *InventorySlot, prodSlot int16) ([]byte, bool, error) {

	if len(items) < 3 {
		return nil, false, nil
	}

	fusion := Fusions[items[0].ItemID]
	seed := int(utils.RandInt(0, 1000))

	cost := uint64(fusion.Cost)
	if c.Gold < cost {
		return FUSION_FAILED, false, nil
	}

	if items[0].ItemID != fusion.Item1 || items[1].ItemID != fusion.Item2 || items[2].ItemID != fusion.Item3 {
		return FUSION_FAILED, false, nil
	}

	c.LootGold(-cost)
	rate := float64(fusion.Probability)
	if special != nil {
		info := Items[special.ItemID]
		rate *= float64(info.SellPrice+100) / 100
	}

	if float64(seed) < rate { // Success
		resp := utils.Packet{}
		itemData, _, err := c.AddItem(&InventorySlot{ItemID: fusion.Production, Quantity: 1}, prodSlot, false)
		if err != nil {
			return nil, false, err
		} else if itemData == nil {
			return nil, false, nil
		}

		resp.Concat(*itemData)
		resp.Concat(FUSION_SUCCESS)
		logger.Log(logging.ACTION_ADVANCED_FUSION, c.ID, fmt.Sprintf("Advanced fusion (%d) success", items[0].ItemID), c.UserID, c.Name)
		return resp, true, nil

	} else { // Failed
		resp := FUSION_FAILED
		resp.Concat(c.GetGold())
		logger.Log(logging.ACTION_ADVANCED_FUSION, c.ID, fmt.Sprintf("Advanced fusion (%d) failed", items[0].ItemID), c.UserID, c.Name)
		return resp, false, nil
	}
}

func (c *Character) Dismantle(item, special *InventorySlot) ([]byte, bool, error) {

	melting := Meltings[int(item.ItemID)]
	if melting == nil {
		return nil, false, nil
	}
	cost := uint64(melting.Cost)

	if c.Gold < cost {
		return nil, false, nil
	}

	meltedItems, err := melting.GetMeltedItems()
	if err != nil {
		return nil, false, err
	}

	itemCounts, err := melting.GetItemCounts()
	if err != nil {
		return nil, false, err
	}

	c.LootGold(-cost)

	info := Items[item.ItemID]

	profit := utils.RandFloat(1, melting.ProfitMultiplier) * float64(info.BuyPrice*2)
	c.LootGold(uint64(profit))

	resp := utils.Packet{}
	r := DISMANTLE_SUCCESS
	r.Insert(utils.IntToBytes(uint64(profit), 8, true), 9) // profit

	count, index := 0, 18
	for i := 0; i < 3; i++ {
		id := meltedItems[i]
		if id == 0 {
			continue
		}

		maxCount := int64(itemCounts[i])
		meltedCount := utils.RandInt(0, maxCount+1)
		if meltedCount == 0 {
			continue
		}

		count++
		r.Insert(utils.IntToBytes(uint64(id), 4, true), index) // melted item id
		index += 4

		r.Insert([]byte{0x00, 0xA2}, index)
		index += 2

		r.Insert(utils.IntToBytes(uint64(meltedCount), 2, true), index) // melted item count
		index += 2

		freeSlot, err := c.FindFreeSlot()
		if err != nil {
			return nil, false, err
		}

		r.Insert(utils.IntToBytes(uint64(freeSlot), 2, true), index) // free slot id
		index += 2

		r.Insert([]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}, index) // upgrades
		index += 34

		itemData, _, err := c.AddItem(&InventorySlot{ItemID: int64(id), Quantity: uint(meltedCount)}, freeSlot, false)
		if err != nil {
			return nil, false, err
		} else if itemData == nil {
			return nil, false, nil
		}

		resp.Concat(*itemData)
	}

	r[17] = byte(count)
	length := int16(44*count) + 14

	if melting.SpecialItem > 0 {
		seed := int(utils.RandInt(0, 1000))

		if seed < melting.SpecialProbability {

			freeSlot, err := c.FindFreeSlot()
			if err != nil {
				return nil, false, err
			}

			r.Insert([]byte{0x01}, index)
			index++

			r.Insert(utils.IntToBytes(uint64(melting.SpecialItem), 4, true), index) // special item id
			index += 4

			r.Insert([]byte{0x00, 0xA2, 0x01, 0x00}, index)
			index += 4

			r.Insert(utils.IntToBytes(uint64(freeSlot), 2, true), index) // free slot id
			index += 2

			r.Insert([]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}, index) // upgrades
			index += 34

			itemData, _, err := c.AddItem(&InventorySlot{ItemID: int64(melting.SpecialItem), Quantity: 1}, freeSlot, false)
			if err != nil {
				return nil, false, err
			} else if itemData == nil {
				return nil, false, nil
			}

			resp.Concat(*itemData)
			length += 45
		}
	}

	r.SetLength(length)
	resp.Concat(r)
	resp.Concat(c.GetGold())
	logger.Log(logging.ACTION_DISMANTLE, c.ID, fmt.Sprintf("Dismantle (%d) success with %d gold", item.ID, c.Gold), c.UserID, c.Name)
	return resp, true, nil
}

func (c *Character) Extraction(item, special *InventorySlot, itemSlot int16) ([]byte, bool, error) {

	info := Items[item.ItemID]
	code := int(item.GetUpgrades()[item.Plus-1])
	cost := uint64(info.SellPrice) * uint64(HaxCodes[code].ExtractionMultiplier) / 1000

	if c.Gold < cost {
		return nil, false, nil
	}

	c.LootGold(-cost)
	item.Plus--
	item.SetUpgrade(int(item.Plus), 0)

	resp := utils.Packet{}
	r := EXTRACTION_SUCCESS
	r.Insert(utils.IntToBytes(uint64(item.ItemID), 4, true), 9)    // item id
	r.Insert(utils.IntToBytes(uint64(item.Quantity), 2, true), 15) // item quantity
	r.Insert(utils.IntToBytes(uint64(itemSlot), 2, true), 17)      // item slot
	r.Insert(item.GetUpgrades(), 19)                               // item upgrades
	r[34] = byte(item.SocketCount)                                 // item socket count
	r.Insert(item.GetUpgrades(), 35)                               // item sockets

	count := 1          //int(utils.RandInt(1, 4))
	r[53] = byte(count) // stone count

	index, length := 54, int16(51)
	for i := 0; i < count; i++ {

		freeSlot, err := c.FindFreeSlot()
		if err != nil {
			return nil, false, err
		}

		id := int64(HaxCodes[code].ExtractedItem)
		itemData, _, err := c.AddItem(&InventorySlot{ItemID: id, Quantity: 1}, freeSlot, false)
		if err != nil {
			return nil, false, err
		} else if itemData == nil {
			return nil, false, nil
		}

		resp.Concat(*itemData)

		r.Insert(utils.IntToBytes(uint64(id), 4, true), index) // extracted item id
		index += 4

		r.Insert([]byte{0x00, 0xA2, 0x01, 0x00}, index)
		index += 4

		r.Insert(utils.IntToBytes(uint64(freeSlot), 2, true), index) // free slot id
		index += 2

		r.Insert([]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}, index) // upgrades
		index += 34

		length += 44
	}

	r.SetLength(length)
	resp.Concat(r)
	resp.Concat(c.GetGold())

	err := item.Update()
	if err != nil {
		return nil, false, err
	}

	logger.Log(logging.ACTION_EXTRACTION, c.ID, fmt.Sprintf("Extraction success for item (%d)", item.ID), c.UserID, c.Name)
	return resp, true, nil
}

func (c *Character) CreateSocket(item, special *InventorySlot, itemSlot, specialSlot int16) ([]byte, error) {

	info := Items[item.ItemID]

	cost := uint64(info.SellPrice * 164)
	if c.Gold < cost {
		return nil, nil
	}

	if item.SocketCount == 0 && special != nil {
		if special.ItemID == 17200186 || special.ItemID == 17502301 {
			fmt.Println("Socket init with no socket")
			resp := utils.Packet{0xAA, 0x55, 0x04, 0x00, 0x54, 0x16, 0x0A, 0xCF, 0x55, 0xAA}
			item.Update()
			return resp, nil
		}
	}

	if item.SocketCount > 0 && special != nil {
		if special.ItemID == 17200186 || special.ItemID == 17502301 {
			resp := c.DecrementItem(specialSlot, 1)
			resp.Concat(item.CreateSocket(itemSlot, 0))
			item.Update()
			return *resp, nil
		}
	}

	if item.SocketCount > 0 {
		resp := utils.Packet{0xAA, 0x55, 0x04, 0x00, 0x54, 0x16, 0x0B, 0xCF, 0x55, 0xAA}
		item.Update()
		return resp, nil
	}

	/*
		if item.SocketCount > 0 && special != nil && special.ItemID == 17200186 || special.ItemID == 17502301 { // socket init
			fmt.Println("init")
			resp := c.DecrementItem(specialSlot, 1)
			resp.Concat(item.CreateSocket(itemSlot, 0))
			return *resp, nil

		} else if item.SocketCount > 0 { // item already has socket
			fmt.Println("already has socket")
			resp := utils.Packet{0xAA, 0x55, 0x04, 0x00, 0x54, 0x16, 0x0B, 0xCF, 0x55, 0xAA}
			return resp, nil

		} else if item.SocketCount == 0 && special != nil && special.ItemID == 17200186 || special.ItemID == 17502301 { // socket init with no sockets
			fmt.Println("Socket init with no socket")
			resp := utils.Packet{0xAA, 0x55, 0x04, 0x00, 0x54, 0x16, 0x0A, 0xCF, 0x55, 0xAA}
			return resp, nil
		}
	*/

	seed := utils.RandInt(0, 1000)
	socketCount := int8(1)
	if seed >= 850 {
		socketCount = 4
	} else if seed >= 650 {
		socketCount = 3
	} else if seed >= 350 {
		socketCount = 2
	}

	c.LootGold(-cost)
	resp := utils.Packet{}
	if special != nil {
		if special.ItemID == 17200185 || special.ItemID == 17402830 || special.ItemID == 17501411 || special.ItemID == 18500113 { // +1 miled stone
			socketCount++

		} else if special.ItemID == 15710239 { // +2 miled stone
			socketCount += 2
			if socketCount > 5 {
				socketCount = 5
			}

		}

		resp.Concat(*c.DecrementItem(specialSlot, 1))
	}

	if special != nil {
		tmpInfo := Items[item.ItemID]
		go logging.AddLogFile(6, c.Socket.User.ID+" idli kullanici ("+c.Name+") isimli karakterine ait ("+info.Name+") iteme socket açtı socket sayısı : +"+strconv.Itoa(int(socketCount))+" Kullanılan item: "+tmpInfo.Name)
	} else {
		go logging.AddLogFile(6, c.Socket.User.ID+" idli kullanici ("+c.Name+") isimli karakterine ait ("+info.Name+") iteme socket açtı socket sayısı : +"+strconv.Itoa(int(socketCount))+" Kullanılan item: Yok")
	}

	item.SocketCount = socketCount
	item.Update()
	resp.Concat(item.CreateSocket(itemSlot, socketCount))
	resp.Concat(c.GetGold())
	return resp, nil
}

func (c *Character) UpgradeSocket(item, socket, special, edit *InventorySlot, itemSlot, socketSlot, specialSlot, editSlot int16, locks []bool) ([]byte, error) {

	info := Items[item.ItemID]
	cost := uint64(info.SellPrice * 164)
	if c.Gold < cost {
		return nil, nil
	}

	if c.TradeID != "" {
		return nil, nil
	}

	if item.SocketCount == 0 { // No socket on item
		resp := utils.Packet{0xAA, 0x55, 0x04, 0x00, 0x54, 0x16, 0x10, 0xCF, 0x55, 0xAA}
		return resp, nil
	}

	if socket.Plus < uint8(item.SocketCount) { // Insufficient socket
		resp := utils.Packet{0xAA, 0x55, 0x04, 0x00, 0x54, 0x17, 0x0D, 0xCF, 0x55, 0xAA}
		return resp, nil
	}

	//stabilize := special != nil && special.ItemID == 17200187 || special.ItemID == 17200570
	stabilize := false

	if special != nil {
		if special.ItemID == 17200187 || special.ItemID == 17200570 {
			stabilize = true
		}
	}

	if edit != nil {
		if edit.ItemID < 17503030 && edit.ItemID > 17503032 {
			return nil, nil
		}
	}

	upgradesArray := []byte{}

	tmpType := info.GetType()
	if tmpType == ARMOR_TYPE {
		upgradesArray = bytes.Join([][]byte{SocketArmorUpgrades, SocketAccUpgrades}, []byte{})
	} else if tmpType == WEAPON_TYPE {
		upgradesArray = bytes.Join([][]byte{SocketWeaponUpgrades, SocketAccUpgrades}, []byte{})
	} else {
		upgradesArray = bytes.Join([][]byte{SocketArmorUpgrades, SocketWeaponUpgrades, SocketAccUpgrades}, []byte{})
	}

	sockets := make([]byte, item.SocketCount)
	socks := item.GetSockets()
	for i := int8(0); i < item.SocketCount; i++ {
		if locks[i] {
			sockets[i] = socks[i]
			continue
		}

		seed := utils.RandInt(0, int64(len(upgradesArray)+1))
		code := upgradesArray[seed]
		if stabilize && code%5 > 0 {
			code++
		} else if !stabilize && code%5 == 0 {
			code--
		}

		if stabilize {
			if code == 74 || code == 75 {
				code = 73
			} else if code == 69 || code == 70 {
				code = 68
			} else if code == 59 || code == 60 {
				code = 58
			} else if code == 49 || code == 50 {
				code = 48
			}
		} else {
			if code == 73 || code == 74 || code == 75 {
				code = 72
			} else if code == 68 || code == 69 || code == 70 {
				code = 67
			} else if code == 58 || code == 59 || code == 60 {
				code = 57
			} else if code == 48 || code == 49 || code == 50 {
				code = 47
			}
		}

		sockets[i] = code
	}

	if special != nil {
		tmpInfo := Items[item.ItemID]
		go logging.AddLogFile(6, c.Socket.User.ID+" idli kullanici ("+c.Name+") isimli karakterine ait ("+info.Name+") iteme socket bastı socket arr : +"+string(item.GetSockets())+" Kullanılan item: "+tmpInfo.Name)
	} else {
		go logging.AddLogFile(6, c.Socket.User.ID+" idli kullanici ("+c.Name+") isimli karakterine ait ("+info.Name+") iteme socket açtı socket sayısı : +"+string(item.GetSockets())+" Kullanılan item: Yok")
	}

	c.LootGold(-cost)
	resp := utils.Packet{}
	resp.Concat(item.UpgradeSocket(itemSlot, sockets))
	resp.Concat(c.GetGold())
	resp.Concat(*c.DecrementItem(socketSlot, 1))

	if special != nil {
		resp.Concat(*c.DecrementItem(specialSlot, 1))
	}

	if edit != nil {
		resp.Concat(*c.DecrementItem(editSlot, 1))
	}

	return resp, nil
}

func (c *Character) HolyWaterUpgrade(item, holyWater *InventorySlot, itemSlot, holyWaterSlot int16) ([]byte, error) {

	itemInfo := Items[item.ItemID]
	hwInfo := Items[holyWater.ItemID]

	if (itemInfo.GetType() == WEAPON_TYPE && (hwInfo.HolyWaterUpg1 < 66 || hwInfo.HolyWaterUpg1 > 105)) ||
		(itemInfo.GetType() == ARMOR_TYPE && (hwInfo.HolyWaterUpg1 < 41 || hwInfo.HolyWaterUpg1 > 65)) ||
		(itemInfo.GetType() == ACC_TYPE && hwInfo.HolyWaterUpg1 > 40) { // Mismatch type

		resp := utils.Packet{0xAA, 0x55, 0x04, 0x00, 0x54, 0x10, 0x36, 0x11, 0x55, 0xAA}
		return resp, nil
	}

	if item.Plus == 0 {
		resp := utils.Packet{0xAA, 0x55, 0x04, 0x00, 0x54, 0x10, 0x37, 0x11, 0x55, 0xAA}
		return resp, nil
	}

	resp := utils.Packet{}
	seed, upgrade := int(utils.RandInt(0, 60)), 0
	if seed < hwInfo.HolyWaterRate1 {
		upgrade = hwInfo.HolyWaterUpg1
	} else if seed < hwInfo.HolyWaterRate2 {
		upgrade = hwInfo.HolyWaterUpg2
	} else if seed < hwInfo.HolyWaterRate3 {
		upgrade = hwInfo.HolyWaterUpg3
	} else {
		resp = HOLYWATER_FAILED
	}

	if upgrade > 0 {
		randSlot := utils.RandInt(0, int64(item.Plus))
		preUpgrade := item.GetUpgrades()[randSlot]
		item.SetUpgrade(int(randSlot), byte(upgrade))

		if preUpgrade == byte(upgrade) {
			resp = HOLYWATER_FAILED
		} else {
			resp = HOLYWATER_SUCCESS

			r := ITEM_UPGRADED
			r.Insert(utils.IntToBytes(uint64(item.ItemID), 4, true), 9) // item id
			r.Insert(utils.IntToBytes(uint64(itemSlot), 2, true), 17)   // slot id
			r.Insert(item.GetUpgrades(), 19)                            // item upgrades
			r[34] = byte(item.SocketCount)                              // socket count
			r.Insert(item.GetSockets(), 35)                             // item sockets
			resp.Concat(r)

			new := funk.Map(item.GetUpgrades()[:item.Plus], func(upg byte) string {
				return HaxCodes[int(upg)].Code
			}).([]string)

			old := make([]string, len(new))
			copy(old, new)
			old[randSlot] = HaxCodes[int(preUpgrade)].Code

			msg := fmt.Sprintf("[%s] has been upgraded from [%s] to [%s].", itemInfo.Name, strings.Join(old, ""), strings.Join(new, ""))
			msgData := messaging.InfoMessage(msg)
			resp.Concat(msgData)
		}
	}

	itemData, _ := c.RemoveItem(holyWaterSlot)
	resp.Concat(itemData)

	err := item.Update()
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (c *Character) RegisterItem(item *InventorySlot, price uint64, itemSlot int16) ([]byte, error) {

	if c.UsedConsig {
		return nil, nil
	}

	c.UsedConsig = true
	time.AfterFunc(time.Second, func() {
		c.UsedConsig = false
	})

	items, err := FindConsignmentItemsBySellerID(c.ID)
	if err != nil {
		return nil, err
	}

	if len(items) >= 10 {
		return nil, nil
	}

	commision := uint64(math.Min(float64(price/100), 50000000))
	if c.Gold < commision {
		return nil, nil
	}

	info, ok := Items[item.ItemID]
	if !ok {
		return nil, nil
	}

	// if !info.Tradable {
	// 	return nil, nil
	// }
	if info.Tradable == 2 {
		return nil, nil
	}

	consItem := &ConsignmentItem{
		ID:       item.ID,
		SellerID: c.ID,
		ItemName: info.Name,
		Quantity: int(item.Quantity),
		IsSold:   false,
		Price:    price,
	}

	if err := consItem.Create(); err != nil {
		return nil, err
	}

	go logging.AddLogFile(3, c.Socket.User.ID+" idli kullanici ("+c.Name+") isimli karakteri ile consa bir item kayıt etti. Item : ("+info.Name+") Fiyat: ("+strconv.Itoa(int(price))+") (CONSIG)")

	c.LootGold(-commision)
	resp := ITEM_REGISTERED
	resp.Insert(utils.IntToBytes(uint64(consItem.ID), 4, true), 9)  // consignment item id
	resp.Insert(utils.IntToBytes(uint64(item.ItemID), 4, true), 29) // item id

	if item.Pet != nil {
		resp[34] = byte(item.SocketCount)
	}

	resp.Insert(utils.IntToBytes(uint64(item.Quantity), 2, true), 35) // item count
	resp.Insert(item.GetUpgrades(), 37)                               // item upgrades

	if item.Pet != nil {
		resp[42] = 0 // item socket count
	} else {
		resp[42] = byte(item.SocketCount) // item socket count
	}

	resp.Insert(item.GetSockets(), 43) // item sockets

	newItem := NewSlot()
	*newItem = *item
	newItem.SlotID = -1
	newItem.Consignment = true
	newItem.Update()
	InventoryItems.Add(newItem.ID, newItem)

	*item = *NewSlot()
	resp.Concat(c.GetGold())
	resp.Concat(item.GetData(itemSlot))

	claimData, err := c.ClaimMenu()
	if err != nil {
		return nil, err
	}
	resp.Concat(claimData)

	return resp, nil
}

func (c *Character) ClaimMenu() ([]byte, error) {
	items, err := FindConsignmentItemsBySellerID(c.ID)
	if err != nil {
		return nil, err
	}

	resp := CLAIM_MENU
	resp.SetLength(int16(len(items)*0x6B + 6))
	resp.Insert(utils.IntToBytes(uint64(len(items)), 2, true), 8) // items count

	index := 10
	for _, item := range items {

		slot, err := FindInventorySlotByID(item.ID) // FIX: Buyer can destroy the item..
		if err != nil {
			continue
		}
		if slot == nil {
			slot = NewSlot()
			slot.ItemID = 17502455
			slot.Quantity = 1
		}

		info := Items[int64(slot.ItemID)]

		if item.IsSold {
			resp.Insert([]byte{0x01}, index)
		} else {
			resp.Insert([]byte{0x00}, index)
		}
		index++

		resp.Insert(utils.IntToBytes(uint64(item.ID), 4, true), index) // consignment item id
		index += 4

		resp.Insert([]byte{0x5E, 0x15, 0x01, 0x00}, index)
		index += 4

		resp.Insert([]byte(c.Name), index) // seller name
		index += len(c.Name)

		for j := len(c.Name); j < 20; j++ {
			resp.Insert([]byte{0x00}, index)
			index++
		}

		resp.Insert(utils.IntToBytes(item.Price, 8, true), index) // item price
		index += 8

		time := item.ExpiresAt.Time.Format("2006-01-02 15:04:05") // expires at
		resp.Insert([]byte(time), index)
		index += 19

		resp.Insert([]byte{0x00, 0x09, 0x00, 0x00, 0x00, 0x99, 0x31, 0xF5, 0x00}, index)
		index += 9

		resp.Insert(utils.IntToBytes(uint64(slot.ItemID), 4, true), index) // item id
		index += 4

		resp.Insert([]byte{0x00, 0xA1}, index)
		index += 2

		if info.GetType() == PET_TYPE {
			resp[index-1] = byte(slot.SocketCount)
		}

		resp.Insert(utils.IntToBytes(uint64(slot.Quantity), 2, true), index) // item count
		index += 2

		resp.Insert(slot.GetUpgrades(), index) // item upgrades
		index += 15

		resp.Insert([]byte{byte(slot.SocketCount)}, index) // socket count
		index++

		resp.Insert(slot.GetSockets(), index)
		index += 15

		resp.Insert([]byte{0x00, 0x00, 0x00}, index)
		index += 3

		if slot.Appearance != 0 {
			resp.Overwrite(utils.IntToBytes(uint64(slot.Appearance), 4, true), index-4) //16 volt
		}

	}

	return resp, nil
}

func (c *Character) BuyConsignmentItem(consignmentID int) ([]byte, error) {
	if c.UsedConsig {
		return nil, nil
	}

	c.UsedConsig = true
	time.AfterFunc(time.Second, func() {
		c.UsedConsig = false
	})
	consignmentItem, err := FindConsignmentItemByID(consignmentID)
	if err != nil || consignmentItem == nil || consignmentItem.IsSold {
		return nil, err
	}

	slots, err := c.InventorySlots()
	if err != nil {
		return nil, err
	}

	slot, err := FindInventorySlotByID(consignmentItem.ID)
	if err != nil {
		return nil, err
	}

	if c.Gold < consignmentItem.Price {
		return nil, nil
	}

	seller, err := FindCharacterByID(int(slot.CharacterID.Int64))
	if err != nil {
		return nil, err
	}

	if seller.ID == c.ID {
		return nil, nil
	}

	resp := CONSIGMENT_ITEM_BOUGHT
	resp.Insert(utils.IntToBytes(uint64(consignmentID), 4, true), 8) // consignment item id

	slotID, err := c.FindFreeSlot()
	if err != nil {
		return nil, nil
	}

	newItem := NewSlot()
	*newItem = *slot
	newItem.Consignment = false
	newItem.UserID = null.StringFrom(c.UserID)
	newItem.CharacterID = null.IntFrom(int64(c.ID))
	newItem.SlotID = slotID

	err = newItem.Update()
	if err != nil {
		return nil, err
	}

	*slots[slotID] = *newItem
	InventoryItems.Add(newItem.ID, slots[slotID])
	c.LootGold(-consignmentItem.Price)

	resp.Concat(newItem.GetData(slotID))
	resp.Concat(c.GetGold())

	s, ok := Sockets[seller.UserID]
	if ok {
		s.Write(CONSIGMENT_ITEM_SOLD)
	}

	go logging.AddLogFile(3, c.Socket.User.ID+" idli kullanici ("+c.Name+") isimli karakteri ile consdan bir item satın aldı Item : ("+strconv.Itoa(newItem.ID)+") Fiyat: ("+strconv.Itoa(int(consignmentItem.Price))+") Satıcı: ("+seller.Name+")("+seller.UserID+") (CONSIG)")

	logger.Log(logging.ACTION_BUY_CONS_ITEM, c.ID, fmt.Sprintf("Bought consignment item (%d) with %d gold from (%d)", newItem.ID, consignmentItem.Price, seller.ID), c.UserID, c.Name)
	consignmentItem.IsSold = true
	go consignmentItem.Update()
	return resp, nil
}

func (c *Character) ClaimConsignmentItem(consignmentID int, isCancel bool) ([]byte, error) {

	if c.UsedConsig {
		return nil, nil
	}

	c.UsedConsig = true
	time.AfterFunc(time.Second, func() {
		c.UsedConsig = false
	})

	consignmentItem, err := FindConsignmentItemByID(consignmentID)
	if err != nil || consignmentItem == nil {
		return nil, err
	}

	resp := CONSIGMENT_ITEM_CLAIMED
	resp.Insert(utils.IntToBytes(uint64(consignmentID), 4, true), 10) // consignment item id

	if isCancel {
		if consignmentItem.IsSold {
			return nil, nil
		}

		slots, err := c.InventorySlots()
		if err != nil {
			return nil, err
		}

		slotID, err := c.FindFreeSlot()
		if err != nil {
			return nil, err
		}

		slot, err := FindInventorySlotByID(consignmentItem.ID)
		if err != nil {
			return nil, err
		}

		newItem := NewSlot()
		*newItem = *slot
		newItem.Consignment = false
		newItem.SlotID = slotID

		err = newItem.Update()
		if err != nil {
			return nil, err
		}

		*slots[slotID] = *newItem
		InventoryItems.Add(newItem.ID, slots[slotID])

		resp.Concat(slot.GetData(slotID))

	} else {
		if !consignmentItem.IsSold {
			return nil, nil
		}

		logger.Log(logging.ACTION_BUY_CONS_ITEM, c.ID, fmt.Sprintf("Claimed consignment item (consid:%d) with %d gold", consignmentID, consignmentItem.Price), c.UserID, c.Name)

		c.LootGold(consignmentItem.Price)
		resp.Concat(c.GetGold())
	}

	s, _ := FindInventorySlotByID(consignmentItem.ID)
	if s != nil && !s.UserID.Valid && !s.CharacterID.Valid {
		s.Delete()
	}

	go consignmentItem.Delete()
	return resp, nil
}

func (c *Character) FindItemInUsed(itemIDs []int64) (bool, error) {
	slots, err := c.InventorySlots()
	if err != nil {
		return false, err
	}

	for index, slot := range slots {
		if ok, _ := utils.Contains(itemIDs, slot.ItemID); ok {
			if index >= 0x43 && index <= 0x132 {
				continue
			}

			infoItem := Items[slot.ItemID]
			if slots[index].InUse && infoItem.HtType != 21 {
				return true, nil
			}
		}
	}
	return false, nil
}

func (c *Character) UseConsumable(item *InventorySlot, slotID int16) ([]byte, error) {
	defer func() {
		if err := recover(); err != nil {
			log.Println("ConnectedIP: ", c.Socket.User.ConnectedIP)
			log.Println("ConnectingIP: ", c.Socket.User.ConnectingIP)
			log.Println("RemoteAddr: ", c.Socket.Conn.RemoteAddr().String())
			log.Println("CHR Name: ", c.Name)
			log.Println(err)
			log.Printf("%+v", string(dbg.Stack()))

			r := utils.Packet{}
			r.Concat(*c.DecrementItem(slotID, 0))
			c.Socket.Write(r)
		}
	}()

	if c == nil || c.Socket.User == nil {
		return nil, nil
	}
	stat := c.Socket.Stats
	if stat.HP <= 0 {
		return nil, nil
		//fmt.Println("Karakter ölü. ", slotID)
		//return *c.DecrementItem(slotID, 0), nil
	}

	info := Items[item.ItemID]
	if info == nil {
		return nil, nil
	} else if info.MinLevel > c.Level || (info.MaxLevel > 0 && info.MaxLevel < c.Level) {
		resp := utils.Packet{0xAA, 0x55, 0x04, 0x00, 0x59, 0x04, 0xF0, 0x03, 0x55, 0xAA} // inappropriate level
		return resp, nil
	}

	usedEarlier, _ := c.FindItemInUsed([]int64{item.ItemID})
	if usedEarlier {
		return []byte{0xAA, 0x55, 0x04, 0x00, 0x59, 0x04, 0xFA, 0x03, 0x55, 0xAA}, nil
	}

	resp := utils.Packet{}
	canUse := c.CanUse(info.CharacterType)
	switch info.GetType() {
	case AFFLICTION_TYPE:
		err := stat.Reset()
		if err != nil {
			return nil, err
		}

		statData, _ := c.GetStats()
		resp.Concat(statData)

	case CHARM_OF_RETURN_TYPE:
		c.Targets = []*Target{}
		c.PlayerTargets = []*PlayerTarget{}
		c.Selection = 0
		if c.IsinLastMan {
			c.IsinLastMan = false
			LastManMutex.Lock()
			delete(LastManCharacters, c.ID)
			LastManMutex.Unlock()
		}
		d := SavePoints[uint8(c.Map)]
		coordinate := ConvertPointToLocation(d.Point)
		resp.Concat(c.Teleport(coordinate))

		slots, err := c.InventorySlots()
		if err == nil {
			pet := slots[0x0A].Pet
			if pet != nil && pet.IsOnline {
				pet.IsOnline = false
				resp.Concat(DISMISS_PET)
				showpet, _ := c.ShowItems()
				resp.Concat(showpet)
				c.IsMounting = false
			}
		}

	case DEAD_SPIRIT_INCENSE_TYPE:
		slots, err := c.InventorySlots()
		if err != nil {
			return nil, err
		}

		pet := slots[0x0A].Pet
		if pet != nil && !pet.IsOnline && pet.HP <= 0 {
			pet.HP = pet.MaxHP / 10
			resp.Concat(c.GetPetStats())
			resp.Concat(c.TogglePet())
		} else {
			goto FALLBACK
		}

	case MOVEMENT_SCROLL_TYPE:
		mapID := int16(info.SellPrice)
		data, _ := c.ChangeMap(mapID, nil)
		resp.Concat(data)

	case BAG_EXPANSION_TYPE:
		buff, err := FindBuffByID(int(item.ItemID), c.ID)
		if err != nil {
			return nil, err
		} else if buff != nil {
			return []byte{0xAA, 0x55, 0x04, 0x00, 0x59, 0x04, 0xFA, 0x03, 0x55, 0xAA}, nil // FIX: Already Exists => Already have the same effect
		}

		buff = &Buff{ID: int(item.ItemID), CharacterID: c.ID, Name: info.Name, BagExpansion: true, StartedAt: c.Epoch, Duration: int64(info.Timer) * 60}
		err = buff.Create()
		if err != nil {
			return nil, err
		}

		resp = BAG_EXPANDED

	case FIRE_SPIRIT:
		buff, err := FindBuffByID(int(item.ItemID), c.ID)
		if err != nil {
			return nil, err
		} else if buff != nil {
			return []byte{0xAA, 0x55, 0x04, 0x00, 0x59, 0x04, 0xFA, 0x03, 0x55, 0xAA}, nil // FIX: Already Exists => Already have the same effect
		}

		buff, err = FindBuffByID(19000019, c.ID) // check for water spirit
		if err != nil {
			return nil, err
		} else if buff != nil {
			return []byte{0xAA, 0x55, 0x04, 0x00, 0x59, 0x04, 0xFA, 0x03, 0x55, 0xAA}, nil // FIX: Already Exists => Already have the same effect
		}

		buff = &Buff{ID: int(item.ItemID), CharacterID: c.ID, Name: info.Name, EXPMultiplier: 15, DropMultiplier: 2, DEFRate: 5, ArtsDEFRate: 5,
			ATKRate: 4, ArtsATKRate: 4, StartedAt: c.Epoch, Duration: 2592000}
		err = buff.Create()
		if err != nil {
			return nil, err
		}

		c.ExpMultiplier += 0.15
		c.DropMultiplier += 0.02
		itemData, _, _ := c.AddItem(&InventorySlot{ItemID: 17502645, Quantity: 1}, -1, false)
		resp.Concat(*itemData)

		data, _ := c.GetStats()
		resp.Concat(data)

	case WATER_SPIRIT:
		buff, err := FindBuffByID(int(item.ItemID), c.ID)
		if err != nil {
			return nil, err
		} else if buff != nil {
			return []byte{0xAA, 0x55, 0x04, 0x00, 0x59, 0x04, 0xFA, 0x03, 0x55, 0xAA}, nil // FIX: Already Exists => Already have the same effect
		}

		buff, err = FindBuffByID(19000018, c.ID) // check for fire spirit
		if err != nil {
			return nil, err
		} else if buff != nil {
			return []byte{0xAA, 0x55, 0x04, 0x00, 0x59, 0x04, 0xFA, 0x03, 0x55, 0xAA}, nil // FIX: Already Exists => Already have the same effect
		}

		buff = &Buff{ID: int(item.ItemID), CharacterID: c.ID, Name: info.Name, EXPMultiplier: 30, DropMultiplier: 5, DEFRate: 15, ArtsDEFRate: 15,
			ATKRate: 8, ArtsATKRate: 8, StartedAt: c.Epoch, Duration: 2592000}
		err = buff.Create()
		if err != nil {
			return nil, err
		}

		c.ExpMultiplier += 0.3
		c.DropMultiplier += 0.05
		itemData, _, _ := c.AddItem(&InventorySlot{ItemID: 17502646, Quantity: 1}, -1, false)
		resp.Concat(*itemData)

		data, _ := c.GetStats()
		resp.Concat(data)

	case FORTUNE_BOX_TYPE:

		c.InvMutex.Lock()
		defer c.InvMutex.Unlock()

		gambling := GamblingItems[int(item.ItemID)]
		if gambling == nil || gambling.Cost > c.Gold { // FIX Gambling null
			resp := utils.Packet{0xAA, 0x55, 0x04, 0x00, 0x59, 0x08, 0xF9, 0x03, 0x55, 0xAA} // not enough gold
			return resp, nil
		}

		slots, err := c.InventorySlots()
		if err != nil {
			return nil, err
		}

		if item.ItemID == 18500874 {
			_, mysteriousKey, err := c.FindItemInInventory(nil, 18500875)
			if err != nil {
				return nil, err
			} else if mysteriousKey == nil {
				resp := messaging.InfoMessage("You don't have mysterious key")
				return resp, nil
			}

			resp.Concat(*c.DecrementItem(mysteriousKey.SlotID, 1))
		}

		c.LootGold(-gambling.Cost)
		resp.Concat(c.GetGold())

		drop, ok := Drops[gambling.DropID]
		if drop == nil || !ok {
			goto FALLBACK
		}

		var itemID int
		for ok {
			index := 0
			seed := int(utils.RandInt(0, 1000))
			items := drop.GetItems()
			probabilities := drop.GetProbabilities()

			for _, prob := range probabilities {
				if float64(seed) > float64(prob) {
					index++
					continue
				}
				break
			}

			if index >= len(items) {
				break
			}

			itemID = items[index]
			drop, ok = Drops[itemID]
		}

		plus, quantity, upgs := uint8(0), uint(1), []byte{}
		rewardInfo := Items[int64(itemID)]
		if rewardInfo != nil {
			if rewardInfo.ID == 235 || rewardInfo.ID == 242 || rewardInfo.ID == 254 || rewardInfo.ID == 255 { // Socket-PP-Ghost Dagger-Dragon Scale
				var rates []int
				if rewardInfo.ID == 235 { // Socket
					rates = []int{320, 580, 790, 850, 1000}
				} else if rewardInfo.ID == 254 {
					rates = []int{320}
				} else if rewardInfo.ID == 242 {
					rates = []int{320}
				} else {
					rates = []int{400, 900, 1000}
				}

				seed := int(utils.RandInt(0, 1000))
				for ; seed > rates[plus]; plus++ {
				}
				plus++

				upgs = utils.CreateBytes(byte(rewardInfo.ID), int(plus), 15)
			} else if rewardInfo.GetType() == MARBLE_TYPE { // Marble
				rates := []int{200, 300, 500, 750, 950, 1000}
				seed := int(utils.RandInt(0, 1000))
				for i := 0; seed > rates[i]; i++ {
					itemID++
				}

				rewardInfo = Items[int64(itemID)]
			} else if funk.Contains(haxBoxes, item.ItemID) { // Hax Box
				seed := utils.RandInt(0, 1000)
				plus = uint8(sort.SearchInts(plusRates, int(seed)) + 1)

				if plus > 5 {
					plus = 5
				}

				upgradesArray := []byte{}
				rewardType := rewardInfo.GetType()
				if rewardType == WEAPON_TYPE {
					upgradesArray = WeaponUpgrades
				} else if rewardType == ARMOR_TYPE {
					upgradesArray = ArmorUpgrades
				} else if rewardType == ACC_TYPE {
					upgradesArray = AccUpgrades
				}

				index := utils.RandInt(0, int64(len(upgradesArray)))
				code := upgradesArray[index]
				if (code-1)%5 == 3 {
					code--
				} else if (code-1)%5 == 4 {
					code -= 2
				}

				upgs = utils.CreateBytes(byte(code), int(plus), 15)
			}

			if q, ok := rewardCounts[item.ItemID]; ok {
				quantity = q
			}

			if box, ok := rewardCounts2[item.ItemID]; ok {
				if q, ok := box[rewardInfo.ID]; ok {
					quantity = q
				}
			}

			item := &InventorySlot{ItemID: rewardInfo.ID, Plus: uint8(plus), Quantity: quantity}
			item.SetUpgrades(upgs)

			if rewardInfo.GetType() == PET_TYPE {
				petInfo := Pets[int64(rewardInfo.ID)]
				petExpInfo := PetExps[int16(petInfo.Level)]

				targetExps := []int{petExpInfo.ReqExpEvo1, petExpInfo.ReqExpEvo2, petExpInfo.ReqExpEvo3, petExpInfo.ReqExpHt}
				item.Pet = &PetSlot{
					Fullness: 100, Loyalty: 100,
					Exp:   uint64(targetExps[petInfo.Evolution-1]),
					HP:    petInfo.BaseHP,
					Level: byte(petInfo.Level),
					Name:  petInfo.Name,
					CHI:   petInfo.BaseChi,
				}
			}

			_, slot, err := c.AddItem(item, -1, true)
			if err != nil {
				return nil, err
			}

			if slot == -1 {
				return nil, nil
			}

			// 01.12.2023 // BUG FIX
			if slots[slot] == nil {
				return nil, nil
			}

			resp.Concat(messaging.InfoMessage(fmt.Sprintf("You have acquired %s", rewardInfo.Name)))

			resp.Concat(slots[slot].GetData(slot))
		}

	case NPC_SUMMONER_TYPE:
		if item.ItemID == 17502966 || item.ItemID == 17100004 || item.ItemID == 18500033 { // Tavern
			r := utils.Packet{0xAA, 0x55, 0x07, 0x00, 0x57, 0x03, 0x01, 0x06, 0x00, 0x00, 0x00, 0x55, 0xAA}
			resp.Concat(r)
		} else if item.ItemID == 17502967 || item.ItemID == 17100005 || item.ItemID == 18500034 { // Bank
			resp.Concat(c.BankItems())
		}

	case PASSIVE_SKILL_BOOK_TYPE:
		if info.CharacterType > 0 && !canUse { // invalid character type
			return INVALID_CHARACTER_TYPE, nil
		}

		skills, err := FindSkillsByID(c.ID)
		if err != nil {
			return nil, err
		}

		skillSlots, err := skills.GetSkills()
		if err != nil {
			return nil, err
		}

		i := -1
		if info.Name == "Wind Drift Arts" {
			i = 7
			if skillSlots.Slots[i].BookID > 0 {
				return SKILL_BOOK_EXISTS, nil
			}

		} else {
			for j := 5; j < 7; j++ {
				if skillSlots.Slots[j].BookID == 0 {
					i = j
					break
				} else if skillSlots.Slots[j].BookID == item.ItemID { // skill book exists
					return SKILL_BOOK_EXISTS, nil
				}
			}
		}

		if i == -1 {
			return NO_SLOTS_FOR_SKILL_BOOK, nil // FIX resp
		}

		set := &SkillSet{BookID: item.ItemID}
		set.Skills = append(set.Skills, &SkillTuple{SkillID: int(info.ID), Plus: 0})
		skillSlots.Slots[i] = set
		skills.SetSkills(skillSlots)

		go skills.Update()

		skillsData, err := skills.GetSkillsData()
		if err != nil {
			return nil, err
		}

		resp.Concat(skillsData)

	case PET_POTION_TYPE:
		slots, err := c.InventorySlots()
		if err != nil {
			return nil, err
		}

		petSlot := slots[0x0A]
		pet := petSlot.Pet

		if pet == nil || !pet.IsOnline {
			goto FALLBACK
		}

		pet.HP = int(math.Min(float64(pet.HP+info.HpRecovery), float64(pet.MaxHP)))
		pet.CHI = int(math.Min(float64(pet.CHI+info.ChiRecovery), float64(pet.MaxCHI)))
		pet.Fullness = byte(math.Min(float64(pet.Fullness+5), float64(100)))
		resp.Concat(c.GetPetStats())

	case POTION_TYPE:

		/*
			if c.UsedPotion {
				return item.GetData(slotID), nil
			}
		*/

		hpRec := info.HpRecovery
		chiRec := info.ChiRecovery
		if hpRec == 0 && chiRec == 0 {
			hpRec = 50000
			chiRec = 50000
		}

		if c.Map == 233 || c.Map == 230 {
			if hpRec != 0 {
				hpRec = 100
			}
		}

		stat.HP = int(math.Min(float64(stat.HP+hpRec), float64(stat.MaxHP)))
		stat.CHI = int(math.Min(float64(stat.CHI+chiRec), float64(stat.MaxCHI)))
		resp.Concat(c.GetHPandChi())

		/*
			go func() {
				c.UsedPotion = true
				time.AfterFunc(time.Millisecond*250, func() {
					c.UsedPotion = false
				})
			}()
		*/

	case FILLER_POTION_TYPE:
		hpRecovery, chiRecovery := math.Min(float64(stat.MaxHP-stat.HP), 50000), float64(0)
		if hpRecovery > float64(item.Quantity) {
			hpRecovery = float64(item.Quantity)
		} else {
			chiRecovery = math.Min(float64(stat.MaxCHI-stat.CHI), 50000)
			if chiRecovery+hpRecovery > float64(item.Quantity) {
				chiRecovery = float64(item.Quantity) - hpRecovery
			}
		}

		if c.Map == 233 {
			hpRecovery = 100
		}

		stat.HP = int(math.Min(float64(stat.HP)+hpRecovery, float64(stat.MaxHP)))
		stat.CHI = int(math.Min(float64(stat.CHI)+chiRecovery, float64(stat.MaxCHI)))
		resp.Concat(c.GetHPandChi())
		resp.Concat(*c.DecrementItem(slotID, uint(hpRecovery+chiRecovery)))
		resp.Concat(item.GetData(slotID))
		return resp, nil

	case SKILL_BOOK_TYPE:
		if info.CharacterType > 0 && !canUse { // invalid character type
			return INVALID_CHARACTER_TYPE, nil
		}

		skills, err := FindSkillsByID(c.ID)
		if err != nil {
			return nil, err
		}

		skillSlots, err := skills.GetSkills()
		if err != nil {
			return nil, err
		}

		i := -1
		for j := 0; j < 5; j++ {
			if skillSlots.Slots[j].BookID == 0 {
				i = j
				break
			} else if skillSlots.Slots[j].BookID == item.ItemID { // skill book exists
				return SKILL_BOOK_EXISTS, nil
			}
		}

		if i == -1 {
			return NO_SLOTS_FOR_SKILL_BOOK, nil // FIX resp
		}

		skillInfos := SkillInfosByBook[item.ItemID]
		set := &SkillSet{BookID: item.ItemID}
		c := 0
		for i := 1; i <= 24; i++ { // there should be 24 skills with empty ones

			if len(skillInfos) <= c {
				set.Skills = append(set.Skills, &SkillTuple{SkillID: 0, Plus: 0})
			} else if si := skillInfos[c]; si.Slot == i {
				tuple := &SkillTuple{SkillID: si.ID, Plus: 0}
				set.Skills = append(set.Skills, tuple)
				c++
			} else {
				set.Skills = append(set.Skills, &SkillTuple{SkillID: 0, Plus: 0})
			}
		}

		if info.MinLevel < 100 {
			divtuple := &DivineTuple{DivineID: 0, DivinePlus: 0}
			div2tuple := &DivineTuple{DivineID: 1, DivinePlus: 0}
			div3tuple := &DivineTuple{DivineID: 2, DivinePlus: 0}
			set.DivinePoints = append(set.DivinePoints, divtuple, div2tuple, div3tuple)
		}

		skillSlots.Slots[i] = set
		skills.SetSkills(skillSlots)

		go skills.Update()

		skillsData, err := skills.GetSkillsData()
		if err != nil {
			return nil, err
		}
		resp.Concat(skillsData)
	case TRANSFORMATION_TYPE:
		if item.ItemID == 15400005 {
			if c.Level < 101 {
				c.Socket.Write(messaging.InfoMessage("Divine item"))
				return nil, nil
			}

			skillSlots, err := c.Socket.Skills.GetSkills()
			if err != nil {
				return nil, err
			}

			skillPoints := 0
			for i := range skillSlots.Slots {
				if skillSlots.Slots[i].BookID == 100030013 || skillSlots.Slots[i].BookID == 100030015 || skillSlots.Slots[i].BookID == 100030014 || skillSlots.Slots[i].BookID == 100030016 {
					skillTuple := skillSlots.Slots[i].Skills
					for i := range skillTuple {
						skillPoints += skillTuple[i].Plus
					}
					skillSlots.Slots[i] = &SkillSet{}
				}

				if skillSlots.Slots[i].BookID == 100031001 || skillSlots.Slots[i].BookID == 100031003 || skillSlots.Slots[i].BookID == 100031002 || skillSlots.Slots[i].BookID == 100031004 {
					skillTuple := skillSlots.Slots[i].Skills
					for i := range skillTuple {
						for b := 1; b <= skillTuple[i].Plus; b++ {
							switch b {
							case 1:
								skillPoints += 1
							case 2:
								skillPoints += 1
							case 3:
								skillPoints += 1
							case 4:
								skillPoints += 2
							case 5:
								skillPoints += 2
							case 6:
								skillPoints += 4
							case 7:
								skillPoints += 6
							case 8:
								skillPoints += 9
							case 9:
								skillPoints += 19
							case 10:
								skillPoints += 21
							case 11:
								skillPoints += 31
							case 12:
								skillPoints += 47
							}
						}

					}
					skillSlots.Slots[i] = &SkillSet{}
				}
			}

			c.Class = 0

			c.Socket.Skills.SetSkills(skillSlots)
			c.Socket.Skills.SkillPoints += skillPoints
			c.Socket.Skills.Update()
			c.Update()
			c.Socket.User.Update()

			item.Delete()
			stData, _ := c.GetStats()
			resp.Concat(stData)
			resp.Concat(*c.DecrementItem(slotID, 1))

			time.AfterFunc(time.Duration(1*time.Second), func() {
				CharacterSelect := utils.Packet{0xAA, 0x55, 0x04, 0x00, 0x01, 0x05, 0x0A, 0x00, 0x55, 0xAA}
				CHARACTER_MENU := utils.Packet{0xAA, 0x55, 0x03, 0x00, 0x09, 0x09, 0x00, 0x55, 0xAA}
				resp := CHARACTER_MENU
				resp.Concat(CharacterSelect)
				//c.Socket.Conn.Write(resp)
				c.Socket.Write(resp)
			})
		}

		if item.ItemID == 15400002 {
			skillPoint := 0
			for i := range EXPs {
				if int(EXPs[i].Level) <= c.Level {
					skillPoint += EXPs[i].SkillPoints
				}
			}
			c.Class = 0
			c.Socket.Skills.SkillPoints = skillPoint
			c.Socket.Skills.Delete()
			c.Socket.Skills.Create(c)
			c.Socket.Skills.Update()
			c.Update()
			c.Socket.User.Update()

			item.Delete()
			stData, _ := c.GetStats()
			resp.Concat(stData)
			resp.Concat(*c.DecrementItem(slotID, 1))

			time.AfterFunc(time.Duration(1*time.Second), func() {
				CharacterSelect := utils.Packet{0xAA, 0x55, 0x04, 0x00, 0x01, 0x05, 0x0A, 0x00, 0x55, 0xAA}
				CHARACTER_MENU := utils.Packet{0xAA, 0x55, 0x03, 0x00, 0x09, 0x09, 0x00, 0x55, 0xAA}
				resp := CHARACTER_MENU
				resp.Concat(CharacterSelect)
				//c.Socket.Conn.Write(resp)
				c.Socket.Write(resp)
			})
		}

		return nil, nil

	case ESOTERIC_POTION_TYPE:
		if item == nil {
			goto FALLBACK
		}
		c.Injury = 0 // reset injury
		c.Update()

		resp = c.GetHPandChi()
		stat, _ := c.GetStats()
		resp.Concat(stat)

	case WRAPPER_BOX_TYPE:
		c.InvMutex.Lock()
		defer c.InvMutex.Unlock()

		if item.ItemID == 13000015 {
			c.AidTime += 7200 //aid 2h
			item.Delete()
			stData, _ := c.GetStats()
			resp.Concat(stData)
			resp.Concat(*c.DecrementItem(slotID, 1))
			return resp, nil
		}

		if item.ItemID == 13000074 {
			c.AidTime += 14400 //aid 4h
			item.Delete()
			stData, _ := c.GetStats()
			resp.Concat(stData)
			resp.Concat(*c.DecrementItem(slotID, 1))
			return resp, nil
		}

		if item.ItemID == 13000172 {
			c.AidTime += 21600 //aid 6h
			item.Delete()
			stData, _ := c.GetStats()
			resp.Concat(stData)
			resp.Concat(*c.DecrementItem(slotID, 1))
			return resp, nil
		}

		if item.ItemID == 13000173 {
			c.AidTime += 28800 //aid 8h
			item.Delete()
			stData, _ := c.GetStats()
			resp.Concat(stData)
			resp.Concat(*c.DecrementItem(slotID, 1))
			return resp, nil
		}

		if item.ItemID == 13000011 {
			c.AidTime += 86400 //aid 1d
			item.Delete()
			stData, _ := c.GetStats()
			resp.Concat(stData)
			resp.Concat(*c.DecrementItem(slotID, 1))
			return resp, nil
		}

		if item.ItemID == 18500032 {
			c.AidTime += 7200 //aid 2h tavern
			item.Delete()
			stData, _ := c.GetStats()
			resp.Concat(stData)
			resp.Concat(*c.DecrementItem(slotID, 1))
			return resp, nil
		}

		if item.ItemID == 18500114 {
			c.AidTime += 7200 // Aid tonic 2h coin
			item.Delete()
			stData, _ := c.GetStats()
			resp.Concat(stData)
			resp.Concat(*c.DecrementItem(slotID, 1))
			return resp, nil
		}

		if item.ItemID == 18500210 {
			c.AidTime += 28800 // Aid tonic 8h starter box
			item.Delete()
			stData, _ := c.GetStats()
			resp.Concat(stData)
			resp.Concat(*c.DecrementItem(slotID, 1))
			return resp, nil
		}

		if item.ItemID == 18500141 {
			c.Socket.User.NCash += 1000 // ncash 1k
			go c.Socket.User.Update()

			resp.Concat(*c.DecrementItem(slotID, 1))
			return resp, nil
		}

		if item.ItemID == 18500142 {
			c.Socket.User.NCash += 2000 // ncash 2k
			go c.Socket.User.Update()

			resp.Concat(*c.DecrementItem(slotID, 1))
			return resp, nil
		}

		if item.ItemID == 18500143 {
			c.Socket.User.NCash += 3000 // ncash 3k
			go c.Socket.User.Update()

			resp.Concat(*c.DecrementItem(slotID, 1))
			return resp, nil
		}

		if item.ItemID == 18500144 {
			c.Socket.User.NCash += 5000 // ncash 5k
			go c.Socket.User.Update()

			resp.Concat(*c.DecrementItem(slotID, 1))
			return resp, nil
		}

		if item.ItemID == 18500145 {
			c.Socket.User.NCash += 10000 // ncash 10k
			go c.Socket.User.Update()

			resp.Concat(*c.DecrementItem(slotID, 1))
			return resp, nil
		}

		if item.ItemID == 18500146 {
			c.Socket.User.NCash += 50000 // ncash 50k
			go c.Socket.User.Update()

			resp.Concat(*c.DecrementItem(slotID, 1))
			return resp, nil
		}

		if item.ItemID == 18500147 {
			c.Socket.User.NCash += 100000 // ncash 100k
			go c.Socket.User.Update()

			resp.Concat(*c.DecrementItem(slotID, 1))
			return resp, nil
		}

		gambling := GamblingItems[int(item.ItemID)]
		d := Drops[gambling.DropID]
		items := d.GetItems()

		if c.Gold >= gambling.Cost {
			c.Gold -= gambling.Cost
		} else {
			resp := utils.Packet{0xAA, 0x55, 0x04, 0x00, 0x59, 0x08, 0xF9, 0x03, 0x55, 0xAA} // not enough gold
			return resp, nil
		}

		if gambling.DropID == 200001 {
			if c.Type == FEMALE_BLADE || c.Type == FEMALE_ROD || c.Type == DUAL_BLADE {
				for i := range items {
					if items[i] == 99032002 || items[i] == 99033002 || items[i] == 99050001 || items[i] == 99011101 {
						items[i] = 0
					}
				}
			} else {
				for i := range items {
					if items[i] == 99042002 || items[i] == 99043002 || items[i] == 99051001 || items[i] == 99021101 {
						items[i] = 0
					}
				}
			}
		}

		if item.ItemID == 18500003 || item.ItemID == 18500031 || item.ItemID == 92000001 || item.ItemID == 18500045 || item.ItemID == 18500044 || item.ItemID == 18500067 ||
			item.ItemID == 18500068 || item.ItemID == 18500174 || item.ItemID == 18500175 {
			min := 0
			max := len(items)
			randomVal := rand.Intn(max-min) + min
			tmpItem := items[randomVal]
			items = []int{}
			items = append(items, tmpItem)
		}

		_, err := c.FindFreeSlots(len(items))
		if err != nil {
			goto FALLBACK
		}

		slots, err := c.InventorySlots()
		if err != nil {
			goto FALLBACK
		}

		i := 0
		for _, itemID := range items {
			if itemID == 0 {
				continue
			}
			info := Items[int64(itemID)]
			reward := NewSlot()
			reward.ItemID = int64(itemID)
			reward.Quantity = 1

			i++

			itemType := info.GetType()
			if info.Timer > 0 && itemType != BAG_EXPANSION_TYPE {
				reward.Quantity = uint(info.Timer)
			} else if q, ok := rewardCounts[item.ItemID]; ok {
				reward.Quantity = q
			} else if itemType == FILLER_POTION_TYPE {
				reward.Quantity = uint(info.SellPrice)
			}

			plus, upgs := uint8(0), []byte{}
			if info.ID == 235 || info.ID == 242 || info.ID == 254 || info.ID == 255 { // Socket-PP-Ghost Dagger-Dragon Scale
				var rates []int
				if info.ID == 235 { // Socket ayarı
					rates = []int{300, 550, 750, 900, 1000}
				} else if info.ID == 254 {
					rates = []int{320}
				} else if info.ID == 242 {
					rates = []int{320}
				} else {
					rates = []int{300, 550, 750, 900, 1000}
				}

				/*
					seed := int(utils.RandInt(0, 1000))
					for ; seed > rates[plus]; plus++ {
					}
					plus++
				*/

				if len(rates) == 1 {
					plus = 1
				} else {
					seed := int(utils.RandInt(0, 1000))
					for ; seed > rates[plus]; plus++ {
					}
					plus++
				}

				/*
					for i := 0; i < len(rates); i++ {
						if seed > rates[i] {
							plus++
						}
					}
				*/

				upgs = utils.CreateBytes(byte(info.ID), int(plus), 15)
			}

			reward.Plus = plus
			reward.SetUpgrades(upgs)

			_, slot, _ := c.AddItem(reward, -1, true)
			resp.Concat(slots[slot].GetData(slot))
		}

		resp.Concat(c.GetGold())

	case HOLY_WATER_TYPE:
		goto FALLBACK
	case FORM_TYPE:
		info, ok := Items[int64(item.ItemID)]
		if !ok || item.Activated != c.Morphed {
			goto FALLBACK
		}

		if c.Map == 233 || c.Map == 74 {
			return nil, nil
		}

		item.Activated = !item.Activated
		item.InUse = !item.InUse
		c.Morphed = item.Activated
		c.MorphedNPCID = info.NPCID

		if item.Activated {

			r := FORM_ACTIVATED
			r.Insert(utils.IntToBytes(uint64(info.NPCID), 4, true), 5) // form npc id
			resp.Concat(r)

			characters, err := c.GetNearbyCharacters()
			if err != nil {
				log.Println(err)
			}

			for _, chars := range characters {
				delete(chars.OnSight.Players, c.ID)
			}

		} else {
			c.MorphedNPCID = 0
			resp.Concat(FORM_DEACTIVATED)

			characters, err := c.GetNearbyCharacters()
			if err != nil {
				log.Println(err)
				//return
			}

			//test := utils.Packet{0xAA, 0x55, 0x05, 0x00, 0x21, 0x02, 0x00, 0x55, 0xAA}
			for _, chars := range characters {
				delete(chars.OnSight.Players, c.ID)
			}
		}

		resp.Concat(item.GetData(item.SlotID))

		go item.Update()

		statData, err := c.GetStats()
		if err != nil {
			return nil, err
		}

		resp.Concat(statData)
		goto FALLBACK

	default:

		if info.Timer > 0 {
			/*
				if c.UsedConsumable {
					resp.Concat(item.GetData(slotID))
					return resp, nil
				}
			*/

			itemHave, err := c.FindItemInUsed([]int64{item.ItemID})
			if err != nil {
				return nil, err
			}

			if itemHave {
				return []byte{0xAA, 0x55, 0x04, 0x00, 0x59, 0x04, 0xFA, 0x03, 0x55, 0xAA}, nil // FIX: Already Exists => Already have the same effect
			}

			if item.ItemID == 100080002 || item.ItemID == 100080008 {
				if c != nil && !c.DetectionMode {
					c.DetectionMode = true

					time.AfterFunc(time.Minute*time.Duration(item.Quantity), func() {
						if c != nil {
							item.Delete()
							c.UsedConsumables.ItemMutex.Lock()
							delete(c.UsedConsumables.Items, 100080002)
							c.UsedConsumables.ItemMutex.Unlock()
							c.DetectionMode = false

							statData, _ := c.GetStats()
							c.Socket.Write(statData)

							p := &nats.CastPacket{CastNear: true, CharacterID: c.ID, Data: c.GetHPandChi()}
							p.Cast()
						}
					})

					statData, _ := c.GetStats()
					c.Socket.Write(statData)

					p := &nats.CastPacket{CastNear: true, CharacterID: c.ID, Data: c.GetHPandChi()}
					p.Cast()
				}
			}

			/*

				if item.ItemID == 17300006 {
					if !item.Activated && !item.InUse {
						if c.EyeballOfDragon {
							return []byte{0xAA, 0x55, 0x04, 0x00, 0x59, 0x04, 0xFA, 0x03, 0x55, 0xAA}, nil // FIX: Already Exists => Already have the same effect
						}
					}
				}

			*/

			c.UsedConsumables.ItemMutex.Lock()

			if !item.Activated && !item.InUse {
				if c.UsedConsumables.Items[item.ItemID] {
					return []byte{0xAA, 0x55, 0x04, 0x00, 0x59, 0x04, 0xFA, 0x03, 0x55, 0xAA}, nil // FIX: Already Exists => Already have the same effect
				}
			}

			item.Activated = !item.Activated
			item.InUse = !item.InUse
			resp.Concat(item.GetData(slotID))

			if item.Activated && item.InUse {
				c.UsedConsumables.Items[item.ItemID] = true
			} else {
				delete(c.UsedConsumables.Items, item.ItemID)
			}

			c.UsedConsumables.ItemMutex.Unlock()

			/*
				if item.ItemID == 17300006 {
					if item.Activated && item.InUse {
						c.EyeballOfDragon = true
					} else {
						c.EyeballOfDragon = false
					}
				}



					c.UsedConsumable = true

					go func() {
						time.AfterFunc(time.Millisecond*300, func() {
							c.UsedConsumable = false
						})
					}()
			*/

			statsData, _ := c.GetStats()
			resp.Concat(statsData)
			goto FALLBACK
		} else {
			if item.ItemID == 13370163 {

				chrs, err := FindCharactersByUserID(c.Socket.User.ID)
				if err != nil {
					fmt.Println(err)
					return nil, nil
				}

				for _, chr := range chrs {
					if chr.GuildID > 0 {
						chr.GuildID = -1
						chr.Update()
					}

					if chr.Faction == 1 {
						chr.Faction = 2
					} else if chr.Faction == 2 {
						chr.Faction = 1
					}
					chr.Update()
				}
				item.Delete()

				CharacterSelect := utils.Packet{0xAA, 0x55, 0x04, 0x00, 0x01, 0x05, 0x0A, 0x00, 0x55, 0xAA}
				CHARACTER_MENU := utils.Packet{0xAA, 0x55, 0x03, 0x00, 0x09, 0x09, 0x00, 0x55, 0xAA}
				resp.Concat(CHARACTER_MENU)
				resp.Concat(CharacterSelect)
			}
			goto FALLBACK
		}
	}

	resp.Concat(*c.DecrementItem(slotID, 1))
	return resp, nil

FALLBACK:
	resp.Concat(*c.DecrementItem(slotID, 0))
	return resp, nil
}

func (c *Character) CanUse(t int) bool {
	if c.Type == 0x32 && (t == 0x32 || t == 0x01 || t == 0x03) { // MALE BEAST
		return true
	} else if c.Type == 0x33 && (t == 0x33 || t == 0x02 || t == 0x03) { // FEMALE BEAST
		return true
	} else if c.Type == 0x34 && (t == 0x34 || t == 0x01) { // Monk
		return true
	} else if c.Type == 0x35 && (t == 0x35 || t == 0x37 || t == 0x01) { //MALE_BLADE
		return true
	} else if c.Type == 0x36 && (t == 0x36 || t == 0x37 || t == 0x02) { //FEMALE_BLADE
		return true
	} else if c.Type == 0x38 && (t == 0x38 || t == 0x3A || t == 0x01) { //AXE
		return true
	} else if c.Type == 0x39 && (t == 0x39 || t == 0x3A || t == 0x02) { //FEMALE_ROD
		return true
	} else if c.Type == 0x3B && (t == 0x3B || t == 0x02) { //DUAL_BLADE
		return true
	} else if c.Type == 0x3C && (t == 0x3C || t == 0x01 || t == 0x03 || t == 0x0A) { // DIVINE MALE BEAST
		return true
	} else if c.Type == 0x3D && (t == 0x3D || t == 0x02 || t == 0x03 || t == 0x0A) { // DIVINE FEMALE BEAST
		return true
	} else if c.Type == 0x3E && (t == 0x3E || t == 0x01 || t == 0x34 || t == 0x0A) { //DIVINE MONK
		return true
	} else if c.Type == 0x3F && (t == 0x3F || t == 0x41 || t == 0x01 || t == 0x35 || t == 0x37 || t == 0x0A) { //DIVINE MALE_BLADE
		return true
	} else if c.Type == 0x40 && (t == 0x40 || t == 0x41 || t == 0x02 || t == 0x36 || t == 0x37 || t == 0x0A) { //DIVINE FEMALE_BLADE
		return true
	} else if c.Type == 0x42 && (t == 0x42 || t == 0x44 || t == 0x01 || t == 0x38 || t == 0x3A || t == 0x0A) { //DIVINE MALE_AXE
		return true
	} else if c.Type == 0x43 && (t == 0x43 || t == 0x44 || t == 0x02 || t == 0x39 || t == 0x3A || t == 0x0A) { //DIVINE FEMALE_ROD
		return true
	} else if c.Type == 0x45 && (t == 0x45 || t == 0x02 || t == 0x3B || t == 0x0A) { //DIVINE Dual Sword
		return true
	} else if c.Type == 0x46 && (t == 0x46 || t == 0x01 || t == 0x03 || t == 0x0A) { // DARK LORD MALE BEAST
		return true
	} else if c.Type == 0x47 && (t == 0x47 || t == 0x02 || t == 0x03 || t == 0x0A) { // DARK LORD FEMALE BEAST
		return true
	} else if c.Type == 0x48 && (t == 0x48 || t == 0x01 || t == 0x3E || t == 0x34 || t == 0x14) { //DARK LORD MONK
		return true
	} else if c.Type == 0x49 && (t == 0x49 || t == 0x4B || t == 0x01 || t == 0x35 || t == 0x37 || t == 0x41 || t == 0x3F || t == 0x14) { //DARK LORD MALE_BLADE
		return true
	} else if c.Type == 0x4A && (t == 0x4A || t == 0x4B || t == 0x02 || t == 0x36 || t == 0x37 || t == 0x40 || t == 0x41 || t == 0x14) { //DARK LORD FEMALE_BLADE
		return true
	} else if c.Type == 0x4C && (t == 0x4C || t == 0x4E || t == 0x01 || t == 0x38 || t == 0x3A || t == 0x42 || t == 0x44 || t == 0x14) { //DARK LORD MALE_AXE
		return true
	} else if c.Type == 0x4D && (t == 0x4D || t == 0x4E || t == 0x02 || t == 0x39 || t == 0x3A || t == 0x43 || t == 0x44 || t == 0x14) { //DARK LORD FEMALE_ROD
		return true
	} else if c.Type == 0x4F && (t == 0x4F || t == 0x02 || t == 0x45 || t == 0x3B) { //DARK LORD Dual Sword
		return true
	} else if t == 0x00 || t == 0x20 { //All character Type
		return true
	}

	return false
}

func (c *Character) UpgradeSkill(slotIndex, skillIndex byte) ([]byte, error) {
	skills, err := FindSkillsByID(c.ID)
	if err != nil {
		return nil, err
	}

	skillSlots, err := skills.GetSkills()
	if err != nil {
		return nil, err
	}

	set := skillSlots.Slots[slotIndex]
	skill := set.Skills[skillIndex]

	info := SkillInfos[skill.SkillID]
	if int8(skill.Plus) >= info.MaxPlus {
		return nil, nil
	}

	requiredSP := 1
	if info.ID >= 28000 && info.ID <= 28007 { // 2nd job passives (non-divine)
		requiredSP = SkillPTS["sjp"][skill.Plus]
	} else if info.ID >= 29000 && info.ID <= 29007 { // 2nd job passives (divine)
		requiredSP = SkillPTS["dsjp"][skill.Plus]
	} else if info.ID >= 20193 && info.ID <= 20217 { // 3nd job passives (darkness)
		requiredSP = SkillPTS["dsjp"][skill.Plus]
	}

	if skills.SkillPoints < requiredSP {
		return nil, nil
	}

	skills.SkillPoints -= requiredSP
	skill.Plus++
	resp := SKILL_UPGRADED
	resp[8] = slotIndex
	resp[9] = skillIndex
	resp.Insert(utils.IntToBytes(uint64(skill.SkillID), 4, true), 10) // skill id
	resp[14] = byte(skill.Plus)

	skills.SetSkills(skillSlots)
	skills.Update()

	if info.Passive {
		statData, err := c.GetStats()
		if err == nil {
			resp.Concat(statData)
		}
	}

	resp.Concat(c.GetExpAndSkillPts())
	return resp, nil
}

func (c *Character) DowngradeSkill(slotIndex, skillIndex byte) ([]byte, error) {
	skills, err := FindSkillsByID(c.ID)
	if err != nil {
		return nil, err
	}

	skillSlots, err := skills.GetSkills()
	if err != nil {
		return nil, err
	}

	set := skillSlots.Slots[slotIndex]
	skill := set.Skills[skillIndex]

	info := SkillInfos[skill.SkillID]
	if int8(skill.Plus) <= 0 {
		return nil, nil
	}

	requiredSP := 1
	if info.ID >= 28000 && info.ID <= 28007 && skill.Plus > 0 { // 2nd job passives (non-divine)
		requiredSP = SkillPTS["sjp"][skill.Plus-1]
	} else if info.ID >= 29000 && info.ID <= 29007 && skill.Plus > 0 { // 2nd job passives (divine)
		requiredSP = SkillPTS["dsjp"][skill.Plus-1]
	} else if info.ID >= 20193 && info.ID <= 20217 { // 3nd job passives (darkness)
		requiredSP = SkillPTS["dsjp"][skill.Plus]
	}

	skills.SkillPoints += requiredSP
	skill.Plus--
	resp := SKILL_DOWNGRADED
	resp[8] = slotIndex
	resp[9] = skillIndex
	resp.Insert(utils.IntToBytes(uint64(skill.SkillID), 4, true), 10) // skill id
	resp[14] = byte(skill.Plus)
	resp.Insert([]byte{0, 0, 0}, 15) //

	skills.SetSkills(skillSlots)
	skills.Update()

	if info.Passive {
		statData, err := c.GetStats()
		if err == nil {
			resp.Concat(statData)
		}
	}

	resp.Concat(c.GetExpAndSkillPts())
	return resp, nil
}

func (c *Character) DivineUpgradeSkills(skillIndex, slot int, bookID int64) ([]byte, error) {
	skills, err := FindSkillsByID(c.ID)
	if err != nil {
		return nil, err
	}

	skillSlots, err := skills.GetSkills()
	if err != nil {
		return nil, err
	}
	resp := utils.Packet{}
	//divineID := 0
	bonusPlus := 0
	usedPoints := 0
	for _, skill := range skillSlots.Slots {
		if skill.BookID == bookID {
			if len(skill.DivinePoints) == 0 {
				divtuple := &DivineTuple{DivineID: 0, DivinePlus: 0}
				div2tuple := &DivineTuple{DivineID: 1, DivinePlus: 0}
				div3tuple := &DivineTuple{DivineID: 2, DivinePlus: 0}
				skill.DivinePoints = append(skill.DivinePoints, divtuple, div2tuple, div3tuple)
				skills.SetSkills(skillSlots)
				skills.Update()
			}
			for _, point := range skill.DivinePoints {
				usedPoints += point.DivinePlus
				//if point.DivineID == slot {
				if usedPoints >= 10 {
					return nil, nil
				}
				//	divineID = point.DivineID
				if point.DivineID == slot {
					bonusPlus = point.DivinePlus
				}
			}
			skill.DivinePoints[slot].DivinePlus++
		}
	}
	bonusPlus++
	resp = DIVINE_SKILL_BOOk
	resp[8] = byte(skillIndex)
	index := 9
	resp.Insert([]byte{byte(slot)}, index) // divine id
	index++
	resp.Insert(utils.IntToBytes(uint64(bookID), 4, true), index) // book id
	index += 4
	resp.Insert([]byte{byte(bonusPlus)}, index) // divine plus
	index++
	skills.SetSkills(skillSlots)
	skills.Update()
	return resp, nil
}

func (c *Character) RemoveSkill(slotIndex byte, bookID int64) ([]byte, error) {
	skills, err := FindSkillsByID(c.ID)
	if err != nil {
		return nil, err
	}

	skillSlots, err := skills.GetSkills()
	if err != nil {
		return nil, err
	}

	set := skillSlots.Slots[slotIndex]
	if set.BookID != bookID {
		return nil, fmt.Errorf("RemoveSkill: skill book not found")
	}

	skillSlots.Slots[slotIndex] = &SkillSet{}
	skills.SetSkills(skillSlots)
	skills.Update()

	resp := SKILL_REMOVED
	resp[8] = slotIndex
	resp.Insert(utils.IntToBytes(uint64(bookID), 4, true), 9) // book id

	return resp, nil
}

func (c *Character) UpgradePassiveSkill(slotIndex, skillIndex byte) ([]byte, error) {
	skills, err := FindSkillsByID(c.ID)
	if err != nil {
		return nil, err
	}

	skillSlots, err := skills.GetSkills()
	if err != nil {
		return nil, err
	}

	set := skillSlots.Slots[skillIndex]
	if len(set.Skills) == 0 || set.Skills[0].Plus >= 12 {
		return nil, nil
	}

	if skillIndex == 5 || skillIndex == 6 { // 1st job skill
		requiredSP := SkillPTS["fjp"][set.Skills[0].Plus]
		if skills.SkillPoints < requiredSP {
			return nil, nil
		}

		skills.SkillPoints -= requiredSP

	} else if skillIndex == 7 { // running
		requiredSP := SkillPTS["wd"][set.Skills[0].Plus]
		if skills.SkillPoints < requiredSP {
			return nil, nil
		}

		skills.SkillPoints -= requiredSP
		c.RunningSpeed = 10.0 + (float64(set.Skills[0].Plus) * 0.5) //c.RunningSpeed = 10.0 + (float64(set.Skills[0].Plus) * 0.2)
	}

	set.Skills[0].Plus++

	skills.SetSkills(skillSlots)
	skills.Update()

	resp := PASSIVE_SKILL_UGRADED
	resp[8] = slotIndex
	resp[9] = byte(set.Skills[0].Plus)

	statData, err := c.GetStats()
	if err != nil {
		return nil, err
	}

	resp.Concat(statData)
	resp.Concat(c.GetExpAndSkillPts())
	return resp, nil
}

func (c *Character) DowngradePassiveSkill(slotIndex, skillIndex byte) ([]byte, error) {
	skills, err := FindSkillsByID(c.ID)
	if err != nil {
		return nil, err
	}

	skillSlots, err := skills.GetSkills()
	if err != nil {
		return nil, err
	}

	set := skillSlots.Slots[skillIndex]
	if len(set.Skills) == 0 || set.Skills[0].Plus <= 0 {
		return nil, nil
	}

	if skillIndex == 5 && set.Skills[0].Plus > 0 { // 1st job skill
		requiredSP := SkillPTS["fjp"][set.Skills[0].Plus-1]

		skills.SkillPoints += requiredSP

	} else if skillIndex == 7 && set.Skills[0].Plus > 0 { // running
		requiredSP := SkillPTS["wd"][set.Skills[0].Plus]

		skills.SkillPoints += requiredSP
		c.RunningSpeed = 10.0 + (float64(set.Skills[0].Plus-1) * 0.2)
	}

	set.Skills[0].Plus--

	skills.SetSkills(skillSlots)
	skills.Update()

	resp := PASSIVE_SKILL_UGRADED
	resp[8] = slotIndex
	resp[9] = byte(set.Skills[0].Plus)

	statData, err := c.GetStats()
	if err != nil {
		return nil, err
	}
	log.Println("----")

	resp.Concat(statData)
	resp.Concat(c.GetExpAndSkillPts())
	return resp, nil
}

func (c *Character) RemovePassiveSkill(slotIndex, skillIndex byte, bookID int64) ([]byte, error) {
	skills, err := FindSkillsByID(c.ID)
	if err != nil {
		return nil, err
	}

	skillSlots, err := skills.GetSkills()
	if err != nil {
		return nil, err
	}

	set := skillSlots.Slots[skillIndex]
	if set.BookID != bookID {
		return nil, fmt.Errorf("RemovePassiveSkill: skill book not found")
	}

	skillSlots.Slots[skillIndex] = &SkillSet{}
	skills.SetSkills(skillSlots)
	skills.Update()

	resp := PASSIVE_SKILL_REMOVED
	resp.Insert(utils.IntToBytes(uint64(bookID), 4, true), 8) // book id
	resp[12] = slotIndex

	return resp, nil
}

func (c *Character) CastSkill(attackCounter, skillID, targetID int, cX, cY, cZ float64) ([]byte, error) {

	if c.Socket.Stats.HP <= 0 {
		return nil, nil
	}

	slots, err := c.InventorySlots()
	if err != nil {
		return nil, err
	}
	petSlot := slots[0x0A]
	pet := petSlot.Pet
	petInfo, ok := Pets[petSlot.ItemID]
	if pet != nil && ok && pet.IsOnline && !petInfo.Combat {
		return nil, nil
	}
	stat := c.Socket.Stats
	user := c.Socket.User
	skills := c.Socket.Skills

	canCast := false
	skillInfo := SkillInfos[skillID]
	weapon := slots[c.WeaponSlot]
	if weapon.ItemID == 0 { // there are some skills which can be casted without weapon such as monk skills
		if c.Type == MONK || c.Type == DIVINE_MONK {
			canCast = true
		}
	} else {
		weaponInfo := Items[weapon.ItemID]
		canCast = weaponInfo.CanUse(skillInfo.Type)

		if !canCast {
			if c.WeaponSlot == 3 {
				c.WeaponSlot = 4
			} else if c.WeaponSlot == 4 {
				c.WeaponSlot = 3
			}

			weapon2 := slots[c.WeaponSlot]
			if weapon2 != nil && weapon2.ItemID != 0 {
				weapon2Info := Items[weapon2.ItemID]
				canCast = weapon2Info.CanUse(skillInfo.Type)

				if canCast {
					itemsData, err := c.ShowItems()
					if err != nil {
						return nil, err
					}

					p := nats.CastPacket{CastNear: true, CharacterID: c.ID, Type: nats.SHOW_ITEMS, Data: itemsData}
					if err = p.Cast(); err != nil {
						return nil, err
					}

					statData, err := c.GetStats()
					if err != nil {
						return nil, err
					}

					resp := utils.Packet{}
					resp.Concat(itemsData)
					resp.Concat(statData)
					c.Socket.Write(resp)
				}
			}
		}
	}

	if !canCast {
		return nil, nil
	}

	plus, err := skills.GetPlus(skillID)
	if err != nil {
		return nil, err
	}
	skillSlots, err := c.Socket.Skills.GetSkills()
	if err != nil {
		return nil, err
	}
	plusCooldown := 0
	plusChiCost := 0
	divinePlus := 0
	for _, slot := range skillSlots.Slots {
		if slot.BookID == skillInfo.BookID {
			for _, points := range slot.DivinePoints {
				if points.DivineID == 0 && points.DivinePlus > 0 {
					divinePlus = points.DivinePlus
					plusChiCost = 50
				}
				if points.DivinePlus == 2 && points.DivinePlus > 0 {
					plusCooldown = 100
				}
			}
		}
	}
	t := c.SkillHistory.Get(skillID)
	if t != nil {
		castedAt := t.(time.Time)
		cooldown := time.Duration(skillInfo.Cooldown*100) * time.Millisecond
		cooldown -= time.Duration(plusCooldown * divinePlus) //plusCooldown * divinePlus
		if time.Now().Sub(castedAt) < cooldown {
			return nil, nil
		}
	}
	c.SkillHistory.Add(skillID, time.Now())

	addChiCost := float64(skillInfo.AdditionalChi*int(plus)) * 2.2 / 3 // some bad words here
	chiCost := skillInfo.BaseChi + int(addChiCost) - (plusChiCost * divinePlus)
	if stat.CHI < chiCost {
		return nil, nil
	}

	stat.CHI -= chiCost
	resp := utils.Packet{}
	if target := skillInfo.Target; target == 0 || target == 2 { // buff skill
		character := c
		if target == 2 {
			ch := FindCharacterByPseudoID(c.Socket.User.ConnectedServer, uint16(c.Selection))
			if ch != nil {
				character = ch
			} else {
				if skillID == 41002 || skillID == 41300 {
					goto COMBAT
				}
			}
		}

		/*petSlot := slots[0x0A]
		pet := petSlot.Pet
		if pet == nil || petSlot.ItemID == 0 || !pet.IsOnline {

		}*/

		/*{
		if skillInfo.ID == 41602 {
			st := character.Socket.Stats
			st.HP += 2500 * int(plus)
			if st.HP > st.MaxHP {
				st.HP = st.MaxHP

		}*/
		if skillInfo.InfectionID == 0 {
			if skillInfo.ID == 41700 {
				st := character.Socket.Stats
				st.HP += 1250 * int(plus)
				if st.HP > st.MaxHP {
					st.HP = st.MaxHP
				}
			}
			/*if skillInfo.ID == 41603 {
				buff, err := FindBuffByID(61, character.ID)
				if err == nil && buff != nil {
					buff.Delete()
				}
				buff, err = FindBuffByID(251, character.ID)
				if err == nil && buff != nil {
					buff.Delete()
				}
				buff, err = FindBuffByID(71, character.ID)
				if err == nil && buff != nil {
					buff.Delete()
				}
			}*/

			character.Socket.Stats.Calculate()
			character.Update()

			itemsData, err := character.ShowItems()
			if err != nil {
				return nil, err
			}

			statData, err := character.GetStats()
			if err != nil {
				return nil, err
			}

			character.HandleBuffs()

			resp := utils.Packet{}
			resp.Concat(itemsData)
			resp.Concat(statData)
			resp.Concat(character.GetHPandChi())
			character.Socket.Write(resp)
			goto COMBAT
		}

		infection := BuffInfections[skillInfo.InfectionID]
		duration := (skillInfo.BaseTime + skillInfo.AdditionalTime*int(plus)) / 10

		if skillInfo.InfectionID == 92 {
			if c == character {
				return nil, nil
			}

			if !c.CanAttack(character) {
				return nil, nil
			}

			seed := utils.RandInt(0, 1000)
			if seed > 0 {
				buff, err := FindBuffByID(92, character.ID)
				if err == nil && buff == nil {
					buffinfo := BuffInfections[92]
					buff = &Buff{ID: int(92), CharacterID: character.ID, Name: buffinfo.Name, BagExpansion: false, StartedAt: character.Epoch, Duration: int64(duration)}

					err := buff.Create()
					if err != nil {
						fmt.Println("Hata buff oluşturulmadı ", err)
					}

					character.Update()
					character.Socket.Stats.Calculate()

					itemsData, err := character.ShowItems()
					if err != nil {
						return nil, err
					}

					statData, err := character.GetStats()
					if err != nil {
						return nil, err
					}

					character.HandleBuffs()

					resp := utils.Packet{}
					resp.Concat(itemsData)
					resp.Concat(statData)
					resp.Concat(character.GetHPandChi())
					character.Socket.Write(resp)
				}
			}

			goto COMBAT
		}

		if skillInfo.InfectionID == 246 {
			if c == character {
				return nil, nil
			}

			if !c.CanAttack(character) {
				return nil, nil
			}

			seed := utils.RandInt(0, 1000)
			if seed > 0 {
				buff, err := FindBuffByID(246, character.ID)
				if err == nil && buff == nil {
					buffinfo := BuffInfections[246]
					buff = &Buff{ID: int(246), CharacterID: character.ID, Name: buffinfo.Name, BagExpansion: false, StartedAt: character.Epoch, Duration: int64(duration)}

					err := buff.Create()
					if err != nil {
						fmt.Println("Hata buff oluşturulmadı ", err)
					}

					character.Update()
					character.Socket.Stats.Calculate()

					itemsData, err := character.ShowItems()
					if err != nil {
						return nil, err
					}

					statData, err := character.GetStats()
					if err != nil {
						return nil, err
					}

					character.HandleBuffs()

					resp := utils.Packet{}
					resp.Concat(itemsData)
					resp.Concat(statData)
					resp.Concat(character.GetHPandChi())
					character.Socket.Write(resp)
				}
			}

			goto COMBAT
		}

		expire := true
		if skillInfo.InfectionID != 0 && duration == 0 {
			expire = false
		}

		buff, err := FindBuffByID(infection.ID, character.ID)
		if err != nil {
			return nil, err
		} else if buff != nil {
			if buff.ID == 41002 || buff.ID == 41500 || buff.ID == 41300 {
				return nil, err
			}
			buff = &Buff{ID: infection.ID, CharacterID: character.ID, StartedAt: character.Epoch, Duration: int64(duration), Name: skillInfo.Name,
				ATK: infection.BaseATK + infection.AdditionalATK*int(plus), ArtsATK: infection.BaseArtsATK + infection.AdditionalArtsATK*int(plus),
				ArtsDEF: infection.ArtsDEF + infection.AdditionalArtsDEF*int(plus), ConfusionDEF: infection.ConfusionDef + infection.AdditionalConfusionDef*int(plus),
				DEF: infection.BaseDef + infection.AdditionalDEF*int(plus), DEX: infection.DEX + infection.AdditionalDEX*int(plus), HPRecoveryRate: infection.HPRecoveryRate + infection.AdditionalHPRecovery*int(plus), INT: infection.INT + infection.AdditionalINT*int(plus),
				MaxHP: infection.MaxHP + infection.AdditionalHP*int(plus), ParalysisDEF: infection.ParalysisDef + infection.AdditionalParalysisDef*int(plus), PoisonDEF: infection.ParalysisDef + infection.AdditionalPoisonDef*int(plus), STR: infection.STR + infection.AdditionalSTR*int(plus),
				Accuracy: infection.Accuracy + infection.AdditionalAccuracy*int(plus), Dodge: infection.DodgeRate + infection.AdditionalDodgeRate*int(plus), RunningSpeed: infection.MovSpeed + infection.AdditionalMovSpeed*float64(plus), SkillPlus: int(plus), CanExpire: expire, Water: infection.Water + infection.AdditionalWater*int(plus), Fire: infection.Fire + infection.AdditionalFire*int(plus)}
			buff.Update()
		} else if buff == nil {
			//c.HandleBuffs()
			if !infection.IsPercent {
				buff = &Buff{ID: infection.ID, CharacterID: character.ID, StartedAt: character.Epoch, Duration: int64(duration), Name: skillInfo.Name,
					ATK: infection.BaseATK + infection.AdditionalATK*int(plus), ArtsATK: infection.BaseArtsATK + infection.AdditionalArtsATK*int(plus),
					ArtsDEF: infection.ArtsDEF + infection.AdditionalArtsDEF*int(plus), ConfusionDEF: infection.ConfusionDef + infection.AdditionalConfusionDef*int(plus),
					DEF: infection.BaseDef + infection.AdditionalDEF*int(plus), DEX: infection.DEX + infection.AdditionalDEX*int(plus), HPRecoveryRate: infection.HPRecoveryRate + infection.AdditionalHPRecovery*int(plus), INT: infection.INT + infection.AdditionalINT*int(plus),
					MaxHP: infection.MaxHP + infection.AdditionalHP*int(plus), ParalysisDEF: infection.ParalysisDef + infection.AdditionalParalysisDef*int(plus), PoisonDEF: infection.ParalysisDef + infection.AdditionalPoisonDef*int(plus), STR: infection.STR + infection.AdditionalSTR*int(plus),
					Accuracy: infection.Accuracy + infection.AdditionalAccuracy*int(plus), Dodge: infection.DodgeRate + infection.AdditionalDodgeRate*int(plus), RunningSpeed: infection.MovSpeed + infection.AdditionalMovSpeed*float64(plus), SkillPlus: int(plus), CanExpire: expire, Water: infection.Water + infection.AdditionalWater*int(plus), Fire: infection.Fire + infection.AdditionalFire*int(plus)}
			} else {
				percentArtsDEF := int(float64(character.Socket.Stats.ArtsDEF) * (float64(infection.ArtsDEF+infection.AdditionalArtsDEF*int(plus)) / 1000))
				percentDEF := int(float64(character.Socket.Stats.DEF) * (float64(infection.BaseDef+infection.AdditionalDEF*int(plus)) / 1000))
				percentATK := int(float64(character.Socket.Stats.MinATK) * (float64(infection.BaseATK+infection.AdditionalATK*int(plus)) / 1000))
				percentArtsATK := int(float64(character.Socket.Stats.MinArtsATK) * (float64(infection.BaseArtsATK+infection.AdditionalArtsATK*int(plus)) / 1000))
				buff = &Buff{ID: infection.ID, CharacterID: character.ID, StartedAt: character.Epoch, Duration: int64(duration), Name: skillInfo.Name,
					ATK: percentATK, ArtsATK: percentArtsATK,
					ArtsDEF: percentArtsDEF, ConfusionDEF: infection.ConfusionDef + infection.AdditionalConfusionDef*int(plus),
					DEF: percentDEF, DEX: infection.DEX + infection.AdditionalDEX*int(plus), HPRecoveryRate: infection.HPRecoveryRate + infection.AdditionalHPRecovery*int(plus), INT: infection.INT + infection.AdditionalINT*int(plus),
					MaxHP: infection.MaxHP + infection.AdditionalHP*int(plus), ParalysisDEF: infection.ParalysisDef + infection.AdditionalParalysisDef*int(plus), PoisonDEF: infection.ParalysisDef + infection.AdditionalPoisonDef*int(plus), STR: infection.STR + infection.AdditionalSTR*int(plus),
					Accuracy: infection.Accuracy + infection.AdditionalAccuracy*int(plus), Dodge: infection.DodgeRate + infection.AdditionalDodgeRate*int(plus), SkillPlus: int(plus), CanExpire: expire, Wind: infection.Wind + infection.AdditionalWind*int(plus), Water: infection.Water + infection.AdditionalWater*int(plus), Fire: infection.Fire + infection.AdditionalFire*int(plus)}
			}
			buff.Create()
		}

		if buff.ID == 241 || buff.ID == 244 || buff.ID == 50 || buff.ID == 138 || buff.ID == 139 { // invisibility
			time.AfterFunc(time.Second*1, func() {
				if character != nil {
					character.Invisible = true
				}
				time.AfterFunc(time.Second*50, func() {
					if character != nil {
						character.Invisible = false
					}
				})
			})
		} else if buff.ID == 242 || buff.ID == 245 || buff.ID == 105 || buff.ID == 142 || buff.ID == 244 { // detection arts
			character.DetectionMode = true
		}

		statData, _ := character.GetStats()
		character.Socket.Write(statData)

		p := &nats.CastPacket{CastNear: true, CharacterID: character.ID, Data: character.GetHPandChi()}
		p.Cast()
	} else { // combat skill
		goto COMBAT
	}

COMBAT:
	target := GetFromRegister(user.ConnectedServer, c.Map, uint16(targetID))
	if skillInfo.PassiveType == 34 {
		teleport := c.Teleport(ConvertPointToLocation(fmt.Sprintf("%.1f,%.1f", cX, cY)))
		c.Socket.Write(teleport)
	}
	if ai, ok := target.(*AI); ok { // attacked to ai
		if skillID == 41201 || skillID == 41301 { // howl of tame
			c.TamingAI = ai
			c.SkillHistory.Delete(skillID)
			goto OUT
		}
		if skillID == 41002 || skillID == 41300 {
			goto OUT
		}
		if skillInfo.PassiveType == 14 {
			st := c.Socket.Stats
			st.HP += 25 + 50*int(plus)
			c.Socket.Write(c.GetHPandChi())
			goto OUT
		}
		pos := NPCPos[ai.PosID]
		if pos.Attackable { // target is attackable
			castLocation := ConvertPointToLocation(c.Coordinate)
			if skillInfo.AreaCenter == 1 || skillInfo.AreaCenter == 2 {
				castLocation = ConvertPointToLocation(ai.Coordinate)
			}
			skillSlots, err := c.Socket.Skills.GetSkills()
			if err != nil {
				return nil, err
			}
			plusRange := 0.0
			divinePlus := 0
			plusDamage := 0
			for _, slot := range skillSlots.Slots {
				if slot.BookID == skillInfo.BookID {
					for _, points := range slot.DivinePoints {
						if points.DivineID == 2 && points.DivinePlus > 0 {
							divinePlus = points.DivinePlus
							plusRange = 0.5
						}
						if points.DivineID == 1 && points.DivinePlus > 0 {
							divinePlus = points.DivinePlus
							plusDamage = 250
						}
					}
				}
			}
			castRange := skillInfo.BaseRadius + skillInfo.AdditionalRadius*float64(plus+1) + (float64(plusRange) * float64(divinePlus))
			candidates := AIsByMap[ai.Server][ai.Map]

			candidates = funk.Filter(candidates, func(cand *AI) bool {
				nPos := NPCPos[cand.PosID]
				if nPos == nil {
					return false
				}

				aiCoordinate := ConvertPointToLocation(cand.Coordinate)
				return (cand.PseudoID == ai.PseudoID || (utils.CalculateDistance(aiCoordinate, castLocation) < castRange)) && cand.HP > 0 && nPos.Attackable
			}).([]*AI)

			if skillInfo.InfectionID != 0 && skillInfo.Target == 1 {
				//c.DealBuffInfection(ai, nil, skillID)
			}
			for _, mob := range candidates {
				npcPos := NPCPos[mob.PosID]
				if npcPos == nil {
					return nil, nil
				}

				npc := NPCs[npcPos.NPCID]
				if npc == nil {
					return nil, nil
				}

				dmg, _ := c.CalculateDamage(mob, true)
				dmg += plusDamage * divinePlus

				if skillID == 41002 || skillID == 41102 || skillID == 41300 {
					dmg = 3
				}

				// Master extra attack
				if skillInfo.BookID == 16100033 || skillInfo.BookID == 16100034 || skillInfo.BookID == 16100035 || skillInfo.BookID == 16100036 || skillInfo.BookID == 16100037 || skillInfo.BookID == 16100038 {
					masterPercent := (10 / 100.0) * float32(dmg)
					masterSkillInc := int(masterPercent)
					dmg += masterSkillInc
				}

				// Divine extra attack
				if skillInfo.BookID == 16100101 || skillInfo.BookID == 16100103 || skillInfo.BookID == 16100107 || skillInfo.BookID == 16100109 || skillInfo.BookID == 16100111 || skillInfo.BookID == 16100113 || skillInfo.BookID == 16100115 {
					divinePercen := (20 / 100.0) * float32(dmg)
					divineSkillInc := int(divinePercen)
					dmg += divineSkillInc
				}

				if npc.ID == 9999991 || npc.ID == 9999992 || npc.ID == 9999993 || npc.ID == 9999994 || npc.ID == 9999995 || npc.ID == 9999996 || npc.ID == 9999997 || npc.ID == 9999998 || npc.ID == 423308 || npc.ID == 423310 || npc.ID == 423312 || npc.ID == 423314 || npc.ID == 423316 || npc.ID == 18600047 || npc.ID == 18600048 || npc.ID == 18600049 || npc.ID == 18600050 || npc.ID == 18600051 || npc.ID == 43301 || npc.ID == 43302 || npc.ID == 43403 || npc.ID == 43402 || npc.ID == 43401 || npc.ID == 18600078 {
					dmg = 100
				}
				if npc.ID == 18600053 {
					if c.Level < 160 {
						dmg = 0
					}
				}
				if npc.ID == 18600054 {
					if c.Level > 100 {
						dmg = 0
					}
				}

				c.Targets = append(c.Targets, &Target{Damage: dmg, AI: mob, Skill: true})
			}
		} else { // target is not attackable
			if funk.Contains(miningSkills, skillID) { // mining skill
				c.Targets = []*Target{{Damage: 10, AI: ai, Skill: true}}
			}
		}
	} else {
		if c.IsActive && !skillInfo.Passive {
			enemy := FindCharacterByPseudoID(user.ConnectedServer, uint16(targetID))
			if enemy != nil && enemy.IsActive && skillInfo.PassiveType != 14 && enemy != c && c.CanAttack(enemy) && enemy.Socket.Stats.HP > 0 {
				dmg, _ := c.CalculateDamageToPlayer(enemy, true)
				if skillID == 41002 || skillID == 41102 || skillID == 41701 || skillID == 41300 || skillID == 26706 || skillID == 41105 {
					dmg = 3
				}
				if skillInfo.BookID == 16100033 || skillInfo.BookID == 16100034 || skillInfo.BookID == 16100035 || skillInfo.BookID == 16100036 || skillInfo.BookID == 16100037 || skillInfo.BookID == 16100038 {
					masterPercent := (10 / 100.0) * float32(dmg)
					masterSkillInc := int(masterPercent)
					dmg += masterSkillInc
				}

				if skillInfo.BookID == 16100101 || skillInfo.BookID == 16100103 || skillInfo.BookID == 16100107 || skillInfo.BookID == 16100109 || skillInfo.BookID == 16100111 || skillInfo.BookID == 16100113 || skillInfo.BookID == 16100115 {
					divinePercen := (20 / 100.0) * float32(dmg)
					divineSkillInc := int(divinePercen)
					dmg += divineSkillInc
				}
				c.PlayerTargets = append(c.PlayerTargets, &PlayerTarget{Damage: dmg, Enemy: enemy, Skill: true})
				candidates := c.OnSight.Players
				for _, candidate := range candidates {
					enemyPseudoID := candidate.(uint16)
					enemyCandidate := FindCharacterByPseudoID(c.Socket.User.ConnectedServer, enemyPseudoID)
					if enemyCandidate == nil {
						continue
					}
					if enemy == enemyCandidate || enemyCandidate == c || !c.CanAttack(enemyCandidate) {
						continue
					}
					enemyCoord := ConvertPointToLocation(enemy.Coordinate)
					candidateCoord := ConvertPointToLocation(enemyCandidate.Coordinate)

					plusRange := 0.0
					divinePlus := 0
					plusDamage := 0
					for _, slot := range skillSlots.Slots {
						if slot.BookID == skillInfo.BookID {
							for _, points := range slot.DivinePoints {
								if points.DivineID == 2 && points.DivinePlus > 0 {
									divinePlus = points.DivinePlus
									plusRange = 0.5
								}
								if points.DivineID == 1 && points.DivinePlus > 0 {
									divinePlus = points.DivinePlus
									plusDamage = 250
								}
							}
						}
					}

					distance := utils.CalculateDistance(enemyCoord, candidateCoord)
					castRange := skillInfo.BaseRadius + skillInfo.AdditionalRadius*float64(plus+1) + (float64(plusRange) * float64(divinePlus))
					if distance < castRange && enemyCandidate.IsActive && !skillInfo.Passive && !enemyCandidate.Invisible {
						dmg, _ := c.CalculateDamageToPlayer(enemyCandidate, true)
						dmg += plusDamage * divinePlus

						if skillID == 41002 || skillID == 41102 || skillID == 41701 || skillID == 41300 {
							dmg = 3
						}

						if skillInfo.BookID == 16100033 || skillInfo.BookID == 16100034 || skillInfo.BookID == 16100035 || skillInfo.BookID == 16100036 || skillInfo.BookID == 16100037 || skillInfo.BookID == 16100038 {
							masterPercent := (10 / 100.0) * float32(dmg)
							masterSkillInc := int(masterPercent)
							dmg += masterSkillInc
						}

						if skillInfo.BookID == 16100101 || skillInfo.BookID == 16100103 || skillInfo.BookID == 16100107 || skillInfo.BookID == 16100109 || skillInfo.BookID == 16100111 || skillInfo.BookID == 16100113 || skillInfo.BookID == 16100115 {
							divinePercen := (20 / 100.0) * float32(dmg)
							divineSkillInc := int(divinePercen)
							dmg += divineSkillInc
						}
						c.PlayerTargets = append(c.PlayerTargets, &PlayerTarget{Damage: dmg, Enemy: enemyCandidate})
					}

				}
			}
		}
	}

	/*
		else { // FIX => attacked to player
			enemy := FindCharacterByPseudoID(user.ConnectedServer, uint16(targetID))
			if enemy != nil && enemy.IsActive && skillInfo.PassiveType != 14 && enemy != c {
				dmg, _ := c.CalculateDamageToPlayer(enemy, true)
				c.PlayerTargets = append(c.PlayerTargets, &PlayerTarget{Damage: dmg, Enemy: enemy, Skill: true})
			}
		}
	*/

OUT:
	r := SKILL_CASTED
	r.Insert(utils.IntToBytes(uint64(c.PseudoID), 2, true), 7) // character pseudo id
	r[9] = byte(attackCounter)
	r.Insert(utils.IntToBytes(uint64(skillID), 4, true), 10)  // skill id
	r.Insert(utils.FloatToBytes(cX, 4, true), 14)             // coordinate-x
	r.Insert(utils.FloatToBytes(cY, 4, true), 18)             // coordinate-y
	r.Insert(utils.FloatToBytes(cZ, 4, true), 22)             // coordinate-z
	r.Insert(utils.IntToBytes(uint64(targetID), 2, true), 27) // target id
	r.Insert(utils.IntToBytes(uint64(targetID), 2, true), 30) // target id

	p := nats.CastPacket{CastNear: true, CharacterID: c.ID, Type: nats.CAST_SKILL, Data: r}
	if err = p.Cast(); err != nil {
		return nil, err
	}

	resp.Concat(r)
	resp.Concat(c.GetHPandChi())
	//go stat.Update()
	return resp, nil
}

func (c *Character) CalculateDamage(ai *AI, isSkill bool) (int, error) {

	st := c.Socket.Stats

	npcPos := NPCPos[ai.PosID]
	npc := NPCs[npcPos.NPCID]

	def, min, max := npc.DEF, st.MinATK, st.MaxATK
	if isSkill {
		def, min, max = npc.ArtsDEF, st.MinArtsATK, st.MaxArtsATK
	}

	dmg := int(utils.RandInt(int64(min), int64(max))) - def
	if dmg < 3 {
		dmg = 3
	} else if dmg > ai.HP {
		dmg = ai.HP
	}

	if c.Map == 243 {
		if YingYangMobsCounter[c.Map] == nil {
			YingYangMobsCounter[c.Map] = &DungeonMobsCounter{
				BlackBandits:       50,
				Rogues:             50,
				Ghosts:             50,
				Animals:            50,
				BlackBanditsLeader: 1,
				RogueKingsLeader:   1,
				GhostWarriorKing:   1,
				BeastMaster:        1,
				Paechun:            1,
			}
		}
		YingYangMobsMutex.Lock()
		counter := YingYangMobsCounter[c.Map]
		YingYangMobsMutex.Unlock()
		if npcPos.NPCID == 60003 && counter.BlackBandits > 5 {
			dmg = 0
		} else if npcPos.NPCID == 60005 && counter.Rogues > 15 {
			dmg = 0
		} else if npcPos.NPCID == 60008 && counter.Ghosts > 15 {
			dmg = 0
		} else if (npcPos.NPCID == 60013 || npcPos.NPCID == 60014) && (counter.BlackBanditsLeader > 0 || counter.RogueKingsLeader > 0 || counter.GhostWarriorKing > 0) {
			dmg = 0
		}
	}

	if diff := int(npc.Level) - c.Level; diff > 0 {
		reqAcc := utils.SigmaFunc(float64(diff))
		if float64(st.Accuracy) < reqAcc {
			probability := float64(st.Accuracy) * 1000 / reqAcc
			if utils.RandInt(0, 1000) > int64(probability) {
				dmg = 0
			}
		}
	}

	if c.Map == 1 {
		if AchievementMobsCounter[c.Map] == nil {
			AchievementMobsCounter[c.Map] = &AchievementMobCounter{
				WolfPup: 3,
			}
		}
		AchievementMobsMutex.Lock()
		counter := AchievementMobsCounter[c.Map]
		AchievementMobsMutex.Unlock()
		if npcPos.NPCID == 40101 {
			if counter.WolfPup > 0 {
				counter.WolfPup--
			}
		} /*else if counter.WolfPup == 0 {
			c.Socket.Write(messaging.InfoMessage(fmt.Sprintf("You have been promoted to Beginner!")))
			ANNOUNCEMENT := utils.Packet{0xAA, 0x55, 0x54, 0x00, 0x71, 0x14, 0x51, 0x55, 0xAA}
			msg := " has accomplished the [Beginning] stage"
			announce := ANNOUNCEMENT
			index := 6
			announce[index] = byte(len(c.Name) + len(msg))
			index++
			announce.Insert([]byte("["+c.Name+"]"), index) // character name
			index += len(c.Name) + 2
			announce.Insert([]byte(msg), index) // character name
			announce.SetLength(int16(binary.Size(announce) - 6))
			p := nats.CastPacket{CastNear: false, Data: announce}
			p.Cast()
			return dmg, nil
		}*/
	}
	return dmg, nil
}

func (c *Character) CalculateDamageToPlayer(enemy *Character, isSkill bool) (int, error) {
	st := c.Socket.Stats

	enemySt := enemy.Socket.Stats

	def, min, max := enemySt.DEF, st.MinATK, st.MaxATK
	if isSkill {
		def, min, max = enemySt.ArtsDEF, st.MinArtsATK, st.MaxArtsATK
	}

	def = utils.PvPFunc(def)

	dmg := int(utils.RandInt(int64(float64(min)*1.02), int64(float64(max)*0.97))) - def // Apo damage fix // eskisi min 1.02 - max 0.97

	if dmg < 0 {
		dmg = 3
	}

	/*
		if dmg < 0 {
			dmg = 3
		} else if dmg > enemySt.HP {
			dmg = enemySt.HP
		}
	*/

	reqAcc := float64(enemySt.Dodge) - float64(st.Accuracy) + float64(c.Level-int(enemy.Level))*10
	//probability := float64(st.Accuracy) * 1000 / reqAcc
	if utils.RandInt(0, 1000) < int64(reqAcc) {
		different := int(c.Level - enemy.Level)
		if math.Signbit(float64(different)) {
			different = int(math.Abs(float64(different)))
		}

		if different >= 0 && different <= 20 {
			dmg = 0
		}

	}

	return dmg, nil
}

func (c *Character) CancelTrade() {

	trade := FindTrade(c)
	if trade == nil {
		return
	}

	receiver, sender := trade.Receiver.Character, trade.Sender.Character
	trade.Delete()

	resp := TRADE_CANCELLED
	sender.Socket.Write(resp)
	receiver.Socket.Write(resp)
}

func (c *Character) OpenSale(name string, slotIDs []int16, prices []uint64) ([]byte, error) {

	slots, err := c.InventorySlots()
	if err != nil {
		return nil, err
	}

	sale := &Sale{ID: c.PseudoID, Seller: c, Name: name}
	for i := 0; i < len(slotIDs); i++ {
		slotID := slotIDs[i]
		price := prices[i]
		item := slots[slotID]
		info := Items[item.ItemID]

		if slotID == 0 || price == 0 || item == nil || item.ItemID == 0 || info.Tradable == 2 {
			continue
		}

		saleItem := &SaleItem{SlotID: slotID, Price: price, IsSold: false}
		sale.Items = append(sale.Items, saleItem)
	}

	sale.Data, err = sale.SaleData()
	if err != nil {
		return nil, err
	}

	go sale.Create()

	c.SaleActive = true
	resp := OPEN_SALE
	spawnData, err := c.SpawnCharacter()
	if err == nil {
		p := nats.CastPacket{CastNear: true, CharacterID: c.ID, Type: nats.PLAYER_SPAWN, Data: spawnData}
		err := p.Cast()
		if err != nil {
			fmt.Println(err)
		}
		resp.Concat(spawnData)

		go func(c *Character, spawnData []byte) {
			characters, err := c.GetNearbyCharacters()
			if err != nil {
				log.Println(err)
				return
			}
			//r.Concat([]byte{0xAA, 0x55, 0x02, 0x00, 0x2A, 0x04, 0x55, 0xAA})
			_ = funk.Filter(characters, func(chr *Character) bool {
				if chr.ID != c.ID {
					chr.OnSight.PlayerMutex.Lock()
					delete(chr.OnSight.Players, c.ID)
					chr.OnSight.PlayerMutex.Unlock()
					chr.Socket.Write(spawnData)
				}
				return true
			}).([]*Character)

			//fmt.Println(len(onSightChars))

		}(c, spawnData)
	}

	return resp, nil
}

func FindSaleVisitors(saleID uint16) []*Character {

	characterMutex.RLock()
	allChars := funk.Values(characters)
	characterMutex.RUnlock()

	return funk.Filter(allChars, func(c *Character) bool {
		return c.IsOnline && c.VisitedSaleID == saleID
	}).([]*Character)
}

func (c *Character) CloseSale() ([]byte, error) {
	sale := FindSale(c.PseudoID)
	if sale != nil {
		sale.Delete()
		resp := CLOSE_SALE
		c.SaleActive = false
		c.SaleActiveEpoch = 0

		spawnData, err := c.SpawnCharacter()
		if err == nil {
			p := nats.CastPacket{CastNear: true, CharacterID: c.ID, Type: nats.PLAYER_SPAWN, Data: spawnData}
			p.Cast()
			resp.Concat(spawnData)
		}

		return resp, nil
	}

	return nil, nil
}

func (c *Character) BuySaleItem(saleID uint16, saleSlotID, inventorySlotID int16) ([]byte, error) {
	sale := FindSale(saleID)
	if sale == nil {
		return nil, nil
	}

	mySlots, err := c.InventorySlots()
	if err != nil {
		return nil, err
	}

	seller := sale.Seller
	slots, err := seller.InventorySlots()
	if err != nil {
		return nil, err
	}

	saleItem := sale.Items[saleSlotID]
	if saleItem == nil || saleItem.IsSold {
		return nil, nil
	}

	item := slots[saleItem.SlotID]
	if item == nil || item.ItemID == 0 || c.Gold < saleItem.Price {
		return nil, nil
	}

	c.LootGold(-saleItem.Price)
	seller.Gold += saleItem.Price

	resp := BOUGHT_SALE_ITEM
	resp.Insert(utils.IntToBytes(c.Gold, 8, true), 8)                   // buyer gold
	resp.Insert(utils.IntToBytes(uint64(item.ItemID), 4, true), 17)     // sale item id
	resp.Insert(utils.IntToBytes(uint64(item.Quantity), 2, true), 23)   // sale item quantity
	resp.Insert(utils.IntToBytes(uint64(inventorySlotID), 2, true), 25) // inventory slot id
	resp.Insert(item.GetUpgrades(), 27)                                 // sale item upgrades
	resp[42] = byte(item.SocketCount)                                   // item socket count
	resp.Insert(item.GetSockets(), 43)                                  // sale item sockets

	myItem := NewSlot()
	*myItem = *item
	myItem.CharacterID = null.IntFrom(int64(c.ID))
	myItem.UserID = null.StringFrom(c.UserID)
	myItem.SlotID = int16(inventorySlotID)
	mySlots[inventorySlotID] = myItem
	myItem.Update()
	InventoryItems.Add(myItem.ID, myItem)

	resp.Concat(item.GetData(inventorySlotID))
	//logger.Log(logging.ACTION_BUY_SALE_ITEM, c.ID, fmt.Sprintf("Bought sale item (%d) with %d gold from seller (%d)", myItem.ID, saleItem.Price, seller.ID), c.UserID, c.Name)
	itemInfo := Items[item.ItemID]
	go logging.AddLogFile(3, c.Socket.User.ID+" idli kullanici ("+c.Name+") isimli karakteriyle, ("+seller.Socket.User.ID+") idye sahip ("+seller.Name+") isimli karakterin kurduğu pazardan item çekti. İtem:  (+"+strconv.Itoa(int(item.Plus))+" "+itemInfo.Name+") Fiyat: ("+strconv.Itoa(int(saleItem.Price))+")G (SALES)")
	saleItem.IsSold = true

	sellerResp := SOLD_SALE_ITEM
	sellerResp.Insert(utils.IntToBytes(uint64(saleSlotID), 2, true), 8)  // sale slot id
	sellerResp.Insert(utils.IntToBytes(seller.Gold, 8, true), 10)        // seller gold
	sellerResp.Insert(utils.IntToBytes(uint64(c.PseudoID), 2, true), 18) // buyer pseudo id

	*item = *NewSlot()
	sellerResp.Concat(item.GetData(saleItem.SlotID))

	remainingCount := len(funk.Filter(sale.Items, func(i *SaleItem) bool {
		return i.IsSold == false
	}).([]*SaleItem))

	if remainingCount > 0 {
		sale.Data, _ = sale.SaleData()
		resp.Concat(sale.Data)

	} /*else {
		sale.Delete()

		spawnData, err := seller.SpawnCharacter()
		if err == nil {
			p := nats.CastPacket{CastNear: true, CharacterID: seller.ID, Type: nats.PLAYER_SPAWN, Data: spawnData}
			p.Cast()
			resp.Concat(spawnData)
		}

		visitors := FindSaleVisitors(sale.ID)
		for _, v := range visitors {
			v.Socket.Write(CLOSE_SALE)
			v.VisitedSaleID = 0
		}

		//resp.Concat(CLOSE_SALE)
		sellerResp.Concat(CLOSE_SALE)
	}*/

	seller.Socket.Write(sellerResp)
	return resp, nil
}

func (c *Character) UpdatePartyStatus() {

	user := c.Socket.User
	stat := c.Socket.Stats

	party := FindParty(c)
	if party == nil {
		return
	}

	coordinate := ConvertPointToLocation(c.Coordinate)

	resp := PARTY_STATUS
	resp.Insert(utils.IntToBytes(uint64(c.ID), 4, true), 6)             // character id
	resp.Insert(utils.IntToBytes(uint64(stat.HP), 4, true), 10)         // character hp
	resp.Insert(utils.IntToBytes(uint64(stat.MaxHP), 4, true), 14)      // character max hp
	resp.Insert(utils.FloatToBytes(float64(coordinate.X), 4, true), 19) // coordinate-x
	resp.Insert(utils.FloatToBytes(float64(coordinate.Y), 4, true), 23) // coordinate-y
	resp.Insert(utils.IntToBytes(uint64(stat.CHI), 4, true), 27)        // character chi
	resp.Insert(utils.IntToBytes(uint64(stat.MaxCHI), 4, true), 31)     // character max chi
	resp.Insert(utils.IntToBytes(uint64(c.Level), 4, true), 35)         // character level
	resp[39] = byte(c.Type)                                             // character type
	resp[41] = byte(user.ConnectedServer - 1)                           // connected server id

	//fmt.Println("Server: ", user.ConnectedServer-1)

	members := party.GetMembers()
	members = funk.Filter(members, func(m *PartyMember) bool {
		return m.Accepted
	}).([]*PartyMember)

	party.Leader.Socket.Write(resp)
	for _, m := range members {
		m.Socket.Write(resp)
	}
}

func (c *Character) LeaveParty() {

	party := FindParty(c)
	if party == nil {
		return
	}

	if c.Map == 243 || c.IsDungeon {
		return
	}

	c.PartyID = ""

	members := party.GetMembers()
	members = funk.Filter(members, func(m *PartyMember) bool {
		return m.Accepted
	}).([]*PartyMember)

	resp := utils.Packet{}
	if c.ID == party.Leader.ID { // disband party
		resp = PARTY_DISBANDED
		party.Leader.Socket.Write(resp)

		for _, member := range members {
			member.PartyID = ""
			member.Socket.Write(resp)
		}

		party.Delete()

	} else { // leave party
		member := party.GetMember(c.ID)
		party.RemoveMember(member)

		resp = LEFT_PARTY
		resp.Insert(utils.IntToBytes(uint64(c.ID), 4, true), 8) // character id

		leader := party.Leader
		if len(party.GetMembers()) == 0 {
			leader.PartyID = ""
			resp.Concat(PARTY_DISBANDED)
			party.Delete()

		}

		leader.Socket.Write(resp)
		for _, m := range members {
			m.Socket.Write(resp)
		}

	}
}

func (c *Character) GetGuildData() ([]byte, error) {

	if c.GuildID > 0 {
		guild, err := FindGuildByID(c.GuildID)
		if err != nil {
			return nil, err
		} else if guild == nil {
			return nil, nil
		}

		return guild.GetData(c)
	}

	return nil, nil
}

func (c *Character) JobPassives(stat *Stat) error {

	//stat := c.Socket.Stats
	skills, err := FindSkillsByID(c.ID)
	if err != nil {
		return err
	}

	skillSlots, err := skills.GetSkills()
	if err != nil {
		return err
	}

	if passive := skillSlots.Slots[5]; passive.BookID > 0 {
		info := JobPassives[int8(c.Class)]
		if info != nil {
			plus := passive.Skills[0].Plus
			stat.MaxHP += info.MaxHp * plus
			stat.MaxCHI += info.MaxChi * plus
			stat.MinATK += info.ATK * plus
			stat.MaxATK += info.ATK * plus
			stat.MinArtsATK += info.ArtsATK * plus
			stat.MaxArtsATK += info.ArtsATK * plus
			stat.DEF += info.DEF * plus
			stat.ArtsDEF += info.ArtsDef * plus
			stat.Accuracy += info.Accuracy * plus
			stat.Dodge += info.Dodge * plus
		}
	}

	slots := funk.Filter(skillSlots.Slots, func(slot *SkillSet) bool { // get 2nd job passive book
		return slot.BookID == 16100200 || slot.BookID == 16100300 || slot.BookID == 100030021 || slot.BookID == 100030023 || slot.BookID == 100030025
	}).([]*SkillSet)

	for _, slot := range slots {
		for _, skill := range slot.Skills {
			info := SkillInfos[skill.SkillID]
			if info == nil {
				continue
			}

			amount := info.BasePassive + info.AdditionalPassive*skill.Plus
			switch info.PassiveType {
			case 1: // passive hp
				stat.MaxHP += amount
			case 2: // passive chi
				stat.MaxCHI += amount
			case 3: // passive arts defense
				stat.ArtsDEF += amount
			case 4: // passive defense
				stat.DEF += amount
			case 5: // passive accuracy
				stat.Accuracy += amount
			case 6: // passive dodge
				stat.Dodge += amount
			case 7: // passive arts atk
				stat.MinArtsATK += amount
				stat.MaxArtsATK += amount
			case 8: // passive atk
				stat.MinATK += amount
				stat.MaxATK += amount
			case 9: //HP AND CHI
				stat.MaxHP += amount
				stat.MaxCHI += amount
			case 11: //Dodge RAte AND ACCURACY
				stat.Accuracy += amount
				stat.Dodge += amount
			case 12: //EXTERNAL ATK AND INTERNAL ATK
				stat.MinArtsATK += amount
				stat.MaxArtsATK += amount
				stat.MinATK += amount
				stat.MaxATK += amount
			case 13: //INTERNAL ATTACK AND INTERNAL DEF
				stat.MinATK += amount
				stat.MaxATK += amount
				stat.DEF += amount
			case 14: //EXTERNAL ATK MINUS AND HP +
				stat.MaxHP += amount
				stat.MinArtsATK -= amount
				stat.MaxArtsATK -= amount
			case 15: //DAMAGE + HP
				stat.MaxHP += amount
				stat.MinATK += amount
				stat.MaxATK += amount
			case 16: //MINUS HP AND PLUS DEFENSE
				stat.MaxHP -= 15 //
				stat.DEF += amount
			}
		}
	}

	return nil
}

func (c *Character) BuffEffects(stat *Stat) error {

	buffs, err := FindBuffsByCharacterID(c.ID)
	if err != nil {
		return err
	}

	//stat := c.Socket.Stats

	for _, buff := range buffs {
		if buff.Duration == 0 && buff.CanExpire {
			buff.Delete()
			continue
		}
		if buff.StartedAt+buff.Duration > c.Epoch {

			stat.PoisonATK += buff.PoisonDamage
			stat.PoisonDEF += buff.PoisonDEF
			stat.ParalysisATK += buff.PoisonDamage
			stat.ParalysisDEF += buff.ParalysisDEF
			stat.ConfusionATK += buff.ConfusionDamage
			stat.ConfusionDEF += buff.ConfusionDEF

			stat.MinATK += buff.ATK
			stat.MaxATK += buff.ATK
			stat.ATKRate += buff.ATKRate
			stat.Accuracy += buff.Accuracy
			stat.MinArtsATK += buff.ArtsATK
			stat.MaxArtsATK += buff.ArtsATK
			stat.ArtsATKRate += buff.ArtsATKRate
			stat.ArtsDEF += buff.ArtsDEF
			stat.ArtsDEFRate += buff.ArtsDEFRate
			stat.CHIRecoveryRate += buff.CHIRecoveryRate
			stat.ConfusionDEF += buff.ConfusionDEF
			stat.DEF += buff.DEF
			stat.DefRate += buff.DEFRate
			stat.DEXBuff += buff.DEX
			stat.Dodge += buff.Dodge
			stat.HPRecoveryRate += buff.HPRecoveryRate
			stat.INTBuff += buff.INT
			stat.MaxCHI += buff.MaxCHI
			stat.MaxHP += buff.MaxHP
			stat.ParalysisDEF += buff.ParalysisDEF
			stat.PoisonDEF += buff.PoisonDEF
			stat.STRBuff += buff.STR
			stat.FireBuff += buff.Fire
			stat.WaterBuff += buff.Water
			stat.WindBuff += buff.Wind
			if c != nil {
				c.RunningSpeed += buff.RunningSpeed
				if c.Socket != nil && c.Socket.User != nil {
					if c.Socket.User.UserType <= 2 {
						c.RunningSpeed = 20
					}
				}

			}
		}
	}

	return nil
}

func (c *Character) GetLevelText() string {
	if c.Level < 10 {
		return fmt.Sprintf("%dKyu", c.Level)
	} else if c.Level <= 100 {
		return fmt.Sprintf("%dDan %dKyu", c.Level/10, c.Level%10)
	} else if c.Level < 110 {
		return fmt.Sprintf("Divine %dKyu", c.Level%100)
	} else if c.Level <= 200 {
		return fmt.Sprintf("Divine %dDan %dKyu", (c.Level-100)/10, c.Level%100)
	} else if c.Level > 200 && c.Level <= 209 {
		return fmt.Sprintf("Darkness %dKyu", c.Level%200)
	} else if c.Level >= 210 {
		return fmt.Sprintf("Darkness %dDan %dKyu", (c.Level-200)/10, c.Level%200)
	}

	return ""
}

func (c *Character) RelicDrop(itemID int64) []byte {

	itemName := Items[itemID].Name
	relic := Relics[int(itemID)]
	msg := fmt.Sprintf("%s has acquired %d / (out of %d) [%s].", c.Name, relic.Count, relic.Limit, itemName)
	length := int16(len(msg) + 3)

	resp := RELIC_DROP
	resp.SetLength(length)
	resp[6] = byte(len(msg))
	resp.Insert([]byte(msg), 7)

	return resp
}

func (c *Character) AidStatus() []byte {

	resp := utils.Packet{}
	if c.AidMode {
		resp = utils.Packet{0xAA, 0x55, 0x02, 0x00, 0xFA, 0x01, 0x55, 0xAA}

		r2 := utils.Packet{0xAA, 0x55, 0x04, 0x00, 0x43, 0x01, 0x55, 0xAA}
		r2.Insert(utils.IntToBytes(uint64(c.PseudoID), 2, true), 5) // pseudo id

		resp.Concat(r2)

	} else {
		resp = utils.Packet{0xAA, 0x55, 0x02, 0x00, 0xFA, 0x00, 0x55, 0xAA}

		r2 := utils.Packet{0xAA, 0x55, 0x04, 0x00, 0x43, 0x01, 0x55, 0xAA}
		r2.Insert(utils.IntToBytes(uint64(c.PseudoID), 2, true), 5) // pseudo id

		resp.Concat(r2)
	}

	return resp
}

func (c *Character) PickaxeActivated() bool {

	slots, err := c.InventorySlots()
	if err != nil {
		return false
	}

	pickaxeIDs := []int64{17200219, 17300005, 17501009, 17502536, 17502537, 17502538}

	return len(funk.Filter(slots, func(slot *InventorySlot) bool {
		return slot.Activated && funk.Contains(pickaxeIDs, slot.ItemID)
	}).([]*InventorySlot)) > 0
}

func (c *Character) TogglePet() []byte {

	if c == nil || c.Socket.User == nil {
		return nil
	}

	slots, err := c.InventorySlots()
	if err != nil {
		return nil
	}

	petSlot := slots[0x0A]
	if petSlot.ItemID == 50000001 {
		return nil
	}
	pet := petSlot.Pet
	if pet == nil {
		return nil
	}

	if c.Map == 230 || c.Map == 243 {
		return nil
	}

	spawnData, _ := c.SpawnCharacter()
	pet.PetOwner = c
	petInfo := Pets[petSlot.ItemID]
	if petInfo.Combat || !petInfo.Combat {
		location := ConvertPointToLocation(c.Coordinate)
		pet.Coordinate = utils.Location{X: location.X + 3, Y: location.Y}
		pet.IsOnline = !pet.IsOnline

		if pet.IsOnline {
			GeneratePetID(c, pet)
			pet.PetCombatMode = 0
			pet.CombatPet = true
			c.PetHandlerCB = c.PetHandler
			go c.PetHandlerCB()

			resp := utils.Packet{
				0xAA, 0x55, 0x0B, 0x00, 0x75, 0x00, 0x01, 0x00, 0x80, 0xa1, 0x43, 0x00, 0x00, 0x3d, 0x43, 0x55, 0xAA,
				0xAA, 0x55, 0x05, 0x00, 0x51, 0x01, 0x0A, 0x00, 0x00, 0x55, 0xAA,
				0xAA, 0x55, 0x06, 0x00, 0x51, 0x05, 0x0A, 0x00, 0x00, 0x00, 0x55, 0xAA,
				0xAA, 0x55, 0x05, 0x00, 0x51, 0x06, 0x0A, 0x00, 0x00, 0x55, 0xAA,
				0xAA, 0x55, 0x05, 0x00, 0x51, 0x07, 0x0A, 0x00, 0x00, 0x55, 0xAA,
				0xaa, 0x55, 0x12, 0x00, 0x51, 0x08, 0x0a, 0x00, 0x03, 0x01, 0x3b, 0x00, 0x3b, 0x00, 0x26, 0x00, 0x00, 0x00, 0x8d, 0x00, 0x00, 0x00, 0x55, 0xaa,
			}

			resp.Concat(spawnData)
			return resp
		}
	} else {
		return nil
		/*location := ConvertPointToLocation(c.Coordinate)
		pet.Coordinate = utils.Location{X: location.X, Y: location.Y}
		pet.IsOnline = !pet.IsOnline
		if pet.IsOnline {
			GeneratePetID(c, pet)
			pet.PetCombatMode = 0
			c.IsMounting = true
			pet.CombatPet = false
			c.PetHandlerCB = c.PetHandler
			go c.PetHandlerCB()
			c.Socket.Write([]byte{0xAA, 0x55, 0x0B, 0x00, 0x75, 0x00, 0x01, 0x00, 0x80, 0xA1, 0x43, 0x00, 0x00, 0x3D, 0x43, 0x55, 0xAA})
			c.Socket.Write([]byte{0xAA, 0x55, 0x05, 0x00, 0x51, 0x01, 0x0A, 0x00, 0x00, 0x55, 0xAA})
			c.Socket.Write(spawnData)
			c.Socket.Write([]byte{0xAA, 0x55, 0x05, 0x00, 0x51, 0x06, 0x0A, 0x00, 0x00, 0x55, 0xAA})
			c.Socket.Write([]byte{0xAA, 0x55, 0x06, 0x00, 0x51, 0x05, 0x0A, 0x00, 0x00, 0x00, 0x55, 0xAA})
			c.Socket.Write([]byte{0xAA, 0x55, 0x05, 0x00, 0x51, 0x07, 0x0A, 0x00, 0x00, 0x55, 0xAA})
		}*/
	}
	pet.Target = 0
	pet.Casting = false
	pet.IsMoving = false
	c.PetHandlerCB = nil
	c.IsMounting = false
	RemovePetFromRegister(c)
	resp := DISMISS_PET
	return resp

	/*
		location := ConvertPointToLocation(c.Coordinate)
		pet.Coordinate = utils.Location{X: location.X + 3, Y: location.Y}
		pet.IsOnline = !pet.IsOnline

		spawnData, _ := c.SpawnCharacter()
		pet.PetOwner = c
		petInfo, ok := Pets[petSlot.ItemID]
		if ok && !petInfo.Combat {
			p := nats.CastPacket{CastNear: true, CharacterID: c.ID, Data: spawnData}
			p.Cast()
		}

		if pet.IsOnline {
			GeneratePetID(c, pet)

			c.PetHandlerCB = c.PetHandler
			go c.PetHandlerCB()

			resp := utils.Packet{
				0xAA, 0x55, 0x0B, 0x00, 0x75, 0x00, 0x01, 0x00, 0x80, 0xA1, 0x43, 0x00, 0x00, 0x3D, 0x43, 0x55, 0xAA,
				0xAA, 0x55, 0x05, 0x00, 0x51, 0x01, 0x0A, 0x00, 0x00, 0x55, 0xAA,
				0xAA, 0x55, 0x06, 0x00, 0x51, 0x05, 0x0A, 0x00, 0x00, 0x00, 0x55, 0xAA,
				0xAA, 0x55, 0x05, 0x00, 0x51, 0x06, 0x0A, 0x00, 0x00, 0x55, 0xAA,
				0xAA, 0x55, 0x05, 0x00, 0x51, 0x07, 0x0A, 0x00, 0x00, 0x55, 0xAA,
			}

			resp.Concat(spawnData)

			return resp
		}

		pet.Target = 0
		pet.Casting = false
		pet.IsMoving = false
		c.PetHandlerCB = nil
		RemovePetFromRegister(c)
		return DISMISS_PET
	*/
}

func (c *Character) DealDamageToPlayer(char *Character, dmg int) {

	if c == nil {
		log.Println("character is nil")
		return
	} else if char.Socket.Stats.HP <= 0 {
		return
	} else if c.ID == char.ID {
		return
	}
	if dmg > char.Socket.Stats.HP {
		dmg = char.Socket.Stats.HP
	}

	char.Socket.Stats.HP -= dmg
	if char.Socket.Stats.HP <= 0 {
		char.Socket.Stats.HP = 0
	}

	meditresp := utils.Packet{}
	if char.Meditating {
		meditresp = MEDITATION_MODE
		meditresp.Insert(utils.IntToBytes(uint64(char.PseudoID), 2, true), 6) // character pseudo id
		meditresp[8] = 0
		char.Meditating = false
	}
	buffs, err := FindBuffsByCharacterID(int(char.PseudoID))
	index := 5
	r := DEAL_DAMAGE
	r.Insert(utils.IntToBytes(uint64(char.PseudoID), 2, true), index) // ai pseudo id
	index += 2
	r.Insert(utils.IntToBytes(uint64(c.PseudoID), 2, true), index) // ai pseudo id
	index += 2
	r.Insert(utils.IntToBytes(uint64(char.Socket.Stats.HP), 4, true), index) // ai current hp
	index += 4
	r.Insert(utils.IntToBytes(uint64(char.Socket.Stats.CHI), 4, true), index) // ai current chi
	index += 4

	if err == nil {
		r.Overwrite(utils.IntToBytes(uint64(len(buffs)), 1, true), index) //BUFF ID
		index++
		//index = 22
		count := 0
		for _, buff := range buffs {
			r.Insert(utils.IntToBytes(uint64(buff.ID), 4, true), index) //BUFF ID
			index += 4
			if count < len(buffs)-1 {
				r.Insert(utils.IntToBytes(uint64(0), 2, true), index) //BUFF ID
				index += 2
			}
			count++
		}
		index += 4
	} else {
		fmt.Println("Valami error: %s", err)
	}
	r.SetLength(int16(binary.Size(r) - 6))
	c.Socket.Write(r)
	r.Concat(meditresp)
	char.Socket.Write(r)
}

func (c *Character) DealDamage(ai *AI, dmg int) {

	if c == nil {
		log.Println("character is nil")
		return
	} else if ai.HP <= 0 {
		return
	}
	//s := c.Socket
	if dmg > ai.HP {
		dmg = ai.HP
	}

	ai.HP -= dmg
	if ai.HP <= 0 {
		ai.HP = 0
	}
	ai.TargetPlayerID = int(c.PseudoID)
	d := ai.DamageDealers.Get(c.ID)
	if d == nil {
		ai.DamageDealers.Add(c.ID, &Damage{Damage: dmg, DealerID: c.ID})
	} else {
		d.(*Damage).Damage += dmg
		ai.DamageDealers.Add(c.ID, d)
	}

	if c.Invisible {
		buff, _ := FindBuffByID(241, c.ID)
		if buff != nil {
			buff.Duration = 0
			go buff.Update()
		}

		buff, _ = FindBuffByID(244, c.ID)
		if buff != nil {
			buff.Duration = 0
			go buff.Update()
		}

		buff, _ = FindBuffByID(138, c.ID)
		if buff != nil {
			buff.Duration = 0
			go buff.Update()
		}

		buff, _ = FindBuffByID(139, c.ID)
		if buff != nil {
			buff.Duration = 0
			go buff.Update()
		}

		if c.DuelID > 0 {
			opponent, _ := FindCharacterByID(c.DuelID)
			spawnData, _ := c.SpawnCharacter()

			r := utils.Packet{}
			r.Concat(spawnData)
			r.Overwrite(utils.IntToBytes(500, 2, true), 13) // duel state

			sock := GetSocket(opponent.UserID)
			if sock != nil {
				sock.Write(r)
			}
		}
	}
	// injuryNumbers := c.CalculateInjury()
	// injury1 := fmt.Sprintf("%x", injuryNumbers[1]) //0.7
	// injury0 := fmt.Sprintf("%x", injuryNumbers[0]) //0.1
	// injury3 := fmt.Sprintf("%x", injuryNumbers[3]) //17.48
	// injury2 := fmt.Sprintf("%x", injuryNumbers[2]) //1.09
	// injuryByte1 := string(injury0 + injury1)
	// data, err := hex.DecodeString(injuryByte1)
	// if err != nil {
	// 	panic(err)
	// }
	// injuryByte2 := string(injury3 + injury2)
	// data2, err := hex.DecodeString(injuryByte2)
	// if err != nil {
	// 	panic(err)
	// }
	npcID := uint64(NPCPos[ai.PosID].NPCID)
	npc := NPCs[int(npcID)]
	npcMaxHPHalf := (npc.MaxHp / 2) / 10
	index := 5
	r := DEAL_DAMAGE
	r.Insert(utils.IntToBytes(uint64(ai.PseudoID), 2, true), index) // ai pseudo id
	index += 2
	r.Insert(utils.IntToBytes(uint64(c.PseudoID), 2, true), index) // ai pseudo id
	index += 2
	r.Insert(utils.IntToBytes(uint64(ai.HP), 4, true), index) // ai current hp
	index += 4
	r.Insert(utils.IntToBytes(uint64(npcMaxHPHalf), 4, true), index) // ai current chi
	index += 4

	// Atak kritik tüm olaylar burada
	index += 3
	r.Insert([]byte{0x00, 0x00, 0x00}, index) // INJURY
	index += 2
	r.SetLength(int16(binary.Size(r) - 6))
	p := &nats.CastPacket{CastNear: true, MobID: ai.ID, Data: r, Type: nats.MOB_ATTACK}
	if err := p.Cast(); err != nil {
		log.Println("deal damage broadcast error:", err)
		return
	}
	CheckMobDead(ai, c)
	//c.Socket.Write(r)

}

func CheckMobDead(ai *AI, c *Character) {
	npcPos := NPCPos[ai.PosID]
	if npcPos == nil {
		log.Println("npc pos is nil")
		return
	}

	npc := NPCs[npcPos.NPCID]
	if npc == nil {
		log.Println("npc is nil")
		return
	}
	/*if npc.ID == 50065{
		health := ai.HP - (ai.HP /20)
		if ai.HP <= health{

		}
	}*/

	if !npcPos.Attackable {
		go ai.DropHandler(c)
	}
	if ai.HP <= 0 { // ai died
		if c.Map == 243 {
			if YingYangMobsCounter[c.Map] == nil {
				YingYangMobsCounter[c.Map] = &DungeonMobsCounter{
					BlackBandits:       0,
					Rogues:             0,
					Ghosts:             0,
					Animals:            0,
					BlackBanditsLeader: 0,
					RogueKingsLeader:   0,
					GhostWarriorKing:   0,
					BeastMaster:        0,
					Paechun:            0,
				}
			}

			YingYangMobsMutex.Lock()
			counter := YingYangMobsCounter[c.Map]
			YingYangMobsMutex.Unlock()
			if npcPos.NPCID == 60001 || npcPos.NPCID == 60002 || npcPos.NPCID == 60015 || npcPos.NPCID == 60016 {
				counter.BlackBandits--
				if counter.BlackBandits < 0 {
					counter.BlackBandits = 0
				}
				c.Socket.Write(messaging.InfoMessage(fmt.Sprintf("You have %d BlackBandits left to kill", counter.BlackBandits)))
			} else if npcPos.NPCID == 60004 || npcPos.NPCID == 60018 {
				counter.Rogues--
				if counter.Rogues < 0 {
					counter.Rogues = 0
				}
				c.Socket.Write(messaging.InfoMessage(fmt.Sprintf("You have %d Rogues left to kill", counter.Rogues)))
			} else if npcPos.NPCID == 60006 || npcPos.NPCID == 60007 || npcPos.NPCID == 60020 || npcPos.NPCID == 60021 {
				counter.Ghosts--
				if counter.Ghosts < 0 {
					counter.Ghosts = 0
				}
				c.Socket.Write(messaging.InfoMessage(fmt.Sprintf("You have %d Ghosts left to kill", counter.Ghosts)))
			} else if npcPos.NPCID == 60009 || npcPos.NPCID == 60010 || npcPos.NPCID == 60011 || npcPos.NPCID == 60012 ||
				npcPos.NPCID == 60023 || npcPos.NPCID == 60024 || npcPos.NPCID == 60025 || npcPos.NPCID == 60026 {
				counter.Animals--
				if counter.Animals < 0 {
					counter.Animals = 0
				}
				c.Socket.Write(messaging.InfoMessage(fmt.Sprintf("You have %d animals left to kill", counter.Animals)))
			} else if npcPos.NPCID == 60003 {
				counter.BlackBanditsLeader--
				if counter.BlackBanditsLeader < 0 {
					counter.BlackBanditsLeader = 0
				}
			} else if npcPos.NPCID == 60005 {
				counter.RogueKingsLeader--
				if counter.RogueKingsLeader < 0 {
					counter.RogueKingsLeader = 0
				}
			} else if npcPos.NPCID == 60008 {
				counter.GhostWarriorKing--
				if counter.GhostWarriorKing < 0 {
					counter.GhostWarriorKing = 0
				}
			} else if npcPos.NPCID == 60013 || npcPos.NPCID == 60014 {
				if npcPos.NPCID == 60013 {
					counter.BeastMaster--
					if counter.BeastMaster < 0 {
						counter.BeastMaster = 0
					}
				}
				if npcPos.NPCID == 60014 {
					counter.Paechun--
					if counter.Paechun < 0 {
						counter.Paechun = 0
					}
				}
			}
		}

		/*if ai.HP <= 0 { // ai died
			if c.Map == 1 {
				if AchievementMobsCounter[c.Map] == nil {
					AchievementMobsCounter[c.Map] = &AchievementMobCounter{
						WolfPup: 0,
					}

				}

				AchievementMobsMutex.Lock()
				counter := AchievementMobsCounter[c.Map]
				AchievementMobsMutex.Unlock()
				if npcPos.NPCID == 40101 {
					counter.WolfPup--
					if counter.WolfPup < 0 {
						counter.WolfPup = 0
						c.Socket.Write(messaging.InfoMessage(fmt.Sprintf("Kill %d More Wolf Pups to achieve the [Beginning] Achievement", counter.WolfPup)))
					}
					return
				}
			}
		}*/

		// 			//Tusan					Tokma           // Hoho            // Rakma           // Red Dragon		//Leopard			//Ancient 		// Clan Leader 		// Hulma			// Mahu			// Devil Jin
		if npc.ID == 9999991 || npc.ID == 9999992 || npc.ID == 9999993 || npc.ID == 9999994 || npc.ID == 9999995 || npc.ID == 9999996 || npc.ID == 9999997 || npc.ID == 9999998 || npc.ID == 42561 || npc.ID == 42562 || npc.ID == 43206 {
			bossType := 0
			query := `insert into hops.boss_hunting (boss_type, boss_name, killed_by, respawn_time) values ($1, $2, $3, $4);`

			if npc.Exp > 0 {
				bossType = 0
			} else if npc.DivineExp > 0 {
				bossType = 1
			} else if npc.DarknessExp > 0 {
				bossType = 2
			}
			db.Exec(query, bossType, npc.Name+" (D"+strconv.Itoa(ai.Server)+")", c.Name, time.Now().UTC())
		}

		if ai.Once {
			ai.Handler = nil
		} else {
			time.AfterFunc(time.Duration(npcPos.RespawnTime)*time.Second/2, func() { // respawn mob n secs later
				curCoordinate := ConvertPointToLocation(ai.Coordinate)
				minCoordinate := ConvertPointToLocation(npcPos.MinLocation)
				maxCoordinate := ConvertPointToLocation(npcPos.MaxLocation)

				X := utils.RandFloat(minCoordinate.X, maxCoordinate.X)
				Y := utils.RandFloat(minCoordinate.Y, maxCoordinate.Y)

				X = (X / 3) + 2*curCoordinate.X/3
				Y = (Y / 3) + 2*curCoordinate.Y/3

				coordinate := &utils.Location{X: X, Y: Y}
				ai.TargetLocation = *coordinate
				ai.SetCoordinate(coordinate)

				ai.HP = npc.MaxHp
				ai.IsDead = false

				/*
					if npc.ID == 18600007 {
						//makeAnnouncement("Santa has spawned.")
					}
				*/

				/*if npc.ID == 18600036 {
					makeAnnouncement("Wife has spawned.")
				}

				if npc.ID == 18600037 {
					makeAnnouncement("Husband has spawned.")
				}

				if npc.ID == 18600034 {
					makeAnnouncement("Black Leopard has spawned.")
				}

				if npc.ID == 18600035 {
					makeAnnouncement("Ancient Slayer has spawned.")
				}

				if npc.ID == 18600038 {
					makeAnnouncement("Wolf Pup has spawned.")
				}

				if npc.ID == 18600073 {
					makeAnnouncement("Soul has spawned.")
				}
				*/

				if npc.ID == 9999991 {
					makeAnnouncement("The Dragon Castle Worldboss has arrived! Gather your strength and conquer it! ")
				}
				if npc.ID == 9999992 {
					makeAnnouncement("The Highlands Worldboss has appeared! Will you rise to the challenge?")
				}
				if npc.ID == 9999993 {
					makeAnnouncement("Danger stirs in Venom Swamp — the Worldboss has spawned! Face it if you dare!")
				}
				if npc.ID == 9999994 {
					makeAnnouncement("The Spirit Spire Worldboss awakens! Gather your allies and fight! ")
				}
				if npc.ID == 9999995 {
					makeAnnouncement("The Southern Plains tremble — the Worldboss has arrived! Rally now! ")
				}
				if npc.ID == 9999996 {
					makeAnnouncement("A mighty foe looms over Bamboo Mountain! The Worldboss awaits your strength! ")
				}
				if npc.ID == 9999997 {
					makeAnnouncement("The earth shakes in Stone Valley — the Worldboss has emerged! Battle begins now!")
				}
				if npc.ID == 9999998 {
					makeAnnouncement("TSilence breaks in Silent Valley — the Worldboss rises! Will you stand or fall?")
				}

				if npc.ID == 423316 || npc.ID == 423310 || npc.ID == 423314 || npc.ID == 423308 || npc.ID == 423312 {
					makeAnnouncement("Temples are weakened right now. Team your guild up and conquer some of them! May the glory be upon you! ")
				}

				if npc.ID == 18600079 {
					chars := FindCharactersInMap(76)
					for _, c := range chars {
						if c != nil && c.Socket != nil {
							tmpData, _ := c.ChangeMap(1, nil)
							c.Socket.Write(tmpData)
						}
					}
				}
			})
		}
		if c.Selection == int(ai.PseudoID) {
			c.Selection = 0
		}

		if npc.ID == 424201 && WarStarted {
			OrderPoints = 0 //OrderPoints -= 200
		} else if npc.ID == 424202 && WarStarted {
			ShaoPoints = 0 // ShaoPoints -= 200
		}

		if isFactionWarStarted {
			if npc.ID == 425501 {
				AddPointsToFactionWarFaction(1, 2)
			} else if npc.ID == 425502 {
				AddPointsToFactionWarFaction(15, 2)
			} else if npc.ID == 425503 {
				AddPointsToFactionWarFaction(2, 2)
			} else if npc.ID == 425504 {
				AddPointsToFactionWarFaction(1000, 2)
			} else if npc.ID == 425505 {
				AddPointsToFactionWarFaction(15, 1)
			} else if npc.ID == 425506 {
				AddPointsToFactionWarFaction(1, 1)
			} else if npc.ID == 425507 {
				AddPointsToFactionWarFaction(2, 1)
			} else if npc.ID == 425508 {
				AddPointsToFactionWarFaction(1000, 1)
			}
		}

		if npc.ID == 18600038 {
			dealers := ai.DamageDealers.Values()
			if len(dealers) == 0 {
				return
			}

			exp := int64(npc.Exp / int64(len(dealers)))
			for i := range dealers {
				tmpChr, err := FindCharacterByID(dealers[i].(*Damage).DealerID)
				if err == nil {
					if tmpChr != nil {
						if tmpChr.Level <= 100 {
							if tmpChr.IsOnline && tmpChr.Map == 10 {
								r, levelUp := tmpChr.AddExp(exp)
								if levelUp {
									statData, err := tmpChr.GetStats()
									if err == nil {
										tmpChr.Socket.Write(statData)
									}
								}
								tmpChr.Socket.Write(r)
							}
						}
					}
				}
			}
			return
		}

		exp := int64(0)
		if c.Level <= 100 {
			exp = npc.Exp
		} else if c.Level <= 200 {
			exp = npc.DivineExp
		} else {
			exp = npc.DarknessExp
		}

		// EXP gained
		r, levelUp := c.AddExp(exp)
		if levelUp {
			statData, err := c.GetStats()
			if err == nil {
				c.Socket.Write(statData)
			}

		}
		c.Socket.Write(r)

		// EXP gain for party members
		party := FindParty(c)
		if party != nil {
			members := funk.Filter(party.GetMembers(), func(m *PartyMember) bool {
				return m.Accepted || m.ID == c.ID
			}).([]*PartyMember)
			members = append(members, &PartyMember{Character: party.Leader, Accepted: true})

			coordinate := ConvertPointToLocation(c.Coordinate)
			for _, m := range members {
				user, err := FindUserByID(m.UserID)
				if err != nil || user == nil { //  || (c.Level-m.Level) > 20
					continue
				}

				memberCoordinate := ConvertPointToLocation(m.Coordinate)

				if m.ID == c.ID && !m.Accepted {
					break
				}

				if m.ID == c.ID || m.Map != c.Map || c.Socket.User.ConnectedServer != user.ConnectedServer ||
					utils.CalculateDistance(coordinate, memberCoordinate) > 100 || m.Socket.Stats.HP <= 0 {
					continue
				}

				exp := int64(0)
				if m.Level <= 100 {
					exp = npc.Exp
				} else if m.Level <= 200 {
					exp = npc.DivineExp
				} else {
					exp = npc.DarknessExp
				}

				exp /= int64(len(members))

				r, levelUp := m.AddExp(exp)
				if levelUp {
					statData, err := m.GetStats()
					if err == nil {
						m.Socket.Write(statData)
					}
				}
				m.Socket.Write(r)
			}
		}

		// PTS gained LOOT
		dropMaxLevel := int(npc.Level + 25)
		if c.Level <= dropMaxLevel {
			c.PTS++
			if c.PTS%50 == 0 {
				r = c.GetPTS()
				c.HasLot = true
				c.Socket.Write(r)
			}
		}
		// Gold dropped
		goldDrop := int64(npc.GoldDrop)
		if goldDrop > 0 {
			amount := uint64(utils.RandInt(goldDrop/2, goldDrop))
			if GOLD_EVENT == 1 {
				r = c.LootGold(amount * uint64(GOLD_RATE))
			} else {
				r = c.LootGold(amount)
			}

			c.Socket.Write(r)
		}

		claimer, _ := ai.FindClaimer()
		if funk.Contains(FiveclanMobs, npc.ID) {
			if claimer.GuildID != -1 {
				/*
					guildTempleCounter := 0

					if FiveClans[1].ClanID == claimer.GuildID {
						guildTempleCounter++
					}

					if FiveClans[2].ClanID == claimer.GuildID {
						guildTempleCounter++
					}

					if FiveClans[3].ClanID == claimer.GuildID {
						guildTempleCounter++
					}

					if FiveClans[4].ClanID == claimer.GuildID {
						guildTempleCounter++
					}

					if FiveClans[5].ClanID == claimer.GuildID {
						guildTempleCounter++
					}

					if guildTempleCounter >= 3 {
						return
					}
				*/
				exp := time.Now().UTC().Add(time.Hour * 6)
				if npc.ID == 423308 { //HWARANG GUARDIAN STATUE //SOUTHERN WOOD TEMPLE
					FiveClans[4].ClanID = c.GuildID
					FiveClans[4].ExpiresAt = null.TimeFrom(exp)
					FiveClans[4].Update()
					guild, err := FindGuildByID(c.GuildID)
					if err != nil {
						return
					}
					if guild == nil {
						return
					}
					allmembers, err := guild.GetMembers()
					if err != nil {
						return
					}
					for _, clanmembers := range allmembers {
						char, err := FindCharacterByID(clanmembers.ID)
						if err != nil {
							continue
						}

						// 5 Drop
						// 15 Exp
						infection := BuffInfections[70004]
						buff := &Buff{ID: int(70004), CharacterID: char.ID, Name: infection.Name, EXPMultiplier: 5, DropMultiplier: 2, StartedAt: char.Epoch, Duration: 21500, CanExpire: true}
						err = buff.Create()
						if err != nil {
							continue
						}

						char.ExpMultiplier += 0.05 // char.ExpMultiplier += 0.2
						char.DropMultiplier += 0.02
						char.Update()
					}

					makeAnnouncement("[" + FiveClans[4].TempleName + "] has been conquered by [" + guild.Name + "]")
				} else if npc.ID == 423310 { //SUGUN GUARDIAN STATUE //LIGHTNING HILL TEMPLE
					FiveClans[3].ClanID = c.GuildID
					FiveClans[3].ExpiresAt = null.TimeFrom(exp)
					FiveClans[3].Update()
					guild, err := FindGuildByID(c.GuildID)
					if err != nil {
						fmt.Println(err)
						return
					}
					if guild == nil {
						return
					}
					allmembers, err := guild.GetMembers()
					if err != nil {
						return
					}
					for _, clanmembers := range allmembers {
						char, err := FindCharacterByID(clanmembers.ID)
						if err != nil {
							fmt.Println(err)
							continue
						}

						infection := BuffInfections[70003]
						buff := &Buff{ID: int(70003), CharacterID: char.ID, Name: infection.Name, EXPMultiplier: 5, DropMultiplier: 2, StartedAt: char.Epoch, Duration: 21500, CanExpire: true}
						err = buff.Create()
						if err != nil {
							fmt.Println(err)
							continue
						}

						char.ExpMultiplier += 0.05
						char.DropMultiplier += 0.02
						char.Update()
					}

					makeAnnouncement("[" + FiveClans[3].TempleName + "] has been conquered by [" + guild.Name + "]")
				} else if npc.ID == 423312 { //CHUNKYUNG GUARDIAN STATUE //OCEAN ARMY TEMPLE
					FiveClans[2].ClanID = c.GuildID
					FiveClans[2].ExpiresAt = null.TimeFrom(exp)
					FiveClans[2].Update()
					guild, err := FindGuildByID(c.GuildID)
					if err != nil {
						return
					}
					if guild == nil {
						return
					}
					allmembers, err := guild.GetMembers()
					if err != nil {
						return
					}
					for _, clanmembers := range allmembers {
						char, err := FindCharacterByID(clanmembers.ID)
						if err != nil {
							continue
						}
						infection := BuffInfections[70002]
						buff := &Buff{ID: int(70002), CharacterID: char.ID, Name: infection.Name, EXPMultiplier: 2, DropMultiplier: 2, StartedAt: char.Epoch, Duration: 21500, CanExpire: true}
						err = buff.Create()
						if err != nil {
							continue
						}

						char.DropMultiplier += 0.02 // char.DropMultiplier += 0.2
						char.ExpMultiplier += 0.02
						char.Update()
					}

					makeAnnouncement("[" + FiveClans[2].TempleName + "] has been conquered by [" + guild.Name + "]")
				} else if npc.ID == 423314 { //MOKNAM GUARDIAN STATUE //FLAME WOLF TEMPLE
					FiveClans[1].ClanID = c.GuildID
					FiveClans[1].ExpiresAt = null.TimeFrom(exp)
					FiveClans[1].Update()
					guild, err := FindGuildByID(c.GuildID)
					if err != nil {
						return
					}
					if guild == nil {
						return
					}
					allmembers, err := guild.GetMembers()
					if err != nil {
						return
					}
					for _, clanmembers := range allmembers {
						char, err := FindCharacterByID(clanmembers.ID)
						if err != nil {
							continue
						}
						infection := BuffInfections[70001]
						buff := &Buff{ID: int(70001), CharacterID: char.ID, Name: infection.Name, EXPMultiplier: 3, DropMultiplier: 2, StartedAt: char.Epoch, Duration: 21500, CanExpire: true}
						err = buff.Create()
						if err != nil {
							continue
						}

						char.DropMultiplier += 0.02
						char.ExpMultiplier += 0.03 // char.ExpMultiplier += 0.2
						char.Update()
					}

					makeAnnouncement("[" + FiveClans[1].TempleName + "] has been conquered by [" + guild.Name + "]")
				} else if npc.ID == 423316 { //JISU GUARDIAN STATUE //WESTERN LAND TEMPLE
					FiveClans[5].ClanID = c.GuildID
					FiveClans[5].ExpiresAt = null.TimeFrom(exp)
					FiveClans[5].Update()
					guild, err := FindGuildByID(c.GuildID)
					if err != nil {
						return
					}
					if guild == nil {
						return
					}
					allmembers, err := guild.GetMembers()
					if err != nil {
						return
					}
					for _, clanmembers := range allmembers {
						char, err := FindCharacterByID(clanmembers.ID)
						if err != nil {
							continue
						}
						infection := BuffInfections[70005]
						buff := &Buff{ID: int(70005), CharacterID: char.ID, Name: infection.Name, EXPMultiplier: 5, DropMultiplier: 2, StartedAt: char.Epoch, Duration: 21500, CanExpire: true}
						err = buff.Create()
						if err != nil {
							continue
						}
						char.ExpMultiplier += 0.05 // char.ExpMultiplier += 0.1
						char.DropMultiplier += 0.02
						char.Update()
					}

					makeAnnouncement("[" + FiveClans[5].TempleName + "] has been conquered by [" + guild.Name + "]")
				}
			}
		}

		if funk.Contains(GuildWarMobs, npc.ID) {
			if claimer.GuildID != -1 {
				exp := time.Now().UTC().Add(time.Hour * 24 * 6)
				if npc.ID == 18600047 { //HWARANG GUARDIAN STATUE //SOUTHERN WOOD TEMPLE
					GuildWarAreas[4].ClanID = c.GuildID
					GuildWarAreas[4].ExpiresAt = null.TimeFrom(exp)
					GuildWarAreas[4].Update()
					guild, err := FindGuildByID(c.GuildID)
					if err != nil {
						return
					}
					if guild == nil {
						return
					}
					allmembers, err := guild.GetMembers()
					if err != nil {
						return
					}
					for _, clanmembers := range allmembers {
						char, err := FindCharacterByID(clanmembers.ID)
						if err != nil {
							continue
						}

						infection := BuffInfections[70013]
						buff := &Buff{ID: int(70013), CharacterID: char.ID, Name: infection.Name, EXPMultiplier: 15, DropMultiplier: 3, StartedAt: char.Epoch, Duration: 604500, CanExpire: true}
						err = buff.Create()
						if err != nil {
							continue
						}

						char.ExpMultiplier += 0.15 // char.ExpMultiplier += 0.2
						char.DropMultiplier += 0.03
						char.Update()
					}

					makeAnnouncement("[" + GuildWarAreas[4].TempleName + "] has been conquered by [" + guild.Name + "]")
				} else if npc.ID == 18600048 { //SUGUN GUARDIAN STATUE //LIGHTNING HILL TEMPLE
					GuildWarAreas[3].ClanID = c.GuildID
					GuildWarAreas[3].ExpiresAt = null.TimeFrom(exp)
					GuildWarAreas[3].Update()
					guild, err := FindGuildByID(c.GuildID)
					if err != nil {
						fmt.Println(err)
						return
					}
					if guild == nil {
						return
					}
					allmembers, err := guild.GetMembers()
					if err != nil {
						return
					}
					for _, clanmembers := range allmembers {
						char, err := FindCharacterByID(clanmembers.ID)
						if err != nil {
							fmt.Println(err)
							continue
						}

						infection := BuffInfections[70012]
						buff := &Buff{ID: int(70012), CharacterID: char.ID, Name: infection.Name, EXPMultiplier: 15, DropMultiplier: 3, StartedAt: char.Epoch, Duration: 604500, CanExpire: true}
						err = buff.Create()
						if err != nil {
							fmt.Println(err)
							continue
						}

						char.ExpMultiplier += 0.15
						char.DropMultiplier += 0.03
						char.Update()
					}

					makeAnnouncement("[" + GuildWarAreas[3].TempleName + "] has been conquered by [" + guild.Name + "]")
				} else if npc.ID == 18600049 { //CHUNKYUNG GUARDIAN STATUE //OCEAN ARMY TEMPLE
					GuildWarAreas[2].ClanID = c.GuildID
					GuildWarAreas[2].ExpiresAt = null.TimeFrom(exp)
					GuildWarAreas[2].Update()
					guild, err := FindGuildByID(c.GuildID)
					if err != nil {
						return
					}
					if guild == nil {
						return
					}
					allmembers, err := guild.GetMembers()
					if err != nil {
						return
					}
					for _, clanmembers := range allmembers {
						char, err := FindCharacterByID(clanmembers.ID)
						if err != nil {
							continue
						}
						infection := BuffInfections[70009]
						buff := &Buff{ID: int(70009), CharacterID: char.ID, Name: infection.Name, EXPMultiplier: 15, StartedAt: char.Epoch, Duration: 604500, CanExpire: true}
						err = buff.Create()
						if err != nil {
							continue
						}

						char.ExpMultiplier += 0.15
						char.Update()
					}

					makeAnnouncement("[" + GuildWarAreas[2].TempleName + "] has been conquered by [" + guild.Name + "]")
				} else if npc.ID == 18600050 { //MOKNAM GUARDIAN STATUE //FLAME WOLF TEMPLE
					GuildWarAreas[1].ClanID = c.GuildID
					GuildWarAreas[1].ExpiresAt = null.TimeFrom(exp)
					GuildWarAreas[1].Update()
					guild, err := FindGuildByID(c.GuildID)
					if err != nil {
						return
					}
					if guild == nil {
						return
					}
					allmembers, err := guild.GetMembers()
					if err != nil {
						return
					}
					for _, clanmembers := range allmembers {
						char, err := FindCharacterByID(clanmembers.ID)
						if err != nil {
							continue
						}
						infection := BuffInfections[70010]
						buff := &Buff{ID: int(70010), CharacterID: char.ID, Name: infection.Name, DropMultiplier: 5, StartedAt: char.Epoch, Duration: 604500, CanExpire: true}
						err = buff.Create()
						if err != nil {
							continue
						}

						char.DropMultiplier += 0.05 // char.ExpMultiplier += 0.2
						char.Update()
					}

					makeAnnouncement("[" + GuildWarAreas[1].TempleName + "] has been conquered by [" + guild.Name + "]")
				} else if npc.ID == 18600051 { //JISU GUARDIAN STATUE //WESTERN LAND TEMPLE
					GuildWarAreas[5].ClanID = c.GuildID
					GuildWarAreas[5].ExpiresAt = null.TimeFrom(exp)
					GuildWarAreas[5].Update()
					guild, err := FindGuildByID(c.GuildID)
					if err != nil {
						return
					}
					if guild == nil {
						return
					}
					allmembers, err := guild.GetMembers()
					if err != nil {
						return
					}
					for _, clanmembers := range allmembers {
						char, err := FindCharacterByID(clanmembers.ID)
						if err != nil {
							continue
						}
						infection := BuffInfections[70011]
						buff := &Buff{ID: int(70011), CharacterID: char.ID, Name: infection.Name, EXPMultiplier: 15, DropMultiplier: 3, StartedAt: char.Epoch, Duration: 604500, CanExpire: true}
						err = buff.Create()
						if err != nil {
							continue
						}
						char.ExpMultiplier += 0.15 // char.ExpMultiplier += 0.1
						char.DropMultiplier += 0.03
						char.Update()
					}

					makeAnnouncement("[" + GuildWarAreas[5].TempleName + "] has been conquered by [" + guild.Name + "]")
				}
			}
		}

		if npc.ID == 18600078 {
			exp := time.Now().UTC().Add(time.Hour * 24 * 23)
			GoldenBasinArea.ExpiresAt = null.TimeFrom(exp)
			GoldenBasinArea.FactionID = claimer.Faction
			err := GoldenBasinArea.Update()
			if err != nil {
				fmt.Println("Golden basin update err : ", err)
			}
			if claimer.Faction == 1 {
				//makeAnnouncement("Zhuangs has conquered the Golden Basin")
				chars := FindCharactersInMap(76)
				for _, c := range chars {
					if c != nil && c.Socket != nil {
						if c.Faction != 1 {
							tmpData, _ := c.ChangeMap(1, nil)
							c.Socket.Write(tmpData)
						}
					}
				}
			} else if claimer.Faction == 2 {
				//makeAnnouncement("Shaos has conquered the Golden Basin")
				chars := FindCharactersInMap(76)
				for _, c := range chars {
					if c != nil && c.Socket != nil {
						if c.Faction != 2 {
							tmpData, _ := c.ChangeMap(1, nil)
							c.Socket.Write(tmpData)
						}
					}
				}
			}
		}

		//Item dropped
		go func() {
			claimer, err := ai.FindClaimer()
			if err != nil || claimer == nil {
				return
			}
			dropMaxLevel := int(npc.Level + 25) //max Level +25 char pode dropar
			if c.Level <= dropMaxLevel {
				ai.DropHandler(claimer)
			}
			ai.DropHandler(claimer)
			time.AfterFunc(time.Second, func() { // time para que o player possa pegar o item !
				ai.DamageDealers.Clear()
			})
		}()

		time.AfterFunc(time.Second, func() { // disappear mob 1 sec later
			ai.TargetPlayerID = 0
			ai.TargetPetID = 0
			ai.IsDead = true
		})
	} else if ai.TargetPlayerID == 0 {
		ai.IsMoving = false
		ai.MovementToken = 0
		ai.TargetPlayerID = c.ID
	} else {
		ai.IsMoving = false
		ai.MovementToken = 0
	}
}

func (c *Character) GetPetStats() []byte {

	slots, err := c.InventorySlots()
	if err != nil {
		return nil
	}

	petSlot := slots[0x0A]
	pet := petSlot.Pet
	if pet == nil {
		return nil
	}

	resp := utils.Packet{}
	resp = petSlot.GetPetStats(c)
	resp.Concat(petSlot.GetData(0x0A))
	return resp
}

func (c *Character) StartPvP(timeLeft int) {

	info, resp := "", utils.Packet{}
	if timeLeft > 0 {
		info = fmt.Sprintf("Duel will start %d seconds later.", timeLeft)
		time.AfterFunc(time.Second, func() {
			c.StartPvP(timeLeft - 1)
		})

	} else if c.DuelID > 0 {
		info = "Duel has started."
		resp.Concat(c.OnDuelStarted())
	}

	resp.Concat(messaging.InfoMessage(info))
	c.Socket.Write(resp)
}

func (c *Character) CanAttack(enemy *Character) bool {

	if c.DuelID == enemy.ID && c.DuelStarted {
		return true
	}

	if funk.Contains(PVPServers, int16(enemy.Socket.User.ConnectedServer)) {
		return true
	}

	if c.Map == 250 && enemy.Map == 250 {
		return true
	}

	if c.Map == 12 {
		return true
	}

	if c.Map == 108 || c.Map == 255 && c.Faction != enemy.Faction {
		return true
	}

	if c.Map == 233 || c.Map == 74 {
		if c.GuildID != enemy.GuildID {
			return true
		}
	}

	if c.Map == 254 && LastmanStarted && c.IsinLastMan && enemy.IsinLastMan {
		return true
	}

	if c.IsinWar && enemy.IsinWar && c.Faction != enemy.Faction {
		if WarStarted {
			return true
		}
	}

	if c.Map == 76 && enemy.Faction != c.Faction {
		return true
	}

	if c.Map == 77 && enemy.Faction == 0 {
		return true
	}

	return false

	//return (c.DuelID == enemy.ID && c.DuelStarted) || funk.Contains(PVPServers, int16(c.Socket.User.ConnectedServer)) || c.Map == 233 || c.Map == 108 || c.Map == 255 //|| c.Faction != enemy.Faction
}

func (c *Character) OnDuelStarted() []byte {

	c.DuelStarted = true
	statData, _ := c.GetStats()

	opponent, err := FindCharacterByID(c.DuelID)
	if err != nil || opponent == nil {
		return nil
	}

	opData, err := opponent.SpawnCharacter()
	if err != nil || opData == nil || len(opData) < 13 {
		return nil
	}

	r := utils.Packet{}
	r.Concat(opData)
	r.Overwrite(utils.IntToBytes(500, 2, true), 13) // duel state

	resp := utils.Packet{}
	resp.Concat(opponent.GetHPandChi())
	resp.Concat(r)
	resp.Concat(statData)
	resp.Concat([]byte{0xAA, 0x55, 0x02, 0x00, 0x2A, 0x04, 0x55, 0xAA})
	return resp
}

func (c *Character) HasAidBuff() bool {
	slots, err := c.InventorySlots()
	if err != nil {
		return false
	}

	return len(funk.Filter(slots, func(s *InventorySlot) bool {
		return (s.ItemID == 13000170 || s.ItemID == 13000171) && s.Activated && s.InUse
	}).([]*InventorySlot)) > 0
}

func (c *Character) ChangeName(newname string) ([]byte, error) {
	ok, err := IsValidUsername(newname)
	if err != nil {
		return nil, err
	} else if !ok {
		return messaging.SystemMessage(messaging.INVALID_NAME), nil
	}

	if ok {
		c.Name = newname
	}
	c.Update()
	CHARACTER_MENU := utils.Packet{0xAA, 0x55, 0x03, 0x00, 0x09, 0x09, 0x00, 0x55, 0xAA}
	resp := CHARACTER_MENU
	if c != nil {
		c.Socket.User.ConnectingIP = c.Socket.ClientAddr
		c.Socket.User.ConnectingTo = c.Socket.User.ConnectedServer
		c.Logout()
	}
	return resp, nil
}

func (c *Character) GoDarkness() {
	s := c.Socket

	c.Level = 201
	c.Type += 10
	c.Exp = 544951059311

	skills := s.Skills
	skillSlots, err := skills.GetSkills()
	if err != nil {
		return
	}

	sd := utils.Packet{}

	for i := 0; i <= 7; i++ {
		if skillSlots.Slots[i].BookID != 0 {
			if i < 5 {
				if skillSlots.Slots[i].BookID != 0 {
					skillData, _ := c.RemoveSkill(byte(i), skillSlots.Slots[i].BookID)
					sd.Concat(skillData)
				}
			} else {
				skillIndex := byte(0)
				if i == 0 {
					skillIndex = 5
				} else if i == 1 {
					skillIndex = 6
				} else if i == 7 {
					skillIndex = 8
				}
				skillData, _ := c.RemovePassiveSkill(skillIndex, byte(i), skillSlots.Slots[i].BookID)
				sd.Concat(skillData)
			}
		}
	}

	spIndex := utils.SearchUInt64(SkillPoints, uint64(1))
	spIndex2 := utils.SearchUInt64(SkillPoints, uint64(c.Exp))
	skPts := spIndex2 - spIndex
	//skPts += c.Level * int(c.Reborns) * 3
	skills.SkillPoints = skPts
	if c.Level > 100 {
		for i := 101; i <= c.Level; i++ {
			skills.SkillPoints += EXPs[int16(i)].SkillPoints
		}
	}

	c.Socket.Skills.Update()
	c.Class = 40
	c.Update()
	s.User.Update()

	ATARAXIA := utils.Packet{0xAA, 0x55, 0x03, 0x00, 0x57, 0x21, 0xE3, 0x55, 0xAA, 0xaa, 0x55, 0x0b, 0x00, 0x75, 0x00, 0x01, 0x00, 0x80, 0xa1, 0x43, 0x00, 0x00, 0x3d, 0x43, 0x55, 0xaa}
	resp := ATARAXIA
	resp[6] = byte(c.Type) // character type

	ANNOUNCEMENT := utils.Packet{0xAA, 0x55, 0x54, 0x00, 0x71, 0x14, 0x51, 0x55, 0xAA}
	msg := "At this moment I mark my name on list of Darklord Hero."
	announce := ANNOUNCEMENT
	index := 6
	announce[index] = byte(len(c.Name) + len(msg))
	index++
	announce.Insert([]byte("["+c.Name+"]"), index) // character name
	index += len(c.Name) + 2
	announce.Insert([]byte(msg), index) // character name
	announce.SetLength(int16(binary.Size(announce) - 6))
	p := nats.CastPacket{CastNear: false, Data: announce}
	p.Cast()

	statData, _ := c.GetStats()
	resp.Concat(statData)
	resp.Concat(sd)

	skillsData, err := s.Skills.GetSkillsData()
	if err != nil {
		return
	}
	/*
		rr, _, err := c.AddItem(&InventorySlot{ItemID: 17504477, Quantity: 1}, -1, false)
		if err != nil {
			return
		}
		s.Write(*rr)
	*/

	s.Write(skillsData)
	s.Write(resp)

	time.AfterFunc(time.Duration(14*time.Second), func() {
		CharacterSelect := utils.Packet{0xAA, 0x55, 0x04, 0x00, 0x01, 0x05, 0x0A, 0x00, 0x55, 0xAA}
		CHARACTER_MENU := utils.Packet{0xAA, 0x55, 0x03, 0x00, 0x09, 0x09, 0x00, 0x55, 0xAA}
		resp := CHARACTER_MENU
		resp.Concat(CharacterSelect)
		//s.Conn.Write(resp)
		s.Write(resp)
	})
}

func ConvertPointToCoordinate(X float64, Y float64) string {

	str := fmt.Sprintf("%.1f,%.1f", X, Y)

	return str
}

func (c *Character) Enchant(bookID int64, matsSlots []int16, matsIds []int64) ([]byte, error) {
	resp := utils.Packet{}
	bookSlotID, book, err := c.FindItemInInventory(nil, bookID)
	if err != nil || book == nil {
		return nil, err
	}

	fusionSlotID, fusion, err := c.FindItemInInventory(nil, 15710004)
	if err != nil || fusion == nil {
		return nil, err
	}

	enhancement := Enhancements[int(bookID)]
	if enhancement == nil {
		return ENCHANT_ERROR, nil
	}

	reqmats := []int64{enhancement.Material1, enhancement.Material2, enhancement.Material3}

	checked := 0
	for _, reqmat := range reqmats {
		if reqmat == 0 {
			checked++
			continue
		}
		for _, mat := range matsIds {
			if mat == 0 {
				continue
			}
			if reqmat == mat {
				mat = 0
				checked++
			}
		}

	}
	if checked != 3 {
		return nil, fmt.Errorf("Enchant: materialsAmount = %d", checked)
	}
	//purceed
	/*
		for _, mats := range matsSlots {
			data := c.DecrementItem(mats, 1)
			resp.Concat(*data)
		}
	*/

	data := c.DecrementItem(bookSlotID, 1)
	resp.Concat(*data)
	data = c.DecrementItem(fusionSlotID, 1)
	resp.Concat(*data)

	rand := utils.RandInt(0, 1000)
	if rand < int64(enhancement.Rate) {
		for _, mats := range matsSlots {
			data := c.DecrementItem(mats, 1)
			resp.Concat(*data)
		}
		resp.Concat(ENCHANT_SUCCESS)
		additem, _, err := c.AddItem(&InventorySlot{ItemID: int64(enhancement.Result), Quantity: 1}, -1, false)
		if err != nil {
			return nil, err
		} else {
			resp.Concat(*additem)
		}
	} else {
		resp.Concat(ENCHANT_FAILED)
	}

	return resp, nil

}
func (c *Character) CalculateInjury() []int {
	remaining := c.Injury
	divCount := []int{0, 0, 0, 0}
	divNumbers := []float64{0.1, 0.7, 1.09, 17.48}
	for i := len(divNumbers) - 1; i >= 0; i-- {
		if remaining < divNumbers[i] || remaining == 0 {
			continue
		}
		test := remaining / divNumbers[i]
		if test > 15 {
			test = 15
		}
		divCount[i] = int(test)
		test2 := test * divNumbers[i]
		remaining -= test2
	}
	return divCount
}
func (c *Character) SpecialEffects(infection *BuffInfection, duration int64) {

	if infection == nil || c == nil {
		return
	}
	expire := true
	if duration == 0 {
		duration = 1
	}
	buff, err := FindBuffByID(infection.ID, c.ID)
	if err != nil {
		return
	} else {
		if buff == nil {
			buff = &Buff{ID: infection.ID, CharacterID: c.ID, StartedAt: c.Epoch, Duration: int64(duration), Name: infection.Name,
				PoisonDamage:    infection.PoisonDamage + infection.AdditionalPoisonDamage*int(10),
				PoisonDEF:       infection.ParalysisDef + infection.AdditionalPoisonDef*int(10),
				ParalysisDamage: infection.ParalysisDamage + infection.AdditionalParalysisDamage*int(10),
				ParalysisDEF:    infection.ParalysisDef + infection.AdditionalParalysisDef*int(10),
				ConfusionDamage: infection.ConfusionDamage + infection.AdditionalConfusionDamage*int(10),
				ConfusionDEF:    infection.ConfusionDef + infection.AdditionalConfusionDef*int(10),
				SkillPlus:       int(10), CanExpire: expire,
			}
			err := buff.Create()
			if err != nil {
				log.Print(err)
				return
			}
		} else {
			buff.StartedAt = c.Epoch
			buff.Update()
		}
	}
	data, _ := c.GetStats()
	c.Socket.Write(data)
}
