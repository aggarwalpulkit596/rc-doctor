// Package config reads .rc-doctor.json, a project's rc-doctor settings.
//
//	{
//	  "$schema": "https://raw.githubusercontent.com/aggarwalpulkit596/rc-doctor/main/schema/config.schema.json",
//	  "rules": { "restore-purchases": "off", "app-id": "error" },
//	  "ignore": ["examples/**", "legacy/"]
//	}
//
// The file sits at the root rc-doctor checks, next to rc's own .revenuecat.json.
// Unknown fields and unknown rule IDs are errors, so a typo never silently
// turns a check off.
package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/aggarwalpulkit596/rc-doctor/internal/model"
)

// FileName is the config file rc-doctor looks for at the root it checks.
const FileName = ".rc-doctor.json"

// Off turns a rule off.
const Off = "off"

// Config is a parsed .rc-doctor.json.
type Config struct {
	Schema string `json:"$schema,omitempty"`
	// Rules maps a rule ID to "off" or a severity that replaces the rule's own.
	Rules map[string]string `json:"rules,omitempty"`
	// Ignore lists globs of root-relative paths whose findings are dropped.
	// They follow .gitignore: "*" matches within a path segment, "**" across
	// segments, a pattern ending in "/" matches everything under that
	// directory, and a pattern is anchored to the root only when it has a "/"
	// at the start or in the middle.
	Ignore []string `json:"ignore,omitempty"`

	// Path is where the config was read from ("" for the default config).
	Path string `json:"-"`

	ignore []*regexp.Regexp
}

// Load reads the config at path, or, when path is "", root/.rc-doctor.json if
// it exists. No file means the default config.
func Load(root, path string) (*Config, error) {
	explicit := path != ""
	if !explicit {
		path = filepath.Join(root, FileName)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		if !explicit && errors.Is(err, fs.ErrNotExist) {
			return &Config{}, nil
		}
		return nil, fmt.Errorf("read config: %w", err)
	}
	var c Config
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&c); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	c.Path = path
	for _, p := range c.Ignore {
		re, err := compileGlob(p)
		if err != nil {
			return nil, fmt.Errorf("%s: bad ignore pattern %q: %w", path, p, err)
		}
		c.ignore = append(c.ignore, re)
	}
	return &c, nil
}

// Validate checks every configured rule ID against known and every value
// against the allowed settings.
func (c *Config) Validate(known []string) error {
	ids := map[string]bool{}
	for _, k := range known {
		ids[k] = true
	}
	for id, v := range c.Rules {
		if !ids[id] {
			return fmt.Errorf("%s: unknown rule %q (see `rc-doctor list rules`)", c.where(), id)
		}
		switch v {
		case Off, string(model.Error), string(model.Warning), string(model.Info):
		default:
			return fmt.Errorf("%s: rule %q: %q is not off, error, warning or info", c.where(), id, v)
		}
	}
	return nil
}

// Enabled reports whether a rule should run.
func (c *Config) Enabled(rule string) bool { return c.Rules[rule] != Off }

// Severity returns the configured severity for a rule, if one is set.
func (c *Config) Severity(rule string) (model.Severity, bool) {
	v, ok := c.Rules[rule]
	if !ok || v == Off {
		return "", false
	}
	return model.Severity(v), true
}

// Ignored reports whether a root-relative, slash-separated path matches an
// ignore pattern.
func (c *Config) Ignored(file string) bool {
	for _, re := range c.ignore {
		if re.MatchString(file) {
			return true
		}
	}
	return false
}

func (c *Config) where() string {
	if c.Path == "" {
		return FileName
	}
	return c.Path
}

// compileGlob turns an ignore pattern into a regexp over slash paths.
func compileGlob(p string) (*regexp.Regexp, error) {
	p = strings.TrimPrefix(filepath.ToSlash(strings.TrimSpace(p)), "./")
	if p == "" {
		return nil, errors.New("empty pattern")
	}
	// Like .gitignore: a slash at the start or in the middle anchors the
	// pattern to the root; otherwise it matches at any depth.
	anchored := strings.Contains(strings.TrimSuffix(p, "/"), "/")
	if strings.HasSuffix(p, "/") {
		p += "**"
	}
	p = strings.TrimPrefix(p, "/")
	var b strings.Builder
	b.WriteString("^")
	if !anchored {
		b.WriteString("(?:.*/)?")
	}
	for i := 0; i < len(p); i++ {
		switch {
		case strings.HasPrefix(p[i:], "**/"):
			b.WriteString("(?:.*/)?")
			i += 2
		case strings.HasPrefix(p[i:], "**"):
			b.WriteString(".*")
			i++
		case p[i] == '*':
			b.WriteString("[^/]*")
		case p[i] == '?':
			b.WriteString("[^/]")
		default:
			b.WriteString(regexp.QuoteMeta(p[i : i+1]))
		}
	}
	b.WriteString("$")
	return regexp.Compile(b.String())
}
