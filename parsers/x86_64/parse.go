package x86_64

import (
	"github.com/vsoch/gosmeagle/descriptor"
	"github.com/vsoch/gosmeagle/parsers/file"
	"github.com/vsoch/gosmeagle/pkg/debug/dwarf"
	"log"
	"reflect"
	"strings"
)

// ParseFunction parses a function parameters
func ParseFunction(f *file.File, symbol file.Symbol, entry *file.DwarfEntry) descriptor.FunctionDescription {

	// Prepare list of function parameters
	params := []descriptor.Parameter{}

	// Keep a record of names and components we've seen
	seen := map[string]file.Component{}

	// Create an allocator for the function
	allocator := NewRegisterAllocator()

	// Data is needed by typedef to look up full struct, class, or union info
	data := (*entry).GetData()

	// A return value will be included here with name "return"
	for _, c := range (*entry).GetComponents() {

		indirections := int64(0)

		// Parse the parameter!
		param := ParseParameter(c, data, symbol, &indirections, &seen, allocator)
		if param != nil {
			params = append(params, param)
		}
	}

	// TODO do we need a location here too?
	return descriptor.FunctionDescription{Parameters: params, Name: symbol.GetName(), Type: "Function"}
}

// ParseParameter will parse a general parameter
func ParseParameter(c file.Component, d *dwarf.Data, symbol file.Symbol, indirections *int64, seen *map[string]file.Component, a *RegisterAllocator) descriptor.Parameter {

	// Parse parameter based on the type
	switch c.Class {
	case "Pointer":
		return ParsePointerType(c, d, symbol, indirections, seen, a)
	case "Qualified":
		return ParseQualifiedType(c, d, symbol, indirections, seen, a)
	case "Basic", "Uint", "Int", "Float", "Char", "Uchar", "Complex", "Bool", "Unspecified", "Address":
		return ParseBasicType(c, d, symbol, indirections, seen, a)
	case "Enum":
		return ParseEnumType(c, symbol, indirections, a)
	case "Typedef":
		convert := (*d).StructCache[c.Name]
		if convert != nil {
			return ParseStructure(convert, d, symbol, indirections, seen, a)
		}
		return ParseTypedef(c, symbol, indirections, seen)
	case "Structure":
		convert := c.RawType.(*dwarf.StructType)
		return ParseStructure(convert, d, symbol, indirections, seen, a)
	case "Array":
		return ParseArray(c, d, symbol, indirections, seen, a)
	case "Function":
		// TODO need to debug if this should be here
		return descriptor.BasicParameter{}
	case "", "Undefined":
		return nil
	default:
		log.Fatalf("Unparsed parameter class", c.Class)
	}
	return nil
}

// ParseTypeDef parses a type definition
func ParseTypedef(c file.Component, symbol file.Symbol, indirections *int64, seen *map[string]file.Component) descriptor.Parameter {
	convert := c.RawType.(*dwarf.TypedefType)
	return descriptor.BasicParameter{Name: convert.Name, Size: convert.CommonType.Size(), Type: convert.Type.Common().Name}
}

// ParseEnumType parses an enum type
func ParseEnumType(c file.Component, symbol file.Symbol, indirections *int64, a *RegisterAllocator) descriptor.Parameter {

	convert := c.RawType.(*dwarf.EnumType)

	// Get a list of constants (name and value)
	constants := map[string]int64{}
	for _, value := range convert.Val {
		constants[value.Name] = value.Val
	}

	// Get the location
	enumClass := ClassifyEnum(convert, &c, indirections)
	loc := a.GetRegisterString(enumClass.Lo, enumClass.Hi, c.Size, c.Class)

	return descriptor.EnumParameter{Name: convert.EnumName, Type: c.Type, Class: c.Class, Location: loc,
		Size: convert.CommonType.ByteSize, Length: len(convert.Val), Constants: constants}
}

// ParsePointerType parses a pointer and returns an abi description for a function parameter
func ParsePointerType(c file.Component, d *dwarf.Data, symbol file.Symbol, indirections *int64, seen *map[string]file.Component,
	a *RegisterAllocator) descriptor.Parameter {

	// Convert Original back to Pointer Type to get underlying type
	convert := c.RawType.(*dwarf.PtrType)

	// Default will return nil (no underlying type to continue parsing)
	underlyingType := ParseParameter(file.Component{}, d, nil, indirections, seen, a)

	// Only parse things we haven't seen
	seenComponent, ok := (*seen)[convert.Type.Common().Name]
	if !ok {
		comp := file.Component{Name: convert.Type.Common().Name, Class: file.GetStringType(convert.Type),
			Size: convert.Type.Size(), RawType: convert.Type}

		// If we've hit another pointer, this is an indirection
		(*indirections) += 1

		// Mark as seen, and parse the underlying type
		(*seen)[convert.Type.Common().Name] = comp
		underlyingType = ParseParameter(comp, d, nil, indirections, seen, a)
	}

	// We only know the direction if we have a symbol
	direction := ""
	if symbol != nil {
		direction = symbol.GetDirection()
	}

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
func ParseArray(c file.Component, d *dwarf.Data, symbol file.Symbol, indirections *int64, seen *map[string]file.Component, a *RegisterAllocator) descriptor.Parameter {

	convert := c.RawType.(*dwarf.ArrayType)

	// Default will return nil (no underlying type to continue parsing)
	underlyingType := ParseParameter(file.Component{}, d, nil, indirections, seen, a)

	// Only parse things we haven't seen
	seenComponent, ok := (*seen)[convert.Type.Common().Name]
	if !ok {
		comp := file.Component{Name: convert.Type.Common().Name, Class: file.GetStringType(convert.Type),
			Size: convert.Type.Size(), RawType: convert.Type}

		// If we've hit another pointer, this is an indirection
		(*indirections) += 1

		// Mark as seen, and parse the underlying type
		(*seen)[convert.Type.Common().Name] = comp
		underlyingType = ParseParameter(comp, d, nil, indirections, seen, a)
	}

	arrayClass := ClassifyArray(convert, &seenComponent, indirections)
	loc := a.GetRegisterString(arrayClass.Lo, arrayClass.Hi, seenComponent.Size, seenComponent.Class)

	// TODO need to debug why this can be nil (shouldn't be)
	if underlyingType != nil {
		return descriptor.ArrayParameter{Length: convert.Count, Name: convert.CommonType.Name, Type: convert.Type.String(),
			Size: convert.Count * underlyingType.GetSize(), Class: "Array", ItemType: underlyingType, Location: loc}
	}
	return descriptor.ArrayParameter{}
}

// ParseStructure parses a structure type
func ParseStructure(convert *dwarf.StructType, d *dwarf.Data, symbol file.Symbol, indirections *int64, seen *map[string]file.Component, a *RegisterAllocator) descriptor.Parameter {

	fields := []descriptor.Parameter{}
	for _, field := range convert.Field {
		c := file.Component{Name: field.Name, Class: file.GetStringType(field.Type),
			Size: field.Type.Size(), RawType: field.Type}
		newField := ParseParameter(c, d, nil, indirections, seen, a)
		if newField != nil {
			fields = append(fields, newField)
		}
	}
	// Get the location ?
	// structClass := ClassifyStruct(convert, &c, indirections)
	// loc := a.GetRegisterString(structClass.Lo, structClass.Hi, c.Size, c.Class)

	return descriptor.StructureParameter{Fields: fields, Class: strings.Title(convert.Kind), Type: convert.StructName,
		Size: convert.CommonType.Size()}
}

// ParseQualified parses a qualified type (a size and type)
func ParseQualifiedType(c file.Component, d *dwarf.Data, symbol file.Symbol, indirections *int64, seen *map[string]file.Component, a *RegisterAllocator) descriptor.Parameter {

	// We can still have a pointer here!
	switch c.RawType.(type) {
	case *dwarf.PtrType:
		return ParsePointerType(c, d, symbol, indirections, seen, a)
	case *dwarf.QualType:
		convert := c.RawType.(*dwarf.QualType)
		return descriptor.QualifiedParameter{Size: convert.Type.Size(), Type: convert.Type.String(), Class: "Qual"}
	}
	return descriptor.QualifiedParameter{}
}

// ParseBasicType parses a basic type
func ParseBasicType(c file.Component, d *dwarf.Data, symbol file.Symbol, indirections *int64, seen *map[string]file.Component, a *RegisterAllocator) descriptor.Parameter {

	switch c.RawType.(type) {
	case *dwarf.IntType:
		convert := c.RawType.(*dwarf.IntType)
		return descriptor.BasicParameter{Size: convert.CommonType.Size(), Type: convert.CommonType.Name, Class: "Int"}
	case *dwarf.FloatType:
		convert := c.RawType.(*dwarf.FloatType)
		return descriptor.BasicParameter{Size: convert.CommonType.Size(), Type: convert.CommonType.Name, Class: "Float"}
	case *dwarf.UintType:
		convert := c.RawType.(*dwarf.UintType)
		return descriptor.BasicParameter{Size: convert.CommonType.Size(), Type: convert.CommonType.Name, Class: "Uint"}
	case *dwarf.UcharType:
		convert := c.RawType.(*dwarf.UcharType)
		return descriptor.BasicParameter{Size: convert.CommonType.Size(), Type: convert.CommonType.Name, Class: "Uchar"}
	case *dwarf.CharType:
		convert := c.RawType.(*dwarf.CharType)
		return descriptor.BasicParameter{Size: convert.CommonType.Size(), Type: convert.CommonType.Name, Class: "Char"}
	case *dwarf.ComplexType:
		convert := c.RawType.(*dwarf.ComplexType)
		return descriptor.BasicParameter{Size: convert.CommonType.Size(), Type: convert.CommonType.Name, Class: "Complex"}
	case *dwarf.BoolType:
		convert := c.RawType.(*dwarf.BoolType)
		return descriptor.BasicParameter{Size: convert.CommonType.Size(), Type: convert.CommonType.Name, Class: "Bool"}
	case *dwarf.UnspecifiedType:
		convert := c.RawType.(*dwarf.UnspecifiedType)
		return descriptor.BasicParameter{Size: convert.CommonType.Size(), Type: convert.CommonType.Name, Class: "Unspecified"}
	case *dwarf.AddrType:
		convert := c.RawType.(*dwarf.AddrType)
		return descriptor.BasicParameter{Size: convert.CommonType.Size(), Type: convert.CommonType.Name, Class: "Address"}
	case *dwarf.PtrType:
		return ParsePointerType(c, d, symbol, indirections, seen, a)
	case *dwarf.BasicType:
		convert := c.RawType.(*dwarf.BasicType)
		return descriptor.BasicParameter{Size: convert.CommonType.Size(), Type: convert.CommonType.Name}
	default:
		log.Fatalf("Type not accounted for:", reflect.TypeOf(c.RawType))
	}
	return descriptor.BasicParameter{}
}

// ParseVariable parses a global variable
func ParseVariable(f *file.File, symbol file.Symbol, entry *file.DwarfEntry) descriptor.VariableDescription {

	// We only need one variable
	variable := descriptor.VariableDescription{}

	// A variable will only have one component for itself
	for _, v := range (*entry).GetComponents() {
		variable = descriptor.VariableDescription{Name: v.Name, Type: v.Type, Size: v.Size}
	}
	return variable
}
