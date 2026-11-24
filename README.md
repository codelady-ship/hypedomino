# HypeDomino

Real-time online multiplayer mobile domino game.

## Layout

| Path | What |
|---|---|
| `client/` | Unity 6 LTS client, URP Universal Renderer (2D sprites), C#, Addressables |
| `server/` | Go server, WebSocket with binary protobuf, PostgreSQL, Redis |
| `admin/` | Next.js admin panel, TypeScript, Tailwind |
| `proto/` | Shared protobuf contracts |
| `docs/` | Specs and ADRs. Game rules: [docs/rules.md](docs/rules.md) |
| `assets-src/` | Source art, stored with Git LFS, never shipped directly |
| `tools/` | Scripts and pipelines |

## Setup

Requirements: Git, Git LFS and Node.js 22 or newer. Go, Unity and the admin toolchain are needed only for the area you work on.

```bash
git clone https://github.com/codelady-ship/hypedomino.git
cd hypedomino
git lfs install
npm install
```

`npm install` installs nothing; it only points Git at the repository hooks in `.githooks/`.

### Local stack

Requirements: Docker with Compose 2.22 or newer, `make` and `openssl`.

| Command | What |
|---|---|
| `make dev` | Starts PostgreSQL, Redis, the migrations, the game server (`:8080`), `admin-api` (`:8090`) and the admin panel (`:3000`), then watches the code: a change under `server/` rebuilds the server and restarts `admin-api`; the admin panel reloads through `next dev`. On the first run it writes `.env` from `.env.example` with a fresh `HD_TOKEN_KEYS` and seeds an empty database |
| `make seed` | Publishes the example configs of `docs/*/examples` through `cmd/config-publish` and adds the synthetic players `DevPlayer1` to `DevPlayer8` (`tools/dev/players.sql`, `docs/test-strategy.md` 7.2) |
| `make bots` | Runs `cmd/loadbot local`: bot players fill rooms on an in-process server; pass flags with `BOTS="-rooms 10 -hold 30s"` |
| `make logs`, `make down`, `make reset` | Follow the logs, stop the stack, stop it and delete its volumes |

Signing in from a client or the admin panel needs real Google client ids in `.env` (`HD_GOOGLE_AUDIENCES`, `HD_ADMIN_GOOGLE_*`); the defaults only let the services start. `.env` stays local (`docs/environments.md` section 3).

## Agent setup

The skills named in [CLAUDE.md](CLAUDE.md) come from pinned sources. The plugins are listed in the repository's marketplace `hypedomino-agents` (`.claude-plugin/marketplace.json`), each pinned to a release tag and its commit; `.claude/settings.json` declares the marketplace and enables the plugins. After you trust the folder in Claude Code, register the marketplace for yourself and install the plugins once. Registering with `--scope local` keeps the shared `.claude/settings.json` unchanged; `--scope project` would rewrite it.

```bash
claude plugin marketplace add Alievs-corp/hypedomino --scope local
claude plugin install caveman@hypedomino-agents --scope project
claude plugin install ponytail@hypedomino-agents --scope project
claude plugin install andrej-karpathy-skills@hypedomino-agents --scope project
claude plugin list --json
```

The check passes when the list shows `caveman` 4.0.1, `ponytail` 5.3.3 and `andrej-karpathy-skills` 1.0.0 from `hypedomino-agents`, each with `"enabled": true`. Their hooks need Node.js and run locally. Leave ponytail's `webhookUrl` empty so that nothing leaves your machine.

graphify needs Python 3.10+ and uv:

```bash
uv tool install graphifyy==0.9.102
graphify install --platform claude
./tools/graphify/graph.sh
graphify query "how is a move intent validated on the server"
```

`graphify install --platform claude` registers the `/graphify` skill in your user directory. `tools/graphify/graph.sh` builds the code graph into `graphify-out/` (ignored by Git) on the first run and updates it afterwards; `--rebuild` starts over after large structural changes. It runs with `GRAPHIFY_BACKEND=none` and `--code-only`, so code is parsed locally and nothing is sent to a model. `graphify explain "<symbol>"` and `graphify path "<a>" "<b>"` answer from the same graph.

To move to a newer version, review the plugin's hooks at the new commit, then change `ref` and `sha` in `.claude-plugin/marketplace.json` (or `version` in `tools/graphify/graph.sh`) in one PR.

## Commands

| Command | What |
|---|---|
| `npm run lint` | Comment lint over all tracked code files |
| `npm run test:tools` | Tests for the scripts in `tools/` |

The pre-commit hook runs the comment lint on staged files and `gofmt` on staged Go files.

## Rules for contributors

Read [CLAUDE.md](CLAUDE.md) before the first change: code without comments, server-authoritative logic, performance budgets, branch and PR workflow.

Code must not contain comments. Names explain intent. Directives, license headers required by a dependency and generated files are allowed. Paths that must be excluded go into `.commentlintignore`, one per line (`dir/`, `*suffix` or an exact path).
