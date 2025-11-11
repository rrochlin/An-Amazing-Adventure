import { Fragment, useEffect, useMemo, useState } from "react";
import { type GameState } from "../types/types";
import { Stage, Layer, Circle, Line, Text, Rect, Arrow } from "react-konva";
import { Box, IconButton, Typography, Chip, Stack } from "@mui/material";
import ZoomInIcon from "@mui/icons-material/ZoomIn";
import ZoomOutIcon from "@mui/icons-material/ZoomOut";
import RestartAltIcon from "@mui/icons-material/RestartAlt";

// RoomMap Component with enhanced features
export const RoomMap = ({ gameState }: { gameState: GameState }) => {
  const stageWidth = 400;
  const stageHeight = 400;
  const roomRadius = 15;
  const playerIconRadius = 6;

  const [hoveredRoom, setHoveredRoom] = useState<string | null>(null);
  const [tooltip, setTooltip] = useState<{
    x: number;
    y: number;
    text: string;
  } | null>(null);
  const [selectedLayer, setSelectedLayer] = useState(0); // Z-axis layer
  const [zoom, setZoom] = useState(1.0);

  // Get all unique Z-levels in the game
  const zLevels = useMemo(() => {
    if (!gameState.rooms) return [0];
    const levels = new Set(
      Object.values(gameState.rooms).map((room) => room.coordinates.z)
    );
    return Array.from(levels).sort((a, b) => b - a); // Top to bottom
  }, [gameState.rooms]);

  // Set initial layer to player's current layer
  useEffect(() => {
    if (gameState.current_room) {
      setSelectedLayer(gameState.current_room.coordinates.z);
    }
  }, [gameState.current_room]);

  // Convert server coordinates to canvas coordinates
  const roomPositions = useMemo(() => {
    if (!gameState.rooms) return {};

    const positions: { [key: string]: { x: number; y: number; z: number } } = {};
    const centerX = stageWidth / 2;
    const centerY = stageHeight / 2;

    Object.keys(gameState.rooms).forEach((roomId) => {
      const room = gameState.rooms![roomId];
      positions[roomId] = {
        x: centerX + room.coordinates.x * zoom,
        y: centerY + room.coordinates.y * zoom,
        z: room.coordinates.z,
      };
    });

    return positions;
  }, [gameState.rooms, zoom, stageWidth, stageHeight]);

  // Filter rooms by selected layer
  const roomsOnLayer = useMemo(() => {
    if (!gameState.rooms) return [];
    return Object.keys(gameState.rooms).filter(
      (roomId) => gameState.rooms![roomId].coordinates.z === selectedLayer
    );
  }, [gameState.rooms, selectedLayer]);

  // Track visited rooms (rooms that are connected or have been explored)
  const visitedRooms = useMemo(() => {
    const visited = new Set<string>();
    if (!gameState.rooms) return visited;

    // Add current room and all connected rooms
    visited.add(gameState.current_room.id);
    Object.values(gameState.current_room.connections).forEach((roomId) => {
      visited.add(roomId);
    });

    // Add all rooms connected to visited rooms (could track this better in game state)
    gameState.connected_rooms?.forEach((roomId) => visited.add(roomId));

    return visited;
  }, [gameState.current_room, gameState.connected_rooms, gameState.rooms]);

  const handleRoomHover = (roomId: string, pos: { x: number; y: number }) => {
    setHoveredRoom(roomId);
    const isAdjacent =
      Object.values(gameState.current_room.connections).includes(roomId) ||
      (gameState.rooms?.[roomId] &&
        Object.values(gameState.rooms[roomId].connections).includes(
          gameState.current_room.id
        ));
    const isCurrent = roomId === gameState.current_room.id;

    let tooltipText = roomId;
    if (isCurrent) {
      tooltipText = "You are here";
    } else if (isAdjacent) {
      // Find the direction
      for (const [dir, connectedId] of Object.entries(
        gameState.current_room.connections
      )) {
        if (connectedId === roomId) {
          tooltipText = `${roomId} (${dir})`;
          break;
        }
      }
    } else {
      tooltipText = visitedRooms.has(roomId) ? `${roomId} (explored)` : roomId;
    }

    setTooltip({ x: pos.x, y: pos.y - 30, text: tooltipText });
  };

  const handleRoomLeave = () => {
    setHoveredRoom(null);
    setTooltip(null);
  };

  const handleZoomIn = () => setZoom((z) => Math.min(z + 0.2, 3.0));
  const handleZoomOut = () => setZoom((z) => Math.max(z - 0.2, 0.5));
  const handleResetZoom = () => setZoom(1.0);

  return (
    <Box sx={{ display: "flex", flexDirection: "column", gap: 1, width: "100%" }}>
      {/* Controls */}
      <Stack direction="row" spacing={1} justifyContent="space-between" alignItems="center">
        {/* Layer Selector */}
        <Stack direction="row" spacing={0.5} flexWrap="wrap">
          {zLevels.map((level) => {
            const hasPlayer = gameState.current_room.coordinates.z === level;
            return (
              <Chip
                key={level}
                label={level === 0 ? "Ground" : level > 0 ? `+${level}` : level}
                size="small"
                onClick={() => setSelectedLayer(level)}
                color={selectedLayer === level ? "primary" : "default"}
                variant={selectedLayer === level ? "filled" : "outlined"}
                icon={
                  hasPlayer ? (
                    <Box
                      sx={{
                        width: 8,
                        height: 8,
                        borderRadius: "50%",
                        backgroundColor: "#FFD700",
                      }}
                    />
                  ) : undefined
                }
                sx={{
                  fontSize: "0.75rem",
                  height: "24px",
                }}
              />
            );
          })}
        </Stack>

        {/* Zoom Controls */}
        <Stack direction="row" spacing={0}>
          <IconButton size="small" onClick={handleZoomIn} sx={{ color: "#E0E0E0" }}>
            <ZoomInIcon fontSize="small" />
          </IconButton>
          <IconButton size="small" onClick={handleResetZoom} sx={{ color: "#E0E0E0" }}>
            <RestartAltIcon fontSize="small" />
          </IconButton>
          <IconButton size="small" onClick={handleZoomOut} sx={{ color: "#E0E0E0" }}>
            <ZoomOutIcon fontSize="small" />
          </IconButton>
        </Stack>
      </Stack>

      {/* Map Canvas */}
      <Box
        sx={{
          border: "1px solid #424242",
          borderRadius: "4px",
          backgroundColor: "#1E1E1E",
          overflow: "hidden",
        }}
      >
        <Stage width={stageWidth} height={stageHeight}>
          <Layer>
            {/* Draw connections on current layer */}
            {gameState.rooms &&
              roomsOnLayer.map((roomId) => {
                const room = gameState.rooms![roomId];
                return Object.entries(room.connections).map(
                  ([direction, connectedRoomId]) => {
                    // Only draw connections to rooms on the same layer
                    if (
                      !roomsOnLayer.includes(connectedRoomId) &&
                      direction !== "up" &&
                      direction !== "down"
                    )
                      return null;

                    const start = roomPositions[roomId];
                    const end = roomPositions[connectedRoomId];
                    if (!start || !end) return null;

                    // Skip vertical connections for line drawing
                    if (direction === "up" || direction === "down") return null;

                    const dx = end.x - start.x;
                    const dy = end.y - start.y;
                    const angle = Math.atan2(dy, dx);

                    const startX = start.x + Math.cos(angle) * roomRadius;
                    const startY = start.y + Math.sin(angle) * roomRadius;
                    const endX = end.x - Math.cos(angle) * roomRadius;
                    const endY = end.y - Math.sin(angle) * roomRadius;

                    const isCurrentRoomConnection =
                      roomId === gameState.current_room.id ||
                      connectedRoomId === gameState.current_room.id;

                    return (
                      <Line
                        key={`${roomId}-${direction}-${connectedRoomId}`}
                        points={[startX, startY, endX, endY]}
                        stroke={isCurrentRoomConnection ? "#4CAF50" : "#666"}
                        strokeWidth={isCurrentRoomConnection ? 3 : 2}
                        opacity={isCurrentRoomConnection ? 1 : 0.6}
                      />
                    );
                  }
                );
              })}

            {/* Draw rooms on current layer */}
            {gameState.rooms &&
              roomsOnLayer.map((roomId) => {
                const pos = roomPositions[roomId];
                const room = gameState.rooms![roomId];
                if (!pos) return null;

                const isCurrentRoom = roomId === gameState.current_room.id;
                const isConnectedToCurrent =
                  Object.values(gameState.current_room.connections).includes(
                    roomId
                  ) ||
                  Object.values(room.connections).includes(
                    gameState.current_room.id
                  );
                const isVisited = visitedRooms.has(roomId);

                // Color coding
                let fillColor = "#555"; // Unvisited
                let strokeColor = "#777";
                if (isCurrentRoom) {
                  fillColor = "#4CAF50"; // Current: Green
                  strokeColor = "#81C784";
                } else if (isConnectedToCurrent) {
                  fillColor = "#2196F3"; // Connected: Blue
                  strokeColor = "#64B5F6";
                } else if (isVisited) {
                  fillColor = "#757575"; // Visited: Gray
                  strokeColor = "#9E9E9E";
                }

                // Check for vertical connections
                const hasUpConnection = Object.keys(room.connections).includes("up");
                const hasDownConnection = Object.keys(room.connections).includes("down");

                return (
                  <Fragment key={roomId}>
                    {/* Main room circle */}
                    <Circle
                      x={pos.x}
                      y={pos.y}
                      radius={roomRadius}
                      fill={fillColor}
                      stroke={strokeColor}
                      strokeWidth={isCurrentRoom ? 3 : 2}
                      shadowColor="black"
                      shadowBlur={isCurrentRoom ? 10 : 5}
                      shadowOpacity={0.4}
                      onMouseEnter={() => handleRoomHover(roomId, pos)}
                      onMouseLeave={handleRoomLeave}
                    />

                    {/* Up arrow indicator */}
                    {hasUpConnection && (
                      <Text
                        x={pos.x - 5}
                        y={pos.y - roomRadius - 15}
                        text="↑"
                        fontSize={14}
                        fill="#FFD700"
                        fontStyle="bold"
                      />
                    )}

                    {/* Down arrow indicator */}
                    {hasDownConnection && (
                      <Text
                        x={pos.x - 5}
                        y={pos.y + roomRadius + 5}
                        text="↓"
                        fontSize={14}
                        fill="#FFD700"
                        fontStyle="bold"
                      />
                    )}

                    {/* Player icon in current room */}
                    {isCurrentRoom && (
                      <Circle
                        x={pos.x}
                        y={pos.y}
                        radius={playerIconRadius}
                        fill="#FFD700"
                        stroke="#000"
                        strokeWidth={2}
                        shadowColor="black"
                        shadowBlur={8}
                        shadowOpacity={0.6}
                      />
                    )}

                    {/* Room label */}
                    {(isCurrentRoom || isConnectedToCurrent || hoveredRoom === roomId) && (
                      <Fragment>
                        <Rect
                          x={pos.x - 60}
                          y={pos.y + roomRadius + 5}
                          width={120}
                          height={18}
                          fill={hoveredRoom === roomId ? "#424242" : "#2D2D2D"}
                          cornerRadius={3}
                          shadowColor="black"
                          shadowBlur={3}
                          shadowOpacity={0.3}
                        />
                        <Text
                          x={pos.x - 55}
                          y={pos.y + roomRadius + 8}
                          text={roomId
                            .split("_")
                            .map(
                              (word) =>
                                word.charAt(0).toUpperCase() +
                                word.slice(1).toLowerCase()
                            )
                            .join(" ")}
                          fontSize={10}
                          fill={hoveredRoom === roomId ? "#FFFFFF" : "#E0E0E0"}
                          width={110}
                          align="center"
                        />
                      </Fragment>
                    )}
                  </Fragment>
                );
              })}

            {/* Compass Rose */}
            <Fragment>
              {/* North */}
              <Arrow
                points={[stageWidth - 30, 30, stageWidth - 30, 15]}
                fill="#888"
                stroke="#888"
                strokeWidth={2}
              />
              <Text
                x={stageWidth - 35}
                y={10}
                text="N"
                fontSize={12}
                fill="#888"
                fontStyle="bold"
              />
              {/* East */}
              <Text
                x={stageWidth - 15}
                y={25}
                text="E"
                fontSize={10}
                fill="#666"
              />
              {/* South */}
              <Text
                x={stageWidth - 35}
                y={40}
                text="S"
                fontSize={10}
                fill="#666"
              />
              {/* West */}
              <Text
                x={stageWidth - 50}
                y={25}
                text="W"
                fontSize={10}
                fill="#666"
              />
            </Fragment>

            {/* Tooltip */}
            {tooltip && (
              <Fragment>
                <Rect
                  x={tooltip.x - 60}
                  y={tooltip.y - 20}
                  width={120}
                  height={25}
                  fill="#2D2D2D"
                  cornerRadius={4}
                  shadowColor="black"
                  shadowBlur={5}
                  shadowOpacity={0.5}
                />
                <Text
                  x={tooltip.x - 55}
                  y={tooltip.y - 15}
                  text={tooltip.text}
                  fontSize={11}
                  fill="#E0E0E0"
                  width={110}
                  align="center"
                />
              </Fragment>
            )}
          </Layer>
        </Stage>
      </Box>

      {/* Legend */}
      <Box sx={{ mt: 1 }}>
        <Typography variant="caption" sx={{ color: "#888", display: "block", mb: 0.5 }}>
          Legend:
        </Typography>
        <Stack direction="row" spacing={1} flexWrap="wrap" useFlexGap>
          <Box sx={{ display: "flex", alignItems: "center", gap: 0.5 }}>
            <Box
              sx={{
                width: 12,
                height: 12,
                borderRadius: "50%",
                backgroundColor: "#4CAF50",
                border: "2px solid #81C784",
              }}
            />
            <Typography variant="caption" sx={{ color: "#888" }}>
              Current
            </Typography>
          </Box>
          <Box sx={{ display: "flex", alignItems: "center", gap: 0.5 }}>
            <Box
              sx={{
                width: 12,
                height: 12,
                borderRadius: "50%",
                backgroundColor: "#2196F3",
                border: "2px solid #64B5F6",
              }}
            />
            <Typography variant="caption" sx={{ color: "#888" }}>
              Adjacent
            </Typography>
          </Box>
          <Box sx={{ display: "flex", alignItems: "center", gap: 0.5 }}>
            <Box
              sx={{
                width: 12,
                height: 12,
                borderRadius: "50%",
                backgroundColor: "#757575",
                border: "2px solid #9E9E9E",
              }}
            />
            <Typography variant="caption" sx={{ color: "#888" }}>
              Explored
            </Typography>
          </Box>
          <Box sx={{ display: "flex", alignItems: "center", gap: 0.5 }}>
            <Typography variant="caption" sx={{ color: "#FFD700", fontWeight: "bold" }}>
              ↑↓
            </Typography>
            <Typography variant="caption" sx={{ color: "#888" }}>
              Stairs
            </Typography>
          </Box>
        </Stack>
      </Box>
    </Box>
  );
};
