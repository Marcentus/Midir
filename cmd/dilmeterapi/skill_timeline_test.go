package main

import (
	"encoding/json"
	"testing"
)

func TestPreciseDamageEvidenceFromSavedLog(t *testing.T) {
	start := int64(1000)
	playerID := "101"
	summary := &FightSummary{Players: map[string]PlayerStats{
		playerID: {ID: playerID, Name: "TestPlayer"},
	}}
	hits := []eventDamage{
		{eventBase: eventBase{EventId: eventIdDamage, At: 1002, Id: playerID}, AtMs: 1002234,
			TargetId: "202", SkillId: 1234, Damage: 200, IsCritical: true},
		// Legacy logs have no AtMs; keep them usable without faking precision.
		{eventBase: eventBase{EventId: eventIdDamage, At: 1003, Id: playerID},
			TargetId: "202", SkillId: 1234, Damage: 50, IsDelayed: true},
		{eventBase: eventBase{EventId: eventIdDamage, At: 1004, Id: playerID},
			TargetId: "202", SkillId: 1234, Damage: -30},
	}
	attachSkillGroupsFromDamageEvents(summary, hits, start)
	evidence := summary.Players[playerID].DamageEvents
	if len(evidence) != 2 {
		t.Fatalf("expected 2 positive per-hit records, got %d", len(evidence))
	}
	if evidence[0].AtMs != 1002234 || evidence[0].Damage != 200 || evidence[0].TargetID != "202" || !evidence[0].Critical {
		t.Fatalf("first precise record incorrect: %+v", evidence[0])
	}
	if evidence[1].AtMs != 1003000 || !evidence[1].Delayed {
		t.Fatalf("legacy fallback and delayed flag incorrect: %+v", evidence[1])
	}
	if len(summary.Players[playerID].SkillUses) != 2 {
		t.Fatal("legacy per-second scatter grouping was lost")
	}
	data, err := json.Marshal(summary.Players[playerID])
	if err != nil {
		t.Fatal(err)
	}
	var restored PlayerStats
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatal(err)
	}
	if len(restored.DamageEvents) != 2 || restored.DamageEvents[0].AtMs != 1002234 {
		t.Fatalf("precise evidence did not round-trip: %+v", restored.DamageEvents)
	}
}
