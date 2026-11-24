# 0012. Animated 2D cosmetics: sprite sheet loops

Status: Proposed. The format follows from constraints checked below. A first device measurement is recorded below; the measurement on the Mid reference device is still owed before this ADR is accepted.
Date: 2026-11-06
Issue: #93

This ADR covers the animated cosmetics: avatars, frames, emoji and table layers. The game has no characters (ADR 0009 is superseded).

## Context

Animated avatars appear in lists, the shop and the profile as well as at the seats. They have to work inside a Canvas, including under scrolling and masking. Each avatar has a size limit of 300 KB (#171, a loop of 1 to 2 s), and the install limit in ADR 0002 keeps the art out of the install package. An item's art goes through the asset pipeline of #135 and `docs/asset-workflow.md` 5.2, where After Effects and Blender already deliver PNG sequences. `LoopArt` already plays emoji from an array of sprites.

## Decision

- An animated 2D item is a loop of whole frames in one atlas in the item's group (one atlas per loop, so one per table layer; `docs/cosmetics.md` 8.2), compressed with ASTC, without mipmaps, with its static sprite in the same group.
- Frames keep the class canvas and pivot untrimmed, so swapping a frame or the static shows no jump (`docs/art-direction.md` section 7).
- `AnimatedImage` plays the loop on a UI `Image`. It sets the sprite only when the frame changes and allocates nothing while playing.
- It shows the static on the low effect level, with reduce motion, for an item with no loop, and when the screen's budget is spent (`docs/cosmetics.md` 8.3). A row that is disabled frees its slot and rests on the static.
- Each screen holds one `AnimationBudget`. Its initial limit is `maxAnimatedCosmetics` in `docs/design-tokens.json`: 0, 10 and 10 for low, medium and high. The 10 comes from the acceptance criterion of #93, not from a measurement, and the measurement below confirms or changes it by PR.
- A screen needs no nested Canvas for its animated images. The measurement below found no gain from one on a screen of 50 elements. A screen with far more elements measures both ways before adding one.

### Size rule

ASTC stores 16 bytes per block, so the texture size follows from the atlas and the block size. These are before bundle compression and include no bundle overhead:

| Atlas | Block | Bytes | Fits the 300 KB of #171 |
|---|---|---|---|
| 1024 x 1024 | 6x6 | 467,856 | No |
| 768 x 768 | 6x6 | 262,144 | Yes |
| 1024 x 1024 | 8x8 | 262,144 | Yes |

As an illustration, 768 x 768 holds 16 frames of 192 x 192. The canvas per class, the block size per class and the frame rate stay open (`docs/asset-workflow.md` O-5, O-6). 8x8 may blur the fine ornament and the pip edges of `docs/art-direction.md` section 4; that needs a look at real art on the reference panel.

## Alternatives considered and why rejected

None of these was measured on a device. Each is rejected on a constraint.

- Unity 2D Animation rig: `SpriteSkin` carries `[RequireComponent(typeof(SpriteRenderer))]` in version 10.1 of the package, so it does not drive a UI `Image`. Avatars in a Canvas could not use it without a separate world-space path for lists and masks. It would also add a package the project does not have.
- Spine: it has a UI component, but needs a third-party runtime and a commercial licence, which is a purchase for the owner. Price and licence terms were not checked. It would be reconsidered if sprite sheets fail the size rule on real art.
- Lottie: no first-party Unity player, so it adds a third-party runtime that rasterises vectors at run time. Painted art from the pipeline in `docs/asset-workflow.md` is not vector art.
- Video: already rejected for in-match effects in ADR 0002.

## First measurement

Run on 2026-11-07 with `AnimationBench` (workflow `Animation bench APK`, app id `com.hypedomino.animationbench`). The screen holds 10 avatars of 300 x 300 units playing 16 frames at 12 fps from generated 768 x 768 ASTC 6x6 atlases, over 40 static panels. Each phase ran 2 windows of 1200 frames in each of 3 cycles, so 7200 frames per phase, after 5 s to settle. The ranges cover its 6 windows.

Device: Samsung Galaxy A36 5G, Snapdragon 6 Gen 3, 8 GB, Android 16, 1080 x 2340 at 60 Hz, Vulkan. It is Mid class under the class rule of `docs/performance.md` section 1 and is not in the matrix.

| Phase | Work p50 (ms) | Work p95 (ms) | Main p95 (ms) | Render p95 (ms) | GPU p95 (ms) | Missed vsyncs | GC frames | Batches | SetPass | Memory (MB) |
|---|---|---|---|---|---|---|---|---|---|---|
| `still`: statics, nothing playing | 5.01 to 5.05 | 5.07 to 5.09 | 3.83 to 3.93 | 1.25 to 1.32 | 5.07 to 5.09 | 0% | 0 | 13 | 3 | 261 to 270 |
| `nested`: 10 playing, own Canvas | 5.04 to 5.05 | 5.08 to 5.10 | 4.23 to 4.33 | 1.25 to 1.32 | 5.08 to 5.10 | 0% | 0 | 13 | 4 | 266 to 271 |
| `shared`: 10 playing, screen Canvas | 5.04 to 5.05 | 5.08 to 5.09 | 4.24 to 4.33 | 1.25 to 1.32 | 5.08 to 5.09 | 0% | 0 | 13 | 3 | 269 to 271 |

Work is the largest of the main thread, render thread and GPU times of a frame, against 11.0 and 14.0 ms in `docs/performance.md` section 3.

What this shows:

- Playing 10 avatars costs about 0.4 ms of main thread at p95 and allocates nothing.
- The nested Canvas saved no time here and added one SetPass call, so it is not a rule.
- GPU time does not move between phases, as expected: every phase draws the same pixels. GPU time is the largest part of work in all three, and why it sits so flat was not looked into.

What this does not show:

- The Mid criterion. The device is outside the matrix and has a different SoC and GPU from the Mid reference.
- Section 2 conditions. This was a development build, on AC power (battery 87 to 90%, 32.7 to 33.0 C), with adb over Wi-Fi. There was no replay and no match scene.
- Memory of the art. All 10 atlases stay loaded in every phase, so the column cannot show their cost. By the size rule their textures are about 2.6 MB.
- A real avatar. The art is generated, and no bundle was built.

## Measurement owed

Before this ADR is accepted, on a Mid reference device of `docs/performance.md` section 1 under its section 2 conditions:

- 10 animated avatars on one screen hold 60 fps (acceptance criterion of #93), in a release build for the fps numbers and a development build for batches, SetPass calls, GC and memory.
- The bundle of one real avatar, against the 300 KB of #171.

The owner may accept a device outside the matrix instead, by PR to this file. Until then the result above shows headroom and does not close the criterion.

## Consequences

- Easier: one player for avatars, frames, emoji and table layers; no new package; the admin can measure frames and atlas size from the file (#135); art needs no Unity rig.
- Harder: every frame is texture memory, so bytes grow with length and size, and a smooth long loop does not fit the limit. Reuse of parts between frames is not possible as it would be with a rig.
- `docs/asset-workflow.md` 5.2 drops the Unity 2D rig path for these types.
