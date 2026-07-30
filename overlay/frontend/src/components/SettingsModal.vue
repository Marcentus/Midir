<template>
  <div class="settings-overlay no-drag" @click.self="$emit('close')">
    <div class="settings-card">
      <div class="card-header">
        <h3>Overlay Settings</h3>
        <button class="btn-close" @click="$emit('close')">✕</button>
      </div>

      <div class="card-body">
        <!-- Server URL -->
        <div class="setting-item">
          <label>Midir Server URL</label>
          <input 
            type="text" 
            :value="settings.serverUrl" 
            @change="updateSetting('serverUrl', ($event.target as HTMLInputElement).value)"
            placeholder="http://localhost:8030"
          />
        </div>

        <!-- Background Opacity Slider -->
        <div class="setting-item">
          <div class="setting-label-row">
            <label>Background Opacity</label>
            <span class="value">{{ Math.round(settings.bgOpacity * 100) }}%</span>
          </div>
          <input 
            type="range" 
            min="0" 
            max="1" 
            step="0.05" 
            :value="settings.bgOpacity" 
            @input="updateSetting('bgOpacity', parseFloat(($event.target as HTMLInputElement).value))"
          />
          <span class="hint">0% = Completely Transparent, 100% = Solid Dark Blue.</span>
        </div>

        <!-- Font Size Slider -->
        <div class="setting-item">
          <div class="setting-label-row">
            <label>Font Size</label>
            <span class="value">{{ settings.fontSize }}px</span>
          </div>
          <input 
            type="range" 
            min="11" 
            max="18" 
            step="1" 
            :value="settings.fontSize" 
            @input="updateSetting('fontSize', parseInt(($event.target as HTMLInputElement).value))"
          />
        </div>

        <!-- Toggles -->
        <div class="setting-grid">
          <label class="toggle-item">
            <input 
              type="checkbox" 
              :checked="settings.showTimer" 
              @change="updateSetting('showTimer', ($event.target as HTMLInputElement).checked)"
            />
            <span>Show Encounter Timer</span>
          </label>

          <label class="toggle-item">
            <input 
              type="checkbox" 
              :checked="settings.isDragLocked" 
              @change="updateSetting('isDragLocked', ($event.target as HTMLInputElement).checked)"
            />
            <span>Lock Window Position</span>
          </label>

          <label class="toggle-item">
            <input 
              type="checkbox" 
              :checked="settings.isResizeLocked" 
              @change="updateSetting('isResizeLocked', ($event.target as HTMLInputElement).checked)"
            />
            <span>Lock Window Size</span>
          </label>

          <label class="toggle-item">
            <input 
              type="checkbox" 
              :checked="settings.hideNames" 
              @change="updateSetting('hideNames', ($event.target as HTMLInputElement).checked)"
            />
            <span>Hide Player Names (Anonymize Names)</span>
          </label>

          <label class="toggle-item">
            <input 
              type="checkbox" 
              :checked="settings.alwaysOnTop" 
              @change="updateSetting('alwaysOnTop', ($event.target as HTMLInputElement).checked)"
            />
            <span>Always on Top</span>
          </label>

          <label class="toggle-item">
            <input 
              type="checkbox" 
              :checked="settings.autoSwapEnabled" 
              @change="updateSetting('autoSwapEnabled', ($event.target as HTMLInputElement).checked)"
            />
            <span>Auto Swap Target on First Hit</span>
          </label>
        </div>

        <!-- Auto Swap Section -->
        <div v-if="settings.autoSwapEnabled" class="auto-swap-container">
          <label class="section-label">Auto-Swap Targets</label>
          
          <!-- Chips List -->
          <div class="chips-wrapper">
            <div 
              v-for="(target, idx) in settings.autoSwapTargets || []" 
              :key="target.raceId + '-' + idx" 
              class="target-chip"
            >
              <span class="chip-label">🏷️ {{ target.name || ('Race ' + target.raceId) }} <small>(ID: {{ target.raceId }})</small></span>
              <button class="chip-remove" @click="removeTarget(idx)" title="Remove target">✕</button>
            </div>
            <span v-if="!settings.autoSwapTargets || settings.autoSwapTargets.length === 0" class="empty-hint">
              No auto-swap targets added yet.
            </span>
          </div>

          <!-- Add Controls -->
          <div class="add-controls-row">
            <!-- 1-Click Select from Session Targets -->
            <div class="quick-add-group">
              <select v-model="selectedActiveRaceId" class="session-target-select">
                <option value="">-- Add Active Target from Fight --</option>
                <option v-for="t in availableSessionTargets" :key="t.raceId" :value="t.raceId">
                  {{ t.name }} (ID: {{ t.raceId }})
                </option>
              </select>
              <button 
                class="btn-add-chip" 
                :disabled="!selectedActiveRaceId" 
                @click="addActiveTarget"
              >
                + Add
              </button>
            </div>

            <!-- Custom Race ID input toggle / inline form -->
            <div v-if="showCustomInput" class="custom-input-group">
              <input 
                type="number" 
                v-model.number="customRaceId" 
                placeholder="Race ID" 
                class="custom-id-input"
              />
              <input 
                type="text" 
                v-model="customRaceName" 
                placeholder="Label (optional)" 
                class="custom-name-input"
              />
              <button class="btn-add-chip" :disabled="!customRaceId" @click="addCustomTarget">Save</button>
              <button class="btn-cancel-custom" @click="showCustomInput = false">Cancel</button>
            </div>
            <button v-else class="btn-toggle-custom" @click="showCustomInput = true">
              + Custom Race ID
            </button>
          </div>
        </div>
      </div>

      <div class="card-footer">
        <button class="btn-done" @click="$emit('close')">Done</button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from "vue";
import { OverlaySettings, TargetStats, AutoSwapTarget } from "../types";

const props = defineProps<{
  settings: OverlaySettings;
  targets?: { [targetId: string]: TargetStats };
}>();

const emit = defineEmits<{
  (e: "update-setting", key: keyof OverlaySettings, value: any): void;
  (e: "close"): void;
}>();

const selectedActiveRaceId = ref<number | "">("");
const showCustomInput = ref(false);
const customRaceId = ref<number | "">("");
const customRaceName = ref("");

const updateSetting = (key: keyof OverlaySettings, value: any) => {
  emit("update-setting", key, value);
};

// Filter active session targets that have a valid raceId and aren't already added
const availableSessionTargets = computed(() => {
  if (!props.targets) return [];
  const existingIds = new Set((props.settings.autoSwapTargets || []).map((t) => t.raceId));
  
  const map = new Map<number, { raceId: number; name: string }>();
  for (const t of Object.values(props.targets)) {
    if (t.raceId && !existingIds.has(t.raceId)) {
      if (!map.has(t.raceId)) {
        map.set(t.raceId, { raceId: t.raceId, name: t.name || `Race ${t.raceId}` });
      }
    }
  }
  return Array.from(map.values());
});

const addActiveTarget = () => {
  if (!selectedActiveRaceId.value) return;
  const targetObj = availableSessionTargets.value.find((t) => t.raceId === selectedActiveRaceId.value);
  if (targetObj) {
    const updated = [...(props.settings.autoSwapTargets || []), targetObj];
    updateSetting("autoSwapTargets", updated);
    selectedActiveRaceId.value = "";
  }
};

const addCustomTarget = () => {
  if (!customRaceId.value || typeof customRaceId.value !== "number") return;
  const newTarget: AutoSwapTarget = {
    raceId: customRaceId.value,
    name: customRaceName.value.trim() || `Race ${customRaceId.value}`,
  };
  const updated = [...(props.settings.autoSwapTargets || []), newTarget];
  updateSetting("autoSwapTargets", updated);
  customRaceId.value = "";
  customRaceName.value = "";
  showCustomInput.value = false;
};

const removeTarget = (index: number) => {
  const updated = [...(props.settings.autoSwapTargets || [])];
  updated.splice(index, 1);
  updateSetting("autoSwapTargets", updated);
};
</script>

<style scoped>
.settings-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(20, 23, 31, 0.92);
  z-index: 1000;
  display: flex;
  flex-direction: column;
  padding: 8px;
}

.settings-card {
  width: 100%;
  height: 100%;
  background: #171b24;
  border: 1px solid var(--border-color);
  border-radius: 6px;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 12px;
  background: #14171f;
  border-bottom: 1px solid var(--border-color);
}

.card-header h3 {
  font-size: 0.9em;
  font-weight: 700;
  color: #f8fafc;
}

.btn-close {
  background: transparent;
  border: none;
  color: #94a3b8;
  cursor: pointer;
  font-size: 1em;
  padding: 2px 6px;
  border-radius: 4px;
}

.btn-close:hover {
  color: var(--accent-rose);
  background: rgba(248, 113, 113, 0.15);
}

.card-body {
  padding: 10px 12px;
  display: flex;
  flex-direction: column;
  gap: 10px;
  flex: 1;
  overflow-y: auto;
}

.setting-item {
  display: flex;
  flex-direction: column;
  gap: 3px;
}

.setting-item label {
  font-size: 0.8em;
  font-weight: 600;
  color: var(--text-muted);
}

.setting-label-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.setting-item input[type="text"] {
  background: #14171f;
  border: 1px solid var(--border-color);
  color: #ffffff;
  padding: 4px 8px;
  border-radius: 4px;
  font-size: 0.85em;
  outline: none;
}

.setting-item input[type="range"] {
  accent-color: var(--accent-primary);
  cursor: pointer;
}

.value {
  font-size: 0.82em;
  font-weight: 700;
  color: var(--accent-primary);
}

.hint {
  font-size: 0.7em;
  color: var(--text-dim);
}

.setting-grid {
  display: flex;
  flex-direction: column;
  gap: 6px;
  margin-top: 2px;
}

.toggle-item {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 0.82em;
  cursor: pointer;
  color: #e2e8f0;
}

.toggle-item input[type="checkbox"] {
  accent-color: var(--accent-primary);
  width: 14px;
  height: 14px;
  cursor: pointer;
}

.auto-swap-container {
  display: flex;
  flex-direction: column;
  gap: 6px;
  background: rgba(20, 23, 31, 0.6);
  border: 1px solid var(--border-color);
  padding: 8px;
  border-radius: 6px;
  margin-top: 2px;
}

.section-label {
  font-size: 0.8em;
  font-weight: 700;
  color: var(--accent-primary);
}

.chips-wrapper {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
  min-height: 28px;
  align-items: center;
  padding: 4px;
  background: #14171f;
  border: 1px dashed var(--border-color);
  border-radius: 4px;
  max-height: 90px;
  overflow-y: auto;
}

.target-chip {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  background: rgba(129, 138, 248, 0.18);
  border: 1px solid rgba(129, 138, 248, 0.4);
  color: #f8fafc;
  padding: 2px 8px;
  border-radius: 12px;
  font-size: 0.76em;
  font-weight: 600;
}

.chip-label small {
  color: #94a3b8;
  font-weight: 400;
}

.chip-remove {
  background: transparent;
  border: none;
  color: var(--accent-rose);
  cursor: pointer;
  font-size: 0.9em;
  line-height: 1;
  padding: 0 2px;
  border-radius: 50%;
}

.chip-remove:hover {
  background: rgba(248, 113, 113, 0.25);
}

.empty-hint {
  font-size: 0.75em;
  color: var(--text-dim);
  font-style: italic;
}

.add-controls-row {
  display: flex;
  flex-direction: column;
  gap: 6px;
  margin-top: 2px;
}

.quick-add-group {
  display: flex;
  gap: 6px;
}

.session-target-select {
  flex: 1;
  background: #14171f;
  border: 1px solid var(--border-color);
  color: #f8fafc;
  padding: 3px 6px;
  border-radius: 4px;
  font-size: 0.78em;
  outline: none;
}

.btn-add-chip {
  background: var(--accent-primary);
  color: #ffffff;
  border: none;
  padding: 3px 10px;
  border-radius: 4px;
  font-size: 0.78em;
  font-weight: 600;
  cursor: pointer;
}

.btn-add-chip:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.btn-toggle-custom {
  align-self: flex-start;
  background: transparent;
  border: 1px dashed var(--accent-primary);
  color: var(--accent-primary);
  padding: 2px 8px;
  border-radius: 4px;
  font-size: 0.75em;
  font-weight: 600;
  cursor: pointer;
}

.btn-toggle-custom:hover {
  background: rgba(129, 138, 248, 0.15);
}

.custom-input-group {
  display: flex;
  gap: 4px;
  align-items: center;
}

.custom-id-input {
  width: 85px;
  background: #14171f;
  border: 1px solid var(--border-color);
  color: #ffffff;
  padding: 3px 6px;
  border-radius: 4px;
  font-size: 0.78em;
  outline: none;
}

.custom-name-input {
  flex: 1;
  background: #14171f;
  border: 1px solid var(--border-color);
  color: #ffffff;
  padding: 3px 6px;
  border-radius: 4px;
  font-size: 0.78em;
  outline: none;
}

.btn-cancel-custom {
  background: transparent;
  border: none;
  color: var(--text-muted);
  font-size: 0.75em;
  cursor: pointer;
  padding: 2px 6px;
}

.btn-cancel-custom:hover {
  color: var(--accent-rose);
}

.card-footer {
  padding: 6px 12px;
  background: #14171f;
  border-top: 1px solid var(--border-color);
  display: flex;
  justify-content: flex-end;
}

.btn-done {
  background: var(--accent-primary);
  border: none;
  color: #ffffff;
  padding: 4px 14px;
  border-radius: 4px;
  font-size: 0.82em;
  font-weight: 600;
  cursor: pointer;
}

.btn-done:hover {
  background: #6366f1;
}
</style>
