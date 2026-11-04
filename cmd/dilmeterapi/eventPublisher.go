// eventPublisher.go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Marcentus/Midir/packet"
)

// WebSocketMessage wraps all messages sent over the socket
type WebSocketMessage struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

type entityHPLogState struct {
	LastCurrentHP float32
	LastBaseHP    float32
	LastBonusHP   float32
	LastLoggedAt  time.Time
}

type pendingDisable struct {
	packet    *packet.GamePacket
	cancelled bool
}

type eventPublisher struct {
	sync.Mutex

	// non-mutable
	ctx        context.Context
	packetCh   <-chan *packet.GamePacket
	clientMap  map[uint32]*eventClient
	sm         *SessionManager
	aggregator *Aggregator
	// mutable
	currentClientId   uint32
	playerUpdateBatch []*PlayerInfo
	batchMu           sync.Mutex

	// Async logging
	logCh       chan iEvent
	damageLogMu sync.Mutex

	// HP tracking and spam prevention
	hpLogStates            map[uint64]*entityHPLogState
	lastCombatAt           map[uint64]time.Time
	lastHealingCasterID    uint64
	preCombatBufferSeconds atomic.Int32
	preCombatUses          []*eventSkillUse
	preCombatMu            sync.Mutex

	// Condition tracking to prevent redundant log spam
	activeConditions map[uint64]map[uint32]string

	// Debouncing character condition drops (stack refreshes)
	pendingDisables map[string]*pendingDisable
	flushCh         chan string
	pendingMu       sync.Mutex
}

type eventClient struct {
	ctx context.Context
	ch  chan<- []byte
}

// Opcodes that the aggregator and/or logger needs to process
const (
	opcodeEntityAppear            = 0x520c
	opcodeEntitiesAppear          = 0x5334
	opcodeCombatAction            = 0x7926
	opcodeHealing                 = packet.OpCodeHealing
	opcodeEffectDelayed           = 0x9095
	opcodeEffect                  = 0x9093
	opcodeCharacterCondition      = 0xa028
	opcodeEntityDisappear         = 0x520d
	opcodeEntitiesDisappear       = 0x5335
	opcodeEntityUpdateCombatPower = 0x9c6d
	opcodeSkillUse                = 0x6988
	opcodeSkillStart              = 0x698c
	opcodeIsNowDead               = 0x53fc
	opcodeSetFinisher             = 0x7921
	opcodeDeadFeather             = 0x5403
	opcodePublicStatUpdate        = 0x7532
)

// This map contains skill IDs for delayed damage effects (like bleeds)
// that should contribute to total damage but NOT count as a "hit" for crit rate purposes.
var doCountDelayedSkills = map[uint16]bool{
	58100: true,
	58101: true,
	58009: true,
	58104: true,
}

func newEventPublisher(ctx context.Context, packetCh <-chan *packet.GamePacket, sm *SessionManager, isLive bool) *eventPublisher {
	v := &eventPublisher{
		ctx:        ctx,
		packetCh:   packetCh,
		clientMap:  make(map[uint32]*eventClient),
		sm:         sm,
		aggregator: NewAggregator(),

		currentClientId:   1,
		playerUpdateBatch: make([]*PlayerInfo, 0),
		logCh:             make(chan iEvent, 1000), // Buffered channel for events

		hpLogStates: make(map[uint64]*entityHPLogState),

		lastCombatAt:     make(map[uint64]time.Time),
		activeConditions: make(map[uint64]map[uint32]string),

		pendingDisables: make(map[string]*pendingDisable),
		flushCh:         make(chan string, 1000),
	}

	if cfg := loadConfig(); cfg != nil {
		v.preCombatBufferSeconds.Store(int32(cfg.PreCombatBufferSeconds))
	}
	v.aggregator.SetLive(isLive)

	go v.loop()
	go v.startLogger() // Start the logger worker

	return v
}

func (t *eventPublisher) ClearCache() {
	t.aggregator.Clear()
	t.preCombatMu.Lock()
	t.preCombatUses = nil
	t.preCombatMu.Unlock()
	t.Lock()
	t.lastHealingCasterID = 0
	t.Unlock()
	logger.Println("Clear command received, aggregator state has been reset.")
}

func (t *eventPublisher) QueuePlayerUpdate(p *PlayerInfo) {
	t.batchMu.Lock()
	defer t.batchMu.Unlock()
	t.playerUpdateBatch = append(t.playerUpdateBatch, p)
}

// REWRITTEN: The loop now processes packets for BOTH the aggregator and the event logger.
func (t *eventPublisher) loop() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	var totalPackets uint64
	var packetsThisSecond uint64
	var lastPacketAt time.Time
	var lastOp uint32
	opCounts := make(map[uint32]uint64)
	opCountsThisSecond := make(map[uint32]uint64)

	for {
		select {
		case <-t.ctx.Done():
			// Flush any remaining pending disables on shutdown
			t.pendingMu.Lock()
			for _, pd := range t.pendingDisables {
				if !pd.cancelled {
					t.aggregator.ProcessPacket(pd.packet)
					t.logPacketAsEvent(pd.packet)
				}
			}
			t.pendingDisables = make(map[string]*pendingDisable)
			t.pendingMu.Unlock()
			return
		case key := <-t.flushCh:
			t.pendingMu.Lock()
			pd, exists := t.pendingDisables[key]
			if exists && !pd.cancelled {
				p := pd.packet
				delete(t.pendingDisables, key)
				t.pendingMu.Unlock()

				// Normal processing for the expired Disable event
				t.aggregator.ProcessPacket(p)
				t.logPacketAsEvent(p)
			} else {
				t.pendingMu.Unlock()
			}
		case p := <-t.packetCh:
			totalPackets++
			packetsThisSecond++
			lastPacketAt = p.At
			lastOp = p.Op
			opCounts[p.Op]++
			opCountsThisSecond[p.Op]++

			// if p.Op != packet.OpCodeSystemError && p.Op != packet.OpCodeSystemWarning {
			// 	logger.Printf("--> Processing packet with OpCode: 0x%X", p.Op)
			// }

			// --- PATH 0: System Error Handling ---
			if p.Op == packet.OpCodeSystemError {
				logger.Printf("Received SYSTEM ERROR packet. Broadcasting to clients.")
				errMsg := string(p.RawPacket) // We stored the message here
				sysMsg := WebSocketMessage{
					Type: "system_error",
					Data: errMsg,
				}
				sysBytes, err := json.Marshal(sysMsg)
				if err != nil {
					logger.Println("Failed to marshal system error:", err)
				} else {
					t.publish(sysBytes)
				}
				continue
			}

			if p.Op == packet.OpCodeSystemWarning {
				// We no longer broadcast system warnings to the frontend to avoid UX clutter.
				// They are already logged to the backend console by the packet reader.
				continue
			}

			// --- PATH 0.5: Debounce Character Condition Disables (Stack Refreshes) ---
			if p.Op == opcodeCharacterCondition {
				if cond, err := packet.ParseCharacterConditionPacket(p); err == nil {
					key := strconv.FormatUint(cond.Id, 10) + "_" + strconv.FormatUint(uint64(cond.CCId), 10)
					if !cond.IsEnable {
						// BUFFERING DISABLE: Hold for 100ms
						t.pendingMu.Lock()
						t.pendingDisables[key] = &pendingDisable{
							packet:    p,
							cancelled: false,
						}
						t.pendingMu.Unlock()

						time.AfterFunc(100*time.Millisecond, func() {
							select {
							case t.flushCh <- key:
							case <-t.ctx.Done():
							}
						})
						continue // Do not process or log this packet yet!
					} else {
						// An Enable arrived within 100ms of a pending Disable!
						t.pendingMu.Lock()
						pd, hasPending := t.pendingDisables[key]

						// Check if metadata is actually changing
						metaChanged := true
						storedMeta := "none"
						if active, exists := t.activeConditions[cond.Id]; exists {
							if meta, found := active[cond.CCId]; found {
								storedMeta = meta
								if storedMeta == normalizeMetaData(cond.MetaData) {
									metaChanged = false
								}
							}
						}

						if hasPending {
							if !metaChanged {
								// CASE A: Same metadata (redundant refresh). Cancel/discard the Disable packet completely.
								pd.cancelled = true
								delete(t.pendingDisables, key)
								t.pendingMu.Unlock()
							} else {
								// CASE B: Different metadata (stack change). We want to log both, but align timestamps!
								disablePacket := pd.packet
								disablePacket.At = p.At // Align timestamps to prevent second-boundary gaps!
								delete(t.pendingDisables, key)
								t.pendingMu.Unlock()

								// Process the Disable packet immediately with aligned timestamp
								t.aggregator.ProcessPacket(disablePacket)
								t.logPacketAsEvent(disablePacket)
							}
						} else {
							t.pendingMu.Unlock()
						}
						// Process this Enable immediately
					}
				}
			}

			// --- PATH 1: Update the live aggregator (for the UI) ---
			t.aggregator.ProcessPacket(p)

			// --- PATH 2: Parse and log individual events (for saving files) ---
			t.logPacketAsEvent(p)

			// logger.Printf("<-- [END] Finished processing packet with OpCode: 0x%X", p.Op)

		case <-ticker.C:
			topOps := make([]map[string]interface{}, 0, len(opCountsThisSecond))
			for op, count := range opCountsThisSecond {
				topOps = append(topOps, map[string]interface{}{
					"op":    op,
					"count": count,
					"total": opCounts[op],
				})
			}

			// Calculate diagnostic statistics
			activeCondsCount := 0
			for _, conds := range t.activeConditions {
				activeCondsCount += len(conds)
			}

			t.aggregator.mu.RLock()
			trackedEntitiesCount := len(t.aggregator.entityCache)
			t.aggregator.mu.RUnlock()

			eventsCount := 0
			eventsBytes := 0
			eventBreakdown := make(map[string]int)
			t.sm.mu.RLock()
			if t.sm.currentSession != nil {
				eventsCount = len(t.sm.currentSession.events)
				// Updated incrementally at append time; do not remarshal every event each second.
				eventsBytes = t.sm.currentSession.eventBytes
				for k, v := range t.sm.currentSession.eventTypes {
					eventBreakdown[k] = v
				}
			}
			t.sm.mu.RUnlock()

			var pcapDrops, parserErrors, networkLoss, queueDrops uint32
			if globalReader != nil {
				pcapDrops, parserErrors, networkLoss, queueDrops = globalReader.GetStats()
			}

			goroutinesCount := runtime.NumGoroutine()
			var m runtime.MemStats
			runtime.ReadMemStats(&m)
			heapAllocBytes := m.Alloc

			packetStatus := WebSocketMessage{
				Type: "packet_status",
				Data: map[string]interface{}{
					"total":            totalPackets,
					"perSecond":        packetsThisSecond,
					"lastPacketAt":     lastPacketAt,
					"lastOp":           lastOp,
					"topOps":           topOps,
					"activeConditions": activeCondsCount,
					"trackedEntities":  trackedEntitiesCount,
					"bufferEvents":     eventsCount,
					"bufferBytes":      eventsBytes,
					"goroutines":       goroutinesCount,
					"heapAlloc":        heapAllocBytes,
					"eventBreakdown":   eventBreakdown,
					"pcapDrops":        pcapDrops,
					"parserErrors":     parserErrors,
					"networkLoss":      networkLoss,
					"queueDrops":       queueDrops,
				},
			}
			packetStatusBytes, err := json.Marshal(packetStatus)
			if err != nil {
				logger.Println("Failed to marshal packet status:", err)
			} else {
				t.publish(packetStatusBytes)
			}
			packetsThisSecond = 0
			opCountsThisSecond = make(map[uint32]uint64)

			// 1. Send Summary
			summary := t.aggregator.GetSummary()
			summaryMsg := WebSocketMessage{
				Type: "summary",
				Data: summary,
			}
			summaryBytes, err := json.Marshal(summaryMsg)
			if err != nil {
				logger.Println("Failed to marshal summary:", err)
			} else {
				t.publish(summaryBytes)
			}

			// 2. Send Player Updates Batch
			t.batchMu.Lock()
			if len(t.playerUpdateBatch) > 0 {
				batchMsg := WebSocketMessage{
					Type: "player_update_batch",
					Data: t.playerUpdateBatch,
				}
				batchBytes, err := json.Marshal(batchMsg)
				if err != nil {
					logger.Println("Failed to marshal player batch:", err)
				} else {
					t.publish(batchBytes)
				}
				// Clear the batch
				t.playerUpdateBatch = make([]*PlayerInfo, 0)
			}
			t.batchMu.Unlock()
		}
	}
}

// Worker to handle logging events to disk
func (t *eventPublisher) startLogger() {
	for {
		select {
		case <-t.ctx.Done():
			// Drain the channel before exiting?
			// For now, we just exit to be responsive to shutdown.
			// Ideally, we might want to flush remaining events.
			close(t.logCh)
			for e := range t.logCh {
				if err := t.sm.WriteEventToLog(e); err != nil {
					logger.Println("Failed to write event to log (shutdown):", err)
				}
			}
			return
		case e := <-t.logCh:
			if err := t.sm.WriteEventToLog(e); err != nil {
				logger.Println("Failed to write event to log:", err)
			}
		}
	}
}

// Handles parsing a packet and writing it to the session log.
func (t *eventPublisher) logPacketAsEvent(p *packet.GamePacket) {
	var err error
	var events []iEvent

	switch p.Op {

	case opcodeCombatAction:
		var pack *packet.CombatActionPackPacket
		pack, err = packet.ParseCombatActionPackPacket(p)
		if err != nil {
			break
		}

		// CORRECTED LOGIC: Find the attacker and the single, correct skill ID first.
		var attackerId uint64
		var attackSkillId uint16
		for _, sub := range pack.SubPackets {
			if sub.Type&packet.CombatActionTypeAttacker != 0 {
				attackerId = sub.EntityId
				attackSkillId = sub.SkillId // Capture the skill ID from the attacker's packet
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
			break
		}

		// Record combat timestamps
		t.lastCombatAt[attackerId] = p.At
		for _, sub := range pack.SubPackets {
			if sub.Hit != nil {
				t.lastCombatAt[sub.EntityId] = p.At
			}
		}

		// Now, create damage events for each hit using the correct skill ID.
		for _, sub := range pack.SubPackets {
			if sub.Hit != nil && (sub.Hit.Damage != 0 || sub.Hit.ManaDamage > 0) {
				isCrit := (sub.Hit.Options & packet.CombatActionHitOptionsCritical) != 0
				e := &eventDamage{
					eventBase: eventBase{
						EventId: eventIdDamage,
						At:      p.At.Unix(),
						Id:      strconv.FormatUint(attackerId, 10),
					},
					AtMs:       p.At.UnixMilli(),
					TargetId:   strconv.FormatUint(sub.EntityId, 10),
					SkillId:    attackSkillId, // Use the correct, captured skill ID here
					Damage:     sub.Hit.Damage,
					ManaDamage: float32(sub.Hit.ManaDamage),
					IsCritical: isCrit,
					IsDelayed:  false,
				}
				events = append(events, e)
			}
		}

	case opcodeEffect:
		// Opcode 0x9093 is shared by generic effects and the dedicated healing
		// packet variants. Try the specific healing structure first; ordinary
		// effect packets fall through to the existing parser below.
		if heal, parseErr := packet.ParseHealingPacket(p); parseErr == nil {
			switch heal.Type {
			case 0x19, 0x28:
				t.lastHealingCasterID = heal.EntityID
			case 0x0A:
				// Mirror the live-healing validity checks before persisting an event.
				// Non-player targets (e.g. Shard of Solace spawn HP) must not
				// reappear as healing when the saved session is rebuilt.
				if recipient, ok := playerCache.Get(heal.EntityID); !ok || recipient == nil || recipient.Name == "" {
					break
				}
				maximumHeal := float32(100000)
				if _, _, _, maxHP, exists := t.aggregator.GetEntityHP(heal.EntityID); exists && maxHP > 0 && maxHP < 100000000 {
					maximumHeal = 2 * maxHP
				}
				if float32(heal.Amount) > maximumHeal {
					break
				}
				if t.lastHealingCasterID != 0 && heal.Amount > 0 {
					e := &eventDamage{
						eventBase: eventBase{EventId: eventIdDamage, At: p.At.Unix(), Id: strconv.FormatUint(t.lastHealingCasterID, 10)},
						TargetId:  strconv.FormatUint(heal.EntityID, 10), AtMs: p.At.UnixMilli(), SkillId: 0, Damage: -float32(heal.Amount), ManaDamage: 0,
					}
					events = append(events, e)
				}
			}
			break
		}
		// Check against the structure of: Int, Byte, Int, Int, Long, Short, Byte
		if len(p.Msg) < 7 ||
			p.Msg[0].Type() != packet.MessageElemTypeInt ||
			p.Msg[0].Data().(uint32) != 352 || // Type 352 check
			p.Msg[1].Type() != packet.MessageElemTypeByte ||
			p.Msg[2].Type() != packet.MessageElemTypeInt ||
			p.Msg[3].Type() != packet.MessageElemTypeInt ||
			p.Msg[4].Type() != packet.MessageElemTypeLong ||
			p.Msg[5].Type() != packet.MessageElemTypeShort ||
			p.Msg[6].Type() != packet.MessageElemTypeByte {
			break
		}

		damage := float32(p.Msg[2].Data().(uint32))
		attackerId := p.Msg[4].Data().(uint64)
		skillId := p.Msg[5].Data().(uint16)
		targetId := p.Id

		t.lastCombatAt[attackerId] = p.At
		t.lastCombatAt[targetId] = p.At

		e := &eventDamage{
			eventBase: eventBase{
				EventId: eventIdDamage,
				At:      p.At.Unix(),
				Id:      strconv.FormatUint(attackerId, 10),
			},
			AtMs:       p.At.UnixMilli(),
			TargetId:   strconv.FormatUint(targetId, 10),
			SkillId:    skillId,
			Damage:     damage,
			IsCritical: false,
			IsDelayed:  true,
		}
		events = append(events, e)

	case opcodeEffectDelayed:
		// NEW: Check against the full, correct packet structure.
		if len(p.Msg) < 7 ||
			p.Msg[1].Type() != packet.MessageElemTypeInt ||
			p.Msg[1].Data().(uint32) != 319 || // Check for the specific sub-ID (this changes value some updates)
			p.Msg[2].Type() != packet.MessageElemTypeInt ||
			p.Msg[5].Type() != packet.MessageElemTypeLong ||
			p.Msg[6].Type() != packet.MessageElemTypeShort {
			// This is not an error, just a different packet type we can safely ignore.
			break
		}

		// CORRECTED: Damage is a uint32, which we cast to float32 for the event struct.
		damage := float32(p.Msg[2].Data().(uint32))
		attackerId := p.Msg[5].Data().(uint64)
		skillId := p.Msg[6].Data().(uint16)
		targetId := p.Id

		t.lastCombatAt[attackerId] = p.At
		t.lastCombatAt[targetId] = p.At

		e := &eventDamage{
			eventBase: eventBase{
				EventId: eventIdDamage,
				At:      p.At.Unix(),
				Id:      strconv.FormatUint(attackerId, 10),
			},
			AtMs:       p.At.UnixMilli(),
			TargetId:   strconv.FormatUint(targetId, 10),
			SkillId:    skillId,
			Damage:     damage,
			IsCritical: false,
			IsDelayed:  true,
		}
		events = append(events, e)

	case opcodeEntityAppear:
		var entity *packet.EntityInfo
		entity, err = packet.ParseEntityAppearPacket(p.Msg)
		if err == nil && entity != nil {
			events = append(events, newEventFromEntity(entity, p.At))
			t.trackEntityConditions(entity.Id, entity.CharacterConditionMap)
		}

	case opcodeEntitiesAppear:
		var entities []*packet.EntityInfo
		entities, err = packet.ParseEntitiesAppearPacket(p)
		if err == nil {
			for _, entity := range entities {
				events = append(events, newEventFromEntity(entity, p.At))
				t.trackEntityConditions(entity.Id, entity.CharacterConditionMap)
			}
		}

	case opcodeEntityDisappear:
		// UPDATED: Use new parser for single disappear
		dID, err := packet.ParseEntityDisappearPacket(p)
		if err == nil {
			events = append(events, &eventEntityDisappear{
				eventBase: eventBase{
					EventId: eventIdEntityDisappear,
					At:      p.At.Unix(),
					Id:      strconv.FormatUint(dID, 10),
				},
			})
			t.cleanupEntityState(dID)
		}

	case opcodeEntitiesDisappear:
		// NEW: Handle batch disappear
		dIDs, err := packet.ParseEntitiesDisappearPacket(p)
		if err == nil {
			for _, dID := range dIDs {
				events = append(events, &eventEntityDisappear{
					eventBase: eventBase{
						EventId: eventIdEntityDisappear,
						At:      p.At.Unix(),
						Id:      strconv.FormatUint(dID, 10),
					},
				})
				t.cleanupEntityState(dID)
			}
		}

	case opcodeCharacterCondition:
		var cond *packet.CharacterConditionPacket
		cond, err = packet.ParseCharacterConditionPacket(p)
		if err == nil {
			if cond.IsEnable {
				alreadyActive := false
				if active, exists := t.activeConditions[cond.Id]; exists {
					if storedMeta, found := active[cond.CCId]; found {
						if storedMeta == normalizeMetaData(cond.MetaData) {
							alreadyActive = true
						}
					}
				}

				if !alreadyActive {
					if t.activeConditions[cond.Id] == nil {
						t.activeConditions[cond.Id] = make(map[uint32]string)
					}
					t.activeConditions[cond.Id][cond.CCId] = normalizeMetaData(cond.MetaData)

					events = append(events, &eventCharacterConditionEnable{
						eventBase: eventBase{
							EventId: eventIdCharacterConditionEnable,
							At:      p.At.Unix(),
							Id:      strconv.FormatUint(cond.Id, 10),
						},
						CCId:       cond.CCId,
						DisableAt:  cond.DisableAt,
						MetaData:   cond.MetaData,
						AttackerId: strconv.FormatUint(cond.AttackerId, 10),
					})
				}
			} else {
				if active, exists := t.activeConditions[cond.Id]; exists {
					delete(active, cond.CCId)
					if len(active) == 0 {
						delete(t.activeConditions, cond.Id)
					}
				}

				events = append(events, &eventCharacterConditionDisable{
					eventBase: eventBase{
						EventId: eventIdCharacterConditionDisable,
						At:      p.At.Unix(),
						Id:      strconv.FormatUint(cond.Id, 10),
					},
					CCId: cond.CCId,
				})
			}
		}

	case opcodeIsNowDead:
		// IsNowDead only applies to players
		if t.aggregator.IsPlayerSafe(p.Id) {
			events = append(events, &eventEntityDeath{
				eventBase: eventBase{
					EventId: eventIdEntityDeath,
					At:      p.At.Unix(),
					Id:      strconv.FormatUint(p.Id, 10),
				},
			})
		}

	case opcodeSetFinisher:
		// SetFinisher only applies to enemies
		if !t.aggregator.IsPlayerSafe(p.Id) {
			events = append(events, &eventEntityDeath{
				eventBase: eventBase{
					EventId: eventIdEntityDeath,
					At:      p.At.Unix(),
					Id:      strconv.FormatUint(p.Id, 10),
				},
			})
		}

	case opcodeDeadFeather:
		if len(p.Msg) >= 3 &&
			p.Msg[0].Type() == packet.MessageElemTypeShort &&
			p.Msg[0].Data().(uint16) == 1 &&
			p.Msg[1].Type() == packet.MessageElemTypeInt &&
			p.Msg[1].Data().(uint32) == 0 &&
			p.Msg[2].Type() == packet.MessageElemTypeByte &&
			p.Msg[2].Data().(uint8) == 0 {

			events = append(events, &eventEntityRevive{
				eventBase: eventBase{
					EventId: eventIdEntityRevive,
					At:      p.At.Unix(),
					Id:      strconv.FormatUint(p.Id, 10),
				},
			})
		}

	case opcodeSkillUse, opcodeSkillStart:
		if len(p.Msg) < 1 {
			break
		}
		if p.Msg[0].Type() != packet.MessageElemTypeShort {
			break
		}

		var targetID uint64
		if len(p.Msg) > 1 && p.Msg[1].Type() == packet.MessageElemTypeLong {
			rawTargetID := p.Msg[1].Data().(uint64)
			casterID := p.Id
			if rawTargetID == casterID {
				targetID = 0
			} else {
				targetID = rawTargetID
			}
		}

		t.lastCombatAt[p.Id] = p.At
		if targetID != 0 {
			t.lastCombatAt[targetID] = p.At
		}
		// Record ONLY observed uses, and only from verified player entities.
		// Skill Start still supports combat timing but does not generate events.
		// The final CombatRecordingActive gate below avoids town/idle capture.
		if p.Op == opcodeSkillUse && t.aggregator.IsPlayerSafe(p.Id) {
			use := &eventSkillUse{
				eventBase: eventBase{EventId: eventIdSkillUse, At: p.At.Unix(), Id: strconv.FormatUint(p.Id, 10)},
				AtMs:      p.At.UnixMilli(), SkillId: p.Msg[0].Data().(uint16),
			}
			if targetID != 0 {
				use.TargetId = strconv.FormatUint(targetID, 10)
			}
			events = append(events, use)
		}

	case opcodePublicStatUpdate:
		var statPack *packet.PublicStatUpdatePacket
		statPack, err = packet.ParsePublicStatUpdatePacket(p)
		if err != nil {
			break
		}

		// Retrieve active HP in a thread-safe way from the aggregator
		curHP, baseHP, additionalHP, maxHP, exists := t.aggregator.GetEntityHP(statPack.EntityId)
		if !exists {
			// Skip logging if the entity is not currently tracked in the active entityCache
			break
		}

		// Filter rule: Only track true player characters or engaged combat targets.
		// Excludes unengaged town NPCs, environmental props, and passive objects.
		_, isPlayer := playerCache.Get(statPack.EntityId)
		if !isPlayer && !t.aggregator.IsCombatTarget(statPack.EntityId) {
			break
		}

		// Look up or create the last logged HP state
		state, found := t.hpLogStates[statPack.EntityId]
		if !found {
			state = &entityHPLogState{}
			t.hpLogStates[statPack.EntityId] = state
		}

		// If nothing changed, we do not log
		if curHP == state.LastCurrentHP && baseHP == state.LastBaseHP && additionalHP == state.LastBonusHP && !state.LastLoggedAt.IsZero() {
			break
		}

		// Apply HP change threshold check:
		// Skip if this is not the first log, HP is not 0, Max HP didn't change, and the delta is < 0.1% of Max HP
		if !state.LastLoggedAt.IsZero() && curHP > 0 && baseHP == state.LastBaseHP && additionalHP == state.LastBonusHP {
			hpDiff := curHP - state.LastCurrentHP
			if hpDiff < 0 {
				hpDiff = -hpDiff
			}
			threshold := 0.001 * maxHP
			if hpDiff < threshold {
				break
			}
		}

		// Apply the Hybrid Throttling Scheme:
		// 1. Log immediately if Current HP is 0 (death milestone)
		// 2. Log immediately if Max HP (BaseHP or AdditionalHP) changed
		// 3. Otherwise, check combat context:
		//    - If recently active in combat (within 5 seconds), log at most once per 3.0s.
		//    - If idle, log at most once per 10.0s.
		shouldLog := false
		if curHP == 0 {
			shouldLog = true
		} else if baseHP != state.LastBaseHP || additionalHP != state.LastBonusHP {
			shouldLog = true
		} else {
			lastCombat := t.lastCombatAt[statPack.EntityId]
			inCombat := !lastCombat.IsZero() && p.At.Sub(lastCombat) <= 5*time.Second

			throttleDuration := 10 * time.Second
			if inCombat {
				throttleDuration = 3 * time.Second
			}

			if state.LastLoggedAt.IsZero() || p.At.Sub(state.LastLoggedAt) >= throttleDuration {
				shouldLog = true
			}
		}

		if shouldLog {
			e := &eventEntityHPUpdate{
				eventBase: eventBase{
					EventId: eventIdEntityHPUpdate,
					At:      p.At.Unix(),
					Id:      strconv.FormatUint(statPack.EntityId, 10),
				},
				CurrentHP:    curHP,
				BaseHP:       baseHP,
				AdditionalHP: additionalHP,
				MaxHP:        maxHP,
			}
			events = append(events, e)

			// Update the logged state
			state.LastCurrentHP = curHP
			state.LastBaseHP = baseHP
			state.LastBonusHP = additionalHP
			state.LastLoggedAt = p.At
		}
	}

	if err != nil {
		logger.Printf("Packet parsing error for logging (Op: 0x%X): %v", p.Op, err)
	}

	// A few structural fields are captured in combat only. No skill matching,
	// temporal grouping, name lookup, or per-skill inference happens live.
	if (p.Op == opcodeCombatAction || p.Op == opcodeEffect || p.Op == opcodeEffectDelayed) && t.aggregator.CombatRecordingActive() {
		if p.Op == opcodeCombatAction {
			if pack, parseErr := packet.ParseCombatActionPackPacket(p); parseErr == nil {
				for _, sub := range pack.SubPackets {
					if sub.SkillId == 0 || sub.Type&packet.CombatActionTypeAttacker == 0 || !t.aggregator.IsPlayerSafe(sub.EntityId) {
						continue
					}
					events = append(events, &eventSkillObservation{
						eventBase: eventBase{EventId: eventIdSkillObservation, At: p.At.Unix(), Id: strconv.FormatUint(sub.EntityId, 10)},
						AtMs:      p.At.UnixMilli(), Opcode: p.Op, SkillId: sub.SkillId, ActionType: uint8(sub.Type), ActorVerified: true,
					})
				}
			}
		} else if p.Op == opcodeEffect || p.Op == opcodeEffectDelayed {
			// Use only player-attributed messages; other sources/targets are not guessed.
			// Every field is bounded, with binary payloads deliberately excluded.
			if t.aggregator.IsPlayerSafe(p.Id) && len(p.Msg) > 0 {
				fields := make([]string, 0, 12)
				fieldTypes := make([]uint8, 0, 12)
				for _, element := range p.Msg {
					if len(fields) >= 12 {
						break
					}
					fieldTypes = append(fieldTypes, uint8(element.Type()))
					switch element.Type() {
					case packet.MessageElemTypeByte, packet.MessageElemTypeShort, packet.MessageElemTypeInt, packet.MessageElemTypeLong, packet.MessageElemTypeFloat:
						fields = append(fields, fmt.Sprint(element.Data()))
					case packet.MessageElemTypeString:
						v := fmt.Sprint(element.Data())
						if len(v) > 160 {
							v = v[:160]
						}
						fields = append(fields, v)
					default:
						fields = append(fields, "")
					}
				}
				if len(fields) > 0 {
					var subtype uint32
					if p.Msg[0].Type() == packet.MessageElemTypeInt {
						subtype = p.Msg[0].Data().(uint32)
					}
					events = append(events, &eventSkillObservation{
						eventBase: eventBase{EventId: eventIdSkillObservation, At: p.At.Unix(), Id: strconv.FormatUint(p.Id, 10)},
						AtMs:      p.At.UnixMilli(), Opcode: p.Op, Subtype: subtype, Fields: fields, FieldTypes: fieldTypes,
					})
				}
			}
		}
	}

	// Bound the saved event log to active combat targets. Decide at packet time,
	// not when the asynchronous logger drains its queue after the boss dies.
	if !t.aggregator.CombatRecordingActive() {
		// Keep only a small, bounded cache of observed player uses. The
		// regular logger, encounter gate and encounter shutdown are unchanged.
		if t.preCombatBufferSeconds.Load() > 0 && p.Op == opcodeSkillUse {
			t.preCombatMu.Lock()
			cutoff := p.At.Add(-time.Duration(t.preCombatBufferSeconds.Load()) * time.Second).UnixMilli()
			kept := t.preCombatUses[:0]
			for _, use := range t.preCombatUses {
				if use.AtMs >= cutoff {
					kept = append(kept, use)
				}
			}
			t.preCombatUses = kept
			for _, e := range events {
				if use, ok := e.(*eventSkillUse); ok {
					t.preCombatUses = append(t.preCombatUses, use)
				}
			}
			if len(t.preCombatUses) > 512 {
				t.preCombatUses = t.preCombatUses[len(t.preCombatUses)-512:]
			}
			t.preCombatMu.Unlock()
		}
		return
	}
	// Flush buffered actions only after the original combat gate opens.
	// A buffered event is persisted at most once; never record town activity.
	t.preCombatMu.Lock()
	if len(t.preCombatUses) > 0 {
		cutoff := p.At.Add(-time.Duration(t.preCombatBufferSeconds.Load()) * time.Second).UnixMilli()
		for _, use := range t.preCombatUses {
			if use.AtMs < cutoff {
				continue
			}
			select {
			case t.logCh <- use:
				t.aggregator.RecordSkillActivation(parseUint64(use.Id), SkillActivation{AtMs: use.AtMs, SkillID: use.SkillId, TargetID: use.TargetId})
			default:
				logger.Println("Log channel full, dropping buffered skill use!")
			}
		}
		t.preCombatUses = nil
	}
	t.preCombatMu.Unlock()
	for _, e := range events {
		select {
		case t.logCh <- e:
			// Match the durable combat-recording gate. A town/idle packet
			// must not leak into the live report if it was not saved.
			if use, ok := e.(*eventSkillUse); ok && t.aggregator != nil {
				t.aggregator.RecordSkillActivation(p.Id, SkillActivation{
					AtMs: use.AtMs, SkillID: use.SkillId, TargetID: use.TargetId,
				})
			}
		default:
			logger.Println("Log channel full, dropping event!")
		}
	}
}

// NEW HELPER: Converts an EntityInfo packet to an eventEntityAppear struct.
func newEventFromEntity(entity *packet.EntityInfo, at time.Time) iEvent {
	cond := make([]EventConditionData, 0, len(entity.CharacterConditionMap))
	for _, c := range entity.CharacterConditionMap {
		cond = append(cond, EventConditionData{
			CCId:       c.CCId,
			DisableAt:  c.DisableAt,
			MetaData:   c.MetaData,
			AttackerId: strconv.FormatUint(c.AttackerId, 10),
		})
	}

	return &eventEntityAppear{
		eventBase: eventBase{
			EventId: eventIdEntityAppear,
			At:      at.Unix(),
			Id:      strconv.FormatUint(entity.Id, 10),
		},
		Name:             entity.Name,
		RaceId:           entity.RaceId,
		OwnerId:          strconv.FormatUint(entity.OwnerId, 10),
		EntityType:       entity.EntityType,
		SecondaryOwnerId: strconv.FormatUint(entity.SecondaryOwnerId, 10),
		CurrentHP:        entity.CurrentHP,
		MaxHP:            entity.MaxHP,
		Conditions:       cond,
	}
}

func (t *eventPublisher) Broadcast(msgType string, data interface{}) {
	msg := WebSocketMessage{
		Type: msgType,
		Data: data,
	}
	bytes, err := json.Marshal(msg)
	if err == nil {
		t.publish(bytes)
	}
}

func (t *eventPublisher) publish(payload []byte) {
	t.Lock()
	defer t.Unlock()

	for k, c := range t.clientMap {
		select {
		case <-c.ctx.Done():
			delete(t.clientMap, k)
			continue
		default:
		}
		select {
		case c.ch <- payload:
		default:
			delete(t.clientMap, k)
			logger.Println("queue full... force close socket", k)
			continue
		}
	}
}

func (t *eventPublisher) addClient(ctx context.Context, ch chan<- []byte) uint32 {
	t.Lock()
	defer t.Unlock()

	t.currentClientId++
	clientId := t.currentClientId

	t.clientMap[clientId] = &eventClient{
		ctx: ctx,
		ch:  ch,
	}

	logger.Println("Client connected:", clientId)
	return clientId
}

// Helper to track conditions from an appearing entity
func (t *eventPublisher) trackEntityConditions(entityID uint64, condMap map[uint32]*packet.EntityCharacterCondition) {
	if len(condMap) == 0 {
		return
	}
	if t.activeConditions[entityID] == nil {
		t.activeConditions[entityID] = make(map[uint32]string)
	}
	for _, cond := range condMap {
		t.activeConditions[entityID][cond.CCId] = normalizeMetaData(cond.MetaData)
	}
}

// Helper to clean up all tracked state for a disappeared entity (to prevent memory leaks)
func (t *eventPublisher) cleanupEntityState(entityID uint64) {
	delete(t.activeConditions, entityID)
	delete(t.hpLogStates, entityID)
	delete(t.lastCombatAt, entityID)
}

func getEventName(id eventId) string {
	switch id {
	case eventIdEntityHPUpdate:
		return "HP Update"
	case eventIdEntityAppear:
		return "Entity Appear"
	case eventIdEntityDisappear:
		return "Entity Disappear"
	case eventIdDamage:
		return "Damage"
	case eventIdSkillObservation:
		return "Skill Observation"
	case eventIdSkillUse:
		return "Skill Use"
	case eventIdCharacterConditionEnable:
		return "Condition Enable"
	case eventIdCharacterConditionDisable:
		return "Condition Disable"
	case eventIdEntityDeath:
		return "Entity Death"
	case eventIdEntityRevive:
		return "Entity Revive"
	case eventIdSessionSummary:
		return "Session Summary"
	default:
		return "Unknown Event"
	}
}
