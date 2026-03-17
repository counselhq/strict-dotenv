package strictdotenv

import (
	"bytes"
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"
)

// ----------------------------------------------------------------------------
// Tokens
// ----------------------------------------------------------------------------

type Pos int

type token struct {
	typ   tokenType // the category of token (key, comment, etc.)
	start Pos       // byte offset where the token starts in lexer's bytes
	end   Pos       // byte offset where the token ends in lexer's bytes
	line  int       // line number for error reporting
	err   string    // error message for error reporting
}

// String provides a human-readable representation of a token for debugging.
func (t token) String() string {
	switch t.typ {
	case tokenEOF:
		return "EOF"
	case tokenError:
		return t.err
	default:
		return fmt.Sprintf("%s[%d:%d]", t.typ, t.start, t.end)
	}
}

// Every token the lexer produces is classified into one of these types.
// The parser uses these type tags to decide how to interpret each token.
type tokenType int

const (
	tokenError tokenType = iota
	tokenInvalidLiteral
	tokenBOM
	tokenEOF

	tokenNewline
	tokenWhitespace
	tokenAssignmentOperator
	tokenExport
	tokenComment

	tokenKey

	tokenUnquotedValue

	tokenLeftSingleQuote
	tokenSingleQuoteValue
	tokenRightSingleQuote

	tokenLeftDoubleQuote
	tokenDoubleQuoteValue
	tokenRightDoubleQuote
)

// String returns the name of the token type as a human-readable string.
func (t tokenType) String() string {
	switch t {
	case tokenError:
		return "tokenError"
	case tokenInvalidLiteral:
		return "tokenInvalidLiteral"
	case tokenBOM:
		return "tokenBOM"
	case tokenEOF:
		return "tokenEOF"

	case tokenNewline:
		return "tokenNewline"
	case tokenWhitespace:
		return "tokenWhitespace"
	case tokenAssignmentOperator:
		return "tokenAssignmentOperator"
	case tokenExport:
		return "tokenExport"
	case tokenComment:
		return "tokenComment"

	case tokenKey:
		return "tokenKey"

	case tokenUnquotedValue:
		return "tokenUnquotedValue"

	case tokenLeftSingleQuote:
		return "tokenLeftSingleQuote"
	case tokenSingleQuoteValue:
		return "tokenSingleQuoteValue"
	case tokenRightSingleQuote:
		return "tokenRightSingleQuote"

	case tokenLeftDoubleQuote:
		return "tokenLeftDoubleQuote"
	case tokenDoubleQuoteValue:
		return "tokenDoubleQuoteValue"
	case tokenRightDoubleQuote:
		return "tokenRightDoubleQuote"

	default:
		return "tokenUnknown"
	}
}

// eof is a sentinel rune value (-1) returned by next() when the lexer has
// consumed all bytes. Using -1 avoids collisions with any valid Unicode
// code point (which are all >= 0).
const eof = -1

// ----------------------------------------------------------------------------
// Lexer
// ----------------------------------------------------------------------------

// The lexer struct holds all mutable scanning state. It walks through the bytes slice rune by rune.
type lexer struct {
	name  string // name of the file | named pipe | reader source
	bytes []byte // raw bytes being scanned

	pos       Pos  // current byte offset in bytes; advances forward as runes are consumed
	lastWidth int  // byte width of the rune most recently returned by next(); used by backup()
	start     Pos  // byte offset where the current (not-yet-emitted) token begins
	atEOF     bool // true after next() returns eof, prevents double-backup past the end

	line      int // current line number (1-based), incremented on each line terminator
	startLine int // line number at the start of the current token, used in emitted tokens

	token    token // the most recently emitted token, read by nextToken()
	hasToken bool  // true when a token has been emitted and not yet consumed
	done     bool  // true after EOF | error

	state stateFn
}

// lexFromBytes creates a new lexer directly from a pre-loaded byte slice,
// and then initialises the state machine at lexStart.
func lexFromBytes(name string, bytes []byte) (*lexer, error) {
	if name == "" {
		return nil, fmt.Errorf("name must not be empty")
	}
	if !utf8.Valid(bytes) {
		return nil, fmt.Errorf("%s: invalid UTF-8 input", name)
	}

	l := &lexer{
		name:      name,
		bytes:     bytes,
		line:      1,
		startLine: 1,
		state:     lexStart,
	}
	return l, nil
}

// ----------------------------------------------------------------------------
// Lexer - Rune Navigation (next, peek, backup)
// ----------------------------------------------------------------------------

func (l *lexer) next() rune {
	if int(l.pos) >= len(l.bytes) {
		l.atEOF = true
		l.lastWidth = 0
		return eof
	}
	l.atEOF = false

	// ASCII fast path
	b := l.bytes[l.pos]
	if b < utf8.RuneSelf {
		if b == '\r' {
			// Consume CRLF as a single 2-byte unit.
			if int(l.pos+1) < len(l.bytes) && l.bytes[l.pos+1] == '\n' {
				l.lastWidth = 2
				l.pos += 2
			} else {
				l.lastWidth = 1
				l.pos++
			}
			l.line++
			return '\r'
		}
		l.lastWidth = 1
		l.pos++
		if b == '\n' {
			l.line++
		}
		return rune(b)
	}

	r, w := utf8.DecodeRune(l.bytes[l.pos:])
	l.lastWidth = w
	l.pos += Pos(w)
	return r
}

func (l *lexer) peek() rune {
	if int(l.pos) >= len(l.bytes) {
		return eof
	}
	// ASCII fast-path
	b := l.bytes[l.pos]
	if b < utf8.RuneSelf {
		return rune(b)
	}
	r, _ := utf8.DecodeRune(l.bytes[l.pos:])
	return r
}

func (l *lexer) backup() {
	if l.atEOF {
		l.atEOF = false
		return
	}
	if l.lastWidth == 0 {
		return
	}
	l.pos -= Pos(l.lastWidth)
	l.lastWidth = 0
	if b := l.bytes[l.pos]; b == '\n' || b == '\r' {
		l.line--
	}
}

// ----------------------------------------------------------------------------
// Token Emission
// ----------------------------------------------------------------------------

func (l *lexer) thisToken(t tokenType) token {
	return token{
		typ:   t,
		start: l.start,
		end:   l.pos,
		line:  l.startLine,
	}
}

func (l *lexer) emit(t tokenType) {
	l.emitToken(l.thisToken(t))
}

func (l *lexer) emitToken(t token) {
	l.token = t
	l.hasToken = true
	l.start = l.pos
	l.startLine = l.line
}

func (l *lexer) errorf(format string, args ...any) stateFn {
	l.emitToken(token{
		typ:   tokenError,
		start: l.start,
		end:   l.pos,
		line:  l.startLine,
		err:   fmt.Sprintf(format, args...),
	})
	l.done = true
	return nil
}

func (l *lexer) nextToken() token {
	if l.done {
		return token{typ: tokenEOF, start: l.pos, end: l.pos, line: l.line}
	}

	for {
		if l.state == nil {
			l.done = true
			return token{typ: tokenEOF, start: l.pos, end: l.pos, line: l.line}
		}

		l.hasToken = false
		l.state = l.state(l)
		if l.hasToken {
			if l.token.typ == tokenEOF || l.token.typ == tokenError {
				l.done = true
			}
			return l.token
		}
	}
}

// ----------------------------------------------------------------------------
// State Functions
// ----------------------------------------------------------------------------

// stateFn is the heart of the lexer's state-machine design. Each state is
// represented as a function that:
//  1. Examines the current input character(s)
//  2. Emits zero or one tokens
//  3. Returns the NEXT state function to run (or nil to stop)
type stateFn func(*lexer) stateFn

// lexStart is the initial state. It runs exactly once at the beginning of
// input. Its only job is to detect (and skip) an optional UTF-8 BOM at byte
// position 0, then hand off to lexLineStart for normal line-by-line scanning.
// If no BOM is present, it immediately delegates to lexLineStart.
func lexStart(l *lexer) stateFn {
	if l.pos == 0 && bytes.HasPrefix(l.bytes, bomPrefix) {
		l.pos += Pos(len(bomPrefix))
		l.emit(tokenBOM)
		return lexLineStart
	}
	return lexLineStart(l)
}

// lexLineStart is the main dispatch state — it decides what kind of construct
// starts the current line (or continues after a previous line's newline).
func lexLineStart(l *lexer) stateFn {
	switch r := l.peek(); {
	case r == eof:
		l.emit(tokenEOF)
		return nil
	case isNewlineStart(r):
		consumeNewlines(l)
		l.emit(tokenNewline)
		return lexLineStart
	case isWhitespace(r):
		consumeWhitespace(l)
		l.emit(tokenWhitespace)
		return lexLineStart
	case r == '#':
		return lexComment
	case r == '=':
		l.next()
		l.emit(tokenAssignmentOperator)
		return lexAfterAssign
	case r == '\'':
		l.next()
		l.emit(tokenLeftSingleQuote)
		return lexSingleQuoteValue
	case r == '"':
		l.next()
		l.emit(tokenLeftDoubleQuote)
		return lexDoubleQuoteValue
	case isKeyOrExportChar(r):
		return lexKeyOrExport(l, true, lexAfterKey)
	default:
		return lexInvalidLiteral(l, lexLineStart)
	}
}

// lexAfterExport handles scanning after the "export" keyword has been emitted.
// It is structurally similar to lexLineStart but with one key difference:
// allowExport=false is passed to lexKeyOrExport, because nested "export export KEY"
// is not valid — "export" can only appear once at the start of a line.
// The parser enforces that at least one whitespace token appears between
// "export" and the key.
func lexAfterExport(l *lexer) stateFn {
	switch r := l.peek(); {
	case r == eof:
		l.emit(tokenEOF)
		return nil
	case isNewlineStart(r):
		consumeNewlines(l)
		l.emit(tokenNewline)
		return lexLineStart
	case isWhitespace(r):
		consumeWhitespace(l)
		l.emit(tokenWhitespace)
		return lexAfterExport
	case r == '#':
		return lexComment
	case r == '=':
		l.next()
		l.emit(tokenAssignmentOperator)
		return lexAfterAssign
	case r == '\'':
		l.next()
		l.emit(tokenLeftSingleQuote)
		return lexSingleQuoteValue
	case r == '"':
		l.next()
		l.emit(tokenLeftDoubleQuote)
		return lexDoubleQuoteValue
	case isKeyOrExportChar(r):
		return lexKeyOrExport(l, false, lexAfterKey)
	default:
		return lexInvalidLiteral(l, lexAfterExport)
	}
}

// lexAfterKey handles the state after a key token has been emitted.
// At this point, the grammar expects optional whitespace followed by the
// assignment operator '='. Other valid tokens (newline, EOF, comment) are
// also accepted — the parser will decide whether the absence of '=' is an
// error at the grammar level. The lexer's job is just to tokenize faithfully.
func lexAfterKey(l *lexer) stateFn {
	switch r := l.peek(); {
	case r == eof:
		l.emit(tokenEOF)
		return nil
	case isNewlineStart(r):
		consumeNewlines(l)
		l.emit(tokenNewline)
		return lexLineStart
	case isWhitespace(r):
		consumeWhitespace(l)
		l.emit(tokenWhitespace)
		return lexAfterKey
	case r == '=':
		l.next()
		l.emit(tokenAssignmentOperator)
		return lexAfterAssign
	case r == '#':
		return lexComment
	case isKeyOrExportChar(r):
		return lexKeyOrExport(l, false, lexAfterKey)
	default:
		return lexInvalidLiteral(l, lexAfterKey)
	}
}

// lexAfterAssign handles the state immediately after the '=' operator.
// This is a critical decision point – the very next non-whitespace character
// determines the VALUE MODE (single-quoted, double-quoted, or unquoted).
//
// Note: a comment '#' is NOT valid here directly after '=' — it would be
// treated as an unquoted value character. The comment only becomes valid
// after whitespace is seen (handled in lexAfterAssignAndWhitespace).
func lexAfterAssign(l *lexer) stateFn {
	switch r := l.peek(); {
	case r == eof:
		l.emit(tokenEOF)
		return nil
	case isNewlineStart(r):
		consumeNewlines(l)
		l.emit(tokenNewline)
		return lexLineStart
	case isWhitespace(r):
		consumeWhitespace(l)
		l.emit(tokenWhitespace)
		return lexAfterAssignAndWhitespace
	case r == '\'':
		l.next()
		l.emit(tokenLeftSingleQuote)
		return lexSingleQuoteValue
	case r == '"':
		l.next()
		l.emit(tokenLeftDoubleQuote)
		return lexDoubleQuoteValue
	default:
		return lexUnquotedValue
	}
}

// lexAfterAssignAndWhitespace is entered when whitespace has been seen after '='.
// It is almost identical to lexAfterAssign but with one crucial addition:
// '#' is now recognized as a COMMENT start. This implements the rule:
//
//	KEY= # comment   → empty value, then comment
//	KEY=#            → unquoted value "#"
//
// The distinction exists because a '#' immediately after '=' (no space) is
// ambiguous with literal '#' content, so the spec treats it as a value char.
// Once whitespace appears, '#' unambiguously starts a comment.
//
// Note: this state is only ever entered from lexAfterAssign's whitespace case,
// which calls consumeWhitespace (consuming ALL horizontal whitespace) before
// transitioning here. Therefore the isWhitespace case can never fire on entry
// and is intentionally absent.
func lexAfterAssignAndWhitespace(l *lexer) stateFn {
	switch r := l.peek(); {
	case r == eof:
		l.emit(tokenEOF)
		return nil
	case isNewlineStart(r):
		consumeNewlines(l)
		l.emit(tokenNewline)
		return lexLineStart
	case r == '#':
		return lexComment
	case r == '\'':
		l.next()
		l.emit(tokenLeftSingleQuote)
		return lexSingleQuoteValue
	case r == '"':
		l.next()
		l.emit(tokenLeftDoubleQuote)
		return lexDoubleQuoteValue
	default:
		return lexUnquotedValue
	}
}

// lexUnquotedValue scans an unquoted value — the most complex value-scanning
// state. Unlike quoted values which have clear delimiters (' or "), an unquoted
// value's boundaries are determined by several rules:
//
//   - It ends at a newline or EOF.
//   - It ends before a '#' that is preceded by whitespace (inline comment).
//   - A '#' NOT preceded by whitespace is a literal character (e.g., KEY=a#b → "a#b").
//   - Trailing whitespace is stripped from the value.
//
// Implementation strategy:
// Rather than using the normal next()/backup() rune methods, this function
// scans ahead through the raw byte slice directly for efficiency. It tracks:
//   - scanPos:    the forward-scanning cursor
//   - lastNonWhitespace: the position just past the last non-whitespace character seen
//   - prevWasWhitespace: whether the previous character was horizontal whitespace
//
// After the scan, l.pos is set to lastNonWhitespace, which effectively strips trailing
// whitespace. The token [start, lastNonWhitespace) is then emitted as tokenUnquotedValue.
func lexUnquotedValue(l *lexer) stateFn {
	scanPos := l.pos
	lastNonWhitespace := l.pos
	prevWasWhitespace := false

	for int(scanPos) < len(l.bytes) {
		r, w := utf8.DecodeRune(l.bytes[scanPos:])
		if r == '\n' || r == '\r' {
			break
		}
		if r == '#' && prevWasWhitespace {
			break
		}

		scanPos += Pos(w)
		if r == ' ' || r == '\t' {
			prevWasWhitespace = true
		} else {
			prevWasWhitespace = false
			lastNonWhitespace = scanPos
		}
	}

	l.pos = lastNonWhitespace
	l.emit(tokenUnquotedValue)
	return lexAfterUnquotedValue
}

// lexAfterUnquotedValue handles the state after an unquoted value has been emitted.
// Valid following content: whitespace, a comment, a newline, or EOF. Any other
// character is treated as an invalid literal (the lexer emits it and the parser
// will reject it at the grammar level).
func lexAfterUnquotedValue(l *lexer) stateFn {
	switch r := l.peek(); {
	case r == eof:
		l.emit(tokenEOF)
		return nil
	case isNewlineStart(r):
		consumeNewlines(l)
		l.emit(tokenNewline)
		return lexLineStart
	case isWhitespace(r):
		consumeWhitespace(l)
		l.emit(tokenWhitespace)
		return lexAfterUnquotedValue
	case r == '#':
		return lexComment
	default:
		return lexInvalidLiteral(l, lexAfterUnquotedValue)
	}
}

// lexSingleQuoteValue scans the content between opening and closing single quotes.
// Single-quoted values are the simplest: every character is literal (no escape
// sequences) and newlines are NOT allowed. The scan simply consumes characters
// until it finds the closing "'", EOF (error), or a newline (error).
//
// When the closing quote is found, the cursor is backed up so that the quote
// itself is NOT included in the value token. The value [start, pos) is emitted,
// and then lexRightSingleQuote consumes the closing quote as a separate token.
func lexSingleQuoteValue(l *lexer) stateFn {
	for {
		switch l.next() {
		case eof:
			return l.errorf("unterminated single-quoted value")
		case '\n', '\r':
			return l.errorf("single-quoted values cannot span lines")
		case '\'':
			l.backup()
			l.emit(tokenSingleQuoteValue)
			return lexRightSingleQuote
		}
	}
}

// lexRightSingleQuote consumes the closing single-quote character and emits it
// as a distinct tokenRightSingleQuote token. Separating the quotes from the
// content makes the token stream unambiguous for the parser.
func lexRightSingleQuote(l *lexer) stateFn {
	if l.next() != '\'' {
		return l.errorf("expected closing single quote")
	}
	l.emit(tokenRightSingleQuote)
	return lexAfterQuotedValue
}

// lexDoubleQuoteValue scans the content between opening and closing double quotes.
// Double-quoted values support:
//   - ESCAPE SEQUENCES: the lexer preserves backslash escapes verbatim and lets
//     the parser apply ParseOptions during value processing.
//   - MULTI-LINE content: actual newlines inside the quotes are part of the value.
//   - Optionally, CRLF and CR line endings within the value are transformed by
//     the parser after unescaping, according to ParseOptions.
//
// The scan consumes characters until it finds an *unescaped* closing '"'.
// When a backslash is encountered, the next character is consumed unconditionally
// (so '\"' does not end the string).
//
// When the closing quote is found, the cursor backs up (to exclude the quote
// from the value range), and the token is emitted.
func lexDoubleQuoteValue(l *lexer) stateFn {
	for {
		switch r := l.next(); r {
		case eof:
			return l.errorf("unterminated double-quoted value")
		case '\\':
			esc := l.next()
			if esc == eof {
				return l.errorf("unterminated double-quoted value")
			}
		case '"':
			l.backup()
			l.emit(tokenDoubleQuoteValue)
			return lexRightDoubleQuote
		}
	}
}

// lexRightDoubleQuote consumes the closing double-quote character and emits it
// as a distinct tokenRightDoubleQuote token, mirroring lexRightSingleQuote.
func lexRightDoubleQuote(l *lexer) stateFn {
	if l.next() != '"' {
		return l.errorf("expected closing double quote")
	}
	l.emit(tokenRightDoubleQuote)
	return lexAfterQuotedValue
}

// lexAfterQuotedValue handles the state after a complete quoted value (single
// or double) has been emitted. Per the spec, the only valid content after a
// closing quote is: whitespace, a comment, a newline, or EOF. Any other
// character is an error (e.g., KEY='value'extraStuff).
func lexAfterQuotedValue(l *lexer) stateFn {
	switch r := l.peek(); {
	case r == eof:
		l.emit(tokenEOF)
		return nil
	case isNewlineStart(r):
		consumeNewlines(l)
		l.emit(tokenNewline)
		return lexLineStart
	case isWhitespace(r):
		consumeWhitespace(l)
		l.emit(tokenWhitespace)
		return lexAfterQuotedValue
	case r == '#':
		return lexComment
	default:
		return lexInvalidLiteral(l, lexAfterQuotedValue)
	}
}

// lexComment scans a comment. Comments start with '#' and extend to the end of
// the line (or EOF). The entire comment (including the '#') is captured as a
// single tokenComment token. Any following newline run is emitted separately by
// lexAfterComment as one tokenNewline. This clean separation keeps the token
// stream uniform and makes it easy for the parser to count lines.
func lexComment(l *lexer) stateFn {
	for {
		r := l.next()
		if r == eof {
			break
		}
		if r == '\n' || r == '\r' {
			l.backup()
			break
		}
	}
	l.emit(tokenComment)
	return lexAfterComment
}

// lexAfterComment follows a comment. The only valid next tokens are a newline
// (move to the next line) or EOF (end of input). Anything else would be a
// continuation of the same line, which is impossible since comments consume
// everything up to the line boundary.
func lexAfterComment(l *lexer) stateFn {
	switch r := l.peek(); {
	case r == eof:
		l.emit(tokenEOF)
		return nil
	case isNewlineStart(r):
		consumeNewlines(l)
		l.emit(tokenNewline)
		return lexLineStart
	default:
		return lexInvalidLiteral(l, lexAfterComment)
	}
}

// lexInvalidLiteral handles characters that don't fit the expected grammar at
// the current position. Rather than immediately erroring, it collects the
// unexpected characters into a tokenInvalidLiteral token and returns to the
// specified next state. This allows the PARSER (not the lexer) to decide how
// to report the error with full context. The lexer's philosophy is: tokenize
// faithfully, let the parser enforce grammar.
//
// The scan consumes characters until it hits a structural delimiter (whitespace,
// newline, '#', '=', or EOF), then emits whatever was collected.
func lexInvalidLiteral(l *lexer, next stateFn) stateFn {
	for {
		r := l.peek()
		if r == eof || isWhitespace(r) || isNewlineStart(r) || r == '#' || r == '=' {
			break
		}
		l.next()
	}

	if l.pos == l.start {
		if l.peek() == eof {
			l.emit(tokenEOF)
			return nil
		}
		l.next()
	}

	l.emit(tokenInvalidLiteral)
	return next
}

// lexKeyOrExport scans a contiguous run of ASCII word characters ([a-zA-Z0-9_]).
// After consuming the word, it checks whether the word is the "export" keyword.
// A word is classified as "export" only if ALL of these are true:
//  1. allowExport is true (we're at line start, not after another export)
//  2. The scanned bytes exactly match "export"
//  3. The very next character is horizontal whitespace (so "exporter" is a key,
//     not a keyword)
//
// If the word is "export", it's emitted as tokenExport and control moves to
// lexAfterExport. Otherwise, it's emitted as tokenKey and control moves to
// the caller-specified next state (typically lexAfterKey).
func lexKeyOrExport(l *lexer, allowExport bool, next stateFn) stateFn {
	for isKeyOrExportChar(l.peek()) {
		l.next()
	}

	if allowExport && slices.Equal(l.bytes[l.start:l.pos], exportBytes) && isWhitespace(l.peek()) {
		l.emit(tokenExport)
		return lexAfterExport
	}

	l.emit(tokenKey)
	return next
}

// ─── Helper Functions ────────────────────────────────────────────────────────
// These small utility functions are used throughout the state functions for
// common character-classification and consumption tasks.

// consumeWhitespace advances past a full run of horizontal whitespace (spaces
// and tabs). It is called after peeking confirms at least one whitespace
// character, so the resulting token [start, pos) always contains at least one
// character.
func consumeWhitespace(l *lexer) {
	for isWhitespace(l.peek()) {
		l.next()
	}
}

// consumeNewlines advances past a full run of adjacent line terminators:
// \n, \r, or \r\n in any consecutive combination. Line counting and CRLF
// handling are fully managed by next().
func consumeNewlines(l *lexer) {
	for isNewlineStart(l.peek()) {
		l.next()
	}
}

// isWhitespace returns true if the rune is horizontal whitespace: ASCII space (0x20)
// or tab (0x09). Other Unicode whitespace characters (e.g., non-breaking space)
// are intentionally excluded — the dotenv spec only recognizes these two.
func isWhitespace(r rune) bool {
	return r == ' ' || r == '\t'
}

// isNewlineStart returns true if the rune starts a line break: either LF (\n)
// or CR (\r, which may be standalone or the start of a CRLF pair).
func isNewlineStart(r rune) bool {
	return r == '\n' || r == '\r'
}

// bomPrefix is the UTF-8 encoding of the Unicode Byte Order Mark (U+FEFF).
// Some text editors prepend this invisible character to UTF-8 files.
var bomPrefix = []byte("\uFEFF")

// exportBytes is the byte slice for the "export" keyword, used for efficient comparison in lexKeyOrExport.
var exportBytes = []byte("export")

// lettersDigitsAndUnderscore is a lookup table for valid key characters:
// ASCII letters, digits, and underscore.
var lettersDigitsAndUnderscore = [128]bool{
	'0': true, '1': true, '2': true, '3': true, '4': true,
	'5': true, '6': true, '7': true, '8': true, '9': true,

	'A': true, 'B': true, 'C': true, 'D': true, 'E': true, 'F': true, 'G': true,
	'H': true, 'I': true, 'J': true, 'K': true, 'L': true, 'M': true, 'N': true,
	'O': true, 'P': true, 'Q': true, 'R': true, 'S': true, 'T': true, 'U': true,
	'V': true, 'W': true, 'X': true, 'Y': true, 'Z': true,

	'_': true,

	'a': true, 'b': true, 'c': true, 'd': true, 'e': true, 'f': true, 'g': true,
	'h': true, 'i': true, 'j': true, 'k': true, 'l': true, 'm': true, 'n': true,
	'o': true, 'p': true, 'q': true, 'r': true, 's': true, 't': true, 'u': true,
	'v': true, 'w': true, 'x': true, 'y': true, 'z': true,
}

// lettersAndUnderscore is a lookup table for valid key start characters:
// ASCII letters and underscore only - no digits.
var lettersAndUnderscore = [128]bool{
	'A': true, 'B': true, 'C': true, 'D': true, 'E': true, 'F': true, 'G': true,
	'H': true, 'I': true, 'J': true, 'K': true, 'L': true, 'M': true, 'N': true,
	'O': true, 'P': true, 'Q': true, 'R': true, 'S': true, 'T': true, 'U': true,
	'V': true, 'W': true, 'X': true, 'Y': true, 'Z': true,

	'_': true,

	'a': true, 'b': true, 'c': true, 'd': true, 'e': true, 'f': true, 'g': true,
	'h': true, 'i': true, 'j': true, 'k': true, 'l': true, 'm': true, 'n': true,
	'o': true, 'p': true, 'q': true, 'r': true, 's': true, 't': true, 'u': true,
	'v': true, 'w': true, 'x': true, 'y': true, 'z': true,
}

// isKeyOrExportChar returns true if the rune is valid in a key name
func isKeyOrExportChar(r rune) bool {
	return uint32(r) < 128 && lettersDigitsAndUnderscore[r]
}

// processValue applies resolved parse options to the raw byte content between
// double quotes while iterating the value once. Recognized backslash sequences
// are unescaped first, then the resulting stream is normalized as if CRLF and
// CR transforms were applied in order.
//
// When an Unescape* option is disabled, the corresponding backslash sequence is
// preserved literally. Escape sequences that are not specifically recognized are
// also passed through literally (backslash + character).
//
// A lone trailing backslash (no following byte) is treated as a literal '\' when
// UnescapeBackslashBackslash is false, and is an error otherwise.
func processValue(b []byte, opts resolvedParseOptions) (string, error) {
	var buf strings.Builder
	buf.Grow(len(b))

	pendingCR := false

	for i := 0; i < len(b); {
		c := b[i]

		if c == '\\' {
			if i+1 >= len(b) {
				if !opts.UnescapeBackslashBackslash {
					if pendingCR {
						if opts.TransformCRToLF {
							buf.WriteByte('\n')
						} else {
							buf.WriteByte('\r')
						}
						pendingCR = false
					}
					buf.WriteByte('\\')
					i++
					continue
				}
				return "", fmt.Errorf("trailing backslash in double-quoted value")
			}

			next := b[i+1]
			switch next {
			case '\\':
				if pendingCR {
					if opts.TransformCRToLF {
						buf.WriteByte('\n')
					} else {
						buf.WriteByte('\r')
					}
					pendingCR = false
				}
				if opts.UnescapeBackslashBackslash {
					buf.WriteByte('\\')
				} else {
					buf.WriteByte('\\')
					buf.WriteByte('\\')
				}
			case '"':
				if pendingCR {
					if opts.TransformCRToLF {
						buf.WriteByte('\n')
					} else {
						buf.WriteByte('\r')
					}
					pendingCR = false
				}
				if opts.UnescapeBackslashDoubleQuote {
					buf.WriteByte('"')
				} else {
					buf.WriteByte('\\')
					buf.WriteByte('"')
				}
			case '\'':
				if pendingCR {
					if opts.TransformCRToLF {
						buf.WriteByte('\n')
					} else {
						buf.WriteByte('\r')
					}
					pendingCR = false
				}
				if opts.UnescapeBackslashSingleQuote {
					buf.WriteByte('\'')
				} else {
					buf.WriteByte('\\')
					buf.WriteByte('\'')
				}
			case 'a':
				if pendingCR {
					if opts.TransformCRToLF {
						buf.WriteByte('\n')
					} else {
						buf.WriteByte('\r')
					}
					pendingCR = false
				}
				if opts.UnescapeBackslashA {
					buf.WriteByte('\a')
				} else {
					buf.WriteByte('\\')
					buf.WriteByte('a')
				}
			case 'b':
				if pendingCR {
					if opts.TransformCRToLF {
						buf.WriteByte('\n')
					} else {
						buf.WriteByte('\r')
					}
					pendingCR = false
				}
				if opts.UnescapeBackslashB {
					buf.WriteByte('\b')
				} else {
					buf.WriteByte('\\')
					buf.WriteByte('b')
				}
			case 'f':
				if pendingCR {
					if opts.TransformCRToLF {
						buf.WriteByte('\n')
					} else {
						buf.WriteByte('\r')
					}
					pendingCR = false
				}
				if opts.UnescapeBackslashF {
					buf.WriteByte('\f')
				} else {
					buf.WriteByte('\\')
					buf.WriteByte('f')
				}
			case 'n':
				if opts.UnescapeBackslashN {
					if pendingCR {
						if opts.TransformCRLFToLF {
							buf.WriteByte('\n')
							pendingCR = false
						} else {
							if opts.TransformCRToLF {
								buf.WriteByte('\n')
							} else {
								buf.WriteByte('\r')
							}
							pendingCR = false
							buf.WriteByte('\n')
						}
					} else {
						buf.WriteByte('\n')
					}
				} else {
					if pendingCR {
						if opts.TransformCRToLF {
							buf.WriteByte('\n')
						} else {
							buf.WriteByte('\r')
						}
						pendingCR = false
					}
					buf.WriteByte('\\')
					buf.WriteByte('n')
				}
			case 'r':
				if opts.UnescapeBackslashR {
					if pendingCR {
						if opts.TransformCRToLF {
							buf.WriteByte('\n')
						} else {
							buf.WriteByte('\r')
						}
					}
					pendingCR = true
				} else {
					if pendingCR {
						if opts.TransformCRToLF {
							buf.WriteByte('\n')
						} else {
							buf.WriteByte('\r')
						}
						pendingCR = false
					}
					buf.WriteByte('\\')
					buf.WriteByte('r')
				}
			case 't':
				if pendingCR {
					if opts.TransformCRToLF {
						buf.WriteByte('\n')
					} else {
						buf.WriteByte('\r')
					}
					pendingCR = false
				}
				if opts.UnescapeBackslashT {
					buf.WriteByte('\t')
				} else {
					buf.WriteByte('\\')
					buf.WriteByte('t')
				}
			case 'v':
				if pendingCR {
					if opts.TransformCRToLF {
						buf.WriteByte('\n')
					} else {
						buf.WriteByte('\r')
					}
					pendingCR = false
				}
				if opts.UnescapeBackslashV {
					buf.WriteByte('\v')
				} else {
					buf.WriteByte('\\')
					buf.WriteByte('v')
				}
			default:
				if pendingCR {
					if opts.TransformCRToLF {
						buf.WriteByte('\n')
					} else {
						buf.WriteByte('\r')
					}
					pendingCR = false
				}
				r, w := utf8.DecodeRune(b[i+1:])
				buf.WriteByte('\\')
				buf.WriteRune(r)
				i += 1 + w
				continue
			}

			i += 2
		} else if c < utf8.RuneSelf {
			if pendingCR {
				if c == '\n' && opts.TransformCRLFToLF {
					buf.WriteByte('\n')
					pendingCR = false
					i++
					continue
				}
				if opts.TransformCRToLF {
					buf.WriteByte('\n')
				} else {
					buf.WriteByte('\r')
				}
				pendingCR = false
			}
			if c == '\r' {
				pendingCR = true
				i++
				continue
			}
			buf.WriteByte(c)
			i++
		} else {
			if pendingCR {
				if opts.TransformCRToLF {
					buf.WriteByte('\n')
				} else {
					buf.WriteByte('\r')
				}
				pendingCR = false
			}

			_, w := utf8.DecodeRune(b[i:])
			buf.Write(b[i : i+w])
			i += w
		}
	}

	if pendingCR {
		if opts.TransformCRToLF {
			buf.WriteByte('\n')
		} else {
			buf.WriteByte('\r')
		}
	}

	return buf.String(), nil
}
