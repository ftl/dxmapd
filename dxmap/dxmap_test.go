package dxmap

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRun_CloseChannelWhenContextIsCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	messages := Run(ctx, "")

	assert.NotNil(t, messages)

	time.Sleep(3 * time.Millisecond)
	cancel()

	select {
	case messages <- nil:
		assert.Fail(t, "messages channel was not closed")
	default:
		// success
	}
}
