package corpus

import (
	"fmt"
	"github.com/vsoch/gosmeagle/descriptor"
	"github.com/vsoch/gosmeagle/parsers/file"
	"github.com/vsoch/gosmeagle/parsers/x86_64"
	"log"
)

// A corpus holds a library name, a list of Functions and variables
type Corpus struct {
	Library   string                           `json:"library"`
	Functions []descriptor.FunctionDescription `json:"functions"`
	Variables []string                         `json:"variables"`
}

// Get a corpus from a filename
func GetCorpus(filename string) Corpus {

	corpus := Corpus{Library: filename}

	f, err := file.Open(filename)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	// Populate the corpus depending on the Architecture
	corpus.Parse(f)
	return corpus
}

func (c *Corpus) Parse(f *file.File) {

	// Parse dwarf for each entry to use
	f.ParseDwarf()

	// Parse entries based on type (function or variable)
	for _, e := range f.Entries {

		// These are dynamic symbol table symbols
		symbols, err := e.Symbols()
		if err != nil {
			log.Fatalf("Issue retriving symbols from %s", c.Library)
		}
		for _, symbol := range symbols {

			// If we have a function, parse function
			if symbol.GetType() == "STT_FUNC" {
				c.parseFunction(f, symbol)
			}

			// TODO if it's a variable AND has global linkage, parse

			//fmt.Println("Name:", symbol.Name)
			//fmt.Println("Address:", symbol.Address)
			//fmt.Println("Size:", symbol.Size)
			//fmt.Println("Code:", symbol.Code)
			//fmt.Println("Type:", symbol.Type)
			//fmt.Println("Binding:", symbol.Binding)
			//fmt.Println("Relocs:", symbol.Relocations)
		}
	}
}

// parse a dynamic function symbol
func (c *Corpus) parseFunction(f *file.File, symbol file.Symbol) {

	//fmt.Println(symbol)
	switch f.GoArch() {
	case "amd64":
		{
			c.Functions = append(c.Functions, x86_64.ParseFunction(f, symbol))
			//		c.Functions = append(c.Functions, x86.ParseReturnValue(symbol)
		}
	default:
		fmt.Printf("Unsupported architecture %s\n", f.GoArch())
	}
}

// Serialize corpus to json
func (c *Corpus) ToJson() {

	//outJson, _ := json.Marshal(c)
	//output := string(outJson)
	//fmt.Println(output)
}
