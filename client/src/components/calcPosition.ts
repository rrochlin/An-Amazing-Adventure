import { type Area } from "../types/types";

// Force-directed graph layout algorithm
export const calculateRoomPositions = (rooms: { [key: string]: Area }) => {
  const positions: { [key: string]: { x: number; y: number } } = {};
  const width = 1000;
  const height = 1000;
  const centerX = 200;
  const centerY = 200;
  const maxRadius = Math.min(width, height) * 0.35; // Slightly reduced from 0.4

  // Initialize positions in a circle
  const roomIds = Object.keys(rooms);
  roomIds.forEach((roomId, index) => {
    const angle = (index / roomIds.length) * 2 * Math.PI;
    positions[roomId] = {
      x: centerX + Math.cos(angle) * maxRadius,
      y: centerY + Math.sin(angle) * maxRadius,
    };
  });

  // Force-directed layout parameters
  const iterations = 150; // Increased from 100
  const repulsion = 150; // Increased from 100
  const attraction = 0.08; // Reduced from 0.1
  const damping = 0.8;
  const targetDistance = 150; // Increased from 100

  // Run force-directed layout
  for (let i = 0; i < iterations; i++) {
    const forces: { [key: string]: { x: number; y: number } } = {};
    roomIds.forEach((roomId) => {
      forces[roomId] = { x: 0, y: 0 };
    });

    // Calculate repulsion forces between all nodes
    for (let j = 0; j < roomIds.length; j++) {
      for (let k = j + 1; k < roomIds.length; k++) {
        const room1 = roomIds[j];
        const room2 = roomIds[k];
        const pos1 = positions[room1];
        const pos2 = positions[room2];

        const dx = pos2.x - pos1.x;
        const dy = pos2.y - pos1.y;
        const distance = Math.sqrt(dx * dx + dy * dy);

        if (distance > 0) {
          const force = repulsion / (distance * distance);
          const fx = (dx / distance) * force;
          const fy = (dy / distance) * force;

          forces[room1].x -= fx;
          forces[room1].y -= fy;
          forces[room2].x += fx;
          forces[room2].y += fy;
        }
      }
    }

    // Calculate attraction forces between connected nodes
    roomIds.forEach((roomId) => {
      const connections = rooms[roomId].connections;
      connections.forEach((conn) => {
        const pos1 = positions[roomId];
        const pos2 = positions[conn.id];
        const dx = pos2.x - pos1.x;
        const dy = pos2.y - pos1.y;
        const distance = Math.sqrt(dx * dx + dy * dy);

        if (distance > 0) {
          const force = (distance - targetDistance) * attraction;
          const fx = (dx / distance) * force;
          const fy = (dy / distance) * force;

          forces[roomId].x += fx;
          forces[roomId].y += fy;
          forces[conn.id].x -= fx;
          forces[conn.id].y -= fy;
        }
      });
    });

    // Apply forces with damping
    roomIds.forEach((roomId) => {
      positions[roomId].x += forces[roomId].x * damping;
      positions[roomId].y += forces[roomId].y * damping;

      // Keep nodes within bounds with more padding
      positions[roomId].x = Math.max(
        70,
        Math.min(width - 70, positions[roomId].x),
      );
      positions[roomId].y = Math.max(
        70,
        Math.min(height - 70, positions[roomId].y),
      );
    });
  }

  return positions;
};
