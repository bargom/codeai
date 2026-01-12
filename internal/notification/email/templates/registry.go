// Package templates provides email template management.
package templates

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"sync"
	texttemplate "text/template"
)

//go:embed html/*.html text/*.txt
var templateFS embed.FS

// TemplateType represents the type of an email template.
type TemplateType string

// Template types for email notifications.
const (
	TemplateWorkflowCompleted  TemplateType = "workflow_completed"
	TemplateWorkflowFailed     TemplateType = "workflow_failed"
	TemplateJobCompleted       TemplateType = "job_completed"
	TemplateJobFailed          TemplateType = "job_failed"
	TemplateTestResultsSummary TemplateType = "test_results"
	TemplateWelcome            TemplateType = "welcome"
)

// Template holds information about an email template.
type Template struct {
	Type        TemplateType
	Subject     string
	HTMLPath    string
	TextPath    string
	BrevoID     int64 // Template ID in Brevo dashboard (0 means use local template)
}

// Registry manages email templates.
type Registry struct {
	templates    map[TemplateType]*Template
	htmlCache    map[TemplateType]*template.Template
	textCache    map[TemplateType]*texttemplate.Template
	subjectCache map[TemplateType]*texttemplate.Template
	mu           sync.RWMutex
}

// NewRegistry creates a new template registry with default templates.
func NewRegistry() *Registry {
	r := &Registry{
		templates:    make(map[TemplateType]*Template),
		htmlCache:    make(map[TemplateType]*template.Template),
		textCache:    make(map[TemplateType]*texttemplate.Template),
		subjectCache: make(map[TemplateType]*texttemplate.Template),
	}

	// Register default templates
	r.registerDefaults()

	return r
}

// registerDefaults registers the default email templates.
func (r *Registry) registerDefaults() {
	r.templates[TemplateWorkflowCompleted] = &Template{
		Type:     TemplateWorkflowCompleted,
		Subject:  "Workflow Completed: {{.WorkflowID}}",
		HTMLPath: "html/workflow_completed.html",
		TextPath: "text/workflow_completed.txt",
	}

	r.templates[TemplateWorkflowFailed] = &Template{
		Type:     TemplateWorkflowFailed,
		Subject:  "Workflow Failed: {{.WorkflowID}}",
		HTMLPath: "html/workflow_failed.html",
		TextPath: "text/workflow_failed.txt",
	}

	r.templates[TemplateJobCompleted] = &Template{
		Type:     TemplateJobCompleted,
		Subject:  "Job Completed: {{.JobID}}",
		HTMLPath: "html/job_completed.html",
		TextPath: "text/job_completed.txt",
	}

	r.templates[TemplateJobFailed] = &Template{
		Type:     TemplateJobFailed,
		Subject:  "Job Failed: {{.JobID}}",
		HTMLPath: "html/job_failed.html",
		TextPath: "text/job_failed.txt",
	}

	r.templates[TemplateTestResultsSummary] = &Template{
		Type:     TemplateTestResultsSummary,
		Subject:  "Test Results: {{.TestRunID}}",
		HTMLPath: "html/test_results.html",
		TextPath: "text/test_results.txt",
	}

	r.templates[TemplateWelcome] = &Template{
		Type:     TemplateWelcome,
		Subject:  "Welcome to CodeAI",
		HTMLPath: "html/welcome.html",
		TextPath: "text/welcome.txt",
	}
}

// GetTemplate returns the template for the given type.
func (r *Registry) GetTemplate(templateType TemplateType) (*Template, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tmpl, ok := r.templates[templateType]
	if !ok {
		return nil, fmt.Errorf("template not found: %s", templateType)
	}

	return tmpl, nil
}

// RenderTemplate renders an HTML template with the given data.
func (r *Registry) RenderTemplate(tmpl *Template, data map[string]interface{}) (string, error) {
	r.mu.Lock()
	htmlTmpl, ok := r.htmlCache[tmpl.Type]
	if !ok {
		content, err := templateFS.ReadFile(tmpl.HTMLPath)
		if err != nil {
			r.mu.Unlock()
			return "", fmt.Errorf("read template: %w", err)
		}

		htmlTmpl, err = template.New(string(tmpl.Type)).Parse(string(content))
		if err != nil {
			r.mu.Unlock()
			return "", fmt.Errorf("parse template: %w", err)
		}

		r.htmlCache[tmpl.Type] = htmlTmpl
	}
	r.mu.Unlock()

	var buf bytes.Buffer
	if err := htmlTmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("execute template: %w", err)
	}

	return buf.String(), nil
}

// RenderTextTemplate renders a plain text template with the given data.
func (r *Registry) RenderTextTemplate(tmpl *Template, data map[string]interface{}) (string, error) {
	r.mu.Lock()
	textTmpl, ok := r.textCache[tmpl.Type]
	if !ok {
		content, err := templateFS.ReadFile(tmpl.TextPath)
		if err != nil {
			r.mu.Unlock()
			// Text template is optional
			return "", nil
		}

		textTmpl, err = texttemplate.New(string(tmpl.Type)).Parse(string(content))
		if err != nil {
			r.mu.Unlock()
			return "", fmt.Errorf("parse text template: %w", err)
		}

		r.textCache[tmpl.Type] = textTmpl
	}
	r.mu.Unlock()

	var buf bytes.Buffer
	if err := textTmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("execute text template: %w", err)
	}

	return buf.String(), nil
}

// RenderSubject renders the subject line template with the given data.
func (r *Registry) RenderSubject(tmpl *Template, data map[string]interface{}) (string, error) {
	r.mu.Lock()
	subjectTmpl, ok := r.subjectCache[tmpl.Type]
	if !ok {
		var err error
		subjectTmpl, err = texttemplate.New(string(tmpl.Type) + "_subject").Parse(tmpl.Subject)
		if err != nil {
			r.mu.Unlock()
			return "", fmt.Errorf("parse subject template: %w", err)
		}

		r.subjectCache[tmpl.Type] = subjectTmpl
	}
	r.mu.Unlock()

	var buf bytes.Buffer
	if err := subjectTmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("execute subject template: %w", err)
	}

	return buf.String(), nil
}

// RegisterTemplate registers a custom template.
func (r *Registry) RegisterTemplate(tmpl *Template) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.templates[tmpl.Type] = tmpl
}

// ListTemplates returns all registered templates.
func (r *Registry) ListTemplates() []*Template {
	r.mu.RLock()
	defer r.mu.RUnlock()

	templates := make([]*Template, 0, len(r.templates))
	for _, tmpl := range r.templates {
		templates = append(templates, tmpl)
	}

	return templates
}
