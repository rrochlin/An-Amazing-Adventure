# Scene maps (interiors)

> #152. A scene can be played on a map of its own, an interior of its room, instead of the room's patch
> of the village map. Briarwell's three cottage scenes (the morning errand, the birthday, the raid) are
> played inside the cottage.

## Authoring

In `campaign.json`, `map.scene_maps` declares the interior and the room it is inside. A story node
names it with `scene_map`:

```json
"map": {
  "scene_maps": {
    "cottage_interior": { "room_id": "family_cottage", "name": "Inside the Cottage", "description": "..." }
  }
},
"story_nodes": { "birthday": { "location_id": "family_cottage", "scene_map": "cottage_interior" } }
```

Validation requires that a scene map's room exists, that it has a name, and that a node using it has
that room as its `location_id`. The description goes into the game master's brief while the player is
there, so the prose matches what is drawn.

## Runtime

Entering a node sets `campaign.active_scene_map` (`{id, room_id, name}`) and leaving it clears it. The
client draws the interior from `art.interiors[id]` (in `client/src/components/map/art/`) **while the
player is in that room**. If the story moves them out mid-scene, the village map comes back. The
location sign shows the interior's name.

Interior art is an ordinary `MapArt` with `interior: true`:

- One room rectangle, for that room.
- `_` ground is empty, so the dark surrounds it.
- It is fitted whole to the view instead of filling it, with no terrain continuing past its walls.
- Furniture comes from the Tiny Dungeon sheet (`{ dungeon: n }` in the legend): table, stools, straw
  beds, chest, shelves and hearth.

The map test checks every scene map has art, each interior covers only its room, and its points of
interest belong to that room.
