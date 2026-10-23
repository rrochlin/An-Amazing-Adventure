package ai

import (
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"
)

func TestParseGameMasterStream_TextThenDecision(t *testing.T) {
	events := make(chan types.ConverseStreamOutput, 16)
	text := func(s string) {
		events <- &types.ConverseStreamOutputMemberContentBlockDelta{Value: types.ContentBlockDeltaEvent{Delta: &types.ContentBlockDeltaMemberText{Value: s}}}
	}
	text("The meadow ")
	text("opens before you.")
	events <- &types.ConverseStreamOutputMemberContentBlockStop{}
	events <- &types.ConverseStreamOutputMemberContentBlockStart{Value: types.ContentBlockStartEvent{Start: &types.ContentBlockStartMemberToolUse{Value: types.ToolUseBlockStart{Name: aws.String("end_turn"), ToolUseId: aws.String("t1")}}}}
	for _, part := range []string{`{"world_events":[{"type":"move_player",`, `"input":{"room_name":"Meadow"}}],`, `"beats":{}}`} {
		events <- &types.ConverseStreamOutputMemberContentBlockDelta{Value: types.ContentBlockDeltaEvent{Delta: &types.ContentBlockDeltaMemberToolUse{Value: types.ToolUseBlockDelta{Input: aws.String(part)}}}}
	}
	events <- &types.ConverseStreamOutputMemberContentBlockStop{}
	events <- &types.ConverseStreamOutputMemberMessageStop{Value: types.MessageStopEvent{StopReason: types.StopReasonToolUse}}
	events <- &types.ConverseStreamOutputMemberMetadata{Value: types.ConverseStreamMetadataEvent{Usage: &types.TokenUsage{InputTokens: aws.Int32(10), OutputTokens: aws.Int32(5)}}}
	close(events)

	var chunks []string
	got := parseGameMasterStream(events, "end_turn", func(s string) { chunks = append(chunks, s) })
	if got.narrative != "The meadow opens before you." || strings.Join(chunks, "") != "The meadow opens before you." {
		t.Fatalf("narrative %q chunks %v", got.narrative, chunks)
	}
	if got.decision != `{"world_events":[{"type":"move_player","input":{"room_name":"Meadow"}}],"beats":{}}` {
		t.Fatalf("decision %q", got.decision)
	}
	if got.stopReason != "tool_use" || got.tokens.InputTokens != 10 || got.tokens.OutputTokens != 5 {
		t.Fatalf("stop %q tokens %+v", got.stopReason, got.tokens)
	}
}

func TestParseGameMasterStream_IgnoresOtherTools(t *testing.T) {
	events := make(chan types.ConverseStreamOutput, 4)
	events <- &types.ConverseStreamOutputMemberContentBlockStart{Value: types.ContentBlockStartEvent{Start: &types.ContentBlockStartMemberToolUse{Value: types.ToolUseBlockStart{Name: aws.String("something_else")}}}}
	events <- &types.ConverseStreamOutputMemberContentBlockDelta{Value: types.ContentBlockDeltaEvent{Delta: &types.ContentBlockDeltaMemberToolUse{Value: types.ToolUseBlockDelta{Input: aws.String(`{"x":1}`)}}}}
	close(events)
	if got := parseGameMasterStream(events, "end_turn", nil); got.decision != "" {
		t.Fatalf("another tool's input must not be taken as the decision: %q", got.decision)
	}
}

func TestParseGameMasterStream_NarrationStreamsOutOfTheToolInput(t *testing.T) {
	events := make(chan types.ConverseStreamOutput, 16)
	events <- &types.ConverseStreamOutputMemberContentBlockStart{Value: types.ContentBlockStartEvent{Start: &types.ContentBlockStartMemberToolUse{Value: types.ToolUseBlockStart{Name: aws.String("end_turn")}}}}
	// Split on purpose: inside the key, inside an escape, inside a \u escape.
	for _, part := range []string{`{"narra`, `tion": "She said \"go`, `\" and left.\nThe meadow \u00e9`, `tait quiet.", "world_events":[],"beats":{}}`} {
		events <- &types.ConverseStreamOutputMemberContentBlockDelta{Value: types.ContentBlockDeltaEvent{Delta: &types.ContentBlockDeltaMemberToolUse{Value: types.ToolUseBlockDelta{Input: aws.String(part)}}}}
	}
	events <- &types.ConverseStreamOutputMemberContentBlockStop{}
	close(events)
	var chunks []string
	got := parseGameMasterStream(events, "end_turn", func(s string) { chunks = append(chunks, s) })
	want := "She said \"go\" and left.\nThe meadow était quiet."
	if strings.Join(chunks, "") != want {
		t.Fatalf("streamed %q, want %q", strings.Join(chunks, ""), want)
	}
	if got.narrative != want {
		t.Fatalf("narrative %q", got.narrative)
	}
	if strings.Contains(withoutNarration(got.decision), "narration") {
		t.Fatalf("trace copy should drop the prose: %s", withoutNarration(got.decision))
	}
}
