package main

import "testing"

func TestAttachSkillGroupsFromDamageEvents(t *testing.T) {
	summary := &FightSummary{Players: map[string]PlayerStats{
		"123": {ID: "123", Name: "Caster"},
	}}
	events := []eventDamage{
		{eventBase: eventBase{At: 100, Id: "123"}, SkillId: 10, Damage: 100, IsCritical: true},
		{eventBase: eventBase{At: 100, Id: "123"}, SkillId: 10, Damage: 50},
		{eventBase: eventBase{At: 101, Id: "123"}, SkillId: 20, Damage: 25},
		{eventBase: eventBase{At: 101, Id: "123"}, SkillId: 30, Damage: -40}, // healing is not charted as damage
	}

	attachSkillGroupsFromDamageEvents(summary, events, 100)
	player := summary.Players["123"]
	if len(player.SkillUses) != 2 {
		t.Fatalf("expected 2 timeline buckets, got %d", len(player.SkillUses))
	}
	if player.SkillUses[0].Timestamp != 0 || player.SkillUses[0].TotalDmg != 150 {
		t.Fatalf("unexpected first bucket: %#v", player.SkillUses[0])
	}
	if len(player.SkillUses[0].Skills) != 1 {
		t.Fatalf("expected one skill in first bucket, got %d", len(player.SkillUses[0].Skills))
	}
	skill := player.SkillUses[0].Skills[0]
	if skill.SkillID != 10 || skill.HitCount != 2 || skill.Damage != 150 || skill.MaxDamage != 100 || skill.CritCount != 1 {
		t.Fatalf("unexpected skill summary: %#v", skill)
	}
	if player.SkillUses[1].Timestamp != 1 || player.SkillUses[1].TotalDmg != 25 {
		t.Fatalf("unexpected second bucket: %#v", player.SkillUses[1])
	}
}

func TestApplySavedSupportAndSkillDataRestoresSavedOvershield(t *testing.T) {
	live := &FightSummary{Players: map[string]PlayerStats{}}
	healing := newDamageBreakdown()
	healing.TotalDamage = 125
	stored := &SessionSummaryData{Players: []SessionSummaryPlayer{{
		ID: "456", Name: "Healer", OverallHealing: healing,
		OvershieldProvided: 300,
	}}}

	applySavedSupportAndSkillData(live, stored)
	player, ok := live.Players["456"]
	if !ok {
		t.Fatal("expected support-only player to be restored from saved summary")
	}
	if player.OverallHealing.TotalDamage != 125 || player.OvershieldProvided != 300 {
		t.Fatalf("saved support totals were not restored: healing=%v overshield=%v", player.OverallHealing.TotalDamage, player.OvershieldProvided)
	}
}
