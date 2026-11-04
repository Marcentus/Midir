/** Assign minor mobs to a recorded boss attempt when possible.
 * Between two boss attempts there is no explicit dungeon instance token in
 * TargetStats, so the midpoint of the between-boss interval is an estimated
 * run boundary, not protocol proof. Keep unanchored combat clusters separate.
 */
export interface TimedMinor { id:string; startTime?:number }
export interface TimedAttempt { startUnixSeconds:number; endUnixSeconds:number|null }
export interface MinorRun<T extends TimedMinor> { name:string; members:T[]; attemptIndex:number|null }
export function partitionMinorRuns<T extends TimedMinor>(mobs:T[],attempts:TimedAttempt[],gapSeconds=15*60):MinorRun<T>[] {
  const ordered=[...mobs].sort((a,b)=>Number(a.startTime||0)-Number(b.startTime||0)||a.id.localeCompare(b.id));
  if (!ordered.length)return [];
  const anchors=[...attempts].sort((a,b)=>a.startUnixSeconds-b.startUnixSeconds);
  const bounds:number[]=[];
  for(let i=0;i<anchors.length-1;i++){
    const end=Number(anchors[i].endUnixSeconds)||anchors[i].startUnixSeconds;
    const next=anchors[i+1].startUnixSeconds;
    bounds.push(end<next ? (end+next)/2 : next);
  }
  const buckets=new Map<number,T[]>();
  for(const mob of ordered){
    const at=Number(mob.startTime||0);
    // Each attempt owns the mobs leading to its boss and those immediately
    // following it, up to the midpoint toward the next recorded attempt.
    let index=0;while(index<bounds.length && at>=bounds[index])index++;
    if(!buckets.has(index))buckets.set(index,[]);
    buckets.get(index)!.push(mob);
  }
  const runs:MinorRun<T>[]=[];
  for(const [bucket,members] of buckets){
    let segments:T[][]=[];
    for(const mob of members){
      const prev=segments[segments.length-1];
      const last=prev?.[prev.length-1];
      if(!prev || Number(mob.startTime||0)-Number(last?.startTime||0)>gapSeconds)segments.push([mob]);
      else prev.push(mob);
    }
    for(let i=0;i<segments.length;i++)runs.push({
      name: anchors[bucket] ? `Run ${bucket+1}${segments.length>1?` · Segment ${i+1}`:''}` : `Combat segment ${runs.length+1}`,
      members:segments[i],attemptIndex:anchors[bucket]?bucket:null,
    });
  }
  return runs;
}
