package pcaputil

import (
	"context"
	"errors"
	"log"
	"os"
	"time"

	"github.com/Marcentus/Midir/packet"
	"github.com/gopacket/gopacket/pcap"
)

var logger = log.New(os.Stdout, "pcaputil ", log.LstdFlags|log.Lshortfile)

// FindNic now accepts the BPF filter and ExitLag/Mudfish status to use when searching.
func FindNic(filter string, exitlagEnabled, mudfishEnabled bool) (string, error) {
	// 게임 서버 패킷이 수신되는 네트워크 인터페이스를 찾는다.
	packetWaitTime := time.Second * 5

	nics, err := pcap.FindAllDevs()
	if err != nil {
		logger.Println(err)
		return "", err
	}

	found := ""
	for _, nic := range nics {
		ctx, cancel := context.WithCancel(context.Background())

		// Pass the dynamic filter and mode flags to the packet reader
		r, err := packet.NewGameServerPacketReader(&packet.GameServerPacketReaderOpt{
			Ctx:            ctx,
			NicName:        nic.Name,
			Filter:         filter,
			ExitLagEnabled: exitlagEnabled,
			MudfishEnabled: mudfishEnabled,
		})

		if err != nil {
			logger.Println("findNic failed", err, nic.Name)
			cancel()
			continue
		}

		select {
		case <-time.After(packetWaitTime):
			logger.Println("findNic timeout", nic.Name)

		case <-r.PacketCh():
			found = nic.Name
			logger.Println("findNic success", nic.Name)
		}

		cancel()
		r.Close()
	}

	if found == "" {
		err := errors.New("findNic failed: not found")
		logger.Println(err)
		return "", err
	}

	logger.Println("findNic success:", found)
	return found, nil
}
