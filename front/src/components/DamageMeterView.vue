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
          <div class="conditions-bar mt-4" v-if="selectedTargetId">
            <div class="header-label opacity-50">TARGET CONDITIONS</div>
            <target-condition-view
              :conditions="selectedTargetConditions"
              :attackerNameMap="attackerNameMap"
              class="ml-0"
            />
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
import { defineComponent, ref, computed } from "vue";
import { fightSummary, selectedTargetId } from "@/store";
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
  color: rgba(255, 255, 255, 0.4);
  margin-bottom: 8px;
}

.target-select-refined :deep(.v-field) {
  background: rgba(255, 255, 255, 0.03) !important;
  border-radius: 8px;
  border: 1px solid rgba(255, 255, 255, 0.08);
  transition: border-color 0.2s ease;
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
  flex-direction: column;
  align-items: flex-start;
  background: rgba(255, 255, 255, 0.02);
  padding: 12px 16px;
  border-radius: 8px;
  border: 1px solid rgba(255, 255, 255, 0.05);
}

.main-dashboard-content {
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 12px;
  overflow: hidden;
  background: rgba(255, 255, 255, 0.01);
  padding: 24px;
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
  opacity: 0.4;
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
