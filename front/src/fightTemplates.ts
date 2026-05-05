// src/fightTemplates.ts

import { reactive } from "vue";
import { v4 as uuidv4 } from "uuid"; // You'll need to install uuid: npm install uuid

const CACHE_KEY = "midir_fight_templates";

export interface FightGroup {
  id: string;
  name: string;
  raceIds: number[];
}

export interface FightTemplate {
  id: string;
  name: string;
  groups: FightGroup[];
}

// Use a reactive object to hold all templates, keyed by their ID
export const fightTemplates = reactive<Record<string, FightTemplate>>({});

/**
 * Loads templates from localStorage into the reactive store.
 */
export function initFightTemplates(): void {
  try {
    const stored = localStorage.getItem(CACHE_KEY);
    if (stored && Object.keys(JSON.parse(stored)).length > 0) {
      Object.assign(fightTemplates, JSON.parse(stored));
    } else {
      // Create a default empty template for first-time users
      const defaultId = uuidv4();
      fightTemplates[defaultId] = {
        id: defaultId,
        name: "New Fight Template",
        groups: [
          { id: uuidv4(), name: "Boss", raceIds: [] },
          { id: uuidv4(), name: "Adds", raceIds: [] },
        ],
      };
    }
  } catch (e) {
    console.error("Failed to load fight templates.", e);
  }
}

/**
 * Saves the current state of all templates to localStorage.
 */
export function saveFightTemplates(): void {
  try {
    localStorage.setItem(CACHE_KEY, JSON.stringify(fightTemplates));
  } catch (e) {
    console.error("Failed to save fight templates.", e);
  }
}

// Helper functions to make managing templates easier from the component
export function createNewTemplate() {
  const newId = uuidv4();
  fightTemplates[newId] = {
    id: newId,
    name: `New Template ${Object.keys(fightTemplates).length + 1}`,
    groups: [{ id: uuidv4(), name: "Boss", raceIds: [] }],
  };
  saveFightTemplates();
  return newId;
}

export function deleteTemplate(id: string) {
  delete fightTemplates[id];
  saveFightTemplates();
}

export function addGroupToTemplate(templateId: string) {
  const template = fightTemplates[templateId];
  if (template) {
    template.groups.push({
      id: uuidv4(),
      name: `New Group ${template.groups.length + 1}`,
      raceIds: [],
    });
    saveFightTemplates();
  }
}

export function removeGroupFromTemplate(templateId: string, groupId: string) {
  const template = fightTemplates[templateId];
  if (template) {
    template.groups = template.groups.filter((g) => g.id !== groupId);
    saveFightTemplates();
  }
}
