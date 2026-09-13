package systemd

import (
	"fmt"
	"strconv"
	"strings"
)

func validateDirectiveValue(d Directive, value string) string {
	if value == "" || strings.Contains(value, "\\") {
		// Empty assignments reset settings. Escaped/continued values require
		// the full systemd syntax parser, so avoid guessing about their value.
		return ""
	}
	unexpected := func() string { return fmt.Sprintf("unexpected value %q for %s", value, d.Name) }
	switch d.ValueKind {
	case "enum", "enum-list", "resource-list":
		list := value
		inverted := d.ValueKind == "resource-list" && strings.HasPrefix(list, "~")
		if inverted {
			list = strings.TrimPrefix(list, "~")
		}
		words, complete := valueWords(list)
		if !complete {
			return ""
		}
		if d.ValueKind == "resource-list" && !inverted && len(words) == 1 && validEnumValue(boolValues, words[0]) {
			return ""
		}
		if d.ValueKind == "enum" && len(words) != 1 {
			return unexpected()
		}
		for _, word := range words {
			if d.ValueKind == "resource-list" {
				if word != "extensions" && word != "credentials" {
					return unexpected()
				}
			} else if !validEnumValue(d.Values, word) {
				return unexpected()
			}
		}
	case "fraction":
		// Condition parsing recognizes | first, then !. They are prefixes,
		// not part of the optional rollout tag.
		argument := strings.TrimPrefix(strings.TrimPrefix(value, "|"), "!")
		words := strings.Fields(argument)
		if len(words) < 1 || len(words) > 2 {
			return fmt.Sprintf("%s expects an optional tag followed by 0%%–100%%, with at most two decimal places", d.Name)
		}
		fraction, err := parseFraction(words[len(words)-1])
		if err != nil {
			return fmt.Sprintf("%s expects an optional tag followed by 0%%–100%%, with at most two decimal places (or an equivalent ‰/‱ value)", d.Name)
		}
		if fraction < 0 || fraction > 10000 {
			return fmt.Sprintf("%s percentage must be between 0%% and 100%%", d.Name)
		}
	case "concurrency":
		if value != "infinity" {
			if _, err := parseSystemdUnsigned(value); err != nil {
				return fmt.Sprintf("%s expects an unsigned 32-bit integer or infinity", d.Name)
			}
		}
	default:
		if len(d.Values) > 0 && shouldValidateValue(d, value) && !containsFold(d.Values, value) {
			return unexpected()
		}
	}
	return ""
}

func validEnumValue(values []string, value string) bool {
	if contains(values, value) {
		return true
	}
	if contains(values, "true") && contains(values, "false") {
		switch strings.ToLower(value) {
		case "y", "n", "t", "f":
			return true
		}
	}
	// Boolean spellings are case-insensitive; symbolic enum values are not.
	return containsFold(boolValues, value) && containsFold(values, value)
}

// Match safe_atou(): decimal, C-style octal, and 0x/0o/0b prefixes. Go's
// underscore syntax is not part of systemd's integer syntax.
func parseSystemdUnsigned(value string) (uint64, error) {
	if strings.Contains(value, "_") {
		return 0, strconv.ErrSyntax
	}
	negative := strings.HasPrefix(value, "-")
	digits := value
	if strings.HasPrefix(value, "+") || negative {
		digits = value[1:]
	}
	n, err := strconv.ParseUint(digits, 0, 32)
	if negative && n != 0 {
		return 0, strconv.ErrRange
	}
	return n, err
}

// ConditionFraction uses parse_permyriad(), which also accepts per-mille and
// per-ten-thousand notation. Preserve these valid forms rather than reporting
// them as invalid merely because the manual's examples use percentages.
func parseFraction(value string) (int64, error) {
	precision, multiplier := 0, int64(1)
	switch {
	case strings.HasSuffix(value, "%"):
		precision, multiplier = 2, 100
		value = strings.TrimSuffix(value, "%")
	case strings.HasSuffix(value, "‰"):
		precision, multiplier = 1, 10
		value = strings.TrimSuffix(value, "‰")
	case strings.HasSuffix(value, "‱"):
		value = strings.TrimSuffix(value, "‱")
	default:
		return 0, strconv.ErrSyntax
	}
	if strings.Contains(value, "_") {
		return 0, strconv.ErrSyntax
	}
	whole, fraction, dot := strings.Cut(value, ".")
	if dot && (len(fraction) == 0 || len(fraction) > precision) {
		return 0, strconv.ErrSyntax
	}
	n, err := strconv.ParseInt(whole, 0, 32)
	if err != nil {
		return 0, err
	}
	if n < 0 {
		return -1, nil
	}
	f := int64(0)
	for _, digit := range fraction {
		if digit < '0' || digit > '9' {
			return 0, strconv.ErrSyntax
		}
		f = f*10 + int64(digit-'0')
	}
	for i := len(fraction); i < precision; i++ {
		f *= 10
	}
	return n*multiplier + f, nil
}

// valueWords handles plain and quoted words. Escapes are intentionally left to
// systemd, and unfinished quotes are ignored while the user is still typing.
func valueWords(value string) ([]string, bool) {
	var words []string
	var word strings.Builder
	var quote rune
	started := false
	for _, r := range value {
		if r == '\\' {
			return nil, false
		}
		if quote != 0 {
			if r == quote {
				quote = 0
			} else {
				word.WriteRune(r)
			}
		} else {
			switch r {
			case '\'', '"':
				quote = r
				started = true
			case ' ', '\t':
				if started {
					words = append(words, word.String())
					word.Reset()
					started = false
				}
			default:
				word.WriteRune(r)
				started = true
			}
		}
	}
	if quote != 0 {
		return nil, false
	}
	if started {
		words = append(words, word.String())
	}
	return words, true
}
