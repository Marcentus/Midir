<template>
  <v-card class="mt-4" variant="text">
    <v-card-title class="text-subtitle-1 font-weight-bold px-0">
      <div class="d-flex align-center">
        <v-icon start icon="mdi-format-color-fill"></v-icon>
        Name Color Generation
        <v-spacer></v-spacer>
        <v-btn
          color="error"
          variant="text"
          size="small"
          prepend-icon="mdi-refresh"
          @click="resetDefaults"
        >
          Reset Defaults
        </v-btn>
      </div>
    </v-card-title>
    <v-card-text class="px-0">
      <!-- Generator Seed Section -->
      <div class="d-flex align-center mb-4">
        <v-text-field
          v-model="colorSeed"
          label="Color Generation Seed"
          hide-details
          variant="outlined"
          density="compact"
          class="mr-2"
          hint="Changing this value will re-roll all random colors"
          persistent-hint
        ></v-text-field>
        <v-btn icon="mdi-dice-5" variant="text" @click="randomizeSeed" color="primary" title="Randomize Seed"></v-btn>
      </div>

      <v-divider class="mb-4"></v-divider>

      <v-row>
        <v-col cols="12" md="6">
          <div class="text-subtitle-2 mb-2">Saturation Range ({{ saturation[0] }}% - {{ saturation[1] }}%)</div>
          <v-range-slider
            v-model="saturation"
            :min="0"
            :max="100"
            step="1"
            strict
            thumb-label
            color="primary"
          ></v-range-slider>
          <div class="text-caption text-grey">
            Controls how intense the colors are. Higher values mean more vibrant colors.
          </div>
        </v-col>

        <v-col cols="12" md="6">
          <div class="text-subtitle-2 mb-2">Lightness Range ({{ lightness[0] }}% - {{ lightness[1] }}%)</div>
          <v-range-slider
            v-model="lightness"
            :min="0"
            :max="100"
            step="1"
            strict
            thumb-label
            color="secondary"
          ></v-range-slider>
          <div class="text-caption text-grey">
            Controls brightness. Lower is darker, higher is more pastel/white.
          </div>
        </v-col>
      </v-row>

      <v-divider class="my-4"></v-divider>

      <div class="text-subtitle-2 mb-2">Live Preview</div>
      <div class="d-flex flex-wrap gap-2 preview-container pa-3 rounded" style="background: rgba(var(--v-theme-surface), 0.8); border: 1px solid rgba(255,255,255,0.05);">
        <v-chip
          v-for="name in previewNames"
          :key="name"
          :style="{ color: getColor(name), borderColor: getColor(name) }"
          variant="outlined"
          class="font-weight-bold"
        >
          {{ name }}
        </v-chip>
      </div>
    </v-card-text>

    <!-- Reset Confirmation Dialog -->
    <v-dialog v-model="resetDialogVisible" max-width="400px" persistent>
      <v-card class="dialog-card">
        <v-card-title class="text-h5 d-flex align-center pt-4 px-6">
          <v-icon color="warning" class="mr-2">mdi-alert-decagram</v-icon>
          Reset Colors
        </v-card-title>
        <v-card-text class="pt-2 px-6 pb-4 text-grey-lighten-1">
          Are you sure you want to reset all name color settings to their default values?
        </v-card-text>
        <v-card-actions class="px-6 pb-4">
          <v-spacer></v-spacer>
          <v-btn variant="text" class="font-weight-bold" @click="resetDialogVisible = false">Cancel</v-btn>
          <v-btn color="primary" variant="flat" class="font-weight-bold ml-2" @click="confirmReset">Reset</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>
  </v-card>
</template>

<script lang="ts">
import { defineComponent, computed, ref } from "vue";
import { nameColorSaturation, nameColorLightness, nameColorSeed } from "@/store";
import { getMabiNameColor } from "@/util";

export default defineComponent({
  name: "ColorSettings",
  setup() {
    // Mabinogi-themed names for better context
    const previewNames = ref([
      "Nao", "Morrighan", "Cichol", "Ruairi", 
      "Mari", "Tarlach", "Duncan", "Kristell", 
      "Ferghus", "Pan", "Altam", "Avelin",
      "Pihne", "Caswyn", "Llywelyn", "Talvish"
    ]);

    const getColor = (name: string) => {
        // Force reactivity by accessing store values inside the render loop implicitly via getMabiNameColor
        return getMabiNameColor(name);
    }

    const randomizeSeed = () => {
        const chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789";
        let result = "";
        for (let i = 0; i < 6; i++) {
            result += chars.charAt(Math.floor(Math.random() * chars.length));
        }
        nameColorSeed.value = result;
    };

    const resetDialogVisible = ref(false);

    const resetDefaults = () => {
        resetDialogVisible.value = true;
    };

    const confirmReset = () => {
        resetDialogVisible.value = false;
        nameColorSaturation.value = [50, 90];
        nameColorLightness.value = [30, 50];
        nameColorSeed.value = "T0F89V";
    };

    return {
      saturation: nameColorSaturation,
      lightness: nameColorLightness,
      colorSeed: nameColorSeed,
      previewNames,
      getColor,
      randomizeSeed,
      resetDefaults,
      resetDialogVisible,
      confirmReset
    };
  },
});
</script>

<style scoped>
.gap-2 {
  gap: 8px;
}
</style>
