<template>
  <header class="header-bar drag-region">
    <!-- Left Section: Clear (Refresh), Save, Timer -->
    <div class="header-left no-drag">
      <!-- Clear (Refresh) Session Button -->
      <button 
        class="seamless-btn danger-hover" 
        @click="$emit('clear-session')" 
        title="Reset / Clear Session"
      >
        <svg viewBox="0 0 24 24" width="15" height="15" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <path d="M21.5 2v6h-6M21.34 15.57a10 10 0 1 1-.57-8.38l5.67-5.67"/>
        </svg>
      </button>

      <!-- Save Session Button -->
      <button 
        class="seamless-btn success-hover" 
        @click="$emit('save-session')" 
        title="Save Session"
      >
        <svg viewBox="0 0 24 24" width="15" height="15" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <path d="M19 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h11l5 5v11a2 2 0 0 1-2 2z"/>
          <polyline points="17 21 17 13 7 13 7 21"/>
          <polyline points="7 3 7 8 15 8"/>
        </svg>
      </button>

      <!-- Timer Display -->
      <div v-if="settings.showTimer" class="seamless-timer" title="Encounter Duration">
        <svg viewBox="0 0 24 24" width="13" height="13" fill="none" stroke="currentColor" stroke-width="2">
          <circle cx="12" cy="12" r="10"/><path d="M12 6v6l4 2"/>
        </svg>
        <span>{{ formattedTimer }}</span>
      </div>
    </div>

    <!-- Center Drag Handle Space -->
    <div class="header-center drag-region"></div>

    <!-- Right Section: Target Selector & 3-Dot Settings -->
    <div class="header-right no-drag">
      <!-- Target Selector Dropdown -->
      <select 
        class="target-select" 
        :value="settings.selectedTargetId" 
        @change="$emit('update-target', ($event.target as HTMLSelectElement).value)"
        title="Select combat target"
      >
        <option value="">All Targets</option>
        <option v-for="target in sortedTargets" :key="target.id" :value="target.id">
          {{ target.name || target.id }}{{ formatCompact(target.totalDamage) }}
        </option>
      </select>

      <!-- 3-Dot Settings Button -->
      <button 
        class="seamless-btn" 
        :class="{ active: showSettings }" 
        @click="$emit('toggle-settings')" 
        title="Settings"
      >
        <svg viewBox="0 0 24 24" width="16" height="16" fill="currentColor">
          <circle cx="12" cy="5" r="2"/>
          <circle cx="12" cy="12" r="2"/>
          <circle cx="12" cy="19" r="2"/>
        </svg>
      </button>
    </div>
  </header>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { OverlaySettings, TargetStats, PlayerStats } from "../types";

const props = defineProps<{
  isConnected: boolean;
  encounterDuration: number;
  targets: { [targetId: string]: TargetStats };
  settings: OverlaySettings;
  showSettings: boolean;
  selectedPlayer?: PlayerStats | null;
}>();

defineEmits<{
  (e: "save-session"): void;
  (e: "clear-session"): void;
  (e: "toggle-settings"): void;
  (e: "update-target", targetId: string): void;
  (e: "back"): void;
}>();

const formattedTimer = computed(() => {
  const dur = props.encounterDuration || 0;
  const mins = Math.floor(dur / 60);
  const secs = Math.floor(dur % 60);
  return `${mins.toString().padStart(2, "0")}:${secs.toString().padStart(2, "0")}`;
});

const formatCompact = (num?: number): string => {
  if (!num || isNaN(num)) return "";
  if (num >= 1000000) return ` (${(num / 1000000).toFixed(1)}M)`;
  if (num >= 1000) return ` (${(num / 1000).toFixed(0)}k)`;
  return ` (${num.toFixed(0)})`;
};

interface TargetItem extends TargetStats {
  id: string;
}

const sortedTargets = computed<TargetItem[]>(() => {
  if (!props.targets) return [];
  const list: TargetItem[] = Object.entries(props.targets).map(([id, t]) => ({
    id,
    ...t,
  }));

  return list.sort((a, b) => {
    const startA = a.startTime || 0;
    const startB = b.startTime || 0;
    if (startB !== startA) {
      return startB - startA; // 1. Most recent spawn time first
    }
    return (b.totalDamage || 0) - (a.totalDamage || 0); // 2. Highest damage first
  });
});
</script>

<style scoped>
.header-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 4px 8px;
  background: #14171f;
  border-bottom: 1px solid var(--border-color);
  min-height: 32px;
  gap: 8px;
}

.header-left {
  display: flex;
  align-items: center;
  gap: 6px;
}

.back-btn {
  display: flex;
  align-items: center;
  gap: 4px;
  color: var(--accent-primary);
  font-weight: 700;
  font-size: 0.85em;
  padding: 2px 6px;
}

.back-btn:hover {
  color: #ffffff;
  background: rgba(129, 138, 248, 0.25);
}

.header-center {
  flex: 1;
  height: 100%;
  min-width: 16px;
}

.header-right {
  display: flex;
  align-items: center;
  gap: 6px;
}

/* Seamless Header Button Styling */
.seamless-btn {
  background: transparent;
  border: none;
  color: #94a3b8;
  padding: 4px;
  border-radius: 4px;
  cursor: pointer;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  transition: all 0.15s ease;
}

.seamless-btn:hover {
  color: #ffffff;
  background: rgba(129, 138, 248, 0.15);
}

.seamless-btn.danger-hover:hover {
  color: var(--accent-rose);
  background: rgba(248, 113, 113, 0.15);
}

.seamless-btn.success-hover:hover {
  color: var(--accent-green);
  background: rgba(74, 222, 128, 0.15);
}

.seamless-btn.active {
  color: var(--accent-primary);
  background: rgba(129, 138, 248, 0.2);
}

/* Seamless Timer Display */
.seamless-timer {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 0.82em;
  font-weight: 600;
  color: var(--accent-amber);
  padding: 2px 4px;
}

/* Seamless Target Dropdown */
.target-select {
  background: rgba(23, 27, 36, 0.9);
  border: 1px solid var(--border-color);
  color: #f8fafc;
  padding: 2px 6px;
  border-radius: 4px;
  font-size: 0.8em;
  outline: none;
  cursor: pointer;
  max-width: 130px;
}

.target-select:hover {
  border-color: var(--accent-primary);
}

.target-select option {
  background: #14171f;
  color: #f8fafc;
}
</style>
