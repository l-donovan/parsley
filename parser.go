package parsley

import (
	"fmt"
	"os"
	"regexp"

	"github.com/l-donovan/parsley/common"
)

type TokenDefinition struct {
	Name    string
	Pattern regexp.Regexp
}

type LexerToken struct {
	Name     string
	Contents string
}

func (t LexerToken) IsOfType(names ...string) bool {
	for _, name := range names {
		if t.Name == name {
			return true
		}
	}

	return false
}

type Parser struct {
	tokens []LexerToken
}

var tokenDefinitions []TokenDefinition

func init() {
	tokenDefinitions = []TokenDefinition{
		{Name: "Colon", Pattern: *regexp.MustCompile(`^:`)},
		{Name: "Pipe", Pattern: *regexp.MustCompile(`^\|`)},
		{Name: "QuestionMark", Pattern: *regexp.MustCompile(`^\?`)},
		{Name: "Newline", Pattern: *regexp.MustCompile(`^[\n\r]+`)},
		{Name: "LineComment", Pattern: *regexp.MustCompile(`^#.+?[\n\r]+`)},
		{Name: "RegularExpression", Pattern: *regexp.MustCompile(`^/(?:[^/\\]|\\.)*/`)},
		{Name: "Keyword", Pattern: *regexp.MustCompile(`^[\^!?.]?[\w_]+`)},
		{Name: "String", Pattern: *regexp.MustCompile(`^"(?:[^"\\]|\\.)*"`)},
		{Name: "Star", Pattern: *regexp.MustCompile(`^\*`)},
		{Name: "Plus", Pattern: *regexp.MustCompile(`^\+`)},
		{Name: "LeftAngleBracket", Pattern: *regexp.MustCompile(`^<`)},
		{Name: "RightAngleBracket", Pattern: *regexp.MustCompile(`^>`)},
		{Name: "LeftParenthesis", Pattern: *regexp.MustCompile(`^\(`)},
		{Name: "RightParenthesis", Pattern: *regexp.MustCompile(`^\)`)},
		{Name: "AtSign", Pattern: *regexp.MustCompile(`^@`)},
		{Name: "Whitespace", Pattern: *regexp.MustCompile(`^[\t\f\v ]+`)},
	}
}

func (p *Parser) Lex(text string) ([]LexerToken, error) {
	var tokens []LexerToken
	var found bool

	for len(text) > 0 {
		found = false

		for _, tokenDefinition := range tokenDefinitions {
			match := tokenDefinition.Pattern.FindStringSubmatchIndex(text)

			if match == nil {
				continue
			}

			found = true
			contents := text[match[0]:match[1]]
			token := LexerToken{Name: tokenDefinition.Name, Contents: contents}
			text = text[match[1]:]

			if tokenDefinition.Name != "Whitespace" && tokenDefinition.Name != "LineComment" {
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

	if !name.IsOfType("Keyword") {
		return common.Empty, fmt.Errorf("rule name cannot be %s", name.Name)
	}

	sep := p.popToken()

	if !sep.IsOfType("Colon") {
		return common.Empty, fmt.Errorf("expected colon after rule name, found %s instead", sep.Name)
	}

	var contents []common.Expression

	for !p.peekToken().IsOfType("Newline") {
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

	for !p.peekToken().IsOfType("RightAngleBracket") {
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

	for !p.peekToken().IsOfType("RightParenthesis") {
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

	token := p.popToken()

	if token.IsOfType("String") {
		val := token.Contents[1 : len(token.Contents)-1]
		expr = common.Expression{Definition: &StringLiteral, Values: map[string]any{"val": val}}
	} else if token.IsOfType("LeftAngleBracket") {
		unionExpr, err := p.parseUnionExpression()

		if err != nil {
			return common.Empty, err
		}

		expr = unionExpr
	} else if token.IsOfType("Keyword") {
		if p.peekToken().IsOfType("Pipe") {
			p.popToken()

			lhs := common.Expression{Definition: &RuleRef, Values: map[string]any{"ref": token.Contents}}
			rhs, err := p.parseExpression()

			if err != nil {
				return common.Empty, err
			}

			expr = common.Expression{Definition: &Or, Values: map[string]any{"lhs": lhs, "rhs": rhs}}
		} else {
			expr = common.Expression{Definition: &RuleRef, Values: map[string]any{"ref": token.Contents}}
		}
	} else if token.IsOfType("LeftParenthesis") {
		groupExpr, err := p.parseGroupExpression()

		if err != nil {
			return common.Empty, err
		}

		expr = groupExpr
	} else if token.IsOfType("RegularExpression") {
		val := token.Contents[1 : len(token.Contents)-1]
		expr = common.Expression{Definition: &RegularExpression, Values: map[string]any{"val": val}}
	} else {
		return common.Empty, fmt.Errorf("unexpected token of type %s", token.Name)
	}

	if p.peekToken().IsOfType("Star") {
		p.popToken()
		expr = common.Expression{Definition: &ZeroOrMore, Values: map[string]any{"expr": expr}}
	} else if p.peekToken().IsOfType("Plus") {
		p.popToken()
		expr = common.Expression{Definition: &OneOrMore, Values: map[string]any{"expr": expr}}
	} else if p.peekToken().IsOfType("QuestionMark") {
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

func (p *Parser) ParseGrammar(path string) (common.Expression, map[string]any, error) {
	grammarContents, err := os.ReadFile(path)

	if err != nil {
		return common.Empty, nil, err
	}

	tokens, err := p.Lex(string(grammarContents))

	if err != nil {
		return common.Empty, nil, err
	}

	p.tokens = tokens

	fileExpr, err := p.parseFileExpression()

	if err != nil {
		return common.Empty, nil, err
	}

	rules := fileExpr.Values["rules"].([]common.Expression)
	globals := map[string]any{}

	for _, rule := range rules {
		globals[rule.Values["name"].(string)] = rule.Values["contents"].([]common.Expression)
	}

	return fileExpr, globals, nil
}
