<template>
  <v-navigation-drawer
    v-model="isNavDrawerOpen"
    floating
    class="modern-drawer"
    width="320"
  >
    <div class="d-flex flex-column fill-height" ref="mainContainerRef">
      <!-- Sticky Header Section -->
      <div ref="headerRef">
        <v-list-item>
          <v-list-item-title class="text-h6"> Sessions </v-list-item-title>
        </v-list-item>

        <div class="px-3 pb-2">
          <v-text-field
            v-model="searchQuery"
            density="compact"
            variant="solo-filled"
            hide-details
            placeholder="Search name, player, arcana..."
            prepend-inner-icon="mdi-magnify"
            clearable
            class="text-caption"
          ></v-text-field>
        </div>

        <v-divider></v-divider>

        <v-list density="compact" nav>
          <v-list-item
            prepend-icon="mdi-access-point"
            title="Live"
            :active="activeSessionId === 'live'"
            @click="selectSession('live')"
          >
            <template v-slot:append>
              <v-icon color="red" icon="mdi-record" />
            </template>
          </v-list-item>
        </v-list>
        
        <v-divider></v-divider>
      </div>

      <div class="d-flex flex-column" style="flex: 1 1 auto; overflow: hidden;">
          <v-list density="compact" nav class="fill-height pa-0 overflow-hidden">
            <v-virtual-scroll
              :items="filteredSessions"
              item-height="95"
              height="100%"
            >
              <template v-slot="{ item }">
                <v-list-item
                  :key="item.id"
                  :active="activeSessionId === item.id"
                  @click="selectSession(item.id)"
                  class="session-item py-2"
                >
                  <div class="d-flex flex-column w-100">
                    <!-- Top Row: Name | Datetime -->
                    <div class="d-flex align-center justify-space-between w-100 mb-2">
                      <div class="text-subtitle-2 font-weight-bold text-truncate" style="flex: 1; min-width: 0;">{{ item.name }}</div>
                      
                      <!-- Right: Datetime -->
                      <div class="text-caption text-grey text-right" style="flex: 1;">
                        {{ formatSessionTime(item) }}
                      </div>
                    </div>
                    
                    <!-- Middle Row: Players & Enemies -->
                    <div v-if="item.summary" class="d-flex" style="gap: 16px;">
                      <!-- Players Column -->
                      <div v-if="item.summary.players && item.summary.players.length > 0" class="d-flex flex-column" style="gap: 2px; flex: 1; min-width: 0;">
                        <template v-for="(p, idx) in item.summary.players.slice(0, 3)" :key="'p'+idx">
                          <div class="d-flex align-center" style="gap: 4px;">
                            <v-img v-if="p.arcanaIcon" :src="p.arcanaIcon" width="14" height="14" class="rounded flex-shrink-0" style="flex: 0 0 14px;"></v-img>
                            <v-icon v-else size="14" color="grey" style="flex: 0 0 14px;">mdi-account</v-icon>
                            <span class="text-truncate" style="font-size: 11px;">{{ p.name }}</span>
                          </div>
                        </template>
                        <span v-if="item.summary.players.length > 3" class="text-grey mt-1" style="font-size: 10px;">
                          +{{ item.summary.players.length - 3 }} more
                        </span>
                      </div>
                      
                      <!-- Enemies Column -->
                      <div v-if="item.summary.enemies && item.summary.enemies.length > 0" class="d-flex flex-column" style="gap: 2px; flex: 1; min-width: 0;">
                        <template v-for="(e, idx) in getUniqueEnemies(item.summary.enemies).slice(0, 3)" :key="'e'+idx">
                          <div class="d-flex align-center" style="gap: 4px;">
                            <v-icon size="14" color="red-lighten-2" style="flex: 0 0 14px;">mdi-sword-cross</v-icon>
                            <span class="text-truncate text-grey" style="font-size: 11px;">{{ e.name }}</span>
                          </div>
                        </template>
                        <span v-if="getUniqueEnemies(item.summary.enemies).length > 3" class="text-grey mt-1" style="font-size: 10px;">
                          +{{ getUniqueEnemies(item.summary.enemies).length - 3 }} more
                        </span>
                      </div>
                    </div>
                  </div>

                  <template v-slot:append>
                    <div class="session-actions align-self-start pl-2">
                      <v-btn variant="text" icon="mdi-pencil" size="x-small" @click.stop="openRenameDialog(item)" density="compact"></v-btn>
                      <v-btn variant="text" icon="mdi-delete" color="red-lighten-2" size="x-small" @click.stop="openDeleteDialog(item)" density="compact"></v-btn>
                    </div>
                  </template>
                </v-list-item>
                <v-divider></v-divider>
              </template>
            </v-virtual-scroll>
          </v-list>
      </div>
    </div>
  </v-navigation-drawer>
  <v-dialog v-model="renameDialog.visible" max-width="500px">
    <v-card>
      <v-card-title> Rename Session </v-card-title>
      <v-card-text>
        <v-text-field
          v-model="renameDialog.newName"
          label="New session name"
          @keydown.enter="saveRename"
        ></v-text-field>
      </v-card-text>
      <v-card-actions>
        <v-spacer></v-spacer>
        <v-btn variant="text" @click="renameDialog.visible = false"
          >Cancel</v-btn
        >
        <v-btn color="primary" @click="saveRename">Save</v-btn>
      </v-card-actions>
    </v-card>
  </v-dialog>

  <!-- ADDED DELETE CONFIRMATION DIALOG -->
  <v-dialog v-model="deleteDialog.visible" max-width="500px">
    <v-card>
      <v-card-title class="text-h5">Confirm Deletion</v-card-title>
      <v-card-text>
        Are you sure you want to permanently delete the session
        <strong>"{{ deleteDialog.session?.name }}"</strong>? This will delete
        both the event log and the packet capture. This action cannot be undone.
      </v-card-text>
      <v-card-actions>
        <v-spacer></v-spacer>
        <v-btn variant="text" @click="deleteDialog.visible = false"
          >Cancel</v-btn
        >
        <v-btn color="red-darken-1" variant="tonal" @click="confirmDelete"
          >Delete</v-btn
        >
      </v-card-actions>
    </v-card>
  </v-dialog>


</template>

<script lang="ts">
import { defineComponent, onMounted, onUnmounted, inject, reactive, ref } from "vue";
import { getSessions, renameSession, deleteSession } from "@/apicall";
import { sessions, activeSessionId, isNavDrawerOpen, fightSummary, condNameMap, liveFavoriteConditions, liveHiddenConditions, toggleLiveConditionPref } from "@/store";
import { Session } from "@/types";
import { EntityState, ActiveCondition } from "@/protocols";
import { computed } from "vue";

export default defineComponent({
  name: "SessionPanel",
  setup() {
    const appEvent = inject("appEvent");

    const searchQuery = ref("");

    const filteredSessions = computed(() => {
      if (!searchQuery.value) return sessions.value;
      const q = searchQuery.value.toLowerCase();
      return sessions.value.filter(s => {
        if (s.name.toLowerCase().includes(q)) return true;
        if (s.summary) {
          if (s.summary.players?.some(p => p.name.toLowerCase().includes(q) || (p.arcanaName && p.arcanaName.toLowerCase().includes(q)))) return true;
          if (s.summary.enemies?.some(e => e.name.toLowerCase().includes(q))) return true;
        }
        return false;
      });
    });

    const renameDialog = reactive({
      visible: false,
      session: null as Session | null,
      newName: "",
    });

    // ADDED STATE FOR DELETE DIALOG
    const deleteDialog = reactive({
      visible: false,
      session: null as Session | null,
    });


    // --- RESIZE LOGIC ---
    const splitRatio = ref(0.6); // Default 60% for sessions
    const isResizing = ref(false);
    const mainContainerRef = ref<HTMLElement | null>(null);
    const headerRef = ref<HTMLElement | null>(null);
    
    // Track initial values for smooth dragging (delta-based)
    let dragStartY = 0;
    let dragStartRatio = 0;

    const startResize = (e: MouseEvent) => {
      isResizing.value = true;
      dragStartY = e.clientY;
      dragStartRatio = splitRatio.value;
      
      window.addEventListener('mousemove', onResize);
      window.addEventListener('mouseup', stopResize);
      document.body.style.cursor = 'ns-resize';
      document.body.style.userSelect = 'none';
    };

    const stopResize = () => {
      isResizing.value = false;
      window.removeEventListener('mousemove', onResize);
      window.removeEventListener('mouseup', stopResize);
      document.body.style.cursor = '';
      document.body.style.userSelect = '';
      localStorage.setItem('session_panel_split', splitRatio.value.toString());
    };

    const onResize = (e: MouseEvent) => {
      if (!isResizing.value || !mainContainerRef.value || !headerRef.value) return;
      
      const containerRect = mainContainerRef.value.getBoundingClientRect();
      const headerRect = headerRef.value.getBoundingClientRect();
      
      // Calculate available height for the two panels (Total - Header)
      const availableHeight = containerRect.height - headerRect.height;
      if (availableHeight <= 0) return;

      // Calculate how much pixel distance we moved
      const deltaY = e.clientY - dragStartY;
      
      // Convert pixel delta to ratio delta
      const deltaRatio = deltaY / availableHeight;
      
      // Apply to initial ratio
      let newRatio = dragStartRatio + deltaRatio;
      
      // Clamp values
      if (newRatio < 0.2) newRatio = 0.2;
      if (newRatio > 0.8) newRatio = 0.8;
      
      splitRatio.value = newRatio;
    };

    onMounted(() => {
        const saved = localStorage.getItem('session_panel_split');
        if (saved) {
            const val = parseFloat(saved);
            if (!isNaN(val) && val >= 0.1 && val <= 0.9) {
                splitRatio.value = val;
            }
        }
    });
    // --------------------

    const fetchSessions = async () => {
      try {
        sessions.value = await getSessions();
      } catch (e) {
        console.error("Failed to fetch sessions:", e);
        alert("Could not fetch sessions from the backend.");
      }
    };

    const selectSession = (id: string | "live") => {
      activeSessionId.value = id;
    };

    const openRenameDialog = (session: Session) => {
      renameDialog.session = session;
      renameDialog.newName = session.name;
      renameDialog.visible = true;
    };

    // ADDED FUNCTION TO OPEN DELETE DIALOG
    const openDeleteDialog = (session: Session) => {
      deleteDialog.session = session;
      deleteDialog.visible = true;
    };

    const saveRename = async () => {
      if (!renameDialog.session || !renameDialog.newName.trim()) return;
      try {
        await renameSession(
          renameDialog.session.id,
          renameDialog.newName.trim()
        );
        await fetchSessions(); // Refresh the list
      } catch (e) {
        console.error("Failed to rename session:", e);
        alert("Could not rename session.");
      } finally {
        renameDialog.visible = false;
      }
    };

    // ADDED FUNCTION TO HANDLE CONFIRMED DELETION
    const confirmDelete = async () => {
      if (!deleteDialog.session) return;
      try {
        await deleteSession(deleteDialog.session.id);
        // If we deleted the currently viewed session, switch back to live
        if (activeSessionId.value === deleteDialog.session.id) {
          activeSessionId.value = "live";
        }
        await fetchSessions(); // Refresh the list
      } catch (e) {
        console.error("Failed to delete session:", e);
        alert("Could not delete session.");
      } finally {
        deleteDialog.visible = false;
      }
    };

    const formatSessionTime = (session: Session) => {
      const date = new Date(session.startTime * 1000);
      return date.toLocaleString();
    };

    onMounted(fetchSessions);

    appEvent.value.addEventListener("refresh-sessions", fetchSessions);


    const formatDuration = (seconds: number) => {
      if (!seconds) return "0s";
      const m = Math.floor(seconds / 60);
      const s = Math.floor(seconds % 60);
      if (m > 0) return `${m}m ${s}s`;
      return `${s}s`;
    };

    const formatDamage = (dmg: number) => {
      if (!dmg) return "0";
      if (dmg >= 1000000) return (dmg / 1000000).toFixed(1) + "M";
      if (dmg >= 1000) return (dmg / 1000).toFixed(1) + "K";
      return dmg.toFixed(0);
    };

    const getUniqueEnemies = (enemies: any[]) => {
      if (!enemies) return [];
      const seen = new Set<string>();
      const unique: any[] = [];
      for (const e of enemies) {
        if (!seen.has(e.name)) {
          seen.add(e.name);
          unique.push(e);
        }
      }
      return unique;
    };

    return {
      sessions,
      searchQuery,
      filteredSessions,
      activeSessionId,
      isNavDrawerOpen,
      selectSession,
      renameDialog,
      openRenameDialog,
      saveRename,
      // EXPOSE NEW ITEMS TO TEMPLATE
      deleteDialog,
      openDeleteDialog,
      confirmDelete,
      formatSessionTime,
      formatDuration,
      formatDamage,
      getUniqueEnemies
    };
  },
});
</script>
<style scoped>
.modern-drawer {
  background: rgba(23, 27, 36, 0.75) !important;
  backdrop-filter: blur(16px);
  border: 1px solid rgba(129, 138, 248, 0.15) !important;
  margin: 12px !important;
  height: calc(100% - 24px) !important;
  border-radius: 12px !important;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.4);
}

.session-item .session-actions {
  opacity: 0;
  transition: opacity 0.2s ease-in-out;
}

.session-item:hover .session-actions {
  opacity: 1;
}


</style>
