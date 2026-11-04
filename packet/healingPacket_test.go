package packet

import (
	"encoding/binary"
	"testing"
)

func TestParseDedicatedHealingPacket(t *testing.T) {
	raw := make([]byte, 31)
	raw[0] = 1
	binary.LittleEndian.PutUint32(raw[1:5], uint32(len(raw)))
	binary.BigEndian.PutUint32(raw[6:10], OpCodeHealing)
	binary.BigEndian.PutUint64(raw[10:18], 12345)
	raw[18] = 0x0A
	binary.BigEndian.PutUint32(raw[27:31], 678)
	p := &GamePacket{Op: OpCodeHealing, RawPacket: raw}
	got, err := ParseHealingPacket(p)
	if err != nil {
		t.Fatal(err)
	}
	if got.Type != 0x0A || got.EntityID != 12345 || got.Amount != 678 {
		t.Fatalf("unexpected parsed healing packet: %+v", got)
	}
}
