package util

import (
	"time"
	_ "time/tzdata"
)

const pstOffset = -8 * 60 * 60 // PST is UTC-8
const pdtOffset = -7 * 60 * 60 // PDT is UTC-7

var (
	pacificLoc *time.Location
)

// GetPacificLocation returns the time.Location for America/Los_Angeles (Pacific Time).
// It attempts to load the location from the system's timezone database.
func GetPacificLocation() *time.Location {
	if pacificLoc != nil {
		return pacificLoc
	}
	loc, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		// Fallback to fixed PST if loading fails
		return time.FixedZone("PST", pstOffset)
	}
	pacificLoc = loc
	return pacificLoc
}

// ParseMabiTime converts a Mabinogi timestamp (ticks) to a time.Time object.
// Mabinogi servers use Pacific Time. This function handles both PST and PDT
// by loading the America/Los_Angeles location.
//
// NOTE FOR FUTURE DEVELOPERS:
// If the timers are off by 1 hour, it is likely due to a Daylight Saving Time (DST) transition.
// Go's time.LoadLocation("America/Los_Angeles") should handle this automatically by using the system's tzdata.
// If it fails, check if the system timezone database is up to date or if tzdata is bundled.
func ParseMabiTime(t uint64) time.Time {
	t = t / 1000

	// Mabinogi uses C# ticks offset from year 0. 
	// To get Unix Epoch offset (1970), we subtract 62135596800 seconds.
	t -= 62135596800

	// The timestamp from the server is typically "Local Server Time" (Pacific).
	// To correctly handle DST, we use local time arithmetic.
	pacific := GetPacificLocation()
	utcTime := time.Unix(int64(t), 0).UTC()

	// We treat the Unix timestamp as if it was already in Pacific time to get the correct components,
	// then we re-interpret it in the Pacific location to let Go determine the correct DST offset.
	return time.Date(
		utcTime.Year(), utcTime.Month(), utcTime.Day(),
		utcTime.Hour(), utcTime.Minute(), utcTime.Second(),
		0, pacific,
	)
}

// TimeToMabiTime is the inverse of ParseMabiTime, converting a time.Time to Mabinogi ticks.
func TimeToMabiTime(t time.Time) uint64 {
	pacific := GetPacificLocation()
	local := t.In(pacific)

	// Convert local components to a "UTC" timestamp that represents the raw local seconds
	utcEquivalent := time.Date(
		local.Year(), local.Month(), local.Day(),
		local.Hour(), local.Minute(), local.Second(),
		local.Nanosecond(), time.UTC,
	)

	return uint64(utcEquivalent.UnixMilli() + 62135596800000)
}
