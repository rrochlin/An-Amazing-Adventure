package yarnc

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"sort"
	"strconv"
)

// ProgramText renders the program as protobuf text format (.yarnc), in the
// same hand-readable shape as the hand-assembled campaign files.
//
// It is written by hand rather than with prototext.Format because that
// output is intentionally unstable, and compiled files are committed and
// checked for staleness in tests.
func (p *Program) ProgramText() []byte {
	var b bytes.Buffer
	fmt.Fprintf(&b, "name: %s\n", strconv.Quote(p.Name))
	for _, n := range p.Nodes {
		fmt.Fprintf(&b, "nodes: {\n  key: %s\n  value: {\n", strconv.Quote(n.Name))
		fmt.Fprintf(&b, "    name: %s\n", strconv.Quote(n.Name))
		for _, in := range n.Instructions {
			if len(in.Operands) == 0 {
				fmt.Fprintf(&b, "    instructions: { opcode: %s }\n", in.Opcode)
				continue
			}
			fmt.Fprintf(&b, "    instructions: {\n      opcode: %s\n", in.Opcode)
			for _, op := range in.Operands {
				fmt.Fprintf(&b, "      operands: { %s }\n", operandText(op))
			}
			b.WriteString("    }\n")
		}
		labels := make([]string, 0, len(n.Labels))
		for k := range n.Labels {
			labels = append(labels, k)
		}
		sort.Strings(labels)
		for _, k := range labels {
			fmt.Fprintf(&b, "    labels: {\n      key: %s\n      value: %d\n    }\n", strconv.Quote(k), n.Labels[k])
		}
		for _, h := range n.Headers {
			fmt.Fprintf(&b, "    headers: {\n      key: %s\n      value: %s\n    }\n", strconv.Quote(h[0]), strconv.Quote(h[1]))
		}
		b.WriteString("  }\n}\n")
	}
	return b.Bytes()
}

func operandText(op any) string {
	switch v := op.(type) {
	case string:
		return "string_value: " + strconv.Quote(v)
	case float64:
		return "float_value: " + strconv.FormatFloat(v, 'g', -1, 32)
	case bool:
		return "bool_value: " + strconv.FormatBool(v)
	default:
		panic(fmt.Sprintf("yarnc: unsupported operand type %T", op))
	}
}

// StringsCSV renders the string table in the two-column id,text form the
// dialogue loader reads.
func (p *Program) StringsCSV() []byte {
	var b bytes.Buffer
	w := csv.NewWriter(&b)
	_ = w.Write([]string{"id", "text"})
	for _, row := range p.Strings {
		_ = w.Write([]string{row.ID, row.Text})
	}
	w.Flush()
	return b.Bytes()
}
