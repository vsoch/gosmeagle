package x86_64

import (
	"github.com/vsoch/gosmeagle/descriptor"
	"github.com/vsoch/gosmeagle/parsers/file"
)

// ParseFunction parses a function parameters
func ParseFunction(f *file.File, symbol file.Symbol, entry *file.DwarfEntry) descriptor.FunctionDescription {

	// Prepare list of function parameters
	params := []descriptor.FunctionParameter{}

	// A return value will be included here with name "return"
	for _, c := range (*entry).GetComponents() {
		param := descriptor.FunctionParameter{Name: c.Name, TypeName: c.Type, Size: c.Size, Direction: symbol.GetDirection()}
		params = append(params, param)
	}

	function := descriptor.FunctionDescription{Parameters: params, Name: symbol.GetName()}
	return function
}

// ParseVariable parses a global variable
func ParseVariable(f *file.File, symbol file.Symbol, entry *file.DwarfEntry) descriptor.VariableDescription {

	// We only need one variable
	variable := descriptor.VariableDescription{}

	// A variable will only have one component for itself
	for _, v := range (*entry).GetComponents() {
		variable = descriptor.VariableDescription{Name: v.Name, Type: v.Type, Size: v.Size, Direction: symbol.GetDirection()}
	}
	return variable
}
