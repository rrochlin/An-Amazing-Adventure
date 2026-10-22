# Decisions

The running log of design and process decisions, newest first. Each entry says what was decided,
why, and what it replaces. Work is planned in the GitHub Project (see `README.md`); this file records
the conclusions that work should follow.

Phase-era decisions live in `archive/Phase 7/Vision and Design Decisions.md`. The ones still in
force are summarised at the end of this file.

---

## 2026-10-22 — Plan with a GitHub Project and epics, not phases

- Work is tracked in one GitHub Project using **Kanban** (no sprints): a prioritised backlog, little in
  progress at once, and an occasional review.
- Large outcomes are **epics**: issues labelled `epic` with sub-issues and a "Done when" list.
  Research that needs a decision before building is a **spike** (`type:spike`).
- Labels carry type, area, priority (P0–P2) and size (S/M/L).
- **Platform vs content:** campaign-specific work (Briarwell's story, endings, a rival party's route,
  combat stat blocks, run length) is content (`type:content`). The platform is the engine those run on.
- Design notes are organised by topic under `docs/`. The phase folders are archived.

## 2026-10-22 — The script stays in control; the game master sits between director and DM

- A pre-written script guides the plot, world, decisions and characters.
- The story agent is a **game master** between a film director and a tabletop DM: it keeps to the
  script and owns scene boundaries, but improvises when the player goes off the rails. A traditional DM
  writes their own loose script; here control of the script matters more.
- This keeps Phase 7's principle that the authored script, not the AI, drives the main arc.
- Epic: game master (#99).

## 2026-10-22 — Characters get only what they need to know (framing still open)

- Characters receive information **on a need-to-know basis**, never the whole world state by default.
  Facts are recorded once in world state, and the game master shares the relevant ones.
- **Not decided:** whether characters are framed as **actors** (film: performing a part from a script)
  or **players** (tabletop: pursuing their own motivations). Rob leans toward players but is still
  debating. The framing spike (#92) compares them; see `story-engine/framing.md`.
- Epics: story model (#91), character agents (#103), world state (#109).

## 2026-10-22 — Scenes decide who acts

- Only characters in the current scene (with permission to act) are run each turn, plus threads the game
  master deliberately runs in the background, like a DM handling a party that has split up.
- This gives each human player their own thread, the foundation for multiple humans playing alongside AI.
- Epic: scenes and threads (#106).

## 2026-10-22 — A character's role lives in its description; changing relationships are state

- Who a character is (role, relationships, personality) is defined in its description and character
  sheet, and must reach whichever agent plays it. In session b31d479d the narrator's cast list carried
  only Tomas's goal, so it narrated the player's father as a stranger.
- Relationships that change during play (trust, loyalty) are world state that the character reads, not
  description. This replaces Phase 7 decision 2's "no deep arcs": arcs are allowed once they are state.

## 2026-10-22 — Pre-alpha: fix defects directly

- The game has one user and disposable infrastructure. Defects are fixed directly, without
  production-style gating or migrations. PRs, CI and Terraform still apply.

---

## Phase 7 decisions still in force

From `archive/Phase 7/Vision and Design Decisions.md`:

- **1** Solo first; the architecture must not preclude human co-op (now an explicit goal of scenes and
  threads).
- **3** The physical room graph is first-class; story nodes overlay it rather than replace it.
  (Navigation, #113; map overhaul, #117.)
- **4** Code resolves outcomes; the narrator only describes them. (Show the mechanics, #119.)
- **7** Permadeath one-shot runs; failure leads to an authored aftermath.
- **8 / 15** Authored endings selected by code from flags; finished runs are read-only.
- **9** AI classifies what happened; code decides what it means.
- **10 / 13** One post-turn resolution step with per-scene allowlists. Now folded into the game master
  (#102).

Content, not platform (per the platform vs content decision): 5 (campaign clock), 6 (rival party rail),
11 (run length and branching), 12 (the Fable-opening campaign), and the content side of 14 (stat blocks,
odds). The combat engine under 14 is platform.
