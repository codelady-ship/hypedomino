import assert from "node:assert/strict";
import { test } from "node:test";
import { assetStoreOrigin, contentSecurityPolicy, newNonce, staticSecurityHeaders } from "./headers.ts";

function directives(policy: string): Map<string, string> {
  return new Map(policy.split("; ").map((directive) => {
    const [name, ...sources] = directive.split(" ");
    return [name, sources.join(" ")];
  }));
}

test("scripts run only with the request nonce and never inline or from eval in production", () => {
  const policy = directives(contentSecurityPolicy("abc123", false));
  assert.equal(policy.get("script-src"), "'self' 'nonce-abc123' 'strict-dynamic'");
  assert.equal(policy.get("default-src"), "'self'");
  assert.doesNotMatch(contentSecurityPolicy("abc123", false), /unsafe-inline|unsafe-eval/);
});

test("development adds eval for React debugging and nothing else", () => {
  assert.equal(directives(contentSecurityPolicy("n", true)).get("script-src"), "'self' 'nonce-n' 'strict-dynamic' 'unsafe-eval'");
  assert.doesNotMatch(contentSecurityPolicy("n", true), /unsafe-inline/);
});

test("the panel cannot be framed, post elsewhere, change its base or load plugins", () => {
  const policy = directives(contentSecurityPolicy("n", false));
  assert.equal(policy.get("frame-ancestors"), "'none'");
  assert.equal(policy.get("form-action"), "'self'");
  assert.equal(policy.get("base-uri"), "'none'");
  assert.equal(policy.get("object-src"), "'none'");
});

test("Google sign-in may load its button frame, style and calls and nothing broader", () => {
  const policy = directives(contentSecurityPolicy("n", false));
  assert.equal(policy.get("frame-src"), "https://accounts.google.com/gsi/");
  assert.equal(policy.get("connect-src"), "'self' https://accounts.google.com/gsi/");
  assert.equal(policy.get("style-src"), "'self' 'nonce-n' https://accounts.google.com/gsi/style");
});

test("the asset store may receive uploads and show images and nothing broader", () => {
  const policy = directives(contentSecurityPolicy("n", false, assetStoreOrigin("https://store.example/bucket/ignored")));
  assert.equal(policy.get("connect-src"), "'self' https://accounts.google.com/gsi/ https://store.example");
  assert.equal(policy.get("img-src"), "'self' data: https://store.example");
  assert.equal(policy.get("script-src"), "'self' 'nonce-n' 'strict-dynamic'");
  assert.equal(policy.get("default-src"), "'self'");
});

test("a missing or malformed asset origin adds nothing to the policy", () => {
  for (const configured of [undefined, "", "store.example", "https://"]) {
    assert.equal(contentSecurityPolicy("n", false, assetStoreOrigin(configured)), contentSecurityPolicy("n", false), String(configured));
  }
});

test("every nonce is new and carries 128 bits", () => {
  const nonces = new Set(Array.from({ length: 100 }, newNonce));
  assert.equal(nonces.size, 100);
  for (const nonce of nonces) assert.equal(Buffer.from(nonce, "base64").length, 16);
});

test("HSTS is sent only in production; the other headers always", () => {
  const names = (production: boolean) => staticSecurityHeaders(production).map((header) => header.key);
  assert.ok(names(true).includes("Strict-Transport-Security"));
  assert.ok(!names(false).includes("Strict-Transport-Security"));
  for (const name of ["X-Content-Type-Options", "Referrer-Policy", "X-Frame-Options", "Cross-Origin-Opener-Policy", "Permissions-Policy"]) {
    assert.ok(names(false).includes(name), name);
  }
});
