<template>
  <div v-if="chartData.datasets.length > 0" class="graph-container">
    <div class="pa-4">
      <div class="d-flex ga-4 align-center mb-4 flex-wrap">
        <div v-if="conditionItems.length > 0" style="max-width: 320px; flex-grow: 1;">
          <v-select
            v-model="selectedConditionId"
            :items="conditionItems"
            label="Highlight Condition on Timeline"
            density="compact"
            hide-details
            clearable
            variant="outlined"
            color="primary"
          >
            <template v-slot:item="{ props, item }">
              <v-list-item v-bind="props" :title="item.raw.title">
                <template v-slot:prepend>
                  <v-avatar size="24" class="mr-2" rounded="sm" v-if="item.raw.icon">
                    <v-img :src="item.raw.icon" />
                  </v-avatar>
                </template>
              </v-list-item>
            </template>
            <template v-slot:selection="{ item }">
              <div class="d-flex align-center">
                <v-avatar size="20" class="mr-2" rounded="sm" v-if="item.raw.icon">
                  <v-img :src="item.raw.icon" />
                </v-avatar>
                <span>{{ item.raw.title }}</span>
              </div>
            </template>
          </v-select>
        </div>
        <v-checkbox
          v-model="showDeaths"
          label="Highlight Player Deaths"
          density="compact"
          hide-details
          color="error"
        />
      </div>
      <div style="height: 500px" class="chart-wrapper">
        <LineChart :data="chartData" :options="chartOptions" />
      </div>
      <div v-if="selectedTargetConditions" class="mt-4">
          <ConditionTimeline 
            :conditions="selectedTargetConditions" 
            :start-time="encounterStartTime" 
            :x-axis-config="xAxisConfig"
          />
      </div>
    </div>
  </div>
  <v-sheet v-else class="d-flex align-center justify-center rounded-lg" height="400" style="background: #0f172a; border: 1px solid rgba(255, 255, 255, 0.08)">
    <div class="text-h6 text-grey-darken-1">
      Graph data is only available for saved sessions.
    </div>
  </v-sheet>
</template>

<script lang="ts">
import { defineComponent, computed, inject, ref, watch } from "vue";
import { Line as LineChart } from "vue-chartjs";
import ConditionTimeline from "./ConditionTimeline.vue";
import {
  Chart as ChartJS,
  Title,
  Tooltip,
  Legend,
  LineElement,
  LinearScale,
  PointElement,
  CategoryScale,
  Scale,
  CoreScaleOptions,
  TooltipItem,
  Filler,
} from "chart.js";
// Import selectedTargetId from the store
import { fightSummary, selectedTargetId, hiddenPlayers, showClassColorsForVisiblePlayers, globalHideMode } from "@/store";
import { getMabiNameColor } from "@/util";
import zoomPlugin from "chartjs-plugin-zoom";
import annotationPlugin from "chartjs-plugin-annotation";

ChartJS.register(
  Title,
  Tooltip,
  Legend,
  LineElement,
  LinearScale,
  PointElement,
  CategoryScale,
  Filler,
  zoomPlugin,
  annotationPlugin
);

export default defineComponent({
  name: "DamageGraph",
  components: { LineChart, ConditionTimeline },
  setup() {
    const condNameMap = inject("condNameMap") as any;
    const selectedConditionId = ref<number | null>(null);
    const conditionImage = ref<HTMLImageElement | null>(null);

    // Watcher to pre-load the condition icon image reactively
    watch(selectedConditionId, (newId) => {
      if (!newId) {
        conditionImage.value = null;
        return;
      }
      const iconUrl = condNameMap.value[newId]?.iconUrl;
      if (!iconUrl) {
        conditionImage.value = null;
        return;
      }
      const img = new Image();
      img.src = iconUrl;
      img.onload = () => {
        conditionImage.value = img;
      };
    });

    const formatTimeLabel = (seconds: number) => {
      if (isNaN(seconds)) return "0:00";
      const minutes = Math.floor(seconds / 60);
      const remainingSeconds = Math.round(seconds % 60);
      return `${minutes}:${remainingSeconds.toString().padStart(2, "0")}`;
    };

    const encounterStartTime = computed(() => {
      if (!selectedTargetId.value) {
        return fightSummary.startTime || 0;
      }
      for (const player of Object.values(fightSummary.players)) {
        const breakdown = player.damageByTarget[selectedTargetId.value];
        if (breakdown && breakdown.startTime) {
          return breakdown.startTime;
        }
      }
      return fightSummary.startTime || 0;
    });

    const conditionItems = computed(() => {
      if (!selectedTargetId.value) return [];
      const targetStats = fightSummary.targets[selectedTargetId.value];
      if (!targetStats || !targetStats.conditions) return [];

      return Object.keys(targetStats.conditions)
        .map((idStr) => {
          const id = parseInt(idStr, 10);
          const name = condNameMap.value[id]?.name || `Unknown Condition ${id}`;
          const icon = condNameMap.value[id]?.iconUrl || "";
          return {
            title: name,
            value: id,
            icon,
          };
        })
        .sort((a, b) => a.title.localeCompare(b.title));
    });

    const activeIntervals = computed(() => {
      if (!selectedTargetId.value || !selectedConditionId.value) return [];
      const targetStats = fightSummary.targets[selectedTargetId.value];
      if (!targetStats || !targetStats.conditions) return [];
      const cond = targetStats.conditions[selectedConditionId.value];
      if (!cond || !cond.intervals) return [];

      return cond.intervals.map((iv, index) => {
        let start = iv.start - encounterStartTime.value;
        let end = iv.end - encounterStartTime.value;
        if (start < 0) start = 0;
        if (end < start) end = start;
        if (end - start < 0.5) {
          end = start + 0.5; // guarantee minimum thickness
        }
        return {
          id: `ann-${index}`,
          start,
          end,
        };
      });
    });

    const showDeaths = ref(false);

    const playerDeaths = computed(() => {
      const deathsList: { time: number; playerName: string }[] = [];
      for (const playerId in fightSummary.players) {
        const player = fightSummary.players[playerId];
        if (player.deaths && player.deaths.length > 0) {
          player.deaths.forEach((deathTime) => {
            // Convert to relative time
            const relativeTime = deathTime - encounterStartTime.value;
            if (relativeTime >= 0) {
              deathsList.push({
                time: relativeTime,
                playerName: player.name,
              });
            }
          });
        }
      }
      return deathsList;
    });

    // START: NEW LOGIC FOR DYNAMIC X-AXIS
    const fightDuration = computed(() => {
      const labels = chartData.value.labels;
      if (labels && labels.length > 0) {
        // Assuming labels are timestamps in seconds and are sorted
        return labels[labels.length - 1];
      }
      return 0;
    });

    const xAxisConfig = computed(() => {
      const duration = fightDuration.value;
      if (duration <= 0) {
        // Provide a sensible default if there's no data
        return { stepSize: 60, max: 300 };
      }

      // We want to aim for a certain number of ticks for readability
      const targetTicks = 12;

      // Define our 'nice' intervals in seconds
      // e.g., 5s, 15s, 30s, 1m, 2m, 5m, 10m, 15m, 30m
      const niceIncrements = [5, 15, 30, 60, 120, 300, 600, 900, 1800];

      // Calculate a rough step size based on the duration and target ticks
      const roughStep = duration / targetTicks;

      // Find the 'nice' increment that is closest to our rough calculation
      const stepSize = niceIncrements.reduce((prev, curr) => {
        return Math.abs(curr - roughStep) < Math.abs(prev - roughStep)
          ? curr
          : prev;
      });

      // To make the graph look complete, we'll round the max value of the axis
      // up to the next full increment.
      const max = Math.ceil(duration / stepSize) * stepSize;

      return { stepSize, max };
    });
    // END: NEW LOGIC FOR DYNAMIC X-AXIS

    const chartOptions = computed(() => {
      const annotations: Record<string, any> = {};

      if (selectedConditionId.value) {
        activeIntervals.value.forEach((iv) => {
          annotations[iv.id] = {
            type: "box" as const,
            xMin: iv.start,
            xMax: iv.end,
            backgroundColor: "rgba(6, 182, 212, 0.06)", // subtle vibrant transparent cyan overlay
            borderColor: "rgba(6, 182, 212, 0.35)",     // subtle cyan dashed border
            borderWidth: 1,
            borderDash: [4, 4],
            label: conditionImage.value ? {
              display: true,
              content: conditionImage.value,
              width: 20,
              height: 20,
              position: {
                x: "center" as const,
                y: "start" as const,
              },
              yAdjust: 12,
            } : undefined
          };
        });
      }

      if (showDeaths.value) {
        playerDeaths.value.forEach((death, index) => {
          annotations[`death-${index}`] = {
            type: "line" as const,
            scaleID: "x",
            value: death.time,
            borderColor: "rgba(239, 68, 68, 0.6)",
            borderWidth: 1.5,
            borderDash: [6, 4],
            label: {
              display: true,
              content: `☠️ ${death.playerName}`,
              position: "start" as const,
              yAdjust: 35,
              backgroundColor: "rgba(15, 23, 42, 0.85)",
              color: "#fca5a5",
              font: { family: "Outfit, Inter, sans-serif", size: 10, weight: "bold" as const },
              padding: 4,
            }
          };
        });
      }

      return {
        responsive: true,
        maintainAspectRatio: false,
        interaction: { mode: "index" as const, intersect: false },
        scales: {
          x: {
            type: "linear" as const,
            title: { 
              display: true, 
              text: "Time (minutes:seconds)", 
              color: "#ffffff",
              font: { family: "Outfit, Inter, sans-serif", size: 13, weight: "bold" as const }
            },
            // Set the max value of the axis to our calculated nice, round number
            max: xAxisConfig.value.max,
            ticks: {
              // Set the explicit step size for our ticks
              stepSize: xAxisConfig.value.stepSize,
              color: "#cbd5e1", // bright slate gray
              font: { family: "Outfit, Inter, sans-serif", size: 11 },
              callback: function (
                this: Scale<CoreScaleOptions>,
                tickValue: string | number
              ) {
                return formatTimeLabel(Number(tickValue));
              },
            },
            grid: {
              color: "rgba(255, 255, 255, 0.08)", // high-contrast subtle grid line
            },
          },
          y: {
            type: "linear" as const,
            display: true,
            position: "left" as const,
            title: { 
              display: true, 
              text: "15s Rolling DPS", 
              color: "#ffffff",
              font: { family: "Outfit, Inter, sans-serif", size: 13, weight: "bold" as const }
            },
            ticks: {
              color: "#cbd5e1",
              font: { family: "Outfit, Inter, sans-serif", size: 11 },
            },
            grid: {
              color: "rgba(255, 255, 255, 0.08)",
            },
          },
          yHp: {
            type: "linear" as const,
            display: false,
            min: 0,
            max: 100,
            grid: { drawOnChartArea: false },
          },
        },
        plugins: {
          annotation: {
            annotations,
          },
          zoom: {
            zoom: {
              wheel: {
                enabled: true,
                modifierKey: "ctrl" as const,
              },
              pinch: {
                enabled: true,
              },
              mode: "x" as const,
            },
            pan: {
              enabled: true,
              mode: "x" as const,
            },
          },
          legend: {
            labels: {
              color: "#ffffff",
              font: { family: "Outfit, Inter, sans-serif", size: 12, weight: "bold" as const },
              padding: 15,
            },
          },
          tooltip: {
            backgroundColor: "rgba(15, 23, 42, 0.95)", // dark premium slate tooltip
            titleColor: "#ffffff",
            bodyColor: "#cbd5e1",
            footerColor: "#38bdf8", // bright sky blue footer
            borderColor: "rgba(255, 255, 255, 0.1)",
            borderWidth: 1,
            padding: 12,
            titleFont: { family: "Outfit, Inter, sans-serif", weight: "bold" as const },
            bodyFont: { family: "Outfit, Inter, sans-serif" },
            footerFont: { family: "Outfit, Inter, sans-serif", weight: "bold" as const },
            itemSort: (a: TooltipItem<"line">, b: TooltipItem<"line">) => {
              // Keep HP at the bottom of the tooltip, then sort players by DPS descending
              if (a.dataset.yAxisID === "yHp") return 1;
              if (b.dataset.yAxisID === "yHp") return -1;
              return (b.raw as number) - (a.raw as number);
            },
            callbacks: {
              title: (tooltipItems: any) =>
                `Time: ${formatTimeLabel(parseFloat(tooltipItems[0].label))}`,
              footer: (tooltipItems: any) => {
                let totalDps = 0;
                tooltipItems.forEach((item: any) => {
                  if (item.dataset.yAxisID === "y") {
                    totalDps += item.raw;
                  }
                });
                return `Total Raid DPS: ${Math.round(totalDps).toLocaleString()}`;
              },
            },
          },
        },
      };
    });

    const chartData = computed(() => {
      const graphDataForView = fightSummary.graphData?.[selectedTargetId.value];

      if (!graphDataForView || Object.keys(graphDataForView).length === 0) {
        return { labels: [], datasets: [] };
      }

      const firstPlayerId = Object.keys(graphDataForView)[0];
      const labels = graphDataForView[firstPlayerId]?.map((p) => p.time) ?? [];
      const datasets: any[] = [];

      // Add Enemy HP dataset if a target is selected and has hpHistory
      if (selectedTargetId.value) {
        const targetStats = fightSummary.targets[selectedTargetId.value];
        if (targetStats && targetStats.hpHistory && targetStats.hpHistory.length > 0) {
          const getTargetHPAtTime = (t: number, hpHistory: any[]) => {
            if (!hpHistory || hpHistory.length === 0) return 100;
            let lastPt = hpHistory[0];
            for (const pt of hpHistory) {
              if (pt.time <= t) {
                lastPt = pt;
              } else {
                break;
              }
            }
            return lastPt.maxHp > 0 ? (lastPt.currentHp / lastPt.maxHp) * 100 : 0;
          };

          datasets.push({
            label: `${targetStats.name} HP %`,
            backgroundColor: "rgba(244, 63, 94, 0.05)", // smooth rose/red transparent area
            borderColor: "rgba(244, 63, 94, 0.25)",     // more defined boundary line
            data: labels.map((t) => getTargetHPAtTime(t, targetStats.hpHistory!)),
            yAxisID: "yHp",
            pointRadius: 0,
            borderWidth: 1.5,
            fill: true,
            tension: 0.3, // organic, curved line
            order: 99, // drawn behind DPS lines
          });
        }
      }

      for (const playerId in graphDataForView) {
        const playerData = graphDataForView[playerId];
        const isHidden = globalHideMode.value || hiddenPlayers.has(playerId);
        const player = fightSummary.players[playerId];

        let displayLabel = player?.name ?? "Unknown";
        let displayColor = getMabiNameColor(displayLabel);

        if (isHidden) {
          displayLabel = player?.talentName || "Hidden";
          displayColor = player?.talentColor || "#808080";
        } else {
          if (showClassColorsForVisiblePlayers.value && player?.talentColor) {
            displayColor = player.talentColor;
          }
        }

        datasets.push({
          label: `${displayLabel} - 15s DPS`,
          backgroundColor: displayColor,
          borderColor: displayColor,
          data: playerData.map((p) => p.rollingDPS),
          yAxisID: "y",
          pointRadius: 0,
          borderWidth: 3, // Thicker, bold lines that stand out wonderfully
          tension: 0.3,   // Curved lines for modern flowing aesthetic
          order: 1, // drawn in front of HP area
        });
      }

      return { labels, datasets };
    });

    const selectedTargetConditions = computed(() => {
        if (!selectedTargetId.value) return undefined;
        return fightSummary.targets[selectedTargetId.value]?.conditions;
    });

    return {
      chartData,
      chartOptions,
      selectedTargetConditions,
      encounterStartTime,
      xAxisConfig,
      selectedConditionId,
      conditionItems,
      showDeaths
    };
  },
});
</script>

<style scoped>
.graph-container {
  background: transparent;
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 12px;
  overflow: hidden;
  transition: all 0.3s ease;
}

.chart-wrapper {
  background: rgba(255, 255, 255, 0.02); /* extremely subtle gray backdrop overlay */
  border-radius: 8px;
  padding: 16px;
  border: 1px solid rgba(255, 255, 255, 0.03);
}
</style>
