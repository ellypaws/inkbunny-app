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
	var debug *errors.Error
	if errors.As(err, &debug) {
		return ErrorResponse{Error: err.Error(), Debug: debug.ErrorStack()}
	}
	return ErrorResponse{Error: err.Error(), Debug: err}
}

func (e ErrorResponse) String() string {
	return e.Error
}

func (e ErrorResponse) DebugString() string {
	return TrimPath(errors.New(e.Debug).ErrorStack())
}

func (e ErrorResponse) Map() map[string]any {
	return MapPath(errors.New(e.Debug).ErrorStack())
}

func (e ErrorResponse) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Error string         `json:"error"`
		Debug map[string]any `json:"debug,omitempty"`
	}{
		Error: e.Error,
		Debug: e.Map(),
	})
}

var Crashed = errors.Errorf("oh dear")

func Crash() error {
	return errors.New(Crashed)
}

const (
	projectPrefix = "inkbunny-app/cmd/api"
	apiPrefix     = "inkbunny/api"
)

// TrimPath cleans up the stack trace by only showing the callers
func TrimPath(s string) string {
	lines := strings.Split(s, "\n")

	var keepNext bool
	var out []string
	for i, line := range lines {
		line = strings.TrimSpace(line)
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

// MapPath returns a map of the stack trace.
// The keys are the callers and the values are the lines of the stack trace.
func MapPath(s string) map[string]any {
	lines := strings.Split(s, "\n")

	var out = make(map[string]any)
	var keepNext bool
	for i, line := range lines {
		line = strings.TrimSpace(line)
		switch {
		case keepNext:
			out[lines[i-1]] = line
			keepNext = false
		case strings.Contains(line, projectPrefix):
			lines[i] = removePrefix(line, projectPrefix)
			out[lines[i]] = lines[i]
			keepNext = true
		case strings.Contains(line, apiPrefix):
			lines[i] = removePrefix(line, apiPrefix)
			out[lines[i]] = lines[i]
			keepNext = true
		}
	}
	return out
}

func removePrefix(line string, prefix string) string {
	index := strings.Index(line, prefix)
	return line[index:]
}
