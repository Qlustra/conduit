package layout

import (
	"crypto/sha256"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFileFingerprintScanMetadataRecordsCurrentOnly(t *testing.T) {
	file := NewFile(filepath.Join(t.TempDir(), "payload.txt"))
	mtime := time.Unix(1_700_000_100, 0)
	if err := os.WriteFile(file.Path(), []byte("payload"), 0o644); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}
	if err := file.Chtimes(mtime, mtime, DefaultContext); err != nil {
		t.Fatalf("Chtimes() error = %v", err)
	}

	fp := file.Fingerprint()
	state, err := fp.ScanMetadata()
	if err != nil {
		t.Fatalf("ScanMetadata() error = %v", err)
	}
	if state != DiskPresent {
		t.Fatalf("ScanMetadata() state = %v, want %v", state, DiskPresent)
	}
	if got := fp.File(); got.Path() != file.Path() {
		t.Fatalf("File().Path() = %q, want %q", got.Path(), file.Path())
	}
	if got := fp.DiskState(); got != DiskPresent {
		t.Fatalf("DiskState() = %v, want %v", got, DiskPresent)
	}
	if got := fp.PreviousDiskState(); got != DiskUnknown {
		t.Fatalf("PreviousDiskState() = %v, want %v", got, DiskUnknown)
	}
	if !fp.HasKnownDiskState() {
		t.Fatal("HasKnownDiskState() = false, want true")
	}
	if fp.HasPreviousKnownDiskState() {
		t.Fatal("HasPreviousKnownDiskState() = true, want false")
	}
	if !fp.WasObservedOnDisk() {
		t.Fatal("WasObservedOnDisk() = false, want true")
	}
	if fp.WasPreviouslyObservedOnDisk() {
		t.Fatal("WasPreviouslyObservedOnDisk() = true, want false")
	}
	if got := fp.Size(); got != int64(len("payload")) {
		t.Fatalf("Size() = %d, want %d", got, len("payload"))
	}
	if got := fp.PreviousSize(); got != 0 {
		t.Fatalf("PreviousSize() = %d, want 0", got)
	}
	if got := fp.ModTime(); !got.Equal(mtime) {
		t.Fatalf("ModTime() = %v, want %v", got, mtime)
	}
	if got := fp.PreviousModTime(); !got.IsZero() {
		t.Fatalf("PreviousModTime() = %v, want zero time", got)
	}
	if got := fp.Checksum(); got != "" {
		t.Fatalf("Checksum() = %q, want empty", got)
	}
	if got := fp.PreviousChecksum(); got != "" {
		t.Fatalf("PreviousChecksum() = %q, want empty", got)
	}
	if fp.ChangedMetadata() {
		t.Fatal("ChangedMetadata() = true, want false after first scan")
	}
	if fp.ChangedContent() {
		t.Fatal("ChangedContent() = true, want false without content scans")
	}
	if fp.Changed() {
		t.Fatal("Changed() = true, want false after first scan")
	}
	if fp.PresenceChanged() {
		t.Fatal("PresenceChanged() = true, want false after first scan")
	}
}

func TestFileFingerprintScanTracksPresenceAndContentChange(t *testing.T) {
	file := NewFile(filepath.Join(t.TempDir(), "payload.txt"))
	fp := file.Fingerprint()

	state, err := fp.Scan()
	if err != nil {
		t.Fatalf("first Scan() error = %v", err)
	}
	if state != DiskMissing {
		t.Fatalf("first Scan() state = %v, want %v", state, DiskMissing)
	}
	if got := fp.DiskState(); got != DiskMissing {
		t.Fatalf("DiskState() after first scan = %v, want %v", got, DiskMissing)
	}
	if got := fp.Checksum(); got != "" {
		t.Fatalf("Checksum() after first scan = %q, want empty", got)
	}

	mtime := time.Unix(1_700_000_101, 0)
	content := "payload"
	if err := os.WriteFile(file.Path(), []byte(content), 0o644); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}
	if err := file.Chtimes(mtime, mtime, DefaultContext); err != nil {
		t.Fatalf("Chtimes() error = %v", err)
	}

	state, err = fp.Scan()
	if err != nil {
		t.Fatalf("second Scan() error = %v", err)
	}
	if state != DiskPresent {
		t.Fatalf("second Scan() state = %v, want %v", state, DiskPresent)
	}
	if got := fp.PreviousDiskState(); got != DiskMissing {
		t.Fatalf("PreviousDiskState() = %v, want %v", got, DiskMissing)
	}
	if !fp.HasPreviousKnownDiskState() {
		t.Fatal("HasPreviousKnownDiskState() = false, want true")
	}
	if !fp.PresenceChanged() {
		t.Fatal("PresenceChanged() = false, want true")
	}
	if fp.ChangedMetadata() {
		t.Fatal("ChangedMetadata() = true, want false for missing -> present")
	}
	if !fp.ChangedContent() {
		t.Fatal("ChangedContent() = false, want true for missing -> present")
	}
	if !fp.Changed() {
		t.Fatal("Changed() = false, want true")
	}
	if got := fp.Checksum(); got != HashHexString(content, sha256.New()) {
		t.Fatalf("Checksum() = %q, want %q", got, HashHexString(content, sha256.New()))
	}
	if got := fp.PreviousChecksum(); got != "" {
		t.Fatalf("PreviousChecksum() = %q, want empty", got)
	}
}

func TestFileFingerprintScanContentDistinguishesMetadataOnlyChange(t *testing.T) {
	file := NewFile(filepath.Join(t.TempDir(), "payload.txt"))
	content := "payload"
	firstMTime := time.Unix(1_700_000_102, 0)
	secondMTime := firstMTime.Add(2 * time.Second)

	if err := os.WriteFile(file.Path(), []byte(content), 0o644); err != nil {
		t.Fatalf("os.WriteFile(first) error = %v", err)
	}
	if err := file.Chtimes(firstMTime, firstMTime, DefaultContext); err != nil {
		t.Fatalf("Chtimes(first) error = %v", err)
	}

	fp := file.Fingerprint()
	if _, err := fp.ScanContent(); err != nil {
		t.Fatalf("first ScanContent() error = %v", err)
	}

	if err := file.Chtimes(secondMTime, secondMTime, DefaultContext); err != nil {
		t.Fatalf("Chtimes(second) error = %v", err)
	}

	if _, err := fp.ScanContent(); err != nil {
		t.Fatalf("second ScanContent() error = %v", err)
	}
	if fp.PresenceChanged() {
		t.Fatal("PresenceChanged() = true, want false for present -> present")
	}
	if !fp.ChangedMetadata() {
		t.Fatal("ChangedMetadata() = false, want true for mtime-only change")
	}
	if fp.ChangedContent() {
		t.Fatal("ChangedContent() = true, want false when checksum stays the same")
	}
	if !fp.Changed() {
		t.Fatal("Changed() = false, want true")
	}
	if got := fp.PreviousChecksum(); got != HashHexString(content, sha256.New()) {
		t.Fatalf("PreviousChecksum() = %q, want %q", got, HashHexString(content, sha256.New()))
	}
	if got := fp.Checksum(); got != HashHexString(content, sha256.New()) {
		t.Fatalf("Checksum() = %q, want %q", got, HashHexString(content, sha256.New()))
	}
}

func TestFileFingerprintScanContentTracksChecksumChange(t *testing.T) {
	file := NewFile(filepath.Join(t.TempDir(), "payload.txt"))
	firstContent := "alpha"
	secondContent := "alpha-beta"
	firstMTime := time.Unix(1_700_000_103, 0)
	secondMTime := firstMTime.Add(2 * time.Second)

	if err := os.WriteFile(file.Path(), []byte(firstContent), 0o644); err != nil {
		t.Fatalf("os.WriteFile(first) error = %v", err)
	}
	if err := file.Chtimes(firstMTime, firstMTime, DefaultContext); err != nil {
		t.Fatalf("Chtimes(first) error = %v", err)
	}

	fp := file.Fingerprint()
	if _, err := fp.ScanContent(); err != nil {
		t.Fatalf("first ScanContent() error = %v", err)
	}

	if err := os.WriteFile(file.Path(), []byte(secondContent), 0o644); err != nil {
		t.Fatalf("os.WriteFile(second) error = %v", err)
	}
	if err := file.Chtimes(secondMTime, secondMTime, DefaultContext); err != nil {
		t.Fatalf("Chtimes(second) error = %v", err)
	}

	if _, err := fp.ScanContent(); err != nil {
		t.Fatalf("second ScanContent() error = %v", err)
	}
	if !fp.ChangedMetadata() {
		t.Fatal("ChangedMetadata() = false, want true")
	}
	if !fp.ChangedContent() {
		t.Fatal("ChangedContent() = false, want true")
	}
	if !fp.Changed() {
		t.Fatal("Changed() = false, want true")
	}
	if got := fp.PreviousChecksum(); got != HashHexString(firstContent, sha256.New()) {
		t.Fatalf("PreviousChecksum() = %q, want %q", got, HashHexString(firstContent, sha256.New()))
	}
	if got := fp.Checksum(); got != HashHexString(secondContent, sha256.New()) {
		t.Fatalf("Checksum() = %q, want %q", got, HashHexString(secondContent, sha256.New()))
	}
}

func TestFileFingerprintScanErrorPreservesMetadataAndContentHistory(t *testing.T) {
	base := t.TempDir()
	parent := filepath.Join(base, "parent")
	file := NewFile(filepath.Join(parent, "payload.txt"))
	content := "payload"
	if err := os.MkdirAll(parent, 0o755); err != nil {
		t.Fatalf("os.MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(file.Path(), []byte(content), 0o644); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	fp := file.Fingerprint()
	if _, err := fp.Scan(); err != nil {
		t.Fatalf("first Scan() error = %v", err)
	}
	checksum := fp.Checksum()

	if err := os.Remove(file.Path()); err != nil {
		t.Fatalf("os.Remove(file) error = %v", err)
	}
	if err := os.Remove(parent); err != nil {
		t.Fatalf("os.Remove(parent) error = %v", err)
	}
	if err := os.WriteFile(parent, []byte("not a directory"), 0o644); err != nil {
		t.Fatalf("os.WriteFile(parent file) error = %v", err)
	}

	state, err := fp.Scan()
	if err == nil {
		t.Fatal("Scan() error = nil, want non-nil")
	}
	if state != DiskPresent {
		t.Fatalf("Scan() state = %v, want %v after error", state, DiskPresent)
	}
	if got := fp.DiskState(); got != DiskPresent {
		t.Fatalf("DiskState() = %v, want %v", got, DiskPresent)
	}
	if got := fp.PreviousDiskState(); got != DiskUnknown {
		t.Fatalf("PreviousDiskState() = %v, want %v", got, DiskUnknown)
	}
	if fp.HasPreviousKnownDiskState() {
		t.Fatal("HasPreviousKnownDiskState() = true, want false")
	}
	if got := fp.Checksum(); got != checksum {
		t.Fatalf("Checksum() = %q, want %q", got, checksum)
	}
	if got := fp.PreviousChecksum(); got != "" {
		t.Fatalf("PreviousChecksum() = %q, want empty", got)
	}
	if fp.ChangedMetadata() {
		t.Fatal("ChangedMetadata() = true, want false when failed scan preserves history")
	}
	if fp.ChangedContent() {
		t.Fatal("ChangedContent() = true, want false when failed scan preserves history")
	}
	if fp.Changed() {
		t.Fatal("Changed() = true, want false when failed scan preserves history")
	}
}
