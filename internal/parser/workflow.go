// Package parser provides workflow and job parsing for CodeAI DSL.
package parser

import (
	"github.com/alecthomas/participle/v2"
	"github.com/alecthomas/participle/v2/lexer"

	"github.com/bargom/codeai/internal/ast"
)

// =============================================================================
// Workflow Lexer - Dedicated lexer for workflow/job parsing
// =============================================================================

var workflowLexer = lexer.MustStateful(lexer.Rules{
	"Root": {
		{Name: "Whitespace", Pattern: `[\s\t\n\r]+`, Action: nil},
		{Name: "Comment", Pattern: `//[^\n]*`, Action: nil},
		{Name: "MultiLineComment", Pattern: `/\*[^*]*\*+(?:[^/*][^*]*\*+)*/`, Action: nil},

		// Keywords - order matters, longer matches first
		{Name: "Workflow", Pattern: `\bworkflow\b`, Action: nil},
		{Name: "Job", Pattern: `\bjob\b`, Action: nil},
		{Name: "Trigger", Pattern: `\btrigger\b`, Action: nil},
		{Name: "Event", Pattern: `\bevent\b`, Action: nil},
		{Name: "Schedule", Pattern: `\bschedule\b`, Action: nil},
		{Name: "Manual", Pattern: `\bmanual\b`, Action: nil},
		{Name: "Timeout", Pattern: `\btimeout\b`, Action: nil},
		{Name: "Steps", Pattern: `\bsteps\b`, Action: nil},
		{Name: "Parallel", Pattern: `\bparallel\b`, Action: nil},
		{Name: "Activity", Pattern: `\bactivity\b`, Action: nil},
		{Name: "Input", Pattern: `\binput\b`, Action: nil},
		{Name: "If", Pattern: `\bif\b`, Action: nil},
		{Name: "Retry", Pattern: `\bretry\b`, Action: nil},
		{Name: "MaxAttempts", Pattern: `\bmax_attempts\b`, Action: nil},
		{Name: "InitialInterval", Pattern: `\binitial_interval\b`, Action: nil},
		{Name: "BackoffMultiplier", Pattern: `\bbackoff_multiplier\b`, Action: nil},
		{Name: "Task", Pattern: `\btask\b`, Action: nil},
		{Name: "Queue", Pattern: `\bqueue\b`, Action: nil},

		// Literals
		{Name: "Float", Pattern: `[0-9]+\.[0-9]+`, Action: nil},
		{Name: "Number", Pattern: `[0-9]+`, Action: nil},
		{Name: "String", Pattern: `"[^"]*"`, Action: nil},
		{Name: "Ident", Pattern: `[a-zA-Z_][a-zA-Z0-9_]*`, Action: nil},

		// Operators and punctuation
		{Name: "Arrow", Pattern: `->`, Action: nil},
		{Name: "LBrace", Pattern: `\{`, Action: nil},
		{Name: "RBrace", Pattern: `\}`, Action: nil},
		{Name: "LParen", Pattern: `\(`, Action: nil},
		{Name: "RParen", Pattern: `\)`, Action: nil},
		{Name: "Comma", Pattern: `,`, Action: nil},
		{Name: "Colon", Pattern: `:`, Action: nil},
	},
})

// =============================================================================
// Workflow Grammar Structs
// =============================================================================

// pWorkflowDecl represents the parsed workflow declaration.
type pWorkflowDecl struct {
	pos     lexer.Position
	Name    string           `parser:"\"workflow\" @Ident \"{\""`
	Trigger *pTrigger        `parser:"@@"`
	Timeout *string          `parser:"( \"timeout\" @String )?"`
	Steps   []*pWorkflowStep `parser:"\"steps\" \"{\" @@* \"}\""`
	Retry   *pRetryPolicy    `parser:"@@?"`
	End     struct{}         `parser:"\"}\""`
}

// pTrigger represents a workflow trigger.
type pTrigger struct {
	pos      lexer.Position
	Event    *string `parser:"\"trigger\" ( \"event\" @String"`
	Schedule *string `parser:"         | \"schedule\" @String"`
	Manual   bool    `parser:"         | @\"manual\" )"`
}

// pWorkflowStep represents a step in the workflow.
type pWorkflowStep struct {
	pos      lexer.Position
	Parallel *pParallelBlock `parser:"  @@"`
	Regular  *pRegularStep   `parser:"| @@"`
}

// pRegularStep represents a regular (non-parallel) workflow step.
type pRegularStep struct {
	pos       lexer.Position
	Name      string       `parser:"@Ident \"{\""`
	Activity  *string      `parser:"( \"activity\" @String )?"`
	Input     *pInputBlock `parser:"@@?"`
	Condition *string      `parser:"( \"if\" @String )?"`
	End       struct{}     `parser:"\"}\""`
}

// pParallelBlock represents a parallel execution block.
type pParallelBlock struct {
	pos   lexer.Position
	Steps []*pRegularStep `parser:"\"parallel\" \"{\" @@* \"}\""`
}

// pInputBlock represents input mappings for a step.
type pInputBlock struct {
	pos      lexer.Position
	Mappings []*pInputMapping `parser:"\"input\" \"{\" @@* \"}\""`
}

// pInputMapping represents a single input mapping.
type pInputMapping struct {
	pos   lexer.Position
	Key   string `parser:"@Ident \":\""`
	Value string `parser:"@String"`
}

// pRetryPolicy represents retry configuration.
type pRetryPolicy struct {
	pos               lexer.Position
	MaxAttempts       *int     `parser:"\"retry\" \"{\" ( \"max_attempts\" @Number )?"`
	InitialInterval   *string  `parser:"( \"initial_interval\" @String )?"`
	BackoffMultiplier *float64 `parser:"( \"backoff_multiplier\" @Float )? \"}\""`
}

// =============================================================================
// Job Grammar Structs
// =============================================================================

// pJobDecl represents the parsed job declaration.
type pJobDecl struct {
	pos      lexer.Position
	Name     string        `parser:"\"job\" @Ident \"{\""`
	Schedule *string       `parser:"( \"schedule\" @String )?"`
	Task     string        `parser:"\"task\" @String"`
	Queue    *string       `parser:"( \"queue\" @String )?"`
	Retry    *pRetryPolicy `parser:"@@?"`
	End      struct{}      `parser:"\"}\""`
}

// =============================================================================
// Parser Instances
// =============================================================================

var workflowParser = participle.MustBuild[pWorkflowDecl](
	participle.Lexer(workflowLexer),
	participle.UseLookahead(2),
	participle.Elide("Whitespace", "Comment", "MultiLineComment"),
)

var jobParser = participle.MustBuild[pJobDecl](
	participle.Lexer(workflowLexer),
	participle.UseLookahead(2),
	participle.Elide("Whitespace", "Comment", "MultiLineComment"),
)

// =============================================================================
// Public API
// =============================================================================

// ParseWorkflow parses a workflow declaration from the given input string.
func ParseWorkflow(input string) (*ast.WorkflowDecl, error) {
	parsed, err := workflowParser.ParseString("", input)
	if err != nil {
		return nil, err
	}
	return convertWorkflowFromParsed(parsed), nil
}

// ParseJob parses a job declaration from the given input string.
func ParseJob(input string) (*ast.JobDecl, error) {
	parsed, err := jobParser.ParseString("", input)
	if err != nil {
		return nil, err
	}
	return convertJobFromParsed(parsed), nil
}

// =============================================================================
// Conversion Functions
// =============================================================================

// convertWorkflowFromParsed converts a parsed workflow to an AST node.
func convertWorkflowFromParsed(p *pWorkflowDecl) *ast.WorkflowDecl {
	workflow := &ast.WorkflowDecl{
		Name: p.Name,
	}

	// Convert trigger
	if p.Trigger != nil {
		workflow.Trigger = convertTriggerFromParsed(p.Trigger)
	}

	// Set timeout
	if p.Timeout != nil {
		workflow.Timeout = trimQuotes(*p.Timeout)
	}

	// Convert steps
	for _, step := range p.Steps {
		workflow.Steps = append(workflow.Steps, convertWorkflowStepFromParsed(step))
	}

	// Convert retry policy
	if p.Retry != nil {
		workflow.Retry = convertRetryPolicyFromParsed(p.Retry)
	}

	return workflow
}

// convertTriggerFromParsed converts a parsed trigger to an AST node.
func convertTriggerFromParsed(p *pTrigger) *ast.Trigger {
	trigger := &ast.Trigger{}

	if p.Event != nil {
		trigger.TrigType = ast.TriggerTypeEvent
		trigger.Value = trimQuotes(*p.Event)
	} else if p.Schedule != nil {
		trigger.TrigType = ast.TriggerTypeSchedule
		trigger.Value = trimQuotes(*p.Schedule)
	} else if p.Manual {
		trigger.TrigType = ast.TriggerTypeManual
		trigger.Value = ""
	}

	return trigger
}

// convertWorkflowStepFromParsed converts a parsed workflow step to an AST node.
func convertWorkflowStepFromParsed(p *pWorkflowStep) *ast.WorkflowStep {
	step := &ast.WorkflowStep{}

	if p.Parallel != nil {
		// Handle parallel block
		step.Parallel = true
		for _, regularStep := range p.Parallel.Steps {
			step.Steps = append(step.Steps, convertRegularStepFromParsed(regularStep))
		}
	} else if p.Regular != nil {
		// Handle regular step
		return convertRegularStepFromParsed(p.Regular)
	}

	return step
}

// convertRegularStepFromParsed converts a parsed regular step to an AST node.
func convertRegularStepFromParsed(p *pRegularStep) *ast.WorkflowStep {
	step := &ast.WorkflowStep{
		Name: p.Name,
	}

	if p.Activity != nil {
		step.Activity = trimQuotes(*p.Activity)
	}

	if p.Input != nil {
		for _, mapping := range p.Input.Mappings {
			step.Input = append(step.Input, &ast.InputMapping{
				Key:   mapping.Key,
				Value: trimQuotes(mapping.Value),
			})
		}
	}

	if p.Condition != nil {
		step.Condition = trimQuotes(*p.Condition)
	}

	return step
}

// convertRetryPolicyFromParsed converts a parsed retry policy to an AST node.
func convertRetryPolicyFromParsed(p *pRetryPolicy) *ast.RetryPolicyDecl {
	retry := &ast.RetryPolicyDecl{}

	if p.MaxAttempts != nil {
		retry.MaxAttempts = *p.MaxAttempts
	}

	if p.InitialInterval != nil {
		retry.InitialInterval = trimQuotes(*p.InitialInterval)
	}

	if p.BackoffMultiplier != nil {
		retry.BackoffMultiplier = *p.BackoffMultiplier
	}

	return retry
}

// convertJobFromParsed converts a parsed job to an AST node.
func convertJobFromParsed(p *pJobDecl) *ast.JobDecl {
	job := &ast.JobDecl{
		Name: p.Name,
		Task: trimQuotes(p.Task),
	}

	if p.Schedule != nil {
		job.Schedule = trimQuotes(*p.Schedule)
	}

	if p.Queue != nil {
		job.Queue = trimQuotes(*p.Queue)
	}

	if p.Retry != nil {
		job.Retry = convertRetryPolicyFromParsed(p.Retry)
	}

	return job
}

// trimQuotes removes surrounding double quotes from a string.
func trimQuotes(s string) string {
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		return s[1 : len(s)-1]
	}
	return s
}
