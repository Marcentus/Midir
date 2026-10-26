package main

import (
	"encoding/json"
	"testing"
)

// This intentionally compiles with only event.go and the Go standard library,
// so it can run even when third-party project dependencies are unavailable.
func TestSkillUseEventIDAndJSON(t *testing.T) {
	if eventIdSkillUse != 8 || eventIdEntityHPUpdate != 10 {
		t.Fatalf("event IDs must remain compatible with pre-existing logs: use=%d hp=%d", eventIdSkillUse, eventIdEntityHPUpdate)
	}
	recorded := eventSkillUse{
		eventBase: eventBase{EventId: eventIdSkillUse, At: 1792643233, Id: "4767482425800000"},
		AtMs:      1792643233123, SkillId: 58100, TargetId: "4767482425917391",
	}
	bytes, err := json.Marshal(&recorded)
	if err != nil {
		t.Fatal(err)
	}
	var restored eventSkillUse
	if err := json.Unmarshal(bytes, &restored); err != nil {
		t.Fatal(err)
	}
	if restored.EventId != eventIdSkillUse || restored.Id != recorded.Id || restored.SkillId != recorded.SkillId || restored.TargetId != recorded.TargetId || restored.AtMs != recorded.AtMs {
		t.Fatalf("skill-use event failed to round-trip: %s", bytes)
	}
}

func TestDamageTimestampLegacyCompatibility(t *testing.T) {
	var legacy eventDamage
	if err := json.Unmarshal([]byte(`{"EventId":3,"At":100,"Id":"11","TargetId":"22","SkillId":42,"Damage":125}`), &legacy); err != nil {
		t.Fatal(err)
	}
	if legacy.AtMs != 0 || legacy.At != 100 || legacy.TargetId != "22" {
		t.Fatalf("legacy data changed: %+v", legacy)
	}
	updated := eventDamage{eventBase: eventBase{EventId: eventIdDamage, At: 100, Id: "11"}, AtMs: 100123, TargetId: "22", SkillId: 42, Damage: 125}
	bytes, err := json.Marshal(updated)
	if err != nil {
		t.Fatal(err)
	}
	var restored eventDamage
	if err := json.Unmarshal(bytes, &restored); err != nil {
		t.Fatal(err)
	}
	if restored.AtMs != 100123 || restored.Damage != 125 {
		t.Fatalf("millisecond data lost: %s", bytes)
	}
}
