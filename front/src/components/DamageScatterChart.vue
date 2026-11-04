<template>
  <div class="scatter-container mt-6">
    <div class="scatter-header px-4 py-2 d-flex align-items-center justify-content-between flex-wrap ga-2">
      <span class="panel-title font-weight-bold">🎯 Chronological Action Timeline</span>
      <div class="d-flex align-center ga-2 flex-wrap">
        <span class="ceiling-label text-slate text-caption">1s Hit Grouping</span>
        <v-btn v-if="targetIds.length" size="x-small" variant="tonal" @click="bossOnly = !bossOnly">
          {{ bossOnly ? 'Boss / selected target only' : 'All targets (click for selected target)' }}
        </v-btn>
      </div>
    </div>
    <div class="player-toggles px-4 py-3 d-flex flex-wrap ga-3">
      <v-btn
        v-for="player in players"
        :key="player.stats.id"
        size="small"
        variant="tonal"
        :color="(!globalHideMode && !hiddenPlayers.has(player.stats.id)) ? 'primary' : 'grey'"
        @click="togglePlayer(player.stats.id)"
      >
        <v-icon start>{{ (!globalHideMode && !hiddenPlayers.has(player.stats.id)) ? 'mdi-eye' : 'mdi-eye-off' }}</v-icon>
        {{ player.stats.name }}
      </v-btn>
    </div>
    <div v-if="enemies.length" class="px-4 py-2 enemy-controls">
      <div class="text-caption mb-1">Boss-originated outgoing attacks (not party DPS)</div>
      <v-btn size="x-small" variant="tonal" :disabled="bossOnly" @click="showEnemies = !showEnemies">
        {{ showEnemies ? 'Hide enemy attacks' : 'Show enemy attacks' }}
      </v-btn>
      <span v-if="bossOnly" class="text-caption ml-3">Switch to All Targets to inspect incoming boss attacks</span>
      <span v-else-if="showEnemies" class="text-caption ml-3">Verified boss entities are labeled by race / attempt</span>
    </div>
    <div class="scatter-chart-wrapper position-relative" style="height: 320px;">
      <!-- Custom Tooltip Intercept Overlay -->
      <div 
        v-if="tooltip.visible" 
        class="custom-vue-tooltip position-absolute"
        :style="{ left: tooltip.x + 'px', top: tooltip.y + 'px', borderColor: tooltip.playerColor }"
      >
        <div class="tooltip-time mb-1 font-weight-bold">{{ tooltip.playerName }} · ⏱️ {{ formatTimeLabel(tooltip.time) }}</div>
        <div class="tooltip-damage font-weight-bold mb-2">💥 {{ tooltip.isEnemy ? 'Enemy outgoing damage' : 'Combined player damage' }}: {{ tooltip.totalDmg.toLocaleString() }}</div>
        <div class="tooltip-divider mb-2"></div>
        <div v-for="sk in tooltip.skills" :key="sk.skillId" class="tooltip-skill-row d-flex align-items-center justify-content-between mb-1">
          <span class="sk-name text-truncate pr-4">{{ getSkillName(sk.skillId) }}</span>
          <span class="sk-meta text-right font-weight-medium">
            {{ sk.damage.toLocaleString() }} <span class="text-slate">({{ sk.hitCount }}x)</span>
            <span v-if="sk.critCount > 0" class="text-red ml-1">🔥 Crit</span>
          </span>
        </div>
      </div>

      <div v-if="bossOnly && unavailableLegacy" class="text-warning text-caption px-4 py-2">
        Older skill groups without recipient IDs cannot be attributed to this target; excluded from target-only scatter.
      </div>
      <canvas v-if="visibleDatasets.length > 0" ref="canvasRef"></canvas>
      <div v-else class="empty-chart pa-8 text-center text-medium-emphasis">
        {{ players.length === 0 ? 'No individual skill-hit events are available for this session.' : 'No scatter points are available for the current target selection, or all players are hidden.' }}
      </div>
    </div>
  </div>
</template>

<!-- Rotation scatter: per-hit points and critical indicators. Player toggles
     are shared with Damage Graph; pan/zoom affects only this chart viewport. -->
<script lang="ts">
import { defineComponent, onMounted, ref, computed, inject, watch, onBeforeUnmount, nextTick, PropType } from "vue";
import { hiddenPlayers, toggleHiddenPlayer, globalHideMode } from "@/store";
import { Chart as ChartJS, ScatterController, PointElement, LinearScale, Tooltip, Legend } from "chart.js";
import zoomPlugin from "chartjs-plugin-zoom";
import { scopeScatterGroups, hasTargetAttribution } from "@/utils/scatterTargets";

ChartJS.register(ScatterController, PointElement, LinearScale, Tooltip, Legend, zoomPlugin);

export default defineComponent({
  name: "DamageScatterChart",
  props: {
    players: { type: Array as PropType<any[]>, required: true },
    enemies: { type: Array as PropType<any[]>, default: () => [] },
    targetIds: { type: Array as PropType<string[]>, default: () => [] }
  },
  setup(props) {
    const canvasRef = ref<HTMLCanvasElement | null>(null);
    const bossOnly = ref(false);
    const showEnemies = ref(false);
    const unavailableLegacy = computed(() => bossOnly.value && (props.players as any[]).some(p =>
      (p.stats.skillUses || []).some((g:any) => !hasTargetAttribution(g))));
    watch(() => props.targetIds.join(','), (value) => { bossOnly.value = Boolean(value); void nextTick(syncChart); }, {immediate:true});
    const skillNameMap = inject("skillNameMap") as any;
    let chartInstance: ChartJS | null = null;
    const visiblePlayerIds = computed(() => new Set<string>((props.players as any[]).filter((p: any) => !globalHideMode.value && !hiddenPlayers.has(p.stats.id)).map((p: any) => p.stats.id)));

    const tooltip = ref({ visible: false, x: 0, y: 0, time: 0, totalDmg: 0, isEnemy: false, playerName: "", playerColor: "#38bdf8", skills: [] as any[] });

    const formatTimeLabel = (seconds: number) => {
      const mins = Math.floor(seconds / 60);
      const secs = Math.round(seconds % 60);
      return `${mins}:${secs.toString().padStart(2, "0")}`;
    };

    const getSkillName = (id: number) => skillNameMap.value?.[id]?.name || `Skill ${id}`;

    const togglePlayer = (playerId: string) => {
      toggleHiddenPlayer(playerId);
      void nextTick(() => syncChart());
    };

    const visibleDatasets = computed(() => {
      // Boss tracks are a separate optional overlay; selecting a boss as a
      // recipient must never show its outgoing attacks as boss damage dealt.
      const playerRows = (props.players as any[])
        .filter((player: any) => visiblePlayerIds.value.has(player.stats.id));
      const enemyRows = showEnemies.value && !bossOnly.value ? (props.enemies as any[]) : [];
      return [...playerRows, ...enemyRows]
        .map((player: any) => ({
          label: player.stats.name,
          playerId: player.stats.id,
          backgroundColor: player.color,
          borderColor: player.color,
          pointRadius: 5,
          pointHoverRadius: 8,
          data: scopeScatterGroups(player.stats.skillUses || [], bossOnly.value && props.targetIds.length ? new Set(props.targetIds) : null).map((evt: any) => ({
            x: evt.timestamp,
            y: evt.totalDmg,
            rawSkills: evt.skills,
            playerName: player.stats.name,
            playerColor: player.color,
            isEnemy: player.kind === 'enemy',
          })),
        }));
    });

    const syncChart = () => {
      if (!canvasRef.value || visibleDatasets.value.length === 0) {
        chartInstance?.destroy();
        chartInstance = null;
        return;
      }
      if (!chartInstance) {
        initChart();
        return;
      }
      chartInstance.data.datasets = visibleDatasets.value as any;
      chartInstance.update("none");
    };

    const initChart = () => {
      if (!canvasRef.value) return;
      
      chartInstance = new ChartJS(canvasRef.value, {
        type: "scatter",
        data: { datasets: visibleDatasets.value as any },
        options: {
          responsive: true,
          maintainAspectRatio: false,
          plugins: {
            legend: { display: true, labels: { color: "#cbd5e1" } },
            zoom: {
              // Drag horizontally to inspect points anywhere along long fights.
              pan: { enabled: true, mode: "x", threshold: 3 },
              zoom: {
                mode: "x",
                wheel: { enabled: true, modifierKey: "ctrl" },
                pinch: { enabled: true },
                drag: { enabled: false },
              },
            },
            tooltip: {
              enabled: false, // Turn off native toolkit layer to let our Custom Vue frame take over
              external: (context) => {
                const model = context.tooltip;
                if (model.opacity === 0) {
                  tooltip.value.visible = false;
                  return;
                }
                const pt = model.dataPoints[0].raw as any;
                const wrapper = canvasRef.value?.parentElement;
                const maxX = Math.max(8, (wrapper?.clientWidth || 600) - 290);
                tooltip.value.x = Math.min(model.caretX + 15, maxX);
                const height = wrapper?.querySelector(".custom-vue-tooltip")?.getBoundingClientRect().height || 180;
                const available = wrapper?.clientHeight || 320;
                tooltip.value.y = Math.max(8, Math.min(model.caretY - 30, available - height - 8));
                tooltip.value.time = pt.x;
                tooltip.value.totalDmg = pt.y;
                tooltip.value.playerName = pt.playerName || "Unknown player";
                tooltip.value.playerColor = pt.playerColor || "#38bdf8";
                tooltip.value.isEnemy = pt.isEnemy === true;
                tooltip.value.skills = pt.rawSkills || [];
                tooltip.value.visible = true;
              }
            }
          },
          scales: {
            x: {
              type: "linear",
              title: { display: true, text: "Timeline (M:S)", color: "#ffffff" },
              ticks: { color: "#cbd5e1", callback: (val) => formatTimeLabel(Number(val)) },
              grid: { color: "rgba(255,255,255,0.05)" }
            },
            y: {
              type: "linear",
              title: { display: true, text: "Hit Value", color: "#ffffff" },
              ticks: { color: "#cbd5e1" },
              grid: { color: "rgba(255,255,255,0.05)" }
            }
          }
        }
      });
    };

    watch(visibleDatasets, () => { void nextTick(() => syncChart()); }, { deep: true });
    watch(() => props.players, () => { void nextTick(() => syncChart()); }, { deep: true });
    watch(() => [globalHideMode.value, [...hiddenPlayers].join(",")], () => { void nextTick(() => syncChart()); });

    onMounted(() => { if (visibleDatasets.value.length > 0) initChart(); });
    watch([visibleDatasets, bossOnly], () => void nextTick(syncChart));
    onBeforeUnmount(() => chartInstance?.destroy());

    return { canvasRef, tooltip, formatTimeLabel, getSkillName, players: computed(() => props.players), enemies: computed(() => props.enemies), visiblePlayerIds, togglePlayer, visibleDatasets, hiddenPlayers, globalHideMode, bossOnly, showEnemies, unavailableLegacy };
  }
});
</script>

<style scoped>
.scatter-container {
  background: rgba(23, 27, 36, 0.4);
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 8px;
  overflow: hidden;
}
.enemy-controls { border-bottom: 1px solid rgba(222, 145, 106, 0.35); color: #f2c6a6; }
.player-toggles {
  border-bottom: 1px solid rgba(255, 255, 255, 0.05);
}
.empty-chart {
  color: #94a3b8;
  min-height: 180px;
}
.scatter-header {
  background: rgba(255, 255, 255, 0.02);
  border-bottom: 1px solid rgba(255, 255, 255, 0.05);
}
.custom-vue-tooltip {
  background: rgba(15, 23, 42, 0.98);
  border: 1px solid transparent;
  border-radius: 6px;
  padding: 10px;
  box-shadow: 0 12px 24px rgba(0,0,0,0.5);
  z-index: 999;
  pointer-events: none;
  font-size: 0.85rem;
  color: #cbd5e1;
  min-width: 250px;
  max-height: calc(100% - 16px);
  overflow-y: auto;
}
.tooltip-divider { height: 1px; background: rgba(255,255,255,0.1); }
.text-red { color: #f87171; }
</style>
