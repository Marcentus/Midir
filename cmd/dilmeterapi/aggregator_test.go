package main

import (
	"database/sql"

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
	if !agg.seenAppear[playerID] {
		t.Errorf("Expected seenAppear to preserve/re-populate playerID after soft clear")
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

	// 4. Simulate Player Revival (DeadFeather opcode 0x5403)
	pRevPlayer := &packet.GamePacket{
		Op: opcodeDeadFeather,
		Id: playerID,
		At: time.Now(),
		Msg: []packet.IMessageElem{
			packet.NewMessageElemShort(1),
			packet.NewMessageElemInt(0),
			packet.NewMessageElemByte(0),
		},
	}
	agg.ProcessPacket(pRevPlayer)

	if agg.deadEntities[playerID] {
		t.Errorf("Expected player to be revived (alive) after DeadFeather")
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

func TestAggregator_InvincibilityFilter(t *testing.T) {
	agg := NewAggregator()
	agg.SetLive(true)

	playerID := uint64(5555)
	enemyID := uint64(7777)

	// Mock player and enemy in cache
	agg.entityCache[playerID] = &packet.EntityInfo{Id: playerID, Name: "Alice", RaceId: 8001}
	agg.entityCache[enemyID] = &packet.EntityInfo{Id: enemyID, Name: "7777", RaceId: 2000} // numeric name = enemy

	// 1. Target is NOT invincible initially
	if agg.isInvincible(enemyID) {
		t.Errorf("Expected enemy to not be invincible initially")
	}

	// 2. Enable invincibility condition 494 on enemy
	pEnable := &packet.GamePacket{
		Op: opcodeCharacterCondition,
		Id: enemyID,
		At: time.Now(),
		Msg: []packet.IMessageElem{
			packet.NewMessageElemByte(1),                     // isEnable = true
			packet.NewMessageElemInt(494),                    // ccId = 494
			packet.NewMessageElemLong(0),                     // disableAt
			packet.NewMessageElemString("Invincible Shield"), // metadata
			packet.NewMessageElemLong(0),                     // attackerId
		},
	}
	agg.ProcessPacket(pEnable)

	// Verify target is now invincible
	if !agg.isInvincible(enemyID) {
		t.Errorf("Expected enemy to be invincible after enabling condition 494")
	}

	// 3. Disable condition 494
	pDisable := &packet.GamePacket{
		Op: opcodeCharacterCondition,
		Id: enemyID,
		At: time.Now(),
		Msg: []packet.IMessageElem{
			packet.NewMessageElemByte(0),  // isEnable = false
			packet.NewMessageElemInt(494), // ccId = 494
		},
	}
	agg.ProcessPacket(pDisable)

	// Verify target is no longer invincible
	if agg.isInvincible(enemyID) {
		t.Errorf("Expected enemy to not be invincible after disabling condition 494")
	}

	// 4. Test delayed effect filtering
	// Re-enable invincibility condition 277 this time
	pEnable277 := &packet.GamePacket{
		Op: opcodeCharacterCondition,
		Id: enemyID,
		At: time.Now(),
		Msg: []packet.IMessageElem{
			packet.NewMessageElemByte(1),
			packet.NewMessageElemInt(277),
			packet.NewMessageElemLong(0),
			packet.NewMessageElemString("Shield"),
			packet.NewMessageElemLong(0),
		},
	}
	agg.ProcessPacket(pEnable277)

	if !agg.isInvincible(enemyID) {
		t.Errorf("Expected enemy to be invincible with condition 277")
	}

	// Process effect delayed packet (opcodeEffectDelayed = 0x9095)
	// Expected to set damage to 0 because target is invincible
	pDelayed := &packet.GamePacket{
		Op: opcodeEffectDelayed,
		Id: enemyID, // target ID
		At: time.Now(),
		Msg: []packet.IMessageElem{
			packet.NewMessageElemByte(0),
			packet.NewMessageElemInt(318),  // sub-ID
			packet.NewMessageElemInt(5000), // damage = 5000
			packet.NewMessageElemInt(0),
			packet.NewMessageElemInt(0),
			packet.NewMessageElemLong(playerID), // attacker
			packet.NewMessageElemShort(999),     // skill ID
		},
	}
	agg.ProcessPacket(pDelayed)

	// Verify that player Alice stats did NOT record the 5000 damage
	stats := agg.playerStats[playerID]
	if stats != nil && stats.OverallStats.TotalDamage > 0 {
		t.Errorf("Expected 0 aggregated damage due to invincibility, got %f", stats.OverallStats.TotalDamage)
	}
}

func TestAggregator_EffectPacket(t *testing.T) {
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

	playerID := uint64(5555)
	enemyID := uint64(7777)

	// Insert mock player into DB
	_, err = db.Exec("INSERT INTO players (id, name, race_id) VALUES (?, ?, ?)", playerID, "Alice", 8001)
	if err != nil {
		t.Fatal(err)
	}

	agg := NewAggregator()
	agg.SetLive(true)

	// Mock player and enemy in cache
	agg.entityCache[playerID] = &packet.EntityInfo{Id: playerID, Name: "Alice", RaceId: 8001}
	agg.entityCache[enemyID] = &packet.EntityInfo{Id: enemyID, Name: "7777", RaceId: 2000} // numeric name = enemy

	// 1. Process valid Effect packet (opcodeEffect = 0x9093, Type = 352)
	pEffect := &packet.GamePacket{
		Op: opcodeEffect,
		Id: enemyID, // target ID
		At: time.Now(),
		Msg: []packet.IMessageElem{
			packet.NewMessageElemInt(352),       // type = 352
			packet.NewMessageElemByte(0),        // skip byte
			packet.NewMessageElemInt(2500),      // damage = 2500
			packet.NewMessageElemInt(0),         // skip int
			packet.NewMessageElemLong(playerID), // attacker ID
			packet.NewMessageElemShort(888),     // skill ID
			packet.NewMessageElemByte(0),        // skip byte
		},
	}
	agg.ProcessPacket(pEffect)

	// Verify Alice's damage stats recorded the 2500 damage
	stats := agg.playerStats[playerID]
	if stats == nil {
		t.Fatalf("Expected stats for player Alice, got nil")
	}
	if stats.OverallStats.TotalDamage != 2500 {
		t.Errorf("Expected 2500 damage, got %f", stats.OverallStats.TotalDamage)
	}

	// 2. Process invalid Effect packet (type != 352)
	pInvalidType := &packet.GamePacket{
		Op: opcodeEffect,
		Id: enemyID,
		At: time.Now(),
		Msg: []packet.IMessageElem{
			packet.NewMessageElemInt(999), // type = 999 (ignored)
			packet.NewMessageElemByte(0),
			packet.NewMessageElemInt(5000), // damage
			packet.NewMessageElemInt(0),
			packet.NewMessageElemLong(playerID),
			packet.NewMessageElemShort(888),
			packet.NewMessageElemByte(0),
		},
	}
	agg.ProcessPacket(pInvalidType)

	// Total damage should still be 2500 (the 5000 is ignored)
	if stats.OverallStats.TotalDamage != 2500 {
		t.Errorf("Expected damage to remain 2500, got %f", stats.OverallStats.TotalDamage)
	}

	// 3. Process Effect packet during target invincibility
	// Enable invincibility on enemy
	pEnableInvincible := &packet.GamePacket{
		Op: opcodeCharacterCondition,
		Id: enemyID,
		At: time.Now(),
		Msg: []packet.IMessageElem{
			packet.NewMessageElemByte(1),                     // isEnable = true
			packet.NewMessageElemInt(494),                    // ccId = 494
			packet.NewMessageElemLong(0),                     // disableAt
			packet.NewMessageElemString("Invincible Shield"), // metadata
			packet.NewMessageElemLong(0),                     // attackerId
		},
	}
	agg.ProcessPacket(pEnableInvincible)

	pEffectInvincible := &packet.GamePacket{
		Op: opcodeEffect,
		Id: enemyID,
		At: time.Now(),
		Msg: []packet.IMessageElem{
			packet.NewMessageElemInt(352),
			packet.NewMessageElemByte(0),
			packet.NewMessageElemInt(4000), // damage = 4000 (should be zeroed)
			packet.NewMessageElemInt(0),
			packet.NewMessageElemLong(playerID),
			packet.NewMessageElemShort(888),
			packet.NewMessageElemByte(0),
		},
	}
	agg.ProcessPacket(pEffectInvincible)

	// Total damage should still be 2500 (the 4000 is ignored/zeroed out)
	if stats.OverallStats.TotalDamage != 2500 {
		t.Errorf("Expected damage to remain 2500 under invincibility, got %f", stats.OverallStats.TotalDamage)
	}
}
