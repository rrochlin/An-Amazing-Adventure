# Agent Issue Tracker

Purpose: shared incident log for coding/debug agents working in this repository.

Usage:
- Add new incidents at the top under `Issue NNN`.
- Keep entries evidence-based and include concrete reproduction/debug commands.
- Update `Status` and `Owner` as work progresses.

## Issue 001 — Off-theme narration in session 07f10e44-a39f-4a22-bdc4-706cc6df88d4

- **Date:** 2026-06-19
- **Status:** Open
- **Priority:** High (player-facing narrative quality regression)
- **Owner:** Unassigned
- **Session ID:** `07f10e44-a39f-4a22-bdc4-706cc6df88d4`
- **Environment:** prod (`us-west-2`)
- **DynamoDB table:** `amazing-adventure-prod-sessions`

### Summary

Session narrative drifted from dark-fantasy campaign context (`Lich's Labyrinth`) into a modern retail/Walmart-like scene after the player entered `continue?`. The AI generated automatic doors, a greeter in a red vest named PHIL, fluorescent aisles, and shoppers — entirely unrelated to the campaign setting.

### Evidence Collected

**Session metadata (retrieved via app-native `db.GetGame` path):**

| Field | Value |
|---|---|
| `title` | `Lich's Labyrinth` |
| `theme` | `dark-fantasy` |
| `campaign.campaign_id` | `lichs-labyrinth` |
| `campaign.active_node_id` | `camp_briefing` |
| `campaign.current_objective` | `Understand the mission and the rival party before committing to a plan.` |
| Player name | `boromir` (dwarf / mountain-dwarf / fighter) |
| Player location ID | `f1_target_entry` |
| Room name resolved for `f1_target_entry` | **`Target Entrance`** (empty description) |
| `conversation_count` | 1 |

**Persisted `chat_history` in order:**
1. `[narrative]` "Rain whispers over the canvas of the mercenary camp while steel and bad intentions glint in the firelight."
2. `[narrative]` "The camp briefing begins."
3. `[player]` `continue?`
4. `[narrative]` *(bad output)* "The automatic doors wheeze open with a gust of refrigerated air that smells of plastic packaging, floor wax, and something faintly floral — a display of potted tulips just inside the entrance… A greeter — an older man with a red vest and a name tag reading *PHIL*…"

**Persisted `narrative` context (what the LLM actually sees as conversation history):**
- `[user]` `continue?`
- `[assistant]` *(same bad retail output — now baked into context for all subsequent turns)*

### Technical RCA

**Primary cause — Confidence: High**

The narrator system prompt grounds on player name + current room name only (`server/internal/ai/bedrock.go`, `narratorSystemPrompt`). At time of this turn the exact prompt injected was:

```
The player's name is "boromir" and they are currently in "Target Entrance".
```

`Target Entrance` is semantically ambiguous in modern English — it strongly evokes a retail store entrance (cf. Target Corporation). With player input `continue?` providing no fantasy constraint, the model generates a coherent scene for that semantic context.

**Contributing factors:**

1. **No campaign/theme grounding in narrator prompt** — `narratorSystemPrompt` does not inject campaign id, active node, objective, or required tone. The prompt is generic DM instructions with no setting lock.
2. **Empty room description** — `f1_target_entry` has `description: ""` in the session. Only the bare name `Target Entrance` reaches the model.
3. **Nil narrative history on campaign bootstrap** — `http-games` calls `g.ToSaveState(nil, history)`. The first freeform chat turn has zero prior LLM narrative turns for continuity.
4. **Non-test campaign uses freeform NarrateStream** — only `campaign_id == "test"` takes the deterministic bypass (`ws-chat/main.go:194`). `lichs-labyrinth` goes through open-ended narration.

### Code References

| File | Location | Relevance |
|---|---|---|
| `server/internal/ai/bedrock.go` | `narratorSystemPrompt()` | Builds the system prompt — only injects `owner.Name` and `room.Name` |
| `server/cmd/ws-chat/main.go` | ~line 199 `aiClient.NarrateStream(...)` | Narrator invocation; campaign bypass check at line 194 |
| `server/cmd/http-games/main.go` | `g.ToSaveState(nil, history)` | Bootstrap saves nil narrative history |
| `server/campaigns/lichs-labyrinth/campaign.json` | Room definitions | Source of `Target Entrance` room label |

### Recommended Fixes

1. **Harden `narratorSystemPrompt` with campaign context** *(highest impact)*
   - When `g.Campaign != nil`, inject: campaign id, active node title/summary, current objective, and theme.
   - Add explicit anti-drift guardrail: *"This is a dark fantasy setting. Do not introduce modern real-world settings (stores, vehicles, electricity, technology) unless the campaign explicitly calls for them."*

2. **Include room description in narrator prompt**
   - Currently only `room.Name` is injected; also inject `room.Description` so the model has authored context rather than a semantically bare label.

3. **Rename ambiguous authored room labels**
   - `Target Entrance` → something unambiguous, e.g. `Rival Party Gate` or `Labyrinth Outer Threshold`, in `server/campaigns/lichs-labyrinth/campaign.json`.

4. **Seed narrative history for campaign sessions**
   - On bootstrap, persist the authored opening scene into `narrative` so the first freeform chat turn has prior LLM context. Change `ToSaveState(nil, history)` to pass a `[]game.NarrativeMessage` seeded with the opening narrative.

### Verification Checklist

- [ ] Reproduce: send `continue?` in a fresh `lichs-labyrinth` `camp_briefing` session — confirm it currently produces off-theme output.
- [ ] Apply fix — confirm narration is dark-fantasy coherent for the same input.
- [ ] Unit test: assert `narratorSystemPrompt` contains campaign title/objective when `g.Campaign != nil`.
- [ ] Confirm `test` deterministic campaign path is unaffected.
- [ ] Audit all authored room names in `lichs-labyrinth` for modern English collisions.

### Useful Diagnostic Commands

```bash
# App-native session dump (same binary-key decode path as production)
cd server
SESSIONS_TABLE=amazing-adventure-prod-sessions AWS_REGION=us-west-2 \
  go run ./cmd/_session_dump 07f10e44-a39f-4a22-bdc4-706cc6df88d4 > /tmp/session.json

# Key metadata
jq '{session_id,title,theme,conversation_count,active_node:(.campaign.active_node_id),location:(.players[.owner_id].location_id)}' /tmp/session.json

# Full chat transcript
jq -r '.chat_history | to_entries[] | "[\(.key)] type=\(.value.type)\n\(.value.content)\n---"' /tmp/session.json

# Resolve player location room name
jq -r --arg lid "$(jq -r '.players[.owner_id].location_id' /tmp/session.json)" '.rooms[] | select(.id==$lid) | {id,name,description}' /tmp/session.json

# List all session IDs in prod (decode binary keys)
aws dynamodb scan --region us-west-2 --table-name amazing-adventure-prod-sessions \
  --projection-expression session_id --output json \
  | jq -r '.Items[].session_id.B' \
  | while read b; do printf '%s  ->  ' "$b"; printf '%s' "$b" | base64 -d; echo; done

# Binary key encoding helper
printf '%s' '07f10e44-a39f-4a22-bdc4-706cc6df88d4' | base64
# MDdmMTBlNDQtYTM5Zi00YTIyLWJkYzQtNzA2Y2M2ZGY4OGQ0
```
