package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	brDocument "github.com/aws/aws-sdk-go-v2/service/bedrockruntime/document"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"
	"github.com/rrochlin/an-amazing-adventure/internal/game"
	"github.com/rrochlin/an-amazing-adventure/internal/trace"
)

// GameMasterRequest is one game master turn: the brief as the system prompt,
// the conversation so far, the player's input, and the decision tool the
// model hands its decision to (its schema is built per scene).
type GameMasterRequest struct {
	System      string
	History     []game.NarrativeMessage
	PlayerInput string
	ToolName    string
	ToolSchema  map[string]any
	// OnChunk receives narration as it streams.
	OnChunk func(string)
}

// GameMasterResult is what came back: the prose, the decision tool's raw
// JSON input (empty when the model didn't call it), and the usual
// bookkeeping.
type GameMasterResult struct {
	Narrative   string
	Decision    string
	NewMessages []game.NarrativeMessage
	Tokens      TokenUsage
	StopReason  string
	Calls       []trace.ModelCall
}

// GameMasterTurn runs one game master turn: a single streaming call that
// narrates (text deltas, streamed to OnChunk) and then calls the decision
// tool with what the turn changed.
func (c *Client) GameMasterTurn(ctx context.Context, req GameMasterRequest) (GameMasterResult, error) {
	trimmed, err := c.TrimHistory(ctx, req.History)
	if err != nil {
		trimmed = req.History
	}
	history := append(append([]game.NarrativeMessage(nil), trimmed...), game.NarrativeMessage{
		Role:    "user",
		Content: []game.NarrativeBlock{{Type: "text", Text: req.PlayerInput}},
	})
	call := trace.ModelCall{Role: "game_master", Model: ModelNarrator, System: req.System, Messages: history}
	started := time.Now()
	fail := func(err error) (GameMasterResult, error) {
		call.Error = err.Error()
		call.DurationMs = time.Since(started).Milliseconds()
		return GameMasterResult{Calls: []trace.ModelCall{call}}, err
	}

	schema := map[string]any{}
	for k, v := range req.ToolSchema {
		schema[k] = v
	}
	resp, err := c.br.ConverseStream(ctx, &bedrockruntime.ConverseStreamInput{
		ModelId:  aws.String(ModelNarrator),
		System:   []types.SystemContentBlock{&types.SystemContentBlockMemberText{Value: req.System}},
		Messages: toBedrockMessages(history),
		ToolConfig: &types.ToolConfiguration{
			Tools: []types.Tool{&types.ToolMemberToolSpec{Value: types.ToolSpecification{
				Name:        aws.String(req.ToolName),
				Description: aws.String("Hand in the whole turn: the narration first, then the world changes it made happen, then the beats it settled."),
				InputSchema: &types.ToolInputSchemaMemberJson{Value: brDocument.NewLazyDocument(schema)},
			}}},
			// Forced, so a turn always comes with its decision; the narration
			// streams out of the tool input as it is written.
			ToolChoice: &types.ToolChoiceMemberTool{Value: types.SpecificToolChoice{Name: aws.String(req.ToolName)}},
		},
		InferenceConfig: &types.InferenceConfiguration{
			MaxTokens:   aws.Int32(4096),
			Temperature: aws.Float32(0.7),
		},
	})
	if err != nil {
		return fail(fmt.Errorf("converse stream: %w", err))
	}
	stream := resp.GetStream()
	parsed := parseGameMasterStream(stream.Events(), req.ToolName, req.OnChunk)
	if err := stream.Err(); err != nil {
		call.Output = parsed.narrative
		return fail(fmt.Errorf("stream error: %w", err))
	}

	call.Output = parsed.narrative
	if parsed.decision != "" {
		call.Output += "\n\n[" + req.ToolName + "] " + withoutNarration(parsed.decision)
	}
	call.StopReason = parsed.stopReason
	call.InputTok, call.OutputTok = parsed.tokens.InputTokens, parsed.tokens.OutputTokens
	call.DurationMs = time.Since(started).Milliseconds()

	return GameMasterResult{
		Narrative: parsed.narrative,
		Decision:  parsed.decision,
		NewMessages: append(history, game.NarrativeMessage{
			Role:    "assistant",
			Content: []game.NarrativeBlock{{Type: "text", Text: parsed.narrative}},
		}),
		Tokens:     parsed.tokens,
		StopReason: parsed.stopReason,
		Calls:      []trace.ModelCall{call},
	}, nil
}

type gameMasterStream struct {
	narrative  string
	decision   string
	stopReason string
	tokens     TokenUsage
}

// parseGameMasterStream reads a Converse stream. The turn arrives as the
// decision tool's input in partial JSON deltas; its "narration" string is
// decoded as it streams and handed to onChunk. Plain text blocks, if the
// model writes any, are narration too.
func parseGameMasterStream(events <-chan types.ConverseStreamOutput, toolName string, onChunk func(string)) gameMasterStream {
	var out gameMasterStream
	var narrative, decision strings.Builder
	inDecision := false
	narr := &narrationStreamer{emit: func(s string) {
		narrative.WriteString(s)
		if onChunk != nil {
			onChunk(s)
		}
	}}
	for event := range events {
		switch e := event.(type) {
		case *types.ConverseStreamOutputMemberContentBlockStart:
			if start, ok := e.Value.Start.(*types.ContentBlockStartMemberToolUse); ok {
				inDecision = aws.ToString(start.Value.Name) == toolName
			}
		case *types.ConverseStreamOutputMemberContentBlockDelta:
			switch d := e.Value.Delta.(type) {
			case *types.ContentBlockDeltaMemberText:
				narrative.WriteString(d.Value)
				if onChunk != nil {
					onChunk(d.Value)
				}
			case *types.ContentBlockDeltaMemberToolUse:
				if inDecision {
					part := aws.ToString(d.Value.Input)
					decision.WriteString(part)
					narr.feed(part)
				}
			}
		case *types.ConverseStreamOutputMemberContentBlockStop:
			inDecision = false
		case *types.ConverseStreamOutputMemberMessageStop:
			out.stopReason = string(e.Value.StopReason)
		case *types.ConverseStreamOutputMemberMetadata:
			if e.Value.Usage != nil {
				out.tokens.InputTokens += int(aws.ToInt32(e.Value.Usage.InputTokens))
				out.tokens.OutputTokens += int(aws.ToInt32(e.Value.Usage.OutputTokens))
			}
		}
	}
	out.decision = strings.TrimSpace(decision.String())
	// The parsed narration is authoritative; the stream is its preview.
	if n, ok := narrationOf(out.decision); ok {
		out.narrative = strings.TrimSpace(n)
	} else {
		out.narrative = strings.TrimSpace(narrative.String())
	}
	return out
}

// narrationOf reads the narration string out of the decision JSON.
func narrationOf(decision string) (string, bool) {
	var doc struct {
		Narration *string `json:"narration"`
	}
	if err := json.Unmarshal([]byte(decision), &doc); err != nil || doc.Narration == nil {
		return "", false
	}
	return *doc.Narration, true
}

// withoutNarration is the decision JSON minus the prose, for the trace.
func withoutNarration(decision string) string {
	var doc map[string]any
	if err := json.Unmarshal([]byte(decision), &doc); err != nil {
		return decision
	}
	delete(doc, "narration")
	raw, err := json.Marshal(doc)
	if err != nil {
		return decision
	}
	return string(raw)
}

// narrationStreamer decodes the JSON string value of "narration" as the
// tool input streams in, so the prose reaches the player while the model
// is still writing the rest of the decision.
type narrationStreamer struct {
	emit    func(string)
	buf     string // undecoded input not yet consumed
	state   int    // 0 looking for the key, 1 inside the string, 2 done
	pending string // an escape sequence waiting for more characters
}

var narrationKey = regexp.MustCompile(`"narration"\s*:\s*"`)

func (n *narrationStreamer) feed(part string) {
	if n.state == 2 {
		return
	}
	n.buf += part
	if n.state == 0 {
		loc := narrationKey.FindStringIndex(n.buf)
		if loc == nil {
			return
		}
		n.buf = n.buf[loc[1]:]
		n.state = 1
	}
	var out strings.Builder
	i := 0
	for i < len(n.buf) {
		c := n.buf[i]
		if c == '"' {
			n.state = 2
			i++
			break
		}
		if c != '\\' {
			out.WriteByte(c)
			i++
			continue
		}
		// An escape: needs the next character, and \u needs four more.
		if i+1 >= len(n.buf) {
			break
		}
		esc := n.buf[i+1]
		if esc == 'u' {
			if i+6 > len(n.buf) {
				break
			}
			if r, err := strconv.ParseUint(n.buf[i+2:i+6], 16, 32); err == nil {
				out.WriteRune(rune(r))
			}
			i += 6
			continue
		}
		switch esc {
		case 'n':
			out.WriteByte('\n')
		case 't':
			out.WriteByte('\t')
		case 'r':
			out.WriteByte('\r')
		case 'b', 'f':
			// rare in prose; dropped
		default: // \" \\ \/
			out.WriteByte(esc)
		}
		i += 2
	}
	n.buf = n.buf[i:]
	if out.Len() > 0 {
		n.emit(out.String())
	}
}
