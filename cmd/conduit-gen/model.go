package main

import (
	"fmt"
	"go/token"
	"go/types"
	"reflect"
	"sort"
	"strconv"
	"strings"
)

const modulePath = "github.com/qlustra/conduit"

type topology struct {
	name, generated string
	layout          types.Type
	root            *field
	fields          []*field
}

type field struct {
	name, kind, from, bind, generated string
	value, bound                      types.Type
	topology                          *topology
	owner                             *field
	tags                              reflect.StructTag
	ops                               []string
	creation, bootstrap               *recipe
}

type recipe struct {
	name  string
	input types.Type
}

type model struct {
	pkg    *types.Package
	types  map[string]*topology
	active map[string]bool
}

func (m *model) topology(name string) (*topology, error) {
	if m.active[name] {
		return nil, fmt.Errorf("recursive management topology through %s is unsupported", name)
	}
	if t := m.types[name]; t != nil {
		return t, nil
	}
	obj := m.pkg.Scope().Lookup(name)
	if _, ok := obj.(*types.TypeName); !ok {
		return nil, fmt.Errorf("%s is not a type in package %s", name, m.pkg.Name())
	}
	named, ok := types.Unalias(obj.Type()).(*types.Named)
	if !ok || named.TypeParams().Len() != 0 {
		return nil, fmt.Errorf("%s must be a non-generic named topology struct", name)
	}
	st, ok := named.Underlying().(*types.Struct)
	if !ok {
		return nil, fmt.Errorf("%s must be a topology struct", name)
	}
	t := &topology{name: name, generated: generatedName(name)}
	m.active[name] = true
	defer delete(m.active, name)
	for i := 0; i < st.NumFields(); i++ {
		v := st.Field(i)
		if !v.Exported() || v.Embedded() {
			return nil, fmt.Errorf("%s.%s: topology fields must be named and exported", name, v.Name())
		}
		f := &field{name: v.Name(), tags: reflect.StructTag(st.Tag(i))}
		if err := validateTopologyTags(string(f.tags)); err != nil {
			return nil, fmt.Errorf("%s.%s: %w", name, f.name, err)
		}
		var err error
		f.from, f.bind, err = bindingTag(f.tags.Get("scope"))
		if err != nil {
			return nil, fmt.Errorf("%s.%s: %w", name, f.name, err)
		}
		f.kind, f.value = tokenKind(v.Type())
		if f.kind == "" {
			n, ok := types.Unalias(v.Type()).(*types.Named)
			if !ok || n.Obj().Pkg() != m.pkg {
				return nil, fmt.Errorf("%s.%s: expected spec token or local topology struct", name, f.name)
			}
			f.kind = "Topology"
			f.topology, err = m.topology(n.Obj().Name())
			if err != nil {
				return nil, fmt.Errorf("%s.%s: %w", name, f.name, err)
			}
			f.value = f.topology.layout
		}
		if f.kind == "Collection" {
			n, ok := types.Unalias(f.value).(*types.Named)
			if !ok || n.Obj().Pkg() != m.pkg {
				return nil, fmt.Errorf("%s.%s: Collection argument must be a local topology struct", name, f.name)
			}
			f.topology, err = m.topology(n.Obj().Name())
			if err != nil {
				return nil, fmt.Errorf("%s.%s: %w", name, f.name, err)
			}
			if f.topology.root.kind != "Scope" {
				return nil, fmt.Errorf("%s.%s: collection members currently require Scope-root topology", name, f.name)
			}
			f.value = f.topology.layout
		}
		if (f.kind == "Project" || f.kind == "Space" || f.kind == "Scope") && f.from == "" && (f.bind == "" || f.bind == ".") {
			if t.root != nil {
				return nil, fmt.Errorf("%s: multiple root declarations (%s and %s); give child bindings an explicit from", name, t.root.name, f.name)
			}
			t.root, t.layout, f.bind = f, f.value, "."
		}
		t.fields = append(t.fields, f)
	}
	if t.root == nil {
		return nil, fmt.Errorf("%s: exactly one root spec.Project, spec.Space, or spec.Scope is required", name)
	}
	if _, pointer := types.Unalias(t.layout).(*types.Pointer); pointer {
		return nil, fmt.Errorf("%s: root layout type must be a struct, not a pointer", name)
	}
	if _, ok := t.layout.Underlying().(*types.Struct); !ok {
		return nil, fmt.Errorf("%s: root layout type must be a struct", name)
	}
	byName := map[string]*field{}
	for _, f := range t.fields {
		byName[f.name] = f
		f.generated = t.generated + f.name + f.kind
	}
	if err := m.resolveField(t, t.root, byName, map[*field]bool{}); err != nil {
		return nil, err
	}
	for _, f := range t.fields {
		if err := m.resolveField(t, f, byName, map[*field]bool{}); err != nil {
			return nil, err
		}
	}
	// Stable dependency order allows from=OtherScope without declaration-order rules.
	var ordered []*field
	seen := map[*field]bool{}
	var add func(*field)
	add = func(f *field) {
		if seen[f] {
			return
		}
		if f.owner != nil {
			add(f.owner)
		}
		seen[f] = true
		ordered = append(ordered, f)
	}
	for _, f := range t.fields {
		add(f)
	}
	t.fields = ordered
	m.types[name] = t
	return t, nil
}

func (m *model) resolveField(t *topology, f *field, fields map[string]*field, active map[*field]bool) error {
	if f.bound != nil {
		return nil
	}
	fail := func(format string, args ...any) error {
		return fmt.Errorf("%s.%s: %s", t.name, f.name, fmt.Sprintf(format, args...))
	}
	if active[f] {
		return fail("cyclic from binding")
	}
	active[f] = true
	defer delete(active, f)
	if f == t.root {
		f.bound = f.value
	} else {
		if f.from == "" {
			f.from = t.root.name
		}
		f.owner = fields[f.from]
		if f.owner == nil {
			return fail("unknown from field %q", f.from)
		}
		if f.owner.kind == "Collection" || f.owner.kind == "Document" {
			return fail("from=%s is not a layout-bearing handle", f.from)
		}
		if err := m.resolveField(t, f.owner, fields, active); err != nil {
			return err
		}
		if f.bind == "" {
			return fail("non-root declaration requires bind=LayoutField or bind=.")
		}
		var err error
		f.bound, err = boundType(f.owner.value, f.bind)
		if err != nil {
			return fail("%v", err)
		}
		if f.kind == "Collection" {
			n, ok := types.Unalias(deref(f.bound)).(*types.Named)
			if !ok || packagePath(n.Obj()) != modulePath+"/layout" || n.Obj().Name() != "Slot" || n.TypeArgs().Len() != 1 {
				return fail("collection must bind layout.Slot[*ChildLayout]")
			}
			p, ok := types.Unalias(n.TypeArgs().At(0)).(*types.Pointer)
			if !ok || !types.Identical(p.Elem(), f.value) {
				return fail("slot child type does not match %s root layout", f.topology.name)
			}
		} else if f.kind == "Document" {
			if !documentType(f.bound, f.value) {
				return fail("document binding must expose Read() (%s, error) and the supported typed content contract", f.value)
			}
		} else if !types.Identical(deref(f.bound), f.value) {
			return fail("bound type %s does not match declared layout %s", f.bound, f.value)
		}
		if f.kind == "Project" || f.kind == "Topology" && f.topology.root.kind == "Project" {
			return fail("nested Project generation is not supported; bind a Project explicitly with mgmt.BindProject")
		}
		if f.kind == "Space" || f.kind == "Topology" && f.topology.root.kind == "Space" {
			if _, err := layoutRoot(f.value); err != nil {
				return fail("child Space: %v", err)
			}
		}
	}
	for _, name := range []string{"create", "bootstrap"} {
		value, present := f.tags.Lookup(name)
		if !present {
			continue
		}
		if value == "" {
			return fail("%s binding cannot be empty", name)
		}
		if name == "create" && f.kind != "Collection" || name == "bootstrap" && f.kind != "Scope" && f.kind != "Space" && f.kind != "Project" {
			return fail("%s binding is not applicable to %s", name, f.kind)
		}
		r, err := m.recipe(value, f.value, name)
		if err != nil {
			return fail("%v", err)
		}
		if name == "create" {
			f.creation = r
		} else {
			f.bootstrap = r
		}
	}
	ops, err := operations(f)
	if err != nil {
		return fail("%v", err)
	}
	f.ops = ops
	return nil
}

func (m *model) recipe(name string, layoutType types.Type, operation string) (*recipe, error) {
	if !token.IsIdentifier(name) {
		return nil, fmt.Errorf("%s must name a local variable of type mgmt.Creation or mgmt.Initialization", operation)
	}
	obj, ok := m.pkg.Scope().Lookup(name).(*types.Var)
	if !ok {
		return nil, fmt.Errorf("%s binding %q must name a local variable", operation, name)
	}
	named, ok := types.Unalias(obj.Type()).(*types.Named)
	want := "Creation"
	if operation == "bootstrap" {
		want = "Initialization"
	}
	if !ok || packagePath(named.Obj()) != modulePath+"/mgmt" || named.Obj().Name() != want || named.TypeArgs().Len() != 2 {
		return nil, fmt.Errorf("%s binding %s must have type mgmt.%s[Layout, Input]", operation, name, want)
	}
	if !types.Identical(named.TypeArgs().At(0), layoutType) {
		return nil, fmt.Errorf("%s binding %s has mismatched layout type", operation, name)
	}
	return &recipe{name: name, input: named.TypeArgs().At(1)}, nil
}

func tokenKind(t types.Type) (string, types.Type) {
	named, ok := types.Unalias(t).(*types.Named)
	if !ok || packagePath(named.Obj()) != modulePath+"/spec" || named.TypeArgs().Len() != 1 {
		return "", nil
	}
	switch name := named.Obj().Name(); name {
	case "Project", "Space", "Scope", "Document", "Collection":
		return name, named.TypeArgs().At(0)
	}
	return "", nil
}

func packagePath(obj *types.TypeName) string {
	if obj.Pkg() == nil {
		return ""
	}
	return obj.Pkg().Path()
}

func deref(t types.Type) types.Type {
	if p, ok := types.Unalias(t).(*types.Pointer); ok {
		return p.Elem()
	}
	return t
}

func generatedName(name string) string {
	if strings.HasSuffix(name, "Management") && name != "Management" {
		return strings.TrimSuffix(name, "Management")
	}
	return name + "Controller"
}

func bindingTag(tag string) (from, bind string, err error) {
	if tag == "" {
		return
	}
	seen := map[string]bool{}
	for _, part := range strings.Split(tag, ";") {
		pair := strings.SplitN(strings.TrimSpace(part), "=", 2)
		if len(pair) != 2 || strings.TrimSpace(pair[1]) == "" {
			return "", "", fmt.Errorf("invalid scope term %q", part)
		}
		key, value := strings.TrimSpace(pair[0]), strings.TrimSpace(pair[1])
		if seen[key] {
			return "", "", fmt.Errorf("duplicate scope term %q", key)
		}
		seen[key] = true
		switch key {
		case "from":
			from = value
		case "bind":
			bind = value
		default:
			return "", "", fmt.Errorf("unknown scope term %q", key)
		}
	}
	return
}

func boundType(root types.Type, path string) (types.Type, error) {
	if path == "." {
		return root, nil
	}
	t := root
	for _, part := range strings.Split(path, ".") {
		if !token.IsIdentifier(part) || !token.IsExported(part) {
			return nil, fmt.Errorf("bind path %q must use exported Go field names", path)
		}
		st, ok := deref(t).Underlying().(*types.Struct)
		if !ok {
			return nil, fmt.Errorf("bind path %q crosses a non-struct", path)
		}
		var found types.Type
		for i := 0; i < st.NumFields(); i++ {
			if st.Field(i).Name() == part {
				found = st.Field(i).Type()
				break
			}
		}
		if found == nil {
			return nil, fmt.Errorf("bind path %q: field %s does not exist", path, part)
		}
		t = found
	}
	return t, nil
}

func layoutRoot(t types.Type) (string, error) {
	st, ok := deref(t).Underlying().(*types.Struct)
	if !ok {
		return "", fmt.Errorf("layout must have an exported layout.Dir tagged layout:\".\"")
	}
	for i := 0; i < st.NumFields(); i++ {
		n, ok := types.Unalias(st.Field(i).Type()).(*types.Named)
		if ok && n.Obj().Name() == "Dir" && packagePath(n.Obj()) == modulePath+"/layout" && reflect.StructTag(st.Tag(i)).Get("layout") == "." && st.Field(i).Exported() {
			return st.Field(i).Name(), nil
		}
	}
	return "", fmt.Errorf("layout must have an exported layout.Dir tagged layout:\".\"")
}

func documentType(node, value types.Type) bool {
	ms := types.NewMethodSet(types.NewPointer(deref(node)))
	for _, name := range []string{"Path", "Read", "Set", "Get", "EncodeValue", "DecodeValue", "HasContent", "DiskState", "MemoryState", "PrepareWrite", "SnapshotState", "Load", "Unload"} {
		sel := ms.Lookup(nil, name)
		if sel == nil {
			return false
		}
		if name == "Read" {
			sig := sel.Obj().Type().(*types.Signature)
			if sig.Params().Len() != 0 || sig.Results().Len() != 2 || !types.Identical(sig.Results().At(0).Type(), value) {
				return false
			}
		}
	}
	return true
}

func operations(f *field) ([]string, error) {
	var available []string
	switch f.kind {
	case "Project", "Space", "Scope":
		available = []string{"inspect", "validate", "scaffold", "bootstrap"}
		if hasMethod(f.value, "Load", map[types.Type]bool{}) {
			available = append(available, "load")
		}
		if hasMethod(f.value, "PrepareWrite", map[types.Type]bool{}) {
			available = append(available, "save")
		}
		if hasDerived(f.value, map[types.Type]bool{}) {
			available = append(available, "refresh")
		}
		if f.kind == "Project" {
			available = append(available, "detect")
		}
	case "Document":
		available = []string{"read", "load", "save", "update", "validate"}
	case "Collection":
		available = []string{"at", "list"}
		if f.creation != nil {
			available = append(available, "create")
		}
	case "Topology":
		if _, ok := f.tags.Lookup("ops"); ok {
			return nil, fmt.Errorf("ops on nested topology is unsupported; restrict its root declaration")
		}
		return nil, nil
	}
	if value, explicit := f.tags.Lookup("ops"); explicit {
		var selected []string
		seen := map[string]bool{}
		if value != "" {
			for _, op := range strings.Split(value, ",") {
				op = strings.TrimSpace(op)
				found := false
				for _, supported := range available {
					if supported == op {
						found = true
						break
					}
				}
				if !found {
					return nil, fmt.Errorf("operation %q is not applicable; supported operations: %s", op, strings.Join(available, ", "))
				}
				if seen[op] {
					return nil, fmt.Errorf("duplicate operation %q", op)
				}
				seen[op] = true
				selected = append(selected, op)
			}
		}
		available = selected
	}
	sort.Strings(available)
	return available, nil
}

func hasMethod(t types.Type, method string, seen map[types.Type]bool) bool {
	t = deref(t)
	if seen[t] {
		return false
	}
	seen[t] = true
	if types.NewMethodSet(types.NewPointer(t)).Lookup(nil, method) != nil {
		return true
	}
	if n, ok := types.Unalias(t).(*types.Named); ok && packagePath(n.Obj()) == modulePath+"/layout" && n.Obj().Name() == "Slot" {
		return hasMethod(n.TypeArgs().At(0), method, seen)
	}
	if st, ok := t.Underlying().(*types.Struct); ok {
		for i := 0; i < st.NumFields(); i++ {
			if st.Field(i).Exported() && hasMethod(st.Field(i).Type(), method, seen) {
				return true
			}
		}
	}
	return false
}

func hasDerived(t types.Type, seen map[types.Type]bool) bool {
	t = deref(t)
	if seen[t] {
		return false
	}
	seen[t] = true
	if n, ok := types.Unalias(t).(*types.Named); ok && packagePath(n.Obj()) == modulePath+"/layout" && n.Obj().Name() == "Slot" {
		return hasDerived(n.TypeArgs().At(0), seen)
	}
	if st, ok := t.Underlying().(*types.Struct); ok {
		for i := 0; i < st.NumFields(); i++ {
			if !st.Field(i).Exported() {
				continue
			}
			for _, term := range strings.FieldsFunc(reflect.StructTag(st.Tag(i)).Get("manage"), func(r rune) bool { return r == ',' || r == ';' }) {
				if strings.TrimSpace(term) == "role=derived" && hasRenderContract(st.Field(i).Type()) {
					return true
				}
			}
			if hasDerived(st.Field(i).Type(), seen) {
				return true
			}
		}
	}
	return false
}

// Topology tags are the generator's entire declaration surface. Reject unknown
// and malformed tags so an intended operation binding cannot silently vanish.
func validateTopologyTags(tag string) error {
	seen := map[string]bool{}
	for strings.TrimSpace(tag) != "" {
		tag = strings.TrimLeft(tag, " ")
		end := strings.IndexByte(tag, ':')
		if end < 1 || end+1 >= len(tag) || tag[end+1] != '"' {
			return fmt.Errorf("malformed topology tag %q", tag)
		}
		key := tag[:end]
		if strings.ContainsAny(key, " \t\n\r\"") {
			return fmt.Errorf("malformed topology tag key %q", key)
		}
		tag = tag[end+1:]
		quotedEnd := 1
		for quotedEnd < len(tag) {
			if tag[quotedEnd] == '\\' {
				quotedEnd += 2
				continue
			}
			if tag[quotedEnd] == '"' {
				break
			}
			quotedEnd++
		}
		if quotedEnd >= len(tag) {
			return fmt.Errorf("unterminated topology tag %s", key)
		}
		if _, err := strconv.Unquote(tag[:quotedEnd+1]); err != nil {
			return fmt.Errorf("invalid topology tag %s: %w", key, err)
		}
		tag = tag[quotedEnd+1:]
		if seen[key] {
			return fmt.Errorf("duplicate topology tag %s", key)
		}
		seen[key] = true
		switch key {
		case "scope", "ops", "create", "bootstrap":
		default:
			return fmt.Errorf("unsupported %s topology tag; use supported declaration tags or runtime options", key)
		}
	}
	return nil
}

// Match the same two leaf rendering contracts supported by mgmt.Refresh.
// TextTemplate supplies RenderTemplate and SetRendered; a wrapper may supply
// Template alone, without having a Render method.
func hasRenderContract(t types.Type) bool {
	methods := types.NewMethodSet(types.NewPointer(deref(t)))
	textType := types.Typ[types.String]
	errorType := types.Universe.Lookup("error").Type()
	matches := func(name string, params, results []types.Type) bool {
		selected := methods.Lookup(nil, name)
		if selected == nil {
			return false
		}
		signature, ok := selected.Type().(*types.Signature)
		if !ok || signature.Variadic() || signature.Params().Len() != len(params) || signature.Results().Len() != len(results) {
			return false
		}
		for i, want := range params {
			if !types.Identical(signature.Params().At(i).Type(), want) {
				return false
			}
		}
		for i, want := range results {
			if !types.Identical(signature.Results().At(i).Type(), want) {
				return false
			}
		}
		return true
	}
	if !matches("SetRendered", []types.Type{textType}, nil) {
		return false
	}
	if matches("Render", nil, []types.Type{textType, errorType}) {
		return true
	}
	return matches("Template", nil, []types.Type{textType}) && matches("RenderTemplate", []types.Type{textType}, []types.Type{textType, errorType})
}
