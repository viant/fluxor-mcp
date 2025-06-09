package tool

import "strings"

// Name represents tool name
type Name string

// Canonical converts various user-facing name formats into the canonical
// internal representation: servicePathWithUnderscores-method where service
// path components are joined by underscores and the last dash separates the
// method name.
//
// Supported inputs:
//  1. Canonical form itself – returned unchanged.
//  2. "service/path.method"   – commonly used by CLI commands (slash separated
//     path, dot before method). Example: system/exec.execute →
//     system_exec-execute.
//  3. "service/path-method"   – dash between service and method but slashes
//     kept in service path.
//  4. "service/path/method"   – slash between service & method.
//
// The function purposely stays lenient so that future formats continue to
// work as long as service and method can be unambiguously derived.
func Canonical(raw string) string {
	if raw == "" {
		return ""
	}

	// Already canonical – contains dash separator but no slash/dot afterwards.
	if strings.Contains(raw, "-") && !strings.ContainsAny(raw, "/.") {
		return raw
	}

	var service, method string

	// Handle dot separator first (highest priority to avoid confusion when
	// service path also contains dashes).
	if idx := strings.LastIndex(raw, "."); idx != -1 {
		service, method = raw[:idx], raw[idx+1:]
	} else if idx := strings.LastIndex(raw, "-"); idx != -1 && strings.Contains(raw, "/") {
		// path-with-dash format (service path may include slash) e.g. s/p-m
		service, method = raw[:idx], raw[idx+1:]
	} else if idx := strings.LastIndex(raw, "/"); idx != -1 {
		// service/path/method
		service, method = raw[:idx], raw[idx+1:]
	} else {
		// Fallback: cannot split reliably – return as is.
		return raw
	}

	service = strings.ReplaceAll(service, "/", "_")
	return service + "-" + method
}

func (t Name) Service() string {
	tool := string(t)
	if idx := strings.LastIndex(tool, "-"); idx != -1 {
		return strings.ReplaceAll(tool[:idx], "_", "/")
	}
	return tool
}

func (t Name) Method() string {
	tool := string(t)
	if idx := strings.LastIndex(tool, "-"); idx != -1 {
		return tool[idx+1:]
	}
	return ""
}

func (t Name) ToolName() string {
	r := string(t)
	r = strings.ReplaceAll(r, "/", "_")
	return r
}

func (t Name) String() string {
	return string(t)
}

// NewName new name
func NewName(service, name string) Name {
	return Name(strings.ReplaceAll(service, "/", "_") + "-" + name)
}
