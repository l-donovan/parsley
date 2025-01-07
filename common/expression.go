package common

import (
	"errors"
	"fmt"
	"strings"
)

// Expression

type ExpressionDefinition struct {
	Name      string
	Evaluate  func(values map[string]any, input any, globals map[string]any) (EvaluateResult, error)
	Serialize func(values map[string]any, config *SerializerConfig, indentLevel int) (string, error)
}

type Expression struct {
	Definition *ExpressionDefinition
	Values     map[string]any
}

func (e Expression) Evaluate(input any, globals map[string]any) (EvaluateResult, error) {
	if e.Definition.Evaluate == nil {
		return NoMatch, fmt.Errorf("no Evaluate method defined for expression of type %s", e.Definition.Name)
	}

	out, err := e.Definition.Evaluate(e.Values, input, globals)

	if err != nil {
		return NoMatch, err
	}

	return out, nil
}

func (e Expression) String() string {
	var pairs []string

	for key, val := range e.Values {
		pairs = append(pairs, fmt.Sprintf("%s: %s", key, val))
	}

	return fmt.Sprintf("%s<%s>", e.Definition.Name, strings.Join(pairs, ", "))
}

func (e Expression) Serialize(config *SerializerConfig, indentLevel int) (string, error) {
	if e.Definition.Serialize == nil {
		return "", fmt.Errorf("no Serialize method defined for expression of type %s", e.Definition.Name)
	}

	return e.Definition.Serialize(e.Values, config, indentLevel)
}

var Empty = Expression{}

// Evaluate result

type EvaluateResult interface {
	Condense() (TreeItem, error)
	String() string
	Remaining() string
}

func Match(result EvaluateResult) bool {
	_, noMatch := result.(NoMatchResult)
	return !noMatch
}

func Discard(result EvaluateResult) bool {
	_, discard := result.(DiscardResult)
	return discard
}

// No match result

type NoMatchResult struct{}

func (r NoMatchResult) String() string {
	return "NoMatch"
}

func (r NoMatchResult) Condense() (TreeItem, error) {
	return TreeItem{}, errors.New("can't condense NoMatchResult")
}

func (r NoMatchResult) Remaining() string {
	return ""
}

var NoMatch = NoMatchResult{}

// Single result

type SingleResult struct {
	result     EvaluateResult
	remaining  string
	identifier string
}

func (r SingleResult) String() string {
	return fmt.Sprintf("%s<%s>", r.identifier, r.result)
}

func NewSingleResult(result EvaluateResult, remaining, identifier string) SingleResult {
	return SingleResult{result, remaining, identifier}
}

func (r SingleResult) Condense() (TreeItem, error) {
	val, err := r.result.Condense()

	if err != nil {
		return TreeItem{}, err
	}

	return TreeItem{r.identifier, val}, nil
}

func (r SingleResult) Remaining() string {
	return r.remaining
}

// Multiple result

type MultipleResult struct {
	results   []EvaluateResult
	remaining string
}

func NewMultipleResult(results []EvaluateResult, remaining string) MultipleResult {
	return MultipleResult{results, remaining}
}

func (r MultipleResult) String() string {
	resultStrs := make([]string, len(r.results))

	for i, result := range r.results {
		resultStrs[i] = result.String()
	}

	return fmt.Sprintf("Multiple<%s>", strings.Join(resultStrs, ", "))
}

func (r MultipleResult) Condense() (TreeItem, error) {
	subVals := make([]TreeItem, len(r.results))

	for i, subResult := range r.results {
		subTreeItem, err := subResult.Condense()

		if err != nil {
			return TreeItem{}, err
		}

		subVals[i] = subTreeItem
	}

	return TreeItem{"Multiple", subVals}, nil
}

func (r MultipleResult) Remaining() string {
	return r.remaining
}

// String result

type StringResult struct {
	val       string
	remaining string
}

func NewStringResult(val, remaining string) StringResult {
	return StringResult{val, remaining}
}

func (r StringResult) String() string {
	return fmt.Sprintf("String<%s>", r.val)
}

func (r StringResult) Condense() (TreeItem, error) {
	return TreeItem{"String", r.val}, nil
}

func (r StringResult) Remaining() string {
	return r.remaining
}

// Discard result

type DiscardResult struct {
	remaining string
}

func NewDiscardResult(remaining string) DiscardResult {
	return DiscardResult{remaining}
}

func (r DiscardResult) String() string {
	return "Discard"
}

func (r DiscardResult) Condense() (TreeItem, error) {
	return TreeItem{}, errors.New("can't condense DiscardResult")
}

func (r DiscardResult) Remaining() string {
	return r.remaining
}

// Tree item

type TreeItem struct {
	Name string
	Val  any
}

func (t TreeItem) String() string {
	switch val := t.Val.(type) {
	case string:
		return fmt.Sprintf("%#v", val)
	case []TreeItem:
		subVals := make([]string, len(val))

		for i, subVal := range val {
			subVals[i] = subVal.String()
		}

		return fmt.Sprintf("[%s]", strings.Join(subVals, ", "))
	default:
		return fmt.Sprintf("%s%s", t.Name, t.Val)
	}
}
