# Midir-44 — Skill Use Timeline / No Damage Dealt — testing guide

Midir-44 includes **all Midir-43 Skill Use recording changes**. You do not need to compile or run Midir-43 separately unless you want an A/B comparison. Both ZIP archives remain independently usable; Midir-44 is intended to be a follow-up test candidate.

## What the exported HTML looks like

The normal Combat Report now has exactly one chart displayed at a time. Above the encounter view selector are two buttons:

- **Damage / Boss HP** (default): unchanged one-second damage scatter, enemy tracks, HP curves, phase search, zoom/pan, and party statistics.
- **Skill Use Timeline**: independent horizontal timeline of observed `0x6988` packets. Each party player has a colored use-marker lane. Individual following damage hits are marked with smaller diamonds on the same lane; hover a use to see its matched total/hit count and hover a diamond for the original hit amount/time. **No Damage Dealt** is a dedicated bottom band for uses with no associated positive damage hit.

The report view selector still offers the session and individual boss attempts. Skill-use events are filtered to the selected encounter window; matching uses in a boss-only encounter still display all matched target damage, with separate boss-recipient damage in the tooltip. Enemy-originated attacks remain on the **Damage / Boss HP** chart only. Zoom with the mouse wheel, drag to pan, double-click to reset in both modes. Large full-session hit sets downsample visible hit *markers* (not calculations); zoom in for detailed hit symbols.

## Required test sequence

1. Compile Midir-44 with your Windows environment. Use a **new** recording: old Midir-42 reports and old NDJSON cannot contain skill-use packet evidence that was never saved.
2. During a boss encounter, activate a non-damaging buff, a normal damaging ability, and a multi-hit ability. Where possible, use a skill in a boss invulnerability/mechanic phase. Note approximate times and names. Keep the boss entity present.
3. Save the new session. Verify `EventId:8` lines exist with player `Id`, `SkillId`, `AtMs`. Verify positive `EventId:3` records carry `AtMs`. There should be **no new Skill Start event ID 9 records**.
4. Reload the saved session and export the normal self-contained HTML. Open locally in Chrome or Edge. The default chart should look identical to Midir-42/43. Select **Skill Use Timeline** and confirm the observed use markers, individually matched damage diamonds, and **No Damage Dealt** bottom band.
5. Hover all three examples; verify times, skill names, player names, and hit totals. Non-damaging uses should be present, not omitted. Check both Gate 2 attempts and the Gate 3 boss-only view, and test the two chart mode buttons, zoom/pan/reset, and player filters.
6. Import the **same exported HTML** into the Companion. Its existing embedded `companion.version=1` has not been changed. New timeline data is stored in the same `<script type="application/json" id="data">` block.

### Inspecting the HTML directly (DevTools)

```js
const report = JSON.parse(document.getElementById('data').textContent);
console.log('Observed uses:', report.skillUseEvents.length);
console.log('Precise damage hits:', report.damageHitEvents.length);
console.table(report.skillUseEvents.slice(0, 10));
console.table(report.damageHitEvents.slice(0, 10));
```

`skillUseEvents` is the observed skill-use stream. `damageHitEvents` is a separate positive-hit stream with `atMs`, `relativeMs`, `playerId`, `skillId`, `targetId`, `damage`, `critical`, and `delayed`. IDs are strings to preserve 64-bit precision. Both streams occur **once** at the top level of the report, not duplicated for every boss view. The Companion can consume them directly from the HTML. The human-readable report has no need for a separate JSON file.

### Interpretation and limitations

- An `0x6988` use packet is **observed activity**, not yet validated to mean exactly one completed cast for every Mabinogi skill. Midir-43's controlled packet test is still important.
- Matching is by **player ID + skill ID + latest preceding observed use**, with a maximum **30-second** gap. A newer use of the same skill takes precedence. Associations are inferred and can be imperfect for procs, overlapping delayed damage, or skills with different damage skill IDs.
- **No Damage Dealt** means no *matching, recorded, positive hit* in the 30-second matching window. It does not assert that the skill was useless, that a healing/buff skill failed, or that the player made an error.
- If a use occurs before the first damage hit of a session, it may appear at a negative session-relative time; this is intentional and correctly represents pre-damage preparation.
- This revision does not change the pet attribution/small hidden pet damage issue and does not add a separate cast chart to Midir's live Graph component. It adds the alternate chart to the **exported HTML**.
- The source candidate was syntax-checked and tested in Chromium using **injected use/hit events reconstructed from a Midir-42 summary**. A full compiled application and live packet stream have **not** been tested here.
