export interface SkillStats {
  id: number;
  count: number;
  critCount: number;
  totalDamage: number;
  totalDamageCrit: number;
  totalDamageNonCrit: number;
  maxDamage: number;
}

export interface DamageBreakdown {
  totalDamage: number;
  hitCount: number;
  critCount: number;
  dps: number;
  critRate: number;
  startTime?: number;
  endTime?: number;
  skills?: { [skillId: number]: SkillStats };
}

export interface PlayerStats {
  id: string;
  name: string;
  talentIcon?: string;
  talentName?: string;
  talentColor?: string;
  overallStats: DamageBreakdown;
  damageByTarget: { [targetId: string]: DamageBreakdown };
}

export interface TargetStats {
  name: string;
  raceId?: number;
  seenDead?: boolean;
  seenAppear?: boolean;
  disappeared?: boolean;
  startTime?: number;
  endTime?: number;
}

export interface FightSummary {
  encounterDuration: number;
  startTime?: number;
  endTime?: number;
  totalDamage: number;
  players: { [playerId: string]: PlayerStats };
  targets: { [targetId: string]: TargetStats };
}

export interface WebSocketMessage {
  type: string;
  data: any;
}

export interface OverlaySettings {
  serverUrl: string;
  bgOpacity: number; // 0 to 1
  fontSize: number;  // px (e.g., 11 to 18)
  showTimer: boolean;
  hideNames?: boolean;
  isDragLocked?: boolean;
  isResizeLocked?: boolean;
  alwaysOnTop?: boolean;
  selectedTargetId: string; // "" for All Targets
}
