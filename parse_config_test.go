package strictdotenv

import (
	"testing"
)

// ---------------------------------------------------------------------------
// NewParseConfig Tests
// ---------------------------------------------------------------------------

func TestNewParseConfigReturnsEmptyConfig(t *testing.T) {
	cfg := NewParseConfig()

	if cfg.ParseOptions != (ParseOptions{}) {
		t.Errorf("NewParseConfig().ParseOptions = %+v, want zero ParseOptions", cfg.ParseOptions)
	}
	if cfg.KeyOptions != nil {
		t.Errorf("NewParseConfig().KeyOptions = %v, want nil", cfg.KeyOptions)
	}
}

func TestWithRecommendedDefaultsAppliesDefaults(t *testing.T) {
	cfg := NewParseConfig().
		WithBaseOptions(&CustomParseOptions{UnescapeBackslashN: new(false)}).
		WithRecommendedDefaults()
	want := resolveCustom(&defaultParseOptions)

	if cfg.ParseOptions != want {
		t.Errorf("WithRecommendedDefaults().ParseOptions = %+v, want %+v", cfg.ParseOptions, want)
	}
	if cfg.KeyOptions != nil {
		t.Errorf("WithRecommendedDefaults().KeyOptions = %v, want nil", cfg.KeyOptions)
	}
}

func TestWithBaseOptionsLeavesUnsetFieldsUntouched(t *testing.T) {
	cfg := NewParseConfig().WithBaseOptions(&CustomParseOptions{
		Overwrite:          new(true),
		UnescapeBackslashN: new(true),
	})

	if !cfg.ParseOptions.Overwrite {
		t.Error("expected Overwrite=true")
	}
	if !cfg.ParseOptions.UnescapeBackslashN {
		t.Error("expected UnescapeBackslashN=true")
	}
	// Unset fields should stay at the config's existing value (zero here).
	if cfg.ParseOptions.UnescapeBackslashBackslash {
		t.Error("expected UnescapeBackslashBackslash=false")
	}
	if cfg.ParseOptions.TransformCRLFToLF {
		t.Error("expected TransformCRLFToLF=false")
	}
}

func TestWithBaseOptionsAfterRecommendedDefaultsPreservesExistingValues(t *testing.T) {
	cfg := NewParseConfig().WithRecommendedDefaults().WithBaseOptions(&CustomParseOptions{
		Overwrite:          new(true),
		UnescapeBackslashN: new(false),
	})

	if !cfg.ParseOptions.Overwrite {
		t.Error("expected Overwrite=true")
	}
	if cfg.ParseOptions.UnescapeBackslashN {
		t.Error("expected UnescapeBackslashN=false")
	}
	if !cfg.ParseOptions.UnescapeBackslashBackslash {
		t.Error("expected UnescapeBackslashBackslash=true (recommended default)")
	}
	if !cfg.ParseOptions.TransformCRLFToLF {
		t.Error("expected TransformCRLFToLF=true (recommended default)")
	}
}

// ---------------------------------------------------------------------------
// WithKeyOptions Tests
// ---------------------------------------------------------------------------

func TestWithKeyOptionsInheritsRecommendedBase(t *testing.T) {
	cfg := NewParseConfig().WithRecommendedDefaults().WithBaseOptions(&CustomParseOptions{
		Overwrite: new(true),
	}).WithKeyOptions("SECRET", &CustomParseOptions{
		UnescapeBackslashN: new(false),
	})

	keyOpts := cfg.KeyOptions["SECRET"]

	// Explicit key override should take effect.
	if keyOpts.UnescapeBackslashN {
		t.Error("expected SECRET UnescapeBackslashN=false")
	}
	// Should inherit Overwrite=true from base.
	if !keyOpts.Overwrite {
		t.Error("expected SECRET Overwrite=true (inherited from base)")
	}
	// Should preserve other defaults.
	if !keyOpts.UnescapeBackslashBackslash {
		t.Error("expected SECRET UnescapeBackslashBackslash=true (default)")
	}
}

func TestWithKeyOptionsInheritsUntouchedBase(t *testing.T) {
	cfg := NewParseConfig().WithKeyOptions("SECRET", &CustomParseOptions{
		UnescapeBackslashN: new(true),
	})

	keyOpts := cfg.KeyOptions["SECRET"]

	if !keyOpts.UnescapeBackslashN {
		t.Error("expected SECRET UnescapeBackslashN=true")
	}
	if keyOpts.UnescapeBackslashBackslash {
		t.Error("expected SECRET UnescapeBackslashBackslash=false (inherited from untouched base)")
	}
}

func TestWithKeyOptionsChaining(t *testing.T) {
	cfg := NewParseConfig().WithRecommendedDefaults().
		WithKeyOptions("A", &CustomParseOptions{Overwrite: new(true)}).
		WithKeyOptions("B", &CustomParseOptions{UnescapeBackslashN: new(false)})

	if !cfg.KeyOptions["A"].Overwrite {
		t.Error("expected A Overwrite=true")
	}
	if cfg.KeyOptions["B"].UnescapeBackslashN {
		t.Error("expected B UnescapeBackslashN=false")
	}
	// B should still have default Overwrite (false).
	if cfg.KeyOptions["B"].Overwrite {
		t.Error("expected B Overwrite=false (default)")
	}
}

func TestWithKeyOptionsNilOverridesUsesBase(t *testing.T) {
	base := NewParseConfig().WithRecommendedDefaults().WithBaseOptions(&CustomParseOptions{Overwrite: new(true)})
	cfg := base.WithKeyOptions("KEY", nil)

	if cfg.KeyOptions["KEY"] != base.ParseOptions {
		t.Errorf("WithKeyOptions(nil) should equal base\ngot  %+v\nwant %+v",
			cfg.KeyOptions["KEY"], base.ParseOptions)
	}
}

func TestNilParseConfigDoesNotUseRecommendedDefaults(t *testing.T) {
	run(t, nil, nil, testCase{
		name:   "nil config preserves literal escapes",
		dotenv: "KEY=\"line1\\nline2\"",
		want:   EnvStore{"KEY": `line1\nline2`},
	})
}

// ---------------------------------------------------------------------------
// Overwrite Option Tests
// ---------------------------------------------------------------------------
// 	- Default | Overwrite: false
// 		- if the same key appears multiple times, the first one wins
// 	- Overwrite: true:
// 		- if the same key appears multiple times, the last one wins
// ---------------------------------------------------------------------------

func TestConfigOptionsOverwrite(t *testing.T) {
	defaultCfg := NewParseConfig().WithRecommendedDefaults()

	run(t, nil, defaultCfg, testCase{
		name:   "DEFAULT: repeated keys first one wins",
		dotenv: "KEY=1\nKEY=2",
		want:   EnvStore{"KEY": "1"},
	})

	cfg := NewParseConfig().WithRecommendedDefaults().WithBaseOptions(&CustomParseOptions{Overwrite: new(false)})

	run(t, nil, cfg, testCase{
		name:   "Overwrite: false (same as default) - repeated keys first one wins",
		dotenv: "KEY=1\nKEY=2",
		want:   EnvStore{"KEY": "1"},
	})

	cfg = NewParseConfig().WithRecommendedDefaults().WithBaseOptions(&CustomParseOptions{Overwrite: new(true)})

	run(t, nil, cfg, testCase{
		name:   "Overwrite: true - repeated keys last one wins",
		dotenv: "KEY=1\nKEY=2",
		want:   EnvStore{"KEY": "2"},
	})

	cfg = NewParseConfig().WithRecommendedDefaults().
		WithKeyOptions("KEY", &CustomParseOptions{Overwrite: new(true)})

	run(t, nil, cfg, testCase{
		name:   "Key-specific Overwrite",
		dotenv: "KEY=1\nKEY=2\nOTHER=3\nOTHER=4",
		want:   EnvStore{"KEY": "2", "OTHER": "3"},
	})

	cfg = NewParseConfig().WithRecommendedDefaults().WithBaseOptions(&CustomParseOptions{Overwrite: new(true)}).
		WithKeyOptions("KEY", &CustomParseOptions{Overwrite: new(false)})

	run(t, nil, cfg, testCase{
		name:   "Obey different key-specific and base overwrite settings",
		dotenv: "KEY=1\nKEY=2\nOTHER=3\nOTHER=4",
		want:   EnvStore{"KEY": "1", "OTHER": "4"},
	})

	cfg = NewParseConfig().WithRecommendedDefaults().
		WithKeyOptions("T", &CustomParseOptions{Overwrite: new(true)}).
		WithKeyOptions("F", &CustomParseOptions{Overwrite: new(false)})

	run(t, nil, cfg, testCase{
		name:   "obey different key-specific overwrite settings",
		dotenv: "T=1\nT=2\nF=3\nF=4",
		want:   EnvStore{"T": "2", "F": "3"},
	})
}

// ---------------------------------------------------------------------------
// UnescapeBackslashBackslash Option Tests
// ---------------------------------------------------------------------------
// 	- Default | UnescapeBackslashBackslash: true
// 		- if the value contains escaped backslashes, they should be unescaped
// 	- UnescapeBackslashBackslash: false:
// 		- if the value contains escaped backslashes, they should be preserved
// ---------------------------------------------------------------------------

func TestConfigOptionsUnescapeBackslashBackslash(t *testing.T) {
	defaultCfg := NewParseConfig().WithRecommendedDefaults()

	run(t, nil, defaultCfg, testCase{
		name:   "DEFAULT: escaped backslash is unescaped",
		dotenv: "KEY=\"a\\\\b\"",
		want:   EnvStore{"KEY": `a\b`},
	})

	cfg := NewParseConfig().WithRecommendedDefaults().WithBaseOptions(&CustomParseOptions{UnescapeBackslashBackslash: new(true)})

	run(t, nil, cfg, testCase{
		name:   "UnescapeBackslashBackslash: true (same as default) - escaped backslash is unescaped",
		dotenv: "KEY=\"a\\\\b\"",
		want:   EnvStore{"KEY": `a\b`},
	})

	cfg = NewParseConfig().WithRecommendedDefaults().WithBaseOptions(&CustomParseOptions{UnescapeBackslashBackslash: new(false)})

	run(t, nil, cfg, testCase{
		name:   "UnescapeBackslashBackslash: false - escaped backslash is preserved",
		dotenv: "KEY=\"a\\\\b\"",
		want:   EnvStore{"KEY": `a\\b`},
	})

	cfg = NewParseConfig().WithRecommendedDefaults().
		WithKeyOptions("KEY", &CustomParseOptions{UnescapeBackslashBackslash: new(false)})

	run(t, nil, cfg, testCase{
		name:   "Key-specific UnescapeBackslashBackslash",
		dotenv: "KEY=\"a\\\\b\"\nOTHER=\"c\\\\d\"",
		want:   EnvStore{"KEY": `a\\b`, "OTHER": `c\d`},
	})

	cfg = NewParseConfig().WithRecommendedDefaults().WithBaseOptions(&CustomParseOptions{UnescapeBackslashBackslash: new(false)}).
		WithKeyOptions("KEY", &CustomParseOptions{UnescapeBackslashBackslash: new(true)})

	run(t, nil, cfg, testCase{
		name:   "Obey different key-specific and base UnescapeBackslashBackslash settings",
		dotenv: "KEY=\"a\\\\b\"\nOTHER=\"c\\\\d\"",
		want:   EnvStore{"KEY": `a\b`, "OTHER": `c\\d`},
	})

	cfg = NewParseConfig().WithRecommendedDefaults().
		WithKeyOptions("T", &CustomParseOptions{UnescapeBackslashBackslash: new(true)}).
		WithKeyOptions("F", &CustomParseOptions{UnescapeBackslashBackslash: new(false)})

	run(t, nil, cfg, testCase{
		name:   "obey different key-specific UnescapeBackslashBackslash settings",
		dotenv: "T=\"a\\\\b\"\nF=\"c\\\\d\"",
		want:   EnvStore{"T": `a\b`, "F": `c\\d`},
	})
}

// ---------------------------------------------------------------------------
// UnescapeBackslashDoubleQuote Option Tests
// ---------------------------------------------------------------------------
// 	- Default | UnescapeBackslashDoubleQuote: true
// 		- if the value contains escaped double quotes, they should be unescaped
// 	- UnescapeBackslashDoubleQuote: false:
// 		- if the value contains escaped double quotes, they should be preserved
// ---------------------------------------------------------------------------

func TestConfigOptionsUnescapeBackslashDoubleQuote(t *testing.T) {
	defaultCfg := NewParseConfig().WithRecommendedDefaults()

	run(t, nil, defaultCfg, testCase{
		name:   "DEFAULT: escaped double quote is unescaped",
		dotenv: "KEY=\"a\\\"b\"",
		want:   EnvStore{"KEY": `a"b`},
	})

	cfg := NewParseConfig().WithRecommendedDefaults().WithBaseOptions(&CustomParseOptions{UnescapeBackslashDoubleQuote: new(true)})

	run(t, nil, cfg, testCase{
		name:   "UnescapeBackslashDoubleQuote: true (same as default) - escaped double quote is unescaped",
		dotenv: "KEY=\"a\\\"b\"",
		want:   EnvStore{"KEY": `a"b`},
	})

	cfg = NewParseConfig().WithRecommendedDefaults().WithBaseOptions(&CustomParseOptions{UnescapeBackslashDoubleQuote: new(false)})

	run(t, nil, cfg, testCase{
		name:   "UnescapeBackslashDoubleQuote: false - escaped double quote is preserved",
		dotenv: "KEY=\"a\\\"b\"",
		want:   EnvStore{"KEY": `a\"b`},
	})

	cfg = NewParseConfig().WithRecommendedDefaults().
		WithKeyOptions("KEY", &CustomParseOptions{UnescapeBackslashDoubleQuote: new(false)})

	run(t, nil, cfg, testCase{
		name:   "Key-specific UnescapeBackslashDoubleQuote",
		dotenv: "KEY=\"a\\\"b\"\nOTHER=\"c\\\"d\"",
		want:   EnvStore{"KEY": `a\"b`, "OTHER": `c"d`},
	})

	cfg = NewParseConfig().WithRecommendedDefaults().WithBaseOptions(&CustomParseOptions{UnescapeBackslashDoubleQuote: new(false)}).
		WithKeyOptions("KEY", &CustomParseOptions{UnescapeBackslashDoubleQuote: new(true)})

	run(t, nil, cfg, testCase{
		name:   "Obey different key-specific and base UnescapeBackslashDoubleQuote settings",
		dotenv: "KEY=\"a\\\"b\"\nOTHER=\"c\\\"d\"",
		want:   EnvStore{"KEY": `a"b`, "OTHER": `c\"d`},
	})

	cfg = NewParseConfig().WithRecommendedDefaults().
		WithKeyOptions("T", &CustomParseOptions{UnescapeBackslashDoubleQuote: new(true)}).
		WithKeyOptions("F", &CustomParseOptions{UnescapeBackslashDoubleQuote: new(false)})

	run(t, nil, cfg, testCase{
		name:   "obey different key-specific UnescapeBackslashDoubleQuote settings",
		dotenv: "T=\"a\\\"b\"\nF=\"c\\\"d\"",
		want:   EnvStore{"T": `a"b`, "F": `c\"d`},
	})
}

// ---------------------------------------------------------------------------
// UnescapeBackslashSingleQuote Option Tests
// ---------------------------------------------------------------------------
// 	- Default | UnescapeBackslashSingleQuote: false
// 		- if the value contains escaped single quotes, they should be preserved
// 	- UnescapeBackslashSingleQuote: true:
// 		- if the value contains escaped single quotes, they should be unescaped
// ---------------------------------------------------------------------------

func TestConfigOptionsUnescapeBackslashSingleQuote(t *testing.T) {
	defaultCfg := NewParseConfig().WithRecommendedDefaults()

	run(t, nil, defaultCfg, testCase{
		name:   "DEFAULT: escaped single quote is preserved",
		dotenv: "KEY=\"a\\'b\"",
		want:   EnvStore{"KEY": `a\'b`},
	})

	cfg := NewParseConfig().WithRecommendedDefaults().WithBaseOptions(&CustomParseOptions{UnescapeBackslashSingleQuote: new(false)})

	run(t, nil, cfg, testCase{
		name:   "UnescapeBackslashSingleQuote: false (same as default) - escaped single quote is preserved",
		dotenv: "KEY=\"a\\'b\"",
		want:   EnvStore{"KEY": `a\'b`},
	})

	cfg = NewParseConfig().WithRecommendedDefaults().WithBaseOptions(&CustomParseOptions{UnescapeBackslashSingleQuote: new(true)})

	run(t, nil, cfg, testCase{
		name:   "UnescapeBackslashSingleQuote: true - escaped single quote is unescaped",
		dotenv: "KEY=\"a\\'b\"",
		want:   EnvStore{"KEY": "a'b"},
	})

	cfg = NewParseConfig().WithRecommendedDefaults().
		WithKeyOptions("KEY", &CustomParseOptions{UnescapeBackslashSingleQuote: new(true)})

	run(t, nil, cfg, testCase{
		name:   "Key-specific UnescapeBackslashSingleQuote",
		dotenv: "KEY=\"a\\'b\"\nOTHER=\"c\\'d\"",
		want:   EnvStore{"KEY": "a'b", "OTHER": `c\'d`},
	})

	cfg = NewParseConfig().WithRecommendedDefaults().WithBaseOptions(&CustomParseOptions{UnescapeBackslashSingleQuote: new(true)}).
		WithKeyOptions("KEY", &CustomParseOptions{UnescapeBackslashSingleQuote: new(false)})

	run(t, nil, cfg, testCase{
		name:   "Obey different key-specific and base UnescapeBackslashSingleQuote settings",
		dotenv: "KEY=\"a\\'b\"\nOTHER=\"c\\'d\"",
		want:   EnvStore{"KEY": `a\'b`, "OTHER": "c'd"},
	})

	cfg = NewParseConfig().WithRecommendedDefaults().
		WithKeyOptions("T", &CustomParseOptions{UnescapeBackslashSingleQuote: new(true)}).
		WithKeyOptions("F", &CustomParseOptions{UnescapeBackslashSingleQuote: new(false)})

	run(t, nil, cfg, testCase{
		name:   "obey different key-specific UnescapeBackslashSingleQuote settings",
		dotenv: "T=\"a\\'b\"\nF=\"c\\'d\"",
		want:   EnvStore{"T": "a'b", "F": `c\'d`},
	})
}

// ---------------------------------------------------------------------------
// UnescapeBackslashA Option Tests
// ---------------------------------------------------------------------------
// 	- Default | UnescapeBackslashA: false
// 		- if the value contains \a, it should be preserved
// 	- UnescapeBackslashA: true:
// 		- if the value contains \a, it should be unescaped to a bell character
// ---------------------------------------------------------------------------

func TestConfigOptionsUnescapeBackslashA(t *testing.T) {
	defaultCfg := NewParseConfig().WithRecommendedDefaults()

	run(t, nil, defaultCfg, testCase{
		name:   "DEFAULT: escaped alert is preserved",
		dotenv: "KEY=\"a\\ab\"",
		want:   EnvStore{"KEY": `a\ab`},
	})

	cfg := NewParseConfig().WithRecommendedDefaults().WithBaseOptions(&CustomParseOptions{UnescapeBackslashA: new(false)})

	run(t, nil, cfg, testCase{
		name:   "UnescapeBackslashA: false (same as default) - escaped alert is preserved",
		dotenv: "KEY=\"a\\ab\"",
		want:   EnvStore{"KEY": `a\ab`},
	})

	cfg = NewParseConfig().WithRecommendedDefaults().WithBaseOptions(&CustomParseOptions{UnescapeBackslashA: new(true)})

	run(t, nil, cfg, testCase{
		name:   "UnescapeBackslashA: true - escaped alert is unescaped",
		dotenv: "KEY=\"a\\ab\"",
		want:   EnvStore{"KEY": "a\ab"},
	})

	cfg = NewParseConfig().WithRecommendedDefaults().
		WithKeyOptions("KEY", &CustomParseOptions{UnescapeBackslashA: new(true)})

	run(t, nil, cfg, testCase{
		name:   "Key-specific UnescapeBackslashA",
		dotenv: "KEY=\"a\\ab\"\nOTHER=\"c\\ad\"",
		want:   EnvStore{"KEY": "a\ab", "OTHER": `c\ad`},
	})

	cfg = NewParseConfig().WithRecommendedDefaults().WithBaseOptions(&CustomParseOptions{UnescapeBackslashA: new(true)}).
		WithKeyOptions("KEY", &CustomParseOptions{UnescapeBackslashA: new(false)})

	run(t, nil, cfg, testCase{
		name:   "Obey different key-specific and base UnescapeBackslashA settings",
		dotenv: "KEY=\"a\\ab\"\nOTHER=\"c\\ad\"",
		want:   EnvStore{"KEY": `a\ab`, "OTHER": "c\ad"},
	})

	cfg = NewParseConfig().WithRecommendedDefaults().
		WithKeyOptions("T", &CustomParseOptions{UnescapeBackslashA: new(true)}).
		WithKeyOptions("F", &CustomParseOptions{UnescapeBackslashA: new(false)})

	run(t, nil, cfg, testCase{
		name:   "obey different key-specific UnescapeBackslashA settings",
		dotenv: "T=\"a\\ab\"\nF=\"c\\ad\"",
		want:   EnvStore{"T": "a\ab", "F": `c\ad`},
	})
}

// ---------------------------------------------------------------------------
// UnescapeBackslashB Option Tests
// ---------------------------------------------------------------------------
// 	- Default | UnescapeBackslashB false
// 		- if the value contains \b, it should be preserved
// 	- UnescapeBackslashB true:
// 		- if the value contains \b, it should be unescaped to a backspace character
// ---------------------------------------------------------------------------

func TestConfigOptionsUnescapeBackslashB(t *testing.T) {
	defaultCfg := NewParseConfig().WithRecommendedDefaults()

	run(t, nil, defaultCfg, testCase{
		name:   "DEFAULT: escaped backspace is preserved",
		dotenv: "KEY=\"a\\bb\"",
		want:   EnvStore{"KEY": `a\bb`},
	})

	cfg := NewParseConfig().WithRecommendedDefaults().WithBaseOptions(&CustomParseOptions{UnescapeBackslashB: new(false)})

	run(t, nil, cfg, testCase{
		name:   "UnescapeBackslashB: false (same as default) - escaped backspace is preserved",
		dotenv: "KEY=\"a\\bb\"",
		want:   EnvStore{"KEY": `a\bb`},
	})

	cfg = NewParseConfig().WithRecommendedDefaults().WithBaseOptions(&CustomParseOptions{UnescapeBackslashB: new(true)})

	run(t, nil, cfg, testCase{
		name:   "UnescapeBackslashB: true - escaped backspace is unescaped",
		dotenv: "KEY=\"a\\bb\"",
		want:   EnvStore{"KEY": "a\bb"},
	})

	cfg = NewParseConfig().WithRecommendedDefaults().
		WithKeyOptions("KEY", &CustomParseOptions{UnescapeBackslashB: new(true)})

	run(t, nil, cfg, testCase{
		name:   "Key-specific UnescapeBackslashB",
		dotenv: "KEY=\"a\\bb\"\nOTHER=\"c\\bd\"",
		want:   EnvStore{"KEY": "a\bb", "OTHER": `c\bd`},
	})

	cfg = NewParseConfig().WithRecommendedDefaults().WithBaseOptions(&CustomParseOptions{UnescapeBackslashB: new(true)}).
		WithKeyOptions("KEY", &CustomParseOptions{UnescapeBackslashB: new(false)})

	run(t, nil, cfg, testCase{
		name:   "Obey different key-specific and base UnescapeBackslashB settings",
		dotenv: "KEY=\"a\\bb\"\nOTHER=\"c\\bd\"",
		want:   EnvStore{"KEY": `a\bb`, "OTHER": "c\bd"},
	})

	cfg = NewParseConfig().WithRecommendedDefaults().
		WithKeyOptions("T", &CustomParseOptions{UnescapeBackslashB: new(true)}).
		WithKeyOptions("F", &CustomParseOptions{UnescapeBackslashB: new(false)})

	run(t, nil, cfg, testCase{
		name:   "obey different key-specific UnescapeBackslashB settings",
		dotenv: "T=\"a\\bb\"\nF=\"c\\bd\"",
		want:   EnvStore{"T": "a\bb", "F": `c\bd`},
	})
}

// ---------------------------------------------------------------------------
// UnescapeBackslashF Option Tests
// ---------------------------------------------------------------------------
// 	- Default | UnescapeBackslashF: false
// 		- if the value contains \f, it should be preserved
// 	- UnescapeBackslashF: true:
// 		- if the value contains \f, it should be unescaped to a form feed character
// ---------------------------------------------------------------------------

func TestConfigOptionsUnescapeBackslashF(t *testing.T) {
	defaultCfg := NewParseConfig().WithRecommendedDefaults()

	run(t, nil, defaultCfg, testCase{
		name:   "DEFAULT: escaped form feed is preserved",
		dotenv: "KEY=\"a\\fb\"",
		want:   EnvStore{"KEY": `a\fb`},
	})

	cfg := NewParseConfig().WithRecommendedDefaults().WithBaseOptions(&CustomParseOptions{UnescapeBackslashF: new(false)})

	run(t, nil, cfg, testCase{
		name:   "UnescapeBackslashF: false (same as default) - escaped form feed is preserved",
		dotenv: "KEY=\"a\\fb\"",
		want:   EnvStore{"KEY": `a\fb`},
	})

	cfg = NewParseConfig().WithRecommendedDefaults().WithBaseOptions(&CustomParseOptions{UnescapeBackslashF: new(true)})

	run(t, nil, cfg, testCase{
		name:   "UnescapeBackslashF: true - escaped form feed is unescaped",
		dotenv: "KEY=\"a\\fb\"",
		want:   EnvStore{"KEY": "a\fb"},
	})

	cfg = NewParseConfig().WithRecommendedDefaults().
		WithKeyOptions("KEY", &CustomParseOptions{UnescapeBackslashF: new(true)})

	run(t, nil, cfg, testCase{
		name:   "Key-specific UnescapeBackslashF",
		dotenv: "KEY=\"a\\fb\"\nOTHER=\"c\\fd\"",
		want:   EnvStore{"KEY": "a\fb", "OTHER": `c\fd`},
	})

	cfg = NewParseConfig().WithRecommendedDefaults().WithBaseOptions(&CustomParseOptions{UnescapeBackslashF: new(true)}).
		WithKeyOptions("KEY", &CustomParseOptions{UnescapeBackslashF: new(false)})

	run(t, nil, cfg, testCase{
		name:   "Obey different key-specific and base UnescapeBackslashF settings",
		dotenv: "KEY=\"a\\fb\"\nOTHER=\"c\\fd\"",
		want:   EnvStore{"KEY": `a\fb`, "OTHER": "c\fd"},
	})

	cfg = NewParseConfig().WithRecommendedDefaults().
		WithKeyOptions("T", &CustomParseOptions{UnescapeBackslashF: new(true)}).
		WithKeyOptions("F", &CustomParseOptions{UnescapeBackslashF: new(false)})

	run(t, nil, cfg, testCase{
		name:   "obey different key-specific UnescapeBackslashF settings",
		dotenv: "T=\"a\\fb\"\nF=\"c\\fd\"",
		want:   EnvStore{"T": "a\fb", "F": `c\fd`},
	})
}

// ---------------------------------------------------------------------------
// UnescapeBackslashN Option Tests
// ---------------------------------------------------------------------------
// 	- Default | UnescapeBackslashN: true
// 		- if the value contains \n, it should be unescaped to a newline
// 	- UnescapeBackslashN: false:
// 		- if the value contains \n, it should be preserved
// ---------------------------------------------------------------------------

func TestConfigOptionsUnescapeBackslashN(t *testing.T) {
	defaultCfg := NewParseConfig().WithRecommendedDefaults()

	run(t, nil, defaultCfg, testCase{
		name:   "DEFAULT: escaped newline is unescaped",
		dotenv: "KEY=\"a\\nb\"",
		want:   EnvStore{"KEY": "a\nb"},
	})

	cfg := NewParseConfig().WithRecommendedDefaults().WithBaseOptions(&CustomParseOptions{UnescapeBackslashN: new(true)})

	run(t, nil, cfg, testCase{
		name:   "UnescapeBackslashN: true (same as default) - escaped newline is unescaped",
		dotenv: "KEY=\"a\\nb\"",
		want:   EnvStore{"KEY": "a\nb"},
	})

	cfg = NewParseConfig().WithRecommendedDefaults().WithBaseOptions(&CustomParseOptions{UnescapeBackslashN: new(false)})

	run(t, nil, cfg, testCase{
		name:   "UnescapeBackslashN: false - escaped newline is preserved",
		dotenv: "KEY=\"a\\nb\"",
		want:   EnvStore{"KEY": `a\nb`},
	})

	cfg = NewParseConfig().WithRecommendedDefaults().
		WithKeyOptions("KEY", &CustomParseOptions{UnescapeBackslashN: new(false)})

	run(t, nil, cfg, testCase{
		name:   "Key-specific UnescapeBackslashN",
		dotenv: "KEY=\"a\\nb\"\nOTHER=\"c\\nd\"",
		want:   EnvStore{"KEY": `a\nb`, "OTHER": "c\nd"},
	})

	cfg = NewParseConfig().WithRecommendedDefaults().WithBaseOptions(&CustomParseOptions{UnescapeBackslashN: new(false)}).
		WithKeyOptions("KEY", &CustomParseOptions{UnescapeBackslashN: new(true)})

	run(t, nil, cfg, testCase{
		name:   "Obey different key-specific and base UnescapeBackslashN settings",
		dotenv: "KEY=\"a\\nb\"\nOTHER=\"c\\nd\"",
		want:   EnvStore{"KEY": "a\nb", "OTHER": `c\nd`},
	})

	cfg = NewParseConfig().WithRecommendedDefaults().
		WithKeyOptions("T", &CustomParseOptions{UnescapeBackslashN: new(true)}).
		WithKeyOptions("F", &CustomParseOptions{UnescapeBackslashN: new(false)})

	run(t, nil, cfg, testCase{
		name:   "obey different key-specific UnescapeBackslashN settings",
		dotenv: "T=\"a\\nb\"\nF=\"c\\nd\"",
		want:   EnvStore{"T": "a\nb", "F": `c\nd`},
	})
}

// ---------------------------------------------------------------------------
// UnescapeBackslashR Option Tests
// ---------------------------------------------------------------------------
// 	- Default | UnescapeBackslashR: true
// 		- if the value contains \r, it should be unescaped
// 	- UnescapeBackslashR: false:
// 		- if the value contains \r, it should be preserved
// ---------------------------------------------------------------------------

func TestConfigOptionsUnescapeBackslashR(t *testing.T) {
	defaultCfg := NewParseConfig().WithRecommendedDefaults()

	run(t, nil, defaultCfg, testCase{
		name:   "DEFAULT: escaped carriage return is unescaped and normalized",
		dotenv: "KEY=\"a\\rb\"",
		want:   EnvStore{"KEY": "a\nb"},
	})

	cfg := NewParseConfig().WithRecommendedDefaults().WithBaseOptions(&CustomParseOptions{UnescapeBackslashR: new(true)})

	run(t, nil, cfg, testCase{
		name:   "UnescapeBackslashR: true (same as default) - escaped carriage return is unescaped and normalized",
		dotenv: "KEY=\"a\\rb\"",
		want:   EnvStore{"KEY": "a\nb"},
	})

	cfg = NewParseConfig().WithRecommendedDefaults().WithBaseOptions(&CustomParseOptions{UnescapeBackslashR: new(false)})

	run(t, nil, cfg, testCase{
		name:   "UnescapeBackslashR: false - escaped carriage return is preserved",
		dotenv: "KEY=\"a\\rb\"",
		want:   EnvStore{"KEY": `a\rb`},
	})

	cfg = NewParseConfig().WithRecommendedDefaults().
		WithKeyOptions("KEY", &CustomParseOptions{UnescapeBackslashR: new(false)})

	run(t, nil, cfg, testCase{
		name:   "Key-specific UnescapeBackslashR",
		dotenv: "KEY=\"a\\rb\"\nOTHER=\"c\\rd\"",
		want:   EnvStore{"KEY": `a\rb`, "OTHER": "c\nd"},
	})

	cfg = NewParseConfig().WithRecommendedDefaults().WithBaseOptions(&CustomParseOptions{UnescapeBackslashR: new(false)}).
		WithKeyOptions("KEY", &CustomParseOptions{UnescapeBackslashR: new(true)})

	run(t, nil, cfg, testCase{
		name:   "Obey different key-specific and base UnescapeBackslashR settings",
		dotenv: "KEY=\"a\\rb\"\nOTHER=\"c\\rd\"",
		want:   EnvStore{"KEY": "a\nb", "OTHER": `c\rd`},
	})

	cfg = NewParseConfig().WithRecommendedDefaults().
		WithKeyOptions("T", &CustomParseOptions{UnescapeBackslashR: new(true)}).
		WithKeyOptions("F", &CustomParseOptions{UnescapeBackslashR: new(false)})

	run(t, nil, cfg, testCase{
		name:   "obey different key-specific UnescapeBackslashR settings",
		dotenv: "T=\"a\\rb\"\nF=\"c\\rd\"",
		want:   EnvStore{"T": "a\nb", "F": `c\rd`},
	})
}

// ---------------------------------------------------------------------------
// UnescapeBackslashT Option Tests
// ---------------------------------------------------------------------------
// 	- Default | UnescapeBackslashT: true
// 		- if the value contains \t, it should be unescaped to a tab
// 	- UnescapeBackslashT: false:
// 		- if the value contains \t, it should be preserved
// ---------------------------------------------------------------------------

func TestConfigOptionsUnescapeBackslashT(t *testing.T) {
	defaultCfg := NewParseConfig().WithRecommendedDefaults()

	run(t, nil, defaultCfg, testCase{
		name:   "DEFAULT: escaped tab is unescaped",
		dotenv: "KEY=\"a\\tb\"",
		want:   EnvStore{"KEY": "a\tb"},
	})

	cfg := NewParseConfig().WithRecommendedDefaults().WithBaseOptions(&CustomParseOptions{UnescapeBackslashT: new(true)})

	run(t, nil, cfg, testCase{
		name:   "UnescapeBackslashT: true (same as default) - escaped tab is unescaped",
		dotenv: "KEY=\"a\\tb\"",
		want:   EnvStore{"KEY": "a\tb"},
	})

	cfg = NewParseConfig().WithRecommendedDefaults().WithBaseOptions(&CustomParseOptions{UnescapeBackslashT: new(false)})

	run(t, nil, cfg, testCase{
		name:   "UnescapeBackslashT: false - escaped tab is preserved",
		dotenv: "KEY=\"a\\tb\"",
		want:   EnvStore{"KEY": `a\tb`},
	})

	cfg = NewParseConfig().WithRecommendedDefaults().
		WithKeyOptions("KEY", &CustomParseOptions{UnescapeBackslashT: new(false)})

	run(t, nil, cfg, testCase{
		name:   "Key-specific UnescapeBackslashT",
		dotenv: "KEY=\"a\\tb\"\nOTHER=\"c\\td\"",
		want:   EnvStore{"KEY": `a\tb`, "OTHER": "c\td"},
	})

	cfg = NewParseConfig().WithRecommendedDefaults().WithBaseOptions(&CustomParseOptions{UnescapeBackslashT: new(false)}).
		WithKeyOptions("KEY", &CustomParseOptions{UnescapeBackslashT: new(true)})

	run(t, nil, cfg, testCase{
		name:   "Obey different key-specific and base UnescapeBackslashT settings",
		dotenv: "KEY=\"a\\tb\"\nOTHER=\"c\\td\"",
		want:   EnvStore{"KEY": "a\tb", "OTHER": `c\td`},
	})

	cfg = NewParseConfig().WithRecommendedDefaults().
		WithKeyOptions("T", &CustomParseOptions{UnescapeBackslashT: new(true)}).
		WithKeyOptions("F", &CustomParseOptions{UnescapeBackslashT: new(false)})

	run(t, nil, cfg, testCase{
		name:   "obey different key-specific UnescapeBackslashT settings",
		dotenv: "T=\"a\\tb\"\nF=\"c\\td\"",
		want:   EnvStore{"T": "a\tb", "F": `c\td`},
	})
}

// ---------------------------------------------------------------------------
// UnescapeBackslashV Option Tests
// ---------------------------------------------------------------------------
// 	- Default | UnescapeBackslashV: false
// 		- if the value contains \v, it should be preserved
// 	- UnescapeBackslashV: true:
// 		- if the value contains \v, it should be unescaped to a vertical tab
// ---------------------------------------------------------------------------

func TestConfigOptionsUnescapeBackslashV(t *testing.T) {
	defaultCfg := NewParseConfig().WithRecommendedDefaults()

	run(t, nil, defaultCfg, testCase{
		name:   "DEFAULT: escaped vertical tab is preserved",
		dotenv: "KEY=\"a\\vb\"",
		want:   EnvStore{"KEY": `a\vb`},
	})

	cfg := NewParseConfig().WithRecommendedDefaults().WithBaseOptions(&CustomParseOptions{UnescapeBackslashV: new(false)})

	run(t, nil, cfg, testCase{
		name:   "UnescapeBackslashV: false (same as default) - escaped vertical tab is preserved",
		dotenv: "KEY=\"a\\vb\"",
		want:   EnvStore{"KEY": `a\vb`},
	})

	cfg = NewParseConfig().WithRecommendedDefaults().WithBaseOptions(&CustomParseOptions{UnescapeBackslashV: new(true)})

	run(t, nil, cfg, testCase{
		name:   "UnescapeBackslashV: true - escaped vertical tab is unescaped",
		dotenv: "KEY=\"a\\vb\"",
		want:   EnvStore{"KEY": "a\vb"},
	})

	cfg = NewParseConfig().WithRecommendedDefaults().
		WithKeyOptions("KEY", &CustomParseOptions{UnescapeBackslashV: new(true)})

	run(t, nil, cfg, testCase{
		name:   "Key-specific UnescapeBackslashV",
		dotenv: "KEY=\"a\\vb\"\nOTHER=\"c\\vd\"",
		want:   EnvStore{"KEY": "a\vb", "OTHER": `c\vd`},
	})

	cfg = NewParseConfig().WithRecommendedDefaults().WithBaseOptions(&CustomParseOptions{UnescapeBackslashV: new(true)}).
		WithKeyOptions("KEY", &CustomParseOptions{UnescapeBackslashV: new(false)})

	run(t, nil, cfg, testCase{
		name:   "Obey different key-specific and base UnescapeBackslashV settings",
		dotenv: "KEY=\"a\\vb\"\nOTHER=\"c\\vd\"",
		want:   EnvStore{"KEY": `a\vb`, "OTHER": "c\vd"},
	})

	cfg = NewParseConfig().WithRecommendedDefaults().
		WithKeyOptions("T", &CustomParseOptions{UnescapeBackslashV: new(true)}).
		WithKeyOptions("F", &CustomParseOptions{UnescapeBackslashV: new(false)})

	run(t, nil, cfg, testCase{
		name:   "obey different key-specific UnescapeBackslashV settings",
		dotenv: "T=\"a\\vb\"\nF=\"c\\vd\"",
		want:   EnvStore{"T": "a\vb", "F": `c\vd`},
	})
}

// ---------------------------------------------------------------------------
// TransformCRLFToLF Option Tests
// ---------------------------------------------------------------------------
// 	- Default | TransformCRLFToLF: true
// 		- if the value contains CRLF, it should normalize to a single LF
// 	- TransformCRLFToLF: false:
// 		- if the value contains CRLF and TransformCRToLF is true, it becomes two LFs
// ---------------------------------------------------------------------------

func TestConfigOptionsTransformCRLFToLF(t *testing.T) {
	defaultCfg := NewParseConfig().WithRecommendedDefaults()

	run(t, nil, defaultCfg, testCase{
		name:   "DEFAULT: CRLF is normalized to a single LF",
		dotenv: "KEY=\"a\r\nb\"",
		want:   EnvStore{"KEY": "a\nb"},
	})

	cfg := NewParseConfig().WithRecommendedDefaults().WithBaseOptions(&CustomParseOptions{TransformCRLFToLF: new(true)})

	run(t, nil, cfg, testCase{
		name:   "TransformCRLFToLF: true (same as default) - CRLF is normalized to a single LF",
		dotenv: "KEY=\"a\r\nb\"",
		want:   EnvStore{"KEY": "a\nb"},
	})

	cfg = NewParseConfig().WithRecommendedDefaults().WithBaseOptions(&CustomParseOptions{TransformCRLFToLF: new(false)})

	run(t, nil, cfg, testCase{
		name:   "TransformCRLFToLF: false - CRLF expands to two LFs when CR transform is enabled",
		dotenv: "KEY=\"a\r\nb\"",
		want:   EnvStore{"KEY": "a\n\nb"},
	})

	cfg = NewParseConfig().WithRecommendedDefaults().
		WithKeyOptions("KEY", &CustomParseOptions{TransformCRLFToLF: new(false)})

	run(t, nil, cfg, testCase{
		name:   "Key-specific TransformCRLFToLF",
		dotenv: "KEY=\"a\r\nb\"\nOTHER=\"c\r\nd\"",
		want:   EnvStore{"KEY": "a\n\nb", "OTHER": "c\nd"},
	})

	cfg = NewParseConfig().WithRecommendedDefaults().WithBaseOptions(&CustomParseOptions{TransformCRLFToLF: new(false)}).
		WithKeyOptions("KEY", &CustomParseOptions{TransformCRLFToLF: new(true)})

	run(t, nil, cfg, testCase{
		name:   "Obey different key-specific and base TransformCRLFToLF settings",
		dotenv: "KEY=\"a\r\nb\"\nOTHER=\"c\r\nd\"",
		want:   EnvStore{"KEY": "a\nb", "OTHER": "c\n\nd"},
	})

	cfg = NewParseConfig().WithRecommendedDefaults().
		WithKeyOptions("T", &CustomParseOptions{TransformCRLFToLF: new(true)}).
		WithKeyOptions("F", &CustomParseOptions{TransformCRLFToLF: new(false)})

	run(t, nil, cfg, testCase{
		name:   "obey different key-specific TransformCRLFToLF settings",
		dotenv: "T=\"a\r\nb\"\nF=\"c\r\nd\"",
		want:   EnvStore{"T": "a\nb", "F": "c\n\nd"},
	})
}

// ---------------------------------------------------------------------------
// TransformCRToLF Option Tests
// ---------------------------------------------------------------------------
// 	- Default | TransformCRToLF: true
// 		- if the value contains CR, it should normalize to LF
// 	- TransformCRToLF: false:
// 		- if the value contains CR, it should be preserved
// ---------------------------------------------------------------------------

func TestConfigOptionsTransformCRToLF(t *testing.T) {
	defaultCfg := NewParseConfig().WithRecommendedDefaults()

	run(t, nil, defaultCfg, testCase{
		name:   "DEFAULT: CR is normalized to LF",
		dotenv: "KEY=\"a\rb\"",
		want:   EnvStore{"KEY": "a\nb"},
	})

	cfg := NewParseConfig().WithRecommendedDefaults().WithBaseOptions(&CustomParseOptions{TransformCRToLF: new(true)})

	run(t, nil, cfg, testCase{
		name:   "TransformCRToLF: true (same as default) - CR is normalized to LF",
		dotenv: "KEY=\"a\rb\"",
		want:   EnvStore{"KEY": "a\nb"},
	})

	cfg = NewParseConfig().WithRecommendedDefaults().WithBaseOptions(&CustomParseOptions{TransformCRToLF: new(false)})

	run(t, nil, cfg, testCase{
		name:   "TransformCRToLF: false - CR is preserved",
		dotenv: "KEY=\"a\rb\"",
		want:   EnvStore{"KEY": "a\rb"},
	})

	cfg = NewParseConfig().WithRecommendedDefaults().
		WithKeyOptions("KEY", &CustomParseOptions{TransformCRToLF: new(false)})

	run(t, nil, cfg, testCase{
		name:   "Key-specific TransformCRToLF",
		dotenv: "KEY=\"a\rb\"\nOTHER=\"c\rd\"",
		want:   EnvStore{"KEY": "a\rb", "OTHER": "c\nd"},
	})

	cfg = NewParseConfig().WithRecommendedDefaults().WithBaseOptions(&CustomParseOptions{TransformCRToLF: new(false)}).
		WithKeyOptions("KEY", &CustomParseOptions{TransformCRToLF: new(true)})

	run(t, nil, cfg, testCase{
		name:   "Obey different key-specific and base TransformCRToLF settings",
		dotenv: "KEY=\"a\rb\"\nOTHER=\"c\rd\"",
		want:   EnvStore{"KEY": "a\nb", "OTHER": "c\rd"},
	})

	cfg = NewParseConfig().WithRecommendedDefaults().
		WithKeyOptions("T", &CustomParseOptions{TransformCRToLF: new(true)}).
		WithKeyOptions("F", &CustomParseOptions{TransformCRToLF: new(false)})

	run(t, nil, cfg, testCase{
		name:   "obey different key-specific TransformCRToLF settings",
		dotenv: "T=\"a\rb\"\nF=\"c\rd\"",
		want:   EnvStore{"T": "a\nb", "F": "c\rd"},
	})
}

// ---------------------------------------------------------------------------
// Additional Tests for Escaped + Literal CR +LF Interactions
// ---------------------------------------------------------------------------

func TestConfigOptionsTransformCRLFToLFAndTransformCRToLFInteractions(t *testing.T) {
	cfg := NewParseConfig().WithRecommendedDefaults().WithBaseOptions(&CustomParseOptions{
		TransformCRLFToLF: new(false),
		TransformCRToLF:   new(false),
	})

	run(t, nil, cfg, testCase{
		name:   "TransformCRLFToLF false and TransformCRToLF false - CRLF is preserved as CRLF",
		dotenv: "KEY=\"a\r\nb\"\nKEY2=\"c\\r\\nd\"",
		want:   EnvStore{"KEY": "a\r\nb", "KEY2": "c\r\nd"},
	})

	cfg = NewParseConfig().WithRecommendedDefaults().WithBaseOptions(&CustomParseOptions{
		UnescapeBackslashN: new(false),
		UnescapeBackslashR: new(false),
		TransformCRLFToLF:  new(false),
		TransformCRToLF:    new(false),
	})

	run(t, nil, cfg, testCase{
		name:   "No Unescape newlines or transforms",
		dotenv: "KEY=\"a\r\nb\"\nKEY2=\"c\\r\\nd\"",
		want:   EnvStore{"KEY": "a\r\nb", "KEY2": "c\\r\\nd"},
	})
}
