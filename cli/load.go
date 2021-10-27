package cli

// This is just for debugging - it loads and prints the json corpus

import (
	"github.com/DataDrake/cli-ng/v2/cmd"
	"github.com/vsoch/gosmeagle/corpus"
)

type LoadArgs struct {
	JsonFile []string `desc:"A binary to parse."`
}
type LoadFlags struct{}

// Parser looks at symbols and ABI in Go
var Loader = cmd.Sub{
	Name:  "load",
	Alias: "l",
	Short: "Test laoding a Json output",
	Flags: &LoadFlags{},
	Args:  &LoadArgs{},
	Run:   RunLoader,
}

func init() {
	cmd.Register(&Loader)
}

func RunLoader(r *cmd.Root, c *cmd.Sub) {
	args := c.Args.(*LoadArgs)
	C := corpus.Load(args.JsonFile[0])
	corp := C.ToCorpus()
	corp.ToJson(true)
}
