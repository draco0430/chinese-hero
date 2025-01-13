package database

import "hero-server/utils"

type Duel struct {
	EnemyID    int
	Coordinate utils.Location
	Started    bool
}
