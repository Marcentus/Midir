import { findHighTierRuns, HIGH_TIER_MINOR_PREFIX, HIGH_TIER_RUN_PREFIX } from './highTierMinorGrouping';
import { findFahmesStages, FAHMES_STAGE_PREFIX, FAHMES_LEVEL_PREFIX } from './fahmesRuin';
import { findPhantasmRuns, PHANTASM_RUN_PREFIX, PHANTASM_MINOR_PREFIX, PHANTASM_MECHANICS_PREFIX, PHANTASM_BOSS_ROOM_PREFIX } from './phantasmEncounter';
import { findDungeonRunFolders, DUNGEON_RUN_PREFIX } from './dungeonRunFolders';
import { findGlennOpenings, GLENN_OPENING_PREFIX, GLENN_OPENING_BARRIER_PREFIX } from './glennOpening';
import { findAlbyRuns, ALBY_GROUP_PREFIX, ALBY_WAVE_PREFIX } from './albyShineseeker';
import { partitionMinorRuns } from "./instanceMinorGrouping";
import { GLENN_MINOR_GROUP_ID, isGlennMinorRace } from './glennMinorEnemies';
import { findCromAttempts, findCromMechanics, CROM_IRUSAN, CROM_GLAS, CROM_GLAS_VARIANTS, CROM_DISCIPLE_RACES, CROM_MINOR_GROUP_ID, CROM_MINOR_RACES, CROM_IRUSAN_GROUP_PREFIX, CROM_GLAS_GROUP_PREFIX, CROM_MERCH_GROUP_PREFIX, CROM_DISCIPLES_GROUP_PREFIX } from './cromEncounter';
import { findGate1Attempts } from './gate1Encounter';
import { findGlennAttempts } from './glennEncounter';
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
  { bossRaceId: 7585, groupName: "Glenn Bearna — Unfinished Reverie / Winter Tyrant" },
  { bossRaceId: 195012, groupName: "Rabbie Phantasm: Temptation's Path" },
  { bossRaceId: 191681, groupName: "Rabbie Phantasm: Normal" },
  { bossRaceId: 193364, groupName: "Crom Bas — Irusan / Glas" },
  { bossRaceId: 193359, groupName: "Crom Bas — Glas (standalone)" },
  { bossRaceId: 193358, groupName: "Crom Bas — Completed Glas (standalone)" },
  { bossRaceId: 7615, groupName: "Bri Leith: Gate 4" },
  { bossRaceId: 7603, groupName: "Bri Leith: Gate 3" },
  { bossRaceId: 7600, partnerRaceId: 7601, groupName: "Bri Leith: Gate 1" },
  { bossRaceId: 193810, groupName: "Crom Bas Abyss" },
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

  if (targetId.startsWith(FAHMES_STAGE_PREFIX) || targetId.startsWith(FAHMES_LEVEL_PREFIX)) {
    const stages=findFahmesStages(fightSummary.targets || {});
    if(targetId.startsWith(FAHMES_STAGE_PREFIX)) {
      const match=/^(\d+)_(\d+)$/.exec(targetId.slice(FAHMES_STAGE_PREFIX.length));
      return match ? stages.find(s=>s.stage===Number(match[1])&&s.attempt===Number(match[2]))?.entityIds || [] : [];
    }
    const match=/^(\d+)_(\d+)_(\d+)$/.exec(targetId.slice(FAHMES_LEVEL_PREFIX.length));
    return match ? stages.find(s=>s.stage===Number(match[1])&&s.attempt===Number(match[2]))?.levels[Number(match[3])-1]?.entityIds || [] : [];
  }

  if (targetId.startsWith(PHANTASM_RUN_PREFIX) || targetId.startsWith(PHANTASM_MINOR_PREFIX) || targetId.startsWith(PHANTASM_MECHANICS_PREFIX) || targetId.startsWith(PHANTASM_BOSS_ROOM_PREFIX)) {
    const runs = findPhantasmRuns(fightSummary.targets || {});
    if (targetId.startsWith(PHANTASM_RUN_PREFIX)) return runs[Number(targetId.slice(PHANTASM_RUN_PREFIX.length)) - 1]?.entityIds || [];
    for (const [prefix, kind] of [[PHANTASM_MINOR_PREFIX, 'minor'], [PHANTASM_BOSS_ROOM_PREFIX, 'bossRoom']] as const) {
      if (!targetId.startsWith(prefix)) continue;
      const suffix = targetId.slice(prefix.length);
      const match = /^(\d+)(?:_(.*))?$/.exec(suffix);
      if (!match) return [];
      const run = runs[Number(match[1]) - 1];
      if (!run) return [];
      const ids = kind === 'minor' ? run.minorIds : run.bossRoomIds;
      if (!match[2]) return ids;
      return ids.filter(id => {
        const target = fightSummary.targets?.[id];
        const label = ((target as typeof target & {rawName?: string})?.rawName || target?.name || 'Unknown').trim().replace(/[^a-z0-9]/gi, '_');
        return label === match[2];
      });
    }
    const run = runs.find(r => r.queenId === targetId.slice(PHANTASM_MECHANICS_PREFIX.length));
    return run?.bossAddIds || [];
  }

  if (targetId.startsWith(HIGH_TIER_MINOR_PREFIX) || targetId.startsWith(HIGH_TIER_RUN_PREFIX)) {
    const run = findHighTierRuns(fightSummary.targets || {}).find(r =>
      r.minorGroupId === targetId || r.runGroupId === targetId);
    return run ? (targetId.startsWith(HIGH_TIER_MINOR_PREFIX) ? run.minorIds : run.entityIds) : [];
  }

  // The run folder combines existing target scopes without changing their
  // boss-only HP semantics. De-duplicate entities shared by child groups.
  if (targetId.startsWith(DUNGEON_RUN_PREFIX)) {
    const folder = findDungeonRunFolders(fightSummary.targets || {}).find(f => f.id === targetId);
    if (!folder) return [];
    const memberIds = folder.groupIds.flatMap(id => getGroupMemberTargetIds(id, fightSummary));
    // The combined run view includes boss mechanics, but these never alter
    // the dedicated boss-only HP or damage selections.
    for (const childId of folder.groupIds) {
      if (childId.startsWith(GLENN_OPENING_PREFIX)) {
        const opening = findGlennOpenings(fightSummary.targets || {})
          .find(a => a.firstEntityId === childId.slice(GLENN_OPENING_PREFIX.length));
        if (opening) memberIds.push(...opening.barrierIds);
      } else if (childId.startsWith('group_')) {
        const firstId = childId.slice('group_'.length);
        const crom = findCromAttempts(fightSummary.targets || {})
          .find(a => a.firstEntityId === firstId);
        if (crom) {
          const mechanics = findCromMechanics(fightSummary.targets || {}, crom);
          memberIds.push(...mechanics.merchIds, ...mechanics.discipleIds);
        }
      }
    }
    return [...new Set(memberIds)];
  }

  for (const [prefix,kind] of [
    [GLENN_OPENING_PREFIX,'opening'], [GLENN_OPENING_BARRIER_PREFIX,'barrier']
  ] as const) {
    if (!targetId.startsWith(prefix)) continue;
    const opening=findGlennOpenings(fightSummary.targets||{}).find(a=>a.firstEntityId===targetId.slice(prefix.length));
    if (!opening) return [];
    return kind==='opening' ? opening.entityIds : opening.barrierIds;
  }
  if (targetId.startsWith(ALBY_GROUP_PREFIX)) {
    const index=Number(targetId.slice(ALBY_GROUP_PREFIX.length));
    return findAlbyRuns(fightSummary.targets||{})[index-1]?.entityIds || [];
  }
  if (targetId.startsWith(ALBY_WAVE_PREFIX)) {
    const parts=/^(\d+)_(\d+)$/.exec(targetId.slice(ALBY_WAVE_PREFIX.length));
    if (!parts) return [];
    return findAlbyRuns(fightSummary.targets||{})[Number(parts[1])-1]?.waveIds[Number(parts[2])] || [];
  }
  if (targetId.startsWith(CROM_MINOR_GROUP_ID+'_')) {
    const segment = Number(targetId.slice(CROM_MINOR_GROUP_ID.length+1));
    if (!Number.isInteger(segment) || segment<0) return [];
    const otherDungeonIds = new Set([...findPhantasmRuns(fightSummary.targets || {}).flatMap(run => run.entityIds), ...findFahmesStages(fightSummary.targets || {}).flatMap(stage => stage.entityIds)]);
    const claimed = new Set(findCromAttempts(fightSummary.targets || {})
      .flatMap(attempt => { const m=findCromMechanics(fightSummary.targets || {},attempt); return [...m.discipleIds,...m.merchIds]; }));
    const members = Object.entries(fightSummary.targets || {})
      .filter(([id,t]) => !otherDungeonIds.has(id) && (CROM_MINOR_RACES.has(Number(t.raceId)) ||
        (CROM_DISCIPLE_RACES.has(Number(t.raceId)) && !claimed.has(id))))
      .map(([id,t])=>({id,startTime:t.startTime}));
    return partitionMinorRuns(members,findCromAttempts(fightSummary.targets || {}))[segment]?.members.map(m=>m.id)||[];
  }
  if (targetId === CROM_MINOR_GROUP_ID) {
    const otherDungeonIds = new Set([...findPhantasmRuns(fightSummary.targets || {}).flatMap(run => run.entityIds), ...findFahmesStages(fightSummary.targets || {}).flatMap(stage => stage.entityIds)]);
    const claimed = new Set(findCromAttempts(fightSummary.targets || {})
      .flatMap(attempt => { const m=findCromMechanics(fightSummary.targets || {},attempt); return [...m.discipleIds,...m.merchIds]; }));
    return Object.entries(fightSummary.targets || {}).filter(([id,t])=>
      !otherDungeonIds.has(id) && (CROM_MINOR_RACES.has(Number(t.raceId)) || (CROM_DISCIPLE_RACES.has(Number(t.raceId))&&!claimed.has(id)))).map(([id])=>id);
  }
  for (const [prefix, kind] of [
    [CROM_IRUSAN_GROUP_PREFIX, 'irusan'], [CROM_GLAS_GROUP_PREFIX, 'glas'],
    [CROM_MERCH_GROUP_PREFIX, 'merch'], [CROM_DISCIPLES_GROUP_PREFIX, 'disciples'],
  ] as const) {
    if (!targetId.startsWith(prefix)) continue;
    const attempt = findCromAttempts(fightSummary.targets || {})
      .find(a=>a.firstEntityId===targetId.slice(prefix.length));
    if (!attempt) return [];
    if (kind==='irusan') return attempt.irusanId?[attempt.irusanId]:[];
    if (kind==='glas') return attempt.glasId?[attempt.glasId]:[];
    const mechanics = findCromMechanics(fightSummary.targets || {},attempt);
    return kind==='merch' ? mechanics.merchIds : mechanics.discipleIds;
  }

  if (targetId.startsWith(GLENN_MINOR_GROUP_ID+'_')) {
    const segment=Number(targetId.slice(GLENN_MINOR_GROUP_ID.length+1));
    if(!Number.isInteger(segment)||segment<0)return [];
    const members=Object.entries(fightSummary.targets||{})
      .filter(([,t])=>isGlennMinorRace(t.raceId)).map(([id,t])=>({id,startTime:t.startTime}));
    return partitionMinorRuns(members,findGlennAttempts(fightSummary.targets||{}))[segment]?.members.map(m=>m.id)||[];
  }
  if (targetId === GLENN_MINOR_GROUP_ID) {
    return Object.entries(fightSummary.targets||{}).filter(([,t])=>isGlennMinorRace(t.raceId)).map(([id])=>id);
  }

  // Glenn has two independent health encounters: 7585 ends at 10% HP;
  // 7586 hands off at 60% HP to 7587. Raw entity IDs are never merged.
  const glennScope = /^group_glenn_(first|final|reverie|tyrant)_(\d+)$/.exec(targetId);
  if (glennScope) {
    const attempt = findGlennAttempts(fightSummary.targets || {})
      .find(a => a.firstEntityId === glennScope[2]);
    if (!attempt) return [];
    const kind = glennScope[1];
    return attempt.entityIds.filter(id => {
      const race = Number(fightSummary.targets[id]?.raceId);
      return kind === 'first' ? race === 7585
        : kind === 'final' ? race === 7586 || race === 7587
        : kind === 'reverie' ? race === 7585 || race === 7586 // historical selection alias
        : race === 7587;
    });
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
    const run=findPhantasmRuns(targetsMap).find(r=>r.queenId===bossTargetId);
    return run?.queenIds || [bossTargetId];
  }

  // Selecting the logical Vertrag boss means the two boss entities only;
  // phase giants are displayed in the encounter group but are not boss DPS.
  // These encounters expose a single boss as their selectable damage target;
  // adds (including the named succubi) remain inspectable, not boss DPS.
  if (Number(bossTarget.raceId)===CROM_IRUSAN || CROM_GLAS_VARIANTS.has(Number(bossTarget.raceId))) {
    return findCromAttempts(targetsMap).find(a=>a.firstEntityId===bossTargetId)?.entityIds || [bossTargetId];
  }
  if (findPhantasmRuns(targetsMap).some(run=>run.queenId===bossTargetId)) return findPhantasmRuns(targetsMap).find(run=>run.queenId===bossTargetId)!.queenIds;
  if (bossTarget.raceId === 7585) {
    return findGlennAttempts(targetsMap).find(a => a.firstEntityId === bossTargetId)?.entityIds || [bossTargetId];
  }
  if (bossTarget.raceId === 7600) {
    return findGate1Attempts(targetsMap).find(a=>a.firstEntityId===bossTargetId)?.entityIds || [bossTargetId];
  }

  const allTargets = Object.entries(targetsMap).map(([id, stats]) => ({
    id,
    ...stats,
  }));

  const bossStartTime = bossTarget.startTime || 0;
  // Folamh and Midir the Lost are known bosses, but their phases and adds
  // have not been verified from recordings. Keep the boss-only selection
  // strictly on the observed entity rather than absorbing every timed spawn.
  if (bossDef.bossRaceId === 193810 || bossDef.bossRaceId === 7615) {
    return [bossTargetId];
  }
  let bossEndTime =
    bossTarget.seenDead || bossTarget.disappeared
      ? bossTarget.endTime || 0
      : Infinity;

  let partnerTarget: any = null;
  if (bossDef.partnerRaceId === 7601 && bossDef.bossRaceId === 7600) {
    const attempt = findGate1Attempts(targetsMap).find(a => a.firstEntityId === bossTargetId);
    if (attempt?.replacementEntityId) {
      partnerTarget = allTargets.find(t => t.id === attempt.replacementEntityId);
      bossEndTime = Number(partnerTarget?.endTime) || Infinity;
    } else {
      // A first-form wipe must never absorb later unrelated giants or boss attempts.
      bossEndTime = Number(bossTarget.endTime) || Infinity;
    }
  } else if (bossDef.partnerRaceId) {
    partnerTarget = allTargets.find(t => t.raceId === bossDef.partnerRaceId &&
      (t.startTime || 0) >= bossStartTime);
    if (partnerTarget) bossEndTime = Number(partnerTarget.endTime) || Infinity;
  }
  // Exclude any entities spawned on/after the next first-form boss attempt.
  if (bossDef.bossRaceId === 7600) {
    const nextAttemptStart = findGate1Attempts(targetsMap)
      .filter(a => a.startUnixSeconds > bossStartTime)
      .map(a => a.startUnixSeconds).sort((a,b)=>a-b)[0];
    if (nextAttemptStart !== undefined) bossEndTime = Math.min(bossEndTime, nextAttemptStart - 0.001);
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
