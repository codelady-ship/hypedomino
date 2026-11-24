# 0002. Asset delivery: Addressables, CDN, ASTC

Status: Accepted
Date: 2026-10-25
Issue: #10

## Context
Install must stay under 150 MB while the game ships many cosmetic assets, each with a rarity, a status colour and a manifest entry managed from the admin panel (CLAUDE.md). Animated assets (avatars, frames, emoji) are expensive in bytes and GPU time. Low-end devices must still render something correct.

## Decision
- The install package holds only the boot path, the match scene essentials and the default tile set. Everything else is an Addressables group fetched on demand from a CDN.
- Textures use ASTC on both platforms. Block size is chosen per asset class in the asset spec; it is not fixed here.
- Sprites are packed into atlases per group so a match scene batches within the 150 draw-call budget.
- Every animated asset has a static fallback (a single sprite) in the same group. The client falls back when the asset is not yet downloaded, when download fails, or when the device tier disables animation.
- The catalog is versioned; the client learns the active catalog version from the server after the handshake (ADR 0004). Content URLs come from server config, not constants.
- The CDN serves immutable, content-hashed files. Long cache lifetimes are therefore safe and a rollback is a catalog change.
- Manifest entries are managed from the admin panel (ADR 0006). Source art in `assets-src/` is never shipped directly.

## Alternatives considered and why rejected
- Everything in the install package: simplest, but breaks the 150 MB budget as the cosmetic catalog grows and forces a store release for every asset.
- Unity AssetBundles managed by hand: Addressables is built on bundles and adds dependency resolution, labels, remote catalogs and reference counting. Hand-managing them re-implements that.
- A per-platform texture format (ETC2 on Android): rejected; one format keeps pipelines and quality review uniform. Devices without ASTC support are outside the target.
- Streaming assets from the game server: rejected; it competes with real-time traffic and gets no CDN caching.
- Video instead of sprite animation: rejected for in-match effects, because of decoding cost and no per-layer composition. It may be used for non-interactive screens.

## Consequences
- Easier: install size stays flat as content grows; art ships without a store release; rollback is a catalog switch.
- Harder: first use of a new asset needs a download state and a fallback path; offline start must work with the install-time set only. Group design affects load time and memory, so it is reviewed per asset spec.
- Download size per group and the cold-start effect are measured per ADR 0008.
- The client side of this decision is `docs/content-delivery.md`.
