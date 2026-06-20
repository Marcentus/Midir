import { FightSummary } from "./protocols";
import { Session } from "./types";

// A simple interface to describe the shape of our embedded JSON data.
interface DataItem {
  name: string;
  iconUrl?: string;
}

let loadingCount: any;

// --- SESSION API FUNCTIONS ---

export async function getSessions(): Promise<Session[]> {
  const data = await httpCall<Session[]>("/api/sessions");
  return data || [];
}

export async function getSessionSummary(
  sessionId: string
): Promise<FightSummary> {
  return httpCall<FightSummary>(`/api/sessions/${sessionId}/summary`);
}

export async function getSessionLog(sessionId: string): Promise<string> {
  const buf = await httpCallRaw(`/api/sessions/${sessionId}/log`);
  return new TextDecoder("utf-8").decode(buf);
}

export async function startSession(): Promise<Session> {
  return httpCall<Session>("/api/sessions/start", { method: "POST" });
}

export async function saveSession(newName: string): Promise<void> {
  await fetchWithOpts("/api/sessions/save", {
    method: "POST",
    body: JSON.stringify({ name: newName }),
    headers: { "Content-Type": "application/json" },
  });
}

export async function stopSession(): Promise<void> {
  await httpCall<void>("/api/sessions/stop", { method: "POST" });
}

export async function renameSession(
  sessionId: string,
  newName: string
): Promise<Session> {
  const payload = { name: newName };
  return httpCall<Session>(`/api/sessions/${sessionId}`, {
    method: "PUT",
    body: JSON.stringify(payload),
    headers: { "Content-Type": "application/json" },
  });
}

export async function deleteSession(sessionId: string): Promise<void> {
  await httpCall<void>(`/api/sessions/${sessionId}`, {
    method: "DELETE",
  });
}

export async function migrateSession(sessionId: string): Promise<void> {
  await httpCall<void>(`/api/sessions/${sessionId}/migrate`, { method: "POST" });
}

export async function migrateAllSessions(): Promise<{ migrated: number }> {
  return httpCall<{ migrated: number }>("/api/sessions/migrate-all", { method: "POST" });
}

export async function clearBackendState(): Promise<void> {
  await httpCall<void>("/api/state/clear", {
    method: "POST",
    disableLoading: true,
  });
}

export async function getLiveSummary(): Promise<FightSummary> {
  return httpCall<FightSummary>("/api/state/summary");
}

// --- EMBEDDED DATA API FUNCTIONS ---

export async function getSkills(): Promise<Record<string, DataItem>> {
  return httpCall<Record<string, DataItem>>("/api/data/skills.json");
}

export async function getRaces(): Promise<Record<string, DataItem>> {
  return httpCall<Record<string, DataItem>>("/api/data/races.json");
}

export async function getConditions(): Promise<Record<string, DataItem>> {
  return httpCall<Record<string, DataItem>>("/api/data/conditions.json");
}

export interface Overrides {
  conditions: Record<string, DataItem>;
  skills: Record<string, DataItem>;
}

export async function getOverrides(): Promise<Overrides> {
  return httpCall<Overrides>("/api/data/overrides.json").catch(() => ({ conditions: {}, skills: {} }));
}


// --- HTTP HELPER FUNCTIONS ---

export type HttpCallOpt = {
  method?: string;
  headers?: Record<string, string>;
  body?: BodyInit;
  disableLoading?: boolean;
  reload?: boolean;
};

async function httpCall<T>(url: string, opt?: HttpCallOpt): Promise<T> {
  const res = await fetchWithOpts(url, opt);
  if (res.status === 204) {
    return undefined as T;
  }
  return res.json();
}

async function httpCallRaw(
  url: string,
  opt?: HttpCallOpt
): Promise<ArrayBuffer> {
  const res = await fetchWithOpts(url, opt);
  return res.arrayBuffer();
}

async function fetchWithOpts(
  url: string,
  opt?: HttpCallOpt
): Promise<Response> {
  await setLoadingCount();

  try {
    if (!opt?.disableLoading) {
      loadingCount.value++;
    }

    const fetchOptions: RequestInit = {
      method: opt?.method || "GET",
      headers: opt?.headers,
      body: opt?.body,
      cache: opt?.reload ? "reload" : undefined,
    };

    const r = await fetch(url, fetchOptions);
    if (!r.ok) {
      const errorText = await r.text();
      throw new Error(`API Error: ${r.status} ${r.statusText} - ${errorText}`);
    }

    return r;
  } finally {
    if (!opt?.disableLoading) {
      loadingCount.value--;
    }
  }
}

// Helper to avoid circular dependencies
async function setLoadingCount() {
  if (loadingCount) {
    return;
  }
  const { loadingCount: _loadingCount } = await import("@/store");
  loadingCount = _loadingCount;
}
