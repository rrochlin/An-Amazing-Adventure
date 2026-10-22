## Working Title: An Amazing Adventure
### Genre: Text Adventure

### Backend 
- Go Server holding the state of the game
- Simple React Frontend for the gameplay interface
- AI client layer to provide descriptions to the player of the world
- MCP interface for AI client to interact with and control the player inventory etc.
- Generate each game uniquely as a 2d maze with procedurally generated keys and doors
- Give the players a map they can record as they discover more of the game space, so they can visualize where they are
- Give them an inventory screen
- Have the AI act as the dungeon master and assess the validity of their actions with items


# Keep it Simple
- Set your sights low for the first pass
- Just focus on the basic gameplay, maybe use a static maze or even nothing to start with
- can use whatever language you want really

### Project Outline:
- Research MCP to figure out what to do here
- Create base game client in Go
	- Inventory
		- Keys
		- Other items for interacting in the maze
	- Player Object
	- Maze to navigate
		- Walls that limit movement
		- Doors that require keys
		- Keys located around the maze
- Frontend Client
	- Interface to issue commands
		- can search a room
		- can traverse the maze
		- can pick up objects
	- Map of the maze with fog of war
		- Show player position
		- Shows doors if discovered
	- Text streamed from the AI describing things
- MCP Api for the AI to interact with the world?
	- Need to get the base game first before I integrate this I think
	- Without MCP the AI can still describe things etc
	- Don't need MCP to know that until a user "searches" don't display the option to loot key etc.

### Maze Generation Algorithm
- Need to generate rooms and hallways to connect the rooms and then turn that into a 2D grid
- Generating this grid directly from the "pixel" level might prove difficult. Making the sure the rooms connect etc. won't work out well if I'm looking at the algorithm from that level of granularity. I would prefer to on a high level generate a configuration of rooms and connections between rooms then represent them in a 2D maze.
	- All rooms must be connected somehow
	- Paths cannot overlap
	- Rooms have a minimum size requirement
	- There must be multiple rooms in a level
	- Levels have a minimum size requirement
	- The Entrance must be a room: Coord 0, 0
	- The exit must have a hallway: Coord m, n
	- Hallways can branch and turn
- I think it will be difficult to generate a compact maze so I think going larger would be better.

 - Make a Large Maze
 - Randomly pick X points to be the basis for Rooms, also 0, 0 and m, n for the start and exit rooms
 - Merge any points within 3 units of each other to start
 - Grow the rooms until they are constrained by the maze boundary or within 3 units to another maze
	 - Voronoi Algorithm for filling in the space and then go back along boundaries and give them a thickness
 - There will now be 3 types of rooms
	 - Normal Rooms
	 - The Entrance Room
	 - The Exit Room
- All void space between the rooms: the 3 spaces that separate them will now be a common hallway that connects them with doors, the center of the space will be floor and the walls will remain.
- Rooms will get Doors that provide access to the hallway - 1 per wall adjacent to the hallway.
	- In the future doors can be locked and there can be barriers in the hallway.
- 1's represent a hallway and 0's a room
- To draw door I should determine a closest 

alright so we have the maze with doors now...
We should have to reconfigure the game to open doors and then travel through the hall.
Player should have 360 view of the hall that surrounds them?

There are rooms and there is the hall way. When in a room the player can see everything
When in the hallway the player can only see what is immediately in their sight.

Need to create an algorithm that will only show unobstructed squares for the users current position from the current position.

Two main problems
1. Running into out of bounds issues still in checkView
2. describeHall is giving way too much info
	1. The method of only checking the next block up and hoping that all the algorithms paths cancel first isn't working out
	2. I can keep the blocked map but I need to work my way back in the direction of the player and make sure that I do not cross a blocked block or a room block

Can make the frontend for the application better
Integrate AI for controlling movement


### Traversal
- Text Based
	- switch maze to minimap
	- have AI control the movements of the player
		- Just write an mcp server in golang
	- create doors/keys etc to make the maze harder to traverse
- UI Based Interactions
	- Switch maze to 3js
	- Keyboard?
	- Point/click?
	- Combat?
	- Highlight Points of interest in the local FOV and allow clicking to trigger a move action



## UI Improvements
Dice rolling
Interactive equipment
Characters displayed on the map
Zone maps
More focus on the map element






