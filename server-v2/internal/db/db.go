// Package db provides DynamoDB access for the game.
// All table names come from environment variables injected by Lambda.
package db

import (
	"context"
	"fmt"
	"os"

	dnd5echar "github.com/KirkDiggler/rpg-toolkit/rulebooks/dnd5e/character"
	dnd5emonster "github.com/KirkDiggler/rpg-toolkit/rulebooks/dnd5e/monster"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/rrochlin/an-amazing-adventure/internal/combat"
	"github.com/rrochlin/an-amazing-adventure/internal/game"
)

// Client wraps the DynamoDB client with table name config.
type Client struct {
	ddb              *dynamodb.Client
	sessionsTable    string
	connectionsTable string
	mutationsTable   string
	usersTable       string
	invitesTable     string
	membershipsTable string
}

// New creates a Client from the current AWS environment.
// All table names are optional at construction time — panics are deferred to
// the first method call that needs each table. This allows Lambdas to omit
// env vars they don't use.
func New(ctx context.Context) (*Client, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}
	return &Client{
		ddb:              dynamodb.NewFromConfig(cfg),
		sessionsTable:    os.Getenv("SESSIONS_TABLE"),    // checked at use
		connectionsTable: os.Getenv("CONNECTIONS_TABLE"), // checked at use
		mutationsTable:   os.Getenv("MUTATIONS_TABLE"),   // checked at use
		usersTable:       os.Getenv("USERS_TABLE"),       // checked at use
		invitesTable:     os.Getenv("INVITES_TABLE"),     // checked at use
		membershipsTable: os.Getenv("MEMBERSHIPS_TABLE"), // checked at use
	}, nil
}

// requireSessionsTable panics with a clear message if SESSIONS_TABLE was not set.
func (c *Client) requireSessionsTable() {
	if c.sessionsTable == "" {
		panic("required env var SESSIONS_TABLE is not set")
	}
}

// requireConnectionsTable panics with a clear message if CONNECTIONS_TABLE was
// not set. Called at the top of every method that touches that table.
func (c *Client) requireConnectionsTable() {
	if c.connectionsTable == "" {
		panic("required env var CONNECTIONS_TABLE is not set")
	}
}

// requireMutationsTable panics with a clear message if MUTATIONS_TABLE was not set.
func (c *Client) requireMutationsTable() {
	if c.mutationsTable == "" {
		panic("required env var MUTATIONS_TABLE is not set")
	}
}

// requireUsersTable panics with a clear message if USERS_TABLE was not set.
func (c *Client) requireUsersTable() {
	if c.usersTable == "" {
		panic("required env var USERS_TABLE is not set")
	}
}

// requireInvitesTable panics with a clear message if INVITES_TABLE was not set.
func (c *Client) requireInvitesTable() {
	if c.invitesTable == "" {
		panic("required env var INVITES_TABLE is not set")
	}
}

// requireMembershipsTable panics with a clear message if MEMBERSHIPS_TABLE was not set.
func (c *Client) requireMembershipsTable() {
	if c.membershipsTable == "" {
		panic("required env var MEMBERSHIPS_TABLE is not set")
	}
}

// -------------------------------------------------------------------
// Game sessions
// -------------------------------------------------------------------

// saveStateDB is a DynamoDB-specific wrapper around game.SaveState that
// overrides the key fields to marshal as Binary (B), matching the table schema.
type saveStateDB struct {
	SessionID            BinaryID                     `dynamodbav:"session_id"`
	OwnerID              BinaryID                     `dynamodbav:"owner_id,omitempty"`
	UserID               BinaryID                     `dynamodbav:"user_id"`
	SchemaVersion        int                          `dynamodbav:"schema_version"`
	Version              int                          `dynamodbav:"version"`
	Players              map[string]game.Character    `dynamodbav:"players,omitempty"`
	PlayersData          map[string]*dnd5echar.Data   `dynamodbav:"players_data,omitempty"` // v3+
	Player               game.Character               `dynamodbav:"player,omitempty"`       // v1 compat
	PartySize            int                          `dynamodbav:"party_size,omitempty"`
	InviteCode           string                       `dynamodbav:"invite_code,omitempty"`
	Rooms                []game.Area                  `dynamodbav:"rooms"`
	Items                []game.Item                  `dynamodbav:"items"`
	NPCs                 []game.Character             `dynamodbav:"npcs"`
	Narrative            []game.NarrativeMessage      `dynamodbav:"narrative"`
	ChatHistory          []game.ChatMessage           `dynamodbav:"chat_history"`
	Ready                bool                         `dynamodbav:"ready"`
	Title                string                       `dynamodbav:"title,omitempty"`
	Theme                string                       `dynamodbav:"theme,omitempty"`
	QuestGoal            string                       `dynamodbav:"quest_goal,omitempty"`
	TotalTokens          int                          `dynamodbav:"total_tokens,omitempty"`
	ConversationCount    int                          `dynamodbav:"conversation_count,omitempty"`
	CreationParams       game.CharacterCreationData   `dynamodbav:"creation_params,omitempty"`        // v3+
	LegacyCreationParams game.AdventureCreationParams `dynamodbav:"legacy_creation_params,omitempty"` // v1/v2

	// Combat state (v3+) — previously missing from saveStateDB, fixed in v4.
	RoomMonsters         map[string][]*dnd5emonster.Data `dynamodbav:"room_monsters,omitempty"`
	PendingCombatContext string                          `dynamodbav:"pending_combat_context,omitempty"`
	InitiativeOrder      []combat.InitiativeEntry        `dynamodbav:"initiative_order,omitempty"`

	// Dungeon layout (v4+)
	DungeonData *game.DungeonData `dynamodbav:"dungeon_data,omitempty"`
}

func toDBState(s game.SaveState) saveStateDB {
	return saveStateDB{
		SessionID:            BinaryID(s.SessionID),
		OwnerID:              BinaryID(s.OwnerID),
		UserID:               BinaryID(s.UserID),
		SchemaVersion:        s.SchemaVersion,
		Version:              s.Version,
		Players:              s.Players,
		PlayersData:          s.PlayersData,
		Player:               s.Player,
		PartySize:            s.PartySize,
		InviteCode:           s.InviteCode,
		Rooms:                s.Rooms,
		Items:                s.Items,
		NPCs:                 s.NPCs,
		Narrative:            s.Narrative,
		ChatHistory:          s.ChatHistory,
		Ready:                s.Ready,
		Title:                s.Title,
		Theme:                s.Theme,
		QuestGoal:            s.QuestGoal,
		TotalTokens:          s.TotalTokens,
		ConversationCount:    s.ConversationCount,
		CreationParams:       s.CreationParams,
		LegacyCreationParams: s.LegacyCreationParams,
		RoomMonsters:         s.RoomMonsters,
		PendingCombatContext: s.PendingCombatContext,
		InitiativeOrder:      s.InitiativeOrder,
		DungeonData:          s.DungeonData,
	}
}

func fromDBState(d saveStateDB) game.SaveState {
	return game.SaveState{
		SessionID:            string(d.SessionID),
		OwnerID:              string(d.OwnerID),
		UserID:               string(d.UserID),
		SchemaVersion:        d.SchemaVersion,
		Version:              d.Version,
		Players:              d.Players,
		PlayersData:          d.PlayersData,
		Player:               d.Player,
		PartySize:            d.PartySize,
		InviteCode:           d.InviteCode,
		Rooms:                d.Rooms,
		Items:                d.Items,
		NPCs:                 d.NPCs,
		Narrative:            d.Narrative,
		ChatHistory:          d.ChatHistory,
		Ready:                d.Ready,
		Title:                d.Title,
		Theme:                d.Theme,
		QuestGoal:            d.QuestGoal,
		TotalTokens:          d.TotalTokens,
		ConversationCount:    d.ConversationCount,
		CreationParams:       d.CreationParams,
		LegacyCreationParams: d.LegacyCreationParams,
		RoomMonsters:         d.RoomMonsters,
		PendingCombatContext: d.PendingCombatContext,
		InitiativeOrder:      d.InitiativeOrder,
		DungeonData:          d.DungeonData,
	}
}

// PutGame writes a SaveState to DynamoDB using optimistic locking.
// If the current version in the DB doesn't match state.Version, the write
// is rejected with a ConditionalCheckFailedException — caller should retry.
func (c *Client) PutGame(ctx context.Context, state game.SaveState) error {
	c.requireSessionsTable()
	item, err := attributevalue.MarshalMap(toDBState(state))
	if err != nil {
		return fmt.Errorf("marshal save state: %w", err)
	}

	var condExpr *string
	var exprAttrVals map[string]types.AttributeValue

	if state.Version > 0 {
		prevVersion, _ := attributevalue.Marshal(state.Version - 1)
		condExpr = aws.String("version = :prev")
		exprAttrVals = map[string]types.AttributeValue{":prev": prevVersion}
	} else {
		condExpr = aws.String("attribute_not_exists(session_id)")
	}

	_, err = c.ddb.PutItem(ctx, &dynamodb.PutItemInput{
		TableName:                 aws.String(c.sessionsTable),
		Item:                      item,
		ConditionExpression:       condExpr,
		ExpressionAttributeValues: exprAttrVals,
	})
	if err != nil {
		return fmt.Errorf("put game: %w", err)
	}
	return nil
}

// GetGame loads and deserialises a full SaveState by session UUID string.
func (c *Client) GetGame(ctx context.Context, sessionID string) (game.SaveState, error) {
	c.requireSessionsTable()
	out, err := c.ddb.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(c.sessionsTable),
		Key:       sessionKey(sessionID),
	})
	if err != nil {
		return game.SaveState{}, fmt.Errorf("get game: %w", err)
	}
	if out.Item == nil {
		return game.SaveState{}, fmt.Errorf("game not found: %s", sessionID)
	}
	var d saveStateDB
	if err := attributevalue.UnmarshalMap(out.Item, &d); err != nil {
		return game.SaveState{}, fmt.Errorf("unmarshal game: %w", err)
	}
	return fromDBState(d), nil
}

// GetGameReady does a projection-only read to check the ready flag cheaply.
func (c *Client) GetGameReady(ctx context.Context, sessionID string) (bool, error) {
	c.requireSessionsTable()
	out, err := c.ddb.GetItem(ctx, &dynamodb.GetItemInput{
		TableName:            aws.String(c.sessionsTable),
		Key:                  sessionKey(sessionID),
		ProjectionExpression: aws.String("#r"),
		ExpressionAttributeNames: map[string]string{
			"#r": "ready",
		},
	})
	if err != nil {
		return false, fmt.Errorf("get game ready: %w", err)
	}
	if out.Item == nil {
		return false, fmt.Errorf("game not found: %s", sessionID)
	}
	var partial struct {
		Ready bool `dynamodbav:"ready"`
	}
	if err := attributevalue.UnmarshalMap(out.Item, &partial); err != nil {
		return false, err
	}
	return partial.Ready, nil
}

// ListGames returns all SaveState summaries for a given user ID.
func (c *Client) ListGames(ctx context.Context, userID string) ([]game.SaveState, error) {
	c.requireSessionsTable()
	out, err := c.ddb.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(c.sessionsTable),
		IndexName:              aws.String("user-sessions-index"),
		KeyConditionExpression: aws.String("user_id = :uid"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":uid": binaryIDVal(userID),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("list games: %w", err)
	}
	saves := make([]game.SaveState, 0, len(out.Items))
	for _, item := range out.Items {
		var d saveStateDB
		if err := attributevalue.UnmarshalMap(item, &d); err != nil {
			return nil, fmt.Errorf("unmarshal game list item: %w", err)
		}
		saves = append(saves, fromDBState(d))
	}
	return saves, nil
}

// DeleteGame removes a session record owned by userID.
func (c *Client) DeleteGame(ctx context.Context, sessionID, userID string) error {
	c.requireSessionsTable()
	_, err := c.ddb.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName:           aws.String(c.sessionsTable),
		Key:                 sessionKey(sessionID),
		ConditionExpression: aws.String("user_id = :uid"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":uid": binaryIDVal(userID),
		},
	})
	if err != nil {
		return fmt.Errorf("delete game: %w", err)
	}
	return nil
}

func sessionKey(sessionID string) map[string]types.AttributeValue {
	return map[string]types.AttributeValue{"session_id": binaryIDVal(sessionID)}
}

// ListGamesByOwner returns session IDs owned by userID (via user-sessions-index GSI).
func (c *Client) ListGamesByOwner(ctx context.Context, userID string) ([]string, error) {
	c.requireSessionsTable()
	out, err := c.ddb.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(c.sessionsTable),
		IndexName:              aws.String("user-sessions-index"),
		KeyConditionExpression: aws.String("user_id = :uid"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":uid": binaryIDVal(userID),
		},
		ProjectionExpression: aws.String("session_id"),
	})
	if err != nil {
		return nil, fmt.Errorf("ListGamesByOwner: %w", err)
	}
	ids := make([]string, 0, len(out.Items))
	for _, item := range out.Items {
		var d struct {
			SessionID BinaryID `dynamodbav:"session_id"`
		}
		if err := attributevalue.UnmarshalMap(item, &d); err != nil {
			continue
		}
		ids = append(ids, string(d.SessionID))
	}
	return ids, nil
}

// BatchGetSessions retrieves multiple sessions by their IDs in a single batch request.
// Sessions not found in DynamoDB are silently skipped.
func (c *Client) BatchGetSessions(ctx context.Context, sessionIDs []string) ([]game.SaveState, error) {
	c.requireSessionsTable()
	if len(sessionIDs) == 0 {
		return nil, nil
	}
	// DynamoDB BatchGetItem limit is 100 items per call — chunk if needed.
	const batchSize = 100
	var allStates []game.SaveState
	for i := 0; i < len(sessionIDs); i += batchSize {
		end := i + batchSize
		if end > len(sessionIDs) {
			end = len(sessionIDs)
		}
		chunk := sessionIDs[i:end]
		keys := make([]map[string]types.AttributeValue, 0, len(chunk))
		for _, id := range chunk {
			keys = append(keys, sessionKey(id))
		}
		out, err := c.ddb.BatchGetItem(ctx, &dynamodb.BatchGetItemInput{
			RequestItems: map[string]types.KeysAndAttributes{
				c.sessionsTable: {Keys: keys},
			},
		})
		if err != nil {
			return nil, fmt.Errorf("BatchGetSessions: %w", err)
		}
		for _, item := range out.Responses[c.sessionsTable] {
			var d saveStateDB
			if err := attributevalue.UnmarshalMap(item, &d); err != nil {
				continue
			}
			allStates = append(allStates, fromDBState(d))
		}
	}
	return allStates, nil
}

// -------------------------------------------------------------------
// Mutation log
// -------------------------------------------------------------------

// mutationEntryDB is the DynamoDB wire format for a MutationEntry.
// SessionID must be Binary (B) to match the mutations table key schema.
type mutationEntryDB struct {
	SessionID string         `dynamodbav:"session_id"` // B via BinaryID marshal below
	Ts        int64          `dynamodbav:"ts"`
	Turn      int            `dynamodbav:"turn"`
	Tool      string         `dynamodbav:"tool"`
	Input     map[string]any `dynamodbav:"input"`
	Result    string         `dynamodbav:"result"`
}

// PutMutation writes a single MutationEntry to the mutations table.
// SessionID is marshaled as Binary (B) to match the table's key schema.
func (c *Client) PutMutation(ctx context.Context, entry game.MutationEntry) error {
	c.requireMutationsTable()
	row := mutationEntryDB{
		SessionID: entry.SessionID,
		Ts:        entry.Ts,
		Turn:      entry.Turn,
		Tool:      entry.Tool,
		Input:     entry.Input,
		Result:    entry.Result,
	}
	item, err := attributevalue.MarshalMap(row)
	if err != nil {
		return fmt.Errorf("marshal mutation entry: %w", err)
	}
	// Override SessionID key attribute to Binary type
	item["session_id"] = binaryIDVal(entry.SessionID)
	_, err = c.ddb.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(c.mutationsTable),
		Item:      item,
	})
	if err != nil {
		return fmt.Errorf("put mutation: %w", err)
	}
	return nil
}

// -------------------------------------------------------------------
// WebSocket connections
// -------------------------------------------------------------------

// Connection represents a live WebSocket connection record.
// UserID marshals as Binary to match the connections table GSI key type.
type Connection struct {
	ConnectionID string   `dynamodbav:"connection_id"`
	UserID       BinaryID `dynamodbav:"user_id"`
	GameID       string   `dynamodbav:"game_id"`
	ExpiresAt    int64    `dynamodbav:"expires_at"` // Unix epoch seconds; TTL field
	Streaming    bool     `dynamodbav:"streaming"`  // true while AI is generating
}

// PutConnection writes or replaces a connection record.
// ExpiresAt should be ~24h from now so DynamoDB TTL auto-cleans stale records.
func (c *Client) PutConnection(ctx context.Context, conn Connection) error {
	c.requireConnectionsTable()
	item, err := attributevalue.MarshalMap(conn)
	if err != nil {
		return err
	}
	_, err = c.ddb.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(c.connectionsTable),
		Item:      item,
	})
	return err
}

// GetConnection retrieves a connection by its API Gateway connection ID.
func (c *Client) GetConnection(ctx context.Context, connectionID string) (Connection, error) {
	c.requireConnectionsTable()
	v, _ := attributevalue.Marshal(connectionID)
	out, err := c.ddb.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(c.connectionsTable),
		Key:       map[string]types.AttributeValue{"connection_id": v},
	})
	if err != nil {
		return Connection{}, err
	}
	if out.Item == nil {
		return Connection{}, fmt.Errorf("connection not found: %s", connectionID)
	}
	var conn Connection
	if err := attributevalue.UnmarshalMap(out.Item, &conn); err != nil {
		return Connection{}, err
	}
	return conn, nil
}

// DeleteConnection removes a connection record on disconnect.
func (c *Client) DeleteConnection(ctx context.Context, connectionID string) error {
	c.requireConnectionsTable()
	v, _ := attributevalue.Marshal(connectionID)
	_, err := c.ddb.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(c.connectionsTable),
		Key:       map[string]types.AttributeValue{"connection_id": v},
	})
	return err
}

// DeleteUserConnections removes all connection records for a given user ID.
// Deprecated: use DeleteUserConnectionForGame for scoped cleanup in party sessions.
func (c *Client) DeleteUserConnections(ctx context.Context, userID string) error {
	c.requireConnectionsTable()
	out, err := c.ddb.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(c.connectionsTable),
		IndexName:              aws.String("user-connections-index"),
		KeyConditionExpression: aws.String("user_id = :uid"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":uid": binaryIDVal(userID),
		},
	})
	if err != nil {
		return err
	}
	for _, item := range out.Items {
		connIDAttr, ok := item["connection_id"]
		if !ok {
			continue
		}
		_, err := c.ddb.DeleteItem(ctx, &dynamodb.DeleteItemInput{
			TableName: aws.String(c.connectionsTable),
			Key: map[string]types.AttributeValue{
				"connection_id": connIDAttr,
			},
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// DeleteUserConnectionForGame removes the stale connection record for a specific
// (userID, gameID) pair. Called on $connect to allow re-connects without
// evicting the user's connections to other sessions.
func (c *Client) DeleteUserConnectionForGame(ctx context.Context, userID, gameID string) error {
	c.requireConnectionsTable()
	out, err := c.ddb.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(c.connectionsTable),
		IndexName:              aws.String("user-connections-index"),
		KeyConditionExpression: aws.String("user_id = :uid"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":uid": binaryIDVal(userID),
		},
	})
	if err != nil {
		return fmt.Errorf("DeleteUserConnectionForGame query: %w", err)
	}
	for _, item := range out.Items {
		var conn Connection
		if err := attributevalue.UnmarshalMap(item, &conn); err != nil {
			continue
		}
		if conn.GameID != gameID {
			continue
		}
		connIDAttr, ok := item["connection_id"]
		if !ok {
			continue
		}
		_, err := c.ddb.DeleteItem(ctx, &dynamodb.DeleteItemInput{
			TableName: aws.String(c.connectionsTable),
			Key:       map[string]types.AttributeValue{"connection_id": connIDAttr},
		})
		if err != nil {
			return fmt.Errorf("DeleteUserConnectionForGame delete: %w", err)
		}
	}
	return nil
}

// GetConnectionsByGameID returns all active connections for a game session.
// Uses the game-connections-index GSI (game_id is a plain String attribute).
func (c *Client) GetConnectionsByGameID(ctx context.Context, gameID string) ([]Connection, error) {
	c.requireConnectionsTable()
	gameIDVal, _ := attributevalue.Marshal(gameID)
	out, err := c.ddb.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(c.connectionsTable),
		IndexName:              aws.String("game-connections-index"),
		KeyConditionExpression: aws.String("game_id = :gid"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":gid": gameIDVal,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("GetConnectionsByGameID: %w", err)
	}
	conns := make([]Connection, 0, len(out.Items))
	for _, item := range out.Items {
		var conn Connection
		if err := attributevalue.UnmarshalMap(item, &conn); err != nil {
			continue
		}
		conns = append(conns, conn)
	}
	return conns, nil
}

// GetConnectionByUserID returns the most recent active connection for a user,
// or an error if none exists. Used by world-gen to push progress frames.
func (c *Client) GetConnectionByUserID(ctx context.Context, userID string) (Connection, error) {
	c.requireConnectionsTable()
	out, err := c.ddb.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(c.connectionsTable),
		IndexName:              aws.String("user-connections-index"),
		KeyConditionExpression: aws.String("user_id = :uid"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":uid": binaryIDVal(userID),
		},
		Limit: aws.Int32(1), // one connection per user enforced on $connect
	})
	if err != nil {
		return Connection{}, fmt.Errorf("get connection by user: %w", err)
	}
	if len(out.Items) == 0 {
		return Connection{}, fmt.Errorf("no active connection for user %s", userID)
	}
	var conn Connection
	if err := attributevalue.UnmarshalMap(out.Items[0], &conn); err != nil {
		return Connection{}, fmt.Errorf("unmarshal connection: %w", err)
	}
	return conn, nil
}

// SetStreaming atomically sets the streaming flag on a connection record.
func (c *Client) SetStreaming(ctx context.Context, connectionID string, streaming bool) error {
	c.requireConnectionsTable()
	v, _ := attributevalue.Marshal(connectionID)
	sv, _ := attributevalue.Marshal(streaming)
	update := expression.Set(expression.Name("streaming"), expression.Value(streaming))
	expr, err := expression.NewBuilder().WithUpdate(update).Build()
	if err != nil {
		return err
	}
	_, err = c.ddb.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName:                 aws.String(c.connectionsTable),
		Key:                       map[string]types.AttributeValue{"connection_id": v},
		UpdateExpression:          expr.Update(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
	})
	_ = sv
	return err
}
