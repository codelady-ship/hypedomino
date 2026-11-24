# 0001. Client: Unity 6 LTS, URP 2D, IL2CPP

Status: Accepted; the renderer choice was changed to the Universal Renderer by 0009. 0009 is itself superseded (no 3D characters, 2026-11-12), but the client keeps the Universal Renderer until a measured change says otherwise
Date: 2026-10-25
Issue: #10

## Context
The client is a hyperactive, effect-heavy mobile game: particles, animated avatars, frames and emoji, many cosmetic variants. Budgets from CLAUDE.md: stable 60 fps on mid-range Android, 120 fps option on capable devices, cold start under 3 s, match scene under 150 draw calls, 0 B GC allocation per frame during a match, install under 150 MB. The client only renders and sends intents (ADR 0003), so it holds no game rules, but it carries all visual and audio weight.

## Decision
- Unity 6 LTS, Universal Render Pipeline in 2D renderer mode, C#.
- Android and iOS builds use IL2CPP (AOT). Mono is not shipped.
- Match-scene code follows the zero-allocation rule: pooled objects, no LINQ or closures in Update, no per-frame string building.
- The scripting layer only renders server state and sends intents. No rules code lives in the client.
- The engine version is pinned to one Unity 6 LTS release; upgrades are their own PR with a measurement.

## Alternatives considered and why rejected
- Godot 4: lighter engine and smaller base install, open source. Rejected because the 2D skeletal and VFX authoring pipeline, mobile profiling tools (frame debugger, memory profiler, GPU tooling), remote content management comparable to Addressables, store SDKs for purchases, ads and attribution, and the hiring pool are all weaker or less proven at this scale. C# support exists, but the mobile export path and third-party SDK ecosystem carry more integration risk.
- Flutter with Flame: strong for UI-heavy apps and fast iteration. Rejected because Flame is a thin game layer with no built-in skeletal animation, particle authoring or atlas pipeline of the required depth, and Dart GC under heavy effect load makes the 0 B/frame budget harder to prove. Shaders and VFX would be built by hand.
- Native (Kotlin and Swift, own renderer): best raw control. Rejected because it doubles the client codebase and the art pipeline for two platforms, with no engine tooling for animation or VFX, for a performance gain that Unity with disciplined pooling is expected to meet.
- Unity built-in render pipeline or the 3D URP renderer: rejected; the 2D renderer provides 2D lights, sprite batching and lower overdraw cost for a sprite-based game.
- Unity Mono scripting backend: rejected; slower than IL2CPP and not usable for iOS release builds.

## Consequences
- Easier: skeletal 2D rigs, sprite atlases, particle systems, Shader Graph, Profiler and Memory Profiler, Addressables (ADR 0002), mature purchase, ad and analytics SDKs, a large hiring pool.
- Harder: the Unity runtime baseline adds to install size and cold start; both are measured, not assumed (ADR 0008). IL2CPP builds are slower to produce in CI. Unity licensing terms must be tracked.
- GC discipline is a review rule: a match-scene change that allocates is a defect, caught by the allocation check in ADR 0008.
