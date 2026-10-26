# Midir-45 verification

- Set pre-combat Skill Use buffer to 0 seconds; verify no pre-pull uses.
- Set to 30 seconds and Apply; use a known non-damaging skill shortly before a boss is recognized; verify it appears in the subsequent session and HTML Skill Use Timeline.
- Leave Midir running idle in town with no combat for an extended period. Confirm stable RAM/CPU and no persistent logging outside combat.
- Check 30-second expiry and the hard 512-item cap with synthetic stress tests.
- Verify deaths and revives in the HTML `data` JSON: `reviveEvents`, `companion.players[].reviveTimes`.
- Verify `encounters` and `companion.encounters` have all four boss attempts for the reference Gate 1–3 recording.
- Check pet-attributed damage rows and tooltip values; boss damage must reconcile with the visible skill breakdown.
- Verify live Skill Use packet semantics; unchanged Midir-44 prerequisite.
