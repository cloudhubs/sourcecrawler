package cfg

type VariableWrapper interface {
	//This is the identifier
	SetName(name string)
	GetName() string

	//This is the value associated with the variable
	// which will either be another variable, function,
	// or value
	//TODO: define which types are valid for each concrete type
	SetValue(value interface{})
	GetValue() interface{}
}

func (f FnVariableWrapper) SetName(name string) {
	f.Name = name
}

func (f FnVariableWrapper) GetName() string {
	return f.Name
}

//TODO: the input should be a FnWrapper
// or a reference to another FnVariableWrapper
func (f FnVariableWrapper) SetValue(value interface{}) {
	f.Value = value
}

func (f FnVariableWrapper) GetValue() interface{} {
	return f.Value
}

// this concrete type represents
// variables that hold functions
// (literal and named)
type FnVariableWrapper struct {
	Value interface{} //should be VariableConnector or FnWrapper if assigned
	Name string
}


