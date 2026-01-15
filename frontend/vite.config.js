import { defineConfig } from "vite";
import { svelte } from "@sveltejs/vite-plugin-svelte";
import tailwindcss from "@tailwindcss/vite";

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [tailwindcss(), svelte()],
  server: {
    proxy: {
      "/api": "http://localhost:8080",
      "/floppapi": "http://localhost:8080",
      "/macka": "http://localhost:8080",
      
    },
  },
});
