import type { TargetStats } from '@/protocols';
import { findGlennAttempts } from './glennEncounter';

// Cailleach is an opening HP checkpoint, not a 0-HP boss clear.
// 7584 is another Cailleach race, but is deliberately not treated as the
// opening damage target without evidence that it represents the same fight.
export const GLENN_OPENING_RACE = 7575;
export const GLENN_SPRITE_RACES = new Set([7539, 7583]);
export const GLENN_BARRIER_RACES = new Set([7546, 7547, 7548, 7549]);
export const GLENN_OPENING_PREFIX = 'group_glenn_opening_';
export const GLENN_OPENING_BARRIER_PREFIX = 'group_glenn_barriers_';
export const GLENN_OPENING_SPRITE_PREFIX = 'group_glenn_sprites_';

export function findGlennOpenings(targets: Record<string, TargetStats>) {
  const starts = Object.entries(targets)
    .filter(([, t]) => Number(t.raceId) === GLENN_OPENING_RACE && Number(t.startTime) > 0)
    .sort((a, b) => Number(a[1].startTime) - Number(b[1].startTime));
  const bosses = findGlennAttempts(targets);
  return starts.map(([entityId, opening], index) => {
    const start = Number(opening.startTime);
    const nextOpening = starts[index + 1] ? Number(starts[index + 1][1].startTime) : Infinity;
    // Use only the next recorded Reverie lineage after this opener as a boundary.
    const reverieStart = bosses.find(a => a.startUnixSeconds >= start && a.startUnixSeconds < nextOpening)?.startUnixSeconds ?? Infinity;
    const boundary = Math.min(nextOpening, reverieStart, start + 3600);
    const encountered = Object.entries(targets)
      .filter(([, t]) => Number(t.startTime) >= start && Number(t.startTime) < boundary);
    const barrierIds = encountered.filter(([, t]) => GLENN_BARRIER_RACES.has(Number(t.raceId))).map(([id]) => id);
    const spriteIds = encountered.filter(([, t]) => GLENN_SPRITE_RACES.has(Number(t.raceId))).map(([id]) => id);
    return { firstEntityId: entityId, entityIds: [entityId], barrierIds, spriteIds,
      startUnixSeconds: start, endUnixSeconds: Number(opening.endTime) || null,
      // At 90% HP Cailleach leaves; do not call this a defeat.
      completion: 'opening-checkpoint' as const };
  });
}
