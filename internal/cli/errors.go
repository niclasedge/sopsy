package cli

import (
	"errors"

	"github.com/niclasedge/sopsy/internal/hint"
)

// Error is a failure with a message and the steps that fix it. Messages
// never contain a secret value.
type Error struct {
	Msg  string
	Hint []string
	// Code is the exit code; 0 means ExitFail.
	Code int
}

func (e *Error) Error() string { return e.Msg }

// explain turns any error into an *Error with an actionable hint.
func explain(err error) *Error {
	var e *Error
	if errors.As(err, &e) {
		return e
	}
	msg, steps := hint.Explain(err)
	return &Error{Msg: msg, Hint: steps}
}
