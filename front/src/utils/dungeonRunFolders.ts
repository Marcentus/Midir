import type { TargetStats } from '@/protocols';
import { findPhantasmRuns } from './phantasmEncounter';
import { findFahmesStages } from './fahmesRuin';
import { findCromAttempts, findCromMechanics, CROM_MINOR_RACES, CROM_DISCIPLE_RACES, CROM_MINOR_GROUP_ID } from './cromEncounter';
import { findGlennAttempts } from './glennEncounter';
import { findGlennOpenings, GLENN_OPENING_PREFIX } from './glennOpening';
import { isGlennMinorRace, GLENN_MINOR_GROUP_ID } from './glennMinorEnemies';
import { partitionMinorRuns } from './instanceMinorGrouping';

export const DUNGEON_RUN_PREFIX = 'group_dungeon_run_';
export interface DungeonRunFolder {
  id: string;
  name: string;
  dungeon: 'crom' | 'glenn';
  groupIds: string[];
  startTime: number;
}

/** UI grouping only: retain the original boss and minor target scopes.
 * Do not equate an inferred grouping boundary with a confirmed instance ID.
 */
export function findDungeonRunFolders(targets: Record<string, TargetStats>): DungeonRunFolder[] {
  const folders: DungeonRunFolder[] = [];
  const claimedByOtherDungeons = new Set([
    ...findPhantasmRuns(targets).flatMap(run => run.entityIds),
    ...findFahmesStages(targets).flatMap(stage => stage.entityIds),
  ]);
  const crom = findCromAttempts(targets);
  const cromClaimed = new Set(crom.flatMap(a => {
    const m = findCromMechanics(targets, a);
    return [...m.merchIds, ...m.discipleIds];
  }));
  const cromMinors = Object.entries(targets)
    .filter(([id, t]) => !claimedByOtherDungeons.has(id) && (CROM_MINOR_RACES.has(Number(t.raceId)) ||
      (CROM_DISCIPLE_RACES.has(Number(t.raceId)) && !cromClaimed.has(id))))
    .map(([id, t]) => ({id, startTime: t.startTime}));
  const cromSegments = partitionMinorRuns(cromMinors, crom);
  const cromFolders = crom.map((a, i) => ({
    id: `${DUNGEON_RUN_PREFIX}crom_${i+1}`,
    name: `Crom Bas ${i+1}`,
    dungeon: 'crom' as const,
    groupIds: [`group_${a.firstEntityId}`],
    startTime: a.startUnixSeconds,
  }));
  cromSegments.forEach((s, segmentIndex) => {
    if (s.attemptIndex != null && cromFolders[s.attemptIndex]) {
      cromFolders[s.attemptIndex].groupIds.push(`${CROM_MINOR_GROUP_ID}_${segmentIndex}`);
    } else {
      folders.push({id: `${DUNGEON_RUN_PREFIX}crom_minor_${segmentIndex}`,
        name: `Crom Bas · Combat segment ${segmentIndex+1}`, dungeon:'crom',
        groupIds: [`${CROM_MINOR_GROUP_ID}_${segmentIndex}`],
        startTime: Number(s.members[0]?.startTime || 0)});
    }
  });
  folders.push(...cromFolders);

  const openings = findGlennOpenings(targets);
  const glennBosses = findGlennAttempts(targets);
  const glennFolders = openings.map((opening, i) => ({
    id: `${DUNGEON_RUN_PREFIX}glenn_${i+1}`,
    name: `Glenn Bearna ${i+1}`,
    dungeon: 'glenn' as const,
    groupIds: [`${GLENN_OPENING_PREFIX}${opening.firstEntityId}`],
    startTime: opening.startUnixSeconds,
  }));
  glennBosses.forEach((boss, i) => {
    const index = openings.findIndex((opening, j) =>
      opening.startUnixSeconds <= boss.startUnixSeconds &&
      (j === openings.length-1 || openings[j+1].startUnixSeconds > boss.startUnixSeconds) &&
      boss.startUnixSeconds - opening.startUnixSeconds <= 3600);
    if (index >= 0) glennFolders[index].groupIds.push(`group_${boss.firstEntityId}`);
    else glennFolders.push({id:`${DUNGEON_RUN_PREFIX}glenn_boss_${i+1}`,
      name:`Glenn Bearna · Boss attempt ${i+1}`, dungeon:'glenn',
      groupIds:[`group_${boss.firstEntityId}`], startTime:boss.startUnixSeconds});
  });
  const glennMinors = Object.entries(targets)
    .filter(([, t]) => isGlennMinorRace(t.raceId))
    .map(([id, t]) => ({id, startTime:t.startTime}));
  const glennSegments = partitionMinorRuns(glennMinors, glennBosses);
  glennSegments.forEach((segment, segmentIndex) => {
    const start = Number(segment.members[0]?.startTime || 0);
    const openingIndex = openings.findIndex((opening, j) =>
      opening.startUnixSeconds <= start &&
      (j === openings.length-1 || openings[j+1].startUnixSeconds > start) &&
      start - opening.startUnixSeconds <= 3600);
    if (openingIndex >= 0) glennFolders[openingIndex].groupIds.push(`${GLENN_MINOR_GROUP_ID}_${segmentIndex}`);
    else {
      const boss = segment.attemptIndex != null ? glennBosses[segment.attemptIndex] : null;
      const bossFolder = boss && glennFolders.find(f => f.groupIds.includes(`group_${boss.firstEntityId}`));
      if (bossFolder) bossFolder.groupIds.push(`${GLENN_MINOR_GROUP_ID}_${segmentIndex}`);
      else folders.push({id:`${DUNGEON_RUN_PREFIX}glenn_minor_${segmentIndex}`,
        name:`Glenn Bearna · Combat segment ${segmentIndex+1}`, dungeon:'glenn',
        groupIds:[`${GLENN_MINOR_GROUP_ID}_${segmentIndex}`],startTime:start});
    }
  });
  folders.push(...glennFolders);
  return folders.filter(f => f.groupIds.length > 0).sort((a,b)=>a.startTime-b.startTime);
}
