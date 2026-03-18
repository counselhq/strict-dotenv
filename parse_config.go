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
// All options default to false (disabled). Use [Config.MergeGlobalOptions] to
// incrementally build a baseline that applies to every key, [Config.SetGlobalOptions]
// to replace that baseline outright, [Config.MergeKeyOptions] to add per-key
// overrides that layer on top of the global baseline, or [Config.SetKeyOptions]
// to replace one key's overrides outright.
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
//	cfg.MergeGlobalOptions(Options{Overwrite: new(true)})
//	cfg.MergeKeyOptions("SECRET", Options{UnescapeBackslashN: new(false)})
type Config struct {
	globalOptions Options
	keyOptions    map[string]Options
}

// Options is the partial, pointer-field option set passed to
// [Config.MergeGlobalOptions], [Config.SetGlobalOptions],
// [Config.MergeKeyOptions], and [Config.SetKeyOptions].
//
// Every field is a pointer so that only the fields you explicitly set are
// stored. With the Merge* methods, nil fields leave the existing value in the
// config unchanged. With the Set* methods, nil fields leave that option unset
// in the stored config. This supports both incremental, non-destructive updates
// and full replacement of an option set.
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
//   - MergeGlobalOptions - merge fields into the baseline options for all keys
//   - SetGlobalOptions - replace the baseline options for all keys
//   - MergeKeyOptions - merge fields into the overrides for one named key
//   - SetKeyOptions - replace the overrides for one named key
// ---------------------------------------------------------------------------

// MergeGlobalOptions merges the provided options into the existing global options.
// Any unset or nil fields are ignored.
//
// Config maintains its own copy of all options, so callers can reuse
// and modify the same Options without affecting the Config after the call.
func (c *Config) MergeGlobalOptions(options Options) {
	c.globalOptions = mergeOptions(c.globalOptions, options)
}

// SetGlobalOptions sets or replaces the existing global options with the provided options.
// Any field left unset or nil will be left unset in the stored global options.
//
// Config maintains its own copy of all options, so callers can reuse
// and modify the same Options without affecting the Config after the call.
func (c *Config) SetGlobalOptions(options Options) {
	c.globalOptions = mergeOptions(Options{}, options)
}

// MergeKeyOptions merges the provided options into the per-key options for key.
// Any unset or nil fields are ignored.
//
// Per-key options inherit from the global options: the effective value for a field
// is the global value unless the per-key setting explicitly overrides it.
//
// Config maintains its own copy of all options, so callers can reuse
// and modify the same Options without affecting the Config after the call.
func (c *Config) MergeKeyOptions(key string, options Options) {
	if c.keyOptions == nil {
		c.keyOptions = make(map[string]Options)
	}

	c.keyOptions[key] = mergeOptions(c.keyOptions[key], options)
}

// SetKeyOptions sets or replaces the per-key options for key.
// Any field left unset or nil will be left unset in the stored per-key overrides.
//
// Per-key options inherit from the global options: the effective value for a field
// is the global value unless the per-key setting explicitly overrides it.
//
// Config maintains its own copy of all options, so callers can reuse
// and modify the same Options without affecting the Config after the call.
func (c *Config) SetKeyOptions(key string, options Options) {
	if c.keyOptions == nil {
		c.keyOptions = make(map[string]Options)
	}

	c.keyOptions[key] = mergeOptions(Options{}, options)
}

// ---------------------------------------------------------------------------
// Internal helpers for building/resolving a Config
// ---------------------------------------------------------------------------

// mergeOptions safely merges overrides onto the desired base Options.
// Pointer fields are deep-copied so the stored Options owns its memory.
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
