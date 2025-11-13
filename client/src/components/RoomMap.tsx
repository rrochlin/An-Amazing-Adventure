import { Fragment, useEffect, useMemo, useState } from "react";
import { type GameState } from "../types/types";
import { Stage, Layer, Circle, Line, Text, Rect, Arrow } from "react-konva";
import { Box, IconButton, Typography, Chip, Stack, useColorScheme } from "@mui/material";
import ZoomInIcon from "@mui/icons-material/ZoomIn";
import ZoomOutIcon from "@mui/icons-material/ZoomOut";
import RestartAltIcon from "@mui/icons-material/RestartAlt";
import { DungeonColors, ColorTokens } from "../theme/theme";

// RoomMap Component with enhanced features
export const RoomMap = ({ gameState }: { gameState: GameState }) => {
  const { mode } = useColorScheme();
  const isDark = mode === "dark" || mode === "system" || !mode;
  const colorMode = mode === "system" || !mode ? "dark" : mode;
  const colors = ColorTokens[colorMode];

  const stageWidth = 400;
  const stageHeight = 400;
  const roomWidth = 40;
  const roomHeight = 40;
  const corridorWidth = 8;
  const playerIconRadius = 6;

  const [hoveredRoom, setHoveredRoom] = useState<string | null>(null);
  const [tooltip, setTooltip] = useState<{
    x: number;
    y: number;
    text: string;
  } | null>(null);
  const [selectedLayer, setSelectedLayer] = useState(0); // Z-axis layer
  const [zoom, setZoom] = useState(1.0);
  const [pan, setPan] = useState({ x: 0, y: 0 });
  const [isDragging, setIsDragging] = useState(false);
  const [dragStart, setDragStart] = useState({ x: 0, y: 0 });

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
        x: centerX + room.coordinates.x * zoom + pan.x,
        y: centerY + room.coordinates.y * zoom + pan.y,
        z: room.coordinates.z,
      };
    });

    return positions;
  }, [gameState.rooms, zoom, pan, stageWidth, stageHeight]);

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
  const handleResetZoom = () => {
    setZoom(1.0);
    setPan({ x: 0, y: 0 });
  };

  const handleMouseDown = (e: any) => {
    const stage = e.target.getStage();
    const pointerPos = stage.getPointerPosition();
    setIsDragging(true);
    setDragStart({ x: pointerPos.x - pan.x, y: pointerPos.y - pan.y });
  };

  const handleMouseMove = (e: any) => {
    if (!isDragging) return;
    const stage = e.target.getStage();
    const pointerPos = stage.getPointerPosition();
    setPan({
      x: pointerPos.x - dragStart.x,
      y: pointerPos.y - dragStart.y,
    });
  };

  const handleMouseUp = () => {
    setIsDragging(false);
  };

  return (
    <Box sx={{ display: "flex", flexDirection: "column", gap: 1, width: "100%", minWidth: 0, maxWidth: "100%" }}>
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
                  ...(selectedLayer !== level && {
                    borderColor: colors.chipOutline,
                    color: colors.chipText,
                  }),
                }}
              />
            );
          })}
        </Stack>

        {/* Zoom Controls */}
        <Stack direction="row" spacing={0}>
          <IconButton
            size="small"
            onClick={handleZoomIn}
            sx={{ color: colors.icon }}
          >
            <ZoomInIcon fontSize="small" />
          </IconButton>
          <IconButton
            size="small"
            onClick={handleResetZoom}
            sx={{ color: colors.icon }}
          >
            <RestartAltIcon fontSize="small" />
          </IconButton>
          <IconButton
            size="small"
            onClick={handleZoomOut}
            sx={{ color: colors.icon }}
          >
            <ZoomOutIcon fontSize="small" />
          </IconButton>
        </Stack>
      </Stack>

      {/* Map Canvas */}
      <Box
        sx={{
          border:
            isDark
              ? `2px solid ${DungeonColors.wall}`
              : "2px solid #8B6F47",
          borderRadius: "4px",
          backgroundColor:
            isDark
              ? DungeonColors.fog
              : "rgba(212, 197, 169, 0.5)",
          overflow: "hidden",
          boxShadow:
            isDark
              ? `inset 0 0 20px ${DungeonColors.fog}`
              : "inset 0 0 15px rgba(139, 111, 71, 0.2)",
        }}
      >
        <Stage
          width={stageWidth}
          height={stageHeight}
          onMouseDown={handleMouseDown}
          onMouseMove={handleMouseMove}
          onMouseUp={handleMouseUp}
          onMouseLeave={handleMouseUp}
          style={{ cursor: isDragging ? "grabbing" : "grab" }}
        >
          <Layer>
            {/* Draw corridors as thick passages on current layer */}
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

                    // Skip vertical connections for corridor drawing
                    if (direction === "up" || direction === "down") return null;

                    const dx = end.x - start.x;
                    const dy = end.y - start.y;
                    const length = Math.sqrt(dx * dx + dy * dy);
                    const angle = Math.atan2(dy, dx);

                    // Calculate corridor position (connects room edges)
                    const corridorLength = length - roomWidth;
                    const corridorX = start.x + Math.cos(angle) * (roomWidth / 2);
                    const corridorY = start.y + Math.sin(angle) * (roomHeight / 2);

                    const isCurrentRoomConnection =
                      roomId === gameState.current_room.id ||
                      connectedRoomId === gameState.current_room.id;

                    return (
                      <Fragment key={`${roomId}-${direction}-${connectedRoomId}`}>
                        {/* Corridor floor */}
                        <Rect
                          x={corridorX - corridorWidth / 2}
                          y={corridorY - corridorWidth / 2}
                          width={corridorLength}
                          height={corridorWidth}
                          fill={isCurrentRoomConnection ? DungeonColors.corridor : DungeonColors.floor}
                          rotation={(angle * 180) / Math.PI}
                          offsetX={0}
                          offsetY={0}
                        />
                        {/* Corridor walls - top */}
                        <Line
                          points={[
                            corridorX,
                            corridorY - corridorWidth / 2,
                            corridorX + Math.cos(angle) * corridorLength,
                            corridorY + Math.sin(angle) * corridorLength - corridorWidth / 2,
                          ]}
                          stroke={DungeonColors.wall}
                          strokeWidth={2}
                        />
                        {/* Corridor walls - bottom */}
                        <Line
                          points={[
                            corridorX,
                            corridorY + corridorWidth / 2,
                            corridorX + Math.cos(angle) * corridorLength,
                            corridorY + Math.sin(angle) * corridorLength + corridorWidth / 2,
                          ]}
                          stroke={DungeonColors.wall}
                          strokeWidth={2}
                        />
                      </Fragment>
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

                // Color coding with dungeon theme
                let fillColor = DungeonColors.unexploredRoom; // Unvisited
                let strokeColor = DungeonColors.wall;
                if (isCurrentRoom) {
                  fillColor = DungeonColors.currentRoom; // Current: Gold
                  strokeColor = DungeonColors.doorway;
                } else if (isConnectedToCurrent) {
                  fillColor = DungeonColors.adjacentRoom; // Connected: Purple
                  strokeColor = DungeonColors.wallHighlight;
                } else if (isVisited) {
                  fillColor = DungeonColors.exploredRoom; // Visited: Dark brown
                  strokeColor = DungeonColors.wall;
                }

                // Check for vertical connections
                const hasUpConnection = Object.keys(room.connections).includes("up");
                const hasDownConnection = Object.keys(room.connections).includes("down");

                return (
                  <Fragment key={roomId}>
                    {/* Main room rectangle */}
                    <Rect
                      x={pos.x - roomWidth / 2}
                      y={pos.y - roomHeight / 2}
                      width={roomWidth}
                      height={roomHeight}
                      fill={fillColor}
                      stroke={strokeColor}
                      strokeWidth={isCurrentRoom ? 3 : 2}
                      shadowColor="black"
                      shadowBlur={isCurrentRoom ? 10 : 5}
                      shadowOpacity={0.6}
                      cornerRadius={2}
                      listening={true}
                      perfectDrawEnabled={false}
                      onMouseEnter={() => handleRoomHover(roomId, pos)}
                      onMouseLeave={handleRoomLeave}
                    />
                    {/* Inner shadow for depth */}
                    <Rect
                      x={pos.x - roomWidth / 2 + 2}
                      y={pos.y - roomHeight / 2 + 2}
                      width={roomWidth - 4}
                      height={roomHeight - 4}
                      stroke={strokeColor}
                      strokeWidth={1}
                      opacity={0.3}
                      cornerRadius={1}
                    />

                    {/* Up arrow indicator - stairs going up */}
                    {hasUpConnection && (
                      <Fragment>
                        <Rect
                          x={pos.x + roomWidth / 4}
                          y={pos.y - roomHeight / 4}
                          width={8}
                          height={8}
                          fill={DungeonColors.torch}
                          stroke="#000"
                          strokeWidth={1}
                          cornerRadius={1}
                        />
                        <Text
                          x={pos.x + roomWidth / 4}
                          y={pos.y - roomHeight / 4 - 1}
                          text="↑"
                          fontSize={10}
                          fill="#000"
                          fontStyle="bold"
                        />
                      </Fragment>
                    )}

                    {/* Down arrow indicator - stairs going down */}
                    {hasDownConnection && (
                      <Fragment>
                        <Rect
                          x={pos.x - roomWidth / 4 - 8}
                          y={pos.y - roomHeight / 4}
                          width={8}
                          height={8}
                          fill={DungeonColors.torch}
                          stroke="#000"
                          strokeWidth={1}
                          cornerRadius={1}
                        />
                        <Text
                          x={pos.x - roomWidth / 4 - 8}
                          y={pos.y - roomHeight / 4 - 1}
                          text="↓"
                          fontSize={10}
                          fill="#000"
                          fontStyle="bold"
                        />
                      </Fragment>
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
                          y={pos.y + roomHeight / 2 + 5}
                          width={120}
                          height={18}
                          fill={hoveredRoom === roomId ? DungeonColors.wallHighlight : DungeonColors.wall}
                          stroke={DungeonColors.doorway}
                          strokeWidth={1}
                          cornerRadius={3}
                          shadowColor="black"
                          shadowBlur={3}
                          shadowOpacity={0.5}
                        />
                        <Text
                          x={pos.x - 55}
                          y={pos.y + roomHeight / 2 + 8}
                          text={roomId
                            .split("_")
                            .map(
                              (word) =>
                                word.charAt(0).toUpperCase() +
                                word.slice(1).toLowerCase()
                            )
                            .join(" ")}
                          fontSize={10}
                          fill="#E8DCC4"
                          fontFamily="Cinzel, Georgia, serif"
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
                  fill={DungeonColors.wall}
                  stroke={DungeonColors.doorway}
                  strokeWidth={1}
                  cornerRadius={4}
                  shadowColor="black"
                  shadowBlur={5}
                  shadowOpacity={0.7}
                />
                <Text
                  x={tooltip.x - 55}
                  y={tooltip.y - 15}
                  text={tooltip.text}
                  fontSize={11}
                  fill="#E8DCC4"
                  fontFamily="Cinzel, Georgia, serif"
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
        <Typography
          variant="caption"
          sx={{
            color: colors.text.primary,
            display: "block",
            mb: 0.5,
            fontFamily: "Cinzel, Georgia, serif",
            fontWeight: 600,
            letterSpacing: "0.05em",
          }}
        >
          Legend:
        </Typography>
        <Stack direction="row" spacing={1} flexWrap="wrap" useFlexGap>
          <Box sx={{ display: "flex", alignItems: "center", gap: 0.5 }}>
            <Box
              sx={{
                width: 14,
                height: 14,
                backgroundColor: DungeonColors.currentRoom,
                border: `2px solid ${DungeonColors.doorway}`,
                borderRadius: "2px",
              }}
            />
            <Typography
              variant="caption"
              sx={{
                color: colors.text.secondary,
                fontFamily: "Crimson Text, Georgia, serif",
              }}
            >
              Current
            </Typography>
          </Box>
          <Box sx={{ display: "flex", alignItems: "center", gap: 0.5 }}>
            <Box
              sx={{
                width: 14,
                height: 14,
                backgroundColor: DungeonColors.adjacentRoom,
                border: `2px solid ${DungeonColors.wallHighlight}`,
                borderRadius: "2px",
              }}
            />
            <Typography
              variant="caption"
              sx={{
                color: colors.text.secondary,
                fontFamily: "Crimson Text, Georgia, serif",
              }}
            >
              Adjacent
            </Typography>
          </Box>
          <Box sx={{ display: "flex", alignItems: "center", gap: 0.5 }}>
            <Box
              sx={{
                width: 14,
                height: 14,
                backgroundColor: DungeonColors.exploredRoom,
                border: `2px solid ${DungeonColors.wall}`,
                borderRadius: "2px",
              }}
            />
            <Typography
              variant="caption"
              sx={{
                color: colors.text.secondary,
                fontFamily: "Crimson Text, Georgia, serif",
              }}
            >
              Explored
            </Typography>
          </Box>
          <Box sx={{ display: "flex", alignItems: "center", gap: 0.5 }}>
            <Typography
              variant="caption"
              sx={{
                color: colors.accent,
                fontWeight: "bold",
                fontSize: "1rem"
              }}
            >
              ↑↓
            </Typography>
            <Typography
              variant="caption"
              sx={{
                color: colors.text.secondary,
                fontFamily: "Crimson Text, Georgia, serif",
              }}
            >
              Stairs
            </Typography>
          </Box>
        </Stack>
      </Box>
    </Box>
  );
};
