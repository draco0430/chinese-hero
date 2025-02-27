package database

import (
	"database/sql"
	"fmt"
	"sort"
	"sync"
	"time"

	"hero-server/utils"

	"github.com/thoas/go-funk"
	gorp "gopkg.in/gorp.v1"
	null "gopkg.in/guregu/null.v3"
)

var (
	users     = make(map[string]*User)
	userMutex sync.RWMutex

	CLOCK = utils.Packet{0xAA, 0x55, 0x1E, 0x00, 0x72, 0x01, 0x00, 0x00, 0x03, 0x08, 0x00, 0x16, 0x00, 0x24, 0x00, 0x00, 0x00, 0x55, 0xAA}
)

type User struct {
	ID              string    `db:"id" json:"ID"`
	Username        string    `db:"user_name" json:"Username"`
	Password        string    `db:"password" json:"Password"`
	UserType        int8      `db:"user_type" json:"UserType"`
	ConnectedIP     string    `db:"ip" json:"ConnectedIP"`
	ConnectedServer int       `db:"server" json:"ConnectedServer"`
	NCash           uint64    `db:"ncash" json:"NCash"`
	BankGold        uint64    `db:"bank_gold" json:"BankGold"`
	Mail            string    `db:"mail" json:"Mail"`
	CreatedAt       null.Time `db:"created_at" json:"createdAt"`
	DisabledUntil   null.Time `db:"disabled_until" json:"disabledUntil"`

	ConnectingIP string `db:"-"`
	ConnectingTo int    `db:"-"`

	bank []*InventorySlot `db:"-" json:"-"`
}

type DailyAid struct {
	ID           string    `db:"id"`
	Count        int8      `db:"count"`
	LastTakeDate null.Time `db:"last_take_date"`
}

type DailyCheckIn struct {
	ID           string    `db:"id"`
	Count        int8      `db:"count"`
	LastTakeDate null.Time `db:"last_take_date"`
}

func (u *User) PreInsert(s gorp.SqlExecutor) error {
	now := time.Now().UTC()
	u.CreatedAt = null.TimeFrom(now)
	return nil
}

func (u *User) Create() error {
	return db.Insert(u)
}

func (u *User) CreateWithTransaction(tr *gorp.Transaction) error {
	return tr.Insert(u)
}

func (u *User) Update() error {
	_, err := db.Update(u)
	return err
}

func (u *User) Delete() error {
	_, err := db.Delete(u)
	return err
}

func (u *User) SaveRelicDrop(character_name, relic_name, npc_name string, drop_map, npc_id int, drop_rate float64) {
	db.Exec("insert into hops.relic_drop_list (user_id, character_name, relic_name, map, npc_id, npc_name, drop_rate, drop_date) values ($1, $2, $3, $4, $5, $6, $7, $8);", u.ID, character_name, relic_name, drop_map, npc_id, npc_name, drop_rate, time.Now().UTC())
}

func (u *User) GetDailyAid() bool {
	var tmpRes DailyAid
	err := db.SelectOne(&tmpRes, "select * from hops.daily_aid where id = $1", u.ID)
	if err != nil {
		if err == sql.ErrNoRows {
			_, err := db.Exec("insert into hops.daily_aid (id, count) values ($1, 0);", u.ID)
			if err != nil {
				fmt.Println(err)
				return false
			}
			tmpRes.Count = 0
		}
	}

	if tmpRes.Count != 0 {
		return false
	}

	_, err = db.Exec("update hops.daily_aid set count = 1, last_take_date = now() where id = $1;", u.ID)
	return err == nil
}

func (u *User) DailyCheck() (int8, bool) {
	var tmpRes DailyCheckIn
	err := db.SelectOne(&tmpRes, "select * from hops.daily_checkin where id = $1", u.ID)
	if err != nil {
		if err == sql.ErrNoRows {
			_, err := db.Exec("insert into hops.daily_checkin (id, count) values ($1, 0);", u.ID)
			if err != nil {
				fmt.Println(err)
				return -1, false
			}
			tmpRes.Count = 0
			return 0, true
		}
	}

	if !tmpRes.LastTakeDate.Valid {
		return -1, false
	}

	if time.Now().Day() == tmpRes.LastTakeDate.Time.Day() {
		return -1, false
	}

	tmpRes.Count++
	db.Exec("update hops.daily_checkin set count = count + 1, last_take_date = now() where id = $1;", u.ID)
	return tmpRes.Count, true
}

/*
func ResetDaily() {

	var dailyAids []DailyAid
	_, err := db.Select(&dailyAids, "select * from hops.daily_aid;")
	if err != nil {
		fmt.Println("Daily aid error", err)
		return
	}

	for _, aid := range dailyAids {
		if aid.LastTakeDate.Valid {
			if time.Now().Day() != aid.LastTakeDate.Time.Day() {
				db.Exec("update hops.daily_aid set count = 0 where id = $1;", aid.ID)
			}
		}
	}
}
*/

func ResetDailyCheckIn() {
	time.Sleep(time.Minute * 1)
	tmpNow := time.Now().AddDate(0, 0, -2)
	_, err := db.Exec("update hops.daily_checkin set last_take_date = $1;", tmpNow)
	if err != nil {
		fmt.Println("ERROR DAILYCHECKIN: ", err)
	}
}

func FindUserByName(name string) (*User, error) {

	usersCache := AllUsers()

	for _, u := range usersCache {
		if u.Username == name {
			return u, nil
		}
	}

	query := `select * from hops.users where user_name = $1`

	u := &User{}
	if err := db.SelectOne(&u, query, name); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("FindUserByName: %s", err.Error())
	}

	userMutex.Lock()
	defer userMutex.Unlock()
	users[u.ID] = u

	return u, nil
}

func FindUserByID(id string) (*User, error) {

	userMutex.RLock()
	u, ok := users[id]
	userMutex.RUnlock()
	if ok {
		return u, nil
	}

	query := `select * from hops.users where id = $1`

	u = &User{}
	if err := db.SelectOne(&u, query, id); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("FindUserByID: %s", err.Error())
	}

	userMutex.Lock()
	defer userMutex.Unlock()
	users[id] = u

	return u, nil
}

func FindUserByIP(ip string) (*User, error) {

	usersCache := AllUsers()

	for _, u := range usersCache {
		if u.ConnectedIP == ip {
			return u, nil
		}
	}

	query := `select id from hops.users where ip = $1`

	u := &User{}
	if err := db.SelectOne(u, query, ip); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("FindUserByIP: %s", err.Error())
	}

	userMutex.Lock()
	defer userMutex.Unlock()
	users[u.ID] = u

	return u, nil
}

func FindUserByMail(mail string) (*User, error) {

	usersCache := AllUsers()

	for _, u := range usersCache {
		if u.Mail == mail {
			return u, nil
		}
	}

	query := `select id from hops.users where mail = $1`

	u := &User{}
	if err := db.SelectOne(&u, query, mail); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("FindUserByMail: %s", err.Error())
	}

	userMutex.Lock()
	defer userMutex.Unlock()
	users[u.ID] = u

	return u, nil
}

func AllUsers() []*User {
	userMutex.RLock()
	defer userMutex.RUnlock()
	return funk.Values(users).([]*User)
}

func FindUsersInServer(server int) ([]*User, error) {

	userMutex.RLock()
	defer userMutex.RUnlock()

	arr := []*User{}
	for _, u := range users {
		if u.ConnectedServer == server && u.ConnectedIP != "" {
			arr = append(arr, u)
		}
	}

	return arr, nil
}

func (u *User) Logout() {
	u.ConnectedIP = ""
	u.ConnectedServer = 0
	go u.Update()
}

func DeleteUserFromCache(id string) {
	userMutex.Lock()
	defer userMutex.Unlock()
	delete(users, id)
}

func UnbanUsers() {

	userMutex.RLock()
	all := funk.Values(users).([]*User)
	userMutex.RUnlock()

	all = funk.Filter(all, func(u *User) bool {
		return u.UserType == 0 && time.Now().Sub(u.DisabledUntil.Time) >= 0
	}).([]*User)

	for _, u := range all {
		u.UserType = 1
		u.Update()
	}

	time.AfterFunc(time.Minute, func() {
		UnbanUsers()
	})
}

func (u *User) GetTime() []byte {

	resp := CLOCK

	serverName := fmt.Sprintf("Dragon %d", u.ConnectedServer)
	resp[7] = byte(len(serverName))
	resp.Insert([]byte(serverName), 8)

	length := int16(25 + len(serverName))
	resp.SetLength(length)

	now := time.Now().UTC()
	loc, _ := time.LoadLocation("Asia/Shanghai")
	now = now.In(loc)
	year := uint64(now.Year())
	month := uint64(now.Month())
	day := uint64(now.Day())
	h := uint64(now.Hour())
	m := uint64(now.Minute())
	s := uint64(now.Second())

	index := 9 + len(serverName)
	resp.Insert(utils.IntToBytes(year-2003, 2, true), index) // year
	index += 2

	resp.Insert(utils.IntToBytes(month-1, 2, true), index) // month
	index += 2

	resp.Insert(utils.IntToBytes(day, 2, true), index) // day
	index += 2

	index += 8

	resp.Insert(utils.IntToBytes(h, 2, true), index) // hour
	index += 2

	resp.Insert(utils.IntToBytes(m, 2, true), index) // minute
	index += 2

	resp.Insert(utils.IntToBytes(s, 2, true), index) // second
	index += 2

	return resp
}

func FixDropAndExp() {
	for {
		time.Sleep(time.Minute * 10)
		characters, err := FindOnlineCharacters()
		if err != nil {
			return
		}

		online := funk.Values(characters).([]*Character)
		sort.Slice(online, func(i, j int) bool {
			return online[i].Name < online[j].Name
		})

		for _, c := range online {
			c.FixDropAndExp()
		}
	}
}
