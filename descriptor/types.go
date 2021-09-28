package descriptor

// Sizes are in bytes

// A general interface to support different parameter types
type Parameter interface {
	GetSize() int64
}

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

// All types can return a size
func (f FunctionParameter) GetSize() int64  { return f.Size }
func (f StructureParameter) GetSize() int64 { return f.Size }
func (f PointerParameter) GetSize() int64   { return f.Size }
func (f ArrayParameter) GetSize() int64     { return f.Size }
func (f QualifiedParameter) GetSize() int64 { return f.Size }
func (f BasicParameter) GetSize() int64     { return f.Size }

type StructureParameter struct {
	Name   string      `json:"name,omitempty"`
	Type   string      `json:"type,omitempty"`
	Class  string      `json:"class,omitempty"`
	Size   int64       `json:"size,omitempty"`
	Fields []Parameter `json:"fields,omitempty"`
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
	ItemType Parameter `json:"items_type,omitemtpy"`
}

// QualifiedParameter and BasicParameter are the same, but we are modeling after debug/dwarf
type QualifiedParameter struct {
	Type string `json:"type,omitempty"`
	Size int64  `json:"size,omitempty"`
}

type BasicParameter struct {
	Name  string `json:"name,omitempty"`
	Type  string `json:"type,omitempty"`
	Class string `json:"class,omitempty"`
	Size  int64  `json:"size,omitempty"`
}

// A Variable description is general and can also describe an underlying type
type VariableDescription struct {
	Name      string `json:"name,omitempty"`
	Class     string `json:"class,omitempty"`
	Type      string `json:"type,omitempty"`
	Size      int64  `json:"size,omitempty"`
	Direction string `json:"direction,omitempty"`
}

/*
Json Serialization functions for each parameter

func (p *FunctionParameter) MarshalJSON() (b []byte, e error) {
    return json.Marshal(map[string]string{
        "type":  "plant",
        "color": p.Color(),
    })
}

func (a *Animal) Color() string {
    return a.MyColor
}
func (a *Animal) MarshalJSON() (b []byte, e error) {
    return json.Marshal(map[string]string{
        "type":  "animal",
        "color": a.Color(),
    })
}

func main() {
    // First let's create some things to live in the ecosystem
    fern := &Plant{MyColor: "green"}
    flower := &Plant{MyColor: "purple"}

    panther := &Animal{MyColor: "black"}
    lizard := &Animal{MyColor: "green"}

    // Then let's create a ColorfulEcosystem
    colorfulEcosystem := ColorfulEcosystem{
        Things: []ColoredThing{
            fern,
            flower,
            panther,
            lizard,
        },
    }

    // prints out:
    // {"things":[{"color":"green","type":"plant"},{"color":"purple","type":"plant"},{"color":"black","type":"animal"},{"color":"green","type":"animal"}]}
 //   byteSlice, _ := json.Marshal(colorfulEcosystem)
  //  fmt.Println(string(byteSlice))
}*/
