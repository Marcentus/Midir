import type { FightSummary, TargetStats } from '@/protocols';
import { findGate1Attempts } from './gate1Encounter';
import { findGlennAttempts } from './glennEncounter';
import { getGroupMemberTargetIds } from './targetGrouping';

export type BossReportEncounter = {
  id: string; label: string; entityIds: string[]; start: number; end: number;
  hp: [number,number][]; phases: {raceId:number;name:string;hp:[number,number][]}[];
  clear: 'observed-zero-hp'|'unverified';
};
function seconds(v: unknown): number { const n=Number(v); return Number.isFinite(n)?n:0; }
export function targetHP(t: TargetStats | undefined, origin: number, start = origin): [number,number][] {
  if (!t) return [];
  const offset=seconds(t.startTime)-origin;
  return (t.hpHistory || []).filter(h=>seconds(h.maxHp)>0 && Number.isFinite(Number(h.time)))
    .map(h=>[Math.max(0,offset+seconds(h.time)-(start-origin)),Math.max(0,Math.min(100,100*seconds(h.currentHp)/seconds(h.maxHp)))] as [number,number])
    .sort((a,b)=>a[0]-b[0]);
}
const BOSS_RACES: Record<number,string> = {
  7602:'Gate 2 — Bronntanas', 7603:'Gate 3 — Midir the Beloved', 7615:'Gate 4',
  195012:'Succubus Queen', 193364:'Crom Bas — Irusan', 193359:'Crom Bas — Giga Glas Ghaibhleann',
  193810:'Crom Bas Abyss'
};
/** A timeline of *observed* bosses, not a claim that every dungeon mechanic was detected. */
export function findReportEncounters(summary: FightSummary, selectedTargetId = ''): BossReportEncounter[] {
  const targets=summary.targets || {}, origin=seconds(summary.startTime);
  const entries: BossReportEncounter[]=[];
  const claimed=new Set<string>();
  function append(id:string, label:string, ids:string[], startUnix:number, endUnix:number, clear:boolean, phaseIDs?:string[]) {
    const validIds=ids.filter(key=>Boolean(targets[key]));
    if (!validIds.length || !Number.isFinite(startUnix) || startUnix<=0) return;
    const start=Math.max(0,startUnix-origin), end=Math.max(start+1,(endUnix>startUnix?endUnix:startUnix+1)-origin);
    const phases=(phaseIDs || []).map(key=>({raceId:seconds(targets[key]?.raceId),name:targets[key]?.name || 'Boss form',hp:targetHP(targets[key],origin,startUnix)}));
    const hp=phaseIDs?.length ? phases.flatMap(p=>p.hp).sort((a,b)=>a[0]-b[0]) : targetHP(targets[id],origin,startUnix);
    entries.push({id,label,entityIds:validIds,start,end,hp,phases,clear:clear?'observed-zero-hp':'unverified'});
    validIds.forEach(key=>claimed.add(key));
  }
  for (const [index,a] of findGate1Attempts(targets).entries()) {
    const second=a.replacementEntityId;
    append(a.firstEntityId,`Gate 1 — Vertrag · Attempt ${index+1}`,
      a.entityIds,a.startUnixSeconds,a.endUnixSeconds || 0,a.completion==='verified-clear',
      a.entityIds);
    if(second){
      const entry=entries.at(-1);
      if(entry?.id===a.firstEntityId){
        // 7601's separate full HP bar is mechanically the bottom 50%.
        const secondPhase=entry.phases.find(p=>p.raceId===7601);
        if(secondPhase) secondPhase.hp=secondPhase.hp.map(([t,p])=>[t,Math.min(50,p)]);
        entry.hp=entry.phases.flatMap(p=>p.hp).sort((x,y)=>x[0]-y[0]);
      }
    }
  }
  for (const [index,a] of findGlennAttempts(targets).entries())
    append(a.firstEntityId,`Glenn Bearna — Primeval Winter · Attempt ${index+1}`,a.entityIds,
      a.startUnixSeconds,a.endUnixSeconds || 0,a.completion==='verified-clear',a.entityIds);
  const raceCount: Record<number,number> = {};
  for (const [id,t] of Object.entries(targets).sort((a,b)=>seconds(a[1].startTime)-seconds(b[1].startTime))) {
    const race=seconds(t.raceId);
    if (claimed.has(id) || !BOSS_RACES[race] || !(t.hpHistory || []).length) continue;
    raceCount[race]=(raceCount[race] || 0)+1;
    append(id,`${BOSS_RACES[race]} · Attempt ${raceCount[race]}`,[id],seconds(t.startTime),seconds(t.endTime),
      (t.hpHistory || []).some(h=>seconds(h.maxHp)>0 && seconds(h.currentHp)<=0));
  }
  if (selectedTargetId) {
    const ids=getGroupMemberTargetIds(selectedTargetId,summary);
    const matching=entries.filter(e=>e.entityIds.some(id=>ids.includes(id)));
    if(matching.length) return matching;
    // Target selected in the app may be outside the supported boss registry.
    const target=targets[ids[0]];
    if(target && seconds(target.startTime)>0) {
      append(ids[0],target.name || 'Selected target',ids,seconds(target.startTime),seconds(target.endTime),
        ids.some(id=>(targets[id].hpHistory||[]).some(h=>seconds(h.maxHp)>0 && seconds(h.currentHp)<=0)));
      return entries.slice(-1);
    }
    return [];
  }
  return entries.sort((a,b)=>a.start-b.start);
}
