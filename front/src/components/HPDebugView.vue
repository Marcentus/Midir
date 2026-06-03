<!-- front/src/components/HPDebugView.vue -->
<template>
  <v-container fluid class="hp-debug-container">
    <v-card class="mx-auto modern-card" max-width="1200">
      <v-toolbar color="surface" flat class="modern-toolbar">
        <v-toolbar-title class="text-h5 font-weight-black d-flex align-center">
          <v-icon color="primary" class="mr-2">mdi-bug</v-icon>
          Damage to HP Debugger
        </v-toolbar-title>
        <v-spacer></v-spacer>
        <v-btn
          color="error"
          variant="tonal"
          prepend-icon="mdi-trash-can"
          size="small"
          class="font-weight-bold mr-2"
          @click="clearLogs"
        >
          Clear Logs
        </v-btn>
      </v-toolbar>

      <!-- FILTERS PANEL -->
      <v-card-text class="border-bottom">
        <v-row align="center" dense>
          <!-- Search field -->
          <v-col cols="12" md="4">
            <v-text-field
              v-model="searchQuery"
              prepend-inner-icon="mdi-magnify"
              label="Filter by Target Name..."
              variant="outlined"
              density="compact"
              hide-details
              clearable
            ></v-text-field>
          </v-col>

          <!-- Toggle filters -->
          <v-col cols="12" md="8" class="d-flex flex-wrap justify-md-end ga-3">
            <v-checkbox
              v-slot:label
              v-model="filterOptions.showSuccess"
              label="Show Matches"
              color="success"
              hide-details
              density="compact"
              class="ma-0"
            ></v-checkbox>
            <v-checkbox
              v-slot:label
              v-model="filterOptions.showMismatch"
              label="Show Mismatches"
              color="error"
              hide-details
              density="compact"
              class="ma-0"
            ></v-checkbox>
          </v-col>
        </v-row>
      </v-card-text>

      <!-- RESULTS PANEL -->
      <v-card-text class="pa-0">
        <v-window>
          <v-window-item>
            <v-list v-if="filteredEvents.length > 0" class="pa-0 modern-list">
              <v-expansion-panels variant="accordion">
                <v-expansion-panel
                  v-for="(event, idx) in filteredEvents"
                  :key="event.entityId + '-' + event.timestamp + '-' + idx"
                  class="event-panel-item"
                  :class="event.status"
                >
                  <v-expansion-panel-title class="py-3 px-4">
                    <template v-slot:default="{ expanded }">
                      <v-row align="center" no-gutters class="w-100 pr-4">
                        <!-- Icon Status -->
                        <v-col cols="auto" class="mr-3 d-flex align-center">
                          <v-icon
                            v-if="event.status === 'success'"
                            color="success"
                            size="28"
                          >
                            mdi-check-circle
                          </v-icon>
                          <v-icon
                            v-else
                            color="error"
                            size="28"
                          >
                            mdi-close-circle
                          </v-icon>
                        </v-col>

                        <!-- Timestamp & Target -->
                        <v-col cols="12" sm="3" class="text-left">
                          <div class="text-caption text-grey">
                            {{ formatTime(event.timestamp) }}
                          </div>
                          <div class="font-weight-bold text-white text-truncate" style="max-width: 200px;">
                            {{ event.entityName }}
                          </div>
                          <div class="text-caption text-grey-darken-1 font-mono text-truncate">
                            ID: {{ event.entityId }}
                          </div>
                        </v-col>

                        <!-- HP Change Summary -->
                        <v-col cols="6" sm="4" class="text-left">
                          <div class="text-caption text-grey">HP Update</div>
                          <div class="text-body-2 font-weight-medium">
                            <span class="text-white">{{ event.lastHp.toFixed(1) }}</span>
                            <v-icon size="14" class="mx-1 text-grey">mdi-arrow-right</v-icon>
                            <span class="text-white">{{ event.newHp.toFixed(1) }}</span>
                            <span class="text-grey text-caption"> / {{ event.maxHp.toFixed(1) }}</span>
                          </div>
                          <div class="text-caption text-grey-darken-1">
                            Delta: {{ event.actualDelta.toFixed(1) }} HP
                          </div>
                        </v-col>

                        <!-- Verification Stats -->
                        <v-col cols="6" sm="4" class="text-left">
                          <div class="text-caption text-grey">Verification</div>
                          <div class="text-body-2">
                            Expected: <span class="font-weight-bold text-white">{{ event.pendingDamage.toFixed(1) }}</span> damage
                          </div>
                          <div class="text-caption font-weight-bold" :class="getStatusColor(event.status)">
                            {{ getStatusText(event) }}
                          </div>
                        </v-col>
                      </v-row>
                    </template>
                  </v-expansion-panel-title>

                  <v-expansion-panel-text class="bg-card-expanded pa-0 border-top">
                    <!-- Expanded Details: Damage Hits -->
                    <div class="pa-4">
                      <div class="d-flex align-center justify-space-between mb-3">
                        <span class="text-subtitle-2 font-weight-bold text-white">
                          <v-icon size="18" class="mr-1">mdi-format-list-bulleted</v-icon>
                          Damage Hits Contributing to Expected HP ({{ event.damageHits.length }} total)
                        </span>
                        <span class="text-caption text-grey-darken-1 font-mono">
                          Pending Accumulation: {{ event.pendingDamage.toFixed(1) }} Damage
                        </span>
                      </div>

                      <v-table v-if="event.damageHits.length > 0" density="compact" class="hits-table">
                        <thead>
                          <tr>
                            <th class="text-left text-grey text-caption py-2">Time</th>
                            <th class="text-left text-grey text-caption py-2">Attacker</th>
                            <th class="text-left text-grey text-caption py-2">Skill / Ability</th>
                            <th class="text-right text-grey text-caption py-2">Damage</th>
                          </tr>
                        </thead>
                        <tbody>
                          <tr v-for="(hit, hIdx) in event.damageHits" :key="hIdx">
                            <td class="font-mono text-caption text-grey py-1">
                              {{ formatTime(new Date(hit.timestamp).getTime()) }}
                            </td>
                            <td class="font-weight-medium text-white py-1">
                              {{ hit.attackerName }}
                              <span class="text-caption text-grey-darken-1 font-mono ml-1">({{ hit.attackerId }})</span>
                            </td>
                            <td class="py-1">
                              <div class="d-flex align-center">
                                <v-avatar size="20" class="mr-2 rounded" color="grey-darken-3">
                                  <img
                                    v-if="getSkillIcon(hit.skillId)"
                                    :src="getSkillIcon(hit.skillId)"
                                    alt="icon"
                                    class="w-100 h-100 object-fit-cover"
                                  />
                                  <v-icon v-else size="12" color="grey">mdi-sword</v-icon>
                                </v-avatar>
                                <span class="text-white text-body-2">{{ getSkillName(hit.skillId) }}</span>
                                <span class="text-caption text-grey-darken-1 font-mono ml-1">[#{{ hit.skillId }}]</span>
                              </div>
                            </td>
                            <td class="text-right font-weight-black text-amber-accent-2 font-mono py-1">
                              {{ hit.damage.toLocaleString(undefined, { minimumFractionDigits: 1, maximumFractionDigits: 1 }) }}
                            </td>
                          </tr>
                        </tbody>
                      </v-table>

                      <div v-else class="text-center py-6 text-grey text-caption bg-empty rounded">
                        <v-icon size="24" class="mb-1 d-block mx-auto text-grey-darken-2">mdi-alert-circle-outline</v-icon>
                        No damage hits recorded in validation window (natural regen, environment hazard, or missed packet).
                      </div>
                    </div>
                  </v-expansion-panel-text>
                </v-expansion-panel>
              </v-expansion-panels>
            </v-list>

            <!-- EMPTY STATE -->
            <div v-else class="text-center py-12 text-grey-darken-1">
              <v-icon size="64" class="mb-4 text-grey-darken-3">mdi-clipboard-text-search-outline</v-icon>
              <div class="text-h6 text-white font-weight-bold mb-1">No HP verification logs found</div>
              <div class="text-body-2 max-width-500 mx-auto px-4">
                HP changes will appear here automatically when Midir aggregates attacks on active targets and receives an HP update packet from the game.
              </div>
            </div>
          </v-window-item>
        </v-window>
      </v-card-text>
    </v-card>
  </v-container>
</template>

<script lang="ts">
import { defineComponent, ref, computed, inject } from "vue";
import { hpValidationEvents } from "@/store";
import { HPValidationEvent } from "@/protocols";

export default defineComponent({
  name: "HPDebugView",
  setup() {
    const skillNameMap = inject("skillNameMap") as any;
    const searchQuery = ref("");

    const filterOptions = ref({
      showSuccess: true,
      showMismatch: true,
    });

    const clearLogs = () => {
      hpValidationEvents.value = [];
    };

    const filteredEvents = computed(() => {
      return hpValidationEvents.value.filter((event) => {
        // Status filter
        if (event.status === "success" && !filterOptions.value.showSuccess) return false;
        if (event.status === "mismatch" && !filterOptions.value.showMismatch) return false;

        // Search text filter
        if (searchQuery.value) {
          const query = searchQuery.value.toLowerCase();
          const targetName = event.entityName.toLowerCase();
          const targetId = event.entityId;
          if (!targetName.includes(query) && !targetId.includes(query)) {
            return false;
          }
        }
        return true;
      }).slice().reverse(); // Show latest logs on top
    });

    // Helper functions
    const formatTime = (ts: number) => {
      const date = new Date(ts);
      const hrs = String(date.getHours()).padStart(2, "0");
      const mins = String(date.getMinutes()).padStart(2, "0");
      const secs = String(date.getSeconds()).padStart(2, "0");
      const ms = String(date.getMilliseconds()).padStart(3, "0");
      return `${hrs}:${mins}:${secs}.${ms}`;
    };

    const getSkillName = (id: number): string => {
      return skillNameMap.value?.[id]?.name || `Skill ${id}`;
    };

    const getSkillIcon = (id: number): string => {
      return skillNameMap.value?.[id]?.iconUrl || "";
    };

    const getDeltaColor = (event: HPValidationEvent): string => {
      return "text-grey-darken-1";
    };

    const getStatusColor = (status: string): string => {
      if (status === "success") return "text-success";
      return "text-error font-weight-black";
    };

    const getStatusText = (event: HPValidationEvent): string => {
      if (event.status === "success") return "MATCH";
      const diff = event.actualDelta - event.pendingDamage;
      const prefix = diff > 0 ? "+" : "";
      return `MISMATCH (${prefix}${diff.toFixed(1)} HP)`;
    };

    return {
      searchQuery,
      filterOptions,
      clearLogs,
      filteredEvents,
      formatTime,
      getSkillName,
      getSkillIcon,
      getDeltaColor,
      getStatusColor,
      getStatusText,
    };
  },
});
</script>

<style scoped>
.hp-debug-container {
  padding: 24px;
}

.modern-card {
  background: rgba(22, 25, 34, 0.7) !important;
  backdrop-filter: blur(15px);
  border: 1px solid rgba(255, 255, 255, 0.05);
  box-shadow: 0 10px 30px rgba(0, 0, 0, 0.4) !important;
  border-radius: 16px !important;
  overflow: hidden;
}

.modern-toolbar {
  border-bottom: 1px solid rgba(255, 255, 255, 0.05);
  background: rgba(22, 25, 34, 0.9) !important;
}

.border-bottom {
  border-bottom: 1px solid rgba(255, 255, 255, 0.05);
}

.border-top {
  border-top: 1px solid rgba(255, 255, 255, 0.05);
}

.font-mono {
  font-family: "Fira Code", monospace, Consolas, Courier;
}

.modern-list {
  background: transparent !important;
}

/* Expansion Panel Visual Tuning */
.event-panel-item {
  background: transparent !important;
  border-bottom: 1px solid rgba(255, 255, 255, 0.03) !important;
  border-radius: 0 !important;
  transition: all 0.2s ease;
}

.event-panel-item::before {
  display: none !important; /* Hide default expansion panel shadow borders */
}

/* Status highlights on hover */
.event-panel-item.success:hover {
  background: rgba(76, 175, 80, 0.02) !important;
}

.event-panel-item.mismatch:hover {
  background: rgba(244, 67, 54, 0.03) !important;
}

/* Custom background for expanded content */
.bg-card-expanded {
  background: rgba(10, 11, 15, 0.5) !important;
}

.hits-table {
  background: transparent !important;
  border: 1px solid rgba(255, 255, 255, 0.04);
  border-radius: 8px;
}

.hits-table :deep(th) {
  border-bottom: 2px solid rgba(255, 255, 255, 0.06) !important;
  font-weight: 700 !important;
}

.hits-table :deep(td) {
  border-bottom: 1px solid rgba(255, 255, 255, 0.03) !important;
}

.hits-table :deep(tr:last-child td) {
  border-bottom: none !important;
}

.bg-empty, .bg-empty-state {
  background: rgba(255, 255, 255, 0.01);
  border: 1px dashed rgba(255, 255, 255, 0.06);
}

.max-width-500 {
  max-width: 500px;
}

.ga-3 {
  gap: 12px;
}

/* Text sizes & colors */
.text-grey {
  color: #a0aec0 !important;
}
.text-grey-darken-1 {
  color: #718096 !important;
}
.text-grey-darken-2 {
  color: #4a5568 !important;
}
.text-grey-darken-3 {
  color: #2d3748 !important;
}
.text-success {
  color: #4caf50 !important;
}
.text-error {
  color: #f44336 !important;
}
.text-info {
  color: #00bcd4 !important;
}
</style>
