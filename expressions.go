package parsley

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/l-donovan/parsley/common"
)

var (
	ZeroOrMore = common.ExpressionDefinition{
		Name: "ZeroOrMore",
		Evaluate: func(values map[string]any, input any, globals map[string]any) (common.EvaluateResult, error) {
			expr := values["expr"].(common.Expression)
			inputStr := input.(string)
			var results []common.EvaluateResult

			for {
				result, err := expr.Evaluate(strings.TrimSpace(inputStr), globals)

				if err != nil {
					return common.NoMatch, err
				}

				if !common.Match(result) {
					// Zero matches are permissible, so this still counts as a match
					break
				}

				if !common.Discard(result) {
					results = append(results, result)
				}

				inputStr = result.Remaining()
			}

			return common.NewMultipleResult(results, inputStr), nil
		},
	}

	OneOrMore = common.ExpressionDefinition{
		Name: "OneOrMore",
		Evaluate: func(values map[string]any, input any, globals map[string]any) (common.EvaluateResult, error) {
			expr := values["expr"].(common.Expression)
			inputStr := input.(string)
			var results []common.EvaluateResult

			matchedAtLeastOnce := false

			for {
				result, err := expr.Evaluate(strings.TrimSpace(inputStr), globals)

				if err != nil {
					return common.NoMatch, err
				}

				if !common.Match(result) {
					break
				}

				matchedAtLeastOnce = true

				if !common.Discard(result) {
					results = append(results, result)
				}

				inputStr = result.Remaining()
			}

			if !matchedAtLeastOnce {
				return common.NoMatch, nil
			}

			return common.NewMultipleResult(results, inputStr), nil
		},
	}

	ZeroOrOne = common.ExpressionDefinition{
		Name: "ZeroOrOne",
		Evaluate: func(values map[string]any, input any, globals map[string]any) (common.EvaluateResult, error) {
			expr := values["expr"].(common.Expression)
			inputStr := input.(string)
			var results []common.EvaluateResult

			result, err := expr.Evaluate(strings.TrimSpace(inputStr), globals)

			if err != nil {
				return common.NoMatch, err
			}

			if !common.Match(result) {
				// Zero matches are permissible, so this still counts as a match
				return common.NewMultipleResult(results, inputStr), nil
			}

			if !common.Discard(result) {
				results = append(results, result)
			}

			return common.NewMultipleResult(results, result.Remaining()), nil
		},
	}

	Or = common.ExpressionDefinition{
		Name: "Or",
		Evaluate: func(values map[string]any, input any, globals map[string]any) (common.EvaluateResult, error) {
			lhs := values["lhs"].(common.Expression)
			rhs := values["rhs"].(common.Expression)
			inputStr := input.(string)
			var results []common.EvaluateResult

			lhsResult, err := lhs.Evaluate(strings.TrimSpace(inputStr), globals)

			if err != nil {
				return common.NoMatch, err
			}

			if common.Match(lhsResult) {
				inputStr = lhsResult.Remaining()
				results = append(results, lhsResult)
			}

			rhsResult, err := rhs.Evaluate(strings.TrimSpace(inputStr), globals)

			if err != nil {
				return common.NoMatch, err
			}

			if common.Match(rhsResult) {
				inputStr = rhsResult.Remaining()
				results = append(results, rhsResult)
			}

			if !common.Match(lhsResult) && !common.Match(rhsResult) {
				return common.NoMatch, nil
			}

			return common.NewMultipleResult(results, inputStr), nil
		},
	}

	ExclusiveOr = common.ExpressionDefinition{
		Name: "ExclusiveOr",
		Evaluate: func(values map[string]any, input any, globals map[string]any) (common.EvaluateResult, error) {
			lhs := values["lhs"].(common.Expression)
			rhs := values["rhs"].(common.Expression)
			inputStr := input.(string)

			lhsResult, err := lhs.Evaluate(strings.TrimSpace(inputStr), globals)

			if err != nil {
				return common.NoMatch, err
			}

			rhsResult, err := rhs.Evaluate(strings.TrimSpace(inputStr), globals)

			if err != nil {
				return common.NoMatch, err
			}

			if common.Match(lhsResult) == common.Match(rhsResult) {
				return common.NoMatch, nil
			} else if common.Match(lhsResult) {
				return lhsResult, nil
			} else {
				return rhsResult, nil
			}
		},
	}

	Union = common.ExpressionDefinition{
		Name: "Union",
		Evaluate: func(values map[string]any, input any, globals map[string]any) (common.EvaluateResult, error) {
			unionItems := values["unionItems"].([]common.Expression)
			inputStr := input.(string)

			for _, unionItem := range unionItems {
				result, err := unionItem.Evaluate(strings.TrimSpace(inputStr), globals)

				if err != nil {
					return common.NoMatch, err
				}

				if common.Match(result) {
					// Unions are transparent
					return result, nil
				}
			}

			return common.NoMatch, nil
		},
	}

	Rule = common.ExpressionDefinition{
		Name: "Rule",
		Evaluate: func(values map[string]any, input any, globals map[string]any) (common.EvaluateResult, error) {
			contents := values["contents"].([]common.Expression)
			groupExpr := common.Expression{Definition: &Group, Values: map[string]any{"groupItems": contents}}
			return groupExpr.Evaluate(input, globals)
		},
	}

	RuleRef = common.ExpressionDefinition{
		Name: "RuleRef",
		Evaluate: func(values map[string]any, input any, globals map[string]any) (common.EvaluateResult, error) {
			ref := values["ref"].(string)
			inputStr := input.(string)

			// Get the rule
			groupItems, found := globals[ref]

			if !found {
				return common.NoMatch, fmt.Errorf("could not find rule with name %s", ref)
			}

			groupExpr := common.Expression{Definition: &Group, Values: map[string]any{"groupItems": groupItems}}

			result, err := groupExpr.Evaluate(strings.TrimSpace(inputStr), globals)

			if err != nil {
				return common.NoMatch, err
			}

			if !common.Match(result) {
				return common.NoMatch, nil
			}

			return common.NewSingleResult(result, result.Remaining(), ref), nil
		},
	}

	RegularExpression = common.ExpressionDefinition{
		Name: "RegularExpression",
		Evaluate: func(values map[string]any, input any, globals map[string]any) (common.EvaluateResult, error) {
			val := values["val"].(string)
			inputStr := input.(string)

			expr, err := regexp.Compile(`\s*(` + val + ")")

			if err != nil {
				return common.NoMatch, err
			}

			idx := expr.FindStringSubmatchIndex(inputStr)

			if idx == nil || idx[0] > 0 {
				return common.NoMatch, nil
			}

			return common.NewStringResult(inputStr[idx[2]:idx[3]], inputStr[idx[3]:]), nil
		},
	}

	File = common.ExpressionDefinition{
		Name: "File",
		Evaluate: func(values map[string]any, input any, globals map[string]any) (common.EvaluateResult, error) {
			rules := values["rules"].([]common.Expression)
			inputStr := input.(string)

			for _, rule := range rules {
				ruleName := rule.Values["name"].(string)

				if ruleName != "input" {
					continue
				}

				result, err := rule.Evaluate(strings.TrimSpace(inputStr), globals)

				if err != nil {
					return common.NoMatch, err
				}

				return result, nil
			}

			return common.NoMatch, errors.New("no top-level rule found")
		},
	}

	Group = common.ExpressionDefinition{
		Name: "Group",
		Evaluate: func(values map[string]any, input any, globals map[string]any) (common.EvaluateResult, error) {
			groupItems := values["groupItems"].([]common.Expression)
			var results []common.EvaluateResult
			inputStr := input.(string)

			for _, groupItem := range groupItems {
				result, err := groupItem.Evaluate(strings.TrimSpace(inputStr), globals)

				if err != nil {
					return common.NoMatch, err
				}

				if !common.Match(result) {
					return common.NoMatch, nil
				}

				if !common.Discard(result) {
					results = append(results, result)
				}

				inputStr = result.Remaining()
			}

			return common.NewMultipleResult(results, inputStr), nil
		},
	}

	StringLiteral = common.ExpressionDefinition{
		Name: "StringLiteral",
		Evaluate: func(values map[string]any, input any, globals map[string]any) (common.EvaluateResult, error) {
			val := values["val"].(string)
			inputStr := input.(string)

			// The input starts with our string literal
			if strings.Index(inputStr, val) == 0 {
				return common.NewDiscardResult(inputStr[len(val):]), nil
			}

			return common.NoMatch, nil
		},
	}
)
