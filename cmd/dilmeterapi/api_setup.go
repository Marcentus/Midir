package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"sync"

	"github.com/Marcentus/Midir/packet"
	"github.com/go-chi/chi/v5"
	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"
	"github.com/gopacket/gopacket/pcap"
)

var activeExitlag bool
var activeMudfish bool
var activePromiscuous bool
var autodetectCancel context.CancelFunc
var autodetectMu sync.Mutex

const configFile = "settings.json"

type CaptureConfig struct {
	NicName     string `json:"nicName"`
	ExitLag     bool   `json:"exitlag"`
	Mudfish     bool   `json:"mudfish"`
	Promiscuous bool   `json:"promiscuous"`
}

func loadConfig() *CaptureConfig {
	data, err := os.ReadFile(configFile)
	if err != nil {
		return nil
	}
	var config CaptureConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil
	}
	return &config
}

func saveConfig(config *CaptureConfig) {
	data, err := json.MarshalIndent(config, "", "  ")
	if err == nil {
		os.WriteFile(configFile, data, 0644)
	}
}

func setupRouter() http.Handler {
	r := chi.NewRouter()

	r.Get("/status", func(w http.ResponseWriter, r *http.Request) {
		captureMu.Lock()
		defer captureMu.Unlock()

		nic := activeNicName
		exitlag := activeExitlag
		mudfish := activeMudfish
		promiscuous := activePromiscuous

		cfg := loadConfig()
		if cfg != nil {
			if !isCaptureRunning {
				nic = cfg.NicName
				exitlag = cfg.ExitLag
				mudfish = cfg.Mudfish
				promiscuous = cfg.Promiscuous
			}
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"is_running":  isCaptureRunning,
			"nic":         nic,
			"exitlag":     exitlag,
			"mudfish":     mudfish,
			"promiscuous": promiscuous,
		})
	})

	r.Get("/nics", func(w http.ResponseWriter, r *http.Request) {
		nics, err := pcap.FindAllDevs()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		type NicInfo struct {
			Name        string `json:"name"`
			Description string `json:"description"`
			IP          string `json:"ip"`
		}

		var result []NicInfo
		for _, nic := range nics {
			ipStr := ""
			if len(nic.Addresses) > 0 {
				ipStr = nic.Addresses[0].IP.String()
			}
			result = append(result, NicInfo{
				Name:        nic.Name,
				Description: nic.Description,
				IP:          ipStr,
			})
		}

		json.NewEncoder(w).Encode(result)
	})

	r.Post("/start", func(w http.ResponseWriter, req *http.Request) {
		var config CaptureConfig
		if err := json.NewDecoder(req.Body).Decode(&config); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if config.ExitLag && config.Mudfish {
			http.Error(w, "ExitLag and Mudfish modes are mutually exclusive", http.StatusBadRequest)
			return
		}

		// Save the requested settings permanently
		saveConfig(&config)

		filter := buildPcapFilter("", "", config.ExitLag, config.Mudfish)

		err := startPacketCapture(config.NicName, "", config.ExitLag, config.Mudfish, filter, config.Promiscuous, true)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	})

	r.Post("/restart-keep-session", func(w http.ResponseWriter, req *http.Request) {
		var config CaptureConfig
		if err := json.NewDecoder(req.Body).Decode(&config); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if config.ExitLag && config.Mudfish {
			http.Error(w, "ExitLag and Mudfish modes are mutually exclusive", http.StatusBadRequest)
			return
		}

		// Save settings, but preserve current aggregator/session data.
		saveConfig(&config)

		filter := buildPcapFilter("", "", config.ExitLag, config.Mudfish)

		err := startPacketCapture(config.NicName, "", config.ExitLag, config.Mudfish, filter, config.Promiscuous, false)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	})

	r.Post("/stop", func(w http.ResponseWriter, req *http.Request) {
		captureMu.Lock()
		defer captureMu.Unlock()
		stopPacketCaptureSync()
		w.WriteHeader(http.StatusOK)
	})

	r.Post("/autodetect", func(w http.ResponseWriter, req *http.Request) {
		var payload struct {
			NicName     string `json:"nicName"`
			Promiscuous bool   `json:"promiscuous"`
		}
		if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		err := startAutodetect(payload.NicName, payload.Promiscuous)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	r.Post("/autodetect/stop", func(w http.ResponseWriter, req *http.Request) {
		autodetectMu.Lock()
		defer autodetectMu.Unlock()
		if autodetectCancel != nil {
			autodetectCancel()
			autodetectCancel = nil
		}
		w.WriteHeader(http.StatusOK)
	})

	return r
}

func startAutodetect(nicName string, promiscuous bool) error {
	captureMu.Lock()
	stopPacketCaptureSync()
	captureMu.Unlock()

	autodetectMu.Lock()
	if autodetectCancel != nil {
		autodetectCancel()
		autodetectCancel = nil
	}
	ctx, cancel := context.WithCancel(context.Background())
	autodetectCancel = cancel
	autodetectMu.Unlock()

	handle, err := pcap.OpenLive(nicName, 65536, promiscuous, pcap.BlockForever)
	if err != nil {
		cancel()
		return err
	}

	if err := handle.SetBPFFilter("tcp"); err != nil {
		handle.Close()
		cancel()
		return err
	}

	go func() {
		defer handle.Close()

		packetSource := gopacket.NewPacketSource(handle, handle.LinkType())
		counts := make(map[string]int)
		targetCount := 5
		runOpcode := []byte{0x0f, 0x44, 0xbb, 0xa3}

		for {
			select {
			case <-ctx.Done():
				return
			case packet, ok := <-packetSource.Packets():
				if !ok {
					return
				}
				tcpLayer := packet.Layer(layers.LayerTypeTCP)
				ipLayer := packet.Layer(layers.LayerTypeIPv4)
				if tcpLayer != nil && ipLayer != nil {
					tcp, _ := tcpLayer.(*layers.TCP)
					ip, _ := ipLayer.(*layers.IPv4)

					if len(tcp.Payload) > 0 && bytes.Contains(tcp.Payload, runOpcode) {
						var remoteIP string
						var remotePort string

						if !ip.SrcIP.IsPrivate() && !ip.SrcIP.IsLoopback() {
							remoteIP = ip.SrcIP.String()
							remotePort = strconv.Itoa(int(tcp.SrcPort))
						} else if !ip.DstIP.IsPrivate() && !ip.DstIP.IsLoopback() {
							remoteIP = ip.DstIP.String()
							remotePort = strconv.Itoa(int(tcp.DstPort))
						}

						if remoteIP == "" {
							continue
						}

						key := fmt.Sprintf("%s:%s", remoteIP, remotePort)
						counts[key]++

						if globalPub != nil {
							globalPub.Broadcast("autodetect_progress", map[string]interface{}{
								"current": counts[key],
								"target":  targetCount,
							})
						}

						if counts[key] >= targetCount {
							if globalPub != nil {
								globalPub.Broadcast("autodetect_done", map[string]interface{}{
									"ip":   remoteIP,
									"port": remotePort,
								})
							}
							return
						}
					}
				}
			}
		}
	}()
	return nil
}

// stopPacketCaptureSync must be called with captureMu locked!
func stopPacketCaptureSync() {
	if cancelCapture != nil {
		cancelCapture()
		// Abruptly closing the reader ensures Npcap returns io.EOF breaking the inner read loop
		if globalReader != nil {
			globalReader.Close()
		}
		// Wait safely until the goroutine emits completion to guarantee no handles overlap.
		if captureDone != nil {
			<-captureDone
		}
		cancelCapture = nil
		captureDone = nil
		globalReader = nil
	}
	isCaptureRunning = false
	activeNicName = ""
	activeExitlag = false
	activeMudfish = false
	activePromiscuous = false
}

func startPacketCapture(nicName string, fileName string, exitlagEnabled, mudfishEnabled bool, filter string, promiscuous bool, clearSession bool) error {
	captureMu.Lock()
	defer captureMu.Unlock()

	stopPacketCaptureSync()

	if clearSession && globalPub != nil {
		globalPub.ClearCache()
		// Ensure old session cache drops out so users start fresh locally
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancelCapture = cancel
	captureDone = make(chan struct{})

	r, err := packet.NewGameServerPacketReader(&packet.GameServerPacketReaderOpt{
		Ctx:             ctx,
		NicName:         nicName,
		FileName:        fileName,
		Sm:              globalSm,
		ExitLagEnabled:  exitlagEnabled,
		MudfishEnabled:  mudfishEnabled,
		Filter:          filter,
		PromiscuousMode: promiscuous,
	})

	if err != nil {
		cancel()
		cancelCapture = nil
		close(captureDone)
		captureDone = nil
		isCaptureRunning = false
		return err
	}

	globalReader = r
	isCaptureRunning = true
	activeNicName = nicName
	activeExitlag = exitlagEnabled
	activeMudfish = mudfishEnabled
	activePromiscuous = promiscuous

	go func() {
		defer close(captureDone) // Critical: Always signal done whenever goroutine exits!
		defer r.Close()

		for p := range r.PacketCh() {
			select {
			case <-ctx.Done():
				return
			case globalPacketCh <- p:
			}
		}
	}()

	return nil
}
