package scrubber

import (
	"math"
	"regexp"
	"strings"
)

// FindingType categorises what was detected.
type FindingType string

const (
	TypeAPIKey    FindingType = "api_key"
	TypeJWT       FindingType = "jwt"
	TypeAWSCred   FindingType = "aws_credential"
	TypeSSHKey    FindingType = "ssh_key"
	TypeEmail     FindingType = "email"
	TypeSSN       FindingType = "ssn"
	TypePhone     FindingType = "phone"
	TypeHighEntropy FindingType = "high_entropy"
)

// Finding records a single detected secret or PII instance.
type Finding struct {
	Type    FindingType
	Match   string // redacted snippet for display
	Line    int
	Redacted string // what replaced it in the clean output
}

// Result is the output of a scrub operation.
type Result struct {
	Clean    string
	Findings []Finding
	HasSecrets bool
}

// EntropyThreshold is the default Shannon entropy threshold for high-randomness strings.
const EntropyThreshold = 4.5

// patterns maps each FindingType to its detection regex and a redaction label.
var patterns = []struct {
	typ     FindingType
	re      *regexp.Regexp
	replace string
}{
	{
		TypeSSHKey,
		regexp.MustCompile(`-----BEGIN (?:RSA |EC |OPENSSH )?PRIVATE KEY-----[\s\S]*?-----END (?:RSA |EC |OPENSSH )?PRIVATE KEY-----`),
		"[REDACTED:SSH_PRIVATE_KEY]",
	},
	{
		TypeJWT,
		regexp.MustCompile(`eyJ[A-Za-z0-9_-]{10,}\.eyJ[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}`),
		"[REDACTED:JWT]",
	},
	{
		TypeAWSCred,
		regexp.MustCompile(`(?i)(?:AKIA|ASIA|AROA|AIDA|ANPA|ANVA|AIPA)[A-Z0-9]{16}`),
		"[REDACTED:AWS_KEY]",
	},
	{
		TypeAPIKey,
		// Anthropic, OpenAI, generic "sk-" style keys
		regexp.MustCompile(`(?i)(?:sk-[A-Za-z0-9]{20,}|ant[A-Za-z0-9_-]{30,}|[Aa][Pp][Ii][_-]?[Kk][Ee][Yy]\s*[:=]\s*\S{16,})`),
		"[REDACTED:API_KEY]",
	},
	{
		TypeEmail,
		regexp.MustCompile(`\b[A-Za-z0-9._%+\-]+@[A-Za-z0-9.\-]+\.[A-Za-z]{2,}\b`),
		"[REDACTED:EMAIL]",
	},
	{
		TypeSSN,
		regexp.MustCompile(`\b\d{3}[-\s]?\d{2}[-\s]?\d{4}\b`),
		"[REDACTED:SSN]",
	},
	{
		TypePhone,
		regexp.MustCompile(`\b(?:\+?1[-.\s]?)?\(?\d{3}\)?[-.\s]?\d{3}[-.\s]?\d{4}\b`),
		"[REDACTED:PHONE]",
	},
}

// Scrub scans text for secrets and PII, returns a clean version and list of findings.
func Scrub(text string, entropyThreshold float64) Result {
	if entropyThreshold <= 0 {
		entropyThreshold = EntropyThreshold
	}

	clean := text
	var findings []Finding

	for _, p := range patterns {
		matches := p.re.FindAllStringIndex(clean, -1)
		offset := 0
		for _, loc := range matches {
			start, end := loc[0]-offset, loc[1]-offset
			matched := clean[start:end]
			lineNum := strings.Count(text[:loc[0]+offset], "\n") + 1

			// Truncate match for display
			display := matched
			if len(display) > 40 {
				display = display[:20] + "..." + display[len(display)-8:]
			}

			findings = append(findings, Finding{
				Type:     p.typ,
				Match:    display,
				Line:     lineNum,
				Redacted: p.replace,
			})
			clean = clean[:start] + p.replace + clean[end:]
			offset += (end - start) - len(p.replace)
		}
	}

	// Entropy scan on the already-cleaned text (so previously redacted tokens aren't double-counted)
	for i, line := range strings.Split(clean, "\n") {
		for _, word := range strings.Fields(line) {
			if strings.HasPrefix(word, "[REDACTED") {
				continue
			}
			if len(word) >= 20 && shannonEntropy(word) >= entropyThreshold {
				findings = append(findings, Finding{
					Type:  TypeHighEntropy,
					Match: word[:min(len(word), 12)] + "...",
					Line:  i + 1,
				})
			}
		}
	}

	return Result{
		Clean:    clean,
		Findings: findings,
		HasSecrets: len(findings) > 0,
	}
}

// shannonEntropy calculates the Shannon entropy of a string.
func shannonEntropy(s string) float64 {
	if len(s) == 0 {
		return 0
	}
	freq := make(map[rune]int)
	for _, c := range s {
		freq[c]++
	}
	entropy := 0.0
	n := float64(len([]rune(s)))
	for _, count := range freq {
		p := float64(count) / n
		entropy -= p * math.Log2(p)
	}
	return entropy
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
