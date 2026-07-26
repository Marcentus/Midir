package packet

import (
	"encoding/binary"
	"testing"
	"time"

	"github.com/gopacket/gopacket/layers"
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

func TestLooksLikeIPv4Header(t *testing.T) {
	if looksLikeIPv4Header([]byte{0x45}) {
		t.Fatal("too-short buffer should be rejected")
	}

	valid := make([]byte, 20)
	valid[0] = 0x45 // version 4, IHL 5
	if !looksLikeIPv4Header(valid) {
		t.Fatal("valid IPv4 header should be accepted")
	}

	valid[0] = 0x46 // IHL 6 still ok
	if !looksLikeIPv4Header(valid) {
		t.Fatal("IHL >= 5 should be accepted")
	}

	valid[0] = 0x44 // IHL 4 too small
	if looksLikeIPv4Header(valid) {
		t.Fatal("IHL < 5 should be rejected")
	}

	valid[0] = 0x65 // version 6
	if looksLikeIPv4Header(valid) {
		t.Fatal("non-IPv4 version should be rejected")
	}
}

func buildIPv4Datagram(proto byte, payloadLen int) []byte {
	total := 20 + payloadLen
	b := make([]byte, total)
	b[0] = 0x45
	binary.BigEndian.PutUint16(b[2:4], uint16(total))
	b[9] = proto
	return b
}

func TestFindInnerIPv4PacketAtOffset(t *testing.T) {
	inner := buildIPv4Datagram(6, 8) // TCP
	outer := append([]byte{0xaa, 0xbb, 0xcc}, inner...)

	got := findInnerIPv4Packet(outer)
	if got == nil {
		t.Fatal("expected embedded IPv4 datagram")
	}
	if len(got) != len(inner) {
		t.Fatalf("unexpected length: got %d want %d", len(got), len(inner))
	}
	for i := range inner {
		if got[i] != inner[i] {
			t.Fatalf("mismatch at %d: got %02x want %02x", i, got[i], inner[i])
		}
	}
}

func TestFindInnerIPv4PacketRejectsBadLength(t *testing.T) {
	b := make([]byte, 24)
	b[0] = 0x45
	binary.BigEndian.PutUint16(b[2:4], 40) // claims 40 bytes but buffer is 24
	b[9] = 6

	if findInnerIPv4Packet(b) != nil {
		t.Fatal("oversized total length should be rejected")
	}
}

func TestFindInnerIPv4PacketRejectsNonTCPUDP(t *testing.T) {
	inner := buildIPv4Datagram(1, 8) // ICMP
	if findInnerIPv4Packet(inner) != nil {
		t.Fatal("non-TCP/UDP protocol should be rejected")
	}
}

func TestFindInnerIPv4PacketAcceptsUDP(t *testing.T) {
	inner := buildIPv4Datagram(17, 4) // UDP
	if findInnerIPv4Packet(inner) == nil {
		t.Fatal("UDP protocol should be accepted")
	}
}

func TestIsLikelyMudfishInbound(t *testing.T) {
	inbound := layers.TCP{SrcPort: 8085, DstPort: 54340}
	if !isLikelyMudfishInbound(inbound) {
		t.Fatal("service-to-ephemeral traffic should be inbound")
	}

	outbound := layers.TCP{SrcPort: 54340, DstPort: 8085}
	if isLikelyMudfishInbound(outbound) {
		t.Fatal("ephemeral-to-service traffic should be outbound")
	}
}
