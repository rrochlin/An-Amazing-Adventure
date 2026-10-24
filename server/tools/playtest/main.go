// Command playtest plays a campaign locally through the real ws-chat and
// ws-game-action handlers: the session, its connection and its traces live in
// memory (saved to a directory between commands), every frame is recorded,
// and the story calls Bedrock through the real AI client.
//
//	go run ./tools/playtest -dir DIR new [-campaign briarwell] [-name Wren]
//	go run ./tools/playtest -dir DIR say "look around"
//	go run ./tools/playtest -dir DIR choose 2        (a pending choice: number, ID or text)
//	go run ./tools/playtest -dir DIR act move north  (a game action)
//	go run ./tools/playtest -dir DIR state
//	go run ./tools/playtest -dir DIR rewind 3        (back to just after turn 3)
//	go run ./tools/playtest -dir DIR views           (what the client got after each turn, for a UI harness)
//	go run ./tools/playtest -dir DIR -script FILE    (lines of the above, # for comments)
//
// Bedrock needs credentials: AWS_PROFILE=aa-debug AWS_REGION=us-west-2.
// Each turn is printed and appended to DIR/report.md; DIR/traces/NNN.json
// holds the full turn trace (model prompts and outputs included).
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"

	"github.com/rrochlin/an-amazing-adventure/internal/ai"
	"github.com/rrochlin/an-amazing-adventure/internal/campaigns"
	"github.com/rrochlin/an-amazing-adventure/internal/game"
	"github.com/rrochlin/an-amazing-adventure/internal/lambdas/wschat"
	"github.com/rrochlin/an-amazing-adventure/internal/lambdas/wsgameaction"
)

const (
	userID = "player"
	connID = "conn-local"
)

func main() {
	dir := flag.String("dir", "", "directory holding the session between commands (required)")
	campaignID := flag.String("campaign", "briarwell", "campaign for new")
	name := flag.String("name", "Wren", "player name for new")
	script := flag.String("script", "", "file of commands to play in order")
	flag.Parse()
	if *dir == "" {
		log.Fatal("-dir is required")
	}
	// The handlers check the table names they write to; mutations stay in
	// memory here.
	if os.Getenv("MUTATIONS_TABLE") == "" {
		must(os.Setenv("MUTATIONS_TABLE", "playtest-mutations"))
	}
	// The handlers log a lot; the report is the output.
	log.SetOutput(discardUnless(os.Getenv("PLAYTEST_LOGS") != ""))

	p := &player{dir: *dir}
	ctx := context.Background()
	var commands [][]string
	if *script != "" {
		raw, err := os.ReadFile(*script)
		must(err)
		for _, line := range strings.Split(string(raw), "\n") {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			commands = append(commands, splitCommand(line))
		}
	} else {
		commands = [][]string{flag.Args()}
	}
	for _, args := range commands {
		if len(args) == 0 {
			log.Fatal("no command: new, say, choose, act or state")
		}
		if args[0] == "new" {
			must(p.newGame(*campaignID, *name))
			continue
		}
		must(p.load())
		switch args[0] {
		case "state":
			fmt.Print(p.stateSummary())
		case "rewind":
			must(p.rewind(args[1]))
		case "views":
			must(p.views())
		case "say", "choose", "act":
			must(p.turn(ctx, args[0], strings.Join(args[1:], " ")))
		default:
			log.Fatalf("unknown command %q", args[0])
		}
	}
}

// splitCommand splits "say look around" into the verb and the rest.
func splitCommand(line string) []string {
	verb, rest, _ := strings.Cut(line, " ")
	return []string{verb, strings.TrimSpace(rest)}
}

type player struct {
	dir   string
	store *memStore
	n     int // turns played
}

func (p *player) path(name string) string { return filepath.Join(p.dir, name) }

func (p *player) newGame(campaignID, name string) error {
	reg, err := campaigns.LoadEmbeddedRegistry()
	if err != nil {
		return err
	}
	def, ok := reg.Get(campaignID)
	if !ok {
		return fmt.Errorf("unknown campaign %q", campaignID)
	}
	// As http-games creates a campaign session.
	creation := game.CharacterCreationData{Name: name}
	g, history, err := campaigns.BootstrapGame(def, game.NewSessionID(), userID, game.NewCharacter(name, ""), creation)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(p.path("traces"), 0o755); err != nil {
		return err
	}
	p.store = newMemStore(g.ToSaveState(nil, history))
	if err := p.save(); err != nil {
		return err
	}
	header := fmt.Sprintf("# Playtest: %s\n\nSession %s, player %s, %s\n\n", def.Title, g.ID, name, time.Now().UTC().Format(time.RFC3339))
	if err := os.WriteFile(p.path("report.md"), []byte(header), 0o644); err != nil {
		return err
	}
	opening := "## Opening\n\n" + renderHistory(history) + "\n" + p.stateSummary() + "\n"
	fmt.Print(opening)
	return appendFile(p.path("report.md"), opening)
}

type sessionFile struct {
	Turns int            `json:"turns"`
	Save  game.SaveState `json:"save"`
}

func (p *player) load() error {
	raw, err := os.ReadFile(p.path("session.json"))
	if err != nil {
		return fmt.Errorf("no session in %s (run new first): %w", p.dir, err)
	}
	var f sessionFile
	if err := json.Unmarshal(raw, &f); err != nil {
		return err
	}
	p.n = f.Turns
	p.store = newMemStore(f.Save)
	return nil
}

func (p *player) save() error {
	s, err := p.store.GetGame(context.Background(), "")
	if err != nil {
		return err
	}
	raw, err := json.MarshalIndent(sessionFile{Turns: p.n, Save: s}, "", " ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(p.path("session.json"), raw, 0o644); err != nil {
		return err
	}
	// A checkpoint per turn, for rewind.
	if err := os.MkdirAll(p.path("snapshots"), 0o755); err != nil {
		return err
	}
	return os.WriteFile(p.path(filepath.Join("snapshots", fmt.Sprintf("%03d.json", p.n))), raw, 0o644)
}

// views writes DIR/views/NNN.json from each snapshot: the game and campaign
// state the client is sent, so a UI harness can show every step of a run.
func (p *player) views() error {
	snaps, err := filepath.Glob(p.path(filepath.Join("snapshots", "*.json")))
	if err != nil {
		return err
	}
	if err := os.MkdirAll(p.path("views"), 0o755); err != nil {
		return err
	}
	for _, snap := range snaps {
		raw, err := os.ReadFile(snap)
		if err != nil {
			return err
		}
		var f sessionFile
		if err := json.Unmarshal(raw, &f); err != nil {
			return err
		}
		g, err := game.FromSaveState(f.Save)
		if err != nil {
			return err
		}
		out, err := json.Marshal(map[string]any{
			"game":     g.BuildGameStateView(userID, nil),
			"campaign": g.BuildCampaignStateView(),
		})
		if err != nil {
			return err
		}
		if err := os.WriteFile(p.path(filepath.Join("views", filepath.Base(snap))), out, 0o644); err != nil {
			return err
		}
	}
	fmt.Printf("%d views in %s\n", len(snaps), p.path("views"))
	return nil
}

// rewind restores the session as it was just after turn n.
func (p *player) rewind(arg string) error {
	n, err := strconv.Atoi(strings.TrimSpace(arg))
	if err != nil {
		return fmt.Errorf("rewind takes a turn number: %w", err)
	}
	raw, err := os.ReadFile(p.path(filepath.Join("snapshots", fmt.Sprintf("%03d.json", n))))
	if err != nil {
		return err
	}
	if err := os.WriteFile(p.path("session.json"), raw, 0o644); err != nil {
		return err
	}
	note := fmt.Sprintf("\n## Rewound to after turn %d\n\n", n)
	fmt.Print(note)
	return appendFile(p.path("report.md"), note)
}

// turn plays one command through the handler the client would call.
func (p *player) turn(ctx context.Context, verb, arg string) error {
	models, err := ai.New(ctx)
	if err != nil {
		return err
	}
	sender := &recorder{}
	wschat.UseClients(
		func(context.Context) (wschat.DB, error) { return p.store, nil },
		func(context.Context) (wschat.AI, error) { return models, nil },
		func(context.Context) (wschat.Sender, error) { return sender, nil },
	)
	wsgameaction.UseClients(
		func(context.Context) (wsgameaction.DB, error) { return p.store, nil },
		func(context.Context) (wsgameaction.Sender, error) { return sender, nil },
	)

	before, err := p.store.GetGame(ctx, "")
	if err != nil {
		return err
	}
	var body string
	call := wschat.Handler
	label := verb + " " + arg
	switch verb {
	case "say":
		body = mustJSON(map[string]string{"action": "chat", "content": arg})
	case "choose":
		id, text, err := pickChoice(before, arg)
		if err != nil {
			return err
		}
		label = fmt.Sprintf("choose %q", text)
		body = mustJSON(map[string]string{"action": "game_action", "sub_action": "dialogue_choice", "payload": strconv.Itoa(id)})
		call = wsgameaction.Handler
	case "act":
		sub, payload, _ := strings.Cut(arg, " ")
		body = mustJSON(map[string]string{"action": "game_action", "sub_action": sub, "payload": payload})
		call = wsgameaction.Handler
	}

	p.store.traces = nil
	started := time.Now()
	resp, err := call(ctx, events.APIGatewayWebsocketProxyRequest{
		Body:           body,
		RequestContext: events.APIGatewayWebsocketProxyRequestContext{ConnectionID: connID, RequestID: fmt.Sprintf("req-%d", p.n+1)},
	})
	elapsed := time.Since(started)
	if err != nil {
		return err
	}
	p.n++
	after, err := p.store.GetGame(ctx, "")
	if err != nil {
		return err
	}

	var b strings.Builder
	fmt.Fprintf(&b, "## Turn %d: %s\n\n", p.n, label)
	fmt.Fprintf(&b, "- handler status %d, %.1fs\n", resp.StatusCode, elapsed.Seconds())
	fmt.Fprintf(&b, "- frames: %s\n", sender.summary())
	for _, tr := range p.store.traces {
		raw, _ := json.MarshalIndent(tr, "", " ")
		name := fmt.Sprintf("%03d.json", p.n)
		_ = os.WriteFile(p.path(filepath.Join("traces", name)), raw, 0o644)
		for _, c := range tr.Calls {
			fmt.Fprintf(&b, "- model call %s: %.1fs, %d in / %d out, stop %s\n", c.Role, float64(c.DurationMs)/1000, c.InputTok, c.OutputTok, c.StopReason)
		}
		if tr.Decision != nil {
			fmt.Fprintf(&b, "- decision: `%s`\n", compact(tr.Decision))
		}
		for _, e := range tr.Errors {
			fmt.Fprintf(&b, "- **trace error:** %s\n", e)
		}
	}
	for _, e := range sender.errors {
		fmt.Fprintf(&b, "- **error frame:** %s\n", e)
	}
	if streamed := sender.streamed(); streamed != "" {
		fmt.Fprintf(&b, "\n**Streamed to the player:**\n\n%s\n", quote(streamed))
	}
	if added := after.ChatHistory[min(len(before.ChatHistory), len(after.ChatHistory)):]; len(added) > 0 {
		fmt.Fprintf(&b, "\n**Added to the log:**\n\n%s", renderHistory(added))
	}
	fmt.Fprintf(&b, "\n**Changes:**\n\n%s\n", diff(before, after))
	fmt.Fprintf(&b, "%s\n", p.stateFor(after))
	fmt.Print(b.String())
	if err := appendFile(p.path("report.md"), b.String()); err != nil {
		return err
	}
	return p.save()
}

// pickChoice finds a pending choice by number (1-based), ID or text.
func pickChoice(s game.SaveState, arg string) (int, string, error) {
	if s.Campaign == nil || s.Campaign.ActiveDialogue == nil || len(s.Campaign.ActiveDialogue.PendingChoices) == 0 {
		return 0, "", fmt.Errorf("no choice is pending")
	}
	choices := s.Campaign.ActiveDialogue.PendingChoices
	if n, err := strconv.Atoi(arg); err == nil && n >= 1 && n <= len(choices) {
		return choices[n-1].ID, choices[n-1].Text, nil
	}
	for _, c := range choices {
		if strings.Contains(strings.ToLower(c.Text), strings.ToLower(arg)) {
			return c.ID, c.Text, nil
		}
	}
	return 0, "", fmt.Errorf("no pending choice matches %q", arg)
}

func (p *player) stateSummary() string {
	s, err := p.store.GetGame(context.Background(), "")
	if err != nil {
		return err.Error()
	}
	return p.stateFor(s)
}

// stateFor is what the player can see: where they are, who is there, what
// they carry, the scene, the objectives and any choice waiting.
func (p *player) stateFor(s game.SaveState) string {
	g, err := game.FromSaveState(s)
	if err != nil {
		return err.Error()
	}
	view := g.BuildGameStateView(userID, nil)
	var b strings.Builder
	b.WriteString("**Now:**\n\n")
	room := view.CurrentRoom
	people := []string{}
	for _, o := range room.Occupants {
		people = append(people, o.Name)
	}
	items := []string{}
	for _, it := range room.Items {
		items = append(items, it.Name)
	}
	fmt.Fprintf(&b, "- room: %s (%s); people: %s; items: %s\n", room.Name, room.ID, orNone(people), orNone(items))
	carried := []string{}
	for _, it := range view.Self.Inventory {
		carried = append(carried, it.Name)
	}
	fmt.Fprintf(&b, "- carrying: %s\n", orNone(carried))
	if c := g.BuildCampaignStateView(); c != nil {
		scene := c.ActiveNodeID
		if c.ActiveSceneMap != nil {
			scene += ", indoors: " + c.ActiveSceneMap.Name
		}
		fmt.Fprintf(&b, "- scene: %s (%s); objective: %s\n", c.ActiveNodeTitle, scene, c.CurrentObjective)
		for _, o := range c.ActiveObjectives {
			fmt.Fprintf(&b, "  - objective %s: %s\n", o.ID, o.Status)
		}
		if len(c.Labels) > 0 || len(c.BoolFlags) > 0 || len(c.Counters) > 0 {
			fmt.Fprintf(&b, "- labels %v flags %v counters %v\n", c.Labels, c.BoolFlags, c.Counters)
		}
		if d := c.ActiveDialogue; d != nil && len(d.PendingChoices) > 0 {
			b.WriteString("- **choices:**\n")
			for i, ch := range d.PendingChoices {
				fmt.Fprintf(&b, "  %d. %s\n", i+1, ch.Text)
			}
		}
		if c.RunStatus != "" {
			fmt.Fprintf(&b, "- run: %s\n", c.RunStatus)
		}
	}
	return b.String()
}

func renderHistory(h []game.ChatMessage) string {
	var b strings.Builder
	for _, m := range h {
		who := m.Type
		events := []string{}
		for _, e := range m.Events {
			events = append(events, string(e.Type))
		}
		fmt.Fprintf(&b, "- [%s] %s", who, strings.ReplaceAll(strings.TrimSpace(m.Content), "\n", " / "))
		if len(events) > 0 {
			fmt.Fprintf(&b, " _(events: %s)_", strings.Join(events, ", "))
		}
		b.WriteString("\n")
	}
	return b.String()
}

// diff lists what changed in the world between two saves.
func diff(before, after game.SaveState) string {
	var lines []string
	gb, _ := game.FromSaveState(before)
	ga, _ := game.FromSaveState(after)
	if gb == nil || ga == nil {
		return "- (could not load saves)\n"
	}
	pb, _ := gb.GetPlayerCharacter(userID)
	pa, _ := ga.GetPlayerCharacter(userID)
	if pb.LocationID != pa.LocationID {
		lines = append(lines, fmt.Sprintf("player moved %s → %s", pb.LocationID, pa.LocationID))
	}
	// Counted by name: two copper coins are two.
	inv := func(g *game.Game, c game.Character) map[string]int {
		m := map[string]int{}
		for _, id := range c.Inventory {
			if it, ok := g.Items[id]; ok {
				m[it.Name]++
			} else {
				m[id+" (missing item)"]++
			}
		}
		return m
	}
	ib, ia := inv(gb, pb), inv(ga, pa)
	count := func(n int, name string) string {
		if n == 1 {
			return name
		}
		return fmt.Sprintf("%d× %s", n, name)
	}
	for n, c := range ia {
		if c > ib[n] {
			lines = append(lines, "player gained "+count(c-ib[n], n))
		}
	}
	for n, c := range ib {
		if c > ia[n] {
			lines = append(lines, "player lost "+count(c-ia[n], n))
		}
	}
	if pb.Health != pa.Health {
		lines = append(lines, fmt.Sprintf("player health %d → %d", pb.Health, pa.Health))
	}
	for id, nb := range gb.NPCs {
		na, ok := ga.NPCs[id]
		switch {
		case !ok:
			lines = append(lines, "character removed: "+nb.Name)
		case nb.LocationID != na.LocationID:
			lines = append(lines, fmt.Sprintf("%s moved %s → %s", na.Name, nb.LocationID, na.LocationID))
		case nb.Alive != na.Alive:
			lines = append(lines, fmt.Sprintf("%s alive %v → %v", na.Name, nb.Alive, na.Alive))
		}
	}
	for id, na := range ga.NPCs {
		if _, ok := gb.NPCs[id]; !ok {
			lines = append(lines, fmt.Sprintf("character added: %s in %s", na.Name, na.LocationID))
		}
	}
	for id, it := range ga.Items {
		if _, ok := gb.Items[id]; !ok {
			lines = append(lines, "item created: "+it.Name)
		}
	}
	cb, ca := before.Campaign, after.Campaign
	if cb != nil && ca != nil {
		for id := range ca.DiscoveredRoomIDs {
			if !cb.DiscoveredRoomIDs[id] {
				lines = append(lines, "discovered "+ga.Rooms[id].Name)
			}
		}
		if cb.ActiveNodeID != ca.ActiveNodeID {
			lines = append(lines, fmt.Sprintf("scene %s → %s", cb.ActiveNodeID, ca.ActiveNodeID))
		}
		status := func(c *game.CampaignRuntimeState) map[string]string {
			m := map[string]string{}
			for _, o := range c.ActiveObjectives {
				m[o.ID] = string(o.Status)
			}
			return m
		}
		sb, sa := status(cb), status(ca)
		for id, s := range sa {
			if sb[id] != s {
				lines = append(lines, fmt.Sprintf("objective %s: %s → %s", id, orDash(sb[id]), s))
			}
		}
		for k, v := range ca.Labels {
			if cb.Labels[k] != v {
				lines = append(lines, fmt.Sprintf("label %s = %s", k, v))
			}
		}
		for k, v := range ca.BoolFlags {
			if cb.BoolFlags[k] != v {
				lines = append(lines, fmt.Sprintf("flag %s = %v", k, v))
			}
		}
		for k, v := range ca.Counters {
			if cb.Counters[k] != v {
				lines = append(lines, fmt.Sprintf("counter %s %d → %d", k, cb.Counters[k], v))
			}
		}
	}
	if len(lines) == 0 {
		return "- nothing\n"
	}
	return "- " + strings.Join(lines, "\n- ") + "\n"
}

func quote(s string) string {
	return "> " + strings.ReplaceAll(strings.TrimSpace(s), "\n", "\n> ")
}

func compact(v any) string {
	raw, _ := json.Marshal(v)
	return string(raw)
}

func mustJSON(v any) string {
	raw, err := json.Marshal(v)
	must(err)
	return string(raw)
}

func orNone(xs []string) string {
	if len(xs) == 0 {
		return "none"
	}
	return strings.Join(xs, ", ")
}

func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

func appendFile(path, s string) error {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(s)
	return err
}

func must(err error) {
	if err != nil {
		log.SetOutput(os.Stderr)
		log.Fatal(err)
	}
}

type discard struct{}

func (discard) Write(p []byte) (int, error) { return len(p), nil }

func discardUnless(show bool) interface{ Write([]byte) (int, error) } {
	if show {
		return os.Stderr
	}
	return discard{}
}
