### Leading Theory
Using MCP to control movement is not a good idea. We should leverage the mcp to control things within the game world but not in doing movement. We should procedurally generate the world and then use AI to help create dynamic events and generate player narrative. 

I could also create a general game engine. This would have functions for like creating areas, putting enemies in areas, putting items in areas, and managing a character inventory. The area could then persist and we could let the player access it.

We can still use a predominantly text based interaction system, what are the fundamental components we want to have?

In General Create a game creator and let the AI create the game using MCP


### Improve the UI
- Stream responses back
- Better Inventory/Map interaction
- More consistent map

### What not to have the AI do
- Provide precise arguments/inputs
	- move to coord x,y
	- deal x damage
	- give x gold
	- reward x experience

### What does that imply the game engine will need to do then?
- provide an interface to move the player throughout the world
- explain the world to the AI in a way that makes sense to it
	- pictures?
	- bitmap?
	- lists of rooms/dimensions
- handle combat
- 

- World
	- connected group of rooms?
	- Towns/Instances?
- Enemies
- Player


### How to start?
- Start with the world and a player that can move throughout it
- use nodes to represent the world and save the qualities of the rooms etc in the nodes, then make a network to represent the game
	- each node is a valid location for a player to be
	- nodes are connected to at most 4 other nodes
	- these nodes are considered "adjacent"
	- the nodes have a narrative description that the AI will give them on initialization and then the AI can update the description if it wants to.