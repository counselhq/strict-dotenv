package strictdotenv

import (
	"reflect"
	"testing"
)

type collectedToken struct {
	typ  tokenType
	text string
	line int
}

func collectTokens(t *testing.T, input string) []collectedToken {
	t.Helper()

	l, err := lexFromBytes("test.env", []byte(input))
	if err != nil {
		t.Fatalf("lexFromBytes: %v", err)
	}

	var tokens []collectedToken
	for {
		tok := l.nextToken()
		tokens = append(tokens, collectedToken{
			typ:  tok.typ,
			text: input[tok.start:tok.end],
			line: tok.line,
		})
		if tok.typ == tokenEOF || tok.typ == tokenError {
			return tokens
		}
	}
}

func TestLexerGroupsWhitespaceTokens(t *testing.T) {
	input := " \t \tKEY=\t  value\t \t# comment"

	want := []collectedToken{
		{typ: tokenWhitespace, text: " \t \t", line: 1},
		{typ: tokenKey, text: "KEY", line: 1},
		{typ: tokenAssignmentOperator, text: "=", line: 1},
		{typ: tokenWhitespace, text: "\t  ", line: 1},
		{typ: tokenUnquotedValue, text: "value", line: 1},
		{typ: tokenWhitespace, text: "\t \t", line: 1},
		{typ: tokenComment, text: "# comment", line: 1},
		{typ: tokenEOF, text: "", line: 1},
	}

	if got := collectTokens(t, input); !reflect.DeepEqual(got, want) {
		t.Fatalf("tokens mismatch\n got: %#v\nwant: %#v", got, want)
	}
}

func TestLexerGroupsNewlineTokens(t *testing.T) {
	input := "KEY=value\n\r\n\r\tNEXT=ok"

	want := []collectedToken{
		{typ: tokenKey, text: "KEY", line: 1},
		{typ: tokenAssignmentOperator, text: "=", line: 1},
		{typ: tokenUnquotedValue, text: "value", line: 1},
		{typ: tokenNewline, text: "\n\r\n\r", line: 1},
		{typ: tokenWhitespace, text: "\t", line: 4},
		{typ: tokenKey, text: "NEXT", line: 4},
		{typ: tokenAssignmentOperator, text: "=", line: 4},
		{typ: tokenUnquotedValue, text: "ok", line: 4},
		{typ: tokenEOF, text: "", line: 4},
	}

	if got := collectTokens(t, input); !reflect.DeepEqual(got, want) {
		t.Fatalf("tokens mismatch\n got: %#v\nwant: %#v", got, want)
	}
}

func TestLexerGroupsNewlinesAfterComments(t *testing.T) {
	input := "# comment\n\r\n\rKEY=value"

	want := []collectedToken{
		{typ: tokenComment, text: "# comment", line: 1},
		{typ: tokenNewline, text: "\n\r\n\r", line: 1},
		{typ: tokenKey, text: "KEY", line: 4},
		{typ: tokenAssignmentOperator, text: "=", line: 4},
		{typ: tokenUnquotedValue, text: "value", line: 4},
		{typ: tokenEOF, text: "", line: 4},
	}

	if got := collectTokens(t, input); !reflect.DeepEqual(got, want) {
		t.Fatalf("tokens mismatch\n got: %#v\nwant: %#v", got, want)
	}
}
