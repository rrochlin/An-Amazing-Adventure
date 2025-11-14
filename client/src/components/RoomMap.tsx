import { useMemo, useState, useEffect } from "react";
import { type GameState } from "../types/types";
import { Stage, Layer, Rect, Text, Circle, Group, Line } from "react-konva";
import { Box, IconButton, Typography, Chip, Stack, useColorScheme } from "@mui/material";
import ZoomInIcon from "@mui/icons-material/ZoomIn";
import ZoomOutIcon from "@mui/icons-material/ZoomOut";
import RestartAltIcon from "@mui/icons-material/RestartAlt";
import { DungeonColors, ColorTokens } from "../theme/theme";

export const RoomMap = ({ gameState }: { gameState: GameState }) => {
  const { mode } = useColorScheme();
  const isDark = mode === "dark" || mode === "system" || !mode;
  const colorMode = mode === "system" || !mode ? "dark" : mode;
  const colors = ColorTokens[colorMode];

  // Check if we have an AI-generated map image
  const hasMapImage = gameState.map_images && gameState.map_images["world-map"];
  const mapImageUrl = hasMapImage ? gameState.map_images!["world-map"] : null;

  // Room rendering constants
  const baseRoomSize = 50;
  const baseSpacing = 2;

  // Canvas dimensions - make responsive to container
  const [canvasSize, setCanvasSize] = useState({ width: 400, height: 400 });

  useEffect(() => {
    const updateSize = () => {
      const container = document.getElementById('map-container');
      if (container) {
        const width = container.clientWidth;
        const height = container.clientHeight;
        setCanvasSize({ width, height });
      }
    };

    updateSize();
    window.addEventListener('resize', updateSize);
    return () => window.removeEventListener('resize', updateSize);
  }, []);

  const [selectedLayer, setSelectedLayer] = useState(0);
  const [scale, setScale] = useState(1.0);
  const [position, setPosition] = useState({ x: 0, y: 0 });
  const [isDragging, setIsDragging] = useState(false);
  const [dragStart, setDragStart] = useState({ x: 0, y: 0 });
  const [hoveredRoom, setHoveredRoom] = useState<string | null>(null);
  const [torchFlicker, setTorchFlicker] = useState(0);

  // Torch flicker animation
  useEffect(() => {
    const interval = setInterval(() => {
      setTorchFlicker(Math.random() * 0.3 + 0.85); // 0.85 to 1.15
    }, 150);
    return () => clearInterval(interval);
  }, []);

  // Get all unique Z-levels
  const zLevels = useMemo(() => {
    if (!gameState.rooms) return [0];
    const levels = new Set(
      Object.values(gameState.rooms).map((room) => room.coordinates.z)
    );
    return Array.from(levels).sort((a, b) => b - a);
  }, [gameState.rooms]);

  // Set initial layer to player's current layer
  useEffect(() => {
    if (gameState.current_room) {
      setSelectedLayer(gameState.current_room.coordinates.z);
    }
  }, [gameState.current_room]);

  // Calculate room positions (before zoom/pan)
  const roomPositions = useMemo(() => {
    if (!gameState.rooms) return {};

    const positions: { [key: string]: { x: number; y: number; z: number } } = {};
    const centerX = canvasSize.width / 2;
    const centerY = canvasSize.height / 2;

    Object.keys(gameState.rooms).forEach((roomId) => {
      const room = gameState.rooms![roomId];
      positions[roomId] = {
        x: centerX + room.coordinates.x * baseSpacing,
        y: centerY + room.coordinates.y * baseSpacing,
        z: room.coordinates.z,
      };
    });

    return positions;
  }, [gameState.rooms, baseSpacing, canvasSize]);

  // Filter rooms by selected layer
  const roomsOnLayer = useMemo(() => {
    if (!gameState.rooms) return [];
    return Object.keys(gameState.rooms).filter(
      (roomId) => gameState.rooms![roomId].coordinates.z === selectedLayer
    );
  }, [gameState.rooms, selectedLayer]);

  // Track visited rooms
  const visitedRooms = useMemo(() => {
    const visited = new Set<string>();
    if (!gameState.rooms) return visited;

    visited.add(gameState.current_room.id);
    Object.values(gameState.current_room.connections).forEach((roomId) => {
      visited.add(roomId);
    });
    gameState.connected_rooms?.forEach((roomId) => visited.add(roomId));

    return visited;
  }, [gameState.current_room, gameState.connected_rooms, gameState.rooms]);

  // Zoom handlers
  const handleZoomIn = () => {
    setScale((s) => Math.min(s * 1.2, 3.0));
  };

  const handleZoomOut = () => {
    setScale((s) => Math.max(s / 1.2, 0.5));
  };

  const handleResetView = () => {
    setScale(1.0);
    setPosition({ x: 0, y: 0 });
  };

  // Pan handlers
  const handleMouseDown = (e: any) => {
    const stage = e.target.getStage();
    const pointerPos = stage.getPointerPosition();
    setIsDragging(true);
    setDragStart({
      x: pointerPos.x - position.x,
      y: pointerPos.y - position.y,
    });
  };

  const handleMouseMove = (e: any) => {
    if (!isDragging) return;
    const stage = e.target.getStage();
    const pointerPos = stage.getPointerPosition();
    setPosition({
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
                        boxShadow: "0 0 8px rgba(255, 215, 0, 0.8)",
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
          <IconButton size="small" onClick={handleZoomIn} sx={{ color: colors.icon }}>
            <ZoomInIcon fontSize="small" />
          </IconButton>
          <IconButton size="small" onClick={handleResetView} sx={{ color: colors.icon }}>
            <RestartAltIcon fontSize="small" />
          </IconButton>
          <IconButton size="small" onClick={handleZoomOut} sx={{ color: colors.icon }}>
            <ZoomOutIcon fontSize="small" />
          </IconButton>
        </Stack>
      </Stack>

      {/* Map Canvas */}
      <Box
        id="map-container"
        sx={{
          border: isDark
            ? `2px solid ${DungeonColors.wall}`
            : "2px solid #8B6F47",
          borderRadius: "4px",
          backgroundColor: isDark
            ? DungeonColors.fog
            : "rgba(212, 197, 169, 0.5)",
          overflow: "hidden",
          boxShadow: isDark
            ? `inset 0 0 20px ${DungeonColors.fog}`
            : "inset 0 0 15px rgba(139, 111, 71, 0.2)",
          position: "relative",
          width: "100%",
          height: "400px",
          display: "flex",
          alignItems: "center",
          justifyContent: "center",
        }}
      >
        {/* Display AI-generated map image if available */}
        {mapImageUrl ? (
          <Box
            component="img"
            src={mapImageUrl}
            alt="World Map"
            sx={{
              maxWidth: "100%",
              maxHeight: "100%",
              objectFit: "contain",
              borderRadius: "4px",
            }}
          />
        ) : (
        <>
        <Stage
          width={canvasSize.width}
          height={canvasSize.height}
          onMouseDown={handleMouseDown}
          onMouseMove={handleMouseMove}
          onMouseUp={handleMouseUp}
          onMouseLeave={handleMouseUp}
          style={{ cursor: isDragging ? "grabbing" : "grab" }}
        >
          <Layer
            scaleX={scale}
            scaleY={scale}
            x={position.x}
            y={position.y}
          >
            {/* Draw connections/corridors */}
            {gameState.rooms &&
              roomsOnLayer.map((roomId) => {
                const room = gameState.rooms![roomId];
                return Object.entries(room.connections).map(
                  ([direction, connectedRoomId]) => {
                    if (direction === "up" || direction === "down") return null;
                    if (!roomsOnLayer.includes(connectedRoomId)) return null;

                    const start = roomPositions[roomId];
                    const end = roomPositions[connectedRoomId];
                    if (!start || !end) return null;

                    const isActive =
                      roomId === gameState.current_room.id ||
                      connectedRoomId === gameState.current_room.id;

                    const dx = end.x - start.x;
                    const dy = end.y - start.y;
                    const length = Math.sqrt(dx * dx + dy * dy);
                    const angle = Math.atan2(dy, dx);

                    const corridorWidth = 16;
                    const roomOffset = baseRoomSize / 2;
                    const corridorLength = length - baseRoomSize;

                    const startX = start.x + Math.cos(angle) * roomOffset;
                    const startY = start.y + Math.sin(angle) * roomOffset;

                    return (
                      <Group
                        key={`${roomId}-${direction}-${connectedRoomId}`}
                        x={startX}
                        y={startY}
                        rotation={(angle * 180) / Math.PI}
                      >
                        {/* Corridor floor */}
                        <Rect
                          x={0}
                          y={-corridorWidth / 2}
                          width={corridorLength}
                          height={corridorWidth}
                          fillLinearGradientStartPoint={{ x: 0, y: 0 }}
                          fillLinearGradientEndPoint={{ x: 0, y: corridorWidth }}
                          fillLinearGradientColorStops={
                            isActive
                              ? [0, "#4E342E", 0.5, "#6D4C41", 1, "#4E342E"]
                              : [0, "#3E2723", 0.5, "#4E342E", 1, "#3E2723"]
                          }
                          shadowColor="black"
                          shadowBlur={5}
                          shadowOpacity={0.5}
                        />
                        {/* Corridor walls */}
                        <Line
                          points={[0, -corridorWidth / 2, corridorLength, -corridorWidth / 2]}
                          stroke={DungeonColors.wall}
                          strokeWidth={2}
                        />
                        <Line
                          points={[0, corridorWidth / 2, corridorLength, corridorWidth / 2]}
                          stroke={DungeonColors.wall}
                          strokeWidth={2}
                        />
                        {/* Doorway entrance */}
                        <Rect
                          x={-4}
                          y={-corridorWidth / 2 - 2}
                          width={8}
                          height={corridorWidth + 4}
                          fill={DungeonColors.door}
                          stroke={DungeonColors.wall}
                          strokeWidth={2}
                          cornerRadius={2}
                        />
                        {/* Doorway exit */}
                        <Rect
                          x={corridorLength - 4}
                          y={-corridorWidth / 2 - 2}
                          width={8}
                          height={corridorWidth + 4}
                          fill={DungeonColors.door}
                          stroke={DungeonColors.wall}
                          strokeWidth={2}
                          cornerRadius={2}
                        />
                      </Group>
                    );
                  }
                );
              })}

            {/* Draw rooms */}
            {gameState.rooms &&
              roomsOnLayer.map((roomId) => {
                const pos = roomPositions[roomId];
                const room = gameState.rooms![roomId];
                if (!pos) return null;

                const isCurrentRoom = roomId === gameState.current_room.id;
                const isConnected =
                  Object.values(gameState.current_room.connections).includes(roomId) ||
                  Object.values(room.connections).includes(gameState.current_room.id);
                const isVisited = visitedRooms.has(roomId);
                const isHovered = hoveredRoom === roomId;

                const hasUpConnection = "up" in room.connections;
                const hasDownConnection = "down" in room.connections;

                // Dynamic colors
                let fillGradient = ["#2C1810", "#1a0a0a"]; // Unexplored
                let strokeColor = DungeonColors.wall;
                let glowColor = "rgba(0, 0, 0, 0)";

                if (isCurrentRoom) {
                  fillGradient = ["#C9A962", "#8B7355"];
                  strokeColor = "#FFD700";
                  glowColor = `rgba(255, 215, 0, ${0.6 * torchFlicker})`;
                } else if (isConnected) {
                  fillGradient = ["#6B4E9D", "#4a3570"];
                  strokeColor = "#9575CD";
                  glowColor = "rgba(149, 117, 205, 0.3)";
                } else if (isVisited) {
                  fillGradient = ["#5D4037", "#3E2723"];
                  strokeColor = "#6D4C41";
                }

                const roomSize = baseRoomSize;

                return (
                  <Group
                    key={roomId}
                    x={pos.x}
                    y={pos.y}
                    onMouseEnter={() => setHoveredRoom(roomId)}
                    onMouseLeave={() => setHoveredRoom(null)}
                  >
                    {/* Outer glow for current room */}
                    {isCurrentRoom && (
                      <Rect
                        x={-roomSize / 2 - 6}
                        y={-roomSize / 2 - 6}
                        width={roomSize + 12}
                        height={roomSize + 12}
                        fill={glowColor}
                        cornerRadius={8}
                        shadowColor={glowColor}
                        shadowBlur={20}
                        shadowOpacity={torchFlicker}
                        listening={false}
                      />
                    )}

                    {/* Main room */}
                    <Rect
                      x={-roomSize / 2}
                      y={-roomSize / 2}
                      width={roomSize}
                      height={roomSize}
                      fillLinearGradientStartPoint={{ x: 0, y: 0 }}
                      fillLinearGradientEndPoint={{ x: 0, y: roomSize }}
                      fillLinearGradientColorStops={[
                        0,
                        fillGradient[0],
                        1,
                        fillGradient[1],
                      ]}
                      stroke={strokeColor}
                      strokeWidth={isCurrentRoom ? 3 : 2}
                      cornerRadius={4}
                      shadowColor="black"
                      shadowBlur={isCurrentRoom ? 12 : 6}
                      shadowOpacity={0.7}
                      shadowOffsetY={2}
                    />

                    {/* Inner highlight for depth */}
                    <Rect
                      x={-roomSize / 2 + 3}
                      y={-roomSize / 2 + 3}
                      width={roomSize - 6}
                      height={roomSize - 6}
                      stroke={strokeColor}
                      strokeWidth={1}
                      opacity={0.3}
                      cornerRadius={2}
                      listening={false}
                    />

                    {/* Stairs indicators */}
                    {hasUpConnection && (
                      <Group x={roomSize / 4} y={-roomSize / 4} listening={false}>
                        <Rect
                          x={-8}
                          y={-8}
                          width={16}
                          height={16}
                          fill={DungeonColors.torch}
                          stroke="#000"
                          strokeWidth={2}
                          cornerRadius={3}
                          shadowColor="rgba(255, 167, 38, 0.8)"
                          shadowBlur={8 * torchFlicker}
                          shadowOpacity={torchFlicker}
                        />
                        <Text
                          x={-6}
                          y={-8}
                          text="↑"
                          fontSize={16}
                          fill="#000"
                          fontStyle="bold"
                        />
                      </Group>
                    )}

                    {hasDownConnection && (
                      <Group x={-roomSize / 4} y={-roomSize / 4} listening={false}>
                        <Rect
                          x={-8}
                          y={-8}
                          width={16}
                          height={16}
                          fill={DungeonColors.torch}
                          stroke="#000"
                          strokeWidth={2}
                          cornerRadius={3}
                          shadowColor="rgba(255, 167, 38, 0.8)"
                          shadowBlur={8 * torchFlicker}
                          shadowOpacity={torchFlicker}
                        />
                        <Text
                          x={-6}
                          y={-8}
                          text="↓"
                          fontSize={16}
                          fill="#000"
                          fontStyle="bold"
                        />
                      </Group>
                    )}

                    {/* Player marker */}
                    {isCurrentRoom && (
                      <>
                        <Circle
                          x={0}
                          y={0}
                          radius={10}
                          fillRadialGradientStartPoint={{ x: 0, y: 0 }}
                          fillRadialGradientEndPoint={{ x: 0, y: 0 }}
                          fillRadialGradientStartRadius={0}
                          fillRadialGradientEndRadius={10}
                          fillRadialGradientColorStops={[
                            0,
                            "#FFD700",
                            0.7,
                            "#FFA000",
                            1,
                            "#FF6F00",
                          ]}
                          stroke="#000"
                          strokeWidth={2}
                          shadowColor="rgba(255, 215, 0, 0.9)"
                          shadowBlur={12 * torchFlicker}
                          shadowOpacity={torchFlicker}
                          listening={false}
                        />
                        <Circle
                          x={0}
                          y={0}
                          radius={4}
                          fill="#FFF"
                          opacity={0.8}
                          listening={false}
                        />
                      </>
                    )}

                    {/* Room label */}
                    {(isCurrentRoom || isConnected || isHovered) && (
                      <Group y={roomSize / 2 + 10} listening={false}>
                        <Rect
                          x={-50}
                          y={0}
                          width={100}
                          height={20}
                          fillLinearGradientStartPoint={{ x: 0, y: 0 }}
                          fillLinearGradientEndPoint={{ x: 0, y: 20 }}
                          fillLinearGradientColorStops={[
                            0,
                            "rgba(62, 44, 46, 0.95)",
                            1,
                            "rgba(26, 15, 30, 0.95)",
                          ]}
                          stroke={isHovered ? "#FFD700" : DungeonColors.doorway}
                          strokeWidth={1}
                          cornerRadius={4}
                          shadowColor="black"
                          shadowBlur={4}
                          shadowOpacity={0.7}
                        />
                        <Text
                          x={-45}
                          y={4}
                          text={roomId
                            .split("_")
                            .map(
                              (word) =>
                                word.charAt(0).toUpperCase() +
                                word.slice(1).toLowerCase()
                            )
                            .join(" ")}
                          fontSize={11}
                          fill="#E8DCC4"
                          fontFamily="Cinzel, Georgia, serif"
                          width={90}
                          align="center"
                        />
                      </Group>
                    )}

                    {/* Fog of war for unexplored */}
                    {!isVisited && (
                      <Rect
                        x={-roomSize / 2}
                        y={-roomSize / 2}
                        width={roomSize}
                        height={roomSize}
                        fill="rgba(13, 5, 8, 0.7)"
                        cornerRadius={4}
                        listening={false}
                      />
                    )}
                  </Group>
                );
              })}

          </Layer>

          {/* Empty second layer for UI overlay elements if needed */}
          <Layer />
        </Stage>

        {/* Compass Rose SVG Overlay */}
        <Box
          sx={{
            position: "absolute",
            bottom: 16,
            right: 16,
            width: 90,
            height: 90,
            pointerEvents: "none",
            filter: isDark ? "drop-shadow(0 0 8px rgba(201, 169, 98, 0.5))" : "drop-shadow(0 2px 4px rgba(0, 0, 0, 0.3))",
          }}
        >
          <svg viewBox="0 0 100 100" style={{ width: "100%", height: "100%" }}>
            {/* Outer circle */}
            <circle
              cx="50"
              cy="50"
              r="48"
              fill="none"
              stroke={isDark ? "#C9A962" : "#8B6F47"}
              strokeWidth="2"
            />

            {/* Inner circle */}
            <circle
              cx="50"
              cy="50"
              r="42"
              fill="none"
              stroke={isDark ? "#C9A962" : "#8B6F47"}
              strokeWidth="1"
              opacity="0.5"
            />

            {/* Center circle */}
            <circle
              cx="50"
              cy="50"
              r="8"
              fill={isDark ? "#3E2723" : "#D4C5A9"}
              stroke={isDark ? "#C9A962" : "#8B6F47"}
              strokeWidth="2"
            />

            {/* Cardinal direction labels */}
            <text
              x="50"
              y="28"
              textAnchor="middle"
              fill={isDark ? "#FFD700" : "#6B5638"}
              fontSize="24"
              fontFamily="Cinzel, serif"
              fontWeight="bold"
            >
              N
            </text>
            <text
              x="50"
              y="82"
              textAnchor="middle"
              fill={isDark ? "#C9A962" : "#8B6F47"}
              fontSize="20"
              fontFamily="Cinzel, serif"
            >
              S
            </text>
            <text
              x="78"
              y="56"
              textAnchor="middle"
              fill={isDark ? "#C9A962" : "#8B6F47"}
              fontSize="20"
              fontFamily="Cinzel, serif"
            >
              E
            </text>
            <text
              x="22"
              y="56"
              textAnchor="middle"
              fill={isDark ? "#C9A962" : "#8B6F47"}
              fontSize="20"
              fontFamily="Cinzel, serif"
            >
              W
            </text>
          </svg>
        </Box>
        </>
        )}
      </Box>

      {/* Legend - only show on canvas mode */}
      {!mapImageUrl && (
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
                width: 16,
                height: 16,
                background: "linear-gradient(180deg, #C9A962 0%, #8B7355 100%)",
                border: "2px solid #FFD700",
                borderRadius: "3px",
                boxShadow: "0 0 8px rgba(255, 215, 0, 0.5)",
              }}
            />
            <Typography variant="caption" sx={{ color: colors.text.secondary, fontFamily: "Crimson Text, Georgia, serif" }}>
              Current
            </Typography>
          </Box>
          <Box sx={{ display: "flex", alignItems: "center", gap: 0.5 }}>
            <Box
              sx={{
                width: 16,
                height: 16,
                background: "linear-gradient(180deg, #6B4E9D 0%, #4a3570 100%)",
                border: "2px solid #9575CD",
                borderRadius: "3px",
              }}
            />
            <Typography variant="caption" sx={{ color: colors.text.secondary, fontFamily: "Crimson Text, Georgia, serif" }}>
              Adjacent
            </Typography>
          </Box>
          <Box sx={{ display: "flex", alignItems: "center", gap: 0.5 }}>
            <Box
              sx={{
                width: 16,
                height: 16,
                background: "linear-gradient(180deg, #5D4037 0%, #3E2723 100%)",
                border: "2px solid #6D4C41",
                borderRadius: "3px",
              }}
            />
            <Typography variant="caption" sx={{ color: colors.text.secondary, fontFamily: "Crimson Text, Georgia, serif" }}>
              Explored
            </Typography>
          </Box>
          <Box sx={{ display: "flex", alignItems: "center", gap: 0.5 }}>
            <Typography
              variant="caption"
              sx={{
                color: colors.accent,
                fontWeight: "bold",
                fontSize: "1rem",
              }}
            >
              ↑↓
            </Typography>
            <Typography variant="caption" sx={{ color: colors.text.secondary, fontFamily: "Crimson Text, Georgia, serif" }}>
              Stairs
            </Typography>
          </Box>
        </Stack>
      </Box>
      )}
    </Box>
  );
};
