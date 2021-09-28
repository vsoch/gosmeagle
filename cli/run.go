package cli

// This is just for debugging - it does the same thing as parse but
// doesn't print the json

import (
	"github.com/DataDrake/cli-ng/v2/cmd"
	"github.com/vsoch/gosmeagle/corpus"
)

// Args and flags for generate
type RunArgs struct {
	Binary []string `desc:"A binary to parse."`
}
type RunFlags struct{}

// Parser looks at symbols and ABI in Go
var Runner = cmd.Sub{
	Name:  "run",
	Alias: "r",
	Short: "Generate a corpus with no Json output",
	Flags: &RunFlags{},
	Args:  &RunArgs{},
	Run:   RunRunner,
}

func init() {
	cmd.Register(&Runner)
}

// RunParser reads a file and creates a corpus
func RunRunner(r *cmd.Root, c *cmd.Sub) {
	args := c.Args.(*RunArgs)
	corpus.GetCorpus(args.Binary[0])
}
