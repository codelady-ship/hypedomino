package adminapi_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"net/http"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/Alievs-corp/hypedomino/server/internal/rbac"
)

const (
	signingIn = "signingIn"
	signedIn  = "signedIn"
	allowed   = "allowed"
)

type route struct {
	method, path string
	guard        string
	object       rbac.Object
	action       rbac.Action
}

var pathValue = regexp.MustCompile(`\{[^}]+\}`)

func TestEveryRouteIsGuardedAndOnlySignInIsOpen(t *testing.T) {
	routes := registeredRoutes(t)
	var open []string
	found := map[string]string{}
	for _, r := range routes {
		found[r.method+" "+r.path] = r.guard
		if r.guard == signingIn {
			open = append(open, r.method+" "+r.path)
		}
	}
	if want := []string{"POST /admin/nonce", "POST /admin/session"}; !slices.Equal(open, want) {
		t.Fatalf("routes open without a session %v, want %v", open, want)
	}
	for _, want := range []string{"PUT /admin/config/rules", "POST /admin/config/liveops/rollback", "POST /admin/audit/export", "POST /admin/approvals/{id}/approve"} {
		if found[want] == "" {
			t.Errorf("the scan missed %s", want)
		}
	}
}

func TestEveryRouteRefusesNoSessionAndEveryRoleWithoutTheGrant(t *testing.T) {
	a := newAPI(t)
	a.create("vusal@example.com", rbac.Support)
	a.create("leyla@example.com", rbac.LiveOps)
	a.create("orxan@example.com", rbac.Analyst)
	sessions := map[rbac.Role]string{
		rbac.SuperAdmin: a.signIn("root"),
		rbac.Content:    a.signIn("ayla"),
		rbac.Support:    a.signIn("vusal"),
		rbac.LiveOps:    a.signIn("leyla"),
		rbac.Analyst:    a.signIn("orxan"),
	}
	refused := 0
	for _, r := range registeredRoutes(t) {
		if r.guard == signingIn {
			continue
		}
		path := pathValue.ReplaceAllString(r.path, "unknown")
		if status, body := a.call(r.method, path, "", map[string]any{}); status != http.StatusUnauthorized || body["error"] != "session_rejected" {
			t.Errorf("%s %s without a session: %d %v", r.method, r.path, status, body)
		}
		if r.guard != allowed {
			continue
		}
		for role, session := range sessions {
			if rbac.Allows([]rbac.Role{role}, r.object, r.action) {
				continue
			}
			refused++
			if status, body := a.call(r.method, path, session, map[string]any{}); status != http.StatusForbidden || body["error"] != "forbidden" {
				t.Errorf("%s %s as %s: %d %v", r.method, r.path, role, status, body)
			}
		}
	}
	if refused == 0 {
		t.Fatal("no role was refused anywhere")
	}
	status, body := a.call("GET", "/admin/audit", sessions[rbac.SuperAdmin], nil)
	if status != http.StatusOK || len(body["entries"].([]any)) != 0 {
		t.Fatalf("refused calls left entries: %d %v", status, body)
	}
}

func registeredRoutes(t *testing.T) []route {
	t.Helper()
	constants := stringConstants(t, "../rbac/rbac.go")
	file, err := parser.ParseFile(token.NewFileSet(), "handler.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	var routes []route
	for _, decl := range file.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok && fn.Recv == nil && fn.Name.Name == "New" {
			for _, stmt := range fn.Body.List {
				routes = append(routes, routesIn(t, stmt, constants)...)
			}
		}
	}
	if len(routes) == 0 {
		t.Fatal("no routes found in handler.go")
	}
	return routes
}

func routesIn(t *testing.T, stmt ast.Stmt, constants map[string]string) []route {
	switch s := stmt.(type) {
	case *ast.ExprStmt:
		if call, ok := s.X.(*ast.CallExpr); ok && isHandleFunc(call) {
			return []route{parseRoute(t, call, nil, constants)}
		}
	case *ast.RangeStmt:
		return loopRoutes(t, s, constants)
	}
	return nil
}

func loopRoutes(t *testing.T, loop *ast.RangeStmt, constants map[string]string) []route {
	literal, isMap := loop.X.(*ast.CompositeLit)
	key, named := loop.Key.(*ast.Ident)
	if !isMap || !named {
		return nil
	}
	var routes []route
	for _, element := range literal.Elts {
		bound := map[string]string{key.Name: stringValue(t, element.(*ast.KeyValueExpr).Key, nil)}
		for _, stmt := range loop.Body.List {
			if expr, ok := stmt.(*ast.ExprStmt); ok {
				if call, ok := expr.X.(*ast.CallExpr); ok && isHandleFunc(call) {
					routes = append(routes, parseRoute(t, call, bound, constants))
				}
			}
		}
	}
	return routes
}

func isHandleFunc(call *ast.CallExpr) bool {
	selector, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || selector.Sel.Name != "HandleFunc" {
		return false
	}
	mux, ok := selector.X.(*ast.SelectorExpr)
	return ok && mux.Sel.Name == "mux"
}

func parseRoute(t *testing.T, call *ast.CallExpr, bound map[string]string, constants map[string]string) route {
	method, path, _ := strings.Cut(stringValue(t, call.Args[0], bound), " ")
	r := route{method: method, path: path}
	guard, ok := call.Args[1].(*ast.CallExpr)
	if !ok {
		t.Fatalf("%s %s is registered without a guard", method, path)
	}
	name, ok := guard.Fun.(*ast.SelectorExpr)
	if !ok || !slices.Contains([]string{signingIn, signedIn, allowed}, name.Sel.Name) {
		t.Fatalf("%s %s is registered without a guard", method, path)
	}
	r.guard = name.Sel.Name
	if r.guard == allowed {
		r.object = rbac.Object(constant(t, guard.Args[0], constants))
		r.action = rbac.Action(constant(t, guard.Args[1], constants))
	}
	return r
}

func stringValue(t *testing.T, expr ast.Expr, bound map[string]string) string {
	switch e := expr.(type) {
	case *ast.BasicLit:
		value, err := strconv.Unquote(e.Value)
		if err != nil {
			t.Fatal(err)
		}
		return value
	case *ast.BinaryExpr:
		return stringValue(t, e.X, bound) + stringValue(t, e.Y, bound)
	case *ast.Ident:
		if value, ok := bound[e.Name]; ok {
			return value
		}
	}
	t.Fatalf("route pattern %T cannot be read", expr)
	return ""
}

func constant(t *testing.T, expr ast.Expr, constants map[string]string) string {
	selector, ok := expr.(*ast.SelectorExpr)
	if !ok {
		t.Fatalf("rbac argument %T is not a constant", expr)
	}
	value, ok := constants[selector.Sel.Name]
	if !ok {
		t.Fatalf("unknown rbac constant %s", selector.Sel.Name)
	}
	return value
}

func stringConstants(t *testing.T, path string) map[string]string {
	t.Helper()
	file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	constants := map[string]string{}
	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.CONST {
			continue
		}
		for _, spec := range gen.Specs {
			value := spec.(*ast.ValueSpec)
			for i, name := range value.Names {
				constants[name.Name] = stringValue(t, value.Values[i], nil)
			}
		}
	}
	return constants
}
