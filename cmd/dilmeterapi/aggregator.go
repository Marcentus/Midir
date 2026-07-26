package main

import (
	"fmt"
	"sort"
	"strconv"
	"sync"
	"time"

	"strings"

	"github.com/Marcentus/Midir/packet"
)

// Aggregator holds the real-time state of the combat encounter.
type Aggregator struct {
	mu sync.RWMutex
	// Damage Dealt Data
	playerStats        map[uint64]*PlayerStats
	playerTalents      map[uint64]string
	playerTalentNames  map[uint64]string
	playerTalentColors map[uint64]string
	// Damage Taken Data
	damageTaken map[uint64]*PlayerDamageTakenStats
	// Timestamps for accurate, shared DPS calculation
	targetTimestamps map[uint64]struct {
		StartTime int64
		EndTime   int64
	}
	encounterStartTime int64
	encounterEndTime   int64
	// General Entity Info
	entityCache   map[uint64]*packet.EntityInfo
	targetNames   map[uint64]string // Cache for entity names to persist after they disappear
	targetRaceIDs map[uint64]uint32 // Cache for entity race IDs to persist after they disappear

	// Condition Tracking
	// playerConditionActive: PlayerID -> ConditionID -> ActiveCondition
	playerConditionActive map[uint64]map[uint32]ActiveCondition
	// playerConditionHistory: PlayerID -> ConditionID -> List of intervals
	playerConditionHistory map[uint64]map[uint32][]ConditionInterval
	// playerSeenAppear: PlayerID -> bool. True if we've seen an EntityAppear for this session.
	playerSeenAppear map[uint64]bool

	// Live Session Handling
	isLive              bool
	ignorePacketsBefore time.Time

	// Death Tracking Data
	deadEntities map[uint64]bool
	seenDead     map[uint64]bool
	seenAppear   map[uint64]bool
	disappeared  map[uint64]bool
	// Presence Intervals Tracking
	targetPresenceIntervals map[uint64][]PresenceInterval
	// Target HP Tracking (Live session cache)
	lastKnownHP map[uint64]TargetHPPoint
}

// NewAggregator creates and initializes a new Aggregator.
func NewAggregator() *Aggregator {
	return &Aggregator{
		playerStats:        make(map[uint64]*PlayerStats),
		playerTalents:      make(map[uint64]string),
		playerTalentNames:  make(map[uint64]string),
		playerTalentColors: make(map[uint64]string),
		damageTaken:        make(map[uint64]*PlayerDamageTakenStats),
		entityCache:        make(map[uint64]*packet.EntityInfo),
		targetNames:        make(map[uint64]string),
		targetRaceIDs:      make(map[uint64]uint32),
		targetTimestamps: make(map[uint64]struct {
			StartTime int64
			EndTime   int64
		}),
		playerConditionActive:   make(map[uint64]map[uint32]ActiveCondition),
		playerConditionHistory:  make(map[uint64]map[uint32][]ConditionInterval),
		playerSeenAppear:        make(map[uint64]bool),
		isLive:                  false, // Default to false, explicitly enabled by caller if needed
		deadEntities:            make(map[uint64]bool),
		seenDead:                make(map[uint64]bool),
		seenAppear:              make(map[uint64]bool),
		disappeared:             make(map[uint64]bool),
		targetPresenceIntervals: make(map[uint64][]PresenceInterval),
		lastKnownHP:             make(map[uint64]TargetHPPoint),
	}
}

func (a *Aggregator) SetLive(live bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.isLive = live
}

const (
	conditionInvincibleFethFiada = 494
	conditionInvincibleGeneral   = 277
)

// isInvincible checks if the given entity currently has an invincibility condition active.
// Note: This method assumes a.mu is held (either RLock or Lock) or is called within a locked context.
func (a *Aggregator) isInvincible(entityID uint64) bool {
	if active, ok := a.playerConditionActive[entityID]; ok {
		if _, has494 := active[conditionInvincibleFethFiada]; has494 {
			return true
		}
		if _, has277 := active[conditionInvincibleGeneral]; has277 {
			return true
		}
	}
	return false
}

// startPresenceInterval starts a presence interval for the target at the given timestamp.
// Note: This method assumes a.mu is held or is called within a locked context.
func (a *Aggregator) startPresenceInterval(targetID uint64, ts int64) {
	intervals := a.targetPresenceIntervals[targetID]
	// Check if there is already an active interval
	for _, iv := range intervals {
		if iv.End == 0 {
			return // Already active
		}
	}
	// Start a new active interval
	a.targetPresenceIntervals[targetID] = append(intervals, PresenceInterval{
		Start: ts,
		End:   0,
	})
}

// endPresenceInterval ends the active presence interval for the target at the given timestamp.
// Note: This method assumes a.mu is held or is called within a locked context.
func (a *Aggregator) endPresenceInterval(targetID uint64, ts int64) {
	intervals := a.targetPresenceIntervals[targetID]
	for i := len(intervals) - 1; i >= 0; i-- {
		if intervals[i].End == 0 {
			intervals[i].End = ts
			return
		}
	}
}

// isTimestampInPresenceIntervals checks if the given timestamp falls within any active or completed presence intervals for the target.
// Note: This method assumes a.mu is held or is called within a locked context.
func (a *Aggregator) isTimestampInPresenceIntervals(targetID uint64, ts int64) bool {
	intervals, exists := a.targetPresenceIntervals[targetID]
	if !exists || len(intervals) == 0 {
		// Fallback: If no intervals are recorded, check target's windowStart/windowEnd from targetTimestamps
		if times, ok := a.targetTimestamps[targetID]; ok && times.StartTime > 0 {
			end := times.EndTime
			if end == 0 || end < times.StartTime {
				end = ts // Fallback if end is 0
			}
			return ts >= times.StartTime && ts <= end
		}
		return false
	}

	for _, iv := range intervals {
		start := iv.Start
		end := iv.End
		if end == 0 {
			// Active interval: any time after Start is valid
			if ts >= start {
				return true
			}
		} else {
			if ts >= start && ts <= end {
				return true
			}
		}
	}
	return false
}

// updateTimestamps is a new helper to manage all time tracking logic.
// It's called for every damage event to keep a shared timer for each target.
func (a *Aggregator) updateTimestamps(targetId uint64, eventTime time.Time) {
	eventUnix := eventTime.Unix()

	// Update overall encounter timestamps
	if a.encounterStartTime == 0 {
		a.encounterStartTime = eventUnix
	}
	a.encounterEndTime = eventUnix

	// Update per-target timestamps
	timestamps := a.targetTimestamps[targetId]
	if timestamps.StartTime == 0 {
		timestamps.StartTime = eventUnix
	}
	timestamps.EndTime = eventUnix
	a.targetTimestamps[targetId] = timestamps

	// Ensure the name is cached before the entity potentially disappears
	a.resolveAndCacheName(targetId)

	// Start presence interval if target is active in combat and not dead
	if !a.deadEntities[targetId] {
		a.startPresenceInterval(targetId, eventUnix)
	}
}

// resolveAndCacheName attempts to find and store the name and race ID of an entity.
// This is called when an entity is involved in combat to ensure we have a name
// even if the entity disappears from the area later.
func (a *Aggregator) resolveAndCacheName(entityID uint64) {
	if _, exists := a.targetNames[entityID]; exists {
		return
	}
	if entity, ok := a.entityCache[entityID]; ok {
		a.targetNames[entityID] = getRaceName(entity.RaceId)
		a.targetRaceIDs[entityID] = entity.RaceId
	} else if player, ok := playerCache.Get(entityID); ok {
		a.targetNames[entityID] = player.Name
		a.targetRaceIDs[entityID] = player.RaceId
	}
}

// resolveAttacker looks up the attacker ID in the entity cache and, if it is a
// marionette (either by known RaceId range or by having an OwnerId and a numerical
// name template), returns the owner's ID. Otherwise, it returns the original attacker ID.
func (a *Aggregator) resolveAttacker(attackerId uint64) uint64 {
	if entity, ok := a.entityCache[attackerId]; ok {
		if entity.OwnerId != 0 {
			isMarionette := entity.EntityType == 5 || entity.EntityType == 11 || entity.EntityType == 12
			if !isMarionette {
				if _, err := strconv.Atoi(entity.Name); err == nil {
					isMarionette = true
				}
			}
			if isMarionette {
				return entity.OwnerId
			}
		}
	}
	return attackerId
}

// resolveAttackerAndSkill resolves the attacker ID (mapping puppets and pets back to their player owners).
// If the attacker is a pet (and not a puppet), it returns the owner's ID and overrides the skill ID to 9999.
// Otherwise, it attributes puppets to their owners while keeping their original skill IDs.
func (a *Aggregator) resolveAttackerAndSkill(attackerId uint64, skillId uint16) (uint64, uint16) {
	if entity, ok := a.entityCache[attackerId]; ok {
		if entity.OwnerId != 0 {
			isMarionette := entity.EntityType == 5 || entity.EntityType == 11 || entity.EntityType == 12
			if !isMarionette {
				if _, err := strconv.Atoi(entity.Name); err == nil {
					isMarionette = true
				}
			}
			if isMarionette {
				// Marionettes attribute damage to owner, but keep their original skillId
				return entity.OwnerId, skillId
			} else {
				// Pets attribute damage to owner, but group all their damage under a special custom "Pets" skillId (9999)
				return entity.OwnerId, 9999
			}
		}
	}
	return attackerId, skillId
}

// ProcessPacket is the entry point for new game data.
func (a *Aggregator) ProcessPacket(p *packet.GamePacket) {
	// If we are in a live session, we want to ignore packets that are "too old" relative to the last Clear() time.
	// This prevents buffered packets (e.g. from a paused TCP stream or just network lag) from
	// immediately dirtying a fresh session with old timestamps, effectively determining the "Start Time"
	// of the new session to be in the past.
	if a.isLive && !a.ignorePacketsBefore.IsZero() {
		if p.At.Before(a.ignorePacketsBefore) {
			// logger.Printf("Skipping old packet (Time: %v < Cutoff: %v)", p.At, a.ignorePacketsBefore)
			return
		}
	}

	if p.Op == opcodeEntityAppear {
		entity, err := packet.ParseEntityAppearPacket(p.Msg)
		if err == nil && entity != nil {
			a.mu.Lock()
			a.entityCache[entity.Id] = entity
			// Record initial HP to lastKnownHP
			if entity.MaxHP > 0 {
				a.lastKnownHP[entity.Id] = TargetHPPoint{
					Time:      p.At.Unix(),
					CurrentHP: entity.CurrentHP,
					MaxHP:     entity.MaxHP,
				}
			}
			// Mark that we have seen this entity appear, so condition tracking is reliable
			a.playerSeenAppear[entity.Id] = true
			a.seenAppear[entity.Id] = true
			a.disappeared[entity.Id] = false
			a.startPresenceInterval(entity.Id, p.At.Unix())

			// Initialize existing conditions from the appear packet
			if a.playerConditionActive[entity.Id] == nil {
				a.playerConditionActive[entity.Id] = make(map[uint32]ActiveCondition)
			}
			for _, cond := range entity.CharacterConditionMap {
				// If not already active in our tracker, add it
				if _, exists := a.playerConditionActive[entity.Id][cond.CCId]; !exists {
					a.playerConditionActive[entity.Id][cond.CCId] = ActiveCondition{
						Start:      p.At.Unix(),
						DisableAt:  cond.DisableAt, // NEW
						MetaData:   normalizeMetaData(cond.MetaData),
						AttackerID: cond.AttackerId,
					}
				}
			}
			a.mu.Unlock()
			playerCache.Update(entity)
		}
		return
	}
	if p.Op == opcodeEntitiesAppear {
		entities, err := packet.ParseEntitiesAppearPacket(p)
		if err == nil {
			a.mu.Lock()
			for _, entity := range entities {
				a.entityCache[entity.Id] = entity
				// Record initial HP to lastKnownHP
				if entity.MaxHP > 0 {
					a.lastKnownHP[entity.Id] = TargetHPPoint{
						Time:      p.At.Unix(),
						CurrentHP: entity.CurrentHP,
						MaxHP:     entity.MaxHP,
					}
				}
				a.playerSeenAppear[entity.Id] = true
				a.seenAppear[entity.Id] = true
				a.disappeared[entity.Id] = false
				a.startPresenceInterval(entity.Id, p.At.Unix())

				if a.playerConditionActive[entity.Id] == nil {
					a.playerConditionActive[entity.Id] = make(map[uint32]ActiveCondition)
				}
				for _, cond := range entity.CharacterConditionMap {
					if _, exists := a.playerConditionActive[entity.Id][cond.CCId]; !exists {
						a.playerConditionActive[entity.Id][cond.CCId] = ActiveCondition{
							Start:      p.At.Unix(),
							DisableAt:  cond.DisableAt, // NEW
							MetaData:   normalizeMetaData(cond.MetaData),
							AttackerID: cond.AttackerId,
						}
					}
				}
				playerCache.Update(entity)
			}
			a.mu.Unlock()
		}
		return
	}

	if p.Op == opcodeCombatAction {
		a.processCombatAction(p)
	}
	if p.Op == opcodeEffect {
		a.processEffect(p)
	}
	if p.Op == opcodeEffectDelayed {
		a.processEffectDelayed(p)
	}

	if p.Op == opcodeCharacterCondition {
		// Handle condition add/remove updates
		if cond, err := packet.ParseCharacterConditionPacket(p); err == nil {
			a.processCharacterCondition(cond, p.At)
		}
	}
	if p.Op == opcodeEntityDisappear {
		// UPDATED: Use the new parser to get the correct ID
		if id, err := packet.ParseEntityDisappearPacket(p); err == nil {
			a.processEntityDisappear(id, p.At.Unix())
		}
	}
	if p.Op == opcodeEntitiesDisappear {
		// NEW: Handle batch disappear for the live aggregator
		if ids, err := packet.ParseEntitiesDisappearPacket(p); err == nil {
			for _, id := range ids {
				a.processEntityDisappear(id, p.At.Unix())
			}
		}
	}
	if p.Op == opcodeIsNowDead {
		a.processIsNowDead(p)
	}
	if p.Op == opcodeSetFinisher {
		a.processSetFinisher(p)
	}
	if p.Op == opcodeDeadFeather {
		a.processDeadFeather(p)
	}
	if p.Op == opcodePublicStatUpdate {
		a.processPublicStatUpdate(p)
	}
}

func (a *Aggregator) processEffect(p *packet.GamePacket) {
	if len(p.Msg) < 7 ||
		p.Msg[0].Type() != packet.MessageElemTypeInt ||
		p.Msg[0].Data().(uint32) != 352 ||
		p.Msg[1].Type() != packet.MessageElemTypeByte ||
		p.Msg[2].Type() != packet.MessageElemTypeInt ||
		p.Msg[3].Type() != packet.MessageElemTypeInt ||
		p.Msg[4].Type() != packet.MessageElemTypeLong ||
		p.Msg[5].Type() != packet.MessageElemTypeShort ||
		p.Msg[6].Type() != packet.MessageElemTypeByte {
		return
	}
	damage := float32(p.Msg[2].Data().(uint32))
	attackerId := p.Msg[4].Data().(uint64)
	skillId := p.Msg[5].Data().(uint16)
	targetId := p.Id

	a.mu.Lock()
	defer a.mu.Unlock()

	attackerId, skillId = a.resolveAttackerAndSkill(attackerId, skillId)

	if a.isInvincible(targetId) {
		damage = 0
	}

	// Update the shared timers for this target
	a.updateTimestamps(targetId, p.At)

	if attackerInfo, isPlayer := playerCache.Get(attackerId); isPlayer {
		// Check if we have identified the talent color yet.
		if _, known := a.playerTalentColors[attackerId]; !known {
			if iconPath, found := skillToArcanaIcon[skillId]; found {
				a.playerTalents[attackerId] = iconPath
				if name, ok := skillToArcanaName[skillId]; ok {
					a.playerTalentNames[attackerId] = name
				}
				if color, ok := skillToArcanaColor[skillId]; ok {
					a.playerTalentColors[attackerId] = color
				}
			}
		}
		stats := a.getOrCreatePlayerStats(attackerInfo)
		targetIdStr := strconv.FormatUint(targetId, 10)
		tempHitPacket := &packet.CombatActionPacket{Hit: &packet.CombatActionPacketHitInfo{Damage: damage}}
		a.updateBreakdown(&stats.OverallStats, tempHitPacket, skillId, true)
		a.updatePerTargetBreakdown(stats, targetIdStr, tempHitPacket, skillId, true)
	}

	if targetInfo, isPlayerTarget := playerCache.Get(targetId); isPlayerTarget {
		a.updateDamageTaken(targetInfo, attackerId, skillId, damage, 0)
	}
}

func (a *Aggregator) processEffectDelayed(p *packet.GamePacket) {
	if len(p.Msg) < 7 ||
		p.Msg[1].Type() != packet.MessageElemTypeInt ||
		p.Msg[1].Data().(uint32) != 318 ||
		p.Msg[2].Type() != packet.MessageElemTypeInt ||
		p.Msg[5].Type() != packet.MessageElemTypeLong ||
		p.Msg[6].Type() != packet.MessageElemTypeShort {
		return
	}
	damage := float32(p.Msg[2].Data().(uint32))
	attackerId := p.Msg[5].Data().(uint64)
	skillId := p.Msg[6].Data().(uint16)
	targetId := p.Id

	// logger.Println("[Locking] Aggregator.EffectDelayed attempting to lock...")
	a.mu.Lock()
	// logger.Println("...[Locked] Aggregator.EffectDelayed acquired lock.")
	defer func() {
		// logger.Println("[Unlocking] Aggregator.EffectDelayed attempting to unlock.")
		a.mu.Unlock()
		// logger.Println("...[Unlocked] Aggregator.EffectDelayed released lock.")
	}()

	attackerId, skillId = a.resolveAttackerAndSkill(attackerId, skillId)

	if a.isInvincible(targetId) {
		damage = 0
	}

	// Update the shared timers for this target
	a.updateTimestamps(targetId, p.At)

	if attackerInfo, isPlayer := playerCache.Get(attackerId); isPlayer {
		// Check if we have identified the talent color yet.
		if _, known := a.playerTalentColors[attackerId]; !known {
			if iconPath, found := skillToArcanaIcon[skillId]; found {
				a.playerTalents[attackerId] = iconPath
				if name, ok := skillToArcanaName[skillId]; ok {
					a.playerTalentNames[attackerId] = name
				}
				if color, ok := skillToArcanaColor[skillId]; ok {
					a.playerTalentColors[attackerId] = color
					// logger.Printf("Assigned color %s to attacker %d based on delayed skill %d", color, attackerId, skillId)
				}
			}
		}
		stats := a.getOrCreatePlayerStats(attackerInfo)
		targetIdStr := strconv.FormatUint(targetId, 10)
		tempHitPacket := &packet.CombatActionPacket{Hit: &packet.CombatActionPacketHitInfo{Damage: damage}}
		a.updateBreakdown(&stats.OverallStats, tempHitPacket, skillId, true)
		a.updatePerTargetBreakdown(stats, targetIdStr, tempHitPacket, skillId, true)
	}

	if targetInfo, isPlayerTarget := playerCache.Get(targetId); isPlayerTarget {
		a.updateDamageTaken(targetInfo, attackerId, skillId, damage, 0)
	}
}

func (a *Aggregator) processPublicStatUpdate(p *packet.GamePacket) {
	statUpdate, err := packet.ParsePublicStatUpdatePacket(p)
	if err != nil {
		return
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	entity, ok := a.entityCache[statUpdate.EntityId]
	if !ok {
		return
	}

	hpChanged := false
	baseChanged := false
	bonusChanged := false

	if val, ok := statUpdate.Stats[28]; ok {
		entity.CurrentHP = val
		hpChanged = true
	}
	if val, ok := statUpdate.Stats[30]; ok {
		entity.BaseHP = val
		baseChanged = true
	}
	if val, ok := statUpdate.Stats[31]; ok {
		entity.AdditionalHP = val
		bonusChanged = true
	}

	if baseChanged || bonusChanged {
		entity.MaxHP = entity.BaseHP + entity.AdditionalHP
	}
	if hpChanged || baseChanged || bonusChanged {
		if entity.MaxHP > 0 {
			a.lastKnownHP[statUpdate.EntityId] = TargetHPPoint{
				Time:      p.At.Unix(),
				CurrentHP: entity.CurrentHP,
				MaxHP:     entity.MaxHP,
			}
		}
	}
}

// GetEntityHP retrieves the current HP fields for a cached entity in a thread-safe manner.
func (a *Aggregator) GetEntityHP(entityID uint64) (currentHp, baseHp, additionalHp, maxHp float32, exists bool) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	entity, ok := a.entityCache[entityID]
	if !ok {
		return 0, 0, 0, 0, false
	}
	return entity.CurrentHP, entity.BaseHP, entity.AdditionalHP, entity.MaxHP, true
}

func (a *Aggregator) processCharacterCondition(p *packet.CharacterConditionPacket, at time.Time) {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Ensure maps exist
	if a.playerConditionActive[p.Id] == nil {
		a.playerConditionActive[p.Id] = make(map[uint32]ActiveCondition)
	}
	if a.playerConditionHistory[p.Id] == nil {
		a.playerConditionHistory[p.Id] = make(map[uint32][]ConditionInterval)
	}

	if p.IsEnable {
		// Start condition if not already active
		if _, active := a.playerConditionActive[p.Id][p.CCId]; !active {
			a.playerConditionActive[p.Id][p.CCId] = ActiveCondition{
				Start:      at.Unix(),
				DisableAt:  p.DisableAt, // NEW
				MetaData:   normalizeMetaData(p.MetaData),
				AttackerID: p.AttackerId,
			}
		} else {
			// Update existing condition (e.g. refresh duration)
			// fmt.Printf("[DEBUG] Condition Update - Entity: %d, CCId: %d, DisableAt: %d, Meta: %s\n", p.Id, p.CCId, p.DisableAt, p.MetaData)
			existing := a.playerConditionActive[p.Id][p.CCId]
			existing.DisableAt = p.DisableAt
			existing.MetaData = normalizeMetaData(p.MetaData)
			if p.AttackerId != 0 {
				existing.AttackerID = p.AttackerId
			}
			a.playerConditionActive[p.Id][p.CCId] = existing
		}
	} else {
		// End condition if active
		if activeCond, active := a.playerConditionActive[p.Id][p.CCId]; active {
			interval := ConditionInterval{
				Start:      activeCond.Start,
				End:        at.Unix(),
				MetaData:   activeCond.MetaData,
				AttackerID: activeCond.AttackerID,
			}
			a.playerConditionHistory[p.Id][p.CCId] = append(a.playerConditionHistory[p.Id][p.CCId], interval)
			delete(a.playerConditionActive[p.Id], p.CCId)
		}
	}
}

func (a *Aggregator) processCombatAction(p *packet.GamePacket) {
	pack, err := packet.ParseCombatActionPackPacket(p)
	if err != nil {
		return
	}

	// logger.Println("[Locking] Aggregator.CombatAction attempting to lock...")
	a.mu.Lock()
	// logger.Println("...[Locked] Aggregator.CombatAction acquired lock.")
	defer func() {
		// logger.Println("[Unlocking] Aggregator.CombatAction attempting to unlock.")
		a.mu.Unlock()
		// logger.Println("...[Unlocked] Aggregator.CombatAction released lock.")
	}()

	var attackerId uint64
	var attackSkillId uint16
	for _, sub := range pack.SubPackets {
		if sub.Type&packet.CombatActionTypeAttacker != 0 {
			attackerId = sub.EntityId
			attackSkillId = sub.SkillId
			break
		}
	}

	if attackerId == 0 {
		for _, sub := range pack.SubPackets {
			if sub.Hit != nil && sub.Hit.AttackerId != 0 {
				attackerId = sub.Hit.AttackerId
				break
			}
		}
	}

	if attackerId == 0 {
		return
	}

	attackerId, attackSkillId = a.resolveAttackerAndSkill(attackerId, attackSkillId)

	// Check if we have identified the talent color yet. If not, try to identify it.
	if _, known := a.playerTalentColors[attackerId]; !known {
		if iconPath, found := skillToArcanaIcon[attackSkillId]; found {
			a.playerTalents[attackerId] = iconPath
			if name, ok := skillToArcanaName[attackSkillId]; ok {
				a.playerTalentNames[attackerId] = name
			}
			if color, ok := skillToArcanaColor[attackSkillId]; ok {
				a.playerTalentColors[attackerId] = color
				// logger.Printf("Assigned color %s to attacker %d based on skill %d", color, attackerId, attackSkillId)
			}
		}
	}

	for _, sub := range pack.SubPackets {
		if sub.Hit != nil && a.isInvincible(sub.EntityId) {
			sub.Hit.Damage = 0
			sub.Hit.ManaDamage = 0
		}

		if sub.Hit == nil || (sub.Hit.Damage <= 0 && sub.Hit.ManaDamage <= 0) {
			continue
		}
		// Update the shared timers for this target, regardless of who hit it
		a.updateTimestamps(sub.EntityId, p.At)

		// If the attacker is a player, update their damage dealt stats
		if sub.Hit.Damage > 0 {
			if playerInfo, isPlayer := playerCache.Get(attackerId); isPlayer {
				stats := a.getOrCreatePlayerStats(playerInfo)
				a.updateBreakdown(&stats.OverallStats, sub, attackSkillId, false)
				targetIdStr := strconv.FormatUint(sub.EntityId, 10)
				a.updatePerTargetBreakdown(stats, targetIdStr, sub, attackSkillId, false)
			}
		}

		// If the target is a player, update their damage taken stats
		if targetInfo, isPlayerTarget := playerCache.Get(sub.EntityId); isPlayerTarget {
			a.updateDamageTaken(targetInfo, attackerId, attackSkillId, sub.Hit.Damage, float32(sub.Hit.ManaDamage))
		}
	}
}

func (a *Aggregator) getOrCreatePlayerStats(playerInfo *PlayerInfo) *PlayerStats {
	stats, exists := a.playerStats[playerInfo.ID]
	if !exists {
		stats = &PlayerStats{
			ID:             strconv.FormatUint(playerInfo.ID, 10),
			Name:           playerInfo.Name,
			OverallStats:   newDamageBreakdown(),
			DamageByTarget: make(map[string]DamageBreakdown),
		}
		a.playerStats[playerInfo.ID] = stats
	}
	return stats
}

func (a *Aggregator) updatePerTargetBreakdown(stats *PlayerStats, targetIdStr string, hitPacket *packet.CombatActionPacket, skillId uint16, isDelayed bool) {
	targetBreakdown, exists := stats.DamageByTarget[targetIdStr]
	if !exists {
		targetBreakdown = newDamageBreakdown()
	}
	a.updateBreakdown(&targetBreakdown, hitPacket, skillId, isDelayed)
	stats.DamageByTarget[targetIdStr] = targetBreakdown
}

func (a *Aggregator) updateBreakdown(breakdown *DamageBreakdown, hitPacket *packet.CombatActionPacket, skillId uint16, isDelayed bool) {
	damage := hitPacket.Hit.Damage
	isCrit := !isDelayed && (hitPacket.Hit.Options&packet.CombatActionHitOptionsCritical) != 0
	breakdown.TotalDamage += damage
	if !isDelayed && skillId != 9999 {
		breakdown.HitCount++
		if isCrit {
			breakdown.CritCount++
		}
	}
	skillStats := breakdown.Skills[skillId]
	skillStats.ID = skillId
	skillStats.TotalDamage += damage
	isPrimaryHit := !isDelayed
	isCountableDelayedHit := isDelayed && doCountDelayedSkills[skillId]
	if isPrimaryHit || isCountableDelayedHit {
		skillStats.Count++
	}
	if isCrit {
		skillStats.CritCount++
		skillStats.TotalDamageCrit += damage
		if damage > skillStats.MaxDamageCrit {
			skillStats.MaxDamageCrit = damage
		}
	} else {
		skillStats.TotalDamageNonCrit += damage
		if damage > skillStats.MaxDamageNonCrit {
			skillStats.MaxDamageNonCrit = damage
		}
	}
	if damage > skillStats.MaxDamage {
		skillStats.MaxDamage = damage
	}
	breakdown.Skills[skillId] = skillStats
}

func (a *Aggregator) updateDamageTaken(target *PlayerInfo, attackerID uint64, skillID uint16, damage float32, manaDamage float32) {
	stats, exists := a.damageTaken[target.ID]
	if !exists {
		stats = &PlayerDamageTakenStats{
			PlayerID:   strconv.FormatUint(target.ID, 10),
			PlayerName: target.Name,
			Breakdown:  make(map[string]DamageTakenDetails),
		}
		a.damageTaken[target.ID] = stats
	}
	stats.TotalDamage += damage + manaDamage
	stats.TotalManaDamage += manaDamage

	// Resolve Attacker Name first
	attackerName := "Unknown"
	if entity, ok := a.entityCache[attackerID]; ok {
		attackerName = getRaceName(entity.RaceId)
	} else if player, ok := playerCache.Get(attackerID); ok {
		attackerName = player.Name
	}

	// Group by Attacker Name and Skill ID
	breakdownKey := fmt.Sprintf("%s-%d", attackerName, skillID)
	details, exists := stats.Breakdown[breakdownKey]
	if !exists {
		details = DamageTakenDetails{
			AttackerID:   attackerID, // Just use the first one seen as representative
			AttackerName: attackerName,
			SkillID:      skillID,
		}
	}
	details.TotalDamage += damage + manaDamage
	details.TotalManaDamage += manaDamage
	details.HitCount++
	totalHitDamage := damage + manaDamage
	if totalHitDamage > details.MaxDamage {
		details.MaxDamage = totalHitDamage
	}
	if details.MinDamage == 0 || totalHitDamage < details.MinDamage {
		details.MinDamage = totalHitDamage
	}
	stats.Breakdown[breakdownKey] = details
}

// GetSummary now uses the shared timestamps to calculate all DPS values.
func (a *Aggregator) GetSummary() FightSummary {
	a.mu.RLock()
	defer a.mu.RUnlock()

	summary := FightSummary{
		Players:     make(map[string]PlayerStats),
		Targets:     make(map[string]TargetStats),
		DamageTaken: make(map[string]PlayerDamageTakenStats),
	}

	// Set overall duration from the shared encounter timers
	if a.encounterEndTime > a.encounterStartTime {
		summary.EncounterDuration = float64(a.encounterEndTime - a.encounterStartTime)
	}
	summary.StartTime = a.encounterStartTime
	summary.EndTime = a.encounterEndTime

	uniqueTargets := make(map[uint64]bool)
	var totalDamage float32

	for playerID, pStats := range a.playerStats {
		playerCopy := PlayerStats{
			ID:                  pStats.ID,
			Name:                pStats.Name,
			TalentIcon:          a.playerTalents[playerID],
			TalentName:          a.playerTalentNames[playerID],
			TalentColor:         a.playerTalentColors[playerID],
			DamageByTarget:      make(map[string]DamageBreakdown),
			MissingAppearPacket: !a.playerSeenAppear[playerID], // Set the flag
		}

		// Finalize overall stats using the single overall encounter duration
		playerCopy.OverallStats = a.finalizeBreakdown(pStats.OverallStats, summary.EncounterDuration, a.encounterStartTime, a.encounterEndTime, playerID, true, 0)
		playerCopy.OverallStats.StartTime = a.encounterStartTime
		playerCopy.OverallStats.EndTime = a.encounterEndTime

		// Finalize per-target stats using the shared duration for each specific target
		for targetIdStr, breakdown := range pStats.DamageByTarget {
			targetIdUint, _ := strconv.ParseUint(targetIdStr, 10, 64)
			targetTimes := a.targetTimestamps[targetIdUint]
			targetDuration := float64(targetTimes.EndTime - targetTimes.StartTime)

			finalizedBreakdown := a.finalizeBreakdown(breakdown, targetDuration, targetTimes.StartTime, targetTimes.EndTime, playerID, false, targetIdUint)
			finalizedBreakdown.StartTime = targetTimes.StartTime
			finalizedBreakdown.EndTime = targetTimes.EndTime
			playerCopy.DamageByTarget[targetIdStr] = finalizedBreakdown
			uniqueTargets[targetIdUint] = true
		}

		summary.Players[pStats.ID] = playerCopy
		totalDamage += playerCopy.OverallStats.TotalDamage
	}

	summary.TotalDamage = totalDamage
	for _, dtStats := range a.damageTaken {
		summary.DamageTaken[dtStats.PlayerID] = *dtStats
	}

	for targetId := range uniqueTargets {
		targetIdStr := strconv.FormatUint(targetId, 10)
		var name string
		if cachedName, ok := a.targetNames[targetId]; ok {
			name = cachedName
		} else if entity, ok := a.entityCache[targetId]; ok {
			name = getRaceName(entity.RaceId)
		} else {
			name = "Unknown"
		}

		var raceId uint32
		if cachedRaceId, ok := a.targetRaceIDs[targetId]; ok {
			raceId = cachedRaceId
		} else if entity, ok := a.entityCache[targetId]; ok {
			raceId = entity.RaceId
		}

		// Calculate conditions for target
		targetTimes := a.targetTimestamps[targetId]
		targetDuration := float64(targetTimes.EndTime - targetTimes.StartTime)
		conditions := a.calculateConditions(targetId, targetDuration, targetTimes.StartTime, targetTimes.EndTime)

		var hpHistory []TargetHPPoint
		if lastHP, ok := a.lastKnownHP[targetId]; ok {
			hpHistory = []TargetHPPoint{lastHP}
		}

		summary.Targets[targetIdStr] = TargetStats{
			Name:        name,
			RaceID:      raceId,
			Conditions:  conditions,
			SeenDead:    a.seenDead[targetId],
			SeenAppear:  a.seenAppear[targetId],
			Disappeared: a.disappeared[targetId],
			StartTime:   targetTimes.StartTime,
			EndTime:     targetTimes.EndTime,
			HPHistory:   hpHistory,
		}
	}

	// NEW: Populate Current Entities
	for entityID, entity := range a.entityCache {
		// Calculate conditions for current entity
		// We use the direct active map for "Current Entities" snapshots
		var conditions map[uint32]ActiveCondition
		if active, ok := a.playerConditionActive[entityID]; ok {
			conditions = make(map[uint32]ActiveCondition, len(active))
			for k, v := range active {
				conditions[k] = v
			}
		}

		category := getEntityCategory(entity)

		ownerIDStr := ""
		ownerName := ""
		if entity.OwnerId != 0 {
			ownerIDStr = strconv.FormatUint(entity.OwnerId, 10)
			if player, ok := playerCache.Get(entity.OwnerId); ok {
				ownerName = player.Name
			}
		}

		secOwnerIDStr := ""
		secOwnerName := ""
		if entity.SecondaryOwnerId != 0 {
			secOwnerIDStr = strconv.FormatUint(entity.SecondaryOwnerId, 10)
			if player, ok := playerCache.Get(entity.SecondaryOwnerId); ok {
				secOwnerName = player.Name
			}
		}

		summary.CurrentEntities = append(summary.CurrentEntities, EntityState{
			ID:                 strconv.FormatUint(entityID, 10),
			Name:               entity.Name,
			RaceID:             entity.RaceId,
			RaceName:           getRaceName(entity.RaceId),
			Conditions:         conditions,
			CurrentHP:          entity.CurrentHP,
			MaxHP:              entity.MaxHP,
			Category:           category,
			OwnerID:            ownerIDStr,
			OwnerName:          ownerName,
			SecondaryOwnerID:   secOwnerIDStr,
			SecondaryOwnerName: secOwnerName,
			EntityType:         entity.EntityType,
			EntityTypeStr:      a.GetEntityTypeString(entity),
		})
	}

	// Sort entities by name, using ID as a tie-breaker to prevent list jitter (since identical names are common and Go map iteration is random)
	sort.Slice(summary.CurrentEntities, func(i, j int) bool {
		if summary.CurrentEntities[i].Name == summary.CurrentEntities[j].Name {
			return summary.CurrentEntities[i].ID < summary.CurrentEntities[j].ID
		}
		return summary.CurrentEntities[i].Name < summary.CurrentEntities[j].Name
	})

	computePartyBuffs(&summary)

	return summary
}

// getEntityCategory classifies an entity into categorized groups: Players, Marionettes, Pets, Dollbags, Golems, NPCs, Enemies, or Other.
func getEntityCategory(entity *packet.EntityInfo) string {
	if strings.HasPrefix(entity.Name, "_") {
		return "NPCs"
	}

	if isPlayerInfo(entity.Name, entity.RaceId, entity.OwnerId) {
		return "Players"
	}

	effectiveOwner := entity.OwnerId
	if effectiveOwner == 0 {
		effectiveOwner = entity.SecondaryOwnerId
	}

	if effectiveOwner != 0 {
		switch entity.EntityType {
		case 2:
			return "Pets"
		case 6:
			return "Dollbags"
		case 7:
			return "Mini-Gems"
		case 8:
			return "Golems"
		case 5, 11, 12:
			return "Marionettes"
		default:
			if entity.EntityType != 0 {
				return "Unknown Summons"
			}
			isMarionette := entity.EntityType == 5 || entity.EntityType == 11 || entity.EntityType == 12
			if !isMarionette {
				if _, err := strconv.Atoi(entity.Name); err == nil {
					isMarionette = true
				}
			}
			if isMarionette {
				return "Marionettes"
			}
			return "Pets"
		}
	}

	return "Enemies"
}

// isPlayerEntity determines if an entity is a player based on Name and OwnerId.
// Logic:
// 1. Pets/Summons have OwnerId != 0.
// 2. NPCs start with "_".
// 3. Enemies have numeric-only names.
func isPlayerEntity(entity *packet.EntityInfo) bool {
	if strings.TrimSpace(entity.Name) == "" {
		return false
	}

	if entity.OwnerId != 0 {
		return false
	}

	if strings.HasPrefix(entity.Name, "_") {
		return false
	}

	// Check if name is numeric (Enemy)
	if _, err := strconv.Atoi(entity.Name); err == nil {
		return false
	}

	return true
}

// finalizeBreakdown calculates DPS and Condition Uptime based on a provided duration and time window.
func (a *Aggregator) finalizeBreakdown(breakdown DamageBreakdown, duration float64, windowStart, windowEnd int64, playerID uint64, isOverall bool, targetID uint64) DamageBreakdown {
	if duration > 1 {
		breakdown.DPS = breakdown.TotalDamage / float32(duration)
	} else if breakdown.TotalDamage > 0 {
		breakdown.DPS = breakdown.TotalDamage // For very short fights, DPS equals total damage
	}

	if breakdown.HitCount > 0 {
		breakdown.CritRate = (float32(breakdown.CritCount) / float32(breakdown.HitCount)) * 100
	}

	// Calculate Condition Uptime specific to this window using helper
	breakdown.Conditions = a.calculateConditions(playerID, duration, windowStart, windowEnd)

	if breakdown.Skills == nil {
		breakdown.Skills = make(map[uint16]SkillStats)
	}

	return breakdown
}

// calculateConditions calculates condition uptime for a given entity within a time window.
// calculateConditions calculates condition uptime for a given entity within a time window.
func (a *Aggregator) calculateConditions(entityID uint64, duration float64, windowStart, windowEnd int64) map[uint32]*ConditionStats {
	conditions := make(map[uint32]*ConditionStats)
	allIntervals := make(map[uint32][]ConditionInterval)

	// 1. Add historical intervals
	if history, ok := a.playerConditionHistory[entityID]; ok {
		for ccID, intervals := range history {
			allIntervals[ccID] = append(allIntervals[ccID], intervals...)
		}
	}

	// 2. Add currently active intervals (Start -> Current Window End)
	if active, ok := a.playerConditionActive[entityID]; ok {
		for ccID, activeCond := range active {
			// If the condition started before the window ended, it counts.
			// We clamp the end of the ongoing condition to the end of the combat window for calculation purposes.
			if activeCond.Start < windowEnd {
				allIntervals[ccID] = append(allIntervals[ccID], ConditionInterval{
					Start:      activeCond.Start,
					End:        windowEnd,
					MetaData:   activeCond.MetaData,
					AttackerID: activeCond.AttackerID,
				})
			}
		}
	}

	// 3. Calculate Intersection with Window [windowStart, windowEnd]
	for ccID, intervals := range allIntervals {
		var totalActiveTime int64
		var finalIntervals []ConditionInterval

		// Breakdown tracking
		metaStatsMap := make(map[string]*ConditionMetaStats)

		for _, iv := range intervals {
			// Intersection logic: max(start1, start2) to min(end1, end2)
			start := iv.Start
			if start < windowStart {
				start = windowStart
			}

			end := iv.End
			if end > windowEnd {
				end = windowEnd
			}

			if start < end {
				duration := end - start
				totalActiveTime += duration

				actualInterval := ConditionInterval{
					Start:      start,
					End:        end,
					MetaData:   iv.MetaData,
					AttackerID: iv.AttackerID,
				}
				finalIntervals = append(finalIntervals, actualInterval)

				// Meta breakdown
				metaKey := iv.MetaData
				if metaKey == "" {
					metaKey = "Unknown"
				}

				if _, ok := metaStatsMap[metaKey]; !ok {
					metaStatsMap[metaKey] = &ConditionMetaStats{
						MetaData:  metaKey,
						Attackers: []uint64{},
					}
				}
				metaStatsMap[metaKey].Duration += float64(duration)

				// Add attacker if unique
				found := false
				for _, id := range metaStatsMap[metaKey].Attackers {
					if id == iv.AttackerID {
						found = true
						break
					}
				}
				if !found && iv.AttackerID != 0 {
					metaStatsMap[metaKey].Attackers = append(metaStatsMap[metaKey].Attackers, iv.AttackerID)
				}
			}
		}

		// Avoid division by zero
		var uptimePercent float32
		if duration > 0 {
			uptimePercent = (float32(totalActiveTime) / float32(duration)) * 100.0
		}

		// Only add if there was some uptime relevant to this window
		if totalActiveTime > 0 {
			// Convert map to slice
			var metaBreakdown []ConditionMetaStats
			for _, stats := range metaStatsMap {
				if duration > 0 {
					stats.Uptime = (float32(stats.Duration) / float32(duration)) * 100.0
				}
				metaBreakdown = append(metaBreakdown, *stats)
			}

			conditions[ccID] = &ConditionStats{
				ID:            ccID,
				Uptime:        uptimePercent,
				Duration:      float64(totalActiveTime),
				Intervals:     finalIntervals,
				MetaBreakdown: metaBreakdown,
			}
		}
	}
	return conditions
}

func (a *Aggregator) processEntityDisappear(entityID uint64, ts int64) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.endPresenceInterval(entityID, ts)

	// packet ID extraction moved to caller

	// Remove from all tracking maps to free memory and "forget" the entity
	delete(a.entityCache, entityID)
	delete(a.playerSeenAppear, entityID)
	delete(a.playerConditionActive, entityID)
	delete(a.deadEntities, entityID)

	// If it has died, we should consider it dead, not disappeared. Death takes priority.
	if !a.seenDead[entityID] {
		a.disappeared[entityID] = true
	}

	// Note: We do NOT delete playerStats, targetNames, damageTaken, playerTalents, or playerConditionHistory here.
	// Rationale: If a player does 1M damage and then disconnects/teleports,
	// their contribution to the *current session* should still be visible until "Clear" is pressed.
}

func (a *Aggregator) Clear() {
	// logger.Println("[Locking] Aggregator.Clear attempting to lock...")
	a.mu.Lock()
	// logger.Println("...[Locked] Aggregator.Clear acquired lock.")
	defer func() {
		// logger.Println("[Unlocking] Aggregator.Clear attempting to unlock.")
		a.mu.Unlock()
		// logger.Println("...[Unlocked] Aggregator.Clear released lock.")
	}()

	// SOFT CLEAR: Reset only the session metrics.
	// Preserve: entityCache, playerTalents, playerTalentNames, playerTalentColors,
	//           playerConditionActive, playerSeenAppear.

	a.playerStats = make(map[uint64]*PlayerStats)
	// We keep entityCache to know who people are if they are still here
	// We keep playerTalents/Names/Colors to know who people are if they are still here

	a.damageTaken = make(map[uint64]*PlayerDamageTakenStats)
	a.targetNames = make(map[uint64]string)
	a.targetRaceIDs = make(map[uint64]uint32)
	a.targetTimestamps = make(map[uint64]struct {
		StartTime int64
		EndTime   int64
	})
	a.encounterStartTime = 0
	a.encounterEndTime = 0
	a.deadEntities = make(map[uint64]bool)
	a.seenDead = make(map[uint64]bool)
	a.seenAppear = make(map[uint64]bool)
	for id := range a.entityCache {
		a.seenAppear[id] = true
	}
	a.disappeared = make(map[uint64]bool)
	a.targetPresenceIntervals = make(map[uint64][]PresenceInterval)
	a.lastKnownHP = make(map[uint64]TargetHPPoint)

	// Clear condition HISTORY, but keep ACTIVE conditions.
	// This ensures that when the new session starts, we know they still have the buff,
	// but the *uptime percentage* starts fresh from 0 in the new session.
	a.playerConditionHistory = make(map[uint64]map[uint32][]ConditionInterval)
	// playerConditionActive is PRESERVED.

	// If we are live, set the cutoff time to now minus a small grace period (e.g. 3 seconds).
	// This ensures that when the user clicks "Clear", we mostly start fresh, but don't lose
	// packets that literally just arrived.
	if a.isLive {
		a.ignorePacketsBefore = time.Now().Add(-3 * time.Second)
	} else {
		a.ignorePacketsBefore = time.Time{}
	}
}

// isPlayer determines if an entity ID belongs to a player.
// Note: This method assumes a.mu is held or is called within a locked context.
func (a *Aggregator) isPlayer(entityID uint64) bool {
	// First check playerCache
	if _, isPlayer := playerCache.Get(entityID); isPlayer {
		return true
	}
	// Then check entityCache
	if entity, ok := a.entityCache[entityID]; ok {
		return isPlayerEntity(entity)
	}
	return false
}

func (a *Aggregator) IsPlayerSafe(entityID uint64) bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.isPlayer(entityID)
}

// IsCombatTarget checks if an entity ID is tracked in the active combat target timestamps.
func (a *Aggregator) IsCombatTarget(entityID uint64) bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	_, exists := a.targetTimestamps[entityID]
	return exists
}

// getEntityName resolves and returns the name of an entity ID.
// Note: This method assumes a.mu is held or is called within a locked context.
func (a *Aggregator) getEntityName(entityID uint64) string {
	if cachedName, ok := a.targetNames[entityID]; ok {
		return cachedName
	}
	if entity, ok := a.entityCache[entityID]; ok {
		if entity.Name != "" {
			// Check if name is numeric
			if _, err := strconv.Atoi(entity.Name); err == nil {
				return getRaceName(entity.RaceId)
			}
			return entity.Name
		}
		return getRaceName(entity.RaceId)
	}
	if player, ok := playerCache.Get(entityID); ok {
		return player.Name
	}
	return "Unknown"
}

func (a *Aggregator) processIsNowDead(p *packet.GamePacket) {
	a.mu.Lock()
	defer a.mu.Unlock()

	entityID := p.Id
	// IsNowDead only applies to players
	if a.isPlayer(entityID) {
		if !a.deadEntities[entityID] {
			a.deadEntities[entityID] = true
			a.endPresenceInterval(entityID, p.At.Unix())
		}
		a.seenDead[entityID] = true
	}
}

func (a *Aggregator) processSetFinisher(p *packet.GamePacket) {
	a.mu.Lock()
	defer a.mu.Unlock()

	entityID := p.Id
	// SetFinisher only applies to enemies
	if !a.isPlayer(entityID) {
		if !a.deadEntities[entityID] {
			a.deadEntities[entityID] = true
			a.endPresenceInterval(entityID, p.At.Unix())
		}
		a.seenDead[entityID] = true
	}
}

func (a *Aggregator) processDeadFeather(p *packet.GamePacket) {
	if len(p.Msg) < 3 ||
		p.Msg[0].Type() != packet.MessageElemTypeShort ||
		p.Msg[0].Data().(uint16) != 1 ||
		p.Msg[1].Type() != packet.MessageElemTypeInt ||
		p.Msg[1].Data().(uint32) != 0 ||
		p.Msg[2].Type() != packet.MessageElemTypeByte ||
		p.Msg[2].Data().(uint8) != 0 {
		return
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	entityID := p.Id
	if a.deadEntities[entityID] {
		delete(a.deadEntities, entityID)
	}
	a.seenDead[entityID] = false
	a.startPresenceInterval(entityID, p.At.Unix())
}

func (a *Aggregator) GetEntityTypeString(entity *packet.EntityInfo) string {
	if entity == nil {
		return "Unknown"
	}

	if strings.HasPrefix(entity.Name, "_") {
		return "NPC"
	}

	if isPlayerInfo(entity.Name, entity.RaceId, entity.OwnerId) {
		return "Player"
	}

	effectiveOwner := entity.OwnerId
	if effectiveOwner == 0 {
		effectiveOwner = entity.SecondaryOwnerId
	}

	if effectiveOwner != 0 {
		switch entity.EntityType {
		case 2:
			return "Pet"
		case 6:
			return "Dollbag"
		case 7:
			return "Mini-Gem"
		case 8:
			return "Golem"
		case 5, 11, 12:
			return "Marionette (Puppet)"
		default:
			if entity.EntityType != 0 {
				return fmt.Sprintf("Unknown Summon (Type: %d)", entity.EntityType)
			}
			isMarionette := entity.EntityType == 5 || entity.EntityType == 11 || entity.EntityType == 12
			if !isMarionette {
				if _, err := strconv.Atoi(entity.Name); err == nil {
					isMarionette = true
				}
			}
			if isMarionette {
				return "Marionette (Puppet)"
			}
			return "Pet/Summon"
		}
	}

	return "Monster/Enemy"
}
