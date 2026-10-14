### Process
- Will need to give the AI the ability to call a tool and then also provide a response afterwards so it can give more context to the user or just respond to the user

### Map
- Only show the player their immediate surroundings? focus on a 20x20 grid on the UI side for the player to understand their immediate surrounds?

### Tools
- Move
	- Simple tool, moves the player to a location the AI specifies
- Describe
	- Gives the AI context on what the player currently sees

### Issues
- Describe
	- I think the AI is getting too granular of information about the coordinates to make a good decision. Maybe the AI would do better with simply an asci representation or a more focused summary?
- Movement
	- Using the AI to navigate an actual maze seems pretty miserable to be frank. I want the AI optimistically to focus on narrative and adding flavor to the game. Having it move the player probably isn't the right move. Maybe letting it spawn in items and influence the game world by creating enemies etc would be better?
