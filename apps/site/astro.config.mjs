import { defineConfig } from "astro/config";
import react from "@astrojs/react";
import sitemap from "@astrojs/sitemap";
import tailwindcss from "@tailwindcss/vite";

// `site` is the single source of truth for the production origin: canonical
// URLs, og:url, og:image, sitemap, and robots.txt all derive from it. Set to the
// likely host (mojify.jass.gg) — change this one line if that's not final.
// https://astro.build/config
export default defineConfig({
  site: "https://mojify.jass.gg",
  integrations: [react(), sitemap()],
  vite: {
    plugins: [tailwindcss()],
  },
});
