package retrier

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDefaultPoller(t *testing.T) {
	t.Run("success on first try", func(t *testing.T) {
		poller := NewDefaultPoller()
		callCount := 0
		err := poller.Poll(context.Background(), func(ctx context.Context) (bool, error) {
			callCount++
			return true, nil
		})
		assert.NoError(t, err)
		assert.Equal(t, 1, callCount)
	})

	t.Run("success after retries", func(t *testing.T) {
		poller := NewDefaultPoller()
		callCount := 0
		err := poller.Poll(context.Background(), func(ctx context.Context) (bool, error) {
			callCount++
			if callCount < 3 {
				return false, errors.New("temporary error")
			}
			return true, nil
		})
		assert.NoError(t, err)
		assert.Equal(t, 3, callCount)
	})

	t.Run("failure after max retries", func(t *testing.T) {
		poller := NewDefaultPoller()
		callCount := 0
		expectedErr := errors.New("persistent error")
		err := poller.Poll(context.Background(), func(ctx context.Context) (bool, error) {
			callCount++
			return false, expectedErr
		})
		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Equal(t, ValidationMaxRetries, callCount)
	})

	t.Run("context cancellation", func(t *testing.T) {
		poller := NewDefaultPoller()
		ctx, cancel := context.WithCancel(context.Background())
		callCount := 0

		go func() {
			time.Sleep(10 * time.Millisecond)
			cancel()
		}()

		err := poller.Poll(ctx, func(ctx context.Context) (bool, error) {
			callCount++
			time.Sleep(50 * time.Millisecond)
			return false, errors.New("temporary error")
		})

		assert.Error(t, err)
		assert.True(t, errors.Is(err, context.Canceled))
	})

	t.Run("custom parameters", func(t *testing.T) {
		poller := NewDefaultPoller()
		callCount := 0
		customMaxRetries := 3
		expectedErr := errors.New("persistent error")

		err := poller.PollCustom(
			context.Background(),
			5*time.Millisecond,
			50*time.Millisecond,
			customMaxRetries,
			func(ctx context.Context) (bool, error) {
				callCount++
				return false, expectedErr
			},
		)

		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Equal(t, customMaxRetries, callCount)
	})
}

func TestMockPoller(t *testing.T) {
	t.Run("mock poll function", func(t *testing.T) {
		mockPoller := NewMockPoller()
		expectedErr := errors.New("mock error")

		// Set up the mock function
		mockPoller.PollFunc = func(ctx context.Context, conditionFunc func(ctx context.Context) (bool, error)) error {
			return expectedErr
		}

		// Call the function
		err := mockPoller.Poll(context.Background(), func(ctx context.Context) (bool, error) {
			return true, nil
		})

		// Verify the result
		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Equal(t, 1, len(mockPoller.PollCalls))
	})

	t.Run("mock poll custom function", func(t *testing.T) {
		mockPoller := NewMockPoller()
		expectedErr := errors.New("mock custom error")

		// Set up the mock function
		mockPoller.PollCustomFunc = func(ctx context.Context, interval, timeout time.Duration, maxRetries int, conditionFunc func(ctx context.Context) (bool, error)) error {
			return expectedErr
		}

		// Call the function
		err := mockPoller.PollCustom(
			context.Background(),
			5*time.Millisecond,
			50*time.Millisecond,
			3,
			func(ctx context.Context) (bool, error) {
				return true, nil
			},
		)

		// Verify the result
		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Equal(t, 1, len(mockPoller.PollCustomCalls))
		assert.Equal(t, 5*time.Millisecond, mockPoller.PollCustomCalls[0].Interval)
		assert.Equal(t, 50*time.Millisecond, mockPoller.PollCustomCalls[0].Timeout)
		assert.Equal(t, 3, mockPoller.PollCustomCalls[0].MaxRetries)
	})

	t.Run("reset calls", func(t *testing.T) {
		mockPoller := NewMockPoller()

		// Make some calls
		mockPoller.Poll(context.Background(), func(ctx context.Context) (bool, error) {
			return true, nil
		})
		mockPoller.PollCustom(
			context.Background(),
			5*time.Millisecond,
			50*time.Millisecond,
			3,
			func(ctx context.Context) (bool, error) {
				return true, nil
			},
		)

		// Verify calls were tracked
		assert.Equal(t, 1, len(mockPoller.PollCalls))
		assert.Equal(t, 1, len(mockPoller.PollCustomCalls))

		// Reset calls
		mockPoller.ResetCalls()

		// Verify calls were reset
		assert.Equal(t, 0, len(mockPoller.PollCalls))
		assert.Equal(t, 0, len(mockPoller.PollCustomCalls))
	})
}

func TestBackwardCompatibility(t *testing.T) {
	t.Run("PollWithRetries calls DefaultPoller.Poll", func(t *testing.T) {
		callCount := 0
		err := PollWithRetries(context.Background(), func(ctx context.Context) (bool, error) {
			callCount++
			return true, nil
		})
		assert.NoError(t, err)
		assert.Equal(t, 1, callCount)
	})

	t.Run("PollWithRetriesCustom calls DefaultPoller.PollCustom", func(t *testing.T) {
		callCount := 0
		customMaxRetries := 3
		expectedErr := errors.New("persistent error")

		err := PollWithRetriesCustom(
			context.Background(),
			5*time.Millisecond,
			50*time.Millisecond,
			customMaxRetries,
			func(ctx context.Context) (bool, error) {
				callCount++
				return false, expectedErr
			},
		)

		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Equal(t, customMaxRetries, callCount)
	})
}
