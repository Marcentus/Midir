<template>
  <v-navigation-drawer
    rail
    permanent
    location="left"
    name="app-navigation"
  >
    <v-list density="compact" nav>
      <v-list-item
        prepend-icon="mdi-sword"
        title="Damage Meter"
        value="dps"
        :active="activeTool === 'dps'"
        @click="activeTool = 'dps'"
      ></v-list-item>

      <v-spacer></v-spacer>

      <v-divider class="my-2"></v-divider>

      <v-list-item
        prepend-icon="mdi-cog"
        title="Settings"
        value="settings"
        :active="activeTool === 'settings'"
        @click="activeTool = 'settings'"
      ></v-list-item>
    </v-list>
  </v-navigation-drawer>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import { activeTool } from "@/store";

export default defineComponent({
  name: "Navigation",
  setup() {
    return {
      activeTool,
    };
  },
});
</script>
