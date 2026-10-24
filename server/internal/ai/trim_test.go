package ai

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"

	"github.com/rrochlin/an-amazing-adventure/internal/game"
)

func history(n int) []game.NarrativeMessage {
	msgs := make([]game.NarrativeMessage, n)
	for i := range msgs {
		role := "user"
		if i%2 == 1 {
			role = "assistant"
		}
		msgs[i] = game.NarrativeMessage{Role: role, Content: []game.NarrativeBlock{{Type: "text", Text: fmt.Sprintf("message %d", i)}}}
	}
	return msgs
}

// failingBedrock answers every call with an error.
type failingBedrock struct{ fakeBedrockRuntimeClient }

func (failingBedrock) Converse(context.Context, *bedrockruntime.ConverseInput, ...func(*bedrockruntime.Options)) (*bedrockruntime.ConverseOutput, error) {
	return nil, fmt.Errorf("throttled")
}

func TestTrimHistory_UpToTheLimitIsLeftAlone(t *testing.T) {
	fake := &fakeBedrockRuntimeClient{}
	c := &Client{br: fake}
	for _, n := range []int{10, maxHistoryMessages} {
		got, err := c.TrimHistory(context.Background(), history(n))
		if err != nil || len(got) != n {
			t.Fatalf("%d messages: got %d, %v", n, len(got), err)
		}
	}
	if len(fake.converseInputs) != 0 {
		t.Fatalf("no summary should be asked for, got %d calls", len(fake.converseInputs))
	}
}

func TestTrimHistory_SummarisesWhatIsDropped(t *testing.T) {
	fake := &fakeBedrockRuntimeClient{converseOutputs: []*bedrockruntime.ConverseOutput{{
		Output: &types.ConverseOutputMemberMessage{Value: types.Message{
			Role:    types.ConversationRoleAssistant,
			Content: []types.ContentBlock{&types.ContentBlockMemberText{Value: "The child found a doll."}},
		}},
	}}}
	c := &Client{br: fake}
	got, err := c.TrimHistory(context.Background(), history(maxHistoryMessages+10))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != maxHistoryMessages+1 || got[0].Content[0].Text != "[Story so far] The child found a doll." {
		t.Fatalf("expected a summary then the last %d messages, got %d starting %q", maxHistoryMessages, len(got), got[0].Content[0].Text)
	}
	if got[1].Content[0].Text != "message 10" {
		t.Fatalf("kept the wrong messages: first kept %q", got[1].Content[0].Text)
	}
	prompt := fake.converseInputs[0].Messages[0].Content[0].(*types.ContentBlockMemberText).Value
	if !strings.Contains(prompt, "Player: message 0") || strings.Contains(prompt, "message 10") {
		t.Fatalf("the summary should cover only the dropped messages:\n%s", prompt)
	}
}

func TestTrimHistory_KeepsEverythingWhenTheSummaryFails(t *testing.T) {
	c := &Client{br: &failingBedrock{}}
	got, err := c.TrimHistory(context.Background(), history(maxHistoryMessages+10))
	if err != nil || len(got) != maxHistoryMessages+10 {
		t.Fatalf("a failed summary must not break the game: got %d, %v", len(got), err)
	}
}
