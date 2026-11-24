import assert from "node:assert/strict";
import { readdirSync, readFileSync } from "node:fs";
import { test } from "node:test";
import { screens } from "./navigation.ts";

const appDir = new URL("../app/", import.meta.url);

const detailObjects: Readonly<Record<string, string>> = {
  "/matches/[id]": "players",
  "/players/[id]": "players",
  "/promo/[id]": "promo_codes",
};

const signedInOnly = new Set(["/"]);

function pageRoutes(): string[] {
  return readdirSync(appDir, { recursive: true, encoding: "utf8" })
    .map((file) => file.replaceAll("\\", "/"))
    .filter((file) => file === "page.tsx" || file.endsWith("/page.tsx"))
    .map((file) => `/${file.slice(0, -"page.tsx".length)}`.replace(/(.)\/$/, "$1"))
    .sort();
}

function pageBody(route: string): string {
  const source = readFileSync(new URL(`.${route === "/" ? "" : route}/page.tsx`, appDir), "utf8");
  if (source.includes("<KeyedConfigPage")) {
    const shared = readFileSync(new URL("config/KeyedConfigPage.tsx", appDir), "utf8");
    return shared.slice(shared.indexOf("export async function KeyedConfigPage"));
  }
  return source.slice(source.indexOf("export default"));
}

function objectOf(route: string): string | null | undefined {
  const screen = screens.find((candidate) => candidate.href === route);
  return screen ? screen.object : detailObjects[route];
}

function firstApiCall(body: string): number {
  const calls = [/adminApi\w*[<(]/, /\bload\w*\(/].map((pattern) => body.search(pattern)).filter((at) => at >= 0);
  return calls.length === 0 ? body.length : Math.min(...calls);
}

function guardAt(body: string, pattern: RegExp): number {
  const at = body.search(pattern);
  assert.ok(at >= 0, `missing guard ${pattern}`);
  return at;
}

const signInGuard = /const admin = await currentAdmin\(\);\s*if \(!admin\) \{\s*return [^;]*auth\.required/;

function viewGuard(object: string): RegExp {
  return new RegExp(`if \\(!canView\\(admin\\.permissions, "${object}"\\)\\) \\{\\s*return [^;]*auth\\.forbidden`);
}

test("every page is a navigation screen, a detail page with a known object or the signed-in home", () => {
  const unclassified = pageRoutes().filter((route) => objectOf(route) === undefined && !signedInOnly.has(route));
  assert.deepEqual(unclassified, []);
  const missing = screens.map((screen) => screen.href).filter((href) => !pageRoutes().includes(href));
  assert.deepEqual(missing, []);
});

test("every page refuses a visitor without a session before it calls the admin API", () => {
  for (const route of pageRoutes()) {
    const body = pageBody(route);
    assert.ok(guardAt(body, signInGuard) < firstApiCall(body), route);
  }
});

test("every page with an object refuses a role without view on it before it calls the admin API", () => {
  for (const route of pageRoutes()) {
    const object = objectOf(route);
    if (!object) {
      continue;
    }
    const body = pageBody(route);
    assert.ok(guardAt(body, viewGuard(object)) < firstApiCall(body), route);
  }
});
