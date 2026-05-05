export interface PlayerCacheInfo {
  id: string;
  name: string;
  raceId: number;
  guildName: string;
  totalLevel: number;
  combatPower: number;
}

export interface Session {
  id: string;
  name: string;
  startTime: number; // Unix timestamp
  endTime?: number; // Unix timestamp, optional for active sessions
  ndjsonLogPath: string;
  pcapngLogPath: string;
}
