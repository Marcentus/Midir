package packet

import (
	"fmt"
)

// EntityDisappearPacket is usually just the ID of the entity that disappeared.
// However, the packet structure can vary. Based on user feedback:
// EntityDisappear (Single): First element is Long (Entity ID).
func ParseEntityDisappearPacket(p *GamePacket) (uint64, error) {
	if len(p.Msg) < 1 {
		return 0, fmt.Errorf("entity disappear packet too short")
	}

	// Check if first element is Long
	if p.Msg[0].Type() == MessageElemTypeLong {
		return p.Msg[0].Data().(uint64), nil
	}

	return 0, fmt.Errorf("entity disappear packet has unexpected type")
}

// EntitiesDisappearPacket (Batch):
// OpCode: 0x5335
// Structure:
// Short (Count)
// [Short, Long (EntityID), Byte] repeated Count times
func ParseEntitiesDisappearPacket(p *GamePacket) ([]uint64, error) {
	if len(p.Msg) < 1 {
		return nil, fmt.Errorf("entities disappear packet too short")
	}

	// Check if first element is Short (Count)
	if p.Msg[0].Type() != MessageElemTypeShort {
		return nil, fmt.Errorf("entities disappear packet expected short count")
	}

	count := int(p.Msg[0].Data().(uint16))
	msg := p.Msg[1:]
	ids := make([]uint64, 0, count)

	for i := 0; i < count; i++ {
		if len(msg) < 1 {
			break
		}

		// The first element is a tag (Short)
		if msg[0].Type() != MessageElemTypeShort {
			// Unexpected, just break or skip
			break
		}
		tag := msg[0].Data().(uint16)

		// Case 1: Tag 16 -> [Short(16), Long(ID), Byte] -> 3 elements
		if tag == 16 {
			if len(msg) < 3 {
				break
			}
			if msg[1].Type() == MessageElemTypeLong {
				ids = append(ids, msg[1].Data().(uint64))
			}
			msg = msg[3:]
		} else {
			// Case 2: Other tags (e.g. 161) -> [Short(tag), Long(ID)] -> 2 elements
			if len(msg) < 2 {
				break
			}
			if msg[1].Type() == MessageElemTypeLong {
				ids = append(ids, msg[1].Data().(uint64))
			}
			msg = msg[2:]
		}
	}

	return ids, nil
}
