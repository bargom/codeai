package validator

import "fmt"

// SymbolKind distinguishes between different kinds of symbols.
type SymbolKind int

const (
	// SymbolVariable represents a variable declaration.
	SymbolVariable SymbolKind = iota
	// SymbolFunction represents a function declaration.
	SymbolFunction
	// SymbolParameter represents a function parameter.
	SymbolParameter
)

// symbolKindNames maps SymbolKind to string representations.
var symbolKindNames = map[SymbolKind]string{
	SymbolVariable:  "variable",
	SymbolFunction:  "function",
	SymbolParameter: "parameter",
}

// String returns the string representation of SymbolKind.
func (sk SymbolKind) String() string {
	if name, ok := symbolKindNames[sk]; ok {
		return name
	}
	return fmt.Sprintf("Unknown(%d)", sk)
}

// Symbol represents a declared identifier in the program.
type Symbol struct {
	Name       string
	Kind       SymbolKind
	Type       Type       // Inferred type
	ParamCount int        // For functions: number of parameters
}

// scope represents a single lexical scope with its symbol mappings.
type scope struct {
	symbols map[string]*Symbol
	parent  *scope
}

// newScope creates a new scope with optional parent.
func newScope(parent *scope) *scope {
	return &scope{
		symbols: make(map[string]*Symbol),
		parent:  parent,
	}
}

// SymbolTable manages symbol declarations across nested scopes.
type SymbolTable struct {
	current *scope
	global  *scope // Reference to global scope for builtins
}

// NewSymbolTable creates a new symbol table with global scope initialized.
func NewSymbolTable() *SymbolTable {
	global := newScope(nil)
	st := &SymbolTable{
		current: global,
		global:  global,
	}
	// Register builtin functions
	st.registerBuiltins()
	return st
}

// registerBuiltins adds built-in functions to the global scope.
func (st *SymbolTable) registerBuiltins() {
	// print(value) - prints a value
	st.global.symbols["print"] = &Symbol{
		Name:       "print",
		Kind:       SymbolFunction,
		ParamCount: 1,
	}

	// len(value) - returns length of array or string
	st.global.symbols["len"] = &Symbol{
		Name:       "len",
		Kind:       SymbolFunction,
		ParamCount: 1,
	}
}

// EnterScope creates and enters a new nested scope.
func (st *SymbolTable) EnterScope() {
	st.current = newScope(st.current)
}

// ExitScope exits the current scope and returns to the parent.
func (st *SymbolTable) ExitScope() {
	if st.current.parent != nil {
		st.current = st.current.parent
	}
}

// Declare adds a new symbol to the current scope.
// Returns an error if the symbol is already declared in the current scope.
func (st *SymbolTable) Declare(name string, kind SymbolKind) error {
	if _, exists := st.current.symbols[name]; exists {
		return fmt.Errorf("duplicate declaration '%s'", name)
	}
	st.current.symbols[name] = &Symbol{
		Name: name,
		Kind: kind,
		Type: TypeUnknown,
	}
	return nil
}

// DeclareWithType adds a new symbol with a known type.
func (st *SymbolTable) DeclareWithType(name string, kind SymbolKind, typ Type) error {
	if _, exists := st.current.symbols[name]; exists {
		return fmt.Errorf("duplicate declaration '%s'", name)
	}
	st.current.symbols[name] = &Symbol{
		Name: name,
		Kind: kind,
		Type: typ,
	}
	return nil
}

// DeclareFunction adds a function symbol with parameter count.
func (st *SymbolTable) DeclareFunction(name string, paramCount int) error {
	if _, exists := st.current.symbols[name]; exists {
		return fmt.Errorf("duplicate declaration '%s'", name)
	}
	st.current.symbols[name] = &Symbol{
		Name:       name,
		Kind:       SymbolFunction,
		ParamCount: paramCount,
		Type:       TypeFunction,
	}
	return nil
}

// Lookup searches for a symbol starting from the current scope up to global.
// Returns the symbol and true if found, nil and false otherwise.
func (st *SymbolTable) Lookup(name string) (*Symbol, bool) {
	for scope := st.current; scope != nil; scope = scope.parent {
		if sym, exists := scope.symbols[name]; exists {
			return sym, true
		}
	}
	return nil, false
}

// LookupLocal searches for a symbol only in the current scope.
// Used for checking duplicate declarations in the same scope.
func (st *SymbolTable) LookupLocal(name string) (*Symbol, bool) {
	sym, exists := st.current.symbols[name]
	return sym, exists
}

// UpdateType updates the type of an existing symbol.
func (st *SymbolTable) UpdateType(name string, typ Type) {
	if sym, ok := st.Lookup(name); ok {
		sym.Type = typ
	}
}

// CurrentScopeDepth returns the nesting depth of the current scope.
// Useful for debugging scope issues.
func (st *SymbolTable) CurrentScopeDepth() int {
	depth := 0
	for scope := st.current; scope != nil; scope = scope.parent {
		depth++
	}
	return depth
}
