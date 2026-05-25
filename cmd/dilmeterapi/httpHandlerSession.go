package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/Marcentus/Midir/packet"
	"github.com/go-chi/chi/v5"
)

func sessionRouter(sm *SessionManager) http.Handler {
	r := chi.NewRouter()
	r.Get("/", handleGetSessions(sm))
	r.Post("/save", handleSaveSession(sm))
	r.Post("/migrate-all", handleMigrateAllSessions(sm))
	r.Route("/{sessionID}", func(r chi.Router) {
		r.Put("/", handleRenameSession(sm))
		r.Get("/log", handleGetSessionLog(sm))
		r.Delete("/", handleDeleteSession(sm))
		r.Get("/summary", handleGetSessionSummary(sm))
		r.Post("/migrate", handleMigrateSession(sm))
	})
	return r
}

func handleGetSessions(sm *SessionManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sessions, err := sm.GetAllSessions()
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Failed to get sessions: "+err.Error())
			return
		}
		respondWithJSON(w, http.StatusOK, sessions)
	}
}

func handleSaveSession(sm *SessionManager) http.HandlerFunc {
	type saveRequest struct {
		Name string `json:"name"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		var req saveRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondWithError(w, http.StatusBadRequest, "Invalid request body")
			return
		}
		if req.Name == "" {
			req.Name = "Saved Session"
		}
		if err := sm.SaveLiveSession(req.Name); err != nil {
			respondWithError(w, http.StatusInternalServerError, "Failed to save session: "+err.Error())
			return
		}
		if _, err := sm.StartLiveSession(); err != nil {
			respondWithError(w, http.StatusInternalServerError, "Failed to start new live session after saving: "+err.Error())
			return
		}
		if globalPub != nil {
			globalPub.aggregator.mu.RLock()
			var activeEntities []*packet.EntityInfo
			for _, entity := range globalPub.aggregator.entityCache {
				activeEntities = append(activeEntities, entity)
			}
			globalPub.aggregator.mu.RUnlock()
			sm.WriteEntityAppearEvents(activeEntities)
		}
		w.WriteHeader(http.StatusCreated)
	}
}

func handleRenameSession(sm *SessionManager) http.HandlerFunc {
	type renameRequest struct {
		Name string `json:"name"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		sessionID := chi.URLParam(r, "sessionID")
		var req renameRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondWithError(w, http.StatusBadRequest, "Invalid request body")
			return
		}
		if req.Name == "" {
			respondWithError(w, http.StatusBadRequest, "Name cannot be empty")
			return
		}
		session, err := sm.RenameSession(sessionID, req.Name)
		if err != nil {
			respondWithError(w, http.StatusNotFound, "Session not found or could not be renamed: "+err.Error())
			return
		}
		respondWithJSON(w, http.StatusOK, session)
	}
}

func handleGetSessionLog(sm *SessionManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sessionID := chi.URLParam(r, "sessionID")
		logPath := filepath.Join(sm.logDirectory, sessionID)
		f, err := os.Open(logPath)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Could not open session log file")
			return
		}
		defer f.Close()
		fi, err := f.Stat()
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Could not get file info")
			return
		}
		w.Header().Set("Content-Type", "application/x-ndjson")
		http.ServeContent(w, r, sessionID, fi.ModTime(), f)
	}
}

func handleGetSessionSummary(sm *SessionManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sessionID := chi.URLParam(r, "sessionID")
		logPath := filepath.Join(sm.logDirectory, sessionID)

		summary, err := GenerateSummaryFromFile(logPath)
		if err != nil {
			if os.IsNotExist(err) {
				respondWithError(w, http.StatusNotFound, "Session not found")
			} else {
				respondWithError(w, http.StatusInternalServerError, "Could not open session log file")
			}
			return
		}

		respondWithJSON(w, http.StatusOK, summary)
	}
}

func handleMigrateSession(sm *SessionManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sessionID := chi.URLParam(r, "sessionID")
		if err := sm.MigrateSession(sessionID); err != nil {
			respondWithError(w, http.StatusInternalServerError, "Failed to migrate session: "+err.Error())
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func handleMigrateAllSessions(sm *SessionManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sessions, err := sm.GetAllSessions()
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Failed to get sessions: "+err.Error())
			return
		}

		migratedCount := 0
		for _, sess := range sessions {
			if sess.Summary == nil {
				if err := sm.MigrateSession(sess.ID); err == nil {
					migratedCount++
				}
			}
		}

		respondWithJSON(w, http.StatusOK, map[string]int{"migrated": migratedCount})
	}
}

func GenerateSummaryFromFile(logPath string) (*FightSummary, error) {
	file, err := os.Open(logPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// PASS 1: Get all entity appearances for name/race lookups later.
	entitiesInLog := make(map[string]eventEntityAppear)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var event eventBase
		if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
			continue
		}
		if event.EventId == eventIdEntityAppear {
			var appearEvent eventEntityAppear
			if err := json.Unmarshal(scanner.Bytes(), &appearEvent); err != nil {
				continue
			}
			entitiesInLog[appearEvent.Id] = appearEvent
			playerCache.UpdateFromAppear(appearEvent)
		}
	}

	// Reset file reader to the beginning for the next pass.
	file.Seek(0, 0)
	scanner = bufio.NewScanner(file)

	// PASS 2: Collect all damage events AND track condition history
	var allDamageEvents []eventDamage

	// Condition Tracking Data Structures
	conditionHistory := make(map[string]map[uint32][]ConditionInterval)
	activeConditions := make(map[string]map[uint32]ActiveCondition)
	seenAppear := make(map[string]bool)
	targetSeenAppear := make(map[string]bool)
	targetDisappeared := make(map[string]bool)
	targetSeenDead := make(map[string]bool)
	// Skill Uses Data Structure
	skillUsesByPlayer := make(map[string][]SkillUseEvent)
	// HP history tracking by Target/Entity ID
	hpHistoryByTarget := make(map[string][]TargetHPPoint)

	for scanner.Scan() {
		line := scanner.Bytes()
		var baseEvent eventBase
		if err := json.Unmarshal(line, &baseEvent); err != nil {
			continue
		}

		switch baseEvent.EventId {
		case eventIdDamage:
			var damageEvent eventDamage
			if err := json.Unmarshal(line, &damageEvent); err == nil {
				allDamageEvents = append(allDamageEvents, damageEvent)
			}

		case eventIdEntityAppear:
			var appear eventEntityAppear
			if err := json.Unmarshal(line, &appear); err == nil {
				seenAppear[appear.Id] = true
				targetSeenAppear[appear.Id] = true
				targetDisappeared[appear.Id] = false

				// Initialize conditions present on appearance
				if activeConditions[appear.Id] == nil {
					activeConditions[appear.Id] = make(map[uint32]ActiveCondition)
				}
				for _, cond := range appear.Conditions {
					if _, exists := activeConditions[appear.Id][cond.CCId]; !exists {
						activeConditions[appear.Id][cond.CCId] = ActiveCondition{
							Start:      baseEvent.At,
							MetaData:   normalizeMetaData(cond.MetaData),
							AttackerID: parseUint64(cond.AttackerId),
						}
					}
				}

				// Capture initial HP if available
				if appear.MaxHP > 0 {
					hpHistoryByTarget[appear.Id] = append(hpHistoryByTarget[appear.Id], TargetHPPoint{
						Time:      baseEvent.At,
						CurrentHP: appear.CurrentHP,
						MaxHP:     appear.MaxHP,
					})
				}
			}

		case eventIdEntityDisappear:
			var disappear eventEntityDisappear
			if err := json.Unmarshal(line, &disappear); err == nil {
				if !targetSeenDead[disappear.Id] {
					targetDisappeared[disappear.Id] = true
				}
			}

		case eventIdEntityDeath:
			var death eventEntityDeath
			if err := json.Unmarshal(line, &death); err == nil {
				targetSeenDead[death.Id] = true
				targetDisappeared[death.Id] = false
			}

		case eventIdEntityRevive:
			var revive eventEntityRevive
			if err := json.Unmarshal(line, &revive); err == nil {
				targetSeenDead[revive.Id] = false
			}

		case eventIdCharacterConditionEnable:
			var cond eventCharacterConditionEnable
			if err := json.Unmarshal(line, &cond); err == nil {
				if activeConditions[cond.Id] == nil {
					activeConditions[cond.Id] = make(map[uint32]ActiveCondition)
				}
				if _, exists := activeConditions[cond.Id][cond.CCId]; !exists {
					activeConditions[cond.Id][cond.CCId] = ActiveCondition{
						Start:      baseEvent.At,
						MetaData:   normalizeMetaData(cond.MetaData),
						AttackerID: parseUint64(cond.AttackerId),
					}
				}
			}

		case eventIdCharacterConditionDisable:
			var cond eventCharacterConditionDisable
			if err := json.Unmarshal(line, &cond); err == nil {
				if activeConditions[cond.Id] == nil {
					activeConditions[cond.Id] = make(map[uint32]ActiveCondition)
				}
				if activeCond, exists := activeConditions[cond.Id][cond.CCId]; exists {
					if conditionHistory[cond.Id] == nil {
						conditionHistory[cond.Id] = make(map[uint32][]ConditionInterval)
					}
					conditionHistory[cond.Id][cond.CCId] = append(conditionHistory[cond.Id][cond.CCId], ConditionInterval{
						Start:      activeCond.Start,
						End:        baseEvent.At,
						MetaData:   activeCond.MetaData,
						AttackerID: activeCond.AttackerID,
					})
					delete(activeConditions[cond.Id], cond.CCId)
				}
			}

		case eventIdSkillUse, eventIdSkillStart:
			var skillEvent struct {
				eventBase
				SkillId  uint16 `json:"skillId"`
				TargetId string `json:"targetId"`
			}
			if err := json.Unmarshal(line, &skillEvent); err == nil {
				targetID := parseUint64(skillEvent.TargetId)
				casterID := parseUint64(skillEvent.Id)
				if targetID == casterID {
					targetID = 0
				}
				skillUsesByPlayer[skillEvent.Id] = append(skillUsesByPlayer[skillEvent.Id], SkillUseEvent{
					SkillID:   skillEvent.SkillId,
					TargetID:  targetID,
					Timestamp: skillEvent.At,
				})
			}

		case eventIdEntityHPUpdate:
			var hpUpdate eventEntityHPUpdate
			if err := json.Unmarshal(line, &hpUpdate); err == nil {
				hpHistoryByTarget[hpUpdate.Id] = append(hpHistoryByTarget[hpUpdate.Id], TargetHPPoint{
					Time:      baseEvent.At,
					CurrentHP: hpUpdate.CurrentHP,
					MaxHP:     hpUpdate.MaxHP,
				})
			}
		}
	}

	// STEP 1: Process the collected events to generate the main summary tables.
	playerStats, damageTaken, talents, talentNames, talentColors, targets, startTime, endTime, targetTimestamps := processEventsForSummary(allDamageEvents, entitiesInLog)

	// Ensure players who only used skills but did no damage are in playerStats
	for playerIDStr := range skillUsesByPlayer {
		if _, exists := playerStats[playerIDStr]; !exists {
			playerID := parseUint64(playerIDStr)
			if playerInfo, isPlayer := playerCache.Get(playerID); isPlayer {
				playerStats[playerIDStr] = &PlayerStats{
					ID:             playerIDStr,
					Name:           playerInfo.Name,
					OverallStats:   newDamageBreakdown(),
					DamageByTarget: make(map[string]DamageBreakdown),
				}
			}
		}
	}

	// STEP 2: Generate graph data
	graphDataByTarget := make(map[string]map[string][]GraphDataPoint)
	if len(allDamageEvents) > 0 {
		graphDataByTarget[""] = generateGraphDataFromEvents(allDamageEvents, startTime, endTime)
		for targetId := range targets {
			var targetDamageEvents []eventDamage
			for _, de := range allDamageEvents {
				if de.TargetId == targetId {
					targetDamageEvents = append(targetDamageEvents, de)
				}
			}
			if len(targetDamageEvents) > 0 {
				targetStartTime := targetDamageEvents[0].At
				targetEndTime := targetDamageEvents[len(targetDamageEvents)-1].At
				graphDataByTarget[targetId] = generateGraphDataFromEvents(targetDamageEvents, targetStartTime, targetEndTime)
			}
		}
	}

	// STEP 3: Finalize the summary object, attaching conditions and skill uses
	summary := finalizeSummaryFromLog(
		playerStats,
		damageTaken,
		talents,
		talentNames,
		talentColors,
		targets,
		entitiesInLog,
		startTime,
		endTime,
		targetTimestamps,
		conditionHistory,
		activeConditions,
		seenAppear,
		targetSeenAppear,
		targetDisappeared,
		targetSeenDead,
		skillUsesByPlayer,
		hpHistoryByTarget,
	)
	summary.GraphData = graphDataByTarget

	return &summary, nil
}

// processEventsForSummary takes the raw list of damage events and entity data from a log file
// and processes it into structured data needed for the summary view.
func processEventsForSummary(allDamageEvents []eventDamage, entitiesInLog map[string]eventEntityAppear) (
	// RETURN VALUES:
	playerStats map[string]*PlayerStats,
	damageTakenInLog map[string]*PlayerDamageTakenStats,
	playerTalentsInLog map[string]string,
	playerTalentNamesInLog map[string]string, // NEW
	playerTalentColorsInLog map[string]string, // NEW
	uniqueTargets map[string]bool,
	encounterStartTime int64,
	encounterEndTime int64,
	targetTimestamps map[string]struct {
		StartTime int64
		EndTime   int64
	},
) {
	// Initialize all the maps and variables we're going to populate.
	playerStats = make(map[string]*PlayerStats)
	damageTakenInLog = make(map[string]*PlayerDamageTakenStats)
	playerTalentsInLog = make(map[string]string)
	playerTalentNamesInLog = make(map[string]string)
	playerTalentColorsInLog = make(map[string]string)
	uniqueTargets = make(map[string]bool)
	targetTimestamps = make(map[string]struct {
		StartTime int64
		EndTime   int64
	})

	// If there are no damage events, we can return early.
	if len(allDamageEvents) == 0 {
		return
	}

	// The first and last events in the log determine the overall encounter time.
	encounterStartTime = allDamageEvents[0].At
	encounterEndTime = allDamageEvents[len(allDamageEvents)-1].At

	// Request-scoped cache for player lookups
	playerLookup := make(map[uint64]*PlayerInfo)
	isPlayerLookup := make(map[uint64]bool)

	// Helper to get player info with caching
	getPlayer := func(id uint64) (*PlayerInfo, bool) {
		if res, ok := isPlayerLookup[id]; ok {
			return playerLookup[id], res
		}
		info, isP := playerCache.Get(id)
		isPlayerLookup[id] = isP
		if isP {
			playerLookup[id] = info
		}
		return info, isP
	}

	// Iterate over every single damage event that occurred in the log.
	for _, damageEvent := range allDamageEvents {

		// Update the shared combat timer for the specific target that was hit.
		targetIDStr := damageEvent.TargetId
		timestamps := targetTimestamps[targetIDStr]
		if timestamps.StartTime == 0 {
			timestamps.StartTime = damageEvent.At
		}
		timestamps.EndTime = damageEvent.At
		targetTimestamps[targetIDStr] = timestamps

		// Check if we can identify the attacker's talent from the skill they used.
		// Logic updated to ensure we capture Name/Color even if Icon is already found (though unlikely to differ)
		if _, knownIcon := playerTalentsInLog[damageEvent.Id]; !knownIcon {
			if iconPath, found := skillToArcanaIcon[damageEvent.SkillId]; found {
				playerTalentsInLog[damageEvent.Id] = iconPath
			}
		}
		if _, knownName := playerTalentNamesInLog[damageEvent.Id]; !knownName {
			if name, found := skillToArcanaName[damageEvent.SkillId]; found {
				playerTalentNamesInLog[damageEvent.Id] = name
			}
		}
		if _, knownColor := playerTalentColorsInLog[damageEvent.Id]; !knownColor {
			if color, found := skillToArcanaColor[damageEvent.SkillId]; found {
				playerTalentColorsInLog[damageEvent.Id] = color
			}
		}

		// --- A) PROCESS DAMAGE DEALT ---
		// Check if the attacker is a known player.
		attackerId := parseUint64(damageEvent.Id)
		if playerInfo, isPlayer := getPlayer(attackerId); isPlayer {
			// Mark the target as having been engaged in combat.
			uniqueTargets[damageEvent.TargetId] = true

			// Get or create the stats object for this player.
			stats, exists := playerStats[damageEvent.Id]
			if !exists {
				stats = &PlayerStats{
					ID:             damageEvent.Id,
					Name:           playerInfo.Name,
					OverallStats:   newDamageBreakdown(),
					DamageByTarget: make(map[string]DamageBreakdown),
				}
				playerStats[damageEvent.Id] = stats
			}

			// Update the player's overall damage stats.
			updateBreakdownFromLog(&stats.OverallStats, &damageEvent)

			// Update the player's damage stats against this specific target.
			targetBreakdown, exists := stats.DamageByTarget[damageEvent.TargetId]
			if !exists {
				targetBreakdown = newDamageBreakdown()
			}
			updateBreakdownFromLog(&targetBreakdown, &damageEvent)
			stats.DamageByTarget[damageEvent.TargetId] = targetBreakdown
		}

		// --- B) PROCESS DAMAGE TAKEN ---
		// Check if the entity that was hit is a known player.
		targetId := parseUint64(damageEvent.TargetId)
		if _, isPlayerTarget := getPlayer(targetId); isPlayerTarget {
			// If so, update the damage taken records.
			updateDamageTakenFromLog(&damageEvent, damageTakenInLog, entitiesInLog)
		}
	}

	// Return all the populated data structures.
	return
}

// NEW FUNCTION: The core logic for generating graph data.
func generateGraphDataFromEvents(allDamageEvents []eventDamage, startTime, endTime int64) map[string][]GraphDataPoint {
	if len(allDamageEvents) == 0 {
		return nil
	}

	const graphInterval int64 = 2 // seconds - our sampling rate
	const dpsWindow int64 = 60   // seconds - for the rolling DPS calculation

	graphData := make(map[string][]GraphDataPoint)

	// Request-scoped cache
	playerLookup := make(map[uint64]*PlayerInfo)
	isPlayerLookup := make(map[uint64]bool)
	getPlayer := func(id uint64) (*PlayerInfo, bool) {
		if res, ok := isPlayerLookup[id]; ok {
			return playerLookup[id], res
		}
		info, isP := playerCache.Get(id)
		isPlayerLookup[id] = isP
		if isP {
			playerLookup[id] = info
		}
		return info, isP
	}

	// Create a map for quick access to each player's damage events
	playerEvents := make(map[string][]eventDamage)
	for _, de := range allDamageEvents {
		if _, isPlayer := getPlayer(parseUint64(de.Id)); isPlayer {
			playerEvents[de.Id] = append(playerEvents[de.Id], de)
		}
	}

	// Iterate through time in steps of our interval
	for t := startTime; t <= endTime; t += graphInterval {
		// For each player, calculate their stats at this point in time
		for playerID, events := range playerEvents {
			var totalDamage float32
			var damageInWindow float32

			for _, de := range events {
				if de.At <= t {
					totalDamage += de.Damage
					if de.At > t-dpsWindow {
						damageInWindow += de.Damage
					}
				}
			}

			point := GraphDataPoint{
				Time:        t - startTime, // Relative time in seconds from start
				TotalDamage: totalDamage,
				RollingDPS:  damageInWindow / float32(dpsWindow),
			}
			graphData[playerID] = append(graphData[playerID], point)
		}
	}
	return graphData
}

func updateBreakdownFromLog(breakdown *DamageBreakdown, damageEvent *eventDamage) {
	breakdown.TotalDamage += damageEvent.Damage
	if !damageEvent.IsDelayed {
		breakdown.HitCount++
		if damageEvent.IsCritical {
			breakdown.CritCount++
		}
	}
	skillStats := breakdown.Skills[damageEvent.SkillId]
	skillStats.ID = damageEvent.SkillId
	skillStats.TotalDamage += damageEvent.Damage
	isPrimaryHit := !damageEvent.IsDelayed
	isCountableDelayedHit := damageEvent.IsDelayed && doCountDelayedSkills[damageEvent.SkillId]
	if isPrimaryHit || isCountableDelayedHit {
		skillStats.Count++
	}
	if damageEvent.IsCritical {
		skillStats.CritCount++
		skillStats.TotalDamageCrit += damageEvent.Damage
		if damageEvent.Damage > skillStats.MaxDamageCrit {
			skillStats.MaxDamageCrit = damageEvent.Damage
		}
	} else {
		skillStats.TotalDamageNonCrit += damageEvent.Damage
		if damageEvent.Damage > skillStats.MaxDamageNonCrit {
			skillStats.MaxDamageNonCrit = damageEvent.Damage
		}
	}
	if damageEvent.Damage > skillStats.MaxDamage {
		skillStats.MaxDamage = damageEvent.Damage
	}
	breakdown.Skills[damageEvent.SkillId] = skillStats
}

// Updated finalizeBreakdownFromLog to include condition tracking and skill uses
func finalizeBreakdownFromLog(
	breakdown DamageBreakdown,
	duration float64,
	windowStart, windowEnd int64,
	playerID string,
	conditionHistory map[string]map[uint32][]ConditionInterval,
	activeConditions map[string]map[uint32]ActiveCondition,
	skillUses []SkillUseEvent,
	isOverall bool,
	targetID uint64,
	targetTimestamps map[string]struct {
		StartTime int64
		EndTime   int64
	},
) DamageBreakdown {
	if duration > 1 {
		breakdown.DPS = breakdown.TotalDamage / float32(duration)
	} else if breakdown.TotalDamage > 0 {
		breakdown.DPS = breakdown.TotalDamage
	}
	if breakdown.HitCount > 0 {
		breakdown.CritRate = (float32(breakdown.CritCount) / float32(breakdown.HitCount)) * 100
	}

	// --- CONDITION TRACKING LOGIC (Mirrors Aggregator) ---
	breakdown.Conditions = calculateConditionsFromLog(playerID, duration, windowStart, windowEnd, conditionHistory, activeConditions)

	// --- SKILL USES COUNTING LOGIC (Mirrors Aggregator) ---
	skillUsesCount := make(map[uint16]int)
	for _, use := range skillUses {
		if isOverall {
			skillUsesCount[use.SkillID]++
		} else {
			resolvedTargetID := use.TargetID
			if resolvedTargetID != 0 {
				targetIDStr := strconv.FormatUint(resolvedTargetID, 10)
				_, isTracked := targetTimestamps[targetIDStr]
				_, isPlayer := playerCache.Get(resolvedTargetID)
				if !isTracked && !isPlayer {
					resolvedTargetID = 0
				}
			}

			if resolvedTargetID == targetID || resolvedTargetID == 0 {
				targetIDStr := strconv.FormatUint(targetID, 10)
				if times, ok := targetTimestamps[targetIDStr]; ok && times.StartTime > 0 {
					end := times.EndTime
					if end == 0 || end < times.StartTime {
						end = use.Timestamp
					}
					if use.Timestamp >= times.StartTime && use.Timestamp <= end {
						skillUsesCount[use.SkillID]++
					}
				}
			}
		}
	}

	if breakdown.Skills == nil {
		breakdown.Skills = make(map[uint16]SkillStats)
	}

	for skillID, uses := range skillUsesCount {
		skillStats := breakdown.Skills[skillID]
		skillStats.ID = skillID
		skillStats.Uses = uses
		breakdown.Skills[skillID] = skillStats
	}

	return breakdown
}

func calculateConditionsFromLog(
	entityID string,
	duration float64,
	windowStart, windowEnd int64,
	conditionHistory map[string]map[uint32][]ConditionInterval,
	activeConditions map[string]map[uint32]ActiveCondition,
) map[uint32]*ConditionStats {
	conditions := make(map[uint32]*ConditionStats)
	allIntervals := make(map[uint32][]ConditionInterval)

	// 1. Historical intervals
	if history, ok := conditionHistory[entityID]; ok {
		for ccID, intervals := range history {
			allIntervals[ccID] = append(allIntervals[ccID], intervals...)
		}
	}

	// 2. Currently active intervals (Start -> End of Window)
	if active, ok := activeConditions[entityID]; ok {
		for ccID, activeCond := range active {
			if activeCond.Start < windowEnd {
				allIntervals[ccID] = append(allIntervals[ccID], ConditionInterval{
					Start:      activeCond.Start,
					End:        windowEnd,
					MetaData:   activeCond.MetaData,
					AttackerID: activeCond.AttackerID,
				})
			}
		}
	}

	// 3. Calculate Intersection
	for ccID, intervals := range allIntervals {
		var totalActiveTime int64
		var finalIntervals []ConditionInterval

		// Breakdown tracking
		metaStatsMap := make(map[string]*ConditionMetaStats)

		for _, iv := range intervals {
			start := iv.Start
			if start < windowStart {
				start = windowStart
			}
			end := iv.End
			if end > windowEnd {
				end = windowEnd
			}
			if start < end {
				duration := end - start
				totalActiveTime += duration

				actualInterval := ConditionInterval{
					Start:      start,
					End:        end,
					MetaData:   iv.MetaData,
					AttackerID: iv.AttackerID,
				}
				finalIntervals = append(finalIntervals, actualInterval)

				// Meta breakdown
				metaKey := iv.MetaData
				if metaKey == "" {
					metaKey = "Unknown"
				}

				if _, ok := metaStatsMap[metaKey]; !ok {
					metaStatsMap[metaKey] = &ConditionMetaStats{
						MetaData:  metaKey,
						Attackers: []uint64{},
					}
				}
				metaStatsMap[metaKey].Duration += float64(duration)

				// Add attacker if unique
				found := false
				for _, id := range metaStatsMap[metaKey].Attackers {
					if id == iv.AttackerID {
						found = true
						break
					}
				}
				if !found && iv.AttackerID != 0 {
					metaStatsMap[metaKey].Attackers = append(metaStatsMap[metaKey].Attackers, iv.AttackerID)
				}
			}
		}

		var uptimePercent float32
		if duration > 0 {
			uptimePercent = (float32(totalActiveTime) / float32(duration)) * 100.0
		}

		if totalActiveTime > 0 {
			// Convert map to slice
			var metaBreakdown []ConditionMetaStats
			for _, stats := range metaStatsMap {
				if duration > 0 {
					stats.Uptime = (float32(stats.Duration) / float32(duration)) * 100.0
				}
				metaBreakdown = append(metaBreakdown, *stats)
			}

			conditions[ccID] = &ConditionStats{
				ID:            ccID,
				Uptime:        uptimePercent,
				Duration:      float64(totalActiveTime),
				Intervals:     finalIntervals,
				MetaBreakdown: metaBreakdown,
			}
		}
	}
	return conditions
}

func parseUint64(s string) uint64 {
	val, _ := strconv.ParseUint(s, 10, 64)
	return val
}

func updateDamageTakenFromLog(damageEvent *eventDamage, damageTakenInLog map[string]*PlayerDamageTakenStats, entitiesInLog map[string]eventEntityAppear) {
	targetId := damageEvent.TargetId
	stats, exists := damageTakenInLog[targetId]
	if !exists {
		targetInfo, _ := playerCache.Get(parseUint64(targetId))
		stats = &PlayerDamageTakenStats{
			PlayerID:   targetId,
			PlayerName: targetInfo.Name,
			Breakdown:  make(map[string]DamageTakenDetails),
		}
		damageTakenInLog[targetId] = stats
	}
	stats.TotalDamage += damageEvent.Damage + damageEvent.ManaDamage
	stats.TotalManaDamage += damageEvent.ManaDamage // NEW

	breakdownKey := fmt.Sprintf("%s-%d", damageEvent.Id, damageEvent.SkillId)
	details, exists := stats.Breakdown[breakdownKey]
	if !exists {
		attackerName := "Unknown"
		if entity, ok := entitiesInLog[damageEvent.Id]; ok {
			attackerName = getRaceName(entity.RaceId)
		} else if player, ok := playerCache.Get(parseUint64(damageEvent.Id)); ok {
			attackerName = player.Name
		}
		details = DamageTakenDetails{
			AttackerID:   parseUint64(damageEvent.Id),
			AttackerName: attackerName,
			SkillID:      damageEvent.SkillId,
		}
	}
	details.TotalDamage += damageEvent.Damage + damageEvent.ManaDamage
	details.TotalManaDamage += damageEvent.ManaDamage
	details.HitCount++
	damage := damageEvent.Damage + damageEvent.ManaDamage
	if damage > details.MaxDamage {
		details.MaxDamage = damage
	}
	if details.MinDamage == 0 || damage < details.MinDamage {
		details.MinDamage = damage
	}
	stats.Breakdown[breakdownKey] = details
}

func finalizeSummaryFromLog(
	playerStats map[string]*PlayerStats,
	damageTakenInLog map[string]*PlayerDamageTakenStats,
	playerTalentsInLog map[string]string,
	playerTalentNamesInLog map[string]string, // NEW
	playerTalentColorsInLog map[string]string, // NEW
	uniqueTargets map[string]bool,
	entitiesInLog map[string]eventEntityAppear,
	encounterStartTime int64,
	encounterEndTime int64,
	targetTimestamps map[string]struct {
		StartTime int64
		EndTime   int64
	},
	conditionHistory map[string]map[uint32][]ConditionInterval,
	activeConditions map[string]map[uint32]ActiveCondition,
	seenAppear map[string]bool,
	targetSeenAppear map[string]bool,
	targetDisappeared map[string]bool,
	targetSeenDead map[string]bool,
	skillUsesByPlayer map[string][]SkillUseEvent,
	hpHistoryByTarget map[string][]TargetHPPoint, // NEW
) FightSummary {
	summary := FightSummary{
		Players:     make(map[string]PlayerStats),
		Targets:     make(map[string]TargetStats),
		DamageTaken: make(map[string]PlayerDamageTakenStats),
	}
	if encounterEndTime > encounterStartTime {
		summary.EncounterDuration = float64(encounterEndTime - encounterStartTime)
	}
	summary.StartTime = encounterStartTime
	summary.EndTime = encounterEndTime

	var totalDamage float32
	for _, pStats := range playerStats {
		finalizedPstats := *pStats
		finalizedPstats.MissingAppearPacket = !seenAppear[pStats.ID]

		playerSkillUses := skillUsesByPlayer[pStats.ID]

		overallDuration := float64(encounterEndTime - encounterStartTime)
		// Pass overall window and condition/skill data
		finalizedPstats.OverallStats = finalizeBreakdownFromLog(
			pStats.OverallStats, overallDuration, encounterStartTime, encounterEndTime, pStats.ID, conditionHistory, activeConditions, playerSkillUses, true, 0, targetTimestamps)
		finalizedPstats.OverallStats.StartTime = encounterStartTime
		finalizedPstats.OverallStats.EndTime = encounterEndTime

		finalizedPstats.DamageByTarget = make(map[string]DamageBreakdown)
		for targetId, breakdown := range pStats.DamageByTarget {
			targetTimes := targetTimestamps[targetId]
			targetDuration := float64(targetTimes.EndTime - targetTimes.StartTime)
			targetIdUint, _ := strconv.ParseUint(targetId, 10, 64)
			// Pass specific target window and condition/skill data
			finalizedBreakdown := finalizeBreakdownFromLog(
				breakdown, targetDuration, targetTimes.StartTime, targetTimes.EndTime, pStats.ID, conditionHistory, activeConditions, playerSkillUses, false, targetIdUint, targetTimestamps)
			finalizedBreakdown.StartTime = targetTimes.StartTime
			finalizedBreakdown.EndTime = targetTimes.EndTime
			finalizedPstats.DamageByTarget[targetId] = finalizedBreakdown
		}

		if icon, ok := playerTalentsInLog[pStats.ID]; ok {
			finalizedPstats.TalentIcon = icon
		}
		if name, ok := playerTalentNamesInLog[pStats.ID]; ok {
			finalizedPstats.TalentName = name
		}
		if color, ok := playerTalentColorsInLog[pStats.ID]; ok {
			finalizedPstats.TalentColor = color
		}
		summary.Players[pStats.ID] = finalizedPstats
		totalDamage += finalizedPstats.OverallStats.TotalDamage
	}

	summary.TotalDamage = totalDamage
	for _, dtStats := range damageTakenInLog {
		summary.DamageTaken[dtStats.PlayerID] = *dtStats
	}
	for targetIdStr := range uniqueTargets {
		var name string
		var raceId uint32
		if entity, ok := entitiesInLog[targetIdStr]; ok {
			name = getRaceName(entity.RaceId)
			raceId = entity.RaceId
		} else {
			name = "Unknown"
		}

		// Calculate conditions for target
		targetTimes := targetTimestamps[targetIdStr]
		targetDuration := float64(targetTimes.EndTime - targetTimes.StartTime)
		conditions := calculateConditionsFromLog(targetIdStr, targetDuration, targetTimes.StartTime, targetTimes.EndTime, conditionHistory, activeConditions)

		// Process HP history for this target (convert to relative time)
		var hpHistory []TargetHPPoint
		for _, hpPt := range hpHistoryByTarget[targetIdStr] {
			hpHistory = append(hpHistory, TargetHPPoint{
				Time:      hpPt.Time - targetTimes.StartTime,
				CurrentHP: hpPt.CurrentHP,
				MaxHP:     hpPt.MaxHP,
			})
		}

		summary.Targets[targetIdStr] = TargetStats{
			Name:        name,
			RaceID:      raceId,
			Conditions:  conditions,
			SeenDead:    targetSeenDead[targetIdStr],
			SeenAppear:  targetSeenAppear[targetIdStr],
			Disappeared: targetDisappeared[targetIdStr],
			StartTime:   targetTimes.StartTime,
			EndTime:     targetTimes.EndTime,
			HPHistory:   hpHistory,
		}
	}

	computePartyBuffs(&summary)

	return summary
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, _ := json.Marshal(payload)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}

func respondWithError(w http.ResponseWriter, code int, message string) {
	respondWithJSON(w, code, map[string]string{"error": message})
}

func handleDeleteSession(sm *SessionManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sessionID := chi.URLParam(r, "sessionID")
		if err := sm.DeleteSession(sessionID); err != nil {
			respondWithError(w, http.StatusInternalServerError, "Failed to delete session: "+err.Error())
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
