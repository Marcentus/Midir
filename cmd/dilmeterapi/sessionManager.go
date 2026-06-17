package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Marcentus/Midir/packet"
	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"
	"github.com/gopacket/gopacket/pcapgo"
)

const liveSessionFilename = "live-session.ndjson"
const livePcapFilename = "live-session.pcapng"

type SessionSummaryPlayer struct {
	Name        string  `json:"name"`
	DPS         float32 `json:"dps"`
	ArcanaName  string  `json:"arcanaName"`
	ArcanaIcon  string  `json:"arcanaIcon,omitempty"`
	TotalDamage float32 `json:"totalDamage"`
}

type SessionSummaryEnemy struct {
	Name        string  `json:"name"`
	RaceID      uint32  `json:"raceId"`
	TotalDamage float32 `json:"totalDamage"`
}

type SessionSummaryData struct {
	Name        string                 `json:"name"`
	StartTime   int64                  `json:"startTime"`
	Duration    float64                `json:"duration"`
	TotalDamage float32                `json:"totalDamage"`
	Players     []SessionSummaryPlayer `json:"players"`
	Enemies     []SessionSummaryEnemy  `json:"enemies"`
}

type Session struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	StartTime int64  `json:"startTime"`
	EndTime   int64  `json:"endTime,omitempty"`

	Summary *SessionSummaryData `json:"summary,omitempty"`

	ndjsonFile *os.File       `json:"-"`
	pcapWriter *pcapgo.Writer `json:"-"`
	pcapFile   *os.File       `json:"-"`
	events     []iEvent
}

type SessionManager struct {
	logDirectory   string
	currentSession *Session
	mu             sync.RWMutex
	recordPcap     bool
}

func NewSessionManager(logDir string, recordPcap bool) (*SessionManager, error) {
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("could not create log directory: %w", err)
	}

	sm := &SessionManager{
		logDirectory: logDir,
		recordPcap:   recordPcap,
	}

	if err := sm.runMigration(); err != nil {
		logger.Printf("Could not run migration for old session files: %v", err)
	}

	return sm, nil
}

// StartLiveSession creates or overwrites the temporary live session file.
func (sm *SessionManager) StartLiveSession() (*Session, error) {
	// logger.Println("[Locking] sessionManager.StartLiveSession attempting to lock...")
	sm.mu.Lock()
	// logger.Println("...[Locked] sessionManager.StartLiveSession acquired lock.")
	defer func() {
		// logger.Println("[Unlocking] sessionManager.StartLiveSession attempting to unlock.")
		sm.mu.Unlock()
		// logger.Println("...[Unlocked] sessionManager.StartLiveSession released lock.")
	}()

	if sm.currentSession != nil {
		return nil, fmt.Errorf("a session is already active")
	}

	now := time.Now()
	s := &Session{
		ID:        liveSessionFilename,
		Name:      "Live Session",
		StartTime: now.Unix(),
		events:    make([]iEvent, 0, 1000),
	}

	if sm.recordPcap {
		pcapPath := filepath.Join(sm.logDirectory, livePcapFilename)
		pcapFile, err := os.OpenFile(pcapPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
		if err != nil {
			return nil, fmt.Errorf("failed to open pcapng log file: %w", err)
		}
		s.pcapFile = pcapFile
		pcapWriter := pcapgo.NewWriter(pcapFile)
		if err := pcapWriter.WriteFileHeader(65536, layers.LinkTypeEthernet); err != nil {
			pcapFile.Close()
			return nil, fmt.Errorf("failed to write pcapng header: %w", err)
		}
		s.pcapWriter = pcapWriter
	}

	sm.currentSession = s
	return s, nil
}

// StopCurrentSession closes the handles on the current session (likely the live one).
func (sm *SessionManager) StopCurrentSession() {
	// logger.Println("[Locking] sessionManager.StopCurrentSession attempting to lock...")
	sm.mu.Lock()
	// logger.Println("...[Locked] sessionManager.StopCurrentSession acquired lock.")
	defer func() {
		// logger.Println("[Unlocking] sessionManager.StopCurrentSession attempting to unlock.")
		sm.mu.Unlock()
		// logger.Println("...[Unlocked] sessionManager.StopCurrentSession released lock.")
	}()

	if sm.currentSession == nil {
		return
	}

	if sm.currentSession.ndjsonFile != nil {
		sm.currentSession.ndjsonFile.Close()
	}
	if sm.currentSession.pcapFile != nil {
		sm.currentSession.pcapFile.Close()
	}

	sm.currentSession = nil
}

// SaveLiveSession renames the live session file to a permanent one, writing cached in-memory events.
func (sm *SessionManager) SaveLiveSession(name string) error {
	sm.mu.Lock()
	defer func() {
		sm.mu.Unlock()
	}()

	if sm.currentSession == nil || sm.currentSession.ID != liveSessionFilename {
		return fmt.Errorf("no active live session to save")
	}

	// Close PCAP file handle if open
	if sm.currentSession.pcapFile != nil {
		sm.currentSession.pcapFile.Close()
	}

	newNdjsonFilename := fmt.Sprintf("session-%d.ndjson", time.Now().UnixMilli())
	newPath := filepath.Join(sm.logDirectory, newNdjsonFilename)

	var summary *FightSummary
	var err error
	if globalPub != nil && globalPub.aggregator != nil {
		inMemSummary := globalPub.aggregator.GetSummary()
		summary = &inMemSummary
	} else {
		// Fallback (for tests/CLI)
		oldPath := filepath.Join(sm.logDirectory, liveSessionFilename)
		summary, err = GenerateSummaryFromFile(oldPath)
	}

	var summaryData SessionSummaryData
	summaryData.Name = name
	summaryData.StartTime = sm.currentSession.StartTime

	if err == nil && summary != nil {
		summaryData.Duration = summary.EncounterDuration
		summaryData.TotalDamage = summary.TotalDamage

		targetDamageMap := make(map[string]float32)

		for _, pstat := range summary.Players {
			summaryData.Players = append(summaryData.Players, SessionSummaryPlayer{
				Name:        pstat.Name,
				DPS:         pstat.OverallStats.DPS,
				ArcanaName:  pstat.TalentName,
				ArcanaIcon:  "", // Omit from disk serialization (omitempty)
				TotalDamage: pstat.OverallStats.TotalDamage,
			})
			for tID, tBreakdown := range pstat.DamageByTarget {
				targetDamageMap[tID] += tBreakdown.TotalDamage
			}
		}

		sort.Slice(summaryData.Players, func(i, j int) bool {
			return summaryData.Players[i].TotalDamage > summaryData.Players[j].TotalDamage
		})

		for tID, dmg := range targetDamageMap {
			name := "Unknown"
			var raceId uint32
			if tStat, ok := summary.Targets[tID]; ok {
				name = tStat.Name
				raceId = tStat.RaceID
			}
			summaryData.Enemies = append(summaryData.Enemies, SessionSummaryEnemy{
				Name:        name,
				RaceID:      raceId,
				TotalDamage: dmg,
			})
		}
		sort.Slice(summaryData.Enemies, func(i, j int) bool {
			return summaryData.Enemies[i].TotalDamage > summaryData.Enemies[j].TotalDamage
		})
	}

	newF, err := os.Create(newPath)
	if err != nil {
		return fmt.Errorf("failed to create permanent session file: %w", err)
	}
	defer newF.Close()

	sumEvent := eventSessionSummary{
		eventBase: eventBase{EventId: eventIdSessionSummary, At: time.Now().Unix(), Id: ""},
		Type:      "SessionSummary",
		Summary:   summaryData,
	}
	sumBytes, err := json.Marshal(sumEvent)
	if err != nil {
		return fmt.Errorf("failed to marshal summary: %w", err)
	}
	if _, err := newF.Write(append(sumBytes, '\n')); err != nil {
		return fmt.Errorf("failed to write summary: %w", err)
	}

	// Write cached in-memory events to log file
	writer := bufio.NewWriter(newF)
	for _, e := range sm.currentSession.events {
		b, err := json.Marshal(e)
		if err != nil {
			logger.Println("Failed to marshal event during save:", err)
			continue
		}
		if _, err := writer.Write(append(b, '\n')); err != nil {
			return fmt.Errorf("failed to write event: %w", err)
		}
	}
	if err := writer.Flush(); err != nil {
		return fmt.Errorf("failed to flush event writer: %w", err)
	}

	if sm.recordPcap {
		oldPcapPath := filepath.Join(sm.logDirectory, livePcapFilename)
		newPcapPath := strings.TrimSuffix(newPath, ".ndjson") + ".pcapng"
		if _, err := os.Stat(oldPcapPath); err == nil {
			os.Rename(oldPcapPath, newPcapPath)
		}
	}

	sm.currentSession = nil
	return nil
}

// GetAllSessions scans for permanent logs, ignoring the live session file.
func (sm *SessionManager) GetAllSessions() ([]*Session, error) {
	// (This function remains unchanged from the previous version)
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	files, err := os.ReadDir(sm.logDirectory)
	if err != nil {
		return nil, err
	}

	var sessions []*Session
	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".ndjson") || file.Name() == liveSessionFilename {
			continue
		}

		var sess *Session

		// Read the first line to check for a summary header
		logFile, err := os.Open(filepath.Join(sm.logDirectory, file.Name()))
		if err == nil {
			scanner := bufio.NewScanner(logFile)
			if scanner.Scan() {
				var summaryEvent eventSessionSummary
				if err := json.Unmarshal(scanner.Bytes(), &summaryEvent); err == nil && summaryEvent.EventId == eventIdSessionSummary {
					summaryBytes, _ := json.Marshal(summaryEvent.Summary)
					var sumData SessionSummaryData
					if err := json.Unmarshal(summaryBytes, &sumData); err == nil {
						for i, enemy := range sumData.Enemies {
							var lookupId uint32
							if enemy.RaceID != 0 {
								lookupId = enemy.RaceID
							} else if strings.HasPrefix(enemy.Name, "Unknown Race (") {
								idStr := strings.TrimSuffix(strings.TrimPrefix(enemy.Name, "Unknown Race ("), ")")
								if parsed, err := strconv.ParseUint(idStr, 10, 32); err == nil {
									lookupId = uint32(parsed)
									sumData.Enemies[i].RaceID = lookupId
								}
							}
							if lookupId != 0 {
								newName := getRaceName(lookupId)
								if newName != enemy.Name {
									sumData.Enemies[i].Name = newName
								}
							}
						}

						for i, p := range sumData.Players {
							if icon, ok := arcanaNameToIcon[p.ArcanaName]; ok {
								sumData.Players[i].ArcanaIcon = icon
							}
						}

						sess = &Session{
							ID:        file.Name(),
							Name:      sumData.Name,
							StartTime: sumData.StartTime,
							Summary:   &sumData,
						}
					}
				}
			}
			logFile.Close()
		}

		if sess == nil {
			parts := strings.SplitN(file.Name(), "_", 3)
			if len(parts) >= 3 {
				timeStr := parts[0] + " " + strings.ReplaceAll(parts[1], "-", ":")
				startTime, err := time.Parse("2006-01-02 15:04:05", timeStr)
				if err == nil {
					name := strings.TrimSuffix(parts[2], ".ndjson")
					sess = &Session{
						ID:        file.Name(),
						Name:      name,
						StartTime: startTime.Unix(),
					}
				}
			}
		}

		if sess != nil && sess.StartTime > 0 {
			sessions = append(sessions, sess)
		}
	}
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].StartTime > sessions[j].StartTime
	})
	return sessions, nil
}

// DeleteSession now also ensures we don't delete the live file accidentally.
func (sm *SessionManager) DeleteSession(sessionID string) error {
	if sessionID == liveSessionFilename || sessionID == livePcapFilename {
		return fmt.Errorf("cannot delete the active live session file")
	}
	// logger.Println("[Locking] sessionManager.DeleteSession attempting to lock...")
	sm.mu.Lock()
	// logger.Println("...[Locked] sessionManager.DeleteSession acquired lock.")
	defer func() {
		// logger.Println("[Unlocking] sessionManager.DeleteSession attempting to unlock.")
		sm.mu.Unlock()
		// logger.Println("...[Unlocked] sessionManager.DeleteSession released lock.")
	}()

	ndjsonPath := filepath.Join(sm.logDirectory, sessionID)
	if err := os.Remove(ndjsonPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete ndjson log: %w", err)
	}
	pcapPath := strings.TrimSuffix(ndjsonPath, ".ndjson") + ".pcapng"
	if _, err := os.Stat(pcapPath); err == nil {
		os.Remove(pcapPath)
	}
	return nil
}

func updateSessionHeaderName(filePath string, newName string) error {
	f, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer f.Close()

	var lines []string
	scanner := bufio.NewScanner(f)
	if scanner.Scan() {
		firstLine := scanner.Text()
		var summaryEvent eventSessionSummary
		if err := json.Unmarshal([]byte(firstLine), &summaryEvent); err == nil && summaryEvent.EventId == eventIdSessionSummary {
			var firstLineMap map[string]interface{}
			if err := json.Unmarshal([]byte(firstLine), &firstLineMap); err == nil {
				if summaryObj, ok := firstLineMap["summary"].(map[string]interface{}); ok {
					summaryObj["name"] = newName
					firstLineMap["summary"] = summaryObj
					updatedBytes, err := json.Marshal(firstLineMap)
					if err == nil {
						lines = append(lines, string(updatedBytes))
					} else {
						lines = append(lines, firstLine)
					}
				} else {
					lines = append(lines, firstLine)
				}
			} else {
				lines = append(lines, firstLine)
			}
		} else {
			lines = append(lines, firstLine)
		}
	} else {
		return nil
	}

	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	f.Close()

	fWrite, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer fWrite.Close()
	writer := bufio.NewWriter(fWrite)
	for _, line := range lines {
		if _, err := writer.WriteString(line + "\n"); err != nil {
			return err
		}
	}
	return writer.Flush()
}

// RenameSession handles renaming a session, supporting both YYYY-MM-DD_HH-MM-SS_Name.ndjson and session-<timestamp>.ndjson file formats.
func (sm *SessionManager) RenameSession(sessionID, newName string) (*Session, error) {
	sm.mu.Lock()
	defer func() {
		sm.mu.Unlock()
	}()

	sanitizedName := sanitizeFilename(newName)
	oldPath := filepath.Join(sm.logDirectory, sessionID)

	if _, err := os.Stat(oldPath); err != nil {
		return nil, fmt.Errorf("session file not found: %w", err)
	}

	parts := strings.SplitN(sessionID, "_", 3)
	var newNdjsonFilename string
	if len(parts) >= 3 {
		timestampPart := parts[0] + "_" + parts[1]
		newFilenameBase := fmt.Sprintf("%s_%s", timestampPart, sanitizedName)
		newNdjsonFilename = newFilenameBase + ".ndjson"
		newPath := filepath.Join(sm.logDirectory, newNdjsonFilename)

		if err := os.Rename(oldPath, newPath); err != nil {
			return nil, fmt.Errorf("failed to rename session file: %w", err)
		}
		oldPcapPath := strings.TrimSuffix(oldPath, ".ndjson") + ".pcapng"
		newPcapPath := strings.TrimSuffix(newPath, ".ndjson") + ".pcapng"
		if _, err := os.Stat(oldPcapPath); err == nil {
			os.Rename(oldPcapPath, newPcapPath)
		}
		_ = updateSessionHeaderName(newPath, newName)
	} else {
		newNdjsonFilename = sessionID
		if err := updateSessionHeaderName(oldPath, newName); err != nil {
			return nil, fmt.Errorf("failed to update session header: %w", err)
		}
	}

	var startTime int64
	var summaryData *SessionSummaryData
	logFile, err := os.Open(filepath.Join(sm.logDirectory, newNdjsonFilename))
	if err == nil {
		scanner := bufio.NewScanner(logFile)
		if scanner.Scan() {
			var summaryEvent eventSessionSummary
			if err := json.Unmarshal(scanner.Bytes(), &summaryEvent); err == nil && summaryEvent.EventId == eventIdSessionSummary {
				summaryBytes, _ := json.Marshal(summaryEvent.Summary)
				var sumData SessionSummaryData
				if err := json.Unmarshal(summaryBytes, &sumData); err == nil {
					startTime = sumData.StartTime
					summaryData = &sumData
				}
			}
		}
		logFile.Close()
	}

	if startTime == 0 && len(parts) >= 3 {
		timeStr := parts[0] + " " + strings.ReplaceAll(parts[1], "-", ":")
		if t, err := time.Parse("2006-01-02 15:04:05", timeStr); err == nil {
			startTime = t.Unix()
		}
	}

	return &Session{
		ID:        newNdjsonFilename,
		Name:      sanitizedName,
		StartTime: startTime,
		Summary:   summaryData,
	}, nil
}

// WriteEventToLog appends the event to the current live session in-memory event cache.
func (sm *SessionManager) WriteEventToLog(e iEvent) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	s := sm.currentSession
	if s == nil {
		return nil
	}
	s.events = append(s.events, e)
	return nil
}

// WriteEntityAppearEvents generates and writes appearance events for currently cached entities.
func (sm *SessionManager) WriteEntityAppearEvents(entities []*packet.EntityInfo) {
	for _, entity := range entities {
		e := newEventFromEntity(entity, time.Now())
		if err := sm.WriteEventToLog(e); err != nil {
			logger.Println("Failed to write cached entity appear event to log:", err)
		}
	}
}

// WritePacketToLog remains unchanged.
func (sm *SessionManager) WritePacketToLog(ci gopacket.CaptureInfo, data []byte) {
	// (This function is the same as the previous version)
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	s := sm.currentSession
	if s == nil || s.pcapWriter == nil {
		return
	}
	_ = s.pcapWriter.WritePacket(ci, data)
}

// The migration logic remains unchanged.
func (sm *SessionManager) runMigration() error {
	// (This function is the same as the previous version)
	oldMetadataPath := "sessions.json"
	if _, err := os.Stat(oldMetadataPath); os.IsNotExist(err) {
		return nil
	}
	logger.Println("Old sessions.json found. Starting one-time migration...")
	data, err := os.ReadFile(oldMetadataPath)
	if err != nil {
		return fmt.Errorf("could not read old sessions.json: %w", err)
	}
	var oldSessions []*oldSessionDTO
	if err := json.Unmarshal(data, &oldSessions); err != nil {
		return fmt.Errorf("could not parse old sessions.json: %w", err)
	}
	metadataMap := make(map[string]*oldSessionDTO)
	for _, s := range oldSessions {
		originalFilename := filepath.Base(s.NdjsonLogPath)
		metadataMap[originalFilename] = s
	}
	files, err := os.ReadDir(sm.logDirectory)
	if err != nil {
		return fmt.Errorf("could not read log directory for migration: %w", err)
	}
	migratedCount := 0
	for _, file := range files {
		if file.IsDir() || !strings.HasPrefix(file.Name(), "session-log-") {
			continue
		}
		var name string
		var startTime int64
		if metadata, ok := metadataMap[file.Name()]; ok {
			name = metadata.Name
			startTime = metadata.StartTime
		} else {
			logger.Printf("Orphaned log file found: %s. Attempting to recover start time.", file.Name())
			f, err := os.Open(filepath.Join(sm.logDirectory, file.Name()))
			if err != nil {
				logger.Printf("...could not open orphaned file: %v", err)
				continue
			}
			scanner := bufio.NewScanner(f)
			if scanner.Scan() {
				var firstEvent eventBaseForMigration
				if err := json.Unmarshal(scanner.Bytes(), &firstEvent); err == nil {
					startTime = firstEvent.At
					name = "Archived-Session"
				}
			}
			f.Close()
		}
		if startTime == 0 {
			logger.Printf("...could not determine start time for %s. Skipping.", file.Name())
			continue
		}
		t := time.Unix(startTime, 0)
		newFilename := fmt.Sprintf("%s_%s.ndjson", t.Format("2006-01-02_15-04-05"), sanitizeFilename(name))
		oldPath := filepath.Join(sm.logDirectory, file.Name())
		newPath := filepath.Join(sm.logDirectory, newFilename)
		logger.Printf("Migrating: %s -> %s", file.Name(), newFilename)
		if err := os.Rename(oldPath, newPath); err != nil {
			logger.Printf("...failed to rename %s: %v", file.Name(), err)
			continue
		}
		migratedCount++
		oldPcapPath := strings.Replace(oldPath, "session-log-", "session-pcap-", 1)
		oldPcapPath = strings.TrimSuffix(oldPcapPath, ".ndjson") + ".pcapng"
		if _, err := os.Stat(oldPcapPath); err == nil {
			newPcapPath := strings.TrimSuffix(newPath, ".ndjson") + ".pcapng"
			os.Rename(oldPcapPath, newPcapPath)
		}
	}
	logger.Printf("Migration complete. Migrated %d files.", migratedCount)
	return os.Rename(oldMetadataPath, oldMetadataPath+".migrated")
}

// Helper structs for migration
type oldSessionDTO struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	StartTime     int64  `json:"startTime"`
	NdjsonLogPath string `json:"ndjsonLogPath"`
}
type eventBaseForMigration struct {
	At int64 `json:"At"`
}

// MigrateSession rewrites the session log file with a newly generated summary header.
func (sm *SessionManager) MigrateSession(sessionID string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sessionID == liveSessionFilename || sessionID == livePcapFilename {
		return fmt.Errorf("cannot migrate live session")
	}

	oldPath := filepath.Join(sm.logDirectory, sessionID)
	summary, err := GenerateSummaryFromFile(oldPath)
	if err != nil {
		return fmt.Errorf("failed to generate summary for migration: %w", err)
	}

	var summaryData SessionSummaryData
	oldF, err := os.Open(oldPath)
	if err != nil {
		return fmt.Errorf("failed to open session file: %w", err)
	}

	scanner := bufio.NewScanner(oldF)
	hasOldSummary := false
	if scanner.Scan() {
		var firstEvent eventBase
		if err := json.Unmarshal(scanner.Bytes(), &firstEvent); err == nil && firstEvent.EventId == eventIdSessionSummary {
			hasOldSummary = true
			var wrapper struct {
				Summary SessionSummaryData `json:"summary"`
			}
			if err := json.Unmarshal(scanner.Bytes(), &wrapper); err == nil {
				summaryData.Name = wrapper.Summary.Name
				summaryData.StartTime = wrapper.Summary.StartTime
			}
		}
	}

	if summaryData.Name == "" {
		parts := strings.SplitN(sessionID, "_", 3)
		if len(parts) >= 3 {
			summaryData.Name = strings.TrimSuffix(parts[2], ".ndjson")
		} else {
			summaryData.Name = "Migrated Session"
		}
		summaryData.StartTime = summary.StartTime
	}

	summaryData.Duration = summary.EncounterDuration
	summaryData.TotalDamage = summary.TotalDamage
	
	targetDamageMap := make(map[string]float32)

	for _, pstat := range summary.Players {
		summaryData.Players = append(summaryData.Players, SessionSummaryPlayer{
			Name:        pstat.Name,
			DPS:         pstat.OverallStats.DPS,
			ArcanaName:  pstat.TalentName,
			ArcanaIcon:  "", // Omit from disk serialization (omitempty)
			TotalDamage: pstat.OverallStats.TotalDamage,
		})
		for tID, tBreakdown := range pstat.DamageByTarget {
			targetDamageMap[tID] += tBreakdown.TotalDamage
		}
	}
	
	sort.Slice(summaryData.Players, func(i, j int) bool {
		return summaryData.Players[i].TotalDamage > summaryData.Players[j].TotalDamage
	})
	
	for tID, dmg := range targetDamageMap {
		name := "Unknown"
		var raceId uint32
		if tStat, ok := summary.Targets[tID]; ok {
			name = tStat.Name
			raceId = tStat.RaceID
		}
		summaryData.Enemies = append(summaryData.Enemies, SessionSummaryEnemy{
			Name:        name,
			RaceID:      raceId,
			TotalDamage: dmg,
		})
	}
	sort.Slice(summaryData.Enemies, func(i, j int) bool {
		return summaryData.Enemies[i].TotalDamage > summaryData.Enemies[j].TotalDamage
	})

	newPath := oldPath + ".tmp"
	newF, err := os.Create(newPath)
	if err != nil {
		oldF.Close()
		return fmt.Errorf("failed to create temporary migrated file: %w", err)
	}

	sumEvent := eventSessionSummary{
		eventBase: eventBase{EventId: eventIdSessionSummary, At: time.Now().Unix(), Id: ""},
		Type:      "SessionSummary",
		Summary:   summaryData,
	}
	sumBytes, _ := json.Marshal(sumEvent)
	newF.Write(sumBytes)
	newF.Write([]byte("\n"))

	oldF.Seek(0, 0)
	oldScanner := bufio.NewScanner(oldF)
	isFirstLine := true
	for oldScanner.Scan() {
		if isFirstLine {
			isFirstLine = false
			if hasOldSummary {
				continue
			}
		}
		newF.Write(oldScanner.Bytes())
		newF.Write([]byte("\n"))
	}
	newF.Close()
	oldF.Close()

	if err := os.Rename(newPath, oldPath); err != nil {
		os.Remove(newPath)
		return fmt.Errorf("failed to replace old file with migrated one: %w", err)
	}

	return nil
}
