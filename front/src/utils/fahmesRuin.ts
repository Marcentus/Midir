import type { TargetStats } from '@/protocols';

export const FAHMES_STAGE_PREFIX = 'group_fahmes_stage_';
export const FAHMES_LEVEL_PREFIX = 'group_fahmes_level_';
/** Stage is encoded as 121xxx–126xxx. Monster race suffixes identify
 * stage-local spawns, not dependable wave order across all levels. */
export function fahmesStageOf(raceId: unknown): number | null {
  const id=Number(raceId), family=Math.floor(id/1000), suffix=id%1000;
  return Number.isInteger(id) && family>=121 && family<=126 && suffix>=1 && suffix<1000 ? family-120 : null;
}
export interface FahmesLevel {index:number;entityIds:string[];startUnixSeconds:number;endUnixSeconds:number}
export interface FahmesStage {stage:number;attempt:number;entityIds:string[];levels:FahmesLevel[];startUnixSeconds:number;endUnixSeconds:number}
/** Six independently enterable stages. Levels are sequential, though their
 * constituent waves can overlap. Use spawn bursts, never assume that three
 * sequential race IDs correspond to three discrete time windows. */
export function findFahmesStages(targets: Record<string, TargetStats>):FahmesStage[] {
  const result:FahmesStage[]=[];
  for(let stage=1;stage<=6;stage++) {
    const items=Object.entries(targets).filter(([,t])=>fahmesStageOf(t.raceId)===stage && Number(t.startTime)>0)
      .sort((a,b)=>Number(a[1].startTime)-Number(b[1].startTime)||a[0].localeCompare(b[0]));
    if(!items.length)continue;
    // Same stage can be replayed in one session; substantial gaps mark a new
    // challenge. Within an attempt, level boundaries are clustered by spawn.
    const attempts: typeof items[]=[];
    for(const item of items){
      const last=attempts.at(-1), previous=last?.at(-1);
      if(!last || (previous && Number(item[1].startTime)-Number(previous[1].startTime)>180))attempts.push([item]);
      else last.push(item);
    }
    for(let a=0;a<attempts.length;a++) {
      const attempt=attempts[a], levelEntries:typeof items[]=[];
      for(const item of attempt) {
        const previous=levelEntries.at(-1)?.at(-1);
        if(!previous || Number(item[1].startTime)-Number(previous[1].startTime)>12)levelEntries.push([item]);
        else levelEntries[levelEntries.length-1].push(item);
      }
      // A single level may contain delayed enemy spawns, so a fixed 12-second
      // gap can produce more than five clusters even for a complete clear.
      // A fully observed Fahmes stage contains five levels of 30 enemies each.
      // For that specific, corroborated case, use chronological groups of
      // 30 even if delayed spawns falsely merge two nearby levels. Partial
      // or ambiguous captures retain the conservative timing result.
      const usableEntries = attempt.length === 150
        ? Array.from({length: 5}, (_, i) => attempt.slice(i * 30, (i + 1) * 30))
        : levelEntries.length <= 5 ? levelEntries : [];
      const levels=usableEntries.map((members,index)=>({
        index:index+1,entityIds:members.map(([id])=>id),
        startUnixSeconds:Number(members[0][1].startTime),
        endUnixSeconds:Math.max(...members.map(([,t])=>Number(t.endTime)||Number(t.startTime)))
      }));
      result.push({stage,attempt:a+1,entityIds:attempt.map(([id])=>id),levels,
        startUnixSeconds:Number(attempt[0][1].startTime),
        endUnixSeconds:Math.max(...attempt.map(([,t])=>Number(t.endTime)||Number(t.startTime)))});
    }
  }
  return result.sort((a,b)=>a.startUnixSeconds-b.startUnixSeconds);
}
