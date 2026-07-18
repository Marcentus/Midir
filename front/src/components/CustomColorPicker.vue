<template>
  <div class="color-picker-button-wrapper">
    <v-menu :close-on-content-click="false" location="bottom end">
      <template v-slot:activator="{ props }">
        <v-btn
          v-bind="props"
          icon="mdi-palette"
          variant="flat"
          size="small"
          color="rgba(0,0,0,0.4)"
          class="color-picker-btn text-white"
          rounded="0"
        ></v-btn>
      </template>
      
      <v-card class="pa-2 bg-surface border-rgba(255,255,255,0.1)" rounded="0" width="250" elevation="8">
        <!-- Reverted Big Color Picker Canvas at the top -->
        <v-color-picker
          :model-value="modelValue"
          @update:model-value="emitColorUpdate($event)"
          hide-inputs
          hide-sliders
          mode="hex"
          rounded="0"
          flat
          width="234"
        ></v-color-picker>
        
        <!-- Split layout: circular splotch on the left, sliders stacked on the right -->
        <div class="d-flex align-center px-1 mt-2">
          <!-- Left side: Circular color splotch -->
          <div 
            class="circular-color-splotch mr-3 flex-shrink-0"
            :style="{ backgroundColor: modelValue }"
          ></div>
          
          <!-- Right side: Two sliders stacked vertically -->
          <div class="d-flex flex-column justify-space-between flex-grow-1" style="min-width: 0; gap: 8px;">
            <!-- Hue Slider (Color picker slider bar) -->
            <div class="hue-slider-wrapper">
              <v-color-picker
                :model-value="modelValue"
                @update:model-value="emitColorUpdate($event)"
                hide-canvas
                hide-inputs
                mode="hex"
                rounded="0"
                flat
                width="186"
              ></v-color-picker>
            </div>
            
            <!-- Lightness / Darkness Slider -->
            <v-slider
              :model-value="getLightnessValue(modelValue)"
              @update:model-value="updateLightness($event)"
              min="0"
              max="100"
              step="1"
              hide-details
              density="compact"
              class="darkness-slider"
              color="grey-darken-1"
            ></v-slider>
          </div>
        </div>
        
        <v-divider class="my-2 border-rgba(255,255,255,0.1)"></v-divider>
        
        <div class="px-1 d-flex flex-column gap-2 text-caption">
          <!-- Custom HEX Input -->
          <div class="d-flex align-center">
            <span class="text-grey mr-2 font-weight-bold" style="min-width: 32px; font-size: 0.7rem;">HEX</span>
            <div class="native-input-wrapper d-flex align-center flex-grow-1">
              <span class="prefix-hash">#</span>
              <input
                type="text"
                :value="getHexValue(modelValue)"
                @input="updateFromHex(($event.target as HTMLInputElement).value)"
                class="native-color-input flex-grow-1"
                maxlength="6"
              />
            </div>
          </div>
          
          <!-- Custom RGB Input -->
          <div class="d-flex align-center">
            <span class="text-grey mr-2 font-weight-bold" style="min-width: 32px; font-size: 0.7rem;">RGB</span>
            <div class="native-input-wrapper d-flex align-center flex-grow-1">
              <input
                type="text"
                :value="getRgbValue(modelValue)"
                @input="updateFromRgb(($event.target as HTMLInputElement).value)"
                class="native-color-input flex-grow-1"
              />
            </div>
          </div>
        </div>
      </v-card>
    </v-menu>
  </div>
</template>

<script lang="ts">
import { defineComponent } from 'vue';

export default defineComponent({
  name: 'CustomColorPicker',
  props: {
    modelValue: {
      type: String,
      required: true
    }
  },
  emits: ['update:modelValue'],
  setup(props, { emit }) {
    const emitColorUpdate = (val: any) => {
      let hexColor = "";
      if (typeof val === "string") {
        hexColor = val;
      } else if (val && typeof val === "object" && val.hex) {
        hexColor = val.hex;
      } else if (val && typeof val === "object" && typeof val.toString === "function") {
        hexColor = val.toString();
      }

      if (hexColor && hexColor !== props.modelValue) {
        emit('update:modelValue', hexColor);
      }
    };

    // HSL <-> HEX Conversions
    const hexToHsl = (hex: string) => {
      let r = 0, g = 0, b = 0;
      let cleanHex = hex.startsWith("#") ? hex.slice(1) : hex;
      if (cleanHex.length === 3) {
        cleanHex = cleanHex[0] + cleanHex[0] + cleanHex[1] + cleanHex[1] + cleanHex[2] + cleanHex[2];
      }
      if (cleanHex.length >= 6) {
        r = parseInt(cleanHex.slice(0, 2), 16) / 255;
        g = parseInt(cleanHex.slice(2, 4), 16) / 255;
        b = parseInt(cleanHex.slice(4, 6), 16) / 255;
      }
      const max = Math.max(r, g, b);
      const min = Math.min(r, g, b);
      let h = 0, s = 0, l = (max + min) / 2;

      if (max !== min) {
        const d = max - min;
        s = l > 0.5 ? d / (2 - max - min) : d / (max + min);
        switch (max) {
          case r: h = (g - b) / d + (g < b ? 6 : 0); break;
          case g: h = (b - r) / d + 2; break;
          case b: h = (r - g) / d + 4; break;
        }
        h /= 6;
      }

      return {
        h: Math.round(h * 360),
        s: Math.round(s * 100),
        l: Math.round(l * 100)
      };
    };

    const hslToHex = (h: number, s: number, l: number) => {
      h /= 360;
      s /= 100;
      l /= 100;
      let r = l, g = l, b = l;

      if (s !== 0) {
        const hue2rgb = (p: number, q: number, t: number) => {
          if (t < 0) t += 1;
          if (t > 1) t -= 1;
          if (t < 1/6) return p + (q - p) * 6 * t;
          if (t < 1/2) return q;
          if (t < 2/3) return p + (q - p) * (2/3 - t) * 6;
          return p;
        };
        const q = l < 0.5 ? l * (1 + s) : l + s - l * s;
        const p = 2 * l - q;
        r = hue2rgb(p, q, h + 1/3);
        g = hue2rgb(p, q, h);
        b = hue2rgb(p, q, h - 1/3);
      }

      const toHex = (n: number) => {
        const val = Math.round(n * 255);
        const hex = val.toString(16);
        return hex.length === 1 ? "0" + hex : hex;
      };

      return "#" + toHex(r) + toHex(g) + toHex(b);
    };

    const getHexValue = (colorStr: string) => {
      if (!colorStr) return "";
      return colorStr.startsWith("#") ? colorStr.slice(1) : colorStr;
    };

    const getRgbValue = (colorStr: string) => {
      if (!colorStr) return "";
      let hex = colorStr.startsWith("#") ? colorStr.slice(1) : colorStr;
      if (hex.length === 3) {
        hex = hex[0] + hex[0] + hex[1] + hex[1] + hex[2] + hex[2];
      }
      if (hex.length >= 6) {
        const r = parseInt(hex.slice(0, 2), 16);
        const g = parseInt(hex.slice(2, 4), 16);
        const b = parseInt(hex.slice(4, 6), 16);
        return isNaN(r) || isNaN(g) || isNaN(b) ? "" : `${r}, ${g}, ${b}`;
      }
      return "";
    };

    const getLightnessValue = (colorStr: string) => {
      return hexToHsl(colorStr).l;
    };

    const updateLightness = (newL: number) => {
      const hsl = hexToHsl(props.modelValue);
      const newHex = hslToHex(hsl.h, hsl.s, newL);
      emitColorUpdate(newHex);
    };

    const updateFromHex = (val: string) => {
      if (!val) return;
      let cleanVal = val.trim();
      if (!cleanVal.startsWith("#")) {
        cleanVal = "#" + cleanVal;
      }
      const isValidHex = /^#[0-9A-Fa-f]{3}$|^#[0-9A-Fa-f]{6}$/.test(cleanVal);
      if (isValidHex) {
        emitColorUpdate(cleanVal);
      }
    };

    const updateFromRgb = (val: string) => {
      if (!val) return;
      const matches = val.match(/\d+/g);
      if (matches && matches.length >= 3) {
        const r = parseInt(matches[0], 10);
        const g = parseInt(matches[1], 10);
        const b = parseInt(matches[2], 10);
        if (r >= 0 && r <= 255 && g >= 0 && g <= 255 && b >= 0 && b <= 255) {
          const toHex = (n: number) => {
            const hex = n.toString(16);
            return hex.length === 1 ? "0" + hex : hex;
          };
          const hexColor = "#" + toHex(r) + toHex(g) + toHex(b);
          emitColorUpdate(hexColor);
        }
      }
    };

    return {
      emitColorUpdate,
      getHexValue,
      getRgbValue,
      getLightnessValue,
      updateLightness,
      updateFromHex,
      updateFromRgb
    };
  }
});
</script>

<style scoped>
.color-picker-button-wrapper {
  position: relative;
  display: inline-block;
  z-index: 3;
}
.color-picker-btn {
  border: 1px solid rgba(255, 255, 255, 0.2) !important;
  transition: transform 0.1s ease, border-color 0.2s ease, background-color 0.2s ease;
  border-radius: 0px !important;
}
.color-picker-btn:hover {
  transform: scale(1.1);
  border-color: rgba(255, 255, 255, 0.8) !important;
}

/* Custom inputs styling */
.border-rgba\(255\,255\,255\,0\.1\) {
  border-color: rgba(255, 255, 255, 0.1) !important;
}
.native-input-wrapper {
  background-color: transparent;
  border: 1px solid rgba(255, 255, 255, 0.15);
  height: 28px;
  padding: 0 8px;
  font-size: 0.8rem;
  color: white;
  display: flex;
  align-items: center;
}
.prefix-hash {
  color: rgba(255, 255, 255, 0.7);
  margin-right: 4px;
  display: inline-block;
  line-height: 28px;
  height: 28px;
}
.native-color-input {
  background: transparent;
  border: none;
  outline: none;
  color: white;
  width: 100%;
  height: 28px;
  line-height: 28px;
  font-size: 0.8rem;
  padding: 0;
}
.native-input-wrapper:focus-within {
  border-color: var(--v-theme-primary, #2196F3);
}

/* Circular color splotch display */
.circular-color-splotch {
  width: 36px;
  height: 36px;
  border-radius: 50% !important;
  border: 1px solid rgba(255, 255, 255, 0.2);
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.4);
  align-self: center;
}

/* Hue slider clean alignment & hiding default preview dot */
.hue-slider-wrapper :deep(.v-color-picker) {
  background: transparent !important;
}
.hue-slider-wrapper :deep(.v-color-picker-preview) {
  padding: 0 !important;
  display: flex !important;
  align-items: center !important;
}
.hue-slider-wrapper :deep(.v-color-picker-preview__dot) {
  display: none !important;
}
.hue-slider-wrapper :deep(.v-color-picker-preview__sliders) {
  padding: 0 !important;
  flex-grow: 1 !important;
  max-width: 100% !important;
}
.hue-slider-wrapper :deep(.v-color-picker-preview__sliders .v-slider-track__background) {
  border-radius: 0px !important;
  height: 8px !important;
}
.hue-slider-wrapper :deep(.v-color-picker-preview__sliders .v-slider-thumb__surface) {
  border-radius: 0px !important;
}
.hue-slider-wrapper :deep(.v-color-picker-preview__sliders .v-slider-thumb) {
  --v-slider-thumb-size: 10px !important;
}

/* Darkness Slider Gradient Styling */
.darkness-slider :deep(.v-slider-track__background) {
  background: linear-gradient(to right, #000000, #ffffff) !important;
  opacity: 1 !important;
  height: 8px !important;
  border-radius: 0px !important;
}
.darkness-slider :deep(.v-slider-track__fill) {
  background: transparent !important;
}
.darkness-slider :deep(.v-slider-thumb__surface) {
  border-radius: 0px !important;
}
.darkness-slider :deep(.v-slider-thumb) {
  --v-slider-thumb-size: 10px !important;
}
</style>
