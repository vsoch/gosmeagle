package x86_64

import (
	"fmt"
	"github.com/vsoch/gosmeagle/descriptor"
	"github.com/vsoch/gosmeagle/parsers/file"
)

// parse a function parameters
func ParseFunction(f *file.File, symbol file.Symbol) descriptor.FunctionDescription {

	fmt.Println(symbol)
	param := descriptor.FunctionDescription{}
	return param
}

/*
// TODO need to make different function descriptors for different kinds of params?

