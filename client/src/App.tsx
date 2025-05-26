import './App.css'
import React, { useEffect, useState, useRef } from 'react';
import axios from 'axios';
import { Button, TextField, Typography, Box, Paper, CircularProgress, Alert, Grid } from '@mui/material';
import { Stage, Layer, Circle, Line, Text, Rect } from 'react-konva';

const APP_URI = import.meta.env.VITE_APP_URI || 'http://localhost:8080/';
const MAX_WAIT_TIME = 60000; // 1 minute in milliseconds
const INITIAL_BACKOFF = 1000; // Start with 1 second
const MAX_BACKOFF = 8000; // Maximum backoff of 8 seconds

interface RoomInfo {
  id: string;
  description: string;
  connections: string[];
  items: string[];
  occupants: string[];
}

interface GameState {
  description: string;
  inventory: string[];
  health: number;
  position: { x: number; y: number };
  current_room: string;
  rooms: { [key: string]: RoomInfo };
}

interface ChatMessage {
  type: 'player' | 'narrative';
  content: string;
}

// Force-directed graph layout algorithm
const calculateRoomPositions = (rooms: { [key: string]: RoomInfo }, currentRoom: string) => {
  const positions: { [key: string]: { x: number; y: number } } = {};
  const width = 600;
  const height = 500;
  const centerX = width / 2;
  const centerY = height / 2;
  const maxRadius = Math.min(width, height) * 0.35; // Slightly reduced from 0.4

  // Initialize positions in a circle
  const roomIds = Object.keys(rooms);
  roomIds.forEach((roomId, index) => {
    const angle = (index / roomIds.length) * 2 * Math.PI;
    positions[roomId] = {
      x: centerX + Math.cos(angle) * maxRadius,
      y: centerY + Math.sin(angle) * maxRadius
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
    roomIds.forEach(roomId => {
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
    roomIds.forEach(roomId => {
      const connections = rooms[roomId].connections;
      connections.forEach(connId => {
        const pos1 = positions[roomId];
        const pos2 = positions[connId];
        const dx = pos2.x - pos1.x;
        const dy = pos2.y - pos1.y;
        const distance = Math.sqrt(dx * dx + dy * dy);
        
        if (distance > 0) {
          const force = (distance - targetDistance) * attraction;
          const fx = (dx / distance) * force;
          const fy = (dy / distance) * force;

          forces[roomId].x += fx;
          forces[roomId].y += fy;
          forces[connId].x -= fx;
          forces[connId].y -= fy;
        }
      });
    });

    // Apply forces with damping
    roomIds.forEach(roomId => {
      positions[roomId].x += forces[roomId].x * damping;
      positions[roomId].y += forces[roomId].y * damping;

      // Keep nodes within bounds with more padding
      positions[roomId].x = Math.max(70, Math.min(width - 70, positions[roomId].x));
      positions[roomId].y = Math.max(70, Math.min(height - 70, positions[roomId].y));
    });
  }

  return positions;
};

// RoomMap Component
const RoomMap = ({ gameState, onRoomClick }: { gameState: GameState, onRoomClick: (roomId: string) => void }) => {
  const stageWidth = 600;
  const stageHeight = 500;
  const roomRadius = 20;
  const playerIconRadius = 8;
  const [hoveredRoom, setHoveredRoom] = useState<string | null>(null);
  const [tooltip, setTooltip] = useState<{ x: number; y: number; text: string } | null>(null);

  // Calculate room positions using force-directed layout
  const roomPositions = calculateRoomPositions(gameState.rooms, gameState.current_room);

  const handleRoomHover = (roomId: string, pos: { x: number; y: number }) => {
    setHoveredRoom(roomId);
    const isAdjacent = gameState.rooms[gameState.current_room].connections.includes(roomId);
    const isCurrent = roomId === gameState.current_room;
    
    let tooltipText = roomId;
    if (isCurrent) {
      tooltipText = "You are here";
    } else if (isAdjacent) {
      tooltipText = `Click to move to ${roomId}`;
    } else {
      tooltipText = "This room is not accessible from your current location";
    }
    
    setTooltip({ x: pos.x, y: pos.y - 40, text: tooltipText });
  };

  const handleRoomLeave = () => {
    setHoveredRoom(null);
    setTooltip(null);
  };

  const handleRoomClick = (roomId: string) => {
    const isAdjacent = gameState.rooms[gameState.current_room].connections.includes(roomId);
    if (isAdjacent) {
      onRoomClick(roomId);
    }
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
            <React.Fragment key={roomId}>
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
                onClick={() => handleRoomClick(roomId)}
                onMouseEnter={() => handleRoomHover(roomId, pos)}
                onMouseLeave={handleRoomLeave}
                opacity={isAdjacent ? 1 : 0.5}
                cursor={isAdjacent ? "pointer" : "not-allowed"}
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
            </React.Fragment>
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
            <React.Fragment key={`label-${roomId}`}>
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
            </React.Fragment>
          );
        })}

        {/* Tooltip */}
        {tooltip && (
          <React.Fragment>
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
          </React.Fragment>
        )}
      </Layer>
    </Stage>
  );
};

// GameInfo Component
const GameInfo = ({ gameState, onItemClick }: { gameState: GameState, onItemClick: (item: string) => void }) => {
  const currentRoom = gameState.rooms[gameState.current_room];
  
  return (
    <Box sx={{ p: 2 }}>
      <Typography variant="h6">Current Location: {gameState.current_room}</Typography>
      <Typography variant="body1">{currentRoom?.description || 'No description available'}</Typography>
      
      <Typography variant="h6" sx={{ mt: 2 }}>Items in Room:</Typography>
      {currentRoom?.items && currentRoom.items.length > 0 ? (
        <ul>
          {currentRoom.items.map(item => (
            <li key={item}>
              <Button 
                onClick={() => onItemClick(item)}
                sx={{ textTransform: 'none', color: '#2196F3' }}
              >
                {item}
              </Button>
            </li>
          ))}
        </ul>
      ) : (
        <Typography>No items in this room</Typography>
      )}
      
      <Typography variant="h6" sx={{ mt: 2 }}>Occupants:</Typography>
      {currentRoom?.occupants && currentRoom.occupants.length > 0 ? (
        <ul>
          {currentRoom.occupants.map(occupant => (
            <li key={occupant}>{occupant}</li>
          ))}
        </ul>
      ) : (
        <Typography>No occupants in this room</Typography>
      )}
    </Box>
  );
};

// Chat Message Component
const ChatMessage = ({ message }: { message: ChatMessage }) => {
  const isPlayer = message.type === 'player';
  return (
    <Box
      sx={{
        display: 'flex',
        justifyContent: isPlayer ? 'flex-end' : 'flex-start',
        mb: 2,
      }}
    >
      <Paper
        sx={{
          p: 2,
          maxWidth: '70%',
          backgroundColor: isPlayer ? '#2196F3' : '#424242',
          color: isPlayer ? 'white' : '#E0E0E0',
          borderRadius: 2,
          boxShadow: 1,
        }}
      >
        <Typography variant="body1">{message.content}</Typography>
      </Paper>
    </Box>
  );
};

// Chat Component
const Chat = ({ 
  chatHistory, 
  command, 
  setCommand, 
  handleCommand, 
  isLoading 
}: { 
  chatHistory: ChatMessage[], 
  command: string, 
  setCommand: (cmd: string) => void, 
  handleCommand: () => void,
  isLoading: boolean
}) => {
  const chatContainerRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (chatContainerRef.current) {
      chatContainerRef.current.scrollTop = chatContainerRef.current.scrollHeight;
    }
  }, [chatHistory]);

  return (
    <Box sx={{ height: '100%', display: 'flex', flexDirection: 'column' }}>
      <Box
        ref={chatContainerRef}
        sx={{
          flex: 1,
          overflow: 'auto',
          p: 2,
          mb: 2,
          backgroundColor: '#1E1E1E',
          '&::-webkit-scrollbar': {
            width: '8px',
          },
          '&::-webkit-scrollbar-track': {
            background: '#2D2D2D',
            borderRadius: '4px',
          },
          '&::-webkit-scrollbar-thumb': {
            background: '#424242',
            borderRadius: '4px',
          },
        }}
      >
        {chatHistory.map((msg, index) => (
          <ChatMessage key={index} message={msg} />
        ))}
      </Box>
      
      <Box sx={{ 
        display: 'flex', 
        gap: 1, 
        p: 2, 
        borderTop: 1, 
        borderColor: 'divider',
        backgroundColor: '#2D2D2D'
      }}>
        <TextField
          fullWidth
          value={command}
          onChange={(e) => setCommand(e.target.value)}
          onKeyPress={(e) => e.key === 'Enter' && handleCommand()}
          placeholder="Type your command..."
          disabled={isLoading}
          size="small"
          sx={{
            '& .MuiOutlinedInput-root': {
              backgroundColor: '#424242',
              '& fieldset': {
                borderColor: '#666',
              },
              '&:hover fieldset': {
                borderColor: '#888',
              },
              '&.Mui-focused fieldset': {
                borderColor: '#2196F3',
              },
            },
            '& .MuiInputBase-input': {
              color: '#E0E0E0',
            },
            '& .MuiInputLabel-root': {
              color: '#888',
            },
          }}
        />
        <Button
          variant="contained"
          onClick={handleCommand}
          disabled={isLoading || !command.trim()}
          sx={{ minWidth: '100px' }}
        >
          {isLoading ? <CircularProgress size={24} /> : 'Send'}
        </Button>
      </Box>
    </Box>
  );
};

function App() {
  const [gameState, setGameState] = useState<GameState | null>(null);
  const [command, setCommand] = useState('');
  const [gameStarted, setGameStarted] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [worldReady, setWorldReady] = useState(false);
  const [pollingStatus, setPollingStatus] = useState<string>('');
  const [chatHistory, setChatHistory] = useState<ChatMessage[]>([]);

  const checkWorldReady = async () => {
    try {
      const response = await axios.get(`${APP_URI}worldready`);
      if (response.data.ready) {
        setWorldReady(true);
        return true;
      }
      return false;
    } catch (err) {
      console.error('Error checking world status:', err);
      return false;
    }
  };

  const sleep = (ms: number) => new Promise(resolve => setTimeout(resolve, ms));

  const pollWorldStatus = async () => {
    const startTime = Date.now();
    let backoff = INITIAL_BACKOFF;
    let attempts = 0;

    while (Date.now() - startTime < MAX_WAIT_TIME) {
      attempts++;
      setPollingStatus(`Checking world status (attempt ${attempts})...`);
      
      const isReady = await checkWorldReady();
      if (isReady) {
        setPollingStatus('World generation complete!');
        return true;
      }

      // Calculate next backoff time
      const nextBackoff = Math.min(backoff * 1.5, MAX_BACKOFF);
      const remainingTime = MAX_WAIT_TIME - (Date.now() - startTime);
      const actualBackoff = Math.min(nextBackoff, remainingTime);

      if (actualBackoff <= 0) {
        break;
      }

      setPollingStatus(`Waiting ${Math.round(actualBackoff / 1000)} seconds before next check...`);
      await sleep(actualBackoff);
      backoff = nextBackoff;
    }

    setPollingStatus('World generation timeout');
    return false;
  };

  const startGame = async () => {
    setIsLoading(true);
    setError(null);
    setPollingStatus('Starting game...');
    
    try {
      const response = await axios.post(`${APP_URI}startgame`);
      setGameStarted(true);
      
      // Poll for world generation completion
      const worldGenerated = await pollWorldStatus();
      if (!worldGenerated) {
        setError('World generation is taking longer than expected. You can still try to play, but some features might not be available yet.');
      }
      
      // Get initial game state
      setPollingStatus('Loading initial game state...');
      const gameResponse = await axios.get(`${APP_URI}describe`);
      setGameState(gameResponse.data);

      // Get initial narrative
      const narrativeResponse = await axios.post(`${APP_URI}chat`, { chat: "Please provide an introductory narrative for the player." });
      if (narrativeResponse.data && narrativeResponse.data.Response) {
        setChatHistory(prev => [...prev, { type: 'narrative', content: narrativeResponse.data.Response }]);
      }
    } catch (err) {
      setError('Failed to start game. Please check if the server is running and try again.');
      console.error('Error starting game:', err);
    } finally {
      setIsLoading(false);
      setPollingStatus('');
    }
  };

  const handleCommand = async () => {
    if (!command.trim()) return;
    
    setIsLoading(true);
    setError(null);

    try {
      setChatHistory(prev => [...prev, { type: 'player', content: command }]);

      const response = await axios.post(`${APP_URI}chat`, { chat: command });
      
      if (response.data && response.data.Response) {
        setChatHistory(prev => [...prev, { type: 'narrative', content: response.data.Response }]);
      } else {
        console.error('Invalid response format:', response.data);
        setError('Received invalid response from server');
      }

      const gameResponse = await axios.get(`${APP_URI}describe`);
      setGameState(gameResponse.data);
      
      setCommand('');
    } catch (err) {
      setError('Failed to process command. Please try again.');
      console.error('Error processing command:', err);
    } finally {
      setIsLoading(false);
    }
  };

  const handleItemClick = async (item: string) => {
    if (isLoading || !gameState) return;
    
    setIsLoading(true);
    setError(null);

    try {
      // Add player action message
      setChatHistory(prev => [...prev, { type: 'player', content: `Take ${item}` }]);

      // Send item interaction to server
      const response = await axios.post(`${APP_URI}chat`, { chat: `Take ${item}` });
      
      if (response.data && response.data.Response) {
        setChatHistory(prev => [...prev, { type: 'narrative', content: response.data.Response }]);
      }

      // Update game state
      const gameResponse = await axios.get(`${APP_URI}describe`);
      setGameState(gameResponse.data);
    } catch (err) {
      setError('Failed to interact with item. Please try again.');
      console.error('Error interacting with item:', err);
    } finally {
      setIsLoading(false);
    }
  };

  const handleRoomClick = async (roomId: string) => {
    if (isLoading || !gameState) return;
    
    setIsLoading(true);
    setError(null);

    try {
      // Add player movement message
      setChatHistory(prev => [...prev, { type: 'player', content: `Moving to ${roomId}` }]);

      const moveResponse = await axios.post(`${APP_URI}move`, { room_id: roomId });
      
      // Add the narrative response to chat history
      if (moveResponse.data && moveResponse.data.Response) {
        setChatHistory(prev => [...prev, { type: 'narrative', content: moveResponse.data.Response }]);
      }

      // Check if the server generated new areas
      if (moveResponse.data && moveResponse.data.NewAreas) {
        // Update game state with new areas
        setGameState(prev => ({
          ...prev!,
          rooms: {
            ...prev!.rooms,
            ...moveResponse.data.NewAreas
          }
        }));
      } else {
        // Regular game state update
        const gameResponse = await axios.get(`${APP_URI}describe`);
        setGameState(gameResponse.data);
      }
    } catch (err) {
      setError('Failed to move to room. Please try again.');
      console.error('Error moving to room:', err);
    } finally {
      setIsLoading(false);
    }
  };

  if (!gameStarted) {
    return (
      <Box sx={{ p: 4, textAlign: 'center' }}>
        <Typography variant="h4" sx={{ mb: 4 }}>Text Adventure Game</Typography>
        <Button
          variant="contained"
          onClick={startGame}
          disabled={isLoading}
        >
          {isLoading ? <CircularProgress size={24} /> : 'Start Game'}
        </Button>
        {pollingStatus && (
          <Typography sx={{ mt: 2 }}>{pollingStatus}</Typography>
        )}
        {error && (
          <Alert severity="error" sx={{ mt: 2 }}>{error}</Alert>
        )}
      </Box>
    );
  }

  if (!gameState) {
    return (
      <Box sx={{ p: 4, textAlign: 'center' }}>
        <CircularProgress />
        <Typography sx={{ mt: 2 }}>Loading game state...</Typography>
      </Box>
    );
  }

  return (
    <Box sx={{ height: '100vh', display: 'flex', flexDirection: 'column', overflow: 'hidden', backgroundColor: '#1E1E1E' }}>
      {/* Top section - Game Info and Map */}
      <Box sx={{ flex: '0 0 auto', p: 2 }}>
        <Paper sx={{ p: 2, backgroundColor: '#2D2D2D' }}>
          <Grid container spacing={2}>
            <Grid item xs={8}>
              <Box sx={{ height: '500px', display: 'flex', justifyContent: 'center', alignItems: 'center' }}>
                <RoomMap gameState={gameState} onRoomClick={handleRoomClick} />
              </Box>
            </Grid>
            <Grid item xs={4}>
              <GameInfo gameState={gameState} onItemClick={handleItemClick} />
            </Grid>
          </Grid>
        </Paper>
      </Box>

      {/* Bottom section - Chat */}
      <Box sx={{ flex: '1 1 auto', p: 2, minHeight: 0 }}>
        <Paper sx={{ height: '100%', backgroundColor: '#2D2D2D' }}>
          <Chat
            chatHistory={chatHistory}
            command={command}
            setCommand={setCommand}
            handleCommand={handleCommand}
            isLoading={isLoading}
          />
        </Paper>
      </Box>

      {error && (
        <Alert severity="error" sx={{ mt: 2, mx: 2 }}>{error}</Alert>
      )}
    </Box>
  );
}

export default App;
