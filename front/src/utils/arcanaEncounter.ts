import type { FightSummary, PlayerStats } from '@/protocols';
import talentDataRaw from '../../../cmd/dilmeterapi/static_data/talents.json?raw';
import { getGroupMemberTargetIds } from './targetGrouping';
import { getClassColor } from '@/util';

// Resolve Arcana from observed skills, not the first player-wide Arcana label.
// Using the backend's authoritative registry avoids a second, drifting skill list.
type ArcanaIdentity = { name: string; icon: string; color: string };
type ArcanaEntry = ArcanaIdentity & { arcana_name: string; related_skill_ids: number[] };
// Vite's ?raw import keeps the Go registry as the single source of truth,
// without requiring TypeScript resolveJsonModule or an out-of-tree JSON module.
const talentData = JSON.parse(talentDataRaw) as { arcanas: Record<string, ArcanaEntry> };
const skillArcana = new Map<number, ArcanaIdentity>();
for (const entry of Object.values(talentData.arcanas)) {
  const arcana = { name: entry.arcana_name, icon: entry.icon, color: entry.color };
  for (const id of entry.related_skill_ids) skillArcana.set(Number(id), arcana);
}
// Lookup by exact skill ID; shared by the live meter and self-contained export.
// The color is resolved at rendering/export time so Appearance changes apply
// without changing the saved combat data or the authoritative skill mapping.
export function getSkillArcana(skillId: number): ArcanaIdentity | undefined {
  const arcana = skillArcana.get(Number(skillId));
  return arcana ? { ...arcana, color: getClassColor(arcana.name, arcana.color) } : undefined;
}

export type ArcanaResolution = {
  status: 'identified' | 'mixed' | 'unknown';
  names: string[];
  identities: ArcanaIdentity[];
  name: string;
  icon: string;
  color: string;
};
function classify(ids: Iterable<number>): ArcanaResolution {
  const found = new Map<string, ArcanaIdentity>();
  for (const id of ids) {
    const arcana = skillArcana.get(Number(id));
    if (arcana) found.set(arcana.name, { ...arcana, color: getClassColor(arcana.name, arcana.color) });
  }
  const names = [...found.keys()];
  const identities = names.map(name => found.get(name)!);
  if (names.length === 1) {
    const arcana = found.get(names[0])!;
    return { status: 'identified', names, identities, name: arcana.name, icon: arcana.icon, color: arcana.color };
  }
  return { status: names.length ? 'mixed' : 'unknown', names, identities,
    name: names.length ? 'Multiple Arcanas' : 'Unknown Arcana', icon: '', color: identities[0]?.color || '#808080' };
}
function skillIds(breakdowns: ({skills?: Record<number, { totalDamage?: number; count?: number }> } | undefined)[]): number[] {
  const ids: number[] = [];
  for (const breakdown of breakdowns) for (const [id, value] of Object.entries(breakdown?.skills || {})) {
    if (Number(value.totalDamage || 0) > 0 || Number(value.count || 0) > 0) ids.push(Number(id));
  }
  return ids;
}
// Timestamped observations determine the first-observed Arcana order where
// available. Older summaries without event timestamps still use recorded skills.
function chronologicalSkillIds(player: PlayerStats): number[] {
  const observed = [
    ...(player.damageEvents || []).map(hit => ({ at: Number(hit.atMs), id: Number(hit.skillId) })),
    ...(player.skillActivations || []).map(use => ({ at: Number(use.atMs), id: Number(use.skillId) }))
  ].filter(event => Number.isFinite(event.at) && event.at > 0 && Number.isFinite(event.id));
  observed.sort((a, b) => a.at - b.at);
  return observed.map(event => event.id);
}

// `targetId` is the exact selector scope (including reconstructed folder/level).
// A saved target breakdown is more robust than inferring a change from session order.
export function resolvePlayerArcana(player: PlayerStats, summary: FightSummary, targetId = ''): ArcanaResolution {
  if (!targetId) return classify([
    ...chronologicalSkillIds(player),
    ...skillIds([player.overallStats])
  ]);
  const members = getGroupMemberTargetIds(targetId, summary);
  const ids = members.length ? members : summary.targets[targetId] ? [targetId] : [];
  const scopedUses = (player.skillActivations || [])
    .filter(use => use.targetId && ids.includes(String(use.targetId)))
    .map(use => Number(use.skillId));
  return classify([...skillIds(ids.map(id => player.damageByTarget?.[id])), ...scopedUses]);
}

// Portable report's all-target encounter windows require timestamp evidence.
// A hit is evidence of an Arcana effect, NOT a count of activations.
export function resolvePlayerArcanaWindow(player: PlayerStats, startUnix: number, endUnix: number): ArcanaResolution {
  return classify((player.damageEvents || []).filter(hit => {
    const at = Number(hit.atMs) / 1000;
    return Number.isFinite(at) && at >= startUnix && at <= endUnix;
  }).map(hit => Number(hit.skillId)).concat(
    (player.skillActivations || []).filter(use => {
      const at = Number(use.atMs) / 1000;
      return Number.isFinite(at) && at >= startUnix && at <= endUnix;
    }).map(use => Number(use.skillId))));
}

// Exact report boss-entity scope (not a guessed interval).
export function resolvePlayerArcanaForIds(player: PlayerStats, ids: string[]): ArcanaResolution {
  const uses = (player.skillActivations || []).filter(use => use.targetId && ids.includes(String(use.targetId)))
    .map(use => Number(use.skillId));
  return classify([...skillIds(ids.map(id => player.damageByTarget?.[id])), ...uses]);
}

// The length of the painted portion is independent of Arcana count. Mixed
// identities transition smoothly, evenly spaced in first-observed order;
// stops are visual identifiers, not damage or duration weights.
export function arcanaBarBackground(identities: ArcanaIdentity[], fillPercent = 100, fallback = '#808080'): string {
  const width = Math.max(0, Math.min(100, fillPercent));
  if (identities.length < 2) return `linear-gradient(to right, ${identities[0]?.color || fallback} ${width}%, transparent ${width}%)`;
  const stops = identities.map((arcana, index) =>
    `${arcana.color} ${(width * index / (identities.length - 1)).toFixed(3)}%`
  );
  return `linear-gradient(to right, ${stops.join(', ')}, ${identities[identities.length - 1].color} ${width}%, transparent ${width}%)`;
}
