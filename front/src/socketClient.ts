// front/src/socketClient.ts

import { FightSummary, WebSocketMessage, PlayerInfo } from "@/protocols";

export class SocketClient {
  private socket?: WebSocket;

  // true -> connected, false -> disconnected or connecting
  public onConnect?: (isConnected: boolean) => void;
  public onSummary?: (summary: FightSummary) => void;
  public onPlayerBatchUpdate?: (players: PlayerInfo[]) => void;
  public onSystemError?: (msg: string) => void;
  public onSystemWarning?: (msg: string) => void;
  public onPacketDebug?: (info: any) => void; // Header only
  public onPacketDetails?: (details: any) => void; // Full details
  public onPacketStatus?: (status: {total: number, perSecond: number, lastPacketAt: string, lastOp: number, topOps?: {op: number, count: number, total: number}[]}) => void;
  public onAutodetectProgress?: (progress: {current: number, target: number}) => void;
  public onAutodetectDone?: (result: {ip: string, port: string}) => void;

  constructor(private url: string) {}

  public connect() {
    if (this.socket?.readyState === WebSocket.OPEN) {
      return;
    }

    const s = (this.socket = new WebSocket(this.url));

    setTimeout(() => {
      if (s.readyState == WebSocket.CONNECTING) {
        console.log("connection timeout...");
        s.close();
        this.socket = undefined;
      }
    }, 5000);

    s.onopen = () => {
      console.log("socket open");
      this.onConnect?.(true);
    };

    s.onmessage = (e) => {
      try {
        const msg = JSON.parse(e.data) as WebSocketMessage;
        
        // Backward compatibility: if message doesn't have a type, assume it's a summary
        // (This handles any potential race condition during deployment or cached clients)
        if (!msg.type && (msg as any).encounterDuration !== undefined) {
             this.onSummary?.(msg as any as FightSummary);
             return;
        }

        switch (msg.type) {
            case "summary":
                this.onSummary?.(msg.data as FightSummary);
                break;
            case "player_update_batch":
                this.onPlayerBatchUpdate?.(msg.data as PlayerInfo[]);
                break;
            case "system_error":
                this.onSystemError?.(msg.data as string);
                break;
            case "system_warning":
                this.onSystemWarning?.(msg.data as string);
                break;
            case "packet_debug":
                this.onPacketDebug?.(msg.data);
                break;
            case "packet_details":
                this.onPacketDetails?.(msg.data);
                break;
            case "packet_status":
                this.onPacketStatus?.(msg.data as any);
                break;
            case "autodetect_progress":
                this.onAutodetectProgress?.(msg.data as any);
                break;
            case "autodetect_done":
                this.onAutodetectDone?.(msg.data as any);
                break;
            default:
                console.warn("Unknown message type:", msg.type);
        }
      } catch (err) {
        console.error("Failed to parse socket message", err);
      }
    };

    s.onerror = (e) => {
      console.log("socket error", e);
      s.close();
    };

    s.onclose = (e) => {
      console.log("socket close", e);
      this.onConnect?.(false);
      s.close();
      this.socket = undefined;
      setTimeout(() => this.connect(), 3000);
    };
  }

  public send(data: any) {
    if (this.socket?.readyState === WebSocket.OPEN) {
        this.socket.send(JSON.stringify(data));
    } else {
        console.warn("Socket not open, cannot send:", data);
    }
  }

  public close() {
    this.onConnect = undefined;
    this.onSummary = undefined;
    this.onPlayerBatchUpdate = undefined;
    this.onSystemError = undefined;
    this.onSystemWarning = undefined;
    this.onPacketDebug = undefined;
    this.onPacketDetails = undefined;
    this.onPacketStatus = undefined;
    this.onAutodetectProgress = undefined;
    this.onAutodetectDone = undefined;

    if (this.socket) {
      this.socket.onclose = null; 
      this.socket.close();
      this.socket = undefined;
    }
  }
}
