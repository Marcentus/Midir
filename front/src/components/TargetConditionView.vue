<template>
  <div v-if="hasConditions" class="ml-4">
    <v-dialog width="auto" min-width="800">
      <template v-slot:activator="{ props }">
        <div 
          v-bind="props" 
          class="d-flex flex-wrap ga-1 cursor-pointer bg-grey-darken-4 rounded px-2 py-1"
          style="min-height: 32px; align-items: center;"
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
        </div>
      </template>

      <template v-slot:default="{ isActive }">
        <v-card>
          <v-card-title class="d-flex justify-space-between align-center pa-2 bg-grey-darken-3">
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
                <th>Uptime</th>
                <th>Duration</th>
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
                    v-if="cond.metaBreakdown && cond.metaBreakdown.length > 0"
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
              <tr v-if="detailsOpen[cond.id]" class="bg-grey-darken-3">
                <td colspan="5" class="pa-0">
                  <v-table density="compact" class="bg-transparent text-caption pl-10">
                    <thead>
                      <tr>
                        <th>Metadata</th>
                        <th>Uptime</th>
                        <th>Duration</th>
                        <th>Attackers</th>
                      </tr>
                    </thead>
                    <tbody>
                      <tr v-for="meta in cond.metaBreakdown" :key="meta.metaData">
                        <td style="white-space: normal; word-break: break-word; max-width: 400px;">{{ meta.metaData }}</td>
                        <td>{{ meta.uptime.toFixed(0) }}%</td>
                        <td>{{ meta.duration.toFixed(1) }}s</td>
                        <td>
                          <span v-for="(attackerId, index) in meta.attackers" :key="attackerId">
                            {{ getAttackerName(attackerId) }}<span v-if="index < meta.attackers.length - 1">, </span>
                          </span>
                        </td>
                      </tr>
                    </tbody>
                  </v-table>
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
import { favoriteConditions, hiddenConditions, toggleConditionPref } from "@/store";

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

const sortedConditions = computed(() => {
  if (!props.conditions) return [];
  return Object.values(props.conditions).sort((a, b) => {
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
