import { useMemo, useState, useEffect, useCallback, useRef } from "react";
import { type GameStateView, type RoomView } from "../types/types";
import { Stage, Layer, Rect, Text, Circle as KonvaCircle, Group, Line } from "react-konva";
import {
  Box,
  Chip,
  IconButton,
  Stack,
  Tooltip,
  Typography,
  useColorScheme,
} from "@mui/material";
import ZoomInIcon from "@mui/icons-material/ZoomIn";
import ZoomOutIcon from "@mui/icons-material/ZoomOut";
import RestartAltIcon from "@mui/icons-material/RestartAlt";
import ChevronRightIcon from "@mui/icons-material/ChevronRight";
import ChevronLeftIcon from "@mui/icons-material/ChevronLeft";
import { DungeonColors, ColorTokens } from "../theme/theme";
import { useGameStore } from "../store/gameStore";

interface RoomMapProps {
  gameState: GameStateView | null;
  /** Called when the user hovers or clicks a room node.
   *  Pass null to clear the selection (restore current room). */
  onRoomFocus?: (room: RoomView | null) => void;
  /** Called when the user clicks the expand/collapse toggle button. */
  onExpand?: () => void;
  /** Whether the map is currently in expanded (full-width) mode. */
  expanded?: boolean;
}

// ---------------------------------------------------------------------------
// Inner canvas – shared between inline view and modal pop-out
// ---------------------------------------------------------------------------
interface MapCanvasProps {
  gameState: GameStateView | null;
  visitedRooms: Set<string>;
  width: number;
  height: number;
  onRoomFocus?: (room: RoomView | null) => void;
}

const MapCanvas = ({
  gameState,
  visitedRooms,
  width,
  height,
  onRoomFocus,
}: MapCanvasProps) => {
  const { mode } = useColorScheme();
  const isDark = mode === "dark" || mode === "system" || !mode;

  const baseRoomSize = 44;
  const worldUnitToPx = baseRoomSize + 66; // 110px per 100 world-units → 66px corridor gap

  const [scale, setScale] = useState(1.0);
  const [position, setPosition] = useState({ x: 0, y: 0 });
  const [isDragging, setIsDragging] = useState(false);
  const [dragStart, setDragStart] = useState({ x: 0, y: 0 });
  const [hoveredRoom, setHoveredRoom] = useState<string | null>(null);
  const [torchFlicker, setTorchFlicker] = useState(0);
  const [selectedLayer, setSelectedLayer] = useState(0);

  // Track the previous current_room id so we only auto-pan on actual moves
  const prevRoomId = useRef<string | null>(null);

  // Torch flicker animation
  useEffect(() => {
    const interval = setInterval(() => {
      setTorchFlicker(Math.random() * 0.3 + 0.85);
    }, 150);
    return () => clearInterval(interval);
  }, []);

  // Get unique Z-levels that have at least one visited room (plus the player's
  // current floor so the active chip is always present).
  const zLevels = useMemo(() => {
    if (!gameState?.rooms) return [0];
    const levels = new Set<number>();
    Object.entries(gameState.rooms).forEach(([roomId, room]) => {
      if (visitedRooms.has(roomId)) levels.add(room.coordinates.z);
    });
    // Always include the player's current floor
    levels.add(gameState.current_room.coordinates.z);
    return Array.from(levels).sort((a, b) => b - a);
  }, [gameState?.rooms, gameState?.current_room, visitedRooms]);

  // Keep selected layer in sync with player's floor
  useEffect(() => {
    if (gameState?.current_room) {
      setSelectedLayer(gameState.current_room.coordinates.z);
    }
  }, [gameState?.current_room]);

  // Calculate room positions (world-units → canvas pixels, before zoom/pan)
  const roomPositions = useMemo(() => {
    if (!gameState?.rooms) return {} as Record<string, { x: number; y: number; z: number }>;
    const positions: Record<string, { x: number; y: number; z: number }> = {};
    const centerX = width / 2;
    const centerY = height / 2;
    Object.keys(gameState.rooms).forEach((roomId) => {
      const room = gameState.rooms![roomId];
      positions[roomId] = {
        x: centerX + (room.coordinates.x / 100) * worldUnitToPx,
        y: centerY + (room.coordinates.y / 100) * worldUnitToPx,
        z: room.coordinates.z,
      };
    });
    return positions;
  }, [gameState?.rooms, worldUnitToPx, width, height]);

  // UI-FUT-2: Auto-pan to current room whenever the player moves
  useEffect(() => {
    if (!gameState?.current_room) return;
    const roomId = gameState.current_room.id;
    if (roomId === prevRoomId.current) return;
    prevRoomId.current = roomId;

    const pos = roomPositions[roomId];
    if (!pos) return;
    // Center the current room: offset = canvasCenter - roomPos * scale
    setPosition({
      x: width / 2 - pos.x * scale,
      y: height / 2 - pos.y * scale,
    });
  }, [gameState?.current_room?.id, roomPositions, width, height, scale]);

  // Filter rooms by selected layer
  const roomsOnLayer = useMemo(() => {
    if (!gameState?.rooms) return [] as string[];
    return Object.keys(gameState.rooms).filter(
      (roomId) => gameState.rooms![roomId].coordinates.z === selectedLayer
    );
  }, [gameState?.rooms, selectedLayer]);

  const handleZoomIn = () => setScale((s) => Math.min(s * 1.2, 3.0));
  const handleZoomOut = () => setScale((s) => Math.max(s / 1.2, 0.5));

  // Scroll-to-zoom centered on the mouse pointer position
  const handleWheel = (e: any) => {
    e.evt.preventDefault();
    const stage = e.target.getStage();
    const pointer = stage.getPointerPosition();
    if (!pointer) return;

    const zoomFactor = e.evt.deltaY < 0 ? 1.1 : 1 / 1.1;
    const newScale = Math.min(Math.max(scale * zoomFactor, 0.3), 4.0);

    // Adjust position so the point under the mouse stays fixed:
    // newPos = pointer - (pointer - oldPos) * (newScale / oldScale)
    const mousePointTo = {
      x: (pointer.x - position.x) / scale,
      y: (pointer.y - position.y) / scale,
    };
    setPosition({
      x: pointer.x - mousePointTo.x * newScale,
      y: pointer.y - mousePointTo.y * newScale,
    });
    setScale(newScale);
  };
  const handleResetView = () => {
    setScale(1.0);
    setPosition({ x: 0, y: 0 });
    prevRoomId.current = null; // force re-center on next render
  };

  const handleMouseDown = (e: any) => {
    const stage = e.target.getStage();
    const pointerPos = stage.getPointerPosition();
    setIsDragging(true);
    setDragStart({ x: pointerPos.x - position.x, y: pointerPos.y - position.y });
  };
  const handleMouseMove = (e: any) => {
    if (!isDragging) return;
    const stage = e.target.getStage();
    const pointerPos = stage.getPointerPosition();
    setPosition({ x: pointerPos.x - dragStart.x, y: pointerPos.y - dragStart.y });
  };
  const handleMouseUp = () => setIsDragging(false);

  const handleRoomEnter = useCallback(
    (roomId: string) => {
      setHoveredRoom(roomId);
      if (onRoomFocus && gameState?.rooms?.[roomId]) {
        onRoomFocus(gameState.rooms[roomId]);
      }
    },
    [onRoomFocus, gameState?.rooms]
  );

  const handleRoomLeave = useCallback(() => {
    setHoveredRoom(null);
    if (onRoomFocus) onRoomFocus(null);
  }, [onRoomFocus]);

  const handleRoomClick = useCallback(
    (roomId: string) => {
      if (onRoomFocus && gameState?.rooms?.[roomId]) {
        onRoomFocus(gameState.rooms[roomId]);
      }
    },
    [onRoomFocus, gameState?.rooms]
  );

  return (
    <Box sx={{ display: "flex", flexDirection: "column", gap: 1, width: "100%", minWidth: 0 }}>
      {/* Controls row */}
      <Stack direction="row" spacing={1} justifyContent="space-between" alignItems="center">
        {/* Layer Selector */}
        <Stack direction="row" spacing={0.5} flexWrap="wrap">
          {zLevels.map((level) => {
            const hasPlayer = gameState?.current_room.coordinates.z === level;
            const levelLabel =
              level === 0 ? "Ground"
              : level === 1 ? "Upper"
              : level === -1 ? "Underground"
              : level > 0 ? `Floor +${level}`
              : `Floor ${level}`;
            return (
              <Chip
                key={level}
                label={levelLabel}
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
                sx={{ fontSize: "0.75rem", height: "24px" }}
              />
            );
          })}
        </Stack>

        {/* Zoom Controls */}
        <Stack direction="row" spacing={0}>
          <IconButton size="small" onClick={handleZoomIn}>
            <ZoomInIcon fontSize="small" />
          </IconButton>
          <IconButton size="small" onClick={handleResetView}>
            <RestartAltIcon fontSize="small" />
          </IconButton>
          <IconButton size="small" onClick={handleZoomOut}>
            <ZoomOutIcon fontSize="small" />
          </IconButton>
        </Stack>
      </Stack>

      {/* Canvas */}
      <Stage
        width={width}
        height={height}
        onMouseDown={handleMouseDown}
        onMouseMove={handleMouseMove}
        onMouseUp={handleMouseUp}
        onMouseLeave={handleMouseUp}
        onWheel={handleWheel}
        style={{ cursor: isDragging ? "grabbing" : "grab" }}
      >
        <Layer scaleX={scale} scaleY={scale} x={position.x} y={position.y}>
          {/* Corridors — only draw if the source room is visited */}
          {gameState?.rooms &&
            roomsOnLayer.map((roomId) => {
              // Never draw corridors from unvisited rooms
              if (!visitedRooms.has(roomId)) return null;
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
                  // UI-FUT-3: teal highlight for corridors leading to connected exits
                  const isExit =
                    Object.values(gameState.current_room.connections).includes(connectedRoomId) ||
                    Object.values(gameState.current_room.connections).includes(roomId);

                  const dx = end.x - start.x;
                  const dy = end.y - start.y;
                  const length = Math.sqrt(dx * dx + dy * dy);
                  const angle = Math.atan2(dy, dx);

                  const corridorWidth = 16;
                  const roomOffset = baseRoomSize / 2;
                  const corridorLength = length - baseRoomSize;

                  const startX = start.x + Math.cos(angle) * roomOffset;
                  const startY = start.y + Math.sin(angle) * roomOffset;

                  const corridorColors = isExit
                    ? [0, "#1a5f5f", 0.5, "#26888a", 1, "#1a5f5f"]
                    : isActive
                    ? [0, "#4E342E", 0.5, "#6D4C41", 1, "#4E342E"]
                    : [0, "#3E2723", 0.5, "#4E342E", 1, "#3E2723"];

                  return (
                    <Group
                      key={`${roomId}-${direction}-${connectedRoomId}`}
                      x={startX}
                      y={startY}
                      rotation={(angle * 180) / Math.PI}
                    >
                      <Rect
                        x={0}
                        y={-corridorWidth / 2}
                        width={corridorLength}
                        height={corridorWidth}
                        fillLinearGradientStartPoint={{ x: 0, y: 0 }}
                        fillLinearGradientEndPoint={{ x: 0, y: corridorWidth }}
                        fillLinearGradientColorStops={corridorColors}
                        shadowColor="black"
                        shadowBlur={5}
                        shadowOpacity={0.5}
                      />
                      <Line
                        points={[0, -corridorWidth / 2, corridorLength, -corridorWidth / 2]}
                        stroke={isExit ? "#26a0a4" : DungeonColors.wall}
                        strokeWidth={isExit ? 1.5 : 2}
                      />
                      <Line
                        points={[0, corridorWidth / 2, corridorLength, corridorWidth / 2]}
                        stroke={isExit ? "#26a0a4" : DungeonColors.wall}
                        strokeWidth={isExit ? 1.5 : 2}
                      />
                      <Rect
                        x={-4}
                        y={-corridorWidth / 2 - 2}
                        width={8}
                        height={corridorWidth + 4}
                        fill={isExit ? "#26a0a4" : DungeonColors.door}
                        stroke={isExit ? "#4dd0d4" : DungeonColors.wall}
                        strokeWidth={2}
                        cornerRadius={2}
                      />
                      <Rect
                        x={corridorLength - 4}
                        y={-corridorWidth / 2 - 2}
                        width={8}
                        height={corridorWidth + 4}
                        fill={isExit ? "#26a0a4" : DungeonColors.door}
                        stroke={isExit ? "#4dd0d4" : DungeonColors.wall}
                        strokeWidth={2}
                        cornerRadius={2}
                      />
                    </Group>
                  );
                }
              );
            })}

          {/* Rooms */}
          {gameState?.rooms &&
            roomsOnLayer.map((roomId) => {
              const pos = roomPositions[roomId];
              const room = gameState.rooms![roomId];
              if (!pos) return null;

              const isCurrentRoom = roomId === gameState.current_room.id;
              // UI-FUT-3: exits are rooms directly reachable from current room
              const isExit =
                Object.values(gameState.current_room.connections).includes(roomId);
              const isConnected =
                isExit ||
                Object.values(room.connections).includes(gameState.current_room.id);
              const isVisited = visitedRooms.has(roomId);
              const isHovered = hoveredRoom === roomId;

              // Fog of war: unvisited rooms that aren't exits are fully hidden
              if (!isVisited && !isExit) return null;

              // Exits that haven't been visited yet: render as a dim "?" silhouette
              if (!isVisited && isExit) {
                const roomSize = baseRoomSize;
                return (
                  <Group key={roomId} x={pos.x} y={pos.y} opacity={0.45} listening={false}>
                    <Rect
                      x={-roomSize / 2} y={-roomSize / 2}
                      width={roomSize} height={roomSize}
                      fill="#1a0f1a"
                      stroke="#4dd0d4"
                      strokeWidth={1}
                      cornerRadius={4}
                      dash={[4, 4]}
                    />
                    <Text
                      x={-roomSize / 2} y={-10}
                      width={roomSize}
                      text="?"
                      fontSize={20}
                      fill="#4dd0d4"
                      align="center"
                      listening={false}
                    />
                  </Group>
                );
              }

              const hasUpConnection = "up" in room.connections;
              const hasDownConnection = "down" in room.connections;

              let fillGradient = ["#2C1810", "#1a0a0a"];
              let strokeColor = DungeonColors.wall;
              let glowColor = "rgba(0, 0, 0, 0)";

              if (isCurrentRoom) {
                fillGradient = ["#C9A962", "#8B7355"];
                strokeColor = "#FFD700";
                glowColor = `rgba(255, 215, 0, ${0.6 * torchFlicker})`;
              } else if (isExit) {
                // UI-FUT-3: teal for reachable exits
                fillGradient = ["#1a6b6e", "#0f4547"];
                strokeColor = "#4dd0d4";
                glowColor = "rgba(77, 208, 212, 0.3)";
              } else if (isConnected) {
                fillGradient = ["#6B4E9D", "#4a3570"];
                strokeColor = "#9575CD";
                glowColor = "rgba(149, 117, 205, 0.3)";
              } else if (isVisited) {
                fillGradient = ["#5D4037", "#3E2723"];
                strokeColor = "#6D4C41";
              }

              const roomSize = baseRoomSize;

              // UI-FUT-5: collect NPC tokens for this room
              const npcs = room.occupants ?? [];
              const isPlayerHere = isCurrentRoom;

              return (
                <Group
                  key={roomId}
                  x={pos.x}
                  y={pos.y}
                  onMouseEnter={() => handleRoomEnter(roomId)}
                  onMouseLeave={handleRoomLeave}
                  onClick={() => handleRoomClick(roomId)}
                >
                  {/* Outer glow */}
                  {(isCurrentRoom || isExit) && (
                    <Rect
                      x={-roomSize / 2 - 6}
                      y={-roomSize / 2 - 6}
                      width={roomSize + 12}
                      height={roomSize + 12}
                      fill={glowColor}
                      cornerRadius={8}
                      shadowColor={glowColor}
                      shadowBlur={isCurrentRoom ? 20 : 12}
                      shadowOpacity={isCurrentRoom ? torchFlicker : 0.7}
                      listening={false}
                    />
                  )}

                  {/* Main room tile */}
                  <Rect
                    x={-roomSize / 2}
                    y={-roomSize / 2}
                    width={roomSize}
                    height={roomSize}
                    fillLinearGradientStartPoint={{ x: 0, y: 0 }}
                    fillLinearGradientEndPoint={{ x: 0, y: roomSize }}
                    fillLinearGradientColorStops={[0, fillGradient[0], 1, fillGradient[1]]}
                    stroke={isHovered ? "#FFD700" : strokeColor}
                    strokeWidth={isCurrentRoom || isHovered ? 3 : 2}
                    cornerRadius={4}
                    shadowColor="black"
                    shadowBlur={isCurrentRoom ? 12 : 6}
                    shadowOpacity={0.7}
                    shadowOffsetY={2}
                  />

                  {/* Inner highlight */}
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

                  {/* Stairs */}
                  {hasUpConnection && (
                    <Group x={roomSize / 4} y={-roomSize / 4} listening={false}>
                      <Rect
                        x={-8} y={-8} width={16} height={16}
                        fill={DungeonColors.torch} stroke="#000" strokeWidth={2} cornerRadius={3}
                        shadowColor="rgba(255, 167, 38, 0.8)"
                        shadowBlur={8 * torchFlicker} shadowOpacity={torchFlicker}
                      />
                      <Text x={-6} y={-8} text="↑" fontSize={16} fill="#000" fontStyle="bold" />
                    </Group>
                  )}
                  {hasDownConnection && (
                    <Group x={-roomSize / 4} y={-roomSize / 4} listening={false}>
                      <Rect
                        x={-8} y={-8} width={16} height={16}
                        fill={DungeonColors.torch} stroke="#000" strokeWidth={2} cornerRadius={3}
                        shadowColor="rgba(255, 167, 38, 0.8)"
                        shadowBlur={8 * torchFlicker} shadowOpacity={torchFlicker}
                      />
                      <Text x={-6} y={-8} text="↓" fontSize={16} fill="#000" fontStyle="bold" />
                    </Group>
                  )}

                  {/* Player marker */}
                  {isPlayerHere && (
                    <>
                      <KonvaCircle
                        x={0} y={0} radius={10}
                        fillRadialGradientStartPoint={{ x: 0, y: 0 }}
                        fillRadialGradientEndPoint={{ x: 0, y: 0 }}
                        fillRadialGradientStartRadius={0}
                        fillRadialGradientEndRadius={10}
                        fillRadialGradientColorStops={[0, "#FFD700", 0.7, "#FFA000", 1, "#FF6F00"]}
                        stroke="#000" strokeWidth={2}
                        shadowColor="rgba(255, 215, 0, 0.9)"
                        shadowBlur={12 * torchFlicker}
                        shadowOpacity={torchFlicker}
                        listening={false}
                      />
                      <KonvaCircle x={0} y={0} radius={4} fill="#FFF" opacity={0.8} listening={false} />
                    </>
                  )}

                  {/* UI-FUT-5: NPC/character tokens */}
                  {npcs.length > 0 && (
                    <Group listening={false}>
                      {npcs.slice(0, 3).map((npc, idx) => {
                        // Arrange up to 3 tokens in a small row near the bottom of the tile
                        const total = Math.min(npcs.length, 3);
                        const spacing = 12;
                        const startX = -((total - 1) * spacing) / 2;
                        const tx = startX + idx * spacing;
                        const ty = roomSize / 2 - 10;
                        const tokenColor = npc.friendly ? "#4CAF50" : "#F44336";
                        const borderColor = npc.friendly ? "#81C784" : "#EF9A9A";
                        return (
                          <Group key={npc.id} x={tx} y={ty}>
                            <KonvaCircle
                              x={0} y={0} radius={5}
                              fill={tokenColor}
                              stroke={borderColor}
                              strokeWidth={1.5}
                              shadowColor={tokenColor}
                              shadowBlur={4}
                              shadowOpacity={0.6}
                            />
                            {/* skull for dead NPCs */}
                            {!npc.alive && (
                              <Text
                                x={-4} y={-5}
                                text="✕"
                                fontSize={7}
                                fill="#fff"
                              />
                            )}
                          </Group>
                        );
                      })}
                      {/* overflow indicator */}
                      {npcs.length > 3 && (
                        <Text
                          x={-roomSize / 2 + 2}
                          y={roomSize / 2 - 16}
                          text={`+${npcs.length - 3}`}
                          fontSize={8}
                          fill="#ccc"
                        />
                      )}
                    </Group>
                  )}

                  {/* Room label */}
                  {(isCurrentRoom || isConnected || isHovered) && (
                    <Group y={roomSize / 2 + 10} listening={false}>
                      <Rect
                        x={-56} y={0} width={112} height={20}
                        fillLinearGradientStartPoint={{ x: 0, y: 0 }}
                        fillLinearGradientEndPoint={{ x: 0, y: 20 }}
                        fillLinearGradientColorStops={[
                          0, "rgba(62, 44, 46, 0.95)",
                          1, "rgba(26, 15, 30, 0.95)",
                        ]}
                        stroke={isHovered ? "#FFD700" : isExit ? "#4dd0d4" : DungeonColors.doorway}
                        strokeWidth={1}
                        cornerRadius={4}
                        shadowColor="black" shadowBlur={4} shadowOpacity={0.7}
                      />
                      <Text
                        x={-52} y={4}
                        text={room.name}
                        fontSize={11}
                        fill="#E8DCC4"
                        fontFamily="Cinzel, Georgia, serif"
                        width={104}
                        align="center"
                        wrap="none"
                        ellipsis={true}
                      />
                    </Group>
                  )}


                </Group>
              );
            })}
        </Layer>

        {/* UI overlay layer (empty, reserved) */}
        <Layer />
      </Stage>

      {/* Compass Rose */}
      <Box
        sx={{
          position: "absolute",
          bottom: 16,
          right: 16,
          width: 90,
          height: 90,
          pointerEvents: "none",
          filter: isDark
            ? "drop-shadow(0 0 8px rgba(201, 169, 98, 0.5))"
            : "drop-shadow(0 2px 4px rgba(0, 0, 0, 0.3))",
        }}
      >
        <svg viewBox="0 0 100 100" style={{ width: "100%", height: "100%" }}>
          <circle cx="50" cy="50" r="48" fill="none" stroke={isDark ? "#C9A962" : "#8B6F47"} strokeWidth="2" />
          <circle cx="50" cy="50" r="42" fill="none" stroke={isDark ? "#C9A962" : "#8B6F47"} strokeWidth="1" opacity="0.5" />
          <circle cx="50" cy="50" r="8" fill={isDark ? "#3E2723" : "#D4C5A9"} stroke={isDark ? "#C9A962" : "#8B6F47"} strokeWidth="2" />
          <text x="50" y="28" textAnchor="middle" fill={isDark ? "#FFD700" : "#6B5638"} fontSize="24" fontFamily="Cinzel, serif" fontWeight="bold">N</text>
          <text x="50" y="82" textAnchor="middle" fill={isDark ? "#C9A962" : "#8B6F47"} fontSize="20" fontFamily="Cinzel, serif">S</text>
          <text x="78" y="56" textAnchor="middle" fill={isDark ? "#C9A962" : "#8B6F47"} fontSize="20" fontFamily="Cinzel, serif">E</text>
          <text x="22" y="56" textAnchor="middle" fill={isDark ? "#C9A962" : "#8B6F47"} fontSize="20" fontFamily="Cinzel, serif">W</text>
        </svg>
      </Box>
    </Box>
  );
};

// ---------------------------------------------------------------------------
// Public component — inline map with slide-expand support (UI-FUT-1)
// ---------------------------------------------------------------------------
export const RoomMap = ({ gameState, onRoomFocus, onExpand, expanded = false }: RoomMapProps) => {
  const { mode } = useColorScheme();
  const isDark = mode === "dark" || mode === "system" || !mode;
  const colorMode = mode === "system" || !mode ? "dark" : mode;
  const colors = ColorTokens[colorMode];

  const visitedRooms = useGameStore((s) => s.visitedRooms);
  const [canvasSize, setCanvasSize] = useState({ width: 400, height: 400 });
  const containerRef = useRef<HTMLDivElement>(null);

  // Use ResizeObserver so the canvas tracks both window resize and the
  // expand/collapse animation (which changes the container's width and height
  // as the flex-basis transitions from 25% → 100%).
  useEffect(() => {
    const el = containerRef.current;
    if (!el) return;
    const ro = new ResizeObserver((entries) => {
      for (const entry of entries) {
        const { width, height } = entry.contentRect;
        if (width > 0 && height > 0) {
          setCanvasSize({ width, height });
        }
      }
    });
    ro.observe(el);
    return () => ro.disconnect();
  }, []);

  return (
    // Stretch to fill all available vertical space in the parent flex column
    <Box sx={{ display: "flex", flexDirection: "column", gap: 1, width: "100%", flex: 1, minHeight: 0 }}>
      {/* Map canvas container — fills remaining height, no hardcoded px */}
      <Box
        ref={containerRef}
        id="map-container"
        sx={{
          border: isDark ? `2px solid ${DungeonColors.wall}` : "2px solid #8B6F47",
          borderRadius: "4px",
          backgroundColor: isDark ? DungeonColors.fog : "rgba(212, 197, 169, 0.5)",
          overflow: "hidden",
          boxShadow: isDark
            ? `inset 0 0 20px ${DungeonColors.fog}`
            : "inset 0 0 15px rgba(139, 111, 71, 0.2)",
          position: "relative",
          width: "100%",
          flex: 1,
          minHeight: 0,
        }}
      >
        {/* UI-FUT-1: Expand/collapse toggle */}
        {onExpand && (
          <Tooltip title={expanded ? "Collapse map" : "Expand map"}>
            <IconButton
              size="small"
              onClick={onExpand}
              sx={{
                position: "absolute",
                top: 8,
                right: 8,
                zIndex: 10,
                color: isDark ? "rgba(201, 169, 98, 0.7)" : "rgba(107, 86, 56, 0.7)",
                backgroundColor: isDark ? "rgba(26, 15, 30, 0.7)" : "rgba(212, 197, 169, 0.7)",
                backdropFilter: "blur(4px)",
                "&:hover": {
                  color: isDark ? "#FFD700" : "#5D4037",
                  backgroundColor: isDark ? "rgba(26, 15, 30, 0.9)" : "rgba(212, 197, 169, 0.9)",
                },
              }}
            >
              {expanded
                ? <ChevronLeftIcon sx={{ fontSize: "1.1rem" }} />
                : <ChevronRightIcon sx={{ fontSize: "1.1rem" }} />
              }
            </IconButton>
          </Tooltip>
        )}

        <MapCanvas
          gameState={gameState}
          visitedRooms={visitedRooms}
          width={canvasSize.width}
          height={canvasSize.height}
          onRoomFocus={onRoomFocus}
        />
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
            <Box sx={{ width: 16, height: 16, background: "linear-gradient(180deg, #C9A962 0%, #8B7355 100%)", border: "2px solid #FFD700", borderRadius: "3px", boxShadow: "0 0 8px rgba(255, 215, 0, 0.5)" }} />
            <Typography variant="caption" sx={{ color: colors.text.secondary, fontFamily: "Crimson Text, Georgia, serif" }}>Current</Typography>
          </Box>
          <Box sx={{ display: "flex", alignItems: "center", gap: 0.5 }}>
            <Box sx={{ width: 16, height: 16, background: "linear-gradient(180deg, #1a6b6e 0%, #0f4547 100%)", border: "2px solid #4dd0d4", borderRadius: "3px" }} />
            <Typography variant="caption" sx={{ color: colors.text.secondary, fontFamily: "Crimson Text, Georgia, serif" }}>Exit</Typography>
          </Box>
          <Box sx={{ display: "flex", alignItems: "center", gap: 0.5 }}>
            <Box sx={{ width: 16, height: 16, background: "linear-gradient(180deg, #6B4E9D 0%, #4a3570 100%)", border: "2px solid #9575CD", borderRadius: "3px" }} />
            <Typography variant="caption" sx={{ color: colors.text.secondary, fontFamily: "Crimson Text, Georgia, serif" }}>Adjacent</Typography>
          </Box>
          <Box sx={{ display: "flex", alignItems: "center", gap: 0.5 }}>
            <Box sx={{ width: 16, height: 16, background: "linear-gradient(180deg, #5D4037 0%, #3E2723 100%)", border: "2px solid #6D4C41", borderRadius: "3px" }} />
            <Typography variant="caption" sx={{ color: colors.text.secondary, fontFamily: "Crimson Text, Georgia, serif" }}>Explored</Typography>
          </Box>
          <Box sx={{ display: "flex", alignItems: "center", gap: 0.5 }}>
            <Typography variant="caption" sx={{ color: colors.accent, fontWeight: "bold", fontSize: "1rem" }}>↑↓</Typography>
            <Typography variant="caption" sx={{ color: colors.text.secondary, fontFamily: "Crimson Text, Georgia, serif" }}>Stairs</Typography>
          </Box>
          <Box sx={{ display: "flex", alignItems: "center", gap: 0.5 }}>
            <DotCircle size={10} color="#4CAF50" />
            <Typography variant="caption" sx={{ color: colors.text.secondary, fontFamily: "Crimson Text, Georgia, serif" }}>Friendly</Typography>
          </Box>
          <Box sx={{ display: "flex", alignItems: "center", gap: 0.5 }}>
            <DotCircle size={10} color="#F44336" />
            <Typography variant="caption" sx={{ color: colors.text.secondary, fontFamily: "Crimson Text, Georgia, serif" }}>Hostile</Typography>
          </Box>
        </Stack>
      </Box>

    </Box>
  );
};

// Tiny helper so the legend can render colored dots without Konva
const DotCircle = ({ size, color }: { size: number; color: string }) => (
  <Box
    sx={{
      width: size,
      height: size,
      borderRadius: "50%",
      backgroundColor: color,
      border: `1.5px solid ${color === "#4CAF50" ? "#81C784" : "#EF9A9A"}`,
      flexShrink: 0,
    }}
  />
);
