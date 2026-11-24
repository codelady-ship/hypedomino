# Asset workflow

Status: proposal; the owner approves it by merging. How an asset goes from an AI-assisted concept to a published cosmetic: prompt templates and consistency, rights record, naming and folders in `assets-src/`, the clean-up and animation steps, and the sample tile skin walkthrough. It defines no colour, size, duration or budget. Look and rules come from `docs/art-direction.md`, item data and addresses from `docs/cosmetics.md`, rarity from `docs/rarity.md`, delivery from ADR 0002, budgets from `docs/performance.md` and `docs/ui-motion.md` section 7, upload and publish from `docs/admin.md` section 5.2, the content checklist from `docs/localization.md` section 6. `node tools/asset-naming/asset-naming.mjs` checks `assets-src/` against section 4; CI runs its tests through `npm run test:tools`.

## 1. Stages

| Id | Stage | Tool | Output | Gate |
|---|---|---|---|---|
| W-1 | Concept | ChatGPT image generation in the author's own account (O-1): in the browser, or through the Codex CLI built-in image tool signed in with that account (2.5) | Raw images in `concept/`, prompt log in `provenance.md` | Section 2 templates used; section 3 record started |
| W-2 | Concept review | Reference board (`docs/art-direction.md` section 9) | One approved concept sheet per item | Silhouette, smallest-size and palette checks of art-direction sections 2, 3, 5; similarity check (section 3) |
| W-3 | Clean-up | Illustrator, Photoshop | Layered master in `work/` | Section 5.1 |
| W-4 | Animation | After Effects or Blender to a sprite sheet (ADR 0012) | Loop or layers in `work/` | Section 5.2 |
| W-5 | Export | Section 5.3 | `export/` mirrors the Addressables address (section 4.2) | `node tools/asset-naming/asset-naming.mjs` passes |
| W-6 | Pack | Unity Addressables, Sprite Atlas, ASTC | Item group and atlas (`docs/cosmetics.md` 8.2) | Section 5.4 |
| W-7 | Upload and review | Admin assets (`docs/admin.md` 5.2) | Asset records, item draft | Self-check by the author, cultural review by another account (`docs/localization.md` 6.1) |
| W-8 | Publish | Admin change set | Manifest entry | V-1 to V-19 (`docs/cosmetics.md` section 9) |

Raw AI output never reaches W-5. Every exported pixel has passed W-3.

## 2. Prompts and consistency

### 2.1 Rules

- English prompts. The style key (2.2) is pasted verbatim at the start of every prompt; it is never paraphrased or shortened.
- Never name an artist, studio, game, film, brand, mascot or real person, and never ask for a look "like" one (CLAUDE.md, `docs/localization.md` 6, Likeness and brands). Describe shapes, proportions and materials instead.
- Never upload an image we do not own as input. Allowed inputs: our own concept sheets, our own sketches, our own exported sprites.
- No words, letters, numbers in a language, flags, rarity marks, glow or drop shadows in the requested image (`docs/art-direction.md` section 8). They are added at runtime or not at all.
- Colours in prompts are described by the item's own palette in plain words. UI, team, score and role hues are never requested for a cosmetic (`docs/art-direction.md` section 2).
- One change per follow-up prompt (pose, or expression, or palette), so a drift can be traced to its cause.
- Every prompt and every downloaded image is logged in the item's `provenance.md` (section 3) at the time it is made.

### 2.2 Style key

```text
Original 2D game art for an all-ages mobile game with a premium, aristocratic look. Painted, semi-realistic rendering with soft volume and rich materials: polished gold and brass, ivory, deep lacquer, gemstone, crystal, silk and wool. Fine ornament inside a clear, readable outer shape. One warm light from above. No text, letters, numbers, logos, flags, emblems or religious symbols. No weapons. No glow, no drop shadow, no cast shadow, no background scenery. Isolated on a plain flat light grey background that does not appear anywhere in the subject.
```

The plain background makes extraction in W-3 a single selection. Painted art has no forced outline (`docs/art-direction.md` section 4); prompts never try to hit a token value.

### 2.3 Templates

`{style key}` is 2.2. Placeholders in braces are filled per item; nothing else changes.

| Id | Use | Prompt after the style key |
|---|---|---|
| P-1 | Removed 2026-11-12 (character concept sheet; the game has no characters) | |
| P-2 | Removed 2026-11-12 (character model view) | |
| P-7 | Removed 2026-11-12 (character face reference) | |
| P-3 | Avatar | `Painted splash-art head and shoulders portrait of {identity}, face centred, looking slightly toward the viewer, friendly expression, rich costume detail. Keep the outer band of the image empty of anything essential.` |
| P-4 | Frame | `Ornamental round frame for a circular portrait, {theme in plain words}, symmetric, open centre left empty, thin enough not to cover a face. Palette: {colour words}.` |
| P-5 | Table layer | `Top-down view of a woven carpet table cloth, {theme}, {back / middle / front} layer only, original geometric and floral pattern, calm low-contrast centre where game pieces will sit. Palette: {colour words}.` |
| P-6 | Emoji | `Single expressive reaction icon: {emotion}, readable as a silhouette, one subject, no hands making any sign.` |
| P-8 | Tile back skin | `Back face of a domino tile, portrait 1:2, {theme in plain words}, one symmetric central emblem with filigree reaching every edge, polished metal and gemstone, reads as one jewel from across a table. The design fills the whole image edge to edge, so no background shows. Palette: {colour words}.` |

Effect sprites (`docs/art-direction.md` section 6) are white or greyscale shapes; they are drawn by hand in W-3 and need no template.

### 2.4 Keeping an item consistent

1. The approved concept (P-3 to P-6, P-8) is the canonical reference, saved as `concept/<date>-<nn>.png` and marked as canonical in `provenance.md`.
2. Every later prompt for the item attaches the canonical concept and repeats its filled placeholders word for word.
3. All variants of one item are generated in one conversation started from the canonical concept. A new conversation starts again from step 2.
4. A generation that changes proportions, palette or outline weight is rejected, not fixed in the prompt chain. Small drift (a hand, a fold) is fixed in W-3.
5. The final consistency comes from W-3 and W-4: one cleaned master, and every key frame and the static fallback is built from its parts or redrawn over it at the same canvas and pivot (`docs/art-direction.md` 5.1, section 7). AI output is reference for the key frames, not the source of the shipped frames.

### 2.5 Generating through the Codex CLI

The Codex CLI signed in with a ChatGPT account (`codex login status` shows `Logged in using ChatGPT`) generates images with its built-in image tool, which counts against that account's plan, not against API credits. It needs no browser, so an agent can run W-1 from a script; the rules of 2.1 to 2.4 apply unchanged.

```bash
cd assets-src/cosmetic/<type>/<slug>/concept
env -u OPENAI_API_KEY codex exec --skip-git-repo-check --sandbox workspace-write -C "$(pwd)" '$imagegen <style key> <filled template>. Generate exactly one image. Save the final PNG in the current working directory as <YYYYMMDD>-<nn>.png. Do not use any API key and do not edit the image.'
```

- `OPENAI_API_KEY` is removed from the environment so the call cannot switch to API billing.
- An edit of our own canonical concept attaches it with `-i <file>` and is logged on the Inputs line (section 3).
- `provenance.md` records the CLI version, the Codex session id it prints, the UTC time the prompt was sent and the C2PA manifest of the PNG, next to the account holder and plan (RC-1).
- Codex also keeps every generated image under `~/.codex/generated_images/<session id>/`; the file in `concept/` is a byte copy of it.

## 3. Rights and provenance record

Every item folder has a `provenance.md`. It is written during W-1 to W-3 and is checked at W-7 before the cultural review. `artSource` in the manifest (`docs/cosmetics.md` section 3) is `ai_assisted` if any image in `concept/` or `work/` came from a generator, `handmade` otherwise.

```markdown
# <item id>

- artSource: ai_assisted | handmade
- Authors: <names of the people who made W-1 to W-4>
- Generator: <product and model name as shown in the tool>, account <company account id, never a password or token>
- Terms: <URL of the generator's terms in force>, read on <YYYY-MM-DD>
- Canonical concept: concept/<YYYYMMDD>-<nn>.png
- Inputs uploaded to the generator: <files and their origin; only our own art>
- Similarity check: <method>, <YYYY-MM-DD>, <result; any close match and what was changed>
- Human work: <what was redrawn, recoloured, animated, in which work/ files>
- Self-check (docs/localization.md 6.1 step 1): <YYYY-MM-DD>, <author>, pass
- Cultural review: <YYYY-MM-DD>, <reviewer>, approved | rejected

| Date | File | Prompt or edit |
|---|---|---|
| YYYY-MM-DD | concept/YYYYMMDD-01.png | P-3 with {filled placeholders} |
```

What must be verified, per item:

| Id | Check | Evidence in `provenance.md` |
|---|---|---|
| RC-1 | The account holder and its plan are named, and the terms in force on that day of that plan of the generator were read (O-1) | Generator and Terms lines |
| RC-2 | No input image, prompt or file name names or depicts a third party's artist, studio, game, brand, mascot, logo or real person | Prompt log, Inputs line |
| RC-3 | A reverse image search of the canonical concept and the final static finds no close match to existing work | Similarity check line |
| RC-4 | No AI artefacts and no resemblance to a real person in any generated person (`docs/localization.md` 6, Representation) | Self-check line |
| RC-5 | Substantial human work is documented and its layered source files are kept in `work/` | Human work line, `work/` files |
| RC-6 | The item passed `docs/localization.md` section 6 | Self-check and Cultural review lines |

This document draws no legal conclusion on ownership, copyright protection or licensing of AI-generated images; those are owner items O-1 to O-4. The owner decided (2026-10-26) that the record above is enough until launch; O-2 and O-3 are reviewed with legal advice before launch, as part of gate GN-4 (`docs/launch-runbook.md` section 3), and the answer is applied to existing assets through their records.

## 4. Naming and folders

### 4.1 Tree

```text
assets-src/
  README.md
  cosmetic/<type>/<slug>/     type and slug of the item id <type>.<slug>
    provenance.md
    concept/<YYYYMMDD>-<nn>.<png|jpg|jpeg|webp>
    work/<name>.<ext>
    export/static.png
    export/anim/...           avatar, frame, tile, emoji
    export/layer/<layer>/...  table
    export/slice.json         table
  medal/<slug>/               same item layout: provenance.md, concept/, work/, export/
  effect/<slug>/
  ui/<slug>/
  tile/<slug>/
  board/<slug>/               reference board panels, slug b_1 to b_10 (docs/art-direction.md 9)
  sound/<slug>/
```

| Rule | Detail |
|---|---|
| Characters | Every folder and file name is lowercase `[a-z0-9_]`, plus `-` only in concept file names and one `.` before the extension. No spaces, no capitals |
| Slug | The slug of the item id (`[a-z0-9_]+`, `docs/cosmetics.md` section 3). It is stable like the id: a folder is never renamed or reused for another item |
| Versions | No `_v2`, `_final`, `_new` in names. Git is the history; a changed asset replaces the file and the item gets a new `revision` at publish |
| Concept files | `<YYYYMMDD>-<nn>`, date of generation, `nn` from `01` per day |
| Work files | `<slug>` or `<slug>_<part>` (for example `comet_fox_head`), any tool format; heavy formats go through Git LFS (`.gitattributes`) |
| Item folder | Holds only `provenance.md`, `concept/`, `work/` and `export/`. `provenance.md` exists from the first concept on |
| Classes other than `cosmetic` | Folder roots only. Their addresses come from their own specs: medals `medal/<slug>/static` (`docs/admin.md` 5.9), UI kit `docs/ui-kit.md` (#168), tiles `docs/tiles.md` (#169), effects #176, sound `docs/sound.md` (#177) |

### 4.2 Export to Addressables

For cosmetics the path under `export/` is the address tail, so the mapping needs no table lookup:

| Export path in `assets-src/cosmetic/<type>/<slug>/` | Address (`docs/cosmetics.md` 8.1) | Manifest field | Addressables asset in the client (5.4) |
|---|---|---|---|
| `export/static.png` | `cosmetic/<type>/<slug>/static` | `staticAddress` | The sprite `static.png`, not packed in an atlas |
| `export/anim/<nnnn>.png` | `cosmetic/<type>/<slug>/anim` | `animationAddress` | `anim.asset`, a `SpriteLoop` of the frames in name order, packed in `anim.spriteatlasv2` |
| `export/layer/<layer>/<nnnn>.png` | `cosmetic/table/<slug>/layer/<layer>` | `layers[].address` | `layer/<layer>.asset`, a `SpriteLoop` packed in `layer/<layer>.spriteatlasv2`; a layer of one frame is a static layer |
| `export/slice.json` (table only) | No address | No field | Read by `CosmeticContent` on import: the ring depth and edge step of the table, set as the border of the static and of every layer frame (`docs/cosmetics.md` 8.4) |

Item id `<type>.<slug>`, group `cosmetic-<type>-<slug>` (or `cosmetic-builtin` for a `default` item), labels `cosmetic`, `cosmetic-<type>` and `cosmetic-static` on the static sprite (`docs/cosmetics.md` 8.2). Once `export/` exists it holds `export/static.png`, as V-2 requires `staticAddress` for every item. The format is the sprite sheet loop of ADR 0012: loop frames are whole PNGs named `0001.png`, `0002.png` and on without gaps, each at the canvas of the static (`docs/art-direction.md` section 7).

One command takes a reviewed item from `export/` into a manifest and into the client:

```bash
npm run asset:add -- assets-src/cosmetic/avatar/comet_fox --manifest <manifest.json> --rarity rare --availability live --source shop:offer.avatar_comet_fox
```

`tools/asset-export` runs these steps in order and writes nothing when one fails:

1. Naming of the item folder (section 4.1, `tools/asset-naming`).
2. Export: `export/static.png` is present, every PNG has an alpha channel, loop frames are numbered without gaps and share the canvas of the static. A table has `export/slice.json` with a whole positive `depth` that leaves a field and a whole positive `step` shorter than the static, with an even remainder (`docs/cosmetics.md` 8.4). A Git LFS pointer instead of a PNG is reported as such.
3. Size: the budget of 5.4.
4. `provenance.md` (section 3): `artSource` is `handmade` or `ai_assisted`, and the Cultural review line ends in `approved`. A manifest only holds reviewed items (`culturalReview` is `approved`, V-1).
5. Entry: the id, `nameKey`, `animated`, the addresses and `layers` follow from the folder, and `artSource` from `provenance.md`. `rarityId` (`--rarity`), `availability` (`--availability`), `sources` (`--source <type>[:<ref>]`, repeated per source), `group` (`--group`, emoji) and the layer order (`--layers <back>,...,<front>`, a table with more than one layer) come from the options; an item already in the manifest keeps its values for the options left out. `revision` is 1 for a new item and rises by one when the entry changed (V-13). `manifestVersion` stays as it is, as the server sets it at publish (`docs/admin.md`, Cosmetics manifest note).
6. The whole manifest passes `tools/cosmetics-schema` (V-1 to V-11, V-19); only then is the file written.
7. The export is copied to `client/Assets/Cosmetics/<placement>/<type>/<slug>/`, where `<placement>` is `builtin` for an item with a `default` source and `remote` otherwise. The copy in the other placement and frames that are no longer exported are removed with their `.meta` files. `--client <dir>` changes the folder, `--no-client` skips the step.

`--manifest` names an existing manifest file: the published manifest (`GET /admin/config/cosmetics`), a draft, or a copy of `docs/cosmetics/examples/minimal.json` to start from. The admin publish (W-7, W-8) stays the step that puts an item in front of players. After the command, `HypeDomino/Import Cosmetics` in Unity builds the content of 5.4; the PNGs and the generated assets under `client/Assets/Cosmetics` are committed, the PNGs through Git LFS (`.gitattributes`).

## 5. From raw image to game

### 5.1 Clean-up (W-3)

1. Photoshop: remove the background, trim, place on the class canvas at the class pivot (`docs/art-direction.md` 5.1; sizes per class: O-6).
2. Photoshop: paint over the approved concept so material and light follow `docs/art-direction.md` section 4. Fix AI artefacts: hands, eyes, asymmetry, merged or extra parts, smeared ornament.
3. Separate the parts that move (ornament, gems, flames, crystals) onto their own layers for W-4. An outline only where the class spec asks for one (#168 to #175).
4. Remove anything art-direction section 8 forbids: text, glow, drop shadow, rarity colour, team or score hue as decoration.
5. Silhouette test (filled black) and smallest-display-size test (`docs/art-direction.md` sections 3 and 5) on the reference board.

### 5.2 Animation (W-4)

The runtime format and the per-screen animation budget are ADR 0012 (#93); this section only fixes what each path must deliver.

| Path | Steps | Delivers |
|---|---|---|
| After Effects | Animate the layered master, render each clip as a PNG sequence with alpha, no effects that add glow, blur or shadow | Frame sequences for a sprite sheet |
| Blender | Orthographic camera at the class canvas, lighting matching `docs/art-direction.md` section 4, no post-processing, render PNG sequence with alpha | Frame sequences for a sprite sheet |

Per type: avatar, frame, tile and emoji deliver one loop; an animated table delivers its layers, back to front, at most `TableMaxLayers` (`docs/cosmetics.md` section 4). Each clip is checked with reduce motion and on the low effect level, where only the static shows.

Static fallback, every item: a chosen key frame, not frame 0, exported at the same canvas, pivot and framing as the animation; for a table the flattened rest frame of all layers (`docs/art-direction.md` section 7). It is reviewed side by side with the animation on board panel B-8.

### 5.3 Export (W-5)

- PNG with an alpha channel, straight alpha, sRGB, at the class canvas, into the `export/` path of section 4.2. Loop frames are `0001.png` on, without gaps, at the canvas of the static.
- Effect sprites exported white or greyscale for runtime tint (`docs/art-direction.md` section 2).
- `node tools/asset-naming/asset-naming.mjs` passes before the commit. `npm run asset:check -- <item folder>...` runs the export and size checks of 4.2 without a manifest.

| Path | Export |
|---|---|
| Photoshop (static only) | File > Export > Export As: PNG, Transparency on, Convert to sRGB on, size 100% |
| Blender | `blender --background <file>.blend --python-exit-code 1 --python tools/asset-export/blender/export_loop.py -- --frames <item>/export/anim --static <item>/export/static.png --static-frame <n>`. For a table layer, `--frames <item>/export/layer/<layer>` from the scene of that layer. The script refuses a scene without an orthographic camera and a static frame outside the scene range. It removes the numbered frames already in the folder, renders every frame of the scene range as `0001.png` on, and renders the chosen key frame `<n>` as the static. It sets the output: PNG, RGBA, 8 bit, transparent film, resolution at 100%, Standard view transform with no look, exposure 0, gamma 1, sRGB display, compositor and sequencer off. Canvas size, frame range and frame step stay in the `.blend` (O-6). `tools/asset-export/blender-export.test.mjs` runs it with Blender where Blender is installed (`BLENDER` names the executable); CI has no Blender and skips that test |
| After Effects | No script: none could be tested without After Effects, so a script is open on #179. Render Queue output module: format PNG Sequence, channels RGB + Alpha, color Straight (Unmatted), depth Millions of Colors+, colour profile sRGB, Starting # 1 with Use Comp Frame Number off; output file `[####].png` into `export/anim/` or `export/layer/<layer>/`. The static: Composition > Save Frame As > File at the chosen key frame with the same output module, saved as `export/static.png` |

### 5.4 Pack (W-6)

`HypeDomino/Import Cosmetics` in the Unity editor (`CosmeticContent`; batch: `-executeMethod HypeDomino.Editor.CosmeticContent.ImportAllAndExit`) turns every item folder under `client/Assets/Cosmetics` into its Addressables content. It can be run again after any change:

- Group: `cosmetic-<type>-<slug>` with the remote settings of the `ContentSetup` item template (one bundle per entry, content-hashed names), or `cosmetic-builtin` for the `builtin` placement. An item that moved between placements leaves its old group.
- Entries: the addresses and labels of 4.2. Entries of the item whose files are gone are removed, and so are the loop and atlas of a loop that has no frames left.
- Sprites: Sprite, single, full rect mesh, no mipmaps, alpha is transparency, sRGB.
- One Sprite Atlas per loop (`anim.spriteatlasv2`, `layer/<layer>.spriteatlasv2`) in the item's group, without rotation or tight packing, so a UI `Image` draws its sprites. It has no padding (`CosmeticContent.LoopAtlasPadding`), so 16 frames of one canvas fill four by four frames, the atlas the size rule of ADR 0012 counts: an animated frame's 16 frames of 256 px pack into 1024 x 1024 (467,856 bytes at ASTC 6x6). With the default padding of 4 px, four 256 px frames and their gaps no longer fit 1024 px across, so the loop needs a larger texture or a second page, about twice the memory and over the 600,000-byte frame limit (#96). A frame's edge texels therefore sit next to the next frame's, so the export keeps the transparent margin of its framing. The static is not packed, so it downloads on its own and before the animation (`docs/cosmetics.md` 8.2). Effect sprites only in the shared effects atlas (`docs/art-direction.md` section 6).
- ASTC on both platforms (ADR 0002) for sprites and atlases, with the block of `ProjectSetup.SpriteFormat`: ASTC 6x6, the format of the project sprite preset. A block size per asset class is O-5; once it is set, it replaces this default in `CosmeticContent` and in `tools/asset-export/budgets.json`.
- `CosmeticContentTests` (EditMode, client CI job) cover groups, addresses, labels, loops, importer and atlas settings, and pack a loop of 16 edge-to-edge 256 px frames into one 1024 x 1024 page.

Size:

- `tools/asset-export` estimates an item's texture bytes with the size rule of ADR 0012: every sprite of the item, the static and every frame, at 16 bytes per ASTC block of the default block, summed. Atlas padding and bundle overhead only add to it, so an item over its limit here is over it in the build too. An item under it still owes the bundle measurement of ADR 0012.
- Limits are in `tools/asset-export/budgets.json`, per type and kind (static or animated), each with its source; 1 KB reads as 1,000 bytes:

| Type | Static | Animated | Source |
|---|---|---|---|
| Avatar | No limit | Under 300,000 bytes | #171 |
| Emoji | Under 100,000 bytes | Under 100,000 bytes | #175 |
| Frame | Under 125,000 bytes | Under 600,000 bytes | Owner decision 2026-11-16 |
| Table | Under 1,000,000 bytes | Under 5,000,000 bytes, at most 2 layers of which at most 1 animated | Owner decision 2026-11-16 |
| Tile skin | No limit | No limit | O-5 |

- The animated frame limit leaves margin over the 256 px canvas of set 1: 17 sprites, 502,928 bytes. Owner decision 2026-11-16: the static frame limit is 125,000 bytes, raised from the first-named 100,000, and the four 512 px static frames of set 1 (118,336 bytes each) stay at 512 px and pass it; the other static frames are 256 px (29,584 bytes). A frame is never drawn larger than its canvas (`docs/art-direction.md` 5.2). An animated frame on a 512 px canvas (about 2,000,000 bytes) does not fit.
- A layer of an animated table is a folder under `export/layer/`; it is animated when it holds more than one frame. The tool counts both and fails a table over either limit. Set 1's animated tables have a one-frame `base` layer and a 10-frame `motion` layer, 4,930,560 bytes.
- A type and kind without a limit is reported and never fails.
- CI job `assets`, part of `CI complete`, runs when `assets-src/` or these tools change: the naming check over all of `assets-src/`, then the export and size checks over every cosmetic item the change touches, with Git LFS files pulled for those items only. The estimates go to the job summary.
- The admin measures format, size, atlas and frame count on upload (`docs/admin.md` 5.2).
- A new or changed effect, or anything in the match scene, reports the measurements of `docs/performance.md` section 8.

## 6. Sample tile skin walkthrough

The acceptance check of #167. The game has no characters (#584), so the sample is a tile back skin instead of the character the issue first named: `tile.sema`, the sky skin of the approved prototype, in `assets-src/cosmetic/tile/sema/`. It is an animated `tile` item (`docs/cosmetics.md` sections 2 and 4); its board panels are B-4, B-9 and B-10 (`docs/art-direction.md` section 9). Evidence is filled in for steps that are done; a step marked Partial lists what is still missing in the state table below. The item is complete when every row has full evidence.

| Step | Done when | Evidence |
|---|---|---|
| 1. Item id chosen | `tile.sema` free in the manifest, folder `assets-src/cosmetic/tile/sema/` created with `provenance.md` | 2026-11-12: no `tile.sema` in `server/internal/cosmetics/builtin.json` or `docs/cosmetics/examples/`; folder, canonical concept and `provenance.md` added by #167; `node tools/asset-naming/asset-naming.mjs` passes |
| 2. Concept (W-1) | P-8 used with the style key; prompt log, generator, account holder, plan and terms in `provenance.md` (RC-1) | Partial, 2026-11-12: prompt (verbatim), generator, account holder (the project owner's own account), plan (Free), no uploaded inputs and the conversation time matching the C2PA manifest are in `provenance.md`; RC-2 recorded against the prompt. The prompt predates P-8 and the style key, and the terms are not read yet |
| 3. Concept review (W-2) | Canonical concept marked; shown on B-4; silhouette and size tests passed; RC-2, RC-3 recorded | |
| 4. Clean-up (W-3) | 5.1 steps done; layered master in `work/` | Partial, 2026-11-12: layered master (`work/sema_master.png`, `sema_stars.png`, `sema_sparkles.png`) made by an AI agent with `work/sema_cleanup.py`; 5.1 step 3 done, steps 1 and 2 in part, steps 4 and 5 not done (`provenance.md`, Clean-up) |
| 5. Animation (W-4) | One loop per 5.2, in the skin's theme (`docs/art-direction.md` 5.3) | Partial, 2026-11-12: one 16-frame star twinkle loop for 12 fps at 128 x 256 by an AI agent with `work/sema_loop.py`; sheen and glow pulse stay runtime (`docs/art-direction.md` section 4). Reduce-motion and low-level check not done |
| 6. Static fallback | Key frame exported at the same canvas and pivot; B-9 drawn | Yes, 2026-11-22: `export/static.png` is loop frame 9, the rest frame, same canvas and pivot. B-9 (`assets-src/board/b_9/`) shows it over the key frame with the centre pivot marked and no seam, and the skin at the tile's smallest display size, 32 x 64 px |
| 7. Export (W-5) | `export/static.png` and `export/anim/` present; naming check passes | 2026-11-12: `export/static.png` and `export/anim/0001.png` to `0016.png`; `node tools/asset-naming/asset-naming.mjs` and `npm run asset:check -- assets-src/cosmetic/tile/sema` pass (17 sprites, ASTC 6x6 estimate 257,312 bytes, no tile limit yet). `npm run asset:add` against a copy of `docs/cosmetics/examples/minimal.json` (walkthrough values `--rarity legendary --availability disabled --source shop:offer.tile_sema`; the manifest copy is not committed, real values come with the admin draft of step 9) wrote a valid entry and copied the export to `client/Assets/Cosmetics/remote/tile/sema/` |
| 8. Pack (W-6) | Item group and atlas built with ASTC, within the tile limits of O-5 | Partial, 2026-11-12: `HypeDomino.Editor.CosmeticContent.ImportAllAndExit` through the `unity-editor.yml` workflow (run 43250959440) made group `cosmetic-tile-sema`, `anim.asset` and `anim.spriteatlasv2` with ASTC 6x6 on Android and iOS; its output is committed. O-5 sets no tile limit yet, and no bundle was built or measured |
| 9. Upload (W-7) | Admin asset checks pass; self-check and cultural review by another account recorded | |
| 10. Publish (W-8) | Manifest entry valid with `artSource` `ai_assisted`; item renders animated on medium and high and static on low, on Mid and High devices | |

State on 2026-11-12 of the steps without full evidence:

| Step | What exists | What closes it |
|---|---|---|
| 2 | Everything but the terms; the prompt was written before P-8 and asks for a glow | The account holder reads the OpenAI terms in force on 2026-11-12 for the Free plan and records URL and date (an automated fetch was refused). Whether a concept made before P-8 may stay canonical, or is made again with P-8, is the owner's call |
| 3 | Canonical concept marked in `provenance.md`; the skin is on B-4 (#166); RC-2 recorded; the skin at the tile's smallest display size, 32 x 64 px, on B-9 | Silhouette test (filled black); RC-3 once O-7 names a search tool |
| 4 | Scripted clean-up by an AI agent: violet haze around the emblem reduced, moving parts on layers. No paint-over; the lens flare at the gem and the bloom on the crescent remain (`docs/art-direction.md` section 8) | An artist paints over the master (5.1 steps 2 and 4) and documents the human work (RC-5); silhouette and size tests with O-6 |
| 5 | A procedural star twinkle; canvas, frame count and rate taken from the ADR 0012 measurement because O-5 and O-6 are open | Reduce-motion and low-level check in Unity; the class canvas and rate once O-5 and O-6 are set, then a new export |
| 8 | The imported item group, loop and atlas | The tile limits of O-5 and a measured bundle of the item (ADR 0012 size rule) |
| 9 | Cultural review recorded in `provenance.md`: approved by the project owner on 2026-11-12, the crescent-with-gem emblem judged acceptable for the four launch markets. Self-check not signed: W-3 and W-4 have no human author. No admin upload | Self-check by the author (`docs/localization.md` 6.1 step 1); upload with the admin asset checks (`docs/admin.md` 5.2) |
| 10 | Nothing | Steps 1 to 9, a publish and a check on Mid and High devices |

B-10 (do and don't) is drawn from this skin and its render.

## 7. Open items

| Id | Item | Owner |
|---|---|---|
| O-1 | Which generator account is used | Decided 2026-10-28: team members and contractors may use their own accounts. `provenance.md` names the account holder and the plan for every generation (RC-1); O-2 covers the terms of every plan that was used |
| O-2 | The current terms of the chosen generator on ownership and commercial use of output, read and confirmed for this use | Owner, with legal advice, before launch (GN-4) |
| O-3 | Whether copyright protection of AI-assisted art matters for the business, in which jurisdictions, and how much human work (RC-5) that needs | Owner, with legal advice, before launch (GN-4) |
| O-4 | Disclosure and retention | Decided 2026-10-26: AI use is disclosed only where Google Play or App Store rules require it, checked at GN-4; `artSource` stays in the manifest; `provenance.md` is kept in git for as long as the repository exists |
| O-5 | ASTC block size and atlas size per asset class; the maximum item size is set for avatars (animated), emoji, frames and tables (5.4, owner decision 2026-11-16 for frames and tables) and open for static avatars and tile skins | #93, #135, class specs #168 to #175 |
| O-6 | Canvas size, pivot, outline weight and smallest display size per class | #168 to #175 (`docs/art-direction.md` section 10) |
| O-7 | Reverse image search tool for RC-3 | Art lead |
| O-8 | Steps 2 to 5 and 7 to 10 of the sample item `tile.sema` (section 6), and the avatar and frame of the board image B-9 (B-1 to B-7 were committed with #166, B-10 with #167, B-8 with #176, B-9 without the avatar and frame on 2026-11-22) | Art team, #167; the terms of step 2 with the account holder of the 2026-11-12 generation; paint-over and human work of steps 4 and 5 by an artist; cultural review by the owner or a delegate; the avatar and frame of B-9 once O-6 sets their smallest display size (#168) |
