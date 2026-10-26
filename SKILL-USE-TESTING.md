# Midir Skill Use recording — validation guide

This is a **source candidate** on top of Midir-42. Continue using the **same self-contained HTML** to import into the Companion. A separate JSON export is **not** required.

## Capture a new controlled recording

1. Build Midir, open a boss encounter, and use at least one non-damaging buff, one offensive skill, one multi-hit skill, and one ability that targets another party member while the boss is present. Note a rough time and the skills pressed. Test a phase where the boss is invulnerable but remains spawned.
2. Save the new session to an `.ndjson`. Find records with `"EventId":8`: these are the observed `0x6988` events. Check each has `Id`, `SkillId`, `At` and `AtMs`, and optionally `TargetId`. No `EventId:9` Skill Start events should be introduced.
3. Check damage event `EventId:3` records include `AtMs` for newly recorded damage alongside the original `At` seconds. Old sessions still open without `AtMs` and cannot recover historic skill uses.
4. Reload the saved session in Midir, export the normal HTML, and open it in a browser. In DevTools console:

```js
const d = JSON.parse(document.getElementById('data').textContent);
console.table(d.skillUseEvents.slice(0, 20));
console.log('Observed use packet count:', d.skillUseEvents.length);
```

`skillUseEvents` is **top-level in the embedded HTML data**, not a separately downloaded file. Check player attribution, skill IDs/names, order, millisecond timestamps, and non-damaging actions. `relativeMs` measures from the summary's first recorded damage, so negative values are possible for preparations immediately before the first hit. The raw `atMs` is always absolute.

## Regression and interpretation

- Confirm the existing damage scatter, encounter views, HP curve, enemy toggles, zoom/pan, total damage, and Companion v1 import still function.
- Confirm skill-use events on non-damaging mechanic intervals aren't automatically discarded while the boss is present.
- Check whether one player input produces one, multiple, or no `0x6988` events for each tested skill. **Do not label use packets as verified completed casts or APM until this is validated.**
- If a use packet contains its own ID as `TargetId`, the report omits that target hint as non-informative. Other `TargetId` values are hints based on packet shape and not yet verified to represent recipient identity for every skill.
- Compare performance and file sizes with comparable sessions after validation. The change does not send additional network traffic; it retains data from packets Midir already receives. An extra in-memory event is held for each qualifying Skill Use.
- **Midir-44 adds** the Skill Use Timeline and embeds precise damage-hit events inside the same HTML. See `SKILL-USE-TIMELINE-TESTING.md` for the updated test process.

## Known limitations

- Only verified player-entity packets received while Midir considers a combat target present are recorded. Unknown players before their appearance packet, town activity, and actions after the boss disappears are not promised.
- Older `.ndjson` files have no independent use events unless they previously contained event ID 8. Reloading old HTML cannot reconstruct nonexistent use packets.
- Skill names are resolved from the current Midir skill lookup; unknown names are exported with their raw skill ID.
- Fully compiling/running Midir requires the project Go/Node dependencies and appropriate native packet capture libraries on the build host.
