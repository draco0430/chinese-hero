package player

import (
	"hero-server/database"
	"hero-server/nats"
	"hero-server/utils"
)

type (
	AidHandler struct{}
)

func (h *AidHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {

	if s.User == nil || s.Character == nil {
		s.Conn.Close()
		return nil, nil
	}

	if sale := database.FindSale(s.Character.PseudoID); sale != nil {
		return nil, nil
	}

	if s.User.ConnectedServer == 9 {
		return nil, nil
	}

	activated := data[5] == 1
	if len(data) > 20 {
		petfood1 := utils.BytesToInt(data[269:273], true)
		petfood1percent := utils.BytesToInt(data[265:269], true)
		petchi := utils.BytesToInt(data[281:285], true)
		petchipercent := utils.BytesToInt(data[277:281], true)
		s.Character.PlayerAidSettings = &database.AidSettings{PetChiItemID: petchi, PetChiPercent: uint(petchipercent), PetFood1ItemID: petfood1, PetFood1Percent: uint(petfood1percent)}
	}
	resp := utils.Packet{}
	if s.Character.HasAidBuff() && s.Character.AidTime < 60 {
		s.Character.AidTime = 60
		stData, _ := s.Character.GetStats()
		resp.Concat(stData)
	}

	s.Character.AidMode = activated

	resp.Concat(s.Character.AidStatus())

	p := &nats.CastPacket{CastNear: true, CharacterID: s.Character.ID, Data: s.Character.GetHPandChi()}
	p.Cast()

	return resp, nil
}
