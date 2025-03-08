package crashy

import (
	"encoding/json"
	"github.com/ellypaws/inkbunny/api"
	"github.com/go-errors/errors"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCrash(t *testing.T) {
	err := Crash()
	if err != nil {
		if errors.Is(err, Crashed) {
			t.Log(err.(*errors.Error).ErrorStack())
		} else {
			panic(err)
		}
	}
}

func TestGeneric(t *testing.T) {
	err := any(errors.New("oh no"))
	t.Log(err)
}

func TestTrimPath(t *testing.T) {
	err := Crash()
	t.Log(TrimPath(err.(*errors.Error).ErrorStack()))
}

func TestErrorStack(t *testing.T) {
	err := Crash()
	t.Log(err.(*errors.Error).ErrorStack())
}

func TestDebugString(t *testing.T) {
	err := &ErrorResponse{
		ErrorString: "oh no",
		Debug:       Crash(),
	}
	t.Log(err.DebugString())

	marshal, _ := json.MarshalIndent(err, "", "  ")
	t.Log(string(marshal))
}

func TestRealError(t *testing.T) {
	credentials := (*api.Credentials)(nil)
	_, err := credentials.Login()
	if assert.Error(t, err) {
		err := &ErrorResponse{
			ErrorString: "oh no",
			Debug:       err,
		}
		t.Logf("%+v", err.DebugString())
	}
}
