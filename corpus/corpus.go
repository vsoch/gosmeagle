package corpus

import (
	"encoding/json"
	"fmt"
	"github.com/vsoch/gosmeagle/descriptor"
	"github.com/vsoch/gosmeagle/parsers/file"
	"github.com/vsoch/gosmeagle/parsers/x86_64"
	"log"
	"reflect"
)

// A corpus holds a library name, a list of Functions and variables
// TODO should this be a list of Locations instead?
type Corpus struct {
	Library   string                           `json:"library"`
	Functions []descriptor.FunctionDescription `json:"functions,omitempty"`
	Variables []descriptor.VariableDescription `json:"variables,omitempty"`
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
	lookup := f.ParseDwarf()

	// Parse entries based on type (function or variable)
	for _, e := range f.Entries {

		// These are dynamic symbol table symbols
		symbols, err := e.Symbols()
		if err != nil {
			log.Fatalf("Issue retriving symbols from %s", c.Library)
		}
		for _, symbol := range symbols {
			switch symbol.GetType() {
			case "STT_FUNC":
				entry, ok := lookup["functions"][symbol.GetName()]
				if !ok {
					continue
				}
				c.parseFunction(f, symbol, &entry)

			case "STT_OBJECT":

				// Do we have a variable with global linkage?
				entry, ok := lookup["variables"][symbol.GetName()]
				fmt.Println(symbol)
				if ok && symbol.GetBinding() == "STB_GLOBAL" {
					c.parseVariable(f, symbol, &entry)
				}

			}
		}
	}
}

// parse a dynamic function symbol
func (c *Corpus) parseFunction(f *file.File, symbol file.Symbol, entry *file.DwarfEntry) {

	switch f.GoArch() {
	case "amd64":
		c.Functions = append(c.Functions, x86_64.ParseFunction(f, symbol, entry))
	default:
		fmt.Printf("Unsupported architecture %s\n", f.GoArch())
	}
}

// parse a global variable
func (c *Corpus) parseVariable(f *file.File, symbol file.Symbol, entry *file.DwarfEntry) {

	switch f.GoArch() {
	case "amd64":

		// Don't allow variables without name or type
		variable := x86_64.ParseVariable(f, symbol, entry)
		if !reflect.DeepEqual(variable, descriptor.VariableDescription{}) {
			c.Variables = append(c.Variables, variable)
		}
	default:
		fmt.Printf("Unsupported architecture %s\n", f.GoArch())
	}
}

// Serialize corpus to json
func (c *Corpus) ToJson(pretty bool) {

	var outJson []byte
	if pretty {
		outJson, _ = json.MarshalIndent(c, "", "    ")
	} else {
		outJson, _ = json.Marshal(c)
	}
	output := string(outJson)
	fmt.Println(output)
}
