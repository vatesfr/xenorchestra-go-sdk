package integration

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAuthentication(t *testing.T) {
	tc := Setup(t)

	assert.NotNil(t, tc.Client, "Client should not be nil")

	v1Client := tc.Client.V1Client()
	assert.NotNil(t, v1Client, "V1 client should not be nil")

	user, err := v1Client.GetCurrentUser()
	assert.NoError(t, err, "Should be able to get current user")
	assert.NotNil(t, user, "User should not be nil")
	assert.NotEmpty(t, user.Id, "User ID should not be empty")

	t.Logf("Successfully authenticated as user: %s", user.Email)
}
