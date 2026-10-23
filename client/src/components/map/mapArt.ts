import townSheet from '@/assets/kenney/tiny-town.png';
import dungeonSheet from '@/assets/kenney/tiny-dungeon.png';
import { BRIARWELL_ART } from './art/briarwell';

/**
 * Pixel-art map for an authored campaign, drawn from Kenney's CC0 Tiny Town
 * (terrain, buildings, props) and Tiny Dungeon (character sprites) sheets.
 * Room IDs and point-of-interest IDs match the campaign's campaign.json;
 * map.test.ts checks that.
 */
export interface MapArt {
   /** Size in tiles. */
   width: number;
   height: number;
   /** Rows of ground characters (see GROUND_TILES); every cell is filled. */
   ground: string[];
   /** Rows of object characters (see OBJECT_TILES); ' ' is empty. */
   objects: string[];
   /** Room ID → [x, y, w, h] in tiles: the area the room covers. */
   rooms: Record<string, [number, number, number, number]>;
   /** Point-of-interest ID → [x, y] tile. */
   points_of_interest: Record<string, [number, number]>;
   /** Character ID (actor ID, or "player") → sprite. */
   cast: Record<string, CastSprite>;
   /**
    * An interior: drawn on its own, with no terrain continuing past its edge,
    * and fitted to the view rather than filling it.
    */
   interior?: boolean;
   /**
    * Interiors by scene map ID (campaign.json map.scene_maps). Each covers
    * one campaign room; a scene played on it shows it in place of the room.
    */
   interiors?: Record<string, MapArt>;
}

/**
 * A tile: an index in the Tiny Town sheet, or `{ dungeon: n }` for the Tiny
 * Dungeon sheet (indoor furniture). A negative index is empty (an interior's
 * surroundings).
 */
export type TileRef = number | { dungeon: number };

export interface CastSprite {
   /** Tile index in the Tiny Dungeon sheet. */
   sprite: number;
}

export const TILE = 16;
export const SHEET_COLUMNS = 12;
export const TOWN_SHEET = townSheet;
export const DUNGEON_SHEET = dungeonSheet;

/** Ground characters → Tiny Town tile index. */
export const GROUND_TILES: Record<string, TileRef> = {
   '_': -1, // nothing: outside an interior
   f: 25, // earthen floor
   '.': 0, // grass
   ',': 1, // grass tufts
   '*': 2, // flowers
   o: 43, // cobbles
   // dirt patch edges and centre (laid out like a numpad)
   q: 12, w: 13, e: 14,
   a: 24, s: 25, d: 26,
   z: 36, x: 37, c: 38,
};

/** Object characters → Tiny Town tile index. */
export const OBJECT_TILES: Record<string, TileRef> = {
   O: 16, // round tree
   P: 28, // pine
   Y: 3, // autumn tree
   b: 5, // bush
   v: 17, // long grass
   m: 29, // mushrooms
   // forest edges and fill (numpad layout)
   '7': 6, '8': 7, '9': 8,
   '4': 18, '5': 19, '6': 20,
   '1': 30, '2': 31, '3': 32,
   // red roof: top row, bottom row, gable
   R: 52, E: 53, N: 54, r: 64, e: 65, n: 66, G: 67,
   // grey roof: top row, bottom row, gable
   H: 48, I: 49, J: 50, h: 60, i: 61, j: 62, g: 63,
   // timber walls: left, plain, right, window, door
   L: 72, M: 73, K: 75, W: 84, D: 85,
   // stone walls: left, plain, right, window, door
   l: 76, u: 77, k: 79, X: 88, Z: 89,
   // fence: left end, rail, right end, post
   '<': 80, '-': 81, '>': 82, '|': 59,
   '@': 104, // well
   '#': 57, // cart / stall
   '%': 130, // barrel
   '&': 94, // beehive
   '=': 106, // log bench
   '?': 83, // signpost
   p: 116, // pitchfork
   // indoors
   F: { dungeon: 28 }, // hearth
   Q: 107, // cooking pot
   T: { dungeon: 72 }, // table
   t: { dungeon: 73 }, // stool
   B: { dungeon: 66 }, // straw pallet (bed)
   C: { dungeon: 89 }, // chest
   S: { dungeon: 63 }, // shelves
};

/** Sprites for characters a campaign doesn't cast. */
export const DEFAULT_SPRITES = { player: 85, friendly: 98, hostile: 111 };

const ART: Record<string, MapArt> = {
   briarwell: BRIARWELL_ART,
};

/** The pixel-art map for a campaign, if it has one. */
export function mapArtFor(campaignId: string | undefined): MapArt | undefined {
   return campaignId ? ART[campaignId] : undefined;
}

/**
 * The art to draw for the current scene: the scene's interior while the
 * player is in the room it belongs to, else the campaign's map. Interiors
 * share the campaign's cast.
 */
export function sceneArt(
   art: MapArt | undefined,
   sceneMap: { id: string; room_id: string } | null | undefined,
   currentRoomId: string | undefined,
): MapArt | undefined {
   if (!art || !sceneMap || sceneMap.room_id !== currentRoomId) return art;
   const inside = art.interiors?.[sceneMap.id];
   return inside ? { ...inside, cast: { ...art.cast, ...inside.cast } } : art;
}

/** Top-left pixel of a tile in a 12-column sheet. */
export function sheetOffset(index: number): [number, number] {
   return [(index % SHEET_COLUMNS) * TILE, Math.floor(index / SHEET_COLUMNS) * TILE];
}
