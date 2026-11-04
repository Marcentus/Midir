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
                            <span>{{ item.rawName || item.name }}<span v-if="item.raceId" class="text-grey font-weight-normal ml-1">({{ item.raceId }})</span></span>
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
                      <div v-if="runFolderParent" class="target-menu-item cursor-pointer px-4 py-3 text-white font-weight-bold" @click="backToRunFolder">
                        <v-icon icon="mdi-arrow-left" size="small" class="mr-2"></v-icon>{{ runFolderParent.name }}
                      </div>
                      <template v-if="expandedGroup.isDungeonRunGroup">
                        <div v-for="child in expandedGroup.runChildren" :key="child.id"
                          class="target-menu-item cursor-pointer phase-section-header px-4 py-3 d-flex justify-space-between align-center"
                          @click="openRunChild(child)">
                          <span class="text-white font-weight-bold">{{ child.name }}</span>
                          <span class="d-flex align-center ga-2"><span class="text-amber text-caption">{{ formatCompact(child.damage) }}</span><v-icon icon="mdi-chevron-right" size="small"></v-icon></span>
                        </div>
                        <v-divider class="my-1"></v-divider>
                      </template>
                      <!-- Top Option: Select All Targets in Group -->
                      <div
                        class="target-menu-item cursor-pointer position-relative overflow-hidden group-all-item"
                        :class="[
                          getTargetClass(expandedGroup.raceId),
                          selectedTargetId === expandedGroup.id ? 'active-item' : ''
                        ]"
                        @click="selectGroupAll(expandedGroup)"
                      >
                        <div class="target-item-content d-flex justify-space-between align-center px-4 pt-3 pb-2">
                          <span class="target-col-name font-weight-bold d-flex align-center text-white">
                            <span>{{ expandedGroup.isDungeonRunGroup ? 'All enemies in this run' : expandedGroup.isFahmesGroup ? 'All enemies in this stage' : expandedGroup.isAlbyGroup ? 'All enemies in this Alby run' : expandedGroup.isGlennOpeningGroup ? 'Cailleach only (opening HP)' : expandedGroup.isGlennMinorGroup ? 'All Glenn Bearna minor enemies' : expandedGroup.isCromMinorGroup ? 'All Crom Bas minor enemies' : expandedGroup.isPhantasmBossGroup ? 'Succubus Queen (boss HP only)' : expandedGroup.isPhantasmMinorGroup ? 'All Phantasm minor enemies' : expandedGroup.isCromBossGroup ? 'All Crom Bas bosses (boss HP only)' : expandedGroup.isGlennBossGroup ? 'All Glenn bosses (final encounter HP)' : expandedGroup.isGate1BossGroup ? 'Both Vertrag bosses (boss HP only)' : 'All Targets (' + (expandedGroup.rawName || expandedGroup.name) + ')'  }}<span v-if="expandedGroup.raceId" class="text-grey font-weight-normal ml-1">({{ expandedGroup.raceId }})</span></span>
                          </span>

                          <div class="d-flex align-center ga-2">
                            <span class="target-col-damage font-weight-bold text-amber text-right">
                              {{ formatCompact(expandedGroup.damage) }}
                            </span>
                          </div>
                        </div>
                      </div>

                      <v-divider class="my-1" style="border-color: rgba(255,255,255,0.1);"></v-divider>

                      <!-- Glenn Bearna: each attempt's combined boss lineage plus
                           selectable boss-specific aggregates. Phase entities
                           remain individually selectable underneath. -->
                      <template v-if="expandedGroup.isGlennBossGroup || expandedGroup.isCromBossGroup || expandedGroup.isPhantasmBossGroup || expandedGroup.isGlennOpeningGroup || expandedGroup.isAlbyGroup || expandedGroup.isFahmesGroup">
                        <div v-for="boss in [...(expandedGroup.glennBossScopes || []), ...(expandedGroup.cromBossScopes || []), ...(expandedGroup.albyWaveScopes || []), ...(expandedGroup.fahmesLevelScopes || []), ...(expandedGroup.phantasmBossScopes || [])]" :key="boss.id"
                          class="target-menu-item cursor-pointer phase-section-header px-4 py-3 d-flex justify-space-between align-center"
                          :class="selectedTargetId === boss.id ? 'active-item' : ''"
                          @click="selectGroupTarget(boss)">
                          <span class="target-col-name text-white font-weight-bold">{{ boss.name }}</span>
                          <span class="text-amber text-caption">{{ formatCompact(boss.damage) }}</span>
                        </div>
                        <v-divider class="my-1" style="border-color: rgba(255,255,255,0.1);"></v-divider>
                        <div v-if="expandedGroup.isGlennBossGroup || expandedGroup.isCromBossGroup" class="px-4 pt-2 text-caption text-grey-lighten-1">Individual boss phases</div>
                      </template>

                      <!-- Gate 1: clicking the boss row selects BOTH Vertrag forms;
                           the separate chevron expands/collapses their individual forms. -->
                      <div v-if="expandedGroup.isGate1BossGroup"
                        class="target-menu-item cursor-pointer phase-section-header px-4 py-3 d-flex justify-space-between align-center"
                        :class="selectedTargetId === expandedGroup.id ? 'active-item' : ''"
                        @click="selectGroupAll(expandedGroup)">
                        <span class="text-white font-weight-bold d-flex align-center">
                          <button type="button" class="vertrag-expand-button mr-2"
                            :aria-expanded="openPhaseSections.includes(expandedGroup.id + '_vertrag')"
                            aria-label="Expand or collapse individual Vertrag forms"
                            @click.stop="togglePhaseSection(expandedGroup.id + '_vertrag')">
                            <v-icon :icon="openPhaseSections.includes(expandedGroup.id + '_vertrag') ? 'mdi-chevron-down' : 'mdi-chevron-right'" size="small"></v-icon>
                          </button>
                          Vertrag of the Deep Roots (both forms)
                        </span>
                        <span class="text-amber text-caption">{{ formatCompact(expandedGroup.damage) }}</span>
                      </div>
                      <template v-if="!expandedGroup.isGate1BossGroup || openPhaseSections.includes(expandedGroup.id + '_vertrag')">
                      <!-- Boss lineage stays pinned above phase adds. -->
                      <div v-for="subItem in expandedGroup.bossMembers || []" :key="subItem.id"
                        class="target-menu-item cursor-pointer position-relative overflow-hidden"
                        :class="[getTargetClass(subItem.raceId), selectedTargetId === subItem.id ? 'active-item' : '']"
                        @click="selectGroupTarget(subItem)">
                        <div class="target-item-content d-flex justify-space-between align-center px-4 pt-3 pb-2">
                          <span class="target-col-name font-weight-bold text-white">{{ subItem.rawName || subItem.name }} ({{ subItem.raceId }})
                            <span class="text-caption text-teal-lighten-2 ml-2">{{ subItem.raceId === 7600 ? '100–50%' : subItem.raceId === 7601 ? '50–0%' : subItem.raceId === 7585 ? '100–10%' : subItem.raceId === 7586 ? '100–60%' : subItem.raceId === 7587 ? '60–0%' : 'Boss' }}</span>
                          </span>
                          <span v-if="subItem.hasHpUpdates" class="hp-text text-caption text-grey-lighten-1 text-right">{{ formatCompact(subItem.currentHp) }} / {{ formatCompact(subItem.maxHp) }} ({{ subItem.hpPercent.toFixed(1) }}%)</span>
                          <span v-else class="text-amber">{{ formatCompact(subItem.damage) }}</span>
                        </div>
                        <div v-if="subItem.hasHpUpdates" class="hp-bar-separator w-100"><div class="hp-bar-separator-fill hp-enemy" :style="{width:subItem.hpPercent + '%'}"></div></div>
                      </div>

                      </template>

                      <div v-for="section in expandedGroup.phaseSections || []" :key="section.id">
                        <div class="target-menu-item cursor-pointer phase-section-header px-4 py-3 d-flex justify-space-between align-center"
                          @click="togglePhaseSection(section.id)">
                          <span class="text-white font-weight-bold"><v-icon :icon="openPhaseSections.includes(section.id) ? 'mdi-chevron-down' : 'mdi-chevron-right'" size="small"></v-icon>
                            {{ section.label }} ({{ section.targets.length }})</span>
                          <span class="text-amber text-caption">{{ formatCompact(section.damage) }}</span>
                        </div>
                        <template v-if="openPhaseSections.includes(section.id)">
                          <div v-for="subItem in section.targets" :key="subItem.id"
                            class="target-menu-item cursor-pointer position-relative overflow-hidden pl-3"
                            :class="[getTargetClass(subItem.raceId), selectedTargetId === subItem.id ? 'active-item' : '']"
                            @click="selectGroupTarget(subItem)">
                            <div class="target-item-content d-flex justify-space-between align-center px-4 pt-3 pb-2">
                              <span class="target-col-name text-white">{{ subItem.rawName || subItem.name }} ({{ subItem.raceId }})</span>
                              <span v-if="subItem.hasHpUpdates" class="hp-text text-caption text-grey-lighten-1 text-right">{{ formatCompact(subItem.currentHp) }} / {{ formatCompact(subItem.maxHp) }} ({{ subItem.hpPercent.toFixed(1) }}%)</span>
                              <span v-else class="text-amber">{{ formatCompact(subItem.damage) }}</span>
                            </div>
                            <div v-if="subItem.hasHpUpdates" class="hp-bar-separator w-100"><div class="hp-bar-separator-fill hp-enemy" :style="{width:subItem.hpPercent + '%'}"></div></div>
                          </div>
                        </template>
                      </div>
                      <div v-if="expandedGroup.miscellaneous?.length">
                        <div class="target-menu-item cursor-pointer phase-section-header px-4 py-3 d-flex justify-space-between align-center"
                          @click="togglePhaseSection(expandedGroup.id + '_misc')">
                          <span class="text-white font-weight-bold"><v-icon :icon="openPhaseSections.includes(expandedGroup.id + '_misc') ? 'mdi-chevron-down' : 'mdi-chevron-right'" size="small"></v-icon>
                            {{ expandedGroup.isGlennMinorGroup ? 'Glenn Bearna enemies' : expandedGroup.isCromMinorGroup ? 'Crom Bas enemies' : 'Other enemies' }} ({{ expandedGroup.miscellaneous.length }})</span>
                        </div>
                        <template v-if="openPhaseSections.includes(expandedGroup.id + '_misc')">
                          <div v-for="subItem in expandedGroup.miscellaneous" :key="subItem.id"
                            class="target-menu-item cursor-pointer pl-3"
                            :class="[getTargetClass(subItem.raceId), selectedTargetId === subItem.id ? 'active-item' : '']"
                            @click="selectGroupTarget(subItem)">
                            <div class="target-item-content d-flex justify-space-between align-center px-4 pt-3 pb-2">
                              <span class="text-white">{{ subItem.rawName || subItem.name }} ({{ subItem.raceId }})</span>
                              <span v-if="subItem.hasHpUpdates" class="hp-text text-caption text-grey-lighten-1 text-right">{{ formatCompact(subItem.currentHp) }} / {{ formatCompact(subItem.maxHp) }} ({{ subItem.hpPercent.toFixed(1) }}%)</span>
                              <span v-else class="text-amber">{{ formatCompact(subItem.damage) }}</span>
                            </div>
                          </div>
                        </template>
                      </div>
                    </div>
                  </div>
                </div>
              </v-menu>
            </div>

            <!-- Right: Vital Stats -->
            <div class="header-section stats-section">
              <template v-if="selectedTargetHp">
                <div class="stat-item">
                  <div class="header-label text-right">TARGET HP</div>
                  <div class="stat-value">
                    <span style="color: #ff6e6d;">{{ formatNumber(selectedTargetHp.current) }}/{{ formatNumber(selectedTargetHp.max) }}</span>
                    <span class="text-grey text-caption ml-1">({{ selectedTargetHp.percent.toFixed(1) }}%)</span>
                  </div>
                </div>
                <div class="stat-divider mx-5"></div>
              </template>
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
            <!-- Experimental healing UI: optional until effective-healing and shield attribution are validated. -->
            <v-tab v-if="showHealingTab" value="healing">Healing Done</v-tab>
            <v-tab value="damageTaken">Damage Taken</v-tab>
            <v-tab value="graph">Graph</v-tab>
          </v-tabs>

          <v-window v-model="tab">
            <v-window-item value="damageDealt">
              <apply-damage-by-skill :attackerNameMap="attackerNameMap" />
              </v-window-item>
            <v-window-item v-if="showHealingTab" value="healing">
              <apply-healing-by-skill :attackerNameMap="attackerNameMap" />
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
import { fightSummary, selectedTargetId, isNavDrawerOpen, activeSessionId, showHealingTab } from "@/store";
import {
  isGroupTargetId,
  getGroupMemberTargetIds,
  getEncounterDurationForTargets,
  aggregateGroupConditions,
  GROUPED_ENCOUNTERS,
} from "@/utils/targetGrouping";
import { findGate1Attempts, gate1EffectiveHp } from "@/utils/gate1Encounter";
import { findDungeonRunFolders } from '@/utils/dungeonRunFolders';
import { findFahmesStages, FAHMES_STAGE_PREFIX, FAHMES_LEVEL_PREFIX } from '@/utils/fahmesRuin';
import { findPhantasmRuns, PHANTASM_RUN_PREFIX, PHANTASM_MINOR_PREFIX, PHANTASM_MECHANICS_PREFIX, PHANTASM_BOSS_ROOM_PREFIX } from '@/utils/phantasmEncounter';
import { findGlennOpenings, GLENN_OPENING_PREFIX, GLENN_OPENING_BARRIER_PREFIX } from '@/utils/glennOpening';
import { findAlbyRuns, ALBY_RACES, ALBY_WAVES, ALBY_GROUP_PREFIX, ALBY_WAVE_PREFIX } from '@/utils/albyShineseeker';
import { findGlennAttempts } from "@/utils/glennEncounter";
import { findHighTierRuns } from '@/utils/highTierMinorGrouping';
import { findCromAttempts, findCromMechanics, CROM_IRUSAN, CROM_GLAS, CROM_GLAS_VARIANTS, CROM_GLAS_NAMES, CROM_DISCIPLE_RACES, CROM_MINOR_GROUP_ID, CROM_MINOR_RACES, CROM_IRUSAN_GROUP_PREFIX, CROM_GLAS_GROUP_PREFIX, CROM_MERCH_GROUP_PREFIX, CROM_DISCIPLES_GROUP_PREFIX } from "@/utils/cromEncounter";
import { findMechanicGroups } from "@/utils/mechanicGroups";
import { partitionMinorRuns } from "@/utils/instanceMinorGrouping";
import { GLENN_MINOR_GROUP_ID, isGlennMinorRace } from "@/utils/glennMinorEnemies";
import SessionPanel from "@/components/SessionPanel.vue";
import ApplyDamageBySkillComponent from "@/components/applyDamageBySkill.vue";
import ApplyHealingBySkillComponent from "@/components/applyHealingBySkill.vue";
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
    ApplyHealingBySkill: ApplyHealingBySkillComponent,
    DamageTakenBySource: DamageTakenBySourceComponent,
    DamageGraph,
    TargetConditionView,
  },
  setup() {
    const tab = ref("damageDealt");
    // If Healing Done is disabled while selected, return to the regular damage view.
    watch(showHealingTab, (enabled) => {
      if (!enabled && tab.value === "healing") tab.value = "damageDealt";
    });
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
      if (isGroupTargetId(selectedTargetId.value)) {
        const memberIds = getGroupMemberTargetIds(selectedTargetId.value, fightSummary);
        return getEncounterDurationForTargets(fightSummary.players, memberIds);
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
      const isGroup = isGroupTargetId(selectedTargetId.value);
      const memberIds = isGroup
        ? getGroupMemberTargetIds(selectedTargetId.value, fightSummary)
        : [];

      for (const player of Object.values(fightSummary.players)) {
        if (selectedTargetId.value) {
          if (isGroup) {
            for (const tid of memberIds) {
              if (player.damageByTarget[tid]) {
                total += player.damageByTarget[tid].totalDamage;
              }
            }
          } else {
            if (player.damageByTarget[selectedTargetId.value]) {
              total += player.damageByTarget[selectedTargetId.value].totalDamage;
            }
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
    const runFolderParent = ref<any>(null);
    const openRunChild = (child: any) => {
      runFolderParent.value = expandedGroup.value;
      expandedGroup.value = child;
    };
    const backToRunFolder = () => {
      if (runFolderParent.value) expandedGroup.value = runFolderParent.value;
      runFolderParent.value = null;
    };

    const openPhaseSections = ref<string[]>([]);
    const togglePhaseSection = (id: string) => {
      openPhaseSections.value = openPhaseSections.value.includes(id)
        ? openPhaseSections.value.filter(value => value !== id)
        : [...openPhaseSections.value, id];
    };

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

      if (!hasAnyEntityDeath && !targets.some(t => GROUPED_ENCOUNTERS.some(g => g.bossRaceId === t.raceId)) &&
          !targets.some(t => isGlennMinorRace(t.raceId)) &&
          !targets.some(t => CROM_MINOR_RACES.has(Number(t.raceId))) &&
          !targets.some(t => ALBY_RACES.has(Number(t.raceId))) &&
          !targets.some(t => Number(t.raceId)>=121001 && Number(t.raceId)<127000) &&
          !targets.some(t => Number(t.raceId)===7575)) {
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

      const phantasmRuns = findPhantasmRuns(fightSummary.targets || {});
      const otherDungeonOwnedIds = new Set([
        ...phantasmRuns.flatMap(run => run.entityIds),
        ...findFahmesStages(fightSummary.targets || {}).flatMap(stage => stage.entityIds),
      ]);
      const additionalQueenDefs = phantasmRuns.filter(r => ![191681,195012].includes(Number(fightSummary.targets?.[r.queenId]?.raceId)))
        .map(r=>({bossRaceId:Number(fightSummary.targets[r.queenId].raceId),groupName:'Rabbie Phantasm: Queen (variant unverified)'}));
      const bossDefinitions: Array<{ bossRaceId: number; groupName: string; partnerRaceId?: number }> = [
        ...GROUPED_ENCOUNTERS,
        ...additionalQueenDefs,
      ];
      for (const bossDef of bossDefinitions) {
        // Find all boss targets of this bossRaceId
        const bosses = targets.filter(t => t.raceId === bossDef.bossRaceId &&
          (![191681,195012].includes(bossDef.bossRaceId) && !additionalQueenDefs.some(d=>d.bossRaceId===bossDef.bossRaceId) || phantasmRuns.some(r=>r.queenId===t.id)));

        for (const bossTarget of bosses) {
          // A paired Glas is already contained by the preceding Irusan group.
          const cromAttempt = (bossDef.bossRaceId===CROM_IRUSAN || CROM_GLAS_VARIANTS.has(bossDef.bossRaceId))
            ? findCromAttempts(fightSummary.targets || {}).find(a=>a.firstEntityId===bossTarget.id) : undefined;
          if ((bossDef.bossRaceId===CROM_IRUSAN || CROM_GLAS_VARIANTS.has(bossDef.bossRaceId)) && !cromAttempt) continue;
          // Keep both completed and failed attempts visible; never suppress the
          // first form just because it ended before 50% HP.
          const bossStartTime = bossTarget.startTime || 0;
          let bossEndTime = (bossTarget.seenDead || bossTarget.disappeared) ? (bossTarget.endTime || 0) : Infinity;

          let partnerTarget: any = null;
          if (bossDef.bossRaceId === 7585) {
            const attempt = findGlennAttempts(fightSummary.targets || {}).find(a => a.firstEntityId === bossTarget.id);
            if (attempt) {
              const descendants = attempt.entityIds.slice(1).map(id => targets.find(t => t.id === id)).filter(Boolean);
              partnerTarget = descendants[descendants.length - 1] || null;
              bossEndTime = Number(partnerTarget?.endTime || bossTarget.endTime) || Infinity;
            }
            const nextAttempt = findGlennAttempts(fightSummary.targets || {})
              .filter(a => a.startUnixSeconds > bossStartTime).sort((a,b) => a.startUnixSeconds-b.startUnixSeconds)[0];
            if (nextAttempt) bossEndTime = Math.min(bossEndTime, nextAttempt.startUnixSeconds - 0.001);
          } else if (cromAttempt) {
            partnerTarget = cromAttempt.glasId && cromAttempt.glasId !== bossTarget.id
              ? targets.find(t=>t.id===cromAttempt.glasId) : null;
            bossEndTime = Number(partnerTarget?.endTime || bossTarget.endTime) || Infinity;
            const next = findCromAttempts(fightSummary.targets || {})
              .filter(a=>a.startUnixSeconds>bossStartTime).sort((a,b)=>a.startUnixSeconds-b.startUnixSeconds)[0];
            if (next) bossEndTime = Math.min(bossEndTime,next.startUnixSeconds-0.001);
          } else if (bossDef.bossRaceId === 7600) {
            const attempt = findGate1Attempts(fightSummary.targets || {})
              .find(a => a.firstEntityId === bossTarget.id);
            if (attempt?.replacementEntityId) {
              partnerTarget = targets.find(t => t.id === attempt.replacementEntityId);
              bossEndTime = Number(partnerTarget?.endTime) || Infinity;
            } else bossEndTime = Number(bossTarget.endTime) || Infinity;
            const nextAttempt = findGate1Attempts(fightSummary.targets || {})
              .filter(a => a.startUnixSeconds > bossStartTime).sort((a,b)=>a.startUnixSeconds-b.startUnixSeconds)[0];
            if (nextAttempt) bossEndTime = Math.min(bossEndTime, nextAttempt.startUnixSeconds - 0.001);
          } else if (bossDef.bossRaceId === 193810 || bossDef.bossRaceId === 7615) {
            // No recording proves the Abyss/Gate 4 phase or add behavior.
            // Prevent a missing despawn packet from consuming the next attempt.
            const nextBoss = targets.filter(t => t.raceId === bossDef.bossRaceId &&
              Number(t.startTime) > bossStartTime)
              .sort((a,b) => Number(a.startTime)-Number(b.startTime))[0];
            if (nextBoss) bossEndTime = Math.min(bossEndTime, Number(nextBoss.startTime)-0.001);
          } else if (bossDef.partnerRaceId) {
            partnerTarget = targets.find(t => t.raceId === bossDef.partnerRaceId && (t.startTime || 0) >= bossStartTime);
            if (partnerTarget) bossEndTime = (partnerTarget.seenDead || partnerTarget.disappeared) ? (partnerTarget.endTime || 0) : Infinity;
          }

          // Find all targets that appeared during the boss's lifetime
          const members = targets.filter(t => {
            const tStartTime = t.startTime || 0;
            return tStartTime >= bossStartTime && (bossEndTime === Infinity || tStartTime <= bossEndTime);
          });
          // Glenn Bearna minor enemies have their own session-wide folder.
          // They must never be silently folded into a boss's DPS or hidden
          // from the minor-enemy folder by a broad time-window grouping.
          if (bossDef.bossRaceId === 7585) {
            for (let i = members.length - 1; i >= 0; i--) {
              if (isGlennMinorRace(members[i].raceId)) members.splice(i, 1);
            }
          }

          if (cromAttempt) {
            // Do not consume pre-boss trash or Disciples from the prep/Irusan
            // window as if they were boss members. Mechanics remain represented
            // by dedicated sections and never contribute to boss DPS.
            const mechanicIds = findCromMechanics(fightSummary.targets || {},cromAttempt);
            const allowed = new Set([...cromAttempt.entityIds,...mechanicIds.merchIds,...mechanicIds.discipleIds]);
            for (let i=members.length-1; i>=0; i--) {
              const race = Number(members[i].raceId);
              if (!allowed.has(members[i].id) &&
                (CROM_MINOR_RACES.has(race) || CROM_DISCIPLE_RACES.has(race)))
                members.splice(i,1);
            }
          }

          // Explicitly retain the matched replacement even if its spawn time is
          // quantized slightly outside the parent lifetime boundary.
          if (partnerTarget && !members.some(m => m.id === partnerTarget.id)) members.push(partnerTarget);

          // New Succubus Queen means a distinct attempt, not the same boss run.
          // Keep only the Queen and her four simultaneously appearing named adds.
          if (phantasmRuns.some(r=>r.queenId===bossTarget.id)) {
            const nextQueen = [191681,195012].includes(bossDef.bossRaceId) ? targets
              .filter(t => [191681,195012].includes(Number(t.raceId)) && Number(t.startTime) > bossStartTime)
              .sort((a,b)=>Number(a.startTime)-Number(b.startTime))[0] : null;
            const siblingIds = new Set([195019,195020,195021,195022]);
            for (let i=members.length-1;i>=0;i--) {
              const m=members[i];
              const withQueen = phantasmRuns.some(r=>r.queenId===bossTarget.id) && siblingIds.has(Number(m.raceId)) &&
                Math.abs(Number(m.startTime)-bossStartTime) <= 3 &&
                (!nextQueen || Number(m.startTime)<Number(nextQueen.startTime));
              if (!findPhantasmRuns(fightSummary.targets || {}).find(r=>r.queenId===bossTarget.id)?.queenIds.includes(m.id) && !withQueen) members.splice(i,1);
            }
          }
          if (bossDef.bossRaceId === 193810 || bossDef.bossRaceId === 7615) {
            // Conservative boss-only policy until real encounter evidence is available.
            // Keep other entities individually visible instead of attaching unverified
            // Folamh Apparitions, barriers, Shards, etc. as boss damage.
            members.splice(0, members.length, ...members.filter(t => t.id === bossTarget.id));
          }
          if (members.length > 0) {
            const glennBossIds = bossDef.bossRaceId === 7585 ? findGlennAttempts(fightSummary.targets || {}).find(a => a.firstEntityId === bossTarget.id)?.entityIds || [bossTarget.id] : [];
            const independentBoss = phantasmRuns.some(r=>r.queenId===bossTarget.id);
            const groupBossMembers = bossDef.bossRaceId === 7585 ? members.filter(m=>glennBossIds.includes(m.id)) : bossDef.bossRaceId===7600 ? members.filter(m=>m.id===bossTarget.id || m.id===partnerTarget?.id) : cromAttempt ? members.filter(m=>cromAttempt.entityIds.includes(m.id)) : independentBoss ? members.filter(m=>findPhantasmRuns(fightSummary.targets || {}).find(r=>r.queenId===bossTarget.id)?.queenIds.includes(m.id)) : members;
            const groupTotalDamage = groupBossMembers.reduce((sum, m) => sum + m.damage, 0);
            const groupDamagePercent = grandTotalDamage > 0 ? (groupTotalDamage / grandTotalDamage) * 100 : 0;

            // Determine active HP details (use partner's HP if active, else boss's HP)
            const activeBoss = partnerTarget || bossTarget;
            const hpOf = (target: any) => {
              const history = (target?.hpHistory || []).filter((h: any) => Number(h.maxHp) > 0);
              const sample = history[history.length - 1];
              return {max: Number(sample?.maxHp || target?.maxHp || 0), current: Number(sample?.currentHp ?? target?.currentHp ?? 0)};
            };
            const firstHp = hpOf(bossTarget);
            const secondHp = partnerTarget ? hpOf(partnerTarget) : null;
            const effective = bossDef.bossRaceId === 7600 && secondHp
              ? gate1EffectiveHp(firstHp.max, firstHp.current, secondHp.max, secondHp.current, true)
              : bossDef.bossRaceId === 7585 && glennBossIds.length >= 2
              ? (() => {
                  // The 7585 boss fight is independent (100–10%).
                  // The second 7586/7587 fight shares a 100–0% health scale;
                  // Tyrant inherits the remaining ~60% at the handoff.
                  const [,b,c] = glennBossIds.map(id => hpOf(targets.find(t => t.id === id)));
                  if (!b?.max) return null;
                  const finalTarget = targets.find(t => t.id === glennBossIds[2]);
                  const finalStarted = Boolean(finalTarget && Number(finalTarget.startTime)>0 &&
                    (finalTarget.seenAppear || finalTarget.seenDead || finalTarget.disappeared));
                  const maxHp = b.max;
                  const currentHp = finalStarted && c?.max ? Math.max(0,Math.min(c.current,c.max*.6)) / (c.max*.6) * (b.max*.6)
                    : Math.max(0,b.current);
                  return {maxHp,currentHp,hpPercent:maxHp>0?100*currentHp/maxHp:0};
                })() : null;

            const groupPhases = findMechanicGroups(fightSummary.targets || {})
              .filter(w => w.bossEntityIds.includes(bossTarget.id));
            const phaseMemberIds = new Set(groupPhases.flatMap(w => w.memberEntityIds));
            const phaseSections = groupPhases.map(w => {
              const phaseMembers = members.filter(m => w.memberEntityIds.includes(m.id));
              return {
                id: `${bossTarget.id}_${w.type}_${w.phaseHpPercent}`,
                label: w.label,
                damage: phaseMembers.reduce((sum, m) => sum + m.damage, 0),
                targets: phaseMembers,
                expectedMembers: w.expectedMembers,
                evidence: w.evidence,
              };
            }).filter(section => section.targets.length);
            // The boss belongs at the very top, then real phase sections,
            // then compact miscellaneous mobs that can be expanded separately.
            const bossMembers = members.filter(m => bossDef.bossRaceId === 7585 ? glennBossIds.includes(m.id) : cromAttempt ? cromAttempt.entityIds.includes(m.id) : m.id === bossTarget.id || m.id === partnerTarget?.id || (independentBoss && findPhantasmRuns(fightSummary.targets || {}).find(r=>r.queenId===bossTarget.id)?.queenIds.includes(m.id)));
            const cromMechanics = cromAttempt ? findCromMechanics(fightSummary.targets || {},cromAttempt) : null;
            const cromMechanicIds = new Set([...(cromMechanics?.merchIds||[]),...(cromMechanics?.discipleIds||[])]);
            const miscellaneous = members.filter(m => !phaseMemberIds.has(m.id) &&
              !bossMembers.some(b => b.id === m.id) && !cromMechanicIds.has(m.id) &&
              !(independentBoss && [195019,195020,195021,195022].includes(Number(m.raceId))) &&
              !(cromAttempt && CROM_MINOR_RACES.has(Number(m.raceId))));
            const groupItem = {
              id: `group_${bossTarget.id}`,
              name: (cromAttempt
                ? `Crom Bas — Bosses · Attempt ${findCromAttempts(fightSummary.targets || {}).findIndex(a=>a.firstEntityId===bossTarget.id)+1}`
                : bossDef.bossRaceId===7585
                ? `Glenn Bearna — Bosses · Attempt ${findGlennAttempts(fightSummary.targets || {}).findIndex(a=>a.firstEntityId===bossTarget.id)+1}`
                : (bossDef.bossRaceId === 193810 || bossDef.bossRaceId === 7615)
                ? `${bossDef.groupName} — Boss · Attempt ${targets.filter(t => t.raceId === bossDef.bossRaceId && Number(t.startTime) <= bossStartTime).length}`
                : bossDef.groupName),
              rawName: (cromAttempt
                ? `Crom Bas — Bosses · Attempt ${findCromAttempts(fightSummary.targets || {}).findIndex(a=>a.firstEntityId===bossTarget.id)+1}`
                : bossDef.bossRaceId===7585
                ? `Glenn Bearna — Bosses · Attempt ${findGlennAttempts(fightSummary.targets || {}).findIndex(a=>a.firstEntityId===bossTarget.id)+1}`
                : (bossDef.bossRaceId === 193810 || bossDef.bossRaceId === 7615)
                ? `${bossDef.groupName} — Boss · Attempt ${targets.filter(t => t.raceId === bossDef.bossRaceId && Number(t.startTime) <= bossStartTime).length}`
                : bossDef.groupName),
              isGroup: true,
              damage: groupTotalDamage,
              damagePercent: groupDamagePercent,
              currentHp: effective?.currentHp ?? activeBoss.currentHp,
              maxHp: effective?.maxHp ?? activeBoss.maxHp,
              hpPercent: effective?.hpPercent ?? activeBoss.hpPercent,
              seenAppear: activeBoss.seenAppear,
              seenDead: activeBoss.seenDead,
              disappeared: activeBoss.disappeared,
              startTime: bossTarget.startTime,
              endTime: partnerTarget ? partnerTarget.endTime : bossTarget.endTime,
              raceId: activeBoss.raceId,
              hasHpUpdates: activeBoss.hasHpUpdates,
              targets: members,
              bossMembers,
              isPhantasmBossGroup: independentBoss,
              phantasmBossScopes: independentBoss ? (() => {
                const run = findPhantasmRuns(fightSummary.targets || {}).find(r => r.queenId === bossTarget.id);
                const adds = targets.filter(t => run?.bossAddIds.includes(t.id));
                return adds.length ? [{
                  id: PHANTASM_MECHANICS_PREFIX + bossTarget.id,
                  name: 'Boss mechanics — Four Succubi',
                  damage: adds.reduce((n, t) => n + t.damage, 0),
                }] : [];
              })() : [],
              isGate1BossGroup: bossDef.bossRaceId === 7600,
              isGlennBossGroup: bossDef.bossRaceId === 7585,
              isCromBossGroup: Boolean(cromAttempt),
              cromBossScopes: cromAttempt ? [
                ...(cromAttempt.irusanId ? [{id:CROM_IRUSAN_GROUP_PREFIX+bossTarget.id,name:'Irusan (mid-boss)',damage:bossMembers.filter(m=>m.id===cromAttempt.irusanId).reduce((a,m)=>a+m.damage,0)}] : []),
                ...(cromAttempt.glasId ? [{id:CROM_GLAS_GROUP_PREFIX+bossTarget.id,name:(CROM_GLAS_NAMES[Number((fightSummary.targets || {})[cromAttempt.glasId]?.raceId)] || 'Glas Ghaibhleann')+' (final boss)',damage:bossMembers.filter(m=>m.id===cromAttempt.glasId).reduce((a,m)=>a+m.damage,0)}] : []),
              ] : [],
              glennBossScopes: bossDef.bossRaceId === 7585 ? [
                {
                  id: `group_glenn_first_${bossTarget.id}`,
                  name: 'Unfinished Reverie (100–10%)',
                  damage: bossMembers.filter(m => Number(m.raceId) === 7585)
                    .reduce((total, m) => total + m.damage, 0),
                },
                ...(bossMembers.some(m => [7586, 7587].includes(Number(m.raceId))) ? [{
                  id: `group_glenn_final_${bossTarget.id}`,
                  name: 'Unfinished Reverie → Winter Tyrant (100–0%)',
                  damage: bossMembers.filter(m => [7586, 7587].includes(Number(m.raceId)))
                    .reduce((total, m) => total + m.damage, 0),
                }] : []),
              ] : [],
              phaseSections: cromAttempt ? [
                ...(cromMechanics?.merchIds.length ? [{id:CROM_MERCH_GROUP_PREFIX+bossTarget.id,label:'Irusan mechanics — Merch Diafol',targets:targets.filter(t=>cromMechanics.merchIds.includes(t.id)),damage:targets.filter(t=>cromMechanics.merchIds.includes(t.id)).reduce((a,t)=>a+t.damage,0)}]:[]),
                ...(cromMechanics?.discipleIds.length ? [{
                  id:CROM_DISCIPLES_GROUP_PREFIX+bossTarget.id,
                  label:'Glas mechanics — Disciples',
                  targets:targets.filter(t=>cromMechanics.discipleIds.includes(t.id)),
                  damage:targets.filter(t=>cromMechanics.discipleIds.includes(t.id)).reduce((a,t)=>a+t.damage,0),
                }] : []),
              ] : phaseSections,
              miscellaneous,
            };

            groupedList.push(groupItem);
            members.forEach(m => groupedTargetIds.add(m.id));
            if (cromAttempt && cromMechanics) {
              // Mechanic adds can spawn just before their boss and must not
              // become unrelated top-level targets.
              [...cromMechanics.merchIds,...cromMechanics.discipleIds].forEach(id=>groupedTargetIds.add(id));
            }
          }
        }
      }

      // Glenn opening is a separate 100%-to-90% Cailleach checkpoint.
      // Barriers are entrance mechanics; Icicle Sprites belong with minor enemies.
      const openingRuns=findGlennOpenings(fightSummary.targets||{});
      for (const [index,run] of openingRuns.entries()) {
        const cailleach=targets.find(t=>t.id===run.firstEntityId);
        if(!cailleach)continue;
        const barriers=targets.filter(t=>run.barrierIds.includes(t.id));
        const all=[cailleach,...barriers];
        groupedList.push({
          id:GLENN_OPENING_PREFIX+run.firstEntityId,
          name:`Glenn Bearna — Run ${index+1} · Cailleach (opening)`,
          rawName:`Glenn Bearna — Run ${index+1} · Cailleach (opening)`,
          isGroup:true,isGlennOpeningGroup:true,damage:cailleach.damage,
          damagePercent:grandTotalDamage>0?100*cailleach.damage/grandTotalDamage:0,
          currentHp:cailleach.currentHp,maxHp:cailleach.maxHp,hpPercent:cailleach.hpPercent,
          hasHpUpdates:cailleach.hasHpUpdates,startTime:cailleach.startTime,endTime:cailleach.endTime,
          bossMembers:[cailleach],targets:all,miscellaneous:[],
          glennBossScopes:[
            ...(barriers.length?[{id:GLENN_OPENING_BARRIER_PREFIX+run.firstEntityId,name:"Cailleach's Barriers",damage:barriers.reduce((n,t)=>n+t.damage,0)}]:[]),
          ],
          phaseSections:[],
        });
        all.forEach(t=>groupedTargetIds.add(t.id));
      }

      // Alby Shineseeker uses sequential waves, not a final boss HP bar.
      // The first wave's bats establish conservative run boundaries.
      for(const run of findAlbyRuns(fightSummary.targets||{})) {
        const members=targets.filter(t=>run.entityIds.includes(t.id));
        if(!members.length)continue;
        const damage=members.reduce((n,t)=>n+t.damage,0);
        const scopes=ALBY_WAVES.flatMap((wave,waveIndex)=>{
          const inWave=targets.filter(t=>run.waveIds[waveIndex].includes(t.id));
          return inWave.length?[{id:`${ALBY_WAVE_PREFIX}${run.index}_${waveIndex}`,name:wave.label,damage:inWave.reduce((n,t)=>n+t.damage,0)}]:[];
        });
        groupedList.push({
          id:`${ALBY_GROUP_PREFIX}${run.index}`,
          name:`Alby Shineseeker — Run ${run.index} · ${run.completion==='final-wave-defeated'?'Final wave defeated':'Attempted'}`,
          rawName:`Alby Shineseeker — Run ${run.index} · ${run.completion==='final-wave-defeated'?'Final wave defeated':'Attempted'}`,
          isGroup:true,isAlbyGroup:true,damage,
          damagePercent:grandTotalDamage>0?100*damage/grandTotalDamage:0,
          currentHp:0,maxHp:0,hpPercent:0,hasHpUpdates:false,
          startTime:run.startUnixSeconds,endTime:run.endUnixSeconds,
          targets:members,bossMembers:[],albyWaveScopes:scopes,
          phaseSections:[],miscellaneous:[],
        });
        members.forEach(t=>groupedTargetIds.add(t.id));
      }

      // Fahmes Ruin: each independently enterable stage contains five
      // sequential levels. Three waves per level may overlap in time.
      for(const stage of findFahmesStages(fightSummary.targets || {})) {
        const members=targets.filter(t=>stage.entityIds.includes(t.id));
        if(!members.length)continue;
        const damage=members.reduce((n,t)=>n+t.damage,0);
        const scopes=stage.levels.map(level=>{
          const group=members.filter(t=>level.entityIds.includes(t.id));
          return {id:`${FAHMES_LEVEL_PREFIX}${stage.stage}_${stage.attempt}_${level.index}`,
            name:`Level ${level.index} — all recorded waves`,damage:group.reduce((n,t)=>n+t.damage,0)};
        });
        const name=`Fahmes Ruin — Stage ${stage.stage}${stage.attempt>1?` · Attempt ${stage.attempt}`:''}`;
        groupedList.push({id:`${FAHMES_STAGE_PREFIX}${stage.stage}_${stage.attempt}`,
          name,rawName:name,isGroup:true,isFahmesGroup:true,
          damage,damagePercent:grandTotalDamage>0?damage/grandTotalDamage*100:0,
          currentHp:0,maxHp:0,hpPercent:0,hasHpUpdates:false,
          startTime:stage.startUnixSeconds,endTime:stage.endUnixSeconds,
          targets:members,bossMembers:[],fahmesLevelScopes:scopes,phaseSections:[],miscellaneous:[]});
        members.forEach(t=>groupedTargetIds.add(t.id));
      }

      // Group observed Glenn minor enemies by inactivity-separated combat
      // segments. These are not guaranteed instance identifiers; the session
      // does not always include a reliable dungeon entry/exit signal.
      const glennMobs = targets
        .filter(t => isGlennMinorRace(t.raceId) && !groupedTargetIds.has(t.id))
        .sort((a, b) => Number(a.startTime || 0) - Number(b.startTime || 0) || a.id.localeCompare(b.id));
      for (const [segmentIndex, run] of partitionMinorRuns(glennMobs,findGlennAttempts(fightSummary.targets || {})).entries()) {
        const segmentMobs=run.members;
        const mobDamage = segmentMobs.reduce((sum, t) => sum + t.damage, 0);
        const starts = segmentMobs.map(t => Number(t.startTime)).filter(Number.isFinite);
        const ends = segmentMobs.map(t => Number(t.endTime)).filter(Number.isFinite);
        // Present the ten known mob families as separate expandable sections
        // instead of forcing users to scroll hundreds of individual spawns.
        const byName = new Map<string, typeof glennMobs>();
        for (const mob of segmentMobs) {
          const name = (mob.rawName || mob.name || 'Unknown Glenn enemy').trim();
          if (!byName.has(name)) byName.set(name, []);
          byName.get(name)!.push(mob);
        }
        const mobSections = [...byName.entries()]
          .sort(([a], [b]) => a.localeCompare(b))
          .map(([name, members]) => ({
            id: `${GLENN_MINOR_GROUP_ID}_${segmentIndex}_${name.replace(/[^a-z0-9]/gi, '_')}`,
            label: name,
            targets: members,
            damage: members.reduce((sum, mob) => sum + mob.damage, 0),
          }));
        groupedList.push({
          id: `${GLENN_MINOR_GROUP_ID}_${segmentIndex}`,
          name: `Glenn Bearna — ${run.name} · Minor Enemies`, 
          rawName: `Glenn Bearna — ${run.name} · Minor Enemies`, 
          isGroup: true,
          isGlennMinorGroup: true,
          damage: mobDamage,
          damagePercent: grandTotalDamage > 0 ? mobDamage / grandTotalDamage * 100 : 0,
          currentHp: 0, maxHp: 0, hpPercent: 0, hasHpUpdates: false,
          startTime: starts.length ? Math.min(...starts) : undefined,
          endTime: ends.length ? Math.max(...ends) : undefined,
          targets: segmentMobs, bossMembers: [], phaseSections: mobSections,
          miscellaneous: [],
        });
        segmentMobs.forEach(t => groupedTargetIds.add(t.id));
      }

      // Crom Bas minor enemies (pre-boss trash) stay separate from boss HP.
      const cromMobs = targets.filter(t=>CROM_MINOR_RACES.has(Number(t.raceId)) && !groupedTargetIds.has(t.id) && !otherDungeonOwnedIds.has(t.id))
        .sort((a,b)=>Number(a.startTime||0)-Number(b.startTime||0));
      // Disciples before Glas are not Glas mechanics. Put them in the minor
      // folder unless a boss mechanic scope has already claimed them.
      const prepDisciples = targets.filter(t=>CROM_DISCIPLE_RACES.has(Number(t.raceId)) && Number(t.raceId)!==193816 && !groupedTargetIds.has(t.id) && !otherDungeonOwnedIds.has(t.id));
      const cromMinorTargets = [...cromMobs,...prepDisciples];
      for (const [segmentIndex, run] of partitionMinorRuns(cromMinorTargets,findCromAttempts(fightSummary.targets || {})).entries()) {
        const segmentMobs=run.members;
        const byName = new Map<string,typeof cromMinorTargets>();
        for(const mob of segmentMobs){
          const name=(mob.rawName||mob.name||'Crom enemy').trim();
          if(!byName.has(name))byName.set(name,[]);
          byName.get(name)!.push(mob);
        }
        const sections = [...byName.entries()].sort(([a],[b])=>a.localeCompare(b)).map(([name,ts])=>({
          id:CROM_MINOR_GROUP_ID+'_'+segmentIndex+'_'+name.replace(/[^a-z0-9]/gi,'_'), label:name,
          targets:ts, damage:ts.reduce((sum,t)=>sum+t.damage,0),
        }));
        const damage=segmentMobs.reduce((sum,t)=>sum+t.damage,0);
        groupedList.push({id:CROM_MINOR_GROUP_ID+'_'+segmentIndex,name:`Crom Bas — ${run.name} · Minor Enemies`,rawName:`Crom Bas — ${run.name} · Minor Enemies`, 
          isGroup:true,isCromMinorGroup:true,damage,damagePercent:grandTotalDamage>0?damage/grandTotalDamage*100:0,
          currentHp:0,maxHp:0,hpPercent:0,hasHpUpdates:false,targets:segmentMobs,
          bossMembers:[],phaseSections:sections,miscellaneous:[]});
        segmentMobs.forEach(t=>groupedTargetIds.add(t.id));
      }

      // Rabbie Phantasm: one selectable run folder, with boss-only damage,
      // grouped boss-room Succubi, and individually browsable minor mobs.
      for (const run of findPhantasmRuns(fightSummary.targets || {})) {
        const bossGroup = groupedList.find(g => g.id === `group_${run.queenId}`);
        if (!bossGroup) continue;
        const minorTargets = targets.filter(t => run.minorIds.includes(t.id));
        const byName = new Map<string, typeof minorTargets>();
        for (const mob of minorTargets) {
          const name = (mob.rawName || mob.name || 'Phantasm enemy').trim();
          if (!byName.has(name)) byName.set(name, []);
          byName.get(name)!.push(mob);
        }
        const minorDamage = minorTargets.reduce((n, t) => n + t.damage, 0);
        const minorGroup = {
          id: PHANTASM_MINOR_PREFIX + run.index,
          name: 'Minor Enemies', rawName: 'Minor Enemies',
          isGroup: true, isPhantasmMinorGroup: true,
          damage: minorDamage, damagePercent: grandTotalDamage > 0 ? minorDamage / grandTotalDamage * 100 : 0,
          currentHp: 0, maxHp: 0, hpPercent: 0, hasHpUpdates: false,
          targets: minorTargets, bossMembers: [],
          phaseSections: [...byName.entries()].sort(([a], [b]) => a.localeCompare(b)).map(([name, entries]) => ({
            id: `${PHANTASM_MINOR_PREFIX}${run.index}_${name.replace(/[^a-z0-9]/gi, '_')}`,
            label: name, targets: entries, damage: entries.reduce((n, t) => n + t.damage, 0),
          })), miscellaneous: [],
        };
        // Normal Phantasm summons Fiends/Wraiths during the Queen fight.
        // They are part of the run, but not Queen-only damage or HP.
        const bossRoomTargets = targets.filter(t => run.bossRoomIds.includes(t.id));
        const bossRoomByName = new Map<string, typeof bossRoomTargets>();
        for (const mob of bossRoomTargets) {
          const name = (mob.rawName || mob.name || 'Boss-room enemy').trim();
          if (!bossRoomByName.has(name)) bossRoomByName.set(name, []);
          bossRoomByName.get(name)!.push(mob);
        }
        const bossRoomDamage = bossRoomTargets.reduce((n, t) => n + t.damage, 0);
        const bossRoomGroup = {
          id: PHANTASM_BOSS_ROOM_PREFIX + run.index,
          name: 'Boss-room Enemies', rawName: 'Boss-room Enemies',
          isGroup: true, isPhantasmMinorGroup: true,
          damage: bossRoomDamage, damagePercent: grandTotalDamage > 0 ? bossRoomDamage / grandTotalDamage * 100 : 0,
          currentHp: 0, maxHp: 0, hpPercent: 0, hasHpUpdates: false,
          targets: bossRoomTargets, bossMembers: [],
          phaseSections: [...bossRoomByName.entries()].sort(([a], [b]) => a.localeCompare(b)).map(([name, entries]) => ({
            id: `${PHANTASM_BOSS_ROOM_PREFIX}${run.index}_${name.replace(/[^a-z0-9]/gi, '_')}`,
            label: name, targets: entries, damage: entries.reduce((n, t) => n + t.damage, 0),
          })), miscellaneous: [],
        };
        bossGroup.name = bossGroup.rawName = 'Bosses — Succubus Queen';
        const runTargets = targets.filter(t => run.entityIds.includes(t.id));
        const runDamage = runTargets.reduce((n, t) => n + t.damage, 0);
        groupedList.push({
          id: PHANTASM_RUN_PREFIX + run.index,
          name: `Rabbie Phantasm ${run.index}`, rawName: `Rabbie Phantasm ${run.index}`,
          isGroup: true, isDungeonRunGroup: true,
          runChildren: [bossGroup, ...(bossRoomTargets.length ? [bossRoomGroup] : []), ...(minorTargets.length ? [minorGroup] : [])],
          targets: runTargets, bossMembers: [],
          damage: runDamage, damagePercent: grandTotalDamage > 0 ? runDamage / grandTotalDamage * 100 : 0,
          currentHp: 0, maxHp: 0, hpPercent: 0, hasHpUpdates: false,
          startTime: run.startUnixSeconds, phaseSections: [], miscellaneous: [],
        });
        groupedTargetIds.add(run.queenId);
        run.bossAddIds.forEach(id => groupedTargetIds.add(id));
        run.bossRoomIds.forEach(id => groupedTargetIds.add(id));
        run.minorIds.forEach(id => groupedTargetIds.add(id));
        groupedList.splice(groupedList.indexOf(bossGroup), 1);
      }

      // Provisional high-tier run folders. Race membership is known, but the
      // relationship to boss HP thresholds/mechanics is not; keep boss-only
      // damage separate and group each observed minor segment by enemy name.
      for (const run of findHighTierRuns(fightSummary.targets || {})) {
        const mobs = targets.filter(t => run.minorIds.includes(t.id));
        if (!mobs.length) continue;
        const byName = new Map<string, typeof mobs>();
        for (const mob of mobs) {
          const name=(mob.rawName || mob.name || 'Enemy').trim();
          if (!byName.has(name)) byName.set(name, []);
          byName.get(name)!.push(mob);
        }
        const damage=mobs.reduce((sum,t)=>sum+t.damage,0);
        const minorGroup={
          id:run.minorGroupId, name:'Minor Enemies', rawName:'Minor Enemies',
          isGroup:true, damage, damagePercent:grandTotalDamage>0?damage/grandTotalDamage*100:0,
          currentHp:0,maxHp:0,hpPercent:0,hasHpUpdates:false, targets:mobs,bossMembers:[],
          phaseSections:[...byName.entries()].sort(([a],[b])=>a.localeCompare(b)).map(([name,ts])=>({
            id:run.minorGroupId+'_'+name.replace(/[^a-z0-9]/gi,'_'),label:name,targets:ts,
            damage:ts.reduce((sum,t)=>sum+t.damage,0),
          })), miscellaneous:[],
        };
        const bossGroup=run.bossId ? groupedList.find(g=>g.id==='group_'+run.bossId) : null;
        if (bossGroup) {
          const ids=new Set(run.entityIds);
          const runTargets=targets.filter(t=>ids.has(t.id));
          const total=runTargets.reduce((sum,t)=>sum+t.damage,0);
          groupedList.push({
            id:run.runGroupId,name:run.name,rawName:run.name,isGroup:true,isDungeonRunGroup:true,
            runChildren:[bossGroup,minorGroup],targets:runTargets,bossMembers:[],
            damage:total,damagePercent:grandTotalDamage>0?total/grandTotalDamage*100:0,
            currentHp:0,maxHp:0,hpPercent:0,hasHpUpdates:false,
            startTime:run.startTime,phaseSections:[],miscellaneous:[],
          });
          groupedList.splice(groupedList.indexOf(bossGroup),1);
        } else {
          minorGroup.name=minorGroup.rawName=run.name+' · Minor Enemies';
          groupedList.push(minorGroup);
        }
        mobs.forEach(t=>groupedTargetIds.add(t.id));
      }

      // Present boss, opening, and minor-enemy scopes under one run folder.
      // Keep each original group intact so boss-only views, phase mechanics,
      // and individual target selection continue to behave as before.
      const runFolders = findDungeonRunFolders(fightSummary.targets || {});
      const groupedById = new Map(groupedList.map(g => [g.id, g]));
      const claimedGroups = new Set<string>();
      for (const run of runFolders) {
        const children = run.groupIds.map(id => groupedById.get(id)).filter(Boolean);
        if (!children.length) continue;
        for (const child of children) claimedGroups.add(child.id);
        const runMemberIds = new Set(getGroupMemberTargetIds(run.id, fightSummary));
        const runTargets = targets.filter(t => runMemberIds.has(t.id));
        const damage = runTargets.reduce((sum, t) => sum + t.damage, 0);
        groupedList.push({
          id: run.id, name: run.name, rawName: run.name,
          isGroup: true, isDungeonRunGroup: true,
          runChildren: children, targets: runTargets, bossMembers: [],
          damage, damagePercent: grandTotalDamage > 0 ? damage / grandTotalDamage * 100 : 0,
          currentHp: 0, maxHp: 0, hpPercent: 0, hasHpUpdates: false,
          startTime: run.startTime, phaseSections: [], miscellaneous: [],
        });
      }
      for (let i = groupedList.length - 1; i >= 0; i--) {
        if (claimedGroups.has(groupedList[i].id)) groupedList.splice(i, 1);
      }

      // Filter out all targets that have been grouped from the top level
      const ungroupedTargets = targets.filter(t => !groupedTargetIds.has(t.id));

      const combined = [...ungroupedTargets, ...groupedList];

      combined.sort((a, b) => {
        // Keep the encounter HP accessible without scrolling through adds.
        if (a.isGroup !== b.isGroup) return a.isGroup ? -1 : 1;
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
      for (const root of targetList.value) {
        if (root.id === selectedTargetId.value) return root.rawName || root.name;
        for (const item of root.runChildren || [root]) {
        const bossScope = [...(item.glennBossScopes || []),...(item.cromBossScopes || []),...(item.albyWaveScopes || []),...(item.fahmesLevelScopes || []),...(item.phantasmBossScopes || [])].find((scope: any) => scope.id === selectedTargetId.value);
        if (bossScope) return bossScope.name;
        if (item.id === selectedTargetId.value) {
          const name = item.rawName || item.name;
          return item.raceId ? `${name} (${item.raceId})` : name;
        }
        if (item.isGroup && item.targets) {
          const sub = item.targets.find((t: any) => t.id === selectedTargetId.value);
          if (sub) {
            const name = sub.rawName || sub.name;
            return sub.raceId ? `${name} (${sub.raceId})` : name;
          }
        }
        }
      }
      const target = fightSummary.targets[selectedTargetId.value];
      if (target) {
        return target.raceId ? `${target.name} (${target.raceId})` : target.name;
      }
      return "Unknown Target";
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
      for (const root of targetList.value) {
        if (root.id === selectedTargetId.value) return root.damage;
        for (const item of root.runChildren || [root]) {
        const bossScope = [...(item.glennBossScopes || []),...(item.cromBossScopes || []),...(item.albyWaveScopes || []),...(item.fahmesLevelScopes || []),...(item.phantasmBossScopes || [])].find((scope: any) => scope.id === selectedTargetId.value);
        if (bossScope) return bossScope.damage;
        if (item.id === selectedTargetId.value) return item.damage;
        if (item.isGroup && item.targets) {
          const sub = item.targets.find((t: any) => t.id === selectedTargetId.value);
          if (sub) return sub.damage;
        }
        }
      }
      return 0;
    });

    const selectedTargetHp = computed(() => {
      if (!selectedTargetId.value || isGroupTargetId(selectedTargetId.value)) return null;
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
        runFolderParent.value = null;
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

    const selectGroupAll = (groupItem: any) => {
      selectedTargetId.value = groupItem.id;
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
          const includesSelection = (item: any) => item.id === selectedTargetId.value ||
            item.glennBossScopes?.some((scope: any) => scope.id === selectedTargetId.value) ||
            item.albyWaveScopes?.some((scope: any) => scope.id === selectedTargetId.value) ||
            item.fahmesLevelScopes?.some((scope: any) => scope.id === selectedTargetId.value) ||
            item.cromBossScopes?.some((scope: any) => scope.id === selectedTargetId.value) ||
            item.phantasmBossScopes?.some((scope: any) => scope.id === selectedTargetId.value) ||
            item.targets?.some((sub: any) => sub.id === selectedTargetId.value);
          const folder = targetList.value.find(item => item.isDungeonRunGroup &&
            (item.id === selectedTargetId.value || item.runChildren?.some(includesSelection)));
          if (folder && folder.id !== selectedTargetId.value) {
            const child = folder.runChildren.find(includesSelection);
            expandedGroup.value = child || folder;
            runFolderParent.value = child ? folder : null;
          } else {
            const groupContainingTarget = targetList.value.find(item => item.isGroup && includesSelection(item));
            expandedGroup.value = groupContainingTarget || null;
            runFolderParent.value = null;
          }
        } else {
          expandedGroup.value = null;
        }
      }
    });

    watch(activeSessionId, () => {
      openPhaseSections.value = [];
      expandedGroup.value = null;
      runFolderParent.value = null;
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
      showHealingTab,
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
      runFolderParent,
      openRunChild,
      backToRunFolder,
      openPhaseSections,
      togglePhaseSection,
      selectedTargetName,
      selectedTargetSeenAppear,
      selectedTargetDamage,
      selectedTargetHp,
      handleItemClick,
      selectGroupTarget,
      selectGroupAll,

      menuHeight,
      selectedTargetConditions: computed(() => {
        if (!selectedTargetId.value) return undefined;
        if (isGroupTargetId(selectedTargetId.value)) {
          const memberIds = getGroupMemberTargetIds(selectedTargetId.value, fightSummary);
          return aggregateGroupConditions(fightSummary.targets, memberIds);
        }
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
.vertrag-expand-button { display: inline-flex; align-items: center; justify-content: center; color: inherit; background: transparent; border: 0; padding: 5px; border-radius: 4px; cursor: pointer; }
.vertrag-expand-button:hover, .vertrag-expand-button:focus-visible { background: rgba(255,255,255,.14); outline: 1px solid rgba(255,255,255,.35); }
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
