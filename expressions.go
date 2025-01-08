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
		Evaluate: func(values map[string]any, input common.MetaString, globals map[string]any) (common.EvaluateResult, error) {
			expr := values["expr"].(common.Expression)

			var results []common.EvaluateResult
			var deepestRemaining common.MetaString

			for {
				result, err := expr.Evaluate(input, globals)

				if err != nil {
					return common.ErrorResult, err
				}

				// Zero matches are permissible, so this still counts as a match
				if noMatch, didNotMatch := result.(common.NoMatchResult); didNotMatch {
					deepestRemaining = noMatch.Remaining()
					break
				}

				if !common.Discard(result) {
					results = append(results, result)
				}

				input = result.Remaining()
			}

			return common.NewMultipleResult(results, input, &deepestRemaining), nil
		},
	}

	OneOrMore = common.ExpressionDefinition{
		Name: "OneOrMore",
		Evaluate: func(values map[string]any, input common.MetaString, globals map[string]any) (common.EvaluateResult, error) {
			expr := values["expr"].(common.Expression)
			matchedAtLeastOnce := false

			var results []common.EvaluateResult
			var deepestRemaining common.MetaString

			for {
				result, err := expr.Evaluate(input, globals)

				if err != nil {
					return common.ErrorResult, err
				}

				if noMatch, didNotMatch := result.(common.NoMatchResult); didNotMatch {
					deepestRemaining = noMatch.Remaining()
					break
				}

				matchedAtLeastOnce = true

				if !common.Discard(result) {
					results = append(results, result)
				}

				input = result.Remaining()
			}

			if !matchedAtLeastOnce {
				return common.NewNoMatchResult(deepestRemaining), nil
			}

			return common.NewMultipleResult(results, input, &deepestRemaining), nil
		},
	}

	ZeroOrOne = common.ExpressionDefinition{
		Name: "ZeroOrOne",
		Evaluate: func(values map[string]any, input common.MetaString, globals map[string]any) (common.EvaluateResult, error) {
			expr := values["expr"].(common.Expression)

			var results []common.EvaluateResult
			var deepestNextInSeries *common.MetaString

			result, err := expr.Evaluate(input, globals)

			if err != nil {
				return common.ErrorResult, err
			}

			if !common.Match(result) {
				// Zero matches are permissible, so this still counts as a match
				remaining := result.Remaining()
				return common.NewMultipleResult(results, input, &remaining), nil
			}

			if !common.Discard(result) {
				results = append(results, result)
			}

			if multipleResult, ok := result.(common.MultipleResult); ok {
				deepestNextInSeries = multipleResult.Next()
			}

			return common.NewMultipleResult(results, result.Remaining(), deepestNextInSeries), nil
		},
	}

	Or = common.ExpressionDefinition{
		Name: "Or",
		Evaluate: func(values map[string]any, input common.MetaString, globals map[string]any) (common.EvaluateResult, error) {
			lhs := values["lhs"].(common.Expression)
			rhs := values["rhs"].(common.Expression)

			var results []common.EvaluateResult

			lhsResult, err := lhs.Evaluate(input, globals)

			if err != nil {
				return common.ErrorResult, err
			}

			if common.Match(lhsResult) {
				input = lhsResult.Remaining()
				results = append(results, lhsResult)
			}

			rhsResult, err := rhs.Evaluate(input, globals)

			if err != nil {
				return common.ErrorResult, err
			}

			if common.Match(rhsResult) {
				input = rhsResult.Remaining()
				results = append(results, rhsResult)
			}

			if !common.Match(lhsResult) && !common.Match(rhsResult) {
				if lhsResult.Remaining().Loc.Pos > rhsResult.Remaining().Loc.Pos {
					return common.NewNoMatchResult(lhsResult.Remaining()), nil
				} else {
					return common.NewNoMatchResult(rhsResult.Remaining()), nil
				}
			}

			var deepestNextInSeries *common.MetaString

			for _, result := range results {
				if multipleResult, ok := result.(common.MultipleResult); ok {
					if multipleResult.Next() != nil && (deepestNextInSeries == nil || multipleResult.Next().Loc.Pos > deepestNextInSeries.Loc.Pos) {
						deepestNextInSeries = multipleResult.Next()
					}
				}
			}

			return common.NewMultipleResult(results, input, deepestNextInSeries), nil
		},
	}

	ExclusiveOr = common.ExpressionDefinition{
		Name: "ExclusiveOr",
		Evaluate: func(values map[string]any, input common.MetaString, globals map[string]any) (common.EvaluateResult, error) {
			lhs := values["lhs"].(common.Expression)
			rhs := values["rhs"].(common.Expression)

			lhsResult, err := lhs.Evaluate(input, globals)

			if err != nil {
				return common.ErrorResult, err
			}

			rhsResult, err := rhs.Evaluate(input, globals)

			if err != nil {
				return common.ErrorResult, err
			}

			if common.Match(lhsResult) == common.Match(rhsResult) {
				if lhsResult.Remaining().Loc.Pos > rhsResult.Remaining().Loc.Pos {
					return common.NewNoMatchResult(lhsResult.Remaining()), nil
				} else {
					return common.NewNoMatchResult(rhsResult.Remaining()), nil
				}
			} else if common.Match(lhsResult) {
				return lhsResult, nil
			} else {
				return rhsResult, nil
			}
		},
	}

	Union = common.ExpressionDefinition{
		Name: "Union",
		Evaluate: func(values map[string]any, input common.MetaString, globals map[string]any) (common.EvaluateResult, error) {
			unionItems := values["unionItems"].([]common.Expression)

			var deepestRemaining common.MetaString

			for _, unionItem := range unionItems {
				result, err := unionItem.Evaluate(input, globals)

				if err != nil {
					return common.ErrorResult, err
				}

				if noMatch, didNotMatch := result.(common.NoMatchResult); didNotMatch {
					if noMatch.Remaining().Loc.Pos > deepestRemaining.Loc.Pos {
						deepestRemaining = noMatch.Remaining()
					}
				} else {
					// Unions are transparent
					return result, nil
				}
			}

			return common.NewNoMatchResult(deepestRemaining), nil
		},
	}

	Rule = common.ExpressionDefinition{
		Name: "Rule",
		Evaluate: func(values map[string]any, input common.MetaString, globals map[string]any) (common.EvaluateResult, error) {
			contents := values["contents"].([]common.Expression)
			groupExpr := common.Expression{Definition: &Group, Values: map[string]any{"groupItems": contents}}

			return groupExpr.Evaluate(input, globals)
		},
	}

	RuleRef = common.ExpressionDefinition{
		Name: "RuleRef",
		Evaluate: func(values map[string]any, input common.MetaString, globals map[string]any) (common.EvaluateResult, error) {
			ref := values["ref"].(string)

			groupItems, found := globals[ref]

			if !found {
				return common.ErrorResult, fmt.Errorf("could not find rule with name %s", ref)
			}

			groupExpr := common.Expression{Definition: &Group, Values: map[string]any{"groupItems": groupItems}}
			result, err := groupExpr.Evaluate(input, globals)

			if err != nil {
				return common.ErrorResult, err
			}

			if !common.Match(result) {
				return result, nil
			}

			return common.NewSingleResult(result, result.Remaining(), ref), nil
		},
	}

	RegularExpression = common.ExpressionDefinition{
		Name: "RegularExpression",
		Evaluate: func(values map[string]any, input common.MetaString, globals map[string]any) (common.EvaluateResult, error) {
			expr := values["val"].(*regexp.Regexp)
			idx := expr.FindStringSubmatchIndex(input.Val())

			if idx == nil || idx[0] > 0 {
				return common.NewNoMatchResult(input), nil
			}

			result := input.FromPosRange(idx[2], idx[3])
			remaining := input.FromStartPos(idx[3])

			return common.NewStringResult(result, remaining), nil
		},
	}

	File = common.ExpressionDefinition{
		Name: "File",
		Evaluate: func(values map[string]any, input common.MetaString, globals map[string]any) (common.EvaluateResult, error) {
			rules := values["rules"].([]common.Expression)

			for _, rule := range rules {
				ruleName := rule.Values["name"].(string)

				if ruleName != "input" {
					continue
				}

				if result, err := rule.Evaluate(input, globals); err != nil {
					return common.ErrorResult, err
				} else {
					return result, nil
				}
			}

			return common.ErrorResult, errors.New("no top-level rule found")
		},
	}

	Group = common.ExpressionDefinition{
		Name: "Group",
		Evaluate: func(values map[string]any, input common.MetaString, globals map[string]any) (common.EvaluateResult, error) {
			groupItems := values["groupItems"].([]common.Expression)

			var results []common.EvaluateResult
			var deepestNextInSeries *common.MetaString

			for _, groupItem := range groupItems {
				result, err := groupItem.Evaluate(input, globals)

				if err != nil {
					return common.ErrorResult, err
				}

				if !common.Match(result) {
					if deepestNextInSeries == nil || result.Remaining().Loc.Pos > deepestNextInSeries.Loc.Pos {
						return result, nil
					}

					return common.NewNoMatchResult(*deepestNextInSeries), nil
				}

				if multipleResult, ok := result.(common.MultipleResult); ok {
					if multipleResult.Next() != nil && (deepestNextInSeries == nil || multipleResult.Next().Loc.Pos > deepestNextInSeries.Loc.Pos) {
						deepestNextInSeries = multipleResult.Next()
					}
				}

				if !common.Discard(result) {
					results = append(results, result)
				}

				input = result.Remaining()
			}

			return common.NewMultipleResult(results, input, deepestNextInSeries), nil
		},
	}

	StringLiteral = common.ExpressionDefinition{
		Name: "StringLiteral",
		Evaluate: func(values map[string]any, input common.MetaString, globals map[string]any) (common.EvaluateResult, error) {
			val := values["val"].(string)
			trimmedInput := input.FromFirstNotMatching(" \t\f\v\r\n")

			// The input starts with our string literal
			if strings.Index(trimmedInput.Val(), val) == 0 {
				return common.NewDiscardResult(trimmedInput.FromStartPos(len(val))), nil
			}

			return common.NewNoMatchResult(trimmedInput), nil
		},
	}
)
