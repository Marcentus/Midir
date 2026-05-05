package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type Translator struct {
	data map[string]map[string]string // category -> id -> value
}

func NewTranslator() *Translator {
	return &Translator{
		data: make(map[string]map[string]string),
	}
}

func (t *Translator) LoadFile(filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	bufferedReader := bufio.NewReader(file)

	filename := strings.ToLower(strings.TrimSuffix(filepath.Base(filePath), ".english.txt"))
	category := filename

	if _, ok := t.data[category]; !ok {
		t.data[category] = make(map[string]string)
	}

	count := 0
	for {
		line, err := bufferedReader.ReadString('\n')
		if err != nil && err != io.EOF {
			return err
		}
		
		line = strings.TrimSpace(line)
		if line == "" {
			if err == io.EOF {
				break
			}
			continue
		}

		parts := strings.SplitN(line, "\t", 2)
		if len(parts) == 2 {
			id := strings.TrimSpace(parts[0])
			// Trim UTF-8 BOM if present
			id = strings.TrimPrefix(id, "\ufeff")
			
			value := strings.TrimSpace(parts[1])
			t.data[category][id] = value
			count++
		}
		
		if err == io.EOF {
			break
		}
	}
	fmt.Printf("  Loaded %d translations into category '%s' (from %s)\n", count, category, filename)

	return nil
}

func (t *Translator) Resolve(ltTag string) string {
	// Format: _LT[xml.category.id]
	if !strings.HasPrefix(ltTag, "_LT[") || !strings.HasSuffix(ltTag, "]") {
		return ltTag
	}

	content := ltTag[4 : len(ltTag)-1]
	parts := strings.Split(content, ".")
	
	// Sometimes it's _LT[category.id] instead of _LT[xml.category.id]
	var category, id string
	if len(parts) >= 3 {
		category = strings.ToLower(parts[1])
		id = parts[2]
	} else if len(parts) == 2 {
		category = strings.ToLower(parts[0])
		id = parts[1]
	} else {
		return ltTag
	}

	if catData, ok := t.data[category]; ok {
		if val, ok := catData[id]; ok {
			return val
		}
	}

	// Fallback for itemdb: sometimes it's specified as itemdb_etc in tags but might be in main itemdb, or vice-versa
	if strings.HasPrefix(category, "itemdb") {
		if catData, ok := t.data["itemdb"]; ok {
			if val, ok := catData[id]; ok {
				return val
			}
		}
	}

	return ltTag
}
