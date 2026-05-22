package main

import (
	"testing"
	"time"

	"github.com/Marcentus/Midir/packet"
)

func TestAggregator_SoftClear(t *testing.T) {
	agg := NewAggregator()
	agg.SetLive(true)

	playerID := uint64(12345)

	// manually inject player into cache because ProcessPacket requires valid packet data
	// or we can simulate packets

	// 1. Simulate Player Appear
	// We need to construct a fake packet for EntityAppear.
	// Since constructing the binary packet is hard, we can test the internal state directly or use the available parsing helpers if we can mock the byte stream.
	// However, looking at aggregator.go, it uses public maps. We can inspect them.

	// Let's directly manipulate the internal state which is "Unit" testing the Clear method.
	agg.entityCache[playerID] = &packet.EntityInfo{Id: playerID, Name: "TestPlayer"}
	agg.playerSeenAppear[playerID] = true
	agg.playerTalentNames[playerID] = "Dark Knight"
	agg.playerConditionActive[playerID] = make(map[uint32]ActiveCondition)
	agg.playerConditionActive[playerID][1] = ActiveCondition{
		Start:    time.Now().Unix(),
		MetaData: "Buff",
	}

	stats := agg.getOrCreatePlayerStats(&PlayerInfo{ID: playerID, Name: "TestPlayer"})
	stats.OverallStats.TotalDamage = 1000

	// 2. Perform Clear
	agg.Clear()

	// 3. Verify Soft Clear
	// Stats should be wiped
	if len(agg.playerStats) != 0 {
		t.Errorf("Expected playerStats to be empty, got %d", len(agg.playerStats))
	}

	// Identity should be preserved
	if _, ok := agg.entityCache[playerID]; !ok {
		t.Errorf("Expected entityCache to preserve playerID")
	}
	if _, ok := agg.playerTalentNames[playerID]; !ok {
		t.Errorf("Expected playerTalentNames to preserve playerID")
	}
	if _, ok := agg.playerSeenAppear[playerID]; !ok {
		t.Errorf("Expected playerSeenAppear to preserve playerID")
	}

	// Conditions should be preserved (Active)
	if _, ok := agg.playerConditionActive[playerID]; !ok {
		t.Errorf("Expected playerConditionActive to preserve playerID")
	} else {
		if _, ok := agg.playerConditionActive[playerID][1]; !ok {
			t.Errorf("Expected active condition 1 to be preserved")
		}
	}
}

func TestAggregator_EntityDisappear(t *testing.T) {
	agg := NewAggregator()
	agg.SetLive(true)

	playerID := uint64(9999)
	agg.entityCache[playerID] = &packet.EntityInfo{Id: playerID, Name: "Leaver"}
	agg.playerConditionActive[playerID] = make(map[uint32]ActiveCondition)
	agg.playerConditionActive[playerID][1] = ActiveCondition{Start: 100}

	// Create a Disappear Packet
	p := &packet.GamePacket{
		Op: opcodeEntityDisappear,
		Id: playerID,
		At: time.Now(),
		Msg: []packet.IMessageElem{
			packet.NewMessageElemLong(playerID),
		},
	}

	// Process Disappear
	agg.ProcessPacket(p)

	// Verify Cleanup
	if _, ok := agg.entityCache[playerID]; ok {
		t.Errorf("Expected entityCache to DELETE playerID after disappear")
	}
	if _, ok := agg.playerConditionActive[playerID]; ok {
		t.Errorf("Expected playerConditionActive to DELETE playerID after disappear")
	}
}
