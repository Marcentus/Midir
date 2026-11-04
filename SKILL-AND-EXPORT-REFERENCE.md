# Skill Recording & Portable Export — Developer Reference (Midir-45)

**Purpose:** Current data contract and best-supported packet interpretations for the Companion. This document is maintained to describe **what the source does now**, not every experiment that led here. Superseded hypotheses are omitted; unresolved questions are marked explicitly. Midir-45 is a **source candidate**, not a fully built and gameplay-validated release.

## 1. Pipeline and source map

1. **Observe packets:** `cmd/dilmeterapi/eventPublisher.go` captures combat-gated player skill uses and bounded remote evidence. `cmd/dilmeterapi/event.go` defines saved NDJSON event IDs and fields.
2. **Reconstruct sessions:** `cmd/dilmeterapi/httpHandlerSession.go` reads saved events into player skill-use, observation, damage, and encounter data (`summary.go`).
3. **Export HTML:** `front/src/utils/portableReport.ts` writes one JSON payload into `<script type="application/json" id="data">`; `companionExport.ts` retains the embedded Companion v1 structure. No separate JSON download is required.
4. **Interpret offline:** The Companion should correlate packet evidence, hits, target identities, and phases **without treating every row as a completed cast**.

The report currently displays **Damage Scatter & Boss HP**. It has one encounter selector and a separate Boss only / All targets option, player-only chart filters, and **Cleared / Attempted** labels. The old Skill Use Timeline UI and enemy-origin chart checkboxes were removed; their raw analytical evidence was not intentionally deleted.

## 2. Packet identifiers and how to read them

| Opcode | Observed meaning | Saved representation | Interpretation limit |
|---|---|---|---|
| `0x6988` | Explicit **Skill Use** packet for an identified player | `EventId: 8` with `Id`, `SkillId`, `At`, `AtMs`, optional `TargetId` | A message is evidence of use, **not universally one completed cast**. |
| `0x698c` | **Skill Start** contributes to combat timing | No dedicated saved skill-start event | Do not infer a use event from its absence in NDJSON. |
| `0x7926` | Combat Action subpacket can contain attacker entity ID, skill ID, action type | `EventId: 11`, `actorVerified: true` when attacker is a known player | One skill-bearing subpacket can describe one action stage or hit, not one cast. |
| `0x9093` | General **Effect** messages, including visual/audio/status cues and sometimes typed skill IDs | `EventId: 11` with `fields`, `fieldTypes`, `subtype`; `actorVerified: false` | Message addressed to a player does **not** establish that player cast the effect. Effects can repeat. |
| `0x9095` | Delayed/scheduled effect messages | Same observation stream; `actorVerified: false` | A scheduled or repeated effect is not an activation count. |
| `0xa028` | Status-condition observations in earlier diagnostics | Not introduced as a new Midir-45 skill-observation stream | Correlation with buffs/debuffs is **not** a proven cast or caster identifier. |

**What is actually recorded:** `0x7926` observations require a known player as the parsed **attacker** and a nonzero skill ID. `0x9093`/`0x9095` capture messages addressed to known players during active combat; the recipient may be distinct from the caster. The effect collector stores at most **12 typed fields**, truncates strings to **160 characters**, and omits binary payload bytes. World-object-directed effects can therefore be absent. `eventPublisher.go` intentionally does not deduplicate or guess skill matches live.

### Typed skill-ID candidates recognized by the current exporter

`portableReport.ts` extracts an ID **only if** the field is at the supported index, has the expected protocol type, and exists in the current skill-name table. Array indexes below are **zero-based**; the first field is often the effect subtype.

| Packet / subtype | Candidate skill-ID location | Required type code |
|---|---|---|
| `0x9093`, `304` or `145` | `fields[2]` | `2` (Short) |
| `0x9093`, `419` | `fields[1]` | `3` (Int) |
| `0x9093`, `833`, `660`, or `661` | `fields[1]` | `2` (Short) |
| `0x9095` with `fields[1] == 419` | `fields[2]` | `3` (Int) |
| `0x7926` | Parsed `skillId` | From Combat Action parser, not generic effect fields |

Relevant type codes: Byte `1`, Short `2`, Int `3`, Long `4`. `fieldTypes` preserves the original types; do not search all numerical fields for something that merely resembles a skill ID. A nonzero `candidateSkillId` establishes **possible skill identity**, not correct caster or cast count. For effect records, `skillId` may remain zero even when `candidateSkillId` resolves.

### Named signatures from controlled diagnostic work

These are **correlations** and should be tested against neighboring packet events before generating Companion actions:

- `0x9093` subtype `14`: strings such as `healing_firstaid` and `party_healing` were associated with First Aid / Party Healing; an effect may occur even with no HP restored.
- Subtype `304`: recognized music-skill identifiers during Battlefield Overture, Vivace, Enduring Melody, and March Song testing.
- Subtype `419`: skill-bearing cues for Shield of Trust / Celestial Spike in investigated samples.
- Subtype `833`: skill-bearing cues for Bash, Spinning Uppercut, and Charge; format is reused and requires type checking.
- Subtype `145`: observed Wind Blast cue (`shot`); subtype `400` carried a placement cue (`cook_pot_outline`) near Hydra Transmutation.
- Subtypes `335` / `838`: paired Gravity Well-associated effect activity; **not** a unique cast signature.
- Subtypes `776` / `777` / `778` / `779`: associated in prior tests with Blessing of Protection, Voice of Vitality, Pall of Ruination, and Hand of Restoration respectively. Subtype `836` was observed near Astral Reave.
- `0x9095`: delayed effect sequences were seen around Encore; multiple rows do not imply multiple Encore activations.

This list preserves the **most useful tested identifiers**, not a full catalog or a guarantee of unique one-to-one mappings. If a subtype lacks a typed recognized skill ID, keep its raw fields and uncertainty rather than manufacturing a confirmed skill-use event.

## 3. Saved event and export contract

| Field / stream | Meaning |
|---|---|
| Saved `EventId: 8` | Explicit player skill use. `At` is Unix **seconds**; `AtMs` is Unix **milliseconds**; `TargetId` is a hint when available. |
| Saved `EventId: 11` | Skill observation from Combat Action or Effect. This is independent of explicit uses. |
| Saved damage `EventId: 3` | Damage record retains historical seconds and optional precise `AtMs`. Older records may lack it. |
| `skillUseSchemaVersion: 1`; `skillUseEvents` | Explicit observed-use stream. `skillUseEvidence` warns these are `0x6988` observations, not confirmed cast counts. |
| `damageHitSchemaVersion: 1`; `damageHitEvents` | Positive hit stream: `playerId`, `skillId`, `atMs`, `relativeMs`, `targetId`, `damage`, `critical`, `delayed`. |
| `skillObservationSchemaVersion: 1`; `skillObservations` | Evidence: `playerId`, `playerName`, `atMs`, `relativeMs`, `opcode`, `skillId`, `skillName`, `candidateSkillId`, `subtype`, `actionType`, `actorVerified`, `fields`, `fieldTypes`. |
| `encounterSchemaVersion: 1`; `encounters` | Canonical encounters kept alongside legacy representations, **not separate additional fights**. |
| `companion.version: 1` | Existing Companion-compatible structure retained. |

**Time:** `atMs` is absolute Unix milliseconds; `relativeMs` is relative to the report summary's first-damage time and can be **negative** for pre-pull events. **Entity IDs** are strings in exported streams to avoid 64-bit JavaScript rounding. Each stream appears **once**, at top level, not once per report view. Saved-session `players[].skillActivations` (explicit uses) must not be confused with damage-group `players[].skillUses`.

**Pre-combat buffering:** explicit `0x6988` uses can be buffered for **0–30 seconds** (at most **512 entries**) and replayed into an active combat recording. It is **not** a pre-combat buffer for all remote effect observations.

**Completion:** `completion: "observed-zero-hp"` remains the machine-readable successful-clear value; UI says **Cleared**. Other concluded encounters display **Attempted** if no zero boss HP was captured. Attempted **does not prove a wipe**.

## 4. Damage, boss HP, and mechanic selection

- **Enemy source identity:** the Damage Graph, HTML export and Companion filters distinguish boss and minor-enemy attack sources from party players. Numeric-looking source names alone are insufficient: a non-boss source must match a recorded target **entity ID**, with a non-player race and spawn/HP evidence. Example: Glenn Gwyllgi race `7544`; older logs may use a different numeric entity ID as its display name. Recognized boss Race IDs also support corroborated historical name fallbacks. Unverified IDs remain unclassified. See `enemyScatter.ts`.
- **Boss only** includes damage to the recorded boss entity IDs; **All targets** includes applicable non-boss targets within the encounter window. Never treat all-target damage as boss damage. Incoming enemy-origin attacks are not player damage and must not enter Companion achievements.
- A logical encounter may stitch more than one physical boss phase into a normalized HP track. Preserve raw per-entity and per-phase HP histories as well as combined view data. Missing precise timestamps, recipient IDs, or phases cannot be rebuilt from the display.
- **Gate 1 (Vertrag):** `7600 → 7601` forms one attempt; select both or either form. The 100→50→0 presentation is a normalized boss-HP interpretation. Giants remain outside boss-only damage. See `front/src/utils/gate1Encounter.ts`.
- **Gate 3:** Shard groups use HP-family and timing heuristics; they are not confirmed mechanic timestamps. See `front/src/utils/reportEncounters.ts` and related utilities.
- **Glenn Bearna:** Reverie `7585` is a standalone 100%→10% boss encounter. Reverie `7586` (100%→60%) and Winter Tyrant `7587` (60%→0%) share a second, continuous health track. The combined boss-damage view retains all three entity IDs; individual phases remain selectable. See `glennEncounter.ts`, `reportEncounters.ts`, `glennMinorEnemies.ts`.
- **Glenn opening:** Cailleach `7575` is a separate 100%→90% HP checkpoint, **not** a 0-HP defeat. Her entrance barriers (`7546`–`7549`) are opening mechanics; a split-party side may not be captured. Icicle Sprites (`7539`, `7583`) are ordinary minor enemies, not mechanics. See `glennOpening.ts` and `glennMinorEnemies.ts`.
- **Alby Shineseeker:** No final boss. Each run starts with Vampire Bat `195000` / Country Rat `195001`, then Skeletons `195002`–`195004`, Black Wizard/Ghost Armor `195005`–`195006`, Gargoyles `195007`–`195008`, and final Cichol’s Followers `195009`–`195011`. Run boundaries are inferred from fresh first-wave bat spawns. `final-wave-defeated` requires observed zero HP and death flags for **all recorded** final-wave Followers; this is not an authoritative dungeon-clear packet. See `albyShineseeker.ts`.
- **Crom Bas:** Irusan `193364`; final boss Completed Glas `193358` **or** Giga Glas `193359`, whichever was recorded. Merch Diafol `193355` is an Irusan mechanic. Glas Disciples (red `193438/193442`, blue `193439/193443`, white `193440/193444`, yellow `193441/193445`, plus recognized variants) share **one mechanics group**; none contributes to boss HP. Disciples appearing before Glas are not automatically Glas mechanics. See `cromEncounter.ts`, `mechanicGroups.ts`.
- **Run boundaries:** separate observed boss attempts by entity/time. NDJSON does **not** provide a guaranteed dungeon-instance ID; minor-enemy runs or gap-based *combat segments* can be inferred but must not be presented as confirmed instances. See `instanceMinorGrouping.ts`.

## 5. Companion interpretation rules

1. Prefer **explicit use** (`0x6988`) as use evidence, but validate repeat behavior per skill before counting casts.
2. Use `0x7926` as **actor-attributed combat-action evidence**, not automatically a distinct action for each subpacket.
3. Use typed `0x9093`/`0x9095` signatures as **candidate identity / timing clues**, not proof the addressed player cast the skill. Deduplicate repeated effects using contextual evidence, not a global one-row-one-cast rule.
4. Correlate hits by player, skill, target, and time only when those fields are known. A former 30-second matching window was an **experimental visualization heuristic**, not a verified algorithm for Companion cast counts.
5. Do not backfill events missing from older NDJSON logs; an older report lacking `AtMs`, `EventId: 8`, or `EventId: 11` cannot acquire that evidence without new raw capture analysis.

## 6. Verification and outstanding limits

- **Source-based:** typed candidate slots and exported fields above were checked against `portableReport.ts`; recording gates and attribution behavior against `eventPublisher.go`. This documentation pass did **not** run the app or validate new packets in live play.
- **Sample regression:** historical `session-1792643233581.ndjson` distinguished Dynavolt Smash all-target **10,513,142** from known boss damage **3,245,377**. Those figures only describe that fixture; they are not a specification.
- **Still required:** Build frontend/backend, re-export new and old sessions, test browser controls and Companion import, confirm clear/attempt status and boss phase HP, and manually cross-check buff, multi-hit, non-damaging, and remote casts against recordings.
- **Known blind spots:** recipient-versus-caster ambiguity, repeated effect messages, missing world-object-addressed effects, incomplete older timestamps/targets, inferred dungeon segments, and partial encounters.

To inspect an export, open the HTML in a browser and read `JSON.parse(document.getElementById('data').textContent)`. Treat stream counts as **observations** until their event semantics have been tested for the relevant skill.

- **Glenn HP export caveat:** Tyrant may initially report a provisional 100% spawn snapshot before the settled ~60% phase appears. The linked HP chart excludes that artificial reset. Raw per-entity HP observations remain unchanged in Companion metadata.

## Dungeon wave / Queen recognition

Fahmes Ruin uses recorded `121xxx`–`126xxx` stage-local race families for stages 1–6. Five sequential level spawn groups are inferred per recorded stage. For a complete 150-enemy stage, five chronological groups of 30 prevent delayed enemies from collapsing or suppressing the level selector; partial captures retain timing-based segmentation. This is a stage-specific heuristic supported by the combined recording, not an authoritative level-transition packet. Waves inside a level may overlap, so wave-specific timing is not asserted without packet evidence. Each stage can be entered independently. Normal Phantasm Queen `191681` and Temptation's Path Queen `195012` are confirmed. `191699`, `191724`, `191803`, `193551`, `193709`, and `193723` are unverified candidates and require contextual dungeon evidence; possible outfit replacements are additional raw entities of one logical Queen, not independent clears.

### Cross-dungeon race ownership and Phantasm boss-room summons

In combined sessions, Phantasm Wraiths (`191684`) share a race with Crom Bas minor enemies. The Phantasm run detector owns its recorded Wraith entity IDs before Crom minor combat-segment reconstruction; the Crom folder, UI group, and target-selection resolver consistently exclude those entities. Fahmes Stage 5 remains identified from its own `125xxx` races, not from the coincidental Wraith race.

Normal Phantasm spawns Succubus Fiends (`191698`) and Wraiths (`191684`) after the Queen (`191681`) appears. A bounded post-Queen window (at most 300 seconds, ending sooner at a following Queen or Fahmes stage) includes known boss-room Fiend/Wraith races (`191683`, `191684`, `191698`) under a separate **Boss-room Enemies** selection. Earlier mobs remain **Minor Enemies**. Both selections contribute to the unique run-wide total without changing the Queen-only boss damage/HP. This is a chronological inference; the original export does not encode authoritative room boundaries. The older Temptation's Path four named Succubi remain a separate mechanics selection.

### Recognized boss-originated scatter sources

The portable HTML report keeps enemy outgoing attacks separate from party damage. Boss-source labels resolve from exact encounter entity IDs first, then recorded target entity IDs with verified boss Race IDs. For historical sources with a numeric display label but no retained entity match, the numeric value may be interpreted as a boss Race ID **only if that race is independently observed in the same session**. The fallback labels the boss race, not an invented encounter attempt. Candidate Succubus Queen outfit/event IDs are excluded from standalone recognition unless the Phantasm encounter detector independently accepts them. Numeric entity IDs alone are never evidence of enemy ownership. These rules do not rewrite raw damage packets, the players cache, or Companion source IDs.

### Crom Bas Abyss / Bri Leith Gate 4 — provisional minor-enemy grouping

The bundled `races.json` identifies Midir the Lost (`7615`) and Folamh (`193810`) as recognized bosses. These remain **boss-only, separately selectable** targets, with same-race repeat spawns bounding their numbered attempt labels. Neither encounter has a recorded, verified phase/clear model.

Based on identified dungeon associations supplied by a player familiar with gameplay, the provisional **Minor Enemies** registry groups Gate 4 Shards of Solace (`7616`–`7620`, `7622`) and Abyss Folamh's Apparitions (`193811`–`193813`), Abyssal Barriers (`193814`), Dark Erg Regulators (`193815`), Disciples (`193816`), and Erg Slimes (`193817`). These are not assumed to be boss mechanics or synchronized HP-threshold waves. The possible bosses `193809` (Giga Glas) and `193818` (another Folamh) remain unassigned until evidence clarifies their role.

`highTierMinorGrouping.ts` creates separately selectable minor-enemy scopes by recorded name, partitions segments by observed spawn times using `partitionMinorRuns`, and nests a boss selection and its approximate minor segment under a numbered encounter folder when both exist. If no matching boss is recorded, it still exposes the minor segment independently. The between-boss midpoint and 15-minute gap are **heuristics**, not definitive dungeon boundaries; the UI does not assert a clear, phase, or mechanic sequence. Boss HP/DPS remains isolated from minor enemies; an all-enemies folder sums unique recorded targets without rewriting raw events. Normal Crom Bas's `193816` handling is kept separate so an Abyss Disciple is not absorbed into a normal-Crom trash folder.

Standalone HTML encounter recognition continues to show the observed bosses without declaring unverified mechanics. Full validation of high-tier run segmentation and enemy role attribution requires real recordings.


## Encounter-scoped Arcana evidence

- The legacy `PlayerStats.talentName` and the saved-session summary Arcana are player/session-level hints. A player may use multiple Arcanas between distinct encounters in one recording; these hints must not be treated as authoritative encounter classifications.
- Frontend target selections resolve Arcana from recorded per-target skill breakdowns and explicit skill observations with the selected target ID. Whole-session views enumerate every detected Arcana; if multiple are present, display a neutral mixed classification. Unsupported selections remain unknown.
- Portable HTML report `views[].players[]` now includes `arcana` and `arcanaEvidence`, with view-specific colors. Boss-only views use exact recorded recipient entities. All-target encounter windows use timestamped damage hits and explicitly recorded skill activations, not the full-session talent name. Without timestamp evidence, a window cannot assume an Arcana.
- Companion v1 keeps its existing player `arcana` field (a single identity or null) and adds optional `arcanaEvidence`; `encounters[].playerArcanas[]` supplies the per-player identity for a recognized encounter's boss entities. A mixed or unsupported scope yields null for its single Arcana field. These are identities derived from observed skills, not verified activation counts. The Arcana registry comes from the backend's bundled `talents.json`.
- Encounter boundaries reconstructed from target entities remain inferential; when multiple Arcana families are seen in the same selected scope, report a mixed classification instead of silently selecting the first or last.

- Mixed-Arcana display: full-session player bars and exported report headers use smooth, evenly spaced gradients for two or more observed Arcanas. Timestamped hit/use records establish first-observed ordering where available; older summary-only sessions retain recorded skill ordering. Gradients are not estimates of Arcana damage, time, or activation share. Scoped encounters retain their independently resolved identity and solid color.
