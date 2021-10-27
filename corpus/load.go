package corpus

import (
	"github.com/mitchellh/mapstructure"
	"github.com/vsoch/gosmeagle/descriptor"
	"log"
	"strconv"
)

// LoadedCorpus keeps types separate for easy parsing / interaction
type LoadedCorpus struct {
	Functions []descriptor.FunctionDescription
	Variables []descriptor.VariableDescription
	Library   string
}

// ToCorpus converts a loaded corpus (intended to modify or interact with)
// to a corpus with a list of locations we can save
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
	return corp
}

// loadParameter loads a general parameter
func loadParameter(param interface{}) descriptor.Parameter {

	paramClass := param.(map[string]interface{})["class"]

	switch paramClass {
	case "Pointer":
		return loadPointer(param)
	case "Struct":
		return loadStructure(param)
	}

	// Integer, Float
	return loadFunctionParameter(param)
}

func loadFunctionParameter(param interface{}) descriptor.Parameter {
	s := descriptor.FunctionParameter{}
	mapstructure.Decode(param, &s)
	sizeInt, err := strconv.ParseInt(param.(map[string]interface{})["size"].(string), 10, 64)
	if err != nil {
		log.Fatalf("Error converting string of size to int64: %x", err)
	}
	s.Size = sizeInt
	return s
}

func loadPointer(param interface{}) descriptor.Parameter {

	s := descriptor.PointerParameter{}
	mapstructure.Decode(param, &s)
	sizeInt, err := strconv.ParseInt(param.(map[string]interface{})["size"].(string), 10, 64)
	if err != nil {
		log.Fatalf("Error converting string of size to int64: %x", err)
	}
	indirections, err := strconv.ParseInt(param.(map[string]interface{})["indirections"].(string), 10, 64)
	s.UnderlyingType = loadParameter(param.(map[string]interface{})["underlying_type"])
	s.Size = sizeInt
	s.Indirections = indirections
	return s
}

// convertFunctionDescriptor converts to a function descriptor to
func convertFunctionDescriptor(item interface{}) descriptor.FunctionDescription {
	desc := descriptor.FunctionDescription{}
	desc.Name = item.(map[string]interface{})["name"].(string)
	desc.Parameters = []descriptor.Parameter{}
	desc.Type = "Function"

	paramsection := item.(map[string]interface{})["parameters"]
	if paramsection != nil {
		params := paramsection.([]interface{})
		for _, param := range params {
			newParam := loadParameter(param)
			desc.Parameters = append(desc.Parameters, newParam)
		}
	}
	return desc
}

// loadStructure will load a structure
func loadStructure(param interface{}) descriptor.Parameter {

	s := descriptor.StructureParameter{}
	mapstructure.Decode(param, &s)

	// If we don't reset, the above will load nils
	s.Fields = []descriptor.Parameter{}
	sizeInt, err := strconv.ParseInt(param.(map[string]interface{})["size"].(string), 10, 64)
	if err != nil {
		log.Fatalf("Error converting string of size to int64: %x", err)
	}
	fieldsraw := param.(map[string]interface{})["fields"]
	if fieldsraw != nil {
		fields := fieldsraw.([]interface{})
		for _, field := range fields {
			newField := loadParameter(field)
			if newField != nil {
				s.Fields = append(s.Fields, newField)
			}
		}
	}
	s.Size = sizeInt
	return s

}

// convertFunctionDescriptor converts to a function descriptor
func convertVariableDescriptor(item interface{}) descriptor.VariableDescription {
	desc := descriptor.VariableDescription{}
	mapstructure.Decode(item, &desc)
	desc.Type = "Variable"
	return desc
}
