package descriptor

// Sizes are in bytes

type FunctionParameter struct {
	Name      string `json:"name,omitempty"`
	TypeName  string `json:"type,omitempty"`
	ClassName string `json:"class,omitempty"`
	Direction string `json:"direction,omitempty"`
	Location  string `json:"location,omitempty"`
	Size      int64  `json:"sizes,omitempty"`
}

type FunctionDescription struct {
	Parameters []FunctionParameter `json:"parameters,omitempty"`
	Name       string              `json:"name"`
}

type VariableDescription struct {
	Name      string `json:"name"`
	Type      string `json:"type"`
	Size      int64  `json:"sizes,omitempty"`
	Direction string `json:"direction,omitempty"`
}
