package main

import (
	"fmt"
	"os"
	"regexp"

	"github.com/l-donovan/parsley/common"
	"github.com/l-donovan/parsley/parser"
)

func main() {
	defs := []parser.TokenDefinition{
		{Name: "Colon", Pattern: *regexp.MustCompile(`^:`)},
		{Name: "QuestionMark", Pattern: *regexp.MustCompile(`^\?`)},
		{Name: "Newline", Pattern: *regexp.MustCompile(`^[\n\r]+`)},
		{Name: "LineComment", Pattern: *regexp.MustCompile(`^#.+?[\n\r]+`)},
		{Name: "RegularExpression", Pattern: *regexp.MustCompile(`^/(?:[^/\\]|\\.)*/`)},
		{Name: "Keyword", Pattern: *regexp.MustCompile(`^!?[\w_]+`)},
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

	p := parser.Parser{TokenDefinitions: defs, PreserveNewlines: true}

	contents, err := os.ReadFile("flim.parsley")

	if err != nil {
		panic(err)
	}

	expr, err := p.Parse(string(contents))

	if err != nil {
		panic(err)
	}

	rules := expr.Values["rules"].([]common.Expression)
	globals := map[string]any{}

	for _, rule := range rules {
		globals[rule.Values["name"].(string)] = rule.Values["contents"].([]common.Expression)
	}

	flimContents, err := os.ReadFile("easy.flim")

	if err != nil {
		panic(err)
	}

	out, err := expr.Evaluate(string(flimContents), globals)

	if err != nil {
		panic(err)
	}

	fmt.Printf("Output: %s\n", out)
}
