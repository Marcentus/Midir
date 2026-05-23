package packet

import (
	"testing"
	"time"
)

func testGamePacket(at time.Time, raw []byte) *GamePacket {
	return &GamePacket{
		At:        at,
		Sign:      1,
		Length:    uint32(len(raw)),
		Flag:      3,
		Op:        0x0f44bba3,
		Id:        42,
		RawPacket: raw,
	}
}

func TestParsedPacketDeduperDropsDuplicateAcrossFlows(t *testing.T) {
	now := time.Unix(100, 0)
	deduper := newParsedPacketDeduper()
	first := testGamePacket(now, []byte{1, 2, 3, 4})
	duplicate := testGamePacket(now.Add(50*time.Millisecond), []byte{1, 2, 3, 4})

	if deduper.IsDuplicate(first, tcpFlowKey("relay-a>client")) {
		t.Fatal("first packet should not be a duplicate")
	}

	if !deduper.IsDuplicate(duplicate, tcpFlowKey("relay-b>client")) {
		t.Fatal("same parsed packet on a different flow should be dropped")
	}
}

func TestParsedPacketDeduperAllowsSameFlowRepeats(t *testing.T) {
	now := time.Unix(100, 0)
	deduper := newParsedPacketDeduper()
	first := testGamePacket(now, []byte{1, 2, 3, 4})
	repeat := testGamePacket(now.Add(50*time.Millisecond), []byte{1, 2, 3, 4})

	if deduper.IsDuplicate(first, tcpFlowKey("relay-a>client")) {
		t.Fatal("first packet should not be a duplicate")
	}

	if deduper.IsDuplicate(repeat, tcpFlowKey("relay-a>client")) {
		t.Fatal("same-flow repeats should not be treated as multi-route duplicates")
	}
}

func TestParsedPacketDeduperAllowsExpiredDuplicate(t *testing.T) {
	now := time.Unix(100, 0)
	deduper := newParsedPacketDeduper()
	first := testGamePacket(now, []byte{1, 2, 3, 4})
	later := testGamePacket(now.Add(parsedPacketDedupeTTL+time.Second), []byte{1, 2, 3, 4})

	if deduper.IsDuplicate(first, tcpFlowKey("relay-a>client")) {
		t.Fatal("first packet should not be a duplicate")
	}

	if deduper.IsDuplicate(later, tcpFlowKey("relay-b>client")) {
		t.Fatal("duplicate outside the TTL should be allowed")
	}
}

func TestParsedPacketDeduperAllowsDifferentPacketsAcrossFlows(t *testing.T) {
	now := time.Unix(100, 0)
	deduper := newParsedPacketDeduper()
	first := testGamePacket(now, []byte{1, 2, 3, 4})
	different := testGamePacket(now.Add(50*time.Millisecond), []byte{1, 2, 3, 5})

	if deduper.IsDuplicate(first, tcpFlowKey("relay-a>client")) {
		t.Fatal("first packet should not be a duplicate")
	}

	if deduper.IsDuplicate(different, tcpFlowKey("relay-b>client")) {
		t.Fatal("different parsed packets across flows should be allowed")
	}
}
