package parser

type Grammar []Rule

type Rule struct {
	Name       string
	Components []Component
}

type Component any
