# strict-dotenv

`strict-dotenv` is a zero-dependency Go library for parsing dotenv files with a [clear set of strict rules](#strict-rules).

> [!NOTE]
> `strict-dotenv` is a new library - anticipate possible breaking changes before `v1.0.0`.

## Overview

`strict-dotenv` emphasises correctness, explicitness, and configurability:

- strictly-defined (and highly portable) keys
- configurable duplicate-key, unescape, and newline-normalization behavior
- unquoted, single-quoted, and double-quoted values
- multi-line double-quoted value support
- parsing from files, strings, and `io.Reader`
- an `EnvStore` type for layering dotenv data with `os.Environ` or other sources
- helpful error messages with source name and line number
- minimization of unexpected/undocumented behavior

By default, the library is opinionated in a small number of places:

- repeated keys keep the first value
- in double-quoted values, `\\`, `\"`, `\n`, `\r`, and `\t` are unescaped
- in double-quoted values, `CRLF` and `CR` are normalized to `LF`

Those defaults can be changed globally and, when needed, per key.

## Background

The lexer and parser are intentionally simple and explicit. The lexer design is heavily inspired by Go's [`text/template/parse`](https://github.com/golang/go/blob/master/src/text/template/parse/lex.go), which was itself popularized by Rob Pike's talk [Lexical Scanning in Go](https://www.youtube.com/watch?v=HxaD_trXwRE) and the accompanying [slides](https://go.dev/talks/2011/lex.slide).

At CounselHQ, we use `strict-dotenv` alongside [1Password Environments](https://developer.1password.com/docs/environments) for development and test secrets.

## Installation

```sh
go get github.com/counselhq/strict-dotenv
```

## Basic Usage

If you just want to parse a `.env` file into a store and read values back, this is the shortest path:

```go
package main

import (
	"log"

	strictdotenv "github.com/counselhq/strict-dotenv"
)

func main() {
	store := strictdotenv.NewEnvStore()
	cfg := strictdotenv.NewParseConfig().WithRecommendedDefaults()

	if err := store.SetFromRequiredDotEnv(".env", cfg); err != nil {
		log.Fatal(err)
	}

	databaseURL, err := store.GetRequired("DATABASE_URL")
	if err != nil {
		log.Fatal(err)
	}

	log.Println(databaseURL)
}
```

The parse entry points take a `*ParseConfig`. Use `cfg := strictdotenv.NewParseConfig().WithRecommendedDefaults()` for the library defaults. Passing `nil` means an all-zero config; nothing falls back to the library defaults automatically. If your dotenv file is optional, use `SetFromOptionalDotEnv`; if startup should fail when it is absent, use `SetFromRequiredDotEnv`.

## Store Parse Methods

All parse methods write into the receiver `EnvStore`. Parse failures return an error, and the file-based methods differ on missing-file handling as documented below.

| Method                              | Use when                                       | Notes                                                                       |
| ----------------------------------- | ---------------------------------------------- | --------------------------------------------------------------------------- |
| `store.SetFromOptionalDotEnv(path, cfg)` | You have a dotenv file on disk or named pipe, but it is optional | Missing files are ignored; `path` is used in parser error messages; `nil` cfg means all-zero options |
| `store.SetFromRequiredDotEnv(path, cfg)` | You have a dotenv file on disk or named pipe and it must exist | Missing files return `ErrMissingDotEnv`; `path` is used in parser error messages; `nil` cfg means all-zero options |
| `store.SetFromString(s, name, cfg)`      | You already have the dotenv contents in memory                   | `name` is used to identify source in error messages (default `"string"`); `nil` cfg means all-zero options |
| `store.SetFromReader(r, name, cfg)`      | You want to parse from an `io.Reader`                            | `name` is used to identify source in error messages (default `"io.Reader"`); `nil` cfg means all-zero options |

## Parse Configuration

Use a `ParseConfig` when you want explicit control over the base settings or per-key overrides:

- `NewParseConfig()` returns an empty config without applying defaults.
- `WithRecommendedDefaults()` copies the library's current recommended defaults into the base config.
- `WithBaseOptions(...)` applies partial overrides to the current base config.
- `WithKeyOptions(...)` applies overrides for an exact key name.

### Starting points

| Starting point             | API                                                     | Meaning                                                     |
| -------------------------- | ------------------------------------------------------- | ----------------------------------------------------------- |
| Recommended defaults       | `strictdotenv.NewParseConfig().WithRecommendedDefaults()` | Use the library defaults explicitly                         |
| Explicit zero-value config | `strictdotenv.NewParseConfig()`                         | Every option starts at `false` until you opt in to behavior |
| Implicit zero-value config | `nil`                                                   | Equivalent to a zero-value `ParseConfig`                    |

### Base vs key-specific settings

| Scope                 | API                                                                | Use for                                                            |
| --------------------- | ------------------------------------------------------------------ | ------------------------------------------------------------------ |
| Base settings         | `cfg.WithBaseOptions(&strictdotenv.CustomParseOptions{...})`       | Changing the current base config                                   |
| Key-specific settings | `cfg.WithKeyOptions("KEY", &strictdotenv.CustomParseOptions{...})` | Making one key behave differently while inheriting the base config |

Unset fields inherit. That means a key-specific override only needs to mention the fields it wants to change.

```go
cfg := strictdotenv.NewParseConfig().
	WithRecommendedDefaults().
	WithBaseOptions(&strictdotenv.CustomParseOptions{
		Overwrite:          strictdotenv.BoolPtr(true),
		UnescapeBackslashN: strictdotenv.BoolPtr(true),
	}).
	WithKeyOptions("PRIVATE_KEY", &strictdotenv.CustomParseOptions{
		UnescapeBackslashN: strictdotenv.BoolPtr(false),
		TransformCRLFToLF:  strictdotenv.BoolPtr(false),
		TransformCRToLF:    strictdotenv.BoolPtr(false),
	})
```

In that example:

- keys use last-definition-wins semantics because `Overwrite` is `true`
- most double-quoted values unescape `\n`
- `PRIVATE_KEY` preserves `\n`, `CRLF`, and `CR` literally while still inheriting the base `Overwrite` setting

If you want to build a config from all-zero settings instead, skip `WithRecommendedDefaults()` and call `WithBaseOptions(...)` directly.

## Parse Options Reference

All options are fields on `CustomParseOptions`. The `BoolPtr` helper exists to make those fields easy to populate.

`Overwrite` applies to all kinds of key-value pairs. All other options apply only to double-quoted values.

| Field                          | Default | Applies to           | Meaning                                                                           |
| ------------------------------ | ------- | -------------------- | --------------------------------------------------------------------------------- |
| `Overwrite`                    | `false` | all key-value pairs  | When `true`, later values replace earlier ones for the same key                   |
| `UnescapeBackslashBackslash`   | `true`  | double-quoted values | `\\` becomes `\`                                                                  |
| `UnescapeBackslashDoubleQuote` | `true`  | double-quoted values | `\"` becomes `"`                                                                  |
| `UnescapeBackslashSingleQuote` | `false` | double-quoted values | `\'` becomes `'`                                                                  |
| `UnescapeBackslashA`           | `false` | double-quoted values | `\a` becomes the alert/bell character                                             |
| `UnescapeBackslashB`           | `false` | double-quoted values | `\b` becomes the backspace character                                              |
| `UnescapeBackslashF`           | `false` | double-quoted values | `\f` becomes the form feed character                                              |
| `UnescapeBackslashN`           | `true`  | double-quoted values | `\n` becomes line feed                                                            |
| `UnescapeBackslashR`           | `true`  | double-quoted values | `\r` becomes carriage return before newline transforms are applied                |
| `UnescapeBackslashT`           | `true`  | double-quoted values | `\t` becomes tab                                                                  |
| `UnescapeBackslashV`           | `false` | double-quoted values | `\v` becomes vertical tab                                                         |
| `TransformCRLFToLF`            | `true`  | double-quoted values | Literal or unescaped `CRLF` is normalized to `LF` before standalone `CR` handling |
| `TransformCRToLF`              | `true`  | double-quoted values | Remaining literal or unescaped `CR` is normalized to `LF`                         |

A few important points:

- All unescaping happens BEFORE newline normalization, so unescaping can produce newlines that are then normalized according to the `Transform*` settings.
- If `TransformCRLFToLF` is `false` and `TransformCRToLF` is `true`, a literal `CRLF` becomes two `LF` bytes.
- If both `TransformCRLFToLF` and `TransformCRToLF` are `false`, literal `CRLF` and literal `CR` are preserved.

## Working With `EnvStore`

`EnvStore` is a `map[string]string` with a few convenience methods layered on top. You can use it like a normal map, or you can use the helper methods when you want map-style lookups, required-key checks, merges, or `os.Environ` integration.

### Common store methods

| Method                                             | Purpose                                                                 | Notes                                                                                     |
| -------------------------------------------------- | ----------------------------------------------------------------------- | ----------------------------------------------------------------------------------------- |
| `NewEnvStore()`                                    | Create an empty store                                                   | Returns a writable `map[string]string`                                                    |
| `Get(key)`                                         | Read a value with map-style presence reporting                          | Returns `(value, ok)` so empty strings can be distinguished from missing keys             |
| `GetRequired(key)`                                      | Read a value that must exist                                            | Missing keys return `ErrMissingRequiredKey`                                                   |
| `Set(key, value, overwrite)`                            | Write one value                                                         | `overwrite=false` keeps an existing value                                                     |
| `Merge(other, overwrite)`                               | Merge another `EnvStore` into this one                                  | Same overwrite semantics as `Set`                                                              |
| `ProcessValue(key, cfg)`                                | Reprocess one existing value using the double-quoted transform pipeline | `nil` cfg uses all-zero options; base plus key-specific config resolution is applied for that key |
| `ProcessValues(cfg)`                                    | Reprocess every stored value using parser-style config resolution       | Applies base plus per-key options; leaves the store unchanged on error                        |
| `SetFromOptionalDotEnv(path, cfg)`                      | Parse an optional dotenv file into the store                            | Missing files are ignored; `path` is used in parser error messages                            |
| `SetFromRequiredDotEnv(path, cfg)`                      | Parse a required dotenv file into the store                             | Missing files return `ErrMissingDotEnv`; `path` is used in parser error messages             |
| `SetFromString(s, name, cfg)`                           | Parse dotenv contents from a string into the store                      | If `name` is empty, errors use `"string"` as source name                                      |
| `SetFromReader(r, name, cfg)`                           | Parse dotenv contents from an `io.Reader` into the store                | If `name` is empty, errors use `"io.Reader"` as source name                                   |
| `SetFromOsEnviron(allowlist, denylist, overwrite)`      | Import from the current process environment                             | `allowlist` and `denylist` are `map[string]struct{}`                                          |
| `LoadIntoOsEnviron(allowlist, denylist, overwrite)`     | Export store values into the current process environment                | Existing OS values are preserved unless `overwrite` is `true`                                 |
| `FilterKeys(allowlist, denylist)`                       | Remove keys that fail the combined filters                              | `nil` allowlist keeps all; `nil` denylist removes none; denylist wins on overlap              |

### Reprocess existing store values

`ProcessValue` and `ProcessValues` let you apply the same double-quoted unescape and newline-normalization logic to values that are already in an `EnvStore`.

- `ProcessValue(key, cfg)` treats one stored value as if it were the raw contents between double quotes in a dotenv file and resolves that key against the supplied `ParseConfig`.
- `ProcessValues(cfg)` does the same for every key, using `ParseConfig` base settings and key-specific overrides exactly the way the parser resolves them.

`Overwrite` is ignored. It is still part of `ParseConfig` so you can reuse the same config you already use while parsing dotenv files:

```go
cfg := strictdotenv.NewParseConfig().
	WithRecommendedDefaults().
	WithBaseOptions(&strictdotenv.CustomParseOptions{
		UnescapeBackslashN: strictdotenv.BoolPtr(false),
	}).
	WithKeyOptions("PRIVATE_KEY", &strictdotenv.CustomParseOptions{
		UnescapeBackslashN: strictdotenv.BoolPtr(true),
	})

if err := store.ProcessValue("PRIVATE_KEY", cfg); err != nil {
	// handle error
}
```

`ProcessValues` is useful when values came from another source, such as `os.Environ`, and you want parser-style processing after they are already in the store:

```go
store := strictdotenv.NewEnvStore()
store.SetFromOsEnviron(nil, nil, false)

cfg := strictdotenv.NewParseConfig().
	WithRecommendedDefaults().
	WithKeyOptions("PRIVATE_KEY", &strictdotenv.CustomParseOptions{
		UnescapeBackslashN: strictdotenv.BoolPtr(true),
	})

if err := store.ProcessValues(cfg); err != nil {
	log.Fatal(err)
}
```

### Example: layer process env and dotenv data

This pattern is useful when you want existing process environment variables to overwrite any dotenv file values:

```go
package main

import (
	"log"

	strictdotenv "github.com/counselhq/strict-dotenv"
)

func main() {
	store := strictdotenv.NewEnvStore()
	cfg := strictdotenv.NewParseConfig().WithRecommendedDefaults()

	store.SetFromOsEnviron(nil, nil, false)

	if err := store.SetFromOptionalDotEnv(".env", cfg); err != nil {
		log.Fatal(err)
	}

	// Do something with the store:
	databaseURL, err := store.GetRequired("DATABASE_URL")
	if err != nil {
		log.Fatal(err)
	}

	// Maybe populate your own config struct to use elsewhere in your app:
	appConfig := AppConfig{
		DatabaseURL: databaseURL,
		// ...
	}
}
```

### Example: export store values into the process environment

This is useful when other parts of your application (or libraries you depend on) read configuration from `os.Getenv` directly, or when you want dotenv values available to subprocesses:

```go
package main

import (
	"log"

	strictdotenv "github.com/counselhq/strict-dotenv"
)

func main() {
	store := strictdotenv.NewEnvStore()
	cfg := strictdotenv.NewParseConfig().WithRecommendedDefaults()

	if err := store.SetFromRequiredDotEnv(".env", cfg); err != nil {
		log.Fatal(err)
	}

	// Write dotenv values into the current process environment.
	// Existing environment variables are preserved (overwrite=false).
	store.LoadIntoOsEnviron(nil, nil, false)
}
```

## Error Reporting

Parse errors include the source name and line number. For example:

```text
config.env:12: expected closing double quote
```

For `SetFromString` and `SetFromReader`, the `name` argument is what appears in the error prefix. `SetFromRequiredDotEnv` returns a wrapped `ErrMissingDotEnv` when the file does not exist; `SetFromOptionalDotEnv` treats that case as a no-op.

## Parsing Rules

The rules below describe the parser's current behavior. They are intentionally specific and are backed by the test suite.

## Whitespace and Newlines (Outside of values)

- Whitespace means only ASCII spaces (`0x20`) and tabs (`0x09`)
- Multiple consecutive whitespace characters are functionally equivalent to a single space
- Newlines means `LF` (`\n` or `0x0A`), `CR` (`\r` or `0x0D`), and `CRLF` (`\r\n` or `0x0D0A`)
- Multiple consecutive newlines are functionally equivalent to a single newline
- One or more newlines are treated as line terminators and separate key-value pairs
- Mixed newline styles within the same file are permitted
- An empty file or a file containing only whitespace and/or newlines is valid
- Lines containing only whitespace characters are treated the same as empty lines
- Newlines are permitted before and after any valid key-value pair

## Keys

- Keys must only contain ASCII alphanumeric characters (`a-z`, `A-Z`, `0-9`) and underscores (`_`)
- Keys must not start with a digit
- Keys must have a length of at least 1; an empty key (leading `=`) is an error
- Keys are case-sensitive: lowercase, uppercase, and mixed-case keys are all valid
- Quoted or back-ticked keys are not supported
- Leading and trailing whitespace around the key is stripped and ignored
- A valid key must be followed by zero or more whitespace characters and then an assignment operator

## Assignment Operator

- Only `=` is supported as an assignment operator; it must be present between every key-value pair
- A valid key without an accompanying assignment operator before a newline or `EOF` is an error
- An assignment operator may be surrounded by zero or more whitespace characters on either side; any such whitespace characters are stripped and ignored
- If multiple equal signs appear consecutively after a valid key (whether separated by whitespace or not), the first is treated as the assignment operator and the second becomes the first character of the unquoted value. For example, `KEY==value` produces a key of `KEY` and a value of `=value`, while `KEY = = =value` produces a key of `KEY` and a value of `= =value`.

## Values Generally

- Values are one of unquoted, single-quoted, or double-quoted
- The first non-whitespace character after the assignment operator determines the quoting mode: a single quote denotes a single-quoted value, a double quote denotes a double-quoted value, and anything else denotes an unquoted value; if the first non-whitespace character after the assignment operator is a newline or `EOF`, the unquoted value is the empty string
- Backticks are not quoting characters; if a backtick is the first non-whitespace character after the assignment operator, it is treated as a literal character in an unquoted value
- Dollar signs are always treated as literals in all quoting modes; there is no variable or command expansion/substitution

## Unquoted Values

- Surrounding whitespace is always stripped from unquoted values; an unquoted value never starts or ends with whitespace characters
- Escape sequences in unquoted values are never unescaped; everything is treated as a literal, including single quotes, double quotes, backticks, and backslashes
- Control characters other than newlines are preserved in unquoted values; for example, a tab character in the middle of an unquoted value is preserved as a literal tab character, not converted to a space or escape sequence `\t`
- The unquoted value ends at the next newline, comment, or `EOF`

## Single-Quoted Values

- The value consists of all characters between the opening (left) and closing (right) single quote on the same line
- Single-quoted values do not support unescaping any escape sequences; everything is treated as a literal character that is part of the value, including double quotes, backticks, and backslashes
- The first single quote character after the assignment operator is always treated as the opening quote and is not part of the value
- The second single quote character on the same line is always treated as the closing quote and is not part of the value
- A single-quoted value cannot contain a single quote character, whether or not preceded by a backslash
- Newlines are not permitted inside a single-quoted value; a newline before the closing single quote is an error
- Only whitespace characters, a newline, a comment, or `EOF` may follow the closing single quote; anything else (including another single quote) is an error

## Double-Quoted Values

- The value consists of all characters between the opening (left) and closing (right) double quote
- Newlines are permitted inside double-quoted values (multi-line values); parsing continues until the closing quote
- ParseOptions are applied in this order: unescaping, then `TransformCRLFToLF`, then `TransformCRToLF`
- By default, `\"`, `\\`, `\n`, `\r`, and `\t` are unescaped; unescaping `\'`, `\a`, `\b`, `\f`, and `\v` can be enabled via ParseOptions. When a recognized `Unescape*` option is disabled, that backslash escape sequence is preserved literally.
- By default, both `TransformCRLFToLF` and `TransformCRToLF` are enabled, so each `CRLF` and `CR` is normalized to `LF` after unescaping
- Escape sequences without a corresponding ParseOptions switch (for example `\$`, `\x41`, `\u0041`, `\0`) are preserved as literals and not unescaped; for example, `\x41` is treated as the literal characters `\`, `x`, `4`, and `1`, not the single character `A`
- Only whitespace characters, a newline, a comment, or `EOF` may follow the closing double quote; anything else (including another double quote) is an error

## Comments

A comment starts with a `#` and meets one of the following conditions:

1. The `#` is the first non-whitespace character on a line (line comment)
2. The `#` immediately follows an unquoted value (including an empty value) and one or more whitespace characters (inline comment); a hash that is not preceded by whitespace can never start an inline comment after an unquoted value because the hash is treated as a literal character that is part of the unquoted value, so `KEY=value#notacomment` produces a key of `KEY` and a value of `value#notacomment`
3. The `#` appears immediately after the closing single quote of a valid single-quoted value, optionally after spaces and/or tabs
4. The `#` appears immediately after the closing double quote of a valid double-quoted value, optionally after spaces and/or tabs

- Other than the cases above, a `#` is treated as a literal character
- Once a comment is started, it continues until the next newline or `EOF`
- Comments are ignored and are not part of any key or value; they are functionally equivalent to whitespace for parsing purposes

## The `export` Keyword

On any line, an otherwise valid key-value pair may optionally be immediately preceded by exactly the following:

1. Zero or more whitespace characters; then
2. The keyword `export` (case sensitive, lowercase only); then
3. One or more whitespace characters.

If the above pattern is met, the `export` keyword and its surrounding whitespace is stripped and ignored.

Examples and non-examples:

- `export KEY=value` produces key `KEY`
- `export export=value` produces key `export`
- `exportKEY=value` does not use special `export` handling; it produces key `exportKEY`
- `export = value` does not use special `export` handling; `export` itself is the key

## Repeated Keys

- If `Overwrite` is `false` (the default), a repeated key is silently ignored; the first definition wins
- If `Overwrite` is `true`, a repeated key overwrites the previously stored value; the last definition wins
- `Overwrite` applies to every key-value pair, not just double-quoted values

## Character Encoding

- Input must be valid UTF-8; invalid UTF-8 input results in an error
- A UTF-8 byte order mark (`BOM`) is stripped and ignored only when it appears at the very start of the file
- A `BOM` that appears anywhere else outside a value is invalid input and results in an error
- A `BOM` that appears inside an unquoted, single-quoted, or double-quoted value is preserved as part of the value

## Current Unimplemented Features

The library does not support these features today:

- YAML-style `KEY: value` assignment
- variable expansion
- command substitution
- permissive keys
- shell-style parsing semantics beyond the explicit rules above

We are open to considering these features (or others) in the future if there is demand, but for now, the library is focused on a narrow, explicit grammar. If there is demand for variable expansion or command substitution, we would likely want to limit that behavior to BOTH (a) a user-defined list of keys AND (b) a user-defined list of variables/commands. Security and minimization of surprises would be top priorities for any such feature.

## Supported Go Versions

We will support the current Go major version and the previous major version. For example, if the current Go version is `1.26`, we will support `1.26` and `1.25`.

## License

[MIT LICENSE](LICENSE).
