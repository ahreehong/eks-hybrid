package retrier

import (
	"context"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"
)

// Poller is an interface for polling operations with retries
type Poller interface {
	// Poll polls until the context is done, an error is returned, or the condition returns true.
	// It uses the standard validation parameters (interval, timeout) and implements the consecutive errors
	// pattern used throughout the codebase.
	Poll(ctx context.Context, conditionFunc func(ctx context.Context) (bool, error)) error

	// PollCustom is similar to Poll but allows customizing the interval and timeout.
	PollCustom(ctx context.Context, interval, timeout time.Duration, maxRetries int, conditionFunc func(ctx context.Context) (bool, error)) error
}

// DefaultPoller is the default implementation of the Poller interface
type DefaultPoller struct{}

// NewDefaultPoller creates a new DefaultPoller
func NewDefaultPoller() *DefaultPoller {
	return &DefaultPoller{}
}

// Poll polls until the context is done, an error is returned, or the condition returns true.
// It uses the standard validation parameters (interval, timeout) and implements the consecutive errors
// pattern used throughout the codebase.
func (p *DefaultPoller) Poll(ctx context.Context, conditionFunc func(ctx context.Context) (bool, error)) error {
	consecutiveErrors := 0
	return wait.PollUntilContextTimeout(
		ctx,
		ValidationInterval,
		ValidationTimeout,
		true,
		func(ctx context.Context) (bool, error) {
			done, err := conditionFunc(ctx)
			if err != nil {
				consecutiveErrors++
				if consecutiveErrors >= ValidationMaxRetries {
					return false, err
				}
				return false, nil // continue polling
			}
			consecutiveErrors = 0 // Reset counter on success
			return done, nil
		},
	)
}

// PollCustom is similar to Poll but allows customizing the interval and timeout.
func (p *DefaultPoller) PollCustom(ctx context.Context, interval, timeout time.Duration, maxRetries int, conditionFunc func(ctx context.Context) (bool, error)) error {
	consecutiveErrors := 0
	return wait.PollUntilContextTimeout(
		ctx,
		interval,
		timeout,
		true,
		func(ctx context.Context) (bool, error) {
			done, err := conditionFunc(ctx)
			if err != nil {
				consecutiveErrors++
				if consecutiveErrors >= maxRetries {
					return false, err
				}
				return false, nil // continue polling
			}
			consecutiveErrors = 0 // Reset counter on success
			return done, nil
		},
	)
}

// For backward compatibility
var defaultPoller = NewDefaultPoller()

// PollWithRetries is a wrapper around DefaultPoller.Poll for backward compatibility
func PollWithRetries(ctx context.Context, conditionFunc func(ctx context.Context) (bool, error)) error {
	return defaultPoller.Poll(ctx, conditionFunc)
}

// PollWithRetriesCustom is a wrapper around DefaultPoller.PollCustom for backward compatibility
func PollWithRetriesCustom(ctx context.Context, interval, timeout time.Duration, maxRetries int, conditionFunc func(ctx context.Context) (bool, error)) error {
	return defaultPoller.PollCustom(ctx, interval, timeout, maxRetries, conditionFunc)
}
