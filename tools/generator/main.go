package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	// Paths
	dataDir := filepath.Join("..", "data")
	localDir := filepath.Join(dataDir, "local", "xml")
	dbDir := filepath.Join(dataDir, "db")
	outputDir := filepath.Join("out", "static_data")

	// 0. Clean and recreate output directories
	os.RemoveAll(outputDir)
	os.MkdirAll(filepath.Join(outputDir, "images", "skills"), 0755)
	os.MkdirAll(filepath.Join(outputDir, "images", "conditions"), 0755)

	fmt.Println("Mabinogi Data Generator Starting...")
	fmt.Printf("Data Dir: %s\n", dataDir)
	fmt.Printf("Output Dir: %s\n", outputDir)

	// 1. Initialize Translator
	translator := NewTranslator()
	transDir := filepath.Join(localDir)
	files, _ := ioutil.ReadDir(transDir)
	for _, f := range files {
		if strings.HasSuffix(f.Name(), ".english.txt") {
			fmt.Printf("Loading translation: %s\n", f.Name())
			err := translator.LoadFile(filepath.Join(transDir, f.Name()))
			if err != nil {
				fmt.Printf("Error loading %s: %v\n", f.Name(), err)
			}
		}
	}

	// 2. Initialize Image Processor
	imgProc := NewImageProcessor()

	// 3. Process Skills
	fmt.Println("Processing Skills...")
	skillsXml := filepath.Join(dbDir, "Skill", "SkillInfo.xml")
	skills, err := ProcessSkills(skillsXml, dataDir, filepath.Join(outputDir, "images", "skills"), translator, imgProc)
	if err != nil {
		fmt.Printf("Error processing skills: %v\n", err)
	} else {
		saveJson(filepath.Join(outputDir, "skills.json"), skills)
		fmt.Printf("Saved %d skills.\n", len(skills))
	}

	// 4. Process Races
	fmt.Println("Processing Races...")
	racesXml := filepath.Join(dbDir, "Race.xml")
	races, err := ProcessRaces(racesXml, translator)
	if err != nil {
		fmt.Printf("Error processing races: %v\n", err)
	} else {
		saveJson(filepath.Join(outputDir, "races.json"), races)
		fmt.Printf("Saved %d races.\n", len(races))
	}

	// 5. Process Conditions
	fmt.Println("Processing Conditions...")
	condXml := filepath.Join(dbDir, "CharacterCondition.xml")
	conditions, err := ProcessConditions(condXml, dataDir, filepath.Join(outputDir, "images", "conditions"), translator, imgProc)
	if err != nil {
		fmt.Printf("Error processing conditions: %v\n", err)
	} else {
		saveJson(filepath.Join(outputDir, "conditions.json"), conditions)
		fmt.Printf("Saved %d conditions.\n", len(conditions))
	}

	// 6. Process Items
	fmt.Println("Processing Items...")
	itemFiles := []string{
		"ItemDB.xml",
		"ItemDB_MainEquip.xml",
		"ItemDB_SubEquip.xml",
		"itemDB_ETC.xml",
		"itemDB_Script.xml",
		"itemDB_Weapon.xml",
	}
	allItems := make(map[string]ItemData)
	for _, f := range itemFiles {
		fmt.Printf("  Parsing %s...\n", f)
		items, err := ProcessItems(filepath.Join(dbDir, f), translator)
		if err != nil {
			fmt.Printf("Warning: Failed to parse %s: %v\n", f, err)
			continue
		}
		for k, v := range items {
			allItems[k] = v
		}
	}
	saveJson(filepath.Join(outputDir, "items.json"), allItems)
	fmt.Printf("Saved %d total items.\n", len(allItems))

	fmt.Println("Done!")
}

func saveJson(path string, data interface{}) {
	file, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		log.Printf("Error marshalling JSON: %v", err)
		return
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		log.Printf("Error creating directory: %v", err)
		return
	}

	err = ioutil.WriteFile(path, file, 0644)
	if err != nil {
		log.Printf("Error writing JSON file: %v", err)
	}
}
