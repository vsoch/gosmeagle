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
}

// Types that we need to parse
type FunctionEntry struct {
	Entry  *dwarf.Entry
	Type   *dwarf.Type
	Params []FormalParamEntry
	Data   *dwarf.Data
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
		Size: varType.Common().ByteSize})
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

	// Get the Return value - the "type" of
	// TODO can we use := entry.Val(dwarf.AttrVarParam)
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

	for entry, err := entryReader.Next(); entry != nil; entry, err = entryReader.Next() {

		// Reached end of file
		if err == io.EOF {
			break
		}

		switch entry.Tag {

		//		case dwarf.TagArrayType, dwarf.TagPointerType, dwarf.TagStructType, dwarf.TagBaseType, dwarf.TagSubroutineType, dwarf.TagTypedef:
		//			fmt.Println(entry)

		// We found a function - hold onto it for any params
		case dwarf.TagSubprogram:

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

			//		case dwarf.TagTypedef:
			//			if _, ok := entry.Val(dwarf.AttrName).(string); ok {
			//				//fmt.Println(value)
			//			}
		}
	}

	// Parse the last function entry
	if functionEntry != nil {
		newEntry := ParseFunction(dwf, functionEntry, params)
		lookup["functions"][newEntry.Name()] = newEntry
	}

	return lookup
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
