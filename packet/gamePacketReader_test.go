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

func TestCircularArithmetic(t *testing.T) {
	// Old sequence number near the wrap-around point (e.g. from your log)
	var oldSeq uint32 = 4293934746
	// New sequence number wrapped around to a small number
	var newSeq uint32 = 446430

	// Unsigned comparison (which was buggy and caused the parser to fail)
	if !(oldSeq > newSeq) {
		t.Error("unsigned oldSeq should be greater than newSeq")
	}

	// Correct circular comparison: oldSeq is actually BEFORE newSeq
	// (i.e. oldSeq < newSeq in TCP circular sequence space)
	if int32(oldSeq-newSeq) >= 0 {
		t.Error("oldSeq should be before newSeq in circular sequence space")
	}

	if int32(newSeq-oldSeq) <= 0 {
		t.Error("newSeq should be after oldSeq in circular sequence space")
	}

	// Test a normal non-wrapped sequence pair
	var seqA uint32 = 1000
	var seqB uint32 = 1500

	if int32(seqA-seqB) >= 0 {
		t.Error("seqA should be before seqB in normal sequence space")
	}
	if int32(seqB-seqA) <= 0 {
		t.Error("seqB should be after seqA in normal sequence space")
	}
}

