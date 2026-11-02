import type { TargetStats } from '@/protocols';

/** Crom Bas Normal (not Crom Bas Abyss) boss and mechanic IDs, sourced from
 * Midir's bundled races.json and the 1790845007475 session. */
export const CROM_IRUSAN = 193364;
export const CROM_GLAS = 193359; // Giga Glas Ghaibhleann
export const CROM_COMPLETED_GLAS = 193358; // Completed Glas Ghaibhleann (basic difficulty)
export const CROM_GLAS_VARIANTS = new Set([CROM_GLAS, CROM_COMPLETED_GLAS]);
export const CROM_GLAS_NAMES: Record<number,string> = {
  [CROM_GLAS]:'Giga Glas Ghaibhleann',
  [CROM_COMPLETED_GLAS]:'Completed Glas Ghaibhleann',
};
export const CROM_COLORED_DISCIPLES: Record<string,readonly number[]> = {
  Red: [193438,193442], Blue: [193439,193443],
  White: [193440,193444], Yellow: [193441,193445],
};
export const CROM_MERCH = 193355;
export const CROM_DISCIPLE_RACES = new Set([
  193356, 193365, 193366, 193398, 193412, 193424, 193425, 193426,
  193438, 193439, 193440, 193441, 193442, 193443, 193444, 193445,
  193806, 193807, 193808, 193816,
]);
export const CROM_MINOR_RACES = new Set([
  191684, 193350, 193351, 193352, 193353, 193354, 193419, 193420,
  193422, 193423, 193437,
]);
export const CROM_MINOR_GROUP_ID = 'group_crom_minor_enemies';
export const CROM_IRUSAN_GROUP_PREFIX = 'group_crom_irusan_';
export const CROM_GLAS_GROUP_PREFIX = 'group_crom_glas_';
export const CROM_MERCH_GROUP_PREFIX = 'group_crom_merch_';
export const CROM_DISCIPLES_GROUP_PREFIX = 'group_crom_disciples_';

// A gap cap stops an early/aborted Irusan encounter from consuming a Glas fight
// in a later run when save files combine multiple dungeon entries. It is not
// a claim that a Crom run is always shorter than this interval.
export const CROM_MAX_BOSS_TRANSITION_SECONDS = 15 * 60;

export function findCromAttempts(targets: Record<string, TargetStats>) {
  const bosses = Object.entries(targets)
    .filter(([, t]) => (Number(t.raceId)===CROM_IRUSAN || CROM_GLAS_VARIANTS.has(Number(t.raceId))) && Number(t.startTime) > 0)
    .sort((a,b) => Number(a[1].startTime)-Number(b[1].startTime) || a[0].localeCompare(b[0]));
  const used = new Set<string>();
  const attempts: {firstEntityId:string;entityIds:string[];irusanId:string|null;glasId:string|null;startUnixSeconds:number;endUnixSeconds:number|null;completion:'verified-clear'|'unverified'}[] = [];
  for (let index=0; index<bosses.length; index++) {
    const [id, first] = bosses[index];
    if (used.has(id)) continue;
    used.add(id);
    const start = Number(first.startTime);
    let irusanId: string|null = Number(first.raceId)===CROM_IRUSAN ? id : null;
    let glasId: string|null = CROM_GLAS_VARIANTS.has(Number(first.raceId)) ? id : null;
    if (irusanId) {
      const nextIrusanTime = bosses.slice(index+1).filter(([,t])=>Number(t.raceId)===CROM_IRUSAN)
        .map(([,t])=>Number(t.startTime))[0] ?? Infinity;
      const match = bosses.slice(index+1).find(([candidateId,t]) =>
        !used.has(candidateId) && CROM_GLAS_VARIANTS.has(Number(t.raceId)) &&
        Number(t.startTime) < nextIrusanTime &&
        Number(t.startTime) - start <= CROM_MAX_BOSS_TRANSITION_SECONDS);
      if (match) { glasId = match[0]; used.add(glasId); }
    }
    const ids = [irusanId,glasId].filter((x):x is string=>Boolean(x));
    const last = targets[ids[ids.length-1]];
    const zero = (last?.hpHistory||[]).some(h=>Number(h.maxHp)>0 && Number(h.currentHp)<=0);
    attempts.push({firstEntityId:id,entityIds:ids,irusanId,glasId,startUnixSeconds:start,
      endUnixSeconds:Number(last?.endTime)||null,
      completion:glasId && zero?'verified-clear':'unverified'});
  }
  return attempts.sort((a,b)=>a.startUnixSeconds-b.startUnixSeconds);
}

/** Mechanics are offered separately and never included in a boss-HP pool.
 * Merch Diafol belongs to the Irusan window. Disciples spawned during the
 * Glas encounter belong to Glas; early prep-room disciples do not. */
export function findCromMechanics(targets: Record<string, TargetStats>, attempt: ReturnType<typeof findCromAttempts>[number]) {
  const all = Object.entries(targets);
  const nextStart = findCromAttempts(targets).filter(a=>a.startUnixSeconds>attempt.startUnixSeconds)
    .map(a=>a.startUnixSeconds).sort((a,b)=>a-b)[0] ?? Infinity;
  const irusan = attempt.irusanId ? targets[attempt.irusanId] : null;
  const glas = attempt.glasId ? targets[attempt.glasId] : null;
  const irusanStart = Number(irusan?.startTime)||Infinity;
  const glasStart = Number(glas?.startTime)||Infinity;
  const irusanEnd = Math.min(glasStart, Number(irusan?.endTime)||Infinity, nextStart);
  const glasEnd = Math.min(Number(glas?.endTime)||Infinity,nextStart);
  const merchIds = all.filter(([,t])=>Number(t.raceId)===CROM_MERCH &&
    Number(t.startTime)>=irusanStart-3 && Number(t.startTime)<=irusanEnd)
    .map(([id])=>id);
  const discipleIds = all.filter(([,t])=>CROM_DISCIPLE_RACES.has(Number(t.raceId)) &&
    Number(t.startTime)>=glasStart && Number(t.startTime)<=glasEnd)
    .map(([id])=>id);
  const coloredDiscipleIds: Record<string,string[]> = {};
  for (const [color,races] of Object.entries(CROM_COLORED_DISCIPLES)) {
    coloredDiscipleIds[color] = discipleIds.filter(id=>races.includes(Number(targets[id]?.raceId)));
  }
  return {merchIds,discipleIds,coloredDiscipleIds};
}
