## Issues
- Game Engine is more of a loose idea than an engine
	- Only does partial state management
	- Object location management was botched
	- The world representation I think might be a circular mess
	- The item creation, generation, etc, api's are a bit of a mess
	- using JSON tokens wastes AI context

- The AI currently is doing everything
	- World Architecture
	- Narrative
	- World management

- The map looks really lame
- While the AI is generating the game there is nothing happening
- The game is mostly reading nothing visual happening

## Next Steps
 #GameEngine
 - Refine Game engine
	 - Game engine should comprehensively manage the state of the game world
	 - Should allow and have a combat system
		 - Will need ways of engaging multiple enemies
	- Items need to have a better way of being located - not tied to unique names
	- API's for creating items, characters, etc overhaul
		- need better interface for creating and moving things in the world
		- need to be able to view the state of the world in a better way.
		- need to be able to save information on things
	- leave in JSON for now
#AIArchitecture
- Divide the AI's responsiblities
	- Have dedicated chat threads for world creation, game management, and narrative.
	