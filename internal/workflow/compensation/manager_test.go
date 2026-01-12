package compensation_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"

	"github.com/bargom/codeai/internal/workflow/compensation"
	comptesting "github.com/bargom/codeai/internal/workflow/compensation/testing"
)

func TestCompensationManager_BasicCompensation(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	recorder := comptesting.NewCompensationRecorder()

	testWorkflow := func(ctx workflow.Context) error {
		cm := compensation.NewCompensationManager(ctx)

		// Register compensations
		cm.RegisterSimple("step-1", comptesting.MockCompensationFunc(recorder, "step-1"), nil)
		cm.RecordExecution("step-1")

		cm.RegisterSimple("step-2", comptesting.MockCompensationFunc(recorder, "step-2"), nil)
		cm.RecordExecution("step-2")

		cm.RegisterSimple("step-3", comptesting.MockCompensationFunc(recorder, "step-3"), nil)
		cm.RecordExecution("step-3")

		// Trigger compensation
		return cm.Compensate(ctx)
	}

	env.ExecuteWorkflow(testWorkflow)

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	// Verify compensations executed in reverse order
	compensated := recorder.GetCompensated()
	assert.Equal(t, []string{"step-3", "step-2", "step-1"}, compensated)
}

func TestCompensationManager_PartialCompensation(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	recorder := comptesting.NewCompensationRecorder()

	testWorkflow := func(ctx workflow.Context) error {
		cm := compensation.NewCompensationManager(ctx)

		// Register compensations
		cm.RegisterSimple("step-1", comptesting.MockCompensationFunc(recorder, "step-1"), nil)
		cm.RecordExecution("step-1")

		cm.RegisterSimple("step-2", comptesting.MockCompensationFunc(recorder, "step-2"), nil)
		cm.RecordExecution("step-2")

		cm.RegisterSimple("step-3", comptesting.MockCompensationFunc(recorder, "step-3"), nil)
		// step-3 NOT executed

		// Trigger compensation
		return cm.Compensate(ctx)
	}

	env.ExecuteWorkflow(testWorkflow)

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	// Only executed steps should be compensated
	compensated := recorder.GetCompensated()
	assert.Equal(t, []string{"step-2", "step-1"}, compensated)
}

func TestCompensationManager_CompensatePartial(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	recorder := comptesting.NewCompensationRecorder()

	testWorkflow := func(ctx workflow.Context) error {
		cm := compensation.NewCompensationManager(ctx)

		// Register and execute all steps
		cm.RegisterSimple("step-1", comptesting.MockCompensationFunc(recorder, "step-1"), nil)
		cm.RecordExecution("step-1")

		cm.RegisterSimple("step-2", comptesting.MockCompensationFunc(recorder, "step-2"), nil)
		cm.RecordExecution("step-2")

		cm.RegisterSimple("step-3", comptesting.MockCompensationFunc(recorder, "step-3"), nil)
		cm.RecordExecution("step-3")

		// Only compensate specific steps
		return cm.CompensatePartial(ctx, []string{"step-1", "step-3"})
	}

	env.ExecuteWorkflow(testWorkflow)

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	// Only specified steps should be compensated
	compensated := recorder.GetCompensated()
	assert.Contains(t, compensated, "step-1")
	assert.Contains(t, compensated, "step-3")
	assert.NotContains(t, compensated, "step-2")
}

func TestCompensationManager_RecordExecution(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	testWorkflow := func(ctx workflow.Context) ([]string, error) {
		cm := compensation.NewCompensationManager(ctx)

		cm.RecordExecution("step-1")
		cm.RecordExecution("step-2")
		cm.RecordExecution("step-3")

		return cm.GetExecutedActivities(), nil
	}

	env.ExecuteWorkflow(testWorkflow)

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	var result []string
	require.NoError(t, env.GetWorkflowResult(&result))
	assert.Equal(t, []string{"step-1", "step-2", "step-3"}, result)
}

func TestCompensationManager_IsExecuted(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	testWorkflow := func(ctx workflow.Context) ([]bool, error) {
		cm := compensation.NewCompensationManager(ctx)

		cm.RecordExecution("step-1")
		cm.RecordExecution("step-3")

		return []bool{
			cm.IsExecuted("step-1"),
			cm.IsExecuted("step-2"),
			cm.IsExecuted("step-3"),
		}, nil
	}

	env.ExecuteWorkflow(testWorkflow)

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	var result []bool
	require.NoError(t, env.GetWorkflowResult(&result))
	assert.Equal(t, []bool{true, false, true}, result)
}

func TestCompensationManager_Clear(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	testWorkflow := func(ctx workflow.Context) (int, error) {
		cm := compensation.NewCompensationManager(ctx)

		cm.RegisterSimple("step-1", nil, nil)
		cm.RegisterSimple("step-2", nil, nil)
		cm.RecordExecution("step-1")
		cm.RecordExecution("step-2")

		cm.Clear()

		return cm.CompensationCount(), nil
	}

	env.ExecuteWorkflow(testWorkflow)

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	var count int
	require.NoError(t, env.GetWorkflowResult(&count))
	assert.Equal(t, 0, count)
}

func TestCompensationStep_WithTimeout(t *testing.T) {
	step := compensation.CompensationStep{
		ActivityName: "test-step",
		Timeout:      5 * time.Minute,
	}

	assert.Equal(t, "test-step", step.ActivityName)
	assert.Equal(t, 5*time.Minute, step.Timeout)
}

func TestCompensationStep_DefaultTimeout(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	testWorkflow := func(ctx workflow.Context) error {
		cm := compensation.NewCompensationManager(ctx)

		// Register without explicit timeout
		cm.RegisterSimple("step-1", func(ctx workflow.Context, input interface{}) error {
			return nil
		}, nil)

		cm.RecordExecution("step-1")
		return cm.Compensate(ctx)
	}

	env.ExecuteWorkflow(testWorkflow)

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
}
