# UI kit

Status: proposal; the owner approves it by merging. The sprites of #168: buttons, panels, tabs, the elevation shadows and other 9-slice pieces, the interface icon set and the rarity status diamond, how they are drawn, sized and atlased, and what the client still has to do. Values come from `docs/design-tokens.json` and their use from `docs/ui-motion.md` sections 2, 4 and 5; look from `docs/art-direction.md` sections 2 to 5.7; the diamond from `docs/rarity.md` section 2.1. This file defines no colour, radius, duration or price.

## 1. Items

| Item | Source folder | Made by | Client folder |
|---|---|---|---|
| Kit: buttons, panels, tabs, shadows, backdrop, pill, divider, icon shadow | `assets-src/ui/kit/` | `work/kit.py` from the tokens | `client/Assets/HypeDomino/UI/Kit/Panels/` |
| Icons, 127 | `assets-src/ui/icons/` | `work/icons_extract.py` and `work/icons_rasterize.py` from the Iconsax package | `client/Assets/HypeDomino/UI/Kit/Icons/` |
| Rarity diamond and pip overlays | `assets-src/ui/rarity/` | `work/rarity.py` | `client/Assets/HypeDomino/UI/Kit/Rarity/` |

Each folder has a `provenance.md` with the exact commands. Every script is deterministic: two runs into empty folders give byte-identical files. The client folders hold copies of the `export/` PNGs (and `kit.json`); after a script changes an export, the copy is refreshed under the same file names and `HypeDomino/Import UI Kit` is run (section 5).

## 2. Kit sprites and 9-slice borders

Drawn at 2 px per UI unit and imported with pixels per unit 200 against the canvas reference of 100, so one sprite pixel is half a UI unit and corners stay sharp up to a 2160-pixel short side. Borders are in sprite pixels in Unity's order (left, bottom, right, top); `client/Assets/HypeDomino/UI/Kit/Panels/kit.json` holds them for the importer. Colours are read from the tokens at run time; sprites listed as white are tinted through vertex colour.

| Sprite | Pixels | Units | Border left, bottom, right, top (px) | Use |
|---|---|---|---|---|
| `backdrop_soft` | 136 x 128 | 68 x 64 | 64, 60, 64, 60 | White soft pill behind seat names and hints; tint color.scrim at opacity.scrim |
| `button_primary` | 256 x 240 | 128 x 120 | 120, 116, 120, 116 | Primary pill, role.primary fill with the gold gradient |
| `button_secondary` | 80 x 240 | 40 x 120 | 36, 116, 36, 116 | Secondary button, radius.md, role.secondary fill |
| `button_tint` | 80 x 240 | 40 x 120 | 36, 116, 36, 116 | Greyscale button, radius.md, tinted with a role fill |
| `divider` | 128 x 8 | 64 x 4 | 0, 0, 0, 0 | Gold divider line, stretched as a simple sprite |
| `icon_shadow` | 96 x 96 | 48 x 48 | 0, 0, 0, 0 | White soft shadow behind an icon; tint color.shadow |
| `panel` | 120 x 120 | 60 x 60 | 56, 56, 56, 56 | Panel, radius.lg, surface.default lacquer with a gold inlay line |
| `panel_raised` | 88 x 88 | 44 x 44 | 40, 40, 40, 40 | Panel inside a panel, list row, input; radius.md, surface.raised |
| `pill_fill` | 136 x 128 | 68 x 64 | 64, 60, 64, 60 | White crisp pill: toggle track, progress bar, badge, dot |
| `shadow_lg` | 168 x 168 | 84 x 84 | 80, 80, 80, 80 | White elevation shadow for radius.lg owners: panels; tint color.shadow at the elevation alpha, rect grown by 8 units per side |
| `shadow_md` | 136 x 136 | 68 x 68 | 64, 64, 64, 64 | White elevation shadow for radius.md owners: secondary and tinted buttons, raised panels, tabs; tint color.shadow at the elevation alpha, rect grown by 8 units per side |
| `shadow_pill` | 312 x 272 | 156 x 136 | 152, 132, 152, 132 | White elevation shadow for pill owners: the primary button; tint color.shadow at the elevation alpha, rect grown by 8 units per side |
| `tab_off` | 88 x 88 | 44 x 44 | 40, 40, 40, 40 | Tab at rest, radius.md, surface.default with a gold.lo rim |
| `tab_on` | 88 x 88 | 44 x 44 | 40, 40, 40, 40 | Selected tab, radius.md, surface.raised with the gold rim |

Use:
- Button rest, pressed, released and disabled follow `docs/ui-motion.md` section 5: the press scale and `opacity.pressedTint` are a transform and a vertex tint, disabled is `opacity.disabled`, and the one primary button's `glow.soft` is the shared runtime glow sprite. None of these states is a separate sprite.
- `button_primary` is the only pill button. `button_secondary` is the gold secondary button. `button_tint` is greyscale with painted light and takes a role `fill` as its colour, for `success`, `warning`, `danger` (with an icon, ui-motion section 5) and the team roles.
- A shadow sits under its owner as its own `Image`, its rect the owner's grown by 8 units on every side and moved down by the elevation's `offsetY`, tinted `color.shadow` at the elevation's `alpha`: e1 for buttons, e2 for panels (`docs/ui-motion.md` section 4). `shadow_md` goes with radius md owners, `shadow_lg` with panels, `shadow_pill` with the primary button.
- Pills: Unity scales 9-slice borders per axis when a rect is smaller than its borders, which would flatten a pill's ends. A pill shorter than its sprite (120 units for `button_primary` and `shadow_pill`, 64 for `pill_fill` and `backdrop_soft`) sets `Image.pixelsPerUnitMultiplier` to the sprite height over the rect height, in units, so the ends stay round. A square `pill_fill` rect is a circle: badges and dots.
- `backdrop_soft` is the soft backdrop behind seat names and hints, tinted `color.scrim` at `opacity.scrim` (art-direction section 2).
- `divider` and `icon_shadow` have no border and are stretched or placed as simple sprites. `icon_shadow` is the soft dark shadow behind an icon on the carpet (art-direction 5.7), tinted `color.shadow`.

## 3. Icons

Iconsax, bulk style only (`docs/art-direction.md` 5.7), from the npm package `iconsax` 0.1.1. The package declares the MIT licence and ships no LICENSE file; the vendor's terms for the icon artwork are an owner item before the icons ship (`docs/art-direction.md` section 10). The package is not a dependency of the repository: the selected SVGs are kept verbatim in `assets-src/ui/icons/work/icons_source.json`.

- Canvas 96 x 96 px, imported at pixels per unit 100. RGB white; the solid tone has alpha 1 and the bulk second tone keeps the vendor's opacity as alpha, so one vertex-colour gold gradient (`color.gold.hi` to `color.gold.lo`) tints both tones.
- Smallest display size 36 units, the size of the icon row of the approved prototype B-1 (emoji, sound and settings). 96 px stays sharp up to 96 units on a 1080-pixel short side. An icon button keeps its touch area at `size.touchTargetMin` whatever the icon size.
- No icon carries letters, digits, currency signs, brand logos or seasonal or religious motifs; the candidates that did (a `$` coin, a `24` support badge, `%` offer and promo tickets, a language icon with a CJK glyph) were replaced.
- Motion is ours, by transform tweens (art-direction 5.7); no icon has frames.

Which screen uses which icon is decided by the screens; the list below is the set the specs imply today (settings, audio, social, shop and economy, pass, missions, medals, ranking, match, cosmetics). The client has no icon references yet, so none was taken from code. A new icon is added to `work/icons_list.json` and both scripts are run again.

| Key | Iconsax bulk source | Use |
|---|---|---|
| `home` | `home-2` | Home screen tab |
| `back` | `arrow-circle-left` | Back on every pushed screen |
| `forward` | `arrow-circle-right` | Next, open detail |
| `close` | `close-circle` | Close a modal or panel |
| `menu` | `menu` | Main menu |
| `more` | `more` | More actions on a row |
| `add` | `add-circle` | Add, increase |
| `remove` | `minus-circle` | Remove, decrease |
| `confirm` | `tick-circle` | Confirm, success, done |
| `check` | `tick-square` | Checkbox on |
| `refresh` | `refresh-circle` | Retry, reload a list |
| `search` | `search-normal` | Search friends or items |
| `filter` | `filter` | Filter a list |
| `sort` | `sort` | Sort a list |
| `info` | `info-circle` | Information, details |
| `help` | `message-question` | Help, how to play |
| `warning` | `warning-2` | Warning role notices |
| `danger` | `danger` | Danger role notices, destructive confirm |
| `share` | `share` | Share an invite or result |
| `copy` | `copy` | Copy a code or link |
| `paste` | `clipboard-import` | Paste a promo or lobby code |
| `link` | `link-2` | Invite link, account link |
| `edit` | `edit-2` | Edit nickname or a field |
| `delete` | `trash` | Delete |
| `lock` | `lock` | Locked reward, locked item |
| `unlock` | `unlock` | Unlocked reward |
| `visible` | `eye` | Show, spectate |
| `hidden` | `eye-slash` | Hide |
| `download` | `import-circle-01` | Content download |
| `settings` | `setting-2` | Settings |
| `sound` | `volume-high` | Sound effects on |
| `sound_off` | `volume-slash` | Sound effects off |
| `music` | `music` | Music volume |
| `music_off` | `music-square-remove` | Music off |
| `haptics` | `mobile` | Haptics setting |
| `effects` | `magic-star` | Effect level setting |
| `graphics` | `monitor` | Graphics level setting |
| `frame_rate` | `speedometer` | Frame rate setting |
| `battery` | `battery-charging` | Battery saver |
| `reduce_motion` | `flash-slash` | Reduce motion setting |
| `language` | `global` | Language setting |
| `notifications` | `notification` | Push notification settings |
| `notification_new` | `notification-bing` | New notification |
| `privacy` | `shield-tick` | Privacy settings and consent |
| `account` | `security-user` | Account and sign-in |
| `sign_in` | `login-01` | Sign in |
| `sign_out` | `logout-01` | Sign out |
| `delete_account` | `profile-delete` | Delete account |
| `support` | `headphone` | Contact support |
| `rules` | `book-open` | Game rules |
| `document` | `document-text` | Terms and privacy texts |
| `toggle_on` | `toggle-on-circle` | Switch on |
| `toggle_off` | `toggle-off-circle` | Switch off |
| `profile` | `profile-circle` | Own profile |
| `user` | `user` | A player |
| `friends` | `people` | Friends, 2v2 mode |
| `duel` | `profile-2user` | 1v1 mode |
| `friend_add` | `user-add` | Add a friend |
| `friend_remove` | `user-remove` | Remove a friend |
| `friend_accept` | `user-tick` | Accept a friend request |
| `friend_search` | `user-search` | Find a player by code |
| `invite` | `direct-send` | Invite to a lobby |
| `block` | `forbidden` | Block a player |
| `report` | `shield-cross` | Report a player |
| `chat` | `message-text` | Quick messages |
| `chat_off` | `message-remove` | Mute a seat's messages |
| `messages` | `messages-2` | Message list |
| `emoji` | `emoji-happy` | Emoji wheel |
| `emoji_set` | `smileys` | Emoji collection |
| `mute` | `volume-mute` | Mute a seat |
| `online` | `record-circle` | Online state |
| `world` | `global-search` | Global leaderboard, region |
| `shop` | `shop` | Shop tab |
| `bag` | `bag-2` | Purchases, bundles |
| `coin` | `buy-crypto` | Coin currency |
| `crystal` | `diamonds` | Crystal currency |
| `chest` | `strongbox` | Chest |
| `chest_opened` | `strongbox-2` | Opened chest |
| `gift` | `gift` | Reward, gift |
| `wallet` | `empty-wallet` | Balances |
| `offer` | `bag-happy` | Special offer |
| `promo_code` | `ticket` | Promo code entry |
| `price` | `tag` | Price tag |
| `receipt` | `receipt-item` | Purchase history |
| `restore` | `refresh-right` | Restore purchases |
| `inventory` | `category` | Collection, inventory |
| `pass` | `ticket-star` | Season pass |
| `missions` | `task` | Missions tab |
| `mission_done` | `clipboard-tick` | Mission completed |
| `medal` | `medal-star` | Medals tab |
| `award` | `award` | Reward tier |
| `trophy` | `cup` | Match win, ranked results |
| `crown` | `crown` | Rank top, score crown |
| `ranking` | `ranking` | Leaderboard |
| `level` | `level` | Level |
| `rank_up` | `trend-up` | Rank up, positive delta |
| `rank_down` | `trend-down` | Rank down, negative delta |
| `stats` | `status-up` | Statistics |
| `star` | `star` | Favourite, rating |
| `heart` | `heart` | Like |
| `streak` | `flash` | Streak, boost |
| `calendar` | `calendar` | Season dates |
| `daily` | `calendar-tick` | Daily reward |
| `timer` | `timer` | Time left |
| `timer_start` | `timer-start` | Countdown start |
| `clock` | `clock` | Match history, time |
| `hourglass` | `hourglass` | Waiting, queue |
| `play` | `play-circle` | Play |
| `pause` | `pause-circle` | Pause |
| `pass_turn` | `forward` | Pass the turn |
| `draw` | `import-arrow-01` | Draw from the boneyard |
| `end_turn` | `next` | End the turn after a scoring double |
| `leave` | `logout-02` | Leave the match |
| `reconnect` | `refresh-arrow-01` | Reconnecting |
| `connection` | `wifi` | Connection state |
| `connection_weak` | `wifi-square` | Weak connection |
| `bot` | `cpu` | Bot playing a seat |
| `hint` | `lamp-on` | Hint |
| `matchmaking` | `search-status` | Finding a match |
| `lobby` | `home-hashtag` | Friend lobby |
| `avatar` | `user-square` | Avatar cosmetics |
| `frame` | `picture-frame` | Frame cosmetics |
| `table` | `grid-5` | Table cosmetics |
| `tile_skin` | `layer` | Tile back cosmetics |
| `name_effect` | `magicpen` | Name letter effect |
| `customize` | `brush` | Customize look |
| `cloud` | `cloud-change` | Cloud save, sync |

## 4. Rarity diamond

`RarityDiamond` takes a diamond `Image`, a pip `Image` and `Sprite[] pipOverlays` indexed by pip count (`docs/rarity.md` 2.2.1), so the diamond is a sprite, not procedural geometry. `assets-src/ui/rarity/export/diamond.png` is a white square on its corner with greyscale facets lit from above; `pips_1.png` to `pips_6.png` are white dots on the domino-half layout turned 45 degrees with the square (`docs/rarity.md` 2.1), 6 for the reserved sixth tier. All seven share one 192 x 192 canvas and the centre pivot, so the overlay lies on the diamond with no offset. The diamond is tinted with the tier's `fill` and the overlay with its `on`; the client hides both under `rarity.diamondMinSize`. The halo and flow halo of `RarityHalo` are effect sprites of the shared effects atlas (#176), not part of this kit.

## 5. Atlas and import

- One UI atlas, `client/Assets/HypeDomino/UI/Kit/UiKit.spriteatlasv2`, packs the whole `UI/Kit` folder: kit, icons and rarity marks, so a screen's kit pieces, icons and diamonds batch in one draw call with the default UI material. Effect sprites stay in the shared effects atlas (`docs/art-direction.md` section 6).
- `HypeDomino/Import UI Kit` (`HypeDomino.Editor.UiKitContent`; batch `-executeMethod HypeDomino.Editor.UiKitContent.ImportAllAndExit`, also through the `unity-editor.yml` workflow) imports every PNG under `UI/Kit` as a single sprite with a full rect mesh, no mipmaps, clamp, sRGB, alpha as transparency and ASTC with the block of `ProjectSetup.SpriteFormat` on Android and iOS; sets the border and pixels per unit of every sprite listed in `kit.json`; and builds the atlas without rotation or tight packing (a UI `Image` needs both off), padding 4, largest page 2048, ASTC. It can be run again after any change. `UiKitContentTests` (EditMode) check those settings on the committed assets.
- The atlas is not an Addressables entry: the kit ships in the app and is packed into a build only through the screens that reference its sprites.
- Size, by the ADR 0012 rule (16 bytes per ASTC 6 x 6 block of every sprite, summed): icons 520,192 B, kit 142,912 B, rarity 114,688 B, together 777,792 B. With 4 px padding the sprites cover about 1.87 million pixels, which would fill a 2048 x 1024 page to about 89%, so the packer is expected to need one 2048 x 2048 page (1,871,424 B in ASTC 6 x 6). The page is measured in a build before the screens ship, with the measurements of `docs/performance.md` section 8.

## 6. Not done

- Screens: no screen of the client draws a kit sprite, an icon or the diamond yet. The client has no screen prefabs; each screen sets its sprites when it is built, and the `RarityDiamond` and `RarityHalo` prefab work of `docs/rarity.md` 2.2.1 is client work. The gold icon gradient, the press and disabled tints and the primary glow are runtime work of the screens.
- Review: the look on the reference board (B-9, smallest sizes), the cultural review of the icon set, and the vendor's artwork terms (`docs/art-direction.md` section 10).
- Measurement: the atlas page size and the batch count of a real screen.
