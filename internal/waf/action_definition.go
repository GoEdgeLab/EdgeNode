package waf

import "reflect"

// ActionDefinition action definition
type ActionDefinition struct {
	Name        string
	Code        ActionString
	Description string
	Category    string // category: block, verify, allow
	Instance    ActionInterface
	Type        reflect.Type
}
