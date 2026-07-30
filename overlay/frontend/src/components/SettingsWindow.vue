<template>
  <div class="settings-window">
    <div class="header">
      <h2>⚙️ Midir Overlay Settings</h2>
    </div>

    <div class="body">
      <!-- Server URL -->
      <div class="setting-item">
        <label>Midir Server URL</label>
        <input 
          type="text" 
          v-model="settings.serverUrl" 
          @change="updateSetting('serverUrl', settings.serverUrl)"
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
          v-model.number="settings.bgOpacity" 
          @input="updateSetting('bgOpacity', settings.bgOpacity)"
        />
        <span class="hint">0% = Completely Transparent, 100% = Solid Dark Blue</span>
      </div>

      <!-- Custom Background Image -->
      <div class="setting-item">
        <label>Custom Background Image</label>
        <input 
          type="file" 
          ref="fileInputRef" 
          accept="image/*" 
          style="display: none" 
          @change="handleImageUpload" 
        />

        <div v-if="settings.bgImage" class="bg-image-preview-row">
          <div class="bg-thumbnail" :style="{ backgroundImage: `url(${settings.bgImage})` }"></div>
          <div class="bg-image-actions">
            <select 
              :value="settings.bgImageFit || 'cover'" 
              @change="updateSetting('bgImageFit', ($event.target as HTMLSelectElement).value)"
              class="fit-select"
            >
              <option value="cover">Fit: Cover</option>
              <option value="contain">Fit: Contain</option>
              <option value="tile">Fit: Tile</option>
            </select>
            <button class="btn-remove-img" @click="removeImage">Remove</button>
          </div>
        </div>
        <button v-else class="btn-choose-img" @click="triggerFileSelect">
          🖼️ Choose Image...
        </button>
        <span class="hint">Upload custom background image. Opacity slider above adjusts text-contrast tint.</span>
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
          v-model.number="settings.fontSize" 
          @input="updateSetting('fontSize', settings.fontSize)"
        />
      </div>

      <!-- Toggles -->
      <div class="setting-grid">
        <label class="toggle-item">
          <input 
            type="checkbox" 
            v-model="settings.showTimer" 
            @change="updateSetting('showTimer', settings.showTimer)"
          />
          <span>Show Encounter Timer</span>
        </label>

        <label class="toggle-item">
          <input 
            type="checkbox" 
            v-model="settings.isDragLocked" 
            @change="updateSetting('isDragLocked', settings.isDragLocked)"
          />
          <span>Lock Window Position</span>
        </label>

        <label class="toggle-item">
          <input 
            type="checkbox" 
            v-model="settings.isResizeLocked" 
            @change="updateSetting('isResizeLocked', settings.isResizeLocked)"
          />
          <span>Lock Window Size</span>
        </label>

        <label class="toggle-item">
          <input 
            type="checkbox" 
            v-model="settings.alwaysOnTop" 
            @change="updateSetting('alwaysOnTop', settings.alwaysOnTop)"
          />
          <span>Always on Top</span>
        </label>

        <label class="toggle-item">
          <input 
            type="checkbox" 
            v-model="settings.autoSwapEnabled" 
            @change="updateSetting('autoSwapEnabled', settings.autoSwapEnabled)"
          />
          <span>Auto Swap Target on First Hit</span>
        </label>
      </div>

      <!-- Auto Swap Target Race IDs -->
      <div v-if="settings.autoSwapEnabled" class="setting-item" style="margin-top: 10px;">
        <label>Target Race IDs (comma-separated)</label>
        <input 
          type="text" 
          :value="settings.autoSwapRaceIdsInput ?? (settings.autoSwapRaceIds?.join(', ') || '')" 
          @change="handleRaceIdsInputChange(($event.target as HTMLInputElement).value)"
          placeholder="e.g. 7615, 7616"
        />
        <span class="hint">Enter Race IDs separated by commas. Find Race IDs in the target dropdown (ID: XXXX).</span>
      </div>
    </div>

    <div class="footer">
      <button class="btn-close" @click="closeWindow">Close Settings</button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from "vue";
import { OverlaySettings } from "../types";

const channel = new BroadcastChannel("midir_overlay_settings_channel");

const fileInputRef = ref<HTMLInputElement | null>(null);

const settings = reactive<OverlaySettings>({
  serverUrl: "http://localhost:8030",
  bgOpacity: 0.75,
  fontSize: 13,
  showTimer: true,
  isDragLocked: false,
  isResizeLocked: false,
  alwaysOnTop: true,
  selectedTargetId: "",
  autoSwapEnabled: false,
  autoSwapRaceIds: [],
  autoSwapRaceIdsInput: "",
  bgImage: "",
  bgImageFit: "cover",
});

const triggerFileSelect = () => {
  fileInputRef.value?.click();
};

const handleImageUpload = (e: Event) => {
  const target = e.target as HTMLInputElement;
  if (!target.files || target.files.length === 0) return;
  const file = target.files[0];

  const reader = new FileReader();
  reader.onload = (event) => {
    const rawDataUrl = event.target?.result as string;
    if (!rawDataUrl) return;

    const img = new Image();
    img.onload = () => {
      const maxDim = 1280;
      let w = img.width;
      let h = img.height;

      if (w > maxDim || h > maxDim) {
        if (w > h) {
          h = Math.round((h * maxDim) / w);
          w = maxDim;
        } else {
          w = Math.round((w * maxDim) / h);
          h = maxDim;
        }
      }

      const canvas = document.createElement("canvas");
      canvas.width = w;
      canvas.height = h;
      const ctx = canvas.getContext("2d");
      if (ctx) {
        ctx.drawImage(img, 0, 0, w, h);
        const optimizedUrl = canvas.toDataURL("image/webp", 0.85);
        updateSetting("bgImage", optimizedUrl);
      } else {
        updateSetting("bgImage", rawDataUrl);
      }
    };
    img.onerror = () => {
      updateSetting("bgImage", rawDataUrl);
    };
    img.src = rawDataUrl;
  };
  reader.readAsDataURL(file);
};

const removeImage = () => {
  updateSetting("bgImage", "");
};

const handleRaceIdsInputChange = (inputStr: string) => {
  const parsed = inputStr
    .split(",")
    .map((s) => parseInt(s.trim(), 10))
    .filter((n) => !isNaN(n));
  updateSetting("autoSwapRaceIdsInput", inputStr);
  updateSetting("autoSwapRaceIds", parsed);
};

const loadSettings = () => {
  try {
    const raw = localStorage.getItem("midir_overlay_settings");
    if (raw) {
      Object.assign(settings, JSON.parse(raw));
    }
  } catch (e) {
    console.error("Failed to load settings in pop-out window", e);
  }
};

const updateSetting = (key: keyof OverlaySettings, value: any) => {
  (settings as any)[key] = value;
  const jsonStr = JSON.stringify(settings);
  localStorage.setItem("midir_overlay_settings", jsonStr);

  // Broadcast to main overlay window
  channel.postMessage({ type: "UPDATE_SETTING", key, value, settings: { ...settings } });

  // Call Go backend bindings if available
  if (window.go?.main?.App) {
    window.go.main.App.SaveOverlaySettings(jsonStr);
    if (key === "alwaysOnTop") {
      window.go.main.App.SetAlwaysOnTop(value);
    }
  }
};

const closeWindow = () => {
  window.close();
};

onMounted(() => {
  loadSettings();
});
</script>

<style scoped>
.settings-window {
  width: 100vw;
  height: 100vh;
  background: #0f172a;
  color: #f8fafc;
  display: flex;
  flex-direction: column;
  padding: 16px;
  overflow-y: auto;
  font-family: 'Inter', system-ui, -apple-system, sans-serif;
}

.header {
  border-bottom: 1px solid #334155;
  padding-bottom: 12px;
  margin-bottom: 16px;
}

.header h2 {
  font-size: 1.1em;
  font-weight: 700;
  color: #3b82f6;
}

.body {
  display: flex;
  flex-direction: column;
  gap: 14px;
  flex: 1;
}

.setting-item {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.setting-item label {
  font-size: 0.85em;
  font-weight: 600;
  color: #94a3b8;
}

.setting-label-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.setting-item input[type="text"] {
  background: #1e293b;
  border: 1px solid #334155;
  color: #ffffff;
  padding: 6px 10px;
  border-radius: 4px;
  font-size: 0.9em;
  outline: none;
}

.setting-item input[type="text"]:focus {
  border-color: #3b82f6;
}

.setting-item input[type="range"] {
  accent-color: #3b82f6;
  cursor: pointer;
}

.value {
  font-size: 0.88em;
  font-weight: 700;
  color: #3b82f6;
}

.hint {
  font-size: 0.75em;
  color: #64748b;
}

.setting-grid {
  display: flex;
  flex-direction: column;
  gap: 10px;
  margin-top: 6px;
  background: #1e293b;
  padding: 12px;
  border-radius: 6px;
  border: 1px solid #334155;
}

.toggle-item {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 0.88em;
  cursor: pointer;
  color: #e2e8f0;
}

.toggle-item input[type="checkbox"] {
  accent-color: #3b82f6;
  width: 16px;
  height: 16px;
  cursor: pointer;
}

.footer {
  margin-top: 16px;
  padding-top: 12px;
  border-top: 1px solid #334155;
  display: flex;
  justify-content: flex-end;
}

.btn-close {
  background: #3b82f6;
  border: none;
  color: #ffffff;
  padding: 6px 16px;
  border-radius: 4px;
  font-size: 0.88em;
  font-weight: 600;
  cursor: pointer;
  transition: background 0.15s ease;
}

.btn-close:hover {
  background: #2563eb;
}
</style>
