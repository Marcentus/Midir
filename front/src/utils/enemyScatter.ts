import type { FightSummary } from '@/protocols';
import { findReportEncounters, type BossReportEncounter } from './reportEncounters';

/** Recognized boss races from the encounter registry and verified dungeon records.
 * Candidate Phantasm Queen outfits are intentionally excluded unless the encounter
 * detector has already accepted them using dungeon context. */
export const KNOWN_BOSS_RACE_NAMES: Readonly<Record<number, string>> = {
  7575: 'Cailleach', 7585: 'Unfinished Reverie', 7586: 'Unfinished Reverie',
  7587: 'The Winter Tyrant', 7600: 'Vertrag of the Deep Roots',
  7601: 'Vertrag of the Deep Roots', 7602: 'Bronntanas',
  7603: 'Midir the Beloved', 7615: 'Midir the Lost',
  191681: 'Succubus Queen (Normal)', 195012: 'Succubus Queen (Temptation\'s Path)',
  193358: 'Completed Glas Ghaibhleann', 193359: 'Giga Glas Ghaibhleann',
  193364: 'Irusan', 193810: 'Folamh',
};

export interface EnemyScatterIdentity {
  id: string;
  name: string;
  encounterId: string;
  raceId: number;
  color: string;
}
const colors = ['#e7a068', '#d88383', '#d39fe2', '#edbe79', '#c08ad8'];
const numericRaceName = (name: unknown): number | null => {
  const text = String(name ?? '').trim();
  if (!/^\d+$/.test(text)) return null;
  const race = Number(text);
  return Number.isSafeInteger(race) && race > 0 ? race : null;
};
const bossDisplayName = (raceId: number): string =>
  KNOWN_BOSS_RACE_NAMES[raceId] || 'Succubus Queen (variant unverified)';

/**
 * Label outgoing enemy-source tracks without treating them as player DPS.
 *
 * Evidence order:
 * 1. The source entity ID is a member of a reconstructed boss encounter.
 * 2. The source ID is a recorded target with a recognized boss race.
 * 3. A source has a numeric race-name matching a boss race corroborated by a
 *    recorded target/encounter in this session. This supports older captures
 *    that lost the source entity link, without treating arbitrary numbers as
 *    enemies. The association is to the race, not a guessed attempt.
 *
 * Unknown candidate Queen races require a positively reconstructed Phantasm
 * encounter. Never classify a player solely by a numeric-looking entity ID.
 */
export function identifyBossScatterSources(
  summary: FightSummary,
  encounters: BossReportEncounter[] = findReportEncounters(summary)
): Map<string, EnemyScatterIdentity> {
  const found = new Map<string, EnemyScatterIdentity>();
  const targets = summary.targets || {};
  const knownByRace = new Map<number, { name: string; encounterId: string }>();
  const add = (id: string, raceId: number, encounterId: string, name: string) => {
    if (!raceId || found.has(id)) return;
    found.set(id, { id, name: name + ' (Enemy)', encounterId, raceId,
      color: colors[found.size % colors.length] });
  };

  for (const encounter of encounters) {
    for (const id of encounter.entityIds) {
      const raceId = Number(targets[id]?.raceId || 0);
      if (!raceId) continue;
      const phase = raceId === 7600 ? ' — First Form' : raceId === 7601 ? ' — Second Form' : '';
      const name = bossDisplayName(raceId);
      knownByRace.set(raceId, { name, encounterId: encounter.id });
      add(id, raceId, encounter.id, name + phase);
    }
  }

  // Recognized standalone bosses and opening encounters can exist outside the
  // encounter timeline (e.g. Cailleach or a partial run with no boss HP trace).
  for (const [id, target] of Object.entries(targets)) {
    const raceId = Number(target.raceId || 0);
    const name = KNOWN_BOSS_RACE_NAMES[raceId];
    if (!name) continue;
    if (!knownByRace.has(raceId)) knownByRace.set(raceId, { name, encounterId: `race:${raceId}` });
    add(id, raceId, `race:${raceId}`, name);
  }

  // Enemy attacks can also originate from ordinary dungeon mobs, not only
  // bosses. Older captures sometimes assign those attackers an entity-ID-shaped
  // display name. Only accept an exact source/recorded-target entity match with
  // positive race and HP/spawn evidence; never infer identity from the numeric
  // display name itself. Player character race 10001 is explicitly excluded.
  for (const player of Object.values(summary.players || {})) {
    const id = String(player.id);
    if (found.has(id) || !/^\d+$/.test(String(player.name || '').trim())) continue;
    const target = targets[id];
    if (!target) continue;
    const raceId = Number(target.raceId || 0);
    if (!Number.isSafeInteger(raceId) || raceId <= 0 || raceId === 10001) continue;
    if (!target.seenAppear && !(target.hpHistory || []).some(h => Number(h.maxHp) > 0)) continue;
    const readableName = String(target.name || '').trim();
    const name = raceId === 7544 ? 'Gwyllgi' :
      readableName && !/^\d+$/.test(readableName) && readableName !== 'Unknown'
        ? readableName : `Enemy (Race ${raceId})`;
    add(id, raceId, `race:${raceId}`, name);
  }

  // Some old recordings use a numeric race-name for a source whose ID was not
  // retained among target entity IDs. Only a corroborated boss race qualifies.
  for (const player of Object.values(summary.players || {})) {
    if (found.has(String(player.id))) continue;
    const raceId = numericRaceName(player.name);
    if (raceId === null) continue;
    const recognized = knownByRace.get(raceId);
    if (recognized) add(String(player.id), raceId, recognized.encounterId, recognized.name);
  }
  return found;
}
