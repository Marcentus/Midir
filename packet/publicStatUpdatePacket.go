package packet

import (
	"fmt"
)

// PublicStatUpdatePacket represents a parsed public stat update.
type PublicStatUpdatePacket struct {
	EntityId uint64
	Stats    map[uint32]float32
}

// ParsePublicStatUpdatePacket parses opcode 0x7532 packets.
// Layout:
// - Message element 0 is a Byte whose value must be 4.
// - Message element 1 is an Int (or other integer representation) representing the count of pairs N.
// - Followed by N pairs of [Stat ID, Stat Value].
//   - Stat ID is an Int, Short, or Byte.
//   - Stat Value is a Float.
func ParsePublicStatUpdatePacket(p *GamePacket) (*PublicStatUpdatePacket, error) {
	if p == nil {
		return nil, fmt.Errorf("packet is nil")
	}
	if len(p.Msg) < 2 {
		return nil, fmt.Errorf("message too short: length %d", len(p.Msg))
	}

	if p.Msg[0].Type() != MessageElemTypeByte {
		return nil, fmt.Errorf("expected byte at index 0, got %v", p.Msg[0].Type())
	}
	subType := p.Msg[0].Data().(uint8)
	if subType != 4 {
		return nil, fmt.Errorf("expected subType 4, got %v", subType)
	}

	var count int
	switch p.Msg[1].Type() {
	case MessageElemTypeInt:
		count = int(p.Msg[1].Data().(uint32))
	case MessageElemTypeShort:
		count = int(p.Msg[1].Data().(uint16))
	case MessageElemTypeByte:
		count = int(p.Msg[1].Data().(uint8))
	default:
		return nil, fmt.Errorf("expected integer count at index 1, got %v", p.Msg[1].Type())
	}

	if len(p.Msg) < 2+2*count {
		return nil, fmt.Errorf("message too short for %d pairs: length is %d", count, len(p.Msg))
	}

	stats := make(map[uint32]float32)
	for i := 0; i < count; i++ {
		idx := 2 + 2*i
		statIdElem := p.Msg[idx]
		statValElem := p.Msg[idx+1]

		var statId uint32
		switch statIdElem.Type() {
		case MessageElemTypeInt:
			statId = statIdElem.Data().(uint32)
		case MessageElemTypeShort:
			statId = uint32(statIdElem.Data().(uint16))
		case MessageElemTypeByte:
			statId = uint32(statIdElem.Data().(uint8))
		default:
			return nil, fmt.Errorf("expected integer stat ID at index %d, got %v", idx, statIdElem.Type())
		}

		if statValElem.Type() != MessageElemTypeFloat {
			// Skip elements that aren't floats (or we can handle it gracefully)
			continue
		}
		statVal := statValElem.Data().(float32)
		stats[statId] = statVal
	}

	return &PublicStatUpdatePacket{
		EntityId: p.Id,
		Stats:    stats,
	}, nil
}
