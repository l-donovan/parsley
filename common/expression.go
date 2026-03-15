package common

import (
	"errors"
	"fmt"
	"strings"
)

// TreeItem

type TreeItem interface {
	fmt.Stringer
	Val() any
}

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
	return nil, errors.New("can't condense NoMatchResult")
}

func (r NoMatchResult) Remaining() MetaString {
	return r.remaining
}

var ErrorResult = NoMatchResult{}

// RuleResult is the core type for defined rules.
type RuleResult struct {
	result     MultipleResult
	remaining  MetaString
	identifier string
}

func NewRuleResult(result MultipleResult, remaining MetaString, identifier string) RuleResult {
	return RuleResult{result, remaining, identifier}
}

func (r RuleResult) String() string {
	return fmt.Sprintf("%s<%s>", r.identifier, r.result)
}

func (r RuleResult) Condense() (TreeItem, error) {
	val, err := r.result.Condense()

	if err != nil {
		return nil, err
	}

	return RuleTreeItem{r.identifier, val.(MultipleTreeItem)}, nil
}

func (r RuleResult) Remaining() MetaString {
	return r.remaining
}

type RuleTreeItem struct {
	rule   string
	result MultipleTreeItem
}

func (t RuleTreeItem) Val() any {
	// TODO: I dunno.
	return t
}

func (t RuleTreeItem) String() string {
	return fmt.Sprintf("%s<%s>", t.rule, t.result)
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
			return nil, err
		}

		subVals[i] = subTreeItem
	}

	return MultipleTreeItem(subVals), nil
}

func (r MultipleResult) Remaining() MetaString {
	return r.remaining
}

func (r MultipleResult) Next() *MetaString {
	return r.nextInSeries
}

type MultipleTreeItem []TreeItem

func (t MultipleTreeItem) Val() any {
	return t
}

func (t MultipleTreeItem) String() string {
	subVals := make([]string, len(t))

	for i, subVal := range t {
		subVals[i] = subVal.String()
	}

	return fmt.Sprintf("[%s]", strings.Join(subVals, ", "))
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
	return StringTreeItem{r.val.contents}, nil
}

func (r StringResult) Remaining() MetaString {
	return r.remaining
}

type StringTreeItem struct {
	val string
}

func (t StringTreeItem) Val() any {
	return t.val
}

func (t StringTreeItem) String() string {
	return fmt.Sprintf("String<%s>", t.val)
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
	return nil, errors.New("can't condense DiscardResult")
}

func (r DiscardResult) Remaining() MetaString {
	return r.remaining
}
