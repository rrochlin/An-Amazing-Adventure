---
tags:
  - map
  - spike
  - art
---

# Tileset shortlist (spike #148)

> [!summary]
> Research for a richer GBA/SNES-style tileset with walk-animated characters, within about $20.
> Research only: nothing was bought or downloaded. Rob picks; the decision goes in `decisions.md`.

## Ranked shortlist

### 1. Time Fantasy (finalbossblues) — best style match

- [Fantasy RPG Tileset Pack](https://finalbossblues.itch.io/fantasy-rpg-tileset-pack) $15 ·
  [RPG Sprites 1](https://finalbossblues.itch.io/tf-rpg-charactersprites-1) $15 ·
  [Animated NPC Sprites](https://finalbossblues.itch.io/animated-npc-sprites-pack) $5 ·
  [Open RPG Fantasy Tilesets](https://finalbossblues.itch.io/openrtp-tiles) free, CC0
- **Licence** (author statements; no formal file on the store page): "OK for commercial projects, no
  engine limitations, no credit necessary"; "My terms disallow redistribution"; the RPG Maker-only
  clause applies only to the Degica store copies, so buy on itch. Read the licence in the zip.
- **16×16** tiles, characters 16×32. Village and shop interiors and exteriors, castles, caves,
  world-map tiles, animated water and doors. 80+ characters including soldiers, townsfolk, children
  (Animated NPC pack). 4-direction walks, 3 frames (per a commenter).
- **Gaps:** no elderly or bandits listed (soldiers stand in); no night or fire variants; the full
  tiles + characters set is $30.

### 2. Cute Fantasy RPG (Kenmi) — cheapest complete set

- [Store](https://kenmi-art.itch.io/cute-fantasy-rpg); all 10 packs $17.50 on the bundle.
- Licence: commercial use allowed, modify freely, "can't be resold or redistributed", credit Kenmi.
  The free tier is non-commercial only; pay at least $3.99 for the premium licence.
- 16×16; village, interiors, market stalls, customisable player. Chibi, closer to Stardew than FF6.
  No children, elderly or bandits listed; no world-map tiles.

### 3. Ninja Adventure (Pixel-boy & AAA) — best free fallback

- [Store](https://pixel-boy.itch.io/ninja-adventure-asset-pack). CC0.
- 50+ animated characters, monsters, interiors and exteriors, effects, music. NES/GB-Zelda chibi
  with a Japanese theme; every character the same height.

### 4. LPC / Universal LPC generator — free, best variety, wrong scale

- 32×32 tiles, 64×64 frames, 9-frame walks; child and elderly bases exist but many assets aren't
  drawn for them. Licences are mixed (CC0, CC-BY, CC-BY-SA, GPL): share-alike covers the art and
  our edits, and a credits file must be shown in the game. Taller, western-RPG proportions.

### 5. Sunnyside World (Daniel Diggle) — weak on 4-direction walks

- Name your price; commercial OK, no credit needed, no AI training. Main sprites move
  left/right only. Farm-sim look.

## Excluded

- **Mana Seed (Seliel the Shaper):** its licence forbids use "in a project alongside 'AI'
  generated imagery, writing, code" — a direct conflict with an AI game master. Also one product
  per purchase and over budget.
- **Cainos** ($39.99 village), **Sprout Lands** (farm kit, free tier non-commercial), **LimeZu**
  (modern clothing), **GuttyKreum** (modern Japan), **Stonebit "Villagers & Townsfolk"** (AI-assisted,
  published days ago, unrated), **Tiny 16** (no better than Kenney).

## Recommendation

**Time Fantasy: RPG Sprites 1 ($15) + Animated NPC Sprites ($5) + the free CC0 Open RPG tiles = $20.**
The only candidate that reads as SNES/GBA, with the most permissive paid licence. The cost is some
colour matching between the CC0 tiles and the paid sprites. For about $30, the $15 Fantasy RPG
Tileset Pack replaces the CC0 tiles and matches the palette.

Fallbacks: Cute Fantasy ($17.50, chibi) or Ninja Adventure (free, CC0).

## Licence notes for a public web game

- No paid licence addresses files served from a CDN; their bans are on redistribution as an asset
  pack. Ship packed atlases, not the original pack layout, and no browsable asset index.
- **This repo is public.** Paid packs must not be committed to it: they would be redistributed.
  Keep them in a private bucket or a private submodule and pull them in at build time.
