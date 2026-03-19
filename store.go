package strictdotenv

import (
	"errors"
	"fmt"
	"io"
	"maps"
	"os"
	"strings"
)

var (
	ErrMissingDotEnv      = fmt.Errorf("Store missing dotenv file")
	ErrMissingRequiredKey = fmt.Errorf("Store missing required key")
)

// Store holds dotenv key-value pairs.
// The zero value is ready to use.
type Store struct {
	data map[string]string
}

// NewStore makes a new Store with an empty data map sized for up to size entries.
func NewStore(size int) *Store {
	return &Store{
		data: make(map[string]string, size),
	}
}

// Len reports how many key-value pairs are stored.
func (s *Store) Len() int {
	return len(s.data)
}

// Entries returns a copy of the store's key-value pairs.
func (s *Store) Entries() map[string]string {
	return maps.Clone(s.data)
}

// MergeMap copies key-value pairs from the provided map into the store.
func (s *Store) MergeMap(data map[string]string, overwrite bool) {
	if len(data) == 0 {
		return
	}

	if s.data == nil {
		s.data = maps.Clone(data)
		return
	}

	for key, value := range data {
		s.Set(key, value, overwrite)
	}
}

// Get retrieves a value from the store and reports whether the key was set.
func (s *Store) Get(key string) (string, bool) {
	value, ok := s.data[key]
	return value, ok
}

// GetRequired retrieves a value from the store, returning an error if the key is not set.
func (s *Store) GetRequired(key string) (string, error) {
	value, ok := s.Get(key)
	if !ok {
		return "", fmt.Errorf("%w: %s", ErrMissingRequiredKey, key)
	}

	return value, nil
}

// Set sets a value in the store, optionally overwriting an existing value.
func (s *Store) Set(key, value string, overwrite bool) {
	if s.data == nil {
		s.data = make(map[string]string)
	}

	if _, ok := s.data[key]; !ok || overwrite {
		s.data[key] = value
	}
}

// MergeStore copies key-value pairs from another Store into the current one,
// optionally overwriting existing values.
func (s *Store) MergeStore(store *Store, overwrite bool) {
	if store == nil {
		return
	}

	for key, value := range store.data {
		s.Set(key, value, overwrite)
	}
}

// ParseOptionalDotEnv parses a dotenv file into the store using cfg.
// If the file does not exist, it returns nil without mutating the store.
// A nil cfg is treated as an all-zero Config.
func (s *Store) ParseOptionalDotEnv(path string, cfg *Config) error {
	return s.parseDotEnv(path, cfg, true)
}

// ParseRequiredDotEnv parses a dotenv file into the store using cfg.
// If the file does not exist, it returns ErrMissingDotEnv.
// A nil cfg is treated as an all-zero Config.
func (s *Store) ParseRequiredDotEnv(path string, cfg *Config) error {
	return s.parseDotEnv(path, cfg, false)
}

func (s *Store) parseDotEnv(path string, cfg *Config, optional bool) error {
	err := parseDotEnv(path, s, cfg)
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

// ParseString parses dotenv contents from a string into the store using cfg.
// A nil cfg is treated as an all-zero Config.
func (s *Store) ParseString(str, name string, cfg *Config) error {
	return parseString(str, name, s, cfg)
}

// ParseReader parses dotenv contents from an io.Reader into the store using cfg.
// A nil cfg is treated as an all-zero Config.
func (s *Store) ParseReader(r io.Reader, name string, cfg *Config) error {
	return parseReader(r, name, s, cfg)
}

// ImportFromEnv reads the current process environment variables and stores them in the Store.
func (s *Store) ImportFromEnv(allowkeys, denykeys map[string]struct{}, overwrite bool) {
	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)

		if len(parts) != 2 {
			continue
		}

		if allowkeys != nil {
			if _, ok := allowkeys[parts[0]]; !ok {
				continue
			}
		}

		if denykeys != nil {
			if _, ok := denykeys[parts[0]]; ok {
				continue
			}
		}

		s.Set(parts[0], parts[1], overwrite)
	}
}

// ExportToEnv loads the key-value pairs from the Store into
// the process environment variables, optionally overwriting existing values.
func (s *Store) ExportToEnv(allowkeys, denykeys map[string]struct{}, overwrite bool) {
	for key, value := range s.data {
		if allowkeys != nil {
			if _, ok := allowkeys[key]; !ok {
				continue
			}
		}

		if denykeys != nil {
			if _, ok := denykeys[key]; ok {
				continue
			}
		}

		if _, exists := os.LookupEnv(key); !exists || overwrite {
			os.Setenv(key, value)
		}
	}
}

// Filter removes any keys that are not in the allowkeys, or that are in the denykeys.
// A nil allowkeys keeps all keys. A nil denykeys removes no keys.
func (s *Store) Filter(allowkeys, denykeys map[string]struct{}) {
	for key := range s.data {
		if allowkeys != nil {
			if _, ok := allowkeys[key]; !ok {
				delete(s.data, key)
				continue
			}
		}

		if denykeys != nil {
			if _, ok := denykeys[key]; ok {
				delete(s.data, key)
			}
		}
	}
}

// TransformValue applies the double-quoted value transform pipeline to an
// existing store value in place. The stored value is treated as the raw bytes
// that would have appeared between double quotes in a dotenv file. If the key
// is missing, ErrMissingRequiredKey is returned. A nil cfg is treated as an
// all-zero Config. Overwrite is ignored.
func (s *Store) TransformValue(key string, cfg *Config) error {
	value, err := s.GetRequired(key)
	if err != nil {
		return err
	}

	processed, err := processValue([]byte(value), resolveOptions(key, cfg))
	if err != nil {
		return fmt.Errorf("%s: %w", key, err)
	}

	s.data[key] = processed
	return nil
}

// TransformValues applies the double-quoted value transform pipeline to every
// value in the store. The Config is resolved the same way the parser uses
// it: base settings apply to every key unless that key has explicit overrides.
// A nil cfg is treated as an all-zero Config. Overwrite is ignored. If any
// key fails to process, the store is left unchanged.
func (s *Store) TransformValues(cfg *Config) error {
	keys := make([]string, len(s.data))
	values := make([]string, len(s.data))

	i := 0
	for key, raw := range s.data {
		value, err := processValue([]byte(raw), resolveOptions(key, cfg))
		if err != nil {
			return fmt.Errorf("%s: %w", key, err)
		}
		keys[i] = key
		values[i] = value
		i++
	}

	for i, key := range keys {
		s.data[key] = values[i]
	}
	return nil
}
