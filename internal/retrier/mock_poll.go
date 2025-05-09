package retrier

import (
	"context"
	"time"
)

// MockPoller is a mock implementation of the Poller interface for testing
type MockPoller struct {
	// PollFunc is the function that will be called when Poll is called
	PollFunc func(ctx context.Context, conditionFunc func(ctx context.Context) (bool, error)) error

	// PollCustomFunc is the function that will be called when PollCustom is called
	PollCustomFunc func(ctx context.Context, interval, timeout time.Duration, maxRetries int, conditionFunc func(ctx context.Context) (bool, error)) error

	// PollCalls tracks the calls to Poll
	PollCalls []PollCall

	// PollCustomCalls tracks the calls to PollCustom
	PollCustomCalls []PollCustomCall
}

// PollCall represents a call to Poll
type PollCall struct {
	Ctx           context.Context
	ConditionFunc func(ctx context.Context) (bool, error)
}

// PollCustomCall represents a call to PollCustom
type PollCustomCall struct {
	Ctx           context.Context
	Interval      time.Duration
	Timeout       time.Duration
	MaxRetries    int
	ConditionFunc func(ctx context.Context) (bool, error)
}

// NewMockPoller creates a new MockPoller
func NewMockPoller() *MockPoller {
	return &MockPoller{
		PollFunc: func(ctx context.Context, conditionFunc func(ctx context.Context) (bool, error)) error {
			return nil
		},
		PollCustomFunc: func(ctx context.Context, interval, timeout time.Duration, maxRetries int, conditionFunc func(ctx context.Context) (bool, error)) error {
			return nil
		},
		PollCalls:       []PollCall{},
		PollCustomCalls: []PollCustomCall{},
	}
}

// Poll implements the Poller interface
func (m *MockPoller) Poll(ctx context.Context, conditionFunc func(ctx context.Context) (bool, error)) error {
	m.PollCalls = append(m.PollCalls, PollCall{
		Ctx:           ctx,
		ConditionFunc: conditionFunc,
	})
	return m.PollFunc(ctx, conditionFunc)
}

// PollCustom implements the Poller interface
func (m *MockPoller) PollCustom(ctx context.Context, interval, timeout time.Duration, maxRetries int, conditionFunc func(ctx context.Context) (bool, error)) error {
	m.PollCustomCalls = append(m.PollCustomCalls, PollCustomCall{
		Ctx:           ctx,
		Interval:      interval,
		Timeout:       timeout,
		MaxRetries:    maxRetries,
		ConditionFunc: conditionFunc,
	})
	return m.PollCustomFunc(ctx, interval, timeout, maxRetries, conditionFunc)
}

// ResetCalls resets all call tracking
func (m *MockPoller) ResetCalls() {
	m.PollCalls = []PollCall{}
	m.PollCustomCalls = []PollCustomCall{}
}
