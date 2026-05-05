import { playerNameCache } from "@/store";
import { PlayerCacheInfo } from "./types";

const CACHE_KEY = "midir_player_name_cache";

/**
 * Loads the player name cache from localStorage into the reactive store.
 */
export function initPlayerCache(): void {
  try {
    const storedCache = localStorage.getItem(CACHE_KEY);
    if (storedCache) {
      Object.assign(playerNameCache, JSON.parse(storedCache));
      console.log(
        `Player name cache loaded with ${Object.keys(playerNameCache).length} entries.`
      );
    }
  } catch (e) {
    console.error("Failed to load player name cache from localStorage.", e);
    localStorage.removeItem(CACHE_KEY);
  }
}

/**
 * Saves a player's ID and full info to the in-memory cache and localStorage.
 * @param info The player's information object.
 */
export function savePlayerToCache(info: PlayerCacheInfo): void {
  const { id, name } = info;
  if (!id || !name || name.startsWith("unknown:")) {
    return;
  }

  // Only write to localStorage if data has changed to improve performance
  if (JSON.stringify(playerNameCache[id]) !== JSON.stringify(info)) {
    playerNameCache[id] = info;
    try {
      localStorage.setItem(CACHE_KEY, JSON.stringify(playerNameCache));
    } catch (e) {
      console.error("Failed to save player name cache to localStorage.", e);
    }
  }
}
