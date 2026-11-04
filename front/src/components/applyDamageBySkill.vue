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
    <div 
      @dblclick="isolateSinglePlayer(item.id)" 
      class="cursor-pointer d-flex align-center select-none w-100 h-100"
      title="Double-click to isolate this player & open scatter plot"
      style="min-height: 28px;"
    >
      <span v-if="item.talentIdentities?.length" class="d-inline-flex align-center mr-2 ga-1" :title="item.talentIdentities.map((a: any) => a.name).join(' / ')">
        <img v-for="arcana in item.talentIdentities" :key="arcana.name" :src="arcana.icon" :alt="arcana.name" width="28" height="28" />
      </span>
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
        <span class="damage-bar-val">
          {{ formatAbbreviated(item.totalDamage) }}
        </span>
        <div class="damage-bar-wrapper">
          <v-tooltip v-if="savedAnalysis && showGhostRunner" location="top">
            <template #activator="{ props: ghostProps }">
              <div v-bind="ghostProps" class="ghost-column-zone">
                <div class="ghost-column-fill" :style="{width: playerGhostWidth(item.id) + '%',zIndex: ghostDps(item.id) <= item.dps ? 2 : 0}"></div>
                <div class="damage-bar-fill ghost-current-fill" :style="{width: playerAverageWidth(item) + '%',background:getPlayerBarBackground(item, 100)}"></div>
              </div>
            </template>
            <span><v-icon size="x-small">mdi-ghost</v-icon> {{ formatNumber(ghostDps(item.id)) }} productive DPS · {{ formatNumber(item.dps) }} average DPS</span>
          </v-tooltip>
          <div v-else class="damage-bar-fill" :style="{width:(playerDisplayData[0]?.totalDamage ? (item.totalDamage / playerDisplayData[0].totalDamage) * 100 : 0) + '%',background:getPlayerBarBackground(item, 100)}"></div>
        </div>
        <span class="damage-bar-pct">
          {{ totalDamageForView > 0 ? ((item.totalDamage / totalDamageForView) * 100).toFixed(1) : '0.0' }}%
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

          <div v-if="savedAnalysis" class="ma-3 pa-3 rounded border-sm">
            <div v-if="showGhostRunner" class="text-subtitle-2 mb-2"><v-icon size="small" class="mr-1">mdi-ghost</v-icon> Ghost Runner · {{ formatNumber(ghostDps(item.id)) }} productive DPS</div>
            <div class="d-flex flex-wrap ga-2 mb-2">
              <v-menu v-for="(flag, index) in deathSnapshots(item.id)" :key="index" open-on-hover :close-on-content-click="false" location="bottom" :max-width="'min(90vw, 650px)'">
                <template #activator="{ props: menuProps }">
                  <v-btn v-bind="menuProps" size="small" variant="tonal" color="error" prepend-icon="mdi-flag">Death {{ index + 1 }}</v-btn>
                </template>
                <v-card class="death-snapshot-card" max-height="min(75vh, 550px)" style="overflow-y:auto">
                  <v-card-title>Death {{ index + 1 }} · {{ flag.time.toFixed(1) }}s</v-card-title>
                  <v-card-text>
                    <div class="font-weight-bold">{{ formatNumber(flag.dps) }} DPS · {{ flag.hpm.toFixed(1) }} hits/min</div>
                    <div class="death-snapshot-details mt-2">
                      <div v-for="skill in flag.skills" :key="skill.id" class="d-flex justify-space-between ga-4 text-caption py-1">
                        <span>{{ getSkillName(skill.id) }}</span><span>{{ formatNumber(skill.dps) }} DPS · {{ skill.hpm.toFixed(1) }} HPM</span>
                      </div>
                    </div>
                  </v-card-text>
                </v-card>
              </v-menu>
            </div>
            <div v-if="showGhostRunner" class="text-caption text-medium-emphasis">Dark bar = productive Ghost DPS; normal bar = session average DPS. Hover the ghost marker to compare.</div>
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
                <v-icon v-if="skillItem.id === 9999" class="mr-2" size="24" color="amber">mdi-paw</v-icon>
                <img
                  v-else
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
                <v-tooltip v-if="skillProgress(item.id, skillItem.id)" location="top" open-delay="150">
                  <template #activator="{ props: progressProps }">
                    <v-icon v-bind="progressProps" color="success" size="small" class="ml-1" aria-label="New per-skill DPS personal best">mdi-arrow-up-bold</v-icon>
                  </template>
                  <div class="font-weight-bold"><v-icon color="success" size="small">mdi-arrow-up-bold</v-icon> New skill DPS personal best!</div>
                  <div>Previous best: {{ formatNumber(skillProgress(item.id, skillItem.id)!.previousBest) }} DPS</div>
                  <div>Current: {{ formatNumber(skillProgress(item.id, skillItem.id)!.current) }} DPS</div>
                  <div>Improved by {{ skillProgress(item.id, skillItem.id)!.percent.toFixed(1) }}%</div>
                </v-tooltip>
                <v-tooltip v-if="savedAnalysis && showGhostRunner" location="top">
                  <template #activator="{ props: ghostProps }"><v-icon v-bind="ghostProps" size="x-small" class="ml-2 ghost-mark">mdi-ghost</v-icon></template>
                  <span><v-icon size="x-small">mdi-ghost</v-icon> {{ formatNumber(ghostDps(item.id, skillItem.id)) }} productive DPS · {{ formatNumber(skillItem.totalDamage / Math.max(currentEncounterDuration, 0.001)) }} average DPS</span>
                </v-tooltip>
              </div>
            </template>

            <template v-slot:[`item.uses`]="{ item: skillItem }">
              {{ formatNumber(skillItem.uses) }}
            </template>

            <template v-slot:[`item.totalDamage`]="{ item: skillItem }">
              <div v-if="dpsMeterFillMode === 'column'" class="damage-bar-cell skill-bar-cell">
                <span class="damage-bar-val">
                  {{ formatNumber(skillItem.totalDamage) }}
                </span>
                <div class="damage-bar-wrapper">
                  <v-tooltip v-if="savedAnalysis && showGhostRunner" location="top">
                    <template #activator="{ props: ghostProps }">
                      <div v-bind="ghostProps" class="ghost-column-zone">
                        <div class="ghost-column-fill" :style="{width: ghostBarWidth(item.id, skillItem.id, item) + '%',zIndex: ghostDps(item.id, skillItem.id) <= skillItem.totalDamage / Math.max(currentEncounterDuration, 0.001) ? 2 : 0}"></div>
                        <div class="damage-bar-fill ghost-current-fill" :style="{width: averageBarWidth(skillItem, item) + '%',background:getSkillBarPaint(skillItem.id, item)}"></div>
                      </div>
                    </template>
                    <span><v-icon size="x-small">mdi-ghost</v-icon> {{ formatNumber(ghostDps(item.id,skillItem.id)) }} productive DPS · {{ formatNumber(skillItem.totalDamage / Math.max(currentEncounterDuration,0.001)) }} average DPS</span>
                  </v-tooltip>
                  <div v-else class="damage-bar-fill" :style="{width:(getSkillBreakdown(item)[0]?.totalDamage ? (skillItem.totalDamage / getSkillBreakdown(item)[0].totalDamage) * 100 : 0)+'%',background:getSkillBarPaint(skillItem.id, item)}"></div>
                </div>
                <span class="damage-bar-pct">
                  {{ item.totalDamage > 0 ? ((skillItem.totalDamage / item.totalDamage) * 100).toFixed(1) : '0.0' }}%
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
            
            <span v-if="item.talentIdentities?.length" class="d-inline-flex align-center mr-2 ga-1" :title="item.talentIdentities.map((a: any) => a.name).join(' / ')">
              <img v-for="arcana in item.talentIdentities" :key="arcana.name" :src="arcana.icon" :alt="arcana.name" width="32" height="32" />
            </span>
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

              <div v-if="savedAnalysis" class="mb-3">
                <div v-if="showGhostRunner" class="text-caption"><v-icon size="x-small">mdi-ghost</v-icon> Ghost Runner: {{ formatNumber(ghostDps(item.id)) }} productive DPS</div>
                <v-menu v-for="(flag, index) in deathSnapshots(item.id)" :key="index" open-on-hover location="bottom" :max-width="'min(90vw, 650px)'">
                  <template #activator="{ props: menuProps }"><v-btn v-bind="menuProps" size="x-small" color="error" variant="tonal" class="mr-2 mt-1" prepend-icon="mdi-flag">Death {{ index + 1 }}</v-btn></template>
                  <v-card class="death-snapshot-card" max-height="min(75vh, 550px)" style="overflow-y:auto"><v-card-title>Death {{ index + 1 }} · {{ flag.time.toFixed(1) }}s</v-card-title><v-card-text><div>{{ formatNumber(flag.dps) }} DPS · {{ flag.hpm.toFixed(1) }} HPM</div><div class="death-snapshot-details"><div v-for="skill in flag.skills" :key="skill.id" class="d-flex justify-space-between ga-3 text-caption py-1"><span>{{ getSkillName(skill.id) }}</span><span>{{ formatNumber(skill.dps) }} DPS · {{ skill.hpm.toFixed(1) }} HPM</span></div></div></v-card-text></v-card>
                </v-menu>
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
                                  <v-icon v-if="skillItem.id === 9999" v-bind="props" size="24" color="amber" class="mr-1" style="vertical-align: middle;">mdi-paw</v-icon>
                                  <img
                                     v-else
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
                          <v-tooltip v-if="skillProgress(item.id, skillItem.id)" location="top">
                            <template #activator="{ props: progressProps }"><v-icon v-bind="progressProps" size="small" color="success" class="ml-1">mdi-arrow-up-bold</v-icon></template>
                            <div class="font-weight-bold">New skill DPS personal best!</div>
                            <div>Previous best: {{ formatNumber(skillProgress(item.id, skillItem.id)!.previousBest) }} DPS</div>
                            <div>Current: {{ formatNumber(skillProgress(item.id, skillItem.id)!.current) }} DPS</div>
                            <div>Improved by {{ skillProgress(item.id, skillItem.id)!.percent.toFixed(1) }}%</div>
                          </v-tooltip>
                          <v-tooltip v-if="savedAnalysis && showGhostRunner" location="top">
                            <template #activator="{ props: ghostProps }"><v-icon v-bind="ghostProps" size="x-small" class="ghost-mark">mdi-ghost</v-icon></template>
                            <span><v-icon size="x-small">mdi-ghost</v-icon> {{ formatNumber(ghostDps(item.id, skillItem.id)) }} productive DPS · {{ formatNumber(skillItem.totalDamage / Math.max(currentEncounterDuration, 0.001)) }} average DPS</span>
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
      <v-btn v-if="savedAnalysis" size="small" variant="tonal" :color="showGhostRunner ? 'primary' : undefined"
        prepend-icon="mdi-ghost"
        @click="showGhostRunner = !showGhostRunner" class="mr-2">
        Ghost {{ showGhostRunner ? 'On' : 'Off' }}
      </v-btn>
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
  padding-bottom: 420px !important;
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
  text-align: left;
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
  text-align: right;
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
import { getPersonalBestHistoricalSummary, getPersonalBestHistorySessions } from "@/apicall";
import { sessions, personalBestSessionLimit, personalBestHistoryFolder } from "@/store";
import { getMabiNameColor, getClassColor } from "@/util";
import {
  fightSummary,
  activeSessionId,
  selectedTargetId,
} from "@/store";
import { SkillStats, DamageBreakdown, type PlayerStats } from "@/protocols";
import { resolvePlayerArcana, arcanaBarBackground, getSkillArcana } from "@/utils/arcanaEncounter";
import {
  isGroupTargetId,
  getGroupMemberTargetIds,
  aggregatePlayerDamageForGroup,
  getEncounterDurationForTargets,
} from "@/utils/targetGrouping";
import TargetConditionView from "./TargetConditionView.vue";
import { hiddenPlayers, toggleHiddenPlayer, setAllHiddenPlayers, globalHideMode, showClassColorsForVisiblePlayers, listSkillMetrics, cardSkillMetrics, dpsMeterFillMode } from "@/store";

const props = defineProps<{
  attackerNameMap?: { [id: string]: string };
}>();

// A mixed bar is used only when class colors are enabled; otherwise preserve
// the user's selected character-name coloring mode.
const getPlayerBarBackground = (item: any, percentage: number) => {
  const isHidden = globalHideMode.value || hiddenPlayers.has(item.id);
  if ((isHidden || showClassColorsForVisiblePlayers.value) && item.talentIdentities?.length > 1)
    return arcanaBarBackground(item.talentIdentities, percentage);
  return arcanaBarBackground([], percentage, getPlayerColor(item));
};

// A mapped Arcana skill always gets its own customized Arcana color.
// Shared skills follow the current player's resolved Arcana or mixed gradient.
const getSkillBarPaint = (skillId: number, playerItem: any): string => {
  const arcana = getSkillArcana(skillId);
  if (arcana) return arcana.color;
  const identities = playerItem?.talentIdentities || [];
  if (identities.length > 1) return arcanaBarBackground(identities);
  if (identities.length === 1) return identities[0].color;
  return playerItem ? getPlayerColor(playerItem) : '#808080';
};

const getPlayerColor = (item: any) => {
  const isHidden = globalHideMode.value || hiddenPlayers.has(item.id);
  const shouldUseClassColor = isHidden || showClassColorsForVisiblePlayers.value;
  return shouldUseClassColor ? getClassColor(item.talentName, item.talentColor) : getMabiNameColor(item.name);
};

// --- STATE ---
const skillNameMap = inject("skillNameMap") as any;
const condNameMap = inject("condNameMap") as any;
const expanded = ref<string[]>([]);
const savedAnalysis = computed(() => activeSessionId.value !== "live");
const showGhostRunner = ref(false);

const rawPlayer = (id: string) => (fightSummary.players as any)[id] as any;
const skillEvents = (id: string) => (rawPlayer(id)?.skillUses || []) as any[];
const median = (numbers: number[]) => {
  const sorted = numbers.filter(n => n > 0).sort((a, b) => a - b);
  if (!sorted.length) return 0;
  const mid = Math.floor(sorted.length / 2);
  return sorted.length % 2 ? sorted[mid] : (sorted[mid - 1] + sorted[mid]) / 2;
};
// Ghost Runner: productive (nonzero) 15-second damage buckets, median DPS.
// Shared visual comparison for the overall player bar and each skill bar.
const ghostCache = new Map<string, { events: any[]; length: number; value: number }>();
const ghostDps = (playerId: string, skillId?: number): number => {
  const events = skillEvents(playerId);
  const key = `${activeSessionId.value}:${playerId}:${skillId === undefined ? 'all' : skillId}`;
  const cached = ghostCache.get(key);
  if (cached && cached.events === events && cached.length === events.length) return cached.value;
  const buckets = new Map<number, number>();
  for (const event of events) {
    const damage = skillId === undefined ? Number(event.totalDmg || 0)
      : (event.skills || []).filter((skill: any) => Number(skill.skillId) === Number(skillId)).reduce((sum: number, skill: any) => sum + Number(skill.damage || 0), 0);
    if (damage <= 0) continue;
    const bucket = Math.floor(Number(event.timestamp || 0) / 15);
    buckets.set(bucket, (buckets.get(bucket) || 0) + damage);
  }
  const value = median([...buckets.values()].map(d => d / 15));
  ghostCache.set(key, { events, length: events.length, value });
  return value;
};
const ghostSkillMax = (playerId: string, playerItem: any): number => {
  const avgMax = (getSkillBreakdown(playerItem) || []).reduce((max: number, sk: any) =>
    Math.max(max, Number(sk.totalDamage || 0) / Math.max(currentEncounterDuration.value, 0.001)), 0);
  const ghostMax = (getSkillBreakdown(playerItem) || []).reduce((max: number, sk: any) =>
    Math.max(max, ghostDps(playerId, sk.id)), 0);
  return Math.max(avgMax, ghostMax, 0.001);
};
const averageBarWidth = (skill: any, playerItem: any): number =>
  100 * (Number(skill.totalDamage || 0) / Math.max(currentEncounterDuration.value, 0.001)) / ghostSkillMax(playerItem.id, playerItem);
const ghostBarWidth = (playerId: string, skillId: number, playerItem: any): number =>
  100 * ghostDps(playerId, skillId) / ghostSkillMax(playerId, playerItem);

const topPlayerDps = (): number => Math.max(0.001, ...playerDisplayData.value.flatMap((p: any) => [Number(p.dps || 0), ghostDps(p.id)]));
const playerGhostWidth = (id: string): number => 100 * ghostDps(id) / topPlayerDps();
const playerAverageWidth = (player: any): number => 100 * Number(player.dps || 0) / topPlayerDps();
const deathSnapshots = (playerId: string) => {
  const player = rawPlayer(playerId);
  if (!player) return [];
  const start = Number(fightSummary.startTime || 0);
  const events = skillEvents(playerId);
  return [...(player.deaths || [])].sort((a: number, b: number) => a - b).map((death: number) => {
    const time = Math.max(0.001, Number(death) - start);
    const skills = new Map<number, { id: number; damage: number; hits: number }>();
    for (const event of events) {
      if (Number(event.timestamp) > time) continue;
      for (const sk of event.skills || []) {
        const id = Number(sk.skillId);
        const entry = skills.get(id) || { id, damage: 0, hits: 0 };
        entry.damage += Number(sk.damage || 0);
        entry.hits += Number(sk.hitCount || 0);
        skills.set(id, entry);
      }
    }
    const breakdown = [...skills.values()].map(sk => ({ id: sk.id, dps: sk.damage / time, hpm: sk.hits * 60 / time }));
    return { time, dps: breakdown.reduce((sum, sk) => sum + sk.dps, 0), hpm: breakdown.reduce((sum, sk) => sum + sk.hpm, 0), skills: breakdown.sort((a, b) => b.dps - a.dps) };
  });
};
const viewMode = ref<"table" | "cards">("table"); 

watch(activeSessionId, () => {
  ghostCache.clear();
  showGhostRunner.value = false;
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
  // Arcana follows the selected encounter, not the first skill of the session.
  const arcanaFor = (p: PlayerStats) => resolvePlayerArcana(p, fightSummary, selectedTargetId.value);
  const players = Object.values(fightSummary.players);
  if (!selectedTargetId.value) {
    // Overall view
    return players
      .map((p) => ({
        ...p.overallStats,
        id: p.id,
        name: p.name,
        talentIcon: arcanaFor(p).icon,
        talentIdentities: arcanaFor(p).identities,
        talentName: arcanaFor(p).name,
        talentColor: arcanaFor(p).color,
        missingAppearPacket: p.missingAppearPacket,
      }))
      .filter((p) => p.totalDamage > 0)
      .sort((a, b) => b.totalDamage - a.totalDamage);
  } else {
    // Target-specific view
    const isGroup = isGroupTargetId(selectedTargetId.value);
    const memberIds = isGroup
      ? getGroupMemberTargetIds(selectedTargetId.value, fightSummary)
      : [];

    return players
      .map((p) => {
        const breakdown = isGroup
          ? aggregatePlayerDamageForGroup(p, memberIds)
          : p.damageByTarget[selectedTargetId.value];
        return {
          ...breakdown,
          id: p.id,
          name: p.name,
          talentIcon: arcanaFor(p).icon,
          talentIdentities: arcanaFor(p).identities,
          talentName: arcanaFor(p).name,
          talentColor: arcanaFor(p).color,
          missingAppearPacket: p.missingAppearPacket,
        };
      })
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

// Personal Best Progress: saved fights only; annotate skill rows, never the player total.
// Scan configurable prior sessions for the same player/target, ignore Harmonic Saint,
// and celebrate only skill-DPS increases over the best eligible prior value.
type SkillProgress = { current: number; previousBest: number; percent: number; samples: number };
const progressByPlayer = ref<Record<string, Record<number, SkillProgress>>>({});
const summaryPromises = new Map<string, Promise<any>>();
const historyListPromises = new Map<string, Promise<any[]>>();
let progressGeneration = 0;
const isHarmonicSaint = (name?: string): boolean =>
  (name || "").trim().toLowerCase() === "harmonic saint";
const getHistoricalSummary = (id: string, folder: string): Promise<any> => {
  const key = folder + "\0" + id;
  if (!summaryPromises.has(key)) summaryPromises.set(key, getPersonalBestHistoricalSummary(id, folder));
  return summaryPromises.get(key)!;
};
// For the overall table, compare against the player's most-damaged target.
// Explicit target selections always take priority. Multi-target groups are
// deliberately excluded: they cannot be matched reliably by one target name.
const targetForProgress = (player: any): { name: string; breakdown: any } | null => {
  if (!player || (selectedTargetId.value && isGroupTargetId(selectedTargetId.value))) return null;
  const entries = Object.entries(player.damageByTarget || {}) as [string, any][];
  const selected = selectedTargetId.value
    ? entries.find(([id]) => id === selectedTargetId.value)
    : entries.sort((a, b) => Number(b[1]?.totalDamage || 0) - Number(a[1]?.totalDamage || 0))[0];
  if (!selected) return null;
  const targetName = fightSummary.targets?.[selected[0]]?.name?.trim();
  if (!targetName) return null;
  return { name: targetName, breakdown: selected[1] };
};
const durationForTarget = (summary: any, targetId: string): number => {
  let begin = Infinity, end = -Infinity;
  for (const player of Object.values(summary.players || {}) as any[]) {
    const target = player.damageByTarget?.[targetId];
    if (Number.isFinite(target?.startTime) && Number.isFinite(target?.endTime) && target.endTime > target.startTime) {
      begin = Math.min(begin, target.startTime);
      end = Math.max(end, target.endTime);
    }
  }
  return end > begin ? end - begin : Number(summary.encounterDuration || 0);
};
const calculateProgress = async (): Promise<void> => {
  const generation = ++progressGeneration;
  progressByPlayer.value = {};
  if (!savedAnalysis.value) return;
  const active = sessions.value.find(session => session.id === activeSessionId.value);
  if (!active) return;
  const folder = personalBestHistoryFolder.value.trim();
  let historicalSessions = sessions.value;
  if (folder) {
    try {
      if (!historyListPromises.has(folder)) historyListPromises.set(folder, getPersonalBestHistorySessions(folder));
      historicalSessions = await historyListPromises.get(folder)!;
    } catch (error) {
      console.warn('Personal Best history folder could not be read', folder, error);
      historyListPromises.delete(folder);
      return;
    }
    if (generation !== progressGeneration) return;
  }
  const candidates = historicalSessions
    .filter(session => session.id !== active.id && session.id !== 'live' && session.startTime < active.startTime)
    .sort((a, b) => b.startTime - a.startTime);
  const computedProgress: Record<string, Record<number, SkillProgress>> = {};
  // A selected fight is analysed once. Historical summaries are cached across players.
  for (const player of Object.values(fightSummary.players) as any[]) {
    if (generation !== progressGeneration) return;
    if (!player?.name || isHarmonicSaint(player.talentName)) continue;
    const currentTarget = targetForProgress(player);
    if (!currentTarget) continue;
    const currentDuration = durationForTarget(fightSummary, Object.keys(player.damageByTarget || {}).find(id =>
      fightSummary.targets?.[id]?.name === currentTarget.name && player.damageByTarget[id] === currentTarget.breakdown) || '');
    if (!(currentDuration > 0)) continue;
    const earlierBest = new Map<number, number>();
    let qualifyingSessions = 0;
    // The setting controls how many previous saved sessions to inspect, even
    // when a player only appears rarely. Preserve the earlier HS-only extension:
    // if we reach the limit without a non-HS appearance, continue five at a
    // time until that player is absent for a whole window or plays non-HS.
    const scanLimit = Math.max(5, Math.min(200, Number(personalBestSessionLimit.value) || 5));
    let extendHsOnly = false;
    for (let offset = 0; offset < candidates.length && qualifyingSessions < 5; offset += 5) {
      if (offset >= scanLimit && !extendHsOnly) break;
      if (generation !== progressGeneration) return;
      const window = candidates.slice(offset, Math.min(offset + 5, offset < scanLimit ? scanLimit : candidates.length));
      let seenPlayer = false;
      let seenNonHSPlayer = false;
      for (const session of window) {
        if (generation !== progressGeneration) return;
        // Metadata can identify absence, but the saved full summary is still
        // authoritative for class and per-skill damage.
        if (session.summary?.players?.length && !session.summary.players.some(p => p.name === player.name)) continue;
        let previous: any;
        try { previous = await getHistoricalSummary(session.id, folder); }
        catch (error) { console.warn('Unable to compare saved session', session.id, error); continue; }
        if (generation !== progressGeneration) return;
        const appearances = (Object.values(previous.players || {}) as any[]).filter(p => p.name === player.name);
        if (!appearances.length) continue;
        seenPlayer = true;
        const priorPlayer = appearances.find(p => !isHarmonicSaint(p.talentName));
        if (!priorPlayer) continue;
        seenNonHSPlayer = true;
        const matchingTargets = Object.entries(previous.targets || {}) as [string, any][];
        const match = matchingTargets.find(([, target]) => (target?.name || '').trim() === currentTarget.name);
        if (!match) continue;
        const priorBreakdown = priorPlayer.damageByTarget?.[match[0]];
        if (!priorBreakdown || !Number(priorBreakdown.totalDamage)) continue;
        const duration = durationForTarget(previous, match[0]);
        if (!(duration > 0)) continue;
        qualifyingSessions++;
        for (const skill of Object.values(priorBreakdown.skills || {}) as any[]) {
          const id = Number(skill.id);
          if (!Number.isFinite(id) || !(Number(skill.totalDamage) > 0)) continue;
          const dps = Number(skill.totalDamage) / duration;
          earlierBest.set(id, Math.max(earlierBest.get(id) || 0, dps));
        }
        if (qualifyingSessions >= 5) break;
      }
      // Do not stop early on missing-player windows inside the configured
      // lookback. Beyond that limit, preserve the HS-only continuation rule.
      if (offset + window.length >= scanLimit && qualifyingSessions === 0 && seenPlayer && !seenNonHSPlayer) {
        extendHsOnly = true;
      }
      if (offset >= scanLimit && (!seenPlayer || seenNonHSPlayer)) break;
    }
    const improvements: Record<number, SkillProgress> = {};
    for (const skill of Object.values(currentTarget.breakdown.skills || {}) as any[]) {
      const id = Number(skill.id);
      const prior = earlierBest.get(id);
      if (!prior || !(Number(skill.totalDamage) > 0)) continue;
      const current = Number(skill.totalDamage) / currentDuration;
      if (current > prior) improvements[id] = { current, previousBest: prior, percent: (current / prior - 1) * 100, samples: qualifyingSessions };
    }
    if (Object.keys(improvements).length) computedProgress[player.id] = improvements;
  }
  if (generation === progressGeneration) progressByPlayer.value = computedProgress;
};
const skillProgress = (playerId: string, skillId: number): SkillProgress | undefined =>
  savedAnalysis.value ? progressByPlayer.value[playerId]?.[Number(skillId)] : undefined;
watch([activeSessionId, selectedTargetId, sessions, personalBestSessionLimit, personalBestHistoryFolder, () => fightSummary.players], () => { void calculateProgress(); }, { immediate: true });

const getSkillName = (skillId: number): string => {
  if (skillId === 9999) return "Pets";
  return skillNameMap.value[skillId]?.name || `unknownSkill:${skillId}`;
};

const getSkillImageSrc = (skillId: number): string => {
  if (skillId === 9999) return "";
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

    if (savedAnalysis.value && showGhostRunner.value) {
      const avg = playerAverageWidth(item);
      const ghost = playerGhostWidth(item.id);
      return {
        class: "character-row",
        style: {
          background: ghost <= avg
            ? `linear-gradient(to right, color-mix(in srgb, ${bgColor}, black 55%) ${ghost}%, transparent ${ghost}%), linear-gradient(to right, ${bgColor} ${avg}%, transparent ${avg}%)`
            : `linear-gradient(to right, ${bgColor} ${avg}%, transparent ${avg}%), linear-gradient(to right, color-mix(in srgb, ${bgColor}, black 55%) ${ghost}%, transparent ${ghost}%)`,
          backgroundClip: 'padding-box',
        },
        title: `Ghost: ${formatNumber(ghostDps(item.id))} productive DPS | Average: ${formatNumber(item.dps)} DPS`,
      };
    }
    return {
      class: "character-row",
      style: {
        background: getPlayerBarBackground(item, percentage),
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
  if (dpsMeterFillMode.value !== "full") return {};
  if (!topSkillDamage || !skillItem.totalDamage) return { style: {} };
  const percentage = (skillItem.totalDamage / topSkillDamage) * 100;
  // Compute the fill inside the painted region: gradients always cover the
  // complete damage bar, never continue into the empty percentage.
  const paint = playerItem ? getSkillBarPaint(skillItem.id, playerItem) : getMabiNameColor(playerName);
  // CSS background-size confines a continuous multi-Arcana gradient to the
  // measured part of the row, keeping the existing length convention.
  const paintImage = paint.startsWith('linear-gradient(') ? paint : `linear-gradient(to right, ${paint}, ${paint})`;
  if (savedAnalysis.value && showGhostRunner.value && playerItem) {
    const ghostPercent = ghostBarWidth(playerItem.id, skillItem.id, playerItem);
    const avgPercent = averageBarWidth(skillItem, playerItem);
    return {
      style: {
        backgroundImage: `${paintImage}, linear-gradient(to right, rgba(0,0,0,.36) 0% ${ghostPercent}%, transparent ${ghostPercent}%)`,
        backgroundSize: `${Math.max(0, Math.min(100, avgPercent))}% 100%, 100% 100%`,
        backgroundRepeat: 'no-repeat',
        backgroundClip: 'padding-box',
      },
      title: `Ghost: ${formatNumber(ghostDps(playerItem.id, skillItem.id))} productive DPS | Average: ${formatNumber(Number(skillItem.totalDamage || 0) / Math.max(currentEncounterDuration.value, 0.001))} DPS`,
    };
  }
  return {
    style: {
      backgroundImage: paintImage,
      backgroundSize: `${Math.max(0, Math.min(100, percentage))}% 100%`,
      backgroundRepeat: 'no-repeat',
      backgroundClip: 'padding-box',
    },
  };
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
  
  const isolateSinglePlayer = (targetPlayerId: string) => {
  // Clear any active global hiding configurations
  globalHideMode.value = false;
  
  // Set all players to hidden first
  const allIds = playerDisplayData.value.map((p: any) => p.id);
  setAllHiddenPlayers(allIds, true);
  
  // Remove our specific chosen character from the hidden list to keep them visible
  hiddenPlayers.delete(targetPlayerId);
};

// (Old static headers removed, using computed headers now)
</script>

<style scoped>
 .ghost-column-zone {position:relative;width:100%;height:100%;cursor:help}
 .ghost-column-fill {position:absolute;left:0;top:0;height:100%;background:rgba(0,0,0,.58);border-radius:9999px}
 .ghost-current-fill {position:absolute;left:0;top:0;height:100%}
 .ghost-mark {opacity:.8;cursor:help}
@media (max-width: 600px) { .death-snapshot-details { display: none; } .death-snapshot-card { max-width: 90vw; } }
</style>
