import Phaser from 'phaser';
import type { RoomView } from '@/types/types';
import type { PopTone, SpokenLine, StageCue } from '../dialogue/beats';
import { DUNGEON_SHEET, TILE, TOWN_SHEET, type MapArt } from '../map/mapArt';
import { placeTokens, type Token } from '../map/WorldMap';
import {
   DUNGEON_GID,
   STEPS,
   canStep,
   bubbleText,
   findPath,
   roomAt,
   sceneLayers,
   stageCamera,
   standPoint,
   type Direction,
   type Tile,
   whoSpeaks,
} from './stageModel';

/** What the stage shows: React hands this in on every change. */
export interface StageView {
   art: MapArt;
   rooms: Record<string, RoomView>;
   currentRoomId?: string;
   playerName?: string;
   playerScale?: number;
   /** The mark the server has the player standing at, if any. */
   playerMark?: string;
   /** Staging the story has reached, oldest first; each plays once. */
   cues?: StageCue[];
   /** The spoken line on screen: a bubble over the speaker. */
   speaking?: SpokenLine | null;
   /** Pixels covered at the top (the location sign) and bottom (the text panel). */
   insetTop: number;
   insetBottom: number;
   /** The player can't walk: the story is talking, a choice is open. */
   controlsLocked?: boolean;
   /** The player walked into another room. */
   onWalkToRoom?: (roomId: string) => void;
   /** The player walked up to a character to talk to them. */
   onTalkTo?: (characterId: string, name: string) => void;
}

/** How long a camera pan or a character's walk takes. */
const GLIDE_MS = 800;
/** With nothing moving for this long, the scene stops drawing (battery). */
const IDLE_MS = 1500;
/** One step of the player's walk. */
const STEP_MS = 170;
/** One step of a character's walk (a little slower than the player's). */
const NPC_STEP_MS = 220;
/** A longer walk than this glides instead (it's going somewhere far off). */
const MAX_WALK = 30;
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
   private talkKeys: Phaser.Input.Keyboard.Key[] = [];
   /** Who the player is walking up to, to talk to once there. */
   private goingToTalk?: string;
   /** Cues already played. */
   private played = new Set<string>();
   /** Marks the story has sent people to, ahead of the server's say-so. */
   private marks = new Map<string, { mark: string; room?: string }>();
   /** The story is walking the player to their mark (computed on the next sync). */
   private playerToMark = false;
   /** The story's walk for the player: plays even while the controls are locked. */
   private storyPath: Tile[] = [];
   /** The story's walk is taking the player to the room it moved them to (no room changes to report). */
   private crossing = false;
   /** Where the story walked someone up to another, until they next move room. */
   private spots = new Map<string, { x: number; y: number; room: string }>();
   /** Reactions waiting for their character to stop walking. */
   private emotes: { id: string; emote: string }[] = [];
   /** What happened to someone, to show rising over them. */
   private pops: { id: string; text: string; tone: PopTone }[] = [];
   /** The speech bubble on screen, over whoever is speaking. */
   private bubble?: { line: string; who: string; box: Phaser.GameObjects.Container; text: Phaser.GameObjects.Text; panel: Phaser.GameObjects.Graphics };

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
      this.scale.on('resize', () => {
         this.wake();
         this.frame(false);
      });
      // Anything the player does wakes a sleeping scene.
      const wake = () => this.wake();
      window.addEventListener('keydown', wake);
      this.input.on('pointermove', wake);
      this.game.canvas.addEventListener('pointerdown', wake);
      this.events.once('destroy', () => window.removeEventListener('keydown', wake));
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
         this.talkKeys = add(K.E, K.SPACE);
      }
      // Click or tap a tile: walk there.
      this.input.on('pointerdown', (p: Phaser.Input.Pointer) => {
         // A click on the map hands the keys back from the text box (Phaser
         // stops the browser doing that itself).
         const el = document.activeElement;
         if (el instanceof HTMLElement && el !== document.body) el.blur();
         if (!this.view || this.view.controlsLocked || !this.playerTile) return;
         const to = { x: Math.floor(p.worldX / TILE), y: Math.floor(p.worldY / TILE) };
         // Clicking someone walks up to them and talks.
         const who = this.placed.find((t) => !t.isPlayer && t.roomId === this.roomShown(this.view!) && t.x === to.x && t.y === to.y);
         if (who) {
            this.walkUpTo(who);
            return;
         }
         this.goingToTalk = undefined;
         this.path = findPath(this.view.art, this.view.rooms, this.playerTile, to, this.occupied());
      });
   }

   /** Walk to a free tile next to someone, then talk to them. */
   private walkUpTo(who: Token) {
      const view = this.view!;
      if (this.nextTo(this.playerTile!, who)) {
         this.talkTo(who);
         return;
      }
      const occupied = this.occupied();
      let best: Tile[] | undefined;
      for (const d of Object.values(STEPS)) {
         const spot = { x: who.x + d.x, y: who.y + d.y };
         const path = findPath(view.art, view.rooms, this.playerTile!, spot, occupied);
         if (path.length && (!best || path.length < best.length)) best = path;
      }
      if (!best) return;
      this.path = best;
      this.goingToTalk = who.id;
   }

   private nextTo(a: Tile, b: Tile): boolean {
      return Math.abs(a.x - b.x) + Math.abs(a.y - b.y) === 1;
   }

   private talkTo(who: Token) {
      this.goingToTalk = undefined;
      this.path = [];
      this.face(who.x < this.playerTile!.x ? 'left' : 'right');
      this.view?.onTalkTo?.(who.id, who.name);
   }

   /** Whether the player is typing into the page (keys are theirs, not the scene's). */
   private typingElsewhere(): boolean {
      const el = document.activeElement;
      return el instanceof HTMLElement && (el.isContentEditable || ['INPUT', 'TEXTAREA', 'SELECT'].includes(el.tagName));
   }

   /** The direction key held, unless the player is typing somewhere. */
   private heldDirection(): Direction | undefined {
      if (this.typingElsewhere()) return undefined;
      return (Object.keys(this.keys) as Direction[]).find((d) => this.keys[d]?.some((k) => k.isDown));
   }

   update(time: number) {
      this.sleepWhenIdle(time);
      this.placeBubble();
      const view = this.view;
      if (!view || this.walking || !this.playerTile) return;
      // The story walks the player first, whatever the controls.
      if (this.storyPath.length) {
         const next = this.storyPath.shift()!;
         if (canStep(view.art, view.rooms, this.playerTile, next, this.occupied())) {
            this.step(next);
         } else {
            this.storyPath = [];
            if (this.crossing) {
               // Blocked on the way: they're put where the story has them.
               this.crossing = false;
               this.playerTile = undefined;
               this.render(view);
            }
         }
         return;
      }
      if (view.controlsLocked) {
         this.path = [];
         this.goingToTalk = undefined;
         return;
      }
      // Arrived next to someone the player meant to talk to.
      if (this.goingToTalk && !this.path.length) {
         const who = this.placed.find((t) => t.id === this.goingToTalk);
         if (who && this.nextTo(this.playerTile, who)) this.talkTo(who);
         else this.goingToTalk = undefined;
         return;
      }
      const held = this.heldDirection();
      // E or Space next to someone talks to them.
      // (Not while typing, or while a button has focus: Space presses it.)
      const buttonFocused = document.activeElement instanceof HTMLButtonElement;
      if (!held && !this.typingElsewhere() && !buttonFocused && this.talkKeys.some((k) => Phaser.Input.Keyboard.JustDown(k))) {
         const who = this.placed.find((t) => !t.isPlayer && t.roomId === this.roomShown(view) && this.nextTo(this.playerTile!, t));
         if (who) this.talkTo(who);
         return;
      }
      let next: Tile | undefined;
      if (held) {
         this.path = [];
         this.goingToTalk = undefined;
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
      sprite?.setData('tile', next);
      const at = standPoint(next.x, next.y);
      this.walking = true;
      const arrived = () => {
         this.walking = false;
         if (!this.storyPath.length) this.playEmotes();
         // The view as it is now: the server may have answered mid-step.
         const now = this.view ?? view;
         const room = roomAt(now.art, next);
         if (this.crossing) {
            // The story's walk: the server already has them where it's going.
            if (!this.storyPath.length) this.crossing = false;
            return;
         }
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

   private activeAt = 0;

   /**
    * Stop the game loop once nothing has moved for a while: no tweens, no
    * walk, no keys held. The canvas keeps its last frame; any change from
    * React, a key, the pointer or a resize starts it again.
    */
   private sleepWhenIdle(time: number) {
      const busy =
         this.walking ||
         this.path.length > 0 ||
         this.storyPath.length > 0 ||
         this.emotes.length > 0 ||
         this.tweens.getTweens().length > 0 ||
         this.heldDirection() !== undefined;
      if (busy || !this.activeAt) this.activeAt = time;
      if (time - this.activeAt > IDLE_MS) {
         this.activeAt = 0;
         this.game.loop.sleep();
      }
   }

   private wake() {
      if (this.ready && !this.game.loop.running) {
         this.activeAt = 0;
         this.game.loop.wake();
      }
   }

   /** State for browser checks. */
   probe() {
      const p = this.sprites.get('player');
      const tiles = Object.fromEntries([...this.sprites].map(([id, s]) => [id, s.getData('tile') as Tile | undefined]));
      return { asleep: !this.game.loop.running, tile: this.playerTile, room: this.roomShown(this.view!), walking: this.walking, sprite: p && { x: p.x, y: p.y }, tiles, played: this.played.size };
   }

   /** The room on screen: the one the player walked into, until the server agrees. */
   private roomShown(view: StageView): string | undefined {
      return this.walkedInto ?? view.currentRoomId;
   }

   /** Called by React on every change. */
   setView(view: StageView) {
      this.view = view;
      this.wake();
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
      this.takeCues(view, room);
      this.syncPlaces(view, room);
      this.syncCharacters(view, room);
      this.playEmotes();
      this.syncBubble(view, room);
      this.playPops();
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
      this.bubble?.box.destroy();
      this.bubble = undefined;
      this.placed = [];
      this.framedRoom = undefined;
      this.playerTile = undefined;
      this.walkedInto = undefined;
      this.path = [];
      this.storyPath = [];
      this.crossing = false;
      this.marks.clear();
      this.spots.clear();
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
      const before = this.placed;
      this.placed = placeTokens(view.art, view.rooms, view.currentRoomId, view.playerName ?? 'You', view.playerScale ?? 1, this.placed, this.markRecord(view));
      // Someone the story walked up to another stands there while they stay
      // in that room and the tile is free.
      for (const t of this.placed) {
         const spot = this.spots.get(t.id);
         if (!spot) continue;
         if (spot.room !== t.roomId) {
            this.spots.delete(t.id);
            continue;
         }
         if (!this.placed.some((o) => o !== t && o.roomId === t.roomId && o.x === spot.x && o.y === spot.y)) {
            t.x = spot.x;
            t.y = spot.y;
         }
      }
      const here = this.placed.filter((t) => t.isPlayer || t.roomId === room);
      const seen = new Set<string>();
      for (const t of here) {
         seen.add(t.id);
         if (t.isPlayer) {
            // The player stays where they walked while they're in the room on
            // screen; the story moving them elsewhere puts them in their new
            // room's place.
            if (this.playerToMark && this.playerTile && roomAt(view.art, this.playerTile) === room) {
               // The story sends the player to their mark: they walk there.
               this.playerToMark = false;
               const others = new Set(this.placed.filter((o) => !o.isPlayer).map((o) => `${o.x},${o.y}`));
               this.storyPath = findPath(view.art, view.rooms, this.playerTile, { x: t.x, y: t.y }, others);
               this.path = [];
               this.goingToTalk = undefined;
            }
            if (this.playerTile && (this.crossing || roomAt(view.art, this.playerTile) === room)) {
               t.x = this.playerTile.x;
               t.y = this.playerTile.y;
            } else if (!this.walking) {
               // The story moved the player to another room: they walk there
               // when it's a short way, and are put there otherwise.
               const others = new Set(this.placed.filter((o) => !o.isPlayer).map((o) => `${o.x},${o.y}`));
               const path = this.playerTile && this.sprites.has('player') ? findPath(view.art, view.rooms, this.playerTile, { x: t.x, y: t.y }, others) : [];
               if (path.length && path.length <= MAX_WALK) {
                  this.storyPath = path;
                  this.crossing = true;
                  this.path = [];
                  this.goingToTalk = undefined;
                  t.x = this.playerTile!.x;
                  t.y = this.playerTile!.y;
               } else {
                  this.playerTile = { x: t.x, y: t.y };
               }
            }
         }
         if (t.isPlayer && this.walking) {
            // Mid-step: the walk owns the sprite.
            this.sprites.get(t.id)?.setFrame(t.sprite);
            continue;
         }
         const at = standPoint(t.x, t.y);
         let sprite = this.sprites.get(t.id);
         const was = sprite?.getData('tile') as Tile | undefined;
         if (!sprite) {
            // Someone coming in from next door walks in from where they were.
            const from = before.find((b) => b.id === t.id && !t.isPlayer && b.roomId !== t.roomId && b.roomId !== undefined);
            const start = from && this.framedRoom === room ? standPoint(from.x, from.y) : at;
            sprite = this.add.sprite(start.x, start.y, 'dungeon', t.sprite).setOrigin(0.5, 1).setDepth(10);
            sprite.setInteractive();
            this.nameOnHover(sprite, t.name);
            if (t.isPlayer) this.glow(sprite);
            this.sprites.set(t.id, sprite);
            if (start !== at) this.moveSprite(sprite, { x: from!.x, y: from!.y }, t, view);
         } else if (t.isPlayer ? sprite.x !== at.x || sprite.y !== at.y : !was || was.x !== t.x || was.y !== t.y) {
            this.moveSprite(sprite, was, t, view);
         }
         sprite.setData('tile', { x: t.x, y: t.y });
         sprite.setFrame(t.sprite).setScale(t.scale || 1);
         sprite.setData('name', t.name);
      }
      for (const [id, sprite] of this.sprites) {
         if (seen.has(id)) continue;
         this.sprites.delete(id);
         // Someone leaving for next door walks out that way, then is gone.
         const was = before.find((b) => b.id === id);
         const now = this.placed.find((p) => p.id === id);
         if (was && now && was.roomId !== now.roomId && was.roomId === room) {
            this.walkOff(sprite, { x: was.x, y: was.y }, now, view);
         } else {
            sprite.destroy();
         }
      }
   }

   /**
    * A character moves to their new tile: they walk there step by step when
    * there's a way within reach, and glide otherwise (the player, put
    * somewhere by the story, glides too).
    */
   private moveSprite(sprite: Phaser.GameObjects.Sprite, from: Tile | undefined, to: Token, view: StageView) {
      const at = standPoint(to.x, to.y);
      const path = from && !to.isPlayer ? findPath(view.art, view.rooms, from, { x: to.x, y: to.y }, new Set()) : [];
      this.tweens.killTweensOf(sprite);
      sprite.setScale(to.scale || 1).setAngle(0);
      if (!path.length || path.length > MAX_WALK) {
         this.tweens.add({ targets: sprite, x: at.x, y: at.y, duration: GLIDE_MS, ease: 'Sine.easeInOut', onComplete: () => this.playEmotes() });
         return;
      }
      sprite.setData('walking', true);
      let prev = from!;
      const steps = path.map((p) => {
         const stepFrom = prev;
         prev = p;
         const point = standPoint(p.x, p.y);
         return {
            x: point.x,
            y: point.y,
            duration: NPC_STEP_MS,
            ease: 'Linear',
            onStart: () => {
               if (p.x !== stepFrom.x) sprite.setFlipX(p.x < stepFrom.x);
            },
         };
      });
      this.tweens.chain({
         targets: sprite,
         tweens: steps,
         onComplete: () => {
            sprite.setData('walking', false);
            this.playEmotes();
         },
      });
   }

   /** Walk out toward someone's new room and fade at the edge of this one. */
   private walkOff(sprite: Phaser.GameObjects.Sprite, from: Tile, to: Token, view: StageView) {
      const full = findPath(view.art, view.rooms, from, { x: to.x, y: to.y }, new Set());
      // Only as far as the first tile outside this room.
      const out = full.findIndex((t) => roomAt(view.art, t) !== roomAt(view.art, from));
      const path = out === -1 ? [] : full.slice(0, out + 1);
      sprite.disableInteractive();
      if (!path.length || path.length > MAX_WALK) {
         this.tweens.add({ targets: sprite, alpha: 0, duration: 400, onComplete: () => sprite.destroy() });
         return;
      }
      this.tweens.killTweensOf(sprite);
      let prev = from;
      this.tweens.chain({
         targets: sprite,
         tweens: [
            ...path.map((p) => {
               const stepFrom = prev;
               prev = p;
               const at = standPoint(p.x, p.y);
               return { x: at.x, y: at.y, duration: NPC_STEP_MS, ease: 'Linear', onStart: () => p.x !== stepFrom.x && sprite.setFlipX(p.x < stepFrom.x) };
            }),
            { alpha: 0, duration: 250 },
         ],
         onComplete: () => sprite.destroy(),
      });
   }

   /** New cues: marks to stand at (walked on the next sync) and reactions to play. */
   private takeCues(view: StageView, room: string | undefined) {
      for (const cue of view.cues ?? []) {
         if (this.played.has(cue.id)) continue;
         this.played.add(cue.id);
         if (cue.mark) {
            this.spots.delete(cue.character);
            this.marks.set(cue.character, { mark: cue.mark, room: cue.character === 'player' ? room : this.roomOf(view, cue.character) });
            if (cue.character === 'player') this.playerToMark = true;
         }
         if (cue.toward) this.walkToward(view, cue.character, cue.toward);
         if (cue.emote) this.emotes.push({ id: cue.character, emote: cue.emote });
         if (cue.pop) this.pops.push({ id: cue.character, text: cue.pop, tone: cue.tone ?? 'gain' });
      }
   }

   /**
    * Someone walks up beside another: to the free tile next to them that's
    * the shortest walk. The player walks there as a story walk; anyone else
    * stands there until they next move.
    */
   private walkToward(view: StageView, who: string, toward: string) {
      const target = this.placed.find((t) => t.id === toward);
      const mover = who === 'player' ? (this.playerTile && { x: this.playerTile.x, y: this.playerTile.y }) : this.placed.find((t) => t.id === who);
      if (!target || !mover) return;
      if (Math.abs(mover.x - target.x) + Math.abs(mover.y - target.y) === 1) return;
      const occupied = new Set(this.placed.filter((t) => t.id !== who && t.roomId === target.roomId).map((t) => `${t.x},${t.y}`));
      if (who !== 'player' && this.playerTile) occupied.add(`${this.playerTile.x},${this.playerTile.y}`);
      let best: Tile[] | undefined;
      for (const d of Object.values(STEPS)) {
         const spot = { x: target.x + d.x, y: target.y + d.y };
         if (occupied.has(`${spot.x},${spot.y}`)) continue;
         const path = findPath(view.art, view.rooms, mover, spot, occupied);
         if (path.length && (!best || path.length < best.length)) best = path;
      }
      if (!best) return;
      if (who === 'player') {
         this.storyPath = best;
         this.path = [];
         this.goingToTalk = undefined;
      } else {
         const end = best[best.length - 1];
         this.spots.set(who, { x: end.x, y: end.y, room: target.roomId });
      }
   }

   private roomOf(view: StageView, id: string): string | undefined {
      return Object.values(view.rooms).find((r) => r.occupants?.some((c) => c.id === id))?.id;
   }

   /**
    * Who stands at which mark: the story's cues, while the person is still in
    * the room the cue sent them across; otherwise the server's marks (which
    * placeTokens reads from the room itself).
    */
   private markRecord(view: StageView): Record<string, string> {
      const out: Record<string, string> = {};
      if (view.playerMark) out.player = view.playerMark;
      for (const [id, m] of this.marks) {
         const now = id === 'player' ? this.roomShown(view) : this.roomOf(view, id);
         if (now && now === m.room) out[id] = m.mark;
         else this.marks.delete(id);
      }
      return out;
   }

   /** Reactions whose character is standing still (on screen, not walking). */
   private playEmotes() {
      const waiting: typeof this.emotes = [];
      for (const e of this.emotes) {
         const sprite = this.sprites.get(e.id);
         const busy = e.id === 'player' ? this.walking || this.storyPath.length > 0 : sprite?.getData('walking');
         if (!sprite) continue; // not on screen: nothing to see
         if (busy || this.tweens.isTweening(sprite)) {
            waiting.push(e);
            continue;
         }
         this.emote(sprite, e.emote);
      }
      this.emotes = waiting;
   }

   /** A reaction, played on the art we have: a hop, a shake, a bow. */
   private emote(sprite: Phaser.GameObjects.Sprite, emote: string) {
      const y = sprite.y;
      const x = sprite.x;
      const sy = sprite.scaleY;
      const done = () => sprite.setPosition(x, y).setScale(sprite.scaleX, sy).clearTint();
      switch (emote) {
         case 'hop':
            this.tweens.add({ targets: sprite, y: y - 5, duration: 130, yoyo: true, repeat: 1, ease: 'Sine.easeOut', onComplete: done });
            break;
         case 'laugh':
            this.tweens.add({ targets: sprite, y: y - 2, duration: 90, yoyo: true, repeat: 3, ease: 'Sine.easeInOut', onComplete: done });
            break;
         case 'shake':
            this.tweens.add({ targets: sprite, x: x + 1.5, duration: 50, yoyo: true, repeat: 5, ease: 'Linear', onComplete: done });
            break;
         case 'startle':
            this.bang(sprite);
            this.tweens.add({ targets: sprite, y: y - 6, duration: 90, yoyo: true, ease: 'Quad.easeOut', onComplete: done });
            break;
         case 'bow':
            this.tweens.add({ targets: sprite, scaleY: sy * 0.8, duration: 260, yoyo: true, hold: 200, ease: 'Sine.easeInOut', onComplete: done });
            break;
         case 'cry':
            sprite.setTint(0x9fb8ff);
            this.tweens.add({ targets: sprite, scaleY: sy * 0.92, duration: 400, yoyo: true, hold: 600, ease: 'Sine.easeInOut', onComplete: done });
            break;
      }
   }

   /**
    * The spoken line in a bubble over the speaker, while they're on screen:
    * short lines whole, a long one as "…" (the panel below has it all).
    */
   private syncBubble(view: StageView, room: string | undefined) {
      const line = view.speaking;
      const people = this.placed.filter((t) => !t.isPlayer && t.roomId === room).map((t) => {
         const c = view.rooms[t.roomId]?.occupants?.find((o) => o.id === t.id);
         return { id: t.id, name: t.name, called: c?.called };
      });
      const who = line ? whoSpeaks(line.speaker, people, view.playerName) : undefined;
      if (!line || !who || !this.sprites.has(who)) {
         this.bubble?.box.destroy();
         this.bubble = undefined;
         return;
      }
      const words = bubbleText(line.text);
      if (this.bubble && this.bubble.line === line.id && this.bubble.who === who) {
         if (this.bubble.text.text !== words) this.drawBubble(words);
         return;
      }
      this.bubble?.box.destroy();
      const text = this.add.text(0, 0, '', {
         fontFamily: '"Crimson Text", Georgia, serif',
         fontSize: '13px',
         color: '#2a1f14',
         // Narrow, so it covers as few of the people behind as it can.
         wordWrap: { width: 140, useAdvancedWrap: true },
         align: 'center',
      });
      text.setOrigin(0.5, 1);
      const panel = this.add.graphics();
      const box = this.add.container(0, 0, [panel, text]).setDepth(102);
      this.bubble = { line: line.id, who, box, text, panel };
      this.drawBubble(words);
      this.placeBubble();
      // A small pop as it appears.
      const s = 1 / this.cameras.main.zoom;
      box.setScale(s * 0.6).setAlpha(0);
      this.tweens.add({ targets: box, scale: s, alpha: 1, duration: 140, ease: 'Back.easeOut' });
   }

   /** The bubble's panel and tail, drawn around its text (in screen pixels). */
   private drawBubble(words: string) {
      if (!this.bubble) return;
      const { text, panel } = this.bubble;
      text.setText(words);
      const pad = 5;
      const w = Math.max(text.width + pad * 2, 28);
      const h = text.height + pad * 2;
      const tail = 7;
      text.setPosition(0, -tail - pad);
      panel.clear();
      panel.fillStyle(0xf3e9d2, 0.93);
      panel.lineStyle(2, 0x2a1f14, 1);
      panel.fillRoundedRect(-w / 2, -tail - h, w, h, 6);
      panel.strokeRoundedRect(-w / 2, -tail - h, w, h, 6);
      panel.fillTriangle(-5, -tail - 1, 5, -tail - 1, 0, 0);
      panel.lineBetween(-5, -tail, 0, 0);
      panel.lineBetween(5, -tail, 0, 0);
   }

   /** Keep the bubble over its speaker's head, at a constant size on screen. */
   private placeBubble() {
      if (!this.bubble) return;
      const sprite = this.sprites.get(this.bubble.who);
      if (!sprite) return;
      const zoom = this.cameras.main.zoom;
      this.bubble.box.setPosition(sprite.x, sprite.y - sprite.displayHeight - 1);
      if (!this.tweens.isTweening(this.bubble.box)) this.bubble.box.setScale(1 / zoom);
   }

   /**
    * What happened to someone rises over their head and fades: damage in red
    * (with a flash and a shake), healing in green, items gained in gold and
    * lost in grey. Several at once stack upward.
    */
   private playPops() {
      const byWho = new Map<string, number>();
      for (const p of this.pops) {
         const sprite = this.sprites.get(p.id);
         if (!sprite) continue; // not on screen: the side panel still has it
         const stack = byWho.get(p.id) ?? 0;
         byWho.set(p.id, stack + 1);
         const zoom = this.cameras.main.zoom;
         const color = { gain: '#f5d77a', loss: '#c9c2b8', harm: '#ff6b5b', heal: '#7ee08a' }[p.tone];
         const text = this.add
            .text(sprite.x, sprite.y - sprite.displayHeight - 2 - (stack * 17) / zoom, p.text, {
               fontFamily: 'Cinzel, Georgia, serif',
               fontSize: '15px',
               fontStyle: 'bold',
               color,
               stroke: '#0c0810',
               strokeThickness: 3,
            })
            .setOrigin(0.5, 1)
            .setScale(1 / zoom)
            .setDepth(103);
         this.tweens.add({ targets: text, y: text.y - 14 / zoom, alpha: 0, delay: 1800 + stack * 200, duration: 900, ease: 'Sine.easeIn', onComplete: () => text.destroy() });
         if (p.tone === 'harm' && stack === 0) {
            sprite.setTint(0xff7a6a);
            this.time.delayedCall(250, () => sprite.clearTint());
            this.emote(sprite, 'shake');
         }
      }
      this.pops = [];
   }

   /** A "!" over someone's head for a moment. */
   private bang(sprite: Phaser.GameObjects.Sprite) {
      const zoom = this.cameras.main.zoom;
      const mark = this.add
         .text(sprite.x, sprite.y - sprite.displayHeight - 1, '!', { fontFamily: 'Georgia, serif', fontSize: '14px', fontStyle: 'bold', color: '#f5d77a', stroke: '#0c0810', strokeThickness: 3 })
         .setOrigin(0.5, 1)
         .setScale(1 / zoom)
         .setDepth(101);
      this.tweens.add({ targets: mark, y: mark.y - 3, alpha: 0, delay: 500, duration: 300, onComplete: () => mark.destroy() });
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
