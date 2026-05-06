import { ConditionStats, ConditionInterval } from "./protocols";

export interface ConditionCombination {
  id: number;
  name: string;
  iconId: number; // ID of the condition to take the icon from
  subConditionIds: number[];
}

export const CONDITION_COMBINATIONS: ConditionCombination[] = [
  {
    id: 200001,
    name: "Lightning Debuff",
    iconId: 392,
    subConditionIds: [392, 504],
  },
  {
    id: 200002,
    name: "Purrling's Furious Charge",
    iconId: 912,
    subConditionIds: [912, 913],
  },
];

export interface ExtendedConditionStats extends ConditionStats {
  isCombined?: boolean;
  subConditions?: ConditionStats[];
}

/**
 * Merges overlapping and adjacent time intervals.
 */
export function mergeIntervals(intervals: ConditionInterval[]): ConditionInterval[] {
  if (intervals.length === 0) return [];

  // Sort by start time
  const sorted = [...intervals].sort((a, b) => a.start - b.start);
  const merged: ConditionInterval[] = [];

  let current = { ...sorted[0] };

  for (let i = 1; i < sorted.length; i++) {
    const next = sorted[i];
    if (next.start <= current.end) {
      // Overlapping or adjacent, merge them
      current.end = Math.max(current.end, next.end);
    } else {
      // Disjoint, push current and start new
      merged.push(current);
      current = { ...next };
    }
  }
  merged.push(current);

  return merged;
}

/**
 * Processes a record of conditions, combining those specified in CONDITION_COMBINATIONS.
 */
export function processConditions(
  conditions: Record<number, ConditionStats> | undefined,
  encounterDuration: number
): ExtendedConditionStats[] {
  if (!conditions) return [];

  const combinedIds = new Set<number>();
  CONDITION_COMBINATIONS.forEach((c) => {
    c.subConditionIds.forEach((id) => combinedIds.add(id));
  });

  const result: ExtendedConditionStats[] = [];

  // Add standalone conditions that are not part of any combination
  Object.keys(conditions).forEach((idStr) => {
    const id = parseInt(idStr, 10);
    if (!combinedIds.has(id)) {
      result.push({ ...conditions[id] });
    }
  });

  // Add combined conditions
  CONDITION_COMBINATIONS.forEach((combo) => {
    const subStats = combo.subConditionIds
      .map((id) => conditions[id])
      .filter((s) => !!s);

    if (subStats.length > 0) {
      // Merge all intervals from all sub-conditions
      const allIntervals = subStats.flatMap((s) => s.intervals || []);
      const mergedIntervals = mergeIntervals(allIntervals);

      // Combined duration from intervals (union)
      const intervalDuration = mergedIntervals.reduce(
        (sum, iv) => sum + (iv.end - iv.start),
        0
      );

      // Individual maximums and sums (for fallback heuristics)
      const maxSubDuration = Math.max(...subStats.map(s => s.duration), 0);
      const maxSubUptime = Math.max(...subStats.map(s => s.uptime), 0);
      const sumSubDuration = subStats.reduce((sum, s) => sum + s.duration, 0);
      const sumSubUptime = subStats.reduce((sum, s) => sum + s.uptime, 0);

      // Infer the denominator used by the backend for these conditions (e.g., target lifespan)
      // This ensures our combined percentage matches the sub-condition percentages.
      let effectiveDenominator = encounterDuration;
      const subWithData = subStats.find(s => s.uptime > 0 && s.duration > 0);
      if (subWithData) {
        effectiveDenominator = (subWithData.duration / subWithData.uptime) * 100;
      }

      // Logic: 
      // 1. Start with the interval union (most accurate if intervals are complete)
      // 2. Ensure it's at least the max of any single condition
      // 3. If intervals are missing, fallback to the sum of durations (capped at denominator)
      let totalDuration = Math.max(intervalDuration, maxSubDuration);
      if (effectiveDenominator > 0) {
        totalDuration = Math.max(totalDuration, Math.min(effectiveDenominator, sumSubDuration));
      } else {
        totalDuration = Math.max(totalDuration, sumSubDuration);
      }

      // Uptime calculation
      let uptime = 0;
      if (effectiveDenominator > 0) {
        uptime = (totalDuration / effectiveDenominator) * 100;
      } else {
        uptime = Math.max(maxSubUptime, Math.min(100, sumSubUptime));
      }
      
      // Ensure we don't drop below the known max or exceed 100
      uptime = Math.min(100, Math.max(uptime, maxSubUptime));

      result.push({
        id: combo.id,
        uptime: uptime,
        duration: totalDuration,
        intervals: mergedIntervals,
        isCombined: true,
        subConditions: subStats,
      });
    }
  });

  return result;
}
