<template>
  <div>
    <v-dialog width="auto" min-width="800">
      <template v-slot:activator="{ props }">
        <div 
          v-bind="props" 
          class="d-flex flex-wrap ga-1 cursor-pointer rounded px-2 py-1"
          style="min-height: 32px; align-items: center; background: rgba(var(--v-theme-surface), 0.8); border: 1px solid rgba(255,255,255,0.05);"
        >
          <div
            v-for="cond in sortedConditions.filter((c) => favoriteConditions.has(c.id))"
            :key="cond.id"
            class="d-flex flex-column align-center"
          >
            <v-tooltip location="top" :text="getConditionName(cond.id)">
              <template v-slot:activator="{ props: tooltipProps }">
                <div class="position-relative">
                  <img
                    v-bind="tooltipProps"
                    :src="getConditionIcon(cond.id)"
                    width="24"
                    height="24"
                    @error="($event.target as HTMLImageElement).style.display = 'none'"
                  />
                  <!-- Tiny Star for Favorites in Collapsed View -->
                  <v-icon
                    v-if="favoriteConditions.has(cond.id)"
                    icon="mdi-star"
                    color="yellow-darken-2"
                    size="10"
                    style="position: absolute; top: -4px; right: -2px"
                  ></v-icon>
                </div>
              </template>
            </v-tooltip>
            <span
              class="text-caption font-weight-bold"
              :class="{ 'text-yellow': favoriteConditions.has(cond.id) }"
              style="font-size: 0.65rem !important; line-height: 1;"
            >
              {{ cond.uptime.toFixed(0) }}%
            </span>
          </div>
          <span v-if="!sortedConditions.some((c) => favoriteConditions.has(c.id))" class="text-caption text-grey">
             No Favorites
          </span>
          <v-divider vertical class="mx-2" style="height: 16px; align-self: center;" />
          <v-icon icon="mdi-format-list-bulleted" size="18" class="text-grey-lighten-1" />
        </div>
      </template>

      <template v-slot:default="{ isActive }">
        <v-card>
          <v-card-title class="d-flex justify-space-between align-center pa-2" style="background: rgba(var(--v-theme-surface), 1); border-bottom: 1px solid rgba(255,255,255,0.1);">
            <span class="text-subtitle-2">Detailed Conditions</span>
            <v-btn icon variant="text" size="small" @click="isActive.value = false">
              <v-icon>mdi-close</v-icon>
            </v-btn>
          </v-card-title>
          <v-table density="compact" class="bg-transparent text-caption">
            <thead>
              <tr>
                <th width="40">Icon</th>
                <th>Name</th>
                <th width="160">Uptime</th>
                <th width="80">Duration</th>
                <th width="100" class="text-end">Actions</th>
              </tr>
            </thead>
            <tbody>
            <template v-for="cond in sortedConditions" :key="cond.id">
              <tr
                :class="{ 'text-grey-darken-1': hiddenConditions.has(cond.id) }"
              >

                <td>
                  <img
                    :src="getConditionIcon(cond.id)"
                    width="24"
                    height="24"
                    @error="($event.target as HTMLImageElement).style.display = 'none'"
                  />
                </td>
                <td>
                  {{ getConditionName(cond.id) }}
                  <span v-if="getMetadataDisplay(cond)" class="text-caption text-info ml-1">
                    {{ getMetadataDisplay(cond) }}
                  </span>
                  <span v-if="hiddenConditions.has(cond.id)" class="text-caption font-italic ml-1"
                    >(Hidden)</span
                  >
                </td>
                <td>
                  <v-progress-linear
                    :model-value="cond.uptime"
                    :color="hiddenConditions.has(cond.id) ? 'grey-darken-2' : 'info'"
                    height="15"
                  >
                    <template v-slot:default="{ value }">
                      <strong>{{ Math.ceil(value) }}%</strong>
                    </template>
                  </v-progress-linear>
                </td>
                <td>{{ cond.duration.toFixed(1) }}s</td>
                <td class="text-end">
                  <!-- Favorite Button -->
                  <v-btn
                    icon
                    variant="text"
                    size="small"
                    density="compact"
                    @click="toggleConditionPref(cond.id, 'fav')"
                  >
                    <v-icon :color="favoriteConditions.has(cond.id) ? 'yellow' : 'grey-darken-1'">
                      {{ favoriteConditions.has(cond.id) ? "mdi-star" : "mdi-star-outline" }}
                    </v-icon>
                    <v-tooltip activator="parent" location="top">Favorite</v-tooltip>
                  </v-btn>

                  <!-- Hide Button -->
                  <v-btn
                    icon
                    variant="text"
                    size="small"
                    density="compact"
                    @click="toggleConditionPref(cond.id, 'hide')"
                  >
                    <v-icon :color="hiddenConditions.has(cond.id) ? 'red-lighten-1' : 'grey-lighten-1'">
                      {{ hiddenConditions.has(cond.id) ? "mdi-eye-off" : "mdi-eye" }}
                    </v-icon>
                    <v-tooltip activator="parent" location="top">
                      {{ hiddenConditions.has(cond.id) ? "Unhide" : "Hide" }}
                    </v-tooltip>
                  </v-btn>

                  <!-- Details Button -->
                  <v-btn
                    v-if="(cond.metaBreakdown && cond.metaBreakdown.length > 0) || cond.isCombined"
                    icon
                    variant="text"
                    size="small"
                    density="compact"
                    @click="toggleDetails(cond.id)"
                  >
                    <v-icon>{{ detailsOpen[cond.id] ? 'mdi-chevron-up' : 'mdi-chevron-down' }}</v-icon>
                  </v-btn>
                </td>
              </tr>
              <tr v-if="detailsOpen[cond.id]" :style="{ background: cond.isCombined ? 'rgba(var(--v-theme-surface), 0.4)' : 'rgba(var(--v-theme-surface), 0.2)' }">
                <td colspan="5" class="pa-0">
                  <div :class="['border-s-sm py-1 ms-4', cond.isCombined ? 'border-info' : 'border-yellow']">
                    <!-- Case 1: Combined Condition -> Show Sub-conditions -->
                    <template v-if="cond.isCombined && cond.subConditions">
                      <v-table density="compact" class="bg-transparent text-caption pl-2">
                        <tbody>
                          <template v-for="sub in cond.subConditions" :key="sub.id">
                            <tr>
                              <td width="30">
                                <img
                                  :src="getConditionIcon(sub.id)"
                                  width="20"
                                  height="20"
                                  @error="($event.target as HTMLImageElement).style.display = 'none'"
                                />
                              </td>
                              <td>
                                {{ getConditionName(sub.id) }}
                                <span v-if="getMetadataDisplay(sub)" class="text-caption text-info ml-1">
                                  {{ getMetadataDisplay(sub) }}
                                </span>
                              </td>
                              <td class="text-end">
                                <!-- ADJUST OFFSET HERE: Increase padding-right to move metrics more to the left -->
                                <div class="d-flex align-center justify-end ga-2" style="padding-right: 60px">
                                  <v-progress-linear
                                    :model-value="sub.uptime"
                                    color="info"
                                    height="8"
                                    style="width: 80px"
                                  >
                                    <template v-slot:default="{ value }">
                                      <strong style="font-size: 0.65rem">{{ Math.ceil(value) }}%</strong>
                                    </template>
                                  </v-progress-linear>
                                  <span style="min-width: 45px">{{ sub.duration.toFixed(1) }}s</span>
                                </div>
                              </td>
                              <td width="40" class="text-end">
                                <!-- Details Button for Sub-condition -->
                                <v-btn
                                  v-if="sub.metaBreakdown && sub.metaBreakdown.length > 0"
                                  icon
                                  variant="text"
                                  size="small"
                                  density="compact"
                                  @click="toggleDetails(sub.id)"
                                >
                                  <v-icon>{{ detailsOpen[sub.id] ? 'mdi-chevron-up' : 'mdi-chevron-down' }}</v-icon>
                                </v-btn>
                              </td>
                            </tr>
                            <!-- Nested Metadata Row for Sub-condition -->
                            <tr v-if="detailsOpen[sub.id]" style="background: rgba(var(--v-theme-surface), 0.3)">
                              <td colspan="4" class="pa-0">
                                 <div class="border-s-sm border-yellow ms-8 py-1">
                                   <v-table density="compact" class="bg-transparent text-caption pl-2">
                                      <tbody>
                                        <tr v-for="meta in sub.metaBreakdown" :key="meta.metaData">
                                          <td style="white-space: normal; word-break: break-word; max-width: 400px;">{{ meta.metaData }}</td>
                                          <td class="text-end">
                                            <!-- ADJUST OFFSET HERE: padding-right moves metrics left -->
                                            <div class="d-flex align-center justify-end ga-2" style="padding-right: 40px">
                                               <v-progress-linear
                                                :model-value="meta.uptime"
                                                color="info"
                                                height="6"
                                                style="width: 60px"
                                              ></v-progress-linear>
                                              <span style="min-width: 45px">{{ meta.duration.toFixed(1) }}s</span>
                                            </div>
                                          </td>
                                          <td class="text-grey">
                                            <span v-for="(attackerId, index) in meta.attackers" :key="attackerId">
                                              {{ getAttackerName(attackerId) }}<span v-if="index < meta.attackers.length - 1">, </span>
                                            </span>
                                          </td>
                                        </tr>
                                      </tbody>
                                    </v-table>
                                 </div>
                              </td>
                            </tr>
                          </template>
                        </tbody>
                      </v-table>
                    </template>

                    <!-- Case 2: Normal Condition -> Show Metadata Breakdown -->
                    <v-table v-else density="compact" class="bg-transparent text-caption pl-2">
                      <tbody>
                        <tr v-for="meta in cond.metaBreakdown" :key="meta.metaData">
                          <td style="white-space: normal; word-break: break-word; max-width: 400px;">{{ meta.metaData }}</td>
                          <td class="text-end">
                            <div class="d-flex align-center justify-end ga-2">
                               <v-progress-linear
                                :model-value="meta.uptime"
                                color="info"
                                height="6"
                                style="width: 60px"
                              ></v-progress-linear>
                              <span style="min-width: 45px">{{ meta.duration.toFixed(1) }}s</span>
                            </div>
                          </td>
                          <td class="text-grey">
                            <span v-for="(attackerId, index) in meta.attackers" :key="attackerId">
                              {{ getAttackerName(attackerId) }}<span v-if="index < meta.attackers.length - 1">, </span>
                            </span>
                          </td>
                        </tr>
                      </tbody>
                    </v-table>
                  </div>
                </td>
              </tr>
            </template>
            </tbody>
          </v-table>
        </v-card>
      </template>
    </v-dialog>
  </div>
</template>

<script lang="ts" setup>
import { ref, computed, inject } from "vue";
import { ConditionStats } from "@/protocols";
import { favoriteConditions, hiddenConditions, toggleConditionPref, fightSummary } from "@/store";
import { processConditions, ExtendedConditionStats } from "@/conditionCombinations";
import { parseMabinogiMetadata } from "@/utils/metadata";

const props = defineProps<{
  conditions: { [id: number]: ConditionStats } | undefined;
  attackerNameMap?: { [id: string]: string };
}>();

const condNameMap = inject("condNameMap") as any;
const detailsOpen = ref<Record<number, boolean>>({});

const toggleDetails = (id: number) => {
  detailsOpen.value[id] = !detailsOpen.value[id];
};

const getAttackerName = (id: number): string => {
  if (!props.attackerNameMap) return id.toString();
  return props.attackerNameMap[id.toString()] || id.toString();
};

const hasConditions = computed(() => {
  return props.conditions && Object.keys(props.conditions).length > 0;
});

const getConditionName = (id: number): string => {
  return condNameMap.value[id]?.name || `Unknown Condition ${id}`;
};

const getConditionIcon = (id: number): string => {
  return condNameMap.value[id]?.iconUrl || "";
};

const getMetadataDisplay = (cond: ExtendedConditionStats): string => {
  if (cond.id !== 192 && cond.id !== 680) return "";
  if (!cond.metaBreakdown || cond.metaBreakdown.length === 0) return "";

  // Find the entry with the highest uptime
  const bestMeta = [...cond.metaBreakdown].sort((a, b) => b.uptime - a.uptime)[0];
  const parsed = parseMabinogiMetadata(bestMeta.metaData);

  const parts: string[] = [];
  if (cond.id === 680) {
    if (parsed.MCMBAMAX !== undefined) {
      parts.push(`Max Att: ${parsed.MCMBAMAX.toFixed(2)}%`);
    }
  } else if (cond.id === 192) {
    if (parsed.LSMA !== undefined) {
      parts.push(`Magic Att: ${parsed.LSMA.toFixed(2)}%`);
    }
    if (parsed.MFCP !== undefined) {
      parts.push(`Cast Speed: ${parsed.MFCP.toFixed(2)}%`);
    }
  }

  if (parts.length === 0) return "";
  return `(${parts.join(", ")})`;
};

const sortedConditions = computed(() => {
  if (!props.conditions) return [];
  const processed = processConditions(props.conditions, fightSummary.encounterDuration);
  
  return processed.sort((a, b) => {
    // 1. Priority: Favorites
    const aFav = favoriteConditions.has(a.id);
    const bFav = favoriteConditions.has(b.id);
    if (aFav && !bFav) return -1;
    if (!aFav && bFav) return 1;

    // 2. Priority: Hidden (pushed to bottom)
    const aHide = hiddenConditions.has(a.id);
    const bHide = hiddenConditions.has(b.id);
    if (aHide && !bHide) return 1;
    if (!aHide && bHide) return -1;

    // 3. Priority: ID (Ascending)
    return a.id - b.id;
  });
});
</script>
