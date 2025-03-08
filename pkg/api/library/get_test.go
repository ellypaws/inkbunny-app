package library

import (
	"strings"
	"testing"

	"github.com/ellypaws/inkbunny/api"
	"github.com/stretchr/testify/assert"
)

var host = DefaultHost

func TestHost_GetDescription(t *testing.T) {
	// Setup
	const ids = "14576"
	user, err := api.Guest().Login()
	if !assert.NoError(t, err) {
		return
	}
	defer user.Logout()

	r, err := host.GetDescription(user, ids)
	if assert.NoError(t, err) {
		assert.NotEmpty(t, r)
		assert.Equal(t, "Inkbunny Logo (Mascot Only)", r[0].Title)
		assert.Equalf(t, true,
			strings.HasPrefix(r[0].Description, "This image"),
			"Expected description to start with 'This image', got: %s", r[0].Description)
	}
}
