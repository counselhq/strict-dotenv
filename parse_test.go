package strictdotenv

import (
	"maps"
	"testing"
)

type testCase struct {
	name    string
	dotenv  string
	want    EnvStore
	wantErr bool
}

// ---------------------------------------------------------------------------
// Test Empty Dotenv | Whitespace Only | Newlines Only
// ---------------------------------------------------------------------------
//
// 	- Empty dotenv files are ok and should return an empty map
// 	- Dotenv files with only whitespace / newlines should return an empty map
// 	- For strict-dotenv: whitespace is defined as spaces and tabs only
// ---------------------------------------------------------------------------

func TestNoValuesEmptyDotenv(t *testing.T) {
	cfg := NewParseConfig().WithRecommendedDefaults()

	run(t, nil, cfg, testCase{name: "empty file",
		dotenv: "",
		want:   EnvStore{},
	})
}

func TestNoValuesWhitespaceOnly(t *testing.T) {
	cfg := NewParseConfig().WithRecommendedDefaults()

	run(t, nil, cfg, testCase{name: "Dotenv with spaces only",
		dotenv: "   ",
		want:   EnvStore{},
	})

	run(t, nil, cfg, testCase{name: "Dotenv with tabs only",
		dotenv: "\t\t\t",
		want:   EnvStore{},
	})

	run(t, nil, cfg, testCase{name: "Dotenv with spaces and tabs only",
		dotenv: "\t \t \t",
		want:   EnvStore{},
	})
}

func TestNoValuesWhitespaceExclusions(t *testing.T) {
	cfg := NewParseConfig().WithRecommendedDefaults()

	run(t, nil, cfg, testCase{name: "form feed is not whitespace",
		dotenv:  "\f",
		wantErr: true,
	})

	run(t, nil, cfg, testCase{name: "vertical tab is not whitespace",
		dotenv:  "\v",
		wantErr: true,
	})
}

func TestNoValuesNewlinesOnly(t *testing.T) {
	cfg := NewParseConfig().WithRecommendedDefaults()

	run(t, nil, cfg, testCase{name: "Dotenv with LF only",
		dotenv: "\n\n\n",
		want:   EnvStore{},
	})

	run(t, nil, cfg, testCase{name: "Dotenv with CR only",
		dotenv: "\r\r\r",
		want:   EnvStore{},
	})

	run(t, nil, cfg, testCase{name: "Dotenv with CRLF only",
		dotenv: "\r\n\r\n\r\n",
		want:   EnvStore{},
	})

	run(t, nil, cfg, testCase{name: "Dotenv with LF CR and CRLF only",
		dotenv: "\n\r\r\n",
		want:   EnvStore{},
	})

	run(t, nil, cfg, testCase{name: "Dotenv with mixed newlines and whitespace only",
		dotenv: " \n\t\r \r\n \t\r",
		want:   EnvStore{},
	})
}

// ---------------------------------------------------------------------------
// Test Keys
// ---------------------------------------------------------------------------
//
// 	- Must contain only ASCII letters, digits, and underscores
// 	- Must not start with a digit
// 	- Must have length > 0
// ---------------------------------------------------------------------------

func TestKeysSupported(t *testing.T) {
	cfg := NewParseConfig().WithRecommendedDefaults()

	run(t, nil, cfg, testCase{name: "keys with only uppercase letters supported",
		dotenv: "KEY=value",
		want:   EnvStore{"KEY": "value"},
	})

	run(t, nil, cfg, testCase{name: "keys with only lowercase letters supported",
		dotenv: "key=value",
		want:   EnvStore{"key": "value"},
	})

	run(t, nil, cfg, testCase{name: "keys with only underscores supported",
		dotenv: "_=value\n___=value2",
		want:   EnvStore{"_": "value", "___": "value2"},
	})

	run(t, nil, cfg, testCase{name: "keys with digits supported",
		dotenv: "KEY1=value",
		want:   EnvStore{"KEY1": "value"},
	})

	run(t, nil, cfg, testCase{name: "keys with mixed case letters, digits, and underscores supported",
		dotenv: "_K_e_Y_1=value",
		want:   EnvStore{"_K_e_Y_1": "value"},
	})

	run(t, nil, cfg, testCase{name: "keys with leading and trailing underscores supported",
		dotenv: "__SECRET_KEY___=value",
		want:   EnvStore{"__SECRET_KEY___": "value"},
	})

	run(t, nil, cfg, testCase{
		name:   "keys are case sensitive",
		dotenv: "KEY=upper\nkey=lower\nKeY=MixedCase",
		want:   EnvStore{"KEY": "upper", "key": "lower", "KeY": "MixedCase"},
	})
}

func TestKeysUnsupported(t *testing.T) {
	cfg := NewParseConfig().WithRecommendedDefaults()

	run(t, nil, cfg, testCase{name: "single quoted keys not supported",
		dotenv:  "'KEY'=value",
		wantErr: true,
	})

	run(t, nil, cfg, testCase{name: "double quoted keys not supported",
		dotenv:  "\"KEY\"=value",
		wantErr: true,
	})

	run(t, nil, cfg, testCase{name: "back tick keys not supported",
		dotenv:  "`KEY`=value",
		wantErr: true,
	})

	run(t, nil, cfg, testCase{name: "keys cannot start with digit",
		dotenv:  "1KEY=value",
		wantErr: true,
	})

	run(t, nil, cfg, testCase{name: "keys cannot contain hyphens",
		dotenv:  "KEY-WITH-HYPHEN=value",
		wantErr: true,
	})

	run(t, nil, cfg, testCase{name: "keys cannot contain dollar signs",
		dotenv:  "K$EY=value",
		wantErr: true,
	})

	run(t, nil, cfg, testCase{name: "keys cannot contain dots",
		dotenv:  "K.EY=value",
		wantErr: true,
	})

	run(t, nil, cfg, testCase{name: "keys cannot start with equal sign",
		dotenv:  "=KEY=value",
		wantErr: true,
	})

	run(t, nil, cfg, testCase{name: "missing key is an error",
		dotenv:  "=value",
		wantErr: true,
	})
}

// ---------------------------------------------------------------------------
// Test Assignment Operator
// ---------------------------------------------------------------------------
//
// 	- At least one equals sign must be present between each key-value pair
// 	- No YAML syntax allowed (e.g. "KEY: value")
// ---------------------------------------------------------------------------

func TestAssignmentOperatorErrors(t *testing.T) {
	cfg := NewParseConfig().WithRecommendedDefaults()

	run(t, nil, cfg, testCase{name: "key without eq or value is error",
		dotenv:  "KEY",
		wantErr: true,
	})

	run(t, nil, cfg, testCase{name: "key and value without eq is error",
		dotenv:  "KEY VALUE",
		wantErr: true,
	})

	run(t, nil, cfg, testCase{name: "yaml syntax is not supported",
		dotenv:  "KEY: VALUE",
		wantErr: true,
	})

	run(t, nil, cfg, testCase{name: "key colon eq value is not supported",
		dotenv:  "KEY:=VALUE",
		wantErr: true,
	})

	run(t, nil, cfg, testCase{name: "key LF eq value is not supported",
		dotenv:  "KEY\n=value",
		wantErr: true,
	})

	run(t, nil, cfg, testCase{name: "key CR eq value is not supported",
		dotenv:  "KEY\r=value",
		wantErr: true,
	})

	run(t, nil, cfg, testCase{name: "key CRLF eq value is not supported",
		dotenv:  "KEY\r\n=value",
		wantErr: true,
	})
}

// ---------------------------------------------------------------------------
// Test Unquoted Values
// ---------------------------------------------------------------------------
//
// 	- If first char is not a single/double quote, treat the value as unquoted
// 	- Unquoted values continue until comment, newline, or end of file
// 	- No unescaping / everything treated as literal
// ---------------------------------------------------------------------------

func TestUnquoted(t *testing.T) {
	cfg := NewParseConfig().WithRecommendedDefaults()

	run(t, nil, cfg, testCase{name: "base case",
		dotenv: "K=V\nKEY=value",
		want:   EnvStore{"K": "V", "KEY": "value"},
	})

	run(t, nil, cfg, testCase{name: "key eq then LF | CR | CRLF | EOF is empty value",
		dotenv: "KEY=\nKEY2=\rKEY3=\r\nKEY4=",
		want:   EnvStore{"KEY": "", "KEY2": "", "KEY3": "", "KEY4": ""},
	})

	run(t, nil, cfg, testCase{
		name:   "key eq space then LF | CR | CRLF | EOF is empty value",
		dotenv: "KEY= \nKEY2= \rKEY3= \r\nKEY4= ",
		want:   EnvStore{"KEY": "", "KEY2": "", "KEY3": "", "KEY4": ""},
	})

	run(t, nil, cfg, testCase{
		name:   "key eq tab then LF | CR | CRLF | EOF is empty value",
		dotenv: "KEY=\t\nKEY2=\t\rKEY3=\t\r\nKEY4=\t",
		want:   EnvStore{"KEY": "", "KEY2": "", "KEY3": "", "KEY4": ""},
	})

	run(t, nil, cfg, testCase{name: "if starts unquoted, treat any single quotes as literals",
		dotenv: "KEY=a'b'c'd'e",
		want:   EnvStore{"KEY": "a'b'c'd'e"},
	})

	run(t, nil, cfg, testCase{name: "if starts unquoted, treat any double quotes as literals",
		dotenv: `KEY=a"b"c"d"e`,
		want:   EnvStore{"KEY": `a"b"c"d"e`},
	})
}

func TestUnquotedWithWhitespace(t *testing.T) {
	cfg := NewParseConfig().WithRecommendedDefaults()

	run(t, nil, cfg, testCase{name: "space between key and eq ok",
		dotenv: "KEY =value",
		want:   EnvStore{"KEY": "value"},
	})

	run(t, nil, cfg, testCase{name: "space between eq and value ok",
		dotenv: "KEY= value",
		want:   EnvStore{"KEY": "value"},
	})

	run(t, nil, cfg, testCase{name: "spaces between key and value ok",
		dotenv: "KEY = value",
		want:   EnvStore{"KEY": "value"},
	})

	run(t, nil, cfg, testCase{name: "multiple spaces around key and eq and value ok",
		dotenv: "  KEY  =  value  ",
		want:   EnvStore{"KEY": "value"},
	})

	run(t, nil, cfg, testCase{name: "spaces around eq and inside value ok",
		dotenv: "KEY = value with spaces",
		want:   EnvStore{"KEY": "value with spaces"},
	})

	run(t, nil, cfg, testCase{name: "tabs around eq and inside value ok",
		dotenv: "KEY\t=\tvalue\twith\tspaces",
		want:   EnvStore{"KEY": "value\twith\tspaces"},
	})

	run(t, nil, cfg, testCase{name: "spaces inside value ending in space hyphen ok",
		dotenv: "KEY = value with spaces -",
		want:   EnvStore{"KEY": "value with spaces -"},
	})

	run(t, nil, cfg, testCase{name: "trailing spaces after unquoted values are trimmed",
		dotenv: "KEY = value with trailing spaces  ",
		want:   EnvStore{"KEY": "value with trailing spaces"},
	})

	run(t, nil, cfg, testCase{name: "trailing tabs after unquoted values are trimmed",
		dotenv: "KEY = value\twith\ttrailing\tspaces\t\t",
		want:   EnvStore{"KEY": "value\twith\ttrailing\tspaces"},
	})

	run(t, nil, cfg, testCase{
		name:   "mixed tabs and spaces around eq",
		dotenv: " KEY\t\t= \tvalue\t\nKEY2 \t =value2\t\t",
		want:   EnvStore{"KEY": "value", "KEY2": "value2"},
	})

	run(t, nil, cfg, testCase{name: "LF at start and end of file",
		dotenv: "\nKEY=VALUE\n\n\n",
		want:   EnvStore{"KEY": "VALUE"},
	})

	run(t, nil, cfg, testCase{name: "mixed newlines and whitespace",
		dotenv: "\r\n\r\nKEY\t=\tVALUE\r\n\r\n\r\n",
		want:   EnvStore{"KEY": "VALUE"},
	})

	run(t, nil, cfg, testCase{name: "other special characters taken as-is in unquoted values",
		dotenv: "KEY=VALUE\a\b\f\v\t",
		want:   EnvStore{"KEY": "VALUE\a\b\f\v"},
	})

	run(t, nil, cfg, testCase{name: "escape sequences treated as literals in unquoted values",
		dotenv: `KEY=-\\\a\b\f\v\t\n\r\`,
		want:   EnvStore{"KEY": `-\\\a\b\f\v\t\n\r\`},
	})
}

// ---------------------------------------------------------------------------
// Test Single-Quoted Values
// ---------------------------------------------------------------------------
//
// 	- Everything between the single quotes is treated as literal,
// 	  including backslashes and newlines.
// ---------------------------------------------------------------------------

func TestSingleQuoted(t *testing.T) {
	cfg := NewParseConfig().WithRecommendedDefaults()

	run(t, nil, cfg, testCase{name: "single quote base case",
		dotenv: "KEY='value'",
		want:   EnvStore{"KEY": "value"},
	})

	run(t, nil, cfg, testCase{name: "empty single quote",
		dotenv: "KEY=''",
		want:   EnvStore{"KEY": ""},
	})

	run(t, nil, cfg, testCase{name: "multiple single quoted key-value pairs",
		dotenv: "KEY1='Value 1'\nKEY2='Value 2'",
		want:   EnvStore{"KEY1": "Value 1", "KEY2": "Value 2"},
	})
}

func TestSingleQuotedWithWhitespace(t *testing.T) {
	cfg := NewParseConfig().WithRecommendedDefaults()

	run(t, nil, cfg, testCase{name: "spaces between key and value",
		dotenv: "KEY = 'value'",
		want:   EnvStore{"KEY": "value"},
	})

	run(t, nil, cfg, testCase{name: "multiple spaces around key and eq and value",
		dotenv: "  KEY  =  'value'  ",
		want:   EnvStore{"KEY": "value"},
	})

	run(t, nil, cfg, testCase{name: "preserve all whitespace inside single quotes",
		dotenv: "KEY = '	\tvalue with\ttrailing tabs	\t	\t'",
		want:   EnvStore{"KEY": "	\tvalue with\ttrailing tabs	\t	\t"},
	})

	run(t, nil, cfg, testCase{
		name:   "different types of whitespace outside of single quotes",
		dotenv: " KEY\t\t= \t'value'\t\nKEY2 \t ='value2'\t\t",
		want:   EnvStore{"KEY": "value", "KEY2": "value2"},
	})

	run(t, nil, cfg, testCase{name: "mixed newlines before key and after single quoted value",
		dotenv: "\r\n\nKEY='VALUE'\n\r\n\r\n\n\n",
		want:   EnvStore{"KEY": "VALUE"},
	})
}

func TestSingleQuotedEscaping(t *testing.T) {
	cfg := NewParseConfig().WithRecommendedDefaults()

	run(t, nil, cfg, testCase{name: "backslash is literal in single-quoted values",
		dotenv: "KEY='back\\slash'",
		want:   EnvStore{"KEY": `back\slash`},
	})

	run(t, nil, cfg, testCase{name: "backslash before closing single quote is ok literal",
		dotenv: "KEY='value\\'",
		want:   EnvStore{"KEY": `value\`},
	})

	run(t, nil, cfg, testCase{name: "escape sequences treated as literals in single quoted values",
		dotenv: `KEY='-\\\a\b\f\v\t\n\r\'`,
		want:   EnvStore{"KEY": `-\\\a\b\f\v\t\n\r\`},
	})
}
func TestAllowedContentAfterClosingSingleQuote(t *testing.T) {
	cfg := NewParseConfig().WithRecommendedDefaults()

	run(t, nil, cfg, testCase{name: "tab after closing single quote ok",
		dotenv: "KEY='line1'\t",
		want:   EnvStore{"KEY": "line1"},
	})

	run(t, nil, cfg, testCase{name: "space after closing single quote ok",
		dotenv: "KEY='line1' ",
		want:   EnvStore{"KEY": "line1"},
	})

	run(t, nil, cfg, testCase{name: "no space then inline comment after closing single quote ok",
		dotenv: "KEY='line1'# comment",
		want:   EnvStore{"KEY": "line1"},
	})

	run(t, nil, cfg, testCase{name: "space then inline comment after closing single quote ok",
		dotenv: "KEY='line1' # comment",
		want:   EnvStore{"KEY": "line1"},
	})

	run(t, nil, cfg, testCase{name: "tab then inline comment after closing single quote ok",
		dotenv: "KEY='line1'\t# comment",
		want:   EnvStore{"KEY": "line1"},
	})
}

func TestSingleQuotedErrorCases(t *testing.T) {
	cfg := NewParseConfig().WithRecommendedDefaults()

	run(t, nil, cfg, testCase{name: "single quotes treat backslashes as literal",
		dotenv:  "KEY='Value\\''",
		wantErr: true,
	})

	run(t, nil, cfg, testCase{name: "no LF multi-line single quoted values allowed",
		dotenv:  "KEY='VALUE\nVALUE'",
		wantErr: true,
	})

	run(t, nil, cfg, testCase{name: "no CR multi-line single quoted values allowed",
		dotenv:  "KEY='VALUE\rVALUE'",
		wantErr: true,
	})

	run(t, nil, cfg, testCase{name: "no CRLF multi-line single quoted values allowed",
		dotenv:  "KEY='VALUE\r\nVALUE'",
		wantErr: true,
	})

	run(t, nil, cfg, testCase{name: "not allowed after closing single quote - no alphanumeric",
		dotenv:  "KEY='VALUE'alpha",
		wantErr: true,
	})

	run(t, nil, cfg, testCase{name: "not allowed after closing single quote - no single quote",
		dotenv:  "KEY='VALUE''",
		wantErr: true,
	})

	run(t, nil, cfg, testCase{name: "not allowed after closing single quote - no space then alphanumeric",
		dotenv:  "KEY='VALUE' alpha",
		wantErr: true,
	})

	run(t, nil, cfg, testCase{name: "not allowed after closing single quote - no space then single quote",
		dotenv:  "KEY='VALUE' '",
		wantErr: true,
	})

	run(t, nil, cfg, testCase{name: "not allowed after closing single quote - no form feed",
		dotenv:  "KEY='VALUE'\f",
		wantErr: true,
	})

	run(t, nil, cfg, testCase{name: "not allowed after closing single quote - no vertical tab",
		dotenv:  "KEY='VALUE'\v",
		wantErr: true,
	})
}

// ---------------------------------------------------------------------------
// Test Double-Quoted Values
// ---------------------------------------------------------------------------
//
// 	- Supports default unescaping of \\, \", \n, \t, and \r
// 	- Supports optional transforms of unescaped newlines (e.g. \r\n -> \n)
// ---------------------------------------------------------------------------

func TestDoubleQuoted(t *testing.T) {
	cfg := NewParseConfig().WithRecommendedDefaults()

	run(t, nil, cfg, testCase{name: "double quote base case",
		dotenv: "KEY=\"value\"",
		want:   EnvStore{"KEY": "value"},
	})

	run(t, nil, cfg, testCase{name: "empty double quoted value",
		dotenv: "KEY=\"\"",
		want:   EnvStore{"KEY": ""},
	})

	run(t, nil, cfg, testCase{name: "multiple double quoted key-value pairs",
		dotenv: "KEY1=\"Value 1\"\nKEY2=\"Value 2\"",
		want:   EnvStore{"KEY1": "Value 1", "KEY2": "Value 2"},
	})

	run(t, nil, cfg, testCase{name: "value with escaped double quote",
		dotenv: "KEY=\"Value \\\"1\\\"\"",
		want:   EnvStore{"KEY": `Value "1"`},
	})

	run(t, nil, cfg, testCase{name: "value with escape sequences",
		dotenv: "KEY=\"line1\\nline2\\t\\r\\\\\"",
		want:   EnvStore{"KEY": "line1\nline2\t\n\\"},
	})
}

func TestDoubleQuotedWithWhitespaceAndNewlines(t *testing.T) {
	cfg := NewParseConfig().WithRecommendedDefaults()

	run(t, nil, cfg, testCase{name: "spaces between key and value",
		dotenv: "KEY = \"value\"",
		want:   EnvStore{"KEY": "value"},
	})

	run(t, nil, cfg, testCase{name: "multiple spaces around key and eq and value",
		dotenv: "  KEY  =  \"value\"  ",
		want:   EnvStore{"KEY": "value"},
	})

	run(t, nil, cfg, testCase{name: "spaces and tabs inside double quotes are preserved",
		dotenv: "KEY = \"value\twith trailing spaces  \t\"",
		want:   EnvStore{"KEY": "value\twith trailing spaces  \t"},
	})

	run(t, nil, cfg, testCase{
		name:   "mixed whitespace outside of double quote",
		dotenv: " KEY\t\t= \t\"value\"\t\nKEY2 \t =\"value2\"   \t\t",
		want:   EnvStore{"KEY": "value", "KEY2": "value2"},
	})

	run(t, nil, cfg, testCase{name: "newlines LF",
		dotenv: "\nKEY=\"VALUE\"\n\n\n",
		want:   EnvStore{"KEY": "VALUE"},
	})

	run(t, nil, cfg, testCase{name: "newlines CRLF",
		dotenv: "\r\n\r\nKEY=\"VALUE\"\r\n\r\n\r\n",
		want:   EnvStore{"KEY": "VALUE"},
	})

	run(t, nil, cfg, testCase{name: "newlines mixed",
		dotenv: "\r\n\nKEY=\"VALUE\"\n\r\n\r\n\n\n",
		want:   EnvStore{"KEY": "VALUE"},
	})
}

func TestDoubleQuotedMultiLine(t *testing.T) {
	cfg := NewParseConfig().WithRecommendedDefaults()

	run(t, nil, cfg, testCase{name: "basic multi-line value",
		dotenv: "KEY=\"line1\nline2\nline3\"",
		want:   EnvStore{"KEY": "line1\nline2\nline3"},
	})

	run(t, nil, cfg, testCase{name: "multi-line value with export and unspaced comment",
		dotenv: "export KEY=\"line1\nline2\nline3\"#comment",
		want:   EnvStore{"KEY": "line1\nline2\nline3"},
	})

	run(t, nil, cfg, testCase{name: "multi-line value with space comment",
		dotenv: "KEY=\"line1\nline2\nline3\" # comment",
		want:   EnvStore{"KEY": "line1\nline2\nline3"},
	})

	run(t, nil, cfg, testCase{name: "multi-line value preserve leading and trailing LF",
		dotenv: "KEY=\"\nline1\nline2\nline3\n\"",
		want:   EnvStore{"KEY": "\nline1\nline2\nline3\n"},
	})

	run(t, nil, cfg, testCase{name: "normalize each CRLF and CR to LF in multi-line value",
		dotenv: "KEY=\"\r\nline1\r\rline2\r\n\r\n \tline3\r\n\"",
		want:   EnvStore{"KEY": "\nline1\n\nline2\n\n \tline3\n"},
	})
}

func TestDoubleQuotedUnknownEscapeLiteral(t *testing.T) {
	cfg := NewParseConfig().WithRecommendedDefaults()

	run(t, nil, cfg, testCase{name: "unknown escape \\u passes through literally",
		dotenv: "KEY=\"\\u0041\"",
		want:   EnvStore{"KEY": `\u0041`},
	})

	run(t, nil, cfg, testCase{name: "unknown escape \\x passes through literally",
		dotenv: "KEY=\"\\x41\"",
		want:   EnvStore{"KEY": `\x41`},
	})

	run(t, nil, cfg, testCase{name: "escaped dollar sign passes through literally",
		dotenv: "KEY=\"\\$NOTAVAR\"",
		want:   EnvStore{"KEY": `\$NOTAVAR`},
	})

	run(t, nil, cfg, testCase{name: "unknown escape \\0 passes through literally",
		dotenv: "KEY=\"\\0\"",
		want:   EnvStore{"KEY": `\0`},
	})

	run(t, nil, cfg, testCase{name: "unknown escape with unicode char passes through literally",
		dotenv: "KEY=\"\\é\"",
		want:   EnvStore{"KEY": `\é`},
	})
}

func TestDoubleQuotedErrorCases(t *testing.T) {
	cfg := NewParseConfig().WithRecommendedDefaults()

	run(t, nil, cfg, testCase{name: "missing closing double quote one line",
		dotenv:  "KEY=\"VALUE",
		wantErr: true,
	})

	run(t, nil, cfg, testCase{name: "missing closing double quote multi line",
		dotenv:  "KEY=\"VALUE\nNextLine",
		wantErr: true,
	})

	run(t, nil, cfg, testCase{name: "escaped closing double quote consumes the quote leaving no real close",
		dotenv:  `KEY="value\"`,
		wantErr: true,
	})

	run(t, nil, cfg, testCase{name: "not allowed after closing double quote - no alphanumeric",
		dotenv:  "KEY=\"VALUE\"alpha",
		wantErr: true,
	})

	run(t, nil, cfg, testCase{name: "not allowed after closing double quote - no double quote",
		dotenv:  "KEY=\"VALUE\"\"",
		wantErr: true,
	})

	run(t, nil, cfg, testCase{name: "not allowed after closing double quote - no  space then alphanumeric",
		dotenv:  "KEY=\"VALUE\" alpha",
		wantErr: true,
	})

	run(t, nil, cfg, testCase{name: "not allowed after closing double quote - no  space then double quote",
		dotenv:  "KEY=\"VALUE\" \"",
		wantErr: true,
	})

	run(t, nil, cfg, testCase{name: "same rules apply after multi-line double quote value - no alphanumeric",
		dotenv:  "KEY=\"VALUE\nVALUE\" extra",
		wantErr: true,
	})

	run(t, nil, cfg, testCase{name: "not allowed after closing double quote - no form feed",
		dotenv:  "KEY=\"VALUE\"\f",
		wantErr: true,
	})

	run(t, nil, cfg, testCase{name: "not allowed after closing double quote - no vertical tab",
		dotenv:  "KEY=\"VALUE\"\v",
		wantErr: true,
	})
}

// ---------------------------------------------------------------------------
// Test export
// ---------------------------------------------------------------------------

func TestExportCases(t *testing.T) {
	cfg := NewParseConfig().WithRecommendedDefaults()

	run(t, nil, cfg, testCase{name: "multiple exports with different quoting styles",
		dotenv: "export K1=v1\nexport K2='v2'\nexport K3=\"v3\"",
		want:   EnvStore{"K1": "v1", "K2": "v2", "K3": "v3"},
	})

	run(t, nil, cfg, testCase{name: "tabs around export",
		dotenv: "\texport\tKEY=value",
		want:   EnvStore{"KEY": "value"},
	})

	run(t, nil, cfg, testCase{name: "multiple whitespace around export",
		dotenv: "\t \t export\t\t\t  KEY=value  ",
		want:   EnvStore{"KEY": "value"},
	})

	run(t, nil, cfg, testCase{name: "requires whitespace between export and key",
		dotenv: "exportKEY=value",
		want:   EnvStore{"exportKEY": "value"},
	})

	run(t, nil, cfg, testCase{name: "export with inline comment",
		dotenv: "export KEY=value # comment",
		want:   EnvStore{"KEY": "value"},
	})

	run(t, nil, cfg, testCase{name: "export with export key",
		dotenv: "export export=value",
		want:   EnvStore{"export": "value"},
	})

	run(t, nil, cfg, testCase{name: "export with EXPORT key",
		dotenv: "export EXPORT=value",
		want:   EnvStore{"EXPORT": "value"},
	})

	run(t, nil, cfg, testCase{name: "export mid file",
		dotenv: "K1=V1\nexport KEY=value",
		want:   EnvStore{"K1": "V1", "KEY": "value"},
	})
}

func TestNonExportCases(t *testing.T) {
	cfg := NewParseConfig().WithRecommendedDefaults()

	run(t, nil, cfg, testCase{name: "export is the key if no key after export",
		dotenv: "export = value",
		want:   EnvStore{"export": "value"},
	})
}

func TestExportErrorCases(t *testing.T) {
	cfg := NewParseConfig().WithRecommendedDefaults()

	run(t, nil, cfg, testCase{name: "export alone on line is error",
		dotenv:  "export",
		wantErr: true,
	})

	run(t, nil, cfg, testCase{name: "export and key without assignment operator is an error",
		dotenv:  "export KEY",
		wantErr: true,
	})

	run(t, nil, cfg, testCase{name: "no form feed between export and key",
		dotenv:  "export \fKEY=value",
		wantErr: true,
	})

	run(t, nil, cfg, testCase{name: "no vertical tab between export and key",
		dotenv:  "export \vKEY=value",
		wantErr: true,
	})

	run(t, nil, cfg, testCase{name: "key requirements still apply with export",
		dotenv:  "export KEY-WITH-HYPHEN",
		wantErr: true,
	})
}

// ---------------------------------------------------------------------------
// Test Comments
// ---------------------------------------------------------------------------

func TestCommentCases(t *testing.T) {
	cfg := NewParseConfig().WithRecommendedDefaults()

	run(t, nil, cfg, testCase{name: "line comment",
		dotenv: "# KEY=value",
		want:   EnvStore{},
	})

	run(t, nil, cfg, testCase{name: "line comment with leading spaces and tabs",
		dotenv: "  \t \t# KEY=value",
		want:   EnvStore{},
	})

	run(t, nil, cfg, testCase{name: "line comment does not impact next line",
		dotenv: "# KEY1=value1\nKEY2=value2",
		want:   EnvStore{"KEY2": "value2"},
	})

	run(t, nil, cfg, testCase{name: "line comment with multiple hashes",
		dotenv: "## K=V ## \nKEY=value",
		want:   EnvStore{"KEY": "value"},
	})

	run(t, nil, cfg, testCase{name: "space inline comment after unquoted value",
		dotenv: "KEY=value # K=V",
		want:   EnvStore{"KEY": "value"},
	})

	run(t, nil, cfg, testCase{name: "tab inline comment after unquoted value",
		dotenv: "KEY=value\t# K=V",
		want:   EnvStore{"KEY": "value"},
	})

	run(t, nil, cfg, testCase{name: "inline comment after unquoted value with internal spaces",
		dotenv: "KEY=the big cat # K=V",
		want:   EnvStore{"KEY": "the big cat"},
	})

	run(t, nil, cfg, testCase{name: "space hash after unquoted value",
		dotenv: "KEY=value #",
		want:   EnvStore{"KEY": "value"},
	})

	run(t, nil, cfg, testCase{name: "space hash alphanumeric after unquoted value with internal space",
		dotenv: "KEY=word1 word2 #COMMENT1",
		want:   EnvStore{"KEY": "word1 word2"},
	})

	run(t, nil, cfg, testCase{name: "tab hash after unquoted value",
		dotenv: "KEY=value\t#",
		want:   EnvStore{"KEY": "value"},
	})

	run(t, nil, cfg, testCase{name: "comment after single quoted value no space",
		dotenv: "KEY='value'# K=V",
		want:   EnvStore{"KEY": "value"},
	})

	run(t, nil, cfg, testCase{name: "comment after single quoted value with space",
		dotenv: "KEY='value' # K=V",
		want:   EnvStore{"KEY": "value"},
	})

	run(t, nil, cfg, testCase{name: "comment after single quoted value with tab",
		dotenv: "KEY='value'\t# K=V",
		want:   EnvStore{"KEY": "value"},
	})

	run(t, nil, cfg, testCase{name: "comment after double quoted value no space",
		dotenv: "KEY=\"value\"# K=V",
		want:   EnvStore{"KEY": "value"},
	})

	run(t, nil, cfg, testCase{name: "comment after double quoted value with space",
		dotenv: "KEY=\"value\" # K=V",
		want:   EnvStore{"KEY": "value"},
	})

	run(t, nil, cfg, testCase{name: "comment after double quoted value with tab",
		dotenv: "KEY=\"value\"\t# K=V",
		want:   EnvStore{"KEY": "value"},
	})
}

func TestNonCommentCases(t *testing.T) {
	cfg := NewParseConfig().WithRecommendedDefaults()

	run(t, nil, cfg, testCase{name: "hash as unquoted value is not a comment",
		dotenv: "KEY=#",
		want:   EnvStore{"KEY": "#"},
	})

	run(t, nil, cfg, testCase{name: "unquoted value with hash without preceding space is not a comment",
		dotenv: "KEY=value#comment",
		want:   EnvStore{"KEY": "value#comment"},
	})

	run(t, nil, cfg, testCase{name: "unquoted value with multiple hashes without preceding whitespace is not a comment",
		dotenv: "KEY=#value#example# # K=V",
		want:   EnvStore{"KEY": "#value#example#"},
	})

	run(t, nil, cfg, testCase{name: "hash inside single quoted value is literal even if preceded by space",
		dotenv: "KEY='value # not comment'",
		want:   EnvStore{"KEY": "value # not comment"},
	})

	run(t, nil, cfg, testCase{name: "hash inside double quoted value is literal even if preceded by space",
		dotenv: `KEY="value # not comment"`,
		want:   EnvStore{"KEY": "value # not comment"},
	})
}

// ---------------------------------------------------------------------------
// Test Dollar Sign
// ---------------------------------------------------------------------------

func TestDollarSignLiteralInUnquotedValue(t *testing.T) {
	cfg := NewParseConfig().WithRecommendedDefaults()

	run(t, nil, cfg, testCase{name: "dollar sign in unquoted value is literal",
		dotenv: "KEY=$NOTAVAR",
		want:   EnvStore{"KEY": "$NOTAVAR"},
	})

	run(t, nil, cfg, testCase{name: "dollar sign with curly braces in unquoted value is literal",
		dotenv: "KEY=${NOTAVAR}",
		want:   EnvStore{"KEY": "${NOTAVAR}"},
	})

	run(t, nil, cfg, testCase{name: "unclosed variable substitution syntax in unquoted value is literal",
		dotenv: "KEY=${NOTAVAR",
		want:   EnvStore{"KEY": "${NOTAVAR"},
	})

	run(t, nil, cfg, testCase{name: "dollar sign with parens in unquoted value is literal",
		dotenv: "KEY=$(NOTACMD)",
		want:   EnvStore{"KEY": "$(NOTACMD)"},
	})

	run(t, nil, cfg, testCase{name: "unclosed command substitution syntax in unquoted value is literal",
		dotenv: "KEY=$(NOTACMD",
		want:   EnvStore{"KEY": "$(NOTACMD"},
	})
}
func TestDollarSignLiteralInSingleQuotedValue(t *testing.T) {
	cfg := NewParseConfig().WithRecommendedDefaults()

	run(t, nil, cfg, testCase{name: "dollar sign in single quoted value is literal",
		dotenv: "KEY='$NOTAVAR'",
		want:   EnvStore{"KEY": "$NOTAVAR"},
	})

	run(t, nil, cfg, testCase{name: "dollar sign with curly braces in single quoted value is literal",
		dotenv: "KEY='${NOTAVAR}'",
		want:   EnvStore{"KEY": "${NOTAVAR}"},
	})

	run(t, nil, cfg, testCase{name: "unclosed variable substitution syntax in single quoted value is literal",
		dotenv: "KEY='${NOTAVAR'",
		want:   EnvStore{"KEY": "${NOTAVAR"},
	})

	run(t, nil, cfg, testCase{name: "dollar sign with parens in single quoted value is literal",
		dotenv: "KEY='$(NOTACMD)'",
		want:   EnvStore{"KEY": "$(NOTACMD)"},
	})

	run(t, nil, cfg, testCase{name: "unclosed command substitution syntax in single quoted value is literal",
		dotenv: "KEY='$(NOTACMD'",
		want:   EnvStore{"KEY": "$(NOTACMD"},
	})
}
func TestDollarSignLiteralInDoubleQuotedValue(t *testing.T) {
	cfg := NewParseConfig().WithRecommendedDefaults()

	run(t, nil, cfg, testCase{name: "dollar sign in double quoted value is literal",
		dotenv: `KEY="$NOTAVAR"`,
		want:   EnvStore{"KEY": "$NOTAVAR"},
	})

	run(t, nil, cfg, testCase{name: "dollar sign with curly braces in double quoted value is literal",
		dotenv: `KEY="${NOTAVAR}"`,
		want:   EnvStore{"KEY": "${NOTAVAR}"},
	})

	run(t, nil, cfg, testCase{name: "unclosed variable substitution syntax in double quoted value is literal",
		dotenv: `KEY="${NOTAVAR"`,
		want:   EnvStore{"KEY": "${NOTAVAR"},
	})

	run(t, nil, cfg, testCase{name: "dollar sign with parens in double quoted value is literal",
		dotenv: `KEY="$(NOTACMD)"`,
		want:   EnvStore{"KEY": "$(NOTACMD)"},
	})

	run(t, nil, cfg, testCase{name: "unclosed command substitution syntax in double quoted value is literal",
		dotenv: `KEY="$(NOTACMD"`,
		want:   EnvStore{"KEY": "$(NOTACMD"},
	})
}

// ---------------------------------------------------------------------------
// Test Edge Cases
// ---------------------------------------------------------------------------

func TestEdgeCases(t *testing.T) {
	cfg := NewParseConfig().WithRecommendedDefaults()

	run(t, nil, cfg, testCase{name: "multiple eq",
		dotenv: "KEY==Value=With=Equals=",
		want:   EnvStore{"KEY": "=Value=With=Equals="},
	})

	run(t, nil, cfg, testCase{name: "key space eq eq space value",
		dotenv: "KEY == Value",
		want:   EnvStore{"KEY": "= Value"},
	})

	run(t, nil, cfg, testCase{name: "back ticked values do not act like quotes",
		dotenv: "KEY=`value # comment`",
		want:   EnvStore{"KEY": "`value"},
	})

	run(t, nil, cfg, testCase{name: "unicode characters in unquoted value",
		dotenv: "KEY=héllo wörld",
		want:   EnvStore{"KEY": "héllo wörld"},
	})

	run(t, nil, cfg, testCase{name: "unicode characters in single quoted value",
		dotenv: "KEY='héllo wörld'",
		want:   EnvStore{"KEY": "héllo wörld"},
	})

	run(t, nil, cfg, testCase{name: "unicode characters in double-quoted value",
		dotenv: "KEY=\"naïve café\"",
		want:   EnvStore{"KEY": "naïve café"},
	})
}

func TestBOM(t *testing.T) {
	cfg := NewParseConfig().WithRecommendedDefaults()

	run(t, nil, cfg, testCase{name: "BOM at start of file followed by key is ignored",
		dotenv: "\uFEFFKEY=value",
		want:   EnvStore{"KEY": "value"},
	})

	run(t, nil, cfg, testCase{name: "BOM at start of file followed by export is ignored",
		dotenv: "\uFEFFexport KEY=value",
		want:   EnvStore{"KEY": "value"},
	})

	run(t, nil, cfg, testCase{name: "BOM at start of file followed by newline is ignored",
		dotenv: "\uFEFF\nKEY=value",
		want:   EnvStore{"KEY": "value"},
	})

	run(t, nil, cfg, testCase{name: "BOM can be part of an unquoted, single-quoted, or double-quoted value",
		dotenv: "KEY=\uFEFFunq\nKEY2='\uFEFFsq'\nKEY3=\"\uFEFFdq\"",
		want:   EnvStore{"KEY": "\uFEFFunq", "KEY2": "\uFEFFsq", "KEY3": "\uFEFFdq"},
	})

	run(t, nil, cfg, testCase{name: "BOM at end of file on its own line is an error",
		dotenv:  "KEY=value\n\uFEFF",
		wantErr: true,
	})

	run(t, nil, cfg, testCase{name: "BOM before non-first key is an error",
		dotenv:  "KEY1=value1\n\uFEFFKEY2=value2",
		wantErr: true,
	})
}

// ---------------------------------------------------------------------------
// Test runner
// ---------------------------------------------------------------------------

func run(t *testing.T, store EnvStore, cfg *ParseConfig, testCase testCase) {
	t.Helper()

	t.Run(testCase.name, func(t *testing.T) {
		t.Helper()

		// LLM NOTE: you can change the implementation of the parser as needed

		if store == nil {
			store = NewEnvStore()
		}

		err := store.SetFromString(testCase.dotenv, testCase.name, cfg)

		if err != nil {
			if !testCase.wantErr {
				t.Errorf("unexpected error: %v", err)
			}
			return
		}

		if testCase.wantErr {
			t.Errorf("expected error but got nil")
			return
		}

		if !maps.Equal(store, testCase.want) {
			t.Errorf("\ngot  %q\nwant %q", store, testCase.want)
		}
	})
}
