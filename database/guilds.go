package database

import (
	"database/sql"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"sync"

	"hero-server/utils"

	"github.com/thoas/go-funk"
	gorp "gopkg.in/gorp.v1"
)

var (
	Guilds = make(map[int]*Guild)
	gMutex sync.RWMutex

	GUILD_INFO = utils.Packet{0xAA, 0x55, 0x00, 0x00, 0x83, 0x09, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x55, 0xAA}

	GUILD_DATA = utils.Packet{0xAA, 0x55, 0x00, 0x00, 0x83, 0x0A, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x55, 0xAA}

	GUILD_MEMBER_INFO = utils.Packet{0xAA, 0x55, 0x00, 0x00, 0x83, 0x08, 0x00, 0x00, 0x10, 0x27, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x49,
		0x2A, 0x00, 0x04, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x0A, 0x32, 0x30, 0x31, 0x38, 0x2D,
		0x30, 0x38, 0x2D, 0x32, 0x36, 0x00, 0x55, 0xAA}
)

type GuildRole byte

const (
	GROLE_MEMBER GuildRole = iota + 1
	GROLE_BODYGUARD
	GROLE_SAGE
	GROLE_SOLDIER
	GROLE_LEADER
)

type Guild struct {
	ID            int             `db:"id" json:"id"`
	LeaderID      int             `db:"leader_id" json:"leader_id"`
	Name          string          `db:"name" json:"name"`
	MemberCount   int16           `db:"member_count" json:"member_count"`
	Members       json.RawMessage `db:"members" json:"members"`
	Logo          []byte          `db:"logo" json:"logo"`
	Description   string          `db:"description" json:"description"`
	Announcement  string          `db:"announcement" json:"announcement"`
	Faction       int16           `db:"faction" json:"faction"`
	GoldDonation  uint64          `db:"gold_donation" json:"gold_donation"`
	HonorDonation uint64          `db:"honor_donation" json:"honor_donation"`
	Recognition   uint64          `db:"recognition" json:"recognition"`

	mutex sync.RWMutex `db:"-"`
}

type GuildMember struct {
	ID   int       `json:"id"`
	Role GuildRole `json:"role"`
}

func (g *Guild) Create() error {
	return db.Insert(g)
}

func (g *Guild) CreateWithTransaction(tr *gorp.Transaction) error {
	return tr.Insert(g)
}

func (g *Guild) Update() error {
	_, err := db.Update(g)
	return err
}

func (g *Guild) Delete() error {
	gMutex.Lock()
	defer gMutex.Unlock()
	delete(Guilds, g.ID)

	_, err := db.Delete(g)
	return err
}

func (g *Guild) GetMembers() ([]*GuildMember, error) {
	members := []*GuildMember{}
	if g.Members == nil {
		return members, nil
	}

	err := json.Unmarshal(g.Members, &members)
	if err != nil {
		return nil, err
	}

	return members, nil
}

func (g *Guild) GetMember(id int) (*GuildMember, error) {
	members, err := g.GetMembers()
	if err != nil {
		return nil, err
	}

	for _, m := range members {
		if m.ID == id {
			return m, nil
		}
	}

	return nil, nil
}

func (g *Guild) SetMember(member *GuildMember) error {
	members, err := g.GetMembers()
	if err != nil {
		return err
	}

	for _, m := range members {
		if m.ID == member.ID {
			*m = *member
			break
		}
	}

	return g.SetMembers(members)
}

func (g *Guild) SetMembers(members []*GuildMember) error {
	data, err := json.Marshal(members)
	if err != nil {
		return err
	}

	g.Members = data
	return nil
}

func (g *Guild) AddMember(member *GuildMember) error {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	members, err := g.GetMembers()
	if err != nil {
		return err
	}

	members = append(members, member)
	err = g.SetMembers(members)
	if err != nil {
		return err
	}

	g.MemberCount = int16(len(members))
	return nil
}

func (g *Guild) RemoveMember(id int) error {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	members, err := g.GetMembers()
	if err != nil {
		return err
	}

	members = funk.Filter(members, func(m *GuildMember) bool {
		return m.ID != id
	}).([]*GuildMember)

	err = g.SetMembers(members)
	if err != nil {
		return err
	}

	g.MemberCount = int16(len(members))
	return nil
}

func (g *Guild) GetInfo() []byte {

	data := GUILD_INFO
	length := int16(0x31A + len(g.Name))

	data.SetLength(length)
	data.Insert(utils.IntToBytes(uint64(g.ID), 4, true), 6) // guild id
	data[15] = byte(g.Faction)                              // guild faction
	data[17] = byte(len(g.Name))                            // guild name length
	data.Insert([]byte(g.Name), 18)                         // guild name
	index := 18 + len(g.Name)
	data.Overwrite(utils.IntToBytes(uint64(g.Recognition), 4, true), index) // Guild recognition
	data.Insert(g.Logo[:], 18+len(g.Name))                                  // guild logo

	return data
}

func (g *Guild) GetMemberInfo(member *Character) []byte {
	data := GUILD_MEMBER_INFO
	data.Insert(utils.IntToBytes(uint64(g.ID), 4, true), 6)       // guild id
	data.Insert(utils.IntToBytes(uint64(member.ID), 4, true), 10) // character id
	data[14] = byte(len(member.Name))                             // character name length
	data.Insert([]byte(member.Name), 15)                          // character name

	index2 := 15 + len(member.Name)
	data.Insert(utils.IntToBytes(uint64(member.Level), 4, true), index2) // character level
	index2 += 4

	members, err := g.GetMembers()
	if err != nil {
		return nil
	}

	role := GROLE_MEMBER
	for _, m := range members {
		if m.ID == member.ID {
			role = m.Role
			break
		}
	}

	data[index2] = byte(role)
	index2++
	index2 += 14

	if member.IsOnline {
		data[index2] = 1
	} else {
		data[index2] = 0
	}

	index2++
	index2 += 4

	data[index2] = byte(member.Map) // character map
	//length2 := int16(60 + len(member.Name))
	//data.SetLength(length2)

	index2++
	index2 += 4

	data.Insert(utils.IntToBytes(uint64(member.ClanGoldDonation), 8, true), index2) //gold donations
	index2 += 8
	//length2 := int16(60 + len(member.Name))
	data.SetLength(int16(binary.Size(data) - 6))

	return data
}

func (g *Guild) GetData(issuer *Character) ([]byte, error) {

	data := GUILD_DATA
	length := int16(0x31C + len(g.Name))

	data.Insert(utils.IntToBytes(uint64(g.ID), 4, true), 6) // guild id
	data[15] = byte(g.Faction)                              // guild faction
	data[17] = byte(len(g.Name))                            // guild name length
	data.Insert([]byte(g.Name), 18)                         // guild name

	index := 18 + len(g.Name)
	data.Overwrite(utils.IntToBytes(uint64(g.Recognition), 8, true), index) // Guild recognition

	data.Insert(g.Logo[:], index) // guild logo
	index += 0x300
	index += 12

	data.Insert(utils.IntToBytes(uint64(g.MemberCount), 2, true), index) // guild members count
	index += 2

	members, err := g.GetMembers()
	if err != nil {
		return nil, err
	}

	for _, member := range members {
		c, err := FindCharacterByID(member.ID)
		if err != nil || c == nil {
			continue
		}

		data.Insert(utils.IntToBytes(uint64(c.ID), 4, true), index) // member id
		index += 4
		data.Insert(utils.IntToBytes(uint64(len(c.Name)), 1, true), index) // member name length
		index++
		data.Insert([]byte(c.Name), index) // member name
		index += len(c.Name)
		data.Insert(utils.IntToBytes(uint64(c.Level), 4, true), index) // member level
		index += 4
		data.Insert(utils.IntToBytes(uint64(member.Role), 1, true), index) // member role
		index++
		data.Insert([]byte{0x10, 0x27}, index)
		index += 2

		if issuer.ID == c.ID {
			data.Insert([]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x49, 0x2A}, index)
		} else {
			data.Insert([]byte{0x00, 0x00, 0x1B, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x35, 0x02}, index)
		}
		index += 12

		if c.IsOnline {
			data.Insert(utils.IntToBytes(1, 1, true), index) // member online
		} else {
			data.Insert(utils.IntToBytes(0, 1, true), index) // member offline
		}
		index++

		data.Insert([]byte{0x04, 0x00, 0x00, 0x00}, index)
		index += 4
		data.Insert(utils.IntToBytes(uint64(c.Map), 1, true), index) // member map id
		index++
		data.Insert([]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x0A,
			0x32, 0x30, 0x31, 0x38, 0x2D, 0x30, 0x38, 0x2D, 0x32, 0x35, 0x00}, index) // last seen
		index += 24
		length += int16(0x36 + len(c.Name))

		data.Concat(g.GetMemberInfo(c))
	}

	data.SetLength(length)
	data.Concat(g.GetMemberInfo(issuer))

	return data, nil
}

func FindGuildByName(name string) (*Guild, error) {

	var g Guild
	query := `select * from hops.guilds where name = $1`

	if err := db.SelectOne(&g, query, name); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("FindGuildByName: %s", err.Error())
	}

	return &g, nil
}

func FindGuildByID(id int) (*Guild, error) {

	gMutex.Lock()
	defer gMutex.Unlock()
	if g, ok := Guilds[id]; ok {
		return g, nil
	}

	var g Guild
	query := `select * from hops.guilds where id = $1`

	if err := db.SelectOne(&g, query, id); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("FindGuildByID: %s", err.Error())
	}

	Guilds[id] = &g
	return &g, nil
}

func (g *Guild) InformMembers(m *Character) {
	g.mutex.Lock()
	members, err := g.GetMembers()
	g.mutex.Unlock()
	if err != nil {
		return
	}

	members = funk.Filter(members, func(m *GuildMember) bool {
		c, err := FindCharacterByID(m.ID)
		if err != nil || c == nil {
			return false
		}
		return c.IsOnline
	}).([]*GuildMember)

	data := g.GetMemberInfo(m)
	for _, member := range members {
		c, err := FindCharacterByID(member.ID)
		if err != nil || c == nil {
			continue
		}
		if c.Socket != nil {
			c.Socket.Write(data)
		} else {
			c.Socket.OnClose()
		}
	}
}

/*func (g *Guild) GuildDonateCalculate() {
	lvlUp := false

	if g.GoldDonation >= 10000000000 { // 100b
		if g.Recognition < 5 {
			g.Recognition = 5
			lvlUp = true
		}
	} else if g.GoldDonation >= 5000000000 { // 50b
		if g.Recognition < 4 {
			g.Recognition = 4
			lvlUp = true
		}
	} else if g.GoldDonation >= 2500000000 { // 2.5b
		if g.Recognition < 3 {
			g.Recognition = 3
			lvlUp = true
		}
	} else if g.GoldDonation >= 1000000000 { // 1b
		if g.Recognition < 2 {
			g.Recognition = 2
			lvlUp = true
		}
	} else if g.GoldDonation >= 250000000 { // 210m
		if g.Recognition < 1 {
			g.Recognition = 1
			lvlUp = true
		}
	}

	g.Update()

	if lvlUp {
		g.RemoveTierBuff()
		makeAnnouncement(g.Name + " clan has been promoted to Tier " + strconv.Itoa(int(g.Recognition)))
	}
}*/

func (g *Guild) AddGuildTierBuff(c *Character) {

	buffs, err := FindBuffsByCharacterID(c.ID)
	if err == nil {
		for _, buff := range buffs {
			if buff.ID == 70019 || buff.ID == 70018 || buff.ID == 70017 || buff.ID == 70016 || buff.ID == 70015 {
				buff.Delete()
			}
		}
	}

	/*if g.Recognition == 1 {
		tier1Buff, _ := FindBuffByID(70015, c.ID)
		if tier1Buff == nil {
			infection := BuffInfections[70015]
			buff := &Buff{ID: int(70015), CharacterID: c.ID, Name: infection.Name, EXPMultiplier: 3, DropMultiplier: 1, StartedAt: c.Epoch, Duration: int64(999999999), CanExpire: false}
			buff.Create()
		}
	} else if g.Recognition == 2 {
		tier2Buff, _ := FindBuffByID(70016, c.ID)
		if tier2Buff == nil {
			infection := BuffInfections[70016]
			buff := &Buff{ID: int(70016), CharacterID: c.ID, Name: infection.Name, EXPMultiplier: 6, DropMultiplier: 2, StartedAt: c.Epoch, Duration: int64(999999999), CanExpire: false}
			buff.Create()
		}
	} else if g.Recognition == 3 {
		tier3Buff, _ := FindBuffByID(70017, c.ID)
		if tier3Buff == nil {
			infection := BuffInfections[70017]
			buff := &Buff{ID: int(70017), CharacterID: c.ID, Name: infection.Name, EXPMultiplier: 9, DropMultiplier: 3, StartedAt: c.Epoch, Duration: int64(999999999), CanExpire: false}
			buff.Create()
		}
	} else if g.Recognition == 4 {
		tier4Buff, _ := FindBuffByID(70018, c.ID)
		if tier4Buff == nil {
			infection := BuffInfections[70018]
			buff := &Buff{ID: int(70018), CharacterID: c.ID, Name: infection.Name, EXPMultiplier: 12, DropMultiplier: 4, StartedAt: c.Epoch, Duration: int64(999999999), CanExpire: false}
			buff.Create()
		}
	} else if g.Recognition == 5 {
		tier5Buff, _ := FindBuffByID(70019, c.ID)
		if tier5Buff == nil {
			infection := BuffInfections[70019]
			buff := &Buff{ID: int(70019), CharacterID: c.ID, Name: infection.Name, EXPMultiplier: 15, DropMultiplier: 5, StartedAt: c.Epoch, Duration: int64(999999999), CanExpire: false}
			buff.Create()
		}
	}*/
}

/*func (g *Guild) RemoveTierBuff() {
	members, err := g.GetMembers()
	if err != nil {
		fmt.Println("RemoveTierBuff error1: ", err)
		return
	}

	for _, c := range members {
		char, err := FindCharacterByID(c.ID)
		if err == nil {
			buffs, err := FindBuffsByCharacterID(char.ID)
			if err == nil {
				for _, buff := range buffs {
					if buff.ID == 70019 || buff.ID == 70018 || buff.ID == 70017 || buff.ID == 70016 || buff.ID == 70015 {
						buff.Delete()
					}
				}
			}
			g.AddGuildTierBuff(char)
		}
	}
}*/
