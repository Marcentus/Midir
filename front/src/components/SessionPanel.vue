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

      <!-- Scrollable Sessions List Container -->
      <!-- We use style binding to dynamic flex-grow based on splitRatio if activeSession is live, else full height -->
      <div 
        class="d-flex flex-column"
        :style="activeSessionId === 'live' 
            ? { flex: `0 0 ${splitRatio * 100}%`, height: `${splitRatio * 100}%`, minHeight: '100px', overflow: 'hidden' } 
            : { flex: '1 1 auto', overflow: 'hidden' }"
      >
          <v-list density="compact" nav class="fill-height pa-0 overflow-hidden">
            <v-virtual-scroll
              :items="sessions"
              item-height="40"
              height="100%"
            >
              <template v-slot="{ item }">
                <v-list-item
                  :key="item.id"
                  :title="item.name"
                  :subtitle="formatSessionTime(item)"
                  :active="activeSessionId === item.id"
                  @click="selectSession(item.id)"
                  class="session-item"
                >
                  <template v-slot:append>
                    <div class="session-actions">
                      <v-btn
                        variant="text"
                        icon="mdi-pencil"
                        size="x-small"
                        @click.stop="openRenameDialog(item)"
                      ></v-btn>
                      <v-btn
                        variant="text"
                        icon="mdi-delete"
                        color="red-lighten-2"
                        size="x-small"
                        @click.stop="openDeleteDialog(item)"
                      ></v-btn>
                    </div>
                  </template>
                </v-list-item>
              </template>
            </v-virtual-scroll>
          </v-list>
      </div>

      <!-- RESIZER BAR -->
      <div 
        v-if="activeSessionId === 'live'" 
        class="resizer"
        @mousedown.prevent="startResize"
      ></div>

      <!-- Entities in Area Section -->
      <!-- This fills the remaining space -->
      <div 
        v-if="activeSessionId === 'live'"
        class="d-flex flex-column"
        style="flex: 1 1 0px; min-height: 100px; overflow: hidden;"
      >
        <v-divider></v-divider>
        <v-list-item>
          <v-list-item-title class="text-caption font-weight-bold text-uppercase text-medium-emphasis">
            Players in Area ({{ currentEntities.length }})
          </v-list-item-title>
        </v-list-item>

        <v-list density="compact" nav class="flex-grow-1 pa-0 overflow-y-auto">
          <v-virtual-scroll
            :items="currentEntities"
            item-height="32"
            height="100%"
          >
            <template v-slot="{ item }">
              <v-list-item :key="item.id" class="px-2 py-0" min-height="32" @click="selectEntity(item)" link>
                <div class="d-flex align-center justify-space-between w-100" style="gap: 8px;">
                    <div class="d-flex flex-column" style="flex: 1; min-width: 0;">
                      <div class="d-flex align-center justify-space-between w-100">
                        <div class="text-body-2 text-truncate">{{ item.name }}</div>
                        <div v-if="item.conditions && Object.keys(item.conditions).length > 0" class="d-flex align-center justify-end overflow-hidden" style="gap: 2px;">
                          <v-img
                              v-for="condId in getPreviewConditions(item)"
                              :key="condId"
                              :src="getConditionIcon(condId)"
                              width="16"
                              height="16"
                              class="condition-icon"
                              :title="getConditionName(condId)"
                            >
                              <template v-slot:placeholder>
                                <v-sheet color="grey-darken-2" width="100%" height="100%"></v-sheet>
                              </template>
                            </v-img>
                        </div>
                      </div>
                      
                      <!-- HP BAR -->
                      <div v-if="item.maxHp > 0" class="mt-1">
                         <v-progress-linear
                            :model-value="(item.currentHp / item.maxHp) * 100"
                            color="red-darken-2"
                            height="14"
                            rounded
                         >
                            <template v-slot:default>
                                <div class="text-caption font-weight-bold textual-shadow" style="font-size: 10px; color: white;">
                                    {{ Math.ceil(item.currentHp).toLocaleString() }} / {{ Math.ceil(item.maxHp).toLocaleString() }}
                                </div>
                            </template>
                         </v-progress-linear>
                      </div>
                    </div>
                </div>
              </v-list-item>
            </template>
          </v-virtual-scroll>
        </v-list>
      </div>

    <!-- Entity Details Dialog -->
    <v-dialog v-model="detailsOpen" max-width="600">
      <v-card>
        <v-card-title class="text-h6">
          {{ selectedEntity?.name }}
        </v-card-title>
        <v-card-subtitle>
          ID: {{ selectedEntity?.id }} | Race: {{ selectedEntity?.raceId }}
        </v-card-subtitle>

        <v-card-text>
          <div class="text-subtitle-1 font-weight-bold mb-2">Active Conditions</div>
          <v-list density="compact" v-if="selectedEntity?.conditions && Object.keys(selectedEntity.conditions).length > 0" class="overflow-y-auto" style="max-height: 400px">
            <v-list-item v-for="id in getAllSortedConditions(selectedEntity)" :key="id" :value="id" class="align-start py-2">
              <template v-slot:prepend>
                 <div class="d-flex align-center mr-2 align-self-start mt-1">
                    <v-btn icon density="compact" variant="text" size="small" 
                        :color="liveFavoriteConditions.has(id) ? 'amber' : 'grey'" 
                        @click.stop="toggleLiveConditionPref(id, 'fav')">
                        <v-icon>{{ liveFavoriteConditions.has(id) ? 'mdi-star' : 'mdi-star-outline' }}</v-icon>
                    </v-btn>
                    <v-btn icon density="compact" variant="text" size="small" 
                        :color="liveHiddenConditions.has(id) ? 'red' : 'grey'" 
                        @click.stop="toggleLiveConditionPref(id, 'hide')">
                        <v-icon>{{ liveHiddenConditions.has(id) ? 'mdi-eye-off' : 'mdi-eye' }}</v-icon>
                    </v-btn>


                 </div>
                <v-img
                  :src="getConditionIcon(id)"
                  width="24"
                  height="24"
                  class="mr-3 align-self-start mt-1"
                  :style="{ opacity: liveHiddenConditions.has(id) ? 0.5 : 1, flex: 'none' }"
                >
                  <template v-slot:placeholder>
                    <v-sheet color="grey-darken-2" width="100%" height="100%"></v-sheet>
                  </template>
                </v-img>
              </template>
              <v-list-item-title :class="{'text-decoration-line-through text-grey': liveHiddenConditions.has(id)}">
                  {{ getConditionName(id) }}
              </v-list-item-title>
              <v-list-item-subtitle class="text-caption">
                Start: {{ new Date(selectedEntity.conditions[id].start * 1000).toLocaleTimeString() }}
                <span v-if="selectedEntity.conditions[id].attackerId"> | Source: {{ selectedEntity.conditions[id].attackerId }}</span>
                <span v-if="getConditionTimeRemaining(selectedEntity.conditions[id])" class="text-warning font-weight-bold ml-1">
                  | {{ getConditionTimeRemaining(selectedEntity.conditions[id]) }}
                </span>
              </v-list-item-subtitle>
              <div v-if="selectedEntity.conditions[id].metaData" class="mt-1">
                 <div 
                    v-for="(metaPart, idx) in selectedEntity.conditions[id].metaData.split(';').map(s => s.trim()).filter(s => s).sort()" 
                    :key="idx"
                    class="text-caption text-medium-emphasis" 
                    style="white-space: normal; line-height: 1.2;">
                    • {{ metaPart }}
                 </div>
              </div>
            </v-list-item>
          </v-list>
          <div v-else class="text-caption text-grey">No active conditions</div>
        </v-card-text>

        <v-card-actions>
          <v-spacer></v-spacer>
          <v-btn color="primary" text @click="detailsOpen = false">Close</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>
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

    const detailsOpen = ref(false);
    const selectedEntityId = ref<string | null>(null);

    const selectEntity = (entity: EntityState) => {
      selectedEntityId.value = entity.id;
      detailsOpen.value = true;
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

    const getConditionTimeRemaining = (condition: ActiveCondition) => {
        if (!condition.disableAt || condition.disableAt === 0) return null;
        
        const now = Math.floor(Date.now() / 1000);
        const diff = condition.disableAt - now;

        if (diff <= 0) return null;

        if (diff > 3600) {
            return Math.floor(diff / 3600) + 'h';
        } else if (diff > 60) {
            return Math.floor(diff / 60) + 'm';
        } else {
            return diff + 's';
        }
    };


    const getConditionIcon = (id: number) => {
        if (condNameMap && condNameMap.value[id]) {
            return condNameMap.value[id].iconUrl;
        }
        return ""; 
    };


    return {
      sessions,
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
      currentEntities: computed(() => fightSummary.currentEntities || []),
     // Helper to format duration
      getConditionTimeRemaining,
      getConditionIcon,
getConditionName: (id: number) => condNameMap.value[id]?.name || `Unknown Status ${id}`,
      // NEW: Entity Details
      detailsOpen,
      selectedEntity: computed(() => {
        if (!selectedEntityId.value) return null;
        return (fightSummary.currentEntities || []).find(e => e.id === selectedEntityId.value) || null;
      }),
      selectEntity,
      // LIVE CONDITION PREFS
      liveFavoriteConditions,
      liveHiddenConditions,
      toggleLiveConditionPref,
      getPreviewConditions: (entity: EntityState) => {
        if (!entity.conditions) return [];
        const allIds = Object.keys(entity.conditions).map(Number);
        // ONLY show favorites in the preview list
        const favorites = allIds.filter(id => liveFavoriteConditions.has(id));
        favorites.sort((a, b) => a - b);
        return favorites.slice(0, 20); 
      },
      getAllSortedConditions: (entity: EntityState) => {
        if (!entity.conditions) return [];
        const allIds = Object.keys(entity.conditions).map(Number);
        // Sort: Favorites first, then Normal, then Hidden
        allIds.sort((a, b) => {
           const aFav = liveFavoriteConditions.has(a);
           const bFav = liveFavoriteConditions.has(b);
           const aHide = liveHiddenConditions.has(a);
           const bHide = liveHiddenConditions.has(b);
           
           // Favorites always on top
           if (aFav && !bFav) return -1;
           if (!aFav && bFav) return 1;
           
           // Hidden always at bottom
           if (aHide && !bHide) return 1;
           if (!aHide && bHide) return -1;
           
           return a - b;
        });
        return allIds;
      },
      // RESIZE
      splitRatio,
      startResize,
      mainContainerRef,
      headerRef,
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

.condition-icon {
  border: 1px solid #616161; /* Thin dark gray */
  border-radius: 2px;
  /* Force fixed, non-stretching size */
  width: 16px !important;
  height: 16px !important;
  flex: 0 0 16px;
  display: block;
}

.resizer {
  height: 8px;
  background: rgba(var(--v-theme-on-surface), 0.1);
  cursor: ns-resize;
  transition: background 0.2s;
  flex: 0 0 8px; /* Fixed height */
  position: relative;
  z-index: 10;
}
.resizer:hover, .resizer:active {
  background: rgba(var(--v-theme-primary), 0.5);
}
.resizer::after {
   /* Handle indicator */
   content: "";
   position: absolute;
   top: 50%;
   left: 50%;
   transform: translate(-50%, -50%);
   width: 40px;
   height: 2px;
   background: rgba(var(--v-theme-on-surface), 0.3);
   border-radius: 2px;
}
</style>
