package x86_64

import (
	"log"
	"reflect"
	"strings"

	"github.com/vsoch/gosmeagle/descriptor"
	"github.com/vsoch/gosmeagle/parsers/file"
	"github.com/vsoch/gosmeagle/pkg/debug/dwarf"
)

// ParseFunction parses a function parameters
func ParseFunction(f *file.File, symbol file.Symbol, entry *file.DwarfEntry, disasm *file.Disasm, isCallSite bool) descriptor.FunctionDescription {

	// Prepare list of function parameters
	params := []descriptor.Parameter{}

	// Keep a record of names and components we've seen
	seen := map[string]file.Component{}

	// Create an allocator for the function
	allocator := NewRegisterAllocator()

	// Data is needed by typedef to look up full struct, class, or union info
	data := (*entry).GetData()

	// Get the direction for the function
	direction := GetDirection(symbol.GetName(), isCallSite)

	// A return value will be included here with name "return"
	for _, c := range (*entry).GetComponents() {

		indirections := int64(0)

		// Parse the parameter!
		param := ParseParameter(c, data, symbol, &indirections, &seen, allocator, isCallSite)
		if param != nil {
			params = append(params, param)
		}
	}
	return descriptor.FunctionDescription{Parameters: params, Name: symbol.GetName(), Type: "Function", Direction: direction}
}

// ParseParameter will parse a general parameter
func ParseParameter(c file.Component, d *dwarf.Data, symbol file.Symbol, indirections *int64, seen *map[string]file.Component,
	a *RegisterAllocator, isCallSite bool) descriptor.Parameter {

	// Parse parameter based on the type
	switch c.Class {
	case "Pointer":
		return ParsePointerType(c, d, symbol, indirections, seen, a, isCallSite)
	case "Qualified":
		return ParseQualifiedType(c, d, symbol, indirections, seen, a, isCallSite)
	case "Basic", "Uint", "Int", "Float", "Char", "Uchar", "Complex", "Bool", "Unspecified", "Address":
		return ParseBasicType(c, d, symbol, indirections, seen, a, isCallSite)
	case "Enum":
		return ParseEnumType(c, symbol, indirections, a, isCallSite)
	case "Typedef":
		convert := (*d).StructCache[c.Name]
		if convert != nil {
			return ParseStructure(convert, d, symbol, indirections, seen, a, isCallSite)
		}
		return ParseTypedef(c, symbol, indirections, seen, isCallSite)
	case "Structure":
		convert := c.RawType.(*dwarf.StructType)
		return ParseStructure(convert, d, symbol, indirections, seen, a, isCallSite)
	case "Array":
		return ParseArray(c, d, symbol, indirections, seen, a, isCallSite)

	// A nested function here appears to be anonymous (e.g., { Function -1   func(*char) void})
	case "", "Undefined", "Function":
		return nil
	default:
		log.Fatalf("Unparsed parameter class", c.Class)
	}
	return nil
}

// ParseTypeDef parses a type definition
func ParseTypedef(c file.Component, symbol file.Symbol, indirections *int64, seen *map[string]file.Component, isCallSite bool) descriptor.Parameter {
	convert := c.RawType.(*dwarf.TypedefType)
	direction := GetDirection(convert.Name, isCallSite)
	return descriptor.BasicParameter{Name: convert.Name, Size: convert.CommonType.Size(), Type: convert.Type.Common().Name, Direction: direction}
}

// ParseEnumType parses an enum type
func ParseEnumType(c file.Component, symbol file.Symbol, indirections *int64, a *RegisterAllocator, isCallSite bool) descriptor.Parameter {

	convert := c.RawType.(*dwarf.EnumType)

	// Get a list of constants (name and value)
	constants := map[string]int64{}
	for _, value := range convert.Val {
		constants[value.Name] = value.Val
	}

	// Get the location
	enumClass := ClassifyEnum(convert, &c, indirections)
	loc := a.GetRegisterString(enumClass.Lo, enumClass.Hi, c.Size, c.Class)

	// If it's a return value, it's an export, otherwise import (and TODO we need callsite)
	direction := GetDirection(convert.EnumName, isCallSite)

	return descriptor.EnumParameter{Name: convert.EnumName, Type: c.Type, Class: c.Class, Location: loc,
		Size: convert.CommonType.ByteSize, Length: len(convert.Val), Constants: constants, Direction: direction}
}

// Get direction determines the direction (export or import)
func GetDirection(name string, isCallSite bool) string {

	// if is return and not call site, should be export
	if name == "return" && !isCallSite {
		return "export"
	}
	// if is return and call site, should be import
	if name == "return" && isCallSite {
		return "import"
	}
	// If it's not a return and a callsite, should be export
	if isCallSite {
		return "export"
	}
	return "import"
}

// ParsePointerType parses a pointer and returns an abi description for a function parameter
func ParsePointerType(c file.Component, d *dwarf.Data, symbol file.Symbol, indirections *int64, seen *map[string]file.Component,
	a *RegisterAllocator, isCallSite bool) descriptor.Parameter {

	// Convert Original back to Pointer Type to get underlying type
	convert := c.RawType.(*dwarf.PtrType)

	// Default will return nil (no underlying type to continue parsing)
	underlyingType := ParseParameter(file.Component{}, d, nil, indirections, seen, a, isCallSite)

	// Only parse things we haven't seen
	seenComponent, ok := (*seen)[convert.Type.Common().Name]
	if !ok {
		comp := file.Component{Name: convert.Type.Common().Name, Class: file.GetStringType(convert.Type),
			Size: convert.Type.Size(), RawType: convert.Type}

		// If we've hit another pointer, this is an indirection
		(*indirections) += 1

		// Mark as seen, and parse the underlying type
		(*seen)[convert.Type.Common().Name] = comp
		underlyingType = ParseParameter(comp, d, nil, indirections, seen, a, isCallSite)
	}

	// Default direction for a library symbol is import
	// But what about call sites? Why are they all imports?
	direction := GetDirection(c.Name, isCallSite)

	// On x86, all pointers are the same ABI class
	ptrClass := ClassifyPointer(indirections)

	// Allocate space for the pointer (NOT the underlying type)
	ptrLoc := a.GetRegisterString(ptrClass.Lo, ptrClass.Hi, seenComponent.Size, seenComponent.Class)

	return descriptor.PointerParameter{Name: c.Name, Type: c.Type, Class: c.Class, Location: ptrLoc,
		Size: c.Size, Direction: direction, UnderlyingType: underlyingType, Indirections: (*indirections)}
}

/* ParseArray parses an array type

1. For formal function parameters, arrays are pointers and have class "Pointer"
2. For fields of a struct/union/class, arrays are real types and have class "Array"
3 .For callsite parameters, arrays are detected as a real type but will be decayed to a pointer for the function call.

TODO: currently just testing with #2 above
*/
func ParseArray(c file.Component, d *dwarf.Data, symbol file.Symbol, indirections *int64, seen *map[string]file.Component,
	a *RegisterAllocator, isCallSite bool) descriptor.Parameter {

	convert := c.RawType.(*dwarf.ArrayType)

	// Default will return nil (no underlying type to continue parsing)
	underlyingType := ParseParameter(file.Component{}, d, nil, indirections, seen, a, isCallSite)

	// Only parse things we haven't seen
	seenComponent, ok := (*seen)[convert.Type.Common().Name]
	if !ok {
		comp := file.Component{Name: convert.Type.Common().Name, Class: file.GetStringType(convert.Type),
			Size: convert.Type.Size(), RawType: convert.Type}

		// If we've hit another pointer, this is an indirection
		(*indirections) += 1

		// Mark as seen, and parse the underlying type
		(*seen)[convert.Type.Common().Name] = comp
		underlyingType = ParseParameter(comp, d, nil, indirections, seen, a, isCallSite)
	}

	arrayClass := ClassifyArray(convert, &seenComponent, indirections)
	loc := a.GetRegisterString(arrayClass.Lo, arrayClass.Hi, seenComponent.Size, seenComponent.Class)
	direction := GetDirection(convert.CommonType.Name, isCallSite)
	return descriptor.ArrayParameter{Length: convert.Count, Name: convert.CommonType.Name, Type: convert.Type.String(),
		Size: convert.Count * seenComponent.Size, Class: "Array", ItemType: underlyingType, Location: loc, Direction: direction}
}

// ParseStructure parses a structure type
func ParseStructure(convert *dwarf.StructType, d *dwarf.Data, symbol file.Symbol, indirections *int64, seen *map[string]file.Component,
	a *RegisterAllocator, isCallSite bool) descriptor.Parameter {

	fields := []descriptor.Parameter{}
	for _, field := range convert.Field {
		c := file.Component{Name: field.Name, Class: file.GetStringType(field.Type),
			Size: field.Type.Size(), RawType: field.Type}
		newField := ParseParameter(c, d, nil, indirections, seen, a, isCallSite)
		if newField != nil {
			fields = append(fields, newField)
		}
	}

	direction := GetDirection("", isCallSite)
	// Get the location ?
	// structClass := ClassifyStruct(convert, &c, indirections)
	// loc := a.GetRegisterString(structClass.Lo, structClass.Hi, c.Size, c.Class)
	return descriptor.StructureParameter{Fields: fields, Class: strings.Title(convert.Kind), Type: convert.StructName,
		Size: convert.CommonType.Size(), Direction: direction}
}

// ParseQualified parses a qualified type (a size and type)
func ParseQualifiedType(c file.Component, d *dwarf.Data, symbol file.Symbol, indirections *int64, seen *map[string]file.Component,
	a *RegisterAllocator, isCallSite bool) descriptor.Parameter {

	// We can still have a pointer here!
	switch c.RawType.(type) {
	case *dwarf.PtrType:
		return ParsePointerType(c, d, symbol, indirections, seen, a, isCallSite)
	case *dwarf.QualType:
		convert := c.RawType.(*dwarf.QualType)
		direction := GetDirection("", isCallSite)
		return descriptor.QualifiedParameter{Size: convert.Type.Size(), Type: convert.Type.String(), Class: "Qual",
			Direction: direction}
	}
	return descriptor.QualifiedParameter{}
}

// ParseBasicType parses a basic type
func ParseBasicType(c file.Component, d *dwarf.Data, symbol file.Symbol, indirections *int64, seen *map[string]file.Component,
	a *RegisterAllocator, isCallSite bool) descriptor.Parameter {

	direction := GetDirection("", isCallSite)
	switch c.RawType.(type) {
	case *dwarf.IntType:
		convert := c.RawType.(*dwarf.IntType)
		return descriptor.BasicParameter{Size: convert.CommonType.Size(), Type: convert.CommonType.Name, Class: "Int", Direction: direction}
	case *dwarf.FloatType:
		convert := c.RawType.(*dwarf.FloatType)
		return descriptor.BasicParameter{Size: convert.CommonType.Size(), Type: convert.CommonType.Name, Class: "Float", Direction: direction}
	case *dwarf.UintType:
		convert := c.RawType.(*dwarf.UintType)
		return descriptor.BasicParameter{Size: convert.CommonType.Size(), Type: convert.CommonType.Name, Class: "Uint", Direction: direction}
	case *dwarf.UcharType:
		convert := c.RawType.(*dwarf.UcharType)
		return descriptor.BasicParameter{Size: convert.CommonType.Size(), Type: convert.CommonType.Name, Class: "Uchar", Direction: direction}
	case *dwarf.CharType:
		convert := c.RawType.(*dwarf.CharType)
		return descriptor.BasicParameter{Size: convert.CommonType.Size(), Type: convert.CommonType.Name, Class: "Char", Direction: direction}
	case *dwarf.ComplexType:
		convert := c.RawType.(*dwarf.ComplexType)
		return descriptor.BasicParameter{Size: convert.CommonType.Size(), Type: convert.CommonType.Name, Class: "Complex", Direction: direction}
	case *dwarf.BoolType:
		convert := c.RawType.(*dwarf.BoolType)
		return descriptor.BasicParameter{Size: convert.CommonType.Size(), Type: convert.CommonType.Name, Class: "Bool", Direction: direction}
	case *dwarf.UnspecifiedType:
		convert := c.RawType.(*dwarf.UnspecifiedType)
		return descriptor.BasicParameter{Size: convert.CommonType.Size(), Type: convert.CommonType.Name, Class: "Unspecified", Direction: direction}
	case *dwarf.AddrType:
		convert := c.RawType.(*dwarf.AddrType)
		return descriptor.BasicParameter{Size: convert.CommonType.Size(), Type: convert.CommonType.Name, Class: "Address", Direction: direction}
	case *dwarf.PtrType:
		return ParsePointerType(c, d, symbol, indirections, seen, a, isCallSite)
	case *dwarf.BasicType:
		convert := c.RawType.(*dwarf.BasicType)
		return descriptor.BasicParameter{Size: convert.CommonType.Size(), Type: convert.CommonType.Name, Direction: direction}
	default:
		log.Fatalf("Type not accounted for:", reflect.TypeOf(c.RawType))
	}
	return descriptor.BasicParameter{}
}

// ParseVariable parses a global variable
func ParseVariable(f *file.File, symbol file.Symbol, entry *file.DwarfEntry, isCallSite bool) descriptor.VariableDescription {

	// We only need one variable
	variable := descriptor.VariableDescription{}

	// A variable will only have one component for itself
	for _, v := range (*entry).GetComponents() {
		direction := GetDirection(v.Name, isCallSite)
		variable = descriptor.VariableDescription{Name: v.Name, Type: v.Type, Size: v.Size, Direction: direction}
	}
	return variable
}
