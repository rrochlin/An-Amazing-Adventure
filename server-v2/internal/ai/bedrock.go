package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"
	"github.com/rrochlin/an-amazing-adventure/internal/game"
)

// Model IDs — must be cross-region inference profile IDs, not bare model IDs.
// Bare model IDs (without the "us." prefix) are rejected with ValidationException
// "Invocation with on-demand throughput isn't supported".
const (
	ModelNarrator = "us.anthropic.claude-sonnet-4-6"              // heavy — narrator & architect
	ModelSubAgent = "us.anthropic.claude-haiku-4-5-20251001-v1:0" // light — world-gen sub-agents
)

// Client wraps the Bedrock runtime client.
type Client struct {
	br *bedrockruntime.Client
}

// New creates a Client from the current AWS environment.
func New(ctx context.Context) (*Client, error) {
	region := os.Getenv("BEDROCK_REGION")
	if region == "" {
		region = "us-west-2"
	}
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}
	return &Client{br: bedrockruntime.NewFromConfig(cfg)}, nil
}

// ---- Token usage ----

// TokenUsage holds the Bedrock token counts from a single model call.
type TokenUsage struct {
	InputTokens  int
	OutputTokens int
}

// Total returns the sum of input and output tokens.
func (u TokenUsage) Total() int {
	return u.InputTokens + u.OutputTokens
}

// ---- Narrator (streaming chat) ----

// NarratorResult holds everything that comes back from a streaming narrator turn.
// The Narrator produces only prose — world mutations are handled by EngineerScan
// after the narrative stream completes.
type NarratorResult struct {
	Narrative   string                  // accumulated text sent to the player
	NewMessages []game.NarrativeMessage // updated history to persist
	Tokens      TokenUsage              // token usage for this turn
}

// EngineerResult holds the world mutations the Engineer inferred from the narrative.
type EngineerResult struct {
	Events    []game.WorldEvent    // player-visible world events
	Mutations []game.MutationEntry // audit log entries
	Tokens    TokenUsage           // token usage for the Engineer call
}

// NarrateStream runs a single narrator turn with streaming.
// The Narrator has NO tools — it produces pure immersive prose only.
// World mutations are applied separately by EngineerScan after streaming completes.
// onChunk is called for each text delta so ws-chat can push narrative_chunk frames
// immediately without buffering.
func (c *Client) NarrateStream(
	ctx context.Context,
	g *game.Game,
	history []game.NarrativeMessage,
	playerInput string,
	onChunk func(string),
) (NarratorResult, error) {
	// Trim history if it has grown too long to avoid context window exhaustion.
	trimmed, err := c.TrimHistory(ctx, history)
	if err != nil {
		log.Printf("NarrateStream: TrimHistory error (proceeding with full history): %v", err)
		trimmed = history
	}
	messages := toBedrockMessages(trimmed)

	// Append the new player turn
	messages = append(messages, types.Message{
		Role:    types.ConversationRoleUser,
		Content: []types.ContentBlock{&types.ContentBlockMemberText{Value: playerInput}},
	})

	systemPrompt := narratorSystemPrompt(g)

	// Single streaming call — Narrator never calls tools so there is no agentic loop.
	resp, err := c.br.ConverseStream(ctx, &bedrockruntime.ConverseStreamInput{
		ModelId:  aws.String(ModelNarrator),
		System:   []types.SystemContentBlock{&types.SystemContentBlockMemberText{Value: systemPrompt}},
		Messages: messages,
		// No ToolConfig — Narrator is prose-only by construction.
		InferenceConfig: &types.InferenceConfiguration{
			MaxTokens:   aws.Int32(4096),
			Temperature: aws.Float32(0.7),
		},
	})
	if err != nil {
		return NarratorResult{}, fmt.Errorf("converse stream: %w", err)
	}

	var fullNarrative strings.Builder
	var assistantText strings.Builder
	var totalTokens TokenUsage

	stream := resp.GetStream()
	for event := range stream.Events() {
		switch e := event.(type) {
		case *types.ConverseStreamOutputMemberContentBlockDelta:
			if d, ok := e.Value.Delta.(*types.ContentBlockDeltaMemberText); ok {
				fullNarrative.WriteString(d.Value)
				assistantText.WriteString(d.Value)
				if onChunk != nil {
					onChunk(d.Value)
				}
			}
		case *types.ConverseStreamOutputMemberMetadata:
			if e.Value.Usage != nil {
				totalTokens.InputTokens += int(aws.ToInt32(e.Value.Usage.InputTokens))
				totalTokens.OutputTokens += int(aws.ToInt32(e.Value.Usage.OutputTokens))
			}
		}
	}
	if err := stream.Err(); err != nil {
		return NarratorResult{}, fmt.Errorf("stream error: %w", err)
	}

	// Append this exchange to history (user turn + assistant text turn)
	newHistory := append(trimmed,
		game.NarrativeMessage{
			Role:    "user",
			Content: []game.NarrativeBlock{{Type: "text", Text: playerInput}},
		},
		game.NarrativeMessage{
			Role:    "assistant",
			Content: []game.NarrativeBlock{{Type: "text", Text: assistantText.String()}},
		},
	)

	return NarratorResult{
		Narrative:   fullNarrative.String(),
		NewMessages: newHistory,
		Tokens:      totalTokens,
	}, nil
}

// engineerMaxRounds is the maximum number of agentic tool-call rounds the
// Engineer is allowed before we stop regardless of stop reason. Each round is
// one Converse call → dispatch tools → feed tool_results back. In practice
// almost all turns complete in 1 round; the cap prevents infinite loops.
const engineerMaxRounds = 5

// EngineerScan reads the finished narrative and infers what world mutations it
// implies, then executes them against g using the full tool set.
// It runs synchronously after NarrateStream completes so the narrative stream
// is not delayed. Uses ModelSubAgent (Haiku) — fast and cheap.
//
// The Engineer uses an agentic tool loop: after each Converse call it executes
// the returned tool calls, sends tool_result blocks back to the model, and
// continues until the model returns end_turn with no tools or the round cap is
// reached. This gives Haiku visibility into tool failures so it can retry with
// corrected arguments.
func (c *Client) EngineerScan(
	ctx context.Context,
	g *game.Game,
	narrative string,
) (EngineerResult, error) {
	systemPrompt := engineerSystemPrompt()
	userMsg := engineerUserMessage(g, narrative)

	log.Printf("[engineer] START turn=%d narrativeLen=%d", g.ConversationCount, len(narrative))
	log.Printf("[engineer] user message:\n%s", userMsg)

	messages := []types.Message{{
		Role:    types.ConversationRoleUser,
		Content: []types.ContentBlock{&types.ContentBlockMemberText{Value: userMsg}},
	}}

	var result EngineerResult

	for round := 0; round < engineerMaxRounds; round++ {
		resp, err := c.br.Converse(ctx, &bedrockruntime.ConverseInput{
			ModelId:  aws.String(ModelSubAgent),
			System:   []types.SystemContentBlock{&types.SystemContentBlockMemberText{Value: systemPrompt}},
			Messages: messages,
			ToolConfig: &types.ToolConfiguration{
				Tools: NarratorTools(),
			},
			InferenceConfig: &types.InferenceConfiguration{
				MaxTokens:   aws.Int32(2048),
				Temperature: aws.Float32(0.1), // low temperature — structured extraction task
			},
		})
		if err != nil {
			return result, fmt.Errorf("engineer scan round %d: %w", round, err)
		}

		if resp.Usage != nil {
			result.Tokens.InputTokens += int(aws.ToInt32(resp.Usage.InputTokens))
			result.Tokens.OutputTokens += int(aws.ToInt32(resp.Usage.OutputTokens))
		}

		msg, ok := resp.Output.(*types.ConverseOutputMemberMessage)
		if !ok {
			log.Printf("[engineer] round=%d no message in output, stopping", round)
			break
		}

		// Log the full assistant message content
		for _, block := range msg.Value.Content {
			switch b := block.(type) {
			case *types.ContentBlockMemberText:
				if b.Value != "" {
					log.Printf("[engineer] round=%d assistant text: %s", round, b.Value)
				}
			case *types.ContentBlockMemberToolUse:
				rawInput, _ := json.Marshal(b.Value.Input)
				log.Printf("[engineer] round=%d tool_use: name=%s input=%s",
					round, aws.ToString(b.Value.Name), string(rawInput))
			}
		}

		// Collect tool_use blocks from this response
		var toolUseBlocks []*types.ContentBlockMemberToolUse
		for _, block := range msg.Value.Content {
			if tu, ok := block.(*types.ContentBlockMemberToolUse); ok {
				toolUseBlocks = append(toolUseBlocks, tu)
			}
		}

		stopReason := resp.StopReason
		log.Printf("[engineer] round=%d stopReason=%v toolCalls=%d",
			round, stopReason, len(toolUseBlocks))

		// No tool calls → model is done
		if len(toolUseBlocks) == 0 {
			break
		}

		// Execute each tool and collect tool_result blocks to feed back
		var toolResultBlocks []types.ContentBlock
		succeeded, failed := 0, 0
		for _, tu := range toolUseBlocks {
			toolName := aws.ToString(tu.Value.Name)
			toolID := aws.ToString(tu.Value.ToolUseId)

			// Unmarshal the tool input from the lazy document
			var input map[string]any
			raw, _ := json.Marshal(tu.Value.Input)
			_ = json.Unmarshal(raw, &input)

			toolResult, event, dispatchErr := DispatchTool(g, toolName, input)
			if dispatchErr != nil {
				log.Printf("[engineer] round=%d tool=%s FAILED: %v", round, toolName, dispatchErr)
				toolResult = fmt.Sprintf("error: %v", dispatchErr)
				failed++
			} else {
				log.Printf("[engineer] round=%d tool=%s OK: %s", round, toolName, toolResult)
				succeeded++
			}

			if event != nil {
				result.Events = append(result.Events, *event)
			}
			result.Mutations = append(result.Mutations, game.MutationEntry{
				SessionID: g.ID,
				Ts:        time.Now().UnixMilli(),
				Turn:      g.ConversationCount,
				Tool:      toolName,
				Input:     input,
				Result:    toolResult,
			})

			// Build tool_result block so the model can see success/failure and retry
			toolResultBlocks = append(toolResultBlocks, &types.ContentBlockMemberToolResult{
				Value: types.ToolResultBlock{
					ToolUseId: aws.String(toolID),
					Content: []types.ToolResultContentBlock{
						&types.ToolResultContentBlockMemberText{Value: toolResult},
					},
				},
			})
		}

		log.Printf("[engineer] round=%d dispatch results: succeeded=%d failed=%d",
			round, succeeded, failed)

		// Append assistant turn + user turn with tool results to the conversation
		messages = append(messages,
			types.Message{
				Role:    types.ConversationRoleAssistant,
				Content: msg.Value.Content,
			},
			types.Message{
				Role:    types.ConversationRoleUser,
				Content: toolResultBlocks,
			},
		)

		// If the model signalled end_turn even with tools, stop after processing them
		if stopReason == "end_turn" {
			break
		}
	}

	log.Printf("[engineer] DONE turn=%d mutations=%d events=%d tokens=in:%d+out:%d",
		g.ConversationCount, len(result.Mutations), len(result.Events),
		result.Tokens.InputTokens, result.Tokens.OutputTokens)

	return result, nil
}

// engineerSystemPrompt returns the system instructions for the Engineer.
func engineerSystemPrompt() string {
	return `You are a game world engineer. Your job is to read a narrator's text and execute the world mutations it implies using the provided tools.

Rules:
- Call tools ONLY for mutations clearly and unambiguously implied by the narrative text.
- Do NOT invent mutations not supported by the text.
- Do NOT output any narrative text — only tool calls.
- If the narrative implies no world changes, call no tools.
- Prefer precision over completeness: it is better to miss a subtle mutation than to invent one.

Examples of what to look for:
- "The goblin slashes you for 15 damage" → damage_character(player, 15)
- "You find a rusty key on the floor" → give_item_to_player(rusty key) or place_item_in_room
- "The merchant gives you a healing potion" → give_item_to_player(healing potion)
- "The orc falls dead" → set_character_alive(orc, false)
- "The bridge collapses, blocking the northern passage" → update_room(current room, updated description)
- "A cloaked figure emerges from the shadows" → move_character or create_character if not yet present`
}

// engineerUserMessage builds the user-turn message for the Engineer.
// It includes the narrative text plus a compact game state snapshot so the
// Engineer knows what entities exist to reference by name.
func engineerUserMessage(g *game.Game, narrative string) string {
	var sb strings.Builder
	sb.WriteString("## Narrative\n\n")
	sb.WriteString(narrative)
	sb.WriteString("\n\n## Current Game State\n\n")

	// Player
	sb.WriteString(fmt.Sprintf("Player: %s (health %d, alive %v)\n", g.Player.Name, g.Player.Health, g.Player.Alive))
	sb.WriteString("Player inventory: ")
	if len(g.Player.Inventory) == 0 {
		sb.WriteString("empty")
	} else {
		names := make([]string, 0, len(g.Player.Inventory))
		for _, id := range g.Player.Inventory {
			if item, err := g.GetItem(id); err == nil {
				names = append(names, item.Name)
			}
		}
		sb.WriteString(strings.Join(names, ", "))
	}
	sb.WriteString("\n")

	// Current room
	if room, err := g.GetRoom(g.Player.LocationID); err == nil {
		sb.WriteString(fmt.Sprintf("Current room: %s\n", room.Name))
		sb.WriteString("Room items: ")
		if len(room.Items) == 0 {
			sb.WriteString("none")
		} else {
			names := make([]string, 0, len(room.Items))
			for _, id := range room.Items {
				if item, err := g.GetItem(id); err == nil {
					names = append(names, item.Name)
				}
			}
			sb.WriteString(strings.Join(names, ", "))
		}
		sb.WriteString("\n")

		// NPCs in current room
		sb.WriteString("Room occupants: ")
		if len(room.Occupants) == 0 {
			sb.WriteString("none")
		} else {
			parts := make([]string, 0, len(room.Occupants))
			for _, id := range room.Occupants {
				if npc, err := g.GetNPC(id); err == nil {
					parts = append(parts, fmt.Sprintf("%s (health %d, alive %v)", npc.Name, npc.Health, npc.Alive))
				}
			}
			sb.WriteString(strings.Join(parts, "; "))
		}
		sb.WriteString("\n")
	}

	// All NPCs (for cross-room mutations)
	sb.WriteString("All NPCs: ")
	npcParts := make([]string, 0, len(g.NPCs))
	for _, npc := range g.NPCs {
		roomName := ""
		if r, err := g.GetRoom(npc.LocationID); err == nil {
			roomName = r.Name
		}
		npcParts = append(npcParts, fmt.Sprintf("%s (health %d, alive %v, location: %s)", npc.Name, npc.Health, npc.Alive, roomName))
	}
	if len(npcParts) == 0 {
		sb.WriteString("none")
	} else {
		sb.WriteString(strings.Join(npcParts, "; "))
	}
	sb.WriteString("\n\n")
	sb.WriteString("Execute all world mutations clearly implied by the narrative above.")
	return sb.String()
}

// ---- Context compression ----

// maxHistoryMessages is the maximum number of NarrativeMessage entries to send
// to Bedrock before trimming. Each player turn + assistant response = 2 messages,
// so this allows ~20 full exchanges before compression kicks in.
const maxHistoryMessages = 40

// TrimHistory reduces the narrative history if it exceeds maxHistoryMessages.
// It summarises the dropped messages into a single synthetic assistant turn so
// the model retains the plot context without the full token cost.
// The summary is generated using ModelSubAgent (fast/cheap).
func (c *Client) TrimHistory(ctx context.Context, history []game.NarrativeMessage) ([]game.NarrativeMessage, error) {
	if len(history) <= maxHistoryMessages {
		return history, nil
	}

	// Keep the most recent maxHistoryMessages entries; summarise the rest.
	cutoff := len(history) - maxHistoryMessages
	toSummarise := history[:cutoff]
	kept := history[cutoff:]

	// Build a plain-text digest of the dropped messages
	var sb strings.Builder
	for _, m := range toSummarise {
		for _, b := range m.Content {
			if b.Type == "text" && b.Text != "" {
				role := "Narrator"
				if m.Role == "user" {
					role = "Player"
				}
				sb.WriteString(role)
				sb.WriteString(": ")
				sb.WriteString(b.Text)
				sb.WriteString("\n\n")
			}
		}
	}

	prompt := "Summarise the following adventure log in 3-5 concise paragraphs, " +
		"preserving key plot points, character introductions, items found, and locations visited. " +
		"Write in third person past tense.\n\n" + sb.String()

	resp, err := c.br.Converse(ctx, &bedrockruntime.ConverseInput{
		ModelId: aws.String(ModelSubAgent),
		Messages: []types.Message{{
			Role: types.ConversationRoleUser,
			Content: []types.ContentBlock{
				&types.ContentBlockMemberText{Value: prompt},
			},
		}},
		InferenceConfig: &types.InferenceConfiguration{
			MaxTokens: aws.Int32(512),
		},
	})
	if err != nil {
		// Non-fatal: log and return history untrimmed rather than breaking the game
		log.Printf("TrimHistory: summarisation failed (returning untrimmed): %v", err)
		return history, nil
	}

	if resp.Usage != nil {
		log.Printf("TrimHistory: tokens — input: %d, output: %d",
			aws.ToInt32(resp.Usage.InputTokens), aws.ToInt32(resp.Usage.OutputTokens))
	}

	summary := extractText(resp.Output)
	summaryMsg := game.NarrativeMessage{
		Role: "assistant",
		Content: []game.NarrativeBlock{{
			Type: "text",
			Text: "[Story so far] " + summary,
		}},
	}

	log.Printf("TrimHistory: trimmed %d → %d messages (summary: %d chars)",
		len(history), len(kept)+1, len(summary))

	return append([]game.NarrativeMessage{summaryMsg}, kept...), nil
}

// ---- World Generation ----

// WorldBlueprint is the structured JSON the Architect produces.
type WorldBlueprint struct {
	Title        string               `json:"title"`
	Theme        string               `json:"theme"`
	QuestGoal    string               `json:"quest_goal"`
	OpeningScene string               `json:"opening_scene"`
	PlayerName   string               `json:"player_name,omitempty"` // AI-generated name if player left it blank
	Rooms        []BlueprintRoom      `json:"rooms"`
	Items        []BlueprintItem      `json:"items"`
	Characters   []BlueprintCharacter `json:"characters"`
}

// BlueprintRoom describes a room the Architect wants created.
type BlueprintRoom struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Connections map[string]string `json:"connections"` // direction -> room name
}

// BlueprintItem describes an item to create.
type BlueprintItem struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Weight      float64 `json:"weight"`
	PlaceInRoom string  `json:"place_in_room"` // room name; empty = player inventory
}

// BlueprintCharacter describes an NPC to create.
type BlueprintCharacter struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Backstory   string `json:"backstory"`
	Friendly    bool   `json:"friendly"`
	Health      int    `json:"health"`
	RoomName    string `json:"room_name"`
}

// GenerateBlueprint asks the Architect to produce a complete world blueprint
// in a single structured Converse call (no streaming, no tools).
//
// All player parameters are optional — the AI will invent values for any blank
// fields and return them in the blueprint (e.g. player_name if playerName=="").
func (c *Client) GenerateBlueprint(
	ctx context.Context,
	playerName, playerDescription, playerBackstory, themeHint string,
	preferences []string,
) (WorldBlueprint, string, TokenUsage, error) {
	// Build optional player context section
	var playerSections strings.Builder
	if playerName != "" {
		playerSections.WriteString(fmt.Sprintf("Player name: %q\n", playerName))
	} else {
		playerSections.WriteString("Player name: (not provided — invent a fitting name and return it in the \"player_name\" field)\n")
	}
	if playerDescription != "" {
		playerSections.WriteString(fmt.Sprintf("Player appearance/description: %s\n", playerDescription))
	}
	if playerBackstory != "" {
		playerSections.WriteString(fmt.Sprintf("Player backstory: %s\n", playerBackstory))
	}
	if themeHint != "" {
		playerSections.WriteString(fmt.Sprintf("Desired world tone/theme: %s\n", themeHint))
	}
	if len(preferences) > 0 {
		playerSections.WriteString(fmt.Sprintf("Preferred gameplay elements: %s\n", strings.Join(preferences, ", ")))
		playerSections.WriteString("Design the adventure so these elements are prominent.\n")
	}

	prompt := fmt.Sprintf(`You are designing a complete, closed-ended one-shot text adventure game.

%s
Return a single JSON object matching this schema exactly — no markdown, no commentary:
{
  "title": "adventure title",
  "theme": "one-line world description",
  "quest_goal": "what the player must achieve to win",
  "opening_scene": "2-3 sentences the player reads at the start",
  "player_name": "the player's name (only set this if you invented it; leave empty if provided above)",
  "rooms": [
    {
      "name": "unique room name",
      "description": "what the player sees here",
      "connections": {"north": "other room name", ...}
    }
  ],
  "items": [
    {
      "name": "item name",
      "description": "description",
      "weight": 1.0,
      "place_in_room": "room name or empty for player inventory"
    }
  ],
  "characters": [
    {
      "name": "character name",
      "description": "physical description",
      "backstory": "DM backstory notes",
      "friendly": true,
      "health": 100,
      "room_name": "starting room name"
    }
  ]
}

Rules:
- 6-10 rooms connected in a non-linear layout
- At least 2 success paths to the goal
- 4-8 items, some of which are puzzle keys
- 2-4 NPCs, mix of friendly and hostile
- All room connections must be bidirectionally consistent
- Player starts in the first room in the rooms array
- Incorporate any player description/backstory into the opening scene and world lore`, playerSections.String())

	resp, err := c.br.Converse(ctx, &bedrockruntime.ConverseInput{
		ModelId: aws.String(ModelNarrator),
		Messages: []types.Message{
			{
				Role:    types.ConversationRoleUser,
				Content: []types.ContentBlock{&types.ContentBlockMemberText{Value: prompt}},
			},
		},
		InferenceConfig: &types.InferenceConfiguration{
			MaxTokens:   aws.Int32(8192),
			Temperature: aws.Float32(0.8),
		},
	})
	if err != nil {
		return WorldBlueprint{}, "", TokenUsage{}, fmt.Errorf("generate blueprint: %w", err)
	}

	var usage TokenUsage
	if resp.Usage != nil {
		usage.InputTokens = int(aws.ToInt32(resp.Usage.InputTokens))
		usage.OutputTokens = int(aws.ToInt32(resp.Usage.OutputTokens))
	}

	text := extractText(resp.Output)

	// Strip any accidental markdown fences
	text = strings.TrimSpace(text)
	if strings.HasPrefix(text, "```") {
		text = strings.TrimPrefix(text, "```json")
		text = strings.TrimPrefix(text, "```")
		text = strings.TrimSuffix(text, "```")
		text = strings.TrimSpace(text)
	}

	var bp WorldBlueprint
	if err := json.Unmarshal([]byte(text), &bp); err != nil {
		return WorldBlueprint{}, text, usage, fmt.Errorf("parse blueprint JSON: %w (raw: %s)", err, text[:min(200, len(text))])
	}
	return bp, text, usage, nil
}

// BuildWorldFromBlueprint executes the blueprint deterministically — no AI loop.
// All IDs are generated server-side; names are what the AI provided.
func BuildWorldFromBlueprint(g *game.Game, bp WorldBlueprint) error {
	// Pass 1: create all rooms
	nameToID := make(map[string]string)
	for _, rb := range bp.Rooms {
		room := game.NewArea(rb.Name, rb.Description)
		if err := g.AddRoom(room); err != nil {
			return fmt.Errorf("add room %q: %w", rb.Name, err)
		}
		nameToID[rb.Name] = room.ID
	}

	// Pass 2: wire connections (rooms already exist)
	for _, rb := range bp.Rooms {
		fromID := nameToID[rb.Name]
		for dir, toName := range rb.Connections {
			toID, ok := nameToID[toName]
			if !ok {
				log.Printf("blueprint warning: connection %q -> %q not found, skipping", rb.Name, toName)
				continue
			}
			// ConnectRooms is idempotent via ForceConnection
			if err := g.ConnectRooms(fromID, toID, dir); err != nil {
				// Don't fail hard — opposite direction may already be set
				log.Printf("blueprint connect %q -[%s]-> %q: %v", rb.Name, dir, toName, err)
			}
		}
	}

	// Pass 3: place player in first room
	if len(bp.Rooms) > 0 {
		firstID := nameToID[bp.Rooms[0].Name]
		if err := g.PlacePlayer(firstID); err != nil {
			return fmt.Errorf("place player: %w", err)
		}
	}

	// Pass 4: create items
	for _, ib := range bp.Items {
		item := game.NewItem(ib.Name, ib.Description)
		item.Weight = ib.Weight
		if err := g.AddItem(item); err != nil {
			return fmt.Errorf("add item %q: %w", ib.Name, err)
		}
		if ib.PlaceInRoom != "" {
			roomID, ok := nameToID[ib.PlaceInRoom]
			if !ok {
				log.Printf("blueprint warning: item %q place_in_room %q not found, giving to player", ib.Name, ib.PlaceInRoom)
				_ = g.GiveItemToPlayer(item.ID)
				continue
			}
			if err := g.PlaceItemInRoom(item.ID, roomID); err != nil {
				return fmt.Errorf("place item %q: %w", ib.Name, err)
			}
		} else {
			if err := g.GiveItemToPlayer(item.ID); err != nil {
				return fmt.Errorf("give item %q to player: %w", ib.Name, err)
			}
		}
	}

	// Pass 5: create NPCs
	for _, cb := range bp.Characters {
		c := game.NewCharacter(cb.Name, cb.Description)
		c.Backstory = cb.Backstory
		c.Friendly = cb.Friendly
		if cb.Health > 0 {
			c.Health = cb.Health
		}
		if err := g.AddNPC(c); err != nil {
			return fmt.Errorf("add NPC %q: %w", cb.Name, err)
		}
		if cb.RoomName != "" {
			roomID, ok := nameToID[cb.RoomName]
			if !ok {
				log.Printf("blueprint warning: NPC %q room %q not found, skipping placement", cb.Name, cb.RoomName)
				continue
			}
			if err := g.MoveNPC(c.ID, roomID); err != nil {
				return fmt.Errorf("place NPC %q: %w", cb.Name, err)
			}
		}
	}

	// Calculate map coordinates from player start
	g.CalculateRoomCoordinates()
	return nil
}

// ---- History conversion helpers ----

// toBedrockMessages converts our storage format to Bedrock API messages.
func toBedrockMessages(history []game.NarrativeMessage) []types.Message {
	msgs := make([]types.Message, 0, len(history))
	for _, h := range history {
		role := types.ConversationRoleUser
		if h.Role == "assistant" {
			role = types.ConversationRoleAssistant
		}
		var blocks []types.ContentBlock
		for _, b := range h.Content {
			if b.Type == "text" && b.Text != "" {
				blocks = append(blocks, &types.ContentBlockMemberText{Value: b.Text})
			}
		}
		if len(blocks) > 0 {
			msgs = append(msgs, types.Message{Role: role, Content: blocks})
		}
	}
	return msgs
}

// fromBedrockMessages converts Bedrock messages to our storage format.
func fromBedrockMessages(msgs []types.Message) []game.NarrativeMessage {
	out := make([]game.NarrativeMessage, 0, len(msgs))
	for _, m := range msgs {
		role := "user"
		if m.Role == types.ConversationRoleAssistant {
			role = "assistant"
		}
		var blocks []game.NarrativeBlock
		for _, b := range m.Content {
			switch v := b.(type) {
			case *types.ContentBlockMemberText:
				blocks = append(blocks, game.NarrativeBlock{Type: "text", Text: v.Value})
			}
			// tool_use and tool_result blocks are intentionally dropped from storage
			// to keep the saved history compact — only narrative text is preserved.
		}
		if len(blocks) > 0 {
			out = append(out, game.NarrativeMessage{Role: role, Content: blocks})
		}
	}
	return out
}

// narratorSystemPrompt returns the system instructions for the narrator.
// The Narrator has NO tools — it must write pure prose only.
func narratorSystemPrompt(g *game.Game) string {
	room, _ := g.GetRoom(g.Player.LocationID)
	return fmt.Sprintf(`You are an expert Dungeon Master narrating a text adventure game.
The player's name is %q and they are currently in %q.

Your ONLY job is to write immersive, engaging narrative prose.
Do NOT describe what you are about to do or what tools you might call.
Do NOT say things like "I will now..." or "As the DM, I...".
Write only what the player experiences — sights, sounds, dialogue, action.

DM Philosophy:
- Say Yes or Roll the Dice: if nothing is at stake, say yes and move the story forward.
- Fail Forward: failed attempts create complications and drama, never dead ends.
- Pacing: cut to the next interesting scene when things drag; slow for dramatic moments.
- Never list options — narrate the world and let the player decide what to do.
- Be specific and sensory: name the smells, the sounds, the textures.

Write 2-4 paragraphs of vivid prose. Do not break the fourth wall.`,
		g.Player.Name, room.Name)
}

// extractText pulls the first text block from a ConverseOutput.
func extractText(output types.ConverseOutput) string {
	if msg, ok := output.(*types.ConverseOutputMemberMessage); ok {
		for _, block := range msg.Value.Content {
			if t, ok := block.(*types.ContentBlockMemberText); ok {
				return t.Value
			}
		}
	}
	return ""
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
