package dialogue

import (
	"fmt"
	"io"
	"io/fs"
	"regexp"
	"strconv"
	"strings"

	yarnvm "drjosh.dev/yarn"
	yarnpb "drjosh.dev/yarn/bytecode"
	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/proto"
)

type RunResult struct {
	Lines      []string
	Commands   []string
	CurrentNode string
	Variables  map[string]string
	Completed  bool
}

type handler struct {
	currentNode string
	lineText    map[string]string
	lines       []string
	commands    []string
}

func (h *handler) NodeStart(nodeName string) error {
	h.currentNode = nodeName
	return nil
}

func (h *handler) PrepareForLines([]string) error { return nil }

func (h *handler) Line(line yarnvm.Line) error {
	text, ok := h.lineText[line.ID]
	if !ok {
		return fmt.Errorf("dialogue line %q not found in strings table", line.ID)
	}
	h.lines = append(h.lines, substitute(text, line.Substitutions))
	return nil
}

func (h *handler) Options(options []yarnvm.Option) (int, error) {
	for _, option := range options {
		if option.IsAvailable {
			return option.ID, nil
		}
	}
	return 0, fmt.Errorf("dialogue options contained no available choices")
}

func (h *handler) Command(command string) error {
	h.commands = append(h.commands, command)
	return nil
}

func (h *handler) NodeComplete(string) error { return nil }

func (h *handler) DialogueComplete() error { return nil }

// OpeningLine returns the first authored string row for a dialogue asset.
// This remains available as a utility, but real authored runtime integration
// should prefer RunNode so the Yarn program itself is actually executed.
func OpeningLine(fsys fs.FS, stringsPath string) (string, error) {
	rows, ordered, err := loadStrings(fsys, stringsPath)
	if err != nil {
		return "", err
	}
	for _, id := range ordered {
		if text := strings.TrimSpace(rows[id]); text != "" {
			return text, nil
		}
	}
	return "", nil
}

func RunNode(fsys fs.FS, programPath, stringsPath, startNode string, storedVars map[string]string) (RunResult, error) {
	if fsys == nil {
		return RunResult{}, fmt.Errorf("dialogue source fs is not set")
	}
	if programPath == "" {
		return RunResult{}, fmt.Errorf("dialogue program path is required")
	}
	if startNode == "" {
		return RunResult{}, fmt.Errorf("dialogue start node is required")
	}

	prog, err := loadProgram(fsys, programPath)
	if err != nil {
		return RunResult{}, err
	}
	lineText, _, err := loadStrings(fsys, stringsPath)
	if err != nil {
		return RunResult{}, err
	}

	vars := yarnvm.NewMapVariableStorageFromMap(loadVars(storedVars))
	h := &handler{lineText: lineText}
	vm := &yarnvm.VirtualMachine{
		Program: prog,
		Handler: h,
		Vars:    vars,
	}
	if err := vm.Run(startNode); err != nil {
		return RunResult{}, fmt.Errorf("run dialogue node %q: %w", startNode, err)
	}

	return RunResult{
		Lines:       h.lines,
		Commands:    h.commands,
		CurrentNode: h.currentNode,
		Variables:   storeVars(vars.Contents()),
		Completed:   true,
	}, nil
}

func loadProgram(fsys fs.FS, programPath string) (*yarnpb.Program, error) {
	bytes, err := fs.ReadFile(fsys, programPath)
	if err != nil {
		return nil, fmt.Errorf("open dialogue program %q: %w", programPath, err)
	}
	prog := new(yarnpb.Program)
	if err := proto.Unmarshal(bytes, prog); err == nil {
		return prog, nil
	}
	if err := prototext.Unmarshal(bytes, prog); err == nil {
		return prog, nil
	}
	return nil, fmt.Errorf("parse dialogue program %q: unsupported yarnc encoding", programPath)
}

func loadStrings(fsys fs.FS, stringsPath string) (map[string]string, []string, error) {
	if stringsPath == "" {
		return nil, nil, fmt.Errorf("dialogue strings path is required")
	}
	file, err := fsys.Open(stringsPath)
	if err != nil {
		return nil, nil, fmt.Errorf("open dialogue strings %q: %w", stringsPath, err)
	}
	defer file.Close()

	table, err := yarnvm.ReadStringTable(file, "en")
	if err == nil {
		rows := make(map[string]string, len(table.Table))
		ordered := make([]string, 0, len(table.Table))
		for id, row := range table.Table {
			if row != nil {
				rows[id] = row.Text
				ordered = append(ordered, id)
			}
		}
		return rows, ordered, nil
	}

	// Fall back to the minimal Phase 7 two-column csv shape: id,text
	file, err = fsys.Open(stringsPath)
	if err != nil {
		return nil, nil, fmt.Errorf("re-open dialogue strings %q: %w", stringsPath, err)
	}
	defer file.Close()
	return readSimpleCSV(file, stringsPath)
}

func readSimpleCSV(file fs.File, stringsPath string) (map[string]string, []string, error) {
	data, err := io.ReadAll(file)
	if err != nil {
		return nil, nil, fmt.Errorf("read dialogue strings %q: %w", stringsPath, err)
	}
	lines := strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
	rows := make(map[string]string)
	ordered := make([]string, 0)
	for i, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		if i == 0 && strings.EqualFold(strings.TrimSpace(line), "id,text") {
			continue
		}
		parts := strings.SplitN(line, ",", 2)
		if len(parts) < 2 {
			continue
		}
		id := strings.TrimSpace(parts[0])
		text := strings.Trim(strings.TrimSpace(parts[1]), "\"")
		if id != "" {
			ordered = append(ordered, id)
			rows[id] = text
		}
	}
	return rows, ordered, nil
}

var substPattern = regexp.MustCompile(`\{(\d+)\}`)

func substitute(text string, substitutions []string) string {
	if len(substitutions) == 0 {
		return text
	}
	return substPattern.ReplaceAllStringFunc(text, func(match string) string {
		idxText := strings.TrimSuffix(strings.TrimPrefix(match, "{"), "}")
		idx, err := strconv.Atoi(idxText)
		if err != nil || idx < 0 || idx >= len(substitutions) {
			return match
		}
		return substitutions[idx]
	})
}

func loadVars(stored map[string]string) map[string]any {
	vars := make(map[string]any, len(stored))
	for key, value := range stored {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			vars[key] = boolValue
			continue
		}
		if floatValue, err := strconv.ParseFloat(value, 32); err == nil {
			vars[key] = float32(floatValue)
			continue
		}
		vars[key] = value
	}
	return vars
}

func storeVars(vars map[string]any) map[string]string {
	if len(vars) == 0 {
		return nil
	}
	stored := make(map[string]string, len(vars))
	for key, value := range vars {
		switch typed := value.(type) {
		case string:
			stored[key] = typed
		case bool:
			stored[key] = strconv.FormatBool(typed)
		case float32:
			stored[key] = strconv.FormatFloat(float64(typed), 'f', -1, 32)
		case float64:
			stored[key] = strconv.FormatFloat(typed, 'f', -1, 64)
		case int:
			stored[key] = strconv.Itoa(typed)
		default:
			stored[key] = fmt.Sprint(typed)
		}
	}
	return stored
}
