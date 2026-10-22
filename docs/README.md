# Docs

Design notes for An Amazing Adventure. Git is the source of truth; the Obsidian vault folder
`~/Documents/notes/Amazing Adventure` is a symlink to this directory, and notes use `[[wikilinks]]`.

## How planning works

- **Work** is planned in the GitHub Project for the repo: a Kanban board of issues.
- **Epics** are issues labelled `epic`; their sub-issues are the pieces of work. Each epic has a
  "Done when" list.
- **Spikes** (`type:spike`) are time-boxed research whose output is a decision.
- **Labels:** `type:*` (feature, bug, spike, chore, content), `area:*` (story-engine, world, map, ui,
  infra), `priority:P0–P2`, `size:S/M/L`.
- **Decisions** are recorded in [`decisions.md`](decisions.md), newest first.

## Layout

- `decisions.md` — the decision log.
- Topic folders (for example `story-engine/`, `world/`, `map/`) hold design notes for an epic. Create
  one when an epic's first design note is written, and link it from the epic.
- `archive/` — the phase-era notes (Phases 2–8) and the Phase 7 plan, kept for history. Phase 7's
  campaign data model and schema notes are still accurate for the current runtime.
