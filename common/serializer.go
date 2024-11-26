package common

import "strings"

type SerializerConfig struct {
	useTabs    bool
	indentSize int
	minify     bool
}

func (c SerializerConfig) Indent(indentLevel int) string {
	if c.minify {
		return ""
	}

	if c.useTabs {
		return strings.Repeat("\t", c.indentSize*indentLevel)
	}

	return strings.Repeat(" ", c.indentSize*indentLevel)
}

func (c SerializerConfig) Sep(separator string, alt string) string {
	if c.minify {
		return alt
	}

	return separator
}

func Serialize(expr Expression, useTabs bool, indentSize int) (string, error) {
	config := SerializerConfig{useTabs: useTabs, indentSize: indentSize}
	return expr.Serialize(&config, 0)
}

func Minify(expr Expression) (string, error) {
	config := SerializerConfig{minify: true}
	return expr.Serialize(&config, 0)
}
