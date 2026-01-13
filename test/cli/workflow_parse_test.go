//go:build integration

// Package cli provides CLI integration tests for CodeAI workflow parsing.
package cli

import (
	"testing"

	"github.com/bargom/codeai/internal/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestWorkflowParsing tests the workflow parser functionality.
func TestWorkflowParsing(t *testing.T) {
	t.Run("parse event-triggered workflow", func(t *testing.T) {
		input := `
workflow order_fulfillment {
	trigger event "order.created"
	timeout "2h"

	steps {
		validate_order {
			activity "orders.validate"
			input {
				order_id: "workflow.input.order_id"
			}
		}
		process_payment {
			activity "payments.process"
		}
	}

	retry {
		max_attempts 3
		initial_interval "10s"
		backoff_multiplier 2.0
	}
}
`
		wf, err := parser.ParseWorkflow(input)
		require.NoError(t, err)
		assert.Equal(t, "order_fulfillment", wf.Name)
		assert.Equal(t, "event", string(wf.Trigger.TrigType))
		assert.Equal(t, "order.created", wf.Trigger.Value)
		assert.Equal(t, "2h", wf.Timeout)
		assert.Len(t, wf.Steps, 2)
		assert.NotNil(t, wf.Retry)
		assert.Equal(t, 3, wf.Retry.MaxAttempts)
	})

	t.Run("parse schedule-triggered workflow", func(t *testing.T) {
		input := `
workflow daily_report {
	trigger schedule "0 6 * * *"

	steps {
		generate {
			activity "reports.generate"
		}
	}
}
`
		wf, err := parser.ParseWorkflow(input)
		require.NoError(t, err)
		assert.Equal(t, "daily_report", wf.Name)
		assert.Equal(t, "schedule", string(wf.Trigger.TrigType))
		assert.Equal(t, "0 6 * * *", wf.Trigger.Value)
	})

	t.Run("parse manual workflow", func(t *testing.T) {
		input := `
workflow onboarding {
	trigger manual

	steps {
		setup {
			activity "users.setup"
		}
	}
}
`
		wf, err := parser.ParseWorkflow(input)
		require.NoError(t, err)
		assert.Equal(t, "onboarding", wf.Name)
		assert.Equal(t, "manual", string(wf.Trigger.TrigType))
	})

	t.Run("parse workflow with parallel steps", func(t *testing.T) {
		input := `
workflow parallel_workflow {
	trigger event "process.start"

	steps {
		parallel {
			task_a {
				activity "tasks.a"
			}
			task_b {
				activity "tasks.b"
			}
		}
	}
}
`
		wf, err := parser.ParseWorkflow(input)
		require.NoError(t, err)
		require.Len(t, wf.Steps, 1)
		assert.True(t, wf.Steps[0].Parallel)
		assert.Len(t, wf.Steps[0].Steps, 2)
	})

	t.Run("parse workflow with conditional step", func(t *testing.T) {
		input := `
workflow conditional_workflow {
	trigger event "user.action"

	steps {
		maybe_notify {
			activity "notifications.send"
			if "condition.check == true"
		}
	}
}
`
		wf, err := parser.ParseWorkflow(input)
		require.NoError(t, err)
		require.Len(t, wf.Steps, 1)
		assert.Equal(t, "condition.check == true", wf.Steps[0].Condition)
	})

	t.Run("parse workflow with input mappings", func(t *testing.T) {
		input := `
workflow mapping_workflow {
	trigger manual

	steps {
		process {
			activity "process.data"
			input {
				user_id: "workflow.input.user_id"
				email: "workflow.input.email"
			}
		}
	}
}
`
		wf, err := parser.ParseWorkflow(input)
		require.NoError(t, err)
		require.Len(t, wf.Steps, 1)
		require.Len(t, wf.Steps[0].Input, 2)
		assert.Equal(t, "user_id", wf.Steps[0].Input[0].Key)
		assert.Equal(t, "workflow.input.user_id", wf.Steps[0].Input[0].Value)
	})
}

// TestJobParsing tests the job parser functionality.
func TestJobParsing(t *testing.T) {
	t.Run("parse job with schedule", func(t *testing.T) {
		input := `
job cleanup_sessions {
	schedule "0 * * * *"
	task "maintenance.cleanup_sessions"
	queue "low_priority"

	retry {
		max_attempts 2
	}
}
`
		job, err := parser.ParseJob(input)
		require.NoError(t, err)
		assert.Equal(t, "cleanup_sessions", job.Name)
		assert.Equal(t, "0 * * * *", job.Schedule)
		assert.Equal(t, "maintenance.cleanup_sessions", job.Task)
		assert.Equal(t, "low_priority", job.Queue)
		assert.NotNil(t, job.Retry)
		assert.Equal(t, 2, job.Retry.MaxAttempts)
	})

	t.Run("parse minimal job", func(t *testing.T) {
		input := `
job simple_task {
	task "simple.run"
}
`
		job, err := parser.ParseJob(input)
		require.NoError(t, err)
		assert.Equal(t, "simple_task", job.Name)
		assert.Equal(t, "simple.run", job.Task)
		assert.Empty(t, job.Schedule)
		assert.Empty(t, job.Queue)
	})

	t.Run("parse job with full retry policy", func(t *testing.T) {
		input := `
job backup_job {
	schedule "0 2 * * 0"
	task "backup.database"
	queue "critical"

	retry {
		max_attempts 5
		initial_interval "1m"
		backoff_multiplier 2.5
	}
}
`
		job, err := parser.ParseJob(input)
		require.NoError(t, err)
		assert.Equal(t, "backup_job", job.Name)
		assert.Equal(t, "0 2 * * 0", job.Schedule)
		require.NotNil(t, job.Retry)
		assert.Equal(t, 5, job.Retry.MaxAttempts)
		assert.Equal(t, "1m", job.Retry.InitialInterval)
		assert.Equal(t, 2.5, job.Retry.BackoffMultiplier)
	})

	t.Run("parse job without queue defaults to empty", func(t *testing.T) {
		input := `
job hourly_sync {
	schedule "0 * * * *"
	task "sync.data"
}
`
		job, err := parser.ParseJob(input)
		require.NoError(t, err)
		assert.Equal(t, "hourly_sync", job.Name)
		assert.Empty(t, job.Queue)
	})
}

// TestWorkflowParsingErrors tests error handling in workflow parsing.
func TestWorkflowParsingErrors(t *testing.T) {
	t.Run("invalid workflow syntax", func(t *testing.T) {
		input := `workflow { missing name }`
		_, err := parser.ParseWorkflow(input)
		assert.Error(t, err)
	})

	t.Run("missing trigger", func(t *testing.T) {
		input := `
workflow no_trigger {
	steps {
		step1 {
			activity "test"
		}
	}
}
`
		_, err := parser.ParseWorkflow(input)
		assert.Error(t, err)
	})

	t.Run("missing steps", func(t *testing.T) {
		input := `
workflow no_steps {
	trigger manual
}
`
		_, err := parser.ParseWorkflow(input)
		assert.Error(t, err)
	})
}

// TestJobParsingErrors tests error handling in job parsing.
func TestJobParsingErrors(t *testing.T) {
	t.Run("invalid job syntax", func(t *testing.T) {
		input := `job { missing name }`
		_, err := parser.ParseJob(input)
		assert.Error(t, err)
	})

	t.Run("missing task", func(t *testing.T) {
		input := `
job no_task {
	schedule "0 * * * *"
}
`
		_, err := parser.ParseJob(input)
		assert.Error(t, err)
	})
}
