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
