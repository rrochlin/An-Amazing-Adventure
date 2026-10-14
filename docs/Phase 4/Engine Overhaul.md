There are a couple different things we need to juggle for this project
- Need a game engine state machine that can load in an arbitrary state then export the state back to cloud
- Need a set of tool that the LLM can use in order to affect the game world in a meaningful way
- The game engine will need to support features that enable the LLM to control the world

The main "Engine" for this application should be focused on orchestration of the LLM agents, and should use a secondary subengine that is tailor made for the game itself

## What makes a Game Engine?
- The game engine will take in a game object and manipulate said object. The game object represents the state of the game. Depending on the game state that is how certain actions will and won't occur.
1. Locations
	1. A location would be anything that a character is able to occupy the space of
	2. Items can also be "placed" into locations or in a player/characters inventory?
2. Items
	1. Game world objects. They can be at a location or with a player. 
3. Characters
	1. NPC's the player character, etc.
	2. They can have items, always are in a location, and have attributes etc
4. Abilities
	1. Optional


How to start?  A simple combat item system? maybe just make a console game application with these kinds of objects in them. 
Central Class of Entity, where all things are essentially an entity?

We don't really need to draw these things, only destroy them so it's not that big of a deal.

## How do we load a game state
Since this application is going to be running as a REST api server we want to asynchronously handle requests we get to the server and act on those. This means that every request we get will spin up and destroy a game every time. We will want to design this in a way that we could persist games in memory that are "active". Of course this would involve a lot of complexity if we scale that beyond a single server operation so meh.

Mainly we need to instantiate something, could be a game object on the request being received, if the game state even needs to be mutated?
Could just wait for a game state edit request in that same co_routine to actually act on something then go ahead and fetch the game object then, the larger part of the delay is the LLM call, although the read cost really isn't an issue at this time so whatever.

## The Game Engine
