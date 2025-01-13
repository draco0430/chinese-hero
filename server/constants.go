package server

import (
	"hero-server/database"

	cmap "github.com/orcaman/concurrent-map"
	"github.com/thoas/go-funk"
)

const (
	BANNED_USER = iota
	COMMON_USER
	GA_USER
	GAL_USER
	GM_USER
	HGM_USER
)

var (
	MutedPlayers = cmap.New()
)

func init() {
	accUpgrades := []byte{}
	armorUpgrades := []byte{}
	weaponUpgrades := []byte{}

	// for mob drop
	dbAccUpgrades := []byte{}
	dbArmorUpgrades := []byte{}
	dbWeaponUpgrades := []byte{}

	for i := 1; i <= 40; i++ {
		for j := 0; j <= 5-(i%5); j++ {
			accUpgrades = append(accUpgrades, byte(i))
			/*
				if i != 5 && i != 4 && i != 25 && i != 24 {
					accUpgrades = append(accUpgrades, byte(i))
				}
			*/
		}
	}

	for i := 26; i <= 65; i++ {
		for j := 0; j <= 5-(i%5); j++ {
			armorUpgrades = append(armorUpgrades, byte(i))
			/*
				if i != 44 && i != 45 && i != 50 && i != 49 && i != 60 && i != 59 && i != 65 && i != 64 {
					armorUpgrades = append(armorUpgrades, byte(i))
				}
			*/
		}
	}

	for i := 66; i <= 105; i++ {
		for j := 0; j <= 5-(i%5); j++ {
			weaponUpgrades = append(weaponUpgrades, byte(i))
			/*
				if i != 75 && i != 74 && i != 70 && i != 69 {
					weaponUpgrades = append(weaponUpgrades, byte(i))
				}
			*/
		}
	}

	// For mob drop

	for i := 1; i <= 40; i++ {
		for j := 0; j <= 5-(i%5); j++ {
			if i != 49 && i != 50 && i != 59 && i != 60 && i != 69 && i != 70 && i != 74 && i != 75 {
				dbAccUpgrades = append(dbAccUpgrades, byte(i))
			}
		}
	}

	for i := 26; i <= 65; i++ {
		for j := 0; j <= 5-(i%5); j++ {
			if i != 44 && i != 45 && i != 50 && i != 49 && i != 60 && i != 59 && i != 65 && i != 64 {
				dbArmorUpgrades = append(dbArmorUpgrades, byte(i))
			}
		}
	}

	for i := 66; i <= 105; i++ {
		for j := 0; j <= 5-(i%5); j++ {
			if i != 75 && i != 74 && i != 70 && i != 69 {
				dbWeaponUpgrades = append(dbWeaponUpgrades, byte(i))
			}
		}
	}

	database.AccUpgrades = funk.Shuffle(accUpgrades).([]byte)
	database.ArmorUpgrades = funk.Shuffle(armorUpgrades).([]byte)
	database.WeaponUpgrades = funk.Shuffle(weaponUpgrades).([]byte)
	database.SocketAccUpgrades = funk.Shuffle(dbAccUpgrades).([]byte)
	database.SocketArmorUpgrades = funk.Shuffle(dbArmorUpgrades).([]byte)
	database.SocketWeaponUpgrades = funk.Shuffle(dbWeaponUpgrades).([]byte)

}
