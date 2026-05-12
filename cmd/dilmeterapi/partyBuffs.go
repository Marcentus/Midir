package main

import (
	"strconv"
	"strings"
)

// ParseMabinogiMetadata parses a Mabinogi metadata string into a key-value record for float values.
// Format: KEY:TYPE:VALUE;KEY:TYPE:VALUE;...
func ParseMabinogiMetadata(metadata string) map[string]float32 {
	result := make(map[string]float32)
	if metadata == "" {
		return result
	}

	pairs := strings.Split(metadata, ";")
	for _, pair := range pairs {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}

		parts := strings.SplitN(pair, ":", 3)
		if len(parts) < 3 {
			continue
		}

		key := parts[0]
		// typeStr := parts[1]
		rawValue := parts[2]

		// For buff calculations, we only care about numeric types ("f", "i", "d", "8")
		// We can just try to parse it as a float32 regardless.
		if val, err := strconv.ParseFloat(rawValue, 32); err == nil {
			result[key] = float32(val)
		}
	}

	return result
}

// computePartyBuffs processes the players' conditions in FightSummary to extract 
// high-level party buff metrics (BFO, Vivace) for the dashboard.
func computePartyBuffs(summary *FightSummary) {
	ids := []uint32{680, 192} // 680 = BFO, 192 = Vivace
	var results []PartyBuff

	if summary.Players == nil || len(summary.Players) == 0 {
		return
	}

	for _, id := range ids {
		var keys []string
		if id == 680 {
			keys = []string{"MCMBAMAX"}
		} else {
			keys = []string{"LSMA", "MFCP"}
		}

		type metricData struct {
			highest         float32
			highestUptime   float32
			sumVDGlobal     float64
			sumDGlobal      float64
			maxUptimeParty  float32
		}

		metrics := make(map[string]*metricData)
		for _, k := range keys {
			metrics[k] = &metricData{maxUptimeParty: -1.0}
		}

		var maxDurForThisID float64 = 0
		hasMajorityPlayer := false

		// Pass 1: Gather global metadata trends and majority status
		for _, player := range summary.Players {
			// We look at OverallStats for Party Buffs globally
			stats := player.OverallStats

			var durThis, durOther float64
			if condBFO, ok := stats.Conditions[680]; ok {
				if id == 680 {
					durThis = condBFO.Duration
				} else {
					durOther = condBFO.Duration
				}
			}
			if condVivace, ok := stats.Conditions[192]; ok {
				if id == 192 {
					durThis = condVivace.Duration
				} else {
					durOther = condVivace.Duration
				}
			}

			if durThis > maxDurForThisID {
				maxDurForThisID = durThis
			}
			if durThis > 0 && durThis >= durOther {
				hasMajorityPlayer = true
			}

			if cond, ok := stats.Conditions[id]; ok {
				for _, meta := range cond.MetaBreakdown {
					parsed := ParseMabinogiMetadata(meta.MetaData)
					for _, key := range keys {
						if val, found := parsed[key]; found && val > 0 {
							m := metrics[key]
							if val > m.highest {
								m.highest = val
							}
							m.sumVDGlobal += float64(val) * meta.Duration
							m.sumDGlobal += meta.Duration
							if meta.Uptime > m.maxUptimeParty {
								m.maxUptimeParty = meta.Uptime
								m.highestUptime = val
							}
						}
					}
				}
			}
		}

		// Pass 2: Calculate damage-weighted average
		sumVDWeighted := make(map[string]float64)
		sumDRecipient := make(map[string]float64)

		for _, player := range summary.Players {
			stats := player.OverallStats
			playerDamage := float64(stats.TotalDamage)
			if playerDamage <= 0 {
				continue
			}

			var durThis, durOther float64
			if condBFO, ok := stats.Conditions[680]; ok {
				if id == 680 {
					durThis = condBFO.Duration
				} else {
					durOther = condBFO.Duration
				}
			}
			if condVivace, ok := stats.Conditions[192]; ok {
				if id == 192 {
					durThis = condVivace.Duration
				} else {
					durOther = condVivace.Duration
				}
			}

			isEligible := false
			if durThis > 0 && durThis >= durOther {
				isEligible = true
			} else if !hasMajorityPlayer && durThis > 0 && durThis >= maxDurForThisID {
				isEligible = true
			}

			if !isEligible {
				continue
			}

			cond, ok := stats.Conditions[id]
			if !ok {
				continue
			}

			for _, key := range keys {
				var playerAvg float64 = 0
				hasLocalData := false

				if len(cond.MetaBreakdown) > 0 {
					var pSumVD float64 = 0
					var pSumD float64 = 0
					for _, meta := range cond.MetaBreakdown {
						parsed := ParseMabinogiMetadata(meta.MetaData)
						if val, found := parsed[key]; found && val > 0 {
							pSumVD += float64(val) * meta.Duration
							pSumD += meta.Duration
							hasLocalData = true
						}
					}
					if pSumD > 0 {
						playerAvg = pSumVD / pSumD
					}
				}

				if !hasLocalData && metrics[key].sumDGlobal > 0 {
					playerAvg = metrics[key].sumVDGlobal / metrics[key].sumDGlobal
				}

				if playerAvg > 0 {
					sumVDWeighted[key] += playerAvg * playerDamage
					sumDRecipient[key] += playerDamage
				}
			}
		}

		// Finalize metrics
		var formattedMetrics []PartyBuffMetric
		for _, key := range keys {
			m := metrics[key]
			if m.highest > 0 {
				var weightedAvg float64 = 0
				if sumDRecipient[key] > 0 {
					weightedAvg = sumVDWeighted[key] / sumDRecipient[key]
				}

				label := key
				if key == "MCMBAMAX" {
					label = "Max Att"
				} else if key == "LSMA" {
					label = "Magic Att"
				} else if key == "MFCP" {
					label = "Cast Speed"
				}

				formattedMetrics = append(formattedMetrics, PartyBuffMetric{
					Label:         label,
					Highest:       m.highest,
					HighestUptime: m.highestUptime,
					WeightedAvg:   float32(weightedAvg),
				})
			}
		}

		if len(formattedMetrics) > 0 {
			results = append(results, PartyBuff{
				ID:      id,
				Metrics: formattedMetrics,
			})
		}
	}

	summary.PartyBuffs = results
}
