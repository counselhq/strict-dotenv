package strictdotenv

import "testing"

func TestConfigZeroValue(t *testing.T) {
	cfg := new(Config)

	if cfg.globalOptions != (Options{}) {
		t.Errorf("globalOptions = %+v, want zero Options", cfg.globalOptions)
	}
	if cfg.keyOptions != nil {
		t.Errorf("keyOptions = %v, want nil", cfg.keyOptions)
	}
}

func TestConfigMergeGlobalOptionsUpdatesGlobalState(t *testing.T) {
	cfg := new(Config)

	cfg.MergeGlobalOptions(Options{
		Overwrite:          new(true),
		UnescapeBackslashN: new(true),
	})

	if cfg.globalOptions.Overwrite == nil || !*cfg.globalOptions.Overwrite {
		t.Fatal("expected globalOptions.Overwrite=true")
	}
	if cfg.globalOptions.UnescapeBackslashN == nil || !*cfg.globalOptions.UnescapeBackslashN {
		t.Fatal("expected globalOptions.UnescapeBackslashN=true")
	}
	if cfg.globalOptions.TransformCRToLF != nil {
		t.Fatal("expected unrelated globalOptions field to remain nil")
	}

	resolved := resolveOptions("KEY", cfg)
	want := resolvedOptions{
		Overwrite:          true,
		UnescapeBackslashN: true,
	}
	if resolved != want {
		t.Fatalf("resolveOptions(KEY, cfg) = %+v, want %+v", resolved, want)
	}
}

func TestConfigMergeGlobalOptionsClonesPointers(t *testing.T) {
	overwrite := true
	cfg := new(Config)

	cfg.MergeGlobalOptions(Options{Overwrite: &overwrite})
	overwrite = false

	if cfg.globalOptions.Overwrite == nil || !*cfg.globalOptions.Overwrite {
		t.Fatal("expected Config to keep its own copy of Overwrite=true")
	}
}

func TestConfigSetGlobalOptionsReplacesGlobalState(t *testing.T) {
	cfg := new(Config)

	cfg.MergeGlobalOptions(Options{
		Overwrite:          new(true),
		UnescapeBackslashN: new(true),
	})
	cfg.SetGlobalOptions(Options{
		TransformCRToLF: new(true),
	})

	if cfg.globalOptions.Overwrite != nil {
		t.Fatal("expected SetGlobalOptions to clear Overwrite")
	}
	if cfg.globalOptions.UnescapeBackslashN != nil {
		t.Fatal("expected SetGlobalOptions to clear UnescapeBackslashN")
	}
	if cfg.globalOptions.TransformCRToLF == nil || !*cfg.globalOptions.TransformCRToLF {
		t.Fatal("expected SetGlobalOptions to store TransformCRToLF=true")
	}

	resolved := resolveOptions("KEY", cfg)
	want := resolvedOptions{
		TransformCRToLF: true,
	}
	if resolved != want {
		t.Fatalf("resolveOptions(KEY, cfg) = %+v, want %+v", resolved, want)
	}
}

func TestConfigSetGlobalOptionsClonesPointers(t *testing.T) {
	overwrite := true
	cfg := new(Config)

	cfg.SetGlobalOptions(Options{Overwrite: &overwrite})
	overwrite = false

	if cfg.globalOptions.Overwrite == nil || !*cfg.globalOptions.Overwrite {
		t.Fatal("expected Config to keep its own copy of Overwrite=true")
	}
}

func TestConfigResolveOptionsUsesUpdatedGlobalSettingsForKeys(t *testing.T) {
	cfg := new(Config)
	cfg.MergeKeyOptions("KEY", Options{UnescapeBackslashN: new(true)})

	cfg.MergeGlobalOptions(Options{Overwrite: new(true)})

	resolved := resolveOptions("KEY", cfg)
	want := resolvedOptions{
		Overwrite:          true,
		UnescapeBackslashN: true,
	}
	if resolved != want {
		t.Fatalf("resolveOptions(KEY, cfg) = %+v, want %+v", resolved, want)
	}
}

func TestConfigMergeKeyOptionsMergesSameKey(t *testing.T) {
	cfg := new(Config)
	cfg.MergeGlobalOptions(Options{Overwrite: new(true)})

	cfg.MergeKeyOptions("KEY", Options{UnescapeBackslashN: new(true)})
	cfg.MergeKeyOptions("KEY", Options{TransformCRToLF: new(true)})

	keyOptions := cfg.keyOptions["KEY"]
	if keyOptions.UnescapeBackslashN == nil || !*keyOptions.UnescapeBackslashN {
		t.Fatal("expected KEY UnescapeBackslashN=true after merging key options")
	}
	if keyOptions.TransformCRToLF == nil || !*keyOptions.TransformCRToLF {
		t.Fatal("expected KEY TransformCRToLF=true after merging key options")
	}

	resolved := resolveOptions("KEY", cfg)
	want := resolvedOptions{
		Overwrite:          true,
		UnescapeBackslashN: true,
		TransformCRToLF:    true,
	}
	if resolved != want {
		t.Fatalf("resolveOptions(KEY, cfg) = %+v, want %+v", resolved, want)
	}
}

func TestConfigSetKeyOptionsReplacesExistingKeyOptions(t *testing.T) {
	cfg := new(Config)
	cfg.MergeGlobalOptions(Options{
		Overwrite:          new(true),
		UnescapeBackslashN: new(true),
	})
	cfg.MergeKeyOptions("KEY", Options{
		Overwrite:          new(false),
		TransformCRLFToLF:  new(true),
		TransformCRToLF:    new(true),
		UnescapeBackslashN: new(false),
	})

	cfg.SetKeyOptions("KEY", Options{
		TransformCRToLF: new(true),
	})

	keyOptions := cfg.keyOptions["KEY"]
	if keyOptions.Overwrite != nil {
		t.Fatal("expected SetKeyOptions to clear Overwrite override")
	}
	if keyOptions.TransformCRLFToLF != nil {
		t.Fatal("expected SetKeyOptions to clear TransformCRLFToLF override")
	}
	if keyOptions.UnescapeBackslashN != nil {
		t.Fatal("expected SetKeyOptions to clear UnescapeBackslashN override")
	}
	if keyOptions.TransformCRToLF == nil || !*keyOptions.TransformCRToLF {
		t.Fatal("expected SetKeyOptions to store TransformCRToLF=true")
	}

	resolved := resolveOptions("KEY", cfg)
	want := resolvedOptions{
		Overwrite:          true,
		UnescapeBackslashN: true,
		TransformCRToLF:    true,
	}
	if resolved != want {
		t.Fatalf("resolveOptions(KEY, cfg) = %+v, want %+v", resolved, want)
	}
}

func TestConfigSetKeyOptionsClonesPointers(t *testing.T) {
	unescapeBackslashN := true
	cfg := new(Config)

	cfg.SetKeyOptions("KEY", Options{UnescapeBackslashN: &unescapeBackslashN})
	unescapeBackslashN = false

	keyOptions := cfg.keyOptions["KEY"]
	if keyOptions.UnescapeBackslashN == nil || !*keyOptions.UnescapeBackslashN {
		t.Fatal("expected Config to keep its own copy of UnescapeBackslashN=true")
	}
}

func TestConfigMergeKeyOptionsZeroValueUsesGlobalResolution(t *testing.T) {
	cfg := new(Config)
	cfg.MergeGlobalOptions(Options{
		Overwrite:          new(true),
		UnescapeBackslashN: new(true),
	})

	cfg.MergeKeyOptions("KEY", Options{})

	keyOptions, ok := cfg.keyOptions["KEY"]
	if !ok {
		t.Fatal("expected KEY options to exist")
	}
	if keyOptions != (Options{}) {
		t.Fatalf("KEY options = %+v, want zero Options", keyOptions)
	}

	resolved := resolveOptions("KEY", cfg)
	global := resolveOptions("OTHER", cfg)
	if resolved != global {
		t.Fatalf("resolveOptions(KEY, cfg) = %+v, want %+v", resolved, global)
	}
}

func TestResolveOptionsNilConfigReturnsZeroValue(t *testing.T) {
	if got := resolveOptions("KEY", nil); got != (resolvedOptions{}) {
		t.Fatalf("resolveOptions(KEY, nil) = %+v, want zero resolvedOptions", got)
	}
}

func TestNilConfigUsesZeroValueOptions(t *testing.T) {
	run(t, nil, nil, testCase{
		name:   "nil config preserves literal escapes",
		dotenv: "KEY=\"line1\\nline2\"",
		want:   EnvStore{"KEY": `line1\nline2`},
	})
}

func TestConfigOverwrite(t *testing.T) {
	t.Run("all options disabled by default", func(t *testing.T) {
		cfg := new(Config)
		run(t, nil, cfg, testCase{
			name:   "repeated keys keep the first value",
			dotenv: "KEY=1\nKEY=2",
			want:   EnvStore{"KEY": "1"},
		})
	})

	t.Run("MergeGlobalOptions enables overwrite", func(t *testing.T) {
		cfg := new(Config)
		cfg.MergeGlobalOptions(Options{Overwrite: new(true)})
		run(t, nil, cfg, testCase{
			name:   "repeated keys keep the last value",
			dotenv: "KEY=1\nKEY=2",
			want:   EnvStore{"KEY": "2"},
		})
	})

	t.Run("MergeKeyOptions enables overwrite for one key", func(t *testing.T) {
		cfg := new(Config)
		cfg.MergeKeyOptions("KEY", Options{Overwrite: new(true)})
		run(t, nil, cfg, testCase{
			name:   "key-specific overwrite",
			dotenv: "KEY=1\nKEY=2\nOTHER=3\nOTHER=4",
			want:   EnvStore{"KEY": "2", "OTHER": "3"},
		})
	})

	t.Run("key-specific false overrides global true", func(t *testing.T) {
		cfg := new(Config)
		cfg.MergeGlobalOptions(Options{Overwrite: new(true)})
		cfg.MergeKeyOptions("KEY", Options{Overwrite: new(false)})
		run(t, nil, cfg, testCase{
			name:   "key-specific overwrite disable",
			dotenv: "KEY=1\nKEY=2\nOTHER=3\nOTHER=4",
			want:   EnvStore{"KEY": "1", "OTHER": "4"},
		})
	})
}

func TestConfigUnescapeBackslashBackslash(t *testing.T) {
	t.Run("all options disabled by default", func(t *testing.T) {
		cfg := new(Config)
		run(t, nil, cfg, testCase{
			name:   "escaped backslash stays literal",
			dotenv: "KEY=\"a\\\\b\"",
			want:   EnvStore{"KEY": `a\\b`},
		})
	})

	t.Run("MergeGlobalOptions enables escaped backslash unescaping", func(t *testing.T) {
		cfg := new(Config)
		cfg.MergeGlobalOptions(Options{UnescapeBackslashBackslash: new(true)})
		run(t, nil, cfg, testCase{
			name:   "escaped backslash is unescaped",
			dotenv: "KEY=\"a\\\\b\"",
			want:   EnvStore{"KEY": `a\b`},
		})
	})

	t.Run("MergeKeyOptions enables escaped backslash unescaping for one key", func(t *testing.T) {
		cfg := new(Config)
		cfg.MergeKeyOptions("KEY", Options{UnescapeBackslashBackslash: new(true)})
		run(t, nil, cfg, testCase{
			name:   "key-specific escaped backslash unescaping",
			dotenv: "KEY=\"a\\\\b\"\nOTHER=\"c\\\\d\"",
			want:   EnvStore{"KEY": `a\b`, "OTHER": `c\\d`},
		})
	})

	t.Run("key-specific false overrides global true", func(t *testing.T) {
		cfg := new(Config)
		cfg.MergeGlobalOptions(Options{UnescapeBackslashBackslash: new(true)})
		cfg.MergeKeyOptions("KEY", Options{UnescapeBackslashBackslash: new(false)})
		run(t, nil, cfg, testCase{
			name:   "disable escaped backslash unescaping for one key",
			dotenv: "KEY=\"a\\\\b\"\nOTHER=\"c\\\\d\"",
			want:   EnvStore{"KEY": `a\\b`, "OTHER": `c\d`},
		})
	})
}

func TestConfigUnescapeBackslashDoubleQuote(t *testing.T) {
	t.Run("all options disabled by default", func(t *testing.T) {
		cfg := new(Config)
		run(t, nil, cfg, testCase{
			name:   "backslash before closing double quote is literal; the double quote closes the value",
			dotenv: "KEY=\"a\\\"",
			want:   EnvStore{"KEY": `a\`},
		})
	})

	t.Run("MergeGlobalOptions enables escaped double quote unescaping", func(t *testing.T) {
		cfg := new(Config)
		cfg.MergeGlobalOptions(Options{UnescapeBackslashDoubleQuote: new(true)})
		run(t, nil, cfg, testCase{
			name:   "escaped double quote is unescaped",
			dotenv: "KEY=\"a\\\"b\"",
			want:   EnvStore{"KEY": `a"b`},
		})
	})

	t.Run("MergeKeyOptions enables escaped double quote unescaping for one key", func(t *testing.T) {
		cfg := new(Config)
		cfg.MergeKeyOptions("KEY", Options{UnescapeBackslashDoubleQuote: new(true)})
		run(t, nil, cfg, testCase{
			name:   "key-specific escaped double quote unescaping",
			dotenv: "KEY=\"a\\\"b\"\nOTHER=\"c\\\"",
			want:   EnvStore{"KEY": `a"b`, "OTHER": `c\`},
		})
	})

	t.Run("key-specific false overrides global true", func(t *testing.T) {
		cfg := new(Config)
		cfg.MergeGlobalOptions(Options{UnescapeBackslashDoubleQuote: new(true)})
		cfg.MergeKeyOptions("KEY", Options{UnescapeBackslashDoubleQuote: new(false)})
		run(t, nil, cfg, testCase{
			name:   "disable escaped double quote unescaping for one key",
			dotenv: "KEY=\"a\\\"\nOTHER=\"c\\\"d\"",
			want:   EnvStore{"KEY": `a\`, "OTHER": `c"d`},
		})
	})
}

func TestConfigUnescapeBackslashSingleQuote(t *testing.T) {
	t.Run("all options disabled by default", func(t *testing.T) {
		cfg := new(Config)
		run(t, nil, cfg, testCase{
			name:   "escaped single quote stays literal",
			dotenv: `KEY="a\'b"`,
			want:   EnvStore{"KEY": `a\'b`},
		})
	})

	t.Run("MergeGlobalOptions enables escaped single quote unescaping", func(t *testing.T) {
		cfg := new(Config)
		cfg.MergeGlobalOptions(Options{UnescapeBackslashSingleQuote: new(true)})
		run(t, nil, cfg, testCase{
			name:   "escaped single quote is unescaped",
			dotenv: `KEY="a\'b"`,
			want:   EnvStore{"KEY": "a'b"},
		})
	})

	t.Run("MergeKeyOptions enables escaped single quote unescaping for one key", func(t *testing.T) {
		cfg := new(Config)
		cfg.MergeKeyOptions("KEY", Options{UnescapeBackslashSingleQuote: new(true)})
		run(t, nil, cfg, testCase{
			name:   "key-specific escaped single quote unescaping",
			dotenv: "KEY=\"a\\'b\"\nOTHER=\"c\\'d\"",
			want:   EnvStore{"KEY": "a'b", "OTHER": `c\'d`},
		})
	})

	t.Run("key-specific false overrides global true", func(t *testing.T) {
		cfg := new(Config)
		cfg.MergeGlobalOptions(Options{UnescapeBackslashSingleQuote: new(true)})
		cfg.MergeKeyOptions("KEY", Options{UnescapeBackslashSingleQuote: new(false)})
		run(t, nil, cfg, testCase{
			name:   "disable escaped single quote unescaping for one key",
			dotenv: "KEY=\"a\\'b\"\nOTHER=\"c\\'d\"",
			want:   EnvStore{"KEY": `a\'b`, "OTHER": "c'd"},
		})
	})
}

func TestConfigUnescapeBackslashA(t *testing.T) {
	t.Run("all options disabled by default", func(t *testing.T) {
		cfg := new(Config)
		run(t, nil, cfg, testCase{
			name:   "escaped alert stays literal",
			dotenv: `KEY="a\ab"`,
			want:   EnvStore{"KEY": `a\ab`},
		})
	})

	t.Run("MergeGlobalOptions enables escaped alert unescaping", func(t *testing.T) {
		cfg := new(Config)
		cfg.MergeGlobalOptions(Options{UnescapeBackslashA: new(true)})
		run(t, nil, cfg, testCase{
			name:   "escaped alert is unescaped",
			dotenv: `KEY="a\ab"`,
			want:   EnvStore{"KEY": "a\ab"},
		})
	})

	t.Run("MergeKeyOptions enables escaped alert unescaping for one key", func(t *testing.T) {
		cfg := new(Config)
		cfg.MergeKeyOptions("KEY", Options{UnescapeBackslashA: new(true)})
		run(t, nil, cfg, testCase{
			name:   "key-specific escaped alert unescaping",
			dotenv: "KEY=\"a\\ab\"\nOTHER=\"c\\ad\"",
			want:   EnvStore{"KEY": "a\ab", "OTHER": `c\ad`},
		})
	})

	t.Run("key-specific false overrides global true", func(t *testing.T) {
		cfg := new(Config)
		cfg.MergeGlobalOptions(Options{UnescapeBackslashA: new(true)})
		cfg.MergeKeyOptions("KEY", Options{UnescapeBackslashA: new(false)})
		run(t, nil, cfg, testCase{
			name:   "disable escaped alert unescaping for one key",
			dotenv: "KEY=\"a\\ab\"\nOTHER=\"c\\ad\"",
			want:   EnvStore{"KEY": `a\ab`, "OTHER": "c\ad"},
		})
	})
}

func TestConfigUnescapeBackslashB(t *testing.T) {
	t.Run("all options disabled by default", func(t *testing.T) {
		cfg := new(Config)
		run(t, nil, cfg, testCase{
			name:   "escaped backspace stays literal",
			dotenv: `KEY="a\bb"`,
			want:   EnvStore{"KEY": `a\bb`},
		})
	})

	t.Run("MergeGlobalOptions enables escaped backspace unescaping", func(t *testing.T) {
		cfg := new(Config)
		cfg.MergeGlobalOptions(Options{UnescapeBackslashB: new(true)})
		run(t, nil, cfg, testCase{
			name:   "escaped backspace is unescaped",
			dotenv: `KEY="a\bb"`,
			want:   EnvStore{"KEY": "a\bb"},
		})
	})

	t.Run("MergeKeyOptions enables escaped backspace unescaping for one key", func(t *testing.T) {
		cfg := new(Config)
		cfg.MergeKeyOptions("KEY", Options{UnescapeBackslashB: new(true)})
		run(t, nil, cfg, testCase{
			name:   "key-specific escaped backspace unescaping",
			dotenv: "KEY=\"a\\bb\"\nOTHER=\"c\\bd\"",
			want:   EnvStore{"KEY": "a\bb", "OTHER": `c\bd`},
		})
	})

	t.Run("key-specific false overrides global true", func(t *testing.T) {
		cfg := new(Config)
		cfg.MergeGlobalOptions(Options{UnescapeBackslashB: new(true)})
		cfg.MergeKeyOptions("KEY", Options{UnescapeBackslashB: new(false)})
		run(t, nil, cfg, testCase{
			name:   "disable escaped backspace unescaping for one key",
			dotenv: "KEY=\"a\\bb\"\nOTHER=\"c\\bd\"",
			want:   EnvStore{"KEY": `a\bb`, "OTHER": "c\bd"},
		})
	})
}

func TestConfigUnescapeBackslashF(t *testing.T) {
	t.Run("all options disabled by default", func(t *testing.T) {
		cfg := new(Config)
		run(t, nil, cfg, testCase{
			name:   "escaped form feed stays literal",
			dotenv: `KEY="a\fb"`,
			want:   EnvStore{"KEY": `a\fb`},
		})
	})

	t.Run("MergeGlobalOptions enables escaped form feed unescaping", func(t *testing.T) {
		cfg := new(Config)
		cfg.MergeGlobalOptions(Options{UnescapeBackslashF: new(true)})
		run(t, nil, cfg, testCase{
			name:   "escaped form feed is unescaped",
			dotenv: `KEY="a\fb"`,
			want:   EnvStore{"KEY": "a\fb"},
		})
	})

	t.Run("MergeKeyOptions enables escaped form feed unescaping for one key", func(t *testing.T) {
		cfg := new(Config)
		cfg.MergeKeyOptions("KEY", Options{UnescapeBackslashF: new(true)})
		run(t, nil, cfg, testCase{
			name:   "key-specific escaped form feed unescaping",
			dotenv: "KEY=\"a\\fb\"\nOTHER=\"c\\fd\"",
			want:   EnvStore{"KEY": "a\fb", "OTHER": `c\fd`},
		})
	})

	t.Run("key-specific false overrides global true", func(t *testing.T) {
		cfg := new(Config)
		cfg.MergeGlobalOptions(Options{UnescapeBackslashF: new(true)})
		cfg.MergeKeyOptions("KEY", Options{UnescapeBackslashF: new(false)})
		run(t, nil, cfg, testCase{
			name:   "disable escaped form feed unescaping for one key",
			dotenv: "KEY=\"a\\fb\"\nOTHER=\"c\\fd\"",
			want:   EnvStore{"KEY": `a\fb`, "OTHER": "c\fd"},
		})
	})
}

func TestConfigUnescapeBackslashN(t *testing.T) {
	t.Run("all options disabled by default", func(t *testing.T) {
		cfg := new(Config)
		run(t, nil, cfg, testCase{
			name:   "escaped line feed stays literal",
			dotenv: `KEY="a\nb"`,
			want:   EnvStore{"KEY": `a\nb`},
		})
	})

	t.Run("MergeGlobalOptions enables escaped line feed unescaping", func(t *testing.T) {
		cfg := new(Config)
		cfg.MergeGlobalOptions(Options{UnescapeBackslashN: new(true)})
		run(t, nil, cfg, testCase{
			name:   "escaped line feed is unescaped",
			dotenv: `KEY="a\nb"`,
			want:   EnvStore{"KEY": "a\nb"},
		})
	})

	t.Run("MergeKeyOptions enables escaped line feed unescaping for one key", func(t *testing.T) {
		cfg := new(Config)
		cfg.MergeKeyOptions("KEY", Options{UnescapeBackslashN: new(true)})
		run(t, nil, cfg, testCase{
			name:   "key-specific escaped line feed unescaping",
			dotenv: "KEY=\"a\\nb\"\nOTHER=\"c\\nd\"",
			want:   EnvStore{"KEY": "a\nb", "OTHER": `c\nd`},
		})
	})

	t.Run("key-specific false overrides global true", func(t *testing.T) {
		cfg := new(Config)
		cfg.MergeGlobalOptions(Options{UnescapeBackslashN: new(true)})
		cfg.MergeKeyOptions("KEY", Options{UnescapeBackslashN: new(false)})
		run(t, nil, cfg, testCase{
			name:   "disable escaped line feed unescaping for one key",
			dotenv: "KEY=\"a\\nb\"\nOTHER=\"c\\nd\"",
			want:   EnvStore{"KEY": `a\nb`, "OTHER": "c\nd"},
		})
	})
}

func TestConfigUnescapeBackslashR(t *testing.T) {
	t.Run("all options disabled by default", func(t *testing.T) {
		cfg := new(Config)
		run(t, nil, cfg, testCase{
			name:   "escaped carriage return stays literal",
			dotenv: `KEY="a\rb"`,
			want:   EnvStore{"KEY": `a\rb`},
		})
	})

	t.Run("MergeGlobalOptions enables escaped carriage return unescaping", func(t *testing.T) {
		cfg := new(Config)
		cfg.MergeGlobalOptions(Options{UnescapeBackslashR: new(true)})
		run(t, nil, cfg, testCase{
			name:   "escaped carriage return is unescaped",
			dotenv: `KEY="a\rb"`,
			want:   EnvStore{"KEY": "a\rb"},
		})
	})

	t.Run("MergeKeyOptions enables escaped carriage return unescaping for one key", func(t *testing.T) {
		cfg := new(Config)
		cfg.MergeKeyOptions("KEY", Options{UnescapeBackslashR: new(true)})
		run(t, nil, cfg, testCase{
			name:   "key-specific escaped carriage return unescaping",
			dotenv: "KEY=\"a\\rb\"\nOTHER=\"c\\rd\"",
			want:   EnvStore{"KEY": "a\rb", "OTHER": `c\rd`},
		})
	})

	t.Run("key-specific false overrides global true", func(t *testing.T) {
		cfg := new(Config)
		cfg.MergeGlobalOptions(Options{UnescapeBackslashR: new(true)})
		cfg.MergeKeyOptions("KEY", Options{UnescapeBackslashR: new(false)})
		run(t, nil, cfg, testCase{
			name:   "disable escaped carriage return unescaping for one key",
			dotenv: "KEY=\"a\\rb\"\nOTHER=\"c\\rd\"",
			want:   EnvStore{"KEY": `a\rb`, "OTHER": "c\rd"},
		})
	})
}

func TestConfigUnescapeBackslashT(t *testing.T) {
	t.Run("all options disabled by default", func(t *testing.T) {
		cfg := new(Config)
		run(t, nil, cfg, testCase{
			name:   "escaped tab stays literal",
			dotenv: `KEY="a\tb"`,
			want:   EnvStore{"KEY": `a\tb`},
		})
	})

	t.Run("MergeGlobalOptions enables escaped tab unescaping", func(t *testing.T) {
		cfg := new(Config)
		cfg.MergeGlobalOptions(Options{UnescapeBackslashT: new(true)})
		run(t, nil, cfg, testCase{
			name:   "escaped tab is unescaped",
			dotenv: `KEY="a\tb"`,
			want:   EnvStore{"KEY": "a\tb"},
		})
	})

	t.Run("MergeKeyOptions enables escaped tab unescaping for one key", func(t *testing.T) {
		cfg := new(Config)
		cfg.MergeKeyOptions("KEY", Options{UnescapeBackslashT: new(true)})
		run(t, nil, cfg, testCase{
			name:   "key-specific escaped tab unescaping",
			dotenv: "KEY=\"a\\tb\"\nOTHER=\"c\\td\"",
			want:   EnvStore{"KEY": "a\tb", "OTHER": `c\td`},
		})
	})

	t.Run("key-specific false overrides global true", func(t *testing.T) {
		cfg := new(Config)
		cfg.MergeGlobalOptions(Options{UnescapeBackslashT: new(true)})
		cfg.MergeKeyOptions("KEY", Options{UnescapeBackslashT: new(false)})
		run(t, nil, cfg, testCase{
			name:   "disable escaped tab unescaping for one key",
			dotenv: "KEY=\"a\\tb\"\nOTHER=\"c\\td\"",
			want:   EnvStore{"KEY": `a\tb`, "OTHER": "c\td"},
		})
	})
}

func TestConfigUnescapeBackslashV(t *testing.T) {
	t.Run("all options disabled by default", func(t *testing.T) {
		cfg := new(Config)
		run(t, nil, cfg, testCase{
			name:   "escaped vertical tab stays literal",
			dotenv: `KEY="a\vb"`,
			want:   EnvStore{"KEY": `a\vb`},
		})
	})

	t.Run("MergeGlobalOptions enables escaped vertical tab unescaping", func(t *testing.T) {
		cfg := new(Config)
		cfg.MergeGlobalOptions(Options{UnescapeBackslashV: new(true)})
		run(t, nil, cfg, testCase{
			name:   "escaped vertical tab is unescaped",
			dotenv: `KEY="a\vb"`,
			want:   EnvStore{"KEY": "a\vb"},
		})
	})

	t.Run("MergeKeyOptions enables escaped vertical tab unescaping for one key", func(t *testing.T) {
		cfg := new(Config)
		cfg.MergeKeyOptions("KEY", Options{UnescapeBackslashV: new(true)})
		run(t, nil, cfg, testCase{
			name:   "key-specific escaped vertical tab unescaping",
			dotenv: "KEY=\"a\\vb\"\nOTHER=\"c\\vd\"",
			want:   EnvStore{"KEY": "a\vb", "OTHER": `c\vd`},
		})
	})

	t.Run("key-specific false overrides global true", func(t *testing.T) {
		cfg := new(Config)
		cfg.MergeGlobalOptions(Options{UnescapeBackslashV: new(true)})
		cfg.MergeKeyOptions("KEY", Options{UnescapeBackslashV: new(false)})
		run(t, nil, cfg, testCase{
			name:   "disable escaped vertical tab unescaping for one key",
			dotenv: "KEY=\"a\\vb\"\nOTHER=\"c\\vd\"",
			want:   EnvStore{"KEY": `a\vb`, "OTHER": "c\vd"},
		})
	})
}

func TestConfigTransformCRLFToLF(t *testing.T) {
	t.Run("all options disabled by default", func(t *testing.T) {
		cfg := new(Config)
		run(t, nil, cfg, testCase{
			name:   "literal CRLF stays unchanged",
			dotenv: "KEY=\"a\r\nb\"",
			want:   EnvStore{"KEY": "a\r\nb"},
		})
	})

	t.Run("MergeGlobalOptions enables CRLF normalization", func(t *testing.T) {
		cfg := new(Config)
		cfg.MergeGlobalOptions(Options{TransformCRLFToLF: new(true)})
		run(t, nil, cfg, testCase{
			name:   "literal CRLF becomes LF",
			dotenv: "KEY=\"a\r\nb\"",
			want:   EnvStore{"KEY": "a\nb"},
		})
	})

	t.Run("MergeKeyOptions enables CRLF normalization for one key", func(t *testing.T) {
		cfg := new(Config)
		cfg.MergeKeyOptions("KEY", Options{TransformCRLFToLF: new(true)})
		run(t, nil, cfg, testCase{
			name:   "key-specific CRLF normalization",
			dotenv: "KEY=\"a\r\nb\"\nOTHER=\"c\r\nd\"",
			want:   EnvStore{"KEY": "a\nb", "OTHER": "c\r\nd"},
		})
	})

	t.Run("key-specific false overrides global true", func(t *testing.T) {
		cfg := new(Config)
		cfg.MergeGlobalOptions(Options{TransformCRLFToLF: new(true)})
		cfg.MergeKeyOptions("KEY", Options{TransformCRLFToLF: new(false)})
		run(t, nil, cfg, testCase{
			name:   "disable CRLF normalization for one key",
			dotenv: "KEY=\"a\r\nb\"\nOTHER=\"c\r\nd\"",
			want:   EnvStore{"KEY": "a\r\nb", "OTHER": "c\nd"},
		})
	})
}

func TestConfigTransformCRToLF(t *testing.T) {
	t.Run("all options disabled by default", func(t *testing.T) {
		cfg := new(Config)
		run(t, nil, cfg, testCase{
			name:   "literal CR stays unchanged",
			dotenv: "KEY=\"a\rb\"",
			want:   EnvStore{"KEY": "a\rb"},
		})
	})

	t.Run("MergeGlobalOptions enables CR normalization", func(t *testing.T) {
		cfg := new(Config)
		cfg.MergeGlobalOptions(Options{TransformCRToLF: new(true)})
		run(t, nil, cfg, testCase{
			name:   "literal CR becomes LF",
			dotenv: "KEY=\"a\rb\"",
			want:   EnvStore{"KEY": "a\nb"},
		})
	})

	t.Run("MergeKeyOptions enables CR normalization for one key", func(t *testing.T) {
		cfg := new(Config)
		cfg.MergeKeyOptions("KEY", Options{TransformCRToLF: new(true)})
		run(t, nil, cfg, testCase{
			name:   "key-specific CR normalization",
			dotenv: "KEY=\"a\rb\"\nOTHER=\"c\rd\"",
			want:   EnvStore{"KEY": "a\nb", "OTHER": "c\rd"},
		})
	})

	t.Run("key-specific false overrides global true", func(t *testing.T) {
		cfg := new(Config)
		cfg.MergeGlobalOptions(Options{TransformCRToLF: new(true)})
		cfg.MergeKeyOptions("KEY", Options{TransformCRToLF: new(false)})
		run(t, nil, cfg, testCase{
			name:   "disable CR normalization for one key",
			dotenv: "KEY=\"a\rb\"\nOTHER=\"c\rd\"",
			want:   EnvStore{"KEY": "a\rb", "OTHER": "c\nd"},
		})
	})
}

func TestConfigAppliesTransformsAfterUnescaping(t *testing.T) {
	cfg := new(Config)
	cfg.MergeGlobalOptions(Options{
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

func TestConfigTransformCRWithoutCRLFTransformProducesTwoLFBytes(t *testing.T) {
	cfg := new(Config)
	cfg.MergeGlobalOptions(Options{TransformCRToLF: new(true)})

	run(t, nil, cfg, testCase{
		name:   "CRLF becomes two LF bytes when only CR normalization is enabled",
		dotenv: "KEY=\"a\r\nb\"",
		want:   EnvStore{"KEY": "a\n\nb"},
	})
}
