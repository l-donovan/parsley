package common

import (
	"errors"
	"fmt"
	"strings"
)

// Expression

type ExpressionDefinition struct {
	Name      string
	Evaluate  func(values map[string]any, input MetaString, globals map[string]any) (EvaluateResult, error)
	Serialize func(values map[string]any, config *SerializerConfig, indentLevel int) (string, error)
}

type Expression struct {
	Definition *ExpressionDefinition
	Values     map[string]any
}

func (e Expression) Evaluate(input MetaString, globals map[string]any) (EvaluateResult, error) {
	if e.Definition.Evaluate == nil {
		return ErrorResult, fmt.Errorf("no Evaluate method defined for expression of type %s", e.Definition.Name)
	}

	out, err := e.Definition.Evaluate(e.Values, input, globals)

	if err != nil {
		return ErrorResult, err
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

// EvaluateResult

type EvaluateResult interface {
	Condense() (TreeItem, error)
	String() string
	Remaining() MetaString
}

func Match(result EvaluateResult) bool {
	_, noMatch := result.(NoMatchResult)
	return !noMatch
}

func Discard(result EvaluateResult) bool {
	_, discard := result.(DiscardResult)
	return discard
}

// NoMatchResult

type NoMatchResult struct {
	remaining MetaString
}

func NewNoMatchResult(remaining MetaString) NoMatchResult {
	return NoMatchResult{remaining}
}

func (r NoMatchResult) String() string {
	return "NoMatch"
}

func (r NoMatchResult) Condense() (TreeItem, error) {
	return TreeItem{}, errors.New("can't condense NoMatchResult")
}

func (r NoMatchResult) Remaining() MetaString {
	return r.remaining
}

var ErrorResult = NoMatchResult{}

// SingleResult

type SingleResult struct {
	result     EvaluateResult
	remaining  MetaString
	identifier string
}

func NewSingleResult(result EvaluateResult, remaining MetaString, identifier string) SingleResult {
	return SingleResult{result, remaining, identifier}
}

func (r SingleResult) String() string {
	return fmt.Sprintf("%s<%s>", r.identifier, r.result)
}

func (r SingleResult) Condense() (TreeItem, error) {
	val, err := r.result.Condense()

	if err != nil {
		return TreeItem{}, err
	}

	return TreeItem{r.identifier, val}, nil
}

func (r SingleResult) Remaining() MetaString {
	return r.remaining
}

// MultipleResult

type MultipleResult struct {
	results      []EvaluateResult
	remaining    MetaString
	nextInSeries *MetaString
}

func NewMultipleResult(results []EvaluateResult, remaining MetaString, nextInSeries *MetaString) MultipleResult {
	return MultipleResult{results, remaining, nextInSeries}
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

func (r MultipleResult) Remaining() MetaString {
	return r.remaining
}

func (r MultipleResult) Next() *MetaString {
	return r.nextInSeries
}

// StringResult

type StringResult struct {
	val       MetaString
	remaining MetaString
}

func NewStringResult(val, remaining MetaString) StringResult {
	return StringResult{val, remaining}
}

func (r StringResult) String() string {
	return fmt.Sprintf("String<%s>", r.val.Val())
}

func (r StringResult) Condense() (TreeItem, error) {
	return TreeItem{"String", r.val}, nil
}

func (r StringResult) Remaining() MetaString {
	return r.remaining
}

// DiscardResult

type DiscardResult struct {
	remaining MetaString
}

func NewDiscardResult(remaining MetaString) DiscardResult {
	return DiscardResult{remaining}
}

func (r DiscardResult) String() string {
	return "Discard"
}

func (r DiscardResult) Condense() (TreeItem, error) {
	return TreeItem{}, errors.New("can't condense DiscardResult")
}

func (r DiscardResult) Remaining() MetaString {
	return r.remaining
}

// TreeItem

type TreeItem struct {
	Name string
	Val  any
}

func (t TreeItem) String() string {
	switch val := t.Val.(type) {
	case MetaString:
		return val.String()
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
