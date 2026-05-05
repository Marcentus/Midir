package packet

import (
	"log"
	"os"
	"time"
)

var logger = log.New(os.Stdout, "packet ", log.LstdFlags|log.Lshortfile)

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

	RawPacket []byte
}

const (
	OpCodeSystemError   = 0xFFFFFFFF
	OpCodeSystemWarning = 0xFFFFFFFE
)
