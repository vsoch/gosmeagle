package corpus

import (
	"encoding/json"
	"fmt"
	"github.com/vsoch/gosmeagle/descriptor"
	"github.com/vsoch/gosmeagle/parsers/file"
	"github.com/vsoch/gosmeagle/parsers/x86_64"
	"io/ioutil"
	"log"
	"os"
	"reflect"
)

// A corpus holds a library name, a list of Functions and variables
type Corpus struct {
	Library   string                                      `json:"library"`
	Locations []map[string]descriptor.LocationDescription `json:"locations,omitempty"`
	Disasm    *file.Disasm                                `json:"-"`
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

// Get assembly for a filename
func GetDisasm(filename string) *file.Disasm {

	f, err := file.Open(filename)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	// Populate the corpus depending on the Architecture
	disasm, err := f.Disasm()
	if err != nil {
		log.Fatalf("Cannot disassemble binary: %x", err)
	}
	return disasm
}

// Load a corpus from Json
func Load(filename string) Corpus {

	jsonFile, err := os.Open(filename)
	if err != nil {
		log.Fatal(err)
	}
	defer jsonFile.Close()

	// Read as byte array
	byteArray, err := ioutil.ReadAll(jsonFile)
	if err != nil {
		log.Fatalf("Cannot read %s\n", jsonFile)
	}

	c := Corpus{}
	json.Unmarshal(byteArray, &c)
	return c
}

func (c *Corpus) Parse(f *file.File) {

	// Disassemble first for locations
	disasm, err := f.Disasm()
	if err != nil {
		log.Fatalf("Cannot disassemble binary: %x", err)
	}
	c.Disasm = disasm

	// Prepare a symbol lookup for assembly and call sites - main, etc. won't be in dynamic symbol table
	// asm := c.Disasm.GetGNUAssembly()

	// Parse dwarf for each entry to use
	lookup := f.ParseDwarf()

	// Parse entries based on type (function or variable)
	for _, e := range f.Entries {

		// These are dynamic symbol table symbols
		symbols, err := e.DynamicSymbols()
		if err != nil {
			log.Fatalf("Issue retriving symbols from %s", c.Library)
		}

		// TODO we don't do anything with call sites here
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
		newFunction := x86_64.ParseFunction(f, symbol, entry, c.Disasm)
		loc := map[string]descriptor.LocationDescription{}
		loc["function"] = newFunction
		c.Locations = append(c.Locations, loc)
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
			loc := map[string]descriptor.LocationDescription{}
			loc["variable"] = variable
			c.Locations = append(c.Locations, loc)
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
