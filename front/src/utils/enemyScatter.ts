import type { FightSummary } from '@/protocols';
import { findReportEncounters, type BossReportEncounter } from './reportEncounters';

/** Recognize outgoing boss damage by source entity ID, NOT numeric names. */
export interface EnemyScatterIdentity {
  id: string;
  name: string;
  encounterId: string;
  raceId: number;
  color: string;
}
const colors = ['#e7a068', '#d88383', '#d39fe2', '#edbe79', '#c08ad8'];

/** Only positively identified boss entity IDs from recorded encounters qualify. */
export function identifyBossScatterSources(
  summary: FightSummary,
  encounters: BossReportEncounter[] = findReportEncounters(summary)
): Map<string, EnemyScatterIdentity> {
  const found = new Map<string, EnemyScatterIdentity>();
  for (const encounter of encounters) {
    for (const id of encounter.entityIds) {
      const raceId = Number(summary.targets?.[id]?.raceId || 0);
      if (!raceId || found.has(id)) continue;
      const phase = raceId === 7600 ? ' — First Form' : raceId === 7601 ? ' — Second Form' : '';
      found.set(id, {
        id,
        name: encounter.label + phase + ' (Enemy)',
        encounterId: encounter.id,
        raceId,
        color: colors[found.size % colors.length],
      });
    }
  }
  return found;
}
