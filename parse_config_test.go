package strictdotenv

import "testing"

func TestParseConfigZeroValue(t *testing.T) {
	cfg := new(ParseConfig)

	if cfg.GlobalOptions != (ParseOptions{}) {
		t.Errorf("GlobalOptions = %+v, want zero ParseOptions", cfg.GlobalOptions)
	}
	if cfg.KeyOptions != nil {
		t.Errorf("KeyOptions = %v, want nil", cfg.KeyOptions)
	}
	if cfg.resolvedGlobalOptions != (resolvedParseOptions{}) {
		t.Errorf("resolvedGlobalOptions = %+v, want zero resolvedParseOptions", cfg.resolvedGlobalOptions)
	}
	if cfg.resolvedKeyOptions != nil {
		t.Errorf("resolvedKeyOptions = %v, want nil", cfg.resolvedKeyOptions)
	}
}

func TestParseConfigApplyGlobalOptionsUpdatesResolvedState(t *testing.T) {
	cfg := new(ParseConfig)

	cfg.ApplyGlobalOptions(&ParseOptions{
		Overwrite:          new(true),
		UnescapeBackslashN: new(true),
	})

	if cfg.GlobalOptions.Overwrite == nil || !*cfg.GlobalOptions.Overwrite {
		t.Fatal("expected GlobalOptions.Overwrite=true")
	}
	if cfg.GlobalOptions.UnescapeBackslashN == nil || !*cfg.GlobalOptions.UnescapeBackslashN {
		t.Fatal("expected GlobalOptions.UnescapeBackslashN=true")
	}
	if cfg.GlobalOptions.TransformCRToLF != nil {
		t.Fatal("expected unrelated GlobalOptions field to remain nil")
	}
	if !cfg.resolvedGlobalOptions.Overwrite {
		t.Fatal("expected resolvedGlobalOptions.Overwrite=true")
	}
	if !cfg.resolvedGlobalOptions.UnescapeBackslashN {
		t.Fatal("expected resolvedGlobalOptions.UnescapeBackslashN=true")
	}
	if cfg.resolvedGlobalOptions.TransformCRToLF {
		t.Fatal("expected resolvedGlobalOptions.TransformCRToLF=false")
	}
}

func TestParseConfigApplyGlobalOptionsClonesPointers(t *testing.T) {
	overwrite := true
	cfg := new(ParseConfig)

	cfg.ApplyGlobalOptions(&ParseOptions{Overwrite: &overwrite})
	overwrite = false

	if cfg.GlobalOptions.Overwrite == nil || !*cfg.GlobalOptions.Overwrite {
		t.Fatal("expected ParseConfig to keep its own copy of Overwrite=true")
	}
}

func TestParseConfigApplyGlobalOptionsRefreshesResolvedKeyOptions(t *testing.T) {
	cfg := new(ParseConfig)
	cfg.ApplyKeyOptions("KEY", &ParseOptions{UnescapeBackslashN: new(true)})

	cfg.ApplyGlobalOptions(&ParseOptions{Overwrite: new(true)})

	keyOptions, ok := cfg.resolvedKeyOptions["KEY"]
	if !ok {
		t.Fatal("expected KEY resolved options to be present")
	}
	if !keyOptions.Overwrite {
		t.Fatal("expected KEY to inherit resolved global Overwrite=true")
	}
	if !keyOptions.UnescapeBackslashN {
		t.Fatal("expected KEY-specific UnescapeBackslashN=true to be preserved")
	}
}

func TestParseConfigApplyKeyOptionsMergesSameKey(t *testing.T) {
	cfg := new(ParseConfig)
	cfg.ApplyGlobalOptions(&ParseOptions{Overwrite: new(true)})

	cfg.ApplyKeyOptions("KEY", &ParseOptions{UnescapeBackslashN: new(true)})
	cfg.ApplyKeyOptions("KEY", &ParseOptions{TransformCRToLF: new(true)})

	keyOptions := cfg.KeyOptions["KEY"]
	if keyOptions.UnescapeBackslashN == nil || !*keyOptions.UnescapeBackslashN {
		t.Fatal("expected KEY UnescapeBackslashN=true after merging key options")
	}
	if keyOptions.TransformCRToLF == nil || !*keyOptions.TransformCRToLF {
		t.Fatal("expected KEY TransformCRToLF=true after merging key options")
	}

	resolved := cfg.resolvedKeyOptions["KEY"]
	if !resolved.Overwrite {
		t.Fatal("expected KEY to inherit Overwrite=true from global options")
	}
	if !resolved.UnescapeBackslashN {
		t.Fatal("expected KEY resolved UnescapeBackslashN=true")
	}
	if !resolved.TransformCRToLF {
		t.Fatal("expected KEY resolved TransformCRToLF=true")
	}
}

func TestParseConfigApplyKeyOptionsNilOverridesUsesGlobalResolution(t *testing.T) {
	cfg := new(ParseConfig)
	cfg.ApplyGlobalOptions(&ParseOptions{
		Overwrite:          new(true),
		UnescapeBackslashN: new(true),
	})

	cfg.ApplyKeyOptions("KEY", nil)

	keyOptions, ok := cfg.KeyOptions["KEY"]
	if !ok {
		t.Fatal("expected KEY options to exist")
	}
	if keyOptions != (ParseOptions{}) {
		t.Fatalf("KEY options = %+v, want zero ParseOptions", keyOptions)
	}

	if cfg.resolvedKeyOptions["KEY"] != cfg.resolvedGlobalOptions {
		t.Fatalf("resolvedKeyOptions[KEY] = %+v, want %+v",
			cfg.resolvedKeyOptions["KEY"], cfg.resolvedGlobalOptions)
	}
}

func TestNilParseConfigUsesZeroValueOptions(t *testing.T) {
	run(t, nil, nil, testCase{
		name:   "nil config preserves literal escapes",
		dotenv: "KEY=\"line1\\nline2\"",
		want:   EnvStore{"KEY": `line1\nline2`},
	})
}

func TestParseConfigOverwrite(t *testing.T) {
	t.Run("all options disabled by default", func(t *testing.T) {
		cfg := new(ParseConfig)
		run(t, nil, cfg, testCase{
			name:   "repeated keys keep the first value",
			dotenv: "KEY=1\nKEY=2",
			want:   EnvStore{"KEY": "1"},
		})
	})

	t.Run("ApplyGlobalOptions enables overwrite", func(t *testing.T) {
		cfg := new(ParseConfig)
		cfg.ApplyGlobalOptions(&ParseOptions{Overwrite: new(true)})
		run(t, nil, cfg, testCase{
			name:   "repeated keys keep the last value",
			dotenv: "KEY=1\nKEY=2",
			want:   EnvStore{"KEY": "2"},
		})
	})

	t.Run("ApplyKeyOptions enables overwrite for one key", func(t *testing.T) {
		cfg := new(ParseConfig)
		cfg.ApplyKeyOptions("KEY", &ParseOptions{Overwrite: new(true)})
		run(t, nil, cfg, testCase{
			name:   "key-specific overwrite",
			dotenv: "KEY=1\nKEY=2\nOTHER=3\nOTHER=4",
			want:   EnvStore{"KEY": "2", "OTHER": "3"},
		})
	})

	t.Run("key-specific false overrides global true", func(t *testing.T) {
		cfg := new(ParseConfig)
		cfg.ApplyGlobalOptions(&ParseOptions{Overwrite: new(true)})
		cfg.ApplyKeyOptions("KEY", &ParseOptions{Overwrite: new(false)})
		run(t, nil, cfg, testCase{
			name:   "key-specific overwrite disable",
			dotenv: "KEY=1\nKEY=2\nOTHER=3\nOTHER=4",
			want:   EnvStore{"KEY": "1", "OTHER": "4"},
		})
	})
}

func TestParseConfigUnescapeBackslashBackslash(t *testing.T) {
	t.Run("all options disabled by default", func(t *testing.T) {
		cfg := new(ParseConfig)
		run(t, nil, cfg, testCase{
			name:   "escaped backslash stays literal",
			dotenv: "KEY=\"a\\\\b\"",
			want:   EnvStore{"KEY": `a\\b`},
		})
	})

	t.Run("ApplyGlobalOptions enables escaped backslash unescaping", func(t *testing.T) {
		cfg := new(ParseConfig)
		cfg.ApplyGlobalOptions(&ParseOptions{UnescapeBackslashBackslash: new(true)})
		run(t, nil, cfg, testCase{
			name:   "escaped backslash is unescaped",
			dotenv: "KEY=\"a\\\\b\"",
			want:   EnvStore{"KEY": `a\b`},
		})
	})

	t.Run("ApplyKeyOptions enables escaped backslash unescaping for one key", func(t *testing.T) {
		cfg := new(ParseConfig)
		cfg.ApplyKeyOptions("KEY", &ParseOptions{UnescapeBackslashBackslash: new(true)})
		run(t, nil, cfg, testCase{
			name:   "key-specific escaped backslash unescaping",
			dotenv: "KEY=\"a\\\\b\"\nOTHER=\"c\\\\d\"",
			want:   EnvStore{"KEY": `a\b`, "OTHER": `c\\d`},
		})
	})

	t.Run("key-specific false overrides global true", func(t *testing.T) {
		cfg := new(ParseConfig)
		cfg.ApplyGlobalOptions(&ParseOptions{UnescapeBackslashBackslash: new(true)})
		cfg.ApplyKeyOptions("KEY", &ParseOptions{UnescapeBackslashBackslash: new(false)})
		run(t, nil, cfg, testCase{
			name:   "disable escaped backslash unescaping for one key",
			dotenv: "KEY=\"a\\\\b\"\nOTHER=\"c\\\\d\"",
			want:   EnvStore{"KEY": `a\\b`, "OTHER": `c\d`},
		})
	})
}

func TestParseConfigUnescapeBackslashDoubleQuote(t *testing.T) {
	t.Run("all options disabled by default", func(t *testing.T) {
		cfg := new(ParseConfig)
		run(t, nil, cfg, testCase{
			name:   "escaped double quote stays literal",
			dotenv: "KEY=\"a\\\"b\"",
			want:   EnvStore{"KEY": `a\"b`},
		})
	})

	t.Run("ApplyGlobalOptions enables escaped double quote unescaping", func(t *testing.T) {
		cfg := new(ParseConfig)
		cfg.ApplyGlobalOptions(&ParseOptions{UnescapeBackslashDoubleQuote: new(true)})
		run(t, nil, cfg, testCase{
			name:   "escaped double quote is unescaped",
			dotenv: "KEY=\"a\\\"b\"",
			want:   EnvStore{"KEY": `a"b`},
		})
	})

	t.Run("ApplyKeyOptions enables escaped double quote unescaping for one key", func(t *testing.T) {
		cfg := new(ParseConfig)
		cfg.ApplyKeyOptions("KEY", &ParseOptions{UnescapeBackslashDoubleQuote: new(true)})
		run(t, nil, cfg, testCase{
			name:   "key-specific escaped double quote unescaping",
			dotenv: "KEY=\"a\\\"b\"\nOTHER=\"c\\\"d\"",
			want:   EnvStore{"KEY": `a"b`, "OTHER": `c\"d`},
		})
	})

	t.Run("key-specific false overrides global true", func(t *testing.T) {
		cfg := new(ParseConfig)
		cfg.ApplyGlobalOptions(&ParseOptions{UnescapeBackslashDoubleQuote: new(true)})
		cfg.ApplyKeyOptions("KEY", &ParseOptions{UnescapeBackslashDoubleQuote: new(false)})
		run(t, nil, cfg, testCase{
			name:   "disable escaped double quote unescaping for one key",
			dotenv: "KEY=\"a\\\"b\"\nOTHER=\"c\\\"d\"",
			want:   EnvStore{"KEY": `a\"b`, "OTHER": `c"d`},
		})
	})
}

func TestParseConfigUnescapeBackslashSingleQuote(t *testing.T) {
	t.Run("all options disabled by default", func(t *testing.T) {
		cfg := new(ParseConfig)
		run(t, nil, cfg, testCase{
			name:   "escaped single quote stays literal",
			dotenv: `KEY="a\'b"`,
			want:   EnvStore{"KEY": `a\'b`},
		})
	})

	t.Run("ApplyGlobalOptions enables escaped single quote unescaping", func(t *testing.T) {
		cfg := new(ParseConfig)
		cfg.ApplyGlobalOptions(&ParseOptions{UnescapeBackslashSingleQuote: new(true)})
		run(t, nil, cfg, testCase{
			name:   "escaped single quote is unescaped",
			dotenv: `KEY="a\'b"`,
			want:   EnvStore{"KEY": "a'b"},
		})
	})

	t.Run("ApplyKeyOptions enables escaped single quote unescaping for one key", func(t *testing.T) {
		cfg := new(ParseConfig)
		cfg.ApplyKeyOptions("KEY", &ParseOptions{UnescapeBackslashSingleQuote: new(true)})
		run(t, nil, cfg, testCase{
			name:   "key-specific escaped single quote unescaping",
			dotenv: "KEY=\"a\\'b\"\nOTHER=\"c\\'d\"",
			want:   EnvStore{"KEY": "a'b", "OTHER": `c\'d`},
		})
	})

	t.Run("key-specific false overrides global true", func(t *testing.T) {
		cfg := new(ParseConfig)
		cfg.ApplyGlobalOptions(&ParseOptions{UnescapeBackslashSingleQuote: new(true)})
		cfg.ApplyKeyOptions("KEY", &ParseOptions{UnescapeBackslashSingleQuote: new(false)})
		run(t, nil, cfg, testCase{
			name:   "disable escaped single quote unescaping for one key",
			dotenv: "KEY=\"a\\'b\"\nOTHER=\"c\\'d\"",
			want:   EnvStore{"KEY": `a\'b`, "OTHER": "c'd"},
		})
	})
}

func TestParseConfigUnescapeBackslashA(t *testing.T) {
	t.Run("all options disabled by default", func(t *testing.T) {
		cfg := new(ParseConfig)
		run(t, nil, cfg, testCase{
			name:   "escaped alert stays literal",
			dotenv: `KEY="a\ab"`,
			want:   EnvStore{"KEY": `a\ab`},
		})
	})

	t.Run("ApplyGlobalOptions enables escaped alert unescaping", func(t *testing.T) {
		cfg := new(ParseConfig)
		cfg.ApplyGlobalOptions(&ParseOptions{UnescapeBackslashA: new(true)})
		run(t, nil, cfg, testCase{
			name:   "escaped alert is unescaped",
			dotenv: `KEY="a\ab"`,
			want:   EnvStore{"KEY": "a\ab"},
		})
	})

	t.Run("ApplyKeyOptions enables escaped alert unescaping for one key", func(t *testing.T) {
		cfg := new(ParseConfig)
		cfg.ApplyKeyOptions("KEY", &ParseOptions{UnescapeBackslashA: new(true)})
		run(t, nil, cfg, testCase{
			name:   "key-specific escaped alert unescaping",
			dotenv: "KEY=\"a\\ab\"\nOTHER=\"c\\ad\"",
			want:   EnvStore{"KEY": "a\ab", "OTHER": `c\ad`},
		})
	})

	t.Run("key-specific false overrides global true", func(t *testing.T) {
		cfg := new(ParseConfig)
		cfg.ApplyGlobalOptions(&ParseOptions{UnescapeBackslashA: new(true)})
		cfg.ApplyKeyOptions("KEY", &ParseOptions{UnescapeBackslashA: new(false)})
		run(t, nil, cfg, testCase{
			name:   "disable escaped alert unescaping for one key",
			dotenv: "KEY=\"a\\ab\"\nOTHER=\"c\\ad\"",
			want:   EnvStore{"KEY": `a\ab`, "OTHER": "c\ad"},
		})
	})
}

func TestParseConfigUnescapeBackslashB(t *testing.T) {
	t.Run("all options disabled by default", func(t *testing.T) {
		cfg := new(ParseConfig)
		run(t, nil, cfg, testCase{
			name:   "escaped backspace stays literal",
			dotenv: `KEY="a\bb"`,
			want:   EnvStore{"KEY": `a\bb`},
		})
	})

	t.Run("ApplyGlobalOptions enables escaped backspace unescaping", func(t *testing.T) {
		cfg := new(ParseConfig)
		cfg.ApplyGlobalOptions(&ParseOptions{UnescapeBackslashB: new(true)})
		run(t, nil, cfg, testCase{
			name:   "escaped backspace is unescaped",
			dotenv: `KEY="a\bb"`,
			want:   EnvStore{"KEY": "a\bb"},
		})
	})

	t.Run("ApplyKeyOptions enables escaped backspace unescaping for one key", func(t *testing.T) {
		cfg := new(ParseConfig)
		cfg.ApplyKeyOptions("KEY", &ParseOptions{UnescapeBackslashB: new(true)})
		run(t, nil, cfg, testCase{
			name:   "key-specific escaped backspace unescaping",
			dotenv: "KEY=\"a\\bb\"\nOTHER=\"c\\bd\"",
			want:   EnvStore{"KEY": "a\bb", "OTHER": `c\bd`},
		})
	})

	t.Run("key-specific false overrides global true", func(t *testing.T) {
		cfg := new(ParseConfig)
		cfg.ApplyGlobalOptions(&ParseOptions{UnescapeBackslashB: new(true)})
		cfg.ApplyKeyOptions("KEY", &ParseOptions{UnescapeBackslashB: new(false)})
		run(t, nil, cfg, testCase{
			name:   "disable escaped backspace unescaping for one key",
			dotenv: "KEY=\"a\\bb\"\nOTHER=\"c\\bd\"",
			want:   EnvStore{"KEY": `a\bb`, "OTHER": "c\bd"},
		})
	})
}

func TestParseConfigUnescapeBackslashF(t *testing.T) {
	t.Run("all options disabled by default", func(t *testing.T) {
		cfg := new(ParseConfig)
		run(t, nil, cfg, testCase{
			name:   "escaped form feed stays literal",
			dotenv: `KEY="a\fb"`,
			want:   EnvStore{"KEY": `a\fb`},
		})
	})

	t.Run("ApplyGlobalOptions enables escaped form feed unescaping", func(t *testing.T) {
		cfg := new(ParseConfig)
		cfg.ApplyGlobalOptions(&ParseOptions{UnescapeBackslashF: new(true)})
		run(t, nil, cfg, testCase{
			name:   "escaped form feed is unescaped",
			dotenv: `KEY="a\fb"`,
			want:   EnvStore{"KEY": "a\fb"},
		})
	})

	t.Run("ApplyKeyOptions enables escaped form feed unescaping for one key", func(t *testing.T) {
		cfg := new(ParseConfig)
		cfg.ApplyKeyOptions("KEY", &ParseOptions{UnescapeBackslashF: new(true)})
		run(t, nil, cfg, testCase{
			name:   "key-specific escaped form feed unescaping",
			dotenv: "KEY=\"a\\fb\"\nOTHER=\"c\\fd\"",
			want:   EnvStore{"KEY": "a\fb", "OTHER": `c\fd`},
		})
	})

	t.Run("key-specific false overrides global true", func(t *testing.T) {
		cfg := new(ParseConfig)
		cfg.ApplyGlobalOptions(&ParseOptions{UnescapeBackslashF: new(true)})
		cfg.ApplyKeyOptions("KEY", &ParseOptions{UnescapeBackslashF: new(false)})
		run(t, nil, cfg, testCase{
			name:   "disable escaped form feed unescaping for one key",
			dotenv: "KEY=\"a\\fb\"\nOTHER=\"c\\fd\"",
			want:   EnvStore{"KEY": `a\fb`, "OTHER": "c\fd"},
		})
	})
}

func TestParseConfigUnescapeBackslashN(t *testing.T) {
	t.Run("all options disabled by default", func(t *testing.T) {
		cfg := new(ParseConfig)
		run(t, nil, cfg, testCase{
			name:   "escaped line feed stays literal",
			dotenv: `KEY="a\nb"`,
			want:   EnvStore{"KEY": `a\nb`},
		})
	})

	t.Run("ApplyGlobalOptions enables escaped line feed unescaping", func(t *testing.T) {
		cfg := new(ParseConfig)
		cfg.ApplyGlobalOptions(&ParseOptions{UnescapeBackslashN: new(true)})
		run(t, nil, cfg, testCase{
			name:   "escaped line feed is unescaped",
			dotenv: `KEY="a\nb"`,
			want:   EnvStore{"KEY": "a\nb"},
		})
	})

	t.Run("ApplyKeyOptions enables escaped line feed unescaping for one key", func(t *testing.T) {
		cfg := new(ParseConfig)
		cfg.ApplyKeyOptions("KEY", &ParseOptions{UnescapeBackslashN: new(true)})
		run(t, nil, cfg, testCase{
			name:   "key-specific escaped line feed unescaping",
			dotenv: "KEY=\"a\\nb\"\nOTHER=\"c\\nd\"",
			want:   EnvStore{"KEY": "a\nb", "OTHER": `c\nd`},
		})
	})

	t.Run("key-specific false overrides global true", func(t *testing.T) {
		cfg := new(ParseConfig)
		cfg.ApplyGlobalOptions(&ParseOptions{UnescapeBackslashN: new(true)})
		cfg.ApplyKeyOptions("KEY", &ParseOptions{UnescapeBackslashN: new(false)})
		run(t, nil, cfg, testCase{
			name:   "disable escaped line feed unescaping for one key",
			dotenv: "KEY=\"a\\nb\"\nOTHER=\"c\\nd\"",
			want:   EnvStore{"KEY": `a\nb`, "OTHER": "c\nd"},
		})
	})
}

func TestParseConfigUnescapeBackslashR(t *testing.T) {
	t.Run("all options disabled by default", func(t *testing.T) {
		cfg := new(ParseConfig)
		run(t, nil, cfg, testCase{
			name:   "escaped carriage return stays literal",
			dotenv: `KEY="a\rb"`,
			want:   EnvStore{"KEY": `a\rb`},
		})
	})

	t.Run("ApplyGlobalOptions enables escaped carriage return unescaping", func(t *testing.T) {
		cfg := new(ParseConfig)
		cfg.ApplyGlobalOptions(&ParseOptions{UnescapeBackslashR: new(true)})
		run(t, nil, cfg, testCase{
			name:   "escaped carriage return is unescaped",
			dotenv: `KEY="a\rb"`,
			want:   EnvStore{"KEY": "a\rb"},
		})
	})

	t.Run("ApplyKeyOptions enables escaped carriage return unescaping for one key", func(t *testing.T) {
		cfg := new(ParseConfig)
		cfg.ApplyKeyOptions("KEY", &ParseOptions{UnescapeBackslashR: new(true)})
		run(t, nil, cfg, testCase{
			name:   "key-specific escaped carriage return unescaping",
			dotenv: "KEY=\"a\\rb\"\nOTHER=\"c\\rd\"",
			want:   EnvStore{"KEY": "a\rb", "OTHER": `c\rd`},
		})
	})

	t.Run("key-specific false overrides global true", func(t *testing.T) {
		cfg := new(ParseConfig)
		cfg.ApplyGlobalOptions(&ParseOptions{UnescapeBackslashR: new(true)})
		cfg.ApplyKeyOptions("KEY", &ParseOptions{UnescapeBackslashR: new(false)})
		run(t, nil, cfg, testCase{
			name:   "disable escaped carriage return unescaping for one key",
			dotenv: "KEY=\"a\\rb\"\nOTHER=\"c\\rd\"",
			want:   EnvStore{"KEY": `a\rb`, "OTHER": "c\rd"},
		})
	})
}

func TestParseConfigUnescapeBackslashT(t *testing.T) {
	t.Run("all options disabled by default", func(t *testing.T) {
		cfg := new(ParseConfig)
		run(t, nil, cfg, testCase{
			name:   "escaped tab stays literal",
			dotenv: `KEY="a\tb"`,
			want:   EnvStore{"KEY": `a\tb`},
		})
	})

	t.Run("ApplyGlobalOptions enables escaped tab unescaping", func(t *testing.T) {
		cfg := new(ParseConfig)
		cfg.ApplyGlobalOptions(&ParseOptions{UnescapeBackslashT: new(true)})
		run(t, nil, cfg, testCase{
			name:   "escaped tab is unescaped",
			dotenv: `KEY="a\tb"`,
			want:   EnvStore{"KEY": "a\tb"},
		})
	})

	t.Run("ApplyKeyOptions enables escaped tab unescaping for one key", func(t *testing.T) {
		cfg := new(ParseConfig)
		cfg.ApplyKeyOptions("KEY", &ParseOptions{UnescapeBackslashT: new(true)})
		run(t, nil, cfg, testCase{
			name:   "key-specific escaped tab unescaping",
			dotenv: "KEY=\"a\\tb\"\nOTHER=\"c\\td\"",
			want:   EnvStore{"KEY": "a\tb", "OTHER": `c\td`},
		})
	})

	t.Run("key-specific false overrides global true", func(t *testing.T) {
		cfg := new(ParseConfig)
		cfg.ApplyGlobalOptions(&ParseOptions{UnescapeBackslashT: new(true)})
		cfg.ApplyKeyOptions("KEY", &ParseOptions{UnescapeBackslashT: new(false)})
		run(t, nil, cfg, testCase{
			name:   "disable escaped tab unescaping for one key",
			dotenv: "KEY=\"a\\tb\"\nOTHER=\"c\\td\"",
			want:   EnvStore{"KEY": `a\tb`, "OTHER": "c\td"},
		})
	})
}

func TestParseConfigUnescapeBackslashV(t *testing.T) {
	t.Run("all options disabled by default", func(t *testing.T) {
		cfg := new(ParseConfig)
		run(t, nil, cfg, testCase{
			name:   "escaped vertical tab stays literal",
			dotenv: `KEY="a\vb"`,
			want:   EnvStore{"KEY": `a\vb`},
		})
	})

	t.Run("ApplyGlobalOptions enables escaped vertical tab unescaping", func(t *testing.T) {
		cfg := new(ParseConfig)
		cfg.ApplyGlobalOptions(&ParseOptions{UnescapeBackslashV: new(true)})
		run(t, nil, cfg, testCase{
			name:   "escaped vertical tab is unescaped",
			dotenv: `KEY="a\vb"`,
			want:   EnvStore{"KEY": "a\vb"},
		})
	})

	t.Run("ApplyKeyOptions enables escaped vertical tab unescaping for one key", func(t *testing.T) {
		cfg := new(ParseConfig)
		cfg.ApplyKeyOptions("KEY", &ParseOptions{UnescapeBackslashV: new(true)})
		run(t, nil, cfg, testCase{
			name:   "key-specific escaped vertical tab unescaping",
			dotenv: "KEY=\"a\\vb\"\nOTHER=\"c\\vd\"",
			want:   EnvStore{"KEY": "a\vb", "OTHER": `c\vd`},
		})
	})

	t.Run("key-specific false overrides global true", func(t *testing.T) {
		cfg := new(ParseConfig)
		cfg.ApplyGlobalOptions(&ParseOptions{UnescapeBackslashV: new(true)})
		cfg.ApplyKeyOptions("KEY", &ParseOptions{UnescapeBackslashV: new(false)})
		run(t, nil, cfg, testCase{
			name:   "disable escaped vertical tab unescaping for one key",
			dotenv: "KEY=\"a\\vb\"\nOTHER=\"c\\vd\"",
			want:   EnvStore{"KEY": `a\vb`, "OTHER": "c\vd"},
		})
	})
}

func TestParseConfigTransformCRLFToLF(t *testing.T) {
	t.Run("all options disabled by default", func(t *testing.T) {
		cfg := new(ParseConfig)
		run(t, nil, cfg, testCase{
			name:   "literal CRLF stays unchanged",
			dotenv: "KEY=\"a\r\nb\"",
			want:   EnvStore{"KEY": "a\r\nb"},
		})
	})

	t.Run("ApplyGlobalOptions enables CRLF normalization", func(t *testing.T) {
		cfg := new(ParseConfig)
		cfg.ApplyGlobalOptions(&ParseOptions{TransformCRLFToLF: new(true)})
		run(t, nil, cfg, testCase{
			name:   "literal CRLF becomes LF",
			dotenv: "KEY=\"a\r\nb\"",
			want:   EnvStore{"KEY": "a\nb"},
		})
	})

	t.Run("ApplyKeyOptions enables CRLF normalization for one key", func(t *testing.T) {
		cfg := new(ParseConfig)
		cfg.ApplyKeyOptions("KEY", &ParseOptions{TransformCRLFToLF: new(true)})
		run(t, nil, cfg, testCase{
			name:   "key-specific CRLF normalization",
			dotenv: "KEY=\"a\r\nb\"\nOTHER=\"c\r\nd\"",
			want:   EnvStore{"KEY": "a\nb", "OTHER": "c\r\nd"},
		})
	})

	t.Run("key-specific false overrides global true", func(t *testing.T) {
		cfg := new(ParseConfig)
		cfg.ApplyGlobalOptions(&ParseOptions{TransformCRLFToLF: new(true)})
		cfg.ApplyKeyOptions("KEY", &ParseOptions{TransformCRLFToLF: new(false)})
		run(t, nil, cfg, testCase{
			name:   "disable CRLF normalization for one key",
			dotenv: "KEY=\"a\r\nb\"\nOTHER=\"c\r\nd\"",
			want:   EnvStore{"KEY": "a\r\nb", "OTHER": "c\nd"},
		})
	})
}

func TestParseConfigTransformCRToLF(t *testing.T) {
	t.Run("all options disabled by default", func(t *testing.T) {
		cfg := new(ParseConfig)
		run(t, nil, cfg, testCase{
			name:   "literal CR stays unchanged",
			dotenv: "KEY=\"a\rb\"",
			want:   EnvStore{"KEY": "a\rb"},
		})
	})

	t.Run("ApplyGlobalOptions enables CR normalization", func(t *testing.T) {
		cfg := new(ParseConfig)
		cfg.ApplyGlobalOptions(&ParseOptions{TransformCRToLF: new(true)})
		run(t, nil, cfg, testCase{
			name:   "literal CR becomes LF",
			dotenv: "KEY=\"a\rb\"",
			want:   EnvStore{"KEY": "a\nb"},
		})
	})

	t.Run("ApplyKeyOptions enables CR normalization for one key", func(t *testing.T) {
		cfg := new(ParseConfig)
		cfg.ApplyKeyOptions("KEY", &ParseOptions{TransformCRToLF: new(true)})
		run(t, nil, cfg, testCase{
			name:   "key-specific CR normalization",
			dotenv: "KEY=\"a\rb\"\nOTHER=\"c\rd\"",
			want:   EnvStore{"KEY": "a\nb", "OTHER": "c\rd"},
		})
	})

	t.Run("key-specific false overrides global true", func(t *testing.T) {
		cfg := new(ParseConfig)
		cfg.ApplyGlobalOptions(&ParseOptions{TransformCRToLF: new(true)})
		cfg.ApplyKeyOptions("KEY", &ParseOptions{TransformCRToLF: new(false)})
		run(t, nil, cfg, testCase{
			name:   "disable CR normalization for one key",
			dotenv: "KEY=\"a\rb\"\nOTHER=\"c\rd\"",
			want:   EnvStore{"KEY": "a\rb", "OTHER": "c\nd"},
		})
	})
}

func TestParseConfigAppliesTransformsAfterUnescaping(t *testing.T) {
	cfg := new(ParseConfig)
	cfg.ApplyGlobalOptions(&ParseOptions{
		UnescapeBackslashR: new(true),
		UnescapeBackslashN: new(true),
		TransformCRLFToLF:  new(true),
		TransformCRToLF:    new(true),
	})

	run(t, nil, cfg, testCase{
		name:   "escaped CRLF is unescaped before normalization",
		dotenv: `KEY="a\r\nb"`,
		want:   EnvStore{"KEY": "a\nb"},
	})
}

func TestParseConfigTransformCRWithoutCRLFTransformProducesTwoLFBytes(t *testing.T) {
	cfg := new(ParseConfig)
	cfg.ApplyGlobalOptions(&ParseOptions{TransformCRToLF: new(true)})

	run(t, nil, cfg, testCase{
		name:   "CRLF becomes two LF bytes when only CR normalization is enabled",
		dotenv: "KEY=\"a\r\nb\"",
		want:   EnvStore{"KEY": "a\n\nb"},
	})
}
