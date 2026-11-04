import type { TargetStats } from '@/protocols';
import { findGate1Attempts } from './gate1Encounter';

// Spawn-wave grouping is descriptive. It is NOT proof of a completed mechanic
// or of a clear. Entity IDs and independent condition histories are retained.
export type MechanicGroup = {
  type: 'bri-gate-1-giants' | 'bri-gate-3-halos';
  label: string;
  phaseHpPercent: number;
  expectedMembers: number;
  memberEntityIds: string[];
  bossEntityIds: string[];
  firstSpawnUnixSeconds: number;
  evidence: 'observed-spawn-wave' | 'incomplete-spawn-wave';
  phaseIntervalVerified: false;
};

const GIANTS = new Set([7611,7612,7613,7614]);
const validStart = (target: TargetStats) => Number.isFinite(target.startTime) && Number(target.startTime) > 0;

/**
 * Group each Gate 1 giant wave within its own attempt. A new first-form 7600
 * starts a new attempt even if it shares one NDJSON file with earlier wipes.
 * Distinct race IDs 7611–7614 were observed in uploaded cleared/failure logs.
 */
export function findMechanicGroups(targets: Record<string,TargetStats>): MechanicGroup[] {
  const all = Object.entries(targets).filter(([,t]) => validStart(t)).sort((a,b) =>
    Number(a[1].startTime)-Number(b[1].startTime) || a[0].localeCompare(b[0]));
  const result: MechanicGroup[] = [];
  const attempts = findGate1Attempts(targets);
  for (let i=0;i<attempts.length;i++) {
    const attempt=attempts[i];
    const next=attempts[i+1]?.startUnixSeconds ?? Infinity;
    const end=Math.min(attempt.endUnixSeconds ?? Infinity,next);
    const giants=all.filter(([,t]) => GIANTS.has(Number(t.raceId)) &&
      Number(t.startTime)>=attempt.startUnixSeconds && Number(t.startTime)<end);
    // Four different giant race IDs are expected for one wave. Separate
    // repeats into the next wave without collapsing distinct entity IDs.
    let wave: typeof giants=[];
    // Explicit ordinal wave construction; do not fabricate unobserved members.
    const batches: (typeof giants)[]=[];
    for(const giant of giants){
      if(wave.some(([,t])=>t.raceId===giant[1].raceId) || wave.length===4){batches.push(wave);wave=[];}
      wave.push(giant);
    }
    if(wave.length)batches.push(wave);
    batches.slice(0,2).forEach((members,index)=>result.push({
      type:'bri-gate-1-giants',label:index===0?'Gate 1 giants — 70%':'Gate 1 giants — 25%',
      phaseHpPercent:index===0?70:25,expectedMembers:4,
      memberEntityIds:members.map(([id])=>id),bossEntityIds:attempt.entityIds,
      firstSpawnUnixSeconds:Number(members[0][1].startTime),
      evidence:members.length===4?'observed-spawn-wave':'incomplete-spawn-wave',
      phaseIntervalVerified:false
    }));
  }
  // Gate 3 uses the same race IDs for several roles. Distinguish halo
  // families by actual observed max HP, not numeric race ID or a name test.
  // The boss's phase HP, when recorded, separates 80% and 40% 38M waves.
  const bosses=all.filter(([,t])=>t.raceId===7603);
  const near=(actual:number,reference:number)=>Math.abs(actual-reference)/reference<0.08;
  bosses.forEach(([bossId,boss],bossIndex)=>{
    const next=Number(bosses[bossIndex+1]?.[1].startTime ?? Infinity);
    const end=Math.min(Number(boss.endTime || Infinity),next);
    const halos=all.filter(([,t]) => Number(t.raceId)>=7604 && Number(t.raceId)<=7608 &&
      Number(t.startTime)>=Number(boss.startTime) && Number(t.startTime)<end);
    // Saved-session HP samples are seconds relative to this particular boss's
    // startTime. Normalize to the same Unix-second clock used for halo spawns.
    const bossSamples=(boss.hpHistory||[])
      .filter(h=>Number(h.maxHp)>0 && Number.isFinite(Number(h.time)))
      .map(h=>({at:Number(boss.startTime)+Number(h.time), percent:100*Number(h.currentHp)/Number(h.maxHp)}))
      .sort((a,b)=>a.at-b.at);
    // A threshold may be crossed between samples (80.06 -> 79.94) or settle
    // fractionally below it (39.99). Identify the *first* nearby crossing, not
    // exact floating-point equality and not the report's HP-search bucketing.
    const phaseStart=(threshold:number):number|null=>{
      for(let i=0;i<bossSamples.length;i++){
        const now=bossSamples[i], prev=bossSamples[i-1];
        if ((prev && prev.percent>threshold && now.percent<=threshold && now.percent>=threshold-2) ||
            (Math.abs(now.percent-threshold)<=0.35 && (!prev || prev.percent>threshold+0.35)))
          return now.at;
      }
      return null;
    };
    const starts={80:phaseStart(80),40:phaseStart(40)};
    const buckets: Record<'80'|'60'|'40',typeof halos>={'80':[],'60':[],'40':[]};
    const hpFamily=(target:TargetStats):number|null=>{
      // A spawn can initially carry placeholder HP, so use the most frequently
      // observed compatible max-HP family rather than trusting its first packet.
      const observations=(target.hpHistory||[]).map(h=>Number(h.maxHp)).filter(h=>h>0);
      for(const family of [63800000,38300000,23500000]){
        if(observations.some(h=>near(h,family))) return family;
      }
      return null;
    };
    for(const member of halos){
      const hp=hpFamily(member[1]);
      if(hp===63800000){buckets['60'].push(member);continue;}
      if(hp!==38300000)continue; // Ambient 23M Shards never become mechanic halos.
      const at=Number(member[1].startTime);
      const candidates=([80,40] as const).filter(phase=>{
        const start=starts[phase];
        // A little pre-spawn packet skew is expected. Cap association time to
        // avoid swallowing unrelated later spawns when a phase can't be bounded.
        const next=phase===80 ? starts[40] ?? Infinity : Infinity;
        return start!==null && at>=start-5 && at<Math.min(start+100,next);
      });
      if(candidates.length===1)buckets[String(candidates[0]) as '80'|'40'].push(member);
      // Ambiguous waves stay in Other enemies; do not invent phase evidence.
    }
    ([{key:'80' as const,count:3},{key:'60' as const,count:3},{key:'40' as const,count:5}]).forEach(({key,count})=>{
      const members=buckets[key];if(!members.length)return;
      result.push({type:'bri-gate-3-halos',label:`Gate 3 halos — ${key}%`,
        phaseHpPercent:Number(key),expectedMembers:count,
        memberEntityIds:members.map(([id])=>id),bossEntityIds:[bossId],
        firstSpawnUnixSeconds:Number(members[0][1].startTime),
        evidence:members.length>=count?'observed-spawn-wave':'incomplete-spawn-wave',
        phaseIntervalVerified:false});
    });
  });
  return result;
}
