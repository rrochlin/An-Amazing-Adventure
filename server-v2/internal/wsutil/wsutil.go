// Package wsutil provides helpers for pushing messages to WebSocket clients
// via the API Gateway Management API.
package wsutil

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/apigatewaymanagementapi"
)

// FrameType identifies the kind of message being pushed to the client.
type FrameType string

const (
	FrameNarrativeChunk   FrameType = "narrative_chunk"
	FrameNarrativeEnd     FrameType = "narrative_end"
	FrameGameStateUpdate  FrameType = "game_state_update"
	FrameStateDelta       FrameType = "state_delta"
	FrameError            FrameType = "error"
	FrameStreamingBlocked FrameType = "streaming_blocked"
	// World-generation progress frames — sent by the world-gen Lambda while
	// it is running, before the game is marked ready.
	FrameWorldGenLog   FrameType = "world_gen_log"
	FrameWorldGenReady FrameType = "world_gen_ready"
)

// Frame is the JSON envelope sent to the client over WebSocket.
type Frame struct {
	Type    FrameType `json:"type"`
	Payload any       `json:"payload,omitempty"`
}

// Sender pushes WebSocket frames to a specific connection.
type Sender struct {
	mgmt *apigatewaymanagementapi.Client
}

// New creates a Sender using the WEBSOCKET_API_ENDPOINT env var.
// Format: https://<api-id>.execute-api.<region>.amazonaws.com/<stage>
func New(ctx context.Context) (*Sender, error) {
	endpoint := os.Getenv("WEBSOCKET_API_ENDPOINT")
	if endpoint == "" {
		return nil, fmt.Errorf("WEBSOCKET_API_ENDPOINT not set")
	}
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, err
	}
	client := apigatewaymanagementapi.NewFromConfig(cfg, func(o *apigatewaymanagementapi.Options) {
		o.BaseEndpoint = aws.String(endpoint)
	})
	return &Sender{mgmt: client}, nil
}

// Send serialises frame and posts it to the given connectionID.
// Stale/gone connections return a GoneException which callers should handle
// by cleaning up the connection record.
func (s *Sender) Send(ctx context.Context, connectionID string, frame Frame) error {
	data, err := json.Marshal(frame)
	if err != nil {
		return fmt.Errorf("marshal frame: %w", err)
	}
	_, err = s.mgmt.PostToConnection(ctx, &apigatewaymanagementapi.PostToConnectionInput{
		ConnectionId: aws.String(connectionID),
		Data:         data,
	})
	return err
}

// SendNarrativeChunk sends a single streaming text chunk.
func (s *Sender) SendNarrativeChunk(ctx context.Context, connectionID, chunk string) error {
	return s.Send(ctx, connectionID, Frame{
		Type:    FrameNarrativeChunk,
		Payload: map[string]string{"content": chunk},
	})
}

// SendNarrativeEnd signals that the AI has finished streaming.
func (s *Sender) SendNarrativeEnd(ctx context.Context, connectionID string) error {
	return s.Send(ctx, connectionID, Frame{Type: FrameNarrativeEnd})
}

// SendDelta sends a partial state update (changed rooms/player only).
func (s *Sender) SendDelta(ctx context.Context, connectionID string, delta any) error {
	return s.Send(ctx, connectionID, Frame{Type: FrameStateDelta, Payload: delta})
}

// SendError sends an error frame so the client can surface it.
func (s *Sender) SendError(ctx context.Context, connectionID, message string) error {
	return s.Send(ctx, connectionID, Frame{
		Type:    FrameError,
		Payload: map[string]string{"message": message},
	})
}

// SendFullState sends the complete game state (used on connect and after
// game_action mutations).
func (s *Sender) SendFullState(ctx context.Context, connectionID string, state any) error {
	return s.Send(ctx, connectionID, Frame{Type: FrameGameStateUpdate, Payload: state})
}

// SendWorldGenLog pushes a single line of world-generation progress text.
// The client displays these in a terminal-style component while waiting.
func (s *Sender) SendWorldGenLog(ctx context.Context, connectionID, line string) error {
	return s.Send(ctx, connectionID, Frame{
		Type:    FrameWorldGenLog,
		Payload: map[string]string{"line": line},
	})
}

// SendWorldGenReady signals that world generation completed successfully.
// The client should transition from the terminal view to the game.
func (s *Sender) SendWorldGenReady(ctx context.Context, connectionID string) error {
	return s.Send(ctx, connectionID, Frame{Type: FrameWorldGenReady})
}
