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

        <div class="px-3 pb-2 pt-2">
          <v-btn block size="small" color="primary" variant="tonal" prepend-icon="mdi-folder-zip" @click="openBatchExport">Batch Export HTML Reports</v-btn>
        </div>
        <div class="px-3 pb-2">
          <v-btn block size="small" variant="tonal" prepend-icon="mdi-delete-sweep" @click="openBatchDelete">Batch Delete Sessions</v-btn>
        </div>
        <v-divider></v-divider>

        <!-- ADDED MIGRATE ALL BUTTON WITH PROGRESS BAR -->
        <div v-if="sessions.some(s => !s.summary && s.id !== 'live') || (isMigratingAll && totalToMigrate > 0)" class="px-3 pb-2 mt-2">
          <v-btn block color="primary" variant="tonal" size="small" @click="migrateAll" :loading="isMigratingAll && totalToMigrate === 0">
            <v-icon start>mdi-autorenew</v-icon>
            Migrate All Missing
          </v-btn>
          
          <div v-if="isMigratingAll && totalToMigrate > 0" class="mt-2">
            <div class="text-caption text-grey d-flex justify-space-between mb-1">
              <span>Migrating...</span>
              <span>{{ migrationProgress }} / {{ totalToMigrate }}</span>
            </div>
            <v-progress-linear
              :model-value="(migrationProgress / totalToMigrate) * 100"
              color="primary"
              height="6"
              rounded
            ></v-progress-linear>
          </div>
        </div>

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
                  class="session-item py-2 align-start"
                  :class="{ 'can-migrate': !item.summary && item.id !== 'live' }"
                  style="height: 115px;"
                >
                  <div class="d-flex flex-column w-100 justify-start">
                    <!-- Top Row: Name | Datetime -->
                    <div class="d-flex align-center justify-space-between w-100 mb-2">
                      <div class="text-subtitle-2 font-weight-bold text-truncate" style="flex: 1; min-width: 0;">{{ item.name }}</div>
                      
                      <!-- Right: Datetime -->
                      <div class="text-caption text-grey text-right" style="flex: 1;">
                        <v-tooltip :text="getExactSessionTime(item)" location="top">
                          <template v-slot:activator="{ props }">
                            <span v-bind="props" style="cursor: help;">{{ formatSessionTime(item) }}</span>
                          </template>
                        </v-tooltip>
                        <template v-if="item.summary && item.summary.duration"> &bull; {{ formatDuration(item.summary.duration) }}</template>
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
                    <div class="session-actions align-self-start">
                      <v-menu location="bottom end">
                        <template v-slot:activator="{ props }">
                          <v-btn
                            icon="mdi-dots-vertical"
                            variant="text"
                            size="x-small"
                            v-bind="props"
                            @click.stop
                          ></v-btn>
                        </template>
                        <v-list density="compact" nav>
                          <v-list-item 
                            v-if="!item.summary && item.id !== 'live'" 
                            prepend-icon="mdi-autorenew" 
                            title="Generate Summary" 
                            base-color="blue-lighten-2"
                            @click="migrateSingle(item)"
                          ></v-list-item>
                          
                          <v-list-item v-if="item.id === activeSessionId && item.id !== 'live'" prepend-icon="mdi-export-variant" title="Export HTML report" @click="openExport(item)"></v-list-item>
                          <v-list-item prepend-icon="mdi-folder-open" title="Show session file in Explorer" @click="showFile(item)"></v-list-item>
                          <v-list-item 
                            prepend-icon="mdi-pencil" 
                            title="Rename" 
                            @click="openRenameDialog(item)"
                          ></v-list-item>
                          
                          <v-list-item 
                            prepend-icon="mdi-delete" 
                            title="Delete" 
                            base-color="red-lighten-2"
                            @click="openDeleteDialog(item)"
                          ></v-list-item>
                        </v-list>
                      </v-menu>
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
  <v-dialog v-model="exportDialog.visible" max-width="480px"><v-card>
    <v-card-title>Export offline HTML report</v-card-title><v-card-text>
      <v-select label="Include players" v-model="exportDialog.scope" :items="exportOptions" item-title="title" item-value="value" density="comfortable"></v-select>
      <v-checkbox v-model="exportDialog.targetOnly" label="Only damage to selected Combat Target" density="compact" hide-details></v-checkbox><p class="text-caption" v-if="exportDialog.targetOnly && !selectedTargetId">Select a Combat Target in the damage panel before exporting.</p><p class="text-caption">Single-file report with interactive scatter points, skill details and no visible application branding. Individual reports omit other players.</p>
    </v-card-text><v-card-actions><v-spacer/><v-btn variant="text" @click="exportDialog.visible = false">Cancel</v-btn><v-btn color="primary" @click="saveExport">Save HTML</v-btn></v-card-actions></v-card>
  </v-dialog>
  <v-dialog v-model="batchDialog.visible" max-width="600px"><v-card>
    <v-card-title>Batch Export HTML Reports</v-card-title>
    <v-card-text>
      <p class="text-body-2 mb-3">Choose saved sessions to export. Each report will be a separate interactive HTML file in one ZIP. The currently loaded session will not change.</p>
      <v-select v-model="batchDialog.ids" label="Sessions" :items="batchChoices" item-title="title" item-value="value" multiple chips closable-chips density="comfortable"></v-select>
      <v-checkbox v-model="batchDialog.onlySelectedPlayer" label="Export one player from every report" density="compact"></v-checkbox>
      <v-text-field v-if="batchDialog.onlySelectedPlayer" v-model="batchDialog.playerName" label="Exact character name" density="compact" hint="Sessions without that character will be reported as skipped." persistent-hint></v-text-field>
      <p class="text-caption">Batch reports contain all combat targets for their session. Use single-session export for a specific selected Combat Target.</p>
      <v-progress-linear v-if="batchDialog.running" indeterminate class="mt-2"></v-progress-linear>
      <p v-if="batchDialog.progress" class="text-caption">{{ batchDialog.progress }}</p>
    </v-card-text>
    <v-card-actions><v-spacer/><v-btn variant="text" :disabled="batchDialog.running" @click="batchDialog.visible=false">Cancel</v-btn><v-btn color="primary" :loading="batchDialog.running" :disabled="!batchDialog.ids.length || batchDialog.running" @click="saveBatchExport">Download ZIP</v-btn></v-card-actions>
  </v-card></v-dialog>
  <v-dialog v-model="batchDelete.visible" max-width="600px"><v-card>
    <v-card-title>Batch Delete Saved Sessions</v-card-title>
    <v-card-text>
      <v-select v-model="batchDelete.ids" label="Select sessions to delete" :items="batchChoices" item-title="title" item-value="value" multiple chips closable-chips density="comfortable"></v-select>
      <p class="text-body-2">Permanently delete {{ batchDelete.ids.length }} saved session(s), including matching packet captures? This cannot be undone.</p>
      <p v-if="batchDelete.progress" class="text-caption">{{ batchDelete.progress }}</p>
    </v-card-text>
    <v-card-actions><v-spacer/><v-btn variant="text" :disabled="batchDelete.running" @click="batchDelete.visible=false">Cancel</v-btn>
    <v-btn color="red" :disabled="!batchDelete.ids.length || batchDelete.running" :loading="batchDelete.running" @click="confirmBatchDelete">Delete selected</v-btn></v-card-actions>
  </v-card></v-dialog>
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
import { getSessions, getSessionSummary, renameSession, deleteSession, revealSessionFile, migrateSession, migrateAllSessions } from "@/apicall";
import { sessions, activeSessionId, isNavDrawerOpen, fightSummary, condNameMap, liveFavoriteConditions, liveHiddenConditions, toggleLiveConditionPref } from "@/store";
import { Session } from "@/types";
import { EntityState, ActiveCondition } from "@/protocols";
import { computed } from "vue";
import { skillNameMap, selectedTargetId } from "@/store";
import { exportPortableReport, buildPortableReportHTML } from "@/utils/portableReport";
import { makeReportZip } from "@/utils/batchZip";


export default defineComponent({
  name: "SessionPanel",
  setup() {
    const appEvent = inject("appEvent");

    const searchQuery = ref("");
    const isMigratingAll = ref(false);
    const migrationProgress = ref(0);
    const totalToMigrate = ref(0);

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

    const exportDialog = reactive({visible:false, scope:'all', targetOnly:false, session:null as Session | null});
    const exportOptions = computed(() => [{title:'Whole party',value:'all'}, ...Object.values(fightSummary.players || {}).filter(p => !/^(?:Unknown Caster|Misc\. Anonymous Healing)$/i.test(p.name.trim())).map(p=>({title:p.name,value:p.id}))]);
    const openExport = (session:Session) => {exportDialog.session=session;exportDialog.scope='all';exportDialog.targetOnly=Boolean(selectedTargetId.value);exportDialog.visible=true;};
    const saveExport = () => {
      try {if(!exportDialog.session || activeSessionId.value !== exportDialog.session.id)throw new Error('Select and load the saved session first.');
        const nameMap:Record<number,string>={};for(const [id,value] of Object.entries(skillNameMap.value))nameMap[Number(id)]=value.name;
        exportPortableReport(fightSummary, exportDialog.session.name, exportDialog.scope, nameMap, exportDialog.targetOnly ? selectedTargetId.value : '');
        exportDialog.visible=false;
      }catch(e){alert(e instanceof Error?e.message:'Unable to export report.');}
    };
    const batchDialog = reactive({ visible:false, ids:[] as string[], running:false, progress:'', onlySelectedPlayer:false, playerName:'' });
    const batchChoices = computed(() => sessions.value.filter(s=>s.id !== 'live').map(s=>({title:s.name + ' ('+new Date(s.startTime*1000).toLocaleString()+')',value:s.id})));
    const openBatchExport = () => {batchDialog.ids=[];batchDialog.progress='';batchDialog.visible=true;};
    const saveBatchExport = async () => {
      if (batchDialog.running || !batchDialog.ids.length) return;
      batchDialog.running=true; batchDialog.progress='';
      const skipped: string[]=[]; const reports: {name:string;html:string}[]=[];
      const nameMap:Record<number,string>={};
      for(const [id,value] of Object.entries(skillNameMap.value)) nameMap[Number(id)]=value.name;
      try {
        for(const id of batchDialog.ids) {
          const session=sessions.value.find(s=>s.id===id);
          if(!session) {skipped.push(id+': missing session');continue;}
          batchDialog.progress=`Generating ${reports.length + skipped.length + 1} of ${batchDialog.ids.length}: ${session.name}`;
          try {
            const summary=await getSessionSummary(id);
            const player=batchDialog.onlySelectedPlayer ? Object.values(summary.players||{}).find(p=>p.name.trim().toLowerCase()===batchDialog.playerName.trim().toLowerCase()) : null;
            if(batchDialog.onlySelectedPlayer&&!player)throw new Error('character not found');
            const html=buildPortableReportHTML(summary,session.name,player?.id || 'all',nameMap);
            const safe=session.name.replace(/[\\/:*?"<>|\x00-\x1f]/g,'_').slice(0,75) || 'session';
            reports.push({name:`${safe}-${id.replace(/[^a-zA-Z0-9_-]/g,'_')}.html`,html});
          }catch(error){skipped.push(`${session.name}: ${error instanceof Error?error.message:'cannot read saved session'}`);}
        }
        if(!reports.length)throw new Error('No reports generated. '+skipped.join('; '));
        const blob=makeReportZip(reports);const url=URL.createObjectURL(blob);
        const link=document.createElement('a');link.href=url;link.download='midir-html-reports.zip';document.body.append(link);link.click();link.remove();setTimeout(()=>URL.revokeObjectURL(url),30000);
        batchDialog.visible=false;
        if(skipped.length)alert(`${reports.length} reports exported; ${skipped.length} skipped:\n${skipped.join('\n')}`);
      } catch(error) {alert(error instanceof Error?error.message:'Batch export failed');}
      finally{batchDialog.running=false;batchDialog.progress='';}
    };
    const batchDelete = reactive({visible:false, ids:[] as string[], running:false, progress:''});
    const openBatchDelete = () => {batchDelete.ids=[];batchDelete.progress='';batchDelete.visible=true;};
    const showFile = async (session:Session) => {
      try {await revealSessionFile(session.id);} catch(e) {alert(e instanceof Error?e.message:'Unable to locate saved file.');}
    };
    const confirmBatchDelete = async () => {
      if(batchDelete.running || !batchDelete.ids.length) return;
      batchDelete.running=true;
      const failed:string[]=[];
      const selected=[...batchDelete.ids];
      for(let i=0;i<selected.length;i++){
        const id=selected[i];batchDelete.progress=`Deleting ${i+1}/${selected.length}`;
        try {await deleteSession(id);if(activeSessionId.value===id)activeSessionId.value='live';}
        catch(e){failed.push(id+': '+(e instanceof Error?e.message:String(e)));}
      }
      await fetchSessions();
      batchDelete.running=false;batchDelete.visible=false;
      if(failed.length)alert(`${selected.length-failed.length} deleted; ${failed.length} failed:\n${failed.join('\n')}`);
    };
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

    const migrateSingle = async (session: Session) => {
      try {
        await migrateSession(session.id);
        await fetchSessions();
      } catch (e) {
        console.error("Failed to migrate session:", e);
        alert("Could not migrate session.");
      }
    };

    const migrateAll = async () => {
      const toMigrate = sessions.value.filter(s => !s.summary && s.id !== 'live');
      if (toMigrate.length === 0) return;

      isMigratingAll.value = true;
      totalToMigrate.value = toMigrate.length;
      migrationProgress.value = 0;

      try {
        for (const session of toMigrate) {
          try {
            await migrateSession(session.id);
            migrationProgress.value++;
            // Refresh sessions list to show progress visually if needed, 
            // but fetching the whole list might be heavy. 
            // For now, we'll just update the local count.
          } catch (err) {
            console.error(`Failed to migrate session ${session.id}:`, err);
          }
        }
        await fetchSessions(); // Final refresh
      } catch (e) {
        console.error("Failed to migrate all sessions:", e);
        alert("Could not complete migration of all sessions.");
      } finally {
        isMigratingAll.value = false;
        totalToMigrate.value = 0;
        migrationProgress.value = 0;
      }
    };

    const formatSessionTime = (session: Session) => {
      const startTime = session.startTime * 1000;
      const now = Date.now();
      const diffSeconds = Math.floor((now - startTime) / 1000);
      const diff = Math.max(0, diffSeconds);
      
      const diffMinutes = Math.floor(diff / 60);
      const diffHours = Math.floor(diffMinutes / 60);
      const diffDays = Math.floor(diffHours / 24);
      const diffMonths = Math.floor(diffDays / 30);

      if (diffMinutes <= 60) {
        return `${diffMinutes}min ago`;
      } else if (diffHours <= 24) {
        return `${diffHours}h ago`;
      } else if (diffDays < 30) {
        return `${diffDays}d ago`;
      } else {
        return `${diffMonths} month ago`;
      }
    };

    const getExactSessionTime = (session: Session) => {
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
      selectedTargetId,
      exportDialog, exportOptions, openExport, saveExport, batchDialog, batchChoices, openBatchExport, saveBatchExport,
      searchQuery,
      batchDelete, openBatchDelete, confirmBatchDelete, showFile,
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
      getExactSessionTime,
      formatDuration,
      formatDamage,
      getUniqueEnemies,
      migrateSingle,
      migrateAll,
      isMigratingAll,
      migrationProgress,
      totalToMigrate
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

.session-item :deep(.v-list-item__content),
.session-item :deep(.v-list-item__prepend),
.session-item :deep(.v-list-item__append) {
  align-self: flex-start !important;
  margin-top: 0 !important;
}

.session-item.can-migrate {
  background: rgba(30, 58, 138, 0.15) !important;
  border-left: 3px solid #60a5fa !important;
}

.session-item.can-migrate:hover {
  background: rgba(30, 58, 138, 0.25) !important;
}

</style>
