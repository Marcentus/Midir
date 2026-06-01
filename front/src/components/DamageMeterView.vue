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
                COMBAT TARGET
              </div>
              <v-menu
                v-model="isTargetMenuOpen"
                :close-on-content-click="false"
                location="bottom start"
                :offset="4"
              >
                <template v-slot:activator="{ props }">
                  <div
                    v-bind="props"
                    class="target-select-custom-trigger d-flex align-center justify-space-between px-3 py-2 cursor-pointer"
                  >
                    <div class="d-flex align-center font-weight-medium text-body-2" style="min-width: 0; flex: 1;">
                      <span class="text-truncate">{{ selectedTargetName }}</span>
                      <v-tooltip
                        v-if="selectedTargetSeenAppear === false"
                        location="top"
                        text="Spawn not captured by parser."
                      >
                        <template v-slot:activator="{ props: tooltipProps }">
                          <v-icon
                            v-bind="tooltipProps"
                            icon="mdi-alert-circle-outline"
                            color="warning"
                            size="small"
                            class="ml-1 flex-shrink-0"
                          ></v-icon>
                        </template>
                      </v-tooltip>
                    </div>
                    <div class="d-flex align-center flex-shrink-0 ml-2">
                      <span class="text-caption text-grey mr-2" v-if="selectedTargetId">
                        ({{ formatCompact(selectedTargetDamage) }})
                      </span>
                      <v-icon icon="mdi-menu-down" color="grey"></v-icon>
                    </div>
                  </div>
                </template>
                
                <div
                  class="target-menu-container d-flex animate-all"
                  :style="{
                    width: expandedGroup ? '768px' : '380px',
                    height: menuHeight + 'px',
                    transition: 'width 0.25s cubic-bezier(0.4, 0, 0.2, 1)',
                    position: 'relative',
                    overflow: 'hidden',
                    background: 'transparent'
                  }"
                >
                  <!-- Left Column: Target List -->
                  <div
                    class="target-menu-card d-flex flex-column"
                    style="width: 380px; height: 100%; flex-shrink: 0;"
                  >
                    <div class="flex-grow-1 overflow-y-auto pa-0 custom-scrollbar">
                      <div
                        v-for="item in targetList"
                        :key="item.id"
                        class="target-menu-item cursor-pointer position-relative overflow-hidden"
                        :class="[
                          getTargetClass(item.raceId),
                          selectedTargetId === item.id ? 'active-item' : '',
                          expandedGroup && expandedGroup.id === item.id ? 'expanded-group-item' : ''
                        ]"
                        @click="handleItemClick(item)"
                      >
                        <div class="target-item-content d-flex justify-space-between align-center px-4 pt-3 pb-2">
                          <span class="target-col-name font-weight-bold d-flex align-center text-white">
                            <span>{{ item.rawName || item.name }}</span>
                            <v-tooltip
                              v-if="item.id && item.seenAppear === false"
                              location="top"
                              text="Spawn not captured by parser."
                            >
                              <template v-slot:activator="{ props: tooltipProps }">
                                <v-icon
                                  v-bind="tooltipProps"
                                  icon="mdi-alert-circle-outline"
                                  color="warning"
                                  size="small"
                                  class="ml-1 flex-shrink-0"
                                ></v-icon>
                              </template>
                            </v-tooltip>
                          </span>
                          
                          <div class="d-flex align-center ga-2">
                            <span v-if="item.id && item.hasHpUpdates" class="hp-text text-caption text-grey-lighten-1 font-weight-medium text-right">
                              {{ formatCompact(item.currentHp) }} / {{ formatCompact(item.maxHp) }}
                              <span class="text-grey ml-1">({{ item.hpPercent.toFixed(1) }}%)</span>
                            </span>
                            <span v-else class="target-col-damage font-weight-bold text-amber text-right">
                              {{ formatCompact(item.damage) }}
                            </span>
                            <v-icon
                              v-if="item.isGroup"
                              icon="mdi-chevron-right"
                              size="small"
                              class="group-arrow-icon"
                              :class="{ 'rotated': expandedGroup && expandedGroup.id === item.id }"
                            ></v-icon>
                          </div>
                        </div>

                        <!-- HP Bar Separator -->
                        <div v-if="item.id && item.hasHpUpdates" class="hp-bar-separator w-100">
                          <div
                            class="hp-bar-separator-fill hp-enemy"
                            :style="{ width: item.hpPercent + '%' }"
                          ></div>
                        </div>
                      </div>
                    </div>
                  </div>
                  
                  <!-- Right Column: Group Targets -->
                   <div
                    v-if="expandedGroup"
                    class="target-menu-card d-flex flex-column"
                    :style="{
                      width: '380px',
                      position: 'absolute',
                      left: '388px',
                      top: 0,
                      height: '100%',
                      flexShrink: 0
                    }"
                  >
                    <div class="flex-grow-1 overflow-y-auto pa-0 custom-scrollbar">
                      <div
                        v-for="subItem in expandedGroup.targets"
                        :key="subItem.id"
                        class="target-menu-item cursor-pointer position-relative overflow-hidden"
                        :class="[
                          getTargetClass(subItem.raceId),
                          selectedTargetId === subItem.id ? 'active-item' : ''
                        ]"
                        @click="selectGroupTarget(subItem)"
                      >
                        <div class="target-item-content d-flex justify-space-between align-center px-4 pt-3 pb-2">
                          <span class="target-col-name font-weight-bold d-flex align-center text-white">
                            <span>{{ subItem.rawName || subItem.name }}</span>
                            <v-tooltip
                              v-if="subItem.seenAppear === false"
                              location="top"
                              text="Spawn not captured by parser."
                            >
                              <template v-slot:activator="{ props: tooltipProps }">
                                <v-icon
                                  v-bind="tooltipProps"
                                  icon="mdi-alert-circle-outline"
                                  color="warning"
                                  size="small"
                                  class="ml-1 flex-shrink-0"
                                ></v-icon>
                              </template>
                            </v-tooltip>
                          </span>
                          
                          <span v-if="subItem.hasHpUpdates" class="hp-text text-caption text-grey-lighten-1 font-weight-medium text-right">
                            {{ formatCompact(subItem.currentHp) }} / {{ formatCompact(subItem.maxHp) }}
                            <span class="text-grey ml-1">({{ subItem.hpPercent.toFixed(1) }}%)</span>
                          </span>
                          <span v-else class="target-col-damage font-weight-bold text-amber text-right">
                            {{ formatCompact(subItem.damage) }}
                          </span>
                        </div>

                        <!-- HP Bar Separator -->
                        <div v-if="subItem.hasHpUpdates" class="hp-bar-separator w-100">
                          <div
                            class="hp-bar-separator-fill hp-enemy"
                            :style="{ width: subItem.hpPercent + '%' }"
                          ></div>
                        </div>
                      </div>
                    </div>
                  </div>
                </div>
              </v-menu>
            </div>

            <!-- Right: Vital Stats -->
            <div class="header-section stats-section">
              <div class="stat-item">
                <div class="header-label text-right">TARGET HP</div>
                <div class="stat-value">
                  <template v-if="selectedTargetHp">
                    <span style="color: #ff6e6d;">{{ formatNumber(selectedTargetHp.current) }}/{{ formatNumber(selectedTargetHp.max) }}</span>
                    <span class="text-grey text-caption ml-1">({{ selectedTargetHp.percent.toFixed(1) }}%)</span>
                  </template>
                  <template v-else>
                    N/A
                  </template>
                </div>
              </div>
              <div class="stat-divider mx-5"></div>
              <div class="stat-item">
                <div class="header-label text-right">DAMAGE</div>
                <div class="stat-value">{{ formatNumber(selectedTargetDamage) }}</div>
              </div>
              <div class="stat-divider mx-5"></div>
              <div class="stat-item">
                <div class="header-label text-right">PARTY DPS</div>
                <div class="stat-value amber-text">{{ formattedPartyDPS }}</div>
              </div>
              <div class="stat-divider mx-5"></div>
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



      </div>
    </v-main>
  </v-layout>
</template>

<script lang="ts">
import { defineComponent, ref, computed, inject, nextTick, onMounted, onBeforeUnmount, watch } from "vue";
import { fightSummary, selectedTargetId, isNavDrawerOpen, activeSessionId } from "@/store";
import SessionPanel from "@/components/SessionPanel.vue";
import ApplyDamageBySkillComponent from "@/components/applyDamageBySkill.vue";
import DamageTakenBySourceComponent from "@/components/DamageTakenBySource.vue";
import DamageGraph from "@/components/DamageGraph.vue";
import TargetConditionView from "@/components/TargetConditionView.vue";

const SPECIAL_TARGET_CLASSES: Record<number, string> = {
  7600: "target-blue",
  7601: "target-blue",
  7602: "target-green",
  7615: "target-gold",
  7603: "target-pearl",
};

export default defineComponent({
  name: "DamageMeterView",
  components: {
    SessionPanel,
    ApplyDamageBySkill: ApplyDamageBySkillComponent,
    DamageTakenBySource: DamageTakenBySourceComponent,
    DamageGraph,
    TargetConditionView,
  },
  setup() {
    const tab = ref("damageDealt");
    const condNameMap = inject("condNameMap") as any;
    const showBuffDetails = ref(false);



    const formatNumber = (num: number | undefined | null): string => {
      if (num === undefined || num === null) return "0";
      return Math.round(num).toLocaleString("en-US");
    };

    const formatCompact = (num: number | undefined | null): string => {
      if (num === undefined || num === null) return "0";
      return new Intl.NumberFormat("en-US", {
        notation: "compact",
        compactDisplay: "short",
        maximumFractionDigits: 1,
      }).format(num).toLowerCase();
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

    // Target grouping state
    const isTargetMenuOpen = ref(false);
    const expandedGroup = ref<any>(null);

    const GROUPED_ENCOUNTERS = [
      { bossRaceId: 7615, groupName: "Bri Leith: Gate 4" },
      { bossRaceId: 7603, groupName: "Bri Leith: Gate 3" },
      { bossRaceId: 7600, partnerRaceId: 7601, groupName: "Bri Leith: Gate 1" }
    ];

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
      const grandTotalDamage = Object.values(totalDamageByTarget).reduce(
        (sum, dmg) => sum + dmg,
        0
      );
      const targets = Object.entries(fightSummary.targets).map(
        ([id, stats]) => {
          const damage = totalDamageByTarget[id] || 0;
          const damagePercent = grandTotalDamage > 0 ? (damage / grandTotalDamage) * 100 : 0;

          // Determine current and max HP
          let currentHp = 0;
          let maxHp = 0;

          // 1. Live sessions: try to find entity in currentEntities
          if (activeSessionId.value === "live" && fightSummary.currentEntities) {
            const entity = fightSummary.currentEntities.find(e => e.id === id);
            if (entity) {
              currentHp = entity.currentHp;
              maxHp = entity.maxHp;
            }
          }

          // 2. Fallback to lowest HP percentage recorded in hpHistory
          if (maxHp === 0 && stats.hpHistory && stats.hpHistory.length > 0) {
            let lowestPct = Infinity;
            let correspondingCurrentHp = 0;
            let correspondingMaxHp = 0;
            for (const pt of stats.hpHistory) {
              if (pt.maxHp > 0) {
                const pct = pt.currentHp / pt.maxHp;
                if (pct < lowestPct) {
                  lowestPct = pct;
                  correspondingCurrentHp = pt.currentHp;
                  correspondingMaxHp = pt.maxHp;
                }
              }
            }
            if (lowestPct !== Infinity) {
              currentHp = correspondingCurrentHp;
              maxHp = correspondingMaxHp;
            }
          }

          // Determine if we actually found real HP updates
          const hasHpUpdates = maxHp > 0;

          // 3. Last resort fallbacks
          if (maxHp === 0) {
            if (stats.seenDead) {
              currentHp = 0;
              maxHp = 100;
            } else {
              currentHp = 100;
              maxHp = 100;
            }
          }

          // If the target is dead, override current HP to 0
          if (stats.seenDead) {
            currentHp = 0;
          }

          const hpPercent = maxHp > 0 ? Math.max(0, Math.min(100, (currentHp / maxHp) * 100)) : 0;

          return {
            id,
            name: stats.name,
            rawName: stats.name,
            damage,
            damagePercent,
            currentHp,
            maxHp,
            hpPercent,
            seenAppear: stats.seenAppear,
            seenDead: stats.seenDead,
            disappeared: stats.disappeared,
            startTime: stats.startTime,
            endTime: stats.endTime,
            raceId: stats.raceId,
            hasHpUpdates,
          };
        }
      );

      // Check if parse contains any entity deaths (seenDead)
      const hasAnyEntityDeath = targets.some(t => t.seenDead);

      if (!hasAnyEntityDeath) {
        // Return normal sorted flat list
        targets.sort((a, b) => {
          const startA = a.startTime || 0;
          const startB = b.startTime || 0;
          if (startB !== startA) {
            return startB - startA;
          }
          return b.damage - a.damage;
        });
        targets.unshift({
          id: "",
          name: "All Targets",
          rawName: "All Targets",
          damage: grandTotalDamage,
          damagePercent: 100,
          currentHp: 0,
          maxHp: 0,
          hpPercent: 0,
          seenAppear: undefined,
          seenDead: undefined,
          disappeared: undefined,
          startTime: undefined,
          endTime: undefined,
          raceId: undefined,
          hasHpUpdates: false,
        });
        return targets;
      }

      // Grouping Logic
      const groupedList: any[] = [];
      const groupedTargetIds = new Set<string>();

      for (const bossDef of GROUPED_ENCOUNTERS) {
        // Find all boss targets of this bossRaceId
        const bosses = targets.filter(t => t.raceId === bossDef.bossRaceId);

        for (const bossTarget of bosses) {
          // Special reset detection check for Phase 1 boss despawning without reaching 50% HP
          if (bossDef.bossRaceId === 7600) {
            const rawStats = fightSummary.targets[bossTarget.id];
            const round3 = (num: number) => Math.round(num * 1000) / 1000;
            let reached50 = round3(bossTarget.hpPercent) <= 50;
            if (!reached50 && rawStats && rawStats.hpHistory) {
              reached50 = rawStats.hpHistory.some(pt => pt.maxHp > 0 && round3((pt.currentHp / pt.maxHp) * 100) <= 50);
            }

            const ended = bossTarget.seenDead || bossTarget.disappeared;
            if (ended && !reached50) {
              // Run reset! Skip grouping this bossTarget
              continue;
            }
          }

          const bossStartTime = bossTarget.startTime || 0;
          let bossEndTime = (bossTarget.seenDead || bossTarget.disappeared) ? (bossTarget.endTime || 0) : Infinity;

          let partnerTarget: any = null;
          if (bossDef.partnerRaceId) {
            // Look for partner spawning after bossStartTime
            partnerTarget = targets.find(t => t.raceId === bossDef.partnerRaceId && (t.startTime || 0) >= bossStartTime);
            if (partnerTarget) {
              // The encounter ends when the partner dies or disappears
              bossEndTime = (partnerTarget.seenDead || partnerTarget.disappeared) ? (partnerTarget.endTime || 0) : Infinity;
            }
          }

          // Find all targets that appeared during the boss's lifetime
          const members = targets.filter(t => {
            const tStartTime = t.startTime || 0;
            return tStartTime >= bossStartTime && (bossEndTime === Infinity || tStartTime <= bossEndTime);
          });

          if (members.length > 0) {
            const groupTotalDamage = members.reduce((sum, m) => sum + m.damage, 0);
            const groupDamagePercent = grandTotalDamage > 0 ? (groupTotalDamage / grandTotalDamage) * 100 : 0;

            // Determine active HP details (use partner's HP if active, else boss's HP)
            const activeBoss = (partnerTarget && !partnerTarget.seenDead && !partnerTarget.disappeared) ? partnerTarget : bossTarget;

            const groupItem = {
              id: `group_${bossTarget.id}`,
              name: bossDef.groupName,
              rawName: bossDef.groupName,
              isGroup: true,
              damage: groupTotalDamage,
              damagePercent: groupDamagePercent,
              currentHp: activeBoss.currentHp,
              maxHp: activeBoss.maxHp,
              hpPercent: activeBoss.hpPercent,
              seenAppear: activeBoss.seenAppear,
              seenDead: activeBoss.seenDead,
              disappeared: activeBoss.disappeared,
              startTime: bossTarget.startTime,
              endTime: partnerTarget ? partnerTarget.endTime : bossTarget.endTime,
              raceId: activeBoss.raceId,
              hasHpUpdates: activeBoss.hasHpUpdates,
              targets: [...members].sort((a, b) => {
                const startA = a.startTime || 0;
                const startB = b.startTime || 0;
                if (startA !== startB) {
                  return startA - startB; // Chronological ascending order
                }
                return b.damage - a.damage; // Tie-breaker: highest damage first
              }),
            };

            groupedList.push(groupItem);
            members.forEach(m => groupedTargetIds.add(m.id));
          }
        }
      }

      // Filter out all targets that have been grouped from the top level
      const ungroupedTargets = targets.filter(t => !groupedTargetIds.has(t.id));

      const combined = [...ungroupedTargets, ...groupedList];

      combined.sort((a, b) => {
        const startA = a.startTime || 0;
        const startB = b.startTime || 0;
        if (startB !== startA) {
          return startB - startA;
        }
        return b.damage - a.damage;
      });

      combined.unshift({
        id: "",
        name: "All Targets",
        rawName: "All Targets",
        damage: grandTotalDamage,
        damagePercent: 100,
        currentHp: 0,
        maxHp: 0,
        hpPercent: 0,
        seenAppear: undefined,
        seenDead: undefined,
        disappeared: undefined,
        startTime: undefined,
        endTime: undefined,
        raceId: undefined,
        hasHpUpdates: false,
      });

      return combined;
    });

    const selectedTargetName = computed(() => {
      if (!selectedTargetId.value) return "All Targets";
      for (const item of targetList.value) {
        if (item.id === selectedTargetId.value) {
          return item.rawName || item.name;
        }
        if (item.isGroup && item.targets) {
          const sub = item.targets.find((t: any) => t.id === selectedTargetId.value);
          if (sub) {
            return sub.rawName || sub.name;
          }
        }
      }
      return fightSummary.targets[selectedTargetId.value]?.name || "Unknown Target";
    });

    const selectedTargetSeenAppear = computed(() => {
      if (!selectedTargetId.value) return undefined;
      for (const item of targetList.value) {
        if (item.id === selectedTargetId.value) return item.seenAppear;
        if (item.isGroup && item.targets) {
          const sub = item.targets.find((t: any) => t.id === selectedTargetId.value);
          if (sub) return sub.seenAppear;
        }
      }
      return fightSummary.targets[selectedTargetId.value]?.seenAppear;
    });

    const selectedTargetDamage = computed(() => {
      if (!selectedTargetId.value) {
        return targetList.value[0]?.damage || 0;
      }
      for (const item of targetList.value) {
        if (item.id === selectedTargetId.value) return item.damage;
        if (item.isGroup && item.targets) {
          const sub = item.targets.find((t: any) => t.id === selectedTargetId.value);
          if (sub) return sub.damage;
        }
      }
      return 0;
    });

    const selectedTargetHp = computed(() => {
      if (!selectedTargetId.value) return null;
      for (const item of targetList.value) {
        if (item.id === selectedTargetId.value) {
          if (!item.hasHpUpdates) return null;
          return { current: item.currentHp, max: item.maxHp, percent: item.hpPercent };
        }
        if (item.isGroup && item.targets) {
          const sub = item.targets.find((t: any) => t.id === selectedTargetId.value);
          if (sub) {
            if (!sub.hasHpUpdates) return null;
            return { current: sub.currentHp, max: sub.maxHp, percent: sub.hpPercent };
          }
        }
      }
      return null;
    });

    const handleItemClick = (item: any) => {
      if (item.isGroup) {
        if (expandedGroup.value && expandedGroup.value.id === item.id) {
          expandedGroup.value = null;
        } else {
          expandedGroup.value = item;
        }
      } else {
        selectedTargetId.value = item.id;
        isTargetMenuOpen.value = false;
        expandedGroup.value = null;
      }
    };

    const selectGroupTarget = (subItem: any) => {
      selectedTargetId.value = subItem.id;
      isTargetMenuOpen.value = false;
    };


    const attackerNameMap = computed(() => {
      const map: { [id: string]: string } = {};
      if (fightSummary.players) {
        for (const [id, player] of Object.entries(fightSummary.players)) {
          map[id] = player.name;
        }
      }
      return map;
    });

    const menuHeight = ref(400);

    const calculateMaxHeight = () => {
      nextTick(() => {
        const el = document.querySelector(".target-select-custom-trigger");
        if (el) {
          const rect = el.getBoundingClientRect();
          // Calculate available space from activator bottom to viewport bottom (with 24px safe padding)
          const spaceBelow = window.innerHeight - rect.bottom - 24;
          menuHeight.value = Math.max(200, spaceBelow);
        }
      });
    };

    watch(isTargetMenuOpen, (isOpen) => {
      if (isOpen) {
        calculateMaxHeight();
        if (selectedTargetId.value) {
          const groupContainingTarget = targetList.value.find(
            item => item.isGroup && item.targets && item.targets.some((sub: any) => sub.id === selectedTargetId.value)
          );
          if (groupContainingTarget) {
            expandedGroup.value = groupContainingTarget;
          } else {
            expandedGroup.value = null;
          }
        } else {
          expandedGroup.value = null;
        }
      }
    });

    watch(activeSessionId, () => {
      expandedGroup.value = null;
      isTargetMenuOpen.value = false;
    });

    onMounted(() => {
      window.addEventListener("resize", calculateMaxHeight);
    });

    onBeforeUnmount(() => {
      window.removeEventListener("resize", calculateMaxHeight);
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

    const getTargetClass = (raceId: number | undefined): string => {
      if (!raceId) return "";
      return SPECIAL_TARGET_CLASSES[raceId] || "";
    };

    return {
      getTargetClass,
      tab,
      selectedTargetId,
      targetList,
      formattedEncounterDuration,
      formattedPartyDPS,
      attackerNameMap,
      formatNumber,
      formatCompact,
      formatDuration,
      formatTime,
      isTargetMenuOpen,
      expandedGroup,
      selectedTargetName,
      selectedTargetSeenAppear,
      selectedTargetDamage,
      selectedTargetHp,
      handleItemClick,
      selectGroupTarget,

      menuHeight,
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
      isNavDrawerOpen,
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
  width: 380px;
}

.header-label {
  font-size: 10px;
  font-weight: 700;
  letter-spacing: 0.1em;
  color: #fff;
  opacity: 0.9;
  margin-bottom: 8px;
}

.target-select-custom-trigger {
  background: #171b24 !important;
  border-radius: 4px;
  border: 1px solid rgba(255, 255, 255, 0.1);
  height: 40px;
  transition: all 0.2s ease;
  user-select: none;
}

.target-select-custom-trigger:hover {
  background: rgba(255, 255, 255, 0.04) !important;
  border-color: rgba(255, 255, 255, 0.2);
}

.target-menu-card {
  background: #171b24 !important;
  border: 1px solid rgba(255, 255, 255, 0.1) !important;
  border-radius: 4px !important;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.5) !important;
  overflow: hidden !important;
}

.target-menu-column {
  background: #171b24 !important;
}

.target-menu-item {
  transition: background-color 0.15s ease !important;
  border-bottom: 1px solid rgba(255, 255, 255, 0.05) !important;
  background: transparent;
  border-left: 3px solid transparent !important;
}

.target-menu-item:hover {
  background: rgba(255, 255, 255, 0.04) !important;
}

.active-item {
  background: rgba(129, 138, 248, 0.12) !important;
  border-left: 3px solid #818cf8 !important;
}

.expanded-group-item {
  background: rgba(129, 138, 248, 0.12) !important;
  border-left: 3px solid #818cf8 !important;
}

.group-arrow-icon {
  transition: transform 0.25s cubic-bezier(0.4, 0, 0.2, 1);
}

.group-arrow-icon.rotated {
  transform: rotate(90deg);
}

/* Custom Scrollbar for dropdown lists */
.custom-scrollbar::-webkit-scrollbar {
  width: 6px;
}
.custom-scrollbar::-webkit-scrollbar-track {
  background: transparent;
}
.custom-scrollbar::-webkit-scrollbar-thumb {
  background: rgba(255, 255, 255, 0.1);
  border-radius: 3px;
}
.custom-scrollbar::-webkit-scrollbar-thumb:hover {
  background: rgba(255, 255, 255, 0.2);
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

.target-item-container {
  width: 100%;
}

.target-col-name {
  white-space: normal;
  overflow-wrap: break-word;
  word-break: break-word;
  font-size: 0.9rem;
}

.target-col-damage {
  white-space: nowrap;
  text-align: right;
  font-size: 0.9rem;
}

.target-list-item-refined {
  padding: 0 !important;
  min-height: 0 !important;
  border-bottom: 1px solid rgba(255, 255, 255, 0.04) !important;
  transition: background-color 0.2s ease, border-color 0.2s ease !important;
}

.target-list-item-refined:hover {
  background: rgba(255, 255, 255, 0.03) !important;
}

/* Override default list item padding to enable edge-to-edge content */
.target-list-item-refined :deep(.v-list-item__content) {
  padding: 0 !important;
}

.target-list-item-refined :deep(.v-list-item-title) {
  white-space: normal !important;
}

.hp-text {
  font-size: 0.75rem !important;
}

/* Premium full-width borderless HP bar separator stuck flush to the bottom */
.hp-bar-separator {
  height: 4px;
  background: rgba(0, 0, 0, 0.4);
  width: 100%;
  position: relative;
  overflow: hidden;
  border-radius: 0 !important;
}

.hp-bar-separator-fill {
  height: 100%;
  position: relative;
  transition: width 0.3s ease;
  border-radius: 0 !important;
}

/* Subtle target-specific left border markers & elegant hover backgrounds */
.target-blue {
  border-left: 3px solid #3b82f6 !important;
}
.target-blue:hover {
  background: rgba(59, 130, 246, 0.08) !important;
}
.target-blue.active-item {
  background: rgba(59, 130, 246, 0.12) !important;
}
.target-blue.expanded-group-item {
  background: rgba(59, 130, 246, 0.12) !important;
  border-left: 3px solid #3b82f6 !important;
}

.target-green {
  border-left: 3px solid #10b981 !important;
}
.target-green:hover {
  background: rgba(16, 185, 129, 0.08) !important;
}
.target-green.active-item {
  background: rgba(16, 185, 129, 0.12) !important;
}
.target-green.expanded-group-item {
  background: rgba(16, 185, 129, 0.12) !important;
  border-left: 3px solid #10b981 !important;
}

.target-gold {
  border-left: 3px solid #f59e0b !important;
}
.target-gold:hover {
  background: rgba(245, 158, 11, 0.08) !important;
}
.target-gold.active-item {
  background: rgba(245, 158, 11, 0.12) !important;
}
.target-gold.expanded-group-item {
  background: rgba(245, 158, 11, 0.12) !important;
  border-left: 3px solid #f59e0b !important;
}

.target-pearl {
  border-left: 3px solid #f8fafc !important;
}
.target-pearl:hover {
  background: rgba(248, 250, 252, 0.08) !important;
}
.target-pearl.active-item {
  background: rgba(248, 250, 252, 0.12) !important;
}
.target-pearl.expanded-group-item {
  background: rgba(248, 250, 252, 0.12) !important;
  border-left: 3px solid #f8fafc !important;
}

.hp-enemy {
  background: linear-gradient(90deg, #ef4444 0%, #ff7849 100%);
  box-shadow: 0 0 8px rgba(239, 68, 68, 0.4);
}

</style>
