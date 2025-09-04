
import { Button, Typography, Box } from '@mui/material';
import { GameState } from '../types/types';

export const GameInfo = ({ gameState, onItemClick }: { gameState: GameState, onItemClick: (item: string) => void }) => {
	const currentRoom = gameState.rooms[gameState.current_room];

	return (
		<Box sx={{ p: 2 }}>
			<Typography variant="h6">Current Location: {gameState.current_room}</Typography>
			<Typography variant="body1">{currentRoom?.description || 'No description available'}</Typography>

			<Typography variant="h6" sx={{ mt: 2 }}>Your Inventory:</Typography>
			{gameState.inventory && gameState.inventory.length > 0 ? (
				<ul>
					{gameState.inventory.map(item => (
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
				<Typography>Your inventory is empty</Typography>
			)}

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
