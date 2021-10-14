package corpus

import (
	"fmt"
	"github.com/mitchellh/mapstructure"
	"github.com/vsoch/gosmeagle/descriptor"
)

// A loaded corpus keeps types separate
// Haven't needed to convert between them, but could if needed
type LoadedCorpus struct {
	Functions []descriptor.FunctionDescription
	Variables []descriptor.VariableDescription
	Library   string
}

func (c *LoadedCorpus) ToCorpus() *Corpus {

	locs := []map[string]descriptor.LocationDescription{}
	for _, entry := range c.Functions {
		newFunc := map[string]descriptor.LocationDescription{}
		newFunc["function"] = entry
		locs = append(locs, newFunc)
	}
	for _, entry := range c.Variables {
		newVar := map[string]descriptor.LocationDescription{}
		newVar["variable"] = entry
		locs = append(locs, newVar)
	}
	return &Corpus{Library: c.Library, Locations: locs}
}

// Load a corpus from Json
func Load(filename string) LoadedCorpus {

	c := load(filename)
	funcs := []descriptor.FunctionDescription{}
	vars := []descriptor.VariableDescription{}

	for _, entry := range c.Locations {
		function, ok := entry["function"]
		if ok {
			newFunc := convertFunctionDescriptor(function)
			funcs = append(funcs, newFunc)
		}
		variable, ok := entry["variable"]
		if ok {
			newVar := convertVariableDescriptor(variable)
			vars = append(vars, newVar)
		}
	}

	corp := LoadedCorpus{Library: filename}
	corp.Functions = funcs
	corp.Variables = vars
	fmt.Println(corp)
	return corp
}

// convertFunctionDescriptor converts to a function descriptor to
func convertFunctionDescriptor(item interface{}) descriptor.FunctionDescription {
	desc := descriptor.FunctionDescription{}
	desc.Name = item.(map[string]interface{})["name"].(string)
	params := item.(map[string]interface{})["parameters"].([]interface{})
	desc.Parameters = []descriptor.Parameter{}
	desc.Type = "Function"
	for _, param := range params {
		s := descriptor.FunctionParameter{}
		mapstructure.Decode(param, &s)
		desc.Parameters = append(desc.Parameters, s)
	}
	return desc
}

// convertFunctionDescriptor converts to a function descriptor
func convertVariableDescriptor(item interface{}) descriptor.VariableDescription {
	desc := descriptor.VariableDescription{}
	mapstructure.Decode(item, &desc)
	desc.Type = "Variable"
	return desc
}
