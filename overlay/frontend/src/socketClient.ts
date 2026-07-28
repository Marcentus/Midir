import { FightSummary, WebSocketMessage } from "./types";

export class OverlaySocketClient {
  private ws: WebSocket | null = null;
  private serverUrl: string = "http://localhost:8030";
  private reconnectTimer: any = null;

  public onConnectChange?: (connected: boolean) => void;
  public onSummary?: (summary: FightSummary) => void;
  public onError?: (msg: string) => void;

  constructor(serverUrl: string = "http://localhost:8030") {
    this.serverUrl = this.normalizeUrl(serverUrl);
  }

  public updateServerUrl(url: string) {
    const normalized = this.normalizeUrl(url);
    if (this.serverUrl !== normalized) {
      this.serverUrl = normalized;
      this.reconnect();
    }
  }

  private normalizeUrl(url: string): string {
    let clean = url.trim().replace(/\/+$/, "");
    if (!clean.startsWith("http://") && !clean.startsWith("https://")) {
      clean = "http://" + clean;
    }
    return clean;
  }

  public getWsUrl(): string {
    const wsBase = this.serverUrl.replace(/^http/, "ws");
    return `${wsBase}/ws`;
  }

  public connect() {
    if (this.ws && (this.ws.readyState === WebSocket.OPEN || this.ws.readyState === WebSocket.CONNECTING)) {
      return;
    }

    try {
      const wsUrl = this.getWsUrl();
      console.log("[OverlaySocket] Connecting to", wsUrl);
      this.ws = new WebSocket(wsUrl);

      this.ws.onopen = () => {
        console.log("[OverlaySocket] Connected");
        this.onConnectChange?.(true);
        // Fetch initial summary via REST just in case
        this.fetchLiveSummary();
      };

      this.ws.onmessage = (evt) => {
        try {
          const msg = JSON.parse(evt.data) as WebSocketMessage;
          if (msg.type === "summary") {
            this.onSummary?.(msg.data as FightSummary);
          } else if (!msg.type && (msg as any).encounterDuration !== undefined) {
            this.onSummary?.(msg as any as FightSummary);
          }
        } catch (e) {
          console.error("[OverlaySocket] Failed to parse message", e);
        }
      };

      this.ws.onerror = (err) => {
        console.warn("[OverlaySocket] Error:", err);
        this.onConnectChange?.(false);
      };

      this.ws.onclose = () => {
        console.log("[OverlaySocket] Disconnected. Reconnecting in 3s...");
        this.onConnectChange?.(false);
        this.scheduleReconnect();
      };
    } catch (e) {
      console.error("[OverlaySocket] Connection exception", e);
      this.onConnectChange?.(false);
      this.scheduleReconnect();
    }
  }

  private scheduleReconnect() {
    if (this.reconnectTimer) clearTimeout(this.reconnectTimer);
    this.reconnectTimer = setTimeout(() => this.connect(), 3000);
  }

  public reconnect() {
    if (this.reconnectTimer) clearTimeout(this.reconnectTimer);
    if (this.ws) {
      this.ws.onclose = null;
      this.ws.close();
      this.ws = null;
    }
    this.connect();
  }

  public async fetchLiveSummary(): Promise<FightSummary | null> {
    try {
      const res = await fetch(`${this.serverUrl}/api/state/summary`);
      if (res.ok) {
        const data = await res.json();
        this.onSummary?.(data);
        return data;
      }
    } catch (e) {
      console.error("[OverlaySocket] Fetch live summary failed", e);
    }
    return null;
  }

  public async saveSession(name: string = "Overlay Saved Session"): Promise<boolean> {
    try {
      const res = await fetch(`${this.serverUrl}/api/sessions/save`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ name }),
      });
      return res.ok || res.status === 201;
    } catch (e) {
      console.error("[OverlaySocket] Save session failed", e);
      return false;
    }
  }

  public async clearSession(): Promise<boolean> {
    try {
      const res = await fetch(`${this.serverUrl}/api/state/clear`, {
        method: "POST",
      });
      if (res.ok || res.status === 204) {
        // Trigger fetch of empty summary
        this.fetchLiveSummary();
        return true;
      }
    } catch (e) {
      console.error("[OverlaySocket] Clear session failed", e);
    }
    return false;
  }

  public close() {
    if (this.reconnectTimer) clearTimeout(this.reconnectTimer);
    if (this.ws) {
      this.ws.onclose = null;
      this.ws.close();
      this.ws = null;
    }
  }
}
