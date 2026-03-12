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
	ModelNarrator = "us.anthropic.claude-sonnet-4-6"              // heavy — narrator + narrative framing
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

			toolResult, event, dispatchErr := DispatchTool(ctx, g, toolName, input)
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
- "The lever grinds and a hidden passage opens to the east" → connect_rooms(current room, hidden room, east)
- "You find a rusty key on the floor" → give_item_to_player(rusty key) or place_item_in_room
- "The merchant gives you a healing potion" → give_item_to_player(healing potion)
- "A warded chest materializes beside the altar" → create_item + place_item_in_room
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

	// Player — use DnD HP when available (authoritative), fall back to legacy stub
	owner, _ := g.OwnerCharacter()
	playerHP := owner.Health
	playerAlive := owner.Alive
	playerMaxHP := 100
	if dndChar, hasDnD := g.GetDnDCharacter(g.OwnerID); hasDnD && dndChar != nil {
		playerHP = dndChar.GetHitPoints()
		playerAlive = playerHP > 0
		playerMaxHP = dndChar.ToData().MaxHitPoints
	}
	sb.WriteString(fmt.Sprintf("Player: %s (health %d/%d, alive %v)\n", owner.Name, playerHP, playerMaxHP, playerAlive))
	sb.WriteString("Player inventory: ")
	if len(owner.Inventory) == 0 {
		sb.WriteString("empty")
	} else {
		names := make([]string, 0, len(owner.Inventory))
		for _, id := range owner.Inventory {
			if item, err := g.GetItem(id); err == nil {
				names = append(names, item.Name)
			}
		}
		sb.WriteString(strings.Join(names, ", "))
	}
	sb.WriteString("\n")

	// Current room
	if room, err := g.GetRoom(owner.LocationID); err == nil {
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

// NarrativeFraming holds the Claude-generated narrative wrapping for a
// procedurally generated dungeon. It is the output of GenerateNarrativeFraming.
type NarrativeFraming struct {
	Title            string            `json:"title"`
	Theme            string            `json:"theme"`
	QuestGoal        string            `json:"quest_goal"`
	OpeningScene     string            `json:"opening_scene"`
	RoomNames        map[string]string `json:"room_names"`        // zoneID → display name
	RoomDescriptions map[string]string `json:"room_descriptions"` // zoneID → 1-2 sentence atmospheric description
}

// GenerateNarrativeFraming sends a dungeon summary to Claude Sonnet and gets
// back the narrative framing (title, theme, quest, opening scene, room names).
// This is a single non-streaming Converse call where Claude writes narrative
// framing only; world layout is generated procedurally by rpg-toolkit.
func (c *Client) GenerateNarrativeFraming(
	ctx context.Context,
	dungeonSummary string,
	creationParams game.CharacterCreationData,
) (NarrativeFraming, TokenUsage, error) {
	themeHint := ""
	if creationParams.ThemeHint != "" {
		themeHint = fmt.Sprintf("\nDesired tone/theme: %s", creationParams.ThemeHint)
	}
	prefHint := ""
	if len(creationParams.Preferences) > 0 {
		prefHint = fmt.Sprintf("\nPreferred gameplay elements: %s", strings.Join(creationParams.Preferences, ", "))
	}

	prompt := fmt.Sprintf(`You are creating the narrative framing for a D&D 5e dungeon.
Given this procedurally generated dungeon layout, write:
1. An evocative dungeon TITLE (4-8 words)
2. A dark fantasy THEME (1 sentence)
3. A QUEST GOAL that fits the dungeon's structure (1-2 sentences)
4. An OPENING SCENE narrative (2-3 paragraphs) that establishes the atmosphere and hooks the player
5. A display NAME for each room (short, evocative, 2-5 words)
6. A short DESCRIPTION for each room (1-2 sentences of atmospheric, sensory detail — what the player sees, hears, smells on first entry)

Dungeon layout:
%s

Player character: %s the %s (race: %s)%s%s

Respond in JSON only — no markdown fences, no commentary:
{
  "title": "...",
  "theme": "...",
  "quest_goal": "...",
  "opening_scene": "...",
  "room_names": {"<room_id>": "<display name>", ...},
  "room_descriptions": {"<room_id>": "<1-2 sentence description>", ...}
}`,
		dungeonSummary,
		creationParams.Name, creationParams.ClassID, creationParams.RaceID,
		themeHint, prefHint,
	)

	resp, err := c.br.Converse(ctx, &bedrockruntime.ConverseInput{
		ModelId: aws.String(ModelNarrator),
		Messages: []types.Message{
			{
				Role:    types.ConversationRoleUser,
				Content: []types.ContentBlock{&types.ContentBlockMemberText{Value: prompt}},
			},
		},
		InferenceConfig: &types.InferenceConfiguration{
			MaxTokens:   aws.Int32(3000),
			Temperature: aws.Float32(0.8),
		},
	})
	if err != nil {
		return NarrativeFraming{}, TokenUsage{}, fmt.Errorf("generate narrative framing: %w", err)
	}

	var usage TokenUsage
	if resp.Usage != nil {
		usage.InputTokens = int(aws.ToInt32(resp.Usage.InputTokens))
		usage.OutputTokens = int(aws.ToInt32(resp.Usage.OutputTokens))
	}

	text := extractText(resp.Output)
	text = strings.TrimSpace(text)
	if strings.HasPrefix(text, "```") {
		text = strings.TrimPrefix(text, "```json")
		text = strings.TrimPrefix(text, "```")
		text = strings.TrimSuffix(text, "```")
		text = strings.TrimSpace(text)
	}

	var framing NarrativeFraming
	if err := json.Unmarshal([]byte(text), &framing); err != nil {
		return NarrativeFraming{}, usage, fmt.Errorf("parse narrative framing JSON: %w (raw: %.200s)", err, text)
	}
	return framing, usage, nil
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
// When g.PendingCombatContext is non-empty it is injected as a [COMBAT LOG] block
// so Claude narrates the mechanical results dramatically without inventing outcomes.
func narratorSystemPrompt(g *game.Game) string {
	owner, _ := g.OwnerCharacter()
	room, _ := g.GetRoom(owner.LocationID)

	// Build D&D character stats section if available
	charContext := ""
	if dndChar, ok := g.GetDnDCharacter(g.OwnerID); ok && dndChar != nil {
		charContext = "\n\n" + game.BuildCharacterContext(owner.Name, dndChar.ToData())
	}

	// Inject pending combat results so Claude narrates what actually happened
	combatContext := ""
	if g.PendingCombatContext != "" {
		combatContext = fmt.Sprintf("\n\n[COMBAT LOG — narrate these mechanical results dramatically; do not change the outcomes:]\n%s", g.PendingCombatContext)
		// Consume after injecting so it is not repeated on subsequent turns
		g.PendingCombatContext = ""
	}

	return fmt.Sprintf(`You are an expert Dungeon Master narrating a D&D 5e text adventure game.
The player's name is %q and they are currently in %q.%s%s

Your ONLY job is to write immersive, engaging narrative prose.
Do NOT describe what you are about to do or what tools you might call.
Do NOT say things like "I will now..." or "As the DM, I...".
Write only what the player experiences — sights, sounds, dialogue, action.

When narrating combat or physical feats, respect the character's D&D stats (HP, AC, ability scores).
A Barbarian with high STR smashes through doors; a Monk with high DEX moves like water.

DM Philosophy:
- Say Yes or Roll the Dice: if nothing is at stake, say yes and move the story forward.
- Fail Forward: failed attempts create complications and drama, never dead ends.
- Pacing: cut to the next interesting scene when things drag; slow for dramatic moments.
- Never list options — narrate the world and let the player decide what to do.
- Be specific and sensory: name the smells, the sounds, the textures.

Write 2-4 paragraphs of vivid prose. Do not break the fourth wall.`,
		owner.Name, room.Name, charContext, combatContext)
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
