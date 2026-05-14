package layout

import (
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// Entry

// Entry records one path-level outcome from a deep traversal operation.
type Entry struct {
	Op     Operation
	Path   string
	Result ResultCode
	Err    error
}

// IsError reports whether the entry carries an error.
func (e Entry) IsError() bool {
	return e.Err != nil
}

// IsSkipped reports whether the result represents a visited-but-not-applied
// outcome.
func (e Entry) IsSkipped() bool {
	switch e.Result {
	case LoadNotApplicable, DiscoverNotApplicable, ScanNotApplicable, SyncNotApplicable, SyncSkippedNoContent, SyncSkippedPolicy, ValidateNotApplicable:
		return true
	default:
		return false
	}
}

// IsSuccess reports whether the entry completed without error.
func (e Entry) IsSuccess() bool {
	return e.Err == nil
}

// ResultName returns a stable lowercase name for the result code relative to
// the entry's operation.
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
	case OpValidate:
		switch e.Result {
		case ValidateOK:
			return "ok"
		case ValidateTraversed:
			return "traversed"
		case ValidateNotApplicable:
			return "not_applicable"
		case ValidateFailed:
			return "failed"
		}
	}

	return "unknown"
}

// Report

// Report collects Entry values produced during deep traversal.
//
// A Report is safe for concurrent Record calls.
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

// Record appends one entry to the report.
func (r *Report) Record(entry Entry) {
	if r == nil {
		return
	}

	r.mu.Lock()
	r.entries = append(r.entries, entry)
	r.mu.Unlock()
}

// Entries returns a snapshot copy of the currently recorded entries.
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

// Len returns the number of recorded entries.
func (r *Report) Len() int {
	if r == nil {
		return 0
	}

	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.entries)
}

// HasErrors reports whether any recorded entry carries an error.
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

// Filter returns a snapshot of recorded entries for which keep returns true.
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

// Sort reorders the recorded entries in place using less.
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

// SortByPath sorts the recorded entries by path, then operation, then result.
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

// RenderTree renders the recorded entries as a path-oriented tree.
//
// Entries are sorted by path, operation, and result before rendering.
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
	renderRoot := reportRenderRoot(root)
	if renderRoot == root {
		renderReportNode(root, "", &lines)
	} else {
		renderReportNode(&reportTreeNode{children: []*reportTreeNode{renderRoot}}, "", &lines)
	}
	return strings.Join(lines, "\n")
}

// Helpers

func recordResult(ctx Context, op Operation, path string, result ResultCode, err error) (ResultCode, error) {
	return recordResultWithReporter(ctx.Reporter, op, path, result, err)
}

func recordValidateResult(opts ValidateOptions, op Operation, path string, result ResultCode, err error) (ResultCode, error) {
	return recordResultWithReporter(opts.Reporter, op, path, result, err)
}

func recordResultWithReporter(reporter Reporter, op Operation, path string, result ResultCode, err error) (ResultCode, error) {
	recordEntryWithReporter(reporter, Entry{
		Op:     op,
		Path:   path,
		Result: result,
		Err:    err,
	})
	return result, err
}

func recordEntryWithReporter(reporter Reporter, entry Entry) {
	if reporter == nil {
		return
	}
	reporter.Record(entry)
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

func resultForDiscoverFromScanResult(result ResultCode, err error) ResultCode {
	if err != nil {
		return DiscoverFailed
	}

	switch result {
	case ScanPresent:
		return DiscoverPresent
	case ScanMissing:
		return DiscoverMissing
	case ScanTraversed:
		return DiscoverTraversed
	case ScanNotApplicable:
		return DiscoverNotApplicable
	default:
		return DiscoverTraversed
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

func reportRenderRoot(root *reportTreeNode) *reportTreeNode {
	if root == nil {
		return nil
	}

	cur := root
	for len(cur.entries) == 0 && len(cur.children) == 1 && len(cur.children[0].entries) == 0 {
		cur = cur.children[0]
	}

	return cur
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
