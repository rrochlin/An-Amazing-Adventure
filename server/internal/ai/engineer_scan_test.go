package ai

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	brdocument "github.com/aws/aws-sdk-go-v2/service/bedrockruntime/document"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"
	smithydocument "github.com/aws/smithy-go/document"
	"github.com/rrochlin/an-amazing-adventure/internal/game"
)

type fakeBedrockRuntimeClient struct {
	converseOutputs []*bedrockruntime.ConverseOutput
	converseInputs  []*bedrockruntime.ConverseInput
}

func (f *fakeBedrockRuntimeClient) Converse(_ context.Context, in *bedrockruntime.ConverseInput, _ ...func(*bedrockruntime.Options)) (*bedrockruntime.ConverseOutput, error) {
	f.converseInputs = append(f.converseInputs, in)
	if len(f.converseOutputs) == 0 {
		return nil, fmt.Errorf("no fake converse output queued")
	}
	out := f.converseOutputs[0]
	f.converseOutputs = f.converseOutputs[1:]
	return out, nil
}

func (*fakeBedrockRuntimeClient) ConverseStream(context.Context, *bedrockruntime.ConverseStreamInput, ...func(*bedrockruntime.Options)) (*bedrockruntime.ConverseStreamOutput, error) {
	return nil, fmt.Errorf("unexpected ConverseStream call")
}

func newEngineerTestGame(t *testing.T) *game.Game {
	t.Helper()
	g := game.NewGame("session-1", "owner-1")
	g.SetPlayerCharacter("owner-1", game.NewCharacter("Hero", "player"))
	room := game.NewArea("Hall", "Stone hall")
	if err := g.AddRoom(room); err != nil {
		t.Fatalf("add room: %v", err)
	}
	if err := g.PlacePlayer(room.ID); err != nil {
		t.Fatalf("place player: %v", err)
	}
	return g
}

func TestEngineerScan_ToolLoopDispatchesAndReturnsEventsMutations(t *testing.T) {
	g := newEngineerTestGame(t)
	g.ConversationCount = 4

	br := &fakeBedrockRuntimeClient{
		converseOutputs: []*bedrockruntime.ConverseOutput{
			{
				Output: &types.ConverseOutputMemberMessage{Value: types.Message{
					Content: []types.ContentBlock{
						&types.ContentBlockMemberToolUse{Value: types.ToolUseBlock{
							Name:      aws.String("damage_character"),
							ToolUseId: aws.String("tool-1"),
							Input: brdocument.NewLazyDocument(map[string]any{
								"character_name": "player",
								"amount":         6,
							}),
						}},
					},
				}},
				StopReason: types.StopReasonToolUse,
				Usage:      &types.TokenUsage{InputTokens: aws.Int32(10), OutputTokens: aws.Int32(3)},
			},
			{
				Output: &types.ConverseOutputMemberMessage{Value: types.Message{
					Content: []types.ContentBlock{
						&types.ContentBlockMemberText{Value: "done"},
					},
				}},
				StopReason: types.StopReasonEndTurn,
				Usage:      &types.TokenUsage{InputTokens: aws.Int32(4), OutputTokens: aws.Int32(2)},
			},
		},
	}
	client := &Client{br: br}

	var gotInput map[string]any
	originalDispatch := dispatchToolFn
	dispatchToolFn = func(_ context.Context, gameState *game.Game, name string, input map[string]any) (string, *game.WorldEvent, error) {
		gotInput = input
		owner, _ := gameState.OwnerCharacter()
		owner.Health -= 6
		gameState.Players[gameState.OwnerID] = owner
		return "applied 6 damage", &game.WorldEvent{Type: "damage", Message: "You take 6 damage."}, nil
	}
	t.Cleanup(func() {
		dispatchToolFn = originalDispatch
	})

	result, err := client.EngineerScan(context.Background(), g, "A trap injures the player.")
	if err != nil {
		t.Fatalf("EngineerScan: %v", err)
	}
	if len(br.converseInputs) != 2 {
		t.Fatalf("expected 2 converse rounds, got %d", len(br.converseInputs))
	}
	if len(br.converseInputs[1].Messages) != 3 {
		t.Fatalf("expected tool_result follow-up message chain, got %d messages", len(br.converseInputs[1].Messages))
	}
	// The model's arguments must reach the tool (they were silently empty).
	if gotInput["character_name"] != "player" || gotInput["amount"] != float64(6) {
		t.Fatalf("tool received input %v, want the model's arguments", gotInput)
	}
	if len(result.Events) != 1 || result.Events[0].Type != "damage" {
		t.Fatalf("expected one damage event, got %+v", result.Events)
	}
	if len(result.Mutations) != 1 || result.Mutations[0].Tool != "damage_character" {
		t.Fatalf("expected one damage_character mutation, got %+v", result.Mutations)
	}
	if result.Mutations[0].Turn != 4 || result.Mutations[0].SessionID != "session-1" {
		t.Fatalf("unexpected mutation metadata: %+v", result.Mutations[0])
	}
	owner, _ := g.OwnerCharacter()
	if owner.Health != 94 {
		t.Fatalf("expected game mutation to apply damage (health 94), got %d", owner.Health)
	}
	if result.Tokens.InputTokens != 14 || result.Tokens.OutputTokens != 5 {
		t.Fatalf("unexpected token totals: %+v", result.Tokens)
	}
}

func TestToolInputDecodesResponseDocuments(t *testing.T) {
	// A Bedrock response document decodes numbers as document.Number; tools
	// read float64, so toolInput must convert them, nested values included.
	doc := brdocument.NewLazyDocument(map[string]any{
		"room_name": "Meadow",
		"amount":    3,
		"friendly":  true,
		"nested":    map[string]any{"weight": 1.5, "list": []any{2}},
	})
	input, raw := toolInput(types.ToolUseBlock{Name: aws.String("t"), Input: doc})
	if input["room_name"] != "Meadow" || input["amount"] != float64(3) || input["friendly"] != true {
		t.Fatalf("unexpected input %#v", input)
	}
	nested := input["nested"].(map[string]any)
	if nested["weight"] != 1.5 || nested["list"].([]any)[0] != float64(2) {
		t.Fatalf("nested numbers not converted: %#v", nested)
	}
	if !strings.Contains(string(raw), `"room_name":"Meadow"`) {
		t.Fatalf("log JSON should carry the arguments, got %s", raw)
	}
	if got, _ := toolInput(types.ToolUseBlock{Name: aws.String("t")}); len(got) != 0 {
		t.Fatalf("missing input should decode to an empty map, got %v", got)
	}
}

func TestPlainNumbersConvertsDocumentNumber(t *testing.T) {
	if got := plainNumbers(smithydocument.Number("7")); got != float64(7) {
		t.Fatalf("document.Number not converted: %#v", got)
	}
}

func TestExtractJSONObjectTakesTheLastObject(t *testing.T) {
	// Playtest 5, turn 4: the resolver reasoned, wrote a wrong object, then a
	// corrected one. The old parser took the span from the first "{" to the
	// last "}" and choked on the prose between them.
	raw := "```json\n{ \"labels\": { \"deed_doll\": \"kept\" } }\n```\n**Correction**: not yet.\n```json\n{ \"labels\": { \"deed_bully\": \"stopped\" } }\n```"
	if got := extractJSONObject(raw); got != `{ "labels": { "deed_bully": "stopped" } }` {
		t.Fatalf("got %q", got)
	}
	if got := extractJSONObject("no json here"); got != "" {
		t.Fatalf("expected nothing, got %q", got)
	}
}
