// eventPublisher.go
package main

import (
	"context"
	"encoding/json"
	"strconv"
	"sync"
	"time"

	"github.com/Marcentus/Midir/packet"
)

// WebSocketMessage wraps all messages sent over the socket
type WebSocketMessage struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
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
	logCh chan iEvent
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
	opcodeEffectDelayed           = 0x9095
	opcodeCharacterCondition      = 0xa028
	opcodeEntityDisappear         = 0x520d
	opcodeEntitiesDisappear       = 0x5335
	opcodeEntityUpdateCombatPower = 0x9c6d
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
	}

	v.aggregator.SetLive(isLive)

	go v.loop()
	go v.startLogger() // Start the logger worker

	return v
}

func (t *eventPublisher) ClearCache() {
	t.aggregator.Clear()
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

	for {
		select {
		case <-t.ctx.Done():
			return
		case p := <-t.packetCh:
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
				logger.Printf("Received SYSTEM WARNING packet. Broadcasting to clients.")
				warnMsg := string(p.RawPacket)
				sysMsg := WebSocketMessage{
					Type: "system_warning",
					Data: warnMsg,
				}
				sysBytes, err := json.Marshal(sysMsg)
				if err != nil {
					logger.Println("Failed to marshal system warning:", err)
				} else {
					t.publish(sysBytes)
				}
				continue
			}


			// --- PATH 1: Update the live aggregator (for the UI) ---
			t.aggregator.ProcessPacket(p)

			// --- PATH 2: Parse and log individual events (for saving files) ---
			t.logPacketAsEvent(p)

			// logger.Printf("<-- [END] Finished processing packet with OpCode: 0x%X", p.Op)

		case <-ticker.C:
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

		// Now, create damage events for each hit using the correct skill ID.
		for _, sub := range pack.SubPackets {
			if sub.Hit != nil && (sub.Hit.Damage > 0 || sub.Hit.ManaDamage > 0) {
				isCrit := (sub.Hit.Options & packet.CombatActionHitOptionsCritical) != 0
				events = append(events, &eventDamage{
					eventBase: eventBase{
						EventId: eventIdDamage,
						At:      p.At.Unix(),
						Id:      strconv.FormatUint(attackerId, 10),
					},
					TargetId:   strconv.FormatUint(sub.EntityId, 10),
					SkillId:    attackSkillId, // Use the correct, captured skill ID here
					Damage:     sub.Hit.Damage,
					ManaDamage: float32(sub.Hit.ManaDamage),
					IsCritical: isCrit,
					IsDelayed:  false,
				})
			}
		}

	case opcodeEffectDelayed:
		// NEW: Check against the full, correct packet structure.
		if len(p.Msg) < 7 ||
			p.Msg[1].Type() != packet.MessageElemTypeInt ||
			p.Msg[1].Data().(uint32) != 317 || // Check for the specific sub-ID 317
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
		events = append(events, &eventDamage{
			eventBase: eventBase{
				EventId: eventIdDamage,
				At:      p.At.Unix(),
				Id:      strconv.FormatUint(attackerId, 10),
			},
			TargetId:   strconv.FormatUint(targetId, 10),
			SkillId:    skillId,
			Damage:     damage,
			IsCritical: false,
			IsDelayed:  true,
		})

	case opcodeEntityAppear:
		var entity *packet.EntityInfo
		entity, err = packet.ParseEntityAppearPacket(p.Msg)
		if err == nil && entity != nil {
			events = append(events, newEventFromEntity(entity, p.At))
		}

	case opcodeEntitiesAppear:
		var entities []*packet.EntityInfo
		entities, err = packet.ParseEntitiesAppearPacket(p)
		if err == nil {
			for _, entity := range entities {
				events = append(events, newEventFromEntity(entity, p.At))
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
			}
		}

	case opcodeCharacterCondition:
		var cond *packet.CharacterConditionPacket
		cond, err = packet.ParseCharacterConditionPacket(p)
		if err == nil {
			if cond.IsEnable {
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
			} else {
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
	}

	if err != nil {
		logger.Printf("Packet parsing error for logging (Op: 0x%X): %v", p.Op, err)
	}

	// CHANGED: Send to channel instead of writing directly
	for _, e := range events {
		select {
		case t.logCh <- e:
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
		Name:         entity.Name,
		RaceId:       entity.RaceId,
		OwnerId:      strconv.FormatUint(entity.OwnerId, 10),
		CurrentHP:    entity.CurrentHP,
		MaxHP:        entity.MaxHP,
		Conditions:   cond,
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

