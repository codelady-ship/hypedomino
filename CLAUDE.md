# HypeDomino

Real-time online multiplayer mobile domino game. Hyperactive, colorful, effect-heavy UI. Performance is the top priority.

## Source of truth
- Game rules: `docs/rules.md` (Epic 1). Code must match it exactly. Change the doc first, then the code.
- Economy, rarity, ranking, pass specs: `docs/` (Epic 1). Plan and scope: GitHub epics and sub-issues.
- Never mention any third-party game name in code, docs, issues, commits or assets. Describe behavior on its own terms.

## Monorepo
- `client/` Unity 6 LTS, URP (Universal Renderer, 2D sprites; no 3D characters, ADR 0009 superseded), C#, Addressables
- `server/` Go, WebSocket (binary protobuf), PostgreSQL, Redis
- `admin/` Next.js, TypeScript, Tailwind
- `proto/` shared protobuf contracts
- `docs/` specs and ADRs
- `assets-src/` source art (Blender, Adobe, AI-generated), never shipped directly
- `tools/` scripts and pipelines

## Code rules
- Write code only. No comments, no docstrings, no XML doc, no TODO markers. Names must explain intent.
- Exception: license headers required by a dependency, and config files where a format needs a comment.
- Small functions, early returns, no dead code, no speculative abstractions.
- Server is authoritative. Client never decides scores, legality of moves, rewards or currency.
- All game rules live in one pure Go package (`server/internal/rules`) with table-driven tests. The client only renders and sends intents.
- No allocations in hot paths on the client (Update loops, input, animation). Pool everything that spawns repeatedly.
- Every number that designers may tune comes from admin-managed config, not constants.
- Zero trust in the client: no balance, price, reward, inventory or score value is ever accepted from or stored authoritatively on the client. The client sends intents only; modified APKs and memory editors must gain nothing. See Epic 18 and `docs/anti-cheat.md`.
- Review every change against NT-1 to NT-12 in `docs/anti-cheat.md` section 2. Device storage (PlayerPrefs, files, caches) holds only the keys listed under Device storage there; a new key is added to that table in the same PR.
- Purchases are granted only after server-side receipt validation. Time-based logic uses server time only.
- Secrets only from environment. Never commit keys, tokens or service accounts.

## Performance budgets
- Client: stable 60 fps on mid-range Android, 120 fps option on capable devices; cold start under 3 s; match scene draw calls under 150; GC alloc 0 B/frame during a match. UI canvas update (`PostLateUpdate.PlayerUpdateCanvases`) p95 at most 1.5 ms per frame, 0.3 ms on a static screen.
- Install size under 150 MB; extra content via Addressables/CDN on demand.
- Server: p99 move round-trip under 100 ms in region; 10k concurrent rooms per node target.
- Any change touching these paths must include a measurement in the PR.

## Git workflow
Same discipline as the IdeaNest repository. Every task follows this sequence, no exceptions.

```bash
git checkout main
git pull origin main
git checkout -b <type>/<short-description>
git push -u origin <branch>
gh pr create --base main --title "..." --body "..."
```

- Never commit to `main`. Branch types: `feat/ fix/ chore/ docs/ refactor/ test/ perf/`, kebab-case, under about 50 characters (`feat/spinner-double-rules`). If it cannot be described in a few words, split the task.
- Conventional Commits in English: `<type>(<scope>): <imperative subject>`, no trailing period, body explains why, wrapped at 72 columns. Never `--no-verify`.
- Every PR: links its issue (`Closes #N` or `Part of #N`), carries a `type:` label, at least one `area:` label and a `priority:` label, passes CI, lists the commands used to verify, and states what was not done and why.
- Merge only when `CI complete` is green. Squash merge. Delete the branch after merge. A failing check is fixed, never disabled or skipped.
- `CI complete` is the single required check on `main`: it always runs, waits for the other jobs and fails if any job that had work did not succeed. A new job that may block a merge is added to its `needs` and to its checked list. Branch protection also requires an up-to-date branch, resolved conversations, no force pushes, and zero required approvals until a second reviewer exists (then restore 1).

## Issues
- Work is tracked as epics with native sub-issues. Pick a sub-issue, never an epic.
- Assign yourself before starting. One sub-issue, one branch, one PR. If work outgrows the issue, split it.
- `status: needs-decision` means blocked on an owner decision. Do not implement around it.
- Rules, scoring and economy logic require tests before merge. Update docs in the same PR as the change.

## Agent conduct
- Read `docs/rules.md` before touching game logic; read the asset and motion specs before touching UI or art.
- Never invent a color, rarity, duration or price; they exist as tokens or admin config.
- Never claim something works without running it. Report failures as plainly as successes.
- When a rule here conflicts with a habit, the rule wins.

## Skills (use them)
- caveman (https://github.com/juliusbrussee/caveman): terse agent replies and compressed context to save tokens. Default communication style in sessions.
- ponytail (https://github.com/DietrichGebert/ponytail): lazy senior dev mindset. Prefer deleting and reusing over writing. Challenge every new abstraction and dependency.
- andrej-karpathy-skills (https://github.com/multica-ai/andrej-karpathy-skills): think before coding, simplicity first, surgical changes, goal-driven verification.
- graphify (https://github.com/Graphify-Labs/graphify): build and query the codebase knowledge graph (`/graphify`) before broad grep/reads. Rebuild after large structural changes.

## Asset pipeline
- Concept and raw art: ChatGPT (browser), then cleanup in Adobe (Photoshop, Illustrator, After Effects) or Blender.
- Animated avatars and other 2D assets: Blender/After Effects to sprite sheets or skeletal 2D rigs, packed into atlases, compressed (ASTC) and shipped via Addressables.
- Every asset has a rarity, a status color and a manifest entry managed from the admin panel.
- Static fallback exists for every animated asset.

## Language
- Code, identifiers, commits, docs: English.
- Issues and user-facing planning notes: Azerbaijani, technical terms in English.
