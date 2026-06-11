package validation

import (
	"regexp"
	"strings"
)

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

type Errors map[string]string

func (e Errors) Add(field, message string) {
	e[field] = message
}

func (e Errors) HasErrors() bool {
	return len(e) > 0
}

func IsValidEmail(email string) bool {
	return emailRegex.MatchString(email)
}

func IsBlank(s string) bool {
	return strings.TrimSpace(s) == ""
}
