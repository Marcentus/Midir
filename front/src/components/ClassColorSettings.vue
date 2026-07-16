<template>
  <v-card class="mt-4" variant="text" rounded="0">
    <v-card-title class="text-subtitle-1 font-weight-bold px-0">
      <div class="d-flex align-center">
        <v-icon start icon="mdi-palette-swatch"></v-icon>
        Arcana Colors
        <v-spacer></v-spacer>
        <v-btn
          color="error"
          variant="text"
          size="small"
          prepend-icon="mdi-refresh"
          @click="resetAllDialogVisible = true"
          :disabled="!hasCustomColors"
        >
          Reset All Colors
        </v-btn>
      </div>
    </v-card-title>
    
    <v-card-text class="px-0">
      <v-card variant="outlined" style="border-color: rgba(255,255,255,0.08);" rounded="0">
        <v-card-text class="pa-2" style="max-height: 600px; overflow-y: auto;">
          <div v-if="loading" class="text-center py-8">
            <v-progress-circular indeterminate color="primary"></v-progress-circular>
            <div class="text-caption mt-2">Loading arcana configurations...</div>
          </div>
          
          <div v-else class="d-flex flex-column gap-1">
            <v-card
              v-for="item in arcanas"
              :key="item.name"
              class="px-3 py-1 class-item-card d-flex align-center"
              :style="{ backgroundColor: getClassColorValue(item) }"
              variant="flat"
              rounded="0"
            >
              <!-- Left side: Icon & Name -->
              <div class="d-flex align-center flex-grow-1 min-width-0">
                <img
                  :src="item.icon"
                  width="28"
                  height="28"
                  class="mr-3 bg-black-opacity flex-shrink-0"
                  alt=""
                />

                <span class="text-h6 text-truncate mr-2" style="color: white;">
                  {{ item.name }}
                </span>
              </div>

              <!-- Right side: Controls -->
              <div class="d-flex align-center flex-shrink-0">
                    <!-- Reusable Custom Color Picker Component -->
                    <CustomColorPicker
                      :model-value="getClassColorValue(item)"
                      @update:model-value="updateClassColor(item.name, $event)"
                      class="mr-2"
                    />
                
                <!-- Individual Reset Button -->
                <v-btn
                  icon="mdi-undo"
                  variant="text"
                  size="small"
                  color="white"
                  class="reset-btn"
                  :disabled="!isCustomized(item.name)"
                  @click="resetClassColor(item.name)"
                  title="Reset to default"
                ></v-btn>
              </div>
            </v-card>
          </div>
        </v-card-text>
      </v-card>
    </v-card-text>

    <!-- Reset All Confirmation Dialog -->
    <v-dialog v-model="resetAllDialogVisible" max-width="400px" persistent>
      <v-card class="dialog-card" rounded="0">
        <v-card-title class="text-h5 d-flex align-center pt-4 px-6">
          <v-icon color="warning" class="mr-2">mdi-alert-decagram</v-icon>
          Reset Arcana Colors
        </v-card-title>
        <v-card-text class="pt-2 px-6 pb-4 text-grey-lighten-1">
          Are you sure you want to reset all arcana colors back to their default values?
        </v-card-text>
        <v-card-actions class="px-6 pb-4">
          <v-spacer></v-spacer>
          <v-btn variant="text" class="font-weight-bold" @click="resetAllDialogVisible = false" rounded="0">Cancel</v-btn>
          <v-btn color="primary" variant="flat" class="font-weight-bold ml-2" @click="confirmResetAll" rounded="0">Reset All</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>
  </v-card>
</template>

<script lang="ts">
import { defineComponent, computed, ref, onMounted } from "vue";
import { customClassColors } from "@/store";
import { getClassColor } from "@/util";
import CustomColorPicker from "./CustomColorPicker.vue";

interface ClassItem {
  name: string;
  icon: string;
  defaultColor: string;
  isArcana: boolean;
}

export default defineComponent({
  name: "ClassColorSettings",
  components: {
    CustomColorPicker
  },
  setup() {
    const loading = ref(true);
    const resetAllDialogVisible = ref(false);
    
    const arcanas = ref<ClassItem[]>([]);

    const hasCustomColors = computed(() => {
      return Object.keys(customClassColors.value).length > 0;
    });

    // Load arcanas from the talents.json file
    const loadClassData = async () => {
      loading.value = true;
      try {
        const res = await fetch("/api/data/talents.json");
        if (res.ok) {
          const data = await res.json();
          const loadedArcanas: ClassItem[] = [];

          // Process Arcanas only
          if (data.arcanas) {
            for (const key in data.arcanas) {
              const arcana = data.arcanas[key];
              loadedArcanas.push({
                name: arcana.arcana_name || key,
                icon: arcana.icon || "/images/talents/gui_icon_name_multi_class_0_0.png",
                defaultColor: arcana.color || "#808080",
                isArcana: true
              });
            }
          }

          // Sort alphabetically
          arcanas.value = loadedArcanas.sort((a, b) => a.name.localeCompare(b.name));
        }
      } catch (err) {
        console.error("Failed to load class configuration data:", err);
      } finally {
        loading.value = false;
      }
    };

    const getClassColorValue = (item: ClassItem) => {
      return getClassColor(item.name, item.defaultColor);
    };

    const isCustomized = (className: string) => {
      return !!customClassColors.value[className];
    };

    const updateClassColor = (className: string, hexColor: string) => {
      customClassColors.value = {
        ...customClassColors.value,
        [className]: hexColor
      };
    };

    const resetClassColor = (className: string) => {
      const newColors = { ...customClassColors.value };
      delete newColors[className];
      customClassColors.value = newColors;
    };

    const confirmResetAll = () => {
      customClassColors.value = {};
      resetAllDialogVisible.value = false;
    };

    onMounted(loadClassData);

    return {
      loading,
      resetAllDialogVisible,
      arcanas,
      hasCustomColors,
      getClassColorValue,
      isCustomized,
      updateClassColor,
      resetClassColor,
      confirmResetAll
    };
  }
});
</script>

<style scoped>
.gap-2 {
  gap: 8px;
}
.bg-black-opacity {
  background-color: rgba(0, 0, 0, 0.25) !important;
}

/* Card layout & Hover transitions */
.class-item-card {
  transition: transform 0.2s cubic-bezier(0.4, 0, 0.2, 1), box-shadow 0.2s ease, filter 0.2s ease;
  border: 1px solid rgba(255, 255, 255, 0.1) !important;
  box-shadow: 0 4px 6px rgba(0, 0, 0, 0.15) !important;
  cursor: default;
  position: relative;
  overflow: hidden;
  border-radius: 0px !important;
}

.class-item-card::after {
  content: '';
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: linear-gradient(to bottom, rgba(255,255,255,0.05) 0%, rgba(0,0,0,0.15) 100%);
  pointer-events: none;
}

.class-item-card:hover {
  transform: translateY(-2px);
  box-shadow: 0 8px 16px rgba(0, 0, 0, 0.3) !important;
  filter: brightness(1.08);
}

/* Reset button styling */
.reset-btn {
  z-index: 3;
  opacity: 0.8;
}
.reset-btn:hover {
  opacity: 1;
}
.reset-btn:disabled {
  opacity: 0.3 !important;
}
</style>
