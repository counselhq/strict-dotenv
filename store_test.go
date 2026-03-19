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
	store := NewStore(0)
	if store.Len() != 0 {
		t.Fatalf("NewStore returned non-empty store: %v", store)
	}

	store.Set("KEY", "value", false)
	got, ok := store.Get("KEY")
	if !ok || got != "value" {
		t.Fatalf("store should be writable, got %q, ok=%t", got, ok)
	}
}

func TestZeroValueStoreSet(t *testing.T) {
	var store Store

	store.Set("KEY", "value", false)

	assertStoreEqual(t, &store, map[string]string{"KEY": "value"})
}

func TestStoreGet(t *testing.T) {
	store := storeOf(map[string]string{
		"PRESENT": "value",
		"EMPTY":   "",
	})

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
	store := storeOf(map[string]string{"PRESENT": "value"})

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
		store := NewStore(0)
		store.Set("KEY", "value", false)
		assertStoreEqual(t, store, map[string]string{"KEY": "value"})
	})

	t.Run("does not overwrite existing values when overwrite is false", func(t *testing.T) {
		store := storeOf(map[string]string{"KEY": "original"})
		store.Set("KEY", "replacement", false)
		assertStoreEqual(t, store, map[string]string{"KEY": "original"})
	})

	t.Run("overwrites existing values when overwrite is true", func(t *testing.T) {
		store := storeOf(map[string]string{"KEY": "original"})
		store.Set("KEY", "replacement", true)
		assertStoreEqual(t, store, map[string]string{"KEY": "replacement"})
	})
}

func TestStoreMergeMap(t *testing.T) {
	t.Run("does not overwrite existing values when overwrite is false", func(t *testing.T) {
		store := storeOf(map[string]string{"EXISTING": "keep"})
		store.MergeMap(map[string]string{
			"EXISTING": "replace",
			"NEW":      "value",
		}, false)

		assertStoreEqual(t, store, map[string]string{
			"EXISTING": "keep",
			"NEW":      "value",
		})
	})

	t.Run("overwrites existing values when overwrite is true", func(t *testing.T) {
		store := storeOf(map[string]string{"EXISTING": "keep"})
		store.MergeMap(map[string]string{
			"EXISTING": "replace",
			"NEW":      "value",
		}, true)

		assertStoreEqual(t, store, map[string]string{
			"EXISTING": "replace",
			"NEW":      "value",
		})
	})
}

func TestStoreTransformValue(t *testing.T) {
	t.Run("returns wrapped error for missing key", func(t *testing.T) {
		store := NewStore(0)

		err := store.TransformValue("MISSING", nil)
		if err == nil {
			t.Fatal("TransformValue returned nil error for missing key")
		}
		if !errors.Is(err, ErrMissingRequiredKey) {
			t.Fatalf("TransformValue error = %v, want wrapped ErrMissingRequiredKey", err)
		}
		if !strings.Contains(err.Error(), "MISSING") {
			t.Fatalf("TransformValue error = %q, want missing key name included", err.Error())
		}
	})

	t.Run("uses zero-value options when config is nil", func(t *testing.T) {
		store := storeOf(map[string]string{"KEY": "line1\\nline2\rline3"})

		if err := store.TransformValue("KEY", nil); err != nil {
			t.Fatalf("TransformValue returned unexpected error: %v", err)
		}

		got, _ := store.Get("KEY")
		if got != `line1\nline2`+"\r"+`line3` {
			t.Fatalf("processed value = %q, want %q", got, `line1\nline2`+"\r"+`line3`)
		}
	})

	t.Run("resolves base and key-specific parse config", func(t *testing.T) {
		store := storeOf(map[string]string{
			"KEY":   "a\\nb\rc",
			"OTHER": "d\\ne\rf",
		})
		cfg := new(Config)
		cfg.MergeGlobalOptions(Options{
			UnescapeBackslashN: new(true),
			TransformCRToLF:    new(true),
		})
		cfg.MergeKeyOptions("KEY", Options{
			UnescapeBackslashN: new(false),
			TransformCRToLF:    new(false),
		})

		if err := store.TransformValue("KEY", cfg); err != nil {
			t.Fatalf("TransformValue returned unexpected error for key-specific config: %v", err)
		}
		if err := store.TransformValue("OTHER", cfg); err != nil {
			t.Fatalf("TransformValue returned unexpected error for base config: %v", err)
		}

		assertStoreEqual(t, store, map[string]string{
			"KEY":   "a\\nb\rc",
			"OTHER": "d\ne\nf",
		})
	})

	t.Run("leaves the original value unchanged when processing fails", func(t *testing.T) {
		store := storeOf(map[string]string{"KEY": "trailing\\"})
		cfg := new(Config)
		cfg.MergeGlobalOptions(Options{UnescapeBackslashBackslash: new(true)})

		err := store.TransformValue("KEY", cfg)
		if err == nil {
			t.Fatal("TransformValue returned nil error for invalid value")
		}
		if !strings.Contains(err.Error(), "KEY: trailing backslash in double-quoted value") {
			t.Fatalf("TransformValue error = %q, want key-specific processing error", err.Error())
		}
		got, _ := store.Get("KEY")
		if got != "trailing\\" {
			t.Fatalf("processed value = %q, want original value preserved", got)
		}
	})
}

func TestStoreTransformValues(t *testing.T) {
	t.Run("uses zero-value options when config is nil", func(t *testing.T) {
		store := storeOf(map[string]string{
			"A": "a\\nb",
			"B": "c\rd",
		})

		if err := store.TransformValues(nil); err != nil {
			t.Fatalf("TransformValues returned unexpected error: %v", err)
		}

		assertStoreEqual(t, store, map[string]string{
			"A": `a\nb`,
			"B": "c\rd",
		})
	})

	t.Run("applies base and key-specific parse config", func(t *testing.T) {
		store := storeOf(map[string]string{
			"BASE":  "a\\nb",
			"KEY":   "c\\nd\re",
			"OTHER": "f\rg",
		})
		cfg := new(Config)
		cfg.MergeGlobalOptions(Options{
			UnescapeBackslashN: new(true),
			TransformCRToLF:    new(true),
		})
		cfg.MergeKeyOptions("KEY", Options{
			UnescapeBackslashN: new(false),
			TransformCRToLF:    new(false),
		})

		if err := store.TransformValues(cfg); err != nil {
			t.Fatalf("TransformValues returned unexpected error: %v", err)
		}

		assertStoreEqual(t, store, map[string]string{
			"BASE":  "a\nb",
			"KEY":   "c\\nd\re",
			"OTHER": "f\ng",
		})
	})

	t.Run("returns an error and leaves the store unchanged when any value fails", func(t *testing.T) {
		store := storeOf(map[string]string{
			"BAD":  "trailing\\",
			"GOOD": "a\\nb",
		})
		want := store.Entries()
		cfg := new(Config)
		cfg.MergeGlobalOptions(Options{UnescapeBackslashBackslash: new(true)})

		err := store.TransformValues(cfg)
		if err == nil {
			t.Fatal("TransformValues returned nil error for invalid value")
		}
		if !strings.Contains(err.Error(), "BAD: trailing backslash in double-quoted value") {
			t.Fatalf("TransformValues error = %q, want key-specific processing error", err.Error())
		}

		assertStoreEqual(t, store, want)
	})
}

func TestStoreParseOptionalDotEnv(t *testing.T) {
	t.Run("nil config uses zero-value options and preserves existing entries by default", func(t *testing.T) {
		store := storeOf(map[string]string{"EXISTING": "keep"})
		path := writeDotEnvFile(t, "EXISTING=replace\nNEW=\"line1\\nline2\"\n")

		if err := store.ParseOptionalDotEnv(path, nil); err != nil {
			t.Fatalf("ParseOptionalDotEnv returned unexpected error: %v", err)
		}

		assertStoreEqual(t, store, map[string]string{
			"EXISTING": "keep",
			"NEW":      `line1\nline2`,
		})
	})

	t.Run("returns parser errors", func(t *testing.T) {
		store := NewStore(0)
		path := writeDotEnvFile(t, "INVALID LINE")

		err := store.ParseOptionalDotEnv(path, nil)
		if err == nil {
			t.Fatal("ParseOptionalDotEnv returned nil error for invalid dotenv file")
		}
		if !strings.Contains(err.Error(), filepath.Base(path)+":1:") {
			t.Fatalf("ParseOptionalDotEnv error = %q, want file name and line number", err.Error())
		}
	})

	t.Run("is a no-op when the dotenv file is missing", func(t *testing.T) {
		store := storeOf(map[string]string{"EXISTING": "keep"})
		path := filepath.Join(t.TempDir(), "missing.env")

		if err := store.ParseOptionalDotEnv(path, nil); err != nil {
			t.Fatalf("ParseOptionalDotEnv returned unexpected error for missing file: %v", err)
		}

		assertStoreEqual(t, store, map[string]string{"EXISTING": "keep"})
	})
}

func TestStoreParseRequiredDotEnv(t *testing.T) {
	t.Run("honors parse config overwrite", func(t *testing.T) {
		store := storeOf(map[string]string{"EXISTING": "keep"})
		path := writeDotEnvFile(t, "EXISTING=replace\nNEW=value\n")
		cfg := new(Config)
		cfg.MergeGlobalOptions(Options{Overwrite: new(true)})

		if err := store.ParseRequiredDotEnv(path, cfg); err != nil {
			t.Fatalf("ParseRequiredDotEnv returned unexpected error: %v", err)
		}

		assertStoreEqual(t, store, map[string]string{
			"EXISTING": "replace",
			"NEW":      "value",
		})
	})

	t.Run("returns parser errors", func(t *testing.T) {
		store := NewStore(0)
		path := writeDotEnvFile(t, "INVALID LINE")

		err := store.ParseRequiredDotEnv(path, nil)
		if err == nil {
			t.Fatal("ParseRequiredDotEnv returned nil error for invalid dotenv file")
		}
		if !strings.Contains(err.Error(), filepath.Base(path)+":1:") {
			t.Fatalf("ParseRequiredDotEnv error = %q, want file name and line number", err.Error())
		}
	})

	t.Run("returns ErrMissingDotEnv when the dotenv file is missing", func(t *testing.T) {
		store := storeOf(map[string]string{"EXISTING": "keep"})
		path := filepath.Join(t.TempDir(), "missing.env")

		err := store.ParseRequiredDotEnv(path, nil)
		if err == nil {
			t.Fatal("ParseRequiredDotEnv returned nil error for missing dotenv file")
		}
		if !errors.Is(err, ErrMissingDotEnv) {
			t.Fatalf("ParseRequiredDotEnv error = %v, want wrapped ErrMissingDotEnv", err)
		}
		if !strings.Contains(err.Error(), path) {
			t.Fatalf("ParseRequiredDotEnv error = %q, want missing path included", err.Error())
		}

		assertStoreEqual(t, store, map[string]string{"EXISTING": "keep"})
	})
}

func TestStoreParseString(t *testing.T) {
	t.Run("nil config uses zero-value options and preserves existing entries by default", func(t *testing.T) {
		store := storeOf(map[string]string{"EXISTING": "keep"})

		if err := store.ParseString("EXISTING=replace\nNEW=\"line1\\nline2\"\n", "inline.env", nil); err != nil {
			t.Fatalf("ParseString returned unexpected error: %v", err)
		}

		assertStoreEqual(t, store, map[string]string{
			"EXISTING": "keep",
			"NEW":      `line1\nline2`,
		})
	})

	t.Run("uses the default source name when name is empty", func(t *testing.T) {
		store := NewStore(0)

		err := store.ParseString("INVALID LINE", "", nil)
		if err == nil {
			t.Fatal("ParseString returned nil error for invalid dotenv string")
		}
		if !strings.Contains(err.Error(), "string:1:") {
			t.Fatalf("ParseString error = %q, want default source name and line number", err.Error())
		}
	})
}

func TestStoreParseReader(t *testing.T) {
	t.Run("loads values and honors parse config overwrite", func(t *testing.T) {
		store := storeOf(map[string]string{"EXISTING": "keep"})
		cfg := new(Config)
		cfg.MergeGlobalOptions(Options{Overwrite: new(true)})

		err := store.ParseReader(strings.NewReader("EXISTING=replace\nNEW=value\n"), "reader.env", cfg)
		if err != nil {
			t.Fatalf("ParseReader returned unexpected error: %v", err)
		}

		assertStoreEqual(t, store, map[string]string{
			"EXISTING": "replace",
			"NEW":      "value",
		})
	})

	t.Run("uses the default source name when name is empty", func(t *testing.T) {
		store := NewStore(0)

		err := store.ParseReader(strings.NewReader("INVALID LINE"), "", nil)
		if err == nil {
			t.Fatal("ParseReader returned nil error for invalid dotenv reader")
		}
		if !strings.Contains(err.Error(), "io.Reader:1:") {
			t.Fatalf("ParseReader error = %q, want default source name and line number", err.Error())
		}
	})

	t.Run("returns an error for a nil reader", func(t *testing.T) {
		store := NewStore(0)

		err := store.ParseReader(nil, "reader.env", nil)
		if err == nil {
			t.Fatal("ParseReader returned nil error for nil reader")
		}
		if err.Error() != "parse reader cannot be nil" {
			t.Fatalf("ParseReader error = %q, want %q", err.Error(), "parse reader cannot be nil")
		}
	})
}

func TestStoreImportFromEnv(t *testing.T) {
	t.Run("respects allowkeys denykeys and overwrite=false", func(t *testing.T) {
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

		store := storeOf(map[string]string{
			"UNCHANGED": "value",
			existingKey: "from-store",
		})

		store.ImportFromEnv(keySet(allowedKey, deniedKey, existingKey, equalsKey), keySet(deniedKey), false)

		assertStoreEqual(t, store, map[string]string{
			"UNCHANGED": "value",
			allowedKey:  "allowed-value",
			existingKey: "from-store",
			equalsKey:   "left=right",
		})
	})

	t.Run("imports from os environment when allowkeys is nil and overwrite=true", func(t *testing.T) {
		importedKey := testEnvKey(t, "imported")
		overwrittenKey := testEnvKey(t, "overwritten")
		deniedKey := testEnvKey(t, "denied")

		setTestEnv(t, importedKey, "from-os")
		setTestEnv(t, overwrittenKey, "from-os")
		setTestEnv(t, deniedKey, "blocked")

		store := storeOf(map[string]string{
			overwrittenKey: "from-store",
		})

		store.ImportFromEnv(nil, keySet(deniedKey), true)

		got, _ := store.Get(importedKey)
		if got != "from-os" {
			t.Fatalf("imported key = %q, want %q", got, "from-os")
		}
		got, _ = store.Get(overwrittenKey)
		if got != "from-os" {
			t.Fatalf("overwritten key = %q, want %q", got, "from-os")
		}
		if _, ok := store.Get(deniedKey); ok {
			t.Fatalf("denied key %q should not be present in store", deniedKey)
		}
	})
}

func TestStoreExportToEnv(t *testing.T) {
	t.Run("loads missing values and respects filters without overwrite", func(t *testing.T) {
		missingKey := testEnvKey(t, "missing")
		existingKey := testEnvKey(t, "existing")
		deniedKey := testEnvKey(t, "denied")
		filteredKey := testEnvKey(t, "filtered")

		unsetTestEnv(t, missingKey)
		setTestEnv(t, existingKey, "from-os")
		unsetTestEnv(t, deniedKey)
		unsetTestEnv(t, filteredKey)

		store := storeOf(map[string]string{
			missingKey:  "from-store",
			existingKey: "from-store",
			deniedKey:   "blocked",
			filteredKey: "filtered-out",
		})

		store.ExportToEnv(keySet(missingKey, existingKey, deniedKey), keySet(deniedKey), false)

		assertEnvValue(t, missingKey, "from-store")
		assertEnvValue(t, existingKey, "from-os")
		assertEnvMissing(t, deniedKey)
		assertEnvMissing(t, filteredKey)
	})

	t.Run("overwrites existing values when overwrite=true and allowkeys is nil", func(t *testing.T) {
		missingKey := testEnvKey(t, "missing")
		existingKey := testEnvKey(t, "existing")
		deniedKey := testEnvKey(t, "denied")

		unsetTestEnv(t, missingKey)
		setTestEnv(t, existingKey, "from-os")
		unsetTestEnv(t, deniedKey)

		store := storeOf(map[string]string{
			missingKey:  "from-store",
			existingKey: "from-store",
			deniedKey:   "blocked",
		})

		store.ExportToEnv(nil, keySet(deniedKey), true)

		assertEnvValue(t, missingKey, "from-store")
		assertEnvValue(t, existingKey, "from-store")
		assertEnvMissing(t, deniedKey)
	})
}

func TestStoreMergeStore(t *testing.T) {
	t.Run("does not overwrite existing keys when overwrite is false", func(t *testing.T) {
		store := storeOf(map[string]string{"EXISTING": "keep"})
		store.MergeStore(storeOf(map[string]string{
			"EXISTING": "replace",
			"NEW":      "value",
		}), false)

		assertStoreEqual(t, store, map[string]string{
			"EXISTING": "keep",
			"NEW":      "value",
		})
	})

	t.Run("overwrites existing keys when overwrite is true", func(t *testing.T) {
		store := storeOf(map[string]string{"EXISTING": "keep"})
		store.MergeStore(storeOf(map[string]string{
			"EXISTING": "replace",
			"NEW":      "value",
		}), true)

		assertStoreEqual(t, store, map[string]string{
			"EXISTING": "replace",
			"NEW":      "value",
		})
	})
}

func TestStoreFilter(t *testing.T) {
	t.Run("keeps only allowed keys when denykeys is nil", func(t *testing.T) {
		store := storeOf(map[string]string{
			"KEEP": "value",
			"DROP": "value",
		})

		store.Filter(keySet("KEEP"), nil)
		assertStoreEqual(t, store, map[string]string{"KEEP": "value"})
	})

	t.Run("removes only denied keys when allowkeys is nil", func(t *testing.T) {
		store := storeOf(map[string]string{
			"KEEP": "value",
			"DROP": "value",
		})

		store.Filter(nil, keySet("DROP"))
		assertStoreEqual(t, store, map[string]string{"KEEP": "value"})
	})

	t.Run("denykeys wins when a key appears in both filters", func(t *testing.T) {
		store := storeOf(map[string]string{
			"KEEP": "value",
			"BOTH": "value",
			"DROP": "value",
		})

		store.Filter(keySet("KEEP", "BOTH"), keySet("BOTH"))
		assertStoreEqual(t, store, map[string]string{"KEEP": "value"})
	})

	t.Run("does nothing when both filters are nil", func(t *testing.T) {
		store := storeOf(map[string]string{
			"KEEP": "value",
			"DROP": "value",
		})

		store.Filter(nil, nil)
		assertStoreEqual(t, store, map[string]string{
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

func storeOf(data map[string]string) *Store {
	store := NewStore(len(data))
	store.MergeMap(data, true)
	return store
}

func assertStoreEqual(t *testing.T, got *Store, want map[string]string) {
	t.Helper()

	if !maps.Equal(got.Entries(), want) {
		t.Fatalf("store mismatch\n got: %#v\nwant: %#v", got.Entries(), want)
	}
}
