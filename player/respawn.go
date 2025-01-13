package player

import (
	"fmt"
	"hero-server/database"
	"hero-server/nats"
	"hero-server/utils"
)

type RespawnHandler struct {
}

var ()

func (h *RespawnHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {

	if s.User == nil || s.Character == nil {
		s.Conn.Close()
		return nil, nil
	}

	resp := utils.Packet{}
	respawnType := data[5]

	stat := s.Stats
	if stat == nil {
		return nil, nil
	}

	switch respawnType {
	case 1: // Respawn at Safe Zone
		save := database.SavePoints[byte(s.Character.Map)]
		point := database.ConvertPointToLocation(save.Point)
		if s.Character.Map == 230 {
			if s.Character.Faction == 1 {
				x := 75.0
				y := 45.0
				point = database.ConvertPointToLocation(fmt.Sprintf("%.1f,%.1f", x, y))
			}

			if s.Character.Faction == 2 {
				x := 81.0
				y := 475.0
				point = database.ConvertPointToLocation(fmt.Sprintf("%.1f,%.1f", x, y))
			}
		}

		if s.Character.Map == 255 {
			if s.Character.Faction == 1 {
				x := 325.0
				y := 465.0
				point = database.ConvertPointToLocation(fmt.Sprintf("%.1f,%.1f", x, y))
			}

			if s.Character.Faction == 2 {
				x := 179.0
				y := 45.0
				point = database.ConvertPointToLocation(fmt.Sprintf("%.1f,%.1f", x, y))
			}
		}

		if s.Character.Map == 76 {
			if s.Character.Faction == 1 {
				x := 75.0
				y := 431.0
				point = database.ConvertPointToLocation(fmt.Sprintf("%.1f,%.1f", x, y))
			}

			if s.Character.Faction == 2 {
				x := 430.0
				y := 75.0
				point = database.ConvertPointToLocation(fmt.Sprintf("%.1f,%.1f", x, y))
			}
		}

		teleportData := s.Character.Teleport(point)
		resp.Concat(teleportData)

		s.Character.IsActive = false
		stat.HP = stat.MaxHP
		stat.CHI = stat.MaxCHI
		s.Character.Respawning = false
		hpData := s.Character.GetHPandChi()
		resp.Concat(hpData)

		p := nats.CastPacket{CastNear: true, CharacterID: s.Character.ID, Type: nats.PLAYER_RESPAWN}
		p.Cast()
		break
	case 4: // Respawn at Location
		if s.Character.Map != 230 && s.Character.Map != 255 && s.Character.Map != 233 && s.Character.Map != 243 && s.Character.Map != 76 {
			if s.Character.Gold > 5000000 {
				s.Character.Gold -= 5000000
				s.Character.IsActive = false
				stat.HP = stat.MaxHP / 10
				stat.CHI = stat.MaxCHI / 10
				s.Character.Respawning = false

				hpData := s.Character.GetHPandChi()
				resp.Concat(hpData)

				coordinate := database.ConvertPointToLocation(s.Character.Coordinate)
				teleportData := s.Character.Teleport(coordinate)
				resp.Concat(teleportData)

				h := GetGoldHandler{}
				goldData, _ := h.Handle(s)
				resp.Concat(goldData)
				//s.Conn.Write(resp)
				s.Write(resp)
				p := nats.CastPacket{CastNear: true, CharacterID: s.Character.ID, Type: nats.PLAYER_RESPAWN}
				p.Cast()
				break
			}
		}
		save := database.SavePoints[byte(s.Character.Map)]
		point := database.ConvertPointToLocation(save.Point)
		if s.Character.Map == 230 {
			if s.Character.Faction == 1 {
				x := 75.0
				y := 45.0
				point = database.ConvertPointToLocation(fmt.Sprintf("%.1f,%.1f", x, y))
			}

			if s.Character.Faction == 2 {
				x := 81.0
				y := 475.0
				point = database.ConvertPointToLocation(fmt.Sprintf("%.1f,%.1f", x, y))
			}
		}

		if s.Character.Map == 255 {
			if s.Character.Faction == 1 {
				x := 325.0
				y := 465.0
				point = database.ConvertPointToLocation(fmt.Sprintf("%.1f,%.1f", x, y))
			}

			if s.Character.Faction == 2 {
				x := 179.0
				y := 45.0
				point = database.ConvertPointToLocation(fmt.Sprintf("%.1f,%.1f", x, y))
			}
		}

		if s.Character.Map == 76 {
			if s.Character.Faction == 1 {
				x := 75.0
				y := 431.0
				point = database.ConvertPointToLocation(fmt.Sprintf("%.1f,%.1f", x, y))
			}

			if s.Character.Faction == 2 {
				x := 430.0
				y := 75.0
				point = database.ConvertPointToLocation(fmt.Sprintf("%.1f,%.1f", x, y))
			}
		}

		teleportData := s.Character.Teleport(point)
		resp.Concat(teleportData)
		s.Character.Injury = 0

		s.Character.IsActive = false
		stat.HP = stat.MaxHP
		stat.CHI = stat.MaxCHI
		s.Character.Respawning = false
		hpData := s.Character.GetHPandChi()
		resp.Concat(hpData)

		p := nats.CastPacket{CastNear: true, CharacterID: s.Character.ID, Type: nats.PLAYER_RESPAWN}
		p.Cast()
	}

	go s.Character.ActivityStatus(30)
	return resp, nil
}
