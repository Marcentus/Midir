<template>
  <v-app>
    <v-app-bar flat class="modern-header" height="64">
      <v-btn
        icon="mdi-menu"
        variant="text"
        size="small"
        class="mr-2 action-btn"
        @click="isNavDrawerOpen = !isNavDrawerOpen"
      ></v-btn>
      <div class="brand-section">
        <img src="/images/Midir.png" class="mascot-img" alt="Midir Logo" />
        <div class="logo-text">Midir</div>
      </div>

      <v-spacer></v-spacer>

      <div class="action-section">
        <v-progress-circular
          v-if="isLoading"
          indeterminate
          color="primary"
          size="20"
          width="2"
          class="mr-4"
        ></v-progress-circular>

        <template v-if="activeTool === 'dps'">
          <v-btn
            variant="text"
            size="small"
            prepend-icon="mdi-content-save"
            class="action-btn"
            :loading="isSaving"
            @click="triggerSaveSession"
          >
            Save Session
          </v-btn>

          <v-btn
            variant="text"
            size="small"
            prepend-icon="mdi-refresh"
            class="action-btn"
            @click="triggerClearSession"
          >
            Clear
          </v-btn>

          <v-btn
            icon="mdi-cog"
            variant="text"
            size="small"
            class="ml-2 action-btn"
            @click="activeTool = 'settings'"
          ></v-btn>
        </template>

        <template v-else-if="activeTool === 'settings'">
          <v-btn
            variant="tonal"
            size="small"
            prepend-icon="mdi-arrow-left"
            color="primary"
            class="font-weight-bold"
            @click="activeTool = 'dps'"
          >
            Back to Meter
          </v-btn>
        </template>
      </div>
    </v-app-bar>

    <v-main>
      <damage-meter-view v-if="activeTool === 'dps'" />
      <settings-view v-else-if="activeTool === 'settings'" />
    </v-main>

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

    <!-- Custom Clear Confirmation Dialog -->
    <v-dialog v-model="clearDialogVisible" max-width="450px" persistent>
      <v-card class="dialog-card">
        <v-card-title class="text-h5 d-flex align-center pt-4 px-6">
          <v-icon color="warning" class="mr-2">mdi-alert-decagram</v-icon>
          Clear Live Session
        </v-card-title>
        <v-card-text class="pt-2 px-6 pb-4 text-grey-lighten-1">
          Are you sure you want to clear the current live session data? This will reset all current DPS meters and graph data. This action cannot be undone.
        </v-card-text>
        <v-card-actions class="px-6 pb-4">
          <v-spacer></v-spacer>
          <v-btn
            variant="text"
            class="font-weight-bold"
            @click="clearDialogVisible = false"
          >
            Cancel
          </v-btn>
          <v-btn
            color="warning"
            variant="flat"
            class="font-weight-bold ml-2"
            @click="confirmClearSession"
          >
            Clear
          </v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <!-- Custom Save Session Dialog -->
    <v-dialog v-model="saveDialogVisible" max-width="500px" persistent>
      <v-card class="dialog-card">
        <v-card-title class="text-h5 d-flex align-center pt-4 px-6">
          <v-icon color="primary" class="mr-2">mdi-content-save-cog</v-icon>
          Save Live Session
        </v-card-title>
        <v-card-text class="pt-2 px-6 pb-4 text-grey-lighten-1">
          Please enter a descriptive name for this live session to save it to your history.
          <v-text-field
            v-model="saveSessionName"
            label="Session Name"
            variant="outlined"
            density="comfortable"
            class="mt-4"
            hide-details
            autofocus
            color="primary"
            @keyup.enter="confirmSaveSession"
          ></v-text-field>
        </v-card-text>
        <v-card-actions class="px-6 pb-4">
          <v-spacer></v-spacer>
          <v-btn
            variant="text"
            class="font-weight-bold"
            @click="saveDialogVisible = false"
          >
            Cancel
          </v-btn>
          <v-btn
            color="primary"
            variant="flat"
            class="font-weight-bold ml-2"
            :loading="isSaving"
            :disabled="!saveSessionName.trim()"
            @click="confirmSaveSession"
          >
            Save
          </v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>
  </v-app>
</template>

<script lang="ts">
import { defineComponent, onMounted, onUnmounted, inject, ref, watch, computed, provide, reactive } from "vue";
import { socket } from "@/store"; // Static import
import { FightSummary } from "./protocols";
import { initPlayerCache } from "@/playerCache";
import DamageMeterView from "@/components/DamageMeterView.vue";
import SettingsView from "@/components/SettingsView.vue";
import {
  getSessionSummary,
  getLiveSummary,
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
  },
  setup() {
    const raceNameMap = inject("raceNameMap");
    const skillNameMap = inject("skillNameMap");
    const condNameMap = inject("condNameMap");
    const playerBatchUpdateEvent = ref<any[]>([]); 
    
    provide("playerBatchUpdateEvent", playerBatchUpdateEvent);
    
    const appEvent = inject("appEvent");

    const socketConnected = ref(false);

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

    const isSaving = ref(false);

    const clearDialogVisible = ref(false);
    
    const triggerClearSession = () => {
      clearDialogVisible.value = true;
    };

    const confirmClearSession = async () => {
      clearDialogVisible.value = false;
      await clearSession();
    };

    const saveDialogVisible = ref(false);
    const saveSessionName = ref("");

    const triggerSaveSession = () => {
      saveSessionName.value = `Session @ ${new Date().toLocaleTimeString()}`;
      saveDialogVisible.value = true;
    };

    const confirmSaveSession = async () => {
      const name = saveSessionName.value.trim();
      if (!name) return;
      
      saveDialogVisible.value = false;
      isSaving.value = true;
      try {
        await saveSession(name);
        await clearSession();
        appEvent.value.dispatchEvent(new CustomEvent("refresh-sessions"));
        alert(`Session "${name}" saved successfully.`);
      } catch (e) {
        console.error("Failed to save session:", e);
        alert(`Error saving session: ${e}`);
      } finally {
        isSaving.value = false;
      }
    };

    watch(
      activeSessionId,
      async (newId) => {
        console.log(`Active session changed to: ${newId}`);

        selectedTargetId.value = "";
        if (newId === "live") {
          socket.onSummary = handleSummary;
          try {
            if (loadingCount) loadingCount.value++;
            const summary = await getLiveSummary();
            handleSummary(summary);
          } catch (e) {
            console.error("Failed to load live summary:", e);
            alert(`Error loading live summary: ${e}`);
          } finally {
            if (loadingCount) loadingCount.value--;
          }
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
      isSaving,
      socketConnected,
      clearSession,
      activeTool,
      isNavDrawerOpen,
      systemErrorVisible,
      systemErrorMessage,
      systemErrorType,
      clearDialogVisible,
      saveDialogVisible,
      saveSessionName,
      triggerClearSession,
      confirmClearSession,
      triggerSaveSession,
      confirmSaveSession,
    };
  },
});
</script>

<style scoped>
.modern-header {
  background: transparent !important;
  z-index: 1000;
}

.brand-section {
  display: flex;
  align-items: center;
  gap: 12px;
}

.mascot-img {
  height: 42px;
  width: 42px;
  object-fit: cover;
  border-radius: 10px;
  border: 0px solid rgba(129, 138, 248, 0.3);
  box-shadow: 0 0 20px rgba(129, 138, 248, 0.2);
  transition: transform 0.3s ease;
}

.mascot-img:hover {
  transform: scale(1.1) rotate(5deg);
}

.logo-text {
  font-size: 1.5rem;
  font-weight: 900;
  letter-spacing: 0.2em;
  color: #fff;
  text-shadow: 0 0 15px rgba(129, 138, 248, 0.4);
}

.action-section {
  display: flex;
  align-items: center;
}

.action-btn {
  font-weight: 700 !important;
  letter-spacing: 0.02em !important;
  color: #fff !important;
  transition: all 0.2s ease;
}

.action-btn:hover {
  color: #818cf8 !important; /* Primary Indigo */
  background: rgba(129, 138, 248, 0.08);
}

:deep(.v-toolbar__content) {
  padding: 0 24px !important;
}
</style>

<style>
.v-application {
  background: linear-gradient(145deg, #161922 0%, #0e1015 100%) !important;
}

.dialog-card {
  background: #171b24 !important;
  border: 1px solid rgba(129, 138, 248, 0.15) !important;
  box-shadow: 0 8px 32px 0 rgba(0, 0, 0, 0.5) !important;
  border-radius: 12px !important;
}
</style>

