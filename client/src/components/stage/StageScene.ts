import Phaser from 'phaser';
import type { RoomView } from '@/types/types';
import { DUNGEON_SHEET, TILE, TOWN_SHEET, type MapArt } from '../map/mapArt';
import { placeTokens, type Token } from '../map/WorldMap';
import { DUNGEON_GID, sceneLayers, stageCamera, standPoint } from './stageModel';

/** What the stage shows: React hands this in on every change. */
export interface StageView {
   art: MapArt;
   rooms: Record<string, RoomView>;
   currentRoomId?: string;
   playerName?: string;
   playerScale?: number;
   /** Pixels covered at the top (the location sign) and bottom (the text panel). */
   insetTop: number;
   insetBottom: number;
}

/** How long a camera pan or a character's walk takes. */
const GLIDE_MS = 800;
const SHADE = 0x08050c;
const GOLD = 0xf5d77a;

/**
 * The scene as a Phaser game: the current room (or its interior), the
 * characters in it, and its places. The camera frames the room between the
 * sign and the text panel; it pans only when the player changes room, and a
 * character walks only when their tile changes, so nothing flies about when
 * the view resizes or the panel grows.
 */
export class StageScene extends Phaser.Scene {
   private view?: StageView;
   private ready = false;
   private art?: MapArt;
   private layers: Phaser.Tilemaps.TilemapLayer[] = [];
   private shade?: Phaser.GameObjects.Graphics;
   private places: Phaser.GameObjects.GameObject[] = [];
   private sprites = new Map<string, Phaser.GameObjects.Sprite>();
   private placed: Token[] = [];
   private label?: Phaser.GameObjects.Text;
   private framedRoom?: string;

   constructor(view: StageView) {
      super('stage');
      this.view = view;
   }

   preload() {
      this.load.image('town', TOWN_SHEET);
      this.load.spritesheet('dungeon', DUNGEON_SHEET, { frameWidth: TILE, frameHeight: TILE });
   }

   create() {
      this.ready = true;
      this.label = this.add
         .text(0, 0, '', { fontFamily: 'Cinzel, Georgia, serif', fontSize: '12px', color: '#f5d77a', backgroundColor: '#0c0810cc', padding: { x: 4, y: 2 } })
         .setOrigin(0.5, 1)
         .setDepth(100)
         .setVisible(false);
      this.scale.on('resize', () => this.frame(false));
      if (this.view) this.render(this.view);
   }

   /** Called by React on every change. */
   setView(view: StageView) {
      this.view = view;
      if (this.ready) this.render(view);
   }

   private render(view: StageView) {
      if (view.art !== this.art) this.build(view.art);
      this.syncPlaces(view);
      this.syncCharacters(view);
      this.drawShade(view);
      // Pan only when the player has moved to another room.
      this.frame(this.framedRoom !== undefined && this.framedRoom !== view.currentRoomId);
      this.framedRoom = view.currentRoomId;
   }

   /** Tile layers for a map (a new map starts fresh: no pan, no walks). */
   private build(art: MapArt) {
      this.art = art;
      this.layers.forEach((l) => l.tilemap.destroy());
      this.layers = [];
      this.sprites.forEach((s) => s.destroy());
      this.sprites.clear();
      this.placed = [];
      this.framedRoom = undefined;
      const { pad, ground, objects } = sceneLayers(art);
      for (const [i, grid] of [ground, objects].entries()) {
         const map = this.make.tilemap({ data: grid, tileWidth: TILE, tileHeight: TILE });
         const town = map.addTilesetImage('town', 'town', TILE, TILE, 0, 0, 0);
         const dungeon = map.addTilesetImage('dungeon', 'dungeon', TILE, TILE, 0, 0, DUNGEON_GID);
         const layer = map.createLayer(0, [town!, dungeon!], -pad * TILE, -pad * TILE) as Phaser.Tilemaps.TilemapLayer;
         layer.setDepth(i);
         this.layers.push(layer);
      }
   }

   /** The current room stands out: everything around it is dimmed. */
   private drawShade(view: StageView) {
      this.shade ??= this.add.graphics().setDepth(5);
      this.shade.clear();
      const rect = view.currentRoomId ? view.art.rooms[view.currentRoomId] : undefined;
      if (!rect) return;
      const [x, y, w, h] = rect.map((v) => v * TILE);
      const far = 4000;
      this.shade.fillStyle(SHADE, 0.5);
      this.shade.fillRect(-far, -far, far * 2 + w, far + y); // above
      this.shade.fillRect(-far, y + h, far * 2 + w, far); // below
      this.shade.fillRect(-far, y, far + x, h); // left
      this.shade.fillRect(x + w, y, far, h); // right
   }

   /** A small diamond for each place in the room, named on hover. */
   private syncPlaces(view: StageView) {
      this.places.forEach((p) => p.destroy());
      this.places = [];
      const room = view.currentRoomId ? view.rooms[view.currentRoomId] : undefined;
      for (const poi of room?.points_of_interest ?? []) {
         const at = view.art.points_of_interest[poi.id];
         if (!at) continue;
         const diamond = this.add
            .rectangle(at[0] * TILE + TILE / 2, at[1] * TILE - 2, 4, 4, 0xc39bd3)
            .setStrokeStyle(1, 0x000000)
            .setAngle(45)
            .setDepth(8)
            .setInteractive({ useHandCursor: false });
         this.nameOnHover(diamond, poi.name);
         this.places.push(diamond);
      }
   }

   /** Characters in the room: placed where they stand, walking when they move. */
   private syncCharacters(view: StageView) {
      this.placed = placeTokens(view.art, view.rooms, view.currentRoomId, view.playerName ?? 'You', view.playerScale ?? 1, this.placed);
      const here = this.placed.filter((t) => t.roomId === view.currentRoomId);
      const seen = new Set<string>();
      for (const t of here) {
         seen.add(t.id);
         const at = standPoint(t.x, t.y);
         let sprite = this.sprites.get(t.id);
         if (!sprite) {
            sprite = this.add.sprite(at.x, at.y, 'dungeon', t.sprite).setOrigin(0.5, 1).setDepth(10);
            sprite.setInteractive();
            this.nameOnHover(sprite, t.name);
            if (t.isPlayer) this.glow(sprite);
            this.sprites.set(t.id, sprite);
         } else if (sprite.x !== at.x || sprite.y !== at.y) {
            this.tweens.add({ targets: sprite, x: at.x, y: at.y, duration: GLIDE_MS, ease: 'Sine.easeInOut' });
         }
         sprite.setFrame(t.sprite).setScale(t.scale || 1);
         sprite.setData('name', t.name);
      }
      for (const [id, sprite] of this.sprites) {
         if (!seen.has(id)) {
            sprite.destroy();
            this.sprites.delete(id);
         }
      }
   }

   private glow(sprite: Phaser.GameObjects.Sprite) {
      // Filters need WebGL; on the canvas renderer the player just isn't outlined.
      try {
         sprite.enableFilters();
         sprite.filters?.internal.addGlow(GOLD, 2, 0, 1, false, 4, 2);
      } catch {
         /* no filters */
      }
   }

   private nameOnHover(target: Phaser.GameObjects.GameObject & { x: number; y: number; displayHeight?: number }, name: string) {
      target.on('pointerover', () => {
         if (!this.label) return;
         const zoom = this.cameras.main.zoom;
         this.label
            .setText(target.getData('name') ?? name)
            .setScale(1 / zoom)
            .setPosition(target.x, target.y - (target.displayHeight ?? TILE) - 2)
            .setVisible(true);
      });
      target.on('pointerout', () => this.label?.setVisible(false));
   }

   /** Frame the current room between the sign and the text panel. */
   private frame(pan: boolean) {
      if (!this.view || !this.ready) return;
      const cam = this.cameras.main;
      const { zoom, x, y } = stageCamera(this.view.art, this.view.currentRoomId, { w: this.scale.width, h: this.scale.height }, this.view.insetTop, this.view.insetBottom);
      cam.setZoom(zoom);
      if (pan) cam.pan(x, y, GLIDE_MS, 'Sine.easeInOut', true);
      else cam.centerOn(x, y);
   }
}

/** Starts the stage in a host element. */
export function createStage(parent: HTMLElement, view: StageView): { game: Phaser.Game; scene: StageScene } {
   const scene = new StageScene(view);
   const game = new Phaser.Game({
      type: Phaser.AUTO,
      parent,
      pixelArt: true,
      roundPixels: true,
      backgroundColor: '#0c0810',
      banner: false,
      scale: { mode: Phaser.Scale.RESIZE, width: parent.clientWidth || 1, height: parent.clientHeight || 1 },
      scene,
   });
   return { game, scene };
}
