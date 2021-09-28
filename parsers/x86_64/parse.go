package x86_64

import (
	"fmt"
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

	// Keep a record of names that we've seen
	seen := map[string]bool{}

	// A return value will be included here with name "return"
	for _, c := range (*entry).GetComponents() {

		// Parse the parameter!
		param := ParseParameter(c, symbol, 0, &seen)
		if param != nil {
			params = append(params, param)
		}
	}

	function := descriptor.FunctionDescription{Parameters: params, Name: symbol.GetName(), Type: "Function"}
	return function
}

// ParseParameter will parse a general parameter
func ParseParameter(c file.Component, symbol file.Symbol, indirections int64, seen *map[string]bool) descriptor.Parameter {

	// Parse parameter based on the type
	switch c.Class {
	case "Pointer":
		return ParsePointerType(c, symbol, indirections, seen)
	case "Qualified":
		return ParseQualifiedType(c, symbol, indirections, seen)
	case "Basic", "Uint", "Int", "Float":
		return ParseBasicType(c, symbol, indirections, seen)
	case "Typedef":
		return ParseTypedef(c, symbol, indirections, seen)
	case "Structure":
		return ParseStructure(c, symbol, indirections, seen)
	case "Array":
		return ParseArray(c, symbol, indirections, seen)
	case "", "Undefined":
		return nil
	default:
		fmt.Println(c.Class)
	}
	return nil
}

//func ParseUndefined(c file.Component, symbol file.Symbol, indirections int64, seen *map[string]bool) descriptor.Parameter {

//	fmt.Println()
//	fmt.Println("Common Type:", convert.CommonType)
//	fmt.Println("Common Type Size:", convert.CommonType.Size())
//	fmt.Println("Common Type Name:", convert.CommonType.Name)
//	fmt.Println("Type:", convert.Type)
//	fmt.Println("Type size:", convert.Type.Size())
//	fmt.Println("Type class:", file.GetStringType(convert.Type))
//	fmt.Println("Type name:", convert.Type.String())
//	fmt.Println("Count (-1 means incomplete):", convert.Count)
//	fmt.Println("Number of bits to hold each element:", convert.StrideBitSize)

//	return descriptor.BasicParameter{}
//}

// ParsePointerType parses a pointer and returns an abi description for a function parameter
func ParsePointerType(c file.Component, symbol file.Symbol, indirections int64, seen *map[string]bool) descriptor.Parameter {

	// Convert Original back to Pointer Type to get underlying type
	convert := c.RawType.(*dwarf.PtrType)

	// Default will return nil (no underlying type to continue parsing)
	underlyingType := ParseParameter(file.Component{}, nil, indirections, seen)

	// Only parse things we haven't seen
	_, ok := (*seen)[convert.Type.Common().Name]
	if !ok {
		comp := file.Component{Name: convert.Type.Common().Name, Class: file.GetStringType(convert.Type),
			Size: convert.Type.Size(), RawType: convert.Type}

		// If we've hit another pointer, this is an indirection
		indirections += 1

		// Mark as seen, and parse the underlying type
		(*seen)[convert.Type.Common().Name] = true
		underlyingType = ParseParameter(comp, nil, indirections, seen)
	}

	// We only know the direction if we have a symbol
	direction := ""
	if symbol != nil {
		direction = symbol.GetDirection()
	}

	return descriptor.PointerParameter{Name: c.Name, Type: c.Type, Class: c.Class,
		Size: c.Size, Direction: direction, UnderlyingType: underlyingType, Indirections: indirections}
}

/* ParseArray parses an array type

1. For formal function parameters, arrays are pointers and have class "Pointer"
2. For fields of a struct/union/class, arrays are real types and have class "Array"
3 .For callsite parameters, arrays are detected as a real type but will be decayed to a pointer for the function call.

TODO: currently just testing with #2 above
*/
func ParseArray(c file.Component, symbol file.Symbol, indirections int64, seen *map[string]bool) descriptor.Parameter {

	convert := c.RawType.(*dwarf.ArrayType)
	underlyingType := ParseParameter(file.Component{}, nil, indirections, seen)

	// Get the kind of items in the array
	_, ok := (*seen)[convert.Type.Common().Name]
	if !ok {
		comp := file.Component{Name: convert.Type.Common().Name, Class: file.GetStringType(convert.Type),
			Size: convert.Type.Size(), RawType: convert.Type}

		// If we've hit another pointer, this is an indirection
		indirections += 1

		// Mark as seen, and parse the underlying type
		(*seen)[convert.Type.Common().Name] = true
		underlyingType = ParseParameter(comp, nil, indirections, seen)
	}

	return descriptor.ArrayParameter{Length: convert.Count, Name: convert.CommonType.Name, Type: convert.Type.String(),
		Size: convert.Count * underlyingType.GetSize(), Class: "Array", ItemType: underlyingType}
}

// ParseStructure parses a structure type
func ParseStructure(c file.Component, symbol file.Symbol, indirections int64, seen *map[string]bool) descriptor.Parameter {

	convert := c.RawType.(*dwarf.StructType)

	fields := []descriptor.Parameter{}
	for _, field := range convert.Field {
		c := file.Component{Name: field.Name, Class: file.GetStringType(field.Type),
			Size: field.Type.Size(), RawType: field.Type}
		newField := ParseParameter(c, nil, indirections, seen)
		if newField != nil {
			fields = append(fields, newField)
		}
	}
	return descriptor.StructureParameter{Fields: fields, Class: strings.Title(convert.Kind), Type: convert.StructName,
		Size: convert.CommonType.Size()}
}

// ParseQualified parses a qualified type (a size and type)
func ParseQualifiedType(c file.Component, symbol file.Symbol, indirections int64, seen *map[string]bool) descriptor.Parameter {

	// We can still have a pointer here!
	switch c.RawType.(type) {
	case *dwarf.PtrType:
		return ParsePointerType(c, symbol, indirections, seen)
	case *dwarf.QualType:
		convert := c.RawType.(*dwarf.QualType)
		return descriptor.QualifiedParameter{Size: convert.Type.Size(), Type: convert.Type.String()}
	}
	return descriptor.QualifiedParameter{}
}

// ParseTypeDef parses a type definition
func ParseTypedef(c file.Component, symbol file.Symbol, indirections int64, seen *map[string]bool) descriptor.Parameter {
	convert := c.RawType.(*dwarf.TypedefType)
	return descriptor.BasicParameter{Name: convert.Name, Size: convert.CommonType.Size(), Type: convert.Type.Common().Name}
}

// ParseBasicType parses a basic type
func ParseBasicType(c file.Component, symbol file.Symbol, indirections int64, seen *map[string]bool) descriptor.Parameter {

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
		return ParsePointerType(c, symbol, indirections, seen)
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
		//		fmt.Println(v.Class)

		// TODO why not include variable classes?
		variable = descriptor.VariableDescription{Name: v.Name, Type: v.Type, Size: v.Size,
			Direction: symbol.GetDirection()}
	}
	return variable
}
