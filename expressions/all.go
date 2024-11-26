package expressions

// What do we expect?
//
// my_transformer 1
//
// ... will become ...
//
// Rule.file<
//   OneOrMore<[
//     RuleRef.item<
//       RuleRef.expression<[
//         RuleRef.transformer<
//           RuleRef.plain_transformer<[
//             NamedRuleRef.name.name<
//               RegEx<my_transformer>
//             >
//             NamedRuleRef.expression.expression<[
//               RuleRef.literal<
//                 RuleRef.integer<
//                   RegEx<1>
//                 >
//               >
//               ZeroOrOne<>
//             ]>
//           ]>
//         >
//         ZeroOrOne<>
//       ]>
//     >
//   ]>
// >

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/l-donovan/parsley/common"
)

type KeyValuePair struct {
	Key string
	Val any
}

var (
	ZeroOrMore = common.ExpressionDefinition{
		Name: "ZeroOrMore",
		Evaluate: func(values map[string]any, input any, globals map[string]any) (common.EvaluateResult, error) {
			expr := values["expr"].(common.Expression)
			inputStr := input.(string)
			out := []any{}

			for {
				result, err := expr.Evaluate(strings.TrimSpace(inputStr), globals)

				if err != nil {
					return common.EvaluateResult{}, err
				}

				if !result.Match {
					// Zero matches are permissable, so this still counts as a match
					break
				}

				if !result.Discard {
					out = append(out, result.Val)
				}

				inputStr = result.Remaining
			}

			return common.EvaluateResult{Match: true, Val: out, Remaining: inputStr, Identifier: "ZeroOrMore"}, nil
		},
	}

	OneOrMore = common.ExpressionDefinition{
		Name: "OneOrMore",
		Evaluate: func(values map[string]any, input any, globals map[string]any) (common.EvaluateResult, error) {
			expr := values["expr"].(common.Expression)
			inputStr := input.(string)
			results := []common.EvaluateResult{}
			matchedAtLeastOnce := false

			for {
				result, err := expr.Evaluate(strings.TrimSpace(inputStr), globals)

				if err != nil {
					return common.EvaluateResult{}, err
				}

				if !result.Match {
					break
				}

				matchedAtLeastOnce = true

				if !result.Discard {
					results = append(results, result)
				}

				inputStr = result.Remaining
			}

			if !matchedAtLeastOnce {
				return common.EvaluateResult{Match: false}, nil
			}

			return common.EvaluateResult{Match: true, Val: results, Remaining: inputStr, Identifier: "OneOrMore"}, nil
		},
	}

	ZeroOrOne = common.ExpressionDefinition{
		Name: "ZeroOrOne",
		Evaluate: func(values map[string]any, input any, globals map[string]any) (common.EvaluateResult, error) {
			expr := values["expr"].(common.Expression)
			inputStr := input.(string)
			results := []common.EvaluateResult{}

			result, err := expr.Evaluate(strings.TrimSpace(inputStr), globals)

			if err != nil {
				return common.EvaluateResult{}, err
			}

			if !result.Match {
				// Zero matches are permissable, so this still counts as a match
				return common.EvaluateResult{Match: true, Val: results, Remaining: inputStr, Identifier: "ZeroOrOne"}, nil
			}

			if !result.Discard {
				results = append(results, result)
			}

			return common.EvaluateResult{Match: true, Val: results, Remaining: result.Remaining, Identifier: "ZeroOrOne"}, nil
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
					return common.EvaluateResult{}, err
				}

				if result.Match {
					// Unions are transparent
					return result, nil
				}
			}

			return common.EvaluateResult{Match: false}, nil
		},
	}

	Rule = common.ExpressionDefinition{
		Name: "Rule",
		Evaluate: func(values map[string]any, input any, globals map[string]any) (common.EvaluateResult, error) {
			name := values["name"].(string)
			contents := values["contents"].([]common.Expression)
			groupExpr := common.Expression{Definition: &Group, Values: map[string]any{"groupItems": contents}}
			result, err := groupExpr.Evaluate(input, globals)

			if err != nil {
				return common.EvaluateResult{}, err
			}

			if !result.Match {
				return common.EvaluateResult{Match: false}, nil
			}

			if result.Discard {
				return common.EvaluateResult{Match: true, Val: nil, Remaining: result.Remaining, Identifier: fmt.Sprintf("Rule.%s", name)}, nil
			}

			resultVals, ok := result.Val.([]common.EvaluateResult)

			if !ok {
				panic("this is garbage. fix.")
			}

			if len(resultVals) == 1 {
				return common.EvaluateResult{Match: true, Val: resultVals[0], Remaining: result.Remaining, Identifier: fmt.Sprintf("Rule.%s", name)}, nil
			} else {
				return common.EvaluateResult{Match: true, Val: resultVals, Remaining: result.Remaining, Identifier: fmt.Sprintf("Rule.%s", name)}, nil
			}
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
				return common.EvaluateResult{}, fmt.Errorf("could not find rule with name %s", ref)
			}

			groupExpr := common.Expression{Definition: &Group, Values: map[string]any{"groupItems": groupItems}}

			result, err := groupExpr.Evaluate(strings.TrimSpace(inputStr), globals)

			if err != nil {
				return common.EvaluateResult{}, err
			}

			if !result.Match {
				return common.EvaluateResult{Match: false}, nil
			}

			if result.Discard {
				return common.EvaluateResult{Match: true, Val: nil, Remaining: result.Remaining, Identifier: fmt.Sprintf("RuleRef.%s", ref)}, nil
			}

			// The problem is that result could be a Group or a single item

			resultVals, ok := result.Val.([]common.EvaluateResult)

			if !ok {
				panic("this is garbage. fix.")
			}

			if len(resultVals) == 1 {
				return common.EvaluateResult{Match: true, Val: resultVals[0], Remaining: result.Remaining, Identifier: fmt.Sprintf("RuleRef.%s", ref)}, nil
			} else {
				return common.EvaluateResult{Match: true, Val: resultVals, Remaining: result.Remaining, Identifier: fmt.Sprintf("RuleRef.%s", ref)}, nil
			}
		},
	}

	NamedRuleRef = common.ExpressionDefinition{
		Name: "NamedRuleRef",
		Evaluate: func(values map[string]any, input any, globals map[string]any) (common.EvaluateResult, error) {
			ref := values["ref"].(string)
			inputStr := input.(string)

			// Get the rule
			groupItems, found := globals[ref]

			if !found {
				return common.EvaluateResult{}, fmt.Errorf("could not find rule with name %s", ref)
			}

			groupExpr := common.Expression{Definition: &Group, Values: map[string]any{"groupItems": groupItems}}
			result, err := groupExpr.Evaluate(strings.TrimSpace(inputStr), globals)

			if err != nil {
				return common.EvaluateResult{}, err
			}

			if !result.Match {
				return common.EvaluateResult{Match: false}, nil
			}

			if result.Discard {
				return common.EvaluateResult{Match: true, Val: nil, Remaining: result.Remaining, Identifier: fmt.Sprintf("NamedRuleRef.%s", ref)}, nil
			}

			resultVals, ok := result.Val.([]common.EvaluateResult)

			if !ok {
				panic("this is garbage. fix.")
			}

			if len(resultVals) == 1 {
				return common.EvaluateResult{Match: true, Val: resultVals[0], Remaining: result.Remaining, Identifier: fmt.Sprintf("NamedRuleRef.%s", ref)}, nil
			} else {
				return common.EvaluateResult{Match: true, Val: resultVals, Remaining: result.Remaining, Identifier: fmt.Sprintf("NamedRuleRef.%s", ref)}, nil
			}
		},
	}

	RegularExpression = common.ExpressionDefinition{
		Name: "RegularExpression",
		Evaluate: func(values map[string]any, input any, globals map[string]any) (common.EvaluateResult, error) {
			val := values["val"].(string)
			inputStr := input.(string)

			expr, err := regexp.Compile(`\s*(` + val + ")")

			if err != nil {
				return common.EvaluateResult{}, err
			}

			idx := expr.FindStringSubmatchIndex(inputStr)

			if idx == nil || idx[0] > 0 {
				return common.EvaluateResult{Match: false}, nil
			}

			return common.EvaluateResult{Match: true, Val: inputStr[idx[2]:idx[3]], Remaining: inputStr[idx[3]:], Identifier: "RegEx"}, nil
		},
	}

	File = common.ExpressionDefinition{
		Name: "File",
		Evaluate: func(values map[string]any, input any, globals map[string]any) (common.EvaluateResult, error) {
			rules := values["rules"].([]common.Expression)
			inputStr := input.(string)

			for _, rule := range rules {
				ruleName := rule.Values["name"].(string)

				if ruleName[0] == '!' {
					continue
				}

				result, err := rule.Evaluate(strings.TrimSpace(inputStr), globals)

				if err != nil {
					return common.EvaluateResult{}, err
				}

				if result.Match {
					// We just pass the rule through
					return result, nil
				}
			}

			return common.EvaluateResult{Match: false}, nil
		},
	}

	Group = common.ExpressionDefinition{
		Name: "Group",
		Evaluate: func(values map[string]any, input any, globals map[string]any) (common.EvaluateResult, error) {
			groupItems := values["groupItems"].([]common.Expression)
			results := []common.EvaluateResult{}
			inputStr := input.(string)

			for _, groupItem := range groupItems {
				result, err := groupItem.Evaluate(strings.TrimSpace(inputStr), globals)

				if err != nil {
					return common.EvaluateResult{}, err
				}

				if !result.Match {
					return common.EvaluateResult{Match: false}, nil
				}

				if !result.Discard {
					results = append(results, result)
				}

				inputStr = result.Remaining
			}

			return common.EvaluateResult{Match: true, Val: results, Remaining: inputStr, Identifier: "Group"}, nil
		},
	}

	StringLiteral = common.ExpressionDefinition{
		Name: "StringLiteral",
		Evaluate: func(values map[string]any, input any, globals map[string]any) (common.EvaluateResult, error) {
			val := values["val"].(string)
			inputStr := input.(string)

			// The input starts with our string literal
			if strings.Index(inputStr, val) == 0 {
				return common.EvaluateResult{Match: true, Val: val, Remaining: inputStr[len(val):], Identifier: "String", Discard: true}, nil
			}

			return common.EvaluateResult{Match: false}, nil
		},
	}
)
