// protocols.ts

export interface SkillStats {
  id: number;
  count: number;
  critCount: number;
  totalDamage: number;
  totalDamageCrit: number;
  totalDamageNonCrit: number;
  maxDamage: number;
  maxDamageCrit: number;
  maxDamageNonCrit: number;
  uses?: number; // New field
}

// NEW: Interface for condition statistics
export interface ConditionMetaStats {
  metaData: string;
  uptime: number;
  duration: number;
  attackers: number[];
}

// NEW: Interface for condition intervals
export interface ConditionInterval {
  start: number;
  end: number;
  metaData: string;
  attackerId: number;
}

export interface ConditionStats {
  id: number;
  uptime: number;
  duration: number;
  intervals?: ConditionInterval[];
  metaBreakdown?: ConditionMetaStats[];
}

export interface DamageBreakdown {
  totalDamage: number;
  hitCount: number;
  critCount: number;
  dps: number;
  critRate: number;
  startTime?: number;
  endTime?: number;
  skills: { [skillId: number]: SkillStats };
  conditions?: { [id: number]: ConditionStats };
}

export interface DamageTimelineEvent {
  timestamp: number;
  skillId: number;
  targetId: string;
  targetName: string;
  damage: number;
  currentHp: number;
  maxHp: number;
  isCritical?: boolean;
  overkill?: number;
}

export interface PlayerStats {
  id: string;
  name: string;
  talentIcon?: string;
  talentName?: string;
  talentColor?: string;
  missingAppearPacket: boolean; // NEW: Cache warning flag
  overallStats: DamageBreakdown;
  damageByTarget: { [targetId: string]: DamageBreakdown };
  deaths?: number[];
  damageTimeline?: DamageTimelineEvent[];
}

// --- START: NEW INTERFACES FOR DAMAGE TAKEN ---

export interface DamageTakenDetails {
  attackerId: number;
  attackerName: string;
  skillId: number;
  totalDamage: number;
  totalManaDamage: number; // NEW
  hitCount: number;
  minDamage: number;
  maxDamage: number;
}

export interface PlayerDamageTakenStats {
  playerId: string;
  playerName: string;
  totalDamage: number;
  totalManaDamage: number; // NEW
  // The key is a composite string "attackerID-skillID"
  breakdown: { [key: string]: DamageTakenDetails };
}

// --- END: NEW INTERFACES FOR DAMAGE TAKEN ---

export interface TargetHPPoint {
  time: number;
  currentHp: number;
  maxHp: number;
}

export interface TargetStats {
  name: string;
  raceId?: number;
  conditions?: { [id: number]: ConditionStats };
  seenDead?: boolean;
  seenAppear?: boolean;
  disappeared?: boolean;
  startTime?: number;
  endTime?: number;
  hpHistory?: TargetHPPoint[];
}

// NEW: ActiveCondition matches the backend struct for currently active conditions
export interface ActiveCondition {
  start: number;
  disableAt?: number; // Unix timestamp
  metaData: string;
  attackerId: number;
}

// NEW: EntityState represents an entity currently in the area.
export interface EntityState {
  id: string;
  name: string;
  raceId: number;
  conditions?: { [id: number]: ActiveCondition };
  currentHp: number;
  maxHp: number;
  category: string;
}

export interface FightSummary {
  encounterDuration: number;
  startTime?: number;
  endTime?: number;
  totalDamage: number;
  players: { [playerId: string]: PlayerStats };
  targets: { [targetId: string]: TargetStats };
  damageTaken: { [playerId: string]: PlayerDamageTakenStats };
  graphData?: { [targetId: string]: { [playerId: string]: GraphDataPoint[] } };
  currentEntities?: EntityState[]; // NEW: List of entities currently in the area
  partyBuffs?: PartyBuff[]; // NEW: Party buff metrics
}

export interface PartyBuffMetric {
  label: string;
  highest: number;
  highestUptime: number;
  weightedAvg: number;
}

export interface PartyBuff {
  id: number;
  metrics: PartyBuffMetric[];
}

export interface GraphDataPoint {
  time: number;
  totalDamage: number;
  rollingDPS: number;
}

export interface DamageHitInfo {
  attackerId: string;
  attackerName: string;
  skillId: number;
  skillName?: string;
  damage: number;
  isCritical?: boolean;
  timestamp: string;
}

export interface HPValidationEvent {
  timestamp: number;
  entityId: string;
  entityName: string;
  lastHp: number;
  newHp: number;
  maxHp: number;
  actualDelta: number;
  expectedHp: number;
  pendingDamage: number;
  damageHits: DamageHitInfo[];
  status: "success" | "mismatch" | "heal";
}

export interface WebSocketMessage {
  type: "summary" | "player_update_batch" | "system_error" | "system_warning" | "packet_debug" | "packet_details" | "packet_status" | "autodetect_progress" | "autodetect_done" | "hp_validation";
  data: any;
}

export interface PlayerInfo {
  id: number;
  name: string;
  raceId: number;
  guildName?: string;
  totalLevel?: number;
  equipment?: Record<string, number>;
  lastUpdated: number;
}
