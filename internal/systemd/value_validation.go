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
	message := checkDirectiveValue(d, value)
	// Unit files have no trailing comments. Say so when that is the mistake:
	// the value is only valid without the comment, or it is a name or path,
	// where a word starting with # is not something anyone means.
	head, comment, ok := cutTrailingComment(value)
	switch {
	case !ok:
	case message != "" && checkDirectiveValue(d, head) == "",
		message == "" && commentFreeParsers[d.Parser] && !strings.ContainsAny(value, `'"`):
		return fmt.Sprintf("systemd does not support trailing comments; %q is read as part of the value of %s", comment, d.Name)
	}
	return message
}

// Parsers of user and group names, unit names and file system paths. Their
// values are not otherwise validated here.
var commentFreeParsers = map[string]bool{
	"config_parse_user_group_compat":      true,
	"config_parse_user_group_strv_compat": true,
	"config_parse_unit_slice":             true,
	"config_parse_working_directory":      true,
	"config_parse_pid_file":               true,
	"config_parse_unit_env_file":          true,
	"config_parse_unit_path_printf":       true,
	"config_parse_unit_path_strv_printf":  true,
	"config_parse_namespace_path_strv":    true,
	"config_parse_bind_paths":             true,
	"config_parse_exec_directories":       true,
	"config_parse_path_spec":              true,
	"config_parse_unit_condition_path":    true,
}

// cutTrailingComment splits a value at a # or ; that follows whitespace.
func cutTrailingComment(value string) (head, comment string, ok bool) {
	for i := 1; i < len(value); i++ {
		if (value[i] == '#' || value[i] == ';') && (value[i-1] == ' ' || value[i-1] == '\t') {
			head = strings.TrimRight(value[:i], " \t")
			return head, value[i:], head != ""
		}
	}
	return "", "", false
}

// Settings that take exactly one of their listed values, compared the way
// systemd's string tables are: case-sensitively.
var exactValueDirectives = map[string]bool{
	"Type": true, "ExitType": true, "Restart": true, "NotifyAccess": true, "OOMPolicy": true,
	"CollectMode": true, "OnFailureJobMode": true, "OnSuccessJobMode": true,
}

func checkDirectiveValue(d Directive, value string) string {
	if strings.HasPrefix(d.Parser, "config_parse_unit_condition_") {
		// Conditions take | (trigger) and then ! (negate) before the value.
		if rest, ok := strings.CutPrefix(value, "|"); ok {
			value = strings.TrimLeft(rest, whitespace)
		}
		if rest, ok := strings.CutPrefix(value, "!"); ok {
			value = strings.TrimLeft(rest, whitespace)
		}
	}
	unexpected := func() string {
		message := fmt.Sprintf("unexpected value %q for %s", value, d.Name)
		for _, known := range d.Values {
			if known != value && strings.EqualFold(known, value) {
				return message + fmt.Sprintf("; values are case-sensitive, use %q", known)
			}
		}
		return message
	}
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
	case "io-uring-filter":
		// Only a ~ at the very start inverts the list. Operations systemd
		// does not know are skipped with a warning in the journal.
		words, complete := splitWords(strings.TrimPrefix(value, "~"), " \t,")
		if !complete {
			return ""
		}
		for _, word := range words {
			if contains(d.Values, word) {
				continue
			}
			message := fmt.Sprintf("unknown io_uring operation %q for %s; systemd ignores it", word, d.Name)
			if strings.HasPrefix(word, "~") {
				return message + "; ~ is only recognized at the start of the value"
			}
			for _, known := range d.Values {
				if strings.EqualFold(known, word) {
					return message + fmt.Sprintf("; names are case-sensitive, use %q", known)
				}
			}
			return message
		}
	case "concurrency":
		if value != "infinity" {
			if _, err := parseSystemdUnsigned(value); err != nil {
				return fmt.Sprintf("%s expects an unsigned 32-bit integer or infinity", d.Name)
			}
		}
	default:
		switch {
		case len(d.Values) == 0:
			return checkTypedValue(d, value)
		case isBoolDirective(d):
			if !containsFold(boolValues, value) && !isShortBoolean(value) {
				return unexpected()
			}
		case d.Parser == "config_parse_log_level":
			// log_level_from_string() also takes the numeric levels.
			if level, err := strconv.ParseUint(value, 10, 8); !contains(d.Values, value) && (err != nil || level > 7) {
				return unexpected()
			}
		case exactValueDirectives[d.Name]:
			if !contains(d.Values, value) {
				return unexpected()
			}
		case containsFold(boolValues, value) && !containsFold(d.Values, value):
			// The value lists of other settings may be incomplete, but a
			// boolean they do not list is certainly not accepted.
			return unexpected()
		}
	}
	return ""
}

// checkTypedValue validates settings by the systemd function that parses
// them. Value kinds are only a heuristic, so they are not used here.
func checkTypedValue(d Directive, value string) string {
	switch d.Parser {
	case "config_parse_sec", "config_parse_sec_fix_0", "config_parse_sec_def_infinity",
		"config_parse_service_timeout", "config_parse_service_timeout_abort",
		"config_parse_job_timeout_sec", "config_parse_job_running_timeout_sec":
		if !validTimeSpan(value) {
			return fmt.Sprintf("%s expects a time span such as 30, 5min, 1h 30min or infinity", d.Name)
		}
	case "config_parse_memory_limit":
		fraction, err := parseFraction(value)
		if value != "infinity" && !(err == nil && fraction >= 0 && fraction <= 10000) && !validSize(value) {
			return fmt.Sprintf("%s expects a size in bytes with an optional K, M, G or T suffix, a percentage, or infinity", d.Name)
		}
	case "config_parse_iec_size":
		if !validSize(value) {
			return fmt.Sprintf("%s expects a size in bytes with an optional K, M, G or T suffix", d.Name)
		}
	case "config_parse_mode":
		if mode, err := strconv.ParseUint(value, 8, 32); err != nil || mode > 0o7777 || strings.ContainsAny(value, "+_") {
			return fmt.Sprintf("%s expects an octal access mode such as 0755", d.Name)
		}
	case "config_parse_unsigned":
		if _, err := parseSystemdUnsigned(value); err != nil {
			return fmt.Sprintf("%s expects an unsigned integer", d.Name)
		}
	case "config_parse_exec_nice":
		if n, err := parseSystemdInt(value); err != nil || n < -20 || n > 19 {
			return fmt.Sprintf("%s expects an integer between -20 and 19", d.Name)
		}
	case "config_parse_exec_oom_score_adjust":
		if n, err := parseSystemdInt(value); err != nil || n < -1000 || n > 1000 {
			return fmt.Sprintf("%s expects an integer between -1000 and 1000", d.Name)
		}
	case "config_parse_rlimit":
		return checkResourceLimit(d.Name, value)
	case "config_parse_unit_deps":
		return checkUnitNames(d.Name, strings.Fields(value), "")
	case "config_parse_trigger_unit":
		return checkUnitNames(d.Name, []string{value}, "")
	case "config_parse_socket_service":
		return checkUnitNames(d.Name, []string{value}, "service")
	case "config_parse_service_sockets":
		return checkUnitNames(d.Name, strings.Fields(value), "socket")
	}
	switch d.Name {
	case "WantedBy", "RequiredBy", "UpheldBy", "Also":
		// Read by systemctl enable, which fails on a name that is not valid.
		if words, complete := valueWords(value); complete {
			return checkUnitNames(d.Name, words, "")
		}
	}
	return ""
}

// checkExecDirectories follows config_parse_exec_directories(), which skips
// entries at or below private/, the location of DynamicUser= directories.
// The symlink of an entry is only checked from v263 on.
func checkExecDirectories(d Directive, value string, version int) string {
	if d.Parser != "config_parse_exec_directories" || strings.Contains(value, "\\") {
		return ""
	}
	words, complete := valueWords(value)
	if !complete {
		return ""
	}
	for _, word := range words {
		path, symlink, _ := strings.Cut(word, ":")
		symlink, _, _ = strings.Cut(symlink, ":")
		what := ""
		switch {
		case belowPrivate(path):
			what = "path " + strconv.Quote(path)
		case d.Name != "ConfigurationDirectory" && (version == 0 || version >= 263) && belowPrivate(symlink):
			what = "symlink " + strconv.Quote(symlink)
		default:
			continue
		}
		return fmt.Sprintf("%s %s is at or below private/, which is reserved for DynamicUser=; systemd ignores %q", d.Name, what, word)
	}
	return ""
}

// belowPrivate reports whether the first component of a relative path is
// "private". Paths with specifiers are only known once the unit is loaded.
func belowPrivate(path string) bool {
	if strings.HasPrefix(path, "/") || strings.Contains(path, "%") {
		return false
	}
	for _, component := range strings.Split(path, "/") {
		if component != "" && component != "." {
			return component == "private"
		}
	}
	return false
}

var unitNameTypes = []string{"service", "socket", "target", "device", "mount", "automount", "swap", "timer", "path", "slice", "scope"}

// checkUnitNames follows unit_name_is_valid(). Unit files take names as they
// are: unlike systemctl, systemd does not append .service to them. Names
// with specifiers are only known once the unit is loaded.
func checkUnitNames(setting string, names []string, unitType string) string {
	for _, name := range names {
		if strings.Contains(name, "%") {
			continue
		}
		problem := ""
		dot := strings.LastIndex(name, ".")
		for _, r := range name {
			if !(r >= '0' && r <= '9' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || strings.ContainsRune(":-_.\\@", r)) {
				problem = fmt.Sprintf("%q is not allowed, and names are separated by spaces", string(r))
				break
			}
		}
		switch {
		case problem != "":
		case dot < 0:
			problem = "it needs a type suffix such as .service or .target"
		case !contains(unitNameTypes, name[dot+1:]):
			problem = fmt.Sprintf("%q is not a unit type", name[dot:])
		case dot == 0 || name[0] == '@' || len(name) >= 256:
			problem = "it has no proper name before the type suffix"
		case unitType != "" && name[dot+1:] != unitType:
			return fmt.Sprintf("%s expects a .%s unit, not %q", setting, unitType, name)
		default:
			continue
		}
		return fmt.Sprintf("%q is not a valid unit name for %s: %s", name, setting, problem)
	}
	return ""
}

// checkResourceLimit follows rlimit_parse(): one limit, or soft:hard. Limits
// measured in time or as a nice level are left to systemd.
func checkResourceLimit(name, value string) string {
	valid, expects := validSize, "a size in bytes with an optional K, M, G or T suffix"
	switch name {
	case "LimitNOFILE", "LimitNPROC", "LimitLOCKS", "LimitSIGPENDING", "LimitRTPRIO":
		valid, expects = func(part string) bool {
			if strings.Contains(part, "_") {
				return false
			}
			_, err := strconv.ParseUint(strings.TrimPrefix(part, "+"), 0, 64)
			return err == nil
		}, "an unsigned integer"
	case "LimitFSIZE", "LimitDATA", "LimitSTACK", "LimitCORE", "LimitRSS", "LimitAS", "LimitMEMLOCK", "LimitMSGQUEUE":
	default:
		return ""
	}
	parts := strings.Split(value, ":")
	for _, part := range parts {
		if part == "" {
			return ""
		}
		if len(parts) > 2 || part != "infinity" && !valid(part) {
			return fmt.Sprintf("%s expects %s or infinity, optionally as soft:hard", name, expects)
		}
	}
	return ""
}

const whitespace = " \t\n\r"

func digitCount(text string) int {
	n := 0
	for n < len(text) && text[n] >= '0' && text[n] <= '9' {
		n++
	}
	return n
}

// The order matters: parse_time() takes the first unit that is a prefix.
var timeUnits = []string{
	"seconds", "second", "sec", "s", "minutes", "minute", "min", "months", "month", "M",
	"msec", "ms", "m", "hours", "hour", "hr", "h", "days", "day", "d", "weeks", "week", "w",
	"years", "year", "y", "usec", "us", "μs", "µs",
}

// validTimeSpan follows parse_time(): numbers, each with an optional unit.
func validTimeSpan(value string) bool {
	p := strings.TrimLeft(value, whitespace)
	if rest, ok := strings.CutPrefix(p, "infinity"); ok {
		return strings.Trim(rest, whitespace) == ""
	}
	for something := false; ; something = true {
		p = strings.TrimLeft(p, whitespace)
		if p == "" {
			return something
		}
		p = strings.TrimPrefix(p, "+")
		digits := digitCount(p)
		p = p[digits:]
		if strings.HasPrefix(p, ".") {
			// "3." and "3.sec" are rejected, ".5" is not.
			decimals := digitCount(p[1:])
			if decimals == 0 {
				return false
			}
			p = p[1+decimals:]
		} else if digits == 0 {
			return false
		}
		unit := strings.TrimLeft(p, whitespace)
		matched := false
		for _, name := range timeUnits {
			if rest, ok := strings.CutPrefix(unit, name); ok {
				p, matched = rest, true
				break
			}
		}
		if !matched {
			// "12.34.56" is rejected, "12.34 .56" is not.
			if unit == p && unit != "" {
				return false
			}
			p = unit
		}
	}
}

// validSize follows parse_size() for base 1024: numbers with an optional
// suffix, where suffixes must get smaller ("1G 500M", not "500M 1G").
func validSize(value string) bool {
	const suffixes = "EPTGMKB"
	p := value
	for next := 0; ; {
		p = strings.TrimPrefix(strings.TrimLeft(p, whitespace), "+")
		digits := digitCount(p)
		if digits == 0 || next > len(suffixes) {
			return false
		}
		p = p[digits:]
		if strings.HasPrefix(p, ".") {
			p = p[1:]
			p = p[digitCount(p):]
		}
		p = strings.TrimLeft(p, whitespace)
		// Without a suffix that is still allowed, the number counts as bytes.
		suffix := len(suffixes)
		if p != "" {
			if i := strings.IndexByte(suffixes, p[0]); i >= next {
				suffix, p = i, p[1:]
			}
		}
		next = suffix + 1
		if p == "" {
			return true
		}
	}
}

// Match safe_atoi(): like parseSystemdUnsigned, with a sign.
func parseSystemdInt(value string) (int64, error) {
	if strings.Contains(value, "_") {
		return 0, strconv.ErrSyntax
	}
	return strconv.ParseInt(value, 0, 32)
}

func validEnumValue(values []string, value string) bool {
	if contains(values, value) {
		return true
	}
	if contains(values, "true") && contains(values, "false") && isShortBoolean(value) {
		return true
	}
	// Boolean spellings are case-insensitive; symbolic enum values are not.
	return containsFold(boolValues, value) && containsFold(values, value)
}

// parse_boolean() also accepts these single letters, in any case.
func isShortBoolean(value string) bool {
	switch strings.ToLower(value) {
	case "y", "n", "t", "f":
		return true
	}
	return false
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
	return splitWords(value, " \t")
}

// splitWords is valueWords for settings with further separators.
func splitWords(value, separators string) ([]string, bool) {
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
			switch {
			case r == '\'' || r == '"':
				quote = r
				started = true
			case strings.ContainsRune(separators, r):
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
