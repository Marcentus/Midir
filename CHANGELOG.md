# Midir — Changelog

Records **what changed and why**. Paths point to the relevant source in this candidate; they are not commit links. For packet layouts, export schemas, and interpretation rules, see [`SKILL-AND-EXPORT-REFERENCE.md`](SKILL-AND-EXPORT-REFERENCE.md).


## Midir-45 — Skill evidence and export refinements (October–November 2026)

- **Capture explicit skill uses:** Save player-attributed `0x6988` events with millisecond timestamps so non-damaging actions are available to offline analysis. A bounded, configurable pre-combat buffer retains recent uses without recording idle activity. `cmd/dilmeterapi/eventPublisher.go`, `event.go`, `httpHandlerSession.go`.
- **Preserve remote skill clues:** Record combat-action (`0x7926`) and typed effect (`0x9093`/`0x9095`) observations for Companion analysis rather than guessing casts during live capture. `cmd/dilmeterapi/eventPublisher.go`, `event.go`; `front/src/utils/portableReport.ts`.
- **Export independent evidence streams:** Include explicit uses, positive damage hits, and skill observations once in the HTML JSON payload; retain existing Companion v1 and encounter representations. `front/src/utils/portableReport.ts`, `cmd/dilmeterapi/httpHandlerSession.go`.
- **Improve encounter portability:** Embed canonical encounters, revival evidence, and pet-attributed skill-9999 damage while retaining legacy fields for older Companion readers. `front/src/utils/portableReport.ts`, `front/src/utils/companionExport.ts`.
- **Simplify HTML navigation:** Show **Cleared** / **Attempted**; select an encounter once, then choose **Boss only** / **All targets**. Keep player chart filters collapsed, remove boss-origin checkbox clutter and the Skill Use Timeline UI; preserve its underlying data streams. `front/src/utils/portableReport.ts`.
- **Gate 1 — Vertrag:** Make the combined two-form boss row selectable while retaining separate form selections and independent expand/collapse. Boss-only selection excludes Giants. `front/src/utils/gate1Encounter.ts`, `targetGrouping.ts`.
- **Glenn Bearna:** Group Unfinished Reverie (two phases) and Winter Tyrant by attempt, with combined/individual boss selection. Add Cailleach (`7575`) as a separate opening HP checkpoint (100%→90%); show Icicle Sprites and encountered Cailleach barriers as opening mechanics. Keep minor enemies separately grouped. `front/src/utils/glennEncounter.ts`, `glennOpening.ts`, `glennMinorEnemies.ts`, `instanceMinorGrouping.ts`.
- **Crom Bas:** Group Irusan and whichever Glas variant is recorded by attempt, with combined and individual boss views. Separate Merch Diafol and a single combined Disciple-mechanics group from boss HP. `front/src/utils/cromEncounter.ts`, `mechanicGroups.ts`, `instanceMinorGrouping.ts`.
- **Dungeon run folders:** Place Crom Bas bosses and minor enemies, and Glenn Bearna Cailleach openings, bosses, and minor enemies under a single numbered run folder. Keep boss-only HP and individual target scopes available; use recorded timing to avoid linking mismatched runs. `front/src/utils/dungeonRunFolders.ts`, `targetGrouping.ts`, `front/src/components/DamageMeterView.vue`.
- **Alby Shineseeker:** Group each recorded run into five selectable waves, from rats/bats through Cichol’s Followers. Report all observed final-wave Followers defeated without implying a verified dungeon-clear packet. `front/src/utils/albyShineseeker.ts`, `targetGrouping.ts`, `front/src/components/DamageMeterView.vue`.

## Midir-44 — Skill-to-damage exploration (October 2026)

- Added millisecond-precision hit evidence to compare recorded skill uses with later damage. The experimental **Skill Use Timeline** visualization was subsequently removed; its event streams remain. `front/src/utils/portableReport.ts`, `cmd/dilmeterapi/httpHandlerSession.go`.
- Evaluated temporal matching as a diagnostic aid—not a reliable completed-cast counter. Current interpretation rules are in the technical reference.

## Midir-42/43 — Recorder, charts, and build safeguards (October 2026)

- **Target-aware damage:** Track recipient IDs and precise timestamps where present, allowing boss-only damage to differ correctly from multi-target totals. Historical recordings cannot gain missing hit metadata. `cmd/dilmeterapi/event.go`, `httpHandlerSession.go`; `front/src/utils/portableReport.ts`.
- **Enemy-origin attacks:** Identify known boss attackers separately from party players to avoid inflating player DPS and Companion achievements. Earlier HTML enemy checkboxes were later removed. `front/src/utils/enemyScatter.ts`, `companionExport.ts`.
- **Saved-session reliability:** Add explicit skill-use event storage and compatible NDJSON reconstruction without changing legacy second-based timestamps. `cmd/dilmeterapi/event.go`, `httpHandlerSession.go`.
- **Build safeguards:** Add Node-version guardrails and GitHub Actions build checks. `.nvmrc`, `scripts/check-node-security.cjs`, `.github/workflows/build-validation.yml`.

## Earlier work — Historical feature milestones (Midir-18 through Midir-41)

- **Combat analysis:** Introduced interactive damage scatter, HP traces, death markers, player skill breakdowns, sortable panels, Ghost Runner and per-skill personal bests. These serve live review and comparison across saved sessions. `front/src/components/DamageMeterView.vue` and supporting chart utilities.
- **Healing:** Added Healing Done and defensive-effect estimates with checks for implausible values. Caster and shield attribution can remain uncertain; these are not proof of individual casts. Backend healing parsing and aggregator code.
- **Recording:** Improved buffered NDJSON writing, large-line session loading, and session management. Removed the temporary healing-diagnostic recorder and its controls. `cmd/dilmeterapi/httpHandlerSession.go` and recording handlers.
- **Portable reports:** Developed one self-contained HTML report with boss-HP history, target-scoped statistics, logical encounters, and embedded Companion JSON; discontinued the separate Companion JSON button. `front/src/utils/portableReport.ts`, `companionExport.ts`.
- **Encounter handling:** Iterated Gate 1 phase pairing, Gate 3 shard estimates, Glenn Bearna, Crom Bas, and other named encounters. Current rules take precedence over superseded early matching heuristics. `front/src/utils/reportEncounters.ts`, `targetGrouping.ts` and dungeon utilities.
- **Quality and usability:** Added saved-session batch actions and fail-fast build safeguards; old release-note hotfix sequences were consolidated rather than retained as separate features.

## Documentation notes

- Legacy `RELEASE-*.md`, temporary grouping notes, and separate skill-testing documents were consolidated into this changelog and the skill/export reference. This does not change runtime behavior.
- Historical fixture figures and exploratory conclusions are documented only where useful for troubleshooting; they are not universal expected values.
