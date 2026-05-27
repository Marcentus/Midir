<template>


  <v-data-table
    v-if="viewMode === 'table'"
    :headers="mainTableHeaders"
    :items="playerDisplayData"
    item-value="id"
    expand-on-click
    :expanded="expanded"
    @update:expanded="expanded = $event"
    class="elevation-1 pb-16 dps-table"
    :row-props="getRowProps"
    show-expand
    density="compact"
    :items-per-page="-1"
    hide-default-footer
  >
    <template v-slot:header.isHidden>
      <v-tooltip location="top" :text="globalHideMode ? 'Global Hide Active' : (areAllHidden ? 'Session Hide Active' : 'Hide All')">
        <template v-slot:activator="{ props }">
          <v-btn icon variant="text" density="compact" @click="toggleAllPlayersVisibility" v-bind="props">
            <v-icon
              :icon="globalHideMode || areAllHidden ? 'mdi-eye-off' : 'mdi-eye'"
              :color="globalHideMode ? 'error' : ''"
            ></v-icon>
          </v-btn>
        </template>
      </v-tooltip>
    </template>

    <template v-slot:item.isHidden="{ item }">
        <v-btn icon variant="text" density="compact" @click.stop="toggleHiddenPlayer(item.id)">
            <v-icon :icon="(globalHideMode || hiddenPlayers.has(item.id)) ? 'mdi-eye-off' : 'mdi-eye'"
                    :color="(globalHideMode || hiddenPlayers.has(item.id)) ? 'grey' : ''"></v-icon>
        </v-btn>
    </template>

    <template v-slot:[`item.name`]="{ item }">
      <div class="d-flex align-center">
        <img
          v-if="item.talentIcon"
          :src="item.talentIcon"
          width="28"
          height="28"
          class="mr-2"
        />
        <span class="text-h6" style="color: white">
          {{ (globalHideMode || hiddenPlayers.has(item.id)) ? (item.talentName || 'Hidden') : item.name }}
        </span>
        <!-- Warning Icon for Missing Appear Packet -->
        <v-tooltip
          v-if="item.missingAppearPacket"
          location="top"
          text="Entity appearance not recorded. Condition uptime may be inaccurate."
        >
          <template v-slot:activator="{ props }">
            <v-icon
              v-bind="props"
              icon="mdi-alert-circle-outline"
              color="warning"
              size="small"
              class="ml-2"
            ></v-icon>
          </template>
        </v-tooltip>
      </div>
    </template>

    <template v-slot:[`item.dps`]="{ item }">
      <span class="font-weight-bold text-h6">{{ formatNumber(item.dps) }}</span>
    </template>

    <template v-slot:[`item.totalDamage`]="{ item }">
      <div v-if="dpsMeterFillMode === 'column'" class="damage-bar-cell">
        <span class="damage-bar-pct">
          {{ totalDamageForView > 0 ? ((item.totalDamage / totalDamageForView) * 100).toFixed(1) : '0.0' }}%
        </span>
        <div class="damage-bar-wrapper">
          <div
            class="damage-bar-fill"
            :style="{
              width: (playerDisplayData[0]?.totalDamage ? (item.totalDamage / playerDisplayData[0].totalDamage) * 100 : 0) + '%',
              backgroundColor: getPlayerColor(item)
            }"
          ></div>
        </div>
        <span class="damage-bar-val">
          {{ formatAbbreviated(item.totalDamage) }}
        </span>
      </div>
      <div v-else class="text-no-wrap">
        <span class="font-weight-bold text-h6">{{ formatAbbreviated(item.totalDamage) }}</span>
        <span class="ml-1 text-white font-weight-medium" v-if="totalDamageForView > 0">
          ({{ ((item.totalDamage / totalDamageForView) * 100).toFixed(1) }}%)
        </span>
      </div>
    </template>

    <template v-slot:[`item.critRate`]="{ item }">
      <div class="text-no-wrap">
        <span class="text-amber-lighten-1 font-weight-bold text-h6">{{ item.critRate.toFixed(1) }}%</span>
      </div>
    </template>

    <template v-slot:expanded-row="{ columns, item }">
      <tr>
        <td :colspan="columns.length">
          <!-- Condition Uptime Section -->
           <div class="d-flex align-center pa-2 ma-2" v-if="item.conditions && Object.keys(item.conditions).length > 0">

             <TargetConditionView
                :conditions="item.conditions"
                :attackerNameMap="attackerNameMap"
                class="ml-0"
              />
           </div>

          <!-- Skill Breakdown Table -->
          <v-data-table
            :row-props="
              ({ item: skillItem, index }) =>
                getSkillRowProps(
                  skillItem,
                  getSkillBreakdown(item)[0].totalDamage,
                  item.name,
                  item,
                  index
                )
            "
            :headers="computedSkillTableHeaders"
            :items="getSkillBreakdown(item)"
            item-value="id"
            density="compact"
            class="ma-2"
            :items-per-page="-1"
            hide-default-footer
          >
            <template v-slot:[`item.skillName`]="{ item: skillItem }">
              <div class="d-flex align-center">
                <img
                  width="24"
                  height="24"
                  :src="getSkillImageSrc(skillItem.id)"
                  class="mr-2"
                  @error="
                    ($event.target as HTMLImageElement).style.display =
                      'none'
                  "
                />
                <span>{{ getSkillName(skillItem.id) }}</span>
              </div>
            </template>

            <template v-slot:[`item.uses`]="{ item: skillItem }">
              {{ formatNumber(skillItem.uses) }}
            </template>

            <template v-slot:[`item.totalDamage`]="{ item: skillItem }">
              <div v-if="dpsMeterFillMode === 'column'" class="damage-bar-cell skill-bar-cell">
                <span class="damage-bar-pct">
                  {{ item.totalDamage > 0 ? ((skillItem.totalDamage / item.totalDamage) * 100).toFixed(1) : '0.0' }}%
                </span>
                <div class="damage-bar-wrapper">
                  <div
                    class="damage-bar-fill"
                    :style="{
                      width: (getSkillBreakdown(item)[0]?.totalDamage ? (skillItem.totalDamage / getSkillBreakdown(item)[0].totalDamage) * 100 : 0) + '%',
                      backgroundColor: getPlayerColor(item)
                    }"
                  ></div>
                </div>
                <span class="damage-bar-val">
                  {{ formatNumber(skillItem.totalDamage) }}
                </span>
              </div>
              <div v-else class="text-no-wrap">
                <span>{{ formatNumber(skillItem.totalDamage) }}</span>
                <span
                  class="ml-2 text-white font-weight-medium"
                  v-if="item.totalDamage > 0"
                >
                  ({{
                    (
                      (skillItem.totalDamage / item.totalDamage) *
                      100
                    ).toFixed(1)
                  }}%)
                </span>
              </div>
            </template>

            <template v-slot:[`item.dps`]="{ item: skillItem }">
              {{
                currentEncounterDuration > 0
                  ? formatNumber(skillItem.totalDamage / currentEncounterDuration)
                  : "0"
              }}
            </template>
            
            <template v-slot:[`item.hpm`]="{ item: skillItem }">
              {{
                currentEncounterDuration > 0
                  ? (skillItem.count / (currentEncounterDuration / 60)).toFixed(1)
                  : "0.0"
              }}/m
            </template>

            <template v-slot:[`item.avgNonCrit`]="{ item: skillItem }">
              {{ formatNumber((skillItem.count - skillItem.critCount) > 0 ? skillItem.totalDamageNonCrit / (skillItem.count - skillItem.critCount) : 0) }}
            </template>

            <template v-slot:[`item.maxDamageNonCrit`]="{ item: skillItem }">
              {{ formatNumber(skillItem.maxDamageNonCrit) }}
            </template>

            <template v-slot:[`item.avgDamage`]="{ item: skillItem }">
              {{ formatNumber(skillItem.count > 0 ? skillItem.totalDamage / skillItem.count : 0) }}
            </template>

            <template v-slot:[`item.avgCrit`]="{ item: skillItem }">
              {{ formatNumber(skillItem.critCount > 0 ? skillItem.totalDamageCrit / skillItem.critCount : 0) }}
            </template>

            <template v-slot:[`item.maxDamageCrit`]="{ item: skillItem }">
              {{ formatNumber(skillItem.maxDamageCrit) }}
            </template>

            <template v-slot:[`item.maxDamage`]="{ item: skillItem }">
              {{ formatNumber(skillItem.maxDamage) }}
            </template>

            <template v-slot:[`item.critRate`]="{ item: skillItem }">
              {{
                skillItem.count > 0
                  ? (
                      (skillItem.critCount / skillItem.count) *
                      100
                    ).toFixed(1)
                  : "0.0"
              }}%
            </template>
          </v-data-table>
        </td>
      </tr>
    </template>
  </v-data-table>

  <v-container v-else-if="viewMode === 'cards'" fluid class="pb-16">
    <div :style="{ display: 'grid', gridTemplateColumns: `repeat(auto-fill, minmax(${cardGridMinWidth}px, 1fr))`, gap: '16px' }">
      <div
        v-for="item in playerDisplayData"
        :key="item.id"
      >
        <v-card class="h-100" elevation="2">
          <v-card-title class="d-flex align-center">
            <!-- Hide individual button -->
            <v-btn icon variant="text" density="compact" size="small" class="mr-1" @click.stop="toggleHiddenPlayer(item.id)">
                <v-icon :icon="(globalHideMode || hiddenPlayers.has(item.id)) ? 'mdi-eye-off' : 'mdi-eye'"
                        :color="(globalHideMode || hiddenPlayers.has(item.id)) ? 'grey' : ''"
                        size="small"></v-icon>
            </v-btn>
            
            <img
              v-if="item.talentIcon"
              :src="item.talentIcon"
              width="32"
              height="32"
              class="mr-2"
            />
            <span class="text-subtitle-1 font-weight-bold text-truncate" style="color: white">
              {{ (globalHideMode || hiddenPlayers.has(item.id)) ? (item.talentName || 'Hidden') : item.name }}
            </span>
             
             <!-- Warning Icon -->
            <v-tooltip
              v-if="item.missingAppearPacket"
              location="top"
              text="Entity appearance not recorded. Condition uptime may be inaccurate."
            >
              <template v-slot:activator="{ props }">
                <v-icon
                  v-bind="props"
                  icon="mdi-alert-circle-outline"
                  color="warning"
                  size="small"
                  class="ml-2"
                ></v-icon>
              </template>
            </v-tooltip>

            <v-spacer></v-spacer>
            <div class="text-right">
                <div class="text-h5 font-weight-bold mb-n1">{{ formatNumber(item.dps) }} DPS</div>
                <div class="text-subtitle-2 text-white" style="opacity: 0.9;">
                    {{ formatAbbreviated(item.totalDamage) }}
                    <span v-if="totalDamageForView > 0" class="text-white">
                        ({{ ((item.totalDamage / totalDamageForView) * 100).toFixed(1) }}%)
                    </span>
                </div>
            </div>
          </v-card-title>
          
          <v-divider></v-divider>

          <v-card-text>
              <!-- Conditions -->
              <div class="mb-4 d-flex justify-center" style="min-height: 48px;">
                  <TargetConditionView
                    v-if="item.conditions && Object.keys(item.conditions).length > 0"
                    :conditions="item.conditions"
                    :attackerNameMap="attackerNameMap"
                    class="ml-0"
                  />
                   <div v-else class="text-caption text-grey text-center pt-2 d-flex align-center">
                       <v-icon size="small" class="mr-1">mdi-minus-circle-outline</v-icon> No Conditions Applied
                   </div>
              </div>

              <!-- Skill Table -->
              <div style="max-height: 500px; overflow-y: auto; overflow-x: auto;">
                <v-data-table
                    :headers="computedCardSkillTableHeaders"
                    :items="getSkillBreakdown(item)"
                    density="compact"
                    item-value="id"
                    :items-per-page="-1"
                    hide-default-footer
                    class="skill-card-table"
                    :row-props="
                        ({ item: skillItem, index }) =>
                            getSkillRowProps(
                            skillItem,
                            getSkillBreakdown(item)[0].totalDamage,
                            item.name,
                            item,
                            index
                            )
                    "
                >
                    <!-- Icon + Tooltip (No Name Text) -->
                    <template v-slot:item.skillIcon="{ item: skillItem }">
                         <v-tooltip location="top" open-delay="200">
                             <template v-slot:activator="{ props }">
                                 <img
                                    v-bind="props"
                                    width="28"
                                    height="28"
                                    :src="getSkillImageSrc(skillItem.id)"
                                    style="vertical-align: middle; border-radius: 4px;"
                                    @error="($event.target as HTMLImageElement).style.display = 'none'"
                                  />
                             </template>
                             <span>{{ getSkillName(skillItem.id) }}</span>
                         </v-tooltip>
                    </template>

                    <template v-slot:item.uses="{ item: skillItem }">
                         {{ formatNumber(skillItem.uses) }}
                    </template>

                    <!-- Stats columns -->
                    <template v-slot:item.totalDamage="{ item: skillItem }">
                        <div class="text-no-wrap">
                            <span class="font-weight-bold">{{ formatAbbreviated(skillItem.totalDamage) }}</span>
                            <span class="text-caption text-white ml-1" v-if="item.totalDamage > 0">
                                ({{ ((skillItem.totalDamage / item.totalDamage) * 100).toFixed(1) }}%)
                            </span>
                        </div>
                    </template>

                    <template v-slot:item.dps="{ item: skillItem }">
                         {{
                            currentEncounterDuration > 0
                            ? formatAbbreviated(skillItem.totalDamage / currentEncounterDuration)
                            : "0"
                        }}
                    </template>

                    <template v-slot:item.hpm="{ item: skillItem }">
                        <div class="text-no-wrap">
                        {{
                            currentEncounterDuration > 0
                            ? (skillItem.count / (currentEncounterDuration / 60)).toFixed(1)
                            : "0.0"
                        }}/m
                        </div>
                    </template>

                    <template v-slot:item.avgNonCrit="{ item: skillItem }">
                        {{ formatAbbreviated((skillItem.count - skillItem.critCount) > 0 ? skillItem.totalDamageNonCrit / (skillItem.count - skillItem.critCount) : 0) }}
                    </template>

                    <template v-slot:item.maxDamageNonCrit="{ item: skillItem }">
                         {{ formatAbbreviated(skillItem.maxDamageNonCrit) }}
                    </template>

                    <template v-slot:item.avgDamage="{ item: skillItem }">
                        {{ formatAbbreviated(skillItem.count > 0 ? skillItem.totalDamage / skillItem.count : 0) }}
                    </template>

                    <template v-slot:item.avgCrit="{ item: skillItem }">
                        {{ formatAbbreviated(skillItem.critCount > 0 ? skillItem.totalDamageCrit / skillItem.critCount : 0) }}
                    </template>

                    <template v-slot:item.maxDamageCrit="{ item: skillItem }">
                         {{ formatAbbreviated(skillItem.maxDamageCrit) }}
                    </template>

                    <template v-slot:item.maxDamage="{ item: skillItem }">
                         {{ formatAbbreviated(skillItem.maxDamage) }}
                    </template>

                     <template v-slot:item.critRate="{ item: skillItem }">
                      {{
                        skillItem.count > 0
                          ? (
                              (skillItem.critCount / skillItem.count) *
                              100
                            ).toFixed(0)
                          : "0"
                      }}%
                    </template>

                </v-data-table>
              </div>
          </v-card-text>
        </v-card>
      </div>
    </div>
  </v-container>

  <!-- Floating View Controls -->
  <div class="view-controls-fab">
    <div class="fab-blur-bg"></div>
    <div class="fab-content">
      <!-- Hide All Button (Card Mode) -->
      <v-tooltip v-if="viewMode === 'cards'" location="top" :text="globalHideMode ? 'Global Hide Active' : (areAllHidden ? 'Session Hide Active' : 'Hide All')">
        <template v-slot:activator="{ props }">
          <v-btn icon variant="text" density="compact" @click="toggleAllPlayersVisibility" v-bind="props" class="mr-1">
            <v-icon
              :icon="globalHideMode || areAllHidden ? 'mdi-eye-off' : 'mdi-eye'"
              :color="globalHideMode ? 'error' : ''"
              size="small"
            ></v-icon>
          </v-btn>
        </template>
      </v-tooltip>

      <v-btn-toggle
        v-model="viewMode"
        mandatory
        density="compact"
        color="primary"
        variant="text"
        class="view-toggle-group"
      >
        <v-btn value="table" prepend-icon="mdi-table" class="rounded-pill px-3">List</v-btn>
        <v-btn value="cards" prepend-icon="mdi-chart-pie" class="rounded-pill px-3">Cards</v-btn>
      </v-btn-toggle>

      <!-- Columns Menu (Card Mode) -->
      <v-menu v-if="viewMode === 'cards'" :close-on-content-click="false" location="top end" offset="12">
        <template v-slot:activator="{ props }">
          <v-btn icon="mdi-view-column" variant="text" density="compact" v-bind="props" class="ml-1">
          </v-btn>
        </template>
        <v-list density="compact" max-width="300" class="glass-menu">
          <v-list-item v-for="(metric, index) in activeMetrics" :key="metric.key">
            <div class="d-flex align-center w-100">
              <v-checkbox-btn
                v-model="metric.visible"
                :label="ALL_SKILL_METRICS_MAP[metric.key]?.title || metric.key"
                color="primary"
                density="compact"
                hide-details
                class="flex-grow-1 mr-2"
              ></v-checkbox-btn>
              <v-btn icon="mdi-chevron-up" variant="text" density="compact" size="small" :disabled="index === 0" @click="moveMetricUp(index)"></v-btn>
              <v-btn icon="mdi-chevron-down" variant="text" density="compact" size="small" :disabled="index === activeMetrics.length - 1" @click="moveMetricDown(index)"></v-btn>
            </div>
          </v-list-item>
        </v-list>
      </v-menu>
    </div>
  </div>
</template>

<style scoped>
.view-controls-fab {
  position: fixed;
  bottom: 24px;
  right: 24px;
  z-index: 1000;
  display: flex;
  align-items: center;
}

.dps-table {
  /* Border handled by parent container */
}

.dps-table :deep(thead tr) {
  background-color: transparent !important;
}

.dps-table :deep(th) {
  color: #ffffff !important;
  font-weight: 700 !important;
  text-transform: uppercase !important;
  font-size: 0.95rem !important;
  letter-spacing: 0.05em !important;
  border-bottom: 1px solid rgba(255, 255, 255, 0.1) !important;
  height: 48px !important;
  position: relative;
}

.dps-table :deep(.v-data-table-header__content) {
  justify-content: center !important;
  width: 100%;
}

.dps-table :deep(th:nth-child(2) .v-data-table-header__content) {
  justify-content: flex-start !important;
}

/* Absolute position sort icon to keep text centered */
.dps-table :deep(.v-data-table-header__icon) {
  position: absolute !important;
  right: 8px !important;
  opacity: 0.5;
}

.fab-blur-bg {
  position: absolute;
  inset: 0;
  background: rgba(var(--v-theme-surface), 0.5);
  backdrop-filter: blur(12px) saturate(160%);
  border-radius: 28px;
  border: 1px solid rgba(255, 255, 255, 0.1);
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.4);
}

.fab-content {
  position: relative;
  display: flex;
  align-items: center;
  padding: 6px 12px;
  gap: 4px;
}

.view-toggle-group {
  background: transparent !important;
  border: none !important;
}

:deep(.view-toggle-group .v-btn) {
  border: none !important;
  text-transform: none;
  font-weight: 600;
  letter-spacing: 0.5px;
  opacity: 0.7;
  transition: all 0.2s ease;
}

:deep(.view-toggle-group .v-btn--active) {
  opacity: 1;
  background: rgba(var(--v-theme-primary), 0.15) !important;
  color: rgb(var(--v-theme-primary)) !important;
}

.glass-menu {
  background: rgba(var(--v-theme-surface), 0.85) !important;
  backdrop-filter: blur(16px);
  border: 1px solid rgba(255, 255, 255, 0.1);
  box-shadow: 0 10px 40px rgba(0, 0, 0, 0.5);
  border-radius: 12px !important;
}

/* Ensure data table and container have space at the bottom */
:deep(.v-data-table) {
  background: transparent !important;
}

.pb-16 {
  padding-bottom: 80px !important;
}

/* Modern Table Styling with Gaps and Rounded Bars */
.dps-table :deep(table) {
  border-spacing: 0 6px !important;
  border-collapse: separate !important;
}

.dps-table :deep(tbody tr.character-row) {
  position: relative;
  transition: all 0.2s cubic-bezier(0.4, 0, 0.2, 1);
}

.dps-table :deep(tbody tr.character-row:hover) {
  filter: brightness(1.1);
  transform: translateY(-1px);
  z-index: 1;
}

/* Explicitly disable hover effects for ANY other row (like skill breakdown rows or expansion rows) */
.dps-table :deep(tbody tr:not(.character-row):hover) {
  filter: none !important;
  transform: none !important;
}

.dps-table :deep(td) {
  border-bottom: none !important;
  padding-top: 4px !important;
  padding-bottom: 4px !important;
  background-color: rgba(255, 255, 255, 0.03); /* Subtle row background */
}

/* Bar Rounding - Applying to TR and its TD children */
.dps-table :deep(tbody tr) {
  border-radius: 6px !important;
}

.dps-table :deep(tbody tr td:first-child) {
  border-top-left-radius: 6px !important;
  border-bottom-left-radius: 6px !important;
}

.dps-table :deep(tbody tr td:last-child) {
  border-top-right-radius: 6px !important;
  border-bottom-right-radius: 6px !important;
}

/* Ensure the expanded detail row doesn't have the same styling and never hovers */
.dps-table :deep(tr.v-data-table-expanded__content) {
  background: transparent !important;
  filter: none !important;
  transform: none !important;
}

.dps-table :deep(tr.v-data-table-expanded__content:hover) {
  filter: none !important;
  transform: none !important;
}

.dps-table :deep(tr.v-data-table-expanded__content td) {
  background: transparent !important;
  border-radius: 0 !important;
  padding: 0 !important;
}

/* Skill Breakdown table nested inside also gets rounding but smaller gaps */
.dps-table :deep(.v-data-table.ma-2 table) {
  border-spacing: 0 2px !important;
}

.dps-table :deep(.v-data-table.ma-2 tr td:first-child) {
  border-top-left-radius: 4px !important;
  border-bottom-left-radius: 4px !important;
}

.dps-table :deep(.v-data-table.ma-2 tr td:last-child) {
  border-top-right-radius: 4px !important;
  border-bottom-right-radius: 4px !important;
}

.damage-bar-cell {
  display: flex;
  align-items: center;
  gap: 12px;
  width: 100%;
}
.damage-bar-pct {
  width: 50px;
  text-align: right;
  font-weight: 600;
  font-size: 0.9rem;
  color: rgba(255, 255, 255, 0.9);
  flex-shrink: 0;
  font-variant-numeric: tabular-nums;
}
.damage-bar-wrapper {
  flex-grow: 1;
  height: 14px;
  background: rgba(255, 255, 255, 0.08);
  border-radius: 9999px;
  overflow: hidden;
  position: relative;
  border: 1px solid rgba(255, 255, 255, 0.05);
}
.damage-bar-fill {
  height: 100%;
  border-radius: 9999px;
  transition: width 0.6s cubic-bezier(0.4, 0, 0.2, 1);
  position: relative;
}

.damage-bar-val {
  width: 80px;
  text-align: left;
  font-weight: 600;
  font-size: 0.9rem;
  color: rgba(255, 255, 255, 0.9);
  flex-shrink: 0;
}
.skill-bar-cell .damage-bar-val {
  width: 95px;
}
</style>

<script lang="ts" setup>
import { inject, ref, computed, watch } from "vue";
import { getMabiNameColor } from "@/util";
import {
  fightSummary,
  activeSessionId,
  selectedTargetId,
} from "@/store";
import { SkillStats, DamageBreakdown } from "@/protocols";
import TargetConditionView from "./TargetConditionView.vue";
import { hiddenPlayers, toggleHiddenPlayer, setAllHiddenPlayers, globalHideMode, showClassColorsForVisiblePlayers, listSkillMetrics, cardSkillMetrics, dpsMeterFillMode } from "@/store";

const props = defineProps<{
  attackerNameMap?: { [id: string]: string };
}>();

const getPlayerColor = (item: any) => {
  const isHidden = globalHideMode.value || hiddenPlayers.has(item.id);
  const shouldUseClassColor = isHidden || showClassColorsForVisiblePlayers.value;
  return shouldUseClassColor ? (item.talentColor || 'hsl(0, 0%, 50%)') : getMabiNameColor(item.name);
};

// --- STATE ---
const skillNameMap = inject("skillNameMap") as any;
const condNameMap = inject("condNameMap") as any;
const expanded = ref<string[]>([]);
const viewMode = ref<"table" | "cards">("table"); 

watch(activeSessionId, () => {
  expanded.value = [];
});

const ALL_SKILL_METRICS_MAP: Record<string, { title: string, cardTitle: string }> = {
  "uses": { title: "Uses", cardTitle: "Uses" },
  "dps": { title: "DPS", cardTitle: "DPS" },
  "totalDamage": { title: "Total Damage", cardTitle: "Total" },
  "count": { title: "Count", cardTitle: "Hits" },
  "hpm": { title: "HPM", cardTitle: "HPM" },
  "critRate": { title: "Crit Rate", cardTitle: "Crit %" },
  "avgDamage": { title: "Avg Dmg", cardTitle: "Avg Dmg" },
  "maxDamage": { title: "Max Dmg", cardTitle: "Max" },
  "avgNonCrit": { title: "Avg NonCrit", cardTitle: "Avg NC" },
  "maxDamageNonCrit": { title: "Max NonCrit", cardTitle: "Max NC" },
  "avgCrit": { title: "Avg Crit", cardTitle: "Avg Cr" },
  "maxDamageCrit": { title: "Max Crit", cardTitle: "Max Cr" }
};

const activeMetrics = computed(() => viewMode.value === 'table' ? listSkillMetrics.value : cardSkillMetrics.value);

const moveMetricUp = (index: number) => {
    if (index <= 0) return;
    const arr = activeMetrics.value;
    const temp = arr[index];
    arr[index] = arr[index - 1];
    arr[index - 1] = temp;
};

const moveMetricDown = (index: number) => {
    const arr = activeMetrics.value;
    if (index >= arr.length - 1) return;
    const temp = arr[index];
    arr[index] = arr[index + 1];
    arr[index + 1] = temp;
};

const computedSkillTableHeaders = computed(() => {
  if (dpsMeterFillMode.value === "column") {
    return [
      { title: "Skill", key: "skillName", sortable: true, cellProps: { class: "border-e" }, width: "200px" },
      
      // Totals Group
      { title: "Damage Contribution", key: "totalDamage", sortable: true, width: "350px" },
      { title: "DPS", key: "dps", sortable: true, cellProps: { class: "border-e" } },
      
      // Frequency Group
      { title: "Crit %", key: "critRate", sortable: true },
      { title: "Hits", key: "count", sortable: true },
      { title: "HPM", key: "hpm", sortable: true, cellProps: { class: "border-e" } },

      // Damage Metrics Group
      { title: "NonCrit Avg", key: "avgNonCrit", sortable: true },
      { title: "NonCrit Max", key: "maxDamageNonCrit", sortable: true },
      { title: "Avg Dmg", key: "avgDamage", sortable: true },
      { title: "Crit Avg", key: "avgCrit", sortable: true },
      { title: "Crit Max", key: "maxDamageCrit", sortable: true },
      { title: "Max Dmg", key: "maxDamage", sortable: true }
    ];
  } else {
    return [
      { title: "Skill", key: "skillName", sortable: true, cellProps: { class: "border-e" } },
      
      // Totals Group
      { title: "Total Dmg", key: "totalDamage", sortable: true },
      { title: "DPS", key: "dps", sortable: true, cellProps: { class: "border-e" } },
      
      // Frequency Group
      { title: "Crit %", key: "critRate", sortable: true },
      { title: "Hits", key: "count", sortable: true },
      { title: "HPM", key: "hpm", sortable: true, cellProps: { class: "border-e" } },

      // Damage Metrics Group
      { title: "NonCrit Avg", key: "avgNonCrit", sortable: true },
      { title: "NonCrit Max", key: "maxDamageNonCrit", sortable: true },
      { title: "Avg Dmg", key: "avgDamage", sortable: true },
      { title: "Crit Avg", key: "avgCrit", sortable: true },
      { title: "Crit Max", key: "maxDamageCrit", sortable: true },
      { title: "Max Dmg", key: "maxDamage", sortable: true }
    ];
  }
});

const mainTableHeaders = computed(() => {
  if (dpsMeterFillMode.value === "column") {
    return [
      { title: "Hide", key: "isHidden", sortable: false, width: "1%" },
      { title: "Character", key: "name", sortable: true, width: "20%" },
      { title: "Damage Contribution", key: "totalDamage", sortable: true, align: "center", width: "45%" },
      { title: "DPS", key: "dps", sortable: true, align: "center", width: "15%" },
      { title: "Crit %", key: "critRate", sortable: true, align: "center", width: "15%" },
      { title: "", key: "data-table-expand", width: "4%" },
    ] as const;
  } else {
    return [
      { title: "Hide", key: "isHidden", sortable: false, width: "1%" },
      { title: "Character", key: "name", sortable: true },
      { title: "DPS", key: "dps", sortable: true, align: "center", width: "15%" },
      { title: "Total Damage", key: "totalDamage", sortable: true, align: "center", width: "20%" },
      { title: "Crit %", key: "critRate", sortable: true, align: "center", width: "15%" },
      { title: "", key: "data-table-expand" },
    ] as const;
  }
});

const computedCardSkillTableHeaders = computed(() => {
  const base = [{ title: "", key: "skillIcon", sortable: false, width: "40px" }];
  const selected = cardSkillMetrics.value.filter(m => m.visible && ALL_SKILL_METRICS_MAP[m.key]).map(m => ({
    title: ALL_SKILL_METRICS_MAP[m.key].cardTitle,
    key: m.key,
    sortable: true as const,
    align: "end" as const
  }));
  return [...base, ...selected];
});

const cardGridMinWidth = computed(() => {
  // Base width (icon, name, dps) + (num columns * approx width per column)
  return Math.max(300, 200 + (computedCardSkillTableHeaders.value.length - 1) * 75);
});

// --- HELPERS ---
const formatNumber = (num: number | undefined | null): string => {
  if (num === undefined || num === null) return "0";
  return Math.round(num).toLocaleString("en-US");
};

const formatAbbreviated = (num: number | undefined | null): string => {
  if (num === undefined || num === null) return "0";
  if (num >= 1_000_000) {
    return (num / 1_000_000).toFixed(1) + "M";
  }
  if (num >= 1_000) {
    return (num / 1_000).toFixed(0) + "k";
  }
  return Math.round(num).toLocaleString("en-US");
};

const getAttackerName = (id: number): string => {
  if (!props.attackerNameMap) return id.toString();
  return props.attackerNameMap[id.toString()] || id.toString();
};

const getConditionName = (id: number): string => {
  return condNameMap.value[id]?.name || `Unknown Condition ${id}`;
};

const getConditionIcon = (id: number): string => {
  return condNameMap.value[id]?.iconUrl || "";
};

// --- COMPUTED PROPERTIES ---

const playerDisplayData = computed(() => {
  const players = Object.values(fightSummary.players);
  if (!selectedTargetId.value) {
    // Overall view
    return players
      .map((p) => ({
        ...p.overallStats,
        id: p.id,
        name: p.name,
        talentIcon: p.talentIcon,
        talentName: p.talentName,
        talentColor: p.talentColor,
        missingAppearPacket: p.missingAppearPacket,
      }))
      .filter((p) => p.totalDamage > 0)
      .sort((a, b) => b.totalDamage - a.totalDamage);
  } else {
    // Target-specific view
    return players
      .map((p) => ({
        ...p.damageByTarget[selectedTargetId.value],
        id: p.id,
        name: p.name,
        talentIcon: p.talentIcon,
        talentName: p.talentName,
        talentColor: p.talentColor,
        missingAppearPacket: p.missingAppearPacket,
      }))
      .filter((p) => p && p.totalDamage > 0)
      .sort((a, b) => b.totalDamage - a.totalDamage);
  }
});

const totalDamageForView = computed(() => {
  return playerDisplayData.value.reduce((sum, p) => sum + p.totalDamage, 0);
});

const currentEncounterDuration = computed(() => {
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

const getSkillName = (skillId: number): string => {
  return skillNameMap.value[skillId]?.name || `unknownSkill:${skillId}`;
};

const getSkillImageSrc = (skillId: number): string => {
  return skillNameMap.value[skillId]?.iconUrl || "";
};

const getSkillBreakdown = (playerData: DamageBreakdown): SkillStats[] => {
  if (!playerData.skills) return [];
  return Object.values(playerData.skills).sort(
    (a, b) => b.totalDamage - a.totalDamage
  );
};

 const getRowProps = ({ item }: { item: any }) => {
  if (dpsMeterFillMode.value === "full") {
    const topDamage = playerDisplayData.value[0]?.totalDamage;
    if (!topDamage || !item.totalDamage) return { style: {} };
    const percentage = (item.totalDamage / topDamage) * 100;
    const bgColor = getPlayerColor(item);

    return {
      class: "character-row",
      style: {
        background: `linear-gradient(to right, ${bgColor} ${percentage}%, transparent ${percentage}%)`,
        backgroundClip: 'padding-box',
      },
    };
  }
  
  return {
    class: "character-row",
  };
};

const getSkillRowProps = (
  skillItem: any,
  topSkillDamage: number,
  playerName: string,
  playerItem?: any,
  index?: number
) => {
  if (dpsMeterFillMode.value === "full") {
    if (!topSkillDamage || !skillItem.totalDamage) return { style: {} };
    const percentage = (skillItem.totalDamage / topSkillDamage) * 100;
    
    let color = getMabiNameColor(playerName);
    if (playerItem) {
      color = getPlayerColor(playerItem);
    }

    let bgColor = color;
    if (index !== undefined && index % 2 !== 0) {
      bgColor = `color-mix(in srgb, ${color}, black 10%)`;
    }

    return {
      style: {
        background: `linear-gradient(to right, ${bgColor} ${percentage}%, transparent ${percentage}%)`,
        backgroundClip: 'padding-box',
      },
    };
  }
  
  return {};
};

// Helper for Hex -> RGBA
const hexToRgba = (hex: string, alpha: number) => {
    let c: any;
    if(/^#([A-Fa-f0-9]{3}){1,2}$/.test(hex)){
        c= hex.substring(1).split('');
        if(c.length== 3){
            c= [c[0], c[0], c[1], c[1], c[2], c[2]];
        }
        c= '0x'+c.join('');
        return 'rgba('+[(c>>16)&255, (c>>8)&255, c&255].join(',')+','+alpha+')';
    }
    return `rgba(128, 128, 128, ${alpha})`;
};

const areAllHidden = computed(() => {
    if (playerDisplayData.value.length === 0) return false;
    return playerDisplayData.value.every(p => hiddenPlayers.has(p.id));
});

const toggleAllPlayersVisibility = () => {
    if (globalHideMode.value) {
        // State 3 -> 1: Global Hide -> Show All
        globalHideMode.value = false;
        hiddenPlayers.clear();
    } else if (areAllHidden.value) {
        // State 2 -> 3: Session Hide -> Global Hide
        globalHideMode.value = true;
        hiddenPlayers.clear();
    } else {
        // State 1 -> 2: Show/Mixed -> Session Hide
        const allIds = playerDisplayData.value.map(p => p.id);
        setAllHiddenPlayers(allIds, true);
    }
};



// (Old static headers removed, using computed headers now)
</script>
