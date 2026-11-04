package main

import (
	"testing"

	"github.com/Marcentus/Midir/packet"
)

// Attribution tests use active entityCache entries, not persisted player names.
func addTestSaint(a *Aggregator, id uint64) {
	a.entityCache[id] = &packet.EntityInfo{Id: id, Name: "Test Saint"}
	a.playerTalentNames[id] = "Harmonic Saint"
}

func TestOvershieldSingleKnownSaint(t *testing.T) {
	a := NewAggregator()
	addTestSaint(a, 101)
	if got := a.overshieldCasterForTarget(999, 100); got != 101 {
		t.Fatalf("single Saint: got %d, want 101", got)
	}
}

func TestOvershieldMultiSaintRecentAndAmbiguous(t *testing.T) {
	a := NewAggregator()
	addTestSaint(a, 101)
	addTestSaint(a, 202)
	a.recentShieldCasts[101] = healingCasterHint{CasterID: 101, SkillID: 59001, At: 99}
	a.recentShieldCasts[202] = healingCasterHint{CasterID: 202, SkillID: 59002, At: 100}
	if got := a.overshieldCasterForTarget(999, 100); got != 202 {
		t.Fatalf("most recent cast: got %d, want 202", got)
	}
	a.recentShieldCasts[101] = healingCasterHint{CasterID: 101, SkillID: 59001, At: 100}
	if got := a.overshieldCasterForTarget(999, 100); got != 0 {
		t.Fatalf("simultaneous casts must remain unassigned, got %d", got)
	}
	a.lastHealingCaster[999] = healingCasterHint{CasterID: 101, SkillID: 59001, At: 100}
	if got := a.overshieldCasterForTarget(999, 100); got != 101 {
		t.Fatalf("target-specific hint: got %d, want 101", got)
	}
}

func TestOvershieldRejectsNonSaintAndStaleCast(t *testing.T) {
	a := NewAggregator()
	addTestSaint(a, 101)
	addTestSaint(a, 202)
	a.entityCache[303] = &packet.EntityInfo{Id: 303, Name: "Other"}
	a.playerTalentNames[303] = "Dark Diviner"
	a.lastHealingCaster[999] = healingCasterHint{CasterID: 303, SkillID: 59001, At: 100}
	a.recentShieldCasts[101] = healingCasterHint{CasterID: 101, SkillID: 59001, At: 90}
	if got := a.overshieldCasterForTarget(999, 100); got != 0 {
		t.Fatalf("non-Saint and stale hints must not match, got %d", got)
	}
}
