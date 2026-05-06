<template>
  <v-app>
    <navigation />

    <v-app-bar>
      <v-app-bar-nav-icon 
        v-if="activeTool === 'dps'"
        @click="toggleNavDrawer"
      ></v-app-bar-nav-icon>
      <v-toolbar-title>{{ currentToolTitle }}</v-toolbar-title>
      <v-spacer></v-spacer>

      <v-progress-circular
        v-if="isLoading"
        indeterminate
        color="primary"
        size="24"
        width="2"
        class="mr-4"
      ></v-progress-circular>

      <span class="mr-4">
        API:
        <span v-if="socketConnected">
          <v-icon icon="mdi-check" color="success" /> Connected
        </span>
        <span v-else>
          <v-icon icon="mdi-close" color="error" /> Disconnected
        </span>
      </span>

      <template v-if="activeTool === 'dps'">
        <v-btn
          @click="promptAndSaveSession"
          color="primary"
          size="small"
          prepend-icon="mdi-content-save"
          class="ml-1"
          >Save Session</v-btn
        >

        <v-btn
          @click="clearSession"
          color="primary"
          size="small"
          prepend-icon="mdi-close"
          class="ml-1"
          >Clear</v-btn
        >
      </template>
    </v-app-bar>

    <v-snackbar
      v-model="systemErrorVisible"
      :color="systemErrorType"
      timeout="-1"
      location="bottom right"
      multi-line
    >
      <div class="d-flex flex-column">
        <strong>{{ systemErrorType === 'error' ? 'System Error Recovered' : 'Network Warning' }}</strong>
        <span>{{ systemErrorMessage }}</span>
      </div>
      <template v-slot:actions>
        <v-btn
          color="white"
          variant="text"
          @click="systemErrorVisible = false"
        >
          Close
        </v-btn>
      </template>
    </v-snackbar>

    <v-main>
      <damage-meter-view v-if="activeTool === 'dps'" />
      <settings-view v-else-if="activeTool === 'settings'" />
    </v-main>
  </v-app>
</template>

<script lang="ts">
import { defineComponent, onMounted, onUnmounted, inject, ref, watch, computed, provide, reactive } from "vue";
import { socket } from "@/store"; // Static import
import { FightSummary } from "./protocols";
import { initPlayerCache } from "@/playerCache";
import DamageMeterView from "@/components/DamageMeterView.vue";
import SettingsView from "@/components/SettingsView.vue";
import Navigation from "@/components/Navigation.vue";
import {
  getSessionSummary,
  saveSession,
  clearBackendState,
  getSkills,
  getRaces,
  getConditions,
  getOverrides,
} from "@/apicall";
import {
  activeSessionId,
  isNavDrawerOpen,
  fightSummary,
  selectedTargetId,
  activeTool,
  loadingCount,
  isLoading,
} from "@/store";

export default defineComponent({
  name: "App",
  components: {
    DamageMeterView,
    SettingsView,
    Navigation,
  },
  setup() {
    const raceNameMap = inject("raceNameMap");
    const skillNameMap = inject("skillNameMap");
    const condNameMap = inject("condNameMap");
    const playerBatchUpdateEvent = ref<any[]>([]); 
    
    provide("playerBatchUpdateEvent", playerBatchUpdateEvent);
    
    const appEvent = inject("appEvent");

    const socketConnected = ref(false);

    const toggleNavDrawer = () => {
      isNavDrawerOpen.value = !isNavDrawerOpen.value;
    };

    const currentToolTitle = computed(() => {
      switch (activeTool.value) {
        case 'dps': return 'Damage Meter';
        case 'settings': return 'Settings';
        default: return 'Midir';
      }
    });

    // Socket is now statically imported
    socket.onConnect = (isConnected) => (socketConnected.value = isConnected);

    const handleSummary = (summary: FightSummary) => {
      summary.players = summary.players || {};
      summary.targets = summary.targets || {};
      summary.damageTaken = summary.damageTaken || {};
      summary.graphData = summary.graphData || {};
      summary.currentEntities = summary.currentEntities || [];
      Object.assign(fightSummary, summary);
    };
    
    // Handle player updates
    socket.onPlayerBatchUpdate = (players) => {
        playerBatchUpdateEvent.value = players;
    };
    
    // Handle system errors
    const systemErrorVisible = ref(false);
    const systemErrorMessage = ref("");
    const systemErrorType = ref<"error" | "warning">("error");

    socket.onSystemError = (msg) => {
      console.warn("System Error Received:", msg);
      systemErrorMessage.value = msg;
      systemErrorType.value = "error";
      systemErrorVisible.value = true;
    };

    const clearSession = async () => {
      selectedTargetId.value = "";
      Object.assign(fightSummary, {
        encounterDuration: 0,
        totalDamage: 0,
        players: {},
        targets: {},
        damageTaken: {},
        graphData: {},
      });

      try {
        await clearBackendState();
        console.log("Backend state cleared.");
      } catch (e) {
        console.error("Failed to clear backend state:", e);
      }
    };

    const promptAndSaveSession = async () => {
      const sessionName = prompt(
        "Enter a name for this session:",
        `Session @ ${new Date().toLocaleTimeString()}`
      );

      if (sessionName) {
        try {
          await saveSession(sessionName);
          await clearSession();
          appEvent.value.dispatchEvent(new CustomEvent("refresh-sessions"));
          alert(`Session "${sessionName}" saved successfully.`);
        } catch (e) {
          console.error("Failed to save session:", e);
          alert(`Error saving session: ${e}`);
        }
      }
    };

    watch(
      activeSessionId,
      async (newId) => {
        console.log(`Active session changed to: ${newId}`);

        selectedTargetId.value = "";
        if (newId === "live") {
          socket.onSummary = handleSummary;
          clearSession();
        } else if (newId && newId !== "file-load") {
          socket.onSummary = undefined;
          try {
            if (loadingCount) loadingCount.value++;
            const summary = await getSessionSummary(newId);
            handleSummary(summary);
          } catch (e) {
            console.error("Failed to load session summary:", e);
            alert(`Error loading session: ${e}`);
          } finally {
            if (loadingCount) loadingCount.value--;
          }
        } else {
          socket.onSummary = undefined;
        }
      },
      { immediate: true }
    );

    onMounted(async () => {
      initPlayerCache();
      try {
        loadingCount.value++;
        const [skills, races, conditions, overrides, captureStatusResp] = await Promise.all([
          getSkills(),
          getRaces(),
          getConditions(),
          getOverrides(),
          fetch("/api/setup/status").catch(() => null)
        ]);

        if (captureStatusResp && captureStatusResp.ok) {
            const statusData = await captureStatusResp.json();
            if (!statusData.is_running) {
                activeTool.value = "settings";
            }
        }
        for (const idStr in skills) {
          const skill = skills[idStr];
          if (skill.iconUrl && !skill.iconUrl.startsWith('/') && !skill.iconUrl.startsWith('http')) {
            skill.iconUrl = `/images/skills/${skill.iconUrl}`;
          }
          skillNameMap.value[parseInt(idStr, 10)] = skill;
        }
        for (const idStr in races) {
          raceNameMap.value[parseInt(idStr, 10)] =
            `${races[idStr].name} ${idStr}`;
        }
        for (const idStr in conditions) {
          const cond = conditions[idStr];
          if (cond.iconUrl && !cond.iconUrl.startsWith('/') && !cond.iconUrl.startsWith('http')) {
            cond.iconUrl = `/images/conditions/${cond.iconUrl}`;
          }
          condNameMap.value[parseInt(idStr, 10)] = cond;
        }

        // Apply overrides
        if (overrides) {
          if (overrides.skills) {
            for (const idStr in overrides.skills) {
              const id = parseInt(idStr, 10);
              const override = overrides.skills[idStr];
              const existing = skillNameMap.value[id] || { name: `Skill ${id}` };
              if (override.name) existing.name = override.name;
              if (override.iconUrl) {
                existing.iconUrl = override.iconUrl.startsWith('/') ? override.iconUrl : `/images/${override.iconUrl}`;
              }
              skillNameMap.value[id] = existing;
            }
          }
          if (overrides.conditions) {
            for (const idStr in overrides.conditions) {
              const id = parseInt(idStr, 10);
              const override = overrides.conditions[idStr];
              const existing = condNameMap.value[id] || { name: `Condition ${id}` };
              if (override.name) existing.name = override.name;
              if (override.iconUrl) {
                existing.iconUrl = override.iconUrl.startsWith('/') ? override.iconUrl : `/images/${override.iconUrl}`;
              }
              condNameMap.value[id] = existing;
            }
          }
        }

        // Register virtual combined conditions
        const { CONDITION_COMBINATIONS } = await import("@/conditionCombinations");
        for (const combo of CONDITION_COMBINATIONS) {
          const baseCond = condNameMap.value[combo.iconId];
          condNameMap.value[combo.id] = {
            name: combo.name,
            iconUrl: baseCond?.iconUrl || ""
          };
        }

      } catch (e) {
        console.error("Failed to load static data:", e);
        alert("Could not load game data from server.");
      } finally {
        loadingCount.value--;
      }
      socket.connect(); // Ensure we connect here
    });

    return {
      isLoading,
      socketConnected,
      clearSession,
      promptAndSaveSession,
      toggleNavDrawer,
      activeTool,
      currentToolTitle,
      systemErrorVisible,
      systemErrorMessage,
      systemErrorType,
    };
  },
});
</script>
