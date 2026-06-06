import { defineConfig } from "astro/config";
import react from "@astrojs/react";
import sitemap from "@astrojs/sitemap";
import tailwindcss from "@tailwindcss/vite";

// NOTE: `site` is a placeholder until the production domain is confirmed.
// It only affects canonical URLs / sitemap, not the build output.
// https://astro.build/config
export default defineConfig({
  site: "https://mojify.dev",
  integrations: [react(), sitemap()],
  vite: {
    plugins: [tailwindcss()],
  },
});
