package main

import (
	_ "embed"
	"errors"
	"fmt"
	"log"

	"github.com/l-donovan/parsley"
)

//go:embed flim.parsley
var flimGrammarContents string

// Alternately, try bad.flim

//go:embed demo.flim
var flimContents string

func main() {
	flimGrammar, err := parsley.ParseGrammar(flimGrammarContents)

	if err != nil {
		log.Fatalln(err)
	}

	result, err := flimGrammar.Parse(flimContents)

	var parseErr parsley.ParseError

	if errors.As(err, &parseErr) {
		parseErr.PrintContext(4)
	}

	if err != nil {
		log.Fatalln(err)
	}

	tree, err := result.Condense()

	if err != nil {
		log.Fatalln(err)
	}

	fmt.Printf("Condensed: %s\n", tree)
}
