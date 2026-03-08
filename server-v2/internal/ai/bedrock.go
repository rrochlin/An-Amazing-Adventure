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
	brDocument "github.com/aws/aws-sdk-go-v2/service/bedrockruntime/document"
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
type NarratorResult struct {
	Narrative   string                  // accumulated text sent to the player
	NewMessages []game.NarrativeMessage // updated history to persist
	Tokens      TokenUsage              // token usage for this turn
	Events      []game.WorldEvent       // player-visible world events from tool calls this turn
	Mutations   []game.MutationEntry    // audit log entries for all tool calls this turn
}

// NarrateStream runs a single narrator turn with streaming.
// It calls onChunk for each text chunk so ws-chat can push frames immediately.
// Tool calls are executed synchronously against g as they arrive; any state
// mutations are reflected in g by the time the function returns.
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

	// Agentic loop: keep calling Bedrock until the model stops with tool use
	var fullNarrative strings.Builder
	var totalTokens TokenUsage
	var allEvents []game.WorldEvent
	var allMutations []game.MutationEntry
	allMessages := messages

	for {
		resp, err := c.br.ConverseStream(ctx, &bedrockruntime.ConverseStreamInput{
			ModelId:  aws.String(ModelNarrator),
			System:   []types.SystemContentBlock{&types.SystemContentBlockMemberText{Value: systemPrompt}},
			Messages: allMessages,
			ToolConfig: &types.ToolConfiguration{
				Tools: NarratorTools(),
			},
			InferenceConfig: &types.InferenceConfiguration{
				MaxTokens:   aws.Int32(4096),
				Temperature: aws.Float32(0.7),
			},
		})
		if err != nil {
			return NarratorResult{}, fmt.Errorf("converse stream: %w", err)
		}

		// Collect the assistant's response turn
		var assistantBlocks []types.ContentBlock
		var pendingToolUses []pendingToolUse
		var currentText strings.Builder

		stream := resp.GetStream()
		for event := range stream.Events() {
			switch e := event.(type) {
			case *types.ConverseStreamOutputMemberContentBlockDelta:
				switch d := e.Value.Delta.(type) {
				case *types.ContentBlockDeltaMemberText:
					currentText.WriteString(d.Value)
					fullNarrative.WriteString(d.Value)
					if onChunk != nil {
						onChunk(d.Value)
					}
				case *types.ContentBlockDeltaMemberToolUse:
					// Accumulate tool input JSON
					if len(pendingToolUses) > 0 {
						last := &pendingToolUses[len(pendingToolUses)-1]
						last.inputJSON += aws.ToString(d.Value.Input)
					}
				}
			case *types.ConverseStreamOutputMemberContentBlockStart:
				if tu, ok := e.Value.Start.(*types.ContentBlockStartMemberToolUse); ok {
					pendingToolUses = append(pendingToolUses, pendingToolUse{
						id:   aws.ToString(tu.Value.ToolUseId),
						name: aws.ToString(tu.Value.Name),
					})
				}
				if currentText.Len() > 0 {
					assistantBlocks = append(assistantBlocks, &types.ContentBlockMemberText{
						Value: currentText.String(),
					})
					currentText.Reset()
				}
			case *types.ConverseStreamOutputMemberContentBlockStop:
				if currentText.Len() > 0 {
					assistantBlocks = append(assistantBlocks, &types.ContentBlockMemberText{
						Value: currentText.String(),
					})
					currentText.Reset()
				}
			case *types.ConverseStreamOutputMemberMessageStop:
				if currentText.Len() > 0 {
					assistantBlocks = append(assistantBlocks, &types.ContentBlockMemberText{
						Value: currentText.String(),
					})
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

		// Add tool use blocks to assistant message
		for _, ptu := range pendingToolUses {
			var input map[string]any
			_ = json.Unmarshal([]byte(ptu.inputJSON), &input)
			assistantBlocks = append(assistantBlocks, &types.ContentBlockMemberToolUse{
				Value: types.ToolUseBlock{
					ToolUseId: aws.String(ptu.id),
					Name:      aws.String(ptu.name),
					Input:     brDocument.NewLazyDocument(input),
				},
			})
		}

		// Append assistant turn
		allMessages = append(allMessages, types.Message{
			Role:    types.ConversationRoleAssistant,
			Content: assistantBlocks,
		})

		// Execute all tool calls and build a tool_result turn
		if len(pendingToolUses) == 0 {
			break // model finished without tool calls — done
		}

		toolResultBlocks := make([]types.ContentBlock, 0, len(pendingToolUses))
		for _, ptu := range pendingToolUses {
			var input map[string]any
			_ = json.Unmarshal([]byte(ptu.inputJSON), &input)
			result, event, dispatchErr := DispatchTool(g, ptu.name, input)
			if dispatchErr != nil {
				log.Printf("tool %s error: %v", ptu.name, dispatchErr)
				result = fmt.Sprintf("error: %v", dispatchErr)
			}
			if event != nil {
				allEvents = append(allEvents, *event)
			}
			allMutations = append(allMutations, game.MutationEntry{
				SessionID: g.ID,
				Ts:        time.Now().UnixMilli(),
				Turn:      g.ConversationCount,
				Tool:      ptu.name,
				Input:     input,
				Result:    result,
			})
			toolResultBlocks = append(toolResultBlocks, &types.ContentBlockMemberToolResult{
				Value: types.ToolResultBlock{
					ToolUseId: aws.String(ptu.id),
					Content: []types.ToolResultContentBlock{
						&types.ToolResultContentBlockMemberText{Value: result},
					},
				},
			})
		}
		allMessages = append(allMessages, types.Message{
			Role:    types.ConversationRoleUser,
			Content: toolResultBlocks,
		})
	}

	// Convert back to our storage format
	newHistory := fromBedrockMessages(allMessages)

	return NarratorResult{
		Narrative:   fullNarrative.String(),
		NewMessages: newHistory,
		Tokens:      totalTokens,
		Events:      allEvents,
		Mutations:   allMutations,
	}, nil
}

type pendingToolUse struct {
	id        string
	name      string
	inputJSON string
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
func narratorSystemPrompt(g *game.Game) string {
	room, _ := g.GetRoom(g.Player.LocationID)
	return fmt.Sprintf(`You are an expert Dungeon Master running a text adventure game.
The player's name is %q and they are currently in %q.

DM Philosophy:
- Say Yes or Roll the Dice: if nothing is at stake, just say yes and move the story forward.
- Fail Forward: failed attempts create complications and drama, never dead ends.
- Intent and Task: encourage players to describe what they want to achieve and how.
- Pacing: cut to the next interesting scene when things drag; slow down for dramatic moments.
- Never list options for the player — they read the narrative and decide themselves.

Use the provided tools to mutate the game world as the story demands.
Respond with engaging prose. Call tools when the world should change.
Keep narrative responses to 2-4 paragraphs unless the scene warrants more.`,
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
