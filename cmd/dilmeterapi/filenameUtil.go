package main

import (
	"regexp"
	"strings"
)

// sanitizeFilename takes a user-provided string and cleans it to be a valid filename.
func sanitizeFilename(name string) string {
	// Replace spaces with hyphens first.
	sanitized := strings.ReplaceAll(name, " ", "-")

	// Remove all characters that are not letters, numbers, hyphens, or underscores.
	re := regexp.MustCompile(`[^a-zA-Z0-9_-]`)
	sanitized = re.ReplaceAllString(sanitized, "")

	// Trim any leading/trailing hyphens that might have been created.
	sanitized = strings.Trim(sanitized, "-_")

	// Limit length to avoid overly long filenames.
	if len(sanitized) > 100 {
		sanitized = sanitized[:100]
	}
	if sanitized == "" {
		return "Untitled-Session"
	}
	return sanitized
}
