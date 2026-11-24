# Art direction

Status: proposal; the owner approves it by merging. The direction itself (section 1) is an owner decision of 2026-11-12, made on an approved prototype. How every shipped image looks: palette use, shape, materials and light, seats and cosmetics, type and icons, effects, static fallbacks and the reference board. It defines no colour, duration or budget value. Colours are named by their token path in `docs/design-tokens.json`; motion and effect budgets come from `docs/ui-motion.md` and `docs/performance.md`; delivery follows ADR 0002 and ADR 0012; item data follows `docs/cosmetics.md`; content limits follow `docs/localization.md` sections 6 and 7 (PEGI 3 target). `node tools/art-direction/art-direction.mjs` checks that this file contains no colour literal and that every token path it names exists; CI runs its tests through `npm run test:tools`.

Production workflow, prompt templates, naming and folders are `docs/asset-workflow.md` (#167). UI kit is #168, tile set and table base art #169, effects library #176, animated asset format and per-screen animation budget ADR 0012 (#93), rarity and status colours `docs/rarity.md`.

## 1. Direction and pillars

Owner decision 2026-11-12, for every screen: an aristocratic, premium look. A dark lacquered room, a red and indigo carpet on the table, real ivory tiles, gold and brass fittings, painted portraits in animated frames, names in elegant type. The approved prototype is panel B-1 of the reference board (section 9); where this document and the prototype disagree, the prototype wins until this document is changed.

| Decision | Means |
|---|---|
| No characters | The game has no 3D or 2D characters anywhere (#584, ADR 0009 superseded). Around the match table there are only tiles and the seats |
| Seats | A seat shows the avatar, its animated frame and the name. No square panel behind it and no tile count number (section 5.5) |
| Table | The cloth is a carpet in red and indigo with a wood rim and a gold inlay line. The default table is this carpet |
| Tiles | Real-looking ivory tiles: black pips, a dark divider line, a brass pin at its centre |
| Avatars | Premium painted splash-art portraits |
| Frames | Animated, and each one different in shape and motion, not a recolour of another |
| Tile backs | Skins with a legendary feel and their own animation |
| Names | Elegant italic type with a letter effect: gold, lightning, flame or frost |
| Type | Several families by role, all with Azerbaijani letters (section 5.6) |
| Icons | Iconsax, bulk style; motion is ours (section 5.7) |

| Pillar | Means |
|---|---|
| Readable first | Board state (pips, legal ends, turn owner, score) reads at phone size in one glance. Ornament frames the board and never competes with it |
| Luxury from material and light | Gold, brass, ivory, lacquer and wool carry the look at rest; motion and glow add life on events and on equipped cosmetics, inside their budgets |
| Cheap by construction | 2D sprites only: materials and light are painted into the sprite, glows are runtime sprites tinted from tokens, everything sits in shared atlases. A look that needs a post-process, a real-time light or a material instance per item is rejected |
| All ages, original | Every design is ours, passes the cultural checklist and reads the same in all four launch markets. Premium means a salon, never a casino |

## 2. Palette usage

Dark ink and lacquer ground, gold as the house metal, team blue and orange as the only cool and warm accents in the interface. Values and contrast are `docs/ui-motion.md` section 2.

| Layer | Tokens | Rule for art |
|---|---|---|
| Ground | `color.background.base`, `color.background.raised` | Ink and lacquer. Everything is composed to sit on these. Art never fills the ground with a bright flat colour |
| Panels behind art | `color.surface.default`, `color.surface.raised` | Raised lacquer. Icons, avatars and items in lists and shop cards are checked on both |
| Gold ornament | `color.gold.hi`, `color.gold.default`, `color.gold.mid`, `color.gold.lo` | Rims, inlay lines, dividers, icons, the turn timer ring, the score crown. A gold gradient runs from `color.gold.hi` to `color.gold.lo` |
| Wood | `color.wood.hi`, `color.wood.default`, `color.wood.lo` | The table rim of the default table and the hand tray |
| Shadow | `color.shadow` | Baked elevation shadows (ui-motion section 4) and runtime drop shadow sprites under tiles |
| Dimming | `color.scrim` at `opacity.scrim` | Modal backdrop, the `block` board dim, the soft backdrop behind seat names and hints; never painted into an asset |
| Main action | `color.role.primary` | Only the one primary action per screen and its `glow.soft`. Gold, so it is told apart by being the only glowing pill |
| Selection, legal ends | `color.role.secondary` | The pulsing ring around the selected tile and the outline of a legal end on the board |
| Outcome | `color.role.success`, `color.role.warning`, `color.role.danger` | Only with their meaning and always paired with an icon or shape |
| Points | `color.role.score` | Score numerals, score popups, score particles, points flying to the meter |
| Teams | `color.role.teamA`, `color.role.teamB` | The team gems in the score panel, team meters, the `domino` banner and `matchWin` confetti. A cosmetic is never dominated by a team hue, so it cannot be mistaken for a team marker |
| Text | `color.text.primary`, `color.text.secondary`, `color.text.muted` and each role's `text` shade | Warm parchment tones. Text is never baked into art (localization 6) |
| Rarity | `rarity.*`, `docs/rarity.md` | No rarity colour, diamond or rarity glow is painted into an asset. The client draws them from `rarityId` |

Rules:
- Gold is shared by ornament, the primary action, selection and score, so meaning never rides on hue alone: the primary action is the only pill with a glow, selection is a pulsing ring, score is numerals in the `numeral` family (section 5.6).
- Effect and glow sprites are authored white or greyscale and tinted at runtime with a token through vertex colour. One sprite serves every tint; there are no per-colour effect textures.
- A coloured element on a role fill uses that role's `on` shade, as UI text does.
- No colour combination is chosen to evoke a flag, party or side (localization 6).
- UI, effect, team, score and role colours come only from the tokens above. Cosmetic illustration art (avatars, tables, frames, tile skins, emoji) and the shared tile face use their own palette per asset, reviewed on the reference board (section 9). Cosmetic art is never dominated by a team or role hue and never evokes a flag or political side (localization 6).

## 3. Shape language

| Shape | Meaning | Used for |
|---|---|---|
| Rounded rectangle | Solid, touchable, the game piece | Tiles, buttons, panels, cards, the hand tray, the score panel. UI corner radii come from `radius` |
| Circle | People and counting | Pips, avatar crops, frames, the turn timer ring and its count bubble |
| Pill | The main action, soft backdrops | The primary button (`radius.pill`), the backdrop behind names and hints |
| Small diamond | A jewel accent | Team gems, the gem on the score divider, gems set in frames. Never with pips, so it cannot be read as a rarity mark (`docs/rarity.md` 2.1) |
| Filigree and crystal | Ornament | Frames and tile back skins: scrolls, leaves, wings, crystals. Only on cosmetics and gold rims, never inside the board area |
| Star, ray, spark | Impact, an event happening now | Bursts, the `block` lock stamp, `rankUp` rays. Only in effects and only for the effect's lifetime |

Rules:
- The silhouette reads first: an asset still tells what it is as a filled shape at its smallest display size. Ornament adds richness inside a clear outer shape; if the shape is lost, ornament is removed, not shrunk.
- Detail that vanishes at the smallest size the asset is shown at is removed. The smallest display size per asset class is set with the class spec (#168 to #175).
- Pips are identical round dots at a fixed layout per value, so a value reads by pattern, not by counting. Tile proportions and pip layout are `docs/tiles.md` (#169).
- Crystal and ornament points are allowed on frames and skins; nothing reads as a weapon (localization 6).
- No card suits, chips, reels, roulette, green felt or coin piles in shape or silhouette (localization 6, gambling imagery).

## 4. Materials, light and outline

| Topic | Rule |
|---|---|
| Rendering | Painted, semi-realistic: soft volume, rich materials, fine ornament. Light and material are painted into the sprite; the runtime adds no lighting |
| Materials | Polished gold and brass for metal, ivory for tile faces, deep lacquer for panels, wool for the carpet, gemstone and crystal for accents. Plastic or flat cartoon finishes are off direction |
| Key light | One warm lamp above the table centre. Highlights sit on the upper side of every object, cast shadows fall straight down, matching `elevation`, which offsets shadows only on Y. The table art is brighter in the centre and darker toward its rim |
| Tile | Ivory face with a soft bevel highlight, pure dark pips, a dark divider line with a brass pin at its centre. The face is shared art for every player (`docs/cosmetics.md` section 2), so it is painted, not tinted |
| Rim light | Allowed on avatars to separate them from the ground; one band on the side away from the key light |
| Outline | No forced outer outline. Painted assets separate from the ground by value, rim light and their frame or bevel. Where a class needs an outline to read at small size, its weight is set with the class spec (#168 to #175) |
| Glow | Never baked into a sprite. Glow, frame auras and tile skin pulses are runtime additive sprites tinted by a token and scaled by a `glow` level (`glow.soft`, `glow.strong`, `glow.intense`); each costs one overdraw layer |
| Sheen | The moving highlight across gold rims, frames and skins is one shared white streak sprite masked by the asset, moved by a transform tween; no per-item material |
| Drop shadow | Never baked into a sprite. UI elevation is the 9-slice shadow sprite in `color.shadow`; a tile's shadow is a runtime sprite under it |
| Text outline and shadow | Through the font's material presets (ui-motion section 3), never a second text copy |
| Post-processing | None in the match scene at any effect level (ui-motion section 4). Art must look finished without bloom, blur or grading |

## 5. Seat, cosmetics, type and icons

### 5.1 Avatars

| Item | Rule |
|---|---|
| Style | A premium painted splash-art portrait: head and shoulders, face centred, expressive light, rich costume detail |
| Face | Eyes and mouth carry the expression and stay readable at seat size |
| Looks | Varied across a set; no face, skin tone or body type stands for a group, region or trait (localization 6, representation) |
| Crop | The portrait is shown in a circle inside its frame. Nothing essential in the outer band that the frame (#173) or the rarity glow (`docs/rarity.md`) covers; the band width is set with #168 |

All-ages: no sexualised proportions, poses or outfits, no likeness of a real person, no resemblance to an existing mascot, figure or brand. Semi-realistic faces make the real-person check of `docs/asset-workflow.md` RC-4 matter more, not less.

### 5.2 Frames

- A frame is a round ornament around the avatar circle, open in the centre.
- Every frame has its own theme, silhouette and motion; two frames never differ only in colour.
- Motion: the sheen of section 4 across the metal, a runtime aura behind the frame (a pulse, a slow turn or a flicker), and the frame's own ambient detail (embers, snow, stars, sparks, bolts) authored into its animation (ADR 0012). Ambient detail is never a live particle system per seat; it is counted with the seat in the base scene and measured (ui-motion section 7).
- On the low effect level (`effects.levels.low.animatedCosmetics` false) and with reduce motion on, only the static frame shows (section 7).
- Display size, owner decision 2026-11-16: a frame is never drawn larger than its canvas, so it is never upscaled. An animated frame (256 px canvas) covers at most 256 screen pixels across at the device's scale, and a 512 px static frame at most 512; a layout may draw either smaller. `AvatarView` caps the one square of avatar, frame and halo at the pixel size of the frame sprite it shows (`docs/cosmetics.md` 8.3).

### 5.3 Tile back skins

- A skin covers the whole back face edge to edge, with an ornamental, symmetric design that reads as one jewel or emblem from across the table.
- It has a legendary feel: polished metal, crystal or flame, never a flat pattern.
- Motion: the shared sheen sweep and a glow pulse in the skin's theme, staggered across the tiles of one seat so a row shimmers rather than blinks.
- A skin never changes the face, the pips or the divider (`docs/cosmetics.md` section 2).

### 5.4 Tables

- The table art is the carpet cloth inside a wood rim (`color.wood.*` on the default table) with a gold inlay line. The carpet is an original geometric and floral pattern with a calm centre where tiles are played; it holds no script, emblem or religious symbol (localization 6).
- Tables are cosmetics (`docs/cosmetics.md`); every table follows the same layout so tiles and seats read on all of them.
- One table fits every screen shape (owner decision 2026-11-16, `docs/cosmetics.md` 8.4): the ring (rim, inlay, carpet border and its guards) has the same depth on all four sides and is never cut; the field inside it may be cropped, so it holds nothing that must show whole near its edge, and a centre medallion keeps clear of the field's left and right ends. The top and bottom borders repeat one step between their corner pieces, and a screen wider or narrower than the art gains or loses whole steps in the middle of those edges. A unique motif in the middle of a long edge (a cartouche) is therefore not used; a corner piece may be as rich as the art wants. Motion on the ring sits in the corners and on the side bands, or repeats with the step.

### 5.5 The seat

- A seat is the avatar inside its frame, the name under it on the soft `color.scrim` backdrop, and, while it is that seat's turn, the gold timer ring around the frame with the seconds in a gold count bubble.
- There is no square panel behind a seat and no tile count number. A seat's hidden tiles lie face down beside it in its tile skin, so their number reads from the tiles.
- The layout box of a seat is `size.seatPanelWidth` by `size.seatPanelHeight` (ui-motion, Match layout); it is a box, not a drawn panel.
- No character, body or figure appears at the table.

### 5.6 Type and names

| Role | Family token | Family | Use |
|---|---|---|---|
| Numerals | `typography.family.numeral` | Cinzel | Score, counters, the turn timer, the target score. Never words: it has no Ə or ə |
| Names, titles | `typography.family.display` | Playfair Display Italic | Player names (`typography.scale.name`), titles and headings |
| Hints | `typography.family.hint` | Cormorant Garamond Italic | Turn hints and short guidance (`typography.scale.hint`) |
| Labels, body | `typography.family.ui` | Josefin Sans, light | Labels in spaced capitals, body text, buttons, prices |
| Cyrillic fallback | `typography.family.uiCyrillic` | Montserrat | Russian text in `ui` styles; Josefin Sans has no Cyrillic |

- All five are Google Fonts families under the SIL Open Font License 1.1, which allows embedding them in a commercial game. Glyph coverage per family is in ui-motion section 3; currency signs are set only in `ui`.
- Italiana and Marcellus were tried in the prototype and rejected because they lack Azerbaijani letters.
- Name letter effects, one per name: gold (a flowing gold gradient with a soft warm glow), lightning (white to blue with flicker and small bolts), flame (red to yellow rising, with a heat wobble) and frost (cool silver gradient with a slow shimmer). Proposal: a name's effect follows the theme of the equipped frame, so it is not a separate item. Each effect is a font material preset plus a runtime glow; its draw call cost and the preset limit of ui-motion section 3 are an open item (section 10).

### 5.7 Icons

- Interface icons come from Iconsax (npm `iconsax`), bulk style: a solid shape with a second tone at reduced opacity. The free set holds more than 1200 icons. Animated icons are a paid tier and are not used; icon motion is ours, by transform tweens (a wink, a turning gear, pulsing sound waves).
- Icons are exported white with the bulk second tone kept as reduced alpha, packed in the UI atlas and tinted with a gold gradient (`color.gold.hi` to `color.gold.lo`) through vertex colour, with a soft dark drop shadow sprite for legibility on the carpet.
- One icon style everywhere: no mixing with other sets or styles.

## 6. Effect language

Effects are named and budgeted in ui-motion section 7 (`effects.budgets`, `effects.levels`). This section fixes how they look; it never raises a budget.

| Element | Look | Constraint |
|---|---|---|
| Glow | A soft round halo behind its owner, tinted by the event's token | Levels only from `glow`; strength by score step comes from `scoreCascade.steps`; one overdraw layer |
| Burst | Gold sparks and four-point stars for scoring and wins, warm dust for a tile settling (`tilePlace`) | Count per level from the effect's budget, for example `effects.budgets.tilePlace` and `effects.budgets.scoreCascade`. Score bursts tinted by `color.role.score`, team events by the team fill |
| Trail | A tapered gold streak behind a moving thing: points flying to the meter, a tile travelling to its end | Part of its effect's particle and draw call budget; it ends when the motion ends (`motion.duration.base` for tile travel) |
| Impact | Settle with a small squash, ring pulse (`spinnerFormation`), stamp (`block`) | Shake only from `motion.shake.small` or `motion.shake.medium`; full-screen flashes at most `motion.maxFullScreenFlashesPerSecond` |
| Confetti | Paper and gold-leaf flakes in the winning team fill and `color.gold.default` (`matchWin`) | No coins, cash, chips or reel motion (localization 6) |
| Ambient | Slow warm dust motes in the lamp light over the table | Part of the table's animation or the base scene, measured with it; off on the low level |

Rules:
- An effect is designed at the low level first: it must tell the event with the low particle count (`effects.levels.low.maxParticles` across all live effects) and without animated cosmetics (`effects.levels.low.animatedCosmetics`). Higher levels add particles; they do not change timing or meaning.
- Every effect works with reduce motion on (ui-motion section 6): the information stays, the motion drops.
- Effect sprites are small, tightly meshed and in one shared effects atlas so they batch and keep overdraw within `maxOverdrawLayers` of the level.
- Additive or alpha blending from the shared effect material only; no per-effect material instance.
- No video for in-match effects (ADR 0002).
- A new effect or a change to one reports the measurements in ui-motion section 7 and `docs/performance.md` section 8.

## 7. Static fallback

Every animated asset has a static sprite (ADR 0002, `docs/cosmetics.md` V-2). It is shown on the low effect level, while the animation downloads, after a failed download and when the animation budget (ADR 0012) is spent, so it is a finished asset, not a placeholder.

| Rule | Check |
|---|---|
| It is a chosen key frame that carries the item's full identity, not frame 0 by default | Reviewer can name the item from the static alone |
| Same canvas size, pivot and framing as the animation | Swapping static and animated shows no jump |
| A table's static is the flattened composition of all its layers at their rest frame (`docs/cosmetics.md` section 4) | Static and animated tables line up layer for layer |
| A frame's or skin's static shows its metal and gems at rest, with no aura, sheen or ambient detail painted in | The runtime glow still applies on top |
| It meets every rule in this document | Reviewed together with the animation |
| It passes the cultural review with the animation; switching an item off removes both (localization 6.1) | `culturalReview` is `approved` |

## 8. Do and don't

| Do | Don't |
|---|---|
| Name every UI, effect, team, score and role colour by its token path | Use a colour value that has no token for those |
| Paint material and light into the sprite | Rely on real-time lights, shaders or post-processing for the look |
| Author effect sprites white and tint them at runtime | Ship one effect texture per colour |
| Leave glow, auras, drop shadows and rarity marks to the runtime | Paint glow, drop shadows or rarity colour into a sprite |
| Keep text out of images; every word is a localization key | Bake words, numbers in a language, or a language flag into art |
| Test every asset as a silhouette and at its smallest display size | Add ornament that turns to noise at seat or list size |
| Use the team hues only for teams | Use team hues as decoration on cosmetics |
| Give every frame and skin its own shape and motion | Ship recoloured copies of one frame |
| Keep the board centre calm and the pips black on ivory | Put pattern, glow or ornament where tiles are read |
| Design effects at the low level first | Make a look that only works at the high level |
| Make the static fallback a finished key frame | Ship frame 0 or a blank as the fallback |
| Keep expressions friendly and all-ages | Taunting gestures, weapons, blood, sexualised poses |
| Premium as a salon: carpet, lacquer, gold, ivory | Casino cues: green felt, chips, card suits, reels, cash piles, neon signs |
| Create original designs | Copy or imitate another game, brand, mascot or real person |

## 9. Reference board

The board is a set of panels stored in `assets-src/board/b_<n>/` under the naming rules of `docs/asset-workflow.md` section 4, each with a `provenance.md` naming its sources. Panels that show the match are drawn at the 1080-unit short side of the match canvas (ui-motion section 1); sheets are drawn on `color.background.base`. A panel takes UI, effect, team, score and role colours only from the tokens named in this document, shows cosmetic art in its own per-asset palette (section 2), carries no text other than the mock interface text of the prototype, and names no third-party game, studio or product in its files or metadata. Board images are reference only and never ship.

| Panel | Shows | Proves | State |
|---|---|---|---|
| B-1 Match scene | The approved prototype at 20:9: carpet table, wood and gold rim, four seats with avatars, animated frames and name effects, face-down rows in four skins, the hand of ivory tiles with the selected tile, the legal-end outline, the hint, the score panel, the icons | Sections 1, 2 and 5 together on the real ground | Committed 2026-11-12 |
| B-2 Value check | B-1 in greyscale | Pips, legal end, turn owner and score read without hue | Committed 2026-11-12 |
| B-3 Tile faces | Seven ivory faces with black pips and the brass pin | Section 4 tile rule, section 3 pip rule | Committed 2026-11-12 |
| B-4 Tile back skins | Four skins: storm, phoenix, frost, sema | Section 5.3 | Committed 2026-11-12 |
| B-5 Avatars in frames | Four painted portraits in four different frames: celestial, storm, phoenix, frost | Sections 5.1 and 5.2 | Committed 2026-11-12 |
| B-6 Carpet | The default carpet cloth | Section 5.4 | Committed 2026-11-12 |
| B-7 Type and icons | Score panel, hint, the four name effects, the icon row, cut from B-1 | Sections 5.6 and 5.7 | Committed 2026-11-12 |
| B-8 Effects storyboard | `tilePlace`, `scoreCascade` at steps 1, 3 and 6, `domino` and `block`, each at the low and high level | Section 6 and its budgets | Committed 2026-11-16 (#176), drawn from the effects atlas sprites by `work/b_8.py` |
| B-9 Static fallback and size test | One skin's static over its animated key frame with the shared pivot marked; avatar, frame, tile back and an icon at their smallest display size | Sections 3 and 7 | Committed 2026-11-22 (#167, #168), drawn by `work/b_9.py` from the `tile.sema` export, the tile shadow and the icon set: the static over key frame 9 with the centre pivot, the tile back at 32 x 64 px and the `settings` icon at 36 units on the smallest phone. Avatar and frame are not drawn: their smallest display size is open (section 10) |
| B-10 Do and don't | Paired examples for at least: baked text, baked glow, decoration in a team hue, casino cues, a recoloured frame or skin, frame 0 fallback | Section 8 | Committed 2026-11-12, drawn from the sample skin `tile.sema` |

A cosmetic's palette is reviewed per asset on the board: shown on both `color.surface.default` and `color.surface.raised` and beside `color.role.teamA` and `color.role.teamB`, so it never reads as a team marker.

## 10. Open items

- Fonts in the client: done in #592 (ui-motion section 3, "In the client"). The client ships the faces renamed (for example `HypeDomino Display 900 Italic`): Playfair Display and Josefin Sans carry a Reserved Font Name, and a static instance cut from the variable font is a Modified Version under the OFL.
- Name letter effects: whether the effect follows the frame (the proposal of section 5.6) or is its own item, and the draw call cost of one material preset per effect against the three-preset rule of ui-motion section 3. Measured before the effects ship.
- Iconsax: the npm package declares the MIT licence; the terms for the icon artwork on the vendor's site are confirmed by the owner before the icons ship. Which icons are used is #168.
- Placement of the face-down rows and of the name and timer inside the seat box: #168 and #169 with `UI/Match/MatchLayout`.
- Smallest display size, any outline weight and the avatar frame band per asset class: #168 to #175. Documented so far: the tile and its back skins (`docs/tiles.md` section 3) and the icons (`docs/ui-kit.md` section 3). The avatar and the frame have only the upper cap of 5.2; their smallest display size is still open, and B-9 adds them when it is set.
- Board panel B-9: committed 2026-11-22 without the avatar and the frame, which wait for their smallest display size (above); B-8 was committed with #176. Provenance of the prototype art used on B-1 to B-7 (generator, account, prompts) is recorded before any of it becomes a shipped item, as `tile.sema` started in `assets-src/cosmetic/tile/sema/provenance.md` (#167).
