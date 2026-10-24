import { describe, it, expect } from 'vitest';
import { BRIARWELL_ART } from '@/components/map/art/briarwell';
import { GROUND_TILES, OBJECT_TILES } from '@/components/map/mapArt';
import { DUNGEON_GID, SCENE_PAD, layerGrid, sceneLayers, stageCamera, standPoint } from '@/components/stage/stageModel';

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
