package descriptor

// Sizes are in bytes

// A general interface to support different parameter types
type Parameter interface {
	GetSize() int64
	GetClass() string
	GetName() string
}

// A General Location description holds a variable or function
type LocationDescription interface{}

// A function description has a list of parameters
type FunctionDescription struct {
	Parameters []Parameter `json:"parameters,omitempty"`
	Name       string      `json:"name"`
	Type       string      `json:"type"`
}

type FunctionParameter struct {
	Name      string `json:"name,omitempty"`
	Type      string `json:"type,omitempty"`
	Class     string `json:"class,omitempty"`
	Direction string `json:"direction,omitempty"`
	Location  string `json:"location,omitempty"`
	Size      int64  `json:"size,omitempty"`
}

// All types can return a size and name
func (f FunctionParameter) GetSize() int64  { return f.Size }
func (f StructureParameter) GetSize() int64 { return f.Size }
func (f PointerParameter) GetSize() int64   { return f.Size }
func (f ArrayParameter) GetSize() int64     { return f.Size }
func (f QualifiedParameter) GetSize() int64 { return f.Size }
func (f BasicParameter) GetSize() int64     { return f.Size }
func (f EnumParameter) GetSize() int64      { return f.Size }

func (f FunctionParameter) GetClass() string  { return f.Class }
func (f StructureParameter) GetClass() string { return f.Class }
func (f PointerParameter) GetClass() string   { return f.Class }
func (f ArrayParameter) GetClass() string     { return f.Class }
func (f QualifiedParameter) GetClass() string { return f.Class }
func (f BasicParameter) GetClass() string     { return f.Class }
func (f EnumParameter) GetClass() string      { return f.Class }

func (f FunctionParameter) GetName() string  { return f.Name }
func (f StructureParameter) GetName() string { return f.Name }
func (f PointerParameter) GetName() string   { return f.Name }
func (f ArrayParameter) GetName() string     { return f.Name }
func (f QualifiedParameter) GetName() string { return f.Name }
func (f BasicParameter) GetName() string     { return f.Name }
func (f EnumParameter) GetName() string      { return f.Name }

type StructureParameter struct {
	Name     string      `json:"name,omitempty"`
	Type     string      `json:"type,omitempty"`
	Class    string      `json:"class,omitempty"`
	Size     int64       `json:"size,omitempty"`
	Location string      `json:"location,omitempty"`
	Fields   []Parameter `json:"fields,omitempty"`
}

type PointerParameter struct {
	Name           string    `json:"name,omitempty"`
	Type           string    `json:"type,omitempty"`
	Class          string    `json:"class,omitempty"`
	Direction      string    `json:"direction,omitempty"`
	Location       string    `json:"location,omitempty"`
	Size           int64     `json:"size,omitempty"`
	UnderlyingType Parameter `json:"underlying_type,omitempty"`
	Indirections   int64     `json:"indirections,omitempty"`
}

type ArrayParameter struct {
	Name     string    `json:"name,omitempty"`
	Type     string    `json:"type,omitempty"`
	Class    string    `json:"class,omitempty"`
	Size     int64     `json:"size,omitempty"`
	Length   int64     `json:"count,omitempty"`
	Location string    `json:"location,omitempty"`
	ItemType Parameter `json:"items_type,omitemtpy"`
}

type EnumParameter struct {
	Name      string           `json:"name,omitempty"`
	Type      string           `json:"type,omitempty"`
	Class     string           `json:"class,omitempty"`
	Size      int64            `json:"size,omitempty"`
	Location  string           `json:"location,omitempty"`
	Length    int              `json:"count,omitempty"`
	Constants map[string]int64 `json:"constants,omitemtpy"`
}

// QualifiedParameter and BasicParameter are the same, but we are modeling after debug/dwarf
type QualifiedParameter struct {
	Name     string `json:"name,omitempty"`
	Class    string `json:"class,omitempty"`
	Type     string `json:"type,omitempty"`
	Location string `json:"location,omitempty"`
	Size     int64  `json:"size,omitempty"`
}

type BasicParameter struct {
	Name     string `json:"name,omitempty"`
	Type     string `json:"type,omitempty"`
	Class    string `json:"class,omitempty"`
	Location string `json:"location,omitempty"`
	Size     int64  `json:"size,omitempty"`
}

// A Variable description is general and can also describe an underlying type
// TODO should there be location here?
type VariableDescription struct {
	Name      string `json:"name,omitempty"`
	Class     string `json:"class,omitempty"`
	Type      string `json:"type,omitempty"`
	Size      int64  `json:"size"`
	Direction string `json:"direction,omitempty"`
}
