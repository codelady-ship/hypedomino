# HypeDomino

Real-time online multiplayer mobile domino game.

## Layout

| Path | What |
|---|---|
| `client/` | Unity 6 LTS client, URP Universal Renderer (2D sprites, 3D characters), C#, Addressables |
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

## Commands

| Command | What |
|---|---|
| `npm run lint` | Comment lint over all tracked code files |
| `npm run test:tools` | Tests for the scripts in `tools/` |

The pre-commit hook runs the comment lint on staged files and `gofmt` on staged Go files.

## Rules for contributors

Read [CLAUDE.md](CLAUDE.md) before the first change: code without comments, server-authoritative logic, performance budgets, branch and PR workflow.

Code must not contain comments. Names explain intent. Directives, license headers required by a dependency and generated files are allowed. Paths that must be excluded go into `.commentlintignore`, one per line (`dir/`, `*suffix` or an exact path).

<!--
TODO(owner): add a "Team & my role" section. From git log (2026-10-25 to 2026-10-30, non-merge
commits): ismetcahangirov 48 (two emails), antiqa jalilova 35. Please write what you did in
your own words; it was left out on purpose so nothing is guessed.
TODO(owner): this repository is a fork of Alievs-corp/hypedomino. Confirm you are allowed to show
it publicly in your portfolio before linking it.
TODO(owner): no screenshots yet. The Unity client and the Go server could not be run on the
machine that prepared this change; add screenshots from a device or editor build.
-->
