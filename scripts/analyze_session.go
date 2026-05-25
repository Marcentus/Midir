package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"text/tabwriter"
)

type EventID uint16

const (
	eventIdEntityAppear              EventID = 1
	eventIdEntityDisappear           EventID = 2
	eventIdDamage                    EventID = 3
	eventIdCharacterConditionEnable  EventID = 4
	eventIdCharacterConditionDisable EventID = 5
	eventIdEntityDeath               EventID = 6
	eventIdEntityRevive              EventID = 7
	eventIdSkillUse                  EventID = 8
	eventIdSkillStart                EventID = 9
	eventIdEntityHPUpdate            EventID = 10
	eventIdSessionSummary            EventID = 9999
)

var eventNames = map[EventID]string{
	eventIdEntityAppear:              "EntityAppear",
	eventIdEntityDisappear:           "EntityDisappear",
	eventIdDamage:                    "Damage",
	eventIdCharacterConditionEnable:  "CharacterConditionEnable",
	eventIdCharacterConditionDisable: "CharacterConditionDisable",
	eventIdEntityDeath:               "EntityDeath",
	eventIdEntityRevive:              "EntityRevive",
	eventIdSkillUse:                  "SkillUse",
	eventIdSkillStart:                "SkillStart",
	eventIdEntityHPUpdate:            "EntityHPUpdate",
	eventIdSessionSummary:            "SessionSummary",
}

type EventStats struct {
	ID        EventID
	Name      string
	Count     int
	TotalSize int64
}

type PartialEvent struct {
	EventID EventID `json:"EventId"`
}

func main() {
	filePath := "build/logs/session-1779681854060.ndjson"
	if len(os.Args) > 1 {
		filePath = os.Args[1]
	}

	absPath, err := filepath.Abs(filePath)
	if err != nil {
		fmt.Printf("Error getting absolute path: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Analyzing session file: %s\n", absPath)

	file, err := os.Open(absPath)
	if err != nil {
		fmt.Printf("Error opening file: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		fmt.Printf("Error getting file stats: %v\n", err)
		os.Exit(1)
	}
	fileSize := stat.Size()

	statsMap := make(map[EventID]*EventStats)
	
	scanner := bufio.NewScanner(file)
	// Use a larger buffer (10MB) just in case some lines are very long
	const maxCapacity = 10 * 1024 * 1024
	buf := make([]byte, maxCapacity)
	scanner.Buffer(buf, maxCapacity)

	var totalEvents int
	var totalParsedSize int64
	var unmarshalErrors int
	var emptyLines int

	for scanner.Scan() {
		line := scanner.Bytes()
		lineSize := int64(len(line) + 1) // +1 for the newline character
		
		if len(line) == 0 {
			emptyLines++
			continue
		}

		var ev PartialEvent
		if err := json.Unmarshal(line, &ev); err != nil {
			unmarshalErrors++
			continue
		}

		totalEvents++
		totalParsedSize += lineSize

		s, exists := statsMap[ev.EventID]
		if !exists {
			name, ok := eventNames[ev.EventID]
			if !ok {
				name = fmt.Sprintf("Unknown (%d)", ev.EventID)
			}
			s = &EventStats{
				ID:   ev.EventID,
				Name: name,
			}
			statsMap[ev.EventID] = s
		}
		s.Count++
		s.TotalSize += lineSize
	}

	if err := scanner.Err(); err != nil {
		fmt.Printf("Error scanning file: %v\n", err)
	}

	// Sort stats by TotalSize descending
	var statsList []*EventStats
	for _, s := range statsMap {
		statsList = append(statsList, s)
	}
	sort.Slice(statsList, func(i, j int) bool {
		return statsList[i].TotalSize > statsList[j].TotalSize
	})

	fmt.Println("\n--- EVENT ANALYSIS RESULTS ---")
	fmt.Printf("Total File Size:   %.2f MB (%d bytes)\n", float64(fileSize)/(1024*1024), fileSize)
	fmt.Printf("Total Events:      %d\n", totalEvents)
	fmt.Printf("Unmarshal Errors:  %d\n", unmarshalErrors)
	fmt.Printf("Empty Lines:       %d\n\n", emptyLines)

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', tabwriter.AlignRight)
	fmt.Fprintln(w, "EVENT ID & NAME\tCOUNT\tCOUNT %\tTOTAL SIZE\tSIZE %\tAVG SIZE/EVENT")
	fmt.Fprintln(w, "---------------\t-----\t-------\t----------\t------\t--------------")

	for _, s := range statsList {
		countPct := float64(s.Count) / float64(totalEvents) * 100
		sizePct := float64(s.TotalSize) / float64(fileSize) * 100
		avgSize := float64(s.TotalSize) / float64(s.Count)

		sizeStr := ""
		if s.TotalSize >= 1024*1024 {
			sizeStr = fmt.Sprintf("%.2f MB", float64(s.TotalSize)/(1024*1024))
		} else if s.TotalSize >= 1024 {
			sizeStr = fmt.Sprintf("%.2f KB", float64(s.TotalSize)/1024)
		} else {
			sizeStr = fmt.Sprintf("%d B", s.TotalSize)
		}

		fmt.Fprintf(w, "%-32s\t%d\t%.2f%%\t%s\t%.2f%%\t%.1f B\n", 
			fmt.Sprintf("[%d] %s", s.ID, s.Name),
			s.Count,
			countPct,
			sizeStr,
			sizePct,
			avgSize,
		)
	}
	w.Flush()
}
