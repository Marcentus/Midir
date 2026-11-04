// Experimental PvE healing packet decoding; keep separate from hostile-damage parsing.
// Packet subtypes/caster linkage must be validated against live captures.
package packet

import (
	"encoding/binary"
	"fmt"
)

// HealingPacket is decoded from Mabinogi's dedicated 0x90B3 healing opcode.
// Unlike CombatActionPack, this packet reports healing as a recipient/amount
// update and must not pass through hostile-target damage filtering.
type HealingPacket struct {
	Type     uint8
	EntityID uint64 // recipient for type 0x0A; caster for type 0x19/0x28
	Amount   uint32
}

const OpCodeHealing = 0x9093

// ParseHealingPacket decodes the stable fields used by the known healing packet
// variants. RawPacket is used because the healing opcode has a compact body that
// is not shaped like the typed combat-action message.
func ParseHealingPacket(p *GamePacket) (*HealingPacket, error) {
	if p == nil || p.Op != OpCodeHealing {
		return nil, fmt.Errorf("not a healing packet")
	}
	raw := p.RawPacket
	// Mabinogi packet framing: sign(1), length(4), flag(1), opcode(4),
	// entity ID(8), then healing subtype. The amount for subtype 0x0A is
	// at byte offset 27 in the packet (as observed by the reference parser).
	if len(raw) < 19 {
		return nil, fmt.Errorf("healing packet too short: %d", len(raw))
	}
	if binary.BigEndian.Uint32(raw[6:10]) != OpCodeHealing {
		return nil, fmt.Errorf("raw packet opcode does not match healing")
	}
	v := &HealingPacket{EntityID: binary.BigEndian.Uint64(raw[10:18]), Type: raw[18]}
	switch v.Type {
	case 0x0A:
		if len(raw) < 31 {
			return nil, fmt.Errorf("single-heal packet too short: %d", len(raw))
		}
		v.Amount = binary.BigEndian.Uint32(raw[27:31])
	case 0x19, 0x28:
		// These variants identify/update the caster. Multi-target recipients
		// from 0x28 do not include reliable per-recipient amounts in this format.
	default:
		return nil, fmt.Errorf("unknown healing subtype 0x%02x", v.Type)
	}
	return v, nil
}
