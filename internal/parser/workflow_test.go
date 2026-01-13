package parser

import (
	"testing"

	"github.com/bargom/codeai/internal/ast"
)

func TestParseWorkflowWithEventTrigger(t *testing.T) {
	input := `
workflow order_fulfillment {
	trigger event "order.created"

	steps {
		validate_order {
			activity "orders.validate"
			input {
				order_id: "workflow.input.order_id"
			}
		}
		process_payment {
			activity "payments.process"
			input {
				amount: "steps.validate_order.output.total"
			}
		}
	}
}
`

	wf, err := ParseWorkflow(input)
	if err != nil {
		t.Fatalf("ParseWorkflow failed: %v", err)
	}

	if wf.Name != "order_fulfillment" {
		t.Errorf("expected name 'order_fulfillment', got %q", wf.Name)
	}

	if wf.Trigger == nil {
		t.Fatal("expected trigger, got nil")
	}

	if wf.Trigger.TrigType != ast.TriggerTypeEvent {
		t.Errorf("expected trigger type 'event', got %q", wf.Trigger.TrigType)
	}

	if wf.Trigger.Value != "order.created" {
		t.Errorf("expected trigger value 'order.created', got %q", wf.Trigger.Value)
	}

	if len(wf.Steps) != 2 {
		t.Fatalf("expected 2 steps, got %d", len(wf.Steps))
	}

	// Verify first step
	step1 := wf.Steps[0]
	if step1.Name != "validate_order" {
		t.Errorf("expected step name 'validate_order', got %q", step1.Name)
	}
	if step1.Activity != "orders.validate" {
		t.Errorf("expected activity 'orders.validate', got %q", step1.Activity)
	}
	if len(step1.Input) != 1 {
		t.Fatalf("expected 1 input mapping, got %d", len(step1.Input))
	}
	if step1.Input[0].Key != "order_id" {
		t.Errorf("expected input key 'order_id', got %q", step1.Input[0].Key)
	}

	// Verify second step
	step2 := wf.Steps[1]
	if step2.Name != "process_payment" {
		t.Errorf("expected step name 'process_payment', got %q", step2.Name)
	}
}

func TestParseWorkflowWithScheduleTrigger(t *testing.T) {
	input := `
workflow daily_report {
	trigger schedule "0 8 * * *"

	steps {
		generate_report {
			activity "reports.generate_daily"
		}
	}
}
`

	wf, err := ParseWorkflow(input)
	if err != nil {
		t.Fatalf("ParseWorkflow failed: %v", err)
	}

	if wf.Trigger.TrigType != ast.TriggerTypeSchedule {
		t.Errorf("expected trigger type 'schedule', got %q", wf.Trigger.TrigType)
	}

	if wf.Trigger.Value != "0 8 * * *" {
		t.Errorf("expected cron expression '0 8 * * *', got %q", wf.Trigger.Value)
	}
}

func TestParseWorkflowWithManualTrigger(t *testing.T) {
	input := `
workflow manual_process {
	trigger manual

	steps {
		do_work {
			activity "work.do"
		}
	}
}
`

	wf, err := ParseWorkflow(input)
	if err != nil {
		t.Fatalf("ParseWorkflow failed: %v", err)
	}

	if wf.Trigger.TrigType != ast.TriggerTypeManual {
		t.Errorf("expected trigger type 'manual', got %q", wf.Trigger.TrigType)
	}
}

func TestParseWorkflowWithTimeout(t *testing.T) {
	input := `
workflow timed_workflow {
	trigger manual
	timeout "30m"

	steps {
		quick_task {
			activity "tasks.quick"
		}
	}
}
`

	wf, err := ParseWorkflow(input)
	if err != nil {
		t.Fatalf("ParseWorkflow failed: %v", err)
	}

	if wf.Timeout != "30m" {
		t.Errorf("expected timeout '30m', got %q", wf.Timeout)
	}
}

func TestParseWorkflowWithParallelSteps(t *testing.T) {
	input := `
workflow parallel_workflow {
	trigger event "order.paid"

	steps {
		parallel {
			send_email {
				activity "notifications.email"
			}
			send_sms {
				activity "notifications.sms"
			}
		}
	}
}
`

	wf, err := ParseWorkflow(input)
	if err != nil {
		t.Fatalf("ParseWorkflow failed: %v", err)
	}

	if len(wf.Steps) != 1 {
		t.Fatalf("expected 1 step (parallel block), got %d", len(wf.Steps))
	}

	parallelStep := wf.Steps[0]
	if !parallelStep.Parallel {
		t.Error("expected step to be parallel")
	}

	if len(parallelStep.Steps) != 2 {
		t.Fatalf("expected 2 parallel steps, got %d", len(parallelStep.Steps))
	}

	if parallelStep.Steps[0].Name != "send_email" {
		t.Errorf("expected first parallel step 'send_email', got %q", parallelStep.Steps[0].Name)
	}

	if parallelStep.Steps[1].Name != "send_sms" {
		t.Errorf("expected second parallel step 'send_sms', got %q", parallelStep.Steps[1].Name)
	}
}

func TestParseWorkflowWithConditionalStep(t *testing.T) {
	input := `
workflow conditional_workflow {
	trigger event "user.action"

	steps {
		maybe_notify {
			activity "notifications.send"
			if "steps.check.output.should_notify == true"
		}
	}
}
`

	wf, err := ParseWorkflow(input)
	if err != nil {
		t.Fatalf("ParseWorkflow failed: %v", err)
	}

	if len(wf.Steps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(wf.Steps))
	}

	step := wf.Steps[0]
	if step.Condition != "steps.check.output.should_notify == true" {
		t.Errorf("expected condition, got %q", step.Condition)
	}
}

func TestParseWorkflowWithRetryPolicy(t *testing.T) {
	input := `
workflow retry_workflow {
	trigger manual

	steps {
		flaky_step {
			activity "flaky.operation"
		}
	}

	retry {
		max_attempts 5
		initial_interval "1s"
		backoff_multiplier 2.0
	}
}
`

	wf, err := ParseWorkflow(input)
	if err != nil {
		t.Fatalf("ParseWorkflow failed: %v", err)
	}

	if wf.Retry == nil {
		t.Fatal("expected retry policy, got nil")
	}

	if wf.Retry.MaxAttempts != 5 {
		t.Errorf("expected max_attempts 5, got %d", wf.Retry.MaxAttempts)
	}

	if wf.Retry.InitialInterval != "1s" {
		t.Errorf("expected initial_interval '1s', got %q", wf.Retry.InitialInterval)
	}

	if wf.Retry.BackoffMultiplier != 2.0 {
		t.Errorf("expected backoff_multiplier 2.0, got %f", wf.Retry.BackoffMultiplier)
	}
}

func TestParseJobWithCronSchedule(t *testing.T) {
	input := `
job cleanup_logs {
	schedule "0 0 * * 0"
	task "maintenance.cleanup_old_logs"
	queue "low_priority"

	retry {
		max_attempts 3
	}
}
`

	job, err := ParseJob(input)
	if err != nil {
		t.Fatalf("ParseJob failed: %v", err)
	}

	if job.Name != "cleanup_logs" {
		t.Errorf("expected name 'cleanup_logs', got %q", job.Name)
	}

	if job.Schedule != "0 0 * * 0" {
		t.Errorf("expected schedule '0 0 * * 0', got %q", job.Schedule)
	}

	if job.Task != "maintenance.cleanup_old_logs" {
		t.Errorf("expected task 'maintenance.cleanup_old_logs', got %q", job.Task)
	}

	if job.Queue != "low_priority" {
		t.Errorf("expected queue 'low_priority', got %q", job.Queue)
	}

	if job.Retry == nil {
		t.Fatal("expected retry policy, got nil")
	}

	if job.Retry.MaxAttempts != 3 {
		t.Errorf("expected max_attempts 3, got %d", job.Retry.MaxAttempts)
	}
}

func TestParseJobMinimal(t *testing.T) {
	input := `
job simple_job {
	task "simple.task"
}
`

	job, err := ParseJob(input)
	if err != nil {
		t.Fatalf("ParseJob failed: %v", err)
	}

	if job.Name != "simple_job" {
		t.Errorf("expected name 'simple_job', got %q", job.Name)
	}

	if job.Task != "simple.task" {
		t.Errorf("expected task 'simple.task', got %q", job.Task)
	}

	if job.Schedule != "" {
		t.Errorf("expected empty schedule, got %q", job.Schedule)
	}

	if job.Queue != "" {
		t.Errorf("expected empty queue, got %q", job.Queue)
	}
}

func TestParseWorkflowMixedSteps(t *testing.T) {
	input := `
workflow mixed_workflow {
	trigger event "process.start"

	steps {
		first_step {
			activity "first.action"
		}
		parallel {
			parallel_a {
				activity "parallel.a"
			}
			parallel_b {
				activity "parallel.b"
			}
		}
		final_step {
			activity "final.action"
		}
	}
}
`

	wf, err := ParseWorkflow(input)
	if err != nil {
		t.Fatalf("ParseWorkflow failed: %v", err)
	}

	if len(wf.Steps) != 3 {
		t.Fatalf("expected 3 steps, got %d", len(wf.Steps))
	}

	// First step - regular
	if wf.Steps[0].Name != "first_step" {
		t.Errorf("expected first step 'first_step', got %q", wf.Steps[0].Name)
	}
	if wf.Steps[0].Parallel {
		t.Error("first step should not be parallel")
	}

	// Second step - parallel block
	if !wf.Steps[1].Parallel {
		t.Error("second step should be parallel")
	}
	if len(wf.Steps[1].Steps) != 2 {
		t.Errorf("expected 2 parallel steps, got %d", len(wf.Steps[1].Steps))
	}

	// Third step - regular
	if wf.Steps[2].Name != "final_step" {
		t.Errorf("expected third step 'final_step', got %q", wf.Steps[2].Name)
	}
	if wf.Steps[2].Parallel {
		t.Error("third step should not be parallel")
	}
}
