package strictdotenv

// ---------------------------------------------------------------------------
// ParseConfig
// ---------------------------------------------------------------------------

// ParseConfig selects the default ParseOptions for the parse and optionally
// replaces them for specific keys. Use NewParseConfig to construct an empty
// ParseConfig, WithRecommendedDefaults to opt into strict-dotenv's recommended
// defaults, WithBaseOptions to modify the current base options, and
// WithKeyOptions to add per-key overrides that inherit from the current base.
// Overwrite affects every committed key-value pair during parsing; all other
// ParseOptions affect only double-quoted values. EnvStore.ProcessValue and
// EnvStore.ProcessValues ignore Overwrite.
//
// Usage:
//
//	cfg := NewParseConfig() // all zero values
//	cfg := NewParseConfig().
//		WithRecommendedDefaults().
//		WithBaseOptions(&CustomParseOptions{Overwrite: BoolPtr(true)})
//	cfg := NewParseConfig().
//		WithRecommendedDefaults().
//		WithKeyOptions("SECRET", &CustomParseOptions{UnescapeBackslashN: BoolPtr(false)})
type ParseConfig struct {
	ParseOptions ParseOptions
	KeyOptions   map[string]ParseOptions
}

// ---------------------------------------------------------------------------
// ParseOptions and CustomParseOptions
// ---------------------------------------------------------------------------

// ParseOptions controls duplicate-key handling during parsing and the
// double-quoted value transform pipeline used by both parsing and EnvStore
// value-processing helpers.
type ParseOptions struct {
	Overwrite                    bool
	UnescapeBackslashBackslash   bool // \\
	UnescapeBackslashDoubleQuote bool // \"
	UnescapeBackslashSingleQuote bool // \'
	UnescapeBackslashA           bool // \a (alert/bell)
	UnescapeBackslashB           bool // \b (backspace)
	UnescapeBackslashF           bool // \f (form feed)
	UnescapeBackslashN           bool // \n (line feed)
	UnescapeBackslashR           bool // \r (carriage return)
	UnescapeBackslashT           bool // \t (tab)
	UnescapeBackslashV           bool // \v (vertical tab)
	TransformCRLFToLF            bool // after unescaping
	TransformCRToLF              bool // after TransformCRLFToLF
}

// CustomParseOptions is the partial, pointer-based form used for overriding
// only selected ParseOptions fields while leaving the rest unchanged.
type CustomParseOptions struct {
	Overwrite                    *bool
	UnescapeBackslashBackslash   *bool
	UnescapeBackslashDoubleQuote *bool
	UnescapeBackslashSingleQuote *bool
	UnescapeBackslashA           *bool
	UnescapeBackslashB           *bool
	UnescapeBackslashF           *bool
	UnescapeBackslashN           *bool
	UnescapeBackslashR           *bool
	UnescapeBackslashT           *bool
	UnescapeBackslashV           *bool
	TransformCRLFToLF            *bool
	TransformCRToLF              *bool
}

// defaultParseOptions contains the library's current recommended parse
// defaults. Use ParseConfig.WithRecommendedDefaults to copy them into a
// config before applying additional overrides.
var defaultParseOptions = CustomParseOptions{
	Overwrite:                    BoolPtr(false),
	UnescapeBackslashBackslash:   BoolPtr(true),
	UnescapeBackslashDoubleQuote: BoolPtr(true),
	UnescapeBackslashSingleQuote: BoolPtr(false),
	UnescapeBackslashA:           BoolPtr(false),
	UnescapeBackslashB:           BoolPtr(false),
	UnescapeBackslashF:           BoolPtr(false),
	UnescapeBackslashN:           BoolPtr(true),
	UnescapeBackslashR:           BoolPtr(true),
	UnescapeBackslashT:           BoolPtr(true),
	UnescapeBackslashV:           BoolPtr(false),
	TransformCRLFToLF:            BoolPtr(true),
	TransformCRToLF:              BoolPtr(true),
}

// BoolPtr is a helper for easily constructing *bool values from bool literals.
// Remove in favor of new(false) | new(true) when library requires Go 1.26+.
func BoolPtr(b bool) *bool { return &b }

// resolveCustom converts a fully-populated CustomParseOptions to ParseOptions.
func resolveCustom(c *CustomParseOptions) ParseOptions {
	var opts ParseOptions
	if c.Overwrite != nil {
		opts.Overwrite = *c.Overwrite
	}
	if c.UnescapeBackslashBackslash != nil {
		opts.UnescapeBackslashBackslash = *c.UnescapeBackslashBackslash
	}
	if c.UnescapeBackslashDoubleQuote != nil {
		opts.UnescapeBackslashDoubleQuote = *c.UnescapeBackslashDoubleQuote
	}
	if c.UnescapeBackslashSingleQuote != nil {
		opts.UnescapeBackslashSingleQuote = *c.UnescapeBackslashSingleQuote
	}
	if c.UnescapeBackslashA != nil {
		opts.UnescapeBackslashA = *c.UnescapeBackslashA
	}
	if c.UnescapeBackslashB != nil {
		opts.UnescapeBackslashB = *c.UnescapeBackslashB
	}
	if c.UnescapeBackslashF != nil {
		opts.UnescapeBackslashF = *c.UnescapeBackslashF
	}
	if c.UnescapeBackslashN != nil {
		opts.UnescapeBackslashN = *c.UnescapeBackslashN
	}
	if c.UnescapeBackslashR != nil {
		opts.UnescapeBackslashR = *c.UnescapeBackslashR
	}
	if c.UnescapeBackslashT != nil {
		opts.UnescapeBackslashT = *c.UnescapeBackslashT
	}
	if c.UnescapeBackslashV != nil {
		opts.UnescapeBackslashV = *c.UnescapeBackslashV
	}
	if c.TransformCRLFToLF != nil {
		opts.TransformCRLFToLF = *c.TransformCRLFToLF
	}
	if c.TransformCRToLF != nil {
		opts.TransformCRToLF = *c.TransformCRToLF
	}
	return opts
}

// applyCustomOverrides returns a copy of base with any non-nil fields from
// overrides applied. If overrides is nil, base is returned unchanged.
func applyCustomOverrides(base ParseOptions, overrides *CustomParseOptions) ParseOptions {
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

// NewParseConfig creates an empty *ParseConfig without applying any defaults.
//
// Usage:
//
//	cfg := NewParseConfig()                              // all zero values
//	cfg := NewParseConfig().WithRecommendedDefaults()    // recommended defaults
func NewParseConfig() *ParseConfig {
	return &ParseConfig{}
}

// WithRecommendedDefaults replaces the config's base ParseOptions with
// strict-dotenv's current recommended defaults.
func (c *ParseConfig) WithRecommendedDefaults() *ParseConfig {
	c.ParseOptions = resolveCustom(&defaultParseOptions)
	return c
}

// WithBaseOptions applies partial overrides to the config's current base
// ParseOptions. Unset fields are left unchanged.
func (c *ParseConfig) WithBaseOptions(overrides *CustomParseOptions) *ParseConfig {
	c.ParseOptions = applyCustomOverrides(c.ParseOptions, overrides)
	return c
}

// WithKeyOptions returns the ParseConfig with key-specific overrides added.
// Only set the fields you want to change; unset fields inherit from the
// config's current base ParseOptions.
//
// Usage:
//
//	cfg := NewParseConfig().
//		WithRecommendedDefaults().
//		WithKeyOptions("SECRET", &CustomParseOptions{UnescapeBackslashN: BoolPtr(false)}).
//		WithKeyOptions("TOKEN", &CustomParseOptions{Overwrite: BoolPtr(true)})
func (c *ParseConfig) WithKeyOptions(key string, overrides *CustomParseOptions) *ParseConfig {
	if c.KeyOptions == nil {
		c.KeyOptions = make(map[string]ParseOptions)
	}
	c.KeyOptions[key] = applyCustomOverrides(c.ParseOptions, overrides)
	return c
}

func resolveParseOptions(cfg *ParseConfig, key string) ParseOptions {
	if cfg == nil {
		return ParseOptions{}
	}

	if opts, ok := cfg.KeyOptions[key]; ok {
		return opts
	}

	return cfg.ParseOptions
}
