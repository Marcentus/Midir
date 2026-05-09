import { createApp } from "vue";
import App from "@/App.vue";
import * as store from "@/store";

// mdi
import "@mdi/font/css/materialdesignicons.css";

// vuetify
import "vuetify/styles";
import { createVuetify } from "vuetify";

console.log("1. Starting main.ts");

const vuetify = createVuetify({
  theme: {
    defaultTheme: "midirTheme",
    themes: {
      midirTheme: {
        dark: true,
        colors: {
          background: "#0e1015",
          surface: "#171b24",
          primary: "#818cf8", // Indigo 400
          secondary: "#c084fc", // Purple 400
          error: "#f87171",
          info: "#60a5fa",
          success: "#4ade80",
          warning: "#fbbf24",
        },
      },
    },
  },
});

console.log("2. Creating Vue app instance");
const app = createApp(App);

// --- START: REPLACED LOGIC ---
// Register global variables explicitly for better reliability
console.log("3. About to provide store variables");
app.provide("loadingCount", store.loadingCount);
app.provide("isLoading", store.isLoading);
app.provide("region", store.region);
app.provide("lang", store.lang);
app.provide("regionList", store.regionList);
app.provide("raceNameMap", store.raceNameMap);
app.provide("skillNameMap", store.skillNameMap);
app.provide("condNameMap", store.condNameMap);
app.provide("itemNameMap", store.itemNameMap);
app.provide("appEvent", store.appEvent);
app.provide("playerNameCache", store.playerNameCache);
app.provide("sessions", store.sessions);
app.provide("activeSessionId", store.activeSessionId);
app.provide("isNavDrawerOpen", store.isNavDrawerOpen);
app.provide("fightSummary", store.fightSummary);
store.socket.connect();
console.log("4. Finished providing store variables");
// --- END: REPLACED LOGIC ---

app.config.errorHandler = (err) => {
  alert(err);
  console.error(err);
};

console.log("5. About to mount Vue app");
app.use(vuetify).mount("#app");
console.log("6. Vue app mount command issued");
