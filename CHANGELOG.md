# Midir — Changelog

A concise record of **what changed and why**. Features are grouped by purpose rather than by individual hotfix. Source references appear as numbered notes at the end. For current packet layouts, skill-observation rules, and HTML data contracts, see [Skill and Export Reference](SKILL-AND-EXPORT-REFERENCE.md).

## Build and dependency maintenance

- **Capture-mode safety:** CLI capture now honors the saved Npcap promiscuous-mode opt-in instead of forcing it on; absent preferences default off, and offline playback stays non-promiscuous. UI capture, restart, and autodetection retain their existing setting propagation. Added startup regression coverage. [^22]

- **Frontend dependency maintenance:** Raise Vue and Vite to patched compatible release ranges; require patched transitive dependency families identified by the npm audit, and approve only the reviewed `esbuild` installation script version. Build scripts synchronize the frontend lockfile during installation. Replace two ineffective dynamic imports with static imports, keeping loading state and condition-combination behavior unchanged. [^21]

## Combat analysis and player progress

- **Arcana appearance consistency:** Respect customized Appearance Settings colors in multi-Arcana gradients and encounter views. Use each recognized skill’s Arcana color for its damage bar, while shared skills inherit the player’s scoped solid color or mixed gradient in the application and HTML report; keep damage lengths unchanged. [^20]

- **Multi-Arcana session display:** Show all observed Arcana icons and blended Arcana colors in session-wide player bars; encounter selections use their scoped single Arcana when identifiable. Preserve character-name colors when that display mode is chosen. Gradients denote identities, not damage or playtime proportions. [^20]

- **Damage analysis:** Added interactive damage scatter, boss HP timelines, per-player skill breakdowns, and collapsible, reorderable player panels to make damage patterns easier to compare. [^1]
- **Ghost Runner:** Added a reference pace based on the median of nonzero 15-second damage buckets, shown on the graph and optionally as comparison fills in Damage Dealt. It represents a typical productive stretch, not an ideal rotation. [^1]
- **Death markers:** Added timeline markers and approximate damage/skill context around deaths. Values reconstructed from saved samples are estimates, not exact death-time snapshots. [^1]
- **Personal Best Progress:** Added per-skill comparisons against previous sessions for the same player and target, with a 5–200-session lookback and an optional read-only history folder. Comparisons are not an overall DPS ranking. [^1] [^13]
- **Healing Done:** Added healing summaries, largest-heal display, and defensive-effect estimates, including recognized Harmonic Saint caster handling and Veil of Life condition `999`. Suspicious values are screened; caster, skill, and shield attribution can remain uncertain. [^2]

- **Encounter-scoped Arcana:** Detect distinct Arcanas used during a session from observed skill evidence; resolve each selectable encounter independently. Full-session multi-Arcana views no longer retain the first Arcana label, and uncertain scopes stay unassigned rather than borrowing a different encounter’s identity. [^20]

- **Multi-Arcana presentation:** The Arcana registry is loaded through Vite raw JSON with an explicit TypeScript shape, avoiding the `resolveJsonModule` build error. Mixed player bars and portable-report headers now blend observed Arcana colors smoothly, including three or more Arcanas, in first-observed order when timestamps exist. Individual encounter colors and icons remain scoped to observed skills; gradients represent identity, not proportional damage. [^20]

## Recording and saved sessions

- **More reliable recordings:** Improved buffered NDJSON writes, capture boundaries, session counters, large-record reading (up to 32 MiB per line), and temporary-file publication to reduce incomplete saved sessions. [^3]
- **Session management:** Added opening a recording in the file manager, batch deletion with confirmation, and batch HTML export without leaking target selection between sessions. [^3]
- **Skill and hit evidence:** Recorded explicit local uses and remote observations with millisecond timestamps, plus damage-hit timing and target IDs where available. Older session records remain readable; multi-hit packets and effect cues do **not** automatically represent completed casts. [^4]
- **Pre-combat skill buffer:** Added an optional 0–30-second window (off by default) that carries up to 512 recent skill-use events into a new combat encounter, capturing preparation without continuously recording inactive combat. [^14]
- **Pet and puppet damage:** Attributed recognized pets and puppets to their owners so their damage contributes to the appropriate player totals; pet damage uses a dedicated skill category. [^15]
- **Enemy attack separation:** Named recognized boss attacks and exact-entity-matched minor-enemy sources (including numerically named Glenn Gwyllgi), with corroborated Race ID fallbacks; kept enemy attacks outside player DPS and Companion damage records. [^16]
- **Removed temporary diagnostics:** Retired the experimental healing diagnostic recorder and its UI. Historical diagnostic recordings remain independent files. [^2]

## Portable HTML export and Companion data

- **Self-contained reports:** Added portable HTML reports with damage scatter, boss HP search, Arcana-colored player panels, drag and keyboard reordering, and embedded Companion-compatible data. The separate JSON download control was retired; its data remains in the HTML. [^5]
- **Target-aware reporting:** Added boss-only and all-target scopes using recorded entity IDs, with separate damage and HP calculations. Gate 1 exports preserve both Vertrag forms’ raw HP histories as well as the continuous logical timeline. Missing historical target details cannot be reconstructed. [^5] [^6]
- **Skill evidence export:** Added distinct streams for explicit skill uses, damage hits, and typed skill observations (`0x6988`, `0x7926`, `0x9093`, `0x9095`), allowing the Companion to correlate evidence without inventing cast counts. [^4]
- **Clearer report controls:** Replaced duplicate encounter entries with an encounter selector and a separate target-scope selector; labeled outcomes **Cleared** or **Attempted**; removed the visible Skill Use Timeline and redundant boss chart checkboxes while retaining the underlying data. [^5]
- **Encounter compatibility:** Embedded logical boss encounters, phase HP, death/revival evidence, entity provenance, per-encounter Arcana evidence, mixed-color/multi-icon Arcana summaries, and conservative clear evidence without removing older Companion fields. [^5] [^20]

## Dungeon encounters and target grouping

- **Bri Leith — Gate 1:** Associated Vertrag's two forms (`7600` → `7601`) within each attempt, displaying a continuous 100%→50%→0 HP progression. The combined Vertrag row and each form are selectable; Ancient Giants and other mechanics stay outside boss-only HP. [^6]
- **Bri Leith — Gate 3:** Grouped Shards of Solace using observed HP families and timing near boss thresholds, keeping ambiguous entities outside named wave groups. [^7]
- **Glenn Bearna:** Separated Unfinished Reverie (`7585`, 100%→10%) from the shared 100%→0% boss encounter comprising Reverie (`7586`, 100%→60%) and The Winter Tyrant (`7587`, 60%→0%). Kept all-boss and individual views. Cailleach (`7575`, 100%→90%) and her entrance barriers belong to the opening encounter; Icicle Sprites belong with minor enemies. [^8]
- **Crom Bas:** Grouped Irusan (`193364`) with the recorded Glas variant—Completed (`193358`) or Giga (`193359`). Merch Diafol and all colored Disciples are selectable mechanics, excluded from boss HP. [^9]
- **High-tier dungeon grouping:** Kept Crom Bas Abyss Folamh (`193810`) and Bri Leith Gate 4 Midir the Lost (`7615`) as boss-only, attempt-bounded selections. Grouped known associated enemies by name and approximate run segment under provisional Minor Enemies folders, without asserting HP phases or mechanics. Additional possible boss forms remain unassigned. [^7] [^19]
- **Unified dungeon runs:** Combined each Crom Bas or Glenn Bearna run's bosses, opening encounters (where present), and minor enemies into a numbered run folder while retaining individual selections. Run assignment follows recorded entities and timing rather than assuming every similarly named enemy belongs to one attempt. Shared-race Phantasm Wraiths no longer create false Crom Bas combat segments. [^10]
- **Fahmes Ruin:** Organized independently enterable stages into selectable stage and sequential level views. Each level aggregates its recorded enemies, including overlapping waves, without imposing an invented boss or stage-clear packet. Complete 150-enemy stages retain all five level selections even when delayed spawns defeat the initial timing heuristic. [^18]
- **Alby Shineseeker:** Organized enemies into five waves, from rats and bats through Cichol's Followers, with individual-wave and complete-run views. The final-wave status reflects the defeat of recorded Followers, not a separate dungeon-clear packet. [^11]
- **Rabbie Phantasm:** Grouped each recorded run with selectable Queen boss and minor-enemy sections. The four named Succubi appearing with the Queen form one boss-mechanics selection, separate from boss-only damage and HP; earlier colored Succubi remain minor enemies. Confirmed Normal and Temptation's Path Queens are recognized separately; other Queen race IDs require supporting dungeon context and remain unverified. Queen-only runs omit the named-Succubi mechanics group; Fiends and Wraiths summoned in the Normal Queen room remain selectable under Boss-room Enemies. [^17]

## Build and project maintenance

- **Frontend typing:** Kept optional paired-boss Race IDs typed consistently when merging established encounter definitions with provisional Succubus Queen variants. [^19]
- **Build safeguards:** Added Node-version checks and automated build validation to reduce stale or misleading build outputs. [^12]
- **Documentation:** Consolidated release notes, routine grouping updates, and overlapping skill/export notes. Detailed protocol findings now belong in the dedicated technical reference rather than the changelog.

## Source references

References identify relevant files in the Midir source, not specific commits. The historical feature descriptions were also reconciled with the earlier Midir-41 changelog; older `RELEASE-*.md` and `FEATURE-CHANGELOG.md` notes are not treated as independently current documentation.

[^1]: Combat UI, graph, and player comparisons: [`DamageMeterView.vue`](front/src/components/DamageMeterView.vue), [`DamageGraph.vue`](front/src/components/DamageGraph.vue), [`applyDamageBySkill.vue`](front/src/components/applyDamageBySkill.vue). Earlier development notes also document Ghost Runner, death markers, and Personal Best Progress.
[^2]: Healing and defensive-effect handling: [`healingPacket.go`](packet/healingPacket.go), [`aggregator.go`](cmd/dilmeterapi/aggregator.go), [`applyHealingBySkill.vue`](front/src/components/applyHealingBySkill.vue).
[^3]: Saved sessions: [`sessionManager.go`](cmd/dilmeterapi/sessionManager.go), [`httpHandlerSession.go`](cmd/dilmeterapi/httpHandlerSession.go), [`SessionPanel.vue`](front/src/components/SessionPanel.vue).
[^4]: Skill recording, timestamps, and observations: [`eventPublisher.go`](cmd/dilmeterapi/eventPublisher.go), [`event.go`](cmd/dilmeterapi/event.go), [`portableReport.ts`](front/src/utils/portableReport.ts).
[^5]: HTML reporting and Companion metadata: [`portableReport.ts`](front/src/utils/portableReport.ts), [`companionExport.ts`](front/src/utils/companionExport.ts).
[^6]: Gate 1: [`gate1Encounter.ts`](front/src/utils/gate1Encounter.ts), [`targetGrouping.ts`](front/src/utils/targetGrouping.ts).
[^7]: Earlier encounter recognition: [`reportEncounters.ts`](front/src/utils/reportEncounters.ts), [`targetGrouping.ts`](front/src/utils/targetGrouping.ts).
[^8]: Glenn Bearna: [`glennEncounter.ts`](front/src/utils/glennEncounter.ts), [`glennOpening.ts`](front/src/utils/glennOpening.ts), [`glennMinorEnemies.ts`](front/src/utils/glennMinorEnemies.ts).
[^9]: Crom Bas: [`cromEncounter.ts`](front/src/utils/cromEncounter.ts), [`mechanicGroups.ts`](front/src/utils/mechanicGroups.ts).
[^10]: Run association: [`dungeonRunFolders.ts`](front/src/utils/dungeonRunFolders.ts), [`instanceMinorGrouping.ts`](front/src/utils/instanceMinorGrouping.ts), [`targetGrouping.ts`](front/src/utils/targetGrouping.ts).
[^11]: Alby Shineseeker: [`albyShineseeker.ts`](front/src/utils/albyShineseeker.ts).
[^12]: Build automation: [`.github/workflows/build-validation.yml`](.github/workflows/build-validation.yml), [`scripts/check-node-security.cjs`](scripts/check-node-security.cjs).
[^13]: Personal Best history controls: [`SettingsView.vue`](front/src/components/SettingsView.vue), [`store.ts`](front/src/store.ts), [`httpHandlerSession.go`](cmd/dilmeterapi/httpHandlerSession.go).
[^14]: Pre-combat buffer and configuration: [`eventPublisher.go`](cmd/dilmeterapi/eventPublisher.go), [`api_setup.go`](cmd/dilmeterapi/api_setup.go), [`SettingsView.vue`](front/src/components/SettingsView.vue).
[^15]: Pet and puppet attribution: [`aggregator.go`](cmd/dilmeterapi/aggregator.go), [`aggregator_test.go`](cmd/dilmeterapi/aggregator_test.go).
[^16]: Enemy attack separation: [`DamageScatterChart.vue`](front/src/components/DamageScatterChart.vue), [`enemyScatter.ts`](front/src/utils/enemyScatter.ts), [`companionExport.ts`](front/src/utils/companionExport.ts).

[^17]: Rabbie Phantasm run grouping: [`phantasmEncounter.ts`](front/src/utils/phantasmEncounter.ts), [`targetGrouping.ts`](front/src/utils/targetGrouping.ts), [`DamageMeterView.vue`](front/src/components/DamageMeterView.vue).

[^18]: Fahmes Ruin stage and level reconstruction: [`fahmesRuin.ts`](front/src/utils/fahmesRuin.ts), [`targetGrouping.ts`](front/src/utils/targetGrouping.ts), [`DamageMeterView.vue`](front/src/components/DamageMeterView.vue).

[^19]: Conservative Abyss and Gate 4 grouping: [`targetGrouping.ts`](front/src/utils/targetGrouping.ts), [`DamageMeterView.vue`](front/src/components/DamageMeterView.vue), [`reportEncounters.ts`](front/src/utils/reportEncounters.ts), [`highTierMinorGrouping.ts`](front/src/utils/highTierMinorGrouping.ts), and bundled [`races.json`](cmd/dilmeterapi/static_data/races.json).
[^20]: Encounter Arcana resolution: [`arcanaEncounter.ts`](front/src/utils/arcanaEncounter.ts), [`applyDamageBySkill.vue`](front/src/components/applyDamageBySkill.vue), [`DamageGraph.vue`](front/src/components/DamageGraph.vue), [`portableReport.ts`](front/src/utils/portableReport.ts), [`companionExport.ts`](front/src/utils/companionExport.ts), and [`talents.json`](cmd/dilmeterapi/static_data/talents.json).
[^21]: Frontend build configuration: [`front/package.json`](front/package.json), [`front/src/apicall.ts`](front/src/apicall.ts), [`front/src/App.vue`](front/src/App.vue), [`build.bat`](build.bat), and [`build.sh`](build.sh).

[^22]: Npcap capture-mode preference: [`cmd/dilmeterapi/main.go`](cmd/dilmeterapi/main.go), [`cmd/dilmeterapi/api_setup.go`](cmd/dilmeterapi/api_setup.go), [`cmd/dilmeterapi/capture_promiscuous_test.go`](cmd/dilmeterapi/capture_promiscuous_test.go), and [`packet/gamePacketReader.go`](packet/gamePacketReader.go).
