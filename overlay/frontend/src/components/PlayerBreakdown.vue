<template>
  <div class="breakdown-container">
    <!-- Selected Player Header Banner -->
    <div class="player-header-banner">
      <div class="banner-left-group">
        <button 
          class="back-btn no-drag" 
          @click="$emit('back')" 
          title="Back to Player List"
        >
          <svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
            <line x1="19" y1="12" x2="5" y2="12"></line>
            <polyline points="12 19 5 12 12 5"></polyline>
          </svg>
        </button>

        <div class="player-title-section">
          <img 
            v-if="player.talentIcon" 
            :src="getTalentIconUrl(player.talentIcon)" 
            class="banner-talent-icon" 
            :alt="player.talentName || 'Talent'"
            @error="($event.target as HTMLImageElement).style.display = 'none'"
          />
          <div class="title-meta">
            <span class="player-name">{{ displayName }}</span>
            <span class="talent-sub" v-if="player.talentName">{{ player.talentName }}</span>
          </div>
        </div>
      </div>

      <div class="player-stats-summary">
        <div class="stat-badge">
          <span class="lbl">DPS</span>
          <span class="val dps">{{ formatNumber(activeStats.dps) }}/s</span>
        </div>
        <div class="stat-badge">
          <span class="lbl">TOTAL</span>
          <span class="val">{{ formatNumber(activeStats.totalDamage) }}</span>
        </div>
        <div class="stat-badge">
          <span class="lbl">CRIT</span>
          <span class="val amber">{{ (activeStats.critRate || 0).toFixed(1) }}%</span>
        </div>
      </div>
    </div>

    <!-- Skill Breakdown List -->
    <div v-if="!sortedSkills || sortedSkills.length === 0" class="empty-state">
      <span>No skill damage recorded for this target.</span>
    </div>

    <div v-else class="skill-list">
      <div 
        v-for="skill in sortedSkills" 
        :key="skill.id" 
        class="skill-row"
      >
        <!-- Background Damage Fill Bar -->
        <div 
          class="skill-damage-bar" 
          :style="{ 
            width: getBarWidth(skill.totalDamage) + '%',
            backgroundColor: player.talentColor || '#818cf8'
          }"
        ></div>

        <!-- Skill Content -->
        <div class="skill-row-content">
          <!-- Skill Info (Icon + Name + Uses) -->
          <div class="skill-info">
            <img 
              v-if="getSkillIconSrc(skill.id)" 
              :src="getSkillIconSrc(skill.id)" 
              class="skill-icon" 
              :alt="getSkillName(skill.id)"
              @error="($event.target as HTMLImageElement).style.display = 'none'"
            />
            <div class="skill-name-col">
              <span class="skill-name" :title="getSkillName(skill.id)">{{ getSkillName(skill.id) }}</span>
              <span class="skill-count">{{ skill.count }} {{ skill.count === 1 ? 'hit' : 'hits' }} ({{ skill.uses || skill.count }} uses)</span>
            </div>
          </div>

          <!-- Skill Metrics -->
          <div class="skill-metrics">
            <span class="skill-dps">{{ formatNumber(getSkillDps(skill.totalDamage)) }}/s</span>
            <span class="skill-total">{{ formatNumber(skill.totalDamage) }}</span>
            <span class="skill-pct">({{ getDamagePercent(skill.totalDamage) }}%)</span>
            <span class="skill-crit" title="Crit Rate">{{ getSkillCritRate(skill) }}%</span>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, onMounted } from "vue";
import { PlayerStats, DamageBreakdown, SkillStats } from "../types";

const props = defineProps<{
  player: PlayerStats;
  targetId: string;
  hideNames?: boolean;
  serverUrl?: string;
  encounterDuration?: number;
}>();

defineEmits<{
  (e: "back"): void;
}>();

interface SkillMeta {
  name: string;
  iconUrl?: string;
}

const skillNameMap = ref<Record<string, SkillMeta>>({});

const displayName = computed(() => {
  if (props.hideNames) {
    return props.player.talentName || "Hidden Player";
  }
  return props.player.name || `Player ${props.player.id}`;
});

const activeStats = computed<DamageBreakdown>(() => {
  if (props.targetId && props.player.damageByTarget && props.player.damageByTarget[props.targetId]) {
    return props.player.damageByTarget[props.targetId];
  }
  return props.player.overallStats || { totalDamage: 0, dps: 0, hitCount: 0, critCount: 0, critRate: 0 };
});

interface FormattedSkill extends SkillStats {
  uses?: number;
}

const sortedSkills = computed<FormattedSkill[]>(() => {
  const stats = activeStats.value;
  if (!stats || !stats.skills) return [];

  const list: FormattedSkill[] = [];
  for (const idStr in stats.skills) {
    const s = stats.skills[idStr];
    if (s.totalDamage > 0 || s.count > 0) {
      list.push({ ...s });
    }
  }

  return list.sort((a, b) => b.totalDamage - a.totalDamage);
});

const topSkillDamage = computed(() => {
  if (sortedSkills.value.length === 0) return 1;
  return sortedSkills.value[0].totalDamage || 1;
});

const getBarWidth = (damage: number) => {
  if (topSkillDamage.value <= 0) return 0;
  return Math.min(100, Math.max(2, (damage / topSkillDamage.value) * 100));
};

const getDamagePercent = (damage: number) => {
  const total = activeStats.value.totalDamage || topSkillDamage.value || 1;
  if (total <= 0) return "0.0";
  return ((damage / total) * 100).toFixed(1);
};

const getSkillDps = (damage: number) => {
  const dur = props.encounterDuration || 0;
  if (dur > 0) {
    return damage / dur;
  }
  return 0;
};

const getSkillCritRate = (skill: SkillStats) => {
  if (!skill.count || skill.count === 0) return "0.0";
  return ((skill.critCount / skill.count) * 100).toFixed(1);
};

const getTalentIconUrl = (iconPath?: string) => {
  if (!iconPath) return "";
  if (iconPath.startsWith("http://") || iconPath.startsWith("https://")) {
    return iconPath;
  }
  const base = (props.serverUrl || "http://localhost:8030").replace(/\/$/, "");
  const path = iconPath.startsWith("/") ? iconPath : `/${iconPath}`;
  return `${base}${path}`;
};

const getSkillIconSrc = (skillId: number | string) => {
  const idStr = String(skillId);
  if (idStr === "9999") return "";
  const meta = skillNameMap.value[idStr];
  if (meta && meta.iconUrl) {
    if (meta.iconUrl.startsWith("http://") || meta.iconUrl.startsWith("https://")) {
      return meta.iconUrl;
    }
    const base = (props.serverUrl || "http://localhost:8030").replace(/\/$/, "");
    const path = meta.iconUrl.startsWith("/") ? meta.iconUrl : `/${meta.iconUrl}`;
    return `${base}${path}`;
  }
  return "";
};

const getSkillName = (skillId: number | string) => {
  const idStr = String(skillId);
  if (idStr === "9999") return "Pet / Summon Damage";
  const meta = skillNameMap.value[idStr];
  if (meta && meta.name && !meta.name.startsWith("_LT[")) {
    return meta.name;
  }
  return `Skill #${idStr}`;
};

const formatNumber = (num: number): string => {
  if (!num || isNaN(num)) return "0";
  if (num >= 1000000) {
    return (num / 1000000).toFixed(2) + "M";
  }
  if (num >= 1000) {
    return (num / 1000).toFixed(1) + "k";
  }
  return num.toFixed(0);
};

// Async fetch server skills data over HTTP
const fetchServerSkillData = async () => {
  try {
    const base = (props.serverUrl || "http://localhost:8030").replace(/\/$/, "");
    const [skillsRes, overridesRes] = await Promise.all([
      fetch(`${base}/api/data/skills.json`).catch(() => null),
      fetch(`${base}/api/data/overrides.json`).catch(() => null),
    ]);

    if (skillsRes && skillsRes.ok) {
      const skillsData = await skillsRes.json();
      for (const idStr in skillsData) {
        const item = skillsData[idStr];
        let iconUrl = item.iconUrl || "";
        if (iconUrl && !iconUrl.startsWith("/") && !iconUrl.startsWith("http")) {
          iconUrl = `/images/skills/${iconUrl}`;
        }
        const rawName = item.name ? item.name.trim() : "";
        skillNameMap.value[idStr] = {
          name: rawName && !rawName.startsWith("_LT[") ? rawName : `Skill #${idStr}`,
          iconUrl,
        };
      }
    }

    if (overridesRes && overridesRes.ok) {
      const overridesData = await overridesRes.json();
      if (overridesData && overridesData.skills) {
        for (const idStr in overridesData.skills) {
          const ov = overridesData.skills[idStr];
          const existing = skillNameMap.value[idStr] || { name: `Skill #${idStr}` };
          if (ov.name) existing.name = ov.name;
          if (ov.iconUrl) {
            existing.iconUrl = ov.iconUrl.startsWith("/") ? ov.iconUrl : `/images/${ov.iconUrl}`;
          }
          skillNameMap.value[idStr] = existing;
        }
      }
    }
  } catch (e) {
    console.error("Failed to load server skills data", e);
  }
};

onMounted(() => {
  fetchServerSkillData();
});
</script>

<style scoped>
.breakdown-container {
  flex: 1;
  overflow-y: auto;
  padding: 6px;
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.player-header-banner {
  background: rgba(23, 27, 36, 0.9);
  border: 1px solid var(--border-color);
  border-radius: 6px;
  padding: 6px 10px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.banner-left-group {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}

.back-btn {
  background: rgba(129, 140, 248, 0.12);
  border: 1px solid var(--border-color);
  color: var(--accent-primary);
  width: 22px;
  height: 22px;
  padding: 0;
  border-radius: 4px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.15s ease;
  flex-shrink: 0;
}

.back-btn:hover {
  color: #ffffff;
  background: var(--accent-primary);
  border-color: var(--accent-primary);
}

.player-title-section {
  display: flex;
  align-items: center;
  gap: 8px;
}

.banner-talent-icon {
  width: 24px;
  height: 24px;
  border-radius: 4px;
  object-fit: contain;
}

.title-meta {
  display: flex;
  flex-direction: column;
}

.player-name {
  font-weight: 700;
  font-size: 0.92em;
  color: #ffffff;
}

.talent-sub {
  font-size: 0.72em;
  color: var(--accent-primary);
  font-weight: 600;
}

.player-stats-summary {
  display: flex;
  align-items: center;
  gap: 10px;
}

.stat-badge {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
}

.stat-badge .lbl {
  font-size: 0.65em;
  font-weight: 700;
  color: var(--text-dim);
}

.stat-badge .val {
  font-size: 0.85em;
  font-weight: 700;
  color: #ffffff;
}

.stat-badge .val.dps {
  color: var(--accent-green);
}

.stat-badge .val.amber {
  color: var(--accent-amber);
}

.empty-state {
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--text-dim);
  font-style: italic;
  font-size: 0.9em;
}

.skill-list {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.skill-row {
  position: relative;
  height: 32px;
  background: #171b24;
  border: 1px solid var(--border-color);
  border-radius: 4px;
  overflow: hidden;
  display: flex;
  align-items: center;
}

.skill-damage-bar {
  position: absolute;
  left: 0;
  top: 0;
  bottom: 0;
  opacity: 0.6;
  transition: width 0.3s ease;
}

.skill-row-content {
  position: relative;
  z-index: 1;
  width: 100%;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 8px;
}

.skill-info {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}

.skill-icon {
  width: 20px;
  height: 20px;
  border-radius: 3px;
  object-fit: contain;
  flex-shrink: 0;
}

.skill-name-col {
  display: flex;
  flex-direction: column;
  min-width: 0;
}

.skill-name {
  font-weight: 600;
  font-size: 0.84em;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 130px;
  color: #ffffff;
  text-shadow: 0 1px 2px rgba(0, 0, 0, 0.8);
}

.skill-count {
  font-size: 0.68em;
  color: var(--text-muted);
}

.skill-metrics {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 0.86em;
  font-weight: 600;
  text-shadow: 0 1px 2px rgba(0, 0, 0, 0.8);
}

.skill-dps {
  color: var(--accent-green);
}

.skill-total {
  color: #ffffff;
}

.skill-pct {
  color: var(--text-muted);
  font-size: 0.82em;
}

.skill-crit {
  color: var(--accent-amber);
  font-size: 0.82em;
  width: 42px;
  text-align: right;
}
</style>
