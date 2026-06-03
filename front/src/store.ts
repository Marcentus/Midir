import { ref, computed, shallowReactive, reactive, watch } from "vue";
import { PlayerCacheInfo, Session } from "./types";
import { FightSummary, HPValidationEvent } from "@/protocols";
import { SocketClient } from "./socketClient";

export const socket = new SocketClient(
  location.protocol.replace("http", "ws") + "//" + location.host + "/ws"
);

export const loadingCount = ref(0);
export const isLoading = computed(() => loadingCount.value > 0);

const defaultRegion = "us";

export const region = ref(defaultRegion);
export const lang = ref(defaultRegion);
export const regionList = ref([defaultRegion]);

export const skillNameMap = ref<
  Record<number, { name: string; iconUrl?: string }>
>({});
export const condNameMap = ref<
  Record<number, { name: string; iconUrl?: string }>
>({});

export const raceNameMap = ref<Record<number, string>>({});
export const itemNameMap = ref<Record<number, string>>({});

export const appEvent = ref(new EventTarget());

export const playerNameCache = shallowReactive<Record<string, PlayerCacheInfo>>(
  {}
);

// --- SESSION STATE ---
export const sessions = ref<Session[]>([]);
export const activeSessionId = ref<string | "live">("live");
export const isNavDrawerOpen = ref(true);
export const activeTool = ref<"dps" | "settings" | "hpdebug">("dps");
export const selectedTargetId = ref<string>("");
export const hpValidationEvents = ref<HPValidationEvent[]>([]);


export const fightSummary = reactive<FightSummary>({
  encounterDuration: 0,
  totalDamage: 0,
  players: {},
  targets: {},
  damageTaken: {},
  graphData: {},
});

// --- CONDITION PREFERENCES ---
const COND_PREFS_KEY = "midir_cond_prefs";

// Using Sets for O(1) lookup in loops
export const favoriteConditions = reactive(new Set<number>());
export const hiddenConditions = reactive(new Set<number>());

// --- HIDDEN PLAYERS ---
export const hiddenPlayers = reactive(new Set<string>());
export const globalHideMode = ref(false);

watch(activeSessionId, () => {
  if (!globalHideMode.value) {
    hiddenPlayers.clear();
  }
});

export function toggleHiddenPlayer(id: string) {
  if (hiddenPlayers.has(id)) {
    hiddenPlayers.delete(id);
  } else {
    hiddenPlayers.add(id);
  }
}

export function setAllHiddenPlayers(ids: string[], hide: boolean) {
    if (hide) {
        ids.forEach(id => hiddenPlayers.add(id));
    } else {
        ids.forEach(id => hiddenPlayers.delete(id));
    }
}

// --- CUSTOM CONDITION ORDER ---
export const customConditionOrder = ref<number[]>([]);

try {
  const stored = localStorage.getItem(COND_PREFS_KEY);
  if (stored) {
    const parsed = JSON.parse(stored);
    if (Array.isArray(parsed.fav)) {
      parsed.fav.forEach((id: number) => favoriteConditions.add(id));
    }
    if (Array.isArray(parsed.hide)) {
      parsed.hide.forEach((id: number) => hiddenConditions.add(id));
    }
    if (Array.isArray(parsed.order)) {
      customConditionOrder.value = parsed.order;
    }
  }
} catch (e) {
  console.error("Failed to load condition preferences", e);
}

export function saveConditionPrefs() {
    try {
        const data = {
            fav: Array.from(favoriteConditions),
            hide: Array.from(hiddenConditions),
            order: customConditionOrder.value,
        };
        localStorage.setItem(COND_PREFS_KEY, JSON.stringify(data));
    } catch (e) {
        console.error("Failed to save condition preferences", e);
    }
}

export function updateConditionOrder(newOrder: number[]) {
    customConditionOrder.value = newOrder;
    saveConditionPrefs();
}

export function toggleConditionPref(id: number, type: "fav" | "hide") {
  if (type === "fav") {
    if (favoriteConditions.has(id)) {
      favoriteConditions.delete(id);
    } else {
      favoriteConditions.add(id);
      hiddenConditions.delete(id); // Mutually exclusive: cannot be both hidden and fav
    }
  } else {
    if (hiddenConditions.has(id)) {
      hiddenConditions.delete(id);
    } else {
      hiddenConditions.add(id);
      favoriteConditions.delete(id); // Mutually exclusive
    }
  }
  saveConditionPrefs();
  saveConditionPrefs();
}

// --- LIVE ENTITY CONDITION PREFERENCES ---
const LIVE_COND_PREFS_KEY = "midir_live_cond_prefs";

export const liveFavoriteConditions = reactive(new Set<number>());
export const liveHiddenConditions = reactive(new Set<number>());

try {
  const stored = localStorage.getItem(LIVE_COND_PREFS_KEY);
  if (stored) {
    const parsed = JSON.parse(stored);
    if (Array.isArray(parsed.fav)) {
      parsed.fav.forEach((id: number) => liveFavoriteConditions.add(id));
    }
    if (Array.isArray(parsed.hide)) {
      parsed.hide.forEach((id: number) => liveHiddenConditions.add(id));
    }
  }
} catch (e) {
  console.error("Failed to load live condition preferences", e);
}

export function saveLiveConditionPrefs() {
    try {
        const data = {
            fav: Array.from(liveFavoriteConditions),
            hide: Array.from(liveHiddenConditions),
        };
        localStorage.setItem(LIVE_COND_PREFS_KEY, JSON.stringify(data));
    } catch (e) {
        console.error("Failed to save live condition preferences", e);
    }
}

export function toggleLiveConditionPref(id: number, type: "fav" | "hide") {
  if (type === "fav") {
    if (liveFavoriteConditions.has(id)) {
      liveFavoriteConditions.delete(id);
    } else {
      liveFavoriteConditions.add(id);
      liveHiddenConditions.delete(id); 
    }
  } else {
    if (liveHiddenConditions.has(id)) {
      liveHiddenConditions.delete(id);
    } else {
      liveHiddenConditions.add(id);
      liveFavoriteConditions.delete(id); 
    }
  }
  saveLiveConditionPrefs();
}

// --- SETTINGS ---
export const showClassColorsForVisiblePlayers = ref(false);
export const dpsMeterFillMode = ref<"column" | "full">("full");
export const nameColorSaturation = ref<[number, number]>([50, 90]); // [Min, Max]
export const nameColorLightness = ref<[number, number]>([30, 50]); // [Min, Max]
export const nameColorSeed = ref<string>("T0F89V"); // Default seed

export interface MetricPref {
  key: string;
  visible: boolean;
}
const DEFAULT_METRICS: MetricPref[] = [
  { key: 'totalDamage', visible: true },
  { key: 'count', visible: true },
  { key: 'hpm', visible: true },
  { key: 'critRate', visible: true },
  { key: 'avgDamage', visible: true },
  { key: 'maxDamage', visible: true },
  { key: 'dps', visible: false },
  { key: 'avgNonCrit', visible: false },
  { key: 'maxDamageNonCrit', visible: false },
  { key: 'avgCrit', visible: false },
  { key: 'maxDamageCrit', visible: false }
];

export const listSkillMetrics = ref<MetricPref[]>(JSON.parse(JSON.stringify(DEFAULT_METRICS)));
export const cardSkillMetrics = ref<MetricPref[]>(JSON.parse(JSON.stringify(DEFAULT_METRICS)));
const SETTINGS_KEY = "midir_settings";

try {
  const stored = localStorage.getItem(SETTINGS_KEY);
  if (stored) {
    const parsed = JSON.parse(stored);
    if (typeof parsed.showClassColorsForVisiblePlayers === 'boolean') {
      showClassColorsForVisiblePlayers.value = parsed.showClassColorsForVisiblePlayers;
    }
    if (typeof parsed.dpsMeterFillMode === 'string') {
      dpsMeterFillMode.value = parsed.dpsMeterFillMode as "column" | "full";
    }
    if (parsed.nameColorSeed) {
        nameColorSeed.value = parsed.nameColorSeed;
    }
    // Load Saturation Range
    if (Array.isArray(parsed.nameColorSaturation) && parsed.nameColorSaturation.length === 2) {
      nameColorSaturation.value = parsed.nameColorSaturation;
    }
    // Load Lightness Range
    if (Array.isArray(parsed.nameColorLightness) && parsed.nameColorLightness.length === 2) {
      nameColorLightness.value = parsed.nameColorLightness;
    }
    // Load Column Selection
    if (Array.isArray(parsed.listSkillMetrics) && parsed.listSkillMetrics.length > 0) {
      listSkillMetrics.value = parsed.listSkillMetrics;
    }
    if (Array.isArray(parsed.cardSkillMetrics) && parsed.cardSkillMetrics.length > 0) {
      cardSkillMetrics.value = parsed.cardSkillMetrics;
    }
  }
} catch (e) {
  console.error("Failed to load settings", e);
}

export function saveSettings() {
  try {
    const data = {
      showClassColorsForVisiblePlayers: showClassColorsForVisiblePlayers.value,
      dpsMeterFillMode: dpsMeterFillMode.value,
      nameColorSaturation: nameColorSaturation.value,
      nameColorLightness: nameColorLightness.value,
      nameColorSeed: nameColorSeed.value,
      listSkillMetrics: listSkillMetrics.value,
      cardSkillMetrics: cardSkillMetrics.value,
    };
    localStorage.setItem(SETTINGS_KEY, JSON.stringify(data));
  } catch (e) {
    console.error("Failed to save settings", e);
  }
}

watch([showClassColorsForVisiblePlayers, dpsMeterFillMode, nameColorSaturation, nameColorLightness, nameColorSeed, listSkillMetrics, cardSkillMetrics], () => {
  saveSettings();
}, { deep: true });

