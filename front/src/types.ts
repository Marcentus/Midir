export interface PlayerCacheInfo {
  id: string;
  name: string;
  raceId: number;
  guildName: string;
  totalLevel: number;
  combatPower: number;
}

export interface SessionSummaryPlayer {
  name: string;
  dps: number;
  arcanaName: string;
  arcanaIcon: string;
  totalDamage: number;
}

export interface SessionSummaryEnemy {
  name: string;
  raceId: number;
  totalDamage: number;
  startTime?: number;
  endTime?: number;
}

export interface SessionSummaryData {
  startTime: number;
  duration: number;
  totalDamage: number;
  players: SessionSummaryPlayer[];
  enemies: SessionSummaryEnemy[];
}

export interface Session {
  id: string;
  name: string;
  startTime: number; // Unix timestamp
  endTime?: number; // Unix timestamp, optional for active sessions
  summary?: SessionSummaryData;
}
