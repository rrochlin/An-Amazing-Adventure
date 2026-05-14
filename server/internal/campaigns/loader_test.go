package campaigns_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rrochlin/an-amazing-adventure/internal/campaigns"
)

func TestLoadCampaignFile_TestCampaign(t *testing.T) {
	path := filepath.Join("..", "..", "campaigns", "test", "campaign.json")
	def, err := campaigns.LoadCampaignFile(path)
	if err != nil {
		t.Fatalf("LoadCampaignFile: %v", err)
	}
	if def.ID != "test" {
		t.Fatalf("expected id=test, got %q", def.ID)
	}
	if def.StartNodeID != "intro" {
		t.Fatalf("expected start_node_id=intro, got %q", def.StartNodeID)
	}
	if _, ok := def.StoryNodes["relic_prompt"]; !ok {
		t.Fatal("expected relic_prompt story node to exist")
	}
}

func TestLoadRegistry_LoadsAllCampaigns(t *testing.T) {
	root := filepath.Join("..", "..", "campaigns")
	reg, err := campaigns.LoadRegistry(root)
	if err != nil {
		t.Fatalf("LoadRegistry: %v", err)
	}
	if _, ok := reg.Get("test"); !ok {
		t.Fatal("expected registry to contain test campaign")
	}
	if _, ok := reg.Get("lichs-labyrinth"); !ok {
		t.Fatal("expected registry to contain lichs-labyrinth campaign")
	}
	if got := len(reg.List()); got < 2 {
		t.Fatalf("expected at least 2 campaigns, got %d", got)
	}
}

func TestLoadEmbeddedRegistry_LoadsEmbeddedCampaigns(t *testing.T) {
	reg, err := campaigns.LoadEmbeddedRegistry()
	if err != nil {
		t.Fatalf("LoadEmbeddedRegistry: %v", err)
	}
	if _, ok := reg.Get("test"); !ok {
		t.Fatal("expected embedded registry to contain test campaign")
	}
}

func TestLoadCampaignFile_InvalidUnknownNextNode(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "stub.yarnc"), []byte("stub"), 0o644); err != nil {
		t.Fatalf("write stub yarnc: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "stub.csv"), []byte("id,text\n"), 0o644); err != nil {
		t.Fatalf("write stub csv: %v", err)
	}

	bad := `{
		"id":"bad",
		"version":"1",
		"title":"Bad",
		"premise":"Bad premise",
		"character_creation":{"use_standard_creator":true},
		"party":{"max_human_players":1,"starting_room_id":"camp"},
		"map":{"rooms":{"camp":{"name":"Camp"}}},
		"actors":{},
		"dialogue_assets":{"intro":{"format":"yarn","program":"stub.yarnc","strings":"stub.csv"}},
		"story_nodes":{
			"intro":{
				"mode":"yarn_dialogue",
				"dialogue_asset_id":"intro",
				"dialogue_start_node":"Intro",
				"title":"Intro",
				"completion_rules":[{"conditions":[],"next_node_id":"missing"}]
			}
		},
		"start_node_id":"intro"
	}`

	path := filepath.Join(dir, "campaign.json")
	if err := os.WriteFile(path, []byte(bad), 0o644); err != nil {
		t.Fatalf("write invalid campaign: %v", err)
	}

	_, err := campaigns.LoadCampaignFile(path)
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(err.Error(), "next_node_id") {
		t.Fatalf("expected next_node_id error, got %v", err)
	}
}
