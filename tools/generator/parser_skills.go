package main

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
)

type SkillData struct {
	SkillId int    `json:"skillId"`
	Name    string `json:"name"`
	IconUrl string `json:"iconUrl"`
}

type SkillInfo struct {
	XMLName xml.Name `xml:"SkillInfo"`
	List    SkillList `xml:"SkillList"`
}

type SkillList struct {
	Skills []struct {
		SkillID        string `xml:"SkillID,attr"`
		SkillLocalName string `xml:"SkillLocalName,attr"`
		ImageFile      string `xml:"ImageFile,attr"`
		PositionX      string `xml:"PositionX,attr"`
		PositionY      string `xml:"PositionY,attr"`
	} `xml:"Skill"`
}

func ProcessSkills(xmlPath, dataDir, outDir string, translator *Translator, imgProc *ImageProcessor) (map[string]SkillData, error) {
	xmlFile, err := os.Open(xmlPath)
	if err != nil {
		return nil, err
	}
	defer xmlFile.Close()

	decoder := unicode.UTF16(unicode.LittleEndian, unicode.UseBOM).NewDecoder()
	reader := transform.NewReader(xmlFile, decoder)
	byteValue, _ := ioutil.ReadAll(reader)
	// Replace "utf-16" with "utf-8" in the header so xml.Unmarshal doesn't complain
	content := strings.Replace(string(byteValue), "encoding=\"utf-16\"", "encoding=\"utf-8\"", 1)
	
	var info SkillInfo
	err = xml.Unmarshal([]byte(content), &info)
	if err != nil {
		return nil, fmt.Errorf("XML Unmarshal error: %v", err)
	}

	results := make(map[string]SkillData)

	for _, s := range info.List.Skills {
		id, _ := strconv.Atoi(s.SkillID)
		if id == 0 {
			continue
		}

		name := translator.Resolve(s.SkillLocalName)
		
		// Handle icon extraction
		iconFileName := ""
		imageFile := strings.TrimSpace(s.ImageFile)
		if imageFile != "" {
			// Skill icons are usually in data/gfx/image/...
			// We need to map this to our local tools/data/gfx/image/...
			relPath := strings.TrimPrefix(imageFile, "data/")
			fullDdsPath := filepath.Join(dataDir, relPath)
			
			posX, _ := strconv.Atoi(s.PositionX)
			posY, _ := strconv.Atoi(s.PositionY)
			
			baseName := strings.TrimSuffix(filepath.Base(imageFile), ".dds")
			iconFileName = fmt.Sprintf("%s_%d_%d.png", baseName, posX, posY)
			
			// Extract if possible
			if imgProc != nil {
				destPath := filepath.Join(outDir, iconFileName)
				err := imgProc.ExtractIcon(fullDdsPath, destPath, posX, posY)
				if err != nil {
					fmt.Printf("Warning: Failed to extract icon for skill %d: %v\n", id, err)
				}
			}
		}

		results[s.SkillID] = SkillData{
			SkillId: id,
			Name:    name,
			IconUrl: iconFileName,
		}
	}

	return results, nil
}
