package cli

import (
	"github.com/DataDrake/cli-ng/v2/cmd"
	"github.com/vsoch/gosmeagle/corpus"
)

// Args and flags for generate
type ParserArgs struct {
	Binary []string `desc:"A binary to parse."`
}
type ParserFlags struct{}

// Parser looks at symbols and ABI in Go
var Parser = cmd.Sub{
	Name:  "parse",
	Alias: "p",
	Short: "Parse a binary.",
	Flags: &ParserFlags{},
	Args:  &ParserArgs{},
	Run:   RunParser,
}

func init() {
	cmd.Register(&Parser)
}

// RunParser reads a file and creates a corpus
func RunParser(r *cmd.Root, c *cmd.Sub) {
	args := c.Args.(*ParserArgs)
	corpus := corpus.GetCorpus(args.Binary[0])
	corpus.ToJson()
}
