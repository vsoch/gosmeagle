package cli

import (
	"github.com/DataDrake/cli-ng/v2/cmd"
	"github.com/vsoch/gosmeagle/corpus"
	"os"
	"regexp"
)

// Args and flags for generate
type DisasmArgs struct {
	Binary []string `desc:"A binary to dissassemble."`
}
type DisasmFlags struct{}

var Disasm = cmd.Sub{
	Name:  "disasm",
	Alias: "d",
	Short: "Disassemble a binary.",
	Flags: &DisasmFlags{},
	Args:  &DisasmArgs{},
	Run:   RunDisasm,
}

func init() {
	cmd.Register(&Disasm)
}

func RunDisasm(r *cmd.Root, c *cmd.Sub) {
	args := c.Args.(*DisasmArgs)
	disasm := corpus.GetDisasm(args.Binary[0])
	var symRE *regexp.Regexp
	disasm.Print(os.Stdout, symRE, 0, ^uint64(0), true, true)
}
