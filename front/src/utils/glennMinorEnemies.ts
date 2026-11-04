/** Glenn Bearna's ten named minor-enemy families, using exact race IDs.
 * Kept separate from Unfinished Reverie (7585, 7586) and Winter Tyrant (7587).
 * Covers both the older 75xx and observed 126xxx races, without similarly
 * named unused variants. Never infer individual run boundaries from mob IDs.
 */
export const GLENN_MINOR_RACE_IDS = new Set<number>([
  7500, 7501, 7502, 7503, 7504, 7507, 7508, 7509, 7512,
  7514, 7515, 7516, 7522, 7528, 7529, 7530, 7531, 7532,
  7535, 7536, 7537, 7540, 7542, 7543, 7544, 7545, 7555,
  7557, 7559, 7562, 7563, 7564, 7567, 7569, 7571, 7572,
  7573, 7574, 126001, 126002, 126003, 126004, 126005, 126006, 126007,
  126008, 126009, 126010, 126011, 126012, 126013, 126014, 126015, 126016,
  126017, 7539, 7583, // Icicle Sprites are ordinary minor enemies, not Cailleach mechanics.
]);
export const GLENN_MINOR_GROUP_ID = 'group_glenn_minor_enemies';
export function isGlennMinorRace(raceId: unknown): boolean {
  return GLENN_MINOR_RACE_IDS.has(Number(raceId));
}
