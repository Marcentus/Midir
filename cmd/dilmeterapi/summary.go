// Summary models bridge parsed events to live and saved analysis.
// Preserve backward-compatible fields used by Ghost Runner, Death Flags and healing.
package main

// ConditionInterval represents a time range where a condition was active.
type ConditionInterval struct {
	Start      int64  `json:"start"`
	End        int64  `json:"end"`
	MetaData   string `json:"metaData"`
	AttackerID uint64 `json:"attackerId"`
}

// PresenceInterval represents a time range where an entity was alive and in the area.
type PresenceInterval struct {
	Start int64 `json:"start"`
	End   int64 `json:"end"`
}

// ConditionMetaStats holds stats for a specific metadata variant of a condition.
type ConditionMetaStats struct {
	MetaData  string   `json:"metaData"`
	Uptime    float32  `json:"uptime"`
	Duration  float64  `json:"duration"`
	Attackers []uint64 `json:"attackers"`
}

type PartyBuffMetric struct {
	Label         string  `json:"label"`
	Highest       float32 `json:"highest"`
	HighestUptime float32 `json:"highestUptime"`
	WeightedAvg   float32 `json:"weightedAvg"`
}

type PartyBuff struct {
	ID      uint32            `json:"id"`
	Metrics []PartyBuffMetric `json:"metrics"`
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
	OvershieldProvided float32 `json:"overshieldProvided"`
	Uses               int     `json:"uses"` // New skill usage count
}

// DamageBreakdown now includes its own start and end times for precise DPS calculation.
type DamageBreakdown struct {
	TotalDamage        float32                    `json:"totalDamage"`
	HitCount           int                        `json:"hitCount"`
	CritCount          int                        `json:"critCount"`
	DPS                float32                    `json:"dps"`
	CritRate           float32                    `json:"critRate"`
	OvershieldProvided float32                    `json:"overshieldProvided"`
	StartTime          int64                      `json:"startTime,omitempty"` // NEW: Unix timestamp of the first hit
	EndTime            int64                      `json:"endTime,omitempty"`   // NEW: Unix timestamp of the last hit
	Skills             map[uint16]SkillStats      `json:"skills"`
	Conditions         map[uint32]*ConditionStats `json:"conditions,omitempty"` // NEW: Condition uptime stats
}

type SkillUseEvent struct {
	SkillID   uint16
	TargetID  uint64 // 0 if used without target
	Timestamp int64
}

// SkillActivation is one observed 0x6988 Skill Use packet. It is NOT a damage
// group or proof of a unique completed cast. Absolute timestamps allow future
// reporting across encounters without losing pre/post-damage use packets.
type SkillActivation struct {
	AtMs     int64  `json:"atMs"`
	SkillID  uint16 `json:"skillId"`
	TargetID string `json:"targetId,omitempty"`
}

// PreciseDamageEvent is one observed positive damage hit (not a per-second bucket).
// Its timestamp uses the same Unix-millisecond clock as SkillActivation.
// Historic logs without AtMs fall back to whole-second precision.
type SkillObservation struct {
	AtMs          int64    `json:"atMs"`
	Opcode        uint32   `json:"opcode"`
	SkillID       uint16   `json:"skillId,omitempty"`
	Subtype       uint32   `json:"subtype,omitempty"`
	ActionType    uint8    `json:"actionType,omitempty"`
	ActorVerified bool     `json:"actorVerified,omitempty"`
	Fields        []string `json:"fields,omitempty"`
	FieldTypes    []uint8  `json:"fieldTypes,omitempty"`
}

type PreciseDamageEvent struct {
	AtMs     int64   `json:"atMs"`
	SkillID  uint16  `json:"skillId"`
	TargetID string  `json:"targetId"`
	Damage   float32 `json:"damage"`
	Critical bool    `json:"critical,omitempty"`
	Delayed  bool    `json:"delayed,omitempty"`
}

// Replace your singular event struct with a secondary grouped layout
type SkillGroupEvent struct {
	Timestamp int64          `json:"timestamp"` // Relative fight second (e.g., 14)
	TotalDmg  uint64         `json:"totalDmg"`  // Summed damage during this second
	Skills    []SkillHitMeta `json:"skills"`    // List of all unique skills inside this second
	// Targets retains verified recipient identity; nil on legacy summaries.
	Targets []TargetHitMeta `json:"targets,omitempty"`
}

// TargetHitMeta aggregates a single player/second/skill/recipient combination.
// It does not claim that multiple hits mean a single skill activation.
type TargetHitMeta struct {
	TargetID  string `json:"targetId"`
	SkillID   uint16 `json:"skillId"`
	HitCount  uint32 `json:"hitCount"`
	Damage    uint64 `json:"damage"`
	MaxDamage uint64 `json:"maxDamage"`
	CritCount uint32 `json:"critCount"`
}

type SkillHitMeta struct {
	SkillID   uint16 `json:"skillId"`
	HitCount  uint32 `json:"hitCount"`
	Damage    uint64 `json:"damage"`
	MaxDamage uint64 `json:"maxDamage"`
	CritCount uint32 `json:"critCount"`
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
	OverallHealing      DamageBreakdown            `json:"overallHealing"`
	OvershieldProvided  float32                    `json:"overshieldProvided"`
	VeilCurrent         float32                    `json:"veilCurrent,omitempty"`
	VeilPeak            float32                    `json:"veilPeak,omitempty"`
	VeilDMR             float32                    `json:"veilDmr,omitempty"`
	VeilPeakDMR         float32                    `json:"veilPeakDmr,omitempty"`
	HealingByTarget     map[string]DamageBreakdown `json:"healingByTarget"`
	HealingSkills       map[uint16]SkillStats      `json:"healingSkills"`
	DamageByTarget      map[string]DamageBreakdown `json:"damageByTarget"` // Key is Target ID
	SkillUses           []SkillGroupEvent          `json:"skillUses"`
	SkillActivations    []SkillActivation          `json:"skillActivations,omitempty"`
	SkillObservations   []SkillObservation         `json:"skillObservations,omitempty"`
	DamageEvents        []PreciseDamageEvent       `json:"damageEvents,omitempty"`
	Deaths              []int64                    `json:"deaths,omitempty"`
	Revives             []int64                    `json:"revives,omitempty"`
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

// TargetHPPoint represents an enemy's HP at a specific timestamp.
type TargetHPPoint struct {
	Time      int64   `json:"time"`
	CurrentHP float32 `json:"currentHp"`
	MaxHP     float32 `json:"maxHp"`
}

// TargetStats holds information about a specific target, including active conditions.
type TargetStats struct {
	Name        string                     `json:"name"`
	RaceID      uint32                     `json:"raceId"`
	Conditions  map[uint32]*ConditionStats `json:"conditions,omitempty"`
	SeenDead    bool                       `json:"seenDead"`
	SeenAppear  bool                       `json:"seenAppear"`
	Disappeared bool                       `json:"disappeared"`
	StartTime   int64                      `json:"startTime"`
	EndTime     int64                      `json:"endTime"`
	HPHistory   []TargetHPPoint            `json:"hpHistory,omitempty"`
}

// NEW: EntityState represents an entity currently in the area.
type EntityState struct {
	ID                 string                     `json:"id"`
	Name               string                     `json:"name"`
	RaceID             uint32                     `json:"raceId"`
	RaceName           string                     `json:"raceName"`
	Conditions         map[uint32]ActiveCondition `json:"conditions,omitempty"`
	CurrentHP          float32                    `json:"currentHp"`
	MaxHP              float32                    `json:"maxHp"`
	Category           string                     `json:"category"`
	OwnerID            string                     `json:"ownerId,omitempty"`
	OwnerName          string                     `json:"ownerName,omitempty"`
	SecondaryOwnerID   string                     `json:"secondaryOwnerId,omitempty"`
	SecondaryOwnerName string                     `json:"secondaryOwnerName,omitempty"`
	EntityType         uint8                      `json:"entityType"`
	EntityTypeStr      string                     `json:"entityTypeStr,omitempty"`
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
	CurrentEntities   []EntityState                          `json:"currentEntities"`      // NEW: List of entities currently in the area
	PartyBuffs        []PartyBuff                            `json:"partyBuffs,omitempty"` // NEW: Computed party buff metrics
}

// Helper function to initialize a new breakdown
func newDamageBreakdown() DamageBreakdown {
	return DamageBreakdown{
		Skills:     make(map[uint16]SkillStats),
		Conditions: make(map[uint32]*ConditionStats),
	}
}
