Current TODO:
- fix narrative saving to dynamoDB
- add in profile page
- fix up the game map
- improve the loading state between client and server
	- world ready is spammed at start to confirm creation
		- need to limit properties returned from object to just ready
	- on a load of an existing game verify if client's local cache of game is == to server's data
	- fetch ALL info from the server to port over the game save again
		- this means we need to be able to parse the chat history in some way



Below is mostly done
### Overhaul
- changes game model. Session state needs to be saved on cloud and loaded in on a request. The entire conversation history etc should be a single row in the database
- There are 2 patterns of game access:
	- User logs into application and wants to resume the game state
	- User sends chat and the game state should progress
- The chat route handler should load the associate game session from the app and then send/recieve a chat response with gemini.
- Stop using the gemini chat utility, store all context in the game history log and feed it in after every question.