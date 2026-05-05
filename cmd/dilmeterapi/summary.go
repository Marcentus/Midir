package main

// ConditionInterval represents a time range where a condition was active.
type ConditionInterval struct {
	Start      int64  `json:"start"`
	End        int64  `json:"end"`
	MetaData   string `json:"metaData"`
	AttackerID uint64 `json:"attackerId"`
}

// ConditionMetaStats holds stats for a specific metadata variant of a condition.
type ConditionMetaStats struct {
	MetaData  string   `json:"metaData"`
	Uptime    float32  `json:"uptime"`
	Duration  float64  `json:"duration"`
	Attackers []uint64 `json:"attackers"`
}

// ActiveCondition represents a currently active condition on an entity.
// This is used for real-time snapshots (e.g., "Entities in Area").
type ActiveCondition struct {
	Start      int64  `json:"start"`
	DisableAt  int64  `json:"disableAt"` // NEW: Unix timestamp when condition expires
	MetaData   string `json:"metaData"`
	AttackerID uint64 `json:"attackerId"`
}

// ConditionStats holds the calculated uptime for a specific condition.
type ConditionStats struct {
	ID            uint32               `json:"id"`
	Uptime        float32              `json:"uptime"`              // Percentage (0-100)
	Duration      float64              `json:"duration"`            // Total active seconds within the window
	Intervals     []ConditionInterval  `json:"intervals,omitempty"` // NEW: List of active intervals
	MetaBreakdown []ConditionMetaStats `json:"metaBreakdown,omitempty"`
}

// SkillStats holds data for a single skill, tracking critical and non-critical hits.
type SkillStats struct {
	ID                 uint16  `json:"id"`
	Count              int     `json:"count"`
	CritCount          int     `json:"critCount"`
	TotalDamage        float32 `json:"totalDamage"`
	TotalDamageCrit    float32 `json:"totalDamageCrit"`
	TotalDamageNonCrit float32 `json:"totalDamageNonCrit"`
	MaxDamage          float32 `json:"maxDamage"`
	MaxDamageCrit      float32 `json:"maxDamageCrit"`
	MaxDamageNonCrit   float32 `json:"maxDamageNonCrit"`
}

// DamageBreakdown now includes its own start and end times for precise DPS calculation.
type DamageBreakdown struct {
	TotalDamage float32                    `json:"totalDamage"`
	HitCount    int                        `json:"hitCount"`
	CritCount   int                        `json:"critCount"`
	DPS         float32                    `json:"dps"`
	CritRate    float32                    `json:"critRate"`
	StartTime   int64                      `json:"startTime,omitempty"` // NEW: Unix timestamp of the first hit
	EndTime     int64                      `json:"endTime,omitempty"`   // NEW: Unix timestamp of the last hit
	Skills      map[uint16]SkillStats      `json:"skills"`
	Conditions  map[uint32]*ConditionStats `json:"conditions,omitempty"` // NEW: Condition uptime stats
}

// PlayerStats is now more detailed.
type PlayerStats struct {
	ID                  string                     `json:"id"`
	Name                string                     `json:"name"`
	TalentIcon          string                     `json:"talentIcon,omitempty"`
	TalentName          string                     `json:"talentName,omitempty"`
	TalentColor         string                     `json:"talentColor,omitempty"`
	MissingAppearPacket bool                       `json:"missingAppearPacket"` // NEW: True if we haven't seen an appear packet (cache warning)
	OverallStats        DamageBreakdown            `json:"overallStats"`
	DamageByTarget      map[string]DamageBreakdown `json:"damageByTarget"` // Key is Target ID
}

// --- (Structs for Damage Taken remain the same) ---
// DamageTakenDetails holds the aggregated damage a player has taken...
type DamageTakenDetails struct {
	AttackerID      uint64  `json:"attackerId"`
	AttackerName    string  `json:"attackerName"`
	SkillID         uint16  `json:"skillId"`
	TotalDamage     float32 `json:"totalDamage"`
	TotalManaDamage float32 `json:"totalManaDamage"` // NEW
	HitCount        int     `json:"hitCount"`
	MinDamage       float32 `json:"minDamage"`
	MaxDamage       float32 `json:"maxDamage"`
}

// PlayerDamageTakenStats holds all the damage taken information for a single player.
type PlayerDamageTakenStats struct {
	PlayerID        string                        `json:"playerId"`
	PlayerName      string                        `json:"playerName"`
	TotalDamage     float32                       `json:"totalDamage"` // HP + Mana
	TotalManaDamage float32                       `json:"totalManaDamage"`
	Breakdown       map[string]DamageTakenDetails `json:"breakdown"`
}

// ---

type GraphDataPoint struct {
	Time        int64   `json:"time"`
	TotalDamage float32 `json:"totalDamage"`
	RollingDPS  float32 `json:"rollingDPS"`
}

// TargetStats holds information about a specific target, including active conditions.
type TargetStats struct {
	Name       string                     `json:"name"`
	Conditions map[uint32]*ConditionStats `json:"conditions,omitempty"`
}

// NEW: EntityState represents an entity currently in the area.
type EntityState struct {
	ID         string                     `json:"id"`
	Name       string                     `json:"name"`
	RaceID     uint32                     `json:"raceId"`
	Conditions map[uint32]ActiveCondition `json:"conditions,omitempty"`
	CurrentHP  float32                    `json:"currentHp"`
	MaxHP      float32                    `json:"maxHp"`
}

// FightSummary's EncounterDuration now represents the time from the first to last damage event.
type FightSummary struct {
	EncounterDuration float64                                `json:"encounterDuration"`
	StartTime         int64                                  `json:"startTime"` // NEW: Encounter Start Timestamp
	EndTime           int64                                  `json:"endTime"`   // NEW: Encounter End Timestamp
	TotalDamage       float32                                `json:"totalDamage"`
	Players           map[string]PlayerStats                 `json:"players"`
	Targets           map[string]TargetStats                 `json:"targets"`
	DamageTaken       map[string]PlayerDamageTakenStats      `json:"damageTaken"`
	GraphData         map[string]map[string][]GraphDataPoint `json:"graphData,omitempty"`
	CurrentEntities   []EntityState                          `json:"currentEntities"` // NEW: List of entities currently in the area
}

// Helper function to initialize a new breakdown
func newDamageBreakdown() DamageBreakdown {
	return DamageBreakdown{
		Skills:     make(map[uint16]SkillStats),
		Conditions: make(map[uint32]*ConditionStats),
	}
}
