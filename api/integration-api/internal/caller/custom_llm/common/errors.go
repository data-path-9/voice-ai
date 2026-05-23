package internal_custom_llm_common

import "fmt"

type UnsupportedCompatibilityError struct {
	Compatibility Compatibility
}

func (e UnsupportedCompatibilityError) Error() string {
	return fmt.Sprintf("custom-llm: unsupported api compatibility %q", e.Compatibility)
}

type NotImplementedCompatibilityError struct {
	Compatibility Compatibility
	Feature       string
}

func (e NotImplementedCompatibilityError) Error() string {
	return fmt.Sprintf(
		"custom-llm: compatibility %q does not implement %s yet",
		e.Compatibility,
		e.Feature,
	)
}
