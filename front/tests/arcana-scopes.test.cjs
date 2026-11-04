// Regression for a recording where one player switches Arcana between runs.
// Runs with `node tests/arcana-scopes.test.cjs` after npm ci in front/.
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const vm = require('node:vm');
const ts = require('typescript');
const registry = require('../../cmd/dilmeterapi/static_data/talents.json');
const file = path.resolve(__dirname, '../src/utils/arcanaEncounter.ts');
const source = ts.transpileModule(fs.readFileSync(file, 'utf8'), {
  compilerOptions: { module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2022, esModuleInterop: true }
}).outputText;
const moduleRef = { exports: {} };
vm.runInNewContext(source, { module: moduleRef, exports: moduleRef.exports, require(id) {
  if (id.endsWith('/talents.json?raw')) return JSON.stringify(registry);
  if (id === '@/util') return { getClassColor(name, color) { return name === 'Sacred Guardian' ? '#22aa66' : color; } };
  if (id === './targetGrouping') return { getGroupMemberTargetIds(id) {
    if (id === 'stage-5') return ['stage5-enemy'];
    if (id === 'stage-2') return ['stage2-enemy'];
    if (id === 'mixed-group') return ['stage5-enemy', 'stage2-enemy'];
    return [];
  }};
  throw new Error('Unexpected import: ' + id);
}}, { filename: file });
const { resolvePlayerArcana, resolvePlayerArcanaForIds, resolvePlayerArcanaWindow, arcanaBarBackground, getSkillArcana } = moduleRef.exports;
const all = Object.values(registry.arcanas);
const sharpshooter = all.find(a => a.arcana_name === 'Alchemic Sharpshooter');
const guardian = all.find(a => a.arcana_name === 'Sacred Guardian');
assert(sharpshooter && guardian);
assert.equal(getSkillArcana(guardian.related_skill_ids[0]).color, '#22aa66');
assert.equal(getSkillArcana(sharpshooter.related_skill_ids[0]).name, 'Alchemic Sharpshooter');
assert.equal(getSkillArcana(9999), undefined);
const sharpId = sharpshooter.related_skill_ids[0], guardianId = guardian.related_skill_ids[0];
const skill = id => ({ skills: { [id]: { totalDamage: 1000, count: 1 }} });
const player = {
  id: '123', name: 'Example', talentName: 'Alchemic Sharpshooter',
  overallStats: { skills: { ...skill(sharpId).skills, ...skill(guardianId).skills }},
  damageByTarget: { 'stage5-enemy': skill(guardianId), 'stage2-enemy': skill(sharpId) },
  damageEvents: [
    { skillId: sharpId, atMs: 1000 * 101, targetId: 'stage2-enemy' },
    { skillId: guardianId, atMs: 1000 * 201, targetId: 'stage5-enemy' }
  ]
};
const summary = { players: { '123': player }, targets: { 'stage5-enemy': {}, 'stage2-enemy': {} }};
assert.equal(resolvePlayerArcana(player, summary).status, 'mixed');
assert.equal(resolvePlayerArcana(player, summary, 'stage-5').name, 'Sacred Guardian');
assert.equal(resolvePlayerArcana(player, summary, 'stage-5').color, '#22aa66');
assert.equal(resolvePlayerArcana(player, summary, 'stage-2').name, 'Alchemic Sharpshooter');
assert.equal(resolvePlayerArcana(player, summary, 'mixed-group').status, 'mixed');
assert.equal(resolvePlayerArcanaForIds(player, ['stage5-enemy']).name, 'Sacred Guardian');
assert.equal(resolvePlayerArcanaWindow(player, 200, 220).name, 'Sacred Guardian');
assert.equal(resolvePlayerArcanaWindow(player, 100, 110).name, 'Alchemic Sharpshooter');
assert.equal(resolvePlayerArcanaWindow(player, 300, 320).status, 'unknown');
assert.equal(resolvePlayerArcana({ ...player, damageByTarget: {} }, summary, 'stage-5').status, 'unknown');
assert.equal(resolvePlayerArcana({ ...player, skillActivations: [{ atMs: 1000*205, skillId: guardianId, targetId: 'stage5-enemy' }], damageByTarget: {} }, summary, 'stage-5').name, 'Sacred Guardian');
const identities = resolvePlayerArcana(player, summary).identities;
assert.deepEqual(Array.from(identities.map(a => a.name)), ['Alchemic Sharpshooter', 'Sacred Guardian']);
const two = arcanaBarBackground(identities);
assert(two.startsWith('linear-gradient(to right, '));
assert(two.includes(`${identities[0].color} 0.000%`));
assert(two.includes(`${identities[1].color} 100.000%`));
const third = all.find(a => !['Alchemic Sharpshooter', 'Sacred Guardian'].includes(a.arcana_name));
const three = arcanaBarBackground([...identities, { name: third.arcana_name, color: third.color, icon: third.icon }]);
assert(three.includes(`${identities[1].color} 50.000%`));
assert(!three.includes(`${identities[0].color} 33.333%`));
assert(arcanaBarBackground([identities[0]]).includes(identities[0].color));
// The first observation may come from a later-listed skill in the aggregate map.
const reversed = { ...player, damageEvents: [...player.damageEvents].reverse().map((e, i) => ({ ...e, atMs: i === 0 ? 50_000 : 150_000 })) };
assert.equal(resolvePlayerArcana(reversed, summary).names[0], 'Sacred Guardian');
console.log('Arcana scope, customized colors and gradient regression passed');
