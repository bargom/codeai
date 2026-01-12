package definitions

// Activity names for registration.
const (
	ValidateInputActivityName              = "ValidateInput"
	ValidateTestSuiteActivityName          = "ValidateTestSuite"
	ExecuteAgentActivityName               = "ExecuteAgent"
	ExecuteTestCaseActivityName            = "ExecuteTestCase"
	StorePipelineResultActivityName        = "StorePipelineResult"
	StoreTestSuiteResultActivityName       = "StoreTestSuiteResult"
	SendNotificationActivityName           = "SendNotification"
	RollbackAgentExecutionActivityName     = "RollbackAgentExecution"
	RollbackFileOperationActivityName      = "RollbackFileOperation"
	RollbackExternalAPICallActivityName    = "RollbackExternalAPICall"
	RollbackNotificationActivityName       = "RollbackNotification"
	RollbackDatabaseOperationActivityName  = "RollbackDatabaseOperation"
)

// ValidationResult holds the result of input validation.
type ValidationResult struct {
	Valid bool   `json:"valid"`
	Error string `json:"error,omitempty"`
}

// ValidateInputRequest is the input for ValidateInputActivity.
type ValidateInputRequest struct {
	Input PipelineInput `json:"input"`
}

// ValidateTestSuiteRequest is the input for ValidateTestSuiteActivity.
type ValidateTestSuiteRequest struct {
	Input TestSuiteInput `json:"input"`
}

// ExecuteAgentRequest is the input for ExecuteAgentActivity.
type ExecuteAgentRequest struct {
	Agent AgentConfig `json:"agent"`
}

// ExecuteAgentResponse is the output from ExecuteAgentActivity.
type ExecuteAgentResponse struct {
	Result AgentResult `json:"result"`
}

// ExecuteTestCaseRequest is the input for ExecuteTestCaseActivity.
type ExecuteTestCaseRequest struct {
	TestCase TestCase `json:"testCase"`
}

// ExecuteTestCaseResponse is the output from ExecuteTestCaseActivity.
type ExecuteTestCaseResponse struct {
	Result TestCaseResult `json:"result"`
}

// StorePipelineResultRequest is the input for StorePipelineResultActivity.
type StorePipelineResultRequest struct {
	WorkflowID string         `json:"workflowId"`
	Output     PipelineOutput `json:"output"`
}

// StoreTestSuiteResultRequest is the input for StoreTestSuiteResultActivity.
type StoreTestSuiteResultRequest struct {
	SuiteID string          `json:"suiteId"`
	Output  TestSuiteOutput `json:"output"`
}

// StorageResult is the output from storage activities.
type StorageResult struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

// NotificationRequest is the input for SendNotificationActivity.
type NotificationRequest struct {
	Type       string            `json:"type"`
	WorkflowID string            `json:"workflowId"`
	Status     Status            `json:"status"`
	Message    string            `json:"message"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

// NotificationResult is the output from SendNotificationActivity.
type NotificationResult struct {
	Sent  bool   `json:"sent"`
	Error string `json:"error,omitempty"`
}

// SendNotificationRequest is the input for SendNotificationActivity.
type SendNotificationRequest struct {
	WorkflowID string `json:"workflowId"`
	Type       string `json:"type"`
	Message    string `json:"message"`
}

// Activity function type declarations for registration.
var (
	ValidateInputActivity             func(ValidateInputRequest) (ValidationResult, error)
	ValidateTestSuiteActivity         func(ValidateTestSuiteRequest) (ValidationResult, error)
	ExecuteAgentActivity              func(ExecuteAgentRequest) (ExecuteAgentResponse, error)
	ExecuteTestCaseActivity           func(ExecuteTestCaseRequest) (ExecuteTestCaseResponse, error)
	StorePipelineResultActivity       func(StorePipelineResultRequest) (StorageResult, error)
	StoreTestSuiteResultActivity      func(StoreTestSuiteResultRequest) (StorageResult, error)
	SendNotificationActivity          func(NotificationRequest) (NotificationResult, error)
	RollbackAgentExecutionActivity    interface{}
	RollbackFileOperationActivity     interface{}
	RollbackExternalAPICallActivity   interface{}
	RollbackNotificationActivity      interface{}
	RollbackDatabaseOperationActivity interface{}
)
