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

- **Boss only** includes damage to the recorded boss entity IDs; **All targets** includes applicable non-boss targets within the encounter window. Never treat all-target damage as boss damage. Incoming enemy-origin attacks are not player damage and must not enter Companion achievements.
- A logical encounter may stitch more than one physical boss phase into a normalized HP track. Preserve raw per-entity and per-phase HP histories as well as combined view data. Missing precise timestamps, recipient IDs, or phases cannot be rebuilt from the display.
- **Gate 1 (Vertrag):** `7600 → 7601` forms one attempt; select both or either form. The 100→50→0 presentation is a normalized boss-HP interpretation. Giants remain outside boss-only damage. See `front/src/utils/gate1Encounter.ts`.
- **Gate 3:** Shard groups use HP-family and timing heuristics; they are not confirmed mechanic timestamps. See `front/src/utils/reportEncounters.ts` and related utilities.
- **Glenn Bearna:** `7585` and `7586` are Unfinished Reverie phases; `7587` is Winter Tyrant. Select all bosses, both Reverie phases, Tyrant, or an individual phase. Minor enemies are separate. See `glennEncounter.ts`, `glennMinorEnemies.ts`.
- **Glenn opening:** Cailleach `7575` is a separate 100%→90% HP checkpoint, **not** a 0-HP defeat. Its observed barrier races are `7546`–`7549` (a split-party side may not be recorded), and Icicle Sprites are `7539`/`7583`. The opening is selectable apart from Reverie/Tyrant to avoid misleading combined HP. See `glennOpening.ts`.
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
