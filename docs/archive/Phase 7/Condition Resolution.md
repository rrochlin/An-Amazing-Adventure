---
tags:
  - phase-7
  - ai
  - conditions
  - transitions
---

# Condition Resolution

> [!summary]
> Node transitions should be deterministic.
> AI is only used to decide whether a turn satisfied one of the campaign's authored conditions.

Related notes:

- `[[Story]]`
- `[[Narrator Adherence]]`
- `[[Campaign JSON Schema]]`
- `[[Selected Stack]]`

## Why This Layer Exists

Some campaign conditions are ambiguous in natural language.

Examples:

- did the player actually choose the hidden entrance, or just ask about it?
- did the player meaningfully conclude the briefing?
- did the player warn the novices, threaten them, or ignore them?
- did the rival party become alerted?

These are awkward to infer with only rigid parser logic.

## Phase 7 Rule

Use AI for **classification**, not **branch selection**.

The flow should be:

1. campaign JSON defines valid conditions
2. the active scene runs through AI narration, Yarn dialogue, or a hybrid scene
3. condition resolver maps the turn into structured fulfilled conditions
4. code validates the output against the campaign vocabulary
5. code applies flags / labels / counters
6. code evaluates transitions deterministically

## Condition Vocabulary

The resolver should only be able to emit known condition keys.

Examples:

- `briefing_complete`
- `entry_route`
- `timing_strategy`
- `target_party_alerted`
- `novices_warned`
- `ambush_site_prepared`

This vocabulary should come from the campaign definition.

## Suggested Resolver Input

- active story node
- player input
- narrator output
- Yarn output or emitted dialogue commands, if the node is Yarn-backed
- allowed condition keys for this node
- recent structured world events

For Phase 7, the resolver should use the **latest turn only**, not the full session history.

## Suggested Resolver Output

```json
{
  "fulfilled_conditions": [
    {
      "type": "bool_flag",
      "key": "briefing_complete",
      "value": true
    },
    {
      "type": "label",
      "key": "entry_route",
      "value": "hidden_entrance"
    }
  ]
}
```

For Phase 7, keep the output limited to:

- boolean flags
- labels / enum selections
- numeric counters

It should not directly update objectives or emit broad scene-event payloads.

## Validation Rules

The server should reject resolver output if:

- it references an unknown key
- it returns a disallowed value
- it attempts to set `next_node_id`
- it attempts to apply arbitrary effects

## Deterministic Transition Example

If the resolver returns:

```json
{
  "fulfilled_conditions": [
    { "type": "bool_flag", "key": "briefing_complete", "value": true },
    { "type": "label", "key": "entry_route", "value": "hidden_entrance" },
    { "type": "label", "key": "timing_strategy", "value": "before_target" }
  ]
}
```

Then code can deterministically evaluate:

- `briefing_complete == true`
- `entry_route in {target_entrance, novice_entrance, hidden_entrance}`
- `timing_strategy != ""`

and transition from `camp_briefing` to `floor1_shadow_game` or `choose_entry_plan`, depending on the authored rules.

## Guardrail

> [!important]
> If AI says what conditions were met, and code says what that means, the system stays inspectable.
>
> If AI says both what happened and where the story goes next, the campaign will drift.

## Choice UX Implication

Campaign-specific choices like entrance and timing should remain **freeform chat choices**.

The narrator can present them naturally in-scene, and the resolver maps the player's answer onto structured values like:

- `entry_route = hidden_entrance`
- `timing_strategy = before_target`

This preserves immersion while keeping the underlying campaign state explicit.
