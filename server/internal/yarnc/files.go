package yarnc

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Outputs is what compiling one .yarn file produces: sibling .yarnc and
// .csv files with the same stem.
type Outputs struct {
	ProgramPath string
	Program     []byte
	StringsPath string
	Strings     []byte
}

// CompileFile compiles a .yarn file. For campaign dialogue at
// campaigns/<id>/dialogue/<stem>.yarn the program is named "<id>-<stem>".
func CompileFile(path string) (*Outputs, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	stem := strings.TrimSuffix(filepath.Base(path), ".yarn")
	name := stem
	if dir := filepath.Dir(path); filepath.Base(dir) == "dialogue" {
		name = filepath.Base(filepath.Dir(dir)) + "-" + stem
	}
	prog, err := Compile(name, string(src))
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	base := strings.TrimSuffix(path, ".yarn")
	return &Outputs{
		ProgramPath: base + ".yarnc",
		Program:     prog.ProgramText(),
		StringsPath: base + ".csv",
		Strings:     prog.StringsCSV(),
	}, nil
}

// FindSources returns every .yarn file under root.
func FindSources(root string) ([]string, error) {
	var out []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(path, ".yarn") {
			out = append(out, path)
		}
		return nil
	})
	return out, err
}
