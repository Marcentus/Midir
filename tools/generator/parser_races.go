package main

import (
	"encoding/xml"
	"io/ioutil"
	"os"
	"strconv"
	"strings"

	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
)

type RaceData struct {
	RaceId int    `json:"raceId"`
	Name   string `json:"name"`
}

type RaceRoot struct {
	XMLName xml.Name `xml:"Race"`
	List    RaceList `xml:"RaceList"`
}

type RaceList struct {
	Races []struct {
		ID        string `xml:"ID,attr"`
		LocalName string `xml:"LocalName,attr"`
	} `xml:"Race"`
}

func ProcessRaces(xmlPath string, translator *Translator) (map[string]RaceData, error) {
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
	
	var root RaceRoot
	err = xml.Unmarshal([]byte(content), &root)
	if err != nil {
		return nil, err
	}

	results := make(map[string]RaceData)

	for _, r := range root.List.Races {
		if _, exists := results[r.ID]; exists {
			continue
		}

		id, _ := strconv.Atoi(r.ID)
		if id == 0 {
			continue
		}

		name := translator.Resolve(r.LocalName)

		results[r.ID] = RaceData{
			RaceId: id,
			Name:   name,
		}
	}

	return results, nil
}
