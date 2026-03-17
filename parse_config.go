package strictdotenv

// ---------------------------------------------------------------------------
// ParseConfig
// ---------------------------------------------------------------------------

// ParseConfig stores partial parse options plus their resolved concrete values.
// All options are disabled by default. ApplyGlobalOptions sets the base options
// for every key, and ApplyKeyOptions adds or updates exact-key overrides that
// inherit from the current global settings. Overwrite affects every committed
// key-value pair during parsing; all other ParseOptions affect only
// double-quoted values. EnvStore.ProcessValue and EnvStore.ProcessValues ignore
// Overwrite.
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

// ---------------------------------------------------------------------------
// ParseOptions
// ---------------------------------------------------------------------------

// ParseOptions is the partial, pointer-based form used to set only the fields
// you want to enable or disable. Nil fields leave the existing value unchanged
// when applied to a ParseConfig.
type ParseOptions struct {
	Overwrite                    *bool
	UnescapeBackslashBackslash   *bool // \\
	UnescapeBackslashDoubleQuote *bool // \"
	UnescapeBackslashSingleQuote *bool // \'
	UnescapeBackslashA           *bool // \a (alert/bell)
	UnescapeBackslashB           *bool // \b (backspace)
	UnescapeBackslashF           *bool // \f (form feed)
	UnescapeBackslashN           *bool // \n (line feed)
	UnescapeBackslashR           *bool // \r (carriage return)
	UnescapeBackslashT           *bool // \t (tab)
	UnescapeBackslashV           *bool // \v (vertical tab)
	TransformCRLFToLF            *bool // after unescaping
	TransformCRToLF              *bool // after TransformCRLFToLF
}

// resolvedParseOptions is the concrete form used during parsing and value
// processing.
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

func resolveOptions(opts *ParseOptions) resolvedParseOptions {
	var resolved resolvedParseOptions
	if opts == nil {
		return resolved
	}
	if opts.Overwrite != nil {
		resolved.Overwrite = *opts.Overwrite
	}
	if opts.UnescapeBackslashBackslash != nil {
		resolved.UnescapeBackslashBackslash = *opts.UnescapeBackslashBackslash
	}
	if opts.UnescapeBackslashDoubleQuote != nil {
		resolved.UnescapeBackslashDoubleQuote = *opts.UnescapeBackslashDoubleQuote
	}
	if opts.UnescapeBackslashSingleQuote != nil {
		resolved.UnescapeBackslashSingleQuote = *opts.UnescapeBackslashSingleQuote
	}
	if opts.UnescapeBackslashA != nil {
		resolved.UnescapeBackslashA = *opts.UnescapeBackslashA
	}
	if opts.UnescapeBackslashB != nil {
		resolved.UnescapeBackslashB = *opts.UnescapeBackslashB
	}
	if opts.UnescapeBackslashF != nil {
		resolved.UnescapeBackslashF = *opts.UnescapeBackslashF
	}
	if opts.UnescapeBackslashN != nil {
		resolved.UnescapeBackslashN = *opts.UnescapeBackslashN
	}
	if opts.UnescapeBackslashR != nil {
		resolved.UnescapeBackslashR = *opts.UnescapeBackslashR
	}
	if opts.UnescapeBackslashT != nil {
		resolved.UnescapeBackslashT = *opts.UnescapeBackslashT
	}
	if opts.UnescapeBackslashV != nil {
		resolved.UnescapeBackslashV = *opts.UnescapeBackslashV
	}
	if opts.TransformCRLFToLF != nil {
		resolved.TransformCRLFToLF = *opts.TransformCRLFToLF
	}
	if opts.TransformCRToLF != nil {
		resolved.TransformCRToLF = *opts.TransformCRToLF
	}
	return resolved
}

func applyResolvedOptions(base resolvedParseOptions, overrides *ParseOptions) resolvedParseOptions {
	if overrides == nil {
		return base
	}
	if overrides.Overwrite != nil {
		base.Overwrite = *overrides.Overwrite
	}
	if overrides.UnescapeBackslashBackslash != nil {
		base.UnescapeBackslashBackslash = *overrides.UnescapeBackslashBackslash
	}
	if overrides.UnescapeBackslashDoubleQuote != nil {
		base.UnescapeBackslashDoubleQuote = *overrides.UnescapeBackslashDoubleQuote
	}
	if overrides.UnescapeBackslashSingleQuote != nil {
		base.UnescapeBackslashSingleQuote = *overrides.UnescapeBackslashSingleQuote
	}
	if overrides.UnescapeBackslashA != nil {
		base.UnescapeBackslashA = *overrides.UnescapeBackslashA
	}
	if overrides.UnescapeBackslashB != nil {
		base.UnescapeBackslashB = *overrides.UnescapeBackslashB
	}
	if overrides.UnescapeBackslashF != nil {
		base.UnescapeBackslashF = *overrides.UnescapeBackslashF
	}
	if overrides.UnescapeBackslashN != nil {
		base.UnescapeBackslashN = *overrides.UnescapeBackslashN
	}
	if overrides.UnescapeBackslashR != nil {
		base.UnescapeBackslashR = *overrides.UnescapeBackslashR
	}
	if overrides.UnescapeBackslashT != nil {
		base.UnescapeBackslashT = *overrides.UnescapeBackslashT
	}
	if overrides.UnescapeBackslashV != nil {
		base.UnescapeBackslashV = *overrides.UnescapeBackslashV
	}
	if overrides.TransformCRLFToLF != nil {
		base.TransformCRLFToLF = *overrides.TransformCRLFToLF
	}
	if overrides.TransformCRToLF != nil {
		base.TransformCRToLF = *overrides.TransformCRToLF
	}
	return base
}

// ApplyGlobalOptions merges the supplied overrides into the config's global
// options and refreshes all resolved global and key-specific options.
func (c *ParseConfig) ApplyGlobalOptions(overrides *ParseOptions) {
	c.GlobalOptions = mergeParseOptions(c.GlobalOptions, overrides)
	c.resolvedGlobalOptions = resolveOptions(&c.GlobalOptions)

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
		c.resolvedKeyOptions[key] = applyResolvedOptions(c.resolvedGlobalOptions, &keyOptions)
	}
}

// ApplyKeyOptions merges the supplied overrides into the exact-key options and
// refreshes the resolved options for that key only.
func (c *ParseConfig) ApplyKeyOptions(key string, overrides *ParseOptions) {
	if c.KeyOptions == nil {
		c.KeyOptions = make(map[string]ParseOptions)
	}

	c.KeyOptions[key] = mergeParseOptions(c.KeyOptions[key], overrides)
	c.resolvedGlobalOptions = resolveOptions(&c.GlobalOptions)

	if c.resolvedKeyOptions == nil {
		c.resolvedKeyOptions = make(map[string]resolvedParseOptions, len(c.KeyOptions))
	}
	keyOptions := c.KeyOptions[key]
	c.resolvedKeyOptions[key] = applyResolvedOptions(c.resolvedGlobalOptions, &keyOptions)
}

func resolveParseOptions(cfg *ParseConfig, key string) resolvedParseOptions {
	if cfg == nil {
		return resolvedParseOptions{}
	}

	cfg.resolvedGlobalOptions = resolveOptions(&cfg.GlobalOptions)

	if keyOptions, ok := cfg.KeyOptions[key]; ok {
		resolved := applyResolvedOptions(cfg.resolvedGlobalOptions, &keyOptions)
		if cfg.resolvedKeyOptions == nil {
			cfg.resolvedKeyOptions = make(map[string]resolvedParseOptions, len(cfg.KeyOptions))
		}
		cfg.resolvedKeyOptions[key] = resolved
		return resolved
	}

	return cfg.resolvedGlobalOptions
}
