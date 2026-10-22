#AIArchitecture 
## Responsibilities
- World Generation
	- Creating the world with all of it's rooms
	- Connecting Rooms
	- Creating Items for the world
	- Creating Characters for the world
	- Assigning all of these items and characters to locations
	- connecting rooms together
- Narrative
	- responding the the player with a cohesive narrative
	- routinely compress the narrative so the AI can 
- World Management
	- Moving characters to and from locations
	- Moving/Creating Items in response to narrative
	- Creating new rooms and connecting them as needed
	- Removing world resources
- Task orchestration/delegation
	- Need to control and synchronize actions across agents

## Plan
We can delegate each of these task categories to different AI, predominantly driven by the narrative AI. The world generation and management AI would be the same AI, but they would not have their context saved. The narrative AI would get a default SYSTEM level message letting it know the current world state? or we could provide it with tools to call and check state etc.  



