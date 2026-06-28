// Package scrubber detects and strips sensitive information
// (API keys, tokens, PII) from LLM payloads before they leave the machine.
package scrubber

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// RedactionStyle controls how matched secrets are replaced.
type RedactionStyle int

const (
	// RedactFull replaces the entire secret with [REDACTED].
	RedactFull RedactionStyle = iota
	// RedactPartial shows first 4 and last 4 characters.
	RedactPartial
	// RedactMask shows the type but hides the value.
	RedactMask
)

// Rule defines a single scrubbing rule.
type Rule struct {
	Name    string
	Pattern *regexp.Regexp
	Style   RedactionStyle
}

// Scrubber holds a set of rules and applies them to payloads.
type Scrubber struct {
	rules []Rule
}

// New creates a Scrubber from a list of regex patterns.
func New(patternStrs []string) (*Scrubber, error) {
	var rules []Rule
	for i, p := range patternStrs {
		re, err := regexp.Compile(p)
		if err != nil {
			return nil, fmt.Errorf("rule %d: %w", i, err)
		}
		name := fmt.Sprintf("pattern-%d", i)
		rules = append(rules, Rule{Name: name, Pattern: re, Style: RedactMask})
	}
	return &Scrubber{rules: rules}, nil
}

// AddRule adds a custom rule.
func (s *Scrubber) AddRule(name, pattern string, style RedactionStyle) error {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return fmt.Errorf("rule %s: %w", name, err)
	}
	s.rules = append(s.rules, Rule{Name: name, Pattern: re, Style: style})
	return nil
}

// ScrubResult contains the cleaned text and a report of what was found.
type ScrubResult struct {
	Cleaned   string
	Findings  []Finding
}

// Finding describes a single scrubbing event.
type Finding struct {
	RuleName string `json:"rule_name"`
	Count    int    `json:"count"`
	Sample   string `json:"sample,omitempty"`
}

// ScrubText applies all rules to a text payload.
func (s *Scrubber) ScrubText(input string) *ScrubResult {
	result := &ScrubResult{
		Cleaned:  input,
		Findings: []Finding{},
	}

	for _, rule := range s.rules {
		matches := rule.Pattern.FindAllString(result.Cleaned, -1)
		if len(matches) == 0 {
			continue
		}

		sample := matches[0]
		if len(sample) > 20 {
			sample = sample[:4] + "..." + sample[len(sample)-4:]
		}

		result.Findings = append(result.Findings, Finding{
			RuleName: rule.Name,
			Count:    len(matches),
			Sample:   sample,
		})

		replacement := s.redact(rule.Style, matches[0])
		result.Cleaned = rule.Pattern.ReplaceAllString(result.Cleaned, replacement)
	}

	return result
}

// ScrubJSON applies rules to all string values in a JSON payload.
func (s *Scrubber) ScrubJSON(input []byte) (*ScrubResult, error) {
	var data interface{}
	if err := json.Unmarshal(input, &data); err != nil {
		// Not valid JSON — fall back to text scrubbing
		return s.ScrubText(string(input)), nil
	}

	s.scrubValue(&data)

	cleaned, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("marshal scrubbed: %w", err)
	}

	// Still run text-level scrub for any remaining patterns
	return s.ScrubText(string(cleaned))
}

func (s *Scrubber) scrubValue(val *interface{}) {
	switch v := (*val).(type) {
	case string:
		r := s.ScrubText(v)
		if r.Cleaned != v {
			*val = r.Cleaned
		}
	case map[string]interface{}:
		for k, child := range v {
			s.scrubValue(&child)
			v[k] = child
		}
	case []interface{}:
		for i, child := range v {
			s.scrubValue(&child)
			v[i] = child
		}
	}
}

func (s *Scrubber) redact(style RedactionStyle, secret string) string {
	switch style {
	case RedactFull:
		return "[REDACTED]"
	case RedactPartial:
		if len(secret) <= 8 {
			return "[REDACTED]"
		}
		return secret[:4] + "..." + secret[len(secret)-4:]
	case RedactMask:
		if strings.HasPrefix(secret, "sk-") {
			return "sk-[REDACTED]"
		}
		if strings.HasPrefix(secret, "ghp_") {
			return "ghp_[REDACTED]"
		}
		return fmt.Sprintf("[REDACTED:%s]", secret[:min(4, len(secret))])
	default:
		return "[REDACTED]"
	}
}

// ScrubReport produces a human-readable summary of findings.
func ScrubReport(findings []Finding) string {
	if len(findings) == 0 {
		return "No secrets or sensitive data detected."
	}
	var b strings.Builder
	b.WriteString(fmt.Sprintf("🔒 Scrubbed %d secret(s):\n", len(findings)))
	for _, f := range findings {
		b.WriteString(fmt.Sprintf("  • Rule %s: %d occurrence(s)", f.RuleName, f.Count))
		if f.Sample != "" {
			b.WriteString(fmt.Sprintf(" (e.g. %s)", f.Sample))
		}
		b.WriteString("\n")
	}
	return b.String()
}