import type { TargetStats } from '@/protocols';
import { partitionMinorRuns } from './instanceMinorGrouping';

/** Candidate encounter enemies from Midir's race database and observed dungeon context.
 * This is a provisional minor-enemy classification, NOT mechanic or boss-phase proof.
 * 193818 (second Folamh) and 193809 (Giga Glas) stay unassigned as possible bosses.
 */
export const HIGH_TIER_ENCOUNTERS = [
  {key:'abyss', name:'Crom Bas Abyss', bossRace:193810,
   minorRaces:new Set([193811,193812,193813,193814,193815,193816,193817])},
  {key:'gate4', name:'Bri Leith — Gate 4', bossRace:7615,
   minorRaces:new Set([7616,7617,7618,7619,7620,7622])},
] as const;
export const HIGH_TIER_MINOR_PREFIX='group_high_tier_minor_';
export const HIGH_TIER_RUN_PREFIX='group_high_tier_run_';

export function findHighTierRuns(targets: Record<string,TargetStats>) {
  return HIGH_TIER_ENCOUNTERS.flatMap(def => {
    const bosses=Object.entries(targets).filter(([,t])=>Number(t.raceId)===def.bossRace && Number(t.startTime)>0)
      .sort((a,b)=>Number(a[1].startTime)-Number(b[1].startTime));
    const attempts=bosses.map(([,t])=>({startUnixSeconds:Number(t.startTime),endUnixSeconds:Number(t.endTime)||null}));
    const mobs=Object.entries(targets).filter(([,t])=>def.minorRaces.has(Number(t.raceId)))
      .map(([id,t])=>({id,startTime:Number(t.startTime)||0}));
    const partitions=partitionMinorRuns(mobs,attempts);
    // A run folder is created only with independently observed encounter entities.
    // Each segment's mobs are assigned by the same approximate midpoint rule as Crom.
    return partitions.map((part,index)=>{
      const boss=part.attemptIndex===null ? null : bosses[part.attemptIndex]?.[0] || null;
      const ids=part.members.map(m=>m.id);
      const token=`${def.key}_${index}`;
      return {token, name:`${def.name} — ${part.name}`, encounter:def.key,
        bossId:boss, minorIds:ids, entityIds:[...new Set([...(boss?[boss]:[]),...ids])],
        minorGroupId:HIGH_TIER_MINOR_PREFIX+token, runGroupId:HIGH_TIER_RUN_PREFIX+token,
        startTime:Math.min(...part.members.map(m=>Number(m.startTime)||0))};
    });
  });
}
