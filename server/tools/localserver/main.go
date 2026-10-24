// Command localserver runs the game's backend on this machine for end-to-end
// testing of the real client: the HTTP routes the client calls and a
// WebSocket that hands chat and game actions to the real ws-chat and
// ws-game-action handlers. Sessions live in memory (internal/localstack);
// Bedrock is real (AWS_PROFILE=aa-debug AWS_REGION=us-west-2).
//
//	go run ./tools/localserver -addr :8787 [-traces DIR] [-state FILE]
//
// Point the client at it with VITE_APP_URI=http://localhost:8787 and
// VITE_WS_ENDPOINT=ws://localhost:8787/ws, and put any unexpired token in
// localStorage (the server reads the user from the JWT's sub claim, as
// ws-connect does, without checking the signature).
package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/coder/websocket"

	"github.com/rrochlin/an-amazing-adventure/internal/ai"
	"github.com/rrochlin/an-amazing-adventure/internal/campaigns"
	"github.com/rrochlin/an-amazing-adventure/internal/game"
	"github.com/rrochlin/an-amazing-adventure/internal/lambdas/wschat"
	"github.com/rrochlin/an-amazing-adventure/internal/lambdas/wsgameaction"
	"github.com/rrochlin/an-amazing-adventure/internal/localstack"
	"github.com/rrochlin/an-amazing-adventure/internal/trace"
	"github.com/rrochlin/an-amazing-adventure/internal/wsutil"
)

func main() {
	addr := flag.String("addr", ":8787", "listen address")
	traceDir := flag.String("traces", "", "directory to write each turn trace to (optional)")
	stateFile := flag.String("state", "", "file to keep sessions in across restarts (optional)")
	flag.Parse()
	if os.Getenv("MUTATIONS_TABLE") == "" {
		_ = os.Setenv("MUTATIONS_TABLE", "local-mutations")
	}

	reg, err := campaigns.LoadEmbeddedRegistry()
	if err != nil {
		log.Fatal(err)
	}
	store := localstack.NewStore()
	store.File = *stateFile
	if err := store.Load(); err != nil {
		log.Fatal(err)
	}
	if *traceDir != "" {
		if err := os.MkdirAll(*traceDir, 0o755); err != nil {
			log.Fatal(err)
		}
		var n atomic.Int64
		store.OnTrace = func(t *trace.Turn) {
			raw, _ := json.MarshalIndent(t, "", " ")
			name := fmt.Sprintf("%03d-%s.json", n.Add(1), t.Kind)
			_ = os.WriteFile(filepath.Join(*traceDir, name), raw, 0o644)
			log.Printf("turn %s %q: %d model calls, %d errors", t.Kind, t.Input, len(t.Calls), len(t.Errors))
		}
	}
	models, err := ai.New(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	sockets := newSockets()
	wschat.UseClients(
		func(context.Context) (wschat.DB, error) { return store, nil },
		func(context.Context) (wschat.AI, error) { return models, nil },
		func(context.Context) (wschat.Sender, error) { return sockets, nil },
	)
	wsgameaction.UseClients(
		func(context.Context) (wsgameaction.DB, error) { return store, nil },
		func(context.Context) (wsgameaction.Sender, error) { return sockets, nil },
	)

	s := &server{store: store, reg: reg, sockets: sockets}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/campaigns", s.listCampaigns)
	mux.HandleFunc("GET /api/games", s.listGames)
	mux.HandleFunc("POST /api/games", s.createGame)
	mux.HandleFunc("GET /api/games/{id}", s.getGame)
	mux.HandleFunc("DELETE /api/games/{id}", s.deleteGame)
	mux.HandleFunc("GET /ws", s.socket)
	log.Printf("localserver on %s", *addr)
	log.Fatal(http.ListenAndServe(*addr, cors(mux)))
}

type server struct {
	store   *localstack.Store
	reg     *campaigns.Registry
	sockets *sockets
}

// cors lets the Vite dev server (another origin) call the API.
func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// userFrom reads the user from a bearer or query token's sub claim.
func userFrom(token string) string {
	token = strings.TrimPrefix(token, "Bearer ")
	parts := strings.Split(token, ".")
	if len(parts) == 3 {
		if raw, err := base64.RawURLEncoding.DecodeString(parts[1]); err == nil {
			var claims struct {
				Sub string `json:"sub"`
			}
			if json.Unmarshal(raw, &claims) == nil && claims.Sub != "" {
				return claims.Sub
			}
		}
	}
	return "local-player"
}

func (s *server) listCampaigns(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, 200, map[string]any{"campaigns": s.reg.List()})
}

func (s *server) listGames(w http.ResponseWriter, r *http.Request) {
	games := []map[string]any{}
	for _, id := range s.store.Sessions() {
		save, err := s.store.GetGame(r.Context(), id)
		if err != nil {
			continue
		}
		name := ""
		if pd := save.PlayersData[save.OwnerID]; pd != nil {
			name = pd.Name
		}
		games = append(games, map[string]any{
			"session_id": id, "player_name": name, "ready": save.Ready, "title": save.Title,
			"theme": save.Theme, "quest_goal": save.QuestGoal, "conversation_count": save.ConversationCount, "total_tokens": save.TotalTokens,
		})
	}
	writeJSON(w, 200, map[string]any{"games": games, "user_quota": map[string]any{"tokens_used": 0, "token_limit": 0, "ai_enabled": true}})
}

// createGame is http-games' create for a campaign: the rules applied to what
// the player chose, then campaigns.StartSession.
func (s *server) createGame(w http.ResponseWriter, r *http.Request) {
	var body struct {
		CampaignID string `json:"campaign_id"`
		game.CharacterCreationData
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.CampaignID == "" {
		writeJSON(w, 400, map[string]string{"error": "campaign_id is required"})
		return
	}
	def, ok := s.reg.Get(body.CampaignID)
	if !ok {
		writeJSON(w, 400, map[string]string{"error": "unknown campaign"})
		return
	}
	creation, err := def.CharacterCreation.Apply(body.CharacterCreationData)
	if err != nil {
		writeJSON(w, 400, map[string]string{"error": err.Error()})
		return
	}
	g, history, err := campaigns.StartSession(r.Context(), def, game.NewSessionID(), userFrom(r.Header.Get("Authorization")), creation)
	if err != nil {
		writeJSON(w, 400, map[string]string{"error": err.Error()})
		return
	}
	if err := s.store.PutSession(g.ToSaveState(nil, history)); err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, 201, map[string]any{"session_id": g.ID, "ready": true, "preview_mode": false, "campaign_id": def.ID})
}

// getGame is http-games' load: the player's view of the world and the story.
func (s *server) getGame(w http.ResponseWriter, r *http.Request) {
	save, err := s.store.GetGame(r.Context(), r.PathValue("id"))
	if err != nil {
		writeJSON(w, 404, map[string]string{"error": "game not found"})
		return
	}
	g, err := game.FromSaveState(save)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}
	if save.PlayersData != nil {
		_, _ = g.LoadDnDCharacters(r.Context(), save.PlayersData)
	}
	writeJSON(w, 200, map[string]any{
		"session_id": save.SessionID, "ready": save.Ready,
		"state": g.BuildGameStateView(userFrom(r.Header.Get("Authorization")), save.ChatHistory), "campaign": g.BuildCampaignStateView(),
		"title": save.Title, "theme": save.Theme, "quest_goal": save.QuestGoal, "total_tokens": save.TotalTokens,
		"conversation_count": save.ConversationCount, "creation_params": save.CreationParams,
		"needs_character_reset": g.NeedsCharacterReset, "world_gen_logs": save.WorldGenLogs, "owner_id": save.OwnerID,
	})
}

func (s *server) deleteGame(w http.ResponseWriter, r *http.Request) {
	s.store.DeleteSession(r.PathValue("id"))
	w.WriteHeader(http.StatusNoContent)
}

// socket is API Gateway's WebSocket: $connect records the connection, each
// message is handed to the handler its action names (concurrently, as
// Lambda would run them), and $disconnect forgets it.
func (s *server) socket(w http.ResponseWriter, r *http.Request) {
	c, err := websocket.Accept(w, r, &websocket.AcceptOptions{InsecureSkipVerify: true})
	if err != nil {
		return
	}
	c.SetReadLimit(1 << 20)
	connID := fmt.Sprintf("conn-%d", time.Now().UnixNano())
	gameID := r.URL.Query().Get("gameId")
	s.store.Connect(connID, userFrom(r.URL.Query().Get("token")), gameID)
	s.sockets.add(connID, c)
	defer func() {
		s.sockets.remove(connID)
		s.store.Disconnect(connID)
		_ = c.CloseNow()
	}()
	ctx := context.Background()
	for {
		_, data, err := c.Read(r.Context())
		if err != nil {
			return
		}
		var msg struct {
			Action string `json:"action"`
		}
		_ = json.Unmarshal(data, &msg)
		req := events.APIGatewayWebsocketProxyRequest{
			Body:           string(data),
			RequestContext: events.APIGatewayWebsocketProxyRequestContext{ConnectionID: connID, RequestID: fmt.Sprintf("req-%d", time.Now().UnixNano())},
		}
		handler := wschat.Handler
		if msg.Action == "game_action" {
			handler = wsgameaction.Handler
		}
		go func() {
			started := time.Now()
			resp, err := handler(ctx, req)
			log.Printf("%s %s: status %d in %s %v", connID, msg.Action, resp.StatusCode, time.Since(started).Round(time.Millisecond), errOrEmpty(err))
		}()
	}
}

func errOrEmpty(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

// sockets is the API Gateway management API: frames to connections.
type sockets struct {
	mu    sync.Mutex
	conns map[string]*websocket.Conn
}

func newSockets() *sockets { return &sockets{conns: map[string]*websocket.Conn{}} }

func (s *sockets) add(id string, c *websocket.Conn) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.conns[id] = c
}

func (s *sockets) remove(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.conns, id)
}

func (s *sockets) Send(ctx context.Context, id string, f wsutil.Frame) error {
	s.mu.Lock()
	c, ok := s.conns[id]
	s.mu.Unlock()
	if !ok {
		return fmt.Errorf("gone: %s", id)
	}
	data, err := json.Marshal(f)
	if err != nil {
		return err
	}
	return c.Write(ctx, websocket.MessageText, data)
}

func (s *sockets) Broadcast(ctx context.Context, ids []string, f wsutil.Frame) ([]string, error) {
	var stale []string
	for _, id := range ids {
		if err := s.Send(ctx, id, f); err != nil {
			stale = append(stale, id)
		}
	}
	return stale, nil
}

func (s *sockets) SendDelta(ctx context.Context, id string, delta any) error {
	return s.Send(ctx, id, wsutil.Frame{Type: wsutil.FrameStateDelta, Payload: delta})
}

func (s *sockets) SendError(ctx context.Context, id, message string) error {
	return s.Send(ctx, id, wsutil.Frame{Type: wsutil.FrameError, Payload: map[string]string{"message": message}})
}

func (s *sockets) SendFullState(ctx context.Context, id string, state any) error {
	return s.Send(ctx, id, wsutil.Frame{Type: wsutil.FrameGameStateUpdate, Payload: state})
}
