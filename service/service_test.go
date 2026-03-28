package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestServiceStopSetsCancelCause verifies that after Stop() is called,
// context.Cause(ctx) returns the expected cause string.
func TestServiceStopSetsCancelCause(t *testing.T) {
	sv := &Service{}
	sv.Context, sv.CancelCause = context.WithCancelCause(context.Background())

	// Context should not be cancelled yet
	require.NoError(t, sv.Context.Err())

	// Call Stop (passing nil for kservice.Service since binder is nil)
	err := sv.Stop(nil)
	require.NoError(t, err)

	// Context should now be cancelled
	assert.Error(t, sv.Context.Err())

	// Verify the cause is the expected string
	cause := context.Cause(sv.Context)
	require.NotNil(t, cause)
	assert.Equal(t, "service stop requested", cause.Error())
}
