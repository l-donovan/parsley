package main

import (
	"fmt"
	"log"
	"os"

	"github.com/l-donovan/parsley"
)

func main() {
	parsleyParser := parsley.Parser{}
	flimGrammar, flimRules, err := parsleyParser.ParseGrammar("example/flim.parsley")

	if err != nil {
		log.Fatalln(err)
	}

	flimContents, err := os.ReadFile("example/demo.flim")

	if err != nil {
		log.Fatalln(err)
	}

	result, err := flimGrammar.Evaluate(string(flimContents), flimRules)

	if err != nil {
		log.Fatalln(err)
	}

	fmt.Printf("Output: %s\n", result)

	tree, err := result.Condense()

	if err != nil {
		log.Fatalln(err)
	}

	fmt.Printf("Condensed: %s\n", tree)
}
