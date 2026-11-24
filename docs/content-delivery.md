# Content delivery

How the client gets art that is not in the install package: where it lives on the CDN, how the client learns which content is live, what it does offline, how it downloads and retries, and when it frees memory. The decision behind it is ADR 0002 (Addressables, CDN, ASTC). Addresses, groups and labels of cosmetics are `docs/cosmetics.md` 8. This document is #68; its section 7, the disk cache, is #160.

## 1. Install-time and remote content

| Id | Rule |
|---|---|
| CD-1 | The install package holds the boot path, the match scene essentials, the default tile set and the `cosmetic-builtin` group (`docs/cosmetics.md` 8.2). With these alone the game starts, plays a match and shows every screen. |
| CD-2 | Every other Addressables group is remote: built with the catalog, uploaded to the CDN and fetched on demand. Install-time groups never point at the CDN. |
| CD-3 | Remote files are immutable. A bundle file name carries the hash of its content, so a new build of an item is a new file and an unchanged item keeps its file across catalog versions. |

## 2. Where content lives

The server announces two values in an accepted `ServerHello` (`proto/README.md`): `catalog_version`, the active catalog, and `content_base_url`, the CDN root of the environment (`HD_CATALOG_VERSION`, `HD_CONTENT_BASE_URL` in `docs/environments.md` 2.1). The client never builds a CDN host from a constant.

| Path under `content_base_url` | Content |
|---|---|
| `addressables/<platform>/catalog_<catalog_version>.bin` | The catalog of one version |
| `addressables/<platform>/<bundle file>` | Every remote bundle of every version, by its content-hashed name (CD-3) |
| `strings/<language>/<hash>.json` | Every published language bundle, by the SHA-256 hex of its content (`docs/localization.md` 2.2) |
| `cosmetics/<hash>.json` | Every published cosmetics manifest, by the SHA-256 hex of its content (`docs/cosmetics.md` 7.3) |

`<platform>` is `Android` or `iOS`. `content_base_url` is an `https` URL without a trailing slash, query or fragment. `catalog_version` is 1 to 64 letters, digits, `.`, `_` or `-`, starting with a letter or digit, because it is part of a file name and a URL; the server refuses any other value at start. Language bundles share the same root (`docs/localization.md` 2.2). The admin API uploads each one when it is published, with `Content-Type: application/json` and the immutable `Cache-Control` of CD-29, to the content bucket of its environment (CD-27) named by `HD_CONTENT_STORE_*` (`docs/environments.md` 2.2). `content-gc` never touches them (CD-34). Cosmetics manifests are uploaded the same way by every admin publish, rollback and switch-off of the manifest.

## 3. Catalog version

| Id | Rule |
|---|---|
| CD-4 | Start never waits for the network. Addressables starts with the install-time catalog, then loads the last remote catalog the device stored, if any, from disk (CD-6). |
| CD-5 | After an accepted `ServerHello` with a non-empty `catalog_version` and `content_base_url`, the client compares the version with the stored one. A different version is downloaded, stored and loaded. The same version does nothing. An empty value keeps what is loaded. |
| CD-6 | The client stores one remote catalog: the last one it loaded, with its version. A new one replaces it only after it downloaded completely, so a failed download keeps the previous one. |
| CD-7 | A new catalog is loaded outside a match only, like language bundles (`docs/localization.md` 2.2). Received during a match, it is loaded when the match ends. Art already on screen keeps its loaded handle; new loads resolve against the new catalog. |
| CD-8 | A rollback is the server announcing an older version (`docs/launch-runbook.md` 5.4). The client treats it as any other different version. |
| CD-9 | A new item published with a new catalog shows without a store update: its address resolves once the catalog is loaded (#68 acceptance). |

## 4. Download

| Id | Rule |
|---|---|
| CD-10 | A remote asset downloads when a view first needs it. Nothing is prefetched in bulk; a screen that must show content before it opens (the match loading screen, #85) asks for its addresses and shows their progress. |
| CD-11 | Progress is downloaded bytes over the total download size of the requested addresses. Addresses already on disk count as done. |
| CD-12 | A failed catalog download is retried after 2 s, 8 s and 30 s, then fails (CD-14). A failed bundle download is retried twice at once by Addressables, then fails (CD-13). A catalog request that has not finished in 20 s, or a bundle request with no progress for 20 s, counts as failed. These are engineering limits, not design values, so they are constants in the client. |
| CD-13 | A failed cosmetic shows its static fallback, or the built-in static of its type when the static itself failed (`docs/cosmetics.md` 8.3). It is asked for again the next time a screen opens, never in a loop. Input never waits for a download. |
| CD-14 | A failed catalog download keeps the stored catalog and is tried again at the next accepted `ServerHello`. |

## 5. Releasing memory

| Id | Rule |
|---|---|
| CD-15 | Every load is paired with exactly one release. Views do not load directly; they acquire from a shared cache that counts references (`CosmeticArtCache`, `docs/cosmetics.md` 8.3) and release when hidden or reused. |
| CD-16 | The asset is released when its last reference goes. Addressables then unloads the bundle when no loaded asset needs it. |
| CD-17 | A screen releases everything it acquired when it closes. A match releases its seats' art when the match scene unloads. |
| CD-18 | Install-time content follows the same rules; only its download step is missing. |

## 6. Release build

| Id | Rule |
|---|---|
| CD-19 | A release build is made for one catalog version. It names the catalog `catalog_<catalog_version>.bin` and fails when the version does not match 2 or the catalog file is missing after the build. |
| CD-20 | One build makes both the install package and the remote folder `client/ServerData/<platform>`. They are one release candidate: the package's install-time catalog and the remote folder come from the same content. |
| CD-21 | Every release candidate's install size is measured and gated as `docs/performance.md` 3 says. The remote folder is listed beside it, not counted. |

On Android the `Release build` workflow (`.github/workflows/release-build.yml`) runs `HypeDomino.Editor.ReleaseBuild.BuildAndroid` for a given catalog version. It keeps the AAB, the remote folder and the size report as artifacts. A `client-v<version>` tag runs it with the catalog version `<version>-<run number>` and uploads the signed AAB to Play internal testing (`client/README.md`, Release build).

## 7. Disk cache

Remote bundles are kept in Unity's AssetBundle cache under the app's data folder; install-time bundles are part of the install and never in it. This section is #160.

| Id | Rule |
|---|---|
| CD-22 | The cache holds at most 1,000,000,000 bytes. When a new bundle does not fit, Unity first removes the bundles unused the longest. |
| CD-23 | A bundle unused for 30 days is removed. |
| CD-24 | After a remote catalog loads, at start or after a new version (CD-5), every cached bundle that no loaded catalog references is removed, so old versions of an item do not stay on disk. Nothing is removed when the stored catalog failed to load. |
| CD-25 | The cache size shown to the player is the cached bundles plus the files in the content folder: the stored catalog, the kept language bundles (`docs/localization.md` 2.4) and the kept cosmetics manifest (`docs/cosmetics.md` 7.3). |
| CD-26 | The player may clear the cache outside a match only. A clear removes every cached bundle and keeps the stored catalog, the kept language bundles and the kept cosmetics manifest. A bundle in use blocks it, so the clear runs at the next start instead, before any content loads, and the player is told so; a later clear that succeeds cancels the one waiting for the next start. Art needed afterwards downloads again the first time a view asks for it (CD-10). |

The limit and the 30 days were set by the owner on 2026-11-08. They protect the device's storage and are not design values, so they are constants in the client.

What the player is told after a clear (CD-26) is one game string per outcome (`docs/localization.md` 2.1), named by `Game/CacheClearKeys`:

| Outcome | Key |
|---|---|
| Cleared now | `settings.cache.cleared` |
| A bundle is in use, so the clear runs at the next start | `settings.cache.cleared_at_next_start` |
| Refused in a match | `settings.cache.not_in_match` |

Their `en` and `az` texts ship in the built-in bundle (`docs/localization.md`, Cache keys), so a fresh install or an offline start can tell the player. Admin may publish other texts like for any other key; until `tr` and `ru` are translated, those languages fall back to `en` (`docs/localization.md` 1.2).

## 8. Bucket, upload and CDN

| Id | Rule |
|---|---|
| CD-27 | Each environment has its own bucket (`docs/environments.md` E-1), named `hypedomino-content-<environment>` with `<environment>` one of `dev`, `staging`, `production`. Its CDN host is the `content_base_url` of that environment. The key layout inside the bucket is section 2 without any prefix: `addressables/<platform>/catalog_<catalog_version>.bin` per version, and the content-hashed bundles beside it. |
| CD-28 | A catalog version is one catalog file with its `.hash` and `.json` companions. Bundles are shared by every version that references them and are never overwritten, so a client on an older catalog keeps resolving every bundle it needs. A bundle is deleted only when no catalog version that a supported app version can load references it (`HD_MIN_APP_VERSION`). |
| CD-29 | Cache-Control is set on upload. A content-hashed bundle (`<name>_<32 hex>.bundle`) gets `public, max-age=31536000, immutable`. Everything that is not content-addressed, the catalog and its companions, gets `public, max-age=300`, so a re-announced or rolled-back version is seen within minutes. A `.bundle` file without a content hash in its name is refused (CD-3). |
| CD-30 | Compression: bundles and the catalog are already compressed by the build (LZ4 chunks, ASTC textures), so they are uploaded as they are with no `Content-Encoding`. The CDN compresses text types only (`.json`, `.hash`) with Brotli and falls back to Gzip. |
| CD-31 | The CDN serves HTTP/2 and HTTP/3 with TLS 1.2 or newer, honours the `Cache-Control` of the origin without overriding it, caches by path ignoring the query string, and leaves `.bundle` and `.bin` unmodified (no image optimisation, no minify, no Brotli on binary types). A second request for a hashed bundle must be a CDN hit (`CF-Cache-Status: HIT` or the equivalent header of the provider); this is checked after every CDN change. |
| CD-32 | The upload tool is `server/cmd/content-publish`. It uploads bundles first and the catalog last, so no catalog is visible before its bundles; it skips a hashed bundle that already exists; it refuses a catalog version that already exists (a version is never replaced, a fix is a new version); and it refuses an `-environment` that does not appear in the store URL, or a store URL that names another environment. It signs requests with `server/internal/objectstore`, the package the admin asset upload uses for signed URLs. |

```
content-publish -dir client/ServerData/Android -environment staging -platform Android -catalog-version <catalog_version> [-dry-run]
```

The tool reads `HD_CONTENT_STORE_URL`, `HD_CONTENT_STORE_REGION`, `HD_CONTENT_STORE_ACCESS_KEY` and `HD_CONTENT_STORE_SECRET_KEY`. The `Release build` workflow runs it for a `client-v*` tag after the build when the repository variable `CONTENT_STORE_URL` is set; without it the step only warns. The names the owner provides are in `docs/environments.md` 2.3.

### 8.1 Bundle garbage collection

| Id | Rule |
|---|---|
| CD-33 | The bundle garbage collector is `server/cmd/content-gc`. It applies CD-28 to one platform of one environment. A catalog version is in scope when it is the active one (`HD_CATALOG_VERSION`), when its app version is at least the minimum app version, or when its name carries no app version. The app version of a catalog is the `<major>.<minor>.<patch>` of a release catalog version `<major>.<minor>.<patch>-<build number>` (section 6). A content-hashed bundle that no catalog in scope references and that is older than the grace period is unreferenced. |
| CD-34 | It deletes nothing unless `-delete` is given. It never deletes a catalog file or its companions, nor any object other than a content-hashed bundle directly under `addressables/<platform>/`. It refuses to run when a catalog in scope cannot be read, names no content-hashed bundle, or has an invalid version in its file name, and when the active catalog is not in the store: a read error never counts as "nothing references it". It checks the environment against the store URL as CD-32 does. |
| CD-35 | References are read from the binary catalog (`catalog_<catalog_version>.bin`): every length-prefixed string that is a content-hashed bundle file name. A bundle a catalog in scope references but the store does not hold is reported as missing. |

```
content-gc -environment staging -platform Android -active-catalog-version <catalog_version> -min-app-version <version> [-grace 24h] [-delete]
```

| Flag | Default | Meaning |
|---|---|---|
| `-environment` | none | `dev`, `staging` or `production`; must match the store URL |
| `-platform` | `Android` | `Android` or `iOS` |
| `-active-catalog-version` | `HD_CATALOG_VERSION` | The catalog version the server announces now |
| `-min-app-version` | `HD_MIN_APP_VERSION` | The oldest supported app version: the `minimum_app_version` of the published `live_controls` snapshot (`docs/anti-cheat.md` 5.2), or `HD_MIN_APP_VERSION` until one is published |
| `-grace` | `24h` | An unreferenced bundle younger than this, by the store's last-modified time, is kept, so a bundle that a publish in progress just uploaded is never deleted. Operational setting, not game config |
| `-delete` | off | Delete the unreferenced bundles. Without it the run is a dry run |

It reads the same `HD_CONTENT_STORE_*` variables as `content-publish`; a dry run needs them too, because it lists and reads the bucket. It prints every catalog with its scope and reason, every bundle it keeps (referenced or young), every missing bundle and every unreferenced bundle, then the counts.

When to run it: after the minimum app version rises (`docs/launch-runbook.md` 5.2, 5.4), first as a dry run whose output is checked, then with `-delete`, once per platform. Never while `content-publish` runs for the same bucket: a publish skips a bundle that already exists (CD-32), and that bundle can still look unreferenced until the new catalog lands. After a run, a catalog version below the minimum app version may have lost bundles, so it is no longer a rollback target (`docs/launch-runbook.md` 5.4). Scheduling it and purging deleted files from the CDN cache are not part of the tool; a deleted bundle is one no supported client asks for.

## 9. Not here

- Activating an uploaded catalog version (`HD_CATALOG_VERSION`): `docs/launch-runbook.md` 5.4.
- The iOS release build and its App Thinning Size Report: with the iOS pipeline.
- The settings row that shows the cache size and the clear button: #157 (`ContentBoot.Cache`).
