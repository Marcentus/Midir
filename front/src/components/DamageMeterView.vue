<template>
  <v-layout class="h-100">
    <session-panel />

    <v-main>
      <div class="dashboard-wrapper">
        <div class="dashboard-header mb-4">
          <div class="header-main-row">
            <!-- Left: Target Selector -->
            <div class="header-section target-section">
              <div class="header-label d-flex align-center">
                <knot-indicator legend class="mr-1" />
                COMBAT TARGET
              </div>
              <v-select
                v-model="selectedTargetId"
                :items="targetList"
                item-value="id"
                variant="solo-filled"
                flat
                density="compact"
                hide-details
                class="target-select-refined"
                placeholder="All Targets"
                :menu-props="{ minWidth: '520px', maxWidth: '520px', maxHeight: menuMaxHeight, location: 'bottom', offset: 4 }"
                @update:menu="handleMenuUpdate"
              >
                <!-- Selection Display (when closed) -->
                <template v-slot:selection="{ item }">
                  <div class="d-flex align-center w-100 justify-space-between text-body-2 font-weight-medium">
                    <span class="d-flex align-center">
                      <span class="mr-2 d-flex align-center">
                        <knot-indicator
                          v-if="item.raw.id"
                          :left="getKnotColors(item.raw).left"
                          :right="getKnotColors(item.raw).right"
                          :border="getKnotColors(item.raw).border"
                          :tooltip="getKnotColors(item.raw).tooltip"
                        />
                      </span>
                      <span>{{ item.raw.rawName || item.raw.name }}</span>
                      <v-tooltip
                        v-if="item.raw.id && item.raw.seenAppear === false"
                        location="top"
                        text="Spawn not captured by parser."
                      >
                        <template v-slot:activator="{ props }">
                          <v-icon
                            v-bind="props"
                            icon="mdi-alert-circle-outline"
                            color="warning"
                            size="small"
                            class="ml-1"
                          ></v-icon>
                        </template>
                      </v-tooltip>
                    </span>
                    <span class="text-caption text-grey ml-2" v-if="item.raw.id">({{ formatNumber(item.raw.damage) }})</span>
                  </div>
                </template>

                <!-- Item Display (when open in dropdown list) -->
                <template v-slot:item="{ props, item }">
                  <v-list-item v-bind="props" :class="getTargetClass(item.raw.raceId)">
                    <template v-slot:title>
                      <div class="target-item-grid py-1">
                        <!-- Column 1: Status Icons -->
                        <span class="target-col-icons d-flex align-center justify-center">
                          <knot-indicator
                            v-if="item.raw.id"
                            :left="getKnotColors(item.raw).left"
                            :right="getKnotColors(item.raw).right"
                            :border="getKnotColors(item.raw).border"
                            :tooltip="getKnotColors(item.raw).tooltip"
                          />
                        </span>

                        <!-- Column 2: Name -->
                        <span class="target-col-name font-weight-bold d-flex align-center">
                          <span>{{ item.raw.rawName || item.raw.name }}</span>
                          <v-tooltip
                            v-if="item.raw.id && item.raw.seenAppear === false"
                            location="top"
                            text="Spawn not captured by parser."
                          >
                            <template v-slot:activator="{ props }">
                              <v-icon
                                v-bind="props"
                                icon="mdi-alert-circle-outline"
                                color="warning"
                                size="small"
                                class="ml-1 flex-shrink-0"
                              ></v-icon>
                            </template>
                          </v-tooltip>
                        </span>
                        
                        <!-- Column 3: Duration (Relative to fight start) -->
                        <span class="target-col-duration text-grey text-caption">
                          <template v-if="item.raw.id && item.raw.startTime !== undefined && item.raw.endTime !== undefined">
                            {{ formatRelativeTime(item.raw.startTime) }} - {{ formatRelativeTime(item.raw.endTime) }} ({{ formatDuration(item.raw.endTime - item.raw.startTime) }})
                          </template>
                        </span>

                        <!-- Column 4: Damage Dealt -->
                        <span class="target-col-damage font-weight-bold text-amber text-right">{{ formatNumber(item.raw.damage) }}</span>
                      </div>
                    </template>
                  </v-list-item>
                </template>
              </v-select>
            </div>

            <!-- Right: Vital Stats -->
            <div class="header-section stats-section">
              <div class="stat-item">
                <div class="header-label text-right">PARTY DPS</div>
                <div class="stat-value amber-text">{{ formattedPartyDPS }}</div>
              </div>
              <div class="stat-divider mx-8"></div>
              <div class="stat-item">
                <div class="header-label text-right">DURATION</div>
                <div class="stat-value">{{ formattedEncounterDuration }}</div>
              </div>
            </div>
          </div>

          <!-- Conditions Row (Metadata) -->
          <div class="conditions-bar mt-4" v-if="selectedTargetId || hasPartyBuffs">
            <div class="d-flex w-100 justify-space-between align-center">
              <!-- Left: Target Conditions -->
              <div class="d-flex flex-column" v-if="selectedTargetId">
                <div class="header-label" style="opacity: 0.9;">TARGET CONDITIONS</div>
                <target-condition-view
                  :conditions="selectedTargetConditions"
                  :attackerNameMap="attackerNameMap"
                  class="ml-0"
                />
              </div>
              <div v-else></div> <!-- Spacer -->

              <!-- Right: Party Buffs -->
              <div class="d-flex h-100 align-center" v-if="hasPartyBuffs && (partyBuffs.length > 0 || partyBuffDetails.length > 0)">
                <div class="d-flex flex-column align-end mr-2">
                  <div class="header-label mb-1" style="opacity: 0.9;">PARTY BUFFS</div>
                  <div class="d-flex flex-column ga-1 align-end">
                    <div v-for="buff in partyBuffs" :key="buff.id" class="d-flex align-center ga-2">
                      <v-tooltip location="top">
                        <template v-slot:activator="{ props }">
                          <img 
                            v-bind="props" 
                            :src="buff.iconUrl" 
                            width="28" 
                            height="28" 
                            class="rounded-sm" 
                            style="border: 1px solid rgba(255,255,255,0.1);"
                            @error="($event.target as HTMLImageElement).style.display = 'none'"
                          />
                        </template>
                        <span>{{ buff.name }}</span>
                      </v-tooltip>
                      <div class="d-flex flex-column align-end">
                        <span v-for="val in buff.displayValue" :key="val" class="text-caption font-weight-bold text-info" style="font-size: 0.75rem !important; line-height: 1.1;">
                          {{ val }}
                        </span>
                      </div>
                    </div>
                  </div>
                </div>
                <!-- Arrow Button to Pop Details -->
                <v-btn
                  v-if="partyBuffDetails.length > 0"
                  variant="text"
                  color="white"
                  class="buff-detail-btn"
                  :ripple="false"
                  @click="showBuffDetails = true"
                >
                  <v-icon size="18">mdi-chevron-right</v-icon>
                  <v-tooltip activator="parent" location="top">View Detailed Buff Stats</v-tooltip>
                </v-btn>
              </div>
            </div>
          </div>

          <!-- Party Buff Details Dialog -->
          <v-dialog v-model="showBuffDetails" width="auto" min-width="500">
            <v-card class="modern-card">
              <v-card-title class="d-flex justify-space-between align-center pa-4" style="background: rgba(var(--v-theme-surface), 1); border-bottom: 1px solid rgba(255,255,255,0.1);">
                <div class="d-flex align-center ga-2">
                  <v-icon color="info">mdi-chart-line</v-icon>
                  <span class="text-subtitle-1 font-weight-bold">Detailed Party Buff Statistics</span>
                </div>
                <v-btn icon variant="text" size="small" @click="showBuffDetails = false">
                  <v-icon>mdi-close</v-icon>
                </v-btn>
              </v-card-title>
              
              <v-card-text class="pa-4">
                <div class="d-flex flex-column ga-4">
                  <div v-for="buff in partyBuffDetails" :key="buff.id" class="buff-detail-group pa-3 rounded bg-surface-variant-darken-1">
                    <div class="d-flex align-center ga-3 mb-3 border-b-sm border-white-05 pb-2">
                      <img :src="buff.iconUrl" width="32" height="32" class="rounded shadow" />
                      <span class="text-subtitle-2 font-weight-bold">{{ buff.name }}</span>
                    </div>
                    
                    <div class="d-flex flex-column ga-3">
                      <div v-for="metric in buff.metrics" :key="metric.label" class="metric-row">
                        <div class="text-caption text-grey-lighten-1 mb-1 font-weight-bold">{{ metric.label.toUpperCase() }}</div>
                        <v-row no-gutters class="text-center">
                          <v-col cols="4">
                            <div class="text-caption text-grey">Highest Seen</div>
                            <div class="text-body-2 font-weight-bold text-white">{{ metric.highest.toFixed(2) }}%</div>
                          </v-col>
                          <v-col cols="4" class="border-s-sm border-white-05">
                            <div class="text-caption text-grey">High Uptime Value</div>
                            <div class="text-body-2 font-weight-bold text-info">{{ metric.highestUptime.toFixed(2) }}%</div>
                          </v-col>
                          <v-col cols="4" class="border-s-sm border-white-05">
                            <div class="text-caption text-grey">
                              Weighted Avg
                              <span class="ml-1">
                                <v-icon size="12" color="grey-lighten-1">mdi-information-outline</v-icon>
                                <v-tooltip activator="parent" location="top" max-width="300">
                                  Average strength while active, weighted by the damage contribution of players who primarily used this buff.
                                </v-tooltip>
                              </span>
                            </div>
                            <div class="text-body-2 font-weight-bold text-amber">{{ metric.weightedAvg.toFixed(2) }}%</div>
                          </v-col>
                        </v-row>
                      </div>
                    </div>
                  </div>
                </div>
              </v-card-text>
              
              <v-card-actions class="pa-3 justify-end bg-surface-variant-darken-2">
                <v-btn variant="text" size="small" @click="showBuffDetails = false">Close</v-btn>
              </v-card-actions>
            </v-card>
          </v-dialog>
        </div>

        <div class="main-dashboard-content">
          <v-tabs v-model="tab" grow density="compact" class="modern-tabs mb-6">
            <v-tab value="damageDealt">Damage Dealt</v-tab>
            <v-tab value="damageTaken">Damage Taken</v-tab>
            <v-tab value="graph">Graph</v-tab>
          </v-tabs>

          <v-window v-model="tab">
            <v-window-item value="damageDealt">
              <apply-damage-by-skill :attackerNameMap="attackerNameMap" />
            </v-window-item>
            <v-window-item value="damageTaken">
              <damage-taken-by-source />
            </v-window-item>
            <v-window-item value="graph">
              <damage-graph />
            </v-window-item>
          </v-window>
        </div>

        <!-- Entity Viewer Floating Button and Panel -->
        <div 
          class="entity-viewer-container"
          :style="{ left: isNavDrawerOpen ? '344px' : '24px' }"
        >
          <!-- Floating Button when Collapsed -->
          <v-btn
            v-if="!isEntityViewerOpen"
            color="indigo-darken-3"
            class="entity-viewer-toggle-btn"
            prepend-icon="mdi-shield-half-full"
            @click="isEntityViewerOpen = true"
          >
            Entities
            <v-badge
              color="red"
              :content="totalEntitiesCount"
              inline
              class="ml-2"
              v-if="totalEntitiesCount > 0"
            ></v-badge>
          </v-btn>

          <!-- Slide-out Glassmorphic Panel when Expanded -->
          <div 
            class="entity-viewer-panel" 
            :class="isEntityViewerOpen ? 'expanded-state' : 'collapsed-state'"
          >
            <div class="panel-header">
              <span class="panel-title">
                <v-icon color="indigo-lighten-2" class="mr-2">mdi-shield-half-full</v-icon>
                Entities ({{ totalEntitiesCount }})
              </span>
              <v-btn
                icon="mdi-close"
                variant="text"
                size="small"
                class="close-btn"
                @click="isEntityViewerOpen = false"
              ></v-btn>
            </div>

            <v-divider class="border-white-05"></v-divider>

            <div class="panel-content">
              <!-- Grid columns of categories -->
              <div class="panel-columns" v-if="totalEntitiesCount > 0">
                <!-- Loop through categories as columns -->
                <div
                  v-for="(entities, catName) in nonUncategorizedGroups"
                  :key="catName"
                  class="category-column"
                  v-show="entities.length > 0"
                >
                  <div class="category-header d-flex justify-space-between align-center py-1 mb-2">
                    <span class="category-title text-caption font-weight-bold">
                      {{ catName.toUpperCase() }}
                    </span>
                    <span class="category-count text-caption text-grey">
                      {{ entities.length }}
                    </span>
                  </div>
                  
                  <div class="entity-list">
                    <div
                      v-for="entity in entities"
                      :key="entity.id"
                      class="entity-item pa-2 mb-2 rounded"
                    >
                      <div class="entity-meta d-flex justify-space-between align-center mb-1 ga-2">
                        <span class="entity-name font-weight-bold text-truncate text-left" :title="entity.name">
                          {{ entity.name }}
                        </span>
                        <span class="entity-hp-text text-caption text-grey text-right flex-shrink-0">
                          {{ formatNumber(entity.currentHp) }} / {{ formatNumber(entity.maxHp) }}
                        </span>
                      </div>

                      <!-- Beautiful, animated HP bar -->
                      <div class="hp-bar-bg">
                        <div
                          class="hp-bar-fill"
                          :class="getHPBarClass(catName)"
                          :style="{ width: getHPPercent(entity) + '%' }"
                        >
                          <div class="hp-bar-glow"></div>
                        </div>
                      </div>
                    </div>
                  </div>
                </div>
              </div>

              <!-- Empty State -->
              <div v-else class="empty-state text-center pa-4 text-grey h-100 justify-center">
                <v-icon size="36" class="mb-2" style="opacity: 0.3;">mdi-radar</v-icon>
                <div class="text-caption">No entities in area</div>
              </div>
            </div>
          </div>
        </div>

      </div>
    </v-main>
  </v-layout>
</template>

<script lang="ts">
import { defineComponent, ref, computed, inject, nextTick, onMounted, onBeforeUnmount } from "vue";
import { fightSummary, selectedTargetId, isNavDrawerOpen } from "@/store";
import SessionPanel from "@/components/SessionPanel.vue";
import ApplyDamageBySkillComponent from "@/components/applyDamageBySkill.vue";
import DamageTakenBySourceComponent from "@/components/DamageTakenBySource.vue";
import DamageGraph from "@/components/DamageGraph.vue";
import TargetConditionView from "@/components/TargetConditionView.vue";
import KnotIndicator from "@/components/KnotIndicator.vue";

const SPECIAL_TARGET_CLASSES: Record<number, string> = {
  7600: "target-blue",
  7601: "target-blue",
  7602: "target-green",
  7615: "target-gold",
  7603: "target-red",
};

export default defineComponent({
  name: "DamageMeterView",
  components: {
    SessionPanel,
    ApplyDamageBySkill: ApplyDamageBySkillComponent,
    DamageTakenBySource: DamageTakenBySourceComponent,
    DamageGraph,
    TargetConditionView,
    KnotIndicator,
  },
  setup() {
    const tab = ref("damageDealt");
    const condNameMap = inject("condNameMap") as any;
    const showBuffDetails = ref(false);

    // Entity Viewer State
    const isEntityViewerOpen = ref(false);

    const formatNumber = (num: number | undefined | null): string => {
      if (num === undefined || num === null) return "0";
      return Math.round(num).toLocaleString("en-US");
    };

    const formatDuration = (totalSeconds: number) => {
      if (totalSeconds < 0) totalSeconds = 0;
      const minutes = Math.floor(totalSeconds / 60);
      const seconds = Math.round(totalSeconds % 60);
      return `${minutes.toFixed(0).padStart(1, "0")}:${seconds
        .toFixed(0)
        .padStart(2, "0")}`;
    };

    const encounterDurationInSeconds = computed(() => {
      if (!selectedTargetId.value) {
        return fightSummary.encounterDuration;
      }
      let earliestStart = Infinity;
      let latestEnd = 0;
      let hasData = false;
      for (const player of Object.values(fightSummary.players)) {
        const targetBreakdown = player.damageByTarget[selectedTargetId.value];
        if (
          targetBreakdown &&
          targetBreakdown.startTime &&
          targetBreakdown.endTime
        ) {
          hasData = true;
          if (targetBreakdown.startTime < earliestStart) {
            earliestStart = targetBreakdown.startTime;
          }
          if (targetBreakdown.endTime > latestEnd) {
            latestEnd = targetBreakdown.endTime;
          }
        }
      }
      if (hasData && latestEnd > earliestStart) {
        return latestEnd - earliestStart;
      }
      return 0;
    });

    const formattedEncounterDuration = computed(() => {
      return formatDuration(encounterDurationInSeconds.value);
    });

    const totalPartyDamage = computed(() => {
      let total = 0;
      for (const player of Object.values(fightSummary.players)) {
        if (selectedTargetId.value) {
          if (player.damageByTarget[selectedTargetId.value]) {
            total += player.damageByTarget[selectedTargetId.value].totalDamage;
          }
        } else {
          total += player.overallStats.totalDamage;
        }
      }
      return total;
    });

    const formattedPartyDPS = computed(() => {
      const duration = encounterDurationInSeconds.value;
      if (duration > 0) {
        return formatNumber(totalPartyDamage.value / duration);
      }
      return "0";
    });

    const targetList = computed(() => {
      const totalDamageByTarget: Record<string, number> = {};
      for (const player of Object.values(fightSummary.players)) {
        for (const targetId in player.damageByTarget) {
          if (!totalDamageByTarget[targetId]) {
            totalDamageByTarget[targetId] = 0;
          }
          totalDamageByTarget[targetId] +=
            player.damageByTarget[targetId].totalDamage;
        }
      }
      const targets = Object.entries(fightSummary.targets).map(
        ([id, stats]) => {
          const damage = totalDamageByTarget[id] || 0;
          return {
            id,
            name: `${stats.name} (${formatNumber(damage)})`,
            rawName: stats.name,
            damage,
            seenAppear: stats.seenAppear,
            seenDead: stats.seenDead,
            disappeared: stats.disappeared,
            startTime: stats.startTime,
            endTime: stats.endTime,
            raceId: stats.raceId,
          };
        }
      );
      targets.sort((a, b) => {
        const startA = a.startTime || 0;
        const startB = b.startTime || 0;
        if (startB !== startA) {
          return startB - startA;
        }
        return b.damage - a.damage;
      });
      const grandTotalDamage = Object.values(totalDamageByTarget).reduce(
        (sum, dmg) => sum + dmg,
        0
      );
      targets.unshift({
        id: "",
        name: `All Targets (${formatNumber(grandTotalDamage)})`,
        rawName: "All Targets",
        damage: grandTotalDamage,
        seenAppear: undefined,
        seenDead: undefined,
        disappeared: undefined,
        startTime: undefined,
        endTime: undefined,
        raceId: undefined,
      });
      return targets;
    });

    const attackerNameMap = computed(() => {
      const map: { [id: string]: string } = {};
      if (fightSummary.players) {
        for (const [id, player] of Object.entries(fightSummary.players)) {
          map[id] = player.name;
        }
      }
      return map;
    });

    const formatTime = (ts: number | undefined): string => {
      if (!ts) return "";
      const date = new Date(ts * 1000);
      return date.toLocaleTimeString("en-US", { 
        hour12: false, 
        hour: '2-digit', 
        minute: '2-digit', 
        second: '2-digit' 
      });
    };

    const formatRelativeTime = (ts: number | undefined): string => {
      if (ts === undefined || !fightSummary.startTime) return "0:00";
      const relativeSeconds = ts - fightSummary.startTime;
      return formatDuration(relativeSeconds);
    };

    const menuMaxHeight = ref(400);

    const calculateMaxHeight = () => {
      nextTick(() => {
        const el = document.querySelector(".target-select-refined");
        if (el) {
          const rect = el.getBoundingClientRect();
          // Calculate available vertical space below select activator to viewport bottom
          const spaceBelow = window.innerHeight - rect.bottom - 24; // 24px safe padding
          menuMaxHeight.value = Math.max(200, spaceBelow);
        }
      });
    };

    const handleMenuUpdate = (isOpen: boolean) => {
      if (isOpen) {
        calculateMaxHeight();
      }
    };

    onMounted(() => {
      window.addEventListener("resize", calculateMaxHeight);
    });

    onBeforeUnmount(() => {
      window.removeEventListener("resize", calculateMaxHeight);
    });

    const getKnotColors = (target: any) => {
      if (!target) return { left: '#81c784', right: '#81c784', border: '#043916', tooltip: '' };
      
      let centerColor = '#81c784'; // Default Green (still alive)
      let border = '#043916'; // Default Dark Green border
      let statusName = 'Active';
      
      if (target.seenDead) {
        centerColor = '#e97269'; // Red (died)
        border = '#460101'; // Dark Red border
        statusName = 'Died';
      } else if (target.disappeared) {
        centerColor = '#ffb74d'; // Orange (despawned before death)
        border = '#4d320c'; // Dark Orange border
        statusName = 'Despawned';
      }
      
      return {
        left: centerColor,
        right: centerColor,
        border,
        tooltip: statusName
      };
    };

    const getTargetClass = (raceId: number | undefined): string => {
      if (!raceId) return "";
      return SPECIAL_TARGET_CLASSES[raceId] || "";
    };

    return {
      getKnotColors,
      getTargetClass,
      tab,
      selectedTargetId,
      targetList,
      formattedEncounterDuration,
      formattedPartyDPS,
      attackerNameMap,
      formatNumber,
      formatDuration,
      formatTime,
      formatRelativeTime,
      menuMaxHeight,
      handleMenuUpdate,
      selectedTargetConditions: computed(() => {
        if (!selectedTargetId.value) return undefined;
        return fightSummary.targets[selectedTargetId.value]?.conditions;
      }),
      partyBuffs: computed(() => {
        const results: any[] = [];
        if (!fightSummary.partyBuffs) return results;

        for (const buff of fightSummary.partyBuffs) {
          const staticData = condNameMap.value[buff.id];
          const display: string[] = [];
          for (const metric of buff.metrics) {
            let labelText = metric.label;
            if (labelText === "Max Att") labelText = "Max Att";
            else if (labelText === "Magic Att") labelText = "Mgk Att";
            else if (labelText === "Cast Speed") labelText = "Speed";
            
            display.push(`${labelText}: ${metric.highest.toFixed(2)}%`);
          }
          
          results.push({
            id: buff.id,
            name: staticData?.name || `Buff ${buff.id}`,
            iconUrl: staticData?.iconUrl || "",
            displayValue: display
          });
        }
        return results;
      }),
      hasPartyBuffs: computed(() => {
        return !!(fightSummary.partyBuffs && fightSummary.partyBuffs.length > 0);
      }),
      showBuffDetails,
      partyBuffDetails: computed(() => {
        const results: any[] = [];
        if (!fightSummary.partyBuffs) return results;

        for (const buff of fightSummary.partyBuffs) {
          const staticData = condNameMap.value[buff.id];
          results.push({
            id: buff.id,
            name: staticData?.name || `Buff ${buff.id}`,
            iconUrl: staticData?.iconUrl || "",
            metrics: buff.metrics
          });
        }
        return results;
      }),
      isEntityViewerOpen,
      isNavDrawerOpen,
      totalEntitiesCount: computed(() => (fightSummary.currentEntities || []).length),
      nonUncategorizedGroups: computed(() => {
        const groups: Record<string, any[]> = {
          Enemies: [],
          Players: [],
          Pets: [],
          NPCs: [],
          Other: [],
        };
        
        const entities = fightSummary.currentEntities || [];
        entities.forEach((entity) => {
          const cat = entity.category || "Other";
          if (groups[cat]) {
            groups[cat].push(entity);
          } else {
            groups["Other"].push(entity);
          }
        });
        
        return groups;
      }),
      getHPPercent: (entity: any) => {
        if (!entity || entity.maxHp <= 0) return 0;
        const pct = (entity.currentHp / entity.maxHp) * 100;
        return Math.max(0, Math.min(100, pct));
      },
      getHPBarClass: (catName: string) => {
        if (catName === "Enemies") return "hp-enemy";
        if (catName === "Players") return "hp-player";
        if (catName === "Pets") return "hp-pet";
        if (catName === "NPCs") return "hp-npc";
        return "hp-other";
      },
    };
  },
});
</script>
<style scoped>
.dashboard-wrapper {
  max-width: 1600px;
  margin: 0 auto;
  width: 100%;
  padding-left: 24px;
  padding-right: 24px;
}

.dashboard-header {
  padding: 16px 0 16px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.08);
}

.header-main-row {
  display: flex;
  justify-content: space-between;
  align-items: flex-end;
}

.header-section {
  display: flex;
  flex-direction: column;
}

.target-section {
  width: 400px;
}

.header-label {
  font-size: 10px;
  font-weight: 700;
  letter-spacing: 0.1em;
  color: #fff;
  opacity: 0.9;
  margin-bottom: 8px;
}

.target-select-refined :deep(.v-field) {
  background: rgba(255, 255, 255, 0.05) !important;
  border-radius: 8px;
  border: 1px solid rgba(255, 255, 255, 0.12);
  transition: all 0.2s ease;
}

.target-select-refined :deep(.v-field--focused) {
  border-color: rgba(var(--v-theme-primary), 0.4);
}

.stats-section {
  flex-direction: row;
}

.stat-item {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
}

.stat-value {
  font-size: 1.75rem;
  font-weight: 600;
  color: #fff;
  line-height: 1;
}

.amber-text {
  color: #ffb74d;
}

.stat-divider {
  width: 1px;
  height: 32px;
  background: rgba(255, 255, 255, 0.1);
  align-self: center;
}

.conditions-bar {
  display: flex;
  background: rgba(23, 27, 36, 0.6);
  padding: 8px 0 8px 16px;
  border-radius: 8px;
  border: 1px solid rgba(255, 255, 255, 0.08);
  min-height: 72px;
  align-items: center;
  overflow: hidden;
}

.buff-detail-btn {
  height: 72px !important;
  width: 20px !important;
  min-width: 0 !important;
  padding: 0 !important;
  padding-right: 8px !important;
}

.buff-detail-btn :deep(.v-btn__overlay) {
  display: none !important;
}

.buff-detail-btn:hover .v-icon {
  color: #818cf8 !important;
}

.main-dashboard-content {
  border: 1px solid rgba(129, 138, 248, 0.15);
  border-radius: 12px;
  overflow: hidden;
  background: #171b24; /* Pop color */
  padding: 12px;
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.3);
}

.modern-tabs {
  min-height: 40px !important;
  height: 40px !important;
  border-bottom: 1px solid rgba(255, 255, 255, 0.05);
}

.modern-tabs :deep(.v-tab) {
  text-transform: none !important;
  font-weight: 600 !important;
  letter-spacing: 0 !important;
  font-size: 0.875rem !important;
  opacity: 0.7;
  color: #fff !important;
  transition: all 0.2s ease;
}

.modern-tabs :deep(.v-tab--selected) {
  opacity: 1;
  color: #fff !important;
  background: rgba(255, 255, 255, 0.03);
}

.modern-tabs :deep(.v-tab__slider) {
  height: 2px !important;
  bottom: 0 !important;
}

.target-item-grid {
  display: grid;
  grid-template-columns: 36px minmax(100px, 1fr) 140px 110px; /* Precise matching columns, name constrained to wrap */
  align-items: center;
  width: 100%;
  gap: 12px;
}

.target-col-name {
  white-space: normal;
  overflow-wrap: break-word;
  word-break: break-word;
}

.target-col-duration {
  white-space: nowrap;
  text-align: left;
}

.target-col-icons {
  white-space: nowrap;
  text-align: center;
  font-size: 1rem;
}

.target-col-damage {
  white-space: nowrap;
  text-align: right;
}

/* Beautiful target-specific backgrounds & gradients for the dropdown list items */
.target-blue {
  background: linear-gradient(90deg, rgba(37, 99, 235, 0.28) 0%, rgba(29, 78, 216, 0.5) 100%) !important;
  border-left: 5px solid #3b82f6 !important;
  box-shadow: inset 5px 0 10px rgba(59, 130, 246, 0.25) !important;
}
.target-blue:hover {
  background: linear-gradient(90deg, rgba(37, 99, 235, 0.42) 0%, rgba(29, 78, 216, 0.65) 100%) !important;
  border-left: 5px solid #60a5fa !important;
  box-shadow: inset 6px 0 15px rgba(96, 165, 250, 0.45) !important;
}

.target-green {
  background: linear-gradient(90deg, rgba(16, 185, 129, 0.28) 0%, rgba(4, 120, 87, 0.5) 100%) !important;
  border-left: 5px solid #10b981 !important;
  box-shadow: inset 5px 0 10px rgba(16, 185, 129, 0.25) !important;
}
.target-green:hover {
  background: linear-gradient(90deg, rgba(16, 185, 129, 0.42) 0%, rgba(4, 120, 87, 0.65) 100%) !important;
  border-left: 5px solid #34d399 !important;
  box-shadow: inset 6px 0 15px rgba(52, 211, 153, 0.45) !important;
}

.target-gold {
  background: linear-gradient(90deg, rgba(245, 158, 11, 0.28) 0%, rgba(180, 83, 9, 0.5) 100%) !important;
  border-left: 5px solid #f59e0b !important;
  box-shadow: inset 5px 0 10px rgba(245, 158, 11, 0.25) !important;
}
.target-gold:hover {
  background: linear-gradient(90deg, rgba(245, 158, 11, 0.42) 0%, rgba(180, 83, 9, 0.65) 100%) !important;
  border-left: 5px solid #fbbf24 !important;
  box-shadow: inset 6px 0 15px rgba(251, 191, 36, 0.45) !important;
}

.target-red {
  background: linear-gradient(90deg, rgba(239, 68, 68, 0.28) 0%, rgba(185, 28, 28, 0.5) 100%) !important;
  border-left: 5px solid #ef4444 !important;
  box-shadow: inset 5px 0 10px rgba(239, 68, 68, 0.25) !important;
}
.target-red:hover {
  background: linear-gradient(90deg, rgba(239, 68, 68, 0.42) 0%, rgba(185, 28, 28, 0.65) 100%) !important;
  border-left: 5px solid #f87171 !important;
  box-shadow: inset 6px 0 15px rgba(248, 113, 113, 0.45) !important;
}

/* --- ENTITY VIEWER FLOATING PANEL & BUTTON STYLES --- */
.entity-viewer-container {
  position: fixed;
  bottom: 24px;
  z-index: 99;
  transition: left 0.3s cubic-bezier(0.4, 0, 0.2, 1);
  display: flex;
  flex-direction: column-reverse;
  align-items: flex-start;
}

/* Floating button */
.entity-viewer-toggle-btn {
  background: rgba(23, 27, 36, 0.8) !important;
  backdrop-filter: blur(12px);
  border: 1px solid rgba(129, 138, 248, 0.3) !important;
  color: #fff !important;
  font-weight: 700 !important;
  letter-spacing: 0.05em !important;
  text-transform: uppercase !important;
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.3), 0 0 15px rgba(129, 138, 248, 0.15);
  border-radius: 10px !important;
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1) !important;
}

.entity-viewer-toggle-btn:hover {
  transform: translateY(-2px);
  border-color: rgba(129, 138, 248, 0.6) !important;
  box-shadow: 0 6px 24px rgba(0, 0, 0, 0.4), 0 0 20px rgba(129, 138, 248, 0.3);
}

/* Sliding Panel */
.entity-viewer-panel {
  width: 900px;
  max-width: calc(100vw - 360px);
  height: 480px;
  background: rgba(17, 20, 28, 0.95);
  backdrop-filter: blur(16px);
  border: 1px solid rgba(129, 138, 248, 0.25);
  border-radius: 12px;
  box-shadow: 0 12px 40px rgba(0, 0, 0, 0.6);
  display: flex;
  flex-direction: column;
  overflow: hidden;
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
  transform-origin: bottom left;
}

.entity-viewer-panel.collapsed-state {
  transform: scale(0.85);
  opacity: 0;
  pointer-events: none;
  height: 0;
  width: 0;
  margin-bottom: 0;
}

.entity-viewer-panel.expanded-state {
  transform: scale(1);
  opacity: 1;
  pointer-events: auto;
  margin-bottom: 12px;
}

/* Panel Internal Styling */
.panel-header {
  padding: 12px 16px;
  display: flex;
  justify-content: space-between;
  align-items: center;
  background: rgba(255, 255, 255, 0.02);
  flex-shrink: 0;
}

.panel-title {
  font-size: 0.9rem;
  font-weight: 700;
  color: #fff;
  letter-spacing: 0.05em;
  display: flex;
  align-items: center;
}

.panel-content {
  flex: 1;
  overflow: hidden;
  padding: 12px 16px;
  display: flex;
  flex-direction: column;
}

.panel-columns {
  display: flex;
  flex-direction: row;
  gap: 16px;
  height: 100%;
  width: 100%;
  overflow-x: auto;
  overflow-y: hidden;
}

.category-column {
  flex: 1 1 0;
  min-width: 220px;
  max-width: 320px;
  display: flex;
  flex-direction: column;
  background: rgba(255, 255, 255, 0.01);
  border-radius: 8px;
  padding: 8px;
  border: 1px solid rgba(255, 255, 255, 0.03);
  height: 100%;
}

.entity-list {
  flex: 1;
  overflow-y: auto;
  padding-right: 4px;
}

/* Custom Scrollbar for columns scroll */
.panel-columns::-webkit-scrollbar {
  height: 6px;
}
.panel-columns::-webkit-scrollbar-track {
  background: transparent;
}
.panel-columns::-webkit-scrollbar-thumb {
  background: rgba(255, 255, 255, 0.05);
  border-radius: 3px;
}
.panel-columns::-webkit-scrollbar-thumb:hover {
  background: rgba(255, 255, 255, 0.12);
}

/* Custom Scrollbar for entity list in each column */
.entity-list::-webkit-scrollbar {
  width: 4px;
}
.entity-list::-webkit-scrollbar-track {
  background: transparent;
}
.entity-list::-webkit-scrollbar-thumb {
  background: rgba(255, 255, 255, 0.08);
  border-radius: 2px;
}
.entity-list::-webkit-scrollbar-thumb:hover {
  background: rgba(255, 255, 255, 0.15);
}

.category-header {
  border-bottom: 1px solid rgba(255, 255, 255, 0.05);
  margin-bottom: 8px;
  flex-shrink: 0;
}

.category-title {
  color: #818cf8; /* Light indigo */
  letter-spacing: 0.05em;
}

.entity-item {
  background: rgba(255, 255, 255, 0.03);
  border: 1px solid rgba(255, 255, 255, 0.05);
  transition: all 0.2s ease;
}

.entity-item:hover {
  background: rgba(255, 255, 255, 0.05);
  border-color: rgba(129, 138, 248, 0.2);
}

.entity-name {
  color: #fff;
  font-size: 0.85rem;
  max-width: 170px;
}

.entity-hp-text {
  font-size: 0.75rem;
}

/* HP Bar Styling */
.hp-bar-bg {
  width: 100%;
  height: 6px;
  background: rgba(0, 0, 0, 0.4);
  border-radius: 3px;
  overflow: hidden;
  position: relative;
}

.hp-bar-fill {
  height: 100%;
  border-radius: 3px;
  position: relative;
  transition: width 0.3s ease;
}

.hp-bar-glow {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: linear-gradient(
    90deg,
    rgba(255, 255, 255, 0) 0%,
    rgba(255, 255, 255, 0.15) 50%,
    rgba(255, 255, 255, 0) 100%
  );
  animation: hp-shimmer 2s infinite linear;
}

@keyframes hp-shimmer {
  0% { transform: translateX(-100%); }
  100% { transform: translateX(100%); }
}

/* Gradients for different categories */
.hp-enemy {
  background: linear-gradient(90deg, #ef4444 0%, #ff7849 100%);
  box-shadow: 0 0 8px rgba(239, 68, 68, 0.4);
}

.hp-player {
  background: linear-gradient(90deg, #10b981 0%, #059669 100%);
  box-shadow: 0 0 8px rgba(16, 185, 129, 0.4);
}

.hp-pet {
  background: linear-gradient(90deg, #8b5cf6 0%, #a78bfa 100%);
  box-shadow: 0 0 8px rgba(139, 92, 246, 0.4);
}

.hp-npc {
  background: linear-gradient(90deg, #0ea5e9 0%, #38bdf8 100%);
  box-shadow: 0 0 8px rgba(14, 165, 233, 0.4);
}

.hp-other {
  background: linear-gradient(90deg, #6b7280 0%, #9ca3af 100%);
}

.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
}

</style>
