<template>
  <div v-if="chartData.datasets.length > 0">
    <div class="pa-4">
      <div style="height: 500px">
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
  <v-sheet v-else class="d-flex align-center justify-center" height="400">
    <div class="text-h6 text-grey">
      Graph data is only available for saved sessions.
    </div>
  </v-sheet>
</template>

<script lang="ts">
import { defineComponent, computed } from "vue";
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

ChartJS.register(
  Title,
  Tooltip,
  Legend,
  LineElement,
  LinearScale,
  PointElement,
  CategoryScale,
  Filler
);

export default defineComponent({
  name: "DamageGraph",
  components: { LineChart, ConditionTimeline },
  setup() {
    const formatTimeLabel = (seconds: number) => {
      if (isNaN(seconds)) return "0:00";
      const minutes = Math.floor(seconds / 60);
      const remainingSeconds = Math.round(seconds % 60);
      return `${minutes}:${remainingSeconds.toString().padStart(2, "0")}`;
    };

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
      return {
        responsive: true,
        maintainAspectRatio: false,
        interaction: { mode: "index" as const, intersect: false },
        scales: {
          x: {
            type: "linear" as const,
            title: { display: true, text: "Time (minutes:seconds)" },
            // Set the max value of the axis to our calculated nice, round number
            max: xAxisConfig.value.max,
            ticks: {
              // Set the explicit step size for our ticks
              stepSize: xAxisConfig.value.stepSize,
              callback: function (
                this: Scale<CoreScaleOptions>,
                tickValue: string | number
              ) {
                return formatTimeLabel(Number(tickValue));
              },
            },
          },
          y: {
            type: "linear" as const,
            display: true,
            position: "left" as const,
            title: { display: true, text: "60s Rolling DPS" },
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
          tooltip: {
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
            backgroundColor: "rgba(239, 68, 68, 0.05)", // subtle transparent red
            borderColor: "rgba(239, 68, 68, 0.15)",     // subtle red border
            data: labels.map((t) => getTargetHPAtTime(t, targetStats.hpHistory!)),
            yAxisID: "yHp",
            pointRadius: 0,
            borderWidth: 1.5,
            fill: true,
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
          label: `${displayLabel} - 60s DPS`,
          backgroundColor: displayColor,
          borderColor: displayColor,
          data: playerData.map((p) => p.rollingDPS),
          yAxisID: "y",
          pointRadius: 0,
          borderWidth: 2,
          order: 1, // drawn in front of HP area
        });
      }

      return { labels, datasets };
    });

    const selectedTargetConditions = computed(() => {
        if (!selectedTargetId.value) return undefined;
        return fightSummary.targets[selectedTargetId.value]?.conditions;
    });

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

    return {
      chartData,
      chartOptions,
      selectedTargetConditions,
      encounterStartTime,
      xAxisConfig
    };
  },
});
</script>
