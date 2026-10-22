### Work so far
- Basic maze generation algorithm running server side, keeping maze in memory of a single process
- Client will receive limited data from the server related to information regarding the maze and also send requests to the server to move the player
- Concept of rooms/hallways/doors to enable traversal.
- Click interaction client side in order to move the player

### Ideas for Progress
1. Switch over to AI for controlling movement within the maze and only show the maze object as a static minimap
	1. Need to add a chat box and text output where the AI well tell the player information
	2. Need to give the AI knowledge about the maze, what kind of information they have available to them etc.
	3. I think I need to set up an mcp server so the AI can control the server? This might be totally overkill.
	4. Client sends chat to AI -> AI interprets chat and decides what action to take:
		- Move the player object to a new location
		- (Nothing else really possible)
2. Make the game more interesting
	1. Give player an inventory and populate the maze with items
	2. Introduce a combat element to the game
	3. Introduce locking areas and requiring solving puzzles for progression
3. Improve UI
	1. Better chatboxes
	2. styling minimap
	3. adding sprites etc.