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
            <v-alert
              v-if="captureStatus.is_running"
              type="success"
              variant="tonal"
              class="mb-4"
              density="compact"
            >
              Capture is currently <strong>running</strong> on NIC: {{ captureStatus.nic || 'File/Unknown' }} 
              <span v-if="captureStatus.exitlag">(ExitLag Enabled)</span>
            </v-alert>
            <v-alert
              v-else
              type="warning"
              variant="tonal"
              class="mb-4"
              density="compact"
            >
              Capture is currently <strong>stopped</strong>. Waiting for configuration.
            </v-alert>

            <v-alert
              v-if="captureStatus.is_running"
              :type="packetStatus.perSecond > 0 ? 'success' : 'info'"
              variant="tonal"
              class="mb-4"
              density="compact"
            >
              <div class="d-flex align-center justify-space-between flex-wrap ga-2">
                <div>
                  <strong>Decoded packets:</strong>
                  {{ packetStatus.total }} total,
                  {{ packetStatus.perSecond }}/sec
                  <span v-if="packetStatus.lastOp"> · last op: 0x{{ packetStatus.lastOp.toString(16) }}</span>
                </div>
                <div class="text-caption text-grey">
                  {{ packetStatus.perSecond > 0 ? 'Midir is receiving game packets.' : 'No decoded game packets this second.' }}
                </div>
              </div>
            </v-alert>

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
              <div class="text-caption text-grey mb-4 ml-14">
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
                class="mb-2"
              ></v-switch>

              <v-expand-transition>
                <div v-show="captureConfig.exitlag">
                  <div class="mb-4 pa-3 border border-opacity-25 rounded">
                    <div class="d-flex align-center justify-space-between mb-3">
                    <div>
                       <div class="text-subtitle-2 font-weight-bold text-primary">ExitLag Configuration</div>
                       <div class="text-caption text-grey">Auto-detect or manually enter your routed game IP/Port.</div>
                    </div>
                    
                    <v-btn 
                      v-if="!isAutodetecting"
                      size="small" 
                      color="secondary" 
                      variant="elevated"
                      :disabled="!captureConfig.nicName"
                      @click="startAutodetect"
                      prepend-icon="mdi-radar"
                    >
                      Auto-Detect IP/Port
                    </v-btn>
                    <v-btn 
                      v-if="isAutodetecting"
                      size="small" 
                      color="error" 
                      variant="elevated"
                      @click="stopAutodetect"
                    >
                      Cancel Detection
                    </v-btn>
                  </div>
                  
                  <v-expand-transition>
                    <div v-if="isAutodetecting" class="mb-4">
                      <v-alert type="info" variant="tonal" density="compact" class="mb-2">
                        Please run around in-game to generate movement packets.
                      </v-alert>
                      <v-progress-linear 
                        :model-value="(autodetectProgress / 5) * 100" 
                        color="primary" 
                        height="20" 
                        striped
                      >
                        <template v-slot:default>
                          <strong>{{ autodetectProgress }} / 5 Packets Detected</strong>
                        </template>
                      </v-progress-linear>
                    </div>
                  </v-expand-transition>

                  <v-row>
                    <v-col cols="12" md="6">
                      <v-text-field
                        v-model="captureConfig.ip"
                        label="ExitLag Target IP"
                        placeholder="e.g. 192.168.x.x"
                        variant="outlined"
                        density="compact"
                        hide-details
                        :disabled="isAutodetecting"
                      ></v-text-field>
                    </v-col>
                    <v-col cols="12" md="6">
                      <v-text-field
                        v-model="captureConfig.port"
                        label="ExitLag Target Port"
                        placeholder="e.g. 11020"
                        variant="outlined"
                        density="compact"
                        hide-details
                        :disabled="isAutodetecting"
                      ></v-text-field>
                    </v-col>
                  </v-row>
                </div>
                </div>
              </v-expand-transition>
              <div class="d-flex ga-2 mt-4">
                <v-btn
                  color="primary"
                  type="submit"
                  prepend-icon="mdi-play"
                  :loading="isApplying"
                >
                  {{ captureStatus.is_running ? 'Apply & Restart Capture' : 'Start Capture' }}
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
              <div class="text-caption text-grey mt-2">
                Note: Restarting the capture will clear the current ongoing live session data to avoid corrupted aggregator states.
              </div>
            </v-form>
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
            </v-list>
            
            <v-divider class="my-4"></v-divider>
            
            <ColorSettings />
          </v-window-item>
        </v-window>
      </v-card-text>
    </v-card>
  </v-container>
</template>

<script lang="ts">
import { defineComponent, computed, ref, onMounted, onUnmounted } from "vue";
import { showClassColorsForVisiblePlayers, activeTool, socket } from "@/store";
import ColorSettings from "./ColorSettings.vue";

export default defineComponent({
  name: "SettingsView",
  components: {
    ColorSettings,
  },
  setup() {
    const activeTab = ref("capture");
    
    // --- CAPTURE SETTINGS ---
    const nics = ref<any[]>([]);
    const captureStatus = ref({ is_running: false, nic: '', exitlag: false, promiscuous: false });
    const packetStatus = ref({ total: 0, perSecond: 0, lastPacketAt: '', lastOp: 0 });
    const isApplying = ref(false);
    const isStopping = ref(false);
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
          activeTool.value = "dps"; 
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
         packetStatus.value = status;
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

    return {
      activeTab,
      showClassColorsForVisiblePlayers,
      
      // Capture
      nics,
      captureStatus,
      packetStatus,
      captureConfig,
      isApplying,
      isStopping,
      applyCaptureSettings,
      stopCapture,
      
      // Autodetect
      isAutodetecting,
      autodetectProgress,
      startAutodetect,
      stopAutodetect
    };
  },
});
</script>

<style scoped>
.v-list-item-title {
    white-space: normal;
}
.v-list-item-subtitle {
    white-space: normal;
}
</style>
