package strictdotenv

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

var (
	ErrMissingDotEnv      = fmt.Errorf("EnvStore missing dotenv file")
	ErrMissingRequiredKey = fmt.Errorf("EnvStore missing required key")
)

type EnvStore map[string]string

// NewEnvStore makes a new EnvStore with an empty data map.
func NewEnvStore() EnvStore {
	return make(EnvStore)
}

// Get retrieves a value from the store and reports whether the key was set.
func (e EnvStore) Get(key string) (string, bool) {
	value, ok := e[key]
	return value, ok
}

// GetRequired retrieves a value from the store, returning an error if the key is not set.
func (e EnvStore) GetRequired(key string) (string, error) {
	value, ok := e.Get(key)
	if !ok {
		return "", fmt.Errorf("%w: %s", ErrMissingRequiredKey, key)
	}

	return value, nil
}

// Set sets a value in the store, optionally overwriting an existing value.
func (e EnvStore) Set(key, value string, overwrite bool) {
	if _, ok := e[key]; !ok || overwrite {
		e[key] = value
	}
}

// Merge the key-value pairs from another EnvStore into the current one, optionally overwriting existing values.
func (e EnvStore) Merge(store EnvStore, overwrite bool) {
	for k, v := range store {
		e.Set(k, v, overwrite)
	}
}

// SetFromOptionalDotEnv parses a dotenv file into the store using cfg.
// If the file does not exist, it returns nil without mutating the store.
// A nil cfg is treated as an all-zero Config.
func (e EnvStore) SetFromOptionalDotEnv(path string, cfg *Config) error {
	return e.setFromDotEnv(path, cfg, true)
}

// SetFromRequiredDotEnv parses a dotenv file into the store using cfg.
// If the file does not exist, it returns ErrMissingDotEnv.
// A nil cfg is treated as an all-zero Config.
func (e EnvStore) SetFromRequiredDotEnv(path string, cfg *Config) error {
	return e.setFromDotEnv(path, cfg, false)
}

func (e EnvStore) setFromDotEnv(path string, cfg *Config, optional bool) error {
	err := parseDotEnv(path, e, cfg)
	if err == nil {
		return nil
	}

	if errors.Is(err, os.ErrNotExist) {
		if optional {
			return nil
		}
		return fmt.Errorf("%w: %s", ErrMissingDotEnv, path)
	}

	return err
}

// SetFromString parses dotenv contents from a string into the store using cfg.
// A nil cfg is treated as an all-zero Config.
func (e EnvStore) SetFromString(s, name string, cfg *Config) error {
	return parseString(s, name, e, cfg)
}

// SetFromReader parses dotenv contents from an io.Reader into the store using cfg.
// A nil cfg is treated as an all-zero Config.
func (e EnvStore) SetFromReader(r io.Reader, name string, cfg *Config) error {
	return parseReader(r, name, e, cfg)
}

// SetFromOsEnviron reads the current process environment variables and stores them in the EnvStore.
func (e EnvStore) SetFromOsEnviron(allowlist, denylist map[string]struct{}, overwrite bool) {
	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)

		if len(parts) != 2 {
			continue
		}

		if allowlist != nil {
			if _, ok := allowlist[parts[0]]; !ok {
				continue
			}
		}

		if denylist != nil {
			if _, ok := denylist[parts[0]]; ok {
				continue
			}
		}

		e.Set(parts[0], parts[1], overwrite)
	}
}

// LoadIntoOsEnviron loads the key-value pairs from the EnvStore into
// the process environment variables, optionally overwriting existing values.
func (e EnvStore) LoadIntoOsEnviron(allowlist, denylist map[string]struct{}, overwrite bool) {
	for k, v := range e {

		if allowlist != nil {
			if _, ok := allowlist[k]; !ok {
				continue
			}
		}

		if denylist != nil {
			if _, ok := denylist[k]; ok {
				continue
			}
		}

		if _, exists := os.LookupEnv(k); !exists || overwrite {
			os.Setenv(k, v)
		}
	}
}

// FilterKeys removes any keys that are not in the allowlist, or that are in the denylist.
// A nil allowlist keeps all keys. A nil denylist removes no keys.
func (e EnvStore) FilterKeys(allowlist, denylist map[string]struct{}) {
	for storeKey := range e {
		if allowlist != nil {
			if _, ok := allowlist[storeKey]; !ok {
				delete(e, storeKey)
				continue
			}
		}

		if denylist != nil {
			if _, ok := denylist[storeKey]; ok {
				delete(e, storeKey)
			}
		}
	}
}

// ProcessValue applies the double-quoted value transform pipeline to an
// existing store value in place. The stored value is treated as the raw bytes
// that would have appeared between double quotes in a dotenv file. If the key
// is missing, ErrMissingRequiredKey is returned. A nil cfg is treated as an
// all-zero Config. Overwrite is ignored.
func (e EnvStore) ProcessValue(key string, cfg *Config) error {
	value, err := e.GetRequired(key)
	if err != nil {
		return err
	}

	processed, err := processValue([]byte(value), resolveOptions(key, cfg))
	if err != nil {
		return fmt.Errorf("%s: %w", key, err)
	}

	e[key] = processed
	return nil
}

// ProcessValues applies the double-quoted value transform pipeline to every
// value in the store. The Config is resolved the same way the parser uses
// it: base settings apply to every key unless that key has explicit overrides.
// A nil cfg is treated as an all-zero Config. Overwrite is ignored. If any
// key fails to process, the store is left unchanged.
func (e EnvStore) ProcessValues(cfg *Config) error {
	keys := make([]string, len(e))
	values := make([]string, len(e))

	i := 0
	for key, raw := range e {
		value, err := processValue([]byte(raw), resolveOptions(key, cfg))
		if err != nil {
			return fmt.Errorf("%s: %w", key, err)
		}
		keys[i] = key
		values[i] = value
		i++
	}

	for i, key := range keys {
		e[key] = values[i]
	}
	return nil
}
