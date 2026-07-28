<template>
  <div 
    id="app" 
    :class="{ unlocked: !settings.isDragLocked }"
    :style="appStyle"
  >
    <!-- Header Bar -->
    <HeaderBar
      :is-connected="isConnected"
      :encounter-duration="summary?.encounterDuration || 0"
      :targets="summary?.targets || {}"
      :settings="settings"
      :show-settings="showSettings"
      :selected-player="selectedPlayer"
      @save-session="handleSaveSession"
      @clear-session="handleClearSession"
      @toggle-settings="toggleSettings"
      @update-target="handleTargetUpdate"
      @back="clearSelectedPlayer"
    />

    <!-- Player Skill Breakdown View (when a player row is clicked) -->
    <PlayerBreakdown
      v-if="selectedPlayer"
      :player="selectedPlayer"
      :target-id="settings.selectedTargetId"
      :hide-names="settings.hideNames"
      :server-url="settings.serverUrl"
      :encounter-duration="summary?.encounterDuration || 0"
      @back="clearSelectedPlayer"
    />

    <!-- Main DPS List View -->
    <DpsList
      v-else
      :players="summary?.players || {}"
      :target-id="settings.selectedTargetId"
      :total-damage="summary?.totalDamage || 0"
      :hide-names="settings.hideNames"
      :server-url="settings.serverUrl"
      @select-player="handleSelectPlayer"
    />

    <!-- Settings Modal -->
    <SettingsModal
      v-if="showSettings"
      :settings="settings"
      @update-setting="handleUpdateSetting"
      @close="toggleSettings"
    />

    <!-- Resize Grip (when unlocked & not resize locked) -->
    <div 
      v-if="!settings.isDragLocked && !settings.isResizeLocked" 
      class="resize-handle no-drag" 
      @mousedown="startCornerResize"
      title="Drag corner to resize overlay"
    >
      <svg viewBox="0 0 12 12" width="12" height="12" fill="currentColor">
        <path d="M10 2l2 2v8h-8l-2-2v-8h8zm2 4l-6 6h6v-6z" opacity="0.7"/>
      </svg>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted, onUnmounted } from "vue";
import HeaderBar from "./components/HeaderBar.vue";
import DpsList from "./components/DpsList.vue";
import PlayerBreakdown from "./components/PlayerBreakdown.vue";
import SettingsModal from "./components/SettingsModal.vue";
import { OverlaySocketClient } from "./socketClient";
import { FightSummary, OverlaySettings } from "./types";

// Wails bindings reference
declare global {
  interface Window {
    go?: {
      main?: {
        App?: {
          GetWindowSize(): Promise<number[]>;
          SetWindowSize(w: number, h: number): Promise<void>;
          SetAlwaysOnTop(alwaysOnTop: boolean): Promise<void>;
          SetClickThrough(enable: boolean): Promise<void>;
          SaveOverlaySettings(jsonStr: string): Promise<boolean>;
          LoadOverlaySettings(): Promise<string>;
        };
      };
    };
  }
}

const isConnected = ref(false);
const showSettings = ref(false);
const summary = ref<FightSummary | null>(null);
const selectedPlayerId = ref<string | null>(null);

const selectedPlayer = computed(() => {
  if (!selectedPlayerId.value || !summary.value?.players) return null;
  return summary.value.players[selectedPlayerId.value] || null;
});

const handleSelectPlayer = (playerId: string) => {
  selectedPlayerId.value = playerId;
};

const clearSelectedPlayer = () => {
  selectedPlayerId.value = null;
};

const settings = reactive<OverlaySettings>({
  serverUrl: "http://localhost:8030",
  bgOpacity: 0.75,
  fontSize: 13,
  showTimer: true,
  hideNames: false,
  isDragLocked: false,
  isResizeLocked: false,
  alwaysOnTop: true,
  selectedTargetId: "",
});

let socketClient: OverlaySocketClient | null = null;
let savedDimensions: { w: number; h: number } | null = null;

const appStyle = computed(() => {
  const isZeroOpacity = settings.bgOpacity <= 0.02;
  return {
    "--bg-opacity": settings.bgOpacity,
    backgroundColor: isZeroOpacity ? "transparent" : `rgba(20, 23, 31, ${settings.bgOpacity})`,
    border: "none",
    boxShadow: "none",
    "--font-scale": settings.fontSize / 13,
  };
});

// Load persisted settings
const loadSettings = async () => {
  try {
    let raw = "";
    if (window.go?.main?.App?.LoadOverlaySettings) {
      raw = await window.go.main.App.LoadOverlaySettings();
    }
    if (!raw) {
      raw = localStorage.getItem("midir_overlay_settings") || "";
    }
    if (raw) {
      const parsed = JSON.parse(raw);
      Object.assign(settings, parsed);
    }
  } catch (e) {
    console.error("Failed to load overlay settings", e);
  }
};

const saveSettings = async () => {
  try {
    const jsonStr = JSON.stringify(settings);
    localStorage.setItem("midir_overlay_settings", jsonStr);
    if (window.go?.main?.App?.SaveOverlaySettings) {
      await window.go.main.App.SaveOverlaySettings(jsonStr);
    }
  } catch (e) {
    console.error("Failed to save settings", e);
  }
};

const toggleSettings = async () => {
  showSettings.value = !showSettings.value;

  if (window.go?.main?.App) {
    if (showSettings.value) {
      try {
        const size = await window.go.main.App.GetWindowSize();
        if (size && size.length === 2) {
          const [w, h] = size;
          if (w < 400 || h < 340) {
            savedDimensions = { w, h };
            await window.go.main.App.SetWindowSize(Math.max(w, 420), Math.max(h, 380));
          }
        }
      } catch (e) {
        console.error("Failed to expand window for settings", e);
      }
    } else if (savedDimensions) {
      try {
        await window.go.main.App.SetWindowSize(savedDimensions.w, savedDimensions.h);
      } catch (e) {
        console.error("Failed to restore window size", e);
      }
      savedDimensions = null;
    }
  }
};

const handleUpdateSetting = (key: keyof OverlaySettings, value: any) => {
  (settings as any)[key] = value;
  saveSettings();

  if (key === "serverUrl") {
    socketClient?.updateServerUrl(value);
  } else if (key === "alwaysOnTop") {
    window.go?.main?.App?.SetAlwaysOnTop(value);
  }
};

const handleTargetUpdate = (targetId: string) => {
  settings.selectedTargetId = targetId;
  saveSettings();
};

const startCornerResize = (e: MouseEvent) => {
  e.preventDefault();
  e.stopPropagation();

  const startX = e.screenX;
  const startY = e.screenY;

  if (window.go?.main?.App?.GetWindowSize) {
    window.go.main.App.GetWindowSize().then((size) => {
      if (!size || size.length < 2) return;
      const startW = size[0];
      const startH = size[1];

      const onMouseMove = (moveEvt: MouseEvent) => {
        const deltaX = moveEvt.screenX - startX;
        const deltaY = moveEvt.screenY - startY;
        const newW = Math.max(280, startW + deltaX);
        const newH = Math.max(160, startH + deltaY);
        window.go?.main?.App?.SetWindowSize(newW, newH);
      };

      const onMouseUp = () => {
        window.removeEventListener("mousemove", onMouseMove);
        window.removeEventListener("mouseup", onMouseUp);
      };

      window.addEventListener("mousemove", onMouseMove);
      window.addEventListener("mouseup", onMouseUp);
    });
  }
};

const handleSaveSession = async () => {
  if (socketClient) {
    const ok = await socketClient.saveSession();
    if (ok) {
      console.log("Session saved successfully");
    }
  }
};

const handleClearSession = async () => {
  if (socketClient) {
    const ok = await socketClient.clearSession();
    if (ok) {
      summary.value = null;
      selectedPlayerId.value = null;
    }
  }
};

onMounted(async () => {
  await loadSettings();

  socketClient = new OverlaySocketClient(settings.serverUrl);
  socketClient.onConnectChange = (connected) => {
    isConnected.value = connected;
  };
  socketClient.onSummary = (data) => {
    summary.value = data;
  };

  socketClient.connect();

  // Apply initial native Wails options
  if (window.go?.main?.App) {
    window.go.main.App.SetAlwaysOnTop(settings.alwaysOnTop ?? true);
  }
});

onUnmounted(() => {
  socketClient?.close();
});
</script>

<style scoped>
.resize-handle {
  position: absolute;
  right: 2px;
  bottom: 2px;
  cursor: se-resize;
  color: var(--text-dim);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 50;
}
</style>
