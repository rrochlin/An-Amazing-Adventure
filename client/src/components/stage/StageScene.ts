import Phaser from 'phaser';
import type { RoomView } from '@/types/types';
import { DUNGEON_SHEET, TILE, TOWN_SHEET, type MapArt } from '../map/mapArt';
import { placeTokens, type Token } from '../map/WorldMap';
import {
   DUNGEON_GID,
   STEPS,
   canStep,
   findPath,
   roomAt,
   sceneLayers,
   stageCamera,
   standPoint,
   type Direction,
   type Tile,
} from './stageModel';

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
   /** The player can't walk: the story is talking, a choice is open. */
   controlsLocked?: boolean;
   /** The player walked into another room. */
   onWalkToRoom?: (roomId: string) => void;
}

/** How long a camera pan or a character's walk takes. */
const GLIDE_MS = 800;
/** One step of the player's walk. */
const STEP_MS = 170;
/** How long the screen trusts a walk into another room before the server agrees. */
const AGREE_MS = 3000;
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
   /** Where the player stands; theirs to move while they walk. */
   private playerTile?: Tile;
   /** The room the player walked into, until the server agrees. */
   private walkedInto?: string;
   private walkedAt = 0;
   private walking = false;
   private path: Tile[] = [];
   private keys: Partial<Record<Direction, Phaser.Input.Keyboard.Key[]>> = {};

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
      this.setUpControls();
      if (this.view) this.render(this.view);
   }

   private setUpControls() {
      const K = Phaser.Input.Keyboard.KeyCodes;
      const kb = this.input.keyboard;
      if (kb) {
         // No key capture: arrow keys and WASD still type into the text box.
         const add = (...codes: number[]) => codes.map((c) => kb.addKey(c, false));
         this.keys = { up: add(K.UP, K.W), down: add(K.DOWN, K.S), left: add(K.LEFT, K.A), right: add(K.RIGHT, K.D) };
      }
      // Click or tap a tile: walk there.
      this.input.on('pointerdown', (p: Phaser.Input.Pointer) => {
         // A click on the map hands the keys back from the text box (Phaser
         // stops the browser doing that itself).
         const el = document.activeElement;
         if (el instanceof HTMLElement && el !== document.body) el.blur();
         if (!this.view || this.view.controlsLocked || !this.playerTile) return;
         const to = { x: Math.floor(p.worldX / TILE), y: Math.floor(p.worldY / TILE) };
         this.path = findPath(this.view.art, this.view.rooms, this.playerTile, to, this.occupied());
      });
   }

   /** The direction key held, unless the player is typing somewhere. */
   private heldDirection(): Direction | undefined {
      const el = document.activeElement;
      if (el instanceof HTMLElement && (el.isContentEditable || ['INPUT', 'TEXTAREA', 'SELECT'].includes(el.tagName))) return undefined;
      return (Object.keys(this.keys) as Direction[]).find((d) => this.keys[d]?.some((k) => k.isDown));
   }

   update() {
      const view = this.view;
      if (!view || this.walking || !this.playerTile) return;
      if (view.controlsLocked) {
         this.path = [];
         return;
      }
      const held = this.heldDirection();
      let next: Tile | undefined;
      if (held) {
         this.path = [];
         next = { x: this.playerTile.x + STEPS[held].x, y: this.playerTile.y + STEPS[held].y };
      } else {
         next = this.path.shift();
      }
      if (!next) return;
      if (!canStep(view.art, view.rooms, this.playerTile, next, this.occupied())) {
         this.path = [];
         if (held) this.face(held);
         return;
      }
      this.step(next);
   }

   /** Tiles other characters stand on. */
   private occupied(): Set<string> {
      return new Set(this.placed.filter((t) => !t.isPlayer).map((t) => `${t.x},${t.y}`));
   }

   private face(d: Direction) {
      const sprite = this.sprites.get('player');
      if (sprite && (d === 'left' || d === 'right')) sprite.setFlipX(d === 'left');
   }

   /** One step, with a little hop; crossing into another room tells the server. */
   private step(next: Tile) {
      const view = this.view!;
      const sprite = this.sprites.get('player');
      const from = this.playerTile!;
      this.face(next.x < from.x ? 'left' : next.x > from.x ? 'right' : next.y < from.y ? 'up' : 'down');
      this.playerTile = next;
      const at = standPoint(next.x, next.y);
      this.walking = true;
      const arrived = () => {
         this.walking = false;
         // The view as it is now: the server may have answered mid-step.
         const now = this.view ?? view;
         const room = roomAt(now.art, next);
         if (room && room !== this.roomShown(now)) {
            this.walkedInto = room;
            this.walkedAt = Date.now();
            now.onWalkToRoom?.(room);
            this.render(now);
         }
      };
      if (!sprite) return arrived();
      const scale = sprite.scaleY;
      this.tweens.add({ targets: sprite, x: at.x, y: at.y, duration: STEP_MS, ease: 'Linear', onComplete: arrived });
      this.tweens.add({ targets: sprite, scaleY: scale * 1.08, duration: STEP_MS / 2, yoyo: true, ease: 'Sine.easeOut' });
   }

   /** State for browser checks. */
   probe() {
      const p = this.sprites.get('player');
      return { tile: this.playerTile, room: this.roomShown(this.view!), walking: this.walking, sprite: p && { x: p.x, y: p.y } };
   }

   /** The room on screen: the one the player walked into, until the server agrees. */
   private roomShown(view: StageView): string | undefined {
      return this.walkedInto ?? view.currentRoomId;
   }

   /** Called by React on every change. */
   setView(view: StageView) {
      this.view = view;
      if (this.ready) this.render(view);
   }

   private render(view: StageView) {
      if (view.art !== this.art) this.build(view.art);
      // The server agreed with the walk; or it never did (a refused move), and
      // its room stands.
      if (this.walkedInto && (view.currentRoomId === this.walkedInto || Date.now() - this.walkedAt > AGREE_MS)) {
         this.walkedInto = undefined;
      }
      const room = this.roomShown(view);
      this.syncPlaces(view, room);
      this.syncCharacters(view, room);
      this.drawShade(view, room);
      // Pan only when the player has moved to another room.
      this.frame(this.framedRoom !== undefined && this.framedRoom !== room);
      this.framedRoom = room;
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
      this.playerTile = undefined;
      this.walkedInto = undefined;
      this.path = [];
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
   private drawShade(view: StageView, room: string | undefined) {
      this.shade ??= this.add.graphics().setDepth(5);
      this.shade.clear();
      const rect = room ? view.art.rooms[room] : undefined;
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
   private syncPlaces(view: StageView, roomId: string | undefined) {
      this.places.forEach((p) => p.destroy());
      this.places = [];
      const room = roomId ? view.rooms[roomId] : undefined;
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
   private syncCharacters(view: StageView, room: string | undefined) {
      this.placed = placeTokens(view.art, view.rooms, view.currentRoomId, view.playerName ?? 'You', view.playerScale ?? 1, this.placed);
      const here = this.placed.filter((t) => t.isPlayer || t.roomId === room);
      const seen = new Set<string>();
      for (const t of here) {
         seen.add(t.id);
         if (t.isPlayer) {
            // The player stays where they walked while they're in the room on
            // screen; the story moving them elsewhere puts them in their new
            // room's place.
            if (this.playerTile && roomAt(view.art, this.playerTile) === room) {
               t.x = this.playerTile.x;
               t.y = this.playerTile.y;
            } else if (!this.walking) {
               this.playerTile = { x: t.x, y: t.y };
            }
         }
         if (t.isPlayer && this.walking) {
            // Mid-step: the walk owns the sprite.
            this.sprites.get(t.id)?.setFrame(t.sprite);
            continue;
         }
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
      const { zoom, x, y } = stageCamera(this.view.art, this.roomShown(this.view), { w: this.scale.width, h: this.scale.height }, this.view.insetTop, this.view.insetBottom);
      cam.setZoom(zoom);
      // A pan under way finishes where it was going (the server agreeing
      // mid-pan re-frames the same room).
      if (!pan && this.tweens.isTweening(cam)) return;
      this.tweens.killTweensOf(cam);
      if (!pan) {
         cam.centerOn(x, y);
         return;
      }
      // centerOn's scroll, tweened.
      this.tweens.add({
         targets: cam,
         scrollX: x - cam.width * cam.originX,
         scrollY: y - cam.height * cam.originY,
         duration: GLIDE_MS,
         ease: 'Sine.easeInOut',
      });
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
   // For browser checks of a dev build; stripped from production.
   if (import.meta.env.DEV) (window as unknown as { __stage?: StageScene }).__stage = scene;
   return { game, scene };
}

/** What a browser check reads from the stage (dev builds only). */
export function stageProbe(scene: StageScene) {
   return scene.probe();
}
