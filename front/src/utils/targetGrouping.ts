import {
  PlayerStats,
  DamageBreakdown,
  SkillStats,
  TargetStats,
  ConditionStats,
  GraphDataPoint,
  FightSummary,
} from "@/protocols";

export interface GroupItem {
  id: string;
  name: string;
  rawName: string;
  isGroup: boolean;
  damage: number;
  damagePercent: number;
  currentHp: number;
  maxHp: number;
  hpPercent: number;
  seenAppear?: boolean;
  seenDead?: boolean;
  disappeared?: boolean;
  startTime?: number;
  endTime?: number;
  raceId?: number;
  hasHpUpdates: boolean;
  targets: any[];
}

export const GROUPED_ENCOUNTERS = [
  { bossRaceId: 7615, groupName: "Bri Leith: Gate 4" },
  { bossRaceId: 7603, groupName: "Bri Leith: Gate 3" },
  { bossRaceId: 7600, partnerRaceId: 7601, groupName: "Bri Leith: Gate 1" },
];

/**
 * Checks whether a targetId corresponds to a group (starts with "group_").
 */
export function isGroupTargetId(targetId: string): boolean {
  return typeof targetId === "string" && targetId.startsWith("group_");
}

/**
 * Finds member target IDs for a given group target ID from fightSummary.
 */
export function getGroupMemberTargetIds(
  targetId: string,
  fightSummary: FightSummary
): string[] {
  if (!isGroupTargetId(targetId)) {
    return targetId ? [targetId] : [];
  }

  const bossTargetId = targetId.replace("group_", "");
  const targetsMap = fightSummary.targets || {};
  const bossTarget = targetsMap[bossTargetId];
  if (!bossTarget) return [];

  // Find bossDef
  const bossDef = GROUPED_ENCOUNTERS.find(
    (e) => e.bossRaceId === bossTarget.raceId
  );
  if (!bossDef) {
    // Fallback if not matching raceId
    return [bossTargetId];
  }

  const allTargets = Object.entries(targetsMap).map(([id, stats]) => ({
    id,
    ...stats,
  }));

  const bossStartTime = bossTarget.startTime || 0;
  let bossEndTime =
    bossTarget.seenDead || bossTarget.disappeared
      ? bossTarget.endTime || 0
      : Infinity;

  let partnerTarget: any = null;
  if (bossDef.partnerRaceId) {
    partnerTarget = allTargets.find(
      (t) =>
        t.raceId === bossDef.partnerRaceId && (t.startTime || 0) >= bossStartTime
    );
    if (partnerTarget) {
      bossEndTime =
        partnerTarget.seenDead || partnerTarget.disappeared
          ? partnerTarget.endTime || 0
          : Infinity;
    }
  }

  const members = allTargets.filter((t) => {
    const tStartTime = t.startTime || 0;
    return (
      tStartTime >= bossStartTime &&
      (bossEndTime === Infinity || tStartTime <= bossEndTime)
    );
  });

  return members.map((m) => m.id);
}

/**
 * Aggregates a player's DamageBreakdown across a list of target IDs.
 */
export function aggregatePlayerDamageForGroup(
  player: PlayerStats,
  targetIds: string[]
): DamageBreakdown {
  let totalDamage = 0;
  let hitCount = 0;
  let critCount = 0;
  let minStart = Infinity;
  let maxEnd = 0;
  const aggregatedSkills: { [skillId: number]: SkillStats } = {};

  for (const tid of targetIds) {
    const bd = player.damageByTarget?.[tid];
    if (!bd) continue;

    totalDamage += bd.totalDamage || 0;
    hitCount += bd.hitCount || 0;
    critCount += bd.critCount || 0;

    if (bd.startTime && bd.startTime < minStart) {
      minStart = bd.startTime;
    }
    if (bd.endTime && bd.endTime > maxEnd) {
      maxEnd = bd.endTime;
    }

    if (bd.skills) {
      for (const [skillIdStr, skill] of Object.entries(bd.skills)) {
        const skillId = Number(skillIdStr);
        if (!aggregatedSkills[skillId]) {
          aggregatedSkills[skillId] = { ...skill };
        } else {
          const s = aggregatedSkills[skillId];
          s.count = (s.count || 0) + (skill.count || 0);
          s.critCount = (s.critCount || 0) + (skill.critCount || 0);
          s.totalDamage = (s.totalDamage || 0) + (skill.totalDamage || 0);
          s.totalDamageCrit = (s.totalDamageCrit || 0) + (skill.totalDamageCrit || 0);
          s.totalDamageNonCrit = (s.totalDamageNonCrit || 0) + (skill.totalDamageNonCrit || 0);
          s.maxDamage = Math.max(s.maxDamage || 0, skill.maxDamage || 0);
          s.maxDamageCrit = Math.max(s.maxDamageCrit || 0, skill.maxDamageCrit || 0);
          s.maxDamageNonCrit = Math.max(s.maxDamageNonCrit || 0, skill.maxDamageNonCrit || 0);
          if (skill.uses !== undefined) {
            s.uses = (s.uses || 0) + skill.uses;
          }
        }
      }
    }
  }

  const duration =
    maxEnd > minStart && minStart !== Infinity ? maxEnd - minStart : 0;
  const dps = duration > 0 ? totalDamage / duration : 0;
  const critRate = hitCount > 0 ? (critCount / hitCount) * 100 : 0;

  return {
    totalDamage,
    hitCount,
    critCount,
    dps,
    critRate,
    startTime: minStart !== Infinity ? minStart : undefined,
    endTime: maxEnd !== 0 ? maxEnd : undefined,
    skills: aggregatedSkills,
  };
}

/**
 * Calculates start time, end time, and total encounter duration for a set of target IDs.
 */
export function getEncounterDurationForTargets(
  players: Record<string, PlayerStats>,
  targetIds: string[]
): number {
  let earliestStart = Infinity;
  let latestEnd = 0;
  let hasData = false;

  for (const player of Object.values(players)) {
    for (const tid of targetIds) {
      const bd = player.damageByTarget?.[tid];
      if (bd && bd.startTime && bd.endTime) {
        hasData = true;
        if (bd.startTime < earliestStart) {
          earliestStart = bd.startTime;
        }
        if (bd.endTime > latestEnd) {
          latestEnd = bd.endTime;
        }
      }
    }
  }

  if (hasData && latestEnd > earliestStart) {
    return latestEnd - earliestStart;
  }
  return 0;
}

/**
 * Merges target conditions across multiple target IDs.
 */
export function aggregateGroupConditions(
  targets: Record<string, TargetStats>,
  targetIds: string[]
): Record<number, ConditionStats> | undefined {
  const mergedConditions: Record<number, ConditionStats> = {};
  let foundAny = false;

  for (const tid of targetIds) {
    const t = targets[tid];
    if (!t || !t.conditions) continue;

    for (const [condIdStr, cond] of Object.entries(t.conditions)) {
      const condId = Number(condIdStr);
      foundAny = true;
      if (!mergedConditions[condId]) {
        mergedConditions[condId] = {
          id: cond.id,
          uptime: cond.uptime,
          duration: cond.duration,
          intervals: cond.intervals ? [...cond.intervals] : [],
          metaBreakdown: cond.metaBreakdown ? [...cond.metaBreakdown] : [],
        };
      } else {
        const existing = mergedConditions[condId];
        existing.uptime = Math.max(existing.uptime, cond.uptime);
        existing.duration = Math.max(existing.duration, cond.duration);
        if (cond.intervals) {
          existing.intervals = (existing.intervals || []).concat(cond.intervals);
        }
        if (cond.metaBreakdown) {
          existing.metaBreakdown = (existing.metaBreakdown || []).concat(cond.metaBreakdown);
        }
      }
    }
  }

  return foundAny ? mergedConditions : undefined;
}

/**
 * Aggregates 15-second rolling DPS graph data for a set of target IDs.
 */
export function aggregateGroupGraphData(
  graphData: Record<string, Record<string, GraphDataPoint[]>> | undefined,
  targetIds: string[]
): Record<string, GraphDataPoint[]> | undefined {
  if (!graphData) return undefined;

  const validTargetIds = targetIds.filter(
    (tid) => graphData[tid] && Object.keys(graphData[tid]).length > 0
  );
  if (validTargetIds.length === 0) return undefined;

  const playerTimeMap: Record<
    string,
    Record<number, { totalDamage: number; rollingDPS: number }>
  > = {};

  for (const tid of validTargetIds) {
    const targetGraph = graphData[tid];
    for (const [playerId, points] of Object.entries(targetGraph)) {
      if (!playerTimeMap[playerId]) {
        playerTimeMap[playerId] = {};
      }
      for (const pt of points) {
        if (!playerTimeMap[playerId][pt.time]) {
          playerTimeMap[playerId][pt.time] = {
            totalDamage: pt.totalDamage,
            rollingDPS: pt.rollingDPS,
          };
        } else {
          playerTimeMap[playerId][pt.time].totalDamage += pt.totalDamage;
          playerTimeMap[playerId][pt.time].rollingDPS += pt.rollingDPS;
        }
      }
    }
  }

  const result: Record<string, GraphDataPoint[]> = {};
  for (const [playerId, timeMap] of Object.entries(playerTimeMap)) {
    const times = Object.keys(timeMap)
      .map(Number)
      .sort((a, b) => a - b);
    result[playerId] = times.map((t) => ({
      time: t,
      totalDamage: timeMap[t].totalDamage,
      rollingDPS: timeMap[t].rollingDPS,
    }));
  }

  return result;
}
