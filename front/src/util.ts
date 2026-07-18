import { customRef } from "vue";
import { nameColorSaturation, nameColorLightness, nameColorSeed, customClassColors } from "@/store";

/**
 * Generates a vibrant, consistent color from a string using the HSL color model.
 * This function employs an FNV-1a hashing algorithm for better initial distribution
 * and then leverages the Golden Ratio Conjugate to spread hues more evenly across
 * the color wheel, aiming for greater perceptual distinctness between generated colors.
 *
 * @param name The input string (e.g., a player's name, category label).
 * @returns An HSL color string (e.g., 'hsl(150, 85%, 60%)').
 */
export function getMabiNameColor(name: string): string {
  if (!name?.length) {
    return "hsl(0, 0%, 50%)";
  }

  // --- FNV-1a Hash ---
  let hash = 0x811c9dc5;
  const FNV_PRIME = 0x01000193;

  // Mix in the seed first
  const seed = nameColorSeed.value || "RY34CW"; // Default seed if empty
  for (let i = 0; i < seed.length; i++) {
      hash ^= seed.charCodeAt(i);
      hash = (hash * FNV_PRIME) >>> 0;
  }

  // Then mix in the name
  for (let i = 0; i < name.length; i++) {
    hash ^= name.charCodeAt(i);
    hash = (hash * FNV_PRIME) >>> 0;
  }

  // --- Golden Ratio Dist ---
  const GOLDEN_RATIO_CONJUGATE = 0.618033988749895;
  const normalizedValue = (hash * GOLDEN_RATIO_CONJUGATE) % 1;

  // --- Weighted Tue Mapping ---
  // Ranges: [Start, End, Weight]
  // 0-40: Red/Orange (Low weight to reduce neon red)
  // 40-160: Yellow/Green/Neon-Green (Excluded) -> Avoids the "Blinding" Green
  // 160-280: Teal/Blue/Purple (Very High Priority) -> Safe cool colors
  // 280-360: Magenta/Pink (Normal)
  const ranges = [
    { start: 0,   end: 40,  weight: 2 },
    { start: 60,  end: 90, weight: 0.5 },   // Killing the Greens/Yellows
    { start: 160, end: 280, weight: 2 },   // Massive bias to Blue/Purple
    { start: 280, end: 360, weight: 0.7 },
  ];

  let totalWeight = 0;
  for (const r of ranges) {
    totalWeight += (r.end - r.start) * r.weight;
  }

  let target = normalizedValue * totalWeight;
  let hue = 0;

  for (const r of ranges) {
    const rangeSize = r.end - r.start;
    const rangeEffectiveSize = rangeSize * r.weight;

    if (target < rangeEffectiveSize) {
      // Found the range
      const percentInRange = target / rangeEffectiveSize;
      hue = r.start + (percentInRange * rangeSize);
      break;
    } else {
      target -= rangeEffectiveSize;
    }
  }

  // --- Saturation & Lightness ---
  // Use user-configured ranges from store
  const [satMin, satMax] = nameColorSaturation.value;
  const [lightMin, lightMax] = nameColorLightness.value;

  const satVariance = Math.max(1, satMax - satMin);
  const lightVariance = Math.max(1, lightMax - lightMin);

  const saturation = satMin + (hash % satVariance);
  const lightness = lightMin + (Math.abs(hash >> 4) % lightVariance);

  return `hsl(${Math.floor(hue)}, ${saturation}%, ${lightness}%)`;
}

export interface IUpdateCallback {
  setUpdateCallback(track: () => void, trigger: () => void): void;
}

export function CustomReactive<T extends IUpdateCallback>(value: T): T {
  const state = customRef<T>((track, trigger) => {
    value.setUpdateCallback(track, trigger);

    return {
      get() {
        track();
        return value;
      },
      set(newValue: T) {
        value = newValue;
        trigger();
      },
    };
  });

  return state.value;
}

/**
 * Resolves the color for a given class/talent name.
 * Checks for a user-customized color first, then falls back to the default color,
 * and finally to a neutral gray.
 *
 * @param className The name of the class/talent/arcana.
 * @param defaultColor The default color (usually from talents.json).
 * @returns A hex, rgb, or hsl color string.
 */
export function getClassColor(className?: string, defaultColor?: string): string {
  if (!className) {
    return defaultColor || "hsl(0, 0%, 50%)";
  }
  if (customClassColors.value && customClassColors.value[className]) {
    return customClassColors.value[className];
  }
  return defaultColor || "hsl(0, 0%, 50%)";
}
