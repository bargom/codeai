package templates

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegistry_GetTemplate(t *testing.T) {
	registry := NewRegistry()

	tests := []struct {
		name         string
		templateType TemplateType
		wantErr      bool
	}{
		{
			name:         "workflow completed template",
			templateType: TemplateWorkflowCompleted,
			wantErr:      false,
		},
		{
			name:         "workflow failed template",
			templateType: TemplateWorkflowFailed,
			wantErr:      false,
		},
		{
			name:         "job completed template",
			templateType: TemplateJobCompleted,
			wantErr:      false,
		},
		{
			name:         "test results template",
			templateType: TemplateTestResultsSummary,
			wantErr:      false,
		},
		{
			name:         "welcome template",
			templateType: TemplateWelcome,
			wantErr:      false,
		},
		{
			name:         "unknown template",
			templateType: "unknown",
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpl, err := registry.GetTemplate(tt.templateType)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, tmpl)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, tmpl)
				assert.Equal(t, tt.templateType, tmpl.Type)
			}
		})
	}
}

func TestRegistry_RenderTemplate(t *testing.T) {
	registry := NewRegistry()

	tmpl, err := registry.GetTemplate(TemplateWorkflowCompleted)
	require.NoError(t, err)

	data := map[string]interface{}{
		"WorkflowID":   "wf-123",
		"Duration":     "5m30s",
		"Timestamp":    "2024-01-15T10:30:00Z",
		"DashboardURL": "/workflows/wf-123",
	}

	html, err := registry.RenderTemplate(tmpl, data)
	require.NoError(t, err)

	assert.Contains(t, html, "wf-123")
	assert.Contains(t, html, "Workflow Completed")
	assert.Contains(t, html, "/workflows/wf-123")
}

func TestRegistry_RenderTextTemplate(t *testing.T) {
	registry := NewRegistry()

	tmpl, err := registry.GetTemplate(TemplateWorkflowCompleted)
	require.NoError(t, err)

	data := map[string]interface{}{
		"WorkflowID":   "wf-123",
		"Duration":     "5m30s",
		"Timestamp":    "2024-01-15T10:30:00Z",
		"DashboardURL": "/workflows/wf-123",
	}

	text, err := registry.RenderTextTemplate(tmpl, data)
	require.NoError(t, err)

	assert.Contains(t, text, "wf-123")
	assert.Contains(t, text, "WORKFLOW COMPLETED")
}

func TestRegistry_RenderSubject(t *testing.T) {
	registry := NewRegistry()

	tmpl, err := registry.GetTemplate(TemplateWorkflowCompleted)
	require.NoError(t, err)

	data := map[string]interface{}{
		"WorkflowID": "wf-123",
	}

	subject, err := registry.RenderSubject(tmpl, data)
	require.NoError(t, err)

	assert.Equal(t, "Workflow Completed: wf-123", subject)
}

func TestRegistry_ListTemplates(t *testing.T) {
	registry := NewRegistry()

	templates := registry.ListTemplates()

	// Should have all default templates
	assert.GreaterOrEqual(t, len(templates), 6)

	// Check that all expected types are present
	types := make(map[TemplateType]bool)
	for _, tmpl := range templates {
		types[tmpl.Type] = true
	}

	assert.True(t, types[TemplateWorkflowCompleted])
	assert.True(t, types[TemplateWorkflowFailed])
	assert.True(t, types[TemplateJobCompleted])
	assert.True(t, types[TemplateJobFailed])
	assert.True(t, types[TemplateTestResultsSummary])
	assert.True(t, types[TemplateWelcome])
}

func TestRegistry_RegisterTemplate(t *testing.T) {
	registry := NewRegistry()

	customType := TemplateType("custom_notification")
	customTmpl := &Template{
		Type:    customType,
		Subject: "Custom: {{.ID}}",
	}

	registry.RegisterTemplate(customTmpl)

	retrieved, err := registry.GetTemplate(customType)
	require.NoError(t, err)
	assert.Equal(t, customType, retrieved.Type)
}
