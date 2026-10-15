// Package yarnc compiles a small subset of the Yarn Spinner 2 language into
// the bytecode run by drjosh.dev/yarn (the Yarn Spinner 2.3 VM).
//
// The official compiler (ysc) only ships for Yarn Spinner 3, whose bytecode
// the Go VM cannot run, so campaign dialogue is compiled here instead.
//
// Supported:
//   - nodes: header lines (title: is required), "---", body, "==="
//   - lines of dialogue, optionally tagged "#line:<id>"
//   - shortcut options ("-> text"), nested by indentation
//   - <<jump Node>> and <<stop>>
//   - any other <<command ...>>, delivered to the game as a command string
//   - "//" comments
//
// Not supported (rejected with an error): variables (<<set>>, <<declare>>),
// <<if>> blocks and option conditions, and {expression} interpolation.
package yarnc

import (
	"fmt"
	"regexp"
	"strings"
)

// Program is a compiled Yarn program: its nodes in source order, plus the
// string table rows (line ID → text) in the order they were encountered.
type Program struct {
	Name    string
	Nodes   []*Node
	Strings []StringRow
}

type Node struct {
	Name         string
	Headers      [][2]string
	Instructions []Instruction
	Labels       map[string]int
}

type Instruction struct {
	Opcode   string
	Operands []any // string, float64, or bool
}

type StringRow struct {
	ID   string
	Text string
}

var (
	lineTagPattern = regexp.MustCompile(`#line:(\S+)`)
	anyTagPattern  = regexp.MustCompile(`\s#\S+`)
	headerPattern  = regexp.MustCompile(`^([A-Za-z_][A-Za-z0-9_]*):\s*(.*)$`)
	nodeNameRegexp = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_.]*$`)
)

// Compile compiles Yarn source. name becomes the program name, and is the
// prefix for generated line IDs on lines without an explicit #line: tag.
func Compile(name, source string) (*Program, error) {
	lines := strings.Split(strings.ReplaceAll(source, "\r\n", "\n"), "\n")
	prog := &Program{Name: name}
	seenIDs := map[string]int{}

	i := 0
	for i < len(lines) {
		if isBlank(lines[i]) {
			i++
			continue
		}
		node := &Node{Labels: map[string]int{}}
		// Headers until ---
		for ; i < len(lines) && strings.TrimSpace(lines[i]) != "---"; i++ {
			if isBlank(lines[i]) {
				continue
			}
			m := headerPattern.FindStringSubmatch(strings.TrimSpace(lines[i]))
			if m == nil {
				return nil, fmt.Errorf("line %d: expected a node header (key: value) or ---", i+1)
			}
			node.Headers = append(node.Headers, [2]string{m[1], m[2]})
			if m[1] == "title" {
				node.Name = strings.TrimSpace(m[2])
			}
		}
		if i >= len(lines) {
			return nil, fmt.Errorf("line %d: node header is missing ---", i)
		}
		if node.Name == "" {
			return nil, fmt.Errorf("line %d: node is missing a title: header", i+1)
		}
		if !nodeNameRegexp.MatchString(node.Name) {
			return nil, fmt.Errorf("line %d: invalid node title %q", i+1, node.Name)
		}
		for _, existing := range prog.Nodes {
			if existing.Name == node.Name {
				return nil, fmt.Errorf("line %d: duplicate node %q", i+1, node.Name)
			}
		}
		i++ // skip ---

		start := i
		for ; i < len(lines) && strings.TrimSpace(lines[i]) != "==="; i++ {
		}
		if i >= len(lines) {
			return nil, fmt.Errorf("node %q: body is missing ===", node.Name)
		}
		body, err := parseBody(lines[start:i], start)
		if err != nil {
			return nil, fmt.Errorf("node %q: %w", node.Name, err)
		}
		i++ // skip ===

		c := &nodeCompiler{prog: prog, node: node, seenIDs: seenIDs}
		if err := c.emitBlock(body); err != nil {
			return nil, fmt.Errorf("node %q: %w", node.Name, err)
		}
		c.emit("STOP")
		prog.Nodes = append(prog.Nodes, node)
	}
	if len(prog.Nodes) == 0 {
		return nil, fmt.Errorf("no nodes found")
	}

	// Every <<jump>> must target a node in this program.
	names := map[string]bool{}
	for _, n := range prog.Nodes {
		names[n.Name] = true
	}
	for _, n := range prog.Nodes {
		for _, in := range n.Instructions {
			if in.Opcode == "PUSH_STRING" && !names[in.Operands[0].(string)] {
				return nil, fmt.Errorf("node %q: <<jump %s>> targets an unknown node", n.Name, in.Operands[0])
			}
		}
	}
	return prog, nil
}

// statement is one parsed body line, with option children nested under it.
type statement struct {
	kind     string // "line", "option", "command", "jump", "stop"
	text     string
	lineID   string
	srcLine  int
	indent   int
	children []*statement
}

func parseBody(raw []string, offset int) ([]*statement, error) {
	var flat []*statement
	for idx, l := range raw {
		srcLine := offset + idx + 1
		if isBlank(l) {
			continue
		}
		indent := indentWidth(l)
		text := strings.TrimSpace(l)

		st := &statement{srcLine: srcLine, indent: indent}
		switch {
		case strings.HasPrefix(text, "->"):
			st.kind = "option"
			text = strings.TrimSpace(strings.TrimPrefix(text, "->"))
		case strings.HasPrefix(text, "<<") && strings.HasSuffix(text, ">>"):
			cmd := strings.TrimSpace(text[2 : len(text)-2])
			fields := strings.Fields(cmd)
			if len(fields) == 0 {
				return nil, fmt.Errorf("line %d: empty command", srcLine)
			}
			switch fields[0] {
			case "jump":
				if len(fields) != 2 || !nodeNameRegexp.MatchString(fields[1]) {
					return nil, fmt.Errorf("line %d: <<jump>> takes exactly one node name", srcLine)
				}
				st.kind, st.text = "jump", fields[1]
			case "stop":
				st.kind = "stop"
			case "set", "declare", "if", "elseif", "else", "endif", "local", "once", "endonce":
				return nil, fmt.Errorf("line %d: <<%s>> is not supported by this compiler", srcLine, fields[0])
			default:
				if strings.ContainsAny(cmd, "{}") {
					return nil, fmt.Errorf("line %d: {expressions} in commands are not supported", srcLine)
				}
				st.kind, st.text = "command", cmd
			}
			flat = append(flat, st)
			continue
		default:
			st.kind = "line"
		}

		// Lines and options: pull out tags, reject unsupported syntax.
		if strings.Contains(text, "<<") {
			return nil, fmt.Errorf("line %d: inline conditions (<<if>>) are not supported", srcLine)
		}
		if m := lineTagPattern.FindStringSubmatch(text); m != nil {
			st.lineID = "line:" + m[1]
		}
		text = strings.TrimSpace(anyTagPattern.ReplaceAllString(" "+text, ""))
		if strings.ContainsAny(text, "{}") {
			return nil, fmt.Errorf("line %d: {expression} interpolation is not supported", srcLine)
		}
		if text == "" {
			return nil, fmt.Errorf("line %d: empty %s", srcLine, st.kind)
		}
		st.text = text
		flat = append(flat, st)
	}
	return nest(flat)
}

// nest attaches statements indented under an option as that option's
// children. Statements at the same or shallower indent end the option body.
func nest(flat []*statement) ([]*statement, error) {
	var root []*statement
	type frame struct {
		opt *statement
	}
	var stack []frame
	for _, st := range flat {
		for len(stack) > 0 && st.indent <= stack[len(stack)-1].opt.indent {
			stack = stack[:len(stack)-1]
		}
		if len(stack) == 0 {
			root = append(root, st)
		} else {
			parent := stack[len(stack)-1].opt
			parent.children = append(parent.children, st)
		}
		if st.kind == "option" {
			stack = append(stack, frame{opt: st})
		}
	}
	return root, nil
}

type nodeCompiler struct {
	prog     *Program
	node     *Node
	seenIDs  map[string]int
	labelSeq int
	lineSeq  int
}

func (c *nodeCompiler) emit(op string, operands ...any) {
	c.node.Instructions = append(c.node.Instructions, Instruction{Opcode: op, Operands: operands})
}

func (c *nodeCompiler) newLabel(kind string) string {
	c.labelSeq++
	return fmt.Sprintf("L%d%s", c.labelSeq, kind)
}

func (c *nodeCompiler) mark(label string) {
	c.node.Labels[label] = len(c.node.Instructions)
}

func (c *nodeCompiler) addString(st *statement) (string, error) {
	id := st.lineID
	if id == "" {
		id = fmt.Sprintf("line:%s-%s-%d", c.prog.Name, c.node.Name, c.lineSeq)
		c.lineSeq++
	}
	if prev, dup := c.seenIDs[id]; dup {
		return "", fmt.Errorf("line %d: line ID %q already used on line %d", st.srcLine, id, prev)
	}
	c.seenIDs[id] = st.srcLine
	c.prog.Strings = append(c.prog.Strings, StringRow{ID: id, Text: st.text})
	return id, nil
}

func (c *nodeCompiler) emitBlock(stmts []*statement) error {
	for i := 0; i < len(stmts); {
		st := stmts[i]
		if st.kind != "option" {
			if err := c.emitStatement(st); err != nil {
				return err
			}
			i++
			continue
		}
		// A run of consecutive options forms one option group.
		j := i
		for j < len(stmts) && stmts[j].kind == "option" {
			j++
		}
		if err := c.emitOptionGroup(stmts[i:j]); err != nil {
			return err
		}
		i = j
	}
	return nil
}

func (c *nodeCompiler) emitStatement(st *statement) error {
	switch st.kind {
	case "line":
		id, err := c.addString(st)
		if err != nil {
			return err
		}
		c.emit("RUN_LINE", id, 0.0)
	case "command":
		c.emit("RUN_COMMAND", st.text, 0.0)
	case "jump":
		c.emit("PUSH_STRING", st.text)
		c.emit("RUN_NODE")
	case "stop":
		c.emit("STOP")
	default:
		return fmt.Errorf("line %d: unexpected %s", st.srcLine, st.kind)
	}
	return nil
}

// emitOptionGroup mirrors Yarn Spinner 2's shortcut-option codegen:
//
//	ADD_OPTION <line> <label> 0 false   (per option)
//	SHOW_OPTIONS
//	JUMP                                (peeks the chosen label)
//	<label>: <body> JUMP_TO <end>       (per option)
//	<end>: POP                          (drops the chosen label)
func (c *nodeCompiler) emitOptionGroup(opts []*statement) error {
	end := c.newLabel("group_end")
	labels := make([]string, len(opts))
	for k, opt := range opts {
		id, err := c.addString(opt)
		if err != nil {
			return err
		}
		labels[k] = c.newLabel(fmt.Sprintf("shortcutoption_%s_%d", c.node.Name, k+1))
		c.emit("ADD_OPTION", id, labels[k], 0.0, false)
	}
	c.emit("SHOW_OPTIONS")
	c.emit("JUMP")
	for k, opt := range opts {
		c.mark(labels[k])
		if err := c.emitBlock(opt.children); err != nil {
			return err
		}
		c.emit("JUMP_TO", end)
	}
	c.mark(end)
	c.emit("POP")
	return nil
}

func isBlank(l string) bool {
	t := strings.TrimSpace(l)
	return t == "" || strings.HasPrefix(t, "//")
}

func indentWidth(l string) int {
	n := 0
	for _, r := range l {
		switch r {
		case ' ':
			n++
		case '\t':
			n += 4
		default:
			return n
		}
	}
	return n
}
