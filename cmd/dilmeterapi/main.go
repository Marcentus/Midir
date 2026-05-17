package main

import (
	"context"
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/Marcentus/Midir/constants"
	"github.com/Marcentus/Midir/packet"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/gopacket/gopacket/pcap"
	"golang.org/x/net/websocket"
)

const port = 8030

var playerCache = NewPlayerCache("player_cache.json")

// --- CORRECTED EMBEDS ---
// Embed the compiled Vue app (UI)
//
//go:embed static
var staticFiles embed.FS

// Embed our new game data (JSON files, images)
//
//go:embed static_data
var staticData embed.FS

// --- END CORRECTION ---

var logger = log.New(os.Stdout, "dilmeterapi ", log.LstdFlags|log.Lshortfile)

var globalPacketCh = make(chan *packet.GamePacket, 1000)
var cancelCapture context.CancelFunc
var captureMu sync.Mutex
var isCaptureRunning bool
var activeNicName string
var activePcapRecord bool
var globalPub *eventPublisher
var globalSm *SessionManager
var globalReader *packet.GameServerPacketReader
var captureDone chan struct{}

func buildPcapFilter(ips, ports string, exitlagEnabled bool) string {
	if exitlagEnabled {
		// ExitLag can change both relay IP and relay port mid-session. In ExitLag
		// mode we capture all TCP on the selected interface, then follow valid
		// game streams in the Go parser instead of binding libpcap to one socket.
		return "tcp"
	}

	ipList := constants.DefaultGameserverNet
	if ips != "" {
		ipList = strings.Split(ips, ",")
	}
	portList := constants.DefaultGameserverPort
	if ports != "" {
		portList = strings.Split(ports, ",")
	}
	filter := fmt.Sprint("tcp and src net ( ", strings.Join(ipList, " or "), " )")
	filter += fmt.Sprint(" and src port ( ", strings.Join(portList, " or "), " ) ")
	return filter
}

func main() {
	exitlag := flag.Bool("exitlag", false, "Enable if you are using ExitLag.")
	ip := flag.String("ip", "", "Comma-separated list of game server IPs to capture from.")
	portFlag := flag.String("port", "", "Comma-separated list of game server ports to capture from.")
	recordPcap := flag.Bool("record-pcap", false, "Enable to record raw packet capture (.pcapng) files for sessions.")
	flag.Parse()

	playerCache.Load()

	loadRaceNames()
	loadTalentData()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		logger.Println("Shutdown signal received, saving player cache...")
		playerCache.Save()
		os.Exit(0)
	}()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mode := flag.Arg(0)
	logger.Println("* Midir", mode)
	pcapFilter := buildPcapFilter(*ip, *portFlag, *exitlag)

	switch mode {
	// ... (the entire switch block remains unchanged)
	case "list":
		nics, err := pcap.FindAllDevs()
		if err != nil {
			messagebox(fmt.Sprintf("FindAllDevs failed: %v", err))
			logger.Fatalln("FindAllDevs failed:", err)
		}
		sb := strings.Builder{}
		for i, nic := range nics {
			ipStr := "unknownAddress"
			if len(nic.Addresses) > 0 {
				ipStr = nic.Addresses[0].IP.String()
			}
			sb.WriteString(fmt.Sprintln("* nic", i, "name:", nic.Name, "ip:", ipStr))
		}
		s := sb.String()
		messagebox(s)
		logger.Println(s)
		return
	case "file":
		fileName := flag.Arg(1)
		if fileName == "" {
			logger.Fatalln("file mode requires a filename")
		}
		run(ctx, "", fileName, *exitlag, pcapFilter, *recordPcap)
	case "":
		logger.Println("Starting web server without initial packet capture config...")
		run(ctx, "", "", *exitlag, pcapFilter, *recordPcap)
	default:
		_, err := os.Stat(mode)
		fileExists := err == nil
		nicName, fileName := "", ""
		if fileExists {
			fileName = mode
		} else {
			nicName = mode
		}
		run(ctx, nicName, fileName, *exitlag, pcapFilter, *recordPcap)
	}

	for {
		time.Sleep(1 * time.Second)
	}
}

func run(ctx context.Context, nicName string, fileName string, exitlagEnabled bool, filter string, recordPcap bool) {
	activePcapRecord = recordPcap

	sm, err := NewSessionManager("logs", recordPcap)
	if err != nil {
		messagebox(fmt.Sprintf("NewSessionManager failed: %v", err))
		logger.Fatalln("NewSessionManager failed:", err)
	}
	globalSm = sm

	if _, err := sm.StartLiveSession(); err != nil {
		messagebox(fmt.Sprintf("Failed to start live session: %v", err))
		logger.Fatalln("Failed to start live session:", err)
	}

	logger.Println("Initializing Database...")
	InitDB()

	isLive := fileName == ""
	pub := newEventPublisher(ctx, globalPacketCh, sm, isLive)
	globalPub = pub
	playerCache.OnPlayerUpdate = pub.QueuePlayerUpdate

	if nicName != "" || fileName != "" {
		err := startPacketCapture(nicName, fileName, exitlagEnabled, filter, true)
		if err != nil {
			logger.Println("Failed to start capture from CLI arguments:", err)
		}
	} else {
		// Do not auto-start capture from saved settings. Starting capture can clear
		// live data, request elevated packet capture, and in ExitLag mode may need
		// fresh routing state. Let the Web UI start capture explicitly.
		if cfg := loadConfig(); cfg != nil && cfg.NicName != "" {
			logger.Printf("Found saved capture configuration for NIC %s; waiting for manual Start Capture.", cfg.NicName)
		} else {
			logger.Println("No saved capture configuration. Web UI setup required.")
		}
	}

	startWebServer(pub, sm)
	switch runtime.GOOS {
	case "windows":
		go exec.Command("explorer", fmt.Sprintf("http://127.0.0.1:%v", port)).Run()
	case "darwin":
		go exec.Command("open", fmt.Sprintf("http://127.0.0.1:%v", port)).Run()
	}
}

// Replace the existing startWebServer with this corrected version
func startWebServer(pub *eventPublisher, sm *SessionManager) {
	r := chi.NewRouter()
	r.Use(middleware.RequestID, middleware.RealIP, middleware.Recoverer)

	// API handler for our embedded JSON data (races.json, etc.)
	r.Mount("/api/data", dataRouter())

	// Fileserver for our embedded images
	imagesFS, err := fs.Sub(staticData, "static_data/images")
	if err != nil {
		logger.Fatal(err)
	}
	r.Handle("/images/*", http.StripPrefix("/images/", http.FileServer(http.FS(imagesFS))))

	// WebSocket handler
	r.Handle("/ws", websocket.Handler(func(ws *websocket.Conn) {
		wsCtx, wsCtxCancel := context.WithCancel(ws.Request().Context())
		defer wsCtxCancel()
		ch := make(chan []byte, 1000000)
		defer close(ch)
		go pub.addClient(wsCtx, ch)
		go func() {
			defer wsCtxCancel()
			// var msg string
			var msg struct {
				Type  string      `json:"type"`
				Index uint32      `json:"index,omitempty"`
				Data  interface{} `json:"data,omitempty"`
			}
			for {
				if err := websocket.JSON.Receive(ws, &msg); err != nil {
					ws.Close()
					return
				}
			}
		}()
		for {
			select {
			case <-wsCtx.Done():
				return
			case e := <-ch:
				if err := websocket.Message.Send(ws, string(e)); err != nil {
					return
				}
			}
		}
	}))

	r.Mount("/api/sessions", sessionRouter(sm))
	r.Mount("/api/state", stateRouter(pub, sm))
	r.Mount("/api/setup", setupRouter())

	// --- CORRECTED STATIC FILE HANDLER ---
	// This serves the compiled Vue app from the 'static' directory.
	uiFS, err := fs.Sub(staticFiles, "static")
	if err != nil {
		logger.Fatal(err)
	}
	r.Handle("/*", http.FileServer(http.FS(uiFS)))
	// --- END CORRECTION ---

	logger.Printf("Server listening on port %d", port)

	// Print preferred local network IP for other PCs to connect
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err == nil {
		defer conn.Close()
		localAddr := conn.LocalAddr().(*net.UDPAddr)
		logger.Printf("To access the meter from another device on your network, use this link:")
		logger.Printf("  -> http://%s:%d", localAddr.IP.String(), port)
	}
	go func() {
		err := http.ListenAndServe(fmt.Sprintf("0.0.0.0:%d", port), r)
		if err != nil {
			messagebox(fmt.Sprintf("ListenAndServe failed: %v", err))
			logger.Fatalln(err)
		}
	}()
	<-time.After(1 * time.Second)
}
