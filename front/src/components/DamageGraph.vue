<template>
  <div v-if="chartData.datasets.length > 0">
    <div class="pa-4">
      <div class="d-flex ga-2 mb-2">
        <v-btn @click="toggleTotalDmg" size="small">Toggle Total Damage</v-btn>
        <v-btn @click="toggleRollingDps" size="small">Toggle 180s DPS</v-btn>
      </div>
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
import { defineComponent, computed, ref } from "vue";
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
  CategoryScale
);

export default defineComponent({
  name: "DamageGraph",
  components: { LineChart, ConditionTimeline },
  setup() {
    const showTotalDmg = ref(true);
    const showRollingDps = ref(true);

    const toggleTotalDmg = () => (showTotalDmg.value = !showTotalDmg.value);
    const toggleRollingDps = () =>
      (showRollingDps.value = !showRollingDps.value);

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
            title: { display: true, text: "Total Damage" },
          },
          y1: {
            type: "linear" as const,
            display: true,
            position: "right" as const,
            title: { display: true, text: "180s Rolling DPS" },
            grid: { drawOnChartArea: false },
          },
        },
        plugins: {
          tooltip: {
            itemSort: (a: TooltipItem<"line">, b: TooltipItem<"line">) => {
              const aIsTotalDmg = a.dataset.yAxisID === "y";
              const bIsTotalDmg = b.dataset.yAxisID === "y";
              if (aIsTotalDmg && !bIsTotalDmg) return -1;
              if (!aIsTotalDmg && bIsTotalDmg) return 1;
              return (b.raw as number) - (a.raw as number);
            },
            callbacks: {
              title: (tooltipItems: any) =>
                `Time: ${formatTimeLabel(parseFloat(tooltipItems[0].label))}`,
              footer: (tooltipItems: any) => {
                let totalDamage = 0;
                tooltipItems.forEach((item: any) => {
                  if (item.dataset.yAxisID === "y") {
                    totalDamage += item.raw;
                  }
                });
                return `Total Raid Damage: ${totalDamage.toLocaleString()}`;
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

      for (const playerId in graphDataForView) {
        const playerData = graphDataForView[playerId];
        
        // Debugging Reactivity
        // console.log(`[DamageGraph] Processing ${playerId}. GlobalHide: ${globalHideMode.value}, Hidden: ${hiddenPlayers.has(playerId)}`);
        
        const isHidden = globalHideMode.value || hiddenPlayers.has(playerId);
        const player = fightSummary.players[playerId];

        let displayLabel = player?.name ?? "Unknown";
        let displayColor = getMabiNameColor(displayLabel);

        if (isHidden) {
          // Hidden Mode: Use Talent Name and Talent Color
          // If no talent info, fallback to "Hidden" and Grey
          displayLabel = player?.talentName || "Hidden";
          displayColor = player?.talentColor || "#808080";
        } else {
          // Visible Mode: Use Real Name
          // Color depends on preference
          if (showClassColorsForVisiblePlayers.value && player?.talentColor) {
            displayColor = player.talentColor;
          }
        }

        datasets.push({
          label: `${displayLabel} - Total Dmg`,
          backgroundColor: displayColor,
          borderColor: displayColor,
          data: playerData.map((p) => p.totalDamage),
          yAxisID: "y",
          pointRadius: 0,
          borderWidth: 2.5,
          hidden: !showTotalDmg.value,
        });

        datasets.push({
          label: `${displayLabel} - 180s DPS`,
          backgroundColor: displayColor,
          borderColor: displayColor,
          data: playerData.map((p) => p.rollingDPS),
          yAxisID: "y1",
          pointRadius: 0,
          borderWidth: 1.5,
          hidden: !showRollingDps.value,
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
        // If sorting by target, ideally we use the target's start time for relative sync.
        // However, the graph X-axis is "Time since Encounter Start" (0-based) for general view,
        // BUT specific target graph data is typically relative to that target's engagement?
        // Let's check how graph data is generated. 
        // In aggregator.go: `Time: t - startTime` where startTime is targetStartTime if filtered.
        // So graph X=0 corresponds to targetStartTime.
        
        // However, condition intervals are ABSOLUTE timestamps (Unix).
        // So we need to pass the reference "Zero Point" to the timeline component.
        
        // If selectedTargetId present:
        // We need to find the start time used for that target's graph generation.
        // The backend `processEventsForSummary` & `generateGraphDataFromEvents` uses `targetTimes.StartTime`.
        // The issue is `FightSummary` struct doesn't strictly expose per-target StartTime cleanly in a map,
        // but `DamageBreakdown` has `startTime`.
        
        // Let's look up the start time from a player's breakdown against this target.
        // Use the first available player which engaged this target.
        
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
      toggleTotalDmg,
      toggleRollingDps,
      selectedTargetConditions,
      encounterStartTime,
      xAxisConfig
    };
  },
});
</script>
