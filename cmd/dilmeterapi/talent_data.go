// talent_data.go
package main

import (
	"encoding/json"
)

// ArcanaInfo matches the structure of a single arcana object in talents.json
type ArcanaInfo struct {
	Name          string   `json:"arcana_name"`
	RelatedSkills []uint16 `json:"related_skill_ids"`
	Icon          string   `json:"icon"`
	Color         string   `json:"color"`
}

// A global map for quick lookups: skill ID -> icon path
var skillToArcanaIcon = make(map[uint16]string)
var skillToArcanaName = make(map[uint16]string)
var skillToArcanaColor = make(map[uint16]string)

// Global maps for quick lookups: arcana name -> icon/color path
var arcanaNameToIcon = make(map[string]string)
var arcanaNameToColor = make(map[string]string)

// loadTalentData reads the embedded talents.json and populates our lookup map.
func loadTalentData() {
	// 1. Read the embedded talents.json file
	data, err := staticData.ReadFile("static_data/talents.json")
	if err != nil {
		logger.Println("Fatal: Could not read embedded talents.json:", err)
		// Since this is core data, you might want to panic or exit
		// For now, we'll just log it.
		return
	}

	// 2. The JSON has a top-level "arcanas" key
	var talentFile struct {
		Arcanas map[string]ArcanaInfo `json:"arcanas"`
	}
	if err := json.Unmarshal(data, &talentFile); err != nil {
		logger.Println("Error parsing talents.json:", err)
		return
	}

	// 3. Populate our cache for fast lookups
	for name, arcana := range talentFile.Arcanas {
		arcanaNameToIcon[name] = arcana.Icon
		arcanaNameToColor[name] = arcana.Color
		if arcana.Name != "" {
			arcanaNameToIcon[arcana.Name] = arcana.Icon
			arcanaNameToColor[arcana.Name] = arcana.Color
		}
		for _, skillId := range arcana.RelatedSkills {
			skillToArcanaIcon[skillId] = arcana.Icon
			skillToArcanaName[skillId] = arcana.Name
			skillToArcanaColor[skillId] = arcana.Color
		}
	}

	logger.Printf("Loaded %d arcana skill mappings.", len(skillToArcanaIcon))
	logger.Printf("Loaded %d arcana color mappings.", len(skillToArcanaColor))
}
