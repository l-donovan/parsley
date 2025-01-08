package common

import (
	"fmt"
	"strings"
)

type StringPos struct {
	Pos, Line, Col int
}

func (s StringPos) String() string {
	return fmt.Sprintf("%d:%d", s.Line+1, s.Col+1)
}

type MetaString struct {
	contents string
	Loc      StringPos
}

func NewMetaString(contents string) MetaString {
	return MetaString{contents, StringPos{0, 0, 0}}
}

func (m MetaString) getPos(start int) StringPos {
	if start == 0 {
		return m.Loc
	}

	newlineCount := strings.Count(m.contents[:start], "\n")
	pos := m.Loc.Pos + start
	line := m.Loc.Line + newlineCount
	col := start

	if newlineCount == 0 {
		col += m.Loc.Col
	} else {
		lastNewlinePos := strings.LastIndex(m.contents[:start], "\n")
		col -= lastNewlinePos + 1
	}

	return StringPos{pos, line, col}
}

func (m MetaString) FromStartPos(start int) MetaString {
	return MetaString{m.contents[start:], m.getPos(start)}
}

func (m MetaString) FromPosRange(start, stop int) MetaString {
	return MetaString{m.contents[start:stop], m.getPos(start)}
}

func (m MetaString) FromFirstMatching(targetset string) MetaString {
	for i, ch := range m.contents {
		if strings.ContainsRune(targetset, ch) {
			return MetaString{m.contents[i:], m.getPos(i)}
		}
	}

	return MetaString{}
}

func (m MetaString) FromFirstNotMatching(targetset string) MetaString {
	for i, ch := range m.contents {
		if !strings.ContainsRune(targetset, ch) {
			return MetaString{m.contents[i:], m.getPos(i)}
		}
	}

	return MetaString{}
}

func (m MetaString) Val() string {
	return m.contents
}

func (m MetaString) String() string {
	if strings.Contains(m.contents, "\"") {
		return fmt.Sprintf("'%s' %s", m.contents, m.Loc)
	} else {
		return fmt.Sprintf("%#v %s", m.contents, m.Loc)
	}
}
