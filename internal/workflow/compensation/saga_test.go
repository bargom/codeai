package compensation

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSagaManager(t *testing.T) {
	sm := NewSagaManager()
	assert.NotNil(t, sm)
	assert.Equal(t, 0, sm.Count())
}

func TestSagaManagerAddCompensation(t *testing.T) {
	sm := NewSagaManager()

	sm.AddCompensation("comp1", func(ctx context.Context) error { return nil })
	assert.Equal(t, 1, sm.Count())

	sm.AddCompensation("comp2", func(ctx context.Context) error { return nil })
	assert.Equal(t, 2, sm.Count())
}

func TestSagaManagerCompensate(t *testing.T) {
	t.Run("successful compensation in reverse order", func(t *testing.T) {
		sm := NewSagaManager()
		var order []string

		sm.AddCompensation("first", func(ctx context.Context) error {
			order = append(order, "first")
			return nil
		})
		sm.AddCompensation("second", func(ctx context.Context) error {
			order = append(order, "second")
			return nil
		})
		sm.AddCompensation("third", func(ctx context.Context) error {
			order = append(order, "third")
			return nil
		})

		err := sm.Compensate(context.Background())
		require.NoError(t, err)

		// Should be executed in reverse order (LIFO)
		assert.Equal(t, []string{"third", "second", "first"}, order)
	})

	t.Run("continues on failure", func(t *testing.T) {
		sm := NewSagaManager()
		var executed []string
		testErr := errors.New("compensation failed")

		sm.AddCompensation("first", func(ctx context.Context) error {
			executed = append(executed, "first")
			return nil
		})
		sm.AddCompensation("second", func(ctx context.Context) error {
			executed = append(executed, "second")
			return testErr
		})
		sm.AddCompensation("third", func(ctx context.Context) error {
			executed = append(executed, "third")
			return nil
		})

		err := sm.Compensate(context.Background())
		require.Error(t, err)

		// All compensations should have been executed
		assert.Equal(t, []string{"third", "second", "first"}, executed)

		// Error should be a CompensationError
		var compErr *CompensationError
		assert.True(t, errors.As(err, &compErr))
		assert.Len(t, compErr.Errors, 1)
	})
}

func TestSagaManagerClear(t *testing.T) {
	sm := NewSagaManager()

	sm.AddCompensation("comp1", func(ctx context.Context) error { return nil })
	sm.AddCompensation("comp2", func(ctx context.Context) error { return nil })
	assert.Equal(t, 2, sm.Count())

	sm.Clear()
	assert.Equal(t, 0, sm.Count())
}

func TestSagaManagerRecords(t *testing.T) {
	sm := NewSagaManager()

	sm.AddCompensation("comp1", func(ctx context.Context) error { return nil })
	sm.AddCompensation("comp2", func(ctx context.Context) error { return errors.New("test error") })

	_ = sm.Compensate(context.Background())

	records := sm.Records()
	require.Len(t, records, 2)

	// First record (comp2 - executed first in reverse order)
	assert.Equal(t, "comp2", records[0].Name)
	assert.True(t, records[0].Executed)
	assert.NotEmpty(t, records[0].Error)

	// Second record (comp1 - executed second in reverse order)
	assert.Equal(t, "comp1", records[1].Name)
	assert.True(t, records[1].Executed)
	assert.Empty(t, records[1].Error)
}

func TestCompensationError(t *testing.T) {
	t.Run("single error", func(t *testing.T) {
		err := &CompensationError{Errors: []error{errors.New("single")}}
		assert.Equal(t, "single", err.Error())
		assert.NotNil(t, err.Unwrap())
	})

	t.Run("multiple errors", func(t *testing.T) {
		err := &CompensationError{Errors: []error{
			errors.New("first"),
			errors.New("second"),
		}}
		assert.Contains(t, err.Error(), "2 compensation failures")
	})

	t.Run("no errors", func(t *testing.T) {
		err := &CompensationError{Errors: []error{}}
		assert.Nil(t, err.Unwrap())
	})
}

func TestExecuteSaga(t *testing.T) {
	t.Run("all steps succeed", func(t *testing.T) {
		var executed []string

		steps := []SagaStep{
			{
				Name: "step1",
				Action: func(ctx context.Context) error {
					executed = append(executed, "action1")
					return nil
				},
				Compensation: func(ctx context.Context) error {
					executed = append(executed, "comp1")
					return nil
				},
			},
			{
				Name: "step2",
				Action: func(ctx context.Context) error {
					executed = append(executed, "action2")
					return nil
				},
				Compensation: func(ctx context.Context) error {
					executed = append(executed, "comp2")
					return nil
				},
			},
		}

		records, err := ExecuteSaga(context.Background(), steps)
		require.NoError(t, err)
		assert.Nil(t, records)
		assert.Equal(t, []string{"action1", "action2"}, executed)
	})

	t.Run("step fails triggers compensation", func(t *testing.T) {
		var executed []string
		testErr := errors.New("step2 failed")

		steps := []SagaStep{
			{
				Name: "step1",
				Action: func(ctx context.Context) error {
					executed = append(executed, "action1")
					return nil
				},
				Compensation: func(ctx context.Context) error {
					executed = append(executed, "comp1")
					return nil
				},
			},
			{
				Name: "step2",
				Action: func(ctx context.Context) error {
					executed = append(executed, "action2")
					return testErr
				},
				Compensation: func(ctx context.Context) error {
					executed = append(executed, "comp2")
					return nil
				},
			},
		}

		records, err := ExecuteSaga(context.Background(), steps)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "step2")
		assert.Contains(t, err.Error(), "compensated")
		assert.NotNil(t, records)

		// action1 executed, then action2 (failed), then comp1 (only step1 was registered)
		assert.Equal(t, []string{"action1", "action2", "comp1"}, executed)
	})
}

func TestTransactionalSaga(t *testing.T) {
	t.Run("commit prevents rollback", func(t *testing.T) {
		saga := NewTransactionalSaga()
		var compensationCalled bool

		saga.AddCompensation("comp1", func(ctx context.Context) error {
			compensationCalled = true
			return nil
		})

		saga.Commit()
		assert.True(t, saga.IsCommitted())

		err := saga.Rollback(context.Background())
		assert.NoError(t, err)
		assert.False(t, compensationCalled)
	})

	t.Run("rollback without commit executes compensations", func(t *testing.T) {
		saga := NewTransactionalSaga()
		var compensationCalled bool

		saga.AddCompensation("comp1", func(ctx context.Context) error {
			compensationCalled = true
			return nil
		})

		assert.False(t, saga.IsCommitted())

		err := saga.Rollback(context.Background())
		assert.NoError(t, err)
		assert.True(t, compensationCalled)
	})
}

func TestCompensationBuilder(t *testing.T) {
	t.Run("basic compensation", func(t *testing.T) {
		var called bool
		name, fn := NewCompensation("test").
			WithAction(func(ctx context.Context) error {
				called = true
				return nil
			}).
			Build()

		assert.Equal(t, "test", name)
		require.NotNil(t, fn)

		err := fn(context.Background())
		assert.NoError(t, err)
		assert.True(t, called)
	})

	t.Run("with retry", func(t *testing.T) {
		attempts := 0
		_, fn := NewCompensation("test").
			WithAction(func(ctx context.Context) error {
				attempts++
				if attempts < 3 {
					return errors.New("retry")
				}
				return nil
			}).
			WithRetry(3).
			Build()

		err := fn(context.Background())
		assert.NoError(t, err)
		assert.Equal(t, 3, attempts)
	})

	t.Run("retry exhausted", func(t *testing.T) {
		_, fn := NewCompensation("test").
			WithAction(func(ctx context.Context) error {
				return errors.New("always fails")
			}).
			WithRetry(2).
			Build()

		err := fn(context.Background())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "2 attempts")
	})
}
