import { GROUND_TILES, OBJECT_TILES, TILE, type MapArt, type TileRef } from '../map/mapArt';
import { mapScale } from '../map/WorldMap';

/**
 * The scene as data for the Phaser stage: tile layers as tile-index grids,
 * and where the camera looks. Pure, so it is tested without a renderer.
 */

/** Tiles of the Tiny Dungeon sheet start here in a layer's grid (Tiny Town's start at 0). */
export const DUNGEON_GID = 1000;
/**
 * Tiles drawn beyond the map's edge in an outdoor scene, repeating the edge
 * tiles, so a room at the edge is framed by more of the same terrain.
 */
export const SCENE_PAD = 32;
/** Object characters that continue beyond the map's edge: forest. */
export const CONTINUES = new Set('123456789');

const gid = (ref: TileRef | undefined): number =>
   ref === undefined ? -1 : typeof ref === 'number' ? (ref < 0 ? -1 : ref) : DUNGEON_GID + ref.dungeon;

/**
 * A layer as a grid of tile indices (-1 is empty), with `pad` tiles of
 * border that repeat the nearest edge tile. Ground always continues; of the
 * objects only forest does (a beehive at the edge doesn't become a row).
 */
export function layerGrid(rows: string[], legend: Record<string, TileRef>, pad: number, objects: boolean): number[][] {
   const h = rows.length;
   const w = rows[0]?.length ?? 0;
   const clamp = (v: number, max: number) => Math.min(Math.max(v, 0), max - 1);
   const grid: number[][] = [];
   for (let y = -pad; y < h + pad; y++) {
      const row: number[] = [];
      for (let x = -pad; x < w + pad; x++) {
         const outside = y < 0 || y >= h || x < 0 || x >= w;
         const ch = rows[clamp(y, h)]?.[clamp(x, w)];
         row.push(ch === undefined || (outside && objects && !CONTINUES.has(ch)) ? -1 : gid(legend[ch]));
      }
      grid.push(row);
   }
   return grid;
}

/** The art's two layers, padded for an outdoor scene; interiors stand alone. */
export function sceneLayers(art: MapArt): { pad: number; ground: number[][]; objects: number[][] } {
   const pad = art.interior ? 0 : SCENE_PAD;
   return {
      pad,
      ground: layerGrid(art.ground, GROUND_TILES, pad, false),
      objects: layerGrid(art.objects, OBJECT_TILES, pad, true),
   };
}

/**
 * Where the camera looks: a whole-number zoom that frames the current room,
 * and the world point at the canvas centre such that the room sits in the
 * middle of the area left between the sign (top) and the text panel
 * (bottom).
 */
export function stageCamera(
   art: MapArt,
   roomId: string | undefined,
   view: { w: number; h: number },
   insetTop: number,
   insetBottom: number,
): { zoom: number; x: number; y: number } {
   const area = { w: view.w, h: Math.max(0, view.h - insetTop - insetBottom) };
   const zoom = mapScale('scene', area, art.width * TILE, art.height * TILE, art, roomId);
   const [rx, ry, rw, rh] = (roomId && art.rooms[roomId]) || [0, 0, art.width, art.height];
   const cx = (rx + rw / 2) * TILE;
   const cy = (ry + rh / 2) * TILE;
   return { zoom, x: cx, y: cy - (insetTop - insetBottom) / 2 / zoom };
}

/** The world point a character stands on: the bottom centre of its tile. */
export function standPoint(x: number, y: number): { x: number; y: number } {
   return { x: x * TILE + TILE / 2, y: (y + 1) * TILE };
}

export type Tile = { x: number; y: number };
export type Direction = 'up' | 'down' | 'left' | 'right';
export const STEPS: Record<Direction, Tile> = { up: { x: 0, y: -1 }, down: { x: 0, y: 1 }, left: { x: -1, y: 0 }, right: { x: 1, y: 0 } };

/** The room a tile belongs to, if any (ground outside every room isn't walkable). */
export function roomAt(art: MapArt, { x, y }: Tile): string | undefined {
   for (const [id, [rx, ry, rw, rh]] of Object.entries(art.rooms)) {
      if (x >= rx && x < rx + rw && y >= ry && y < ry + rh) return id;
   }
   return undefined;
}

/** Whether the player can stand on a tile: inside a room, on ground, with no object on it. */
export function standable(art: MapArt, t: Tile): boolean {
   if (t.y < 0 || t.y >= art.height || t.x < 0 || t.x >= art.width) return false;
   if (art.ground[t.y]?.[t.x] === '_') return false; // outside an interior
   if ((art.objects[t.y]?.[t.x] ?? ' ') !== ' ') return false;
   return roomAt(art, t) !== undefined;
}

/**
 * Whether the player may step from one tile to the next: the next tile is
 * standable, free of other characters, and in the same room or one the
 * current room connects to.
 */
export function canStep(
   art: MapArt,
   rooms: Record<string, { connections?: Record<string, string> }>,
   from: Tile,
   to: Tile,
   occupied: Set<string>,
): boolean {
   if (!standable(art, to) || occupied.has(`${to.x},${to.y}`)) return false;
   const a = roomAt(art, from);
   const b = roomAt(art, to);
   if (a === b) return true;
   return !!a && !!b && Object.values(rooms[a]?.connections ?? {}).includes(b);
}

/**
 * The shortest walk from one tile to another (breadth-first; the maps are
 * small), as the tiles to step on in order, not counting the start. Empty
 * when there's no way there.
 */
export function findPath(
   art: MapArt,
   rooms: Record<string, { connections?: Record<string, string> }>,
   from: Tile,
   to: Tile,
   occupied: Set<string>,
): Tile[] {
   const key = (t: Tile) => `${t.x},${t.y}`;
   if (key(from) === key(to)) return [];
   const came = new Map<string, Tile | null>([[key(from), null]]);
   const queue: Tile[] = [from];
   while (queue.length) {
      const at = queue.shift()!;
      if (key(at) === key(to)) {
         const path: Tile[] = [];
         for (let t: Tile | null = at; t && key(t) !== key(from); t = came.get(key(t)) ?? null) path.unshift(t);
         return path;
      }
      for (const d of Object.values(STEPS)) {
         const next = { x: at.x + d.x, y: at.y + d.y };
         if (came.has(key(next)) || !canStep(art, rooms, at, next, occupied)) continue;
         came.set(key(next), at);
         queue.push(next);
      }
   }
   return [];
}

/**
 * Whose line it is: the character on screen the speaker's name means, by
 * name, by what else lines call them ("Father"), or by one word of their
 * name ("Bess" for Little Bess). "player" when it's the player's own line.
 */
export function whoSpeaks(
   speaker: string,
   people: { id: string; name: string; called?: string[] }[],
   playerName?: string,
): string | undefined {
   const said = speaker.trim().toLowerCase();
   if (!said) return undefined;
   if (playerName && said === playerName.trim().toLowerCase()) return 'player';
   const exact = people.find((p) => p.name.toLowerCase() === said || p.called?.some((c) => c.toLowerCase() === said));
   if (exact) return exact.id;
   return people.find((p) => p.name.toLowerCase().split(/\s+/).includes(said))?.id;
}

/** A bubble shows lines up to this long; a longer one shows "…" (the panel has it). */
export const BUBBLE_CHARS = 70;

/** What a speech bubble says for a line. */
export function bubbleText(text: string): string {
   const t = text.trim();
   return t.length <= BUBBLE_CHARS ? t : '…';
}
