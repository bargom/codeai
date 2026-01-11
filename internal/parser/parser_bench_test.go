package parser

import (
	"strings"
	"testing"
)

// smallProgram is a small DSL program (~10 lines) for benchmarking.
var smallProgram = `
var name = "test"
var count = 42

function greet(who) {
	var msg = "Hello"
	exec { echo $msg }
}

if active {
	var x = 1
} else {
	var y = 2
}
`

// generateLargeProgram generates a large DSL program with n statements.
func generateLargeProgram(n int) string {
	var b strings.Builder
	for i := 0; i < n; i++ {
		switch i % 5 {
		case 0:
			b.WriteString(`var x` + string(rune('0'+i%10)) + ` = "value"` + "\n")
		case 1:
			b.WriteString(`var count` + string(rune('0'+i%10)) + ` = ` + string(rune('0'+i%10)) + "\n")
		case 2:
			b.WriteString("if active {\n  var temp = 1\n}\n")
		case 3:
			b.WriteString("for item in items {\n  var result = item\n}\n")
		case 4:
			b.WriteString("exec { echo hello }\n")
		}
	}
	return b.String()
}

func BenchmarkParseSmallProgram(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := Parse(smallProgram)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkParseLargeProgram(b *testing.B) {
	largeProgram := generateLargeProgram(200) // ~1000 lines
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := Parse(largeProgram)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkParseVarDecl(b *testing.B) {
	input := `var x = "hello world"`
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := Parse(input)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkParseIfStmt(b *testing.B) {
	input := `if active { var x = 1 } else { var y = 2 }`
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := Parse(input)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkParseForLoop(b *testing.B) {
	input := `for item in items { var result = item }`
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := Parse(input)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkParseFunctionDecl(b *testing.B) {
	input := `function greet(name, age) { var msg = "Hello" }`
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := Parse(input)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkParseExecBlock(b *testing.B) {
	input := `exec { kubectl get pods | grep Running | wc -l }`
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := Parse(input)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkParseArray(b *testing.B) {
	input := `var arr = [1, 2, 3, 4, 5, 6, 7, 8, 9, 10]`
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := Parse(input)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkParseNestedStructures(b *testing.B) {
	input := `
if outer {
	if middle {
		if inner {
			var x = 1
		}
	}
}
`
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := Parse(input)
		if err != nil {
			b.Fatal(err)
		}
	}
}
