package udp

import (
	"reflect"
	"strconv"
	"strings"
)

type apiError struct {
	Code int
	Desc string
}

func (err *apiError) Error() string {
	return fmt.Sprint(err.Code, err.Desc)
}

// The interface for all UDP API replies.
//
// The user should call Error() to verify if the API call completed successfully.
type APIReply interface {
	// An opaque string used as identifying tag.
	Tag() string

	// The integer code for the reply.
	Code() int

	// The description for the reply (first line minus code).
	Text() string

	// Slice with all lines of the reply.
	Lines() []string

	// Indicates whether the network code detected truncation.
	Truncated() bool

	// Returns the underlying error, if any.
	Error() error
}

type errorWrapper struct {
	err error
}

func (_ *errorWrapper) Tag() string {
	return ""
}

func (_ *errorWrapper) Code() int {
	switch e := w.err.(type) {
	case *APIError:
		return e.Code
	default:
		return 999
	}
}

func (w *errorWrapper) Text() string {
	switch e := w.err.(type) {
	case *APIError:
		return e.Desc
	default:
		return e.Error()
	}
}

func (w *errorWrapper) Lines() []string {
	return []string{w.Text()}
}

func (_ *errorWrapper) Truncated() bool {
	return false
}

func (w *errorWrapper) Error() error {
	return w.err
}

func newErrorWrapper(err error) APIReply {
	return &errorWrapper{
		err: err,
	}
}

type genericReply struct {
	raw       []byte
	text      string
	lines     []string
	tag       string
	code      int
	truncated bool
	err       error
}

// Don't expose the newGenericReply call in the documentation
var timeoutResponse = newGenericReply([]byte("604 TIMEOUT - DELAY AND RESUBMIT"))

var TimeoutError = APIReply(timeoutResponse) // API response "604 TIMEOUT - DELAY AND RESUBMIT", also generated on client-side timeouts

func newGenericReply(raw []byte) (r *genericReply) {
	str := string(raw)
	lines := strings.Split(str, "\n")
	parts := strings.Fields(lines[0])

	// invalid packet
	if len(parts) < 1 {
		return nil
	}

	// Drop lines that are only whitespace
	for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
		lines = lines[:len(lines)-1]
	}

	// XXX: REQUIRES that the tag is not parsable as a base 10 number.
	// Just prepending any sent tag with 'T' ought to be enough
	tag := ""
	text := ""
	code, err := strconv.ParseInt(parts[0], 10, 16)
	if err != nil {
		tag = parts[0]
		code, err = strconv.ParseInt(parts[1], 10, 16)

		if len(parts) > 2 {
			text = strings.Join(parts[2:], " ")
		}
	} else if len(parts) > 1 {
		text = strings.Join(parts[1:], " ")
	}

	// Make sure server-side timeouts are comparable against TimeoutError
	if code == timeoutResponse.code {
		return timeoutResponse
	}

	var e *apiError = nil
	// 720-799 range is for notifications
	// 799 is an API server shutdown notice, so I guess it's okay to be an error
	if code < 200 || (code > 299 && code < 720) || code > 798 {
		e = &apiError{Code: int(code), Desc: text}
	}

	return &genericReply{
		tag:   tag,
		code:  int(code),
		text:  text,
		lines: lines,
		err:   e,
	}
}

func (r *genericReply) Tag() string {
	return r.tag
}

func (r *genericReply) Code() int {
	return r.code
}

func (r *genericReply) Text() string {
	return r.text
}

func (r *genericReply) Lines() []string {
	return r.lines
}

func (r *genericReply) Truncated() bool {
	return r.truncated
}

func (r *genericReply) Error() error {
	return r.err
}
