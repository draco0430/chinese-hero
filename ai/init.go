package ai

import (
	"log"

	"hero-server/database"
	"hero-server/server"
)

func Init() {

	database.AIsByMap = make([]map[int16][]*database.AI, database.SERVER_COUNT+1)
	for s := 0; s <= database.SERVER_COUNT; s++ {
		database.AIsByMap[s] = make(map[int16][]*database.AI)
	}

	database.DungeonsByMap = make([]map[int16]int, database.SERVER_COUNT+1)
	for s := 0; s <= database.SERVER_COUNT; s++ {
		database.DungeonsByMap[s] = make(map[int16]int)
	}
	database.DungeonsAiByMap = make([]map[int16][]*database.AI, database.SERVER_COUNT+1)
	for s := 0; s <= database.SERVER_COUNT; s++ {
		database.DungeonsAiByMap[s] = make(map[int16][]*database.AI)
	}

	func() {
		<-server.Init

		var err error

		database.NPCPos, err = database.GetAllNPCPos()
		if err != nil {
			log.Println(err)
			return
		}

		for _, pos := range database.NPCPos {
			if pos.IsNPC && !pos.Attackable {
				server.GenerateIDForNPC(pos)
			}
		}

		database.NPCs, err = database.GetAllNPCs()
		if err != nil {
			log.Println(err)
			return
		}

		err = database.GetAllAI()
		if err != nil {
			log.Println(err)
			return
		}

		for _, ai := range database.AIs {
			database.AIsByMap[ai.Server][ai.Map] = append(database.AIsByMap[ai.Server][ai.Map], ai)
		}

		for _, AI := range database.AIs {
			if AI.ID == 0 {
				continue
			}

			// Puplar

			/*
				if AI.PosID == 5496 || AI.PosID == 5497 || AI.PosID == 5498 || AI.PosID == 5499 || AI.PosID == 5500 || AI.PosID == 5683 || AI.PosID == 5698 || AI.PosID == 5699 || AI.PosID == 5700 || AI.PosID == 5701 || AI.PosID == 5702 || AI.PosID == 5492 || AI.PosID == 5493 {
					continue
				}
			*/

			/*
				// Kurbanlık
				if AI.PosID == 5379 || AI.PosID == 5380 || AI.PosID == 5381 || AI.PosID == 5382 || AI.PosID == 5383 || AI.PosID == 5384 || AI.PosID == 5385 || AI.PosID == 5386 || AI.PosID == 5387 {
					continue
				}
			*/

			pos := database.NPCPos[AI.PosID]
			npc := database.NPCs[pos.NPCID]

			AI.TargetLocation = *database.ConvertPointToLocation(AI.Coordinate)
			AI.HP = npc.MaxHp
			AI.OnSightPlayers = make(map[int]interface{})
			AI.Handler = AI.AIHandler

			/*
				if npc.Level > 200 {
					continue
				}
			*/

			server.GenerateIDForAI(AI)
			if AI.ID == 55281 || AI.ID == 55283 || AI.ID == 55287 || AI.ID == 55289 || AI.ID == 55285 {
				newStone := &database.WarStone{PseudoID: AI.PseudoID, NpcID: pos.NPCID, NearbyZuhang: 0, NearbyShao: 0, ConquereValue: 100}
				database.WarStonesIDs = append(database.WarStonesIDs, AI.PseudoID)
				database.WarStones[int(AI.PseudoID)] = newStone
			}

			/*
				if AI.ID == 424201 {
					AI.Faction = 1
				}

				if AI.ID == 424202 {
					AI.Faction = 2
				}
			*/

			if AI.WalkingSpeed > 0 {
				go AI.Handler()
			}
		}
	}()
}
