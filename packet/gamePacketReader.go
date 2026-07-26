package packet

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"hash/fnv"
	"io"
	"os"
	"sync/atomic"
	"time"

	"github.com/Marcentus/Midir/util"
	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"
	"github.com/gopacket/gopacket/pcap"
)

// pcapWriter defines an interface that our SessionManager will satisfy.
type pcapWriter interface {
	WritePacketToLog(ci gopacket.CaptureInfo, data []byte)
}

type GameServerPacketReader struct {
	// non-mutable
	ctx      context.Context
	packetCh chan *GamePacket
	sm       pcapWriter
	exitlag  bool
	mudfish  bool

	// mutable
	handle *pcap.Handle
	fd     *os.File

	// stats
	pcapDrops    uint32
	parserErrors uint32
	networkLoss  uint32
	queueDrops   uint32
}

type GameServerPacketReaderOpt struct {
	Ctx             context.Context
	FileName        string
	NicName         string
	ClientIp        string
	Sm              pcapWriter
	ExitLagEnabled  bool // Unwrap ExitLag framing before Mabi parse
	MudfishEnabled  bool // Unwrap Mudfish UDP/TCP tunnels
	Filter          string
	PromiscuousMode bool
}

type tcpFlowKey string

type gamePacketPayload struct {
	relSeq  uint32
	data    []byte
	at      time.Time
	flowKey tcpFlowKey
}

type pendingTcpLayer struct {
	tcpLayer layers.TCP
	ci       gopacket.CaptureInfo
}

type tcpStreamState struct {
	baseSeq  uint32
	nextSeq  uint32
	pending  []pendingTcpLayer
	lastSeen time.Time
}

type exitLagStreamState struct {
	buffer               bytes.Buffer
	isReadingMabiPayload bool
	mabiBytesToRead      uint32
	packetStartTime      time.Time
	packetRelSeq         uint32
	lastSeen             time.Time
	lastHeader           []byte
}

type gamePacketAssemblerState struct {
	buffer     *bytes.Buffer
	lastRelSeq uint32
	lastAt     time.Time
	payloads   []gamePacketPayload
	flowKey    tcpFlowKey

	// Mudfish captures every inbound flow to the game client, so a flow is only
	// known to carry Mabinogi traffic once it yields one valid packet.
	verified      bool
	rejected      bool
	parseFailures int
}

type dedupeKey struct {
	sign   uint8
	length uint32
	flag   uint8
	op     uint32
	id     uint64
	hash   uint64
}

type parsedPacketDedupeEntry struct {
	at      time.Time
	flowKey tcpFlowKey
}

type parsedPacketDeduper struct {
	seen      map[dedupeKey]parsedPacketDedupeEntry
	lastSweep time.Time
}

const pcapQueueSize = 4096
const pcapSnapLen = 65536
const pcapBufferSize = 32 * 1024 * 1024
const packetQueueSize = 4096
const parsedPacketDedupeTTL = 3 * time.Second
const maxAssemblerFlows = 64
const maxAssemblerBufferSize = 0x100_0000
const maxUnverifiedFlowParseFailures = 32

var ErrTooShortPacket = errors.New("too short packet")

func gamePacketDedupeKeyStruct(msg *GamePacket) dedupeKey {
	h := fnv.New64a()
	h.Write(msg.RawPacket)
	return dedupeKey{
		sign:   msg.Sign,
		length: msg.Length,
		flag:   msg.Flag,
		op:     msg.Op,
		id:     msg.Id,
		hash:   h.Sum64(),
	}
}

func gamePacketDedupeKey(msg *GamePacket) string {
	h := fnv.New64a()
	h.Write(msg.RawPacket)
	return fmt.Sprintf("%02x:%d:%d:%d:%d:%x", msg.Sign, msg.Length, msg.Flag, msg.Op, msg.Id, h.Sum64())
}

func shortPacketDedupeKey(key string) string {
	if len(key) <= 24 {
		return key
	}
	return key[:24]
}

func newParsedPacketDeduper() *parsedPacketDeduper {
	return &parsedPacketDeduper{
		seen:      make(map[dedupeKey]parsedPacketDedupeEntry),
		lastSweep: time.Now(),
	}
}

func (d *parsedPacketDeduper) IsDuplicate(msg *GamePacket, flowKey tcpFlowKey) bool {
	if d == nil || msg == nil || len(msg.RawPacket) == 0 || flowKey == "" {
		return false
	}

	now := msg.At
	if now.IsZero() {
		now = time.Now()
	}

	if now.Sub(d.lastSweep) > parsedPacketDedupeTTL {
		for key, entry := range d.seen {
			if now.Sub(entry.at) > parsedPacketDedupeTTL {
				delete(d.seen, key)
			}
		}
		d.lastSweep = now
	}

	keyStruct := gamePacketDedupeKeyStruct(msg)
	if entry, ok := d.seen[keyStruct]; ok && now.Sub(entry.at) <= parsedPacketDedupeTTL && entry.flowKey != flowKey {
		return true
	}

	d.seen[keyStruct] = parsedPacketDedupeEntry{
		at:      now,
		flowKey: flowKey,
	}
	return false
}

// This function is provided for completeness and is unchanged from the original file.
func (t *GameServerPacketReader) exitLagPacketLoop(rawPayloadCh <-chan gamePacketPayload, mabiPayloadCh chan<- gamePacketPayload) {
	defer close(mabiPayloadCh)

	streams := make(map[tcpFlowKey]*exitLagStreamState)
	const streamTTL = 30 * time.Second

	for {
		select {
		case <-t.ctx.Done():
			return
		case payloadData, ok := <-rawPayloadCh:
			if !ok {
				return
			}

			now := payloadData.at
			for key, st := range streams {
				if !st.lastSeen.IsZero() && now.Sub(st.lastSeen) > streamTTL {
					delete(streams, key)
				}
			}

			st := streams[payloadData.flowKey]
			if st == nil {
				st = &exitLagStreamState{}
				streams[payloadData.flowKey] = st
			}
			st.lastSeen = now
			st.buffer.Write(payloadData.data)
			if !st.isReadingMabiPayload {
				st.packetStartTime = payloadData.at
				st.packetRelSeq = payloadData.relSeq
			}

		parseLoop:
			for {
				if !st.isReadingMabiPayload {
					const headerSignatureLen = 5
					foundHeader := false
					for st.buffer.Len() >= headerSignatureLen {
						if st.buffer.Bytes()[0] == 0x01 && st.buffer.Bytes()[4] == 0x05 {
							foundHeader = true
							break
						}
						st.buffer.Next(1)
					}

					if !foundHeader {
						break parseLoop
					}
				}

				if st.isReadingMabiPayload {
					if uint32(st.buffer.Len()) < st.mabiBytesToRead {
						break parseLoop
					}

					mabiPayload := make([]byte, st.mabiBytesToRead)
					_, err := st.buffer.Read(mabiPayload)
					if err != nil {
						st.isReadingMabiPayload = false
						continue parseLoop
					}

					// Discard ExitLag keep-alive/heartbeat packets if payload is exactly 5 bytes and equals "pong\n"
					// Theres gotta be a better way of determining exitlag only packets
					if len(mabiPayload) == 5 && bytes.Equal(mabiPayload, []byte("pong\n")) {
						// wholePacket := append([]byte(nil), st.lastHeader...)
						// wholePacket = append(wholePacket, mabiPayload...)
						// logger.Printf("[ExitLag] Skipped pong keep-alive packet. Payload len: %d, Payload hex: %x\nWhole packet (header+payload) hex: %02x", len(mabiPayload), mabiPayload, wholePacket)
						if st.buffer.Len() >= 4 {
							if bytes.Equal(st.buffer.Bytes()[:4], []byte{0x05, 0x25, 0x01, 0x01}) {
								st.buffer.Next(4)
							}
						}
						st.isReadingMabiPayload = false
						continue parseLoop
					}

					mabiPayloadCh <- gamePacketPayload{
						relSeq:  st.packetRelSeq,
						at:      st.packetStartTime,
						data:    mabiPayload,
						flowKey: payloadData.flowKey,
					}

					if st.buffer.Len() >= 4 {
						if bytes.Equal(st.buffer.Bytes()[:4], []byte{0x05, 0x25, 0x01, 0x01}) {
							st.buffer.Next(4)
						}
					}

					st.isReadingMabiPayload = false
					continue parseLoop
				}

				b := st.buffer.Bytes()
				const minHeaderSize = 38
				if len(b) < minHeaderSize {
					break parseLoop
				}

				seqLenIndOffset := 1 + 2 + 30
				seqLenInd := b[seqLenIndOffset]

				if seqLenInd == 0 {
					bodyLen := le.Uint16(b[1:3])
					totalSize := 1 + 2 + int(bodyLen)

					if st.buffer.Len() < totalSize {
						break parseLoop
					}

					if st.buffer.Len() >= totalSize+4 && bytes.Equal(b[totalSize:totalSize+4], []byte{0x05, 0x25, 0x01, 0x01}) {
						st.buffer.Next(totalSize + 4)
					} else {
						st.buffer.Next(totalSize)
					}
					continue parseLoop
				}

				seqLen := int(seqLenInd)
				if seqLen <= 0 || seqLen > 8 {
					st.buffer.Next(1)
					continue parseLoop
				}

				payloadLenIndOffset := seqLenIndOffset + 1 + seqLen
				if st.buffer.Len() < payloadLenIndOffset+1 {
					break parseLoop
				}
				payloadLenInd := b[payloadLenIndOffset]

				var mabiPayloadLenBytes int
				if payloadLenInd == 0x05 {
					mabiPayloadLenBytes = 1
				} else if payloadLenInd == 0x09 {
					mabiPayloadLenBytes = 2
				} else {
					st.buffer.Next(1)
					continue parseLoop
				}

				flagOffset := payloadLenIndOffset + 1
				if st.buffer.Len() < flagOffset+1 {
					break parseLoop
				}
				flag := b[flagOffset]

				if flag != 0x23 {
					st.buffer.Next(1)
					continue parseLoop
				}

				mabiLenOffset := flagOffset + 1
				if st.buffer.Len() < mabiLenOffset+mabiPayloadLenBytes {
					break parseLoop
				}
				mabiLen := uint32(0)
				if mabiPayloadLenBytes == 1 {
					mabiLen = uint32(b[mabiLenOffset])
				} else {
					mabiLen = uint32(le.Uint16(b[mabiLenOffset : mabiLenOffset+2]))
				}

				headerTotalLen := mabiLenOffset + mabiPayloadLenBytes
				st.lastHeader = make([]byte, headerTotalLen)
				copy(st.lastHeader, b[:headerTotalLen])

				st.buffer.Next(headerTotalLen)
				st.isReadingMabiPayload = true
				st.mabiBytesToRead = mabiLen
				continue parseLoop
			}
		}
	}
}

func NewGameServerPacketReader(opt *GameServerPacketReaderOpt) (*GameServerPacketReader, error) {
	if opt == nil {
		return nil, errors.New("opt is nil")
	}

	// Use the filter from the options struct.
	filter := opt.Filter
	if opt.ClientIp != "" {
		// This condition might be legacy, but we'll keep it.
		// A more robust implementation might be to require the IP to be part of the filter from the start.
		filter = " dst host " + opt.ClientIp
	}

	logger.Println("game packet filter...", filter)

	v := &GameServerPacketReader{
		ctx:      opt.Ctx,
		packetCh: make(chan *GamePacket, packetQueueSize),
		sm:       opt.Sm,
		exitlag:  opt.ExitLagEnabled,
		mudfish:  opt.MudfishEnabled,
	}

	rawPayloadCh, err := (<-chan gamePacketPayload)(nil), (error)(nil)
	if opt.FileName != "" {
		rawPayloadCh, err = v.openFile(opt.FileName, filter)
		if err != nil {
			logger.Println("openFile failed", err)
			return nil, err
		}
	} else {
		rawPayloadCh, err = v.openNic(opt.NicName, filter, opt.PromiscuousMode)
		if err != nil {
			logger.Println("openNic failed", err)
			return nil, err
		}
	}

	// This is the channel that the final packetLoop will read from.
	var finalPayloadCh <-chan gamePacketPayload

	if opt.ExitLagEnabled {
		logger.Println("ExitLag mode enabled. Packet processing pipeline includes ExitLag stripper.")
		// If ExitLag is enabled, create the pipeline with the stripping loop.
		mabiPayloadCh := make(chan gamePacketPayload, packetQueueSize)
		go v.exitLagPacketLoop(rawPayloadCh, mabiPayloadCh)
		finalPayloadCh = mabiPayloadCh
	} else {
		if opt.MudfishEnabled {
			logger.Println("Mudfish mode enabled. Capturing UDP/TCP tunnels (no ExitLag unwrap); follow inbound to client.")
		} else {
			logger.Println("ExitLag/Mudfish unwrap disabled. Using raw TCP payloads.")
		}
		finalPayloadCh = rawPayloadCh
	}

	// The final packet loop reads from the configured channel (either direct or post-ExitLag).
	go v.packetLoop(finalPayloadCh)

	return v, nil
}

func (t *GameServerPacketReader) packetLoop(payloadCh <-chan gamePacketPayload) {
	defer close(t.packetCh)
	// Loop forever to allow restarting after a panic recovery.
	for {
		// Define the recovery function for this iteration.
		// If a panic occurs, we log it, reset state, send an error packet,
		// and break out of the inner function to restart the loop.
		func() {
			defer func() {
				if r := recover(); r != nil {
					logger.Printf("!!!!!!!!!!!!!!!!!\n")
					logger.Printf("FATAL: a panic was recovered in packetLoop: %v\n", r)
					logger.Printf("Resetting packet assembler state and continuing...\n")
					logger.Printf("!!!!!!!!!!!!!!!!\n")

					// Create an error message
					errMsg := fmt.Sprintf("Packet processing error: %v. Live updates may have stuttered but should resume.", r)

					// Inject System Error Packet
					// We construct a fake packet with the special OpCode
					errorPacket := &GamePacket{
						At: time.Now(),
						Op: OpCodeSystemError,
						// Encode the error message into the raw packet or handle it specially in the publisher.
						// Since we can't easily adhere to the binary structure here without a lot of work,
						// we'll abuse the `RawPacket` field to hold the string message for now,
						// OR rely on the publisher to just see the OpCode and send a generic alert.
						// Let's do a trick: we'll create a dummy Message that carries the string if possible,
						// but specific implementation in publisher is safer.
						// For now, let's just use the OpCode. The publisher will send a generic warning.
					}

					// If we really want to pass the string, we might need a custom struct or field,
					// but OpCodeSystemError is a strong enough signal.
					// Let's try to put the string in RawPacket just in case we change our mind.
					errorPacket.RawPacket = []byte(errMsg)

					select {
					case t.packetCh <- errorPacket:
					default:
						atomic.AddUint32(&t.queueDrops, 1)
						logger.Println("Packet channel full, dropped error packet.")
					}
				}
			}()

			assemblers := make(map[tcpFlowKey]*gamePacketAssemblerState)
			parsedPacketDedupe := newParsedPacketDeduper()

			getAssembler := func(key tcpFlowKey) *gamePacketAssemblerState {
				if key == "" {
					key = "default"
				}
				st := assemblers[key]
				if st == nil {
					if len(assemblers) >= maxAssemblerFlows {
						var oldestKey tcpFlowKey
						var oldestAt time.Time
						for candidateKey, candidate := range assemblers {
							if oldestAt.IsZero() || candidate.lastAt.Before(oldestAt) {
								oldestKey = candidateKey
								oldestAt = candidate.lastAt
							}
						}
						delete(assemblers, oldestKey)
						logger.Printf("[Packet Assembler] evicted oldest flow %s after reaching the %d-flow limit", oldestKey, maxAssemblerFlows)
					}
					st = &gamePacketAssemblerState{
						buffer:   bytes.NewBuffer(nil),
						lastAt:   time.Now(),
						payloads: make([]gamePacketPayload, 0, 100),
						flowKey:  key,
					}
					assemblers[key] = st
				}
				return st
			}

			skipPayload := func(st *gamePacketAssemblerState, n int) {
				for n > 0 {
					if len(st.payloads) == 0 {
						return
					}
					if n < len(st.payloads[0].data) {
						st.lastRelSeq, st.lastAt = st.payloads[0].relSeq, st.payloads[0].at
						st.payloads[0].data = st.payloads[0].data[n:]
						return
					}
					n -= len(st.payloads[0].data)
					if len(st.payloads) > 0 {
						st.lastRelSeq, st.lastAt = st.payloads[0].relSeq, st.payloads[0].at
						st.payloads = st.payloads[1:]
					}
				}
			}

			nextPayload := func(st *gamePacketAssemblerState) {
				st.buffer.Reset()
				if len(st.payloads) < 1 {
					return
				}
				st.payloads = st.payloads[1:]
				if len(st.payloads) < 1 {
					return
				}
				for _, v := range st.payloads {
					st.buffer.Write(v.data)
				}
				if len(st.payloads) > 0 {
					st.lastRelSeq = st.payloads[0].relSeq
				}
			}

			pushPayload := func(st *gamePacketAssemblerState, payloadData gamePacketPayload) {
				if st.buffer.Len() < 1 {
					st.buffer.Reset()
				}
				if len(st.payloads) < 1 {
					st.lastRelSeq, st.lastAt = payloadData.relSeq, payloadData.at
				}
				st.payloads = append(st.payloads, payloadData)
				st.buffer.Write(payloadData.data)
				if st.buffer.Len() > maxAssemblerBufferSize {
					atomic.AddUint32(&t.parserErrors, 1)
					logger.Printf("[Packet Assembler] discarded flow %s after buffered data exceeded %d bytes", st.flowKey, maxAssemblerBufferSize)
					st.buffer = bytes.NewBuffer(nil)
					st.payloads = st.payloads[:0]
				}
			}

			// Inner processing loop
			for {
				var st *gamePacketAssemblerState
				select {
				case <-t.ctx.Done():
					return
				case payloadData, ok := <-payloadCh:
					if !ok {
						return
					}
					st = getAssembler(payloadData.flowKey)
					if st.rejected {
						continue
					}
					pushPayload(st, payloadData)
				}

			readerLoop:
				for {
					msg, err := ParseGamePacket(st.buffer, st.lastAt)
					if err != nil {
						if err == io.EOF {
							break readerLoop
						}
						// An unverified Mudfish flow may simply be non-game traffic
						// routed to the same client, so failures there are expected
						// until the flow proves itself.
						if t.mudfish && !st.verified {
							st.parseFailures++
							if st.parseFailures >= maxUnverifiedFlowParseFailures {
								st.rejected = true
								st.buffer = bytes.NewBuffer(nil)
								st.payloads = st.payloads[:0]
								logger.Printf("[Mudfish] ignoring flow %s: no valid game packet after %d attempts", st.flowKey, st.parseFailures)
								break readerLoop
							}
							nextPayload(st)
							continue
						}
						atomic.AddUint32(&t.parserErrors, 1)
						b := st.buffer.Bytes()
						note := ""
						if len(b) >= 5 && b[0] == 0x01 && (b[4] == 0x05 || b[4] == 0x03) {
							note = fmt.Sprintf(" [NOTE: MATCHES EXITLAG SIGNATURE! Byte 0: 0x%02x, Byte 4: 0x%02x]", b[0], b[4])
						}
						logger.Printf("[Parse Error] game packet parse error %v %v%s on flow %s", st.lastRelSeq, err, note, st.flowKey)
						nextPayload(st)
						continue
					}
					if msg != nil {
						if !st.verified {
							st.verified = true
							st.parseFailures = 0
							if t.mudfish {
								logger.Printf("[Mudfish] flow %s verified as game traffic", st.flowKey)
							}
						}
						msg.PacketFlowKey = string(st.flowKey)
						if t.exitlag && parsedPacketDedupe.IsDuplicate(msg, st.flowKey) {
							skipPayload(st, len(msg.RawPacket))
							continue
						}
						msg.PacketDedupeKey = gamePacketDedupeKey(msg)
						t.packetCh <- msg
						skipPayload(st, len(msg.RawPacket))
					}
				}
			}

		}()

		// Check if we should exit truly
		select {
		case <-t.ctx.Done():
			return
		default:
			// If we are here, it means the inner func returned (likely via panic recovery).
			// We loop back to the top to restart the inner func with fresh state.
			time.Sleep(100 * time.Millisecond) // partial backoff
			continue
		}
	}
}

func (t *GameServerPacketReader) openNic(nic string, filter string, promiscuous bool) (<-chan gamePacketPayload, error) {
	inactive, err := pcap.NewInactiveHandle(nic)
	if err != nil {
		logger.Println("NewInactiveHandle failed:", err)
		return nil, err
	}
	defer inactive.CleanUp()

	if err := inactive.SetSnapLen(pcapSnapLen); err != nil {
		logger.Println("SetSnapLen failed:", err)
		return nil, err
	}

	if err := inactive.SetPromisc(promiscuous); err != nil {
		logger.Println("SetPromisc failed:", err)
		return nil, err
	}

	if err := inactive.SetTimeout(pcap.BlockForever); err != nil {
		logger.Println("SetTimeout failed:", err)
		return nil, err
	}

	if err := inactive.SetBufferSize(pcapBufferSize); err != nil {
		logger.Println("SetBufferSize failed:", err)
		return nil, err
	}

	handle, err := inactive.Activate()
	if err != nil {
		logger.Println("Activate failed:", err)
		return nil, err
	}
	t.handle = handle

	if err := handle.SetBPFFilter(filter); err != nil {
		handle.Close()
		t.handle = nil
		return nil, err
	}

	logger.Printf(
		"pcap open nic=%q linkType=%v (%d) filter=%q exitlag=%v mudfish=%v",
		nic,
		handle.LinkType().String(),
		handle.LinkType(),
		filter,
		t.exitlag,
		t.mudfish,
	)

	ch := make(chan gamePacketPayload, pcapQueueSize)
	go t.readPacketLoop(ch)
	return ch, nil
}

func (t *GameServerPacketReader) openFile(file string, filter string) (<-chan gamePacketPayload, error) {
	fd, err := os.Open(file)
	if err != nil {
		logger.Println(err)
		return nil, err
	}
	t.fd = fd

	// **CORRECTED LINE**
	// We pass the file descriptor 'fd' to OpenOfflineFile, not the file path string 'file'.
	handle, err := pcap.OpenOfflineFile(fd)
	if err != nil {
		logger.Println(err)
		return nil, err
	}

	if err := handle.SetBPFFilter(filter); err != nil {
		logger.Println(err)
		return nil, err
	}

	t.handle = handle

	ch := make(chan gamePacketPayload, pcapQueueSize)
	time.AfterFunc(1*time.Second, func() {
		logger.Println("start readPacketLoop from file:", file)
		go t.readPacketLoop(ch)
	})

	return ch, nil
}

func looksLikeIPv4Header(b []byte) bool {
	if len(b) < 20 {
		return false
	}
	version := b[0] >> 4
	ihl := b[0] & 0x0f
	return version == 4 && ihl >= 5
}

// findInnerIPv4Packet scans an outer tunnel payload for an embedded IPv4 datagram.
// Mudfish does not encrypt its tunnel; mirrored ISP/Ethernet traffic is often UDP
// (or TCP) to a Mudfish node with the original game IP packet inside.
func findInnerIPv4Packet(b []byte) []byte {
	for off := 0; off+20 <= len(b); off++ {
		if !looksLikeIPv4Header(b[off:]) {
			continue
		}
		total := int(binary.BigEndian.Uint16(b[off+2 : off+4]))
		if total < 20 || off+total > len(b) {
			continue
		}
		proto := b[off+9]
		if proto != 6 && proto != 17 { // TCP or UDP
			continue
		}
		return b[off : off+total]
	}
	return nil
}

func isLikelyMudfishInbound(tcp layers.TCP) bool {
	return tcp.SrcPort < tcp.DstPort
}

func (t *GameServerPacketReader) readPacketLoop(ch chan<- gamePacketPayload) {
	defer close(ch)
	ethLayer := layers.Ethernet{}
	ip4Layer := layers.IPv4{}
	tcpLayer := layers.TCP{}
	udpLayer := layers.UDP{}
	payload := gopacket.Payload{}

	linkType := layers.LinkTypeEthernet
	if t.handle != nil {
		linkType = t.handle.LinkType()
	}

	// Some virtual adapters expose raw IP (DLT_RAW / DLT_IPV4), not Ethernet.
	useRawIP := linkType == layers.LinkTypeRaw ||
		linkType == layers.LinkTypeIPv4 ||
		linkType == layers.LinkTypeIPv6
	if t.mudfish && !useRawIP {
		logger.Printf(
			"Mudfish mode on linkType=%v (typical for switch mirror). Will unwrap UDP/TCP tunnels and follow inner game TCP.",
			linkType,
		)
	}

	ethParser := gopacket.NewDecodingLayerParser(layers.LayerTypeEthernet, &ethLayer, &ip4Layer, &tcpLayer, &udpLayer, &payload)
	rawParser := gopacket.NewDecodingLayerParser(layers.LayerTypeIPv4, &ip4Layer, &tcpLayer, &udpLayer, &payload)
	ethParser.IgnoreUnsupported = true
	rawParser.IgnoreUnsupported = true

	primaryRaw := useRawIP
	packetLayers := []gopacket.LayerType(nil)
	streams := make(map[tcpFlowKey]*tcpStreamState)
	const streamTTL = 30 * time.Second
	// Direction guard: identify the game client LAN IP, then allow any TCP source
	// inbound to that client. ExitLag/direct expect public relay -> private client.
	// Mudfish tunnels often show private->private or public mudfish node IPs.
	var followedClientIP string
	var loggedMudfishUnwrap bool

	flowKeyFor := func(ip layers.IPv4, tcp layers.TCP) tcpFlowKey {
		return tcpFlowKey(fmt.Sprintf("%s:%d>%s:%d", ip.SrcIP, tcp.SrcPort, ip.DstIP, tcp.DstPort))
	}

	findMinSeqIdx := func(layers []pendingTcpLayer) int {
		if len(layers) == 0 {
			return -1
		}
		minIdx := 0
		minSeq := layers[0].tcpLayer.Seq
		for i, l := range layers {
			if l.tcpLayer.Seq < minSeq {
				minSeq = l.tcpLayer.Seq
				minIdx = i
			}
		}
		return minIdx
	}

	drainReadyPending := func(st *tcpStreamState, key tcpFlowKey) {
		for {
			foundIdx := -1
			for idx, p := range st.pending {
				if p.tcpLayer.Seq == st.nextSeq {
					foundIdx = idx
					break
				}
			}

			if foundIdx == -1 {
				return
			}

			v := st.pending[foundIdx]
			ch <- gamePacketPayload{
				relSeq:  v.tcpLayer.Seq - st.baseSeq,
				data:    v.tcpLayer.Payload,
				at:      v.ci.Timestamp,
				flowKey: key,
			}
			st.nextSeq = v.tcpLayer.Seq + uint32(len(v.tcpLayer.Payload))
			st.pending[foundIdx] = st.pending[len(st.pending)-1]
			st.pending = st.pending[:len(st.pending)-1]
		}
	}

	decodePacket := func(b []byte) bool {
		packetLayers = packetLayers[:0]
		if primaryRaw || (t.mudfish && looksLikeIPv4Header(b)) {
			if err := rawParser.DecodeLayers(b, &packetLayers); err == nil {
				return true
			}
			if primaryRaw {
				return false
			}
		}
		if err := ethParser.DecodeLayers(b, &packetLayers); err != nil {
			if t.mudfish && looksLikeIPv4Header(b) {
				packetLayers = packetLayers[:0]
				return rawParser.DecodeLayers(b, &packetLayers) == nil
			}
			return false
		}
		return true
	}

	handleTCPSegment := func(ci gopacket.CaptureInfo, ip layers.IPv4, tcp layers.TCP) {
		// We must process control packets (SYN, FIN, RST) even if they have no payload.
		isControl := tcp.SYN || tcp.FIN || tcp.RST
		if len(tcp.Payload) < 1 && !isControl {
			return
		}

		if followedClientIP == "" {
			if !ip.DstIP.IsPrivate() {
				return
			}
			if !t.mudfish && (ip.SrcIP.IsPrivate() || ip.SrcIP.IsLoopback()) {
				return
			}
			if t.mudfish && ip.SrcIP.IsLoopback() {
				return
			}
			// Mudfish can expose both tunnel directions as private-to-private
			// traffic. Client ephemeral ports are higher than the service port,
			// so ignore the client-to-server direction while learning the client.
			if t.mudfish && !isLikelyMudfishInbound(tcp) {
				return
			}
			followedClientIP = ip.DstIP.String()
			logger.Printf(
				"[Dynamic Follow] inbound traffic to game client %s (exitlag=%v mudfish=%v linkType=%v)",
				followedClientIP,
				t.exitlag,
				t.mudfish,
				linkType,
			)
		} else if ip.DstIP.String() != followedClientIP {
			return
		}

		key := flowKeyFor(ip, tcp)
		for streamKey, st := range streams {
			if !st.lastSeen.IsZero() && ci.Timestamp.Sub(st.lastSeen) > streamTTL {
				delete(streams, streamKey)
			}
		}

		st := streams[key]

		if tcp.FIN || tcp.RST {
			if st != nil {
				delete(streams, key)
			}
			return
		}

		if tcp.SYN {
			if st != nil {
				st.baseSeq = tcp.Seq + 1
				st.nextSeq = tcp.Seq + 1
				st.pending = st.pending[:0]
			} else {
				st = &tcpStreamState{
					baseSeq:  tcp.Seq + 1,
					nextSeq:  tcp.Seq + 1,
					pending:  make([]pendingTcpLayer, 0, packetQueueSize),
					lastSeen: ci.Timestamp,
				}
				streams[key] = st
			}
			return
		}

		if st == nil {
			st = &tcpStreamState{
				baseSeq:  tcp.Seq,
				nextSeq:  tcp.Seq,
				pending:  make([]pendingTcpLayer, 0, packetQueueSize),
				lastSeen: ci.Timestamp,
			}
			streams[key] = st
		}
		st.lastSeen = ci.Timestamp

		if st.nextSeq != 0 && tcp.Seq != st.nextSeq {
			if tcp.Seq < st.nextSeq {
				if int32(tcp.Seq-st.nextSeq) > 0 {
					logger.Printf("[TCP Recovery Anomaly] Packet on %s is in the future but treated as past. Seq: %d, NextSeq: %d, PayloadLen: %d (Potential Wrap-around Bug)", key, tcp.Seq, st.nextSeq, len(tcp.Payload))
				}
				if tcp.Seq+uint32(len(tcp.Payload)) >= st.nextSeq {
					seg := tcp.Payload[st.nextSeq-tcp.Seq:]
					if len(seg) > 0 {
						ch <- gamePacketPayload{
							relSeq:  st.nextSeq - st.baseSeq,
							data:    seg,
							at:      ci.Timestamp,
							flowKey: key,
						}
					}
					st.nextSeq = tcp.Seq + uint32(len(tcp.Payload))
				}
				return
			}

			st.pending = append(st.pending, pendingTcpLayer{
				tcpLayer: tcp,
				ci:       ci,
			})

			if len(st.pending) > 300 {
				minIdx := findMinSeqIdx(st.pending)
				if minIdx != -1 {
					targetLayer := st.pending[minIdx]
					skippedBytes := targetLayer.tcpLayer.Seq - st.nextSeq
					warningMsg := fmt.Sprintf("Network packet loss detected on %s (Gap: %d bytes). Skipping to resume.", key, skippedBytes)
					atomic.AddUint32(&t.networkLoss, 1)

					logger.Printf("[TCP Recovery] %s OldNext: %d, NewNext: %d (Active Streams: %d, Pending Count: %d)",
						warningMsg, st.nextSeq, targetLayer.tcpLayer.Seq, len(streams), len(st.pending))

					st.nextSeq = targetLayer.tcpLayer.Seq

					select {
					case t.packetCh <- &GamePacket{
						At:        time.Now(),
						Op:        OpCodeSystemWarning,
						RawPacket: []byte(warningMsg),
					}:
					default:
						atomic.AddUint32(&t.queueDrops, 1)
						logger.Println("Packet channel full, dropped warning packet.")
					}
				}
			}

			drainReadyPending(st, key)
			return
		}

		ch <- gamePacketPayload{
			relSeq:  tcp.Seq - st.baseSeq,
			data:    tcp.Payload,
			at:      ci.Timestamp,
			flowKey: key,
		}
		st.nextSeq = tcp.Seq + uint32(len(tcp.Payload))
		drainReadyPending(st, key)
	}

	tryHandleInnerIPv4 := func(ci gopacket.CaptureInfo, outer []byte) {
		inner := findInnerIPv4Packet(outer)
		if inner == nil {
			return
		}
		var innerIP layers.IPv4
		var innerTCP layers.TCP
		var innerPayload gopacket.Payload
		innerLayers := []gopacket.LayerType(nil)
		innerParser := gopacket.NewDecodingLayerParser(layers.LayerTypeIPv4, &innerIP, &innerTCP, &innerPayload)
		innerParser.IgnoreUnsupported = true
		if err := innerParser.DecodeLayers(inner, &innerLayers); err != nil {
			return
		}
		hasTCP := false
		for _, lt := range innerLayers {
			if lt == layers.LayerTypeTCP {
				hasTCP = true
				break
			}
		}
		if !hasTCP || len(innerTCP.Payload) < 1 {
			return
		}
		if t.mudfish && !isLikelyMudfishInbound(innerTCP) {
			return
		}
		if !loggedMudfishUnwrap {
			loggedMudfishUnwrap = true
			logger.Printf(
				"[Mudfish] unwrapped inner TCP %s:%d -> %s:%d from tunnel payload",
				innerIP.SrcIP, innerTCP.SrcPort, innerIP.DstIP, innerTCP.DstPort,
			)
		}
		handleTCPSegment(ci, innerIP, innerTCP)
	}

	for i := 0; t.ctx.Err() == nil; i++ {
		b, ci, err := t.handle.ReadPacketData()
		if err != nil {
			if err == io.EOF {
				break
			}
			logger.Println("ReadPacketData non-fatal error, continuing:", err, i)
			time.Sleep(1 * time.Second)
			continue
		}

		if t.sm != nil {
			t.sm.WritePacketToLog(ci, b)
		}

		if !decodePacket(b) {
			continue
		}

		hasIPv4 := false
		hasTCP := false
		hasUDP := false
		for _, layer := range packetLayers {
			switch layer {
			case layers.LayerTypeIPv4:
				hasIPv4 = true
			case layers.LayerTypeTCP:
				hasTCP = true
			case layers.LayerTypeUDP:
				hasUDP = true
			}
		}
		if !hasIPv4 {
			continue
		}

		if t.mudfish && hasUDP && len(udpLayer.Payload) > 0 {
			tryHandleInnerIPv4(ci, udpLayer.Payload)
			continue
		}

		if hasTCP {
			if t.mudfish {
				if len(tcpLayer.Payload) > 0 {
					// Some Mudfish connection modes use TCP tunnels; try unwrap first.
					if findInnerIPv4Packet(tcpLayer.Payload) != nil {
						tryHandleInnerIPv4(ci, tcpLayer.Payload)
						continue
					}
					// On Ethernet mirrors, drop opaque Mudfish TCP tunnel bytes.
					// On raw-IP adapters, fall through to direct game TCP handling.
					if !useRawIP {
						continue
					}
				} else if !useRawIP {
					// Ignore outer TCP control/empty segments on Ethernet so they
					// cannot lock followedClientIP to the LAN address before
					// inner Mudfish VIP traffic is seen.
					continue
				}
			}
			handleTCPSegment(ci, ip4Layer, tcpLayer)
		}
	}
}

func (t *GameServerPacketReader) Close() {
	if t.handle != nil {
		t.handle.Close()
		t.handle = nil
	}
	if t.fd != nil {
		t.fd.Close()
		t.fd = nil
	}
}

func (t *GameServerPacketReader) PacketCh() <-chan *GamePacket {
	return t.packetCh
}

func ParseGamePacket(buffer *bytes.Buffer, at time.Time) (*GamePacket, error) {
	headerSize := 6
	rawPacketBuffer := bytes.NewBuffer(nil)
	b := buffer.Bytes()

	if len(b) < 6 {
		return nil, io.EOF
	}

	sign := b[0]
	length := le.Uint32(b[1:])
	// logger.Printf("[gamePacketReader] Parsed header. Sign: 0x%X, Advertised Length: %d", sign, length)
	flag := b[5]

	if length == 0 || length > 0x100_0000 {
		return nil, fmt.Errorf("invalid packet length %v", length)
	}
	if flag > 4 {
		return nil, fmt.Errorf("invalid flag %v", flag)
	}
	isShortPacket := flag == 1 || flag == 2

	if isShortPacket {
		if len(b) < int(length)-6 {
			return nil, io.EOF
		}
		shortBody := b[6:int(length)]
		rawPacketBuffer.Write(shortBody)
		buffer.Next(int(length))
		v := &GamePacket{
			At:            at,
			Sign:          sign,
			Length:        length,
			Flag:          flag,
			IsShortPacket: true,
			ShortBody:     shortBody,
			RawPacket:     rawPacketBuffer.Bytes(),
		}
		return v, nil
	}

	if int(length) < headerSize+0xd {
		buffer.Next(int(length))
		return nil, ErrTooShortPacket
	}
	if buffer.Len() < int(length) {
		return nil, io.EOF
	}

	body := b[:int(length)]
	rawPacketBuffer.Write(body)
	buffer.Next(int(length))
	body = body[headerSize:]
	op := be.Uint32(body)
	body = body[4:]
	id := be.Uint64(body)
	body = body[8:]
	_, lenbytes := binary.Uvarint(body)
	if lenbytes <= 0 {
		return nil, fmt.Errorf("invalid message length %v", lenbytes)
	}
	if len(body) < lenbytes {
		return nil, fmt.Errorf("invalid message length %v %v", len(body), lenbytes)
	}
	body = body[lenbytes:]

	msg, err := NewMessage(bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	p := &GamePacket{
		At:        at,
		Sign:      sign,
		Length:    length,
		Flag:      flag,
		Op:        op,
		Id:        id,
		Msg:       msg,
		RawPacket: rawPacketBuffer.Bytes(),
	}
	return p, nil
}

func GamePacketBodyReader(r io.Reader) (uint32, uint64, Message, error) {
	b := make([]byte, 8)
	if _, err := io.ReadFull(r, b[:4]); err != nil {
		return 0, 0, nil, err
	}
	op := be.Uint32(b[:4])
	if _, err := io.ReadFull(r, b[:8]); err != nil {
		return 0, 0, nil, err
	}
	id := be.Uint64(b[:8])
	if _, _, err := util.ReadUvarint(r); err != nil {
		return 0, 0, nil, err
	}
	msg, err := NewMessage(r)
	if err != nil {
		return 0, 0, nil, err
	}
	return op, id, msg, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (t *GameServerPacketReader) GetStats() (uint32, uint32, uint32, uint32) {
	if t.handle != nil {
		if stats, err := t.handle.Stats(); err == nil {
			atomic.StoreUint32(&t.pcapDrops, uint32(stats.PacketsDropped))
		}
	}
	return atomic.LoadUint32(&t.pcapDrops),
		atomic.LoadUint32(&t.parserErrors),
		atomic.LoadUint32(&t.networkLoss),
		atomic.LoadUint32(&t.queueDrops)
}
