import type { TargetStats } from '@/protocols';

// Two different Vertrag entities form one logical fight. Do not join across
// another 7600 spawn (a fresh attempt), even when saved together in one session.
export type Gate1Attempt = {
  firstEntityId: string; replacementEntityId: string | null;
  entityIds: string[]; startUnixSeconds: number; endUnixSeconds: number | null;
  completion: 'verified-clear' | 'unverified';
  evidence: string;
};
export const GATE1_REPLACEMENT_TOLERANCE_SECONDS = 30;

export function findGate1Attempts(targets: Record<string, TargetStats>): Gate1Attempt[] {
  const candidates = Object.entries(targets).filter(([,t]) => t.raceId === 7600 || t.raceId === 7601)
    .filter(([,t]) => Number.isFinite(t.startTime) && Number(t.startTime) > 0)
    .sort((a,b) => Number(a[1].startTime) - Number(b[1].startTime) || a[0].localeCompare(b[0]));
  const starters = candidates.filter(([,t]) => t.raceId === 7600);
  const used = new Set<string>();
  return starters.map(([id,first],index) => {
    const start = Number(first.startTime);
    const nextStart = index + 1 < starters.length ? Number(starters[index+1][1].startTime) : Infinity;
    const firstEnd = Number(first.endTime);
    // The first form despawns before its replacement is spawned. Transition
    // animations and recorded timestamp drift may exceed two seconds. Only
    // consider a later 7601 within 30s, and never cross a new 7600 attempt.
    const partner = candidates.find(([otherId,t]) => !used.has(otherId) && t.raceId === 7601 &&
      Number(t.startTime) >= start && Number(t.startTime) < nextStart &&
      Number.isFinite(firstEnd) && firstEnd > 0 &&
      Number(t.startTime) >= firstEnd - 2 &&
      Number(t.startTime) <= firstEnd + GATE1_REPLACEMENT_TOLERANCE_SECONDS);
    if (partner) used.add(partner[0]);
    const [replacementId,replacement] = partner || [null,null];
    const end = Number(replacement?.endTime || first.endTime);
    const zero = Boolean(replacement?.hpHistory?.some(h => Number(h.maxHp)>0 && Number(h.currentHp)<=0));
    return {firstEntityId:id, replacementEntityId:replacementId, entityIds:replacementId?[id,replacementId]:[id],
      startUnixSeconds:start, endUnixSeconds:Number.isFinite(end) && end>0 ? end:null,
      completion:zero?'verified-clear' as const:'unverified' as const,
      evidence:zero?'replacement-observed-zero-hp':replacementId?'replacement-without-zero-hp':'no-matched-replacement'};
  });
}

// Display-only effective HP. Both entities expose a full maximum, but only
// the top half of 7600 and the bottom half of 7601 are damageable.
// Raw individual entity values and phase HP must remain unchanged.
export function gate1EffectiveHp(
  firstMax: number, firstCurrent: number,
  secondMax: number | null, secondCurrent: number | null,
  replacementActive: boolean
): { currentHp: number; maxHp: number; hpPercent: number } | null {
  if (!(Number.isFinite(firstMax) && firstMax > 0)) return null;
  if (!(Number.isFinite(firstCurrent))) return null;
  if (secondMax !== null && !(Number.isFinite(secondMax) && secondMax > 0)) return null;
  const maxHp = firstMax / 2 + (secondMax === null ? 0 : secondMax / 2);
  // Before the second entity is observed, don't invent its maximum HP.
  // The partial first-form bar is meaningful, but not comparable to a full
  // two-form run; callers must preserve an 'incomplete' marker.
  if (replacementActive && (secondMax === null || secondCurrent === null || !Number.isFinite(secondCurrent))) return null;
  const currentHp = replacementActive
    ? Math.max(0, Math.min(secondMax! / 2, secondCurrent!))
    : Math.max(0, firstCurrent - firstMax / 2) + (secondMax === null ? 0 : secondMax / 2);
  return {currentHp, maxHp, hpPercent: Math.max(0,Math.min(100,100 * currentHp / maxHp))};
}

// Independent in-game mechanic axis. This is deliberately not the same as
// damage-weighted effective HP, since 7601 has a different maximum HP.
export function gate1PhasePercent(raceId: 7600 | 7601, currentHp: number, maxHp: number): number | null {
  if (!(Number.isFinite(maxHp) && maxHp > 0 && Number.isFinite(currentHp))) return null;
  const pct=100 * currentHp / maxHp;
  return Math.max(0,Math.min(100,raceId===7600?pct:pct));
}
