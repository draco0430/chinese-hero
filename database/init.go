package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	"hero-server/config"
	"hero-server/logging"
	"hero-server/utils"

	_ "github.com/lib/pq"
	gorp "gopkg.in/gorp.v1"
)

var (
	DROP_LIFETIME     = time.Duration(30) * time.Second
	FREEDROP_LIFETIME = time.Duration(200) * time.Millisecond
	DEFAULT_DROP_RATE = utils.ParseFloat("1.0")
	DEFAULT_EXP_RATE  = utils.ParseFloat("1.0")
	DROP_RATE         = DEFAULT_DROP_RATE
	EXP_RATE          = DEFAULT_EXP_RATE
	DRAGON_BOX        = 0
	GOLD_EVENT        = 0
	GOLD_RATE         = 1.0
	MAX_INJURY        = utils.ParseFloat("100.0")
)

var (
	db                      *gorp.DbMap
	Init                    = make(chan bool, 1)
	GetFromRegister         func(int, int16, uint16) interface{}
	RemoveFromRegister      func(*Character)
	RemovePetFromRegister   func(c *Character)
	FindCharacterByPseudoID func(server int, ID uint16) *Character

	AccUpgrades          []byte
	ArmorUpgrades        []byte
	WeaponUpgrades       []byte
	SocketAccUpgrades    []byte
	SocketArmorUpgrades  []byte
	SocketWeaponUpgrades []byte
	plusRates            = []int{450, 600, 800, 900, 930, 960, 965}
	logger               = logging.Logger
)

func InitDB() error {

	var (
		cfg = config.Default
		//drv         = cfg.Database.Driver
		ip          = cfg.Database.IP
		port        = cfg.Database.Port
		user        = cfg.Database.User
		pass        = cfg.Database.Password
		name        = cfg.Database.Name
		maxIdle     = cfg.Database.ConnMaxIdle
		maxOpen     = cfg.Database.ConnMaxOpen
		maxLifetime = cfg.Database.ConnMaxLifetime
		debug       = cfg.Database.Debug
		sslMode     = cfg.Database.SSLMode
		err         error
		conn        *sql.DB
	)

	conn, err = sql.Open("postgres", fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s", ip, port, user, pass, name, sslMode))
	if err != nil {
		return fmt.Errorf("Database connection error: %s", err.Error())
	}

	conn.SetMaxIdleConns(maxIdle)
	conn.SetMaxOpenConns(maxOpen)
	conn.SetConnMaxLifetime(time.Duration(maxLifetime) * time.Second)

	if err = conn.Ping(); err != nil {
		return fmt.Errorf("Database connection error: %s", err.Error())
	}

	db = &gorp.DbMap{Db: conn, Dialect: gorp.PostgresDialect{}}
	db.AddTableWithNameAndSchema(Emotion{}, "data", "emotions")
	db.AddTableWithNameAndSchema(PetExpInfo{}, "data", "pet_exp_table")
	db.AddTableWithNameAndSchema(ExpInfo{}, "data", "exp_table")
	db.AddTableWithNameAndSchema(NpcPosition{}, "data", "npc_pos_table").SetKeys(false, "id")
	db.AddTableWithNameAndSchema(Item{}, "data", "items").SetKeys(false, "id")
	db.AddTableWithNameAndSchema(SkillInfo{}, "data", "skills").SetKeys(false, "id")
	db.AddTableWithNameAndSchema(Production{}, "data", "productions")
	db.AddTableWithNameAndSchema(Stackable{}, "data", "stackables")
	db.AddTableWithNameAndSchema(Gambling{}, "data", "gambling")
	db.AddTableWithNameAndSchema(JobPassive{}, "data", "job_passives")
	db.AddTableWithNameAndSchema(SavePoint{}, "data", "save_points")
	db.AddTableWithNameAndSchema(HaxCode{}, "data", "hax_codes")
	db.AddTableWithNameAndSchema(ItemMelting{}, "data", "item_meltings")
	db.AddTableWithNameAndSchema(Gate{}, "data", "gates")
	db.AddTableWithNameAndSchema(DropInfo{}, "data", "drops").SetKeys(false, "id")
	db.AddTableWithNameAndSchema(HtItem{}, "data", "ht_shop").SetKeys(false, "id")
	db.AddTableWithNameAndSchema(NPCScript{}, "data", "npc_scripts")
	db.AddTableWithNameAndSchema(Fusion{}, "data", "advanced_fusion")
	db.AddTableWithNameAndSchema(Pet{}, "data", "pets").SetKeys(false, "id")
	db.AddTableWithNameAndSchema(NPC{}, "data", "npc_table").SetKeys(false, "id")
	db.AddTableWithNameAndSchema(BuffIcon{}, "data", "buff_icons")
	db.AddTableWithNameAndSchema(BuffInfection{}, "data", "buff_infections").SetKeys(false, "id")
	db.AddTableWithNameAndSchema(Shop{}, "data", "shop_table").SetKeys(false, "id")
	db.AddTableWithNameAndSchema(ShopItem{}, "data", "shop_items").SetKeys(false, "type")
	db.AddTableWithNameAndSchema(Enhancement{}, "data", "enchant").SetKeys(true, "id")

	db.AddTableWithNameAndSchema(FiveClan{}, "data", "fiveclan_war").SetKeys(false, "id")
	db.AddTableWithNameAndSchema(GuildWar{}, "data", "guild_war").SetKeys(false, "id")
	db.AddTableWithNameAndSchema(GoldenBasin{}, "data", "golden_basin").SetKeys(true, "id")

	db.AddTableWithNameAndSchema(AI{}, "hops", "ai").SetKeys(true, "id")
	//db.AddTableWithNameAndSchema(AiBuff{}, "hops", "ai_buffs").SetKeys(false, "id", "ai_id")
	db.AddTableWithNameAndSchema(Character{}, "hops", "characters").SetKeys(true, "id")
	db.AddTableWithNameAndSchema(Buff{}, "hops", "characters_buffs").SetKeys(false, "id", "character_id")
	db.AddTableWithNameAndSchema(ConsignmentItem{}, "hops", "consignment").SetKeys(false, "id")
	db.AddTableWithNameAndSchema(Guild{}, "hops", "guilds").SetKeys(true, "id")
	db.AddTableWithNameAndSchema(InventorySlot{}, "hops", "items_characters").SetKeys(true, "id")
	db.AddTableWithNameAndSchema(Relic{}, "hops", "relics").SetKeys(false, "id")
	db.AddTableWithNameAndSchema(Server{}, "hops", "servers").SetKeys(true, "id")
	db.AddTableWithNameAndSchema(Skills{}, "hops", "skills").SetKeys(false, "id")
	db.AddTableWithNameAndSchema(Stat{}, "hops", "stats").SetKeys(false, "id")
	db.AddTableWithNameAndSchema(User{}, "hops", "users").SetKeys(true, "id")

	if debug {
		db.TraceOn("[gorp]", log.New(os.Stdout, "myapp:", log.Lmicroseconds))
	}

	if err = resetDB(); err != nil {
		return err
	}

	if err = getAll(); err != nil {
		return err
	}

	Init <- err == nil
	return nil
}

func resetDB() error {

	query := `update hops.characters set is_active = false, is_online = false, exp_multiplier = 1, drop_multiplier = 1;`
	if _, err := db.Exec(query); err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		return fmt.Errorf("Reset DB error1: %s", err.Error())
	}

	query = `update hops.users set ip = $1, server = 0;`
	if _, err := db.Exec(query, ""); err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		return fmt.Errorf("Reset DB error2: %s", err.Error())
	}

	query = `delete from hops.characters_buffs where id = 70001 or id = 70002 or id = 70003 or id = 70004 or id = 70005;`
	if _, err := db.Exec(query); err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		return fmt.Errorf("Reset DB error3: %s", err.Error())
	}

	query = `truncate table hops.boss_hunting restart identity;`
	if _, err := db.Exec(query); err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		return fmt.Errorf("Reset DB error4: %s", err.Error())
	}

	query = `update data.fiveclan_war set clanid = 0;`
	if _, err := db.Exec(query); err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		return fmt.Errorf("Reset DB error5: %s", err.Error())
	}

	return nil
}

func getAll() error {

	callBacks := []func() error{getAllDrops, getScripts, getHaxCodes, getHTItems, getProductions, getAdvancedFusions, getItemMeltings, getGates,
		getStackables, getAllItems, getSkillInfos, getGamblingItems, getJobPassives, getBuffIcons, getBuffInfections, getExps, getAllSavePoints,
		getRelics, GetAllPetExps, GetAllPets, getAllShops, getAllShopItems, getFiveAreas, getAllEmotions, getGuildAreas, getGoldenBasin, getEnhancements}

	for _, cb := range callBacks {
		if err := cb(); err != nil {
			return err
		}
	}

	return nil
}
