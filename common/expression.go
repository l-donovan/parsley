package common

import (
	"fmt"
	"strings"
)

type ExpressionDefinition struct {
	Name      string
	Evaluate  func(values map[string]any, input any, globals map[string]any) (EvaluateResult, error)
	Serialize func(values map[string]any, config *SerializerConfig, indentLevel int) (string, error)
}

type Expression struct {
	Definition *ExpressionDefinition
	Values     map[string]any
}

type EvaluateResult struct {
	Match      bool
	Identifier string
	Val        any
	Remaining  string
	Discard    bool
}

func (r EvaluateResult) String() string {
	return fmt.Sprintf("%s<%s>", r.Identifier, r.Val)
}

func (e Expression) Evaluate(input any, globals map[string]any) (EvaluateResult, error) {
	out, err := e.Definition.Evaluate(e.Values, input, globals)

	if err != nil {
		return EvaluateResult{}, err
	}

	return out, nil
}

func (e Expression) String() string {
	pairs := []string{}

	for key, val := range e.Values {
		pairs = append(pairs, fmt.Sprintf("%s: %s", key, val))
	}

	return fmt.Sprintf("%s<%s>", e.Definition.Name, strings.Join(pairs, ", "))
}

func (e Expression) Serialize(config *SerializerConfig, indentLevel int) (string, error) {
	return e.Definition.Serialize(e.Values, config, indentLevel)
}

var Empty Expression = Expression{}
