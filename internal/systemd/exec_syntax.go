package systemd

import (
	"fmt"
	"regexp"
	"strings"
)

// systemd starts the commands of Exec settings itself, not through a shell.
// Shell operators therefore reach the program as ordinary arguments. Such a
// unit loads and passes systemd-analyze verify, so nothing else points it out.

// A redirection as its own word: >, >>file, 2>&1, &>file, <file. Comparison
// operators such as >= are not one.
var shellRedirection = regexp.MustCompile(`^(?:[0-9]*>>?&?|&>>?|[0-9]*<<?<?)(?:[^<>|&;=][^<>|&;]*)?$`)

// shellSyntaxMessage reports the first word of an Exec setting that only a
// shell would interpret. Quoted or escaped words are meant literally.
func shellSyntaxMessage(d Directive, value string) string {
	if d.Parser != "config_parse_exec" {
		return ""
	}
	start, viaShell := true, false
	for _, word := range execWords(value) {
		switch {
		case word == ";":
			start = true
		case start:
			// The executable. A | among its prefixes hands the command line
			// to the shell of the user the service runs as.
			start = false
			prefixes := word[:len(word)-len(strings.TrimLeft(word, "-@:+!|"))]
			viaShell = strings.Contains(prefixes, "|")
		case viaShell || strings.ContainsAny(word, `'"\`):
		case word == "#":
			return fmt.Sprintf("systemd does not support trailing comments; \"#\" and the words after it are passed to the %s= command as arguments", d.Name)
		case word == "|" || word == "||" || word == "&&" || word == "&":
			return fmt.Sprintf("systemd does not run %s= through a shell, so %q is passed to the command as an argument; run the command with /bin/sh -c '...', or quote the word if that is intended", d.Name, word)
		case shellRedirection.MatchString(word):
			return fmt.Sprintf("systemd does not run %s= through a shell, so %q is passed to the command as an argument; use StandardOutput=, StandardError= or StandardInput=, or /bin/sh -c '...', or quote the word if that is intended", d.Name, word)
		case strings.Contains(word, "$(") || strings.Contains(word, "`"):
			return fmt.Sprintf("systemd does not run %s= through a shell, so the command substitution in %q is not expanded; run the command with /bin/sh -c '...', or quote the word if that is intended", d.Name, word)
		}
	}
	return ""
}
