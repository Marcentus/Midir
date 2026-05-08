<template>
  <v-layout class="h-100">
    <session-panel />

    <v-main>
      <div class="dashboard-wrapper">
        <v-sheet class="d-flex align-center pa-2">
          <v-select
            v-model="selectedTargetId"
            :items="targetList"
            item-title="name"
            item-value="id"
            label="Filter by Target"
            variant="outlined"
            density="compact"
            hide-details
            style="max-width: 400px"
          ></v-select>

          <div class="d-flex align-center ml-4">
            <span class="text-subtitle-1 font-weight-medium">Party DPS: {{ formattedPartyDPS }}</span>
            <target-condition-view
              v-if="selectedTargetId"
              :conditions="selectedTargetConditions"
              :attackerNameMap="attackerNameMap"
            />
          </div>

          <v-spacer></v-spacer>
          <div class="d-flex align-center text-h6 px-4">
            <v-icon start>mdi-timer-outline</v-icon>
            <span class="font-weight-bold">{{ formattedEncounterDuration }}</span>
          </div>
        </v-sheet>

        <v-tabs v-model="tab" grow>
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
}
</style>
