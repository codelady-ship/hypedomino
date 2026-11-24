# Performance budget and measurement method

Budgets come from `CLAUDE.md` and are not changed here; this file fixes how each is measured and when it passes. A budget without a passing measurement is not met. Tool and harness work is tracked in #192 (load bot, replay harness); device certification is #191; battery measurement is #164. Decisions that affect these paths are recorded in `docs/adr/`.

## 1. Device matrix

Initial reference matrix. #191 certifies on it; changing a model needs a PR to this table.

| Tier | Android reference | SoC / GPU / RAM / OS | iOS reference | Role |
|---|---|---|---|---|
| Low | Samsung Galaxy A14 4G | Helio G80, Mali-G52, 4 GB, Android 13 | iPhone XR (A12, 3 GB, iOS 17) | Playable at 30 fps minimum, reduced effects |
| Mid | Samsung Galaxy A34 5G | Dimensity 1080, Mali-G68, 6 GB, Android 14 | iPhone 12 (A14, 4 GB, iOS 17) | Gating tier: 60 fps and all start, draw call, GC, size, battery gates |
| High | Samsung Galaxy S24 | Snapdragon 8 Gen 3, Adreno 750, 8 GB, Android 14 | iPhone 15 Pro (A17 Pro, 8 GB, 120 Hz ProMotion) | 120 fps option |

Class rule for devices outside the matrix: Low = under 4 GB RAM or GPU older than Mali-G57 / Adreno 610 class; Mid = 4 to 8 GB with Mali-G68 / Adreno 619 class or better; High = 8 GB or more, flagship SoC, 120 Hz panel. The tier decides the graphics level (1.1); Low runs the reduced-effects level.

### 1.1 Graphics levels

The player picks Auto (the default) or one of four graphics levels in settings (#157). A level sets the effect level (`docs/ui-motion.md` section 7) and the resolution the camera renders at. Values are admin config (`effects.graphics` in `docs/design-tokens.json`), with the defaults below.

| Graphics level | Effect level | Camera render short side, at most |
|---|---|---|
| Low | low | 720 px |
| Medium | medium | 1080 px |
| High | high | 1080 px |
| Ultra | high | 2160 px |

The render scale is `min(1, cap / native short side)` and is set on the one URP asset (`renderScale`); there is one URP asset for every level. UI on screen-space overlay canvases always renders at the native resolution, so text stays sharp. Ultra differs from High only on a panel whose short side is over 1080 px. The frame rate is #158 and does not change the level.

| Device tier | Auto picks | Highest the player may pick |
|---|---|---|
| Low | Low | Low |
| Mid | Medium | Medium |
| High | High | Ultra |

The player may pick any level from Low up to the highest; a higher one is shown locked (`docs/ui-motion.md` section 7). A saved choice above the highest, for example after the class rule changes, counts as the highest.

**Device tier.** The client applies the class rule above at every start, from `SystemInfo.systemMemorySize`, `SystemInfo.graphicsDeviceName` and the highest refresh rate in `Screen.resolutions` (`effects.graphics.deviceClass`):

| Tier | Rule |
|---|---|
| Low | Memory under 3200 MB (a 4 GB device reports less than 4096 MB), or the GPU name matches a low pattern |
| High | Memory 7000 MB or more, the GPU name matches a high pattern, and a refresh rate of 120 Hz or more |
| Mid | Every other device, including a GPU between the Low and Mid classes; the probe below lowers it if it cannot carry Medium |

| Patterns | Default (regular expressions on the GPU name) |
|---|---|
| Low | `^Mali-(T\d+\|G31\|G51\|G52\|G71\|G72)\b`, `^Adreno \(TM\) ([1-5]\d\d\|60\d)\b`, `^PowerVR Rogue` |
| High | `^Adreno \(TM\) (7[3-9]\d\|8\d\d)\b`, `^Mali-G7[1-2]\d\b`, `^Immortalis`, `^Samsung Xclipse`, `^Apple (A1[5-9]\|A[2-9]\d\|M\d)\b` |

#191 checks the rule on every certified device; a device in the wrong tier is fixed by a PR to the patterns or thresholds.

**Probe.** The class rule is a guess; the first matches check it. While the choice is Auto, a match measures work time (section 2) with `FrameTimingManager` over 1800 frames after 300 warm-up frames (`probe`). If work time p95 is over the `workTimeP95Ms` of the level's effect level, Auto drops one level for the next match, the dropped level becomes the highest the player may pick, and the probe runs again at the new level. Once a window passes, the probe stops. A new app version clears the drops and runs the probe again. The probe never raises a level, never drops below Low, and makes no change on a device without frame timing data.

**Thermal.** When the device reports heat, the level in use drops without changing the saved choice. Android 10 and later report it through the `PowerManager` thermal status listener; Android 7.1 to 9 give no signal and never drop. iOS reports `ProcessInfo.thermalState`.

| Status | Android | iOS | Level in use |
|---|---|---|---|
| Normal | none, light | nominal, fair | The chosen level |
| Moderate | moderate | serious | One level lower, not below Low |
| Severe | severe, critical, emergency, shutdown | critical | Low |

A hotter status applies at once. A cooler one applies only after it has held for 60 s (`thermal.recoverAfterSeconds`), and any hotter reading in between restarts the wait, so the level does not flicker at a boundary. While the status is not normal, the probe does not run and a window that ends hot is ignored. The client watches the status on Android only for now; iOS is in section 9.

### 1.2 Frame rate

The player picks Auto (the default) or a frame rate in settings (#157). Values are admin config (`effects.graphics.frameRate` in `docs/design-tokens.json`, and `targetFps` of each effect level), with the defaults below.

| Rule | Default |
|---|---|
| Rates offered | 30, 60, 90, 120 |
| A rate is offered when | It is at most the `targetFps` of the effect level in use (low 30, medium 60, high 120), and a refresh rate the display reports in `Screen.resolutions` is a whole multiple of it, so frames are paced evenly |
| Auto | The highest offered rate up to 60. A 120 option is opt-in |
| Menus (lobby, shop, profile, settings) | At most 60. The chosen rate applies in a match only |
| Battery saver | 30 everywhere, while the in-game toggle or the system saver (Android `PowerManager.isPowerSaveMode`, iOS Low Power Mode `NSProcessInfo.isLowPowerModeEnabled`) is on. The graphics level does not change |

A saved rate above the highest one offered counts as the highest. The level in use decides the cap, so a thermal drop (1.1) lowers the rate too, and it comes back with the level.

The client sets `QualitySettings.vSyncCount` to 0 and `Application.targetFrameRate` to the rate in use, on every platform. On Android, Optimized Frame Pacing (Swappy, on in Player Settings) paces frames and asks the display for a matching refresh rate. The app flow tells the client when a match starts and ends: the chosen rate applies from entering the `Match` state until the app leaves it (`client/README.md`, App flow). The system saver is read at start and each time the app regains focus, because the quick settings shade and Control Center take focus; it is not polled during a match, so a JNI or native call never allocates in a match frame. On iOS it is read through `Plugins/iOS/HypeDominoPower.mm`.

## 2. Test conditions (all device measurements)

- Development build with the Unity Profiler attached, deep profiling off, for Profiler-based gates. Release build (IL2CPP, ARM64, no Profiler) for cold start, fps and battery gates.
- Device between 100% and 30% battery, not charging, Wi-Fi to the test server, brightness fixed at 50%, no other foreground apps, idle-cool (surface under 35 C) before each run, battery saver / Low Power Mode off.
- `Application.targetFrameRate` 60 (120 for the high option), vsync on. Player Settings "Frame Timing Stats" enabled so `FrameTimingManager` reports per-frame CPU and GPU time.
- With vsync on, the presented interval is quantised to the refresh period (16.7 ms at 60 Hz), so smoothness is judged by two numbers: work time (`max(cpuMainThreadFrameTime, cpuRenderThreadFrameTime, gpuFrameTime)` from `FrameTimingManager.GetLatestTimings`; not `cpuFrameTime`, which spans the end of one frame to the end of the next with every wait included, so it never drops below the refresh period, #445) shows headroom, and missed vsyncs (presented interval longer than 1.5 refresh periods) show visible stutter.
- Scripted match replay: a fixed-seed recorded 2v2 match of 3 minutes, server-fed, effects at the tier's default level, worst-case effect moments included. Played 3 times back to back; all 3 runs must pass and the worst is reported. Harness: #192.

## 3. Client budgets

| Budget | Target | Tier / environment | Method | Sample and duration | Pass / fail |
|---|---|---|---|---|---|
| Stable 60 fps | 60 fps | Mid (Low: section 4) | Replay harness records per frame `FrameTimingManager` CPU and GPU time and the presented interval (`Time.unscaledDeltaTime`), exported as CSV and analysed in Profile Analyzer. Cross-check presented frames with `adb shell dumpsys SurfaceFlinger --latency <layer>` or a Perfetto frame timeline | 3 replays x 3 min (about 10,800 frames each) | All hold: work time p50 <= 11.0 ms and p95 <= 14.0 ms; missed vsyncs (interval > 25.0 ms) <= 1% of frames; no interval over 100 ms |
| 120 fps option | 120 fps | High, 120 Hz panel, option enabled | Same as above, target 120 | 3 replays x 3 min | Work time p50 <= 5.5 ms and p95 <= 7.0 ms; missed vsyncs (interval > 12.5 ms) <= 2%; no interval over 50 ms |
| Cold start | < 3 s | Mid, release build | Command below | 10 runs, force-stop before each, first run after install discarded | Median time to interactive lobby < 3000 ms and worst run < 3500 ms. Low and High are recorded, not gated |
| Draw calls | < 150 | Mid, match scene | `ProfilerRecorder` on `ProfilerCategory.Render` "Batches Count" and "SetPass Calls Count" per frame; inspect the worst frame in Frame Debugger (Window > Analysis > Frame Debugger) | 1 full replay, worst-case effect frame included | Maximum Batches over all frames < 150 (maximum, not average) |
| GC allocation | 0 B/frame | Mid, match scene, development build | `ProfilerRecorder.StartNew(ProfilerCategory.Memory, "GC Allocated In Frame")` per frame; `GC.CollectionCount(0)` before and after; Profiler Memory module and Memory Profiler snapshot to find the source of any allocation | First deal to result screen, 3 replays | Every frame 0 B and `GC.CollectionCount(0)` delta 0. One non-zero frame fails |
| Canvas rebuild | p95 <= 1.5 ms per frame | Mid, development build, UI screens | Canvas bench (`client/README.md` "Canvas bench"): `ProfilerRecorder` on the `PostLateUpdate.PlayerUpdateCanvases` marker per frame over the `static`, `scroll`, `rebind` and `pushpop` phases, read with `adb logcat -s Unity` | 3 cycles x 4 phases x 1200 frames; cycle 1 is warm-up and not gated | Every phase of cycles 2 and 3: p95 <= 1.5 ms, and the `static` phase p95 <= 0.3 ms; `scroll` and `rebind` allocate 0 B. First run (Galaxy A36, Mid, 2026-11-10): static 0.27, scroll 0.98, rebind 1.38, pushpop 1.25 ms p95, pass |
| Install size | < 150 MB | Android AAB and iOS IPA, release | Unity Build Report (`Library/LastBuild.buildreport`, Build Report Inspector) for the content breakdown. Android: `bundletool build-apks --bundle=app.aab --output=app.apks --device-spec=<tier>.json` then `bundletool get-size total --apks=app.apks`. iOS: App Thinning Size Report from the Xcode archive | Every release candidate | Android: max download size over the low, mid and high device specs (`tools/install-size/`) < 150 MB, where 1 MB is 1,000,000 bytes; the `Release build` workflow measures it and fails over the budget (`docs/content-delivery.md` 6), also on every pull request that changes `client/`, as a warning check outside `CI complete`. iOS: compressed size for the mid and high models < 150 MB. Content fetched on demand through Addressables is excluded from this number but listed in the report |

Cold start (Android, run 10 times; `<activity>` is the launcher activity):

```bash
adb shell am force-stop <package>
adb shell am start -W -n <package>/<activity>
```

`am start -W` prints `TotalTime` (process start to first frame). The gate is time to interactive lobby: the client's startup marker in `adb logcat -s Unity` minus the launch time. The marker is the one `Startup state=<state> ms=<n>` line `Game/StartupMarker` logs when the app first leaves its boot state (`client/README.md`, App flow). Only runs whose line says `state=Lobby` count; a run that shows sign-in, an update, maintenance or a running match first is run again. `ms` is Unity's time since startup, which starts after the process does, so it is a lower bound and the gate keeps using the logcat timestamps. iOS: Xcode Instruments "App Launch" template, same 10-run rule.

To find where a slow start spends its time, read the `Startup phases` line logged right after the marker (`client/README.md`, App flow): the time since startup at which the scripts loaded, each boot before the first scene finished, the first scene loaded, `NetBoot` was composed, the sign-in resumed (`resume`), the session start proof arrived (`proof`), the socket opened (`dial`) and the first screen showed after the server hello. It is a profiling aid, not a gate; the gate stays the marker above.

Leak check, reported but not gated until #191 sets a number: Memory Profiler snapshots after replay 1 and replay 3; managed and native totals must not grow.

## 4. Low tier target

| Target | Method | Pass / fail |
|---|---|---|
| Playable at 30 fps minimum with reduced effects | Same replay on the Low reference devices, `targetFrameRate` 30, reduced-effects level | Work time p95 <= 28.0 ms; missed vsyncs (interval > 50.0 ms) <= 1%; no interval over 100 ms. Draw call and GC gates still apply. Move, draw, pass and timer fully functional |

## 5. Battery

Initial target, to be recalibrated after the #164 measurement.

| Target | Tier | Method | Pass / fail |
|---|---|---|---|
| 30-minute match session drains at most 8% battery | Mid reference devices, 60 fps, default effects, Wi-Fi | Android: `adb shell dumpsys batterystats --reset`, play 30 min of back-to-back matches (replay harness in live mode), `adb bugreport`, open in Battery Historian; read battery level delta, app estimated power use and end temperature. iOS: Instruments "Energy Log" over the same 30 min plus battery percentage delta | 3 sessions; median delta <= 8% and worst <= 10%. Until #164 reports, a miss is logged, not blocking |

## 6. Server budgets

Run on the reference node spec fixed in #192 and recorded with the results.

### 6.1 Reference run

- Node under test: a Hetzner Cloud server of the type in `HD_LOAD_NODE_TYPE`, location Falkenstein (`fsn1`), Ubuntu 24.04, one server process and nothing else. Two kinds of run use it:
  - Reference run: CCX23 (4 dedicated vCPU, 16 GB RAM), the default. Dedicated vCPU keeps a neighbour's load out of the numbers, so runs can be compared; the 10,000-room node target of 6.2 and every before and after comparison of section 8 are measured here.
  - Beta run: the shared-vCPU type the game node runs in production (ADR 0014: CX33, or CAX21 on arm64), with the stores host on the database node type (`HD_LOAD_STORES_TYPE=cx33`) and the bot host left on CCX23 so the bot's own scheduling does not count against the node. It proves the beta peak, not a capacity: 1,000 rooms at 50% 2v2 (`-rooms 1000`), about 3,000 seats, three times the CB-3 size of `docs/closed-beta.md` all in a match at once. Shared vCPU is not reproducible run to run, so `check.sh` also prints the average CPU steal of the hold (`st` of `vmstat`); record it with the result. A failed move round-trip gate here is the measurement trigger of ADR 0014: the game node starts on, or moves to, CCX23.
- Bot host: a server of the type in `HD_LOAD_BOT_TYPE` (default CCX23) in the same location, joined to the node by a Hetzner private network; the load bot never runs on the node. The debug port (`HD_DEBUG_ADDR`, `/metrics`) and the profiling port (`HD_PPROF_ADDR`, `/debug/pprof/`) listen on the private address only.
- Peak target: the issue's "expected first-days peak ×3" has no number yet. It is set from the CB-3 size after the closed beta (`docs/closed-beta.md` CE-14); until then the run proves the 10,000-room node target of 6.2.

Steps:

1. Build on any machine: `cd server && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o loadbot ./cmd/loadbot` (`GOARCH=arm64` for a CAX host), then copy `loadbot` and `tools/loadtest/*.sh` to both hosts.
2. Node: `./node.sh <node private address> node-run`. It raises the file limit, records the machine (`node.txt`), samples CPU every 10 s (`vmstat.log`) and runs `loadbot serve` with the same `HD_*` variables as the server.
3. Bot host: `./bot.sh <node private address> bot-run`. Defaults are those of 6.2: 10,000 rooms, 50% 2v2, 5 min ramp, 30 min hold; extra `loadbot run` flags are appended and override them. It samples `/metrics` every 60 s, keeps the bot report (`report.json`, `report.txt`), the final metrics and a heap profile, and exits non-zero when the bot gates fail. It passes `-metrics`, so the bot report also gates the node's live heap and goroutine count: each may grow at most 5% from the first sample after the ramp and the warm-up to the last. 10,000 rooms at 50% 2v2 open 30,000 connections from one address to one port, more than the default Linux ephemeral range (32768 to 60999, 28,232 ports), so widen it on the bot host first: `sysctl -w net.ipv4.ip_local_port_range="1024 65000"`.
4. Copy `node-run/vmstat.log` into `bot-run`, then `./check.sh bot-run 5 30`. It checks heap drift between hold minute 10 and minute 30 (live heap after GC, `go_heap_live_bytes`) and average CPU during the hold, and prints the server histogram p99. Exit code 1 means a gate failed.
5. Record in #192: commit, `node.txt`, `report.txt` and the `check.sh` output. A soak is the same with `-hold 12h` and `./check.sh bot-run 5 720`.

Scripted: `tools/loadtest/hetzner.sh` does steps 1 to 5 from an operator machine with Go, the Hetzner CLI `hcloud`, `ssh` and `openssl`, as soon as the owner creates the Hetzner project (owner input OI-1, `docs/launch-runbook.md`). It reads `HCLOUD_TOKEN` (a read and write API token of that project), `HD_LOAD_SSH_KEY` (the name of an SSH key uploaded to the project) and `HD_LOAD_SSH_IDENTITY` (its private key file). `hetzner.sh all`:

1. Builds `loadbot` and `migrate` for Linux and records the commit, for amd64 and, when a host type is a CAX (arm64) type, also for arm64; each host gets the binaries of its architecture.
2. Creates, all labelled `purpose=hd-load` in `HD_LOAD_LOCATION` (`fsn1`): the private network `10.70.1.0/24`, a firewall that admits only SSH from outside (Hetzner firewalls do not filter the private network), the node (`HD_LOAD_NODE_TYPE`), the bot host (`HD_LOAD_BOT_TYPE`), both defaulting to `HD_LOAD_TYPE` and that to CCX23, and a stores host (`HD_LOAD_STORES_TYPE`, CCX23) running PostgreSQL 16 with `pg_stat_statements` and Redis 7 in Docker on its private address only, with one generated password kept in the work directory `HD_LOAD_WORK` (`~/hd-load`), never printed.
3. Widens the ephemeral port range on node and bot host, copies the binaries and scripts and migrates the database.
4. Runs the 10,000-room load run (`run load`) and the storm run (`run storm`, storm at hold minute 10). Each run starts `node.sh` with `HD_DATABASE_URL`, `HD_REDIS_URL` and `HD_NODE_ID`, runs `bot.sh`, takes a 30 s CPU profile at hold minute 15 (`cpu.pprof`), copies both run directories with the `pg_stat_statements` top 10 and the Redis command stats to `~/hd-load/<run>` and runs `check.sh` there.
5. Deletes every resource with the label on exit, passed or failed.

`hetzner.sh up`, `run <name> [loadbot run flags]`, `fetch <name>` and `down` do the same in parts; `RAMP_MIN` and `HOLD_MIN` (5 and 30) set the ramp and the hold of a run and of its check, so a soak is `HOLD_MIN=720 hetzner.sh run soak` between `up` and `down`. The beta run is `HD_LOAD_NODE_TYPE=cx33 HD_LOAD_STORES_TYPE=cx33 hetzner.sh up`, then `hetzner.sh run beta -rooms 1000` and `hetzner.sh run beta-storm -rooms 1000 -storm 15m`, then `hetzner.sh down`; run it twice at different times of day and record the worse. The script has not run against Hetzner yet; its first run is the first reference run, and a step that fails there is fixed in the script, not by hand.

Reconnect storm: the same run with `./bot.sh <node private address> storm-run -storm 15m` drops every live connection at once at hold minute 10, and each seat dials again at once and resumes with its acked sequence. The report adds the dropped seats, the seats back in their match and the slowest return; the 1-minute windows around the storm show its cost on the move round-trip. The usual gates still decide the exit code. There is no recovery target yet: the first reference storm run is recorded in #192 and the owner sets the target from it.

TLS run: production terminates TLS on the Hetzner load balancer (ADR 0010, ADR 0011), so the TLS run puts one in front of the node and the bot connects through it; the node binary never serves TLS. 10,000 rooms at 50% 2v2 hold 30,000 WebSocket connections (5,000 × 2 + 5,000 × 4), so the run uses an LB31 in `fsn1`; check its connection limit against the report's peak connections before the run. Setup: attach the LB to the node's private network, add the node as a target by its private IP, add an `https` service on 443 to destination port 8080 with a Hetzner managed certificate for a subdomain of the project domain (for example `load.<domain>`) and an HTTP health check on `/readyz`. Then run `./bot.sh <node private address> tls-run -target wss://load.<domain>/ws`: the extra `-target` overrides the default, while `/metrics` and the heap profile are still read from the node's private address. Compare its report with the plain run of the same size; the difference is the cost of the TLS handshake and the load balancer hop, and the storm variant (`-storm 15m` added) shows the handshake cost of 30,000 reconnects at once. Delete the LB and its certificate after the run.

Database and Redis run: a move touches neither PostgreSQL nor Redis, and a connection verifies its token in process, so their load comes per match: the match row and its participants when the room opens, the result, replay and seat results when it ends, and one route claim and one release in Redis per seat. The node runs `node.sh` with `HD_DATABASE_URL` (migrated with `cmd/migrate up`) and `HD_REDIS_URL` plus `HD_NODE_ID` exported, and `loadbot serve` makes those writes for every room. `check.sh` then prints the p99 of `match_begin_seconds`, `match_finish_seconds`, `route_claim_seconds` and `route_release_seconds` and the `hd_store_failures` count; the matches finished in the bot report over the run time give the write rate the node needs. The usual gates still decide the exit code, so the run shows whether the writes cost the move round-trip. Record the database and Redis hosts with the results. There is no write latency target yet: the owner sets it from the first run. Rating, reward and outbox writes after a match are not wired to rooms yet and are not in this run.

### 6.2 Budgets

| Budget | Target | Environment | Method | Sample and duration | Pass / fail |
|---|---|---|---|---|---|
| Move round-trip | p99 < 100 ms | Load bot in the same cloud region as the node, on a different host; production-like TLS and WebSocket | Load bot (`server/cmd/loadbot`, #192) timestamps each move intent send and the resulting broadcast receive. The server also exports a Prometheus histogram `move_roundtrip_seconds` (with the node gauges `go_heap_live_bytes`, `go_goroutines`, `hd_rooms` and `hd_connections` beside it) for its in-process share: an accepted move intent from its receive by the gateway to the resulting broadcast queued for every seat, so the socket writes are left to the bot measurement | 30 min steady state at the 10k-room load, at least 100,000 moves, first 2 min discarded as warm-up | Bot-measured p99 < 100 ms overall and in every 1-minute window; server histogram p99 reported alongside |
| Concurrent rooms | 10,000 per node | Reference node | Load bot ramps to 10,000 active rooms (1v1 and 2v2 mix, moves at human cadence inside the admin-config turn time), then holds. Observe with pprof, `runtime/metrics` and node CPU / RAM | Ramp, then 30 min hold | Holds 10,000 rooms within the move budget; error and disconnect rate under 0.1%; average CPU under 80%; heap flat within 5% between minute 10 and minute 30 |
| Match traffic | < 200 KB per seat per match (#163) | In process, CI | E2E S-1 and S-2 (`server/internal/e2e`, E-7) count every frame payload a seat sends and receives over its WebSocket for a whole match, with a full roster (16-character nicknames, 3 emblems). Payloads are counted before WebSocket compression, so the wire is at most this | 3 seeds per mode | Every seat under 200,000 bytes; fails the test otherwise |
| Hot-path cost | No regression | Developer machine and CI | `go test -run='^$' -bench=. -benchmem -count=10 ./internal/rules/... > new.txt`, then `benchstat old.txt new.txt` | 10 runs per side | No benchmark slower than 5% at `p < 0.05`; allocs/op not higher |

Profiling:

```bash
go test -run='^$' -bench=<Name> -cpuprofile cpu.out -memprofile mem.out ./internal/rules
go tool pprof -http=:0 cpu.out
go tool pprof -http=:0 "http://<node>:<pprof-port>/debug/pprof/profile?seconds=30"
go tool pprof -http=:0 "http://<node>:<pprof-port>/debug/pprof/heap"
```

The profiling port is `HD_PPROF_ADDR` and the metrics port `HD_DEBUG_ADDR`; neither is ever exposed publicly. In Kubernetes pprof listens on the pod's loopback only and is reached with `kubectl port-forward` (`server/README.md` Container and deploy).

## 7. Tools

| Tool | Used for |
|---|---|
| Unity Profiler (development build, deep profiling off), Profile Analyzer | Frame time, GC, batches, before/after capture comparison |
| Frame Debugger | Draw call composition |
| Memory Profiler | Snapshots, leak and allocation source |
| Android GPU Inspector, Perfetto (`adb shell perfetto`, ui.perfetto.dev) | GPU and frame timeline, CPU scheduling, thermal |
| Xcode Instruments (Time Profiler, Metal System Trace, Energy Log, App Launch) | iOS frame, GPU, energy, start |
| `adb shell am start -W`, `adb logcat` | Cold start |
| Unity Build Report, `bundletool`, App Thinning Size Report | Install size |
| Battery Historian, `dumpsys batterystats` | Android battery |
| Go benchmarks, `benchstat`, `pprof`, `runtime/metrics` | Server CPU, allocation, regression |
| Load bot (`server/cmd/loadbot`, #192), Prometheus histograms | Server latency and capacity |

## 8. Pull request rule

A performance path is any change to: client `Update` / `LateUpdate` / `FixedUpdate` loops, input handling, animation and effects, the match scene (prefabs, materials, shaders, atlases, UI), pooling, the server room loop and scheduler, protocol encode / decode and connection handling, and `server/internal/rules` code that runs per move.

Such a PR includes before and after numbers for every budget it can affect, each with device (model and tier), build type, method (the row in section 3 or 6), sample size and duration, and a pass / fail verdict. If a budget cannot be affected, the PR says which and why. A regression against a budget is fixed before merge, or the exception is recorded in `docs/adr/`. The checklist item "Changes on a performance path include a measurement" in `.github/pull_request_template.md` is ticked only when this section is satisfied.

Paste-ready block for the PR body under Verification:

```markdown
### Performance measurement
- Path touched:
- Device / tier / build:
- Method (docs/performance.md row):
- Sample (runs x duration):

| Metric | Before | After | Budget | Result |
|---|---|---|---|---|
| Work time p50 / p95 (ms) | | | 11.0 / 14.0 | |
| Missed vsyncs (%) | | | <= 1 | |
| Max batches | | | < 150 | |
| GC alloc per frame (B) | | | 0 | |
| Cold start median (ms) | | | < 3000 | |
| Install size (MB) | | | < 150 | |
| Move round-trip p99 (ms) | | | < 100 | |
| Benchmark delta (benchstat) | | | no regression | |

Not affected and why:
```

### 6.3 Network

- One WebSocket per session carries the queue and the match; a reconnect resumes the same seat (`proto/README.md`). Frames above `HD_COMPRESSION_THRESHOLD` (256 bytes) are compressed. Client frames are capped by `MaxFrameBytes` (1 KiB) and the hello by `MaxHelloBytes` (8 KiB).
- `MatchView` is the whole visible state of a seat. The roster and table never change in a match and go only in the first view of a connection: with a full roster they were about 75% of the bytes (2v2 seed 1: 201,785 bytes per seat before, 37,797 after). Measured after the change, before compression: 1v1 30,824 to 45,050 bytes per seat, 2v2 28,788 to 37,797. A finer delta of the play state is not needed for the budget.
- Weak network (3G) behavior is a device run with the client and is not covered here.
- The client's `ClientWebSocket` does not offer permessage-deflate, so no frame is compressed today whatever the threshold; the load bot does not offer it either, so the load runs match the client.

### 6.4 Local run, 2026-11-14 (not prod-near)

The Hetzner hosts of 6.1 do not exist yet, so this run shows the shape of the curve and where the time goes on one machine. Its numbers are not the node's capacity and do not pass or fail 6.2.

Why not prod-near: node, bot, PostgreSQL and Redis share one laptop (Intel i7-10870H, 8 cores, 16 threads, 16 GB) through the Docker Desktop WSL2 VM (16 vCPU, 7.7 GB). The node is held to 4 CPUs by a cgroup quota (`--cpus 4`, `GOMAXPROCS` 4), which are time slices of shared hyperthreads, not 4 dedicated cores; the bot needs 1.1 to 1.6 times the node's CPU and takes it from the same cores; turbo and heat change the speed during a run (the Redis script time, which no server change touches, varied by up to two times between runs of the same load, and the soak below shows the laptop throttling); every frame crosses a Docker bridge, so a socket write also runs the receiver's TCP path and costs more than through a NIC; there is no TLS and no load balancer; PostgreSQL and Redis run with default settings on the laptop disk. Windows, the IDE and the browser stayed open.

Command, from the repository root (the per-step files land in `<dir>/rooms-<n>/`):

```bash
bash tools/loadtest/local.sh <dir> 1000 2000 4000 6000 8000 10000 12000
bash tools/loadtest/curve.sh <dir>
```

Each step: a fresh node with `HD_DATABASE_URL` and `HD_REDIS_URL`, 50% 2v2, 1 min ramp, 4 min hold, bot defaults (think 1 to 3 s, 0.02 drops per connection per minute), 2 min warm-up, 1 min windows; `local.sh` waits until no other container runs and the VM load is under 1.5 before each step. Node CPU is the `docker stats` mean during the hold, where 400% is the quota; GC from `runtime.MemStats` at the end; stores from `pg_stat_database` and Redis `INFO` deltas over the step; begin and claim are the `match_begin_seconds` and `route_claim_seconds` p99 bucket. Server code of `main` at 8a924fc with the load bot changes of #192, before the write timer change below.

| Rooms | Peak conns | Moves/s | p50 / p95 / p99 ms | Steady p99 / worst window ms | Errors % | Server p99 ms | Node CPU % / peak MiB | Bot CPU % | Peak live heap MiB | Peak goroutines | GC cycles / GC CPU % | PG commits/s, rows/s | Redis cmds/s, script µs | Begin / claim p99 ms | Result |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 1000 | 3,012 | 1,786 | 1 / 1 / 2 | 2 / 2 | 0 | ≤5 | 46 / 288 | 73 | 76 | 15,051 | 75 / 0.25 | 11, 28 | 99, 17 | 10 / 1 | pass |
| 2000 | 6,018 | 3,571 | 1 / 1 / 2 | 3 / 24 | 0 | ≤5 | 92 / 560 | 130 | 150 | 30,083 | 81 / 0.88 | 22, 56 | 194, 18 | 25 / 1 | pass |
| 4000 | 11,968 | 7,075 | 1 / 16 / 92 | 100 / 110 | 0 | ≤5 | 171 / 974 | 222 | 296 | 59,835 | 86 / 3.16 | 44, 111 | 385, 20 | 50 / 2 | fail |
| 6000 | 17,918 | 10,437 | 1 / 95 / 210 | 220 / 258 | 0 | ≤5 | 263 / 1,616 | 345 | 442 | 89,685 | 86 / 5.02 | 65, 166 | 568, 21 | 250 / 10 | fail |
| 8000 | 23,930 | 13,528 | 2 / 195 / 326 | 342 / 348 | 0 | ≤5 | 366 / 2,164 | 438 | 589 | 119,731 | 84 / 6.54 | 85, 218 | 745, 22 | 500 / 25 | fail |
| 10000 | 29,917 | 15,681 | 36 / 311 / 484 | 502 / 529 | 0 | ≤10 | 412 / 3,019 | 454 | 884 | 152,119 | 76 / 7.30 | 101, 267 | 908, 22 | >1000 / 100 | fail |
| 12000 | 35,850 | 15,979 | 128 / 501 / 773 | 815 / 961 | 0 | ≤25 | 412 / 4,070 | 561 | 1,290 | 186,316 | 66 / 7.65 | 70, 218 | 722, 23 | >1000 / 250 | fail |

The 24 ms worst window at 2000 is the last, partial window of 13 moves while the bot shuts down.

Findings:

- **Knee.** The bot-measured p99 stays at 2 to 3 ms up to 2000 rooms (6,000 connections, 3,600 moves/s), reaches the 100 ms budget at 4000 and grows with load after it. Throughput stops growing at about 16,000 moves/s, where the node uses its whole quota (10,000 rooms). Nothing failed at any step: no rejected intent, no dial or read failure, no store failure, 35,850 connections at 12,000 rooms.
- **Where the tail is.** The server's own share (`move_roundtrip_seconds`, receive to broadcast queued) stays at or under 5 ms for 99% of moves up to 8000 rooms; the rest of the bot's round trip is after the broadcast is queued. A 5 s execution trace at 6000 rooms (`go tool trace -pprof=sched`) shows on average about 190 goroutines runnable and waiting for one of the 4 Ps, and 28% of that wait on the write goroutines woken by `Room.intent` through `RoomSession.Deliver` and `Conn.Send`. The write goroutines wait for a P well before the quota shows as used, most likely because each P stays with its goroutine through the write syscall and the VM's vCPUs are shared with the bot. The cgroup throttled the node in 40 of 3,138 periods at 4000 rooms, so the quota itself is not the cause.
- **CPU.** One `write` syscall per frame per seat is 38% of the node's CPU at 1000 and 4000 rooms and 42% at 10,000 (`internal/poll.(*FD).Write` under `gateway.(*Conn).writeLoop`); the scheduler 2 to 14%, `mallocgc` 4 to 8%, reads 3 to 5%, GC 2.6 to 6%, protobuf encode about 8%. The write timeout cost 5 to 9% (`context.WithDeadlineCause` per frame plus the WebSocket library's `context.AfterFunc` on it) and is fixed in this change (below).
- **Memory.** About 100 KB per connection in the container (3,019 MiB at 29,917 connections), of which about 30 KB live heap; 12,000 rooms reached 4,070 of the 4,096 MiB limit. Every connection keeps 4 goroutines (`readLoop` in the HTTP handler, `writeLoop`, the close waiter in `serve` and `pingLoop`), every room 2 (`Room.Run`, `Rooms.run`; `loadbot serve` adds one per room for its store writes); 4 KB `bufio` reader and writer buffers kept from the HTTP hijack are the largest heap items per connection.
- **PostgreSQL.** Statement time was 0.6% of one core at 1000 rooms and 8.2% at 10,000 (67 match rows written and 34 finished per second). `match_participants` is inserted one row per statement inside the begin transaction and is 68 to 70% of that time; each finish also updates one participant row per seat. Mean statement time stayed at 0.1 to 0.3 ms. `match_begin_seconds` p99 rose from 10 ms to over 1 s only when the node ran out of CPU: the store goroutines waited for the node, not for PostgreSQL.
- **Redis.** About 90 commands per second per 1000 rooms; the route claim and release Lua scripts (`EVALSHA`) take 17 to 23 µs each against 2 to 4 µs for `GET` and `SET`, under 1% of one core at 10,000 rooms.

Write timeout change, before (the build of the table) and after, alternated on the same machine at 4000 rooms with `PREBUILT` (B, A, B, A; same step settings as the table):

| Run | Steady p99 / worst window ms | Node CPU % | Timeout plumbing, s of 30 s profile | `mallocgc`, s | Redis script µs |
|---|---|---|---|---|---|
| Before 1 | 196 / 237 | 228 | 5.19 | 6.46 | 28 |
| After 1 | 47 / 62 | 186 | 0.36 | 2.70 | 22 |
| Before 2 | 113 / 120 | 206 | 3.74 | 4.59 | 21 |
| After 2 | 133 / 144 | 252 | 0.57 | 3.70 | 28 |

Timeout plumbing is `context.WithDeadlineCause` in the CPU profile; what remains after the change is the read and ping path. The cut in timeout and allocation work repeats in both pairs; the p99 and the node CPU move with the machine more than with the change (the Redis script time, which the change does not touch, shows the machine 30% slower during Before 1 and After 2), so this run claims no p99 gain. In process, `BenchmarkSendToAReadingClient` (`server/internal/gateway`) fell from 23 to 12 allocations and from 1,778 to 985 bytes per frame sent and read. The first curve, an hour earlier, measured a steady p99 of 100 ms at 4000 rooms with the before build: runs taken at different times are not comparable on this machine.

Reconnect storm, after the change (`RAMP=1m HOLD=6m BOT_FLAGS="-storm 3m" bash tools/loadtest/local.sh <dir> 2000 4000`): at minute 3 every live connection drops at once and every seat dials again at once, which is also what the client does after a mass disconnect (`ReconnectSchedule`: the first attempt has no delay and no jitter).

| Rooms | Dropped | Back in their match | Not back by the end | Connections back to the pre-storm count | Slowest return | Storm window p99 ms (neighbours) | Errors |
|---|---|---|---|---|---|---|---|
| 2000 | 5,856 | 5,854 | 2 | by the next 10 s sample | 1 min 45 s | 3 (3, 3) | 0 |
| 4000 | 11,726 | 11,684 | 42 | by the next 10 s sample | 2 min 13 s | 166 (115, 154) | 0 |

The storm cost the move round-trip almost nothing at 2000 rooms and about 20 ms of window p99 at 4000, where the steady state was already over budget on this machine. The slow tail is the kernel, not the server: the node's listen queue holds `net.core.somaxconn` = 4096 pending connections, and the 4000-room storm overflowed it 5,420 times (`ListenOverflows` in the node's network namespace, 0 before the storm). Linux retries a dropped SYN 1, 3, 7, 15, 31 and 63 s after the first, which matches returns of up to about 2 minutes. Why 2 and 42 seats had still not received a view 4 minutes after the storm is not established; a seat whose match ended while it was away reopens a fresh `loadbot serve` room alone, which production does not do, and is the first suspect. Every reconnect that got through resumed with no error.

Soak, after the change (`RAMP=1m HOLD=50m bash tools/loadtest/local.sh <dir> 2000`): 2000 rooms (6,033 connections, 3,962 moves/s) for 50 minutes, 36,731 matches finished and 12.1 million moves, with PostgreSQL and Redis writes on. It passed. The node's live heap read 156 to 159 MiB at minutes 5, 10, 20, 30, 40 and 50 (bot gate: +2.0% from minute 3 to the end) and its goroutines 29,558 to 29,927 (+0.5%); container memory was 567 MiB at minute 5 and 565 MiB at minute 30, `runtime.MemStats.Sys` 610 MB at minute 25 and at the end; 596 GC cycles took 2.1% of the node's CPU; no error and no store failure. The window p99 held at 3 ms for 22 minutes and was 42 to 79 ms from minute 23 to the end at the same move rate: at that minute the node's and the bot's CPU for the same work rose together by about 45% (node 143% to 212%), the Redis script time doubled to 36 µs, and Windows read `% Processor Performance` at 86% of the base clock against 168% earlier in the day. The laptop throttled; the server did not drift. A flat heap and goroutine count over 50 minutes is what this run shows; the 12-hour soak needs the reference hosts.

## 9. Open items

- Thresholds are initial; #191 (certification) and #164 (battery) may tighten them by PR to this file.
- The load bot exists (`server/cmd/loadbot`, #53, `server/README.md` "Load test"); its bot-measured round-trip, error rate and heap and goroutine drift are the server gates above; `tools/loadtest/local.sh` runs it on one machine (6.4) and `tools/loadtest/hetzner.sh` on the reference hosts (6.1). The `move_roundtrip_seconds` histogram is served at `/metrics` on the debug listener (`HD_DEBUG_ADDR`) and pprof at `/debug/pprof/` on the profiling listener (`HD_PPROF_ADDR`); each is off when empty (`server/README.md`). The replay harness does not exist yet (#192); until it does, client gates are checked by hand with the Profiler and the commands above.
- Thermal status on iOS (1.1): reading `ProcessInfo.thermalState` needs a native plugin, added when the first iOS build exists (#159). Until then an iOS device never drops for heat.
