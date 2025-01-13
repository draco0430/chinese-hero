package database

import (
	"database/sql"
	"fmt"
	"log"
	"math"
	"sort"
	"sync"
	"time"

	"hero-server/nats"
	"hero-server/utils"

	"github.com/thoas/go-funk"
)

type Damage struct {
	DealerID int
	Damage   int
}

type AI struct {
	ID           int     `db:"id" json:"id"`
	PosID        int     `db:"pos_id" json:"pos_id"`
	Server       int     `db:"server" json:"server"`
	Faction      int     `db:"faction" json:"faction"`
	Map          int16   `db:"map" json:"map"`
	Coordinate   string  `db:"coordinate" json:"coordinate"`
	WalkingSpeed float64 `db:"walking_speed" json:"walking_speed"`
	RunningSpeed float64 `db:"running_speed" json:"running_speed"`
	CanAttack    bool    `db:"canattack" json:"canattack"`

	DamageDealers  utils.SMap          `db:"-"`
	TargetLocation utils.Location      `db:"-"`
	PseudoID       uint16              `db:"-"`
	CHI            int                 `db:"-" json:"chi"`
	HP             int                 `db:"-" json:"hp"`
	IsDead         bool                `db:"-" json:"is_dead"`
	IsMoving       bool                `db:"-" json:"is_moving"`
	MovementToken  int64               `db:"-" json:"-"`
	OnSightPlayers map[int]interface{} `db:"-" json:"players"`
	PlayersMutex   sync.RWMutex        `db:"-"`
	TargetPlayerID int                 `db:"-" json:"target_player"`
	TargetPetID    int                 `db:"-" json:"target_pet"`
	Handler        func()              `db:"-" json:"-"`
	Once           bool                `db:"-"`
}

type DungeonMobsCounter struct {
	BlackBandits int
	Rogues       int
	Ghosts       int
	Animals      int

	BlackBanditsLeader int
	RogueKingsLeader   int
	GhostWarriorKing   int
	BeastMaster        int
	Paechun            int
}

type AchievementMobCounter struct {
	WolfPup int
}

var (
	AIs      = make(map[int]*AI)
	AIMutex  sync.RWMutex
	AIsByMap []map[int16][]*AI

	DungeonsAiByMap []map[int16][]*AI
	DungeonsByMap   []map[int16]int

	YingYangMobsCounter = make(map[int16]*DungeonMobsCounter)
	YingYangMobsMutex   sync.RWMutex

	AchievementAiByMap []map[int16][]*AI
	AchievementsByMap  []map[int16]int

	AchievementMobsCounter = make(map[int16]*AchievementMobCounter)
	AchievementMobsMutex   sync.RWMutex

	eventBosses = []int{1338006, 1338007, 18600007}

	MOB_MOVEMENT    = utils.Packet{0xAA, 0x55, 0x21, 0x00, 0x33, 0x00, 0xBC, 0xDB, 0x9F, 0x41, 0x52, 0x70, 0xA2, 0x41, 0x00, 0x55, 0xAA}
	MOB_ATTACK      = utils.Packet{0xAA, 0x55, 0x0C, 0x00, 0x41, 0x01, 0x00, 0x01, 0x00, 0x01, 0x00, 0x00, 0x55, 0xAA}
	MOB_SKILL       = utils.Packet{0xAA, 0x55, 0x1B, 0x00, 0x42, 0x0A, 0x00, 0xDF, 0x28, 0xFA, 0xBE, 0x01, 0x01, 0x55, 0xAA}
	MOB_DEAL_DAMAGE = utils.Packet{0xAA, 0x55, 0x28, 0x00, 0x16, 0xFF, 0xFF, 0xFF, 0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x10, 0x27, 0x00, 0x00, 0x55, 0xAA}

	ITEM_DROPPED = utils.Packet{0xAA, 0x55, 0x42, 0x00, 0x67, 0x02, 0x01, 0x01, 0x7A, 0xFB, 0x7B, 0xBF, 0x00, 0xA2, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x55, 0xAA}

	MOB_APPEARED = utils.Packet{0xAA, 0x55, 0x54, 0x00, 0x31, 0x01, 0xFF, 0xFF, 0xFF, 0xFF, 0x01, 0x01,
		0x8E, 0xE5, 0x38, 0xC0, 0xD9, 0xB8, 0x05, 0xC0, 0x00, 0x00, 0x00, 0x40, 0xFF, 0xFF, 0x00, 0xFC, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x64, 0x00, 0x55, 0xAA}

	STONE_APPEARED = utils.Packet{0xAA, 0x55, 0x57, 0x00, 0x31, 0x01, 0x01, 0x00, 0x00, 0x00, 0x0c, 0x45,
		0x6d, 0x70, 0x69, 0x72, 0x65, 0x20, 0x53, 0x74, 0x6f, 0x6e, 0x65, 0x01, 0x01, 0x8E, 0xE5, 0x38, 0xC0, 0xD9, 0xB8, 0x05, 0xC0, 0x00, 0x00, 0x00, 0x40, 0xFF, 0xFF, 0x00, 0xFC, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x64, 0x00, 0x55, 0xAA}

	DROP_DISAPPEARED = utils.Packet{0xAA, 0x55, 0x04, 0x00, 0x67, 0x04, 0x55, 0xAA}

	dropOffsets = []*utils.Location{&utils.Location{0, 0}, &utils.Location{0, 1}, &utils.Location{1, 0}, &utils.Location{1, 1}, &utils.Location{-1, 0},
		&utils.Location{-1, 1}, &utils.Location{-1, -1}, &utils.Location{0, -1}, &utils.Location{1, -1}, &utils.Location{-1, 2}, &utils.Location{0, 2},
		&utils.Location{2, 2}, &utils.Location{2, 1}, &utils.Location{2, 0}, &utils.Location{2, -1}, &utils.Location{2, -2}, &utils.Location{1, -2},
		&utils.Location{0, -2}, &utils.Location{-1, -2}, &utils.Location{-2, -2}, &utils.Location{-2, -1}, &utils.Location{-2, 0}, &utils.Location{-2, 1},
		&utils.Location{-2, 2}, &utils.Location{-2, 3}, &utils.Location{-1, 3}, &utils.Location{0, 3}, &utils.Location{1, 3}, &utils.Location{2, 3},
		&utils.Location{3, 3}, &utils.Location{3, 2}, &utils.Location{3, 1}, &utils.Location{3, 0}, &utils.Location{3, -1}, &utils.Location{3, -2},
		&utils.Location{3, -3}, &utils.Location{2, -3}, &utils.Location{1, -3}, &utils.Location{0, -3}, &utils.Location{-1, -3}, &utils.Location{-2, -3},
		&utils.Location{-3, -3}, &utils.Location{-3, -2}, &utils.Location{-3, -1}, &utils.Location{-3, 0}, &utils.Location{-3, 1}, &utils.Location{-3, 2}, &utils.Location{-3, 3}}
)

func FindAIByID(ID int) *AI {
	return AIs[ID]
}

func (ai *AI) SetCoordinate(coordinate *utils.Location) {
	ai.Coordinate = fmt.Sprintf("(%.1f,%.1f)", coordinate.X, coordinate.Y)
}

func (ai *AI) Create() error {
	return db.Insert(ai)
}

func GetAllAI() error {
	var arr []*AI
	query := `select * from hops.ai order by id`

	if _, err := db.Select(&arr, query); err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		return fmt.Errorf("GetAllAI: %s", err.Error())
	}

	for _, a := range arr {
		AIs[a.ID] = a
	}

	return nil
}

func (ai *AI) FindTargetCharacterID() (int, error) {
	var (
		distance = 15.0
	)

	if len(characters) == 0 {
		return 0, nil
	}

	npcPos := NPCPos[ai.PosID]
	minCoordinate := ConvertPointToLocation(npcPos.MinLocation)
	maxCoordinate := ConvertPointToLocation(npcPos.MaxLocation)
	aiCoordinate := ConvertPointToLocation(ai.Coordinate)

	// characterMutex.Lock()
	characterMutex.RLock()
	allChars := funk.Values(characters)
	characterMutex.RUnlock()
	// characterMutex.Unlock()

	filtered := funk.Filter(allChars, func(c *Character) bool {

		if c.Socket == nil || !c.IsOnline {
			return false
		}

		user := c.Socket.User
		stat := c.Socket.Stats

		if user == nil || stat == nil {
			return false
		}

		characterCoordinate := ConvertPointToLocation(c.Coordinate)

		seed := utils.RandInt(0, 1000)

		return user.ConnectedServer == ai.Server && c.Map == ai.Map && stat.HP > 0 && !c.Invisible &&
			characterCoordinate.X >= minCoordinate.X && characterCoordinate.X <= maxCoordinate.X &&
			characterCoordinate.Y >= minCoordinate.Y && characterCoordinate.Y <= maxCoordinate.Y &&
			utils.CalculateDistance(characterCoordinate, aiCoordinate) <= distance && seed < 500
	})

	filtered = funk.Shuffle(filtered)
	characters := filtered.([]*Character)
	if len(characters) > 0 {
		return characters[0].ID, nil
	}

	return 0, nil
}

func (ai *AI) FindTargetPetID(characterID int) (*InventorySlot, error) {

	enemy, err := FindCharacterByID(characterID)
	if err != nil || enemy == nil {
		return nil, err
	}

	slots, err := enemy.InventorySlots()
	if err != nil {
		return nil, err
	}

	pet := slots[0x0A].Pet
	if pet == nil || !pet.IsOnline {
		return nil, nil
	}

	return slots[0x0A], nil
}

func (ai *AI) Move(targetLocation utils.Location, runningMode byte) []byte {

	resp := MOB_MOVEMENT
	currentLocation := ConvertPointToLocation(ai.Coordinate)

	resp.Insert(utils.IntToBytes(uint64(ai.PseudoID), 2, true), 5) // mob pseudo id
	resp[7] = runningMode
	resp.Insert(utils.FloatToBytes(currentLocation.X, 4, true), 8)  // current coordinate-x
	resp.Insert(utils.FloatToBytes(currentLocation.Y, 4, true), 12) // current coordinate-y
	resp.Insert(utils.FloatToBytes(targetLocation.X, 4, true), 20)  // current coordinate-x
	resp.Insert(utils.FloatToBytes(targetLocation.Y, 4, true), 24)  // current coordinate-y

	speeds := []float64{0, ai.WalkingSpeed, ai.RunningSpeed}
	resp.Insert(utils.FloatToBytes(speeds[runningMode], 4, true), 32) // current coordinate-y

	return resp
}

func (ai *AI) Attack() []byte {

	resp := MOB_ATTACK
	character, err := FindCharacterByID(ai.TargetPlayerID)
	if err != nil || character == nil {
		return nil
	}

	pos := NPCPos[ai.PosID]
	if pos == nil {
		return nil
	}

	npc := NPCs[pos.NPCID]
	if npc == nil {
		return nil
	}

	if character == nil {
		return nil
	}

	if character.Socket == nil {
		return nil
	}

	stat := character.Socket.Stats
	if stat == nil {
		return nil
	}

	rawDamage := int(utils.RandInt(int64(npc.MinATK), int64(npc.MaxATK)))
	damage := int(math.Max(float64(rawDamage-stat.DEF), 3))

	if npc.ID == 423307 || npc.ID == 423309 || npc.ID == 423311 || npc.ID == 423313 || npc.ID == 423315 {
		damage = (stat.MaxHP * 25) / 100
	}

	reqAcc := float64(stat.Dodge) + float64(character.Level-int(npc.Level))*10
	//probability := reqAcc
	if utils.RandInt(0, 1000) < int64(reqAcc) {
		damage = 0
	}

	resp.Insert(utils.IntToBytes(uint64(ai.PseudoID), 2, true), 6) // mob pseudo id
	//resp.Insert([]byte{0}, 8)
	resp.Insert(utils.IntToBytes(uint64(character.PseudoID), 2, true), 8) // character pseudo id

	resp[11] = 2
	if damage > 0 {
		resp[12] = 1 // damage sound
	}

	resp.Concat(ai.DealDamage(damage))
	return resp
}

func (ai *AI) CastSkill() []byte {

	defer func() {
		if err := recover(); err != nil {
			log.Println(err)
			log.Println(ai.TargetPlayerID)
			return
		}
	}()

	character, err := FindCharacterByID(ai.TargetPlayerID)
	if err != nil || character == nil {
		return nil
	}

	pos := NPCPos[ai.PosID]
	if pos == nil {
		return nil
	}

	npc := NPCs[pos.NPCID]
	if npc == nil {
		return nil
	}

	if character == nil {
		return nil
	}

	if character.Socket == nil {
		return nil
	}

	stat := character.Socket.Stats
	if stat == nil {
		return nil
	}

	rawDamage := int(utils.RandInt(int64(npc.MinArtsATK), int64(npc.MaxArtsATK)))
	damage := int(math.Max(float64(rawDamage-stat.ArtsDEF), 3))

	if npc.ID == 423307 || npc.ID == 423309 || npc.ID == 423311 || npc.ID == 423313 || npc.ID == 423315 {
		damage = (stat.HP * 25) / 100
	}

	reqAcc := float64(stat.Dodge) + float64(character.Level-int(npc.Level))*10
	if utils.RandInt(0, 1000) < int64(reqAcc) {
		damage = 0
	}

	mC := ConvertPointToLocation(ai.Coordinate)

	skillIds := npc.GetSkills()
	skillsCount := len(skillIds) - 1
	randomSkill := utils.RandInt(0, int64(skillsCount))

	resp := MOB_SKILL
	resp.Insert(utils.IntToBytes(uint64(ai.PseudoID), 2, true), 7)           // mob pseudo id
	resp.Insert(utils.IntToBytes(uint64(skillIds[randomSkill]), 4, true), 9) // pet skill id
	resp.Insert(utils.FloatToBytes(mC.X, 4, true), 13)                       // pet-x
	resp.Insert(utils.FloatToBytes(mC.Y, 4, true), 17)                       // pet-x
	resp.Insert(utils.IntToBytes(uint64(character.PseudoID), 2, true), 25)   // target pseudo id
	resp.Insert(utils.IntToBytes(uint64(character.PseudoID), 2, true), 28)   // target pseudo id

	//time.AfterFunc(time.Second, func() {
	resp.Concat(ai.DealDamage(damage))
	//p := nats.CastPacket{CastNear: true, MobID: ai.ID, Data: r, Type: nats.MOB_ATTACK}
	//p.Cast()
	//})

	return resp
}

func (ai *AI) AttackPet() []byte {

	resp := MOB_ATTACK
	pet, ok := GetFromRegister(ai.Server, ai.Map, uint16(ai.TargetPetID)).(*PetSlot)
	if !ok || pet == nil || !pet.IsOnline || pet.HP <= 0 {
		return nil
	}

	pos := NPCPos[ai.PosID]
	if pos == nil {
		return nil
	}

	npc := NPCs[pos.NPCID]
	if npc == nil {
		return nil
	}

	if pet.Target == 0 {
		pet.Target = int(ai.PseudoID)
	}

	rawDamage := int(utils.RandInt(int64(npc.MinATK), int64(npc.MaxATK)))
	damage := int(math.Max(float64(rawDamage-pet.DEF), 3))

	reqAcc := float64(int(pet.Level)-int(npc.Level)) * 10
	if utils.RandInt(0, 1000) < int64(reqAcc) {
		damage = 0
	}

	resp.Insert(utils.IntToBytes(uint64(ai.PseudoID), 2, true), 6) // mob pseudo id
	//resp.Insert([]byte{0}, 8)
	resp.Insert(utils.IntToBytes(uint64(pet.PseudoID), 2, true), 8) // character pseudo id

	resp[11] = 2
	if damage > 0 {
		resp[12] = 1 // damage sound
	}

	resp.Concat(ai.DealDamageToPet(damage))
	return resp
}

func (ai *AI) CastSkillToPet() []byte {

	pet, ok := GetFromRegister(ai.Server, ai.Map, uint16(ai.TargetPetID)).(*PetSlot)
	if !ok || pet == nil || !pet.IsOnline || pet.HP <= 0 {
		return nil
	}

	pos := NPCPos[ai.PosID]
	if pos == nil {
		return nil
	}

	npc := NPCs[pos.NPCID]
	if npc == nil {
		return nil
	}

	rawDamage := int(utils.RandInt(int64(npc.MinArtsATK), int64(npc.MaxArtsATK)))
	damage := int(math.Max(float64(rawDamage-pet.ArtsDEF), 3))

	dodge := float64(pet.STR)
	reqAcc := dodge + float64(int(pet.Level)-int(npc.Level))*10
	if utils.RandInt(0, 1000) < int64(reqAcc) {
		damage = 0
	}

	mC := ConvertPointToLocation(ai.Coordinate)
	skillIds := npc.GetSkills()
	skillsCount := len(skillIds) - 1
	randomSkill := utils.RandInt(0, int64(skillsCount))

	resp := MOB_SKILL
	resp.Insert(utils.IntToBytes(uint64(ai.PseudoID), 2, true), 7)           // mob pseudo id
	resp.Insert(utils.IntToBytes(uint64(skillIds[randomSkill]), 4, true), 9) // pet skill id
	resp.Insert(utils.FloatToBytes(mC.X, 4, true), 13)                       // pet-x
	resp.Insert(utils.FloatToBytes(mC.Y, 4, true), 17)                       // pet-x
	resp.Insert(utils.IntToBytes(uint64(pet.PseudoID), 2, true), 25)         // target pseudo id
	resp.Insert(utils.IntToBytes(uint64(pet.PseudoID), 2, true), 28)         // target pseudo id

	//time.AfterFunc(time.Second, func() {
	resp.Concat(ai.DealDamageToPet(damage))
	//p := nats.CastPacket{CastNear: true, MobID: ai.ID, Data: r, Type: nats.MOB_ATTACK}
	//p.Cast()
	//})

	return resp
}

func (ai *AI) DealDamage(damage int) []byte {

	defer func() {
		if err := recover(); err != nil {
			log.Println(err)
			log.Println(ai.TargetPlayerID)
			return
		}
	}()

	resp := MOB_DEAL_DAMAGE
	character, err := FindCharacterByID(ai.TargetPlayerID)
	if err != nil || character == nil {
		return nil
	}

	if character == nil {
		return nil
	}

	if character.Socket == nil {
		return nil
	}

	stat := character.Socket.Stats
	if stat == nil {
		return nil
	}

	if damage > 0 {
		if character.Injury < MAX_INJURY {
			character.Injury += 0.001
			if character.Injury > MAX_INJURY {
				character.Injury = MAX_INJURY
			}
			if character.Injury >= 70 {
				statData, err := character.GetStats()
				if err == nil {
					character.Socket.Write(statData)
				}
			}
		}
	}

	stat.HP = int(math.Max(float64(stat.HP-damage), 0)) // deal damage
	if stat.HP <= 0 {
		ai.TargetPlayerID = 0
	}

	if character.Meditating {
		resp := MEDITATION_MODE
		resp.Insert(utils.IntToBytes(uint64(character.PseudoID), 2, true), 6) // character pseudo id
		resp[8] = 0
		character.Meditating = false
		character.Socket.Write(resp)
	}

	index := 5
	resp.Insert(utils.IntToBytes(uint64(character.PseudoID), 2, true), index) // character pseudo id
	index += 2
	resp.Insert(utils.IntToBytes(uint64(ai.PseudoID), 2, true), index) // mob pseudo id
	index += 2
	resp.Insert(utils.IntToBytes(uint64(stat.HP), 4, true), index) // character hp
	index += 4
	resp.Insert(utils.IntToBytes(uint64(stat.CHI), 4, true), index) // character chi
	index += 4

	resp.Concat(character.GetHPandChi())
	return resp
}

func (ai *AI) DealDamageToPet(damage int) []byte {

	resp := MOB_DEAL_DAMAGE
	pet, ok := GetFromRegister(ai.Server, ai.Map, uint16(ai.TargetPetID)).(*PetSlot)
	if !ok || pet == nil || !pet.IsOnline || pet.HP <= 0 {
		return nil
	}

	pet.HP = int(math.Max(float64(pet.HP-damage), 0)) // deal damage
	pet.RefreshStats = true
	if pet.HP <= 0 {
		ai.TargetPetID = 0
	}

	resp.Insert(utils.IntToBytes(uint64(pet.PseudoID), 2, true), 5) // pet pseudo id
	resp.Insert(utils.IntToBytes(uint64(ai.PseudoID), 2, true), 7)  // mob pseudo id
	resp.Insert(utils.IntToBytes(uint64(pet.HP), 2, true), 9)       // pet hp
	resp.Insert(utils.IntToBytes(uint64(pet.CHI), 2, true), 11)     // pet chi
	resp.SetLength(0x24)

	return resp
}

func (ai *AI) MovementHandler(token int64, start, end *utils.Location, speed float64) {

	diff := utils.CalculateDistance(start, end)

	if diff < 1 {
		ai.SetCoordinate(end)
		ai.MovementToken = 0
		ai.IsMoving = false
		return
	}

	ai.SetCoordinate(start)
	ai.TargetLocation = *end

	r := []byte{}
	if speed == ai.RunningSpeed {
		r = ai.Move(*end, 2)
	} else {
		r = ai.Move(*end, 1)
	}

	p := nats.CastPacket{CastNear: true, MobID: ai.ID, Data: r, Type: nats.MOB_MOVEMENT}
	p.Cast()

	if diff <= speed { // target is so close
		*start = *end
		time.AfterFunc(time.Duration(diff/speed)*time.Millisecond, func() {
			if token == ai.MovementToken {
				ai.MovementHandler(token, start, end, speed)
			}
		})
	} else { // target is away
		start.X += (end.X - start.X) * speed / diff
		start.Y += (end.Y - start.Y) * speed / diff
		time.AfterFunc(1000*time.Millisecond, func() {
			if token == ai.MovementToken {
				ai.MovementHandler(token, start, end, speed)
			}
		})
	}
}

func (ai *AI) FindClaimer() (*Character, error) {
	dealers := ai.DamageDealers.Values()
	sort.Slice(dealers, func(i, j int) bool {
		di := dealers[i].(*Damage)
		dj := dealers[j].(*Damage)
		return di.Damage > dj.Damage
	})

	if len(dealers) == 0 {
		return nil, nil
	}

	return FindCharacterByID(dealers[0].(*Damage).DealerID)
}

func (ai *AI) DropHandler(claimer *Character) {

	var (
		err error
	)

	npcPos := NPCPos[ai.PosID]
	if npcPos == nil {
		return
	}

	npc := NPCs[npcPos.NPCID]
	if npc == nil {
		return
	}

	diff := claimer.Level - int(npc.Level)

	if diff > 30 {
		if npc.ID == 9999991 || npc.ID == 9999992 || npc.ID == 9999993 || npc.ID == 9999994 || npc.ID == 9999995 || npc.ID == 9999996 || npc.ID == 9999997 || npc.ID == 9999998 || npc.ID == 18600007 || npc.ID == 50071 || npc.ID == 1338007 || npc.ID == 1338006 || npc.ID == 30026 || npc.ID == 30007 {
		} else {
			return
		}
	}

	//isEventBoss := false
	bossMultiplier, dropCount, count, minCount := 0.0, 0, 0, 0
	baseLocation := ConvertPointToLocation(ai.Coordinate)

	if funk.Contains(bosses, npc.ID) {
		bossMultiplier = 2.5
		minCount = 3

	} else if funk.Contains(eventBosses, npc.ID) {
		bossMultiplier = 5.0
		minCount = 10
		//isEventBoss = true

	} else if !npcPos.Attackable && claimer.PickaxeActivated() {
		bossMultiplier = 0.4
	} else if npc.ID == 50071 {
		minCount = 1
	} else if npc.ID == 1338006 || npc.ID == 1338007 || npc.ID == 18600007 { // Red Dragon ve Rakma drop
		minCount = 70
	} /*else if npc.ID == 18600034 || npc.ID == 18600035 || npc.ID == 18600036 || npc.ID == 18600037 {
		isEventBoss = true
	}*/

BEGIN:
	id := npc.DropID

	drop, ok := Drops[id]
	if !ok || drop == nil {
		// Dark Quest Items
		if npc.ID == 43301 {
			questItem := &InventorySlot{ItemID: 18500573, Quantity: 1}
			rc, _, err := claimer.AddItem(questItem, -1, false)
			if err == nil {
				claimer.Socket.Write(*rc)
				return
			}
		}
		if npc.ID == 43302 {
			questItem := &InventorySlot{ItemID: 18500574, Quantity: 1}
			rc, _, err := claimer.AddItem(questItem, -1, false)
			if err == nil {
				claimer.Socket.Write(*rc)
				return
			}
		}
		if npc.ID == 43402 {
			questItem := &InventorySlot{ItemID: 18500575, Quantity: 1}
			rc, _, err := claimer.AddItem(questItem, -1, false)
			if err == nil {
				claimer.Socket.Write(*rc)
				return
			}
		}
		if npc.ID == 43403 {
			questItem := &InventorySlot{ItemID: 18500576, Quantity: 1}
			rc, _, err := claimer.AddItem(questItem, -1, false)
			if err == nil {
				claimer.Socket.Write(*rc)
				return
			}
		}
		if npc.ID == 43401 {
			questItem := &InventorySlot{ItemID: 18500571, Quantity: 1}
			rc, _, err := claimer.AddItem(questItem, -1, false)
			if err == nil {
				claimer.Socket.Write(*rc)
				return
			}
		}
		if npc.ID == 43206 {
			if claimer.Level == 200 {
				questItem := &InventorySlot{ItemID: 18500572, Quantity: 1}
				rc, _, err := claimer.AddItem(questItem, -1, false)
				if err == nil {
					claimer.Socket.Write(*rc)
					return
				}
			}
		}

		if npc.ID == 1338002 {
			if claimer.Level == 100 {
				questItem := &InventorySlot{ItemID: 13370223, Quantity: 1}
				rc, _, err := claimer.AddItem(questItem, -1, false)
				if err == nil {
					claimer.Socket.Write(*rc)
					return
				}
			}
		}
		if npc.ID == 1338003 {
			if claimer.Level == 100 {
				questItem := &InventorySlot{ItemID: 13370224, Quantity: 1}
				rc, _, err := claimer.AddItem(questItem, -1, false)
				if err == nil {
					claimer.Socket.Write(*rc)
					return
				}
			}
		}
		if npc.ID == 1338004 {
			if claimer.Level == 100 {
				questItem := &InventorySlot{ItemID: 13370225, Quantity: 1}
				rc, _, err := claimer.AddItem(questItem, -1, false)
				if err == nil {
					claimer.Socket.Write(*rc)
					return
				}
			}
		}
		if npc.ID == 1338005 {
			if claimer.Level == 100 {
				questItem := &InventorySlot{ItemID: 13370226, Quantity: 1}
				rc, _, err := claimer.AddItem(questItem, -1, false)
				if err == nil {
					claimer.Socket.Write(*rc)
					return
				}
			}
		}

		if npc.ID == 1338001 {
			if claimer.Level == 100 {
				questItem := &InventorySlot{ItemID: 13370222, Quantity: 1}
				rc, _, err := claimer.AddItem(questItem, -1, false)
				if err == nil {
					claimer.Socket.Write(*rc)
					return
				}
			}
		}
		return
	}

	itemID := 0
	end := false
	for ok {
		index := 0
		seed := int(utils.RandInt(0, 1000))
		items := drop.GetItems()

		probabilities := drop.GetProbabilities()

		var totalDropRate float64

		switch claimer.Map {
		case 5: // Spirit
			totalDropRate = DROP_RATE*(0.5*(claimer.DropMultiplier+claimer.AdditionalDropMultiplier)) + bossMultiplier
		case 7: // Southern
			totalDropRate = DROP_RATE*(0.5*(claimer.DropMultiplier+claimer.AdditionalDropMultiplier)) + bossMultiplier
		case 9: // Stone
			totalDropRate = DROP_RATE*(0.6*(claimer.DropMultiplier+claimer.AdditionalDropMultiplier)) + bossMultiplier
		case 11: // Silen Valley
			totalDropRate = DROP_RATE*(0.6*(claimer.DropMultiplier+claimer.AdditionalDropMultiplier)) + bossMultiplier
		case 13: // Crimson Sky
			totalDropRate = DROP_RATE*(0.5*(claimer.DropMultiplier+claimer.AdditionalDropMultiplier)) + bossMultiplier
		case 16: // Crystal Summit
			totalDropRate = DROP_RATE*(0.7*(claimer.DropMultiplier+claimer.AdditionalDropMultiplier)) + bossMultiplier
		case 17: // Tranquil Valley
			totalDropRate = DROP_RATE*(0.5*(claimer.DropMultiplier+claimer.AdditionalDropMultiplier)) + bossMultiplier
		case 18: // Desert Temple
			totalDropRate = DROP_RATE*(0.6*(claimer.DropMultiplier+claimer.AdditionalDropMultiplier)) + bossMultiplier
		case 19: // Red Dragon
			totalDropRate = DROP_RATE*(0.7*(claimer.DropMultiplier+claimer.AdditionalDropMultiplier)) + bossMultiplier
		case 193: // Desert Mangook
			totalDropRate = DROP_RATE*(0.8*(claimer.DropMultiplier+claimer.AdditionalDropMultiplier)) + bossMultiplier
		case 194: // Red Mangook
			totalDropRate = DROP_RATE*(0.9*(claimer.DropMultiplier+claimer.AdditionalDropMultiplier)) + bossMultiplier
		case 200: // Desert Pawang
			totalDropRate = DROP_RATE*(0.8*(claimer.DropMultiplier+claimer.AdditionalDropMultiplier)) + bossMultiplier
		case 201: // Red Pawang
			totalDropRate = DROP_RATE*(0.9*(claimer.DropMultiplier+claimer.AdditionalDropMultiplier)) + bossMultiplier
		case 28: // Tiger
			totalDropRate = DROP_RATE*(0.7*(claimer.DropMultiplier+claimer.AdditionalDropMultiplier)) + bossMultiplier
			/*
				case 108: // Desert Combat Base
					totalDropRate = DROP_RATE*(0.6*(claimer.DropMultiplier+claimer.AdditionalDropMultiplier)) + bossMultiplier
				case 193: // Mangook
					totalDropRate = DROP_RATE*(0.6*(claimer.DropMultiplier+claimer.AdditionalDropMultiplier)) + bossMultiplier
				case 194: // Mangook
					totalDropRate = DROP_RATE*(0.6*(claimer.DropMultiplier+claimer.AdditionalDropMultiplier)) + bossMultiplier
				case 200: // Pawang
					totalDropRate = DROP_RATE*(0.6*(claimer.DropMultiplier+claimer.AdditionalDropMultiplier)) + bossMultiplier
				case 201: // Pawang
					totalDropRate = DROP_RATE*(0.6*(claimer.DropMultiplier+claimer.AdditionalDropMultiplier)) + bossMultiplier
			*/
		default:
			totalDropRate = (DROP_RATE * (claimer.DropMultiplier + claimer.AdditionalDropMultiplier)) + bossMultiplier
		}

		if npc.ID == 50071 {
			// Snowman
			totalDropRate = (DROP_RATE * (claimer.DropMultiplier + claimer.AdditionalDropMultiplier)) + bossMultiplier
		}

		/*
			if claimer.Map == 16 {
				totalDropRate = DROP_RATE*(0.4*(claimer.DropMultiplier+claimer.AdditionalDropMultiplier)) + bossMultiplier
			} else {
				totalDropRate = (DROP_RATE * (claimer.DropMultiplier + claimer.AdditionalDropMultiplier)) + bossMultiplier
			}
		*/

		/*
			if claimer.Map == 13 || claimer.Map == 17 || claimer.Map == 18 || claimer.Map == 19 || claimer.Map == 28 || claimer.Map == 193 || claimer.Map == 194 {
				totalDropRate = DROP_RATE*(0.6*(claimer.DropMultiplier+claimer.AdditionalDropMultiplier)) + bossMultiplier
			} else if claimer.Map == 16 {
				totalDropRate = DROP_RATE*(0.4*(claimer.DropMultiplier+claimer.AdditionalDropMultiplier)) + bossMultiplier
			} else {
				totalDropRate = (DROP_RATE * (claimer.DropMultiplier + claimer.AdditionalDropMultiplier)) + bossMultiplier
			}
		*/

		//totalDropRate := (DROP_RATE * (claimer.DropMultiplier + claimer.AdditionalDropMultiplier)) + bossMultiplier
		dropFailRate := float64(1000 - probabilities[len(probabilities)-1])
		dropFailRate /= totalDropRate
		newDropFailRate := 1000 - dropFailRate
		probMultiplier := float64(probabilities[len(probabilities)-1]) / newDropFailRate

		if float64(probabilities[len(probabilities)-1])*totalDropRate < 900 {
			probMultiplier = 1
			probabilities = funk.Map(probabilities, func(prob int) int {
				return int(float64(prob) * totalDropRate)
			}).([]int)
		}

		/*
			for _, prob := range probabilities {
				if float64(seed)*probMultiplier > float64(prob) {
					index++
					continue
				}
				break
			}
		*/
		/*
			probabilities = funk.Map(probabilities, func(prob int) int {
				return int(float64(prob) / probMultiplier)
			}).([]int)
		*/

		seed = int(float64(seed) * probMultiplier)
		index = sort.SearchInts(probabilities, seed)
		if index >= len(items) {
			if count >= minCount {
				end = true
				break
			} else {
				drop = Drops[id]
				continue
			}
		}

		itemID = items[index]
		item, exist := Items[int64(itemID)]

		if exist {
			itemType := item.GetType()
			if itemType == QUEST_TYPE || itemType == INGREDIENTS_TYPE {
				drop = Drops[id]
				continue
			}
		}

		drop, ok = Drops[itemID]
	}

	if itemID > 0 && !end { // can drop an item
		count++

		if count >= 20 {
			return
		}

		if ai.Map == 1 || ai.Map == 2 || ai.Map == 3 {
			if itemID == 18500209 {
				itemID = 0
			}
		}

		go func() {

			if itemID == 0 {
				return
			}

			if itemID == 13370000 || itemID == 13370001 || itemID == 13370002 {
				if DRAGON_BOX == 0 {
					return
				}
			}

			resp := utils.Packet{}
			isRelic := false

			if tmpRelic, ok := Relics[itemID]; ok { // relic drop

				requiredItems := tmpRelic.GetRequiredItems()
				slots, err := claimer.InventorySlots()
				if err != nil {
					return
				}

				for i := range requiredItems {
					if int(slots[Items[int64(requiredItems[i])].Slot].ItemID) == requiredItems[i] {
						if tmpRelic.Count < tmpRelic.Limit {
							tmpRelic.Count++
							tmpRelic.Update()
							resp.Concat(claimer.RelicDrop(int64(itemID)))
							isRelic = true
						}
					}
				}
				/*

					if tmpRelic.Count < tmpRelic.Limit {
						tmpRelic.Count++
						tmpRelic.Update()
						resp.Concat(claimer.RelicDrop(int64(itemID)))
						isRelic = true
					}
				*/

				if !isRelic {
					return
				}
			}

			item := Items[int64(itemID)]
			if item != nil {

				/*if item.Type == 70 || item.Type == 71 {
					return
				}*/

				seed := int(utils.RandInt(0, 1000))
				plus := byte(0)
				for i := 0; i < len(plusRates) && !isRelic; i++ {
					if seed > plusRates[i] {
						plus++
						continue
					}
					break
				}

				drop := NewSlot()
				drop.ItemID = item.ID
				drop.Quantity = 1
				if item.ID == 242 {
					drop.Plus = 1
				} else {
					drop.Plus = plus
				}

				if item.Timer > 0 {
					drop.Quantity = uint(item.Timer)
				}

				var upgradesArray []byte
				itemType := item.GetType()
				if itemType == WEAPON_TYPE {
					upgradesArray = WeaponUpgrades
				} else if itemType == ARMOR_TYPE {
					upgradesArray = ArmorUpgrades
				} else if itemType == ACC_TYPE {
					if item.ID == 18500069 || item.ID == 18500070 || item.ID == 18500071 || item.ID == 18500072 {
						drop.Plus = 0
						plus = 0
					}
					upgradesArray = AccUpgrades
				} else if itemType == PENDENT_TYPE || item.ID == 254 || item.ID == 255 {
					if plus == 0 {
						plus = 1
						drop.Plus = 1
					}
					upgradesArray = []byte{byte(item.ID)}

				} else if itemType == SOCKET_TYPE {
					drop.ItemID = 235
					drop.Plus = socketOrePlus[item.ID]
					plus = socketOrePlus[item.ID]
					upgradesArray = []byte{235}

				} else {
					plus = 0
					drop.Plus = 0
				}

				for i := byte(0); i < plus; i++ {
					index := utils.RandInt(0, int64(len(upgradesArray)))
					drop.SetUpgrade(int(i), upgradesArray[index])
				}

				if isRelic || !npcPos.Attackable {

					slot := int16(-1)
					if npcPos.Attackable {
						slot, err = claimer.FindFreeSlot()
						if slot == 0 || err != nil {
							return
						}
					}

					data, _, err := claimer.AddItem(drop, slot, true)
					if err != nil || data == nil {
						return
					}

					if claimer != nil && claimer.Socket != nil {
						claimer.Socket.Write(*data)
						if isRelic {
							claimer.Socket.User.SaveRelicDrop(claimer.Name, item.Name, npc.Name, int(ai.Map), npc.ID, claimer.DropMultiplier+claimer.AdditionalDropMultiplier+DROP_RATE)
						}
					}
				} else {

					offset := dropOffsets[dropCount%len(dropOffsets)]
					dropCount++

					dr := &Drop{Server: ai.Server, Map: ai.Map, Claimer: claimer, Item: drop,
						Location: utils.Location{X: baseLocation.X + offset.X, Y: baseLocation.Y + offset.Y}}

					/*
							if isEventBoss || ai.Map == 10 {
							//dr.Claimer = nil
						}
					*/

					dr.GenerateIDForDrop(ai.Server, ai.Map)

					dropID := uint16(dr.ID)

					if ai.Map == 10 || ai.Map == 243 {
						time.AfterFunc(FREEDROP_LIFETIME, func() { // remove drop after timeout
							//ai.RemoveDrop(ai.Server, ai.Map, dropID)
							characters, _ := dr.Claimer.GetNearbyCharacters()
							dr.Claimer = nil
							for _, chars := range characters {
								r := DROP_DISAPPEARED
								r.Insert(utils.IntToBytes(uint64(dropID), 2, true), 6) //drop id
								chars.Socket.Write(r)
								chars.OnSight.DropsMutex.Lock()
								delete(chars.OnSight.Drops, int(dropID))
								chars.OnSight.DropsMutex.Unlock()
								chars.Socket.Write(r)
							}
						})
					} else if ai.PosID == 5494 || ai.PosID == 5495 || ai.PosID == 5493 || ai.PosID == 5492 || ai.PosID == 5720 || ai.PosID == 5811 || ai.PosID == 5812 || ai.PosID == 5813 || ai.PosID == 5814 || ai.PosID == 5815 || ai.PosID == 5816 || ai.PosID == 5817 || ai.PosID == 5818 {
						time.AfterFunc(FREEDROP_LIFETIME, func() { // remove drop after timeout
							//ai.RemoveDrop(ai.Server, ai.Map, dropID)
							characters, _ := dr.Claimer.GetNearbyCharacters()
							dr.Claimer = nil
							for _, chars := range characters {
								r := DROP_DISAPPEARED
								r.Insert(utils.IntToBytes(uint64(dropID), 2, true), 6) //drop id
								chars.Socket.Write(r)
								chars.OnSight.DropsMutex.Lock()
								delete(chars.OnSight.Drops, int(dropID))
								chars.OnSight.DropsMutex.Unlock()
								chars.Socket.Write(r)
							}
						})
					} else {
						time.AfterFunc(DROP_LIFETIME, func() { // remove drop after timeout
							ai.RemoveDrop(ai.Server, ai.Map, dropID)
							characters, _ := dr.Claimer.GetNearbyCharacters()
							for _, chars := range characters {
								r := DROP_DISAPPEARED
								r.Insert(utils.IntToBytes(uint64(dropID), 2, true), 6) //drop id
								chars.Socket.Write(r)
								chars.OnSight.DropsMutex.Lock()
								delete(chars.OnSight.Drops, int(dropID))
								chars.OnSight.DropsMutex.Unlock()
								chars.Socket.Write(r)
							}
						})
					}

					r := ITEM_DROPPED
					r.Insert(utils.IntToBytes(uint64(dropID), 2, true), 6) // drop id

					r.Insert(utils.FloatToBytes(offset.X+baseLocation.X, 4, true), 10) // drop coordinate-x
					r.Insert(utils.FloatToBytes(offset.Y+baseLocation.Y, 4, true), 18) // drop coordinate-y

					r.Insert(utils.IntToBytes(uint64(itemID), 4, true), 22) // item id
					if drop.Plus > 0 {
						r[27] = 0xA2
						r.Insert(drop.GetUpgrades(), 32)                                  // item upgrades
						r.Insert([]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}, 47) // item sockets
						r.Insert(utils.IntToBytes(uint64(claimer.PseudoID), 2, true), 66) // claimer id
						r.SetLength(0x42)
					} else {
						r[27] = 0xA1
						r.Insert(utils.IntToBytes(uint64(claimer.PseudoID), 2, true), 36) // claimer id
						r.SetLength(0x24)
					}

					resp.Concat(r)
				}

				p := nats.CastPacket{CastNear: true, MobID: ai.ID, Data: resp, Type: nats.ITEM_DROP}
				if isRelic {
					p = nats.CastPacket{CastNear: false, Data: resp, Type: nats.ITEM_DROP}
				} else {
					p = nats.CastPacket{CastNear: true, MobID: ai.ID, Data: resp, Type: nats.BOSS_DROP}
				}

				if err := p.Cast(); err != nil {
					return
				}
			}
		}()
	}

	if !npcPos.Attackable {
		end = true
	}

	if !end {
		goto BEGIN
	}
}

func (ai *AI) RemoveDrop(server int, mapID int16, dropID uint16) {
	drMutex.RLock()
	_, ok := DropRegister[server][mapID][dropID]
	drMutex.RUnlock()

	if ok {
		drMutex.Lock()
		delete(DropRegister[server][mapID], dropID)
		drMutex.Unlock()
	}
}

func (ai *AI) AIHandler() {

	defer func() {
		if err := recover(); err != nil {
			log.Println(err)
			log.Println(ai.TargetPlayerID)
			return
		}
	}()
	/*
		if ai.ID == 127918 || ai.PosID == 5694 {
			fmt.Println("AI Handler Geldim")
		}
	*/

	if len(ai.OnSightPlayers) > 0 && ai.HP > 0 {
		timer := fmt.Sprintf("%s", time.Now().String())
		npcPos := NPCPos[ai.PosID]
		npc := NPCs[npcPos.NPCID]

		ai.PlayersMutex.RLock()
		ids := funk.Keys(ai.OnSightPlayers).([]int)
		ai.PlayersMutex.RUnlock()

		for _, id := range ids {
			remove := false

			c, err := FindCharacterByID(id)
			if err != nil || c == nil || !c.IsOnline || c.Map != ai.Map || c.Faction == ai.Faction {
				remove = true
			}

			if c != nil {
				user, err := FindUserByID(c.UserID)
				if err != nil || user == nil || user.ConnectedIP == "" || user.ConnectedServer == 0 || user.ConnectedServer != ai.Server {
					remove = true
				}
			}

			if remove {
				ai.PlayersMutex.Lock()
				delete(ai.OnSightPlayers, id)
				ai.PlayersMutex.Unlock()
			}
		}

		if ai.TargetPetID > 0 {
			pet, ok := GetFromRegister(ai.Server, ai.Map, uint16(ai.TargetPetID)).(*PetSlot)
			if !ok || pet == nil || !pet.IsOnline || pet.HP <= 0 {
				ai.TargetPetID = 0
			}
		}

		if ai.TargetPlayerID > 0 {
			c, err := FindCharacterByID(ai.TargetPlayerID)
			if err != nil || c == nil || !c.IsOnline || c.Socket == nil || c.Socket.Stats.HP <= 0 || c.Faction == ai.Faction {
				ai.TargetPlayerID = 0
			} else {
				slots, _ := c.InventorySlots()
				petSlot := slots[0x0A]
				pet := petSlot.Pet
				petInfo, ok := Pets[petSlot.ItemID]
				if pet != nil && ok && pet.IsOnline && !petInfo.Combat {
					ai.TargetPlayerID = 0
					ai.TargetPetID = petSlot.Pet.PseudoID
				}
			}
		}

		var err error
		if ai.TargetPetID == 0 && ai.TargetPlayerID == 0 { // gotta find a target

			ai.TargetPlayerID, err = ai.FindTargetCharacterID() // 50% chance to trigger
			if err != nil {
				log.Println("AIHandler FindTargetPlayer error:", err)
			}

			c, err := FindCharacterByID(ai.TargetPlayerID)
			if err != nil || c == nil || !c.IsOnline || c.Socket == nil || c.Socket.Stats.HP <= 0 || c.Faction == ai.Faction {
				ai.TargetPlayerID = 0
			}

			petSlot, err := ai.FindTargetPetID(ai.TargetPlayerID)
			if err != nil {
				log.Println("AIHandler FindTargetPet error:", err)
			}

			if petSlot != nil {
				pet := petSlot.Pet
				//petInfo, ok := Pets[petSlot.ItemID]
				character, _ := FindCharacterByID(ai.TargetPlayerID)
				if pet != nil && ai.TargetPlayerID > 0 && character.IsMounting {
					ai.TargetPlayerID = 0
					ai.TargetPetID = pet.PseudoID
				}
				seed := utils.RandInt(0, 1000)
				if pet != nil && seed > 420 {
					ai.TargetPlayerID = 0
					ai.TargetPetID = pet.PseudoID
				}
			}
		}

		if ai.TargetPlayerID > 0 || ai.TargetPetID > 0 {
			ai.IsMoving = false
		}

		if ai.IsMoving {
			goto OUT
		}

		if ai.TargetPlayerID == 0 && ai.TargetPetID == 0 { // Idle mode
			coordinate := ConvertPointToLocation(ai.Coordinate)
			minCoordinate := ConvertPointToLocation(npcPos.MinLocation)
			maxCoordinate := ConvertPointToLocation(npcPos.MaxLocation)

			if utils.RandInt(0, 1000) < 750 { // 75% chance to move
				ai.IsMoving = true

				targetX := utils.RandFloat(minCoordinate.X, maxCoordinate.X)
				targetY := utils.RandFloat(minCoordinate.Y, maxCoordinate.Y)
				target := utils.Location{X: targetX, Y: targetY}
				ai.TargetLocation = target

				//d := ai.Move(target, 1)
				//p := nats.CastPacket{CastNear: true, MobID: ai.ID, Data: d, Type: nats.MOB_MOVEMENT}
				//p.Cast()

				token := ai.MovementToken
				for token == ai.MovementToken {
					ai.MovementToken = utils.RandInt(1, math.MaxInt64)
				}

				go ai.MovementHandler(ai.MovementToken, coordinate, &target, ai.WalkingSpeed)

			}

		} else if ai.TargetPetID > 0 {
			pet, ok := GetFromRegister(ai.Server, ai.Map, uint16(ai.TargetPetID)).(*PetSlot)
			if !ok || pet == nil || !pet.IsOnline || pet.HP <= 0 {
				ai.TargetPetID = 0
				goto OUT
			}

			aiCoordinate := ConvertPointToLocation(ai.Coordinate)
			distance := utils.CalculateDistance(&pet.Coordinate, aiCoordinate)

			if ai.ShouldGoBack() || distance > 50 { // better to retreat
				ai.TargetPetID = 0
				ai.TargetPlayerID = 0
				ai.MovementToken = 0
				ai.IsMoving = false
				ai.HP = npc.MaxHp

			} else if distance <= 4 && pet.IsOnline && pet.HP > 0 { // attack
				seed := utils.RandInt(1, 1000)
				skillIds := npc.GetSkills()
				skillsCount := len(skillIds) - 1
				randomSkill := utils.RandInt(0, int64(skillsCount))
				_, ok := SkillInfos[skillIds[randomSkill]]

				r := utils.Packet{}
				if seed < 400 && ok {
					r.Concat(ai.CastSkillToPet())
				} else {
					r.Concat(ai.AttackPet())
				}

				p := nats.CastPacket{CastNear: true, MobID: ai.ID, Data: r, Type: nats.MOB_ATTACK}
				p.Cast()

			} else if distance > 3 && distance <= 50 { // chase
				ai.IsMoving = true
				target := GeneratePoint(&pet.Coordinate)
				ai.TargetLocation = target

				token := ai.MovementToken
				for token == ai.MovementToken {
					ai.MovementToken = utils.RandInt(1, math.MaxInt64)
				}

				go ai.MovementHandler(ai.MovementToken, aiCoordinate, &target, ai.RunningSpeed)

			}

		} else if ai.TargetPlayerID > 0 { // Target mode player
			character, err := FindCharacterByID(ai.TargetPlayerID)
			if err != nil || character == nil || (character != nil && (!character.IsOnline || character.Invisible)) || character.IsMounting {
				ai.HP = npc.MaxHp
				ai.TargetPlayerID = 0
				goto OUT
			}

			if character == nil {
				goto OUT
			}

			if character.Socket == nil {
				goto OUT
			}

			stat := character.Socket.Stats
			if stat == nil {
				goto OUT
			}

			characterCoordinate := ConvertPointToLocation(character.Coordinate)
			aiCoordinate := ConvertPointToLocation(ai.Coordinate)
			distance := utils.CalculateDistance(characterCoordinate, aiCoordinate)

			if ai.ShouldGoBack() || distance > 50 { // better to retreat
				ai.TargetPlayerID = 0
				ai.MovementToken = 0
				ai.IsMoving = false

			} else if distance <= 5 && character.IsActive && stat.HP > 0 { // attack
				seed := utils.RandInt(1, 1000)
				skillIds := npc.GetSkills()
				skillsCount := len(skillIds) - 1
				randomSkill := utils.RandInt(0, int64(skillsCount))
				_, ok := SkillInfos[skillIds[randomSkill]]

				r := utils.Packet{}
				if seed < 400 && ok {
					r.Concat(ai.CastSkill())
				} else {
					r.Concat(ai.Attack())
				}

				p := nats.CastPacket{CastNear: true, MobID: ai.ID, Data: r, Type: nats.MOB_ATTACK}
				p.Cast()

			} else if distance > 5 && distance <= 50 { // chase
				ai.IsMoving = true
				target := GeneratePoint(characterCoordinate)
				ai.TargetLocation = target

				token := ai.MovementToken
				for token == ai.MovementToken {
					ai.MovementToken = utils.RandInt(1, math.MaxInt64)
				}

				go ai.MovementHandler(ai.MovementToken, aiCoordinate, &target, ai.RunningSpeed)

			}
		}

		timer = fmt.Sprintf("%s -> %s", timer, time.Now().String())
		//log.Println(timer)
	}

OUT:
	delay := utils.RandFloat(1.0, 1.5) * 1000
	time.AfterFunc(time.Duration(delay)*time.Millisecond, func() {
		if ai.Handler != nil {
			ai.AIHandler()
		}
	})
}

func (ai *AI) ShouldGoBack() bool {

	npcPos := NPCPos[ai.PosID]
	aiMinCoordinate := ConvertPointToLocation(npcPos.MinLocation)
	aiMaxCoordinate := ConvertPointToLocation(npcPos.MaxLocation)
	aiCoordinate := ConvertPointToLocation(ai.Coordinate)

	if aiCoordinate.X >= aiMinCoordinate.X && aiCoordinate.X <= aiMaxCoordinate.X &&
		aiCoordinate.Y >= aiMinCoordinate.Y && aiCoordinate.Y <= aiMaxCoordinate.Y {
		return false
	}

	return true
}

func GeneratePoint(location *utils.Location) utils.Location {

	r := 2.0
	alfa := utils.RandFloat(0, 360)
	targetX := location.X + r*float64(math.Cos(alfa*math.Pi/180))
	targetY := location.Y + r*float64(math.Sin(alfa*math.Pi/180))

	return utils.Location{X: targetX, Y: targetY}
}

func CountYingYangMobs(Map int16) {
	var mobs *DungeonMobsCounter = new(DungeonMobsCounter)

	i, j, k, l := 0, 0, 0, 0

	mobs.BlackBanditsLeader = 1
	mobs.RogueKingsLeader = 1
	mobs.GhostWarriorKing = 1
	mobs.BeastMaster = 1
	mobs.Paechun = 1

	for _, mob := range AIsByMap[1][Map] {
		npcPos, _ := FindNPCPosByID(mob.PosID)
		if npcPos == nil {
			continue
		}
		if npcPos.NPCID == 60001 || npcPos.NPCID == 60002 || npcPos.NPCID == 60015 || npcPos.NPCID == 60016 {
			if !mob.IsDead {
				i++
			}
		} else if npcPos.NPCID == 60004 || npcPos.NPCID == 60018 {
			if !mob.IsDead {
				j++
			}
		} else if npcPos.NPCID == 60006 || npcPos.NPCID == 60007 || npcPos.NPCID == 60020 || npcPos.NPCID == 60021 {
			if !mob.IsDead {
				k++
			}
		} else if npcPos.NPCID == 60009 || npcPos.NPCID == 60010 || npcPos.NPCID == 60011 || npcPos.NPCID == 60012 ||
			npcPos.NPCID == 60023 || npcPos.NPCID == 60024 || npcPos.NPCID == 60025 || npcPos.NPCID == 60026 {
			if !mob.IsDead {
				l++
			}
		}

		mobs.BlackBandits = i
		mobs.Rogues = j
		mobs.Ghosts = k
		mobs.Animals = l
		YingYangMobsMutex.Lock()
		YingYangMobsCounter[Map] = mobs
		YingYangMobsMutex.Unlock()
	}
}
