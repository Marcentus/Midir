<template>
  <v-window model-value="basic">
    <v-window-item value="basic">
      <v-data-table
        :headers="mainTableHeaders"
        :items="playerDisplayData"
        item-value="playerId"
        expand-on-click
        :expanded="expanded"
        @update:expanded="expanded = $event"
        class="elevation-1"
        :row-props="getRowProps"
        show-expand
        :items-per-page="-1"
        hide-default-footer
      >
        <template v-slot:[`item.playerName`]="{ item }">
          <div class="d-flex align-center">
            <img
              v-if="fightSummary.players[item.playerId]?.talentIcon"
              :src="fightSummary.players[item.playerId].talentIcon"
              width="36"
              height="36"
              class="mr-2"
            />
            <span class="text-h6" :style="{ color: hiddenPlayers.has(item.playerId) ? 'grey' : getMabiNameColor(item.playerName) }">
              {{ hiddenPlayers.has(item.playerId) ? "Hidden" : item.playerName }}
            </span>
          </div>
        </template>

        <template v-slot:[`item.totalDamage`]="{ item }">
          <div class="text-no-wrap">
            <span class="text-white font-weight-bold text-h6">{{ formatNumber(item.totalDamage) }}</span>
            <span class="ml-1 text-white font-weight-medium" v-if="totalDamageForView > 0">
              ({{ ((item.totalDamage / totalDamageForView) * 100).toFixed(1) }}%)
            </span>
          </div>
        </template>

        <template v-slot:[`item.hpDamage`]="{ item }">
          <div class="text-no-wrap">
            <span class="text-red-lighten-1 text-h6">{{ formatNumber(item.totalDamage - (item.totalManaDamage || 0)) }}</span>
          </div>
        </template>

        <template v-slot:[`item.manaDamage`]="{ item }">
          <div class="text-no-wrap">
            <span class="text-blue-lighten-1 text-h6">{{ formatNumber(item.totalManaDamage || 0) }}</span>
          </div>
        </template>

        <template v-slot:expanded-row="{ columns, item }">
          <tr>
            <td :colspan="columns.length">
              <v-data-table
                :headers="breakdownTableHeaders"
                :items="getBreakdownDetails(item)"
                item-value="key"
                density="compact"
                class="ma-2"
                :items-per-page="-1"
                hide-default-footer
              >
                <template v-slot:[`item.skillName`]="{ item: breakdownItem }">
                  <div class="d-flex align-center">
                    <img
                      width="32"
                      height="32"
                      :src="getSkillImageSrc(breakdownItem.skillId)"
                      class="mr-2"
                      @error="
                        ($event.target as HTMLImageElement).style.display =
                          'none'
                      "
                    />
                    <span>{{ getSkillName(breakdownItem.skillId) }}</span>
                  </div>
                </template>

                <template v-slot:[`item.totalDamage`]="{ item: breakdownItem }">
                  {{ formatNumber(breakdownItem.totalDamage) }}
                </template>

                <template v-slot:[`item.hpDamage`]="{ item: breakdownItem }">
                  <span class="text-red-lighten-1">{{ formatNumber(breakdownItem.totalDamage - (breakdownItem.totalManaDamage || 0)) }}</span>
                </template>

                <template v-slot:[`item.manaDamage`]="{ item: breakdownItem }">
                  <span class="text-blue-lighten-1">{{ formatNumber(breakdownItem.totalManaDamage || 0) }}</span>
                </template>

                <template v-slot:[`item.avgDamage`]="{ item: breakdownItem }">
                  {{
                    formatNumber(
                      breakdownItem.totalDamage / breakdownItem.hitCount
                    )
                  }}
                </template>

                <template v-slot:[`item.minDamage`]="{ item: breakdownItem }">
                  {{ formatNumber(breakdownItem.minDamage) }}
                </template>

                <template v-slot:[`item.maxDamage`]="{ item: breakdownItem }">
                  {{ formatNumber(breakdownItem.maxDamage) }}
                </template>
              </v-data-table>
            </td>
          </tr>
        </template>
      </v-data-table>
    </v-window-item>
  </v-window>
</template>

<script lang="ts" setup>
import { inject, ref, computed } from "vue";
import { getMabiNameColor } from "@/util";
import { fightSummary, hiddenPlayers } from "@/store";
import { PlayerDamageTakenStats, DamageTakenDetails } from "@/protocols";

const skillNameMap = inject("skillNameMap");
const expanded = ref<string[]>([]);

const playerDisplayData = computed(() => {
  if (!fightSummary.damageTaken) return [];
  return Object.values(fightSummary.damageTaken)
    .filter((p) => p.totalDamage > 0)
    .sort((a, b) => b.totalDamage - a.totalDamage);
});

const totalDamageForView = computed(() => {
  return playerDisplayData.value.reduce((sum, p) => sum + p.totalDamage, 0);
});

const formatNumber = (num: number | undefined | null): string => {
  if (num === undefined || num === null) return "0";
  return Math.round(num).toLocaleString("en-US");
};

// REMOVED formatDuration and formattedEncounterDuration computed property

const getSkillName = (skillId: number): string => {
  return skillNameMap.value[skillId]?.name || `Unknown Skill: ${skillId}`;
};

const getSkillImageSrc = (skillId: number): string => {
  return skillNameMap.value[skillId]?.iconUrl || "";
};

const getBreakdownDetails = (
  playerData: PlayerDamageTakenStats
): any[] => {
  return Object.values(playerData.breakdown)
    .sort((a, b) => b.totalDamage - a.totalDamage)
    .map((item) => ({
      ...item,
      key: `${item.attackerName}-${item.skillId}`,
    }));
};

const getRowProps = ({ item }: { item: PlayerDamageTakenStats }) => {
  const topDamage = playerDisplayData.value[0]?.totalDamage;
  if (!topDamage || !item.totalDamage) return { style: {} };

  const hpDamage = item.totalDamage - (item.totalManaDamage || 0);
  const manaDamage = item.totalManaDamage || 0;

  const hpPercentage = (hpDamage / topDamage) * 100;
  const manaPercentage = (manaDamage / topDamage) * 100;

  // HP Color (Reddish), Mana Color (Blueish)
  const hpColor = hiddenPlayers.has(item.playerId) ? "rgba(128, 128, 128, 0.3)" : "rgba(244, 67, 54, 0.3)"; // Red 400 or Grey
  const manaColor = hiddenPlayers.has(item.playerId) ? "rgba(100, 100, 100, 0.3)" : "rgba(33, 150, 243, 0.3)"; // Blue 500 or Dark Grey

  return {
    style: {
      background: `linear-gradient(to right, ${hpColor} ${hpPercentage}%, ${manaColor} ${hpPercentage}% ${hpPercentage + manaPercentage}%, transparent ${hpPercentage + manaPercentage}%)`,
    },
  };
};

const mainTableHeaders = [
  { title: "Character", key: "playerName", sortable: true },
  { title: "Total Damage Taken", key: "totalDamage", sortable: true, align: "end", width: "20%" },
  { title: "HP Damage", key: "hpDamage", sortable: false, align: "end", width: "15%" },
  { title: "Mana Damage", key: "manaDamage", sortable: false, align: "end", width: "15%" },
  { title: "", key: "data-table-expand" },
] as const;

const breakdownTableHeaders = [
  { title: "Attacker", key: "attackerName", sortable: true },
  { title: "Skill Used", key: "skillName", sortable: true },
  { title: "Total Damage", key: "totalDamage", sortable: true },
  { title: "HP Damage", key: "hpDamage", sortable: false },
  { title: "Mana Damage", key: "manaDamage", sortable: false },
  { title: "Hit Count", key: "hitCount", sortable: true },
  { title: "Avg Hit", key: "avgDamage", sortable: false },
  { title: "Min Hit", key: "minDamage", sortable: true },
  { title: "Max Hit", key: "maxDamage", sortable: true },
];
</script>
