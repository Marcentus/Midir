package main

type eventId uint16

const (
	eventIdEntityAppear eventId = 1 + iota
	eventIdEntityDisappear
	eventIdDamage
	eventIdCharacterConditionEnable
	eventIdCharacterConditionDisable
	eventIdEntityDeath
	eventIdEntityRevive
	eventIdSkillUse // previously reserved ID 8; independently observed 0x6988
	_               // formerly eventIdSkillStart (9)
	eventIdEntityHPUpdate
	eventIdSkillObservation         // combat-gated raw evidence; interpreted at save/export
	eventIdSessionSummary   eventId = 9999
)

type iEvent interface {
	GetEventId() eventId
}

type eventEntityDeath struct {
	eventBase
}

type eventEntityRevive struct {
	eventBase
}

type eventBase struct {
	EventId eventId
	At      int64
	Id      string
}

func (t *eventBase) GetEventId() eventId {
	return t.EventId
}

// --- START: NEW DATA-ONLY STRUCTS FOR NESTING ---

// EventConditionData holds condition info to be nested inside an appearance event.
type EventConditionData struct {
	CCId       uint32 `json:"CCId"`
	DisableAt  int64  `json:"DisableAt"`
	MetaData   string `json:"MetaData"`
	AttackerId string `json:"AttackerId"`
}

// --- END: NEW DATA-ONLY STRUCTS FOR NESTING ---

type eventEntityAppear struct {
	eventBase
	Name             string
	RaceId           uint32
	OwnerId          string
	EntityType       uint8  `json:"entityType,omitempty"`
	SecondaryOwnerId string `json:"secondaryOwnerId,omitempty"`

	CurrentHP float32 `json:"currentHp"`
	MaxHP     float32 `json:"maxHp"`

	Conditions []EventConditionData `json:"conditions"`
}

type eventEntityDisappear struct {
	eventBase
}

// eventSkillUse is an observed use opcode, not necessarily a distinct completed
// cast. It is intentionally independent of damage and of Skill Start (0x698c).
type eventSkillUse struct {
	eventBase
	AtMs     int64 `json:"AtMs"` // Unix milliseconds from the captured packet
	SkillId  uint16
	TargetId string `json:"TargetId,omitempty"` // packet target hint, when present and not self
}

type eventDamage struct {
	eventBase
	AtMs       int64 `json:"AtMs,omitempty"` // preserve precise hit timing for later use/damage correlation
	TargetId   string
	SkillId    uint16
	Damage     float32
	ManaDamage float32
	IsCritical bool
	IsDelayed  bool
}

type eventCharacterConditionEnable struct {
	eventBase
	CCId       uint32
	DisableAt  int64
	MetaData   string
	AttackerId string
}

type eventCharacterConditionDisable struct {
	eventBase
	CCId uint32
}

type eventSessionSummary struct {
	eventBase
	Type    string      `json:"type"`
	Summary interface{} `json:"summary"` // Will hold SessionSummaryData
}

type eventEntityHPUpdate struct {
	eventBase
	CurrentHP    float32 `json:"currentHp"`
	BaseHP       float32 `json:"baseHp"`
	AdditionalHP float32 `json:"additionalHp"`
	MaxHP        float32 `json:"maxHp"`
}

// eventSkillObservation preserves typed protocol evidence, never a cast count.
// Field values are encoded as strings to avoid losing uint64 precision in JS.
type eventSkillObservation struct {
	eventBase
	AtMs          int64    `json:"AtMs"`
	Opcode        uint32   `json:"Opcode"`
	SkillId       uint16   `json:"SkillId,omitempty"`
	Subtype       uint32   `json:"Subtype,omitempty"`
	ActionType    uint8    `json:"ActionType,omitempty"`
	ActorVerified bool     `json:"ActorVerified,omitempty"`
	Fields        []string `json:"Fields,omitempty"`
	FieldTypes    []uint8  `json:"FieldTypes,omitempty"`
}
