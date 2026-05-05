package main

import (
	"encoding/xml"
	"io"
	"os"
	"strconv"

	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
)

type ItemData struct {
	ItemId int    `json:"itemId"`
	Name   string `json:"name"`
}

type MabiItem struct {
	ID        string `xml:"ID,attr"`
	TextName1 string `xml:"Text_Name1,attr"`
}

func ProcessItems(xmlPath string, translator *Translator) (map[string]ItemData, error) {
	xmlFile, err := os.Open(xmlPath)
	if err != nil {
		return nil, err
	}
	defer xmlFile.Close()

	decoder := unicode.UTF16(unicode.LittleEndian, unicode.UseBOM).NewDecoder()
	reader := transform.NewReader(xmlFile, decoder)
	
	// Since we are streaming, we can't easily replace the header in the whole string.
	// However, encoding/xml's Decoder can handle charset conversion if we provide a CharsetReader.
	// But our reader is already UTF-8 after the transform.Reader.
	// The problem is the <?xml ... encoding="utf-16"?> header.
	// We'll wrap the reader to skip the header or just use a CharsetReader that returns the same reader.
	
	xmlDecoder := xml.NewDecoder(reader)
	xmlDecoder.CharsetReader = func(charset string, input io.Reader) (io.Reader, error) {
		return input, nil // We already converted to UTF-8
	}

	results := make(map[string]ItemData)

	for {
		token, err := xmlDecoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		switch se := token.(type) {
		case xml.StartElement:
			if se.Name.Local == "Mabi_Item" {
				var item MabiItem
				if err := xmlDecoder.DecodeElement(&item, &se); err != nil {
					return nil, err
				}
				
				id, _ := strconv.Atoi(item.ID)
				if id == 0 {
					continue
				}

				name := translator.Resolve(item.TextName1)
				// If name is still the _LT tag or empty, we could fallback to TextName0 but it's not in our struct yet
				
				results[item.ID] = ItemData{
					ItemId: id,
					Name:   name,
				}
			}
		}
	}

	return results, nil
}
