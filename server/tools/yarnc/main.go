// Command yarnc compiles campaign dialogue (.yarn) into the .yarnc bytecode
// and .csv string tables the server loads. See internal/yarnc for the
// supported language subset.
//
// Usage (from server/):
//
//	go run ./tools/yarnc            # compile every .yarn under campaigns/
//	go run ./tools/yarnc FILE...    # compile specific files
//	go run ./tools/yarnc -check     # exit 1 if any compiled output is stale
package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"

	"github.com/rrochlin/an-amazing-adventure/internal/yarnc"
)

func main() {
	check := flag.Bool("check", false, "report stale outputs instead of writing them")
	flag.Parse()

	files := flag.Args()
	if len(files) == 0 {
		var err error
		if files, err = yarnc.FindSources("campaigns"); err != nil {
			fail(err)
		}
	}

	stale := 0
	for _, f := range files {
		out, err := yarnc.CompileFile(f)
		if err != nil {
			fail(err)
		}
		for _, w := range []struct {
			path string
			data []byte
		}{{out.ProgramPath, out.Program}, {out.StringsPath, out.Strings}} {
			existing, _ := os.ReadFile(w.path)
			if bytes.Equal(existing, w.data) {
				continue
			}
			if *check {
				fmt.Printf("stale: %s\n", w.path)
				stale++
				continue
			}
			if err := os.WriteFile(w.path, w.data, 0o644); err != nil {
				fail(err)
			}
			fmt.Printf("wrote %s\n", w.path)
		}
	}
	if stale > 0 {
		fmt.Println("run `go run ./tools/yarnc` from server/ to recompile")
		os.Exit(1)
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "yarnc:", err)
	os.Exit(1)
}
