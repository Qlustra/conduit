package formats

import (
	"fmt"
	"github.com/joho/godotenv"
	"github.com/qlustra/conduit/layout"
	"sort"
	"strings"
)

// EnvPrecedence controls which source wins when both process and file provide
// a value for the same key.
type EnvPrecedence uint8

const (
	// EnvProcessWins prefers process values over file values.
	EnvProcessWins EnvPrecedence = iota + 1

	// EnvFileWins prefers file values over process values.
	EnvFileWins
)

// EnvScope controls which keys outside the defaults baseline may enter a
// resolved environment.
type EnvScope uint8

const (
	// EnvBaselineOnly restricts the resolved environment to keys declared in the
	// defaults baseline.
	EnvBaselineOnly EnvScope = 0

	// EnvKeepProcess allows process-only keys outside the defaults baseline.
	EnvKeepProcess EnvScope = 1 << 0

	// EnvKeepFile allows file-only keys outside the defaults baseline.
	EnvKeepFile EnvScope = 1 << 1

	// EnvKeepAll allows extra keys from both process and file sources.
	EnvKeepAll = EnvKeepProcess | EnvKeepFile
)

// EnvResolveOptions controls how defaults, process values, and file values are
// combined into one resolved environment.
type EnvResolveOptions struct {
	Precedence EnvPrecedence
	Scope      EnvScope
}

// EnvCodec marshals and unmarshals managed .env content as key/value strings.
type EnvCodec struct{}

func (c EnvCodec) Marshal(v map[string]string) ([]byte, error) {
	content, err := godotenv.Marshal(v)
	if err != nil {
		return nil, err
	}
	if content == "" {
		return []byte(""), nil
	}
	return []byte(content + "\n"), nil
}

func (c EnvCodec) Unmarshal(data []byte) (map[string]string, error) {
	return godotenv.UnmarshalBytes(data)
}

// EnvFile is a Format that stores managed .env content as map[string]string.
//
// Load accepts godotenv-compatible input syntax. Save and Sync normalize the
// file back to managed key/value content and do not preserve comments,
// duplicate entries, original quoting, or separator style.
type EnvFile struct {
	layout.Format[map[string]string, EnvCodec]
}

// ResolveWithProcess resolves cached file values together with defaults and a
// supplied process environment snapshot.
//
// It returns ok == false when the env file has no cached content loaded.
func (f EnvFile) ResolveWithProcess(defaults, processEnv map[string]string, opts EnvResolveOptions) (map[string]string, bool) {
	fileEnv, ok := f.Get()
	if !ok {
		return nil, false
	}
	return ResolveEnv(defaults, processEnv, fileEnv, opts), true
}

// EnvMap parses KEY=VALUE entries into a map.
//
// Later entries with the same key replace earlier ones. Values may contain '='
// after the first separator. It returns an error for malformed entries or empty
// keys.
func EnvMap(entries []string) (map[string]string, error) {
	values := make(map[string]string, len(entries))
	for _, entry := range entries {
		key, value, ok := strings.Cut(entry, "=")
		if !ok {
			return nil, fmt.Errorf("invalid env entry %q: missing '='", entry)
		}
		if key == "" {
			return nil, fmt.Errorf("invalid env entry %q: empty key", entry)
		}
		values[key] = value
	}
	return values, nil
}

// ResolveEnv combines defaults, process values, and file values into one
// resolved environment.
//
// Defaults define the baseline key set and fallback values. Scope decides
// which extra keys outside that baseline may be admitted. Precedence decides
// whether process or file values win when both are present.
func ResolveEnv(defaults, processEnv, fileEnv map[string]string, opts EnvResolveOptions) map[string]string {
	keys := make(map[string]struct{}, len(defaults))
	for key := range defaults {
		keys[key] = struct{}{}
	}
	if opts.Scope&EnvKeepProcess != 0 {
		for key := range processEnv {
			keys[key] = struct{}{}
		}
	}
	if opts.Scope&EnvKeepFile != 0 {
		for key := range fileEnv {
			keys[key] = struct{}{}
		}
	}

	resolved := make(map[string]string, len(keys))
	for key := range keys {
		if value, ok := resolveEnvValue(key, defaults, processEnv, fileEnv, opts.normalizedPrecedence()); ok {
			resolved[key] = value
		}
	}
	return resolved
}

func resolveEnvValue(key string, defaults, processEnv, fileEnv map[string]string, precedence EnvPrecedence) (string, bool) {
	defaultValue, hasDefault := defaults[key]
	processValue, hasProcess := processEnv[key]
	fileValue, hasFile := fileEnv[key]

	switch precedence {
	case EnvFileWins:
		if hasFile {
			return fileValue, true
		}
		if hasProcess {
			return processValue, true
		}
	default:
		if hasProcess {
			return processValue, true
		}
		if hasFile {
			return fileValue, true
		}
	}

	if hasDefault {
		return defaultValue, true
	}
	return "", false
}

func (opts EnvResolveOptions) normalizedPrecedence() EnvPrecedence {
	if opts.Precedence == EnvFileWins {
		return EnvFileWins
	}
	return EnvProcessWins
}

// EnvList renders env keys into deterministic KEY=VALUE entries sorted by key.
func EnvList(env map[string]string) []string {
	keys := make([]string, 0, len(env))
	for key := range env {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	entries := make([]string, 0, len(keys))
	for _, key := range keys {
		entries = append(entries, key+"="+env[key])
	}
	return entries
}
