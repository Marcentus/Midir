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

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"
	"github.com/gopacket/gopacket/pcap"
	"github.com/Marcentus/Midir/util"
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

type gamePacketPayload struct {
	relSeq uint32
	data   []byte
	at     time.Time
}

type pendingTcpLayer struct {
	tcpLayer layers.TCP
	ci       gopacket.CaptureInfo
}

const pcapQueueSize = 100
const pcapBufferSize = 32 * 1024 * 1024
const packetQueueSize = 100

var ErrTooShortPacket = errors.New("too short packet")

// This function is provided for completeness and is unchanged from the original file.
func (t *GameServerPacketReader) exitLagPacketLoop(rawPayloadCh <-chan gamePacketPayload, mabiPayloadCh chan<- gamePacketPayload) {
	buffer := bytes.NewBuffer(nil)
	var isReadingMabiPayload bool
	var mabiBytesToRead uint32
	var packetStartTime time.Time
	var packetRelSeq uint32

	for {
		select {
		case <-t.ctx.Done():
			return
		case payloadData, ok := <-rawPayloadCh:
			if !ok {
				return // Channel is closed
			}
			buffer.Write(payloadData.data)
			if !isReadingMabiPayload {
				packetStartTime = payloadData.at
				packetRelSeq = payloadData.relSeq
			}
		}

	parseLoop:
		for {
			if !isReadingMabiPayload {
				const headerSignatureLen = 5
				foundHeader := false
				for buffer.Len() >= headerSignatureLen {
					if buffer.Bytes()[0] == 0x01 && buffer.Bytes()[4] == 0x05 {
						foundHeader = true
						break
					}
					buffer.Next(1)
				}

				if !foundHeader {
					break parseLoop
				}
			}

			if isReadingMabiPayload {
				if uint32(buffer.Len()) < mabiBytesToRead {
					break parseLoop
				}

				mabiPayload := make([]byte, mabiBytesToRead)
				_, err := buffer.Read(mabiPayload)
				if err != nil {
					logger.Println("ExitLag Buffer Read Error:", err)
					isReadingMabiPayload = false
					continue parseLoop
				}

				mabiPayloadCh <- gamePacketPayload{
					relSeq: packetRelSeq,
					at:     packetStartTime,
					data:   mabiPayload,
				}

				if buffer.Len() >= 4 {
					if bytes.Equal(buffer.Bytes()[:4], []byte{0x05, 0x25, 0x01, 0x01}) {
						buffer.Next(4)
					}
				}

				isReadingMabiPayload = false
				continue parseLoop

			} else {
				b := buffer.Bytes()
				const minHeaderSize = 38
				if len(b) < minHeaderSize {
					break parseLoop
				}

				seqLenIndOffset := 1 + 2 + 30
				seqLenInd := b[seqLenIndOffset]

				if seqLenInd == 0 {
					bodyLen := le.Uint16(b[1:3])
					totalSize := 1 + 2 + int(bodyLen)

					if buffer.Len() < totalSize {
						break parseLoop
					}

					if buffer.Len() >= totalSize+4 && bytes.Equal(b[totalSize:totalSize+4], []byte{0x05, 0x25, 0x01, 0x01}) {
						buffer.Next(totalSize + 4)
					} else {
						buffer.Next(totalSize)
					}

					// logger.Println("ExitLag Info: Skipped control packet.")
					continue parseLoop
				}

				seqLen := int(seqLenInd)
				if seqLen <= 0 || seqLen > 8 {
					logger.Printf("ExitLag Parse Error: Unreasonable sequence length: %d", seqLen)
					buffer.Next(1)
					continue parseLoop
				}

				payloadLenIndOffset := seqLenIndOffset + 1 + seqLen
				if buffer.Len() < payloadLenIndOffset+1 {
					break parseLoop
				}
				payloadLenInd := b[payloadLenIndOffset]

				var mabiPayloadLenBytes int
				if payloadLenInd == 0x05 {
					mabiPayloadLenBytes = 1
				} else if payloadLenInd == 0x09 {
					mabiPayloadLenBytes = 2
				} else {
					logger.Printf("ExitLag Parse Error: Invalid Mabinogi payload length indicator: 0x%x", payloadLenInd)
					buffer.Next(1)
					continue parseLoop
				}

				flagOffset := payloadLenIndOffset + 1
				if buffer.Len() < flagOffset+1 {
					break parseLoop
				}
				flag := b[flagOffset]

				if flag != 0x23 {
					if flag != 0x2b {
						logger.Printf("ExitLag Info: Skipping non-game packet ( Unknown flag: 0x%x)", flag)
					}
					buffer.Next(1)
					continue parseLoop
				}

				mabiLenOffset := flagOffset + 1
				if buffer.Len() < mabiLenOffset+mabiPayloadLenBytes {
					break parseLoop
				}
				mabiLen := uint32(0)
				if mabiPayloadLenBytes == 1 {
					mabiLen = uint32(b[mabiLenOffset])
				} else {
					mabiLen = uint32(le.Uint16(b[mabiLenOffset : mabiLenOffset+2]))
				}

				headerTotalLen := mabiLenOffset + mabiPayloadLenBytes
				buffer.Next(headerTotalLen)
				isReadingMabiPayload = true
				mabiBytesToRead = mabiLen
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

			buffer := bytes.NewBuffer(nil)
			lastRelSeq, lastAt := uint32(0), time.Now()
			payloads := make([]gamePacketPayload, 0, 100)

			skipPayload := func(n int) {
				for n > 0 {
					if len(payloads) == 0 {
						return
					}
					if n < len(payloads[0].data) {
						lastRelSeq, lastAt = payloads[0].relSeq, payloads[0].at
						payloads[0].data = payloads[0].data[n:]
						return
					}
					n -= len(payloads[0].data)
					if len(payloads) > 0 {
						lastRelSeq, lastAt = payloads[0].relSeq, payloads[0].at
						payloads = payloads[1:]
					}
				}
			}

			nextPayload := func() {
				buffer.Reset()
				if len(payloads) < 1 {
					return
				}
				payloads = payloads[1:]
				if len(payloads) < 1 {
					return
				}
				for _, v := range payloads {
					buffer.Write(v.data)
				}
				if len(payloads) > 0 {
					lastRelSeq = payloads[0].relSeq
				}
			}

			pushPayload := func(payloadData gamePacketPayload) {
				if buffer.Len() < 1 {
					buffer.Reset()
				}
				if len(payloads) < 1 {
					lastRelSeq, lastAt = payloadData.relSeq, payloadData.at
				}
				payloads = append(payloads, payloadData)
				buffer.Write(payloadData.data)
			}

			// Inner processing loop
			for {
				select {
				case <-t.ctx.Done():
					return
				case payloadData, ok := <-payloadCh:
					if !ok {
						return
					}
					pushPayload(payloadData)
				}

			readerLoop:
				for {
					msg, err := ParseGamePacket(buffer, lastAt)
					if err != nil {
						if err == io.EOF {
							break readerLoop
						}
						logger.Printf("game packet parse error %v %v", lastRelSeq, err)
						nextPayload()
						continue
					}
					if msg != nil {
						t.packetCh <- msg
						skipPayload(len(msg.RawPacket))
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

	baseSeq := uint32(0)
	nextSeq, prevDstPort := uint32(0), layers.TCPPort(0)
	pendingTcpLayers := make([]pendingTcpLayer, 0, packetQueueSize)

	// Helper to find the index of the pending layer with the smallest sequence number
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

		if i == 0 {
			baseSeq = tcpLayer.Seq
		}

		for _, layer := range packetLayers {
			if layer != layers.LayerTypeTCP || len(tcpLayer.Payload) < 1 {
				continue
			}

			if nextSeq != 0 && tcpLayer.Seq != nextSeq {
				if prevDstPort != tcpLayer.DstPort {
					// Port change logic (unchanged)
					for _, v := range pendingTcpLayers {
						ch <- gamePacketPayload{
							relSeq: v.tcpLayer.Seq - baseSeq,
							data:   v.tcpLayer.Payload,
							at:     v.ci.Timestamp,
						}
					}
					pendingTcpLayers = pendingTcpLayers[:0]
					prevDstPort = tcpLayer.DstPort
					baseSeq = tcpLayer.Seq
					nextSeq = tcpLayer.Seq + uint32(len(tcpLayer.Payload))
					if len(tcpLayer.Payload) == 4 {
						continue
					}
					ch <- gamePacketPayload{
						relSeq: tcpLayer.Seq - baseSeq,
						data:   tcpLayer.Payload,
						at:     ci.Timestamp,
					}
					continue
				}

				// Check for already processed (duplicate/retransmit)
				if tcpLayer.Seq < nextSeq {
					if tcpLayer.Seq+uint32(len(tcpLayer.Payload)) >= nextSeq {
						payload := tcpLayer.Payload[nextSeq-tcpLayer.Seq:]
						if len(payload) > 0 {
							ch <- gamePacketPayload{
								relSeq: nextSeq - baseSeq,
								data:   payload,
								at:     ci.Timestamp,
							}
						}
						nextSeq = tcpLayer.Seq + uint32(len(tcpLayer.Payload))
					}
					continue
				}

				// GAP DETECTED: Buffer check
				pendingTcpLayers = append(pendingTcpLayers, pendingTcpLayer{
					tcpLayer: tcpLayer,
					ci:       ci,
				})

				// If buffer is too large, force skip
				if len(pendingTcpLayers) > 10 {
					minIdx := findMinSeqIdx(pendingTcpLayers)
					if minIdx != -1 {
						targetLayer := pendingTcpLayers[minIdx]

						// Calculate skipped amount
						skippedBytes := targetLayer.tcpLayer.Seq - nextSeq
						warningMsg := fmt.Sprintf("Network packet loss detected (Gap: %d bytes). Skipping to resume.", skippedBytes)
						logger.Printf("[TCP Recovery] %s OldNext: %d, NewNext: %d", warningMsg, nextSeq, targetLayer.tcpLayer.Seq)

						// Update Expected Sequence
						nextSeq = targetLayer.tcpLayer.Seq

						// Inject System Warning
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

				// Check if the newly updated nextSeq matches any pending layers (including the one we just added or picked)
				// We need to loop because popping one might reveal the next one is also ready.
				// Note: The original code only checked pendingTcpLayers[0], which assumed order.
				// Since we are now potentially skipping, we should scan or sort.
				// For simplicity/performance, we'll iterate and find if ANY match nextSeq.
				// Ideally, pendingTcpLayers should be a priority queue, but for small N (10) a linear scan is fine.

				for {
					foundIdx := -1
					for idx, p := range pendingTcpLayers {
						if p.tcpLayer.Seq == nextSeq {
							foundIdx = idx
							break
						}
					}

					if foundIdx != -1 {
						v := pendingTcpLayers[foundIdx]
						ch <- gamePacketPayload{
							relSeq: v.tcpLayer.Seq - baseSeq,
							data:   v.tcpLayer.Payload,
							at:     v.ci.Timestamp,
						}
						nextSeq = v.tcpLayer.Seq + uint32(len(v.tcpLayer.Payload))

						// Remove from slice (order doesn't strictly matter for the rest, but let's keep it clean)
						pendingTcpLayers[foundIdx] = pendingTcpLayers[len(pendingTcpLayers)-1] // swap with last
						pendingTcpLayers = pendingTcpLayers[:len(pendingTcpLayers)-1]
						// Continue loop to see if nextSeq is also present
					} else {
						break
					}
				}
				continue
			}

			ch <- gamePacketPayload{
				relSeq: tcpLayer.Seq - baseSeq,
				data:   tcpLayer.Payload,
				at:     ci.Timestamp,
			}
			nextSeq = tcpLayer.Seq + uint32(len(tcpLayer.Payload))
			prevDstPort = tcpLayer.DstPort

			// Check pending logic again (copy-paste of the loop abovish, refactored)
			for {
				foundIdx := -1
				for idx, p := range pendingTcpLayers {
					if p.tcpLayer.Seq == nextSeq {
						foundIdx = idx
						break
					}
				}

				if foundIdx != -1 {
					v := pendingTcpLayers[foundIdx]
					ch <- gamePacketPayload{
						relSeq: v.tcpLayer.Seq - baseSeq,
						data:   v.tcpLayer.Payload,
						at:     v.ci.Timestamp,
					}
					nextSeq = v.tcpLayer.Seq + uint32(len(v.tcpLayer.Payload))
					pendingTcpLayers[foundIdx] = pendingTcpLayers[len(pendingTcpLayers)-1]
					pendingTcpLayers = pendingTcpLayers[:len(pendingTcpLayers)-1]
				} else {
					break
				}
			}
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
