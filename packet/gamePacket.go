package packet

import (
	"io"
	"log"
	"os"
	"time"
)

var logger = log.New(os.Stdout, "packet ", log.LstdFlags|log.Lshortfile)

func ConfigureLoggerOutput(w io.Writer) {
	if w != nil {
		logger.SetOutput(w)
	}
}

type GamePacket struct {
	At     time.Time
	Sign   uint8
	Length uint32
	Flag   uint8

	// raw packet
	IsShortPacket bool
	ShortBody     []byte

	// normal packet
	Op  uint32
	Id  uint64
	Msg Message

	// checksum uint32

	RawPacket       []byte
	PacketDedupeKey string
	PacketFlowKey   string
}

const (
	OpCodeSystemError   = 0xFFFFFFFF
	OpCodeSystemWarning = 0xFFFFFFFE
)
