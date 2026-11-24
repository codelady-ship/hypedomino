package contract

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

var forbiddenTokens = map[string]bool{
	"balance": true, "balances": true, "coin": true, "coins": true, "crystal": true, "crystals": true,
	"price": true, "prices": true, "cost": true, "costs": true, "amount": true, "amounts": true,
	"reward": true, "rewards": true, "score": true, "scores": true, "points": true,
	"rating": true, "ratings": true, "mmr": true, "lp": true, "xp": true, "rank": true, "tier": true,
	"inventory": true, "owned": true, "result": true, "results": true,
}

var staffPackages = map[string]bool{"adminapi": true}

var wordBoundary = regexp.MustCompile(`[A-Z]?[a-z0-9]+|[A-Z]+(?:[A-Z][a-z]|$)`)

type intentField struct {
	position token.Position
	request  string
	name     string
}

func TestPlayerRequestsCarryIntentsOnly(t *testing.T) {
	fields := requestFields(t, "..")
	requests := map[string]bool{}
	for _, field := range fields {
		requests[field.request] = true
		for _, word := range words(field.name) {
			if forbiddenTokens[word] {
				t.Errorf("%s: player request %s accepts %q from the client", field.position, field.request, field.name)
			}
		}
	}
	for _, want := range []string{"shop.intent", "pass.levelsRequest", "purchase.completeRequest", "purchase.appleNotice"} {
		if !requests[want] {
			t.Errorf("the scan missed %s; found %v", want, keys(requests))
		}
	}
}

func TestTheScanFindsValuesATamperedRequestWouldAccept(t *testing.T) {
	var found []string
	for _, field := range requestFields(t, "testdata") {
		for _, word := range words(field.name) {
			if forbiddenTokens[word] {
				found = append(found, field.request+" "+field.name)
				break
			}
		}
	}
	want := []string{"tampered.buyRequest price_coins", "tampered.buyRequest rewardXp", "tampered.claim Balance"}
	if !reflect.DeepEqual(found, want) {
		t.Fatalf("found %v, want %v", found, want)
	}
}

func TestTheWordSplitFindsForbiddenTokensInEveryNamingStyle(t *testing.T) {
	cases := map[string][]string{
		"price_coins":   {"price", "coins"},
		"PriceCrystals": {"price", "crystals"},
		"rewardXP":      {"reward", "xp"},
		"request_id":    {"request", "id"},
		"SignedPayload": {"signed", "payload"},
		"opensToPity":   {"opens", "to", "pity"},
	}
	for name, want := range cases {
		if got := words(name); !reflect.DeepEqual(got, want) {
			t.Errorf("%s: %v, want %v", name, got, want)
		}
	}
}

func words(name string) []string {
	var out []string
	for _, part := range wordBoundary.FindAllString(name, -1) {
		out = append(out, strings.ToLower(part))
	}
	return out
}

func keys(set map[string]bool) []string {
	out := make([]string, 0, len(set))
	for key := range set {
		out = append(out, key)
	}
	return out
}

func requestFields(t *testing.T, root string) []intentField {
	t.Helper()
	fset := token.NewFileSet()
	var fields []intentField
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil || !entry.IsDir() {
			return err
		}
		if staffPackages[entry.Name()] || entry.Name() == "pb" || (entry.Name() == "testdata" && path != root) {
			return filepath.SkipDir
		}
		packages, err := parser.ParseDir(fset, path, func(info os.FileInfo) bool { return !strings.HasSuffix(info.Name(), "_test.go") }, parser.ParseComments)
		if err != nil {
			return err
		}
		for name, pkg := range packages {
			fields = append(fields, packageRequestFields(fset, name, pkg)...)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return fields
}

func packageRequestFields(fset *token.FileSet, name string, pkg *ast.Package) []intentField {
	structs := map[string]*ast.StructType{}
	var handlers []*ast.FuncDecl
	for _, file := range pkg.Files {
		for _, decl := range file.Decls {
			switch d := decl.(type) {
			case *ast.GenDecl:
				for _, spec := range d.Specs {
					if ts, ok := spec.(*ast.TypeSpec); ok {
						if st, ok := ts.Type.(*ast.StructType); ok {
							structs[ts.Name.Name] = st
						}
					}
				}
			case *ast.FuncDecl:
				if d.Body != nil && takesRequest(d.Type) {
					handlers = append(handlers, d)
				}
			}
		}
	}
	var fields []intentField
	for _, handler := range handlers {
		for _, target := range decodeTargets(handler) {
			st := target.inline
			label := name + "." + handler.Name.Name
			if st == nil {
				st, label = structs[target.name], name+"."+target.name
			}
			if st != nil {
				fields = append(fields, structFields(fset, label, st, structs, map[string]bool{target.name: true})...)
			}
		}
	}
	return fields
}

func takesRequest(fn *ast.FuncType) bool {
	for _, param := range fn.Params.List {
		star, ok := param.Type.(*ast.StarExpr)
		if !ok {
			continue
		}
		if sel, ok := star.X.(*ast.SelectorExpr); ok && sel.Sel.Name == "Request" {
			if pkg, ok := sel.X.(*ast.Ident); ok && pkg.Name == "http" {
				return true
			}
		}
	}
	return false
}

type requestType struct {
	name   string
	inline *ast.StructType
}

func decodeTargets(handler *ast.FuncDecl) []requestType {
	declared := map[string]requestType{}
	var decoded []string
	ast.Inspect(handler.Body, func(node ast.Node) bool {
		switch n := node.(type) {
		case *ast.ValueSpec:
			for _, name := range n.Names {
				switch typ := n.Type.(type) {
				case *ast.Ident:
					declared[name.Name] = requestType{name: typ.Name}
				case *ast.StructType:
					declared[name.Name] = requestType{inline: typ}
				}
			}
		case *ast.AssignStmt:
			for i, rhs := range n.Rhs {
				if lit, ok := rhs.(*ast.CompositeLit); ok && i < len(n.Lhs) {
					if ident, ok := lit.Type.(*ast.Ident); ok {
						declared[n.Lhs[i].(*ast.Ident).Name] = requestType{name: ident.Name}
					}
				}
			}
		case *ast.CallExpr:
			if decodes(n.Fun) {
				for _, arg := range n.Args {
					if ref, ok := arg.(*ast.UnaryExpr); ok && ref.Op == token.AND {
						if ident, ok := ref.X.(*ast.Ident); ok {
							decoded = append(decoded, ident.Name)
						}
					}
				}
			}
		}
		return true
	})
	types := make([]requestType, 0, len(decoded))
	for _, variable := range decoded {
		if typ, ok := declared[variable]; ok {
			types = append(types, typ)
		}
	}
	return types
}

func decodes(fun ast.Expr) bool {
	switch f := fun.(type) {
	case *ast.SelectorExpr:
		return f.Sel.Name == "Decode"
	case *ast.Ident:
		return strings.HasPrefix(f.Name, "decode")
	}
	return false
}

func structFields(fset *token.FileSet, request string, st *ast.StructType, structs map[string]*ast.StructType, seen map[string]bool) []intentField {
	var fields []intentField
	for _, field := range st.Fields.List {
		if nested, ok := field.Type.(*ast.Ident); ok && structs[nested.Name] != nil && !seen[nested.Name] {
			seen[nested.Name] = true
			fields = append(fields, structFields(fset, request, structs[nested.Name], structs, seen)...)
		}
		for _, name := range field.Names {
			fields = append(fields, intentField{position: fset.Position(name.Pos()), request: request, name: jsonName(field, name.Name)})
		}
	}
	return fields
}

func jsonName(field *ast.Field, goName string) string {
	if field.Tag == nil {
		return goName
	}
	tag, err := strconv.Unquote(field.Tag.Value)
	if err != nil {
		return goName
	}
	name, _, _ := strings.Cut(reflect.StructTag(tag).Get("json"), ",")
	if name == "" || name == "-" {
		return goName
	}
	return name
}
