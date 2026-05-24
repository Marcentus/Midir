package main

import (
	"database/sql"
	"fmt"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/Marcentus/Midir/packet"
	_ "modernc.org/sqlite"
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

func TestAggregator_DeathTracking(t *testing.T) {
	agg := NewAggregator()
	agg.SetLive(true)

	playerID := uint64(5555)
	enemyID := uint64(7777)

	// Add player to entity cache
	agg.entityCache[playerID] = &packet.EntityInfo{
		Id:      playerID,
		Name:    "Alice",
		OwnerId: 0,
		RaceId:  8001,
	}

	// Add enemy to entity cache
	agg.entityCache[enemyID] = &packet.EntityInfo{
		Id:      enemyID,
		Name:    "123456", // Numeric name makes it an enemy
		OwnerId: 0,
		RaceId:  2000,
	}

	// 1. Verify initial state (not dead)
	if agg.deadEntities[playerID] {
		t.Errorf("Expected player to start as alive")
	}
	if agg.deadEntities[enemyID] {
		t.Errorf("Expected enemy to start as alive")
	}

	// 2. Simulate Player Death (IsNowDead opcode 0x53fc)
	pDeadPlayer := &packet.GamePacket{
		Op: opcodeIsNowDead,
		Id: playerID,
		At: time.Now(),
	}
	agg.ProcessPacket(pDeadPlayer)

	if !agg.deadEntities[playerID] {
		t.Errorf("Expected player to be marked as dead after IsNowDead")
	}

	// 3. Simulate Enemy Death (SetFinisher opcode 0x7921)
	pDeadEnemy := &packet.GamePacket{
		Op: opcodeSetFinisher,
		Id: enemyID,
		At: time.Now(),
	}
	agg.ProcessPacket(pDeadEnemy)

	if !agg.deadEntities[enemyID] {
		t.Errorf("Expected enemy to be marked as dead after SetFinisher")
	}

	// Simulate SetFinisher spam on the enemy
	agg.ProcessPacket(pDeadEnemy)
	if !agg.deadEntities[enemyID] {
		t.Errorf("Expected enemy to still be marked as dead after SetFinisher spam")
	}

	// 4. Simulate Player Revival (RiseFromTheDead opcode 0x701d)
	pRevPlayer := &packet.GamePacket{
		Op: opcodeRiseFromTheDead,
		Id: playerID,
		At: time.Now(),
	}
	agg.ProcessPacket(pRevPlayer)

	if agg.deadEntities[playerID] {
		t.Errorf("Expected player to be revived (alive) after RiseFromTheDead")
	}

	// 5. Simulate Enemy Disappear (EntityDisappear opcode 0x520d)
	// First make the enemy dead again
	agg.ProcessPacket(pDeadEnemy)
	if !agg.deadEntities[enemyID] {
		t.Errorf("Expected enemy to be dead again")
	}

	pDisEnemy := &packet.GamePacket{
		Op: opcodeEntityDisappear,
		Id: enemyID,
		At: time.Now(),
		Msg: []packet.IMessageElem{
			packet.NewMessageElemLong(enemyID),
		},
	}
	agg.ProcessPacket(pDisEnemy)

	if agg.deadEntities[enemyID] {
		t.Errorf("Expected enemy's dead status to be cleared after disappear")
	}
}

func TestAggregator_TargetIconStateTracking(t *testing.T) {
	agg := NewAggregator()
	agg.SetLive(true)

	targetID := uint64(8888)

	// 1. Initially it should not be tracked
	if agg.seenAppear[targetID] || agg.seenDead[targetID] || agg.disappeared[targetID] {
		t.Errorf("Expected target to have no status flags initialized")
	}

	// 2. Mock target appearance by manually setting fields
	agg.mu.Lock()
	agg.entityCache[targetID] = &packet.EntityInfo{Id: targetID, Name: "123456"}
	agg.seenAppear[targetID] = true
	agg.disappeared[targetID] = false
	agg.seenDead[targetID] = false
	agg.mu.Unlock()

	if !agg.seenAppear[targetID] {
		t.Errorf("Expected seenAppear to be true")
	}

	// 3. Simulate Death via ProcessPacket (opcodeSetFinisher)
	pFinisher := &packet.GamePacket{
		Op: opcodeSetFinisher,
		Id: targetID,
		At: time.Now(),
	}
	agg.ProcessPacket(pFinisher)

	if !agg.seenDead[targetID] {
		t.Errorf("Expected seenDead to be true after SetFinisher")
	}

	// 4. Simulate Disappear via ProcessPacket (opcodeEntityDisappear) for dead target
	pDisappear := &packet.GamePacket{
		Op: opcodeEntityDisappear,
		Id: targetID,
		At: time.Now(),
		Msg: []packet.IMessageElem{
			packet.NewMessageElemLong(targetID),
		},
	}
	agg.ProcessPacket(pDisappear)

	// Since target had died, disappeared must be false (death takes priority!)
	if agg.disappeared[targetID] {
		t.Errorf("Expected disappeared to be false because target was already dead")
	}
	if !agg.seenDead[targetID] {
		t.Errorf("Expected seenDead to persist as true after EntityDisappear")
	}

	// 5. Simulate a living target disappearing (no prior death)
	livingTargetID := uint64(9991)
	agg.mu.Lock()
	agg.entityCache[livingTargetID] = &packet.EntityInfo{Id: livingTargetID, Name: "123456"}
	agg.seenAppear[livingTargetID] = true
	agg.disappeared[livingTargetID] = false
	agg.seenDead[livingTargetID] = false
	agg.mu.Unlock()

	pDisappearLiving := &packet.GamePacket{
		Op: opcodeEntityDisappear,
		Id: livingTargetID,
		At: time.Now(),
		Msg: []packet.IMessageElem{
			packet.NewMessageElemLong(livingTargetID),
		},
	}
	agg.ProcessPacket(pDisappearLiving)

	if !agg.disappeared[livingTargetID] {
		t.Errorf("Expected disappeared to be true for living entity after EntityDisappear")
	}

	// 6. Simulate Reappearance of the dead target
	// Mock opcodeEntityAppear action as done in real aggregator
	agg.mu.Lock()
	agg.seenAppear[targetID] = true
	agg.disappeared[targetID] = false
	agg.mu.Unlock()

	if !agg.seenAppear[targetID] {
		t.Errorf("Expected seenAppear to remain true")
	}
	if agg.disappeared[targetID] {
		t.Errorf("Expected disappeared to be reset to false after reappear")
	}
	if !agg.seenDead[targetID] {
		t.Errorf("Expected seenDead to REMAIN true after reappear (corpse lingering)")
	}
}

func TestAggregator_SkillUses(t *testing.T) {
	var err error
	db, err = sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		db.Close()
		db = nil
	}()

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS players (
		id INTEGER PRIMARY KEY,
		name TEXT,
		race_id INTEGER
	);`)
	if err != nil {
		t.Fatal(err)
	}

	// Insert mock player
	playerID := uint64(5555)
	_, err = db.Exec("INSERT INTO players (id, name, race_id) VALUES (?, ?, ?)", playerID, "Alice", 8001)
	if err != nil {
		t.Fatal(err)
	}

	agg := NewAggregator()
	agg.SetLive(true)

	// 1. Process standard skill use (opcodeSkillUse = 0x6988)
	p1 := &packet.GamePacket{
		Op: opcodeSkillUse,
		Id: playerID,
		At: time.Now(),
		Msg: []packet.IMessageElem{
			packet.NewMessageElemShort(123), // Skill ID 123
		},
	}
	agg.ProcessPacket(p1)

	// 2. Process new skill use (opcodeSkillStart = 0x698c)
	p2 := &packet.GamePacket{
		Op: opcodeSkillStart,
		Id: playerID,
		At: time.Now(),
		Msg: []packet.IMessageElem{
			packet.NewMessageElemShort(456), // Skill ID 456
		},
	}
	agg.ProcessPacket(p2)

	// 3. Get summary and verify counts
	summary := agg.GetSummary()
	playerStats, exists := summary.Players[strconv.FormatUint(playerID, 10)]
	if !exists {
		t.Fatalf("Expected stats for player %d", playerID)
	}

	skill1Stats, ok1 := playerStats.OverallStats.Skills[123]
	if !ok1 || skill1Stats.Uses != 1 {
		t.Errorf("Expected skill 123 to have 1 use, got ok=%v, uses=%d", ok1, skill1Stats.Uses)
	}

	skill2Stats, ok2 := playerStats.OverallStats.Skills[456]
	if !ok2 || skill2Stats.Uses != 1 {
		t.Errorf("Expected skill 456 to have 1 use, got ok=%v, uses=%d", ok2, skill2Stats.Uses)
	}
}

func TestGenerateSummaryFromFile_SkillUses(t *testing.T) {
	// Initialize database
	var err error
	db, err = sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		db.Close()
		db = nil
	}()

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS players (
		id INTEGER PRIMARY KEY,
		name TEXT,
		race_id INTEGER
	);`)
	if err != nil {
		t.Fatal(err)
	}

	// Insert mock player
	playerID := uint64(5555)
	_, err = db.Exec("INSERT INTO players (id, name, race_id) VALUES (?, ?, ?)", playerID, "Alice", 8001)
	if err != nil {
		t.Fatal(err)
	}

	// Create temporary log file
	tmpFile, err := os.CreateTemp("", "session-*.ndjson")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	// Write mock events to the temporary NDJSON file
	now := time.Now().Unix()
	events := []string{
		fmt.Sprintf(`{"EventId":1,"At":%d,"Id":"5555","Name":"Alice","RaceId":8001,"OwnerId":"0"}`, now),
		fmt.Sprintf(`{"EventId":3,"At":%d,"Id":"5555","TargetId":"9999","SkillId":123,"Damage":100,"IsCritical":false,"IsDelayed":false}`, now),
		fmt.Sprintf(`{"EventId":8,"At":%d,"Id":"5555","SkillId":123,"TargetId":"9999"}`, now),
		fmt.Sprintf(`{"EventId":9,"At":%d,"Id":"5555","SkillId":456,"TargetId":"0"}`, now),
	}

	for _, ev := range events {
		if _, err := tmpFile.WriteString(ev + "\n"); err != nil {
			t.Fatal(err)
		}
	}
	tmpFile.Close()

	// Parse file and generate summary
	summary, err := GenerateSummaryFromFile(tmpFile.Name())
	if err != nil {
		t.Fatal(err)
	}

	// Verify counts in player stats
	playerStats, exists := summary.Players[strconv.FormatUint(playerID, 10)]
	if !exists {
		t.Fatalf("Expected player %d in summary", playerID)
	}

	skill1Stats, ok1 := playerStats.OverallStats.Skills[123]
	if !ok1 || skill1Stats.Uses != 1 {
		t.Errorf("Expected skill 123 to have 1 use, got ok=%v, uses=%d", ok1, skill1Stats.Uses)
	}

	skill2Stats, ok2 := playerStats.OverallStats.Skills[456]
	if !ok2 || skill2Stats.Uses != 1 {
		t.Errorf("Expected skill 456 to have 1 use, got ok=%v, uses=%d", ok2, skill2Stats.Uses)
	}
}
