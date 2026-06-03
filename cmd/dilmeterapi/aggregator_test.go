package main

import (
	"database/sql"
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
			packet.NewMessageElemInt(317),       // sub-ID
			packet.NewMessageElemInt(5000),      // damage = 5000
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
			packet.NewMessageElemInt(999),       // type = 999 (ignored)
			packet.NewMessageElemByte(0),
			packet.NewMessageElemInt(5000),      // damage
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

func TestEventPublisher_HPVerification(t *testing.T) {
	agg := NewAggregator()
	pub := &eventPublisher{
		aggregator:           agg,
		hpVerificationStates: make(map[uint64]*hpVerificationState),
	}

	targetID := uint64(9999)
	attackerID := uint64(5555)

	// Set up target in aggregator cache
	agg.entityCache[targetID] = &packet.EntityInfo{
		Id:        targetID,
		Name:      "TestTarget",
		CurrentHP: 100,
		MaxHP:     100,
	}

	// 1. Record damage hit
	pub.recordDamageHit(targetID, attackerID, 123, 30.0, false, time.Now())

	state := pub.hpVerificationStates[targetID]
	if state == nil {
		t.Fatalf("Expected verification state to be initialized")
	}
	if state.PendingDamage != 30.0 {
		t.Errorf("Expected pending damage to be 30.0, got %f", state.PendingDamage)
	}
	if len(state.DamageHits) != 1 {
		t.Errorf("Expected 1 damage hit, got %d", len(state.DamageHits))
	}

	// 2. Validate HP change - Match (Success)
	// Mock that target HP updated to 70 in aggregator
	agg.entityCache[targetID].CurrentHP = 70.0

	// Run verification. Since it matches, it should clear pending damage.
	pub.verifyHPChange(targetID, 70.0, 100.0)

	if state.PendingDamage != 0 {
		t.Errorf("Expected pending damage to be reset to 0, got %f", state.PendingDamage)
	}
	if len(state.DamageHits) != 0 {
		t.Errorf("Expected damage hits to be cleared, got %d", len(state.DamageHits))
	}
	if state.LastCurrentHP != 70.0 {
		t.Errorf("Expected last current HP to be updated to 70.0, got %f", state.LastCurrentHP)
	}
}

func TestEventPublisher_HPVerification_Overkill(t *testing.T) {
	agg := NewAggregator()
	logCh := make(chan iEvent, 10)
	pub := &eventPublisher{
		aggregator:           agg,
		hpVerificationStates: make(map[uint64]*hpVerificationState),
		logCh:                logCh,
	}

	playerID := uint64(5555)
	targetID := uint64(9999)

	agg.entityCache[targetID] = &packet.EntityInfo{
		Id:        targetID,
		Name:      "TestTarget",
		CurrentHP: 50,
		MaxHP:     100,
	}

	// Initialize the player stats in aggregator
	pInfo := &PlayerInfo{
		ID:     playerID,
		Name:   "TestAttacker",
		RaceId: 8001,
	}
	stats := agg.getOrCreatePlayerStats(pInfo)
	stats.OverallStats.TotalDamage = 80.0
	stats.OverallStats.Skills = make(map[uint16]SkillStats)
	stats.OverallStats.Skills[123] = SkillStats{
		ID:          123,
		TotalDamage: 80.0,
	}
	
	targetIDStr := strconv.FormatUint(targetID, 10)
	targetStats := newDamageBreakdown()
	targetStats.TotalDamage = 80.0
	targetStats.Skills = make(map[uint16]SkillStats)
	targetStats.Skills[123] = SkillStats{
		ID:          123,
		TotalDamage: 80.0,
	}
	stats.DamageByTarget[targetIDStr] = targetStats

	// 1. Record damage hit
	stats.DamageTimeline = append(stats.DamageTimeline, DamageTimelineEvent{
		Timestamp:  time.Now().Unix(),
		SkillID:    123,
		TargetID:   targetIDStr,
		TargetName: "TestTarget",
		Damage:     80.0,
		CurrentHP:  50,
		MaxHP:      100,
	},)
	pub.recordDamageHit(targetID, playerID, 123, 80.0, false, time.Now())
	
	// Set the state's LastCurrentHP to 50
	pub.hpVerificationStates[targetID].LastCurrentHP = 50.0

	// 2. Validate HP change - Mismatch with Overkill
	// Target HP drops to 0 (actual delta = 50, expected = 80)
	agg.entityCache[targetID].CurrentHP = 0.0
	pub.verifyHPChange(targetID, 0.0, 100.0)

	// The recorded damage should have been corrected by subtracting 30.0 reduction
	if stats.OverallStats.TotalDamage != 50.0 {
		t.Errorf("Expected overall total damage to be corrected to 50.0, got %f", stats.OverallStats.TotalDamage)
	}
	if stats.OverallStats.Skills[123].TotalDamage != 50.0 {
		t.Errorf("Expected overall skill 123 damage to be corrected to 50.0, got %f", stats.OverallStats.Skills[123].TotalDamage)
	}
	if stats.DamageByTarget[targetIDStr].TotalDamage != 50.0 {
		t.Errorf("Expected per-target damage to be corrected to 50.0, got %f", stats.DamageByTarget[targetIDStr].TotalDamage)
	}
	if stats.DamageByTarget[targetIDStr].Skills[123].TotalDamage != 50.0 {
		t.Errorf("Expected per-target skill 123 damage to be corrected to 50.0, got %f", stats.DamageByTarget[targetIDStr].Skills[123].TotalDamage)
	}

	// Verify that the negative correction event was logged to the channel
	select {
	case ev := <-logCh:
		dmgEv, ok := ev.(*eventDamage)
		if !ok {
			t.Fatalf("Expected eventDamage type")
		}
		if dmgEv.Damage != -30.0 {
			t.Errorf("Expected correction damage to be -30.0, got %f", dmgEv.Damage)
		}
		if !dmgEv.IsCorrection {
			t.Errorf("Expected IsCorrection to be true")
		}
	default:
		t.Fatalf("Expected a correction event to be written to logCh")
	}

	if len(stats.DamageTimeline) != 1 {
		t.Errorf("Expected 1 timeline event, got %d", len(stats.DamageTimeline))
	} else {
		if stats.DamageTimeline[0].Damage != 50.0 {
			t.Errorf("Expected timeline event damage to be corrected to 50.0, got %f", stats.DamageTimeline[0].Damage)
		}
		if stats.DamageTimeline[0].Overkill != 30.0 {
			t.Errorf("Expected timeline event overkill to be 30.0, got %f", stats.DamageTimeline[0].Overkill)
		}
	}
}

func TestAggregator_PetDamageAttribution(t *testing.T) {
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

	playerID := uint64(1111)
	petID := uint64(2222)
	enemyID := uint64(3333)

	// Mock DB player info and register in playerCache
	_, err = db.Exec("INSERT INTO players (id, name, race_id) VALUES (?, ?, ?)", playerID, "Alice", 8001)
	if err != nil {
		t.Fatal(err)
	}

	agg := NewAggregator()

	// Mock entities in cache
	agg.entityCache[playerID] = &packet.EntityInfo{Id: playerID, Name: "Alice", RaceId: 8001}
	agg.entityCache[petID] = &packet.EntityInfo{Id: petID, Name: "NimbusPet", RaceId: 500, OwnerId: playerID}
	agg.entityCache[enemyID] = &packet.EntityInfo{Id: enemyID, Name: "7777", RaceId: 2000} // enemy

	// Process Effect packet where attacker is petID
	pEffect := &packet.GamePacket{
		Op: opcodeEffect,
		Id: enemyID, // target ID
		At: time.Now(),
		Msg: []packet.IMessageElem{
			packet.NewMessageElemInt(352),       // type = 352
			packet.NewMessageElemByte(0),        // skip byte
			packet.NewMessageElemInt(1500),      // damage = 1500
			packet.NewMessageElemInt(0),         // skip int
			packet.NewMessageElemLong(petID),    // attacker ID is the pet!
			packet.NewMessageElemShort(888),     // skill ID
			packet.NewMessageElemByte(0),        // skip byte
		},
	}
	agg.ProcessPacket(pEffect)

	// Verify Alice's (owner) damage stats recorded the 1500 damage
	stats := agg.playerStats[playerID]
	if stats == nil {
		t.Fatalf("Expected stats for player Alice, got nil")
	}
	if stats.OverallStats.TotalDamage != 1500 {
		t.Errorf("Expected owner Alice to have 1500 damage, got %f", stats.OverallStats.TotalDamage)
	}

	// Verify that the owner's Skills list does NOT contain skill 888 (since it was the pet's hit)
	if _, exists := stats.OverallStats.Skills[888]; exists {
		t.Errorf("Expected owner's skills list to not contain the pet's skill 888")
	}

	// Verify pet stats under owner
	petStats, exists := stats.Pets["2222"]
	if !exists {
		t.Fatalf("Expected pet stats for pet ID '2222', got nil")
	}
	if petStats.Name != "NimbusPet" {
		t.Errorf("Expected pet name to be 'NimbusPet', got '%s'", petStats.Name)
	}
	if petStats.OverallStats.TotalDamage != 1500 {
		t.Errorf("Expected pet stats to record 1500 damage, got %f", petStats.OverallStats.TotalDamage)
	}
	if petStats.OverallStats.Skills[888].TotalDamage != 1500 {
		t.Errorf("Expected pet skill 888 to record 1500 damage, got %f", petStats.OverallStats.Skills[888].TotalDamage)
	}
}
