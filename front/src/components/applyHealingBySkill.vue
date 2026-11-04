<template>
  <v-data-table
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
    <!-- Character Rows -->
     <template v-slot:[`item.name`]="{ item }">
    <div 
      @click.stop="toggleHiddenPlayer(item.id)" 
      class="cursor-pointer d-flex align-center select-none w-100 h-100"
      title="Click player row name to toggle visibility"
      style="min-height: 28px;"
    >
      <v-btn 
        icon 
        variant="text" 
        density="compact" 
        size="small" 
        class="mr-1" 
        @click.stop="toggleHiddenPlayer(item.id)"
      >
        <v-icon>{{ hiddenPlayers.has(item.id) ? 'mdi-eye-off-outline' : 'mdi-eye-outline' }}</v-icon>
      </v-btn>
      <span 
        class="font-weight-medium text-body-1 transition-all"
        :style="{ color: getPlayerColor(item) }"
      >
        {{ hiddenPlayers.has(item.id) ? 'Hidden' : item.name }}
      </span>
    </div>
  </template>


    <!-- Effective Heals -->
    <template v-slot:[`item.overallHealing.totalDamage`]="{ item }">
      <div class="text-right font-weight-bold">
        {{ formatAbbreviated(item.overallHealing?.totalDamage) }}
        <span class="text-grey font-weight-medium text-caption ml-1">
          ({{ totalHealingForView > 0 ? (((item.overallHealing?.totalDamage || 0) / totalHealingForView) * 100).toFixed(1) : '0.0' }}%)
        </span>
      </div>
    </template>

    <template v-slot:[`item.largestHeal`]="{ item }">
      <div class="text-right font-weight-bold">
        {{ item.largestHeal > 0 ? formatNumber(item.largestHeal) : "—" }}
      </div>
    </template>

    <!-- Overshield Volume Column Row Display -->
    <template v-slot:[`item.overallHealing.overshieldProvided`]="{ item }">
      <div class="text-right font-weight-bold text-light-green-accent-3">
        {{ item.veilPeak > 0 ? formatAbbreviated(item.overallHealing?.overshieldProvided || 0) : '—' }}
      </div>
    </template>

    <!-- Condition 999 Veil of Life properties: DMR is a percentage, not additive healing. -->
    <template v-slot:[`item.veilDmr`]='{ item }'>
      <div class="text-right"><template v-if="item.veilPeak > 0">{{ Number(item.veilDmr || 0).toFixed(2) }}% <span class="text-caption text-grey">(peak {{ Number(item.veilPeakDmr || 0).toFixed(2) }}%)</span></template>
        <span v-else class="text-grey">—</span></div>
    </template>

    <template v-slot:[`item.veilPeak`]='{ item }'>
      <div class="text-right">{{ item.veilPeak > 0 ? formatNumber(item.veilPeak) : '—' }}</div>
    </template>

    <!-- Expanded Skills Dropdown Section -->
    <template v-slot:expanded-row="{ columns, item }">
      <tr>
        <td :colspan="columns.length">
          <v-data-table
            :headers="computedSkillTableHeaders"
            :items="getSkillBreakdown(item.overallHealing)"
            density="compact"
            :items-per-page="-1"
            hide-default-footer
            class="ma-2 healing-skills-table rounded-sm elevation-0"
          >
            <template v-slot:[`item.skillName`]="{ item: skillItem }">
              <div class="d-flex align-center px-2 py-1">
                <img
                  v-if="getSkillImageSrc(skillItem.id)"
                  width="24"
                  height="24"
                  :src="getSkillImageSrc(skillItem.id)"
                  class="mr-2 rounded-sm"
                  @error="($event.target as HTMLImageElement).style.display = 'none'"
                />
                <span class="font-weight-bold text-body-2">{{ getSkillName(skillItem.id) }}</span>
              </div>
            </template>

            <template v-slot:[`item.totalDamage`]="{ item: skillItem }">
              <div class="stat-bar text-right font-weight-bold" :style="{ 'background-image': getSkillCombinedGradient(item.overallHealing, skillItem) }">
                {{ skillItem.totalDamage > 0 ? formatNumber(skillItem.totalDamage) : '—' }}
              </div>
            </template>

            <template v-slot:[`item.maxDamage`]="{ item: skillItem }">
              <div class="text-right font-weight-bold">
                {{ (skillItem.maxDamage || 0) > 0 ? formatNumber(skillItem.maxDamage) : '—' }}
              </div>
            </template>

            <template v-slot:[`item.overshieldProvided`]="{ item: skillItem }">
              <div class="stat-bar text-right font-weight-bold text-light-green-accent-3" :style="{ 'background-image': getSkillCombinedGradient(item.overallHealing, skillItem) }">
                {{ (skillItem.overshieldProvided || 0) > 0 ? formatNumber(skillItem.overshieldProvided || 0) : '—' }}
              </div>
            </template>

          </v-data-table>
        </td>
      </tr>
    </template>
  </v-data-table>
</template>

<!-- Experimental Healing Done: shows known healing and anonymous fallback.
     Condition 999 provides observed Veil capacity and DMR; effective restoration needs validation. -->
<script lang="ts">
import { defineComponent, ref, computed, inject, watch } from "vue";
import { fightSummary, activeSessionId, selectedTargetId, hiddenPlayers, globalHideMode, setAllHiddenPlayers, showClassColorsForVisiblePlayers } from "@/store";
import { DamageBreakdown, SkillStats } from "@/protocols";
import { isGroupTargetId, getGroupMemberTargetIds, getEncounterDurationForTargets } from "@/utils/targetGrouping";
import { toggleHiddenPlayer } from "@/store";
import { getMabiNameColor, getClassColor } from "@/util";

export default defineComponent({
  name: "ApplyHealingBySkill",
  setup() {
    const skillNameMap = inject("skillNameMap") as any;
    const expanded = ref<string[]>([]);

    watch(activeSessionId, () => { expanded.value = []; });

    const mainTableHeaders = computed(() => [
      { title: "Hide", key: "isHidden", sortable: false, width: "1%" },
      { title: "Character", key: "name", sortable: true, width: "20%" },
      { title: "Healing Contribution", key: "overallHealing.totalDamage", sortable: true, align: "end" as const, width: "35%" },
      { title: "Largest Heal", key: "largestHeal", sortable: true, align: "end" as const },
      { title: "Veil Gained", key: "overallHealing.overshieldProvided", sortable: true, align: "end" as const, width: "20%" },
      { title: "Veil Peak", key: "veilPeak", sortable: true, align: "end" as const },
      { title: "Veil DMR", key: "veilDmr", sortable: true, align: "end" as const },
      { title: "", key: "data-table-expand", width: "4%" },
    ]);

    const computedSkillTableHeaders = computed(() => [
      { title: "Skill", key: "skillName", sortable: true, width: "200px" },
      { title: "Total Healed", key: "totalDamage", sortable: true, align: "end" as const },
      { title: "Largest Heal", key: "maxDamage", sortable: true, align: "end" as const },
      { title: "Veil Gained", key: "overshieldProvided", sortable: true, align: "end" as const }
    ]);

    const formatNumber = (num: number | undefined | null): string => {
      if (num === undefined || num === null) return "0";
      return Math.round(num).toLocaleString("en-US");
    };

    const formatAbbreviated = (num: number | undefined | null): string => {
      if (num === undefined || num === null) return "0";
      if (num >= 1_000_000) return (num / 1_000_000).toFixed(1) + "M";
      if (num >= 1_000) return (num / 1_000).toFixed(0) + "k";
      return Math.round(num).toLocaleString("en-US");
    };
    
    const currentEncounterDuration = computed(() => {
      return fightSummary?.encounterDuration || 0;
    });

    const playerDisplayData = computed(() => {
      if (!fightSummary?.players) return [];
      const rows = Object.values(fightSummary.players)
        .map((p: any) => ({
          id: p.id,
          name: p.name,
          veilDmr: p.veilDmr, veilPeakDmr: p.veilPeakDmr, veilPeak: p.veilPeak || 0,
          largestHeal: Math.max(0, ...Object.values(p.overallHealing?.skills || {}).map((sk: any) => Number(sk?.maxDamage || 0))),
          // Older captures may use an entity ID as the fallback name.
          overallHealing: p.overallHealing || { totalDamage: 0, hitCount: 0, critCount: 0, dps: 0, overshieldProvided: 0, critRate: 0, skills: {} },
          healingByTarget: p.healingByTarget || {}
        }))
        .filter((p: any) => p.overallHealing.totalDamage > 0 || p.overallHealing.overshieldProvided > 0);

      // Mabinogi character names cannot begin with a digit, so numeric names
      // are unresolved entity IDs, not real character names. Fold them into a
      // single anonymous row rather than exposing a string of numeric IDs.
      const isAnonymous = (p: any) => p.id === "misc-anonymous-healing" || /^\d/.test(String(p.name || p.id || ""));
      const namedRows = rows.filter((p: any) => !isAnonymous(p));
      const anonymousRows = rows.filter(isAnonymous);
      if (anonymousRows.length) {
        const combined: any = {
          id: "misc-anonymous-healing",
          name: "Misc. Anonymous Healing",
          overallHealing: { totalDamage: 0, hitCount: 0, critCount: 0, dps: 0, overshieldProvided: 0, critRate: 0, skills: {} },
          healingByTarget: {}, largestHeal: 0
        };
        for (const row of anonymousRows) {
          const dst = combined.overallHealing;
          const src = row.overallHealing || {};
          dst.totalDamage += Number(src.totalDamage || 0);
          dst.hitCount += Number(src.hitCount || 0);
          dst.critCount += Number(src.critCount || 0);
          dst.overshieldProvided += Number(src.overshieldProvided || 0);
          for (const [skillKey, skillValue] of Object.entries(src.skills || {})) {
            const skill: any = skillValue || {};
            const current = dst.skills[skillKey] || { id: Number(skillKey), count: 0, critCount: 0, totalDamage: 0, totalDamageCrit: 0, totalDamageNonCrit: 0, maxDamage: 0, maxDamageCrit: 0, maxDamageNonCrit: 0, uses: 0, overshieldProvided: 0 };
            current.count += Number(skill.count || 0);
            current.critCount += Number(skill.critCount || 0);
            current.totalDamage += Number(skill.totalDamage || 0);
            current.totalDamageCrit += Number(skill.totalDamageCrit || 0);
            current.totalDamageNonCrit += Number(skill.totalDamageNonCrit || 0);
            current.maxDamage = Math.max(current.maxDamage, Number(skill.maxDamage || 0));
            current.maxDamageCrit = Math.max(current.maxDamageCrit, Number(skill.maxDamageCrit || 0));
            current.maxDamageNonCrit = Math.max(current.maxDamageNonCrit, Number(skill.maxDamageNonCrit || 0));
            current.uses += Number(skill.uses || 0);
            current.overshieldProvided += Number(skill.overshieldProvided || 0);
            dst.skills[skillKey] = current;
          }
        }
        combined.largestHeal = Math.max(0, ...Object.values(combined.overallHealing.skills).map((sk: any) => Number(sk?.maxDamage || 0)));
        const duration = Number(fightSummary?.encounterDuration || 0);
        combined.overallHealing.dps = duration > 0 ? combined.overallHealing.totalDamage / duration : 0;
        combined.overallHealing.critRate = combined.overallHealing.hitCount > 0 ? (combined.overallHealing.critCount / combined.overallHealing.hitCount) * 100 : 0;
        namedRows.push(combined);
      }
      return namedRows.sort((a: any, b: any) => b.overallHealing.totalDamage - a.overallHealing.totalDamage);
    });
       
    // Resolves row tracking colors based on custom user choices or active arcana
    const getPlayerColor = (item: any) => {
      const isHidden = globalHideMode.value || hiddenPlayers.has(item.id);
      const shouldUseClassColor = isHidden || showClassColorsForVisiblePlayers.value;
      return shouldUseClassColor 
        ? getClassColor(item.talentName, item.talentColor) 
        : getMabiNameColor(item.name);
    };

    const totalHealingForView = computed(() => {
      return playerDisplayData.value.reduce((sum: number, p: any) => sum + (p.overallHealing?.totalDamage || 0), 0);
    });

    const getSkillName = (skillId: number): string => {
      if (skillId === 0) return "Uncategorized Healing";
      if (skillId === 9999) return "Pets";
      return skillNameMap.value?.[skillId]?.name || `Unknown Skill: ${skillId}`;
    };

    const getSkillImageSrc = (skillId: number): string => {
      return skillNameMap.value?.[skillId]?.iconUrl || "";
    };

    const getSkillBreakdown = (overallHealing: any): SkillStats[] => {
      if (!overallHealing?.skills) return [];
      return (Object.values(overallHealing.skills) as SkillStats[]);
    };

    // Healing and Veil are independently normalized for visual balance, then
    // displayed as adjacent burgundy/green segments in one combined bar.
    const getSkillPercent = (healing: any, skill: any, metric: 'totalDamage' | 'overshieldProvided'): number => {
      const values = Object.values(healing?.skills || {}) as any[];
      const largest = Math.max(0, ...values.map(v => Number(v?.[metric] || 0)));
      return largest > 0 ? Math.min(100, Math.max(0, Number(skill?.[metric] || 0) / largest * 100)) : 0;
    };
    const getSkillCombinedGradient = (healing: any, skill: any): string => {
      const h = getSkillPercent(healing, skill, 'totalDamage');
      const v = getSkillPercent(healing, skill, 'overshieldProvided');
      const total = h + v;
      if (!total) return 'none';
      const extent = Math.max(h, v); // Bar length anchored to stronger category.
      const burgundy = extent * h / total;
      return `linear-gradient(90deg, rgba(124,43,64,.84) 0% ${burgundy}%, rgba(40,170,87,.82) ${burgundy}% ${extent}%, transparent ${extent}%)`;
    };
    const getRowProps = ({ item }: { item: any }) => {
      const players = playerDisplayData.value;
      const maxHeal = Math.max(0, ...players.map((p: any) => Number(p.overallHealing?.totalDamage || 0)));
      const maxVeil = Math.max(0, ...players.map((p: any) => Number(p.overallHealing?.overshieldProvided || 0)));
      const h = maxHeal > 0 ? Number(item.overallHealing?.totalDamage || 0) / maxHeal : 0;
      const v = maxVeil > 0 ? Number(item.overallHealing?.overshieldProvided || 0) / maxVeil : 0;
      const extent = Math.max(h, v) * 100, split = h + v > 0 ? extent * h / (h + v) : 0;
    return {
        class: "character-row",
        style: { background: `linear-gradient(to right, rgba(124, 43, 64, .75) 0% ${split}%, rgba(40, 170, 87, .7) ${split}% ${extent}%, transparent ${extent}%)` }
      };
    };
    
    return {
      expanded,
      mainTableHeaders,
      computedSkillTableHeaders,
      playerDisplayData,
      totalHealingForView,
      currentEncounterDuration,
      formatNumber,
      formatAbbreviated,
      getSkillName,
      getSkillImageSrc,
      getSkillBreakdown,
      getSkillPercent,
      getSkillCombinedGradient,
      getRowProps,
      getPlayerColor,
      toggleHiddenPlayer,
      hiddenPlayers,
    };
  }
});
</script>

<style scoped>
.pb-16 { 
  padding-bottom: 80px !important; 
}

:deep(.v-data-table) { 
  background: transparent !important; 
}

.dps-table :deep(table) { 
  border-spacing: 0 6px !important; 
  border-collapse: separate !important; 
}

.dps-table :deep(tbody tr.character-row) { 
  position: relative; 
  transition: all 0.2s ease; 
}

.dps-table :deep(td) { 
  border-bottom: none !important; 
  padding-top: 4px !important; 
  padding-bottom: 4px !important; 
  background-color: rgba(255, 255, 255, 0.03); 
}

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
/* Side-by-side burgundy/green skill bars; matching fill in each value column. */
.stat-bar { min-width: 90px; padding: 7px 10px; border-radius: 3px; background-repeat: no-repeat; }
</style>
