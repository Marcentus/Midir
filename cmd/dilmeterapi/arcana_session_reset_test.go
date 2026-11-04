package main

import (
	"testing"

	"github.com/Marcentus/Midir/packet"
)

// Arcana belongs to the current encounter, not the persistent player identity.
// A player may change Arcana between presses of Clear/New Session.
func TestClearDropsOldArcanaButKeepsPlayerIdentity(t *testing.T) {
	a := NewAggregator()
	const id uint64 = 1234
	a.entityCache[id] = &packet.EntityInfo{Id: id, Name: "Test Player"}
	a.playerTalents[id] = "old-icon"
	a.playerTalentNames[id] = "Harmonic Saint"
	a.playerTalentColors[id] = "old-color"
	a.Clear()
	if len(a.playerTalents) != 0 || len(a.playerTalentNames) != 0 || len(a.playerTalentColors) != 0 {
		t.Fatalf("Arcana fields must reset together at the start of a new session")
	}
	if a.entityCache[id] == nil {
		t.Fatalf("new session must keep active entity identities")
	}
	if got := len(a.harmonicSaintIDs()); got != 0 {
		t.Fatalf("old Harmonic Saint must not influence new-session overshields; got %d", got)
	}
}
