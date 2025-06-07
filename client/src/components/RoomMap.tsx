import { Fragment, useState } from "react";
import { GameState } from "../models";
import { calculateRoomPositions } from "./calcPosition";
import { Stage, Layer, Circle, Line, Text, Rect } from 'react-konva';

// RoomMap Component
export const RoomMap = ({ gameState }: { gameState: GameState }) => {
  const stageWidth = 600;
  const stageHeight = 500;
  const roomRadius = 20;
  const playerIconRadius = 8;
  const [hoveredRoom, setHoveredRoom] = useState<string | null>(null);
  const [tooltip, setTooltip] = useState<{ x: number; y: number; text: string } | null>(null);

  // Calculate room positions using force-directed layout
  const roomPositions = calculateRoomPositions(gameState.rooms);

  const handleRoomHover = (roomId: string, pos: { x: number; y: number }) => {
    setHoveredRoom(roomId);
    const isAdjacent = gameState.rooms[gameState.current_room].connections.includes(roomId);
    const isCurrent = roomId === gameState.current_room;

    let tooltipText = roomId;
    if (isCurrent) {
      tooltipText = "You are here";
    } else if (isAdjacent) {
      tooltipText = `Connected to ${roomId}`;
    } else {
      tooltipText = "This room is not accessible from your current location";
    }

    setTooltip({ x: pos.x, y: pos.y - 40, text: tooltipText });
  };

  const handleRoomLeave = () => {
    setHoveredRoom(null);
    setTooltip(null);
  };

  return (
    <Stage width={stageWidth} height={stageHeight}>
      <Layer>
        {/* Draw connections */}
        {Object.entries(gameState.rooms).map(([roomId, room]) =>
          room.connections.map(connId => {
            const start = roomPositions[roomId];
            const end = roomPositions[connId];
            if (!start || !end) return null;

            const dx = end.x - start.x;
            const dy = end.y - start.y;
            const angle = Math.atan2(dy, dx);

            const startX = start.x + Math.cos(angle) * roomRadius;
            const startY = start.y + Math.sin(angle) * roomRadius;
            const endX = end.x - Math.cos(angle) * roomRadius;
            const endY = end.y - Math.sin(angle) * roomRadius;

            const isCurrentRoomConnection =
              (roomId === gameState.current_room || connId === gameState.current_room);

            return (
              <Line
                key={`${roomId}-${connId}`}
                points={[startX, startY, endX, endY]}
                stroke={isCurrentRoomConnection ? "#4CAF50" : "#666"}
                strokeWidth={isCurrentRoomConnection ? 4 : 3}
                dash={[5, 5]}
                opacity={isCurrentRoomConnection ? 1 : 0.8}
              />
            );
          })
        )}

        {/* Draw rooms */}
        {Object.entries(gameState.rooms).map(([roomId, room]) => {
          const pos = roomPositions[roomId];
          if (!pos) return null;
          const isCurrentRoom = roomId === gameState.current_room;
          const isConnectedToCurrent = room.connections.includes(gameState.current_room);
          const isAdjacent = isConnectedToCurrent || isCurrentRoom;

          return (
            <Fragment key={roomId}>
              <Circle
                x={pos.x}
                y={pos.y}
                radius={roomRadius}
                fill={isCurrentRoom ? "#4CAF50" : isConnectedToCurrent ? "#81C784" : "#2196F3"}
                stroke={isCurrentRoom ? "#81C784" : isConnectedToCurrent ? "#4CAF50" : "#64B5F6"}
                strokeWidth={isCurrentRoom ? 5 : isConnectedToCurrent ? 4 : 3}
                shadowColor="black"
                shadowBlur={isCurrentRoom ? 15 : isConnectedToCurrent ? 12 : 10}
                shadowOpacity={isCurrentRoom ? 0.5 : isConnectedToCurrent ? 0.4 : 0.3}
                onMouseEnter={() => handleRoomHover(roomId, pos)}
                onMouseLeave={handleRoomLeave}
                opacity={isAdjacent ? 1 : 0.5}
              />
              {/* Add player icon in current room */}
              {isCurrentRoom && (
                <Circle
                  x={pos.x}
                  y={pos.y}
                  radius={playerIconRadius}
                  fill="#FFD700"
                  stroke="#000"
                  strokeWidth={2}
                  shadowColor="black"
                  shadowBlur={10}
                  shadowOpacity={0.5}
                />
              )}
            </Fragment>
          );
        })}

        {/* Room labels - drawn on top of everything */}
        {Object.entries(gameState.rooms).map(([roomId, room]) => {
          const pos = roomPositions[roomId];
          if (!pos) return null;
          const isHovered = hoveredRoom === roomId;
          const isAdjacent = room.connections.includes(gameState.current_room) ||
            roomId === gameState.current_room;

          // Format room name: replace underscores with spaces and title case
          const formattedName = roomId
            .split('_')
            .map(word => word.charAt(0).toUpperCase() + word.slice(1).toLowerCase())
            .join(' ');

          return (
            <Fragment key={`label-${roomId}`}>
              {/* Label background */}
              <Rect
                x={pos.x - 70}
                y={pos.y + roomRadius + 2}
                width={140}
                height={20}
                fill={isHovered ? "#424242" : "#2D2D2D"}
                cornerRadius={4}
                shadowColor="black"
                shadowBlur={5}
                shadowOpacity={0.3}
                opacity={isAdjacent ? 1 : 0.5}
              />
              {/* Label text */}
              <Text
                x={pos.x - 65}
                y={pos.y + roomRadius + 5}
                text={formattedName}
                fontSize={12}
                fill={isHovered ? "#FFFFFF" : "#E0E0E0"}
                align="center"
                width={130}
                opacity={isAdjacent ? 1 : 0.5}
              />
            </Fragment>
          );
        })}

        {/* Tooltip */}
        {tooltip && (
          <Fragment>
            <Rect
              x={tooltip.x - 100}
              y={tooltip.y - 25}
              width={200}
              height={30}
              fill="#2D2D2D"
              cornerRadius={4}
              shadowColor="black"
              shadowBlur={5}
              shadowOpacity={0.3}
            />
            <Text
              x={tooltip.x - 95}
              y={tooltip.y - 15}
              text={tooltip.text}
              fontSize={12}
              fill="#E0E0E0"
              align="center"
              width={190}
            />
          </Fragment>
        )}
      </Layer>
    </Stage>
  );
};
