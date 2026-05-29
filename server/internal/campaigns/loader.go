package campaigns

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	campaignfiles "github.com/rrochlin/an-amazing-adventure/campaigns"
)

const campaignFileName = "campaign.json"

func LoadCampaignFile(path string) (*CampaignDefinition, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read campaign file %q: %w", path, err)
	}

	var def CampaignDefinition
	dec := json.NewDecoder(strings.NewReader(string(data)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&def); err != nil {
		return nil, fmt.Errorf("decode campaign file %q: %w", path, err)
	}
	def.SourceDir = filepath.Dir(path)
	def.SourceFS = os.DirFS(def.SourceDir)

	if err := def.Validate(); err != nil {
		return nil, fmt.Errorf("validate campaign file %q: %w", path, err)
	}

	return &def, nil
}

func LoadCampaignFS(fsys fs.FS, path string) (*CampaignDefinition, error) {
	data, err := fs.ReadFile(fsys, path)
	if err != nil {
		return nil, fmt.Errorf("read campaign file %q: %w", path, err)
	}

	var def CampaignDefinition
	dec := json.NewDecoder(strings.NewReader(string(data)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&def); err != nil {
		return nil, fmt.Errorf("decode campaign file %q: %w", path, err)
	}
	def.SourceDir = filepath.Dir(path)
	if def.SourceDir == "." {
		def.SourceDir = ""
	}
	if def.SourceDir != "" {
		sub, err := fs.Sub(fsys, def.SourceDir)
		if err != nil {
			return nil, fmt.Errorf("sub fs for %q: %w", path, err)
		}
		def.SourceFS = sub
	} else {
		def.SourceFS = fsys
	}

	if err := def.Validate(); err != nil {
		return nil, fmt.Errorf("validate campaign file %q: %w", path, err)
	}

	return &def, nil
}

func LoadDirectory(root string) (map[string]*CampaignDefinition, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, fmt.Errorf("read campaigns directory %q: %w", root, err)
	}

	defs := make(map[string]*CampaignDefinition)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		path := filepath.Join(root, entry.Name(), campaignFileName)
		if _, err := os.Stat(path); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("stat campaign file %q: %w", path, err)
		}

		def, err := LoadCampaignFile(path)
		if err != nil {
			return nil, err
		}
		if _, exists := defs[def.ID]; exists {
			return nil, fmt.Errorf("duplicate campaign id %q", def.ID)
		}
		defs[def.ID] = def
	}

	if len(defs) == 0 {
		return nil, fmt.Errorf("no campaign definitions found in %q", root)
	}

	return defs, nil
}

func LoadFS(root fs.FS) (map[string]*CampaignDefinition, error) {
	entries, err := fs.ReadDir(root, ".")
	if err != nil {
		return nil, fmt.Errorf("read campaigns fs: %w", err)
	}

	defs := make(map[string]*CampaignDefinition)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		path := filepath.ToSlash(filepath.Join(entry.Name(), campaignFileName))
		if _, err := fs.Stat(root, path); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("stat campaign file %q: %w", path, err)
		}
		def, err := LoadCampaignFS(root, path)
		if err != nil {
			return nil, err
		}
		if _, exists := defs[def.ID]; exists {
			return nil, fmt.Errorf("duplicate campaign id %q", def.ID)
		}
		defs[def.ID] = def
	}

	if len(defs) == 0 {
		return nil, fmt.Errorf("no campaign definitions found in fs")
	}
	return defs, nil
}

type Registry struct {
	byID map[string]*CampaignDefinition
}

func LoadRegistry(root string) (*Registry, error) {
	defs, err := LoadDirectory(root)
	if err != nil {
		return nil, err
	}
	return &Registry{byID: defs}, nil
}

func LoadEmbeddedRegistry() (*Registry, error) {
	defs, err := LoadFS(campaignfiles.FS)
	if err != nil {
		return nil, err
	}
	return &Registry{byID: defs}, nil
}

func (r *Registry) Get(id string) (*CampaignDefinition, bool) {
	if r == nil || r.byID == nil {
		return nil, false
	}
	def, ok := r.byID[id]
	return def, ok
}

func (r *Registry) List() []Manifest {
	if r == nil || r.byID == nil {
		return nil
	}
	out := make([]Manifest, 0, len(r.byID))
	for _, def := range r.byID {
		out = append(out, Manifest{
			ID:              def.ID,
			Version:         def.Version,
			Title:           def.Title,
			Premise:         def.Premise,
			Description:     def.Description,
			Tone:            def.Tone,
			AllowedClasses:  def.CharacterCreation.AllowedClasses,
			AllowedRaces:    def.CharacterCreation.AllowedRaces,
			AllowedSubraces: def.CharacterCreation.AllowedSubraces,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}
