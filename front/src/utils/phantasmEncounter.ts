import type { TargetStats } from '@/protocols';

// Confirmed: Normal 191681; Temptation's Path 195012. Other database
// entries are only candidates: they may be costumes or unrelated content.
export const PHANTASM_QUEEN_RACE = 195012;
export const PHANTASM_CONFIRMED_QUEENS = new Set([191681,195012]);
export const PHANTASM_CANDIDATE_QUEENS = new Set([191699,191724,191803,193551,193709,193723]);
export const PHANTASM_BOSS_ADDS = new Set([195019, 195020, 195021, 195022]);
export const PHANTASM_MINOR_RACES = new Set([
  ...Array.from({ length: 17 }, (_, i) => 191682 + i),
  195013, 195015, 195016, 195017, 195018,
]);
export const PHANTASM_RUN_PREFIX = 'group_phantasm_run_';
export const PHANTASM_MINOR_PREFIX = 'group_phantasm_minor_';
export const PHANTASM_BOSS_ROOM_PREFIX = 'group_phantasm_boss_room_';
// Confirmed boss-room companions in Normal Phantasm: Fiends and Wraiths.
export const PHANTASM_BOSS_ROOM_RACES = new Set([191683, 191684, 191698]);
export const PHANTASM_MECHANICS_PREFIX = 'group_phantasm_mechanics_';

export interface PhantasmRun {
  index: number;
  queenId: string;
  queenIds: string[];
  difficulty: 'normal' | 'temptations-path' | 'unverified';
  queenStart: number;
  startUnixSeconds: number;
  endUnixSeconds: number;
  bossAddIds: string[];
  bossRoomIds: string[];
  minorIds: string[];
  entityIds: string[];
  variant: 'with-succubi' | 'queen-only';
}

/** Construct distinct runs from observed Queen spawns, not the session total.
 * A Queen's four simultaneous named Succubi belong to boss mechanics.
 * The earlier colored Succubi and dungeon trash stay in Minor Enemies.
 * The partition is inferred from spawn chronology, not a network instance ID.
 */
export function findPhantasmRuns(targets: Record<string, TargetStats>): PhantasmRun[] {
  const entries = Object.entries(targets);
  const confirmed = entries
    .filter(([, t]) => PHANTASM_CONFIRMED_QUEENS.has(Number(t.raceId)) && Number(t.startTime) > 0)
    .sort((a, b) => Number(a[1].startTime) - Number(b[1].startTime) || a[0].localeCompare(b[0]));
  const candidates = entries.filter(([, t]) => PHANTASM_CANDIDATE_QUEENS.has(Number(t.raceId)) && Number(t.startTime) > 0);
  // Candidate-only bosses require independent Phantasm context. Never treat
  // an arbitrary similarly named Queen elsewhere as evidence of a run.
  const accepted = [...confirmed];
  for (const item of candidates) {
    const time = Number(item[1].startTime);
    if (confirmed.some(([, q]) => Math.abs(time - Number(q.startTime)) <= 120)) continue;
    const context = entries.filter(([, t]) => PHANTASM_MINOR_RACES.has(Number(t.raceId)) &&
      Number(t.startTime) >= time - 180 && Number(t.startTime) < time - 3);
    const distinct = new Set(context.map(([, t]) => Number(t.raceId)));
    if (context.length >= 12 && distinct.size >= 3 &&
        !accepted.some(([, q]) => Math.abs(time - Number(q.startTime)) <= 120)) accepted.push(item);
  }
  const queens = accepted.sort((a,b)=>Number(a[1].startTime)-Number(b[1].startTime) || a[0].localeCompare(b[0]));
  if (!queens.length) return [];
  const firstQueen = Number(queens[0][1].startTime);
  const firstMinor = entries
    .filter(([, t]) => PHANTASM_MINOR_RACES.has(Number(t.raceId)) &&
      Number(t.startTime) <= firstQueen && Number(t.startTime) >= firstQueen - 1200)
    .reduce((min, [, t]) => Math.min(min, Number(t.startTime)), firstQueen);
  return queens.map(([queenId, queen], i) => {
    const queenStart = Number(queen.startTime);
    const nextStart = i + 1 < queens.length ? Number(queens[i+1][1].startTime) : Infinity;
    const previousQueen = i ? Number(queens[i - 1][1].startTime) : -Infinity;
    const start = i ? previousQueen + 0.001 : firstMinor;
    const candidatesHere = entries.filter(([, t]) => {
      const time = Number(t.startTime);
      return Number.isFinite(time) && time >= start && time < queenStart;
    });
    // Costume swaps (if represented as new entities) stay in this single
    // encounter, with distinct entity IDs and untouched raw HP histories.
    const queenIds = [queenId, ...candidates.filter(([,t]) => {
      const time=Number(t.startTime);
      return time >= queenStart && time <= queenStart+120 && time < nextStart;
    }).map(([id])=>id)];
    const bossAddIds = entries.filter(([, t]) =>
      PHANTASM_BOSS_ADDS.has(Number(t.raceId)) &&
      Math.abs(Number(t.startTime) - queenStart) <= 3).map(([id]) => id);
    // Normal Phantasm summons Fiends and Wraiths AFTER the Queen appears,
    // sometimes much later than the first boss-room spawn. Bound these by
    // both the following Queen and the next recognizable dungeon stage, so
    // a mixed recording cannot leak later enemies into this run.
    const nextFahmesStart = entries
      .filter(([, t]) => {
        const race = Number(t.raceId);
        return race >= 121001 && race <= 126999 && Number(t.startTime) > queenStart;
      })
      .reduce((min, [, t]) => Math.min(min, Number(t.startTime)), Infinity);
    const bossRoomEnd = Math.min(queenStart + 300, nextStart, nextFahmesStart);
    const bossRoomIds = entries.filter(([, t]) =>
      PHANTASM_BOSS_ROOM_RACES.has(Number(t.raceId)) &&
      Number(t.startTime) >= queenStart && Number(t.startTime) < bossRoomEnd
    ).map(([id]) => id);
    const minorIds = candidatesHere.filter(([, t]) =>
      PHANTASM_MINOR_RACES.has(Number(t.raceId))).map(([id]) => id);
    const race=Number(queen.raceId);
    return {
      index: i + 1, queenId, queenIds, queenStart,
      difficulty: race===191681?'normal' as const:race===195012?'temptations-path' as const:'unverified' as const,
      startUnixSeconds: start, endUnixSeconds: Math.max(Number(queen.endTime)||queenStart, ...[...queenIds, ...bossRoomIds].map(id=>Number(targets[id]?.endTime)||Number(targets[id]?.startTime)||queenStart)),
      bossAddIds, bossRoomIds, minorIds,
      entityIds: [...new Set([...queenIds, ...bossAddIds, ...bossRoomIds, ...minorIds])],
      variant: bossAddIds.length ? 'with-succubi' as const : 'queen-only' as const,
    };
  });
}
