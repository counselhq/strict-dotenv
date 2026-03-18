package strictdotenv

// ---------------------------------------------------------------------------
// Types
//
// Public:
//   - Config  - top-level configuration passed to parse functions
//   - Options - partial option set used to build a config
//
// Internal:
//   - resolvedOptions - concrete bool option set derived from [Options]
// ---------------------------------------------------------------------------

// Config holds global parse options plus exact-key overrides.
//
// All options default to false (disabled). Use [Config.ApplyGlobalOptions] to
// set a baseline that applies to every key, and [Config.ApplyKeyOptions] to add
// per-key overrides that layer on top of that global baseline.
//
// Option scoping for EnvStore.ProcessValue and EnvStore.ProcessValues:
//   - [Options.Overwrite] is ignored
//   - All other options apply through the same double-quoted value processing
//     pipeline the parser uses
//
// Option scoping for EnvStore.SetFrom* methods:
//   - [Options.Overwrite] applies to unquoted, single-quoted, and double-quoted values
//   - All other options apply only to double-quoted values
//
// A nil *Config is valid and equivalent to a zero-value Config (all options
// disabled).
//
// Usage:
//
//	cfg := new(Config)
//	cfg.ApplyGlobalOptions(Options{Overwrite: new(true)})
//	cfg.ApplyKeyOptions("SECRET", Options{UnescapeBackslashN: new(false)})
type Config struct {
	globalOptions Options
	keyOptions    map[string]Options
}

// Options is the partial, pointer-field option set passed to
// [Config.ApplyGlobalOptions] and [Config.ApplyKeyOptions].
//
// Every field is a pointer so that only the fields you explicitly set are
// applied; nil fields leave the existing value in the config unchanged. This
// allows incremental, non-destructive configuration, for example enabling one
// option without resetting unrelated ones.
//
// The Unescape* fields control which backslash sequences are expanded into
// their intended character or control action inside a value. The Transform*
// fields run after unescaping and normalize line endings in the resulting
// string.
type Options struct {
	// Overwrite controls whether an existing key-value pair in the store is
	// overwritten when the same key appears again during parsing.
	Overwrite *bool

	UnescapeBackslashBackslash   *bool // \\ -> \
	UnescapeBackslashDoubleQuote *bool // \" -> "
	UnescapeBackslashSingleQuote *bool // \' -> '
	UnescapeBackslashA           *bool // \a -> alert/bell             (U+0007)
	UnescapeBackslashB           *bool // \b -> backspace              (U+0008)
	UnescapeBackslashF           *bool // \f -> form feed              (U+000C)
	UnescapeBackslashN           *bool // \n -> line feed (LF)         (U+000A)
	UnescapeBackslashR           *bool // \r -> carriage return (CR)   (U+000D)
	UnescapeBackslashT           *bool // \t -> horizontal tab         (U+0009)
	UnescapeBackslashV           *bool // \v -> vertical tab           (U+000B)

	// Transform* fields normalize line endings after unescaping.
	TransformCRLFToLF *bool // CRLF -> LF (runs first)
	TransformCRToLF   *bool // CR   -> LF (runs after TransformCRLFToLF)
}

// resolvedOptions is the concrete, non-pointer form of [Options]
// used internally during parsing and value processing.
type resolvedOptions struct {
	Overwrite                    bool
	UnescapeBackslashBackslash   bool
	UnescapeBackslashDoubleQuote bool
	UnescapeBackslashSingleQuote bool
	UnescapeBackslashA           bool
	UnescapeBackslashB           bool
	UnescapeBackslashF           bool
	UnescapeBackslashN           bool
	UnescapeBackslashR           bool
	UnescapeBackslashT           bool
	UnescapeBackslashV           bool
	TransformCRLFToLF            bool
	TransformCRToLF              bool
}

// ---------------------------------------------------------------------------
// Public API for building a Config:
//   - ApplyGlobalOptions - set or update the baseline options for all keys
//   - ApplyKeyOptions - set or update overrides for a single named key
// ---------------------------------------------------------------------------

// ApplyGlobalOptions merges overrides into the global baseline.
//
// Only non-nil fields in overrides are applied; nil fields leave the current
// global value unchanged.
func (c *Config) ApplyGlobalOptions(overrides Options) {
	c.globalOptions = mergeOptions(c.globalOptions, overrides)
}

// ApplyKeyOptions merges overrides into the per-key options for key.
//
// Per-key options inherit from the current global baseline: the effective value
// for a field is the global value unless the per-key setting explicitly
// overrides it. Calling ApplyGlobalOptions after ApplyKeyOptions still affects
// keys that did not override the changed fields.
//
// Only non-nil fields in overrides are applied. Passing a zero-value Options
// registers key without changing any of its options, which is useful when you
// want the key to inherit all current and future global settings explicitly.
func (c *Config) ApplyKeyOptions(key string, overrides Options) {
	if c.keyOptions == nil {
		c.keyOptions = make(map[string]Options)
	}

	c.keyOptions[key] = mergeOptions(c.keyOptions[key], overrides)
}

// ---------------------------------------------------------------------------
// Internal helpers for building/resolving a Config
// ---------------------------------------------------------------------------

// mergeOptions copies overrides's non-nil fields onto base and returns the
// result. base is passed by value so the original is never mutated. Applied
// pointer fields are deep-copied so the stored Options owns its memory.
func mergeOptions(base, overrides Options) Options {
	if overrides.Overwrite != nil {
		base.Overwrite = new(*overrides.Overwrite)
	}
	if overrides.UnescapeBackslashBackslash != nil {
		base.UnescapeBackslashBackslash = new(*overrides.UnescapeBackslashBackslash)
	}
	if overrides.UnescapeBackslashDoubleQuote != nil {
		base.UnescapeBackslashDoubleQuote = new(*overrides.UnescapeBackslashDoubleQuote)
	}
	if overrides.UnescapeBackslashSingleQuote != nil {
		base.UnescapeBackslashSingleQuote = new(*overrides.UnescapeBackslashSingleQuote)
	}
	if overrides.UnescapeBackslashA != nil {
		base.UnescapeBackslashA = new(*overrides.UnescapeBackslashA)
	}
	if overrides.UnescapeBackslashB != nil {
		base.UnescapeBackslashB = new(*overrides.UnescapeBackslashB)
	}
	if overrides.UnescapeBackslashF != nil {
		base.UnescapeBackslashF = new(*overrides.UnescapeBackslashF)
	}
	if overrides.UnescapeBackslashN != nil {
		base.UnescapeBackslashN = new(*overrides.UnescapeBackslashN)
	}
	if overrides.UnescapeBackslashR != nil {
		base.UnescapeBackslashR = new(*overrides.UnescapeBackslashR)
	}
	if overrides.UnescapeBackslashT != nil {
		base.UnescapeBackslashT = new(*overrides.UnescapeBackslashT)
	}
	if overrides.UnescapeBackslashV != nil {
		base.UnescapeBackslashV = new(*overrides.UnescapeBackslashV)
	}
	if overrides.TransformCRLFToLF != nil {
		base.TransformCRLFToLF = new(*overrides.TransformCRLFToLF)
	}
	if overrides.TransformCRToLF != nil {
		base.TransformCRToLF = new(*overrides.TransformCRToLF)
	}
	return base
}

// resolveOptions returns the effective concrete options for key.
//
// Resolution order:
//  1. Start with the global options.
//  2. If key has a per-key override, layer it on top.
//
// A nil cfg is safe and returns an all-zero resolvedOptions.
func resolveOptions(key string, cfg *Config) resolvedOptions {
	if cfg == nil {
		return resolvedOptions{}
	}

	resolved := resolvedOptions{
		Overwrite:                    optionEnabled(cfg.globalOptions.Overwrite),
		UnescapeBackslashBackslash:   optionEnabled(cfg.globalOptions.UnescapeBackslashBackslash),
		UnescapeBackslashDoubleQuote: optionEnabled(cfg.globalOptions.UnescapeBackslashDoubleQuote),
		UnescapeBackslashSingleQuote: optionEnabled(cfg.globalOptions.UnescapeBackslashSingleQuote),
		UnescapeBackslashA:           optionEnabled(cfg.globalOptions.UnescapeBackslashA),
		UnescapeBackslashB:           optionEnabled(cfg.globalOptions.UnescapeBackslashB),
		UnescapeBackslashF:           optionEnabled(cfg.globalOptions.UnescapeBackslashF),
		UnescapeBackslashN:           optionEnabled(cfg.globalOptions.UnescapeBackslashN),
		UnescapeBackslashR:           optionEnabled(cfg.globalOptions.UnescapeBackslashR),
		UnescapeBackslashT:           optionEnabled(cfg.globalOptions.UnescapeBackslashT),
		UnescapeBackslashV:           optionEnabled(cfg.globalOptions.UnescapeBackslashV),
		TransformCRLFToLF:            optionEnabled(cfg.globalOptions.TransformCRLFToLF),
		TransformCRToLF:              optionEnabled(cfg.globalOptions.TransformCRToLF),
	}

	if keyOptions, ok := cfg.keyOptions[key]; ok {
		if keyOptions.Overwrite != nil {
			resolved.Overwrite = *keyOptions.Overwrite
		}
		if keyOptions.UnescapeBackslashBackslash != nil {
			resolved.UnescapeBackslashBackslash = *keyOptions.UnescapeBackslashBackslash
		}
		if keyOptions.UnescapeBackslashDoubleQuote != nil {
			resolved.UnescapeBackslashDoubleQuote = *keyOptions.UnescapeBackslashDoubleQuote
		}
		if keyOptions.UnescapeBackslashSingleQuote != nil {
			resolved.UnescapeBackslashSingleQuote = *keyOptions.UnescapeBackslashSingleQuote
		}
		if keyOptions.UnescapeBackslashA != nil {
			resolved.UnescapeBackslashA = *keyOptions.UnescapeBackslashA
		}
		if keyOptions.UnescapeBackslashB != nil {
			resolved.UnescapeBackslashB = *keyOptions.UnescapeBackslashB
		}
		if keyOptions.UnescapeBackslashF != nil {
			resolved.UnescapeBackslashF = *keyOptions.UnescapeBackslashF
		}
		if keyOptions.UnescapeBackslashN != nil {
			resolved.UnescapeBackslashN = *keyOptions.UnescapeBackslashN
		}
		if keyOptions.UnescapeBackslashR != nil {
			resolved.UnescapeBackslashR = *keyOptions.UnescapeBackslashR
		}
		if keyOptions.UnescapeBackslashT != nil {
			resolved.UnescapeBackslashT = *keyOptions.UnescapeBackslashT
		}
		if keyOptions.UnescapeBackslashV != nil {
			resolved.UnescapeBackslashV = *keyOptions.UnescapeBackslashV
		}
		if keyOptions.TransformCRLFToLF != nil {
			resolved.TransformCRLFToLF = *keyOptions.TransformCRLFToLF
		}
		if keyOptions.TransformCRToLF != nil {
			resolved.TransformCRToLF = *keyOptions.TransformCRToLF
		}
	}

	return resolved
}

func optionEnabled(value *bool) bool {
	return value != nil && *value
}
