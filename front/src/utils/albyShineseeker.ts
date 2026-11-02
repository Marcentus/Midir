import type { TargetStats } from '@/protocols';

export const ALBY_RACES = new Set([195000,195001,195002,195003,195004,195005,195006,195007,195008,195009,195010,195011]);
export const ALBY_START_RACE = 195000; // Vampire Bat, first wave with Country Rat
export const ALBY_FINAL_RACES = new Set([195009,195010,195011]);
export const ALBY_GROUP_PREFIX = 'group_alby_run_';
export const ALBY_WAVE_PREFIX = 'group_alby_wave_';
export const ALBY_WAVES = [
  {label:'Rats and Bats', races:new Set([195000,195001])},
  {label:'Skeletons', races:new Set([195002,195003,195004])},
  {label:'Black Wizards and Ghost Armor', races:new Set([195005,195006])},
  {label:'Gargoyles', races:new Set([195007,195008])},
  {label:"Final wave — Cichol's Followers", races:ALBY_FINAL_RACES},
];

/** Separate runs using the appearance of a new first wave. This is a
 * race/time-based reconstruction, not a dungeon-instance ID from the wire. */
export function findAlbyRuns(targets: Record<string, TargetStats>) {
  const entries=Object.entries(targets)
    .filter(([,t])=>ALBY_RACES.has(Number(t.raceId)) && Number(t.startTime)>0)
    .sort((a,b)=>Number(a[1].startTime)-Number(b[1].startTime));
  const batSpawns=entries.filter(([,t])=>Number(t.raceId)===ALBY_START_RACE);
  const starts:number[]=[];
  for(const [,t] of batSpawns){const at=Number(t.startTime);if(!starts.length || at-starts[starts.length-1]>5)starts.push(at)}
  return starts.map((start,index)=>{
    const end=Math.min(starts[index+1]??Infinity,start+15*60);
    const members=entries.filter(([,t])=>Number(t.startTime)>=start && Number(t.startTime)<end);
    const waveIds=ALBY_WAVES.map(w=>members.filter(([,t])=>w.races.has(Number(t.raceId))).map(([id])=>id));
    const followerIds=waveIds[4];
    const finished=followerIds.length>0 && followerIds.every(id=>{
      const t=targets[id];return Boolean(t.seenDead) && Boolean(t.hpHistory?.some(h=>Number(h.maxHp)>0 && Number(h.currentHp)<=0));
    });
    return {index:index+1,firstEntityId:members[0]?.[0]??'',entityIds:members.map(([id])=>id),waveIds,
      startUnixSeconds:start,endUnixSeconds:Math.max(start,...members.map(([,t])=>Number(t.endTime)||Number(t.startTime))),
      completion: finished ? 'final-wave-defeated' as const : 'unverified' as const};
  });
}
