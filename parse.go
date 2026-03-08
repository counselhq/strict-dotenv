package strictdotenv

import (
	"fmt"
	"io"
	"os"
)

// -----------------------------------------------------------------------------
// Parsers
// -----------------------------------------------------------------------------

// parseDotEnv parses a dotenv file or named pipe into the provided store.
func parseDotEnv(path string, store EnvStore, cfg *ParseConfig) error {
	if path == "" {
		return fmt.Errorf("parse dotenv path cannot be empty")
	}
	if store == nil {
		return fmt.Errorf("parse dotenv store cannot be nil")
	}

	bytes, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	return parse(path, bytes, store, cfg)
}

// parseString parses dotenv contents from a string into the provided store.
func parseString(s, name string, store EnvStore, cfg *ParseConfig) error {
	if store == nil {
		return fmt.Errorf("parse string store cannot be nil")
	}

	if name == "" {
		name = "string"
	}

	return parse(name, []byte(s), store, cfg)
}

// parseReader parses dotenv contents from an io.Reader into the provided store.
func parseReader(r io.Reader, name string, store EnvStore, cfg *ParseConfig) error {
	if r == nil {
		return fmt.Errorf("parse reader cannot be nil")
	}
	if store == nil {
		return fmt.Errorf("parse store cannot be nil")
	}

	if name == "" {
		name = "io.Reader"
	}

	bytes, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	return parse(name, bytes, store, cfg)
}

// -----------------------------------------------------------------------------
// Parser States
// -----------------------------------------------------------------------------

// The parser uses an explicit state enum (unlike the lexer's function-pointer
// approach). Each state represents a position in the expected grammar:
//
//	LineStart → (Export) → Key → Assignment → Value → (Comment) → Newline → LineStart
//
// The state names follow the convention "parseState" + where we are in the
// grammar. For example, parseStateAfterAssignment means we just consumed the
// '=' and are now looking for a value.
type parseState int

const (
	parseStateLineStart parseState = iota
	parseStateAfterExport
	parseStateAfterKey
	parseStateAfterAssignment

	parseStateExpectSingleQuotedValue
	parseStateExpectSingleQuoteClose

	parseStateExpectDoubleQuotedValue
	parseStateExpectDoubleQuoteClose

	parseStateAfterQuotedValue
	parseStateAfterUnquotedValue
	parseStateAfterInlineComment
)

// -----------------------------------------------------------------------------
// Parser Logic
// -----------------------------------------------------------------------------

// parse is the core parsing engine. It creates a lexer from the supplied
// byte slice and then enters a loop that pulls tokens one at a time, using a
// switch-on-state/switch-on-token pattern to validate the grammar and extract
// key-value pairs.
func parse(name string, bytes []byte, store EnvStore, cfg *ParseConfig) error {
	l, err := lexFromBytes(name, bytes)
	if err != nil {
		return err
	}

	var key string
	var value string
	var currentOptions ParseOptions
	var sawWhitespaceAfterExport bool

	commit := func() {
		store.Set(key, value, currentOptions.Overwrite)
		key = ""
		value = ""
	}

	state := parseStateLineStart // BEGINNING STATE

	for {
		tok := l.nextToken()
		if tok.typ == tokenError {
			return parseErr(name, tok, "%s", tok.err)
		}

		switch state {

		case parseStateLineStart:
			switch tok.typ {
			case tokenBOM, tokenWhitespace, tokenNewline, tokenComment:
				continue
			case tokenExport:
				sawWhitespaceAfterExport = false
				state = parseStateAfterExport
			case tokenKey:
				keyBytes := bytes[tok.start:tok.end]
				if !isValidKey(keyBytes) {
					return parseErr(name, tok, "invalid key %q", keyBytes)
				}
				key = string(keyBytes)
				currentOptions = resolveParseOptions(cfg, key)
				state = parseStateAfterKey
			case tokenInvalidLiteral:
				return parseErr(name, tok, "unexpected characters %q", bytes[tok.start:tok.end])
			case tokenEOF:
				return nil
			default:
				return parseErr(name, tok, "expected key, comment, newline, or EOF, got %s", tok.typ)
			}

		case parseStateAfterExport:
			switch tok.typ {
			case tokenWhitespace:
				sawWhitespaceAfterExport = true
			case tokenKey:
				keyBytes := bytes[tok.start:tok.end]
				if !isValidKey(keyBytes) {
					return parseErr(name, tok, "invalid key %q", keyBytes)
				}
				key = string(keyBytes)
				currentOptions = resolveParseOptions(cfg, key)
				state = parseStateAfterKey
			case tokenAssignmentOperator:
				if !sawWhitespaceAfterExport {
					return parseErr(name, tok, "expected whitespace between export and key")
				}
				// If no explicit key follows "export" and assignment starts,
				// treat "export" as the key (e.g. "export = value").
				key = "export"
				currentOptions = resolveParseOptions(cfg, key)
				value = ""
				state = parseStateAfterAssignment
			default:
				return parseErr(name, tok, "expected key after export, got %s", tok.typ)
			}

		case parseStateAfterKey:
			switch tok.typ {
			case tokenWhitespace:
				continue
			case tokenAssignmentOperator:
				value = ""
				state = parseStateAfterAssignment
			default:
				return parseErr(name, tok, "expected assignment operator after key %q", key)
			}

		case parseStateAfterAssignment:
			switch tok.typ {
			case tokenWhitespace:
			case tokenUnquotedValue:
				value = string(bytes[tok.start:tok.end])
				state = parseStateAfterUnquotedValue
			case tokenLeftSingleQuote:
				state = parseStateExpectSingleQuotedValue
			case tokenLeftDoubleQuote:
				state = parseStateExpectDoubleQuotedValue
			case tokenComment:
				value = "" // Empty value + inline comment
				state = parseStateAfterInlineComment
			case tokenNewline:
				value = ""
				commit()
				state = parseStateLineStart
			case tokenEOF:
				value = ""
				commit()
				return nil
			default:
				return parseErr(name, tok, "unexpected token after assignment operator: %s", tok.typ)
			}

		case parseStateExpectSingleQuotedValue:
			if tok.typ != tokenSingleQuoteValue {
				return parseErr(name, tok, "expected single-quoted value")
			}
			value = string(bytes[tok.start:tok.end])
			state = parseStateExpectSingleQuoteClose

		case parseStateExpectSingleQuoteClose:
			if tok.typ != tokenRightSingleQuote {
				return parseErr(name, tok, "expected closing single quote")
			}
			state = parseStateAfterQuotedValue

		case parseStateExpectDoubleQuotedValue:
			if tok.typ != tokenDoubleQuoteValue {
				return parseErr(name, tok, "expected double-quoted value")
			}

			var err error
			value, err = processValue(bytes[tok.start:tok.end], currentOptions)
			if err != nil {
				return parseErr(name, tok, "%s", err)
			}
			state = parseStateExpectDoubleQuoteClose

		case parseStateExpectDoubleQuoteClose:
			if tok.typ != tokenRightDoubleQuote {
				return parseErr(name, tok, "expected closing double quote")
			}
			state = parseStateAfterQuotedValue

		case parseStateAfterQuotedValue:
			switch tok.typ {
			case tokenWhitespace:
				continue
			case tokenComment:
				state = parseStateAfterInlineComment
			case tokenNewline:
				commit()
				state = parseStateLineStart
			case tokenEOF:
				commit()
				return nil
			default:
				return parseErr(name, tok, "unexpected content after quoted value")
			}

		case parseStateAfterUnquotedValue:
			switch tok.typ {
			case tokenWhitespace:
				continue
			case tokenComment:
				state = parseStateAfterInlineComment
			case tokenNewline:
				commit()
				state = parseStateLineStart
			case tokenEOF:
				commit()
				return nil
			default:
				return parseErr(name, tok, "unexpected content after unquoted value")
			}

		case parseStateAfterInlineComment:
			switch tok.typ {
			case tokenNewline:
				commit()
				state = parseStateLineStart
			case tokenEOF:
				commit()
				return nil
			default:
				return parseErr(name, tok, "expected newline or EOF after comment")
			}
		}
	}
}

// -----------------------------------------------------------------------------
// Parser Helpers
// -----------------------------------------------------------------------------

// parseErr is a helper for constructing error messages that include the
// file name and line number from the token.
func parseErr(name string, tok token, format string, args ...any) error {
	msg := fmt.Sprintf(format, args...)
	return fmt.Errorf("%s:%d: %s", name, tok.line, msg)
}

// isValidKey checks whether a key string conforms to the dotenv key rules:
// (1) len > 0
// (2) only ASCII letters, digits, or underscores
// (3) cannot start with a digit
func isValidKey(key []byte) bool {
	if len(key) == 0 {
		return false
	}

	if !lettersAndUnderscore[key[0]] {
		return false
	}

	for _, b := range key {
		if !lettersDigitsAndUnderscore[b] {
			return false
		}
	}

	return true
}
