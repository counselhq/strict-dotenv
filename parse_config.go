package strictdotenv

// ---------------------------------------------------------------------------
// Types
//
// Public:
// 	- ParseConfig  — top-level configuration passed to parse functions
// 	- ParseOptions — partial (pointer-based) option set used to build a config
//
// Internal:
// 	- resolvedParseOptions — concrete bool option set derived from ParseOptions
// ---------------------------------------------------------------------------

// ParseConfig holds parse options and their pre-resolved concrete values.
//
// All options default to false (disabled). Use [ParseConfig.ApplyGlobalOptions]
// to set a baseline that applies to every key, and [ParseConfig.ApplyKeyOptions]
// to add per-key overrides that layer on top of the global baseline.
//
// Option scoping for EnvStore.ProcessValue and EnvStore.ProcessValues:
//   - [ParseOptions.Overwrite] is ignored
//   - All other options apply only to unqoted, single-quoted, and double-quoted values.
//
// Option scoping for EnvStore.SetFrom* methods:
//   - [ParseOptions.Overwrite] applies to unqoted, single-quoted, and double-quoted values.
//   - All other options apply only to double-quoted values.
//
// A nil *ParseConfig is valid and equivalent to a zero-value ParseConfig
// (all options disabled).
//
// Usage:
//
//	cfg := new(ParseConfig)
//	cfg.ApplyGlobalOptions(&ParseOptions{Overwrite: new(true)})
//	cfg.ApplyKeyOptions("SECRET", &ParseOptions{UnescapeBackslashN: new(false)})
type ParseConfig struct {
	GlobalOptions ParseOptions
	KeyOptions    map[string]ParseOptions

	resolvedGlobalOptions resolvedParseOptions
	resolvedKeyOptions    map[string]resolvedParseOptions
}

// ParseOptions is the partial, pointer-based option set passed to
// [ParseConfig.ApplyGlobalOptions] and [ParseConfig.ApplyKeyOptions].
//
// Every field is a pointer so that only the fields you explicitly set are
// applied; nil fields leave the existing value in the config unchanged. This
// allows incremental, non-destructive configuration — for example, enabling
// one option without resetting unrelated ones.
//
// The Unescape* fields control which backslash sequences are expanded into
// their intended character or control action inside a value. The Transform*
// fields run after unescaping and normalise line endings in the resulting string.
type ParseOptions struct {
	// Overwrite controls whether an existing key-value pair in the store is
	// overwritten when the same key appears again during parsing.
	Overwrite *bool

	UnescapeBackslashBackslash   *bool // \\ → \
	UnescapeBackslashDoubleQuote *bool // \" → "
	UnescapeBackslashSingleQuote *bool // \' → '
	UnescapeBackslashA           *bool // \a → alert/bell 			(U+0007)
	UnescapeBackslashB           *bool // \b → backspace  			(U+0008)
	UnescapeBackslashF           *bool // \f → form feed  			(U+000C)
	UnescapeBackslashN           *bool // \n → line feed (LF)  		(U+000A)
	UnescapeBackslashR           *bool // \r → carriage return (CR)	(U+000D)
	UnescapeBackslashT           *bool // \t → horizontal tab  		(U+0009)
	UnescapeBackslashV           *bool // \v → vertical tab  		(U+000B)

	// Transform* fields normalise line endings after unescaping.
	TransformCRLFToLF *bool // CRLF → LF	runs first
	TransformCRToLF   *bool // CR 	→ LF 	runs after TransformCRLFToLF
}

// resolvedParseOptions is the concrete, non-pointer form of [ParseOptions]
// used internally during parsing and value processing. It is derived from a
// ParseOptions via resolveGlobalOptions or resolveKeyOptions and cached inside
// ParseConfig to avoid repeated pointer dereferences on the hot path.
type resolvedParseOptions struct {
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
// Public API for building a ParseConfig:
// 	- ApplyGlobalOptions - set or update the baseline options for all keys
// 	- ApplyKeyOptions - set or update overrides for a single named key
// ---------------------------------------------------------------------------

// ApplyGlobalOptions merges overrides into the global baseline and refreshes
// all cached resolved options.
//
// Only non-nil fields in overrides are applied; nil fields leave the current
// global value unchanged. After the merge, the resolved forms of both the
// global options and every existing per-key override are recalculated so that
// cached state stays consistent.
//
// Passing a nil overrides is a no-op.
func (c *ParseConfig) ApplyGlobalOptions(overrides *ParseOptions) {
	c.GlobalOptions = mergeParseOptions(c.GlobalOptions, overrides)
	c.resolvedGlobalOptions = resolveGlobalOptions(&c.GlobalOptions)

	if len(c.KeyOptions) == 0 {
		c.resolvedKeyOptions = nil
		return
	}

	if c.resolvedKeyOptions == nil {
		c.resolvedKeyOptions = make(map[string]resolvedParseOptions, len(c.KeyOptions))
	} else {
		clear(c.resolvedKeyOptions)
	}

	for key, keyOptions := range c.KeyOptions {
		c.resolvedKeyOptions[key] = resolveKeyOptions(c.resolvedGlobalOptions, &keyOptions)
	}
}

// ApplyKeyOptions merges overrides into the per-key options for key and
// refreshes the cached resolved options for that key.
//
// Per-key options inherit from the current global baseline: the resolved value
// for a field is the global value unless the per-key setting explicitly
// overrides it. Calling ApplyGlobalOptions after ApplyKeyOptions will
// re-resolve all per-key overrides, propagating any new global values.
//
// Only non-nil fields in overrides are applied. Passing a nil overrides
// registers key without changing any of its options (useful for opting a key
// into key-level tracking while inheriting all globals).
func (c *ParseConfig) ApplyKeyOptions(key string, overrides *ParseOptions) {
	if c.KeyOptions == nil {
		c.KeyOptions = make(map[string]ParseOptions)
	}

	c.KeyOptions[key] = mergeParseOptions(c.KeyOptions[key], overrides)
	c.resolvedGlobalOptions = resolveGlobalOptions(&c.GlobalOptions)

	if c.resolvedKeyOptions == nil {
		c.resolvedKeyOptions = make(map[string]resolvedParseOptions, len(c.KeyOptions))
	}
	keyOptions := c.KeyOptions[key]
	c.resolvedKeyOptions[key] = resolveKeyOptions(c.resolvedGlobalOptions, &keyOptions)
}

// ---------------------------------------------------------------------------
// Internal helpers for building/resolving a ParseConfig
// ---------------------------------------------------------------------------

// mergeParseOptions copies overrides's non-nil fields onto base and returns
// the result. base is passed by value so the original is never mutated.
// Pointer fields are deep-copied so the returned ParseOptions owns its memory.
func mergeParseOptions(base ParseOptions, overrides *ParseOptions) ParseOptions {
	if overrides == nil {
		return base
	}
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

// resolveGlobalOptions converts a ParseOptions into a resolvedParseOptions by
// dereferencing non-nil pointers. Nil pointers resolve to false (the zero
// value). A nil opts is safe and returns an all-false resolvedParseOptions.
func resolveGlobalOptions(globalOptions *ParseOptions) resolvedParseOptions {
	var resolved resolvedParseOptions
	if globalOptions == nil {
		return resolved
	}
	if globalOptions.Overwrite != nil {
		resolved.Overwrite = *globalOptions.Overwrite
	}
	if globalOptions.UnescapeBackslashBackslash != nil {
		resolved.UnescapeBackslashBackslash = *globalOptions.UnescapeBackslashBackslash
	}
	if globalOptions.UnescapeBackslashDoubleQuote != nil {
		resolved.UnescapeBackslashDoubleQuote = *globalOptions.UnescapeBackslashDoubleQuote
	}
	if globalOptions.UnescapeBackslashSingleQuote != nil {
		resolved.UnescapeBackslashSingleQuote = *globalOptions.UnescapeBackslashSingleQuote
	}
	if globalOptions.UnescapeBackslashA != nil {
		resolved.UnescapeBackslashA = *globalOptions.UnescapeBackslashA
	}
	if globalOptions.UnescapeBackslashB != nil {
		resolved.UnescapeBackslashB = *globalOptions.UnescapeBackslashB
	}
	if globalOptions.UnescapeBackslashF != nil {
		resolved.UnescapeBackslashF = *globalOptions.UnescapeBackslashF
	}
	if globalOptions.UnescapeBackslashN != nil {
		resolved.UnescapeBackslashN = *globalOptions.UnescapeBackslashN
	}
	if globalOptions.UnescapeBackslashR != nil {
		resolved.UnescapeBackslashR = *globalOptions.UnescapeBackslashR
	}
	if globalOptions.UnescapeBackslashT != nil {
		resolved.UnescapeBackslashT = *globalOptions.UnescapeBackslashT
	}
	if globalOptions.UnescapeBackslashV != nil {
		resolved.UnescapeBackslashV = *globalOptions.UnescapeBackslashV
	}
	if globalOptions.TransformCRLFToLF != nil {
		resolved.TransformCRLFToLF = *globalOptions.TransformCRLFToLF
	}
	if globalOptions.TransformCRToLF != nil {
		resolved.TransformCRToLF = *globalOptions.TransformCRToLF
	}
	return resolved
}

// resolveKeyOptions overlays the non-nil fields of overrides onto a
// resolved base and returns the combined result. It is used to compute the
// effective resolved options for a specific key: start from the resolved
// global baseline (base) and apply only the fields explicitly set for that key
// (overrides). A nil overrides returns base unchanged.
func resolveKeyOptions(resolvedGlobalOptions resolvedParseOptions, overrides *ParseOptions) resolvedParseOptions {
	if overrides == nil {
		return resolvedGlobalOptions
	}
	if overrides.Overwrite != nil {
		resolvedGlobalOptions.Overwrite = *overrides.Overwrite
	}
	if overrides.UnescapeBackslashBackslash != nil {
		resolvedGlobalOptions.UnescapeBackslashBackslash = *overrides.UnescapeBackslashBackslash
	}
	if overrides.UnescapeBackslashDoubleQuote != nil {
		resolvedGlobalOptions.UnescapeBackslashDoubleQuote = *overrides.UnescapeBackslashDoubleQuote
	}
	if overrides.UnescapeBackslashSingleQuote != nil {
		resolvedGlobalOptions.UnescapeBackslashSingleQuote = *overrides.UnescapeBackslashSingleQuote
	}
	if overrides.UnescapeBackslashA != nil {
		resolvedGlobalOptions.UnescapeBackslashA = *overrides.UnescapeBackslashA
	}
	if overrides.UnescapeBackslashB != nil {
		resolvedGlobalOptions.UnescapeBackslashB = *overrides.UnescapeBackslashB
	}
	if overrides.UnescapeBackslashF != nil {
		resolvedGlobalOptions.UnescapeBackslashF = *overrides.UnescapeBackslashF
	}
	if overrides.UnescapeBackslashN != nil {
		resolvedGlobalOptions.UnescapeBackslashN = *overrides.UnescapeBackslashN
	}
	if overrides.UnescapeBackslashR != nil {
		resolvedGlobalOptions.UnescapeBackslashR = *overrides.UnescapeBackslashR
	}
	if overrides.UnescapeBackslashT != nil {
		resolvedGlobalOptions.UnescapeBackslashT = *overrides.UnescapeBackslashT
	}
	if overrides.UnescapeBackslashV != nil {
		resolvedGlobalOptions.UnescapeBackslashV = *overrides.UnescapeBackslashV
	}
	if overrides.TransformCRLFToLF != nil {
		resolvedGlobalOptions.TransformCRLFToLF = *overrides.TransformCRLFToLF
	}
	if overrides.TransformCRToLF != nil {
		resolvedGlobalOptions.TransformCRToLF = *overrides.TransformCRToLF
	}
	return resolvedGlobalOptions
}

// resolveParseOptions returns the effective resolvedParseOptions for key,
// updating the cached resolved state inside cfg as a side effect.
//
// Resolution order:
//  1. Start with the resolved global options.
//  2. If key has a per-key override, layer it on top via resolveKeyOptions.
//
// A nil cfg is safe and returns an all-false resolvedParseOptions.
func resolveParseOptions(cfg *ParseConfig, key string) resolvedParseOptions {
	if cfg == nil {
		return resolvedParseOptions{}
	}

	cfg.resolvedGlobalOptions = resolveGlobalOptions(&cfg.GlobalOptions)

	if keyOptions, ok := cfg.KeyOptions[key]; ok {
		resolved := resolveKeyOptions(cfg.resolvedGlobalOptions, &keyOptions)
		if cfg.resolvedKeyOptions == nil {
			cfg.resolvedKeyOptions = make(map[string]resolvedParseOptions, len(cfg.KeyOptions))
		}
		cfg.resolvedKeyOptions[key] = resolved
		return resolved
	}

	return cfg.resolvedGlobalOptions
}
