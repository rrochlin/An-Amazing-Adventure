import { describe, it, expect } from 'vitest';
import { BRIARWELL_ART } from '@/components/map/art/briarwell';
import { GROUND_TILES, OBJECT_TILES } from '@/components/map/mapArt';
import { DUNGEON_GID, SCENE_PAD, canStep, findPath, layerGrid, roomAt, sceneLayers, stageCamera, standable, standPoint } from '@/components/stage/stageModel';

describe('stage layers', () => {
   it('turns the art into tile indices, Tiny Dungeon tiles offset, empty as -1', () => {
      const grid = layerGrid(['.T', ' o'], { ...GROUND_TILES, ...OBJECT_TILES }, 0, true);
      expect(grid).toEqual([
         [GROUND_TILES['.'], DUNGEON_GID + (OBJECT_TILES.T as { dungeon: number }).dungeon],
         [-1, GROUND_TILES.o],
      ]);
   });

   it('pads an outdoor scene with its edge terrain; forest continues, a beehive does not', () => {
      const ground = layerGrid(['.o'], GROUND_TILES, 1, false);
      expect(ground).toEqual([
         [0, 0, 43, 43],
         [0, 0, 43, 43],
         [0, 0, 43, 43],
      ]);
      const objects = layerGrid(['5&'], OBJECT_TILES, 1, true);
      expect(objects[1]).toEqual([19, 19, 94, -1]);
   });

   it('stands an interior alone: no padding, its void empty', () => {
      const inside = sceneLayers(BRIARWELL_ART.interiors!.cottage_interior);
      expect(inside.pad).toBe(0);
      expect(inside.ground[0][0]).toBe(-1);
      const outside = sceneLayers(BRIARWELL_ART);
      expect(outside.pad).toBe(SCENE_PAD);
      expect(outside.ground).toHaveLength(BRIARWELL_ART.height + 2 * SCENE_PAD);
   });
});

describe('stage camera', () => {
   it('centres the room between the sign and the text panel, at a whole-number zoom', () => {
      // Village square: 10x8 tiles from (8, 9); centre (13, 13) tiles = (208, 208) px.
      const cam = stageCamera(BRIARWELL_ART, 'village_square', { w: 900, h: 650 }, 50, 200);
      expect(Number.isInteger(cam.zoom)).toBe(true);
      expect(cam.x).toBe(208);
      // The visible middle is 75px above the canvas middle (the panel is taller than the
      // sign), so the camera looks 75 screen px below the room centre.
      expect(cam.y).toBeCloseTo(208 - (50 - 200) / 2 / cam.zoom);
   });

   it('stands characters on the bottom centre of their tile', () => {
      expect(standPoint(2, 3)).toEqual({ x: 40, y: 64 });
   });
});

describe('walking', () => {
   const rooms = {
      family_cottage: { connections: { east: 'village_square' } },
      village_square: { connections: { west: 'family_cottage', north: 'market_lane', east: 'old_mill', south: 'meadow' } },
      market_lane: { connections: { south: 'village_square' } },
      old_mill: { connections: { west: 'village_square' } },
      meadow: { connections: { north: 'village_square', south: 'woods_edge' } },
      woods_edge: { connections: { north: 'meadow' } },
   };
   const none = new Set<string>();

   it('stands only on free ground inside a room', () => {
      expect(standable(BRIARWELL_ART, { x: 12, y: 13 })).toBe(true); // the square
      expect(standable(BRIARWELL_ART, { x: 12, y: 11 })).toBe(false); // the well
      expect(standable(BRIARWELL_ART, { x: 1, y: 1 })).toBe(false); // ground outside every room
      expect(roomAt(BRIARWELL_ART, { x: 12, y: 13 })).toBe('village_square');
   });

   it('steps into a connected room, never into one that does not connect', () => {
      // Cottage (x < 8) to the square (x >= 8): connected.
      expect(canStep(BRIARWELL_ART, rooms, { x: 7, y: 13 }, { x: 8, y: 13 }, none)).toBe(true);
      // Square to the mill boundary (x = 17 → 18) connects; the cottage to the lane does not.
      const cottageTop = { x: 6, y: 9 };
      expect(roomAt(BRIARWELL_ART, cottageTop)).toBe('family_cottage');
      expect(canStep(BRIARWELL_ART, rooms, cottageTop, { x: 6, y: 8 }, none)).toBe(false);
   });

   it('walks around objects and other characters by the shortest way', () => {
      const from = { x: 12, y: 12 };
      const to = { x: 12, y: 10 }; // the well (12, 11) is in between
      const path = findPath(BRIARWELL_ART, rooms, from, to, none);
      expect(path.at(-1)).toEqual(to);
      expect(path).toHaveLength(4); // around the well
      expect(path.some((t) => t.x === 12 && t.y === 11)).toBe(false);
      const blocked = findPath(BRIARWELL_ART, rooms, from, to, new Set(['11,11', '13,11', '11,12', '13,12']));
      expect(blocked.some((t) => ['11,11', '13,11'].includes(`${t.x},${t.y}`))).toBe(false);
   });

   it('finds no way to a tile it cannot stand on', () => {
      expect(findPath(BRIARWELL_ART, rooms, { x: 12, y: 12 }, { x: 12, y: 11 }, none)).toEqual([]);
   });
});
