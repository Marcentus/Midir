package main

import (
	"database/sql"
	"log"
	"sync"

	"github.com/Marcentus/Midir/packet"
)

type PlayerInfo struct {
	ID     uint64 `json:"id"`
	Name   string `json:"name"`
	RaceId uint32 `json:"raceId"`
}

// PlayerCache handles player persistence via SQLite.
type PlayerCache struct {
	mu             sync.RWMutex
	OnPlayerUpdate func(*PlayerInfo)
}

// NewPlayerCache creates a new cache instance.
func NewPlayerCache(path string) *PlayerCache {
	return &PlayerCache{}
}

func (p *PlayerInfo) Clone() *PlayerInfo {
	return &PlayerInfo{
		ID:     p.ID,
		Name:   p.Name,
		RaceId: p.RaceId,
	}
}

// Load is a no-op as we rely on SQLite now.
func (c *PlayerCache) Load() {
}

// Save is a no-op as we save on Update.
func (c *PlayerCache) Save() {
}

// Get retrieves a player's info from the DB.
func (c *PlayerCache) Get(id uint64) (*PlayerInfo, bool) {
	if db == nil {
		return nil, false
	}

	var p PlayerInfo
	row := db.QueryRow("SELECT id, name, race_id FROM players WHERE id = ?", id)
	err := row.Scan(&p.ID, &p.Name, &p.RaceId)
	if err == sql.ErrNoRows {
		return nil, false
	}
	if err != nil {
		log.Println("Error querying player:", err)
		return nil, false
	}

	return &p, true
}

// Update adds or updates a player in the DB.
func (c *PlayerCache) Update(entity *packet.EntityInfo) {
	if !isPlayer(entity) {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	player, exists := c.Get(entity.Id)
	if !exists {
		player = &PlayerInfo{
			ID: entity.Id,
		}
	}

	player.Name = entity.Name
	player.RaceId = entity.RaceId

	if err := c.savePlayer(player); err != nil {
		log.Println("Error updating player in DB:", err)
	}

	if c.OnPlayerUpdate != nil {
		c.OnPlayerUpdate(player.Clone())
	}
}

// GetAll retrieves all players from the DB.
func (c *PlayerCache) GetAll() map[uint64]*PlayerInfo {
	if db == nil {
		return make(map[uint64]*PlayerInfo)
	}

	rows, err := db.Query("SELECT id, name, race_id FROM players")
	if err != nil {
		log.Println("Error querying all players:", err)
		return make(map[uint64]*PlayerInfo)
	}
	defer rows.Close()

	result := make(map[uint64]*PlayerInfo)
	for rows.Next() {
		var p PlayerInfo
		if err := rows.Scan(&p.ID, &p.Name, &p.RaceId); err != nil {
			continue
		}
		result[p.ID] = &p
	}
	return result
}

// savePlayer persists a player's data to the database.
func (c *PlayerCache) savePlayer(player *PlayerInfo) error {
	_, err := db.Exec(`INSERT OR REPLACE INTO players 
		(id, name, race_id) 
		VALUES (?, ?, ?)`,
		player.ID, player.Name, player.RaceId)
	return err
}
