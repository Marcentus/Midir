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
              <div class="d-flex flex-column align-end" v-if="hasPartyBuffs && partyBuffs.length > 0">
                <div class="header-label" style="opacity: 0.9;">PARTY BUFFS</div>
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
                      <span>{{ buff.name }} ({{ buff.playerName }})</span>
                    </v-tooltip>
                    <span class="text-caption font-weight-bold text-info" style="font-size: 0.75rem !important;">
                      {{ buff.displayValue }}
                    </span>
                  </div>
                </div>
              </div>
            </div>
          </div>
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
                let display = "";
                if (id === 680) {
                  val = parsed.MCMBAMAX || 0;
                  display = `Max Att: ${val.toFixed(2)}%`;
                } else if (id === 192) {
                  const lsma = parsed.LSMA || 0;
                  const mfcp = parsed.MFCP || 0;
                  val = Math.max(lsma, mfcp);
                  const labels: string[] = [];
                  if (lsma > 0) labels.push(`Magic Att: ${lsma.toFixed(2)}%`);
                  if (mfcp > 0) labels.push(`Cast Speed: ${mfcp.toFixed(2)}%`);
                  display = labels.join(", ");
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
        // Find if ANY player has ANY condition in the list
        if (!fightSummary.players) return false;
        const ids = [680, 192];
        return Object.values(fightSummary.players).some(p => {
           const conditions = selectedTargetId.value 
              ? p.damageByTarget[selectedTargetId.value]?.conditions 
              : p.overallStats.conditions;
           return conditions && ids.some(id => conditions[id]);
        });
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
  padding: 8px 16px;
  border-radius: 8px;
  border: 1px solid rgba(255, 255, 255, 0.08);
  min-height: 60px;
  align-items: center;
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
