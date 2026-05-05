// race_data.go
package main

import (
	"encoding/json"
	"strconv"
)

// RaceInfo matches the structure of entries in your races.json
type RaceInfo struct {
	RaceId int    `json:"raceId"`
	Name   string `json:"name"`
}

// A global map to hold our race data, Key: raceId, Value: raceName
var raceNameCache = make(map[uint32]string)

func loadRaceNames() {
	// Read the embedded races.json file
	data, err := staticData.ReadFile("static_data/races.json")
	if err != nil {
		logger.Println("Could not read embedded races.json:", err)
		return
	}

	// The JSON is a map of string keys to RaceInfo objects
	var rawRaces map[string]RaceInfo
	if err := json.Unmarshal(data, &rawRaces); err != nil {
		logger.Println("Error parsing races.json:", err)
		return
	}

	// Populate our cache for fast lookups
	for _, race := range rawRaces {
		raceNameCache[uint32(race.RaceId)] = race.Name
	}

	logger.Printf("Loaded %d race names.", len(raceNameCache))
}

// Helper to get a race name from its ID
func getRaceName(raceId uint32) string {
	if name, ok := raceNameCache[raceId]; ok {
		return name
	}
	return "Unknown Race (" + strconv.FormatUint(uint64(raceId), 10) + ")"
}
