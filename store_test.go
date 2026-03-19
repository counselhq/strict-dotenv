package strictdotenv

import (
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewStore(t *testing.T) {
	store := NewStore()
	if store == nil {
		t.Fatal("NewStore returned nil")
	}
	if len(store) != 0 {
		t.Fatalf("NewStore returned non-empty store: %v", store)
	}

	store.Set("KEY", "value", false)
	if got := store["KEY"]; got != "value" {
		t.Fatalf("store should be writable, got %q", got)
	}
}

func TestStoreGet(t *testing.T) {
	store := Store{
		"PRESENT": "value",
		"EMPTY":   "",
	}

	t.Run("returns existing value", func(t *testing.T) {
		got, ok := store.Get("PRESENT")
		if !ok {
			t.Fatal("Get reported PRESENT missing")
		}
		if got != "value" {
			t.Fatalf("Get returned %q, want %q", got, "value")
		}
	})

	t.Run("distinguishes empty values from missing keys", func(t *testing.T) {
		got, ok := store.Get("EMPTY")
		if !ok {
			t.Fatal("Get reported EMPTY missing")
		}
		if got != "" {
			t.Fatalf("Get returned %q, want empty string", got)
		}
	})

	t.Run("returns false for missing key", func(t *testing.T) {
		got, ok := store.Get("MISSING")
		if ok {
			t.Fatal("Get reported MISSING present")
		}
		if got != "" {
			t.Fatalf("Get returned %q, want empty string", got)
		}
	})
}

func TestStoreGetRequired(t *testing.T) {
	store := Store{"PRESENT": "value"}

	t.Run("returns existing value", func(t *testing.T) {
		got, err := store.GetRequired("PRESENT")
		if err != nil {
			t.Fatalf("GetRequired returned unexpected error: %v", err)
		}
		if got != "value" {
			t.Fatalf("GetRequired returned %q, want %q", got, "value")
		}
	})

	t.Run("returns wrapped error for missing required key", func(t *testing.T) {
		got, err := store.GetRequired("MISSING")
		if err == nil {
			t.Fatal("GetRequired returned nil error for missing required key")
		}
		if got != "" {
			t.Fatalf("GetRequired returned %q, want empty string", got)
		}
		if !errors.Is(err, ErrMissingRequiredKey) {
			t.Fatalf("GetRequired error = %v, want wrapped ErrMissingRequiredKey", err)
		}
		if !strings.Contains(err.Error(), "MISSING") {
			t.Fatalf("GetRequired error = %q, want missing key name included", err.Error())
		}
	})
}

func TestStoreSet(t *testing.T) {
	t.Run("sets missing keys", func(t *testing.T) {
		store := NewStore()
		store.Set("KEY", "value", false)
		assertStoreEqual(t, store, Store{"KEY": "value"})
	})

	t.Run("does not overwrite existing values when overwrite is false", func(t *testing.T) {
		store := Store{"KEY": "original"}
		store.Set("KEY", "replacement", false)
		assertStoreEqual(t, store, Store{"KEY": "original"})
	})

	t.Run("overwrites existing values when overwrite is true", func(t *testing.T) {
		store := Store{"KEY": "original"}
		store.Set("KEY", "replacement", true)
		assertStoreEqual(t, store, Store{"KEY": "replacement"})
	})
}

func TestStoreProcessValue(t *testing.T) {
	t.Run("returns wrapped error for missing key", func(t *testing.T) {
		store := NewStore()

		err := store.ProcessValue("MISSING", nil)
		if err == nil {
			t.Fatal("ProcessValue returned nil error for missing key")
		}
		if !errors.Is(err, ErrMissingRequiredKey) {
			t.Fatalf("ProcessValue error = %v, want wrapped ErrMissingRequiredKey", err)
		}
		if !strings.Contains(err.Error(), "MISSING") {
			t.Fatalf("ProcessValue error = %q, want missing key name included", err.Error())
		}
	})

	t.Run("uses zero-value options when config is nil", func(t *testing.T) {
		store := Store{"KEY": "line1\\nline2\rline3"}

		if err := store.ProcessValue("KEY", nil); err != nil {
			t.Fatalf("ProcessValue returned unexpected error: %v", err)
		}

		if got := store["KEY"]; got != `line1\nline2`+"\r"+`line3` {
			t.Fatalf("processed value = %q, want %q", got, `line1\nline2`+"\r"+`line3`)
		}
	})

	t.Run("resolves base and key-specific parse config", func(t *testing.T) {
		store := Store{
			"KEY":   "a\\nb\rc",
			"OTHER": "d\\ne\rf",
		}
		cfg := new(Config)
		cfg.MergeGlobalOptions(Options{
			UnescapeBackslashN: new(true),
			TransformCRToLF:    new(true),
		})
		cfg.MergeKeyOptions("KEY", Options{
			UnescapeBackslashN: new(false),
			TransformCRToLF:    new(false),
		})

		if err := store.ProcessValue("KEY", cfg); err != nil {
			t.Fatalf("ProcessValue returned unexpected error for key-specific config: %v", err)
		}
		if err := store.ProcessValue("OTHER", cfg); err != nil {
			t.Fatalf("ProcessValue returned unexpected error for base config: %v", err)
		}

		assertStoreEqual(t, store, Store{
			"KEY":   "a\\nb\rc",
			"OTHER": "d\ne\nf",
		})
	})

	t.Run("leaves the original value unchanged when processing fails", func(t *testing.T) {
		store := Store{"KEY": "trailing\\"}
		cfg := new(Config)
		cfg.MergeGlobalOptions(Options{UnescapeBackslashBackslash: new(true)})

		err := store.ProcessValue("KEY", cfg)
		if err == nil {
			t.Fatal("ProcessValue returned nil error for invalid value")
		}
		if !strings.Contains(err.Error(), "KEY: trailing backslash in double-quoted value") {
			t.Fatalf("ProcessValue error = %q, want key-specific processing error", err.Error())
		}
		if got := store["KEY"]; got != "trailing\\" {
			t.Fatalf("processed value = %q, want original value preserved", got)
		}
	})
}

func TestStoreProcessValues(t *testing.T) {
	t.Run("uses zero-value options when config is nil", func(t *testing.T) {
		store := Store{
			"A": "a\\nb",
			"B": "c\rd",
		}

		if err := store.ProcessValues(nil); err != nil {
			t.Fatalf("ProcessValues returned unexpected error: %v", err)
		}

		assertStoreEqual(t, store, Store{
			"A": `a\nb`,
			"B": "c\rd",
		})
	})

	t.Run("applies base and key-specific parse config", func(t *testing.T) {
		store := Store{
			"BASE":  "a\\nb",
			"KEY":   "c\\nd\re",
			"OTHER": "f\rg",
		}
		cfg := new(Config)
		cfg.MergeGlobalOptions(Options{
			UnescapeBackslashN: new(true),
			TransformCRToLF:    new(true),
		})
		cfg.MergeKeyOptions("KEY", Options{
			UnescapeBackslashN: new(false),
			TransformCRToLF:    new(false),
		})

		if err := store.ProcessValues(cfg); err != nil {
			t.Fatalf("ProcessValues returned unexpected error: %v", err)
		}

		assertStoreEqual(t, store, Store{
			"BASE":  "a\nb",
			"KEY":   "c\\nd\re",
			"OTHER": "f\ng",
		})
	})

	t.Run("returns an error and leaves the store unchanged when any value fails", func(t *testing.T) {
		store := Store{
			"BAD":  "trailing\\",
			"GOOD": "a\\nb",
		}
		want := maps.Clone(store)
		cfg := new(Config)
		cfg.MergeGlobalOptions(Options{UnescapeBackslashBackslash: new(true)})

		err := store.ProcessValues(cfg)
		if err == nil {
			t.Fatal("ProcessValues returned nil error for invalid value")
		}
		if !strings.Contains(err.Error(), "BAD: trailing backslash in double-quoted value") {
			t.Fatalf("ProcessValues error = %q, want key-specific processing error", err.Error())
		}

		assertStoreEqual(t, store, want)
	})
}

func TestStoreSetFromOptionalDotEnv(t *testing.T) {
	t.Run("nil config uses zero-value options and preserves existing entries by default", func(t *testing.T) {
		store := Store{"EXISTING": "keep"}
		path := writeDotEnvFile(t, "EXISTING=replace\nNEW=\"line1\\nline2\"\n")

		if err := store.SetFromOptionalDotEnv(path, nil); err != nil {
			t.Fatalf("SetFromOptionalDotEnv returned unexpected error: %v", err)
		}

		assertStoreEqual(t, store, Store{
			"EXISTING": "keep",
			"NEW":      `line1\nline2`,
		})
	})

	t.Run("returns parser errors", func(t *testing.T) {
		store := NewStore()
		path := writeDotEnvFile(t, "INVALID LINE")

		err := store.SetFromOptionalDotEnv(path, nil)
		if err == nil {
			t.Fatal("SetFromOptionalDotEnv returned nil error for invalid dotenv file")
		}
		if !strings.Contains(err.Error(), filepath.Base(path)+":1:") {
			t.Fatalf("SetFromOptionalDotEnv error = %q, want file name and line number", err.Error())
		}
	})

	t.Run("is a no-op when the dotenv file is missing", func(t *testing.T) {
		store := Store{"EXISTING": "keep"}
		path := filepath.Join(t.TempDir(), "missing.env")

		if err := store.SetFromOptionalDotEnv(path, nil); err != nil {
			t.Fatalf("SetFromOptionalDotEnv returned unexpected error for missing file: %v", err)
		}

		assertStoreEqual(t, store, Store{"EXISTING": "keep"})
	})
}

func TestStoreSetFromRequiredDotEnv(t *testing.T) {
	t.Run("honors parse config overwrite", func(t *testing.T) {
		store := Store{"EXISTING": "keep"}
		path := writeDotEnvFile(t, "EXISTING=replace\nNEW=value\n")
		cfg := new(Config)
		cfg.MergeGlobalOptions(Options{Overwrite: new(true)})

		if err := store.SetFromRequiredDotEnv(path, cfg); err != nil {
			t.Fatalf("SetFromRequiredDotEnv returned unexpected error: %v", err)
		}

		assertStoreEqual(t, store, Store{
			"EXISTING": "replace",
			"NEW":      "value",
		})
	})

	t.Run("returns parser errors", func(t *testing.T) {
		store := NewStore()
		path := writeDotEnvFile(t, "INVALID LINE")

		err := store.SetFromRequiredDotEnv(path, nil)
		if err == nil {
			t.Fatal("SetFromRequiredDotEnv returned nil error for invalid dotenv file")
		}
		if !strings.Contains(err.Error(), filepath.Base(path)+":1:") {
			t.Fatalf("SetFromRequiredDotEnv error = %q, want file name and line number", err.Error())
		}
	})

	t.Run("returns ErrMissingDotEnv when the dotenv file is missing", func(t *testing.T) {
		store := Store{"EXISTING": "keep"}
		path := filepath.Join(t.TempDir(), "missing.env")

		err := store.SetFromRequiredDotEnv(path, nil)
		if err == nil {
			t.Fatal("SetFromRequiredDotEnv returned nil error for missing dotenv file")
		}
		if !errors.Is(err, ErrMissingDotEnv) {
			t.Fatalf("SetFromRequiredDotEnv error = %v, want wrapped ErrMissingDotEnv", err)
		}
		if !strings.Contains(err.Error(), path) {
			t.Fatalf("SetFromRequiredDotEnv error = %q, want missing path included", err.Error())
		}

		assertStoreEqual(t, store, Store{"EXISTING": "keep"})
	})
}

func TestStoreSetFromString(t *testing.T) {
	t.Run("nil config uses zero-value options and preserves existing entries by default", func(t *testing.T) {
		store := Store{"EXISTING": "keep"}

		if err := store.SetFromString("EXISTING=replace\nNEW=\"line1\\nline2\"\n", "inline.env", nil); err != nil {
			t.Fatalf("SetFromString returned unexpected error: %v", err)
		}

		assertStoreEqual(t, store, Store{
			"EXISTING": "keep",
			"NEW":      `line1\nline2`,
		})
	})

	t.Run("uses the default source name when name is empty", func(t *testing.T) {
		store := NewStore()

		err := store.SetFromString("INVALID LINE", "", nil)
		if err == nil {
			t.Fatal("SetFromString returned nil error for invalid dotenv string")
		}
		if !strings.Contains(err.Error(), "string:1:") {
			t.Fatalf("SetFromString error = %q, want default source name and line number", err.Error())
		}
	})
}

func TestStoreSetFromReader(t *testing.T) {
	t.Run("loads values and honors parse config overwrite", func(t *testing.T) {
		store := Store{"EXISTING": "keep"}
		cfg := new(Config)
		cfg.MergeGlobalOptions(Options{Overwrite: new(true)})

		err := store.SetFromReader(strings.NewReader("EXISTING=replace\nNEW=value\n"), "reader.env", cfg)
		if err != nil {
			t.Fatalf("SetFromReader returned unexpected error: %v", err)
		}

		assertStoreEqual(t, store, Store{
			"EXISTING": "replace",
			"NEW":      "value",
		})
	})

	t.Run("uses the default source name when name is empty", func(t *testing.T) {
		store := NewStore()

		err := store.SetFromReader(strings.NewReader("INVALID LINE"), "", nil)
		if err == nil {
			t.Fatal("SetFromReader returned nil error for invalid dotenv reader")
		}
		if !strings.Contains(err.Error(), "io.Reader:1:") {
			t.Fatalf("SetFromReader error = %q, want default source name and line number", err.Error())
		}
	})

	t.Run("returns an error for a nil reader", func(t *testing.T) {
		store := NewStore()

		err := store.SetFromReader(nil, "reader.env", nil)
		if err == nil {
			t.Fatal("SetFromReader returned nil error for nil reader")
		}
		if err.Error() != "parse reader cannot be nil" {
			t.Fatalf("SetFromReader error = %q, want %q", err.Error(), "parse reader cannot be nil")
		}
	})
}

func TestStoreSetFromOsEnviron(t *testing.T) {
	t.Run("respects allowlist denylist and overwrite=false", func(t *testing.T) {
		allowedKey := testEnvKey(t, "allowed")
		deniedKey := testEnvKey(t, "denied")
		existingKey := testEnvKey(t, "existing")
		equalsKey := testEnvKey(t, "equals")
		filteredKey := testEnvKey(t, "filtered")

		setTestEnv(t, allowedKey, "allowed-value")
		setTestEnv(t, deniedKey, "denied-value")
		setTestEnv(t, existingKey, "from-os")
		setTestEnv(t, equalsKey, "left=right")
		setTestEnv(t, filteredKey, "filtered-out")

		store := Store{
			"UNCHANGED": "value",
			existingKey: "from-store",
		}

		store.SetFromOsEnviron(keySet(allowedKey, deniedKey, existingKey, equalsKey), keySet(deniedKey), false)

		assertStoreEqual(t, store, Store{
			"UNCHANGED": "value",
			allowedKey:  "allowed-value",
			existingKey: "from-store",
			equalsKey:   "left=right",
		})
	})

	t.Run("imports from os environment when allowlist is nil and overwrite=true", func(t *testing.T) {
		importedKey := testEnvKey(t, "imported")
		overwrittenKey := testEnvKey(t, "overwritten")
		deniedKey := testEnvKey(t, "denied")

		setTestEnv(t, importedKey, "from-os")
		setTestEnv(t, overwrittenKey, "from-os")
		setTestEnv(t, deniedKey, "blocked")

		store := Store{
			overwrittenKey: "from-store",
		}

		store.SetFromOsEnviron(nil, keySet(deniedKey), true)

		if got := store[importedKey]; got != "from-os" {
			t.Fatalf("imported key = %q, want %q", got, "from-os")
		}
		if got := store[overwrittenKey]; got != "from-os" {
			t.Fatalf("overwritten key = %q, want %q", got, "from-os")
		}
		if _, ok := store[deniedKey]; ok {
			t.Fatalf("denied key %q should not be present in store", deniedKey)
		}
	})
}

func TestStoreLoadIntoOsEnviron(t *testing.T) {
	t.Run("loads missing values and respects filters without overwrite", func(t *testing.T) {
		missingKey := testEnvKey(t, "missing")
		existingKey := testEnvKey(t, "existing")
		deniedKey := testEnvKey(t, "denied")
		filteredKey := testEnvKey(t, "filtered")

		unsetTestEnv(t, missingKey)
		setTestEnv(t, existingKey, "from-os")
		unsetTestEnv(t, deniedKey)
		unsetTestEnv(t, filteredKey)

		store := Store{
			missingKey:  "from-store",
			existingKey: "from-store",
			deniedKey:   "blocked",
			filteredKey: "filtered-out",
		}

		store.LoadIntoOsEnviron(keySet(missingKey, existingKey, deniedKey), keySet(deniedKey), false)

		assertEnvValue(t, missingKey, "from-store")
		assertEnvValue(t, existingKey, "from-os")
		assertEnvMissing(t, deniedKey)
		assertEnvMissing(t, filteredKey)
	})

	t.Run("overwrites existing values when overwrite=true and allowlist is nil", func(t *testing.T) {
		missingKey := testEnvKey(t, "missing")
		existingKey := testEnvKey(t, "existing")
		deniedKey := testEnvKey(t, "denied")

		unsetTestEnv(t, missingKey)
		setTestEnv(t, existingKey, "from-os")
		unsetTestEnv(t, deniedKey)

		store := Store{
			missingKey:  "from-store",
			existingKey: "from-store",
			deniedKey:   "blocked",
		}

		store.LoadIntoOsEnviron(nil, keySet(deniedKey), true)

		assertEnvValue(t, missingKey, "from-store")
		assertEnvValue(t, existingKey, "from-store")
		assertEnvMissing(t, deniedKey)
	})
}

func TestStoreMerge(t *testing.T) {
	t.Run("does not overwrite existing keys when overwrite is false", func(t *testing.T) {
		store := Store{"EXISTING": "keep"}
		store.Merge(Store{
			"EXISTING": "replace",
			"NEW":      "value",
		}, false)

		assertStoreEqual(t, store, Store{
			"EXISTING": "keep",
			"NEW":      "value",
		})
	})

	t.Run("overwrites existing keys when overwrite is true", func(t *testing.T) {
		store := Store{"EXISTING": "keep"}
		store.Merge(Store{
			"EXISTING": "replace",
			"NEW":      "value",
		}, true)

		assertStoreEqual(t, store, Store{
			"EXISTING": "replace",
			"NEW":      "value",
		})
	})
}

func TestStoreFilterKeys(t *testing.T) {
	t.Run("keeps only allowlisted keys when denylist is nil", func(t *testing.T) {
		store := Store{
			"KEEP": "value",
			"DROP": "value",
		}

		store.FilterKeys(keySet("KEEP"), nil)
		assertStoreEqual(t, store, Store{"KEEP": "value"})
	})

	t.Run("removes only denylisted keys when allowlist is nil", func(t *testing.T) {
		store := Store{
			"KEEP": "value",
			"DROP": "value",
		}

		store.FilterKeys(nil, keySet("DROP"))
		assertStoreEqual(t, store, Store{"KEEP": "value"})
	})

	t.Run("denylist wins when a key appears in both filters", func(t *testing.T) {
		store := Store{
			"KEEP": "value",
			"BOTH": "value",
			"DROP": "value",
		}

		store.FilterKeys(keySet("KEEP", "BOTH"), keySet("BOTH"))
		assertStoreEqual(t, store, Store{"KEEP": "value"})
	})

	t.Run("does nothing when both filters are nil", func(t *testing.T) {
		store := Store{
			"KEEP": "value",
			"DROP": "value",
		}

		store.FilterKeys(nil, nil)
		assertStoreEqual(t, store, Store{
			"KEEP": "value",
			"DROP": "value",
		})
	})
}

func writeDotEnvFile(t *testing.T, contents string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("os.WriteFile: %v", err)
	}

	return path
}

func testEnvKey(t *testing.T, suffix string) string {
	t.Helper()

	name := strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z':
			return r - ('a' - 'A')
		case r >= 'A' && r <= 'Z':
			return r
		case r >= '0' && r <= '9':
			return r
		default:
			return '_'
		}
	}, t.Name())
	return fmt.Sprintf("STRICT_DOTENV_%d_%s_%s", os.Getpid(), name, suffix)
}

func setTestEnv(t *testing.T, key, value string) {
	t.Helper()

	old, hadOld := os.LookupEnv(key)
	if err := os.Setenv(key, value); err != nil {
		t.Fatalf("os.Setenv(%q): %v", key, err)
	}

	t.Cleanup(func() {
		if hadOld {
			_ = os.Setenv(key, old)
			return
		}
		_ = os.Unsetenv(key)
	})
}

func unsetTestEnv(t *testing.T, key string) {
	t.Helper()

	old, hadOld := os.LookupEnv(key)
	if err := os.Unsetenv(key); err != nil {
		t.Fatalf("os.Unsetenv(%q): %v", key, err)
	}

	t.Cleanup(func() {
		if hadOld {
			_ = os.Setenv(key, old)
		}
	})
}

func assertEnvValue(t *testing.T, key, want string) {
	t.Helper()

	got, ok := os.LookupEnv(key)
	if !ok {
		t.Fatalf("environment variable %q is not set", key)
	}
	if got != want {
		t.Fatalf("environment variable %q = %q, want %q", key, got, want)
	}
}

func assertEnvMissing(t *testing.T, key string) {
	t.Helper()

	if got, ok := os.LookupEnv(key); ok {
		t.Fatalf("environment variable %q should be unset, got %q", key, got)
	}
}

func keySet(keys ...string) map[string]struct{} {
	set := make(map[string]struct{}, len(keys))
	for _, key := range keys {
		set[key] = struct{}{}
	}
	return set
}

func assertStoreEqual(t *testing.T, got, want Store) {
	t.Helper()

	if !maps.Equal(got, want) {
		t.Fatalf("store mismatch\n got: %#v\nwant: %#v", got, want)
	}
}
