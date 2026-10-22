# Phase 6 — Overview

> Historical note:
> This Phase 6 roadmap is preserved for context, but it is not the best summary of the current live system.
> The project has since moved further into Phase 7 authored-campaign runtime work.
>
> For current architecture/runtime documentation, start in:
> - `Overview.md`
> - `Project Structure.md`
> - `AI Architecture.md`
> - `Game Engine.md`
> - `UI.md`
> - `TODO.md`

Phase 6 is a major architectural evolution of An Amazing Adventure. Phase 5 delivered a working alpha: a solo AI-narrated text adventure running on AWS Lambda + Bedrock + DynamoDB. Phase 6 transforms it into a proper D&D 5e game engine with party play, RBAC, metered usage, and a full rules engine.

This is a long migration. Each phase is an independent feature branch and PR that can ship without blocking the others. The system remains playable throughout.

---

## Context: Where We Are After Phase 5

- Solo player, one WebSocket connection per game session
- Single `game.Character` struct with `Health int` (0–100) — no ability scores, no dice
- Combat: Claude Sonnet narrates outcomes, Claude Haiku infers and applies the mechanical result (`damage_character`, `heal_character`)
- World generation: Claude Sonnet generates a blueprint, deterministic Go code builds the room graph
- Auth: Cognito JWT, no groups, no roles — all signed-up users are equal
- Usage: `TotalTokens` tracked per game session, no per-user cap, no billing guard
- Infrastructure: all single-user; connections table has no way to query by game session

---

## Phase Roadmap

### Terraform Infrastructure (prerequisite)
> See [[Terraform Infrastructure]]

New AWS resources required by all subsequent phases. Must ship first via a PR to `rrochlin/terraform-infrastructure`. No application code changes.

New resources:
- `{prefix}-users` DynamoDB table (roles, quotas, billing mode)
- `{prefix}-invites` DynamoDB table (party invite codes with TTL)
- `{prefix}-memberships` DynamoDB table (user ↔ session many-to-many)
- GSI on `{prefix}-connections` for `game_id` (enables broadcast to party)
- Cognito groups: `admin`, `user`, `restricted`
- New Lambdas: `http-admin`, `http-invites`, `cognito-post-confirm`

---

### Phase 0 — RBAC + Usage Controls
> See [[RBAC and Usage Controls]]

**Goal:** Protect against unbounded Bedrock spend and establish the user permission model before opening the game to anyone.

New users who sign up land in the `restricted` group with `ai_enabled = false`. No Bedrock calls fire for restricted users. An admin panel (accessible only to the `admin` Cognito group) lets you manually approve users, set token limits, and view aggregate usage.

A shell script (`scripts/elevate-admin.sh`) bootstraps the first admin account given a Cognito user sub.

The usage model is designed with the eventual goal of supporting BYOK (bring-your-own API key) or a subscription payment flow — the `billing_mode` field and structure accommodate that without a schema migration.

Ships independently of all rpg-toolkit work.

---

### Phase 1 — Party Sessions + Invite System
> See [[Party Sessions]]

**Goal:** Multiple players can join and play a shared game session. Each has their own character. The dungeon is shared.

The connections table gains a `game_id` GSI so all WebSocket connections for a session can be looked up. `wsutil` gains a `Broadcast` method. `ws-chat` and `ws-game-action` fan out all state deltas to every connected party member.

The session model gains `OwnerID`, `Members []PartyMember`, and `Players map[string]Character`. `SchemaVersion` bumps to 2.

Three join paths:
1. **Invite link** — `https://{domain}/join/{code}` — anyone with the link can view the game and join after authenticating
2. **Join code** — a 6-character alphanumeric code the owner shares; existing users enter it in the games list UI
3. **New user onboarding** — signup URL encodes the invite code; after email confirmation the user is auto-dropped into character creation for that session

Ships independently of rpg-toolkit work.

---

### Phase 2 — rpg-toolkit: Character Model
> See [[rpg-toolkit Integration]]

**Goal:** Replace `game.Character` with the D&D 5e character model from `github.com/KirkDiggler/rpg-toolkit/rulebooks/dnd5e`.

The `game.Character` struct (Health int, 6-slot equipment) is replaced with `dnd5e/character.Character` — full ability scores, skills, saving throws, hit points derived from hit dice + CON modifier, class features, conditions, and resources (rage charges, ki, etc.).

Character class is constrained to **Barbarian, Fighter, Monk** — the only three classes with mechanically implemented features in the toolkit. All 9 PHB races are available (racial ASI and proficiencies apply; special traits are data-only).

`SaveState` bumps to SchemaVersion 3. Old v2 records are migrated on load.

The `game` package is restructured: `game.Game.Players map[string]*dnd5e.Character` replaces the single `game.Game.Player Character`.

---

### Phase 3 — rpg-toolkit: Combat Engine
> See [[rpg-toolkit Integration]]

**Goal:** Move combat resolution from "Claude decides the outcome" to "Go code resolves dice and rules, Claude narrates the result."

`combat.ResolveAttack()` is called server-side. The `AttackResult` struct (roll, AC, hit/miss, damage, damage type) is injected into the Narrator's prompt as structured context. Claude narrates what happened — it no longer invents outcomes.

An `events.EventBus` is instantiated per Lambda invocation, all character conditions subscribe, `Cleanup()` runs at end. Monster turns use `monster.TakeTurn()`.

The Engineer's `damage_character` / `heal_character` / `set_character_alive` tools are removed.

---

### Phase 4 — rpg-toolkit: Procedural World Generation
> See [[rpg-toolkit Integration]]

**Goal:** Replace the AI blueprint world generation pipeline with `tools/environments` + `tools/spawn` from rpg-toolkit.

The dungeon layout (rooms, passages, room types, encounter zones) is generated procedurally. The 8 available monsters are placed using the constraint-based spawn engine (CR-appropriate, distance rules, density control).

After generation, a single Claude call receives a structured dungeon summary and writes `Title`, `Theme`, `QuestGoal`, and `OpeningScene` — informed by the actual dungeon layout and monster roster.

`game.Area` / `map[string]Area` is replaced by `dungeon.DungeonData`. `SchemaVersion` bumps to 4 (or folds into Phase 2/3 bump).

---

### Phase 5 — Frontend: D&D Character Creation Wizard
> See [[Frontend D&D Wizard]]

**Goal:** Replace the current 4-question adventure creation form with a proper D&D character sheet wizard.

Steps: Name → Race (9 cards with ASI summaries) → Class (3 cards: Barbarian/Fighter/Monk with feature descriptions) → Ability Scores (standard array 15/14/13/12/10/8 assigned to STR/DEX/CON/INT/WIS/CHA, or point buy) → Skills (pick N from class skill list) → Review (full character sheet preview).

`POST /api/games` payload changes from `AdventureCreationParams` to a `CharacterCreationData` struct matching `dnd5e/character.CreationData`.

For players joining an existing party session via invite: the wizard skips the theme/world step (world already exists) and only captures character details.

---

### Phase 6 — Cleanup
> See [[rpg-toolkit Integration#Cleanup]]

rpg-toolkit is licensed GPL-3.0, but GPL v3 copyleft is triggered by *distributing* software, not by running it as a hosted network service (that requires AGPL, which rpg-toolkit does not use). We do not distribute binaries or source to users. No open-sourcing obligation applies.

- Remove obsolete AI world-gen prompts and Engineer combat tools replaced by the rules engine
- Add rpg-toolkit attribution comment to `go.mod`
- Update `CLAUDE.md` and this documentation to reflect the completed state

---

## Sequencing and Dependencies

```
Terraform Infra PR (rrochlin/terraform-infrastructure)     ✅ merged
        │
        ▼
Phase 0: RBAC + Usage Controls          [independent]      ✅ merged (#26)
  └── scripts/elevate-admin.sh
        │
        ▼
Phase 1: Party Sessions                 [independent]      ✅ merged (#27)
        │
        ├──────────────────────┐
        ▼                      ▼
Phase 2: Character Model    Phase 4: Procedural World Gen  🔄 Phase 2 open (PR #29)
        │                      │
        ▼                      │
Phase 3: Combat Engine         │
        │                      │
        └──────────┬───────────┘
                   ▼
Phase 5: Frontend D&D Wizard
                   │
                   ▼
Phase 6: Cleanup
```

Phases 2 and 4 can be developed in parallel after Phase 1 merges. Phase 3 depends on Phase 2 (needs the new character model). Phase 5 depends on Phase 2 backend (needs `CharacterCreationData` endpoint).

---

## MANDATORY: Branch Workflow Before Starting Each Phase

**Always sync with remote before creating a new branch.** If a previous PR merged while you were working on the prior phase, your local `main` is stale — starting from stale `main` means your new branch silently includes the previous phase's commits. This creates divergent history and conflicts when you open the PR.

```bash
# Do this at the START of every new phase, before any code changes:
git fetch origin
git log --oneline origin/main -5   # confirm the most recent merged PR is visible
git checkout -b feat/<phase-name> origin/main
```

**If you forgot (already have commits on a stale branch):**
```bash
# 1. Identify your phase-N-only commits
git log --oneline origin/main..HEAD

# 2. Create a clean branch from current main
git checkout -b feat/<phase-name>-rebased origin/main

# 3. Cherry-pick only your commits (not the already-merged ones)
git cherry-pick <sha1> <sha2> ...

# 4. Push and open a new PR; close the stale one
git push -u origin feat/<phase-name>-rebased
gh pr close <old-pr-number> --comment "Superseded by rebased branch"
gh pr create ...
```

---

## Key Design Decisions Recorded

| Decision | Choice | Rationale |
|---|---|---|
| rpg-toolkit scope | Take as-is, no expansion | ~40% of 5e implemented; enough for Barbarian/Fighter/Monk solo and party play |
| rpg-toolkit license | No open-sourcing obligation | GPL v3 copyleft triggered by distribution, not network service use; AGPL (not applicable here) would cover SaaS |
| Combat resolution | Rules engine decides, Claude narrates | Deterministic rules = reproducible, fair; Claude becomes a storyteller not an arbiter |
| World generation | Full procedural (environments + spawn) | Reduces token cost, eliminates hallucinated layouts; LLM still writes narrative framing |
| Party narrator | First-message-wins, rest blocked | Simplest; existing `streaming_blocked` frame already handles it |
| Party member game lookup | Separate `memberships` table | Clean, scalable; GSI enables "list my sessions" without full scan |
| Token quota | Per-user, each member from own quota | Prevents one player burning host's quota |
| Invite auto-approval | Manual admin approval always required | Future path: BYOK or subscription; `billing_mode` field is the extension point |
| Character classes | Barbarian, Fighter, Monk only | Only mechanically implemented classes in rpg-toolkit as of March 2026 |
| Ability scores | Standard array or point buy | Skip random rolling for web UX; deterministic character creation |
| NPC behavior | LLM-driven (unchanged) | rpg-toolkit `behavior/` is a stub; Claude still narrates all NPCs |
| WebSocket auth (WS) | Existing token-in-query-string (unchanged) | Browsers can't set custom WS headers; no JWKS verification on WS connect (documented gap) |

---

## Reference: Current Production State (as of Phase 5)

- **Frontend:** React 19 + TypeScript, Vite, TanStack Router, MUI dark fantasy theme, Zustand, react-konva map
- **Backend:** Go 1.24, AWS Lambda (arm64, provided.al2023), 7 Lambda functions
- **AI:** AWS Bedrock — `us.anthropic.claude-sonnet-4-6` (narrator + architect), `us.anthropic.claude-haiku-4-5-20251001-v1:0` (engineer/sub-agents)
- **Database:** DynamoDB — `sessions`, `connections`, `mutations` tables
- **Auth:** Cognito User Pool, SRP flow, JWT authorizer on HTTP API; manual token decode on WebSocket connect
- **Region:** `us-west-2`, Account: `292826404083`
- **Deployment:** GitHub Actions → S3/CloudFront (client), Lambda ZIP (server); branch protection on `main`
- **Open PRs:** PR #25 (`feat/map-expand-and-scroll-zoom`) — map fog-of-war, scroll-zoom, elevation chips — awaiting merge
