package file

import (
	"fmt"
	"github.com/vsoch/gosmeagle/pkg/debug/dwarf"
	"io"
	"log"
)

// A common interface to represent a dwarf entry (what we need)
type DwarfEntry interface {
	GetComponents() []Component // Can be fields, params, etc.
}

// Types that we need to parse
type FunctionEntry struct {
	Entry  *dwarf.Entry
	Type   *dwarf.Type
	Params []FormalParamEntry
	Data   *dwarf.Data
}

type FormalParamEntry struct {
	Entry *dwarf.Entry
	Type  *dwarf.Type
	Data  *dwarf.Data
}

// A Component can be a Field or param
type Component struct {
	Name      string
	Size      int64
	Type      string
	Framebase string
}

// Function components are the associated fields
func (f *FunctionEntry) GetComponents() []Component {

	comps := []Component{}
	for _, param := range f.Params {

		paramName := param.Entry.Val(dwarf.AttrName)
		if paramName == nil {
			continue
		}
		paramTypeOffset := param.Entry.Val(dwarf.AttrType)
		if paramTypeOffset == nil {
			fmt.Printf("Cannot find offset for %s, skipping\n", paramName)
		}
		paramType, err := f.Data.Type(paramTypeOffset.(dwarf.Offset))
		if err != nil {
			fmt.Printf("Cannot get type for %s\n", paramName)
			continue
		}
		comps = append(comps, Component{Name: (paramName).(string), Type: paramType.Common().Name, Size: paramType.Common().ByteSize})

	}
	fmt.Println(comps)
	return comps
}

// ParseDwarf and populate something / return something?
func ParseDwarf(dwf *dwarf.Data) {

	// TODO - need to save functions to some kind of lookup by name
	// and return to save with the file in two spots?
	// will need to call funcEntry.GetComponents()

	entryReader := dwf.Reader()

	// Keep list of general entries
	entries := []DwarfEntry{}

	// keep track of last function to associate with formal parameters, and if found them
	var functionEntry *dwarf.Entry
	var params []FormalParamEntry

	for entry, err := entryReader.Next(); entry != nil; entry, err = entryReader.Next() {

		// Reached end of file
		if err == io.EOF {
			break
		}

		switch entry.Tag {

		// We found a function - hold onto it for any params
		case dwarf.TagSubprogram:

			// If we have a previous function entry, add it
			if functionEntry != nil {
				entries = append(entries, ParseFunction(dwf, functionEntry, params))
			}

			// Reset params and set new function entry
			functionEntry = entry
			params = []FormalParamEntry{}

		// We match formal parameters to the last function (their parent)
		case dwarf.TagFormalParameter:

			// This shouldn't ever happen
			if functionEntry == nil {
				log.Fatalf("Found formal parameter not associated to function: %s\n", entry)
			}
			params = append(params, ParseFormalParameter(dwf, entry))

		case dwarf.TagTypedef:
			if _, ok := entry.Val(dwarf.AttrName).(string); ok {
				//fmt.Println(value)
			}
		}
	}

	// Parse the last function entry
	if functionEntry != nil {
		entries = append(entries, ParseFunction(dwf, functionEntry, params))
	}
	// Finally, consolidate and parse into records of Create a list of dwarf entry
	// should be like a map so we can look things up?
	// TODO we need a way to look up by id/name
	// function names should be unique for a library?
	//	entries := []DwarfEntry{}

}

// Populate a formal parameter
func ParseFormalParameter(d *dwarf.Data, entry *dwarf.Entry) FormalParamEntry {
	return FormalParamEntry{Entry: entry, Data: d}
}

// Populate a function entry
func ParseFunction(d *dwarf.Data, entry *dwarf.Entry, params []FormalParamEntry) DwarfEntry {
	funcEntry := &FunctionEntry{Entry: entry, Data: d, Params: params}
	funcEntry.GetComponents() // TODO remove, this is debugging only
	return funcEntry
}
