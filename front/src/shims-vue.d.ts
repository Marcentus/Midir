import { ComputedRef, Ref } from "vue";
import { FightSummary } from "./protocols";

declare module "@vue/runtime-core" {
  // --- KEPT ---
  function inject(key: "isLoading"): ComputedRef<boolean>;
  function inject(key: "region"): Ref<string>;
  function inject(key: "lang"): Ref<string>;
  function inject(key: "regionList"): Ref<string[]>;
  function inject(key: "raceNameMap"): Ref<Record<number, string>>;
  function inject(key: "itemNameMap"): Ref<Record<number, string>>;
  function inject(key: "appEvent"): Ref<EventTarget>;
  function inject(key: "loadingCount"): Ref<number>;
  function inject(key: "fightSummary"): FightSummary;

  // --- CORRECTED TYPES ---
  // This now correctly defines the shape of the data being injected.
  function inject(
    key: "skillNameMap"
  ): Ref<Record<number, { name: string; iconUrl?: string }>>;
  function inject(
    key: "condNameMap"
  ): Ref<Record<number, { name: string; iconUrl?: string }>>;
}
