<template>
  <div class="dps-container">
    <div v-if="!sortedPlayers || sortedPlayers.length === 0" class="empty-state">
      <span>Waiting for combat data...</span>
    </div>

    <div v-else class="player-list">
      <div 
        v-for="(player, idx) in sortedPlayers" 
        :key="player.id" 
        class="player-row clickable"
        @click="$emit('select-player', player.id)"
      >
        <!-- Background Damage Bar -->
        <div 
          class="damage-bar" 
          :style="{ 
            width: getBarWidth(player.stats.totalDamage) + '%',
            backgroundColor: player.talentColor || defaultColors[idx % defaultColors.length]
          }"
        ></div>

        <!-- Row Content -->
        <div class="row-content">
          <!-- Rank, Arcana Icon & Name -->
          <div class="player-info">
            <span class="rank">{{ idx + 1 }}</span>
            <img 
              v-if="player.talentIcon" 
              :src="getTalentIconUrl(player.talentIcon)" 
              class="talent-icon" 
              :alt="player.talentName || 'Talent'"
              @error="($event.target as HTMLImageElement).style.display = 'none'"
            />
            <span v-else class="talent-indicator" :style="{ backgroundColor: player.talentColor || defaultColors[idx % defaultColors.length] }"></span>
            <span class="name" :title="player.displayName">{{ player.displayName }}</span>
          </div>

          <!-- Metrics: DPS, Total Damage, %, Crit Rate -->
          <div class="metrics">
            <span class="dps">{{ formatNumber(player.stats.dps) }}/s</span>
            <span class="total">{{ formatNumber(player.stats.totalDamage) }}</span>
            <span class="percent">({{ getDamagePercent(player.stats.totalDamage) }}%)</span>
            <span class="crit" title="Crit Rate">{{ (player.stats.critRate || 0).toFixed(1) }}%</span>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { PlayerStats, DamageBreakdown, TargetStats } from "../types";

const props = defineProps<{
  players: { [id: string]: PlayerStats };
  targetId: string;
  totalDamage: number;
  targets?: { [targetId: string]: TargetStats };
  hideNames?: boolean;
  serverUrl?: string;
}>();

defineEmits<{
  (e: "select-player", playerId: string): void;
}>();

const defaultColors = [
  "#3b82f6", "#10b981", "#f59e0b", "#ef4444", "#8b5cf6", 
  "#ec4899", "#06b6d4", "#84cc16", "#eab308", "#6366f1"
];

interface FormattedPlayer {
  id: string;
  name: string;
  displayName: string;
  talentName?: string;
  talentIcon?: string;
  talentColor?: string;
  stats: DamageBreakdown;
}

const getTalentIconUrl = (iconPath?: string) => {
  if (!iconPath) return "";
  if (iconPath.startsWith("http://") || iconPath.startsWith("https://")) {
    return iconPath;
  }
  const base = (props.serverUrl || "http://localhost:8030").replace(/\/$/, "");
  const path = iconPath.startsWith("/") ? iconPath : `/${iconPath}`;
  return `${base}${path}`;
};

const sortedPlayers = computed<FormattedPlayer[]>(() => {
  if (!props.players) return [];

  const list: FormattedPlayer[] = [];
  const entries = Object.entries(props.players);

  entries.forEach(([id, p], idx) => {
    let stats: DamageBreakdown = p.overallStats || { totalDamage: 0, dps: 0, critRate: 0, hitCount: 0, critCount: 0 };

    if (props.targetId && p.damageByTarget && p.damageByTarget[props.targetId]) {
      stats = p.damageByTarget[props.targetId];
    }

    if (stats.totalDamage > 0 || stats.dps > 0) {
      const rawName = p.name || `Player ${id}`;
      const displayName = props.hideNames 
        ? (p.talentName || `Player ${idx + 1}`) 
        : rawName;

      list.push({
        id: p.id,
        name: rawName,
        displayName,
        talentName: p.talentName,
        talentIcon: p.talentIcon,
        talentColor: p.talentColor,
        stats,
      });
    }
  });

  return list.sort((a, b) => b.stats.totalDamage - a.stats.totalDamage);
});

const topDamage = computed(() => {
  if (sortedPlayers.value.length === 0) return 1;
  return sortedPlayers.value[0].stats.totalDamage || 1;
});

const getBarWidth = (damage: number) => {
  if (topDamage.value <= 0) return 0;
  return Math.min(100, Math.max(2, (damage / topDamage.value) * 100));
};

const totalTargetDamage = computed(() => {
  if (props.targetId) {
    if (props.targets && props.targets[props.targetId] && props.targets[props.targetId].totalDamage) {
      return props.targets[props.targetId].totalDamage || 0;
    }
    return sortedPlayers.value.reduce((sum, p) => sum + p.stats.totalDamage, 0);
  }
  return props.totalDamage;
});

const getDamagePercent = (damage: number) => {
  const total = totalTargetDamage.value || topDamage.value || 1;
  if (total <= 0) return "0.0";
  return ((damage / total) * 100).toFixed(1);
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
</script>

<style scoped>
.dps-container {
  flex: 1;
  overflow-y: auto;
  padding: 6px;
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

.player-list {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.player-row {
  position: relative;
  height: 28px;
  background: #171b24;
  border: 1px solid var(--border-color);
  border-radius: 4px;
  overflow: hidden;
  display: flex;
  align-items: center;
}

.player-row.clickable {
  cursor: pointer;
  transition: all 0.15s ease;
}

.player-row.clickable:hover {
  border-color: var(--accent-primary);
  background: rgba(23, 27, 36, 0.95);
}

.damage-bar {
  position: absolute;
  left: 0;
  top: 0;
  bottom: 0;
  opacity: 0.7;
  transition: width 0.3s ease;
}

.row-content {
  position: relative;
  z-index: 1;
  width: 100%;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 8px;
}

.player-info {
  display: flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
}

.rank {
  font-size: 0.75em;
  font-weight: 700;
  color: var(--text-dim);
  width: 14px;
  text-align: center;
}

.talent-icon {
  width: 18px;
  height: 18px;
  border-radius: 3px;
  object-fit: contain;
  flex-shrink: 0;
}

.talent-indicator {
  width: 6px;
  height: 14px;
  border-radius: 2px;
  flex-shrink: 0;
}

.name {
  font-weight: 600;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 120px;
  color: #ffffff;
  text-shadow: 0 1px 2px rgba(0, 0, 0, 0.8);
}

.metrics {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 0.88em;
  font-weight: 600;
  text-shadow: 0 1px 2px rgba(0, 0, 0, 0.8);
}

.dps {
  color: var(--accent-green);
}

.total {
  color: #ffffff;
}

.percent {
  color: var(--text-muted);
  font-size: 0.82em;
}

.crit {
  color: var(--accent-amber);
  font-size: 0.82em;
  width: 42px;
  text-align: right;
}
</style>
