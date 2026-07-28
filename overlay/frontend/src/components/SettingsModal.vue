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

        </div>
      </div>

      <div class="card-footer">
        <button class="btn-done" @click="$emit('close')">Done</button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { OverlaySettings } from "../types";

const props = defineProps<{
  settings: OverlaySettings;
}>();

const emit = defineEmits<{
  (e: "update-setting", key: keyof OverlaySettings, value: any): void;
  (e: "close"): void;
}>();

const updateSetting = (key: keyof OverlaySettings, value: any) => {
  emit("update-setting", key, value);
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

.toggle-item-wrapper {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.sub-hint {
  font-size: 0.72em;
  color: var(--accent-amber);
  padding-left: 20px;
}

.toggle-item input[type="checkbox"] {
  accent-color: var(--accent-primary);
  width: 14px;
  height: 14px;
  cursor: pointer;
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
