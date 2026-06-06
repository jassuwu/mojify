import type { APIRoute } from "astro";

// Generated from `site` in astro.config so the domain lives in exactly one place.
export const GET: APIRoute = ({ site }) => {
  const lines = ["User-agent: *", "Allow: /"];
  if (site) lines.push("", `Sitemap: ${new URL("sitemap-index.xml", site).href}`);
  return new Response(lines.join("\n") + "\n", {
    headers: { "Content-Type": "text/plain; charset=utf-8" },
  });
};
