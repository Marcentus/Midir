package main

import (
	"sort"
	"strings"
)

// normalizeMetaData cleans up the metadata string by:
// 1. Removing the "SBT" parameter (which is volatile).
// 2. Sorting the remaining parameters alphabetically to ensure consistency.
// 3. Reassembling the string.
func normalizeMetaData(metaData string) string {
	if metaData == "" {
		return ""
	}

	// Split by semicolon
	parts := strings.Split(metaData, ";")
	var validParts []string

	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}
		// Ignore SBT parameter
		if strings.HasPrefix(trimmed, "SBT:") {
			continue
		}
		validParts = append(validParts, trimmed)
	}

	// Sort parts to ensure "A;B" and "B;A" are treated as the same
	sort.Strings(validParts)

	// Rejoin with semicolons
	if len(validParts) == 0 {
		return ""
	}
	return strings.Join(validParts, ";") + ";"
}
