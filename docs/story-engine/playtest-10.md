# Playtest 10: the whole of Briarwell with staging (2026-10-24)

A fresh Briarwell run through the real client, after #193–#199 ([[playtest-9]] found those), taking the
paths playtest 9 didn't: the reluctant answer, the miller's bribe, joining Dunstan, the carved bird and
refusing Holloway. Every authored cue on the way played (Father's shake, Garrick's bow, Ros's laugh,
Father walking to the door and startling, the player's shake), and the game master staged every turn.

## Found and fixed

| Finding | Platform gap | Fix |
|---|---|---|
| The story walked the player to the barn and nothing happened | The barn's nearest free tile was in a corner the mill and barn seal off | #199: people and marks are only placed in a room's walkable area (the tiles that join its ways in) |
| Indoors, Father walked to the door, jumped back a line later, then walked again | `sceneArt` made a new interior object every render; the stage took each as a new map and rebuilt, losing walks, marks and the player's tile | The merged interior is cached, so it's the same object |
| Ros's birthday hop played while the stage still showed Market Lane | Every world change waited for the turn's last line, the scene change home included | Held changes also land at a new scene's title card: the scene plays where it happens |

## Seen, not changed

- The game master still reports coins it shouldn't (dropped every time; the deeds pay through
  `on_success`).
- The game master described Hob behind the barn without staging him there: it stages the player more
  readily than the people they meet.
- Joining Dunstan ends with "a coin finds its way into your pocket all the same": the deed's payment is
  a story rule the narration has to explain.
