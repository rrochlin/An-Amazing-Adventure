# Map prototypes (spike #118)

Throwaway prototypes for choosing a map approach. Not part of the app and never merged; the write-up and
screenshots are in `docs/map/map-prototypes.md` on main.

- `briarwell.ts`: Briarwell's real rooms and cast, plus the hard-coded data the game doesn't have yet
  (points of interest, character positions), each marked `GAP`.
- `NodeGraphMap.tsx` (A): the room graph with points of interest, tokens and highlighted exits.
- `RegionMap.tsx` (B): an illustrated SVG village with clickable points of interest and fog.
- `TileMap.tsx` (C): a top-down tile world with tokens and fog of war.

To view one, render it as the `map` of `GameLayout` in a temporary page (see `index.html` for the page
shell), with `ThemeProvider theme={AppTheme} defaultMode="dark"`, and run `pnpm exec vite`.
