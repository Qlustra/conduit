package layout

import (
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

type Reporter interface {
	Record(Entry)
}

type Operation uint8

const (
	OpEnsure Operation = iota + 1
	OpLoad
	OpDiscover
	OpScan
	OpSync
)

func (op Operation) String() string {
	switch op {
	case OpEnsure:
		return "ensure"
	case OpLoad:
		return "load"
	case OpDiscover:
		return "discover"
	case OpScan:
		return "scan"
	case OpSync:
		return "sync"
	default:
		return "unknown"
	}
}

type ResultCode uint8

const (
	EnsureEnsured ResultCode = iota + 1
	EnsureFailed
)

const (
	LoadLoaded ResultCode = iota + 16
	LoadMissing
	LoadTraversed
	LoadNotApplicable
	LoadFailed
)

const (
	DiscoverPresent ResultCode = iota + 32
	DiscoverMissing
	DiscoverTraversed
	DiscoverNotApplicable
	DiscoverFailed
)

const (
	ScanPresent ResultCode = iota + 48
	ScanMissing
	ScanTraversed
	ScanNotApplicable
	ScanFailed
)

const (
	SyncWritten ResultCode = iota + 64
	SyncTraversed
	SyncNotApplicable
	SyncSkippedNoContent
	SyncSkippedPolicy
	SyncFailed
)

type Entry struct {
	Op     Operation
	Path   string
	Result ResultCode
	Err    error
}

func (e Entry) IsError() bool {
	return e.Err != nil
}

func (e Entry) IsSkipped() bool {
	switch e.Result {
	case LoadNotApplicable, DiscoverNotApplicable, ScanNotApplicable, SyncNotApplicable, SyncSkippedNoContent, SyncSkippedPolicy:
		return true
	default:
		return false
	}
}

func (e Entry) IsSuccess() bool {
	return e.Err == nil
}

func (e Entry) ResultName() string {
	switch e.Op {
	case OpEnsure:
		switch e.Result {
		case EnsureEnsured:
			return "ensured"
		case EnsureFailed:
			return "failed"
		}
	case OpLoad:
		switch e.Result {
		case LoadLoaded:
			return "loaded"
		case LoadMissing:
			return "missing"
		case LoadTraversed:
			return "traversed"
		case LoadNotApplicable:
			return "not_applicable"
		case LoadFailed:
			return "failed"
		}
	case OpDiscover:
		switch e.Result {
		case DiscoverPresent:
			return "present"
		case DiscoverMissing:
			return "missing"
		case DiscoverTraversed:
			return "traversed"
		case DiscoverNotApplicable:
			return "not_applicable"
		case DiscoverFailed:
			return "failed"
		}
	case OpScan:
		switch e.Result {
		case ScanPresent:
			return "present"
		case ScanMissing:
			return "missing"
		case ScanTraversed:
			return "traversed"
		case ScanNotApplicable:
			return "not_applicable"
		case ScanFailed:
			return "failed"
		}
	case OpSync:
		switch e.Result {
		case SyncWritten:
			return "written"
		case SyncTraversed:
			return "traversed"
		case SyncNotApplicable:
			return "not_applicable"
		case SyncSkippedNoContent:
			return "skipped_no_content"
		case SyncSkippedPolicy:
			return "skipped_policy"
		case SyncFailed:
			return "failed"
		}
	}

	return "unknown"
}

type Report struct {
	mu      sync.RWMutex
	entries []Entry
}

type reportTreeNode struct {
	name     string
	path     string
	children []*reportTreeNode
	entries  []Entry
	index    map[string]*reportTreeNode
}

func (r *Report) Record(entry Entry) {
	if r == nil {
		return
	}

	r.mu.Lock()
	r.entries = append(r.entries, entry)
	r.mu.Unlock()
}

func (r *Report) Entries() []Entry {
	if r == nil {
		return nil
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	entries := make([]Entry, len(r.entries))
	copy(entries, r.entries)
	return entries
}

func (r *Report) Len() int {
	if r == nil {
		return 0
	}

	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.entries)
}

func (r *Report) HasErrors() bool {
	if r == nil {
		return false
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, entry := range r.entries {
		if entry.IsError() {
			return true
		}
	}

	return false
}

func (r *Report) Filter(keep func(Entry) bool) []Entry {
	if r == nil {
		return nil
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	entries := make([]Entry, 0, len(r.entries))
	for _, entry := range r.entries {
		if keep(entry) {
			entries = append(entries, entry)
		}
	}

	return entries
}

func (r *Report) Sort(less func(a Entry, b Entry) bool) {
	if r == nil {
		return
	}

	r.mu.Lock()
	sort.Slice(r.entries, func(i int, j int) bool {
		return less(r.entries[i], r.entries[j])
	})
	r.mu.Unlock()
}

func (r *Report) SortByPath() {
	r.Sort(func(a Entry, b Entry) bool {
		if a.Path != b.Path {
			return a.Path < b.Path
		}
		if a.Op != b.Op {
			return a.Op < b.Op
		}
		return a.Result < b.Result
	})
}

func (r *Report) RenderTree() string {
	entries := r.Entries()
	if len(entries) == 0 {
		return ""
	}

	sort.Slice(entries, func(i int, j int) bool {
		if entries[i].Path != entries[j].Path {
			return entries[i].Path < entries[j].Path
		}
		if entries[i].Op != entries[j].Op {
			return entries[i].Op < entries[j].Op
		}
		return entries[i].Result < entries[j].Result
	})

	root := &reportTreeNode{index: make(map[string]*reportTreeNode)}

	for _, entry := range entries {
		parts := splitReportPath(entry.Path)
		cur := root
		full := ""
		for _, part := range parts {
			if full == "" {
				full = part
			} else {
				full = filepath.Join(full, part)
			}

			child, ok := cur.index[part]
			if !ok {
				child = &reportTreeNode{name: part, path: full, index: make(map[string]*reportTreeNode)}
				cur.index[part] = child
				cur.children = append(cur.children, child)
			}
			cur = child
		}
		cur.entries = append(cur.entries, entry)
	}

	var lines []string
	renderReportNode(root, "", &lines)
	return strings.Join(lines, "\n")
}

type reportDeepEnsurer interface {
	ensureDeepReport(Context) error
}

type reportDeepLoader interface {
	loadDeepReport(Context) error
}

type reportLoader interface {
	loadReport(Context) error
}

type reportDeepDiscoverer interface {
	discoverDeepReport(Context) error
}

type reportDiscoverer interface {
	discoverReport(Context) error
}

type reportDeepScanner interface {
	scanDeepReport(Context) error
}

type reportScanner interface {
	scanReport(Context) error
}

type reportDeepSyncer interface {
	syncDeepReport(Context) error
}

type reportSyncer interface {
	syncReport(Context) error
}

func reportEnsure(ctx Context, path string, fn func() error) error {
	err := fn()
	result := EnsureEnsured
	if err != nil {
		result = EnsureFailed
	}
	recordEntry(ctx, Entry{Op: OpEnsure, Path: path, Result: result, Err: err})
	return err
}

func reportLoad(ctx Context, path string, fn func() (ResultCode, error)) error {
	result, err := fn()
	if err != nil && result == 0 {
		result = LoadFailed
	}
	recordEntry(ctx, Entry{Op: OpLoad, Path: path, Result: result, Err: err})
	return err
}

func reportDiscover(ctx Context, path string, fn func() (ResultCode, error)) error {
	result, err := fn()
	if err != nil && result == 0 {
		result = DiscoverFailed
	}
	recordEntry(ctx, Entry{Op: OpDiscover, Path: path, Result: result, Err: err})
	return err
}

func reportScan(ctx Context, path string, fn func() (ResultCode, error)) error {
	result, err := fn()
	if err != nil && result == 0 {
		result = ScanFailed
	}
	recordEntry(ctx, Entry{Op: OpScan, Path: path, Result: result, Err: err})
	return err
}

func reportSync(ctx Context, path string, fn func() (ResultCode, error)) error {
	result, err := fn()
	if err != nil && result == 0 {
		result = SyncFailed
	}
	recordEntry(ctx, Entry{Op: OpSync, Path: path, Result: result, Err: err})
	return err
}

func recordEntry(ctx Context, entry Entry) {
	if ctx.Reporter == nil {
		return
	}
	ctx.Reporter.Record(entry)
}

func pathOf(target any) (string, bool) {
	pather, ok := target.(Pather)
	if !ok {
		return "", false
	}
	return pather.Path(), true
}

func resultFromDiskState(present ResultCode, missing ResultCode, fallback ResultCode, state DiskState) ResultCode {
	switch state {
	case DiskPresent:
		return present
	case DiskMissing:
		return missing
	default:
		return fallback
	}
}

func splitReportPath(path string) []string {
	clean := filepath.Clean(path)
	volume := filepath.VolumeName(clean)
	rest := strings.TrimPrefix(clean, volume)
	rest = strings.TrimPrefix(rest, string(filepath.Separator))

	parts := make([]string, 0, strings.Count(rest, string(filepath.Separator))+1)
	if volume != "" {
		parts = append(parts, volume)
	} else if filepath.IsAbs(clean) {
		parts = append(parts, string(filepath.Separator))
	}

	if rest == "" || rest == "." {
		return parts
	}

	for _, part := range strings.Split(rest, string(filepath.Separator)) {
		if part == "" || part == "." {
			continue
		}
		parts = append(parts, part)
	}

	return parts
}

func renderReportNode(n *reportTreeNode, prefix string, lines *[]string) {
	if n == nil {
		return
	}

	sort.Slice(n.children, func(i int, j int) bool {
		return n.children[i].name < n.children[j].name
	})

	for i, child := range n.children {
		lastChild := i == len(n.children)-1
		connector := "|- "
		nextPrefix := prefix + "|  "
		if lastChild {
			connector = "`- "
			nextPrefix = prefix + "   "
		}

		line := prefix + connector + child.name
		for _, entry := range child.entries {
			line += " [" + entry.Op.String() + ":" + entry.ResultName() + "]"
		}
		*lines = append(*lines, line)
		renderReportNode(child, nextPrefix, lines)
	}
}

func (d Dir) ensureDeepReport(ctx Context) error {
	return reportEnsure(ctx, d.Path(), func() error {
		return d.Ensure(ctx)
	})
}

func (d Dir) loadReport(ctx Context) error {
	return reportLoad(ctx, d.Path(), func() (ResultCode, error) {
		return LoadNotApplicable, nil
	})
}

func (d Dir) discoverReport(ctx Context) error {
	return reportDiscover(ctx, d.Path(), func() (ResultCode, error) {
		return DiscoverNotApplicable, nil
	})
}

func (d Dir) scanReport(ctx Context) error {
	return reportScan(ctx, d.Path(), func() (ResultCode, error) {
		return ScanNotApplicable, nil
	})
}

func (d Dir) syncReport(ctx Context) error {
	return reportSync(ctx, d.Path(), func() (ResultCode, error) {
		return SyncNotApplicable, nil
	})
}

func (f File) ensureDeepReport(ctx Context) error {
	return reportEnsure(ctx, f.Path(), func() error {
		return f.Ensure(ctx)
	})
}

func (f File) loadReport(ctx Context) error {
	return reportLoad(ctx, f.Path(), func() (ResultCode, error) {
		return LoadNotApplicable, nil
	})
}

func (f File) discoverReport(ctx Context) error {
	return reportDiscover(ctx, f.Path(), func() (ResultCode, error) {
		return DiscoverNotApplicable, nil
	})
}

func (f File) scanReport(ctx Context) error {
	return reportScan(ctx, f.Path(), func() (ResultCode, error) {
		return ScanNotApplicable, nil
	})
}

func (f File) syncReport(ctx Context) error {
	return reportSync(ctx, f.Path(), func() (ResultCode, error) {
		return SyncNotApplicable, nil
	})
}

func (e Exec) ensureDeepReport(ctx Context) error {
	return reportEnsure(ctx, e.Path(), func() error {
		return e.Ensure(ctx)
	})
}

func (e Exec) loadReport(ctx Context) error {
	return reportLoad(ctx, e.Path(), func() (ResultCode, error) {
		return LoadNotApplicable, nil
	})
}

func (e Exec) discoverReport(ctx Context) error {
	return reportDiscover(ctx, e.Path(), func() (ResultCode, error) {
		return DiscoverNotApplicable, nil
	})
}

func (e Exec) scanReport(ctx Context) error {
	return reportScan(ctx, e.Path(), func() (ResultCode, error) {
		return ScanNotApplicable, nil
	})
}

func (e Exec) syncReport(ctx Context) error {
	return reportSync(ctx, e.Path(), func() (ResultCode, error) {
		return SyncNotApplicable, nil
	})
}

func (f *Format[T, C]) ensureDeepReport(ctx Context) error {
	return reportEnsure(ctx, f.Path(), func() error {
		return f.File.Ensure(ctx)
	})
}

func (f *Format[T, C]) loadReport(ctx Context) error {
	return reportLoad(ctx, f.Path(), func() (ResultCode, error) {
		loaded, err := f.Load()
		if err != nil {
			return LoadFailed, err
		}
		if loaded {
			return LoadLoaded, nil
		}
		return LoadMissing, nil
	})
}

func (f *Format[T, C]) discoverReport(ctx Context) error {
	return reportDiscover(ctx, f.Path(), func() (ResultCode, error) {
		state, err := f.Discover()
		if err != nil {
			return DiscoverFailed, err
		}
		return resultFromDiskState(DiscoverPresent, DiscoverMissing, DiscoverTraversed, state), nil
	})
}

func (f *Format[T, C]) scanReport(ctx Context) error {
	return reportScan(ctx, f.Path(), func() (ResultCode, error) {
		state, err := f.Scan()
		if err != nil {
			return ScanFailed, err
		}
		return resultFromDiskState(ScanPresent, ScanMissing, ScanTraversed, state), nil
	})
}

func (f *Format[T, C]) syncReport(ctx Context) error {
	return reportSync(ctx, f.Path(), func() (ResultCode, error) {
		if f.content == nil {
			return SyncSkippedNoContent, nil
		}
		if !ctx.syncPolicy().allows(f.memory) {
			return SyncSkippedPolicy, nil
		}
		if err := f.saveLoaded(ctx); err != nil {
			return SyncFailed, err
		}
		return SyncWritten, nil
	})
}
