package descriptor

// A Parameter interface must implement these functions

type Parameter interface {
	Name() string
	TypeName() string
	ClassName() string
	Direction() string
	Location() string
	SizeBytes() int
	ToJson()
}

type FunctionDescription struct {
	Parameters  []Parameter
	ReturnValue Parameter
	Name        string `json:"name"`
}

type VariableDescription struct {
	Name string	`json:"name"`
	Type string	`json:"type"`
}
