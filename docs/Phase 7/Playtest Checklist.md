---
tags:
  - phase-7
  - playtest
  - qa
---

# Phase 7 Playtest Checklist

> [!summary]
> This checklist is intentionally narrow.
>
> Use it to validate the quest state machine, dialogue alignment, and refresh persistence in the deterministic `test` campaign probe.

Related notes:

- `[[Ideas]]`
- `[[implementation]]`

## Session Info

- Date:
- Environment:
- Session ID:
- Campaign: `test`

## Goal

Validate only these things:

- [ ] Node state changes when expected.
- [ ] Objective state changes when expected.
- [ ] Dialogue asset and dialogue node stay aligned with the active story node.
- [ ] State still looks correct after refresh.
- [ ] The hidden branch, target branch, and failure branch all behave deterministically.

## Expected Test Flow

Initial state:

- [ ] Active node is `intro`.
- [ ] Active objective tells me to send a message containing `proceed`.
- [ ] Active dialogue shows the intro dialogue asset/node.
- [ ] Opening dialogue text tells me to send `proceed`.

Step 1:

- [ ] Send a message containing `proceed`.
- [ ] Active node becomes `route_choice`.
- [ ] Intro objective becomes completed.
- [ ] Route choice objective becomes active.
- [ ] Active dialogue switches to the route dialogue asset/node.
- [ ] New dialogue text tells me to send `hidden`, `target`, or `novice`.

Hidden branch:

- [ ] Send `hidden`.
- [ ] Active node becomes `hidden_probe`.
- [ ] Route choice objective becomes completed.
- [ ] Hidden probe objective becomes active.
- [ ] Dialogue state clears because `hidden_probe` is an `ai_scene`.
- [ ] Send a message containing `scan`.
- [ ] Active node becomes `hidden_verification`.
- [ ] Hidden probe objective becomes completed.
- [ ] Hidden verification objective becomes active.
- [ ] Active dialogue switches to the verification dialogue asset/node.
- [ ] Verification dialogue text tells me to send `verify`.
- [ ] Send a message containing `verify`.
- [ ] Active node becomes `complete`.
- [ ] Hidden verification objective becomes completed.
- [ ] Completion objective is completed.
- [ ] Dialogue state no longer points at a dialogue asset/node.

Target branch:

- [ ] Start a fresh `test` session.
- [ ] Send `proceed`.
- [ ] Send `target`.
- [ ] Active node becomes `target_probe`.
- [ ] Target probe objective becomes active.
- [ ] Active dialogue switches to the room dialogue asset/node.
- [ ] Send a message containing `inspect room`.
- [ ] Active node becomes `complete`.
- [ ] Target probe objective becomes completed.
- [ ] Completion objective is completed.

Failure branch:

- [ ] Start a fresh `test` session.
- [ ] Send `proceed`.
- [ ] Send `novice`.
- [ ] Active node becomes `route_failure`.
- [ ] Route choice objective becomes failed.
- [ ] Dialogue state clears because the failure node is an `ai_scene`.

Refresh:

- [ ] Refresh after `route_choice` and state is still `route_choice` with route dialogue active.
- [ ] Refresh after `hidden_verification` and state is still `hidden_verification` with verification dialogue active.
- [ ] Refresh after `complete` and state is still `complete`.

## Feedback

- What state looked correct:
- What state drifted:
- What was still confusing:
- Suggested next systems fix:

## General notes:


## Bug Log

### Bug 1

- Steps:
- Expected:
- Actual:
- Severity:

### Bug 2

- Steps:
- Expected:
- Actual:
- Severity:

### Bug 3

- Steps:
- Expected:
- Actual:
- Severity:
