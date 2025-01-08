package parsley

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/l-donovan/parsley/common"
)

type ParseError struct {
	Contents string
	Loc      common.StringPos
}

func (e ParseError) Error() string {
	return fmt.Sprintf("unknown token beginning at %s", e.Loc)
}

func digitCount(input int) int {
	if input == 0 {
		return 1
	}

	count := 0

	for input != 0 {
		input /= 10
		count++
	}

	return count
}

func (e ParseError) PrintContext(contextLineCount int) {
	lines := strings.Split(e.Contents, "\n")
	startLineNum := max(0, e.Loc.Line-contextLineCount)
	endLineNum := min(e.Loc.Line+contextLineCount+1, len(lines))
	maxLineNumWidth := digitCount(endLineNum + 1)

	fmt.Println("Context:")

	for i := startLineNum; i < endLineNum; i++ {
		if i == e.Loc.Line {
			// We're doing a "best effort" kinda thing here for characters with wide
			// printing widths, specifically tabs.
			tabCount := strings.Count(lines[i][:e.Loc.Col], "\t")
			left := strings.Repeat("\t", tabCount) + strings.Repeat(" ", e.Loc.Col-tabCount)

			// TODO: Make sure TERM supports color before printing a bunch of escape sequences
			fmt.Printf("%*d │ %s\x1b[30;47m%c\x1b[m%s\n", maxLineNumWidth, i+1, lines[i][:e.Loc.Col], lines[i][e.Loc.Col], lines[i][e.Loc.Col+1:])
			fmt.Printf("%*s │ %s╰─── [Starting here]\n", maxLineNumWidth, "", left)
		} else {
			fmt.Printf("%*d │ %s\n", maxLineNumWidth, i+1, lines[i])
		}
	}
}

type TokenDefinition struct {
	Name    string
	Pattern regexp.Regexp
}

type LexerToken struct {
	Name     string
	Contents string
}

type Parser struct {
	tokens []LexerToken
}

type Grammar struct {
	topLevelExpr common.Expression
	rules        map[string]any
}

func (g Grammar) Parse(contents string) (common.EvaluateResult, error) {
	contentsMeta := common.NewMetaString(contents)
	result, err := g.topLevelExpr.Evaluate(contentsMeta, g.rules)

	if err != nil {
		return nil, err
	}

	remaining := result.Remaining()

	if strings.TrimSpace(remaining.Val()) != "" {
		if multipleResult, ok := result.(common.MultipleResult); ok {
			if next := multipleResult.Next(); next != nil {
				return nil, ParseError{contents, next.Loc}
			}
		}

		return nil, ParseError{contents, remaining.Loc}
	}

	return result, nil
}

var tokenDefinitions map[string]*regexp.Regexp

func init() {
	tokenDefinitions = map[string]*regexp.Regexp{
		"Colon":             regexp.MustCompile(`^:`),
		"Pipe":              regexp.MustCompile(`^\|`),
		"Caret":             regexp.MustCompile(`^\^`),
		"QuestionMark":      regexp.MustCompile(`^\?`),
		"Newline":           regexp.MustCompile(`^[\n\r]+`),
		"LineComment":       regexp.MustCompile(`^#.+?[\n\r]+`),
		"RegularExpression": regexp.MustCompile(`^/(?:[^/\\]|\\.)*/`),
		"Keyword":           regexp.MustCompile(`^[\^!?.]?[\w_]+`),
		"String":            regexp.MustCompile(`^"(?:[^"\\]|\\.)*"`),
		"Star":              regexp.MustCompile(`^\*`),
		"Plus":              regexp.MustCompile(`^\+`),
		"LeftAngleBracket":  regexp.MustCompile(`^<`),
		"RightAngleBracket": regexp.MustCompile(`^>`),
		"LeftParenthesis":   regexp.MustCompile(`^\(`),
		"RightParenthesis":  regexp.MustCompile(`^\)`),
		"AtSign":            regexp.MustCompile(`^@`),
		"Whitespace":        regexp.MustCompile(`^[\t\f\v ]+`),
	}
}

func (p *Parser) Lex(text string) ([]LexerToken, error) {
	var tokens []LexerToken
	var found bool

	for len(text) > 0 {
		found = false

		for tokenName, tokenPattern := range tokenDefinitions {
			match := tokenPattern.FindStringSubmatchIndex(text)

			if match == nil {
				continue
			}

			found = true
			contents := text[match[0]:match[1]]
			token := LexerToken{Name: tokenName, Contents: contents}
			text = text[match[1]:]

			if tokenName != "Whitespace" && tokenName != "LineComment" {
				tokens = append(tokens, token)
			}
		}

		if !found {
			return tokens, fmt.Errorf("could not find token matching %s", text)
		}
	}

	return tokens, nil
}

func (p *Parser) popToken() LexerToken {
	token := p.tokens[0]
	p.tokens = p.tokens[1:]
	return token
}

func (p *Parser) peekToken() LexerToken {
	return p.tokens[0]
}

func (p *Parser) parseRuleExpression() (common.Expression, error) {
	name := p.popToken()

	if name.Name != "Keyword" {
		return common.Empty, fmt.Errorf("rule name cannot be %s", name.Name)
	}

	sep := p.popToken()

	if sep.Name != "Colon" {
		return common.Empty, fmt.Errorf("expected colon after rule name, found %s instead", sep.Name)
	}

	var contents []common.Expression

	for p.peekToken().Name != "Newline" {
		item, err := p.parseExpression()

		if err != nil {
			return common.Empty, fmt.Errorf("error when parsing contents for rule %s: %v", name.Contents, err)
		}

		contents = append(contents, item)
	}

	p.popToken()

	return common.Expression{Definition: &Rule, Values: map[string]any{"name": name.Contents, "contents": contents}}, nil
}

func (p *Parser) parseUnionExpression() (common.Expression, error) {
	var unionItems []common.Expression

	for p.peekToken().Name != "RightAngleBracket" {
		unionItem, err := p.parseExpression()

		if err != nil {
			return common.Empty, err
		}

		unionItems = append(unionItems, unionItem)
	}

	p.popToken()

	return common.Expression{Definition: &Union, Values: map[string]any{"unionItems": unionItems}}, nil
}

func (p *Parser) parseGroupExpression() (common.Expression, error) {
	var groupItems []common.Expression

	for p.peekToken().Name != "RightParenthesis" {
		groupItem, err := p.parseExpression()

		if err != nil {
			return common.Empty, err
		}

		groupItems = append(groupItems, groupItem)
	}

	p.popToken()

	return common.Expression{Definition: &Group, Values: map[string]any{"groupItems": groupItems}}, nil
}

func (p *Parser) parseExpression() (common.Expression, error) {
	var expr common.Expression
	var err error

	token := p.popToken()

	switch token.Name {
	case "LeftParenthesis":
		expr, err = p.parseGroupExpression()
	case "LeftAngleBracket":
		expr, err = p.parseUnionExpression()
	case "Keyword":
		expr = common.Expression{Definition: &RuleRef, Values: map[string]any{"ref": token.Contents}}
	case "String":
		val := token.Contents[1 : len(token.Contents)-1]
		expr = common.Expression{Definition: &StringLiteral, Values: map[string]any{"val": val}}
	case "RegularExpression":
		var val *regexp.Regexp
		val, err = regexp.Compile(`\s*(` + token.Contents[1:len(token.Contents)-1] + ")")
		expr = common.Expression{Definition: &RegularExpression, Values: map[string]any{"val": val}}
	default:
		return common.Empty, fmt.Errorf("unexpected token of type %s", token.Name)
	}

	if err != nil {
		return common.Empty, err
	}

	// Infix operators

	infixToken := p.peekToken()

	switch infixToken.Name {
	case "Pipe":
		p.popToken()
		rhs, err := p.parseExpression()

		if err != nil {
			return common.Empty, err
		}

		expr = common.Expression{Definition: &Or, Values: map[string]any{"lhs": expr, "rhs": rhs}}
	case "Caret":
		p.popToken()
		rhs, err := p.parseExpression()

		if err != nil {
			return common.Empty, err
		}

		expr = common.Expression{Definition: &ExclusiveOr, Values: map[string]any{"lhs": expr, "rhs": rhs}}
	}

	// Postfix operators

	postfixToken := p.peekToken()

	switch postfixToken.Name {
	case "Star":
		p.popToken()
		expr = common.Expression{Definition: &ZeroOrMore, Values: map[string]any{"expr": expr}}
	case "Plus":
		p.popToken()
		expr = common.Expression{Definition: &OneOrMore, Values: map[string]any{"expr": expr}}
	case "QuestionMark":
		p.popToken()
		expr = common.Expression{Definition: &ZeroOrOne, Values: map[string]any{"expr": expr}}
	}

	return expr, nil
}

func (p *Parser) parseFileExpression() (common.Expression, error) {
	var rules []common.Expression

	for len(p.tokens) > 0 {
		rule, err := p.parseRuleExpression()

		if err != nil {
			return common.Empty, err
		}

		rules = append(rules, rule)
	}

	return common.Expression{Definition: &File, Values: map[string]any{"rules": rules}}, nil
}

func ParseGrammar(contents string) (*Grammar, error) {
	parser := Parser{}
	tokens, err := parser.Lex(contents)

	if err != nil {
		return nil, err
	}

	parser.tokens = tokens
	fileExpr, err := parser.parseFileExpression()

	if err != nil {
		return nil, err
	}

	rules := fileExpr.Values["rules"].([]common.Expression)
	globals := map[string]any{}

	for _, rule := range rules {
		globals[rule.Values["name"].(string)] = rule.Values["contents"].([]common.Expression)
	}

	return &Grammar{fileExpr, globals}, nil
}
