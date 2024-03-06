package crashy

import (
	"encoding/json"
	"github.com/go-errors/errors"
	"strings"
)

type ErrorResponse struct {
	Error string `json:"error"`
	Debug any    `json:"debug,omitempty"`
}

func Wrap(err error) ErrorResponse {
	return ErrorResponse{Error: err.Error(), Debug: err.(*errors.Error).ErrorStack()}
}

func (e ErrorResponse) String() string {
	return e.Error
}

func (e ErrorResponse) DebugString() string {
	return TrimPath(errors.New(e.Debug).ErrorStack())
}

func (e ErrorResponse) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Error string `json:"error"`
		Debug any    `json:"debug,omitempty"`
	}{
		Error: e.Error,
		Debug: e.DebugString(),
	})
}

var Crashed = errors.Errorf("oh dear")

func Crash() error {
	return errors.New(Crashed)
}

// TrimPath cleans up the stack trace by only showing the callers
func TrimPath(s string) string {
	const projectPrefix = "inkbunny-app/cmd/api"
	const apiPrefix = "inkbunny/api"

	lines := strings.Split(s, "\n")

	var keepNext bool
	var out []string
	for i, line := range lines {
		switch {
		case keepNext:
			out = append(out, line)
			keepNext = false
		case strings.Contains(line, projectPrefix):
			lines[i] = removePrefix(line, projectPrefix)
			out = append(out, lines[i])
			keepNext = true
		case strings.Contains(line, apiPrefix):
			lines[i] = removePrefix(line, apiPrefix)
			out = append(out, lines[i])
			keepNext = true
		}
	}

	return strings.Join(out, "\n")
}

func removePrefix(line string, prefix string) string {
	index := strings.Index(line, prefix)
	return line[index:]
}
