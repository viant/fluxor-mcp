package tool

import "strings"

// Name represents tool name
type Name string

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
