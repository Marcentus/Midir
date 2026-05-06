<template>
  <div v-if="chartData.datasets.length > 0" class="d-flex">
    <!-- Draggable Condition List -->
    <div 
      class="d-flex flex-column pr-2" 
      :style="{ 
        height: containerHeight + 'px', 
        paddingTop: CHART_HEADER_HEIGHT + 'px', 
        width: '40px' 
      }"
    >
      <div
        v-for="cond in sortedConditions"
        :key="cond.id"
        class="condition-item cursor-grab"
        :style="{ height: ROW_HEIGHT + 'px' }"
        draggable="true"
        @dragstart="onDragStart($event, cond.id)"
        @dragover.prevent="onDragOver($event, cond.id)"
        @drop="onDrop($event, cond.id)"
        @dragenter.prevent
        :title="getConditionName(cond.id)"
      >
        <img 
            :src="getConditionIcon(cond.id)" 
            width="24" 
            height="24" 
            class="rounded"
            style="pointer-events: none;" 
        />
      </div>
    </div>

    <!-- Chart -->
    <div class="flex-grow-1" :style="{ height: containerHeight + 'px' }">
      <BarChart :data="chartData" :options="chartOptions" />
    </div>
  </div>
</template>

<script lang="ts">
import { defineComponent, computed, inject, ref, watch } from "vue";
import { Bar as BarChart } from "vue-chartjs";
import {
  Chart as ChartJS,
  Title,
  Tooltip,
  Legend,
  BarElement,
  LinearScale,
  CategoryScale,
  TimeScale,
  TooltipItem,
} from "chart.js";
import { ConditionStats } from "@/protocols";
import { favoriteConditions, hiddenConditions, customConditionOrder, updateConditionOrder, fightSummary } from "@/store";
import { processConditions } from "@/conditionCombinations";

ChartJS.register(
  Title,
  Tooltip,
  Legend,
  BarElement,
  LinearScale,
  CategoryScale,
  TimeScale
);

export default defineComponent({
  name: "ConditionTimeline",
  components: { BarChart },
  props: {
    conditions: {
      type: Object as () => { [id: number]: ConditionStats } | undefined,
      required: false,
    },
    startTime: {
      type: Number,
      required: true,
    },
    xAxisConfig: {
      type: Object as () => { max: number; stepSize: number },
      required: true,
    },
  },
  setup(props) {
    const ROW_HEIGHT = 40;
    const CHART_HEADER_HEIGHT = 30;

    const condNameMap = inject("condNameMap") as any;

    const getConditionName = (id: number): string => {
      return condNameMap.value[id]?.name || `Unknown Condition ${id}`;
    };
    
    const getConditionIcon = (id: number): string => {
      return condNameMap.value[id]?.iconUrl || "";
    };

    // --- Drag and Drop Logic ---
    const draggedId = ref<number | null>(null);

    const onDragStart = (event: DragEvent, id: number) => {
        draggedId.value = id;
        if (event.dataTransfer) {
            event.dataTransfer.effectAllowed = 'move';
            event.dataTransfer.dropEffect = 'move';
        }
    };

    const onDragOver = (event: DragEvent, id: number) => {
        // Essential to allow drop
    };

    const onDrop = (event: DragEvent, targetId: number) => {
        const sourceId = draggedId.value;
        if (sourceId === null || sourceId === targetId) return;

        // Reorder sortedConditions
        // We need to work with the CURRENT sorted list to determine the new index
        const currentList = sortedConditions.value.map(c => c.id);
        const oldIndex = currentList.indexOf(sourceId);
        const newIndex = currentList.indexOf(targetId);

        if (oldIndex !== -1 && newIndex !== -1) {
            const newList = [...currentList];
            // Remove from old
            newList.splice(oldIndex, 1);
            // Insert at new
            newList.splice(newIndex, 0, sourceId);
            
            // Update the global store order
            // Note: The store stores ALL preferences, but we only reordered the VISIBLE ones.
            // We should merge this new specific order into the global order preference.
            // However, simpler is to just save this full list as the preference if it covers enough.
            // Or better: updateConditionOrder with this new list appended with any missing ones?
            // Let's just update with the full visible set for now, that's what the user cares about.
            updateConditionOrder(newList);
        }
        draggedId.value = null;
    };

    const sortedConditions = computed(() => {
        if (!props.conditions) return [];
        const processed = processConditions(props.conditions, fightSummary.encounterDuration);
        let list = processed.filter(c => !hiddenConditions.has(c.id));
        
        // Custom Order Map
        const orderMap = new Map<number, number>();
        customConditionOrder.value.forEach((id, index) => orderMap.set(id, index));

        return list.sort((a, b) => {
            // 1. Priority: Custom Order
            const aIndex = orderMap.has(a.id) ? orderMap.get(a.id)! : 9999;
            const bIndex = orderMap.has(b.id) ? orderMap.get(b.id)! : 9999;
            
            if (aIndex !== bIndex) return aIndex - bIndex;

            // 2. Fallback: Favorites
            const aFav = favoriteConditions.has(a.id);
            const bFav = favoriteConditions.has(b.id);
            if (aFav && !bFav) return -1;
            if (!aFav && bFav) return 1;

            // 3. Fallback: Duration
            return b.duration - a.duration;
        });
    });

    const containerHeight = computed(() => {
        const rows = sortedConditions.value.length;
        // Match standard bar thickness + spacing in Chart.js
        // If we set maintainAspectRatio: false and give explicit height, chart.js fits bars.
        // We want bars to align with our 32px icons.
        // Chart layout padding top/bottom affects this.
        return Math.max(100, rows * 30 + 30); 
    });

    const chartData = computed(() => {
      const labels: string[] = [];
      const dataPoints: any[] = [];
      const backgroundColors: string[] = [];
      const borderColors: string[] = [];

      sortedConditions.value.forEach((cond) => {
        const name = getConditionName(cond.id);
        const uniqueId = cond.id.toString(); // Use ID for unique Y-axis mapping
        labels.push(uniqueId); 

        const color = favoriteConditions.has(cond.id) ? 'rgba(255, 215, 0, 0.6)' : 'rgba(75, 192, 192, 0.6)';
        const borderColor = favoriteConditions.has(cond.id) ? 'rgba(255, 215, 0, 1)' : 'rgba(75, 192, 192, 1)';
        
        if (cond.intervals) {
            cond.intervals.forEach(iv => {
                let start = iv.start - props.startTime;
                let end = iv.end - props.startTime;
                if (start < 0) start = 0;
                if (end < start) end = start; 

                dataPoints.push({
                    x: [start, end],
                    y: uniqueId, // Map to ID
                    _name: name, // Store name for tooltip
                    metaData: iv.metaData,
                    attackerId: iv.attackerId
                });
                backgroundColors.push(color);
                borderColors.push(borderColor);
            });
        }
      });

      return { 
        labels, 
        datasets: [{
            label: "Condition Uptime",
            data: dataPoints,
            backgroundColor: backgroundColors,
            borderColor: borderColors,
            borderWidth: 1,
            barPercentage: 0.8,
            categoryPercentage: 0.9,
        }] 
      };
    });

    const formatTimeLabel = (seconds: number) => {
      if (isNaN(seconds)) return "0:00";
      const minutes = Math.floor(seconds / 60);
      const remainingSeconds = Math.round(seconds % 60);
      return `${minutes}:${remainingSeconds.toString().padStart(2, "0")}`;
    };

    const chartOptions = computed(() => {
      return {
        indexAxis: 'y' as const,
        responsive: true,
        maintainAspectRatio: false,
        layout: {
            padding: {
                top: 0,
                bottom: 0,
                left: 0,
                right: 0
            }
        },
        scales: {
          x: {
            type: "linear" as const,
            min: 0,
            max: props.xAxisConfig.max,
            position: 'top' as const,
            ticks: {
              stepSize: props.xAxisConfig.stepSize,
              callback: function (value: any) {
                return formatTimeLabel(Number(value));
              },
              autoSkip: true,
              maxRotation: 0,
            },
            grid: {
                display: true,
                drawOnChartArea: true,
            },
            afterFit: (axis: any) => {
                axis.height = CHART_HEADER_HEIGHT;
            }
          },
          y: {
            stacked: true,
            grid: { display: false },
            ticks: { display: false },
            afterFit: (axis: any) => {
                axis.width = 0; 
            }
          },
        },
        plugins: {
          legend: { display: false },
          tooltip: {
            callbacks: {
              title: (tooltipItems: any) => tooltipItems[0].raw._name, // Use stored name
              label: (item: TooltipItem<"bar">) => {
                const raw = item.raw as any;
                const start = raw.x[0];
                const end = raw.x[1];
                const duration = (end - start).toFixed(1);
                let text = `Time: ${formatTimeLabel(start)} - ${formatTimeLabel(end)} (${duration}s)`;
                if (raw.metaData) text += ` | Meta: ${raw.metaData}`;
                return text;
              },
            },
          },
        },
      };
    });

    return {
      chartData,
      chartOptions,
      containerHeight,
      sortedConditions,
      getConditionIcon,
      getConditionName,
      onDragStart,
      onDragOver,
      onDrop,
      ROW_HEIGHT,
      CHART_HEADER_HEIGHT
    };
  },
});
</script>

<style scoped>
.condition-item {
    display: flex;
    align-items: center;
    justify-content: center;
}
.cursor-grab {
    cursor: grab;
}
.cursor-grab:active {
    cursor: grabbing;
}
</style>
