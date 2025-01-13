package player

import (
	"fmt"
	"log"
	"math"
	"time"

	"hero-server/database"
	"hero-server/messaging"
	"hero-server/nats"
	"hero-server/server"
	"hero-server/utils"

	"github.com/thoas/go-funk"
)

type (
	AttackHandler        struct{}
	InstantAttackHandler struct{}
	DealDamageHandler    struct{}
	CastSkillHandler     struct{}
	CastMonkSkillHandler struct{}
	RemoveBuffHandler    struct{}
)

var (
	ATTACKED      = utils.Packet{0xAA, 0x55, 0x0C, 0x00, 0x41, 0x01, 0x0D, 0x02, 0x01, 0x00, 0x00, 0x00, 0x55, 0xAA}
	INST_ATTACKED = utils.Packet{0xAA, 0x55, 0x0C, 0x00, 0x41, 0x01, 0x0D, 0x02, 0x01, 0x00, 0x00, 0x00, 0x55, 0xAA}

	PVP_DEAL_DAMAGE          = utils.Packet{0xAA, 0x55, 0x28, 0x00, 0x16, 0xFF, 0xFF, 0xFF, 0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xB8, 0x3B, 0x11, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x10, 0x27, 0x00, 0x00, 0x55, 0xAA}
	PVP_DEAL_CRITICAL_DAMAGE = utils.Packet{0xAA, 0x55, 0x28, 0x00, 0x16, 0xFF, 0xFF, 0xFF, 0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0xB8, 0x3B, 0x11, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x10, 0x27, 0x00, 0x00, 0x55, 0xAA}

	PVP_DEAL_SKILL_DAMAGE          = utils.Packet{0xAA, 0x55, 0x28, 0x00, 0x16, 0x42, 0x75, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xB8, 0x3B, 0x11, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x10, 0x27, 0x00, 0x00, 0x55, 0xAA}
	PVP_DEAL_SKILL_CRITICAL_DAMAGE = utils.Packet{0xAA, 0x55, 0x28, 0x00, 0x16, 0x42, 0x75, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0xB8, 0x3B, 0x11, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x10, 0x27, 0x00, 0x00, 0x55, 0xAA}
)

func (h *AttackHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {

	if s.User == nil || s.Character == nil {
		s.Conn.Close()
		return nil, nil
	}

	c := s.Character
	if c == nil {
		return nil, nil
	}

	/*
		if c.Floating {
			return nil, nil
		}
	*/

	st := s.Stats
	if st == nil {
		return nil, nil
	}

	if c.IsAttacked {
		return nil, nil
	}
	c.IsAttacked = true

	time.AfterFunc(time.Second, func() {
		c.IsAttacked = false
	})

	aiID := uint16(utils.BytesToInt(data[7:9], true))
	ai, ok := database.GetFromRegister(s.User.ConnectedServer, s.Character.Map, aiID).(*database.AI)
	if ok {
		if ai == nil || ai.HP <= 0 {
			return nil, nil
		}

		npcPos := database.NPCPos[ai.PosID]
		if npcPos == nil {
			return nil, nil
		}

		npc := database.NPCs[npcPos.NPCID]
		if npc == nil {
			return nil, nil
		}

		if npcPos.Attackable {
			aiCoordinate := database.ConvertPointToLocation(ai.Coordinate)
			chrCoordinate := database.ConvertPointToLocation(c.Coordinate)
			distance := utils.CalculateDistance(chrCoordinate, aiCoordinate)

			if distance > 5 {
				return nil, nil
			}

			if npc.ID == 18600038 {
				if c.Level > 100 {
					buff, err := database.FindBuffByID(56, c.ID)
					if err != nil {
						return nil, nil
					}
					if buff != nil {
						return nil, nil
					}

					buffinfo := database.BuffInfections[56]
					buff = &database.Buff{ID: 56, CharacterID: c.ID, Name: buffinfo.Name, BagExpansion: false, StartedAt: c.Epoch, Duration: int64(5) * 60}
					err = buff.Create()
					return nil, nil
				}
			}

			if c.IsinWar {
				if npc.ID == 424201 || npc.ID == 424202 {
					if !database.WarStarted {
						return nil, nil
					}
				}
			}

			ai.MovementToken = 0
			ai.IsMoving = false
			ai.TargetPlayerID = c.ID

			dmg, err := c.CalculateDamage(ai, false)
			if err != nil {
				return nil, err
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

			if diff := int(npc.Level) - c.Level; diff > 0 {
				reqAcc := utils.SigmaFunc(float64(diff))
				if float64(st.Accuracy) < reqAcc {
					probability := float64(st.Accuracy) * 1000 / reqAcc
					if utils.RandInt(0, 1000) > int64(probability) {
						dmg = 0
					}
				}
			}

			if npc.ID == 423308 || npc.ID == 423310 || npc.ID == 9999991 || npc.ID == 9999992 || npc.ID == 9999993 || npc.ID == 9999994 || npc.ID == 9999995 || npc.ID == 9999996 || npc.ID == 9999997 || npc.ID == 9999998 || npc.ID == 423312 || npc.ID == 423314 || npc.ID == 423316 || npc.ID == 18600047 || npc.ID == 18600048 || npc.ID == 18600049 || npc.ID == 18600050 || npc.ID == 18600051 || npc.ID == 43301 || npc.ID == 43302 || npc.ID == 43403 || npc.ID == 43402 || npc.ID == 43401 || npc.ID == 18600078 {
				dmg = 50
			}

			c.Targets = append(c.Targets, &database.Target{Damage: dmg, AI: ai})
		}

	} else if enemy := server.FindCharacter(s.User.ConnectedServer, aiID); enemy != nil {
		enemy := server.FindCharacter(s.User.ConnectedServer, aiID)
		if enemy == nil || !enemy.IsActive {
			return nil, nil
		}

		dmg, err := c.CalculateDamageToPlayer(enemy, false)
		if err != nil {
			return nil, err
		}

		c.PlayerTargets = append(c.PlayerTargets, &database.PlayerTarget{Damage: dmg, Enemy: enemy})
	}

	resp := ATTACKED
	resp[4] = data[4]
	resp.Insert(utils.IntToBytes(uint64(c.PseudoID), 2, true), 6) // character pseudo id
	resp.Insert(utils.IntToBytes(uint64(aiID), 2, true), 9)       // ai id

	p := &nats.CastPacket{CastNear: true, CharacterID: c.ID, Data: resp, Type: nats.MOB_ATTACK}
	if err := p.Cast(); err != nil {
		return nil, err
	}

	if c.Class == 34 {
		buff, err := database.FindBuffByID(50, c.ID)
		if err != nil {
			log.Println("İnvisible buff error: ", err)
			return nil, nil
		}

		if buff != nil {
			c.Invisible = false
			buff.Delete()
			r := database.BUFF_EXPIRED
			r.Insert(utils.IntToBytes(uint64(buff.ID), 4, true), 6) // buff infection id
			c.Socket.Write(r)
		}
	}

	return resp, nil
}

func (h *InstantAttackHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {

	if s.User == nil || s.Character == nil {
		s.Conn.Close()
		return nil, nil
	}

	c := s.Character
	if c == nil {
		return nil, nil
	}

	/*
		if c.Floating {
			return nil, nil
		}
	*/

	st := s.Stats
	if st == nil {
		return nil, nil
	}

	aiID := uint16(utils.BytesToInt(data[7:9], true))
	ai, ok := database.GetFromRegister(s.User.ConnectedServer, s.Character.Map, aiID).(*database.AI)
	if ok {
		if ai == nil || ai.HP <= 0 {
			return nil, nil
		}

		npcPos := database.NPCPos[ai.PosID]
		if npcPos == nil {
			return nil, nil
		}

		npc := database.NPCs[npcPos.NPCID]
		if npc == nil {
			return nil, nil
		}

		if npcPos.Attackable {
			if npc.ID == 18600038 {
				if c.Level > 100 {
					buff, err := database.FindBuffByID(56, c.ID)
					if err != nil {
						return nil, nil
					}
					if buff != nil {
						return nil, nil
					}

					buffinfo := database.BuffInfections[56]
					buff = &database.Buff{ID: 56, CharacterID: c.ID, Name: buffinfo.Name, BagExpansion: false, StartedAt: c.Epoch, Duration: int64(5) * 60}
					err = buff.Create()
					return nil, nil
				}
			}

			if c.IsinWar {
				if npc.ID == 424201 || npc.ID == 424202 {
					if !database.WarStarted {
						return nil, nil
					}
				}
			}

			ai.MovementToken = 0
			ai.IsMoving = false
			ai.TargetPlayerID = c.ID

			dmg := int(utils.RandInt(int64(st.MinATK), int64(st.MaxATK))) - npc.DEF
			if dmg < 0 {
				dmg = 0
			} else if dmg > ai.HP {
				dmg = ai.HP
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

			if diff := int(npc.Level) - c.Level; diff > 0 {
				reqAcc := utils.SigmaFunc(float64(diff))
				if float64(st.Accuracy) < reqAcc {
					probability := float64(st.Accuracy) * 1000 / reqAcc
					if utils.RandInt(0, 1000) > int64(probability) {
						dmg = 0
					}
				}
			}

			if npc.ID == 423308 || npc.ID == 423310 || npc.ID == 9999991 || npc.ID == 9999992 || npc.ID == 9999993 || npc.ID == 9999994 || npc.ID == 9999995 || npc.ID == 9999996 || npc.ID == 9999997 || npc.ID == 9999998 || npc.ID == 423312 || npc.ID == 423314 || npc.ID == 423316 || npc.ID == 18600047 || npc.ID == 18600048 || npc.ID == 18600049 || npc.ID == 18600050 || npc.ID == 18600051 || npc.ID == 43301 || npc.ID == 43302 || npc.ID == 43403 || npc.ID == 43402 || npc.ID == 43401 || npc.ID == 18600078 {
				dmg = 50
			}

			time.AfterFunc(time.Second/2, func() { // attack done
				go c.DealDamage(ai, dmg)
			})
		}
	} else if enemy := server.FindCharacter(s.User.ConnectedServer, aiID); enemy != nil {
		if enemy == nil || !enemy.IsActive {
			return nil, nil
		}

		dmg, err := c.CalculateDamageToPlayer(enemy, false)
		if err != nil {
			return nil, err
		}

		time.AfterFunc(time.Millisecond/100, func() { // attack done
			if c.CanAttack(enemy) {
				go DealDamageToPlayer(s, enemy, dmg)
			}
		})
	}

	resp := INST_ATTACKED
	resp[4] = data[4]

	resp.Insert(utils.IntToBytes(uint64(c.PseudoID), 2, true), 6) // character pseudo id
	resp.Insert(utils.IntToBytes(uint64(aiID), 2, true), 9)       // ai id

	p := &nats.CastPacket{CastNear: true, CharacterID: c.ID, Data: resp, Type: nats.MOB_ATTACK}
	if err := p.Cast(); err != nil {
		return nil, err
	}

	if c.Class == 34 {
		buff, err := database.FindBuffByID(50, c.ID)
		if err != nil {
			log.Println("İnvisible buff error: ", err)
			return nil, nil
		}

		if buff != nil {
			c.Invisible = false
			buff.Delete()
			r := database.BUFF_EXPIRED
			r.Insert(utils.IntToBytes(uint64(buff.ID), 4, true), 6) // buff infection id
			c.Socket.Write(r)
		}
	}

	return resp, nil
}

func (h *DealDamageHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {

	if s.User == nil || s.Character == nil {
		s.Conn.Close()
		return nil, nil
	}

	c := s.Character
	if c == nil {
		return nil, nil
	}

	/*
		if c.Floating {
			return nil, nil
		}
	*/

	resp := utils.Packet{}
	if c.TamingAI != nil {
		ai := c.TamingAI
		pos := database.NPCPos[ai.PosID]
		npc := database.NPCs[pos.NPCID]
		petInfo := database.Pets[int64(npc.ID)]

		seed := utils.RandInt(0, 1000)
		if seed < 250 && petInfo != nil {
			go c.DealDamage(ai, ai.HP)

			item := &database.InventorySlot{ItemID: int64(npc.ID), Quantity: 1}
			expInfo := database.PetExps[petInfo.Level-1]
			item.Pet = &database.PetSlot{
				Fullness: 100, Loyalty: 100,
				Exp:   uint64(expInfo.ReqExpEvo1),
				HP:    petInfo.BaseHP,
				Level: byte(petInfo.Level),
				Name:  petInfo.Name,
				CHI:   petInfo.BaseChi,
			}

			r, _, err := s.Character.AddItem(item, -1, true)
			if err != nil {
				return nil, err
			}

			resp.Concat(*r)
		}

		c.TamingAI = nil
		return resp, nil
	}

	targets := c.Targets
	dealt := make(map[int]struct{})
	for _, target := range targets {
		if target == nil {
			continue
		}

		ai := target.AI
		if ai == nil { // 01.12.2023 Bug Fix
			continue
		}
		if _, ok := dealt[ai.ID]; ok {
			continue
		}

		dmg := target.Damage

		go c.DealDamage(ai, dmg)
		dealt[ai.ID] = struct{}{}
	}

	pTargets := c.PlayerTargets
	dealt = make(map[int]struct{})
	for _, target := range pTargets {
		if target == nil {
			continue
		}

		enemy := target.Enemy
		if enemy == nil {
			continue
		}
		if _, ok := dealt[enemy.ID]; ok {
			continue
		}

		if c.CanAttack(enemy) {
			dmg := target.Damage
			go DealDamageToPlayer(s, enemy, dmg)
		}

		dealt[enemy.ID] = struct{}{}
	}

	c.Targets = []*database.Target{}
	c.PlayerTargets = []*database.PlayerTarget{}
	return nil, nil
}

func DealDamageToPlayer(s *database.Socket, enemy *database.Character, dmg int) {
	r := PVP_DEAL_SKILL_DAMAGE
	if s.User == nil || s.Character == nil {
		s.Conn.Close()
		return
	}

	c := s.Character
	enemySt := enemy.Socket.Stats

	if c == nil {
		log.Println("character is nil")
		return
	} else if enemySt.HP <= 0 {
		return
	}

	slots, err := c.InventorySlots()
	if err != nil {
		return
	}

	enemySlots, err := enemy.InventorySlots()
	if err != nil {
		return
	}

	weapon := slots[c.WeaponSlot]

	if weapon.ItemID == 99003803 || weapon.ItemID == 99003703 || weapon.ItemID == 99003603 || weapon.ItemID == 99003503 || weapon.ItemID == 99003403 || weapon.ItemID == 99003303 || weapon.ItemID == 99003203 || weapon.ItemID == 99003103 || weapon.ItemID == 99002303 || weapon.ItemID == 99002103 || weapon.ItemID == 99002203 {
		seed := utils.RandInt(0, 1000) ///CRITICAL CHANCE
		if seed <= int64(80) {
			critical := dmg + int(float32(dmg)*1.4)
			dmg = critical
			r = PVP_DEAL_SKILL_CRITICAL_DAMAGE

			buffinfo := database.BuffInfections[265]
			buff := database.Buff{ID: int(265), CharacterID: enemy.ID, Name: buffinfo.Name, BagExpansion: false, StartedAt: enemy.Epoch, Duration: int64(2) * 2}
			buff.Create()
		}
	}

	if weapon.ItemID == 99100240 || weapon.ItemID == 99100241 || weapon.ItemID == 99100242 || weapon.ItemID == 99100243 || weapon.ItemID == 99100244 || weapon.ItemID == 99100245 || weapon.ItemID == 99100246 || weapon.ItemID == 99100247 || weapon.ItemID == 99100248 || weapon.ItemID == 99100249 || weapon.ItemID == 99100250 {
		seed := utils.RandInt(0, 1000) ///CRITICAL CHANCE
		if seed <= int64(120) {
			critical := dmg + int(float32(dmg)*1.6)
			dmg = critical
			r = PVP_DEAL_SKILL_CRITICAL_DAMAGE

			buffinfo := database.BuffInfections[265]
			buff := database.Buff{ID: int(265), CharacterID: enemy.ID, Name: buffinfo.Name, BagExpansion: false, StartedAt: enemy.Epoch, Duration: int64(2) * 2}
			buff.Create()
		}
	}

	if s.Character.Invisible {
		buff, _ := database.FindBuffByID(241, s.Character.ID)
		if buff != nil {
			buff.Duration = 0
			go buff.Update()
		}

		buff, _ = database.FindBuffByID(244, s.Character.ID)
		if buff != nil {
			buff.Duration = 0
			go buff.Update()
		}

		buff, _ = database.FindBuffByID(138, s.Character.ID)
		if buff != nil {
			buff.Duration = 0
			go buff.Update()
		}

		buff, _ = database.FindBuffByID(139, s.Character.ID)
		if buff != nil {
			buff.Duration = 0
			go buff.Update()
		}
	}

	if enemy.Meditating { //STOP MEDITATION
		enemy.Meditating = false
		med := MEDITATION_MODE
		med.Insert(utils.IntToBytes(uint64(enemy.PseudoID), 2, true), 6) // character pseudo id
		med[8] = 0

		p := nats.CastPacket{CastNear: true, CharacterID: enemy.ID, Type: nats.MEDITATION_MODE, Data: med}
		if err := p.Cast(); err == nil {
			enemy.Socket.Write(med)
		}
	}

	enemyHelm := enemySlots[0]
	enemyMask := enemySlots[1]
	enemyArmor := enemySlots[2]
	enemyBoots := enemySlots[9]

	enemyAcc1 := enemySlots[5]
	enemyAcc2 := enemySlots[6]
	enemyAcc3 := enemySlots[7]
	enemyAcc4 := enemySlots[8]

	reflection := 0
	dmgAbsort := 0

	if enemyHelm.ItemID == 99004003 || enemyHelm.ItemID == 99100251 || enemyHelm.ItemID == 99004013 || enemyHelm.ItemID == 99100252 {
		reflection = reflection + 3
	}

	if enemyMask.ItemID == 99009001 || enemyMask.ItemID == 99100257 || enemyMask.ItemID == 99009011 || enemyMask.ItemID == 99100258 {
		reflection = reflection + 3
	}

	if enemyArmor.ItemID == 99004103 || enemyArmor.ItemID == 99100253 || enemyArmor.ItemID == 99004113 || enemyArmor.ItemID == 99100254 {
		reflection = reflection + 3
	}

	if enemyBoots.ItemID == 99004203 || enemyBoots.ItemID == 99100255 || enemyBoots.ItemID == 99004213 || enemyArmor.ItemID == 99100256 {
		reflection = reflection + 3
	}

	if reflection > 0 {
		seed := utils.RandInt(0, 1000) ///REFLECTION
		if seed <= int64(reflection*10) {
			reflectionDmg := (80 / 100.0) * float32(dmg)
			dmg = int(reflectionDmg)

			buffinfo := database.BuffInfections[221]
			buff := database.Buff{ID: int(221), CharacterID: enemy.ID, Name: buffinfo.Name, BagExpansion: false, StartedAt: enemy.Epoch, Duration: int64(2) * 2}
			buff.Create()
		}
	}

	divineReflection, err := database.FindBuffByID(213, enemy.ID)
	if err == nil && divineReflection != nil {
		seed := utils.RandInt(0, 1000) /// REFLECTION
		if seed <= int64(150) {
			reflectionDmg := (52 / 100.0) * float32(dmg)

			c.Socket.Stats.HP -= int(reflectionDmg)
			dmg = 3

			buffinfo := database.BuffInfections[88]
			buff := database.Buff{ID: int(88), CharacterID: enemy.ID, Name: buffinfo.Name, BagExpansion: false, StartedAt: enemy.Epoch, Duration: int64(2) * 2}
			buff.Create()
		}
	}

	if enemyAcc1.ItemID == 30023311 {
		dmgAbsort += 2
	}

	if enemyAcc2.ItemID == 30023111 {
		dmgAbsort += 2
	}

	if enemyAcc3.ItemID == 30023211 {
		dmgAbsort += 2
	}

	if enemyAcc4.ItemID == 30023011 {
		dmgAbsort += 2
	}

	if dmgAbsort > 0 {
		seed := utils.RandInt(0, 1000) ///DMG ABSORT
		if seed <= int64(dmgAbsort*10) {
			absorbedDmg := (30 / 100.0) * float32(dmg)
			dmg = dmg - int(absorbedDmg)

			buffinfo := database.BuffInfections[103]
			buff := database.Buff{ID: int(103), CharacterID: enemy.ID, Name: buffinfo.Name, BagExpansion: false, StartedAt: enemy.Epoch, Duration: int64(2) * 2}
			buff.Create()
		}
	}

	if c.Map == 12 && enemy.Map == 12 {

		different := int(c.Level - enemy.Level)
		if math.Signbit(float64(different)) {
			different = int(math.Abs(float64(different)))
		}

		if different >= 20 {
			dmg = 0
		}
	}

	enemySt.HP -= dmg
	if enemySt.HP < 0 {
		enemySt.HP = 0
	}

	r.Insert(utils.IntToBytes(uint64(enemy.PseudoID), 2, true), 5) // character pseudo id
	r.Insert(utils.IntToBytes(uint64(c.PseudoID), 2, true), 7)     // mob pseudo id
	r.Insert(utils.IntToBytes(uint64(enemySt.HP), 4, true), 9)     // character hp
	r.Insert(utils.IntToBytes(uint64(enemySt.CHI), 4, true), 13)   // character chi

	//	r.Insert([]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}, 21)
	//	r.Overwrite([]byte{0xFF, 0xFF, 0x00, 0x00}, 17)

	/*
		if enemy.AidMode {
			enemy.Targets = []*database.Target{}
			enemy.PlayerTargets = append(enemy.PlayerTargets, &database.PlayerTarget{Enemy: c})
			enemy.Selection = c.ID
		}
	*/

	//enemy.DealPlayerAttack = true
	r.Concat(enemy.GetHPandChi())
	/*
		time.AfterFunc(3*time.Second, func() {
			enemy.DealPlayerAttack = false
		})
	*/

	//r.SetLength(int16(binary.Size(r) - 6))

	//r.Print()

	p := &nats.CastPacket{CastNear: true, CharacterID: enemy.ID, Data: r, Type: nats.PLAYER_ATTACK}
	if err := p.Cast(); err != nil {
		log.Println("deal damage broadcast error:", err)
		return
	}

	if enemySt.HP <= 0 {
		enemySt.HP = 0
		enemy.Socket.Write(enemy.GetHPandChi())
		info := fmt.Sprintf("[%s] has defeated [%s]", c.Name, enemy.Name)
		r := messaging.InfoMessage(info)
		if database.WarStarted && c.IsinWar && enemy.IsinWar {
			c.WarKillCount++
			if c.Faction == 1 {
				database.ShaoPoints -= 5
			} else {
				database.OrderPoints -= 5
			}
		}

		if funk.Contains(database.LoseEXPServers, int16(c.Socket.User.ConnectedServer)) && funk.Contains(database.LoseEXPServers, int16(enemy.Socket.User.ConnectedServer)) && !c.IsinWar && !enemy.IsinWar {
			if s.Character.Level < 101 && enemy.Level < 101 /* && s.Character.RebornLevel == enemy.RebornLevel */ {
				database.MakeAnnouncement("[" + s.Character.Name + "] has slain [" + enemy.Name + "]")
				different := int(c.Level - enemy.Level)
				if math.Signbit(float64(different)) {
					different = int(math.Abs(float64(different)))
				}

				/*
					if different >= 0 && different <= 20 {
						exp, _ := enemy.LosePlayerExp(1)
						resp, levelUp := c.AddPlayerExp(exp) // pvp exp ayarı
						if levelUp {
							statData, err := c.GetStats()
							if err == nil {
								c.Socket.Write(statData)
							}
						}
						c.Socket.Write(resp)
					}
				*/
			}
		}

		if c.IsinLastMan && enemy.IsinLastMan {
			enemy.IsinLastMan = false
			database.LastManMutex.Lock()
			delete(database.LastManCharacters, enemy.ID)
			database.LastManMutex.Unlock()
		}

		if c.Map != 233 && c.Map != 250 && c.Map != 74 && c.DuelID != enemy.ID {
			if c.Level > 100 && enemy.Level <= 100 {
				buff, _ := database.FindBuffByID(56, c.ID)
				buffinfo := database.BuffInfections[56]
				buff = &database.Buff{ID: 56, CharacterID: c.ID, Name: buffinfo.Name, BagExpansion: false, StartedAt: c.Epoch, Duration: int64(60) * 60}
				buff.Create()
			}
		}

		// Tehlikeli
		if enemy.Socket.Stats.Honor > 9900 && enemy.Socket.Character.Level >= 75 && c.Level >= 75 {
			c.Socket.Stats.Honor += 10
			c.Socket.Stats.Update()
			enemySt.Honor -= 11
			enemySt.Update()
			c.Socket.Write(messaging.InfoMessage(fmt.Sprintf("You acquired 10 Honor points.")))
			enemy.Socket.Write(messaging.InfoMessage(fmt.Sprintf("You have lost 11 Honor points.")))
			stat, _ := c.GetStats()
			c.Socket.Write(stat)
		}

		p := nats.CastPacket{CastNear: true, CharacterID: c.ID, Data: r, Type: nats.PVP_FINISHED}
		p.Cast()
	}

}

func (h *CastSkillHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {

	if s.User == nil || s.Character == nil {
		s.Conn.Close()
		return nil, nil
	}

	if len(data) < 26 {
		return nil, nil
	}

	/*
		if s.Character.Floating {
			return nil, nil
		}
	*/

	attackCounter := int(data[6])
	skillID := int(utils.BytesToInt(data[7:11], true))
	cX := utils.BytesToFloat(data[11:15], true)
	cY := utils.BytesToFloat(data[15:19], true)
	cZ := utils.BytesToFloat(data[19:23], true)
	targetID := int(utils.BytesToInt(data[23:25], true))

	ai, ok := database.GetFromRegister(s.User.ConnectedServer, s.Character.Map, uint16(targetID)).(*database.AI)
	if ok {
		if ai == nil || ai.HP <= 0 {
			return nil, nil
		}

		npcPos := database.NPCPos[ai.PosID]
		if npcPos == nil {
			return nil, nil
		}

		npc := database.NPCs[npcPos.NPCID]
		if npc == nil {
			return nil, nil
		}

		if npcPos.Attackable {
			if npc.ID == 18600038 {
				if s.Character.Level > 100 {
					buff, err := database.FindBuffByID(56, s.Character.ID)
					if err != nil {
						return nil, nil
					}
					if buff != nil {
						return nil, nil
					}

					buffinfo := database.BuffInfections[56]
					buff = &database.Buff{ID: 56, CharacterID: s.Character.ID, Name: buffinfo.Name, BagExpansion: false, StartedAt: s.Character.Epoch, Duration: int64(5) * 60}
					err = buff.Create()
					return nil, nil
				}
			}

			if s.Character.IsinWar {
				if npc.ID == 424201 || npc.ID == 424202 {
					if !database.WarStarted {
						return nil, nil
					}
				}
			}

		}
	}

	if skillID == 41101 {
		return nil, nil
		/*
			s.Character.Floating = true
			time.AfterFunc(34*time.Second, func() {
				s.Character.Floating = false
			})
		*/
	}

	return s.Character.CastSkill(attackCounter, skillID, targetID, cX, cY, cZ)
}

func (h *CastMonkSkillHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {

	if s.User == nil || s.Character == nil {
		s.Conn.Close()
		return nil, nil
	}

	if len(data) < 26 {
		return nil, nil
	}

	attackCounter := 0x1B
	skillID := int(utils.BytesToInt(data[6:10], true))
	cX := utils.BytesToFloat(data[10:14], true)
	cY := utils.BytesToFloat(data[14:18], true)
	cZ := utils.BytesToFloat(data[18:22], true)
	targetID := int(utils.BytesToInt(data[22:24], true))

	/*
		if s.Character.Floating {
			return nil, nil
		}
	*/

	ai, ok := database.GetFromRegister(s.User.ConnectedServer, s.Character.Map, uint16(targetID)).(*database.AI)
	if ok {
		if ai == nil || ai.HP <= 0 {
			return nil, nil
		}

		npcPos := database.NPCPos[ai.PosID]
		if npcPos == nil {
			return nil, nil
		}

		npc := database.NPCs[npcPos.NPCID]
		if npc == nil {
			return nil, nil
		}

		if npcPos.Attackable {
			if s.Character.Level > 100 {
				if npc.ID == 18600038 {
					buff, err := database.FindBuffByID(56, s.Character.ID)
					if err != nil {
						return nil, nil
					}
					if buff != nil {
						return nil, nil
					}

					buffinfo := database.BuffInfections[56]
					buff = &database.Buff{ID: 56, CharacterID: s.Character.ID, Name: buffinfo.Name, BagExpansion: false, StartedAt: s.Character.Epoch, Duration: int64(5) * 60}
					err = buff.Create()
					return nil, nil
				}
			}

			if s.Character.IsinWar {
				if npc.ID == 424201 || npc.ID == 424202 {
					if !database.WarStarted {
						return nil, nil
					}
				}
			}
		}
	}

	if skillID == 41101 {
		return nil, nil
		/*
			s.Character.Floating = true
			time.AfterFunc(34*time.Second, func() {
				s.Character.Floating = false
			})
		*/
	}

	resp := utils.Packet{0xAA, 0x55, 0x16, 0x00, 0x49, 0x10, 0x55, 0xAA}
	resp.Insert(utils.IntToBytes(uint64(s.Character.PseudoID), 2, true), 6) // character pseudo id
	resp.Insert(utils.FloatToBytes(cX, 4, true), 8)                         // coordinate-x
	resp.Insert(utils.FloatToBytes(cY, 4, true), 12)                        // coordinate-y
	resp.Insert(utils.FloatToBytes(cZ, 4, true), 16)                        // coordinate-z
	resp.Insert(utils.IntToBytes(uint64(targetID), 2, true), 20)            // target pseudo id
	resp.Insert(utils.IntToBytes(uint64(skillID), 4, true), 22)             // skill id

	skill := database.SkillInfos[skillID]
	token := s.Character.MovementToken

	time.AfterFunc(time.Duration(skill.CastTime*1000)*time.Millisecond, func() {
		if token == s.Character.MovementToken {
			data, _ := s.Character.CastSkill(attackCounter, skillID, targetID, cX, cY, cZ)
			//s.Conn.Write(data)
			s.Write(data)
		}
	})

	return resp, nil
}

func (h *RemoveBuffHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {

	if s.User == nil || s.Character == nil {
		s.Conn.Close()
		return nil, nil
	}

	infectionID := int(utils.BytesToInt(data[6:10], true))
	buff, err := database.FindBuffByID(infectionID, s.Character.ID)
	if err != nil {
		return nil, err
	} else if buff == nil {
		return nil, nil
	}

	buff.Duration = 0
	go buff.Update()
	return nil, nil
}
