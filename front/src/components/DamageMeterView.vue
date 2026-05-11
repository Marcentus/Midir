<template>
  <v-layout class="h-100">
    <session-panel />

    <v-main>
      <div class="dashboard-wrapper">
        <div class="dashboard-header mb-4">
          <div class="header-main-row">
            <!-- Left: Target Selector -->
            <div class="header-section target-section">
              <div class="header-label">COMBAT TARGET</div>
              <v-select
                v-model="selectedTargetId"
                :items="targetList"
                item-title="name"
                item-value="id"
                variant="solo-filled"
                flat
                density="compact"
                hide-details
                class="target-select-refined"
                placeholder="All Targets"
              ></v-select>
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
      </div>
    </v-main>
  </v-layout>
</template>

<script lang="ts">
import { defineComponent, ref, computed, inject } from "vue";
import { fightSummary, selectedTargetId } from "@/store";
import { parseMabinogiMetadata } from "@/utils/metadata";
import SessionPanel from "@/components/SessionPanel.vue";
import ApplyDamageBySkillComponent from "@/components/applyDamageBySkill.vue";
import DamageTakenBySourceComponent from "@/components/DamageTakenBySource.vue";
import DamageGraph from "@/components/DamageGraph.vue";
import TargetConditionView from "@/components/TargetConditionView.vue";

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
        ([id, stats]) => ({
          id,
          name: `${stats.name} (${formatNumber(totalDamageByTarget[id] || 0)})`,
        })
      );
      targets.sort((a, b) => {
        const damageA = totalDamageByTarget[a.id] || 0;
        const damageB = totalDamageByTarget[b.id] || 0;
        return damageB - damageA;
      });
      const grandTotalDamage = Object.values(totalDamageByTarget).reduce(
        (sum, dmg) => sum + dmg,
        0
      );
      targets.unshift({
        id: "",
        name: `All Targets (${formatNumber(grandTotalDamage)})`,
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

    return {
      tab,
      selectedTargetId,
      targetList,
      formattedEncounterDuration,
      formattedPartyDPS,
      attackerNameMap,
      selectedTargetConditions: computed(() => {
        if (!selectedTargetId.value) return undefined;
        return fightSummary.targets[selectedTargetId.value]?.conditions;
      }),
      partyBuffs: computed(() => {
        const ids = [680, 192];
        const results: any[] = [];

        if (!fightSummary.players) return results;

        for (const id of ids) {
          let bestValue = -1;
          let bestCond: any = null;

          for (const player of Object.values(fightSummary.players)) {
            const conditions = selectedTargetId.value 
              ? player.damageByTarget[selectedTargetId.value]?.conditions 
              : player.overallStats.conditions;

            const cond = conditions?.[id];
            if (cond && cond.metaBreakdown && cond.metaBreakdown.length > 0) {
              for (const meta of cond.metaBreakdown) {
                const parsed = parseMabinogiMetadata(meta.metaData);
                let val = 0;
                let display: string[] = [];
                if (id === 680) {
                  val = parsed.MCMBAMAX || 0;
                  display = [`Max Att: ${val.toFixed(2)}%`];
                } else if (id === 192) {
                  const lsma = parsed.LSMA || 0;
                  const mfcp = parsed.MFCP || 0;
                  val = Math.max(lsma, mfcp);
                  const labels: string[] = [];
                  if (lsma > 0) labels.push(`Mgk Att: ${lsma.toFixed(2)}%`);
                  if (mfcp > 0) labels.push(`Speed: ${mfcp.toFixed(2)}%`);
                  display = labels;
                }

                if (val > bestValue) {
                  bestValue = val;
                  const staticData = condNameMap.value[id];
                  bestCond = {
                    id,
                    name: staticData?.name || `Buff ${id}`,
                    iconUrl: staticData?.iconUrl || "",
                    displayValue: display,
                    playerName: player.name
                  };
                }
              }
            }
          }
          if (bestCond) results.push(bestCond);
        }
        return results;
      }),
      hasPartyBuffs: computed(() => {
        if (!fightSummary.players) return false;
        const ids = [680, 192];
        return Object.values(fightSummary.players).some(p => {
           const conditions = selectedTargetId.value 
              ? p.damageByTarget[selectedTargetId.value]?.conditions 
              : p.overallStats.conditions;
           return conditions && ids.some(id => conditions[id]);
        });
      }),
      showBuffDetails,
      partyBuffDetails: computed(() => {
        const ids = [680, 192];
        const results: any[] = [];

        if (!fightSummary.players) return results;

        for (const id of ids) {
          const keys = id === 680 ? ["MCMBAMAX"] : ["LSMA", "MFCP"];
          const metrics: Record<string, { highest: number, highestUptime: number, sumVD_Global: number, sumD_Global: number, maxUptime_Party: number }> = {};
          keys.forEach(k => metrics[k] = { highest: 0, highestUptime: 0, sumVD_Global: 0, sumD_Global: 0, maxUptime_Party: -1 });

          // Pass 1: Gather global metadata trends and majority status
          let maxDurForThisID = 0;
          let hasMajorityPlayer = false;

          for (const player of Object.values(fightSummary.players)) {
            const stats = selectedTargetId.value ? player.damageByTarget[selectedTargetId.value] : player.overallStats;
            if (!stats) continue;

            const condBFO = stats.conditions?.[680];
            const condVivace = stats.conditions?.[192];
            const durBFO = condBFO?.duration || 0;
            const durVivace = condVivace?.duration || 0;

            const durThis = id === 680 ? durBFO : durVivace;
            const durOther = id === 680 ? durVivace : durBFO;

            maxDurForThisID = Math.max(maxDurForThisID, durThis);
            if (durThis > 0 && durThis >= durOther) hasMajorityPlayer = true;

            const cond = stats.conditions?.[id];
            if (cond && cond.metaBreakdown) {
              for (const meta of cond.metaBreakdown) {
                const parsed = parseMabinogiMetadata(meta.metaData);
                for (const key of keys) {
                  const val = parsed[key] || 0;
                  if (val > 0) {
                    const m = metrics[key];
                    m.highest = Math.max(m.highest, val);
                    m.sumVD_Global += val * meta.duration;
                    m.sumD_Global += meta.duration;
                    if (meta.uptime > m.maxUptime_Party) {
                      m.maxUptime_Party = meta.uptime;
                      m.highestUptime = val;
                    }
                  }
                }
              }
            }
          }

          // Pass 2: Calculate damage-weighted average
          const sumVD_Weighted: Record<string, number> = {};
          const sumD_Recipient: Record<string, number> = {};
          keys.forEach(k => {
            sumVD_Weighted[k] = 0;
            sumD_Recipient[k] = 0;
          });

          for (const player of Object.values(fightSummary.players)) {
            const stats = selectedTargetId.value ? player.damageByTarget[selectedTargetId.value] : player.overallStats;
            const playerDamage = stats?.totalDamage || 0;
            if (!stats || playerDamage <= 0) continue;

            const condBFO = stats.conditions?.[680];
            const condVivace = stats.conditions?.[192];
            const durBFO = condBFO?.duration || 0;
            const durVivace = condVivace?.duration || 0;

            const durThis = id === 680 ? durBFO : durVivace;
            const durOther = id === 680 ? durVivace : durBFO;

            // Majority Eligibility:
            // 1. If player has this buff as majority, they are eligible.
            // 2. If NO ONE in the party has this buff as majority, the player with the highest duration contributes.
            let isEligible = false;
            if (durThis > 0 && durThis >= durOther) {
              isEligible = true;
            } else if (!hasMajorityPlayer && durThis > 0 && durThis >= maxDurForThisID) {
              isEligible = true;
            }

            if (!isEligible) continue;

            const cond = stats.conditions?.[id];
            if (!cond) continue;

            for (const key of keys) {
              let playerAvg = 0;
              let hasLocalData = false;

              if (cond.metaBreakdown && cond.metaBreakdown.length > 0) {
                let pSumVD = 0;
                let pSumD = 0;
                for (const meta of cond.metaBreakdown) {
                  const val = parseMabinogiMetadata(meta.metaData)[key] || 0;
                  if (val > 0) {
                    pSumVD += val * meta.duration;
                    pSumD += meta.duration;
                    hasLocalData = true;
                  }
                }
                if (pSumD > 0) playerAvg = pSumVD / pSumD;
              }

              // Fallback: If player has condition but no metadata, use the global party average
              if (!hasLocalData && metrics[key].sumD_Global > 0) {
                playerAvg = metrics[key].sumVD_Global / metrics[key].sumD_Global;
              }

              if (playerAvg > 0) {
                // The user wants a "Quality" metric: the average strength of the buff while it was applied,
                // weighted by the damage contribution of the players who received it.
                // We no longer dilute this by the encounter duration or buff uptime ratio.
                sumVD_Weighted[key] += playerAvg * playerDamage;
                sumD_Recipient[key] += playerDamage;
              }
            }
          }

          // Finalize metrics
          const formattedMetrics: any[] = [];
          for (const key of keys) {
            const m = metrics[key];
            if (m.highest > 0) {
              const weightedAvg = sumD_Recipient[key] > 0 ? sumVD_Weighted[key] / sumD_Recipient[key] : 0;
              
              let label = key;
              if (key === "MCMBAMAX") label = "Max Att";
              else if (key === "LSMA") label = "Magic Att";
              else if (key === "MFCP") label = "Cast Speed";

              formattedMetrics.push({
                label,
                highest: m.highest,
                highestUptime: m.highestUptime,
                weightedAvg: weightedAvg
              });
            }
          }

          if (formattedMetrics.length > 0) {
            const staticData = condNameMap.value[id];
            results.push({
              id,
              name: staticData?.name || `Buff ${id}`,
              iconUrl: staticData?.iconUrl || "",
              metrics: formattedMetrics
            });
          }
        }
        return results;
      }),
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
  width: 320px;
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
</style>
