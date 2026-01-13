// Package performance provides performance benchmarks for CodeAI system.
package performance

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bargom/codeai/internal/parser"
	"github.com/bargom/codeai/internal/validator"
)

// =============================================================================
// DSL Generation Helpers
// =============================================================================

// generateCompleteAppDSL generates a complete e-commerce app DSL.
func generateCompleteAppDSL() string {
	return `
config {
    database_type: "postgres"
}

database postgres {
    model User {
        id: uuid, primary, auto
        email: string, required, unique
        name: string, required
        role: string, default("customer")
        created_at: timestamp, auto

        index: [email]
    }

    model Product {
        id: uuid, primary, auto
        name: string, required
        price: decimal, required
        stock: int, default(0)
        created_at: timestamp, auto

        index: [name]
    }

    model Order {
        id: uuid, primary, auto
        user_id: ref(User), required
        total: decimal, required
        status: string, default("pending")
        created_at: timestamp, auto

        index: [user_id]
        index: [status]
    }
}

auth jwt_provider {
    method jwt
    jwks_url "https://auth.example.com/.well-known/jwks.json"
    issuer "https://auth.example.com"
    audience "api.example.com"
}

role admin {
    permissions ["users:*", "products:*", "orders:*"]
}

role customer {
    permissions ["products:read", "orders:create", "orders:read"]
}

middleware auth_required {
    type authentication
    config {
        provider: jwt_provider
        required: true
    }
}

middleware rate_limit {
    type rate_limiting
    config {
        requests: 100
        window: "1m"
        strategy: sliding_window
    }
}

event user_created {
    schema {
        user_id uuid
        email string
        created_at timestamp
    }
}

event order_created {
    schema {
        order_id uuid
        user_id uuid
        total decimal
        created_at timestamp
    }
}

on "user.created" do workflow "send_welcome_email" async
on "order.created" do webhook "notify_shipping"

integration stripe {
    type rest
    base_url "https://api.stripe.com/v1"
    auth bearer {
        token: "$STRIPE_SECRET_KEY"
    }
    timeout "30s"
    circuit_breaker {
        threshold 5
        timeout "60s"
        max_concurrent 100
    }
}

webhook notify_shipping {
    event "order.created"
    url "https://shipping.example.com/webhooks/orders"
    method POST
    headers {
        "Content-Type": "application/json"
    }
    retry 5 initial_interval "1s" backoff 2.0
}
`
}

// generateModelsDSL generates a DSL with multiple models.
func generateModelsDSL(count int) string {
	var sb strings.Builder

	sb.WriteString(`config {
    database_type: "postgres"
}

database postgres {
`)

	for i := 0; i < count; i++ {
		sb.WriteString("    model Model")
		sb.WriteString(itoa(i))
		sb.WriteString(` {
        id: uuid, primary, auto
        name: string, required
        description: text, optional
        status: string, default("active")
        created_at: timestamp, auto
        updated_at: timestamp, auto

        index: [name]
        index: [status]
    }

`)
	}

	sb.WriteString("}\n")
	return sb.String()
}

// generateEventsDSL generates a DSL with multiple events.
func generateEventsDSL(count int) string {
	var sb strings.Builder

	for i := 0; i < count; i++ {
		sb.WriteString("event event_")
		sb.WriteString(itoa(i))
		sb.WriteString(` {
    schema {
        id uuid
        name string
        value decimal
        created_at timestamp
    }
}

`)
	}

	return sb.String()
}

// generateIntegrationsDSL generates a DSL with multiple integrations.
func generateIntegrationsDSL(count int) string {
	var sb strings.Builder

	for i := 0; i < count; i++ {
		sb.WriteString("integration integration_")
		sb.WriteString(itoa(i))
		sb.WriteString(` {
    type rest
    base_url "https://api.example.com/v1"
    auth bearer {
        token: "$API_KEY_`)
		sb.WriteString(itoa(i))
		sb.WriteString(`"
    }
    timeout "30s"
    circuit_breaker {
        threshold 5
        timeout "60s"
        max_concurrent 100
    }
}

`)
	}

	return sb.String()
}

// itoa converts int to string without importing strconv.
func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var result []byte
	for i > 0 {
		result = append([]byte{byte('0' + i%10)}, result...)
		i /= 10
	}
	return string(result)
}

// =============================================================================
// End-to-End Benchmarks
// =============================================================================

// BenchmarkParseCompleteApp benchmarks parsing a complete app DSL.
func BenchmarkParseCompleteApp(b *testing.B) {
	content := generateCompleteAppDSL()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := parser.Parse(content)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkValidateCompleteApp benchmarks validating a complete app DSL.
func BenchmarkValidateCompleteApp(b *testing.B) {
	content := generateCompleteAppDSL()
	program, err := parser.Parse(content)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		v := validator.New()
		if err := v.Validate(program); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkParseAndValidateCompleteApp benchmarks the full parse+validate cycle.
func BenchmarkParseAndValidateCompleteApp(b *testing.B) {
	content := generateCompleteAppDSL()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program, err := parser.Parse(content)
		if err != nil {
			b.Fatal(err)
		}
		v := validator.New()
		if err := v.Validate(program); err != nil {
			b.Fatal(err)
		}
	}
}

// =============================================================================
// Model Scaling Benchmarks
// =============================================================================

// BenchmarkParseModels_10 benchmarks parsing 10 models.
func BenchmarkParseModels_10(b *testing.B) {
	content := generateModelsDSL(10)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := parser.Parse(content)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkParseModels_50 benchmarks parsing 50 models.
func BenchmarkParseModels_50(b *testing.B) {
	content := generateModelsDSL(50)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := parser.Parse(content)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkParseModels_100 benchmarks parsing 100 models.
func BenchmarkParseModels_100(b *testing.B) {
	content := generateModelsDSL(100)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := parser.Parse(content)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkValidateModels_10 benchmarks validating 10 models.
func BenchmarkValidateModels_10(b *testing.B) {
	content := generateModelsDSL(10)
	program, _ := parser.Parse(content)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		v := validator.New()
		v.Validate(program)
	}
}

// BenchmarkValidateModels_50 benchmarks validating 50 models.
func BenchmarkValidateModels_50(b *testing.B) {
	content := generateModelsDSL(50)
	program, _ := parser.Parse(content)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		v := validator.New()
		v.Validate(program)
	}
}

// BenchmarkValidateModels_100 benchmarks validating 100 models.
func BenchmarkValidateModels_100(b *testing.B) {
	content := generateModelsDSL(100)
	program, _ := parser.Parse(content)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		v := validator.New()
		v.Validate(program)
	}
}

// =============================================================================
// Event Scaling Benchmarks
// =============================================================================

// BenchmarkParseEvents_10 benchmarks parsing 10 events.
func BenchmarkParseEvents_10(b *testing.B) {
	content := generateEventsDSL(10)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := parser.Parse(content)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkParseEvents_50 benchmarks parsing 50 events.
func BenchmarkParseEvents_50(b *testing.B) {
	content := generateEventsDSL(50)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := parser.Parse(content)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkParseEvents_100 benchmarks parsing 100 events.
func BenchmarkParseEvents_100(b *testing.B) {
	content := generateEventsDSL(100)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := parser.Parse(content)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// =============================================================================
// Integration Scaling Benchmarks
// =============================================================================

// BenchmarkParseIntegrations_10 benchmarks parsing 10 integrations.
func BenchmarkParseIntegrations_10(b *testing.B) {
	content := generateIntegrationsDSL(10)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := parser.Parse(content)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkParseIntegrations_50 benchmarks parsing 50 integrations.
func BenchmarkParseIntegrations_50(b *testing.B) {
	content := generateIntegrationsDSL(50)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := parser.Parse(content)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// =============================================================================
// File Parsing Benchmarks
// =============================================================================

// BenchmarkParseCompleteAppFile benchmarks parsing the complete_app.cai file.
func BenchmarkParseCompleteAppFile(b *testing.B) {
	// Find the complete_app.cai file
	cwd, err := os.Getwd()
	if err != nil {
		b.Skip("Could not get working directory")
	}

	var examplesPath string
	for {
		testPath := filepath.Join(cwd, "examples", "complete_app.cai")
		if _, err := os.Stat(testPath); err == nil {
			examplesPath = testPath
			break
		}
		parent := filepath.Dir(cwd)
		if parent == cwd {
			b.Skip("Could not find examples/complete_app.cai")
		}
		cwd = parent
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := parser.ParseFile(examplesPath)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkValidateCompleteAppFile benchmarks validating the complete_app.cai file.
func BenchmarkValidateCompleteAppFile(b *testing.B) {
	cwd, err := os.Getwd()
	if err != nil {
		b.Skip("Could not get working directory")
	}

	var examplesPath string
	for {
		testPath := filepath.Join(cwd, "examples", "complete_app.cai")
		if _, err := os.Stat(testPath); err == nil {
			examplesPath = testPath
			break
		}
		parent := filepath.Dir(cwd)
		if parent == cwd {
			b.Skip("Could not find examples/complete_app.cai")
		}
		cwd = parent
	}

	program, err := parser.ParseFile(examplesPath)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		v := validator.New()
		v.Validate(program)
	}
}

// =============================================================================
// Memory Allocation Benchmarks
// =============================================================================

// BenchmarkParseCompleteApp_Allocs reports memory allocations.
func BenchmarkParseCompleteApp_Allocs(b *testing.B) {
	content := generateCompleteAppDSL()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := parser.Parse(content)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkValidateCompleteApp_Allocs reports memory allocations for validation.
func BenchmarkValidateCompleteApp_Allocs(b *testing.B) {
	content := generateCompleteAppDSL()
	program, _ := parser.Parse(content)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		v := validator.New()
		v.Validate(program)
	}
}
