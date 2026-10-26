// Reject known-unpatched Node.js releases when building Midir.
// Update this list whenever Node.js publishes new security fixes.
const supportedMinimums = new Map([
  [26, [26, 12, 1]], // 2026-10-21 security release, latest Current
  [24, [24, 22, 1]], // 2026-10-21 security release, latest LTS
]);

function atLeast(actual, required) {
  for (let i = 0; i < required.length; i++) {
    if (actual[i] !== required[i]) return actual[i] > required[i];
  }
  return true;
}

const match = /^v(\d+)\.(\d+)\.(\d+)$/.exec(process.version);
const current = match && match.slice(1).map(Number);
const minimum = current && supportedMinimums.get(current[0]);
if (!current || !minimum || !atLeast(current, minimum)) {
  console.error(`Unsupported or unpatched Node.js ${process.version}. Use Node.js 26.12.1 or later in the 26.x line, or 24.22.1 or later in the 24.x LTS line.`);
  process.exit(1);
}
console.log(`Node.js ${process.version} meets Midir's security baseline.`);
