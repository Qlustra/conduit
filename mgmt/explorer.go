package mgmt

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/qlustra/conduit/layout"
)

// Explorer observes an arbitrary directory tree without a layout declaration or
// project identity. It never writes, and does not follow symbolic links.
//
// Observations are not filesystem snapshots: another process may change an entry
// between inspection and reading. Explorer does not promise descriptor-relative
// traversal or protection against adversarial filesystem replacement races.
type Explorer struct {
	root      layout.Dir
	selection ExploreSelection
}

// ExplorerOptions supplies the default selection for both operations.
type ExplorerOptions struct {
	Selection ExploreSelection
}

// ExploreSelection separates inclusion in the result from directory traversal.
// The zero value selects all descendants and traverses every directory.
type ExploreSelection struct {
	// Select decides which entries appear in the result. Excluding a directory
	// does not exclude its descendants. The root itself is never selected.
	Select func(ExploreEntry) bool
	// Prune prevents descent into a directory, independently of Select. It also
	// receives the root (RelativePath ".", Depth 0). Pruned directories may
	// still appear in the result if selected.
	Prune func(ExploreEntry) bool
	// MaxDepth limits descendants: 1 means immediate children; 0 is unlimited.
	MaxDepth int
}

// ExploreDiagnosticPolicy controls whether independent failures stop traversal.
type ExploreDiagnosticPolicy uint8

const (
	// ExploreCollectDiagnostics continues with independently accessible entries.
	// Any diagnostics still produce a non-nil, joined operation error.
	ExploreCollectDiagnostics ExploreDiagnosticPolicy = iota
	ExploreFailFast
)

// ExploreSymlinkPolicy controls reporting of symbolic links. Neither policy
// traverses a link or gathers its target's content.
type ExploreSymlinkPolicy uint8

const (
	// ExploreSkipSymlinks inventories selected links but does not follow them.
	ExploreSkipSymlinks ExploreSymlinkPolicy = iota
	// ExploreRejectSymlinks additionally reports each encountered link as an
	// error, even when that link is not selected for the inventory.
	ExploreRejectSymlinks
)

// ScanPolicy controls one metadata-only traversal.
type ScanPolicy struct {
	// Selection, when non-nil, replaces the complete default selection.
	Selection   *ExploreSelection
	Diagnostics ExploreDiagnosticPolicy
	Symlinks    ExploreSymlinkPolicy
}

// GatherPolicy applies the same traversal as Scan, then reads each selected
// regular file as it is encountered. Directories, links, and special files may
// appear in Entries but never in Files.
type GatherPolicy struct {
	ScanPolicy
	// MaxFileBytes bounds each file's content; 0 is unlimited. A file exceeding
	// the limit is diagnosed and omitted, rather than silently truncated.
	MaxFileBytes int64
	// MaxTotalBytes bounds the total content returned in Files; 0 is unlimited.
	// With collection enabled, a file that does not fit is diagnosed and later
	// smaller files may still be gathered within the remaining budget.
	MaxTotalBytes int64
}

// ExploreEntry describes an observation made using Lstat semantics.
type ExploreEntry struct {
	Path         string // Absolute filesystem path.
	RelativePath string // Slash-separated path relative to Root.
	Depth        int
	Mode         fs.FileMode
	Size         int64
	ModTime      time.Time
}

func (e ExploreEntry) IsDir() bool     { return e.Mode.IsDir() }
func (e ExploreEntry) IsRegular() bool { return e.Mode.IsRegular() }

// ExploreDiagnostic preserves the location and underlying error of a failure.
type ExploreDiagnostic struct {
	Path         string
	RelativePath string
	Operation    string
	Err          error
}

func (d ExploreDiagnostic) Error() string {
	return fmt.Sprintf("explorer %s %q: %v", d.Operation, d.Path, d.Err)
}

func (d ExploreDiagnostic) Unwrap() error { return d.Err }

// Exploration is a deterministic, depth-first inventory in lexical sibling order.
// Entries exclude the root. Partial entries and diagnostics remain available on
// error. Complete means the requested traversal finished without errors; depth
// limits, pruning, and deliberately skipped links do not make it incomplete.
type Exploration struct {
	Root        string
	Entries     []ExploreEntry
	Diagnostics []ExploreDiagnostic
	Complete    bool
}

// GatheredFile pairs an inventory entry with complete, untruncated file bytes.
type GatheredFile struct {
	Entry   ExploreEntry
	Content []byte
}

// Gathering retains selected metadata even when an entry's content could not be
// gathered. Files contains only successful reads, in the same order as Entries.
type Gathering struct {
	Exploration
	Files []GatheredFile
}

var (
	ErrExploreSymlink = errors.New("symbolic link is not allowed")
	ErrExploreLimit   = errors.New("gather content limit exceeded")
)

// NewExplorer binds a chosen root without reading or creating it. Scan and
// Gather diagnose an absent root, a non-directory root, or a symbolic-link root.
func NewExplorer(root string, options ExplorerOptions) (*Explorer, error) {
	if root == "" {
		return nil, errors.New("explorer root must not be empty")
	}
	if options.Selection.MaxDepth < 0 {
		return nil, errors.New("explorer maximum depth must not be negative")
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("explorer root: %w", err)
	}
	return &Explorer{root: layout.NewDir(abs), selection: options.Selection}, nil
}

// Root returns the bound absolute directory path.
func (e *Explorer) Root() layout.Dir { return e.root }

// Scan inspects metadata without opening file content or populating layout state.
func (e *Explorer) Scan(ctx context.Context, policy ScanPolicy) (Exploration, error) {
	return e.walk(ctx, policy, nil)
}

// Gather reads only selected regular files. Cancellation is checked before each
// traversal step and read; it cannot interrupt an operating-system read already
// in progress. Neither content decoding nor management binding is implicit.
func (e *Explorer) Gather(ctx context.Context, policy GatherPolicy) (Gathering, error) {
	result := Gathering{Exploration: Exploration{Root: e.root.Path()}}
	if policy.MaxFileBytes < 0 || policy.MaxTotalBytes < 0 {
		return result, errors.New("explorer gather limits must not be negative")
	}
	var total int64
	inventory, err := e.walk(ctx, policy.ScanPolicy, func(entry ExploreEntry) error {
		if !entry.IsRegular() {
			return nil
		}
		limit, bounded := policy.MaxFileBytes, policy.MaxFileBytes > 0
		if policy.MaxTotalBytes > 0 {
			remaining := policy.MaxTotalBytes - total
			if !bounded || remaining < limit {
				limit = remaining
			}
			bounded = true
		}
		content, err := readExploreFile(ctx, entry, limit, bounded)
		if err != nil {
			return err
		}
		total += int64(len(content))
		result.Files = append(result.Files, GatheredFile{Entry: entry, Content: content})
		return nil
	})
	result.Exploration = inventory
	return result, err
}

func (e *Explorer) walk(ctx context.Context, policy ScanPolicy, consume func(ExploreEntry) error) (Exploration, error) {
	result := Exploration{Root: e.root.Path()}
	selection := e.selection
	if policy.Selection != nil {
		selection = *policy.Selection
	}
	if selection.MaxDepth < 0 {
		return result, errors.New("explorer maximum depth must not be negative")
	}
	if policy.Diagnostics > ExploreFailFast {
		return result, errors.New("invalid explorer diagnostic policy")
	}
	if policy.Symlinks > ExploreRejectSymlinks {
		return result, errors.New("invalid explorer symbolic-link policy")
	}

	record := func(path, relative, operation string, err error) error {
		diagnostic := ExploreDiagnostic{Path: path, RelativePath: relative, Operation: operation, Err: err}
		result.Diagnostics = append(result.Diagnostics, diagnostic)
		if policy.Diagnostics == ExploreFailFast || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return diagnostic
		}
		return nil
	}
	// WalkDir uses Lstat at the root, does not follow links, and visits children
	// lexically. It can continue into siblings after a failed directory read.
	walkErr := filepath.WalkDir(e.root.Path(), func(path string, dirent fs.DirEntry, walkErr error) error {
		relative, err := filepath.Rel(e.root.Path(), path)
		if err != nil {
			return record(path, "", "scan", err)
		}
		relative = filepath.ToSlash(relative)
		if err := ctx.Err(); err != nil {
			return record(path, relative, "scan", err)
		}
		if walkErr != nil {
			return record(path, relative, "scan", walkErr)
		}
		info, err := dirent.Info()
		if err != nil {
			if err := record(path, relative, "scan", err); err != nil {
				return err
			}
			if dirent.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		entry := ExploreEntry{Path: path, RelativePath: relative, Mode: info.Mode(), Size: info.Size(), ModTime: info.ModTime()}
		if relative != "." {
			entry.Depth = strings.Count(relative, "/") + 1
		} else if !entry.IsDir() {
			rootErr := fmt.Errorf("root is not a directory")
			if entry.Mode&fs.ModeSymlink != 0 {
				rootErr = ErrExploreSymlink
			}
			return record(path, relative, "scan", rootErr)
		}

		selected := entry.Depth > 0 && (selection.Select == nil || selection.Select(entry))
		if selected {
			result.Entries = append(result.Entries, entry)
		}
		if entry.Mode&fs.ModeSymlink != 0 && policy.Symlinks == ExploreRejectSymlinks {
			if err := record(path, relative, "scan", ErrExploreSymlink); err != nil {
				return err
			}
		}
		if selected && consume != nil {
			if err := consume(entry); err != nil {
				if err := record(path, relative, "gather", err); err != nil {
					return err
				}
			}
		}
		if entry.IsDir() && ((selection.MaxDepth > 0 && entry.Depth >= selection.MaxDepth) || (selection.Prune != nil && selection.Prune(entry))) {
			return filepath.SkipDir
		}
		return nil
	})
	// A selector may cancel the context at the last entry, with no further walk
	// callback to observe it. Record that cancellation once before reporting.
	if walkErr == nil && ctx.Err() != nil {
		_ = record(e.root.Path(), ".", "scan", ctx.Err())
	}
	if len(result.Diagnostics) > 0 {
		errs := make([]error, len(result.Diagnostics))
		for i := range result.Diagnostics {
			errs[i] = result.Diagnostics[i]
		}
		return result, errors.Join(errs...)
	}
	if walkErr != nil {
		return result, walkErr
	}
	result.Complete = true
	return result, nil
}

func readExploreFile(ctx context.Context, entry ExploreEntry, limit int64, bounded bool) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if bounded && entry.Size > limit {
		return nil, fmt.Errorf("%w: file size %d exceeds remaining allowance %d", ErrExploreLimit, entry.Size, limit)
	}
	// Recheck the node before opening: it may have changed since inventory.
	// This is a best-effort observation check, not a race-free confinement claim.
	info, err := os.Lstat(entry.Path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, errors.New("selected entry is no longer a regular file")
	}
	file, err := os.Open(entry.Path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	info, err = file.Stat()
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, errors.New("selected entry is no longer a regular file")
	}
	var reader io.Reader = exploreContextReader{ctx: ctx, reader: file}
	// Read one extra byte to distinguish a file exactly at the bound from one
	// that grew after inspection. Avoid overflow for an explicit MaxInt64 bound.
	if bounded && limit < int64(^uint64(0)>>1) {
		reader = io.LimitReader(reader, limit+1)
	}
	content, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	if bounded && int64(len(content)) > limit {
		return nil, fmt.Errorf("%w: file content exceeds remaining allowance %d", ErrExploreLimit, limit)
	}
	return content, nil
}

type exploreContextReader struct {
	ctx    context.Context
	reader io.Reader
}

func (r exploreContextReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.reader.Read(p)
}
