package main

import (
	"database/sql"
	"log"
	"strings"
	"sync"
	"unicode"

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
	if !IsPlayer(entity) {
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

// UpdateFromAppear adds or updates a player in the DB from a log event.
func (c *PlayerCache) UpdateFromAppear(event eventEntityAppear) {
	id := parseUint64(event.Id)
	ownerId := parseUint64(event.OwnerId)

	if !isPlayerInfo(event.Name, event.RaceId, ownerId) {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	player, exists := c.Get(id)
	if !exists {
		player = &PlayerInfo{
			ID: id,
		}
	}

	player.Name = event.Name
	player.RaceId = event.RaceId

	if err := c.savePlayer(player); err != nil {
		log.Println("Error updating player in DB from log:", err)
	}

	if c.OnPlayerUpdate != nil {
		c.OnPlayerUpdate(player.Clone())
	}
}

// IsPlayer determines if an EntityInfo is a player.
func IsPlayer(e *packet.EntityInfo) bool {
	if e == nil {
		return false
	}
	return isPlayerInfo(e.Name, e.RaceId, e.OwnerId)
}

// isPlayerInfo is the centralized logic for identifying a player entity.
func isPlayerInfo(name string, raceId uint32, ownerId uint64) bool {
	if ownerId != 0 {
		return false
	}

	// Filter out NPCs (names start with _)
	if strings.HasPrefix(name, "_") {
		return false
	}

	// Filter out Monsters (names are all numbers)
	isNumeric := true
	for _, c := range name {
		if !unicode.IsDigit(c) {
			isNumeric = false
			break
		}
	}
	if isNumeric && len(name) > 0 {
		return false
	}

	switch raceId {
	case 8001, 8002, 9001, 9002, 10001, 10002:
		return true
	default:
		return false
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
