package systemd

import "strings"

// These checks mirror service_verify(), socket_verify(), timer_verify(),
// path_verify() and mount_verify(): the conditions under which systemd refuses
// to load a unit. They need the whole unit, so drop-ins are not checked.

type unitSettings struct {
	headers  map[string]Entry
	sections map[string][]Entry
}

func collectSettings(doc Document) unitSettings {
	u := unitSettings{headers: map[string]Entry{}, sections: map[string][]Entry{}}
	section := ""
	for _, entry := range doc.Entries {
		switch entry.Kind {
		case EntrySection:
			section = entry.Section
			if _, ok := u.headers[section]; !ok {
				u.headers[section] = entry
			}
		case EntryDirective:
			u.sections[section] = append(u.sections[section], entry)
		}
	}
	return u
}

// assigned returns the assignments to the named settings that are in effect.
// They form one list, which an empty assignment to any of them clears.
func (u unitSettings) assigned(section string, names ...string) []Entry {
	var entries []Entry
	for _, entry := range u.sections[section] {
		if !contains(names, entry.Key) {
			continue
		}
		if entry.Value == "" {
			entries = nil
		} else {
			entries = append(entries, entry)
		}
	}
	return entries
}

// last returns the final assignment to a setting, which is the one systemd
// uses for settings that take a single value.
func (u unitSettings) last(section, name string) Entry {
	var last Entry
	for _, entry := range u.sections[section] {
		if entry.Key == name {
			last = entry
		}
	}
	return last
}

func isTrue(value string) bool {
	return containsFold([]string{"1", "yes", "y", "true", "t", "on"}, value)
}

func loadErrors(doc Document, unitType string) []Diagnostic {
	u := collectSettings(doc)
	var at Entry
	var message string
	switch unitType {
	case "service":
		at, message = serviceLoadError(u)
	case "socket":
		if len(u.assigned("Socket", "ListenStream", "ListenDatagram", "ListenSequentialPacket", "ListenFIFO", "ListenSpecial", "ListenNetlink", "ListenMessageQueue", "ListenUSBFunction")) == 0 {
			at, message = u.headers["Socket"], "Socket unit has no Listen setting (ListenStream=, ListenDatagram=, ListenFIFO=, ...)"
		}
	case "timer":
		timers := u.assigned("Timer", "OnCalendar", "OnActiveSec", "OnBootSec", "OnStartupSec", "OnUnitActiveSec", "OnUnitInactiveSec")
		if len(timers) == 0 && !isTrue(u.last("Timer", "OnClockChange").Value) && !isTrue(u.last("Timer", "OnTimezoneChange").Value) {
			at, message = u.headers["Timer"], "Timer unit has no trigger setting (OnCalendar=, OnBootSec=, OnUnitActiveSec=, ...)"
		}
	case "path":
		if len(u.assigned("Path", "PathExists", "PathExistsGlob", "PathChanged", "PathModified", "DirectoryNotEmpty")) == 0 {
			at, message = u.headers["Path"], "Path unit has no path setting (PathExists=, PathChanged=, PathModified=, ...)"
		}
	case "mount":
		if u.last("Mount", "What").Value == "" {
			at, message = u.headers["Mount"], "Mount unit has no What= setting"
		}
	}
	// Without its section the unit is most likely still being written.
	if message == "" || at.Kind == EntryUnknown {
		return nil
	}
	return []Diagnostic{diagnostic(at, SeverityError, message+"; systemd refuses to load the unit")}
}

var serviceTypes = []string{"simple", "exec", "forking", "oneshot", "dbus", "notify", "notify-reload", "idle"}

// serviceLoadError returns the first problem service_verify() would report,
// and the line it is reported on.
func serviceLoadError(u unitSettings) (Entry, string) {
	header := u.headers["Service"]
	commands := 0
	var extra Entry
	for _, entry := range u.assigned("Service", "ExecStart") {
		commands += commandCount(entry.Value)
		if commands > 1 && extra.Kind == EntryUnknown {
			extra = entry
		}
	}
	stops := 0
	for _, entry := range u.assigned("Service", "ExecStop") {
		stops += commandCount(entry.Value)
	}
	action := u.last("Unit", "SuccessAction").Value
	hasAction := action != "" && action != "none"

	// An invalid Type= is ignored, which leaves the previous one in place.
	var typeEntry Entry
	for _, entry := range u.sections["Service"] {
		if entry.Key == "Type" && contains(serviceTypes, entry.Value) {
			typeEntry = entry
		}
	}
	serviceType := typeEntry.Value
	if serviceType == "" {
		switch {
		case u.last("Service", "BusName").Value != "":
			serviceType = "dbus"
		case len(u.assigned("Service", "LoadCredential", "LoadCredentialEncrypted", "SetCredential", "SetCredentialEncrypted", "ImportCredential")) > 0:
			serviceType = "exec"
		case commands > 0:
			serviceType = "simple"
		default:
			serviceType = "oneshot"
		}
	}
	oneshot := serviceType == "oneshot"

	switch {
	case commands == 0 && stops == 0 && !hasAction:
		return header, "Service has no ExecStart=, ExecStop=, or SuccessAction="
	case commands == 0 && !oneshot:
		return header, "Service has no ExecStart=, which is only allowed for Type=oneshot services"
	case commands == 0 && !hasAction && !isTrue(u.last("Service", "RemainAfterExit").Value):
		return header, "Service has no ExecStart= and no SuccessAction=, and does not set RemainAfterExit=yes"
	case commands > 1 && !oneshot:
		return extra, "Service has more than one ExecStart= command, which is only allowed for Type=oneshot services"
	case oneshot && contains([]string{"always", "on-success"}, u.last("Service", "Restart").Value):
		return u.last("Service", "Restart"), "Restart=always and Restart=on-success are not allowed for Type=oneshot services"
	case oneshot && u.last("Service", "ExitType").Value == "cgroup":
		return u.last("Service", "ExitType"), "ExitType=cgroup is not allowed for Type=oneshot services"
	case serviceType == "dbus" && u.last("Service", "BusName").Value == "":
		return typeEntry, "Type=dbus requires BusName="
	}
	return Entry{}, ""
}

// commandCount counts the command lines of an Exec setting, which a lone
// unquoted ; separates.
func commandCount(value string) int {
	count, inCommand := 0, false
	for _, word := range execWords(value) {
		if word == ";" {
			inCommand = false
		} else if !inCommand {
			count, inCommand = count+1, true
		}
	}
	return count
}

// execWords splits at unquoted whitespace. Quotes and escapes stay in the
// words, so that ";" and \; differ from the separator.
func execWords(value string) []string {
	var words []string
	var word strings.Builder
	quote := byte(0)
	for i := 0; i < len(value); i++ {
		c := value[i]
		switch {
		case c == '\\' && i+1 < len(value):
			word.WriteByte(c)
			i++
			word.WriteByte(value[i])
		case quote != 0:
			if c == quote {
				quote = 0
			}
			word.WriteByte(c)
		case c == ' ' || c == '\t':
			if word.Len() > 0 {
				words = append(words, word.String())
				word.Reset()
			}
		default:
			if c == '\'' || c == '"' {
				quote = c
			}
			word.WriteByte(c)
		}
	}
	if word.Len() > 0 {
		words = append(words, word.String())
	}
	return words
}
