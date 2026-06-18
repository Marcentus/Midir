<template>
  <v-container fluid>
    <v-card class="mx-auto" max-width="800">
      <v-toolbar color="surface" flat>
        <v-toolbar-title class="text-h5">Settings</v-toolbar-title>
      </v-toolbar>
      
      <v-tabs v-model="activeTab" bg-color="surface">
        <v-tab value="capture"><v-icon start>mdi-lan</v-icon> Capture</v-tab>
        <v-tab value="appearance"><v-icon start>mdi-palette</v-icon> Appearance</v-tab>
      </v-tabs>
      
      <v-card-text>
        <v-window v-model="activeTab">
          <!-- CAPTURE TAB -->
          <v-window-item value="capture">
            <!-- CARD 1: CAPTURE CONFIGURATION -->
            <v-card variant="outlined" class="mb-4" style="border-color: rgba(140, 158, 255, 0.25) !important;">
              <v-card-item class="pb-1 pt-2">
                <div class="d-flex align-center justify-space-between w-100 flex-wrap">
                  <v-card-title class="text-subtitle-2 d-flex align-center font-weight-bold text-white" style="font-size: 0.875rem !important;">
                    <v-icon start size="small" color="primary" class="mr-2">mdi-cog</v-icon>
                    Capture Configuration
                  </v-card-title>
                  
                  <v-chip
                    v-if="captureStatus.is_running"
                    color="success"
                    size="small"
                    variant="tonal"
                    class="font-weight-bold px-2"
                    style="height: 22px; font-size: 10px;"
                  >
                    <v-icon start size="12" class="mr-1">mdi-radiobox-marked</v-icon>
                    Running: {{ captureStatus.nic || 'File/Unknown' }}
                    <span v-if="captureStatus.exitlag" class="ml-1 text-grey-lighten-2">(ExitLag)</span>
                  </v-chip>
                  <v-chip
                    v-else
                    color="warning"
                    size="small"
                    variant="tonal"
                    class="font-weight-bold px-2"
                    style="height: 22px; font-size: 10px;"
                  >
                    <v-icon start size="12" class="mr-1">mdi-pause-circle-outline</v-icon>
                    Stopped
                  </v-chip>
                </div>
              </v-card-item>

              <v-card-text class="pt-3">
                <v-form @submit.prevent="applyCaptureSettings">
                  <v-select
                    v-model="captureConfig.nicName"
                    :items="nics"
                    item-title="description"
                    item-value="name"
                    label="Network Interface (NIC)"
                    variant="outlined"
                    density="compact"
                    hide-details
                    class="mb-4"
                  >
                    <template v-slot:item="{ props, item }">
                      <v-list-item v-bind="props" :subtitle="item.raw.ip ? 'IP: ' + item.raw.ip : 'No IP'"></v-list-item>
                    </template>
                    <template v-slot:selection="{ item }">
                      <span>{{ item.raw.description || item.raw.name }} ({{ item.raw.ip || 'No IP' }})</span>
                    </template>
                  </v-select>

                  <v-switch
                    v-model="captureConfig.promiscuous"
                    label="Enable Npcap Promiscuous Mode"
                    color="primary"
                    hide-details
                    inset
                    class="mb-1"
                  ></v-switch>
                  <div class="text-caption text-grey-lighten-2 mb-4 ml-14">
                    Turn this ON if you are capturing packets from a mirrored switch port.
                    <div class="text-error mt-1">
                      <strong>Warning:</strong> Enabling this on the same PC you run the game will NGS you.
                    </div>
                  </div>

                  <v-switch
                    v-model="captureConfig.exitlag"
                    label="Enable ExitLag Routing"
                    color="primary"
                    hide-details
                    inset
                    class="mb-4"
                  ></v-switch>
                  
                  <div class="d-flex ga-2 mt-4">
                    <v-btn
                      v-if="!captureStatus.is_running"
                      color="primary"
                      type="submit"
                      prepend-icon="mdi-play"
                      :loading="isApplying"
                    >
                      Start Capture
                    </v-btn>
                    
                    <v-btn
                      v-if="captureStatus.is_running"
                      color="primary"
                      prepend-icon="mdi-refresh"
                      @click="restartCaptureKeepSession"
                      :loading="isRestartingKeepSession"
                    >
                      Reconnect Capture
                    </v-btn>

                    <v-btn
                      v-if="captureStatus.is_running"
                      color="error"
                      variant="outlined"
                      prepend-icon="mdi-stop"
                      @click="stopCapture"
                      :loading="isStopping"
                    >
                      Stop Capture
                    </v-btn>
                  </div>
                </v-form>
              </v-card-text>
            </v-card>

            <!-- CARD 2: SYSTEM DIAGNOSTICS & TELEMETRY -->
            <v-card variant="outlined" class="mb-4" style="border-color: rgba(140, 158, 255, 0.25) !important;">
              <v-card-item class="pb-1 pt-2">
                <div class="d-flex align-center justify-space-between w-100 flex-wrap">
                  <v-card-title class="text-subtitle-2 d-flex align-center font-weight-bold text-white" style="font-size: 0.875rem !important;">
                    <v-icon start size="small" color="primary" class="mr-2">mdi-chart-line</v-icon>
                    System Diagnostics & Telemetry
                  </v-card-title>
                </div>
              </v-card-item>
              
              <v-card-text class="pb-3">
                <v-row dense>
                  <!-- Col 1: Decoded Packets -->
                  <v-col cols="12" sm="6" class="py-1">
                    <div class="d-flex align-center">
                      <v-icon color="success" size="small" class="mr-2">mdi-swap-horizontal-bold</v-icon>
                      <div>
                        <div class="text-caption font-weight-bold text-white">Decoded Packets</div>
                        <div class="text-caption text-grey-lighten-3">
                          {{ formatNumber(packetStatus.total) }} total ({{ formatNumber(packetStatus.perSecond) }}/sec)
                        </div>
                      </div>
                    </div>
                  </v-col>

                  <!-- Col 2: Dropped Packets -->
                  <v-col cols="12" sm="6" class="py-1">
                    <div class="d-flex align-center">
                      <v-icon 
                        :color="((packetStatus.pcapDrops || 0) + (packetStatus.parserErrors || 0) + (packetStatus.networkLoss || 0) + (packetStatus.queueDrops || 0) > 0) ? 'red-lighten-1' : 'grey-lighten-1'" 
                        size="small" 
                        class="mr-2"
                      >
                        mdi-alert-circle
                      </v-icon>
                      <div>
                        <div class="text-caption font-weight-bold text-white">Dropped Packets</div>
                        <div class="text-caption text-grey-lighten-3">
                          Npcap: {{ packetStatus.pcapDrops || 0 }} · Parser: {{ packetStatus.parserErrors || 0 }} (Loss: {{ packetStatus.networkLoss || 0 }})
                        </div>
                      </div>
                    </div>
                  </v-col>

                  <!-- Col 3: Go Runtime Memory -->
                  <v-col cols="12" sm="6" class="py-1">
                    <div class="d-flex align-center">
                      <v-icon color="teal-lighten-2" size="small" class="mr-2">mdi-server</v-icon>
                      <div>
                        <div class="text-caption font-weight-bold text-white">Go Runtime Memory</div>
                        <div class="text-caption text-grey-lighten-3">
                          Heap: {{ formatBytes(packetStatus.heapAlloc || 0) }} ({{ packetStatus.goroutines || 0 }} threads)
                        </div>
                      </div>
                    </div>
                  </v-col>

                  <!-- Col 4: Active Tracked Entities -->
                  <v-col cols="12" sm="6" class="py-1">
                    <div class="d-flex align-center justify-space-between">
                      <div class="d-flex align-center">
                        <v-icon color="blue-lighten-2" size="small" class="mr-2">mdi-account-group</v-icon>
                        <div>
                          <div class="text-caption font-weight-bold text-white">Active Tracked Entities</div>
                          <div class="text-caption text-grey-lighten-3">
                            {{ packetStatus.trackedEntities || 0 }} cached entities
                          </div>
                        </div>
                      </div>
                      <v-btn
                        variant="tonal"
                        size="x-small"
                        color="blue-lighten-2"
                        class="ml-2 font-weight-bold"
                        style="font-size: 0.65rem !important;"
                        @click="showEntitiesDialog = true"
                      >
                        View List
                      </v-btn>
                    </div>
                  </v-col>
                </v-row>

                <v-divider class="my-3 border-opacity-25"></v-divider>

                <!-- Live Session Event Buffer Section -->
                <div class="d-flex align-center mb-2">
                  <v-icon color="amber-darken-1" size="small" class="mr-2">mdi-database</v-icon>
                  <div>
                    <div class="text-caption font-weight-bold text-white">Live Session Event Buffer</div>
                    <div class="text-caption text-grey-lighten-3">
                      {{ formatNumber(packetStatus.bufferEvents || 0) }} events ({{ formatBytes(packetStatus.bufferBytes || 0) }} total buffered)
                    </div>
                  </div>
                </div>

                <v-row dense class="mt-1">
                  <!-- Col 1: Core Combat & Entity Presence -->
                  <v-col cols="12" sm="4" class="py-1 px-2">
                    <div 
                      v-for="eventName in ['Damage', 'Entity Appear', 'Entity Disappear']"
                      :key="eventName"
                      class="d-flex justify-space-between align-center py-1 border-b" 
                      style="border-color: rgba(255, 255, 255, 0.08) !important; font-size: 0.75rem;"
                    >
                      <span class="text-grey-lighten-2 text-truncate mr-1">{{ eventName }}</span>
                      <span class="font-weight-bold text-white">{{ formatNumber(packetStatus.eventBreakdown?.[eventName] || 0) }}</span>
                    </div>
                  </v-col>

                  <!-- Col 2: Vitality & Status States -->
                  <v-col cols="12" sm="4" class="py-1 px-2">
                    <div 
                      v-for="eventName in ['HP Update', 'Entity Death', 'Entity Revive']"
                      :key="eventName"
                      class="d-flex justify-space-between align-center py-1 border-b" 
                      style="border-color: rgba(255, 255, 255, 0.08) !important; font-size: 0.75rem;"
                    >
                      <span class="text-grey-lighten-2 text-truncate mr-1">{{ eventName }}</span>
                      <span class="font-weight-bold text-white">{{ formatNumber(packetStatus.eventBreakdown?.[eventName] || 0) }}</span>
                    </div>
                  </v-col>

                  <!-- Col 3: Buffs & Session Metadata -->
                  <v-col cols="12" sm="4" class="py-1 px-2">
                    <!-- Blank first row for vertical alignment -->
                    <div class="py-1" style="font-size: 0.75rem; border-bottom: 1px solid transparent;">&nbsp;</div>
                    <div 
                      v-for="eventName in ['Condition Enable', 'Condition Disable']"
                      :key="eventName"
                      class="d-flex justify-space-between align-center py-1 border-b" 
                      style="border-color: rgba(255, 255, 255, 0.08) !important; font-size: 0.75rem;"
                    >
                      <span class="text-grey-lighten-2 text-truncate mr-1">{{ eventName }}</span>
                      <span class="font-weight-bold text-white">{{ formatNumber(packetStatus.eventBreakdown?.[eventName] || 0) }}</span>
                    </div>
                  </v-col>
                </v-row>
              </v-card-text>
            </v-card>
          </v-window-item>

          <!-- APPEARANCE TAB -->
          <v-window-item value="appearance">
            <v-list>
              <v-list-item>
                <template v-slot:prepend>
                  <v-icon icon="mdi-palette"></v-icon>
                </template>
                <v-list-item-title>Use Class Colors for Visible Players</v-list-item-title>
                <v-list-item-subtitle class="mt-1">
                  If enabled, visible players will be colored based on their active Talent/Arcana instead of a random color.
                  Hidden players always use Class Colors.
                </v-list-item-subtitle>
                <template v-slot:append>
                  <v-switch
                    v-model="showClassColorsForVisiblePlayers"
                    color="primary"
                    hide-details
                    inset
                  ></v-switch>
                </template>
              </v-list-item>

              <v-list-item class="mt-2">
                <template v-slot:prepend>
                  <v-icon icon="mdi-chart-bar"></v-icon>
                </template>
                <v-list-item-title>DPS Meter Bar Style</v-list-item-title>
                <v-list-item-subtitle class="mt-1">
                  Choose how the damage contribution percentage is displayed. Either as a dedicated column or filling the entire row.
                </v-list-item-subtitle>
                <template v-slot:append>
                  <v-btn-toggle
                    v-model="dpsMeterFillMode"
                    mandatory
                    density="compact"
                    color="primary"
                    variant="outlined"
                  >
                    <v-btn value="column" size="small">Column</v-btn>
                    <v-btn value="full" size="small">Full Row</v-btn>
                  </v-btn-toggle>
                </template>
              </v-list-item>
            </v-list>
            
            <v-divider class="my-4"></v-divider>
            
            <ColorSettings />
          </v-window-item>
        </v-window>
      </v-card-text>
    </v-card>
  </v-container>

  <!-- Dialog for Categorized Active Tracked Entities -->
  <v-dialog v-model="showEntitiesDialog" max-width="1600" width="95%">
    <v-card class="modern-card border border-opacity-10" style="background: linear-gradient(145deg, #181d28 0%, #10121a 100%) !important; box-shadow: 0 10px 30px rgba(0,0,0,0.5);">
      <v-card-title class="text-h6 font-weight-bold text-white d-flex align-center justify-space-between py-3 px-4" style="border-bottom: 1px solid rgba(255,255,255,0.08) !important;">
        <div class="d-flex align-center">
          <v-icon color="blue-lighten-2" class="mr-2">mdi-account-group</v-icon>
          <span>Active Tracked Entities</span>
        </div>
        <v-btn icon="mdi-close" variant="text" size="small" color="grey" @click="showEntitiesDialog = false"></v-btn>
      </v-card-title>
      
      <v-card-text class="pa-4" style="max-height: 800px; overflow-y: auto;">
        <div v-if="!fightSummary.currentEntities || fightSummary.currentEntities.length === 0" class="text-center text-grey py-8">
          <v-icon size="large" class="mb-2">mdi-alert-circle-outline</v-icon>
          <div>No active entities tracked in the current area.</div>
        </div>
        
        <div v-else>
          <v-row dense class="align-stretch">
            <v-col 
              v-for="(entities, category) in categorizedEntities" 
              :key="category"
              v-show="entities.length > 0"
              class="py-2 px-2 d-flex flex-column"
              style="min-width: 300px; flex: 1 1 300px; max-width: 100%;"
            >
              <div class="text-caption font-weight-bold text-blue-lighten-2 pb-1 mb-2 d-flex justify-space-between" style="border-bottom: 1px solid rgba(255,255,255,0.08) !important;">
                <span>{{ category }}</span>
                <span class="text-grey">({{ entities.length }})</span>
              </div>
              
              <div class="flex-grow-1" style="max-height: 650px; overflow-y: auto;">
                <div 
                  v-for="entity in entities" 
                  :key="entity.id"
                  class="px-3 py-2 rounded mb-2"
                  style="background: rgba(255, 255, 255, 0.03); border: 1px solid rgba(255,255,255,0.05); min-height: 52px;"
                >
                  <!-- Header: Name & HP -->
                  <div class="d-flex justify-space-between align-center">
                    <div class="text-subtitle-2 text-white font-weight-bold" style="word-break: break-word; overflow-wrap: anywhere; line-height: 1.2;">
                      {{ entity.name }}
                    </div>
                    <div v-if="entity.maxHp > 0" class="text-caption font-weight-bold text-green-lighten-2 flex-shrink-0 ml-2" style="font-size: 0.7rem;">
                      {{ formatNumber(Math.round(entity.currentHp)) }} / {{ formatNumber(Math.round(entity.maxHp)) }}
                    </div>
                  </div>

                  <!-- Thin HP Bar (spanning all the way across, below the name) -->
                  <div v-if="entity.maxHp > 0" class="mt-1.5 mb-2 w-100" style="height: 3px; background: rgba(255, 255, 255, 0.05); border-radius: 1.5px; overflow: hidden;">
                    <div 
                      :style="{ width: (entity.currentHp / entity.maxHp) * 100 + '%' }" 
                      style="height: 100%; background: #4caf50; transition: width 0.3s ease;"
                    ></div>
                  </div>
                  <div v-else class="mb-2"></div>

                  <!-- Structured Key-Value Fields -->
                  <div class="d-flex flex-column" style="gap: 3px;">
                    <!-- Row: Entity ID -->
                    <div class="d-flex justify-space-between text-caption text-grey" style="font-size: 0.68rem; line-height: 1.2;">
                      <span class="font-weight-medium">Entity ID</span>
                      <span style="font-family: monospace;" class="text-white">{{ entity.id }}</span>
                    </div>

                    <!-- Row: Owner ID (Only shown if available) -->
                    <div v-if="entity.ownerId" class="d-flex justify-space-between text-caption text-grey" style="font-size: 0.68rem; line-height: 1.2;">
                      <span class="font-weight-medium">Owner ID</span>
                      <span style="font-family: monospace;" class="text-white">{{ entity.ownerId }}</span>
                    </div>

                    <!-- Row: Race (Translated + RaceID in parentheses) -->
                    <div class="d-flex justify-space-between text-caption text-grey" style="font-size: 0.68rem; line-height: 1.2;">
                      <span class="font-weight-medium">Race</span>
                      <span class="text-white text-right font-weight-bold" style="word-break: break-word; overflow-wrap: anywhere; max-width: 70%;">
                        {{ entity.raceName }} <span class="text-grey ml-1 font-weight-normal">({{ entity.raceId }})</span>
                      </span>
                    </div>
                  </div>
                </div>
              </div>
            </v-col>
          </v-row>
        </div>
      </v-card-text>
      
      <v-card-actions class="justify-end py-2 px-4" style="border-top: 1px solid rgba(255,255,255,0.08) !important;">
        <v-btn color="primary" variant="text" @click="showEntitiesDialog = false">Close</v-btn>
      </v-card-actions>
    </v-card>
  </v-dialog>
</template>

<script lang="ts">
import { defineComponent, computed, ref, onMounted, onUnmounted } from "vue";
import { showClassColorsForVisiblePlayers, dpsMeterFillMode, socket, fightSummary } from "@/store";
import ColorSettings from "./ColorSettings.vue";

export default defineComponent({
  name: "SettingsView",
  components: {
    ColorSettings,
  },
  setup() {
    const activeTab = ref("capture");
    const showEntitiesDialog = ref(false);

    const categorizedEntities = computed(() => {
      const groups: Record<string, any[]> = {
        Players: [],
        Pets: [],
        Enemies: [],
        NPCs: [],
        Other: [],
      };
      
      if (fightSummary && fightSummary.currentEntities) {
        fightSummary.currentEntities.forEach((entity: any) => {
          const cat = entity.category || "Other";
          if (groups[cat]) {
            groups[cat].push(entity);
          } else {
            groups["Other"].push(entity);
          }
        });
      }
      
      return groups;
    });
    
    // --- CAPTURE SETTINGS ---
    const nics = ref<any[]>([]);
    const captureStatus = ref({ is_running: false, nic: '', exitlag: false, promiscuous: false });
    const packetStatus = ref<{
      total: number;
      perSecond: number;
      lastPacketAt: string;
      lastOp: number;
      topOps: { op: number; count: number; total: number }[];
      activeConditions?: number;
      trackedEntities?: number;
      bufferEvents?: number;
      bufferBytes?: number;
      goroutines?: number;
      heapAlloc?: number;
      eventBreakdown?: Record<string, number>;
      pcapDrops?: number;
      parserErrors?: number;
      networkLoss?: number;
      queueDrops?: number;
    }>({
      total: 0,
      perSecond: 0,
      lastPacketAt: '',
      lastOp: 0,
      topOps: [],
      activeConditions: 0,
      trackedEntities: 0,
      bufferEvents: 0,
      bufferBytes: 0,
      goroutines: 0,
      heapAlloc: 0,
      eventBreakdown: {},
      pcapDrops: 0,
      parserErrors: 0,
      networkLoss: 0,
      queueDrops: 0,
    });
    const isApplying = ref(false);
    const isStopping = ref(false);
    const isRestartingKeepSession = ref(false);
    const captureConfig = ref({
      nicName: "",
      ip: "",
      port: "",
      exitlag: false,
      promiscuous: false
    });

    const fetchNics = async () => {
      try {
        const res = await fetch("/api/setup/nics");
        if (res.ok) {
          nics.value = (await res.json()) || [];
          if (!captureConfig.value.nicName && nics.value.length > 0) {
            captureConfig.value.nicName = nics.value[0].name;
          }
        }
      } catch (err) {
        console.error("Failed to fetch nics:", err);
      }
    };

    const fetchStatus = async () => {
      try {
        const res = await fetch("/api/setup/status");
        if (res.ok) {
          const data = await res.json();
          captureStatus.value = data;
          
          if (data.nic) captureConfig.value.nicName = data.nic;
          captureConfig.value.exitlag = data.exitlag || false;
          captureConfig.value.promiscuous = data.promiscuous || false;
          if (data.ip) captureConfig.value.ip = data.ip;
          if (data.port) captureConfig.value.port = data.port;
        }
      } catch (err) {
        console.error("Failed to fetch capture status:", err);
      }
    };

    const applyCaptureSettings = async () => {
      if (!captureConfig.value.nicName) {
        alert("Please select a network interface.");
        return;
      }

      isApplying.value = true;
      try {
        const res = await fetch("/api/setup/start", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(captureConfig.value)
        });
        if (res.ok) {
          await fetchStatus();
        } else {
          const errMsg = await res.text();
          alert("Failed to start capture: " + errMsg);
        }
      } catch (err) {
        console.error("Error starting capture:", err);
        alert("Network error while trying to start capture.");
      } finally {
        isApplying.value = false;
      }
    };

    const restartCaptureKeepSession = async () => {
      if (!captureConfig.value.nicName) {
        alert("Please select a network interface.");
        return;
      }

      isRestartingKeepSession.value = true;
      try {
        const res = await fetch("/api/setup/restart-keep-session", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(captureConfig.value)
        });
        if (res.ok) {
          await fetchStatus();
        } else {
          const errMsg = await res.text();
          alert("Failed to reconnect capture: " + errMsg);
        }
      } catch (err) {
        console.error("Error reconnecting capture:", err);
        alert("Network error while trying to reconnect capture.");
      } finally {
        isRestartingKeepSession.value = false;
      }
    };

    const stopCapture = async () => {
      isStopping.value = true;
      try {
        const res = await fetch("/api/setup/stop", { method: "POST" });
        if (res.ok) {
           await fetchStatus();
        }
      } catch (err) {
        console.error("Error stopping capture:", err);
      } finally {
        isStopping.value = false;
      }
    };

    // --- AUTODETECT ---
    const isAutodetecting = ref(false);
    const autodetectProgress = ref(0);

    const startAutodetect = async () => {
      if (!captureConfig.value.nicName) return;
      isAutodetecting.value = true;
      autodetectProgress.value = 0;
      
      try {
        const res = await fetch("/api/setup/autodetect", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ 
            nicName: captureConfig.value.nicName,
            promiscuous: captureConfig.value.promiscuous
          })
        });
        
        if (!res.ok) {
          isAutodetecting.value = false;
          alert("Failed to start auto-detect: " + await res.text());
        }
      } catch (err) {
        isAutodetecting.value = false;
        console.error("Error starting autodetect:", err);
      }
    };

    const stopAutodetect = async () => {
      isAutodetecting.value = false;
      try {
        await fetch("/api/setup/autodetect/stop", { method: "POST" });
      } catch (err) {
        console.error("Error stopping autodetect:", err);
      }
    };

    onMounted(() => {
       fetchNics();
       fetchStatus();
       
       socket.onPacketStatus = (status) => {
         packetStatus.value = { ...status, topOps: status.topOps || [] };
       };

       socket.onAutodetectProgress = (progress) => {
         if (isAutodetecting.value) {
           autodetectProgress.value = progress.current;
         }
       };
       
       socket.onAutodetectDone = (result) => {
         if (isAutodetecting.value) {
           captureConfig.value.ip = result.ip;
           captureConfig.value.port = result.port;
           isAutodetecting.value = false;
           autodetectProgress.value = 5;
         }
       };
    });
    
    onUnmounted(() => {
       socket.onPacketStatus = undefined;
       socket.onAutodetectProgress = undefined;
       socket.onAutodetectDone = undefined;
       if (isAutodetecting.value) {
           fetch("/api/setup/autodetect/stop", { method: "POST" }).catch(console.error);
       }
    });

    const formatBytes = (bytes: number) => {
      if (!bytes) return '0 B';
      const k = 1024;
      const sizes = ['B', 'KB', 'MB', 'GB'];
      const i = Math.floor(Math.log(bytes) / Math.log(k));
      return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
    };

    const formatNumber = (num: number) => {
      return num.toLocaleString();
    };

    return {
      activeTab,
      showClassColorsForVisiblePlayers,
      dpsMeterFillMode,
      
      // Capture
      nics,
      captureStatus,
      packetStatus,
      captureConfig,
      isApplying,
      isStopping,
      isRestartingKeepSession,
      applyCaptureSettings,
      restartCaptureKeepSession,
      stopCapture,
      
      // Autodetect
      isAutodetecting,
      autodetectProgress,
      startAutodetect,
      stopAutodetect,

      // Helpers
      formatBytes,
      formatNumber,

      // Entities Dialog
      fightSummary,
      showEntitiesDialog,
      categorizedEntities
    };
  },
});
</script>

<style scoped>
:deep(.v-list-item-title) {
    white-space: normal !important;
}
:deep(.v-list-item-subtitle) {
    white-space: normal !important;
    overflow: visible !important;
    display: block !important;
    line-clamp: unset !important;
    -webkit-line-clamp: unset !important;
}
</style>
