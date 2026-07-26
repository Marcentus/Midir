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
	_ // formerly eventIdSkillUse (8)
	_ // formerly eventIdSkillStart (9)
	eventIdEntityHPUpdate
	eventIdSessionSummary eventId = 9999
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
	Name       string
	RaceId     uint32
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

type eventDamage struct {
	eventBase
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
