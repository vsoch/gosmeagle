package file

import (
	"fmt"
	"github.com/vsoch/gosmeagle/pkg/debug/dwarf"
	"io"
	"reflect"
)

// A common interface to represent a dwarf entry (what we need)
type DwarfEntry interface {
	GetComponents() []Component // Can be fields, params, etc.
	Name() string
	GetData() *dwarf.Data
	GetEntry() *dwarf.Entry
	GetType() *dwarf.Type
}

// Types that we need to parse
type FunctionEntry struct {
	Entry  *dwarf.Entry
	Type   *dwarf.Type
	Params []FormalParamEntry
	Data   *dwarf.Data
}

// Preparing a call site to link to a function / caller
type CallSite struct {
	Entry  dwarf.Entry
	Params []dwarf.Entry
}

// Types that we need to parse
type VariableEntry struct {
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
	Class     string
	Size      int64
	Type      string
	Framebase string
	RawType   interface{} // the original type
}

// GetUnderlyingType for a parameter or return value from AttrType
func GetUnderlyingType(entry *dwarf.Entry, data *dwarf.Data) (dwarf.Type, error) {
	paramTypeOffset := entry.Val(dwarf.AttrType)

	if paramTypeOffset == nil {
		return nil, nil
	}
	return data.Type(paramTypeOffset.(dwarf.Offset))
}

// Expose data and type
func (f *FunctionEntry) GetData() *dwarf.Data   { return f.Data }
func (v *VariableEntry) GetData() *dwarf.Data   { return v.Data }
func (f *FunctionEntry) GetType() *dwarf.Type   { return f.Type }
func (v *VariableEntry) GetType() *dwarf.Type   { return v.Type }
func (f *FunctionEntry) GetEntry() *dwarf.Entry { return f.Entry }
func (v *VariableEntry) GetEntry() *dwarf.Entry { return v.Entry }

// Get the name of the entry or formal param
func (f *FunctionEntry) Name() string {

	linkageName := f.Entry.Val(dwarf.AttrLinkageName)
	if linkageName != nil {
		return linkageName.(string)
	}
	functionName := f.Entry.Val(dwarf.AttrName)
	if functionName == nil {
		return "anonymous"
	}
	return functionName.(string)
}

// Variable components is just one for the variable
func (v *VariableEntry) GetComponents() []Component {
	comps := []Component{}

	// Can we have variables without names?
	varName := v.Entry.Val(dwarf.AttrName)
	if varName == nil {
		return comps
	}
	varType, err := GetUnderlyingType(v.Entry, v.Data)

	if err != nil {
		return comps
	}
	// It looks like the Common().Name is empty here?
	comps = append(comps, Component{Name: (varName).(string), Type: varType.String(),
		RawType: varType.Common().Original, Class: GetStringType(varType),
		Size: varType.Size()})
	return comps
}

// Get the name of the entry or formal param
func (v *VariableEntry) Name() string {
	name := v.Entry.Val(dwarf.AttrName)
	if name != nil {
		return name.(string)
	}
	return ""
}

// Given a type, return a string representation
func GetStringType(t dwarf.Type) string {
	switch t.Common().Original.(type) {
	case *dwarf.EnumType:
		return "Enum"
	case *dwarf.FuncType:
		return "Function"
	case *dwarf.StructType:
		return "Structure"
	case *dwarf.QualType:
		return "Qualified"
	case *dwarf.PtrType:
		return "Pointer"
	case *dwarf.TypedefType:
		return "Typedef"
	case *dwarf.BasicType:
		return "Basic"
	case *dwarf.IntType:
		return "Int"
	case *dwarf.FloatType:
		return "Float"
	case *dwarf.ArrayType:
		return "Array"
	case *dwarf.UintType:
		return "Uint"
	case *dwarf.CharType:
		return "Char"
	case *dwarf.UcharType:
		return "Uchar"
	case *dwarf.BoolType:
		return "Bool"
	case *dwarf.ComplexType:
		return "Complex"
	case nil:
		return "Undefined"
	default:
		fmt.Println("Unaccounted for type", reflect.TypeOf(t.Common().Original))
	}
	return "Unknown"
}

// Function components are the associated fields
func (f *FunctionEntry) GetComponents() []Component {

	comps := []Component{}
	for _, param := range f.Params {

		paramName := param.Entry.Val(dwarf.AttrName)
		if paramName == nil {
			continue
		}

		paramType, err := GetUnderlyingType(param.Entry, f.Data)

		// TODO do we need to remove const here?
		// From Tim: Dyninst reconstructs CV qualifiers and packedness using DW_TAG_{const,packed,volatile}_type and then manually updating the name. It's a bit hacky, but see DwarfWalker::parseConstPackedVolatile
		if err != nil {
			fmt.Printf("Cannot get type for %s\n", paramName)
			continue
		}

		comps = append(comps, Component{Name: (paramName).(string), Type: paramType.Common().Name,
			Class: GetStringType(paramType), Size: paramType.Common().ByteSize,
			RawType: paramType.Common().Original})

	}

	// Get the Return value - for a library this is the only export (unless a call site)
	returnType, err := GetUnderlyingType(f.Entry, f.Data)
	if returnType != nil && err != nil {
		comps = append(comps, Component{Name: "return", Type: returnType.Common().Name,
			Class: GetStringType(returnType), Size: returnType.Common().ByteSize,
			RawType: returnType.Common().Original})
	}
	return comps
}

// ParseDwarf and populate a lookup of Dwarf entries
func ParseDwarf(dwf *dwarf.Data) map[string]map[string]DwarfEntry {

	// We will return a lookup of Dwarf entry
	lookup := map[string]map[string]DwarfEntry{}
	lookup["functions"] = map[string]DwarfEntry{}
	lookup["variables"] = map[string]DwarfEntry{}

	// The reader will help us parse the DIEs
	entryReader := dwf.Reader()

	// keep track of last function to associate with formal parameters, and if found them
	var functionEntry *dwarf.Entry
	var params []FormalParamEntry

	// Save a cache of call sites, params, and subprogram locations
	var callSite *dwarf.Entry
	var callSites []CallSite
	var callSiteParams []dwarf.Entry
	subprograms := map[dwarf.Offset]dwarf.Entry{}

	for entry, err := entryReader.Next(); entry != nil; entry, err = entryReader.Next() {

		// Reached end of file
		if err == io.EOF {
			break
		}

		switch entry.Tag {

		// DW_TAG_GNU_call_site is older version
		case 0x4109, dwarf.TagCallSite, 0x44:

			// If we have a previous function entry, add it
			if callSite != nil {
				callSites = append(callSites, CallSite{Entry: (*callSite), Params: callSiteParams})
			}

			// Reset params and set new function entry
			callSite = entry
			callSiteParams = []dwarf.Entry{}

		// DW_TAG_GNU_call_site_parameter
		case 0x410a, dwarf.TagCallSiteParameter, 0x45:
			callSiteParams = append(callSiteParams, (*entry))

		// We found a function - hold onto it for any params
		case dwarf.TagClassType:
			subprograms[entry.Offset] = (*entry)

		// We found a function - hold onto it for any params
		case dwarf.TagSubprogram:
			subprograms[entry.Offset] = (*entry)

			// If we have a previous function entry, add it
			if functionEntry != nil {
				newEntry := ParseFunction(dwf, functionEntry, params)
				lookup["functions"][newEntry.Name()] = newEntry
			}

			// Reset params and set new function entry
			functionEntry = entry
			params = []FormalParamEntry{}

		// We match formal parameters to the last function (their parent)
		case dwarf.TagFormalParameter:

			// Skip formal params that don't have linked function
			if functionEntry == nil {
				continue
			}
			params = append(params, ParseFormalParameter(dwf, entry))

		// We've found a variable. We can make this more efficient by limiting to global
		case dwarf.TagVariable:
			newVariable := ParseVariable(dwf, entry)
			lookup["variables"][newVariable.Name()] = newVariable
		}
	}

	// Parse the last function entry
	if functionEntry != nil {
		newEntry := ParseFunction(dwf, functionEntry, params)
		lookup["functions"][newEntry.Name()] = newEntry
	}

	// Match call sites to subprograms
	lookup["calls"] = ParseCallSites(dwf, &callSites, &subprograms, lookup["functions"])
	return lookup
}

// Parse Call sites into a map of DwarfEntry
func ParseCallSites(d *dwarf.Data, callSites *[]CallSite, subprograms *map[dwarf.Offset]dwarf.Entry, functions map[string]DwarfEntry) map[string]DwarfEntry {
	entries := map[string]DwarfEntry{}

	for _, cs := range *callSites {

		loc := cs.Entry.Val(dwarf.AttrLocation)
		if loc == nil {
			loc = cs.Entry.Val(dwarf.AttrCallOrigin)
		}
		if loc != nil {
			programEntry, ok := (*subprograms)[loc.(dwarf.Offset)]

			if ok {

				name := programEntry.Val(dwarf.AttrLinkageName)
				if name == "" {
					name = programEntry.Val(dwarf.AttrName)
				}
				function, ok := functions[name.(string)]
				if ok {
					entries[name.(string)] = function
				} else {
					fmt.Println("no function entry for", cs)
				}

				// NOTE that we have values here, type []uint8 p.Val(dwarf.AttrCallValue) we aren't parsing!
				// TODO not sure how to look up location in location lists
				//newEntry := FunctionEntry{Entry: &programEntry, Data: d}
				//entries[newEntry.Name()] = &newEntry
			} // TODO need to handle these tail calls!
			// These are call sites with CallTailCall true, meaning we need
			// another way to look them up, maybe the return PC?
		}
	}
	return entries
}

// Populate a formal parameter
func ParseFormalParameter(d *dwarf.Data, entry *dwarf.Entry) FormalParamEntry {
	return FormalParamEntry{Entry: entry, Data: d}
}

// Populate a function entry
func ParseFunction(d *dwarf.Data, entry *dwarf.Entry, params []FormalParamEntry) DwarfEntry {
	return &FunctionEntry{Entry: entry, Data: d, Params: params}
}

// Populate a variable entry
func ParseVariable(d *dwarf.Data, entry *dwarf.Entry) DwarfEntry {
	return &VariableEntry{Entry: entry, Data: d}
}
