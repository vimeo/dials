package panels

import (
	"bytes"
	"fmt"
	"strings"
	"text/tabwriter"
)

func (p *Panel[T]) helpString(binaryName string) []byte {
	var b bytes.Buffer
	tw := tabwriter.NewWriter(&b, 0, 8, 0, '\t', 0)

	fmt.Fprintf(&b, "Usage for %s:\n", binaryName)

	// Description
	//
	// Long Usage
	if p.ph != nil {
		fmt.Fprintf(tw, "%s\n\n%s\n\n",
			indentString(p.ph.Description([]string{binaryName})),
			indentString(p.ph.LongUsage([]string{binaryName})))
	}

	// command: description
	// \t short usage
	// \n
	for command, sch := range p.schMap {
		scPath := []string{binaryName, command}
		fmt.Fprintf(tw, "%s: %s\n%s\n\n", command,
			indentString(sch.spHelp().Description(scPath)),
			indentString(sch.spHelp().ShortUsage(scPath)))
	}

	tw.Flush()
	return b.Bytes()
}

func indentString(s string) string {
	return "\t" + strings.ReplaceAll(s, "\n", "\t\n")
}

func (s *SubCmdHandle[RT, T]) helpString(scPath []string) []byte {
	var b bytes.Buffer

	fmt.Fprintf(&b, "Usage for %s:\n", strings.Join(scPath, " "))

	fmt.Fprintf(&b, "%s\n\n%s\n\n",
		indentString(s.sp.Description(scPath)),
		indentString(s.sp.LongUsage(scPath)))

	// TODO: more when recursing

	return b.Bytes()
}
