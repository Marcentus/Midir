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

type ConditionData struct {
	ConditionId int    `json:"conditionId"`
	Name        string `json:"name"`
	IconUrl     string `json:"iconUrl"`
}

type ConditionRoot struct {
	XMLName xml.Name `xml:"CharacterCondition"`
	List    ConditionList `xml:"CharacterConditionList"`
}

type ConditionList struct {
	Conditions []struct {
		ConditionID        string `xml:"ConditionID,attr"`
		ConditionLocalName string `xml:"ConditionLocalName,attr"`
		ImageFile          string `xml:"ImageFile,attr"`
		PositionX          string `xml:"PositionX,attr"`
		PositionY          string `xml:"PositionY,attr"`
	} `xml:"CharacterCondition"`
}

func ProcessConditions(xmlPath, dataDir, outDir string, translator *Translator, imgProc *ImageProcessor) (map[string]ConditionData, error) {
	xmlFile, err := os.Open(xmlPath)
	if err != nil {
		return nil, err
	}
	defer xmlFile.Close()

	decoder := unicode.UTF16(unicode.LittleEndian, unicode.UseBOM).NewDecoder()
	reader := transform.NewReader(xmlFile, decoder)
	byteValue, _ := ioutil.ReadAll(reader)
	
	content := strings.Replace(string(byteValue), "encoding=\"utf-16\"", "encoding=\"utf-8\"", 1)
	
	var root ConditionRoot
	err = xml.Unmarshal([]byte(content), &root)
	if err != nil {
		return nil, err
	}

	results := make(map[string]ConditionData)

	for _, c := range root.List.Conditions {
		id, _ := strconv.Atoi(c.ConditionID)
		
		name := translator.Resolve(c.ConditionLocalName)
		
		iconFileName := ""
		imageFile := strings.TrimSpace(c.ImageFile)
		if imageFile != "" {
			relPath := strings.TrimPrefix(imageFile, "data/")
			fullDdsPath := filepath.Join(dataDir, relPath)
			
			posX, _ := strconv.Atoi(c.PositionX)
			posY, _ := strconv.Atoi(c.PositionY)
			
			baseName := strings.TrimSuffix(filepath.Base(imageFile), ".dds")
			iconFileName = fmt.Sprintf("%s_%d_%d.png", baseName, posX, posY)
			
			if imgProc != nil {
				destPath := filepath.Join(outDir, iconFileName)
				err := imgProc.ExtractIcon(fullDdsPath, destPath, posX, posY)
				if err != nil {
					fmt.Printf("Warning: Failed to extract icon for condition %d: %v\n", id, err)
				}
			}
		}

		results[c.ConditionID] = ConditionData{
			ConditionId: id,
			Name:        name,
			IconUrl:     iconFileName,
		}
	}

	return results, nil
}
