// Spike #118 — throwaway map prototypes. Not part of the app.
//
// Briarwell's real rooms, connections and cast come from
// server/campaigns/briarwell/campaign.json. Everything marked GAP is data the
// prototypes need that the game does not have today (see docs/map/map-prototypes.md).

export type RoomId =
   | 'family_cottage'
   | 'village_square'
   | 'market_lane'
   | 'old_mill'
   | 'meadow'
   | 'woods_edge';

export interface Poi {
   id: string;
   name: string;
   /** GAP: points of interest inside a room don't exist in campaign data. */
   note: string;
}

export interface Room {
   id: RoomId;
   name: string;
   description: string;
   type: string;
   connections: Partial<Record<'north' | 'south' | 'east' | 'west', RoomId>>;
   /** Grid position from the server's exit-based BFS layout (east = +x, north = +y). */
   grid: { x: number; y: number };
   pois: Poi[];
}

// Rooms, descriptions, types and connections: verbatim from campaign.json.
export const ROOMS: Record<RoomId, Room> = {
   family_cottage: {
      id: 'family_cottage',
      name: 'Family Cottage',
      description: 'A low thatched cottage that smells of bread and woodsmoke.',
      type: 'home',
      connections: { east: 'village_square' },
      grid: { x: -1, y: 0 },
      pois: [
         { id: 'hearth', name: 'Hearth', note: 'Where Mother cooks' },
         { id: 'cottage_door', name: 'Front door', note: 'Opens east onto the square' },
         { id: 'garden_gate', name: 'Garden gate', note: 'The back way out' },
      ],
   },
   village_square: {
      id: 'village_square',
      name: 'Village Square',
      description: 'A stone well, a few market-day benches, and gossiping neighbours.',
      type: 'hub',
      connections: { west: 'family_cottage', north: 'market_lane', east: 'old_mill', south: 'meadow' },
      grid: { x: 0, y: 0 },
      pois: [
         { id: 'well', name: 'Stone well', note: 'Bess sits on its edge' },
         { id: 'benches', name: 'Market benches', note: 'Neighbours gossip here' },
      ],
   },
   market_lane: {
      id: 'market_lane',
      name: 'Market Lane',
      description: "Old Garrick's stall of sweets, ribbons, and whittled trinkets.",
      type: 'shop',
      connections: { south: 'village_square' },
      grid: { x: 0, y: 1 },
      pois: [{ id: 'garrick_stall', name: "Garrick's stall", note: 'Sweets, ribbons, carvings' }],
   },
   old_mill: {
      id: 'old_mill',
      name: 'Old Mill',
      description: "The miller's water wheel, and a barn behind it where people meet who should not.",
      type: 'landmark',
      connections: { west: 'village_square' },
      grid: { x: 1, y: 0 },
      pois: [
         { id: 'water_wheel', name: 'Water wheel', note: 'Turning on the millrace' },
         { id: 'barn', name: 'Barn', note: 'Hob slips behind it' },
      ],
   },
   meadow: {
      id: 'meadow',
      name: 'Meadow',
      description: 'Long grass where the village children play, and where Dunstan does his bullying.',
      type: 'outdoors',
      connections: { north: 'village_square', south: 'woods_edge' },
      grid: { x: 0, y: -1 },
      pois: [
         { id: 'fence_post', name: 'Old fence post', note: 'Where the children gather' },
         { id: 'long_grass', name: 'Long grass', note: 'Good cover' },
      ],
   },
   woods_edge: {
      id: 'woods_edge',
      name: "Woods' Edge",
      description: 'A hollow under an old oak at the treeline, a good place to hide.',
      type: 'outdoors',
      connections: { north: 'meadow' },
      grid: { x: 0, y: -2 },
      pois: [{ id: 'oak_hollow', name: 'Oak hollow', note: 'A hiding place' }],
   },
};

export interface Character {
   id: string;
   name: string;
   initials: string;
   /** GAP: the campaign cast has no positions; only railed actors have locations. */
   room: RoomId;
   poi?: string;
   kind: 'player' | 'family' | 'villager';
}

/** The shared prototype state: the same in all three prototypes. */
export const CHARACTERS: Character[] = [
   { id: 'player', name: 'Wren (you)', initials: 'W', room: 'village_square', poi: 'benches', kind: 'player' },
   { id: 'bess', name: 'Little Bess', initials: 'B', room: 'village_square', poi: 'well', kind: 'villager' },
   { id: 'miller', name: 'Hob the Miller', initials: 'H', room: 'old_mill', poi: 'barn', kind: 'villager' },
   { id: 'bully', name: 'Dunstan', initials: 'D', room: 'meadow', poi: 'fence_post', kind: 'villager' },
   { id: 'trader', name: 'Old Garrick', initials: 'G', room: 'market_lane', poi: 'garrick_stall', kind: 'villager' },
   { id: 'father', name: 'Tomas', initials: 'T', room: 'family_cottage', poi: 'hearth', kind: 'family' },
   { id: 'mother', name: 'Elin', initials: 'E', room: 'family_cottage', poi: 'hearth', kind: 'family' },
   { id: 'sister', name: 'Ros', initials: 'R', room: 'family_cottage', poi: 'garden_gate', kind: 'family' },
];

export const PLAYER_ROOM: RoomId = 'village_square';

/** Discovered rooms (Woods' Edge is not). */
export const DISCOVERED = new Set<RoomId>([
   'family_cottage',
   'village_square',
   'market_lane',
   'old_mill',
   'meadow',
]);

export const COLORS = {
   gold: '#C9A962',
   goldLight: '#FFD700',
   goldDark: '#8B7355',
   purple: '#6B4E9D',
   purpleLight: '#9575CD',
   purpleDark: '#3E2C5E',
   paper: '#1a0f1e',
   bg: '#0d0508',
   text: '#E8DCC4',
   textDim: '#9a8f86',
};

export const tokenColor = (c: Character) =>
   c.kind === 'player' ? COLORS.goldLight : c.kind === 'family' ? '#7fb3a6' : '#c98f6a';

export const charactersIn = (room: RoomId) => CHARACTERS.filter((c) => c.room === room);
