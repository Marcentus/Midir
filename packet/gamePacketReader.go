package packet

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
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

	// mutable
	handle *pcap.Handle
	fd     *os.File
}

type GameServerPacketReaderOpt struct {
	Ctx             context.Context
	FileName        string
	NicName         string
	ClientIp        string
	Sm              pcapWriter
	ExitLagEnabled  bool   // New: To control ExitLag processing
	Filter          string // New: To pass in the dynamic BPF filter
	PromiscuousMode bool   // New: To control Promiscuous capture
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
}

type gamePacketAssemblerState struct {
	buffer     *bytes.Buffer
	lastRelSeq uint32
	lastAt     time.Time
	payloads   []gamePacketPayload
}

const pcapQueueSize = 100
const pcapBufferSize = 32 * 1024 * 1024
const packetQueueSize = 100

var ErrTooShortPacket = errors.New("too short packet")

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

					// Silently discard ExitLag keep-alive/heartbeat packets (under 6 bytes)
					if len(mabiPayload) < 6 {
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
		logger.Println("ExitLag mode disabled. Bypassing ExitLag stripper.")
		// If ExitLag is disabled, bypass the stripping loop.
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
						logger.Println("Packet channel full, dropped error packet.")
					}
				}
			}()

			assemblers := make(map[tcpFlowKey]*gamePacketAssemblerState)

			getAssembler := func(key tcpFlowKey) *gamePacketAssemblerState {
				if key == "" {
					key = "default"
				}
				st := assemblers[key]
				if st == nil {
					st = &gamePacketAssemblerState{
						buffer:   bytes.NewBuffer(nil),
						lastAt:   time.Now(),
						payloads: make([]gamePacketPayload, 0, 100),
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
					pushPayload(st, payloadData)
				}

			readerLoop:
				for {
					msg, err := ParseGamePacket(st.buffer, st.lastAt)
					if err != nil {
						if err == io.EOF {
							break readerLoop
						}
						logger.Printf("game packet parse error %v %v", st.lastRelSeq, err)
						nextPayload(st)
						continue
					}
					if msg != nil {
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
	handle, err := pcap.OpenLive(nic, pcapBufferSize, promiscuous, pcap.BlockForever)
	if err != nil {
		logger.Println(err)
		return nil, err
	}
	t.handle = handle

	if err := handle.SetBPFFilter(filter); err != nil {
		return nil, err
	}

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

func (t *GameServerPacketReader) readPacketLoop(ch chan<- gamePacketPayload) {
	defer close(ch)
	ethLayer := layers.Ethernet{}
	ip4Layer := layers.IPv4{}
	tcpLayer := layers.TCP{}
	payload := gopacket.Payload{}

	layerParser := gopacket.NewDecodingLayerParser(layers.LayerTypeEthernet, &ethLayer, &ip4Layer, &tcpLayer, &payload)
	packetLayers := []gopacket.LayerType(nil)
	streams := make(map[tcpFlowKey]*tcpStreamState)
	const streamTTL = 30 * time.Second
	// ExitLag dynamic mode still needs a direction guard. We first identify the
	// game client LAN IP from a public relay -> private client packet, then allow
	// any TCP source inbound to that client. This keeps local/private ExitLag
	// metadata streams while excluding client -> relay traffic that corrupts
	// Mabinogi packet framing.
	var exitLagClientIP string

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

		if err := layerParser.DecodeLayers(b, &packetLayers); err != nil {
			continue
		}

		hasIPv4 := false
		for _, layer := range packetLayers {
			if layer == layers.LayerTypeIPv4 {
				hasIPv4 = true
				break
			}
		}
		if !hasIPv4 {
			continue
		}

		for _, layer := range packetLayers {
			if layer != layers.LayerTypeTCP || len(tcpLayer.Payload) < 1 {
				continue
			}

			if exitLagClientIP == "" {
				if ip4Layer.SrcIP.IsPrivate() || ip4Layer.SrcIP.IsLoopback() || !ip4Layer.DstIP.IsPrivate() {
					continue
				}
				exitLagClientIP = ip4Layer.DstIP.String()
				logger.Printf("[ExitLag Dynamic] following inbound traffic to game client %s", exitLagClientIP)
			} else if ip4Layer.DstIP.String() != exitLagClientIP {
				continue
			}

			key := flowKeyFor(ip4Layer, tcpLayer)
			for streamKey, st := range streams {
				if !st.lastSeen.IsZero() && ci.Timestamp.Sub(st.lastSeen) > streamTTL {
					delete(streams, streamKey)
				}
			}

			st := streams[key]
			if st == nil {
				st = &tcpStreamState{
					baseSeq:  tcpLayer.Seq,
					nextSeq:  tcpLayer.Seq,
					pending:  make([]pendingTcpLayer, 0, packetQueueSize),
					lastSeen: ci.Timestamp,
				}
				streams[key] = st
			}
			st.lastSeen = ci.Timestamp

			if st.nextSeq != 0 && tcpLayer.Seq != st.nextSeq {
				if tcpLayer.Seq < st.nextSeq {
					if tcpLayer.Seq+uint32(len(tcpLayer.Payload)) >= st.nextSeq {
						payload := tcpLayer.Payload[st.nextSeq-tcpLayer.Seq:]
						if len(payload) > 0 {
							ch <- gamePacketPayload{
								relSeq:  st.nextSeq - st.baseSeq,
								data:    payload,
								at:      ci.Timestamp,
								flowKey: key,
							}
						}
						st.nextSeq = tcpLayer.Seq + uint32(len(tcpLayer.Payload))
					}
					continue
				}

				st.pending = append(st.pending, pendingTcpLayer{
					tcpLayer: tcpLayer,
					ci:       ci,
				})

				if len(st.pending) > 10 {
					minIdx := findMinSeqIdx(st.pending)
					if minIdx != -1 {
						targetLayer := st.pending[minIdx]
						skippedBytes := targetLayer.tcpLayer.Seq - st.nextSeq
						warningMsg := fmt.Sprintf("Network packet loss detected on %s (Gap: %d bytes). Skipping to resume.", key, skippedBytes)
						logger.Printf("[TCP Recovery] %s OldNext: %d, NewNext: %d", warningMsg, st.nextSeq, targetLayer.tcpLayer.Seq)
						st.nextSeq = targetLayer.tcpLayer.Seq

						select {
						case t.packetCh <- &GamePacket{
							At:        time.Now(),
							Op:        OpCodeSystemWarning,
							RawPacket: []byte(warningMsg),
						}:
						default:
							logger.Println("Packet channel full, dropped warning packet.")
						}
					}
				}

				drainReadyPending(st, key)
				continue
			}

			ch <- gamePacketPayload{
				relSeq:  tcpLayer.Seq - st.baseSeq,
				data:    tcpLayer.Payload,
				at:      ci.Timestamp,
				flowKey: key,
			}
			st.nextSeq = tcpLayer.Seq + uint32(len(tcpLayer.Payload))
			drainReadyPending(st, key)
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
