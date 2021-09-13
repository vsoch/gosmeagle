package cli

import (
	"fmt"
	"log"

	"github.com/DataDrake/cli-ng/v2/cmd"
	"github.com/vsoch/gosmeagle/gosrc/cmd/objfile"
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

// RunDockerfile updates one or more Dockerfile
func RunParser(r *cmd.Root, c *cmd.Sub) {

	args := c.Args.(*ParserArgs)

	f, err := objfile.Open(args.Binary[0])
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	// Try looping through entries
	for _, entry := range f.Entries() {
		fmt.Println(entry)
		symbols, err := entry.Symbols()
		if err != nil {
			log.Fatalf("Issue retriving symbols from %s", args.Binary[0])
		}
		for _, symbol := range symbols {
			fmt.Println("Name:", symbol.Name)
			fmt.Println("Address:", symbol.Addr)
			fmt.Println("Size:", symbol.Size)
			fmt.Println("Code:", symbol.Code)
			fmt.Println("Type:", symbol.Type)
			fmt.Println("Relocs:", symbol.Relocs)
		}
	}

	// Trying looking at DWARF
	// TODO how to parse this?
	// fmt.Println(f.DWARF())
}
