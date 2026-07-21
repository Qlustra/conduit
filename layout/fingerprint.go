package layout

import (
	"crypto/sha256"
	"os"
	"time"
)

// FileFingerprint is a stateful file observer for change detection.
//
// It keeps the current and previous successful observations of a File's disk
// presence, size, modification time, and content checksum so callers can ask
// whether the file changed since the last scan.
type FileFingerprint struct {
	file File

	disk     DiskState
	prevDisk DiskState

	size     int64
	prevSize int64

	modTime     time.Time
	prevModTime time.Time

	contentDisk     DiskState
	prevContentDisk DiskState

	checksum     string
	prevChecksum string

	hasCurrent  bool
	hasPrevious bool

	hasContentCurrent  bool
	hasContentPrevious bool
}

// Fingerprint returns a stateful fingerprint bound to the file.
func (f File) Fingerprint() FileFingerprint {
	return FileFingerprint{file: f}
}

// File returns the bound file handle.
func (fp FileFingerprint) File() File {
	return fp.file
}

// DiskState returns the last successfully observed disk state.
func (fp FileFingerprint) DiskState() DiskState {
	if !fp.hasCurrent {
		return DiskUnknown
	}
	return fp.disk
}

// PreviousDiskState returns the prior successfully observed disk state.
func (fp FileFingerprint) PreviousDiskState() DiskState {
	if !fp.hasPrevious {
		return DiskUnknown
	}
	return fp.prevDisk
}

// HasKnownDiskState reports whether a successful observation has been recorded.
func (fp FileFingerprint) HasKnownDiskState() bool {
	return fp.hasCurrent && fp.disk != DiskUnknown
}

// HasPreviousKnownDiskState reports whether a prior successful observation has
// been recorded.
func (fp FileFingerprint) HasPreviousKnownDiskState() bool {
	return fp.hasPrevious && fp.prevDisk != DiskUnknown
}

// WasObservedOnDisk reports whether the last successful observation found the
// file present on disk.
func (fp FileFingerprint) WasObservedOnDisk() bool {
	return fp.DiskState() == DiskPresent
}

// WasPreviouslyObservedOnDisk reports whether the prior successful observation
// found the file present on disk.
func (fp FileFingerprint) WasPreviouslyObservedOnDisk() bool {
	return fp.PreviousDiskState() == DiskPresent
}

// Size returns the size from the last successful observation.
//
// It returns zero when the file is missing or has not been observed yet.
func (fp FileFingerprint) Size() int64 {
	return fp.size
}

// PreviousSize returns the size from the prior successful observation.
//
// It returns zero when the file was previously missing or has only been
// observed once.
func (fp FileFingerprint) PreviousSize() int64 {
	return fp.prevSize
}

// ModTime returns the modification time from the last successful observation.
//
// It returns the zero time when the file is missing or has not been observed
// yet.
func (fp FileFingerprint) ModTime() time.Time {
	return fp.modTime
}

// PreviousModTime returns the modification time from the prior successful
// observation.
//
// It returns the zero time when the file was previously missing or has only
// been observed once.
func (fp FileFingerprint) PreviousModTime() time.Time {
	return fp.prevModTime
}

// Checksum returns the content checksum from the last successful content scan.
//
// It returns an empty string when the file was missing or content has not been
// scanned yet.
func (fp FileFingerprint) Checksum() string {
	return fp.checksum
}

// PreviousChecksum returns the content checksum from the prior successful
// content scan.
//
// It returns an empty string when the file was previously missing or content
// has only been scanned once.
func (fp FileFingerprint) PreviousChecksum() string {
	return fp.prevChecksum
}

// Changed reports whether either metadata or content differs from the prior
// successful observation.
func (fp FileFingerprint) Changed() bool {
	return fp.ChangedMetadata() || fp.ChangedContent()
}

// PresenceChanged reports whether the last successful observation changed
// between present and missing.
func (fp FileFingerprint) PresenceChanged() bool {
	if !fp.hasCurrent || !fp.hasPrevious {
		return false
	}
	return fp.disk != fp.prevDisk
}

// ChangedMetadata reports whether size or modification time changed between
// the prior and current successful observations while the file was present in
// both.
func (fp FileFingerprint) ChangedMetadata() bool {
	if !fp.hasCurrent || !fp.hasPrevious {
		return false
	}
	if fp.disk != DiskPresent || fp.prevDisk != DiskPresent {
		return false
	}
	return fp.size != fp.prevSize || !fp.modTime.Equal(fp.prevModTime)
}

// ChangedContent reports whether content presence or checksum changed between
// the prior and current successful content observations.
func (fp FileFingerprint) ChangedContent() bool {
	if !fp.hasContentCurrent || !fp.hasContentPrevious {
		return false
	}
	if fp.contentDisk != fp.prevContentDisk {
		return true
	}
	if fp.contentDisk != DiskPresent {
		return false
	}
	return fp.checksum != fp.prevChecksum
}

// Scan refreshes both metadata and content observations.
func (fp *FileFingerprint) Scan() (DiskState, error) {
	return fp.ScanContent()
}

// ScanMetadata refreshes size and modification-time observation state.
//
// Missing files are recorded as DiskMissing without returning an error. Other
// os.Stat errors leave the current and previous observations unchanged.
func (fp *FileFingerprint) ScanMetadata() (DiskState, error) {
	info, err := os.Stat(fp.file.Path())
	if err != nil {
		if os.IsNotExist(err) {
			fp.advanceMetadata(DiskMissing, 0, time.Time{})
			return fp.disk, nil
		}
		return fp.DiskState(), err
	}

	fp.advanceMetadata(DiskPresent, info.Size(), info.ModTime())
	return fp.disk, nil
}

// ScanContent refreshes content checksum observation state using SHA-256.
//
// Successful content scans also refresh metadata observation state for the same
// file state. Missing files are recorded as DiskMissing without returning an
// error. Open, stat, hash, and close errors leave both metadata and content
// histories unchanged.
func (fp *FileFingerprint) ScanContent() (DiskState, error) {
	handle, err := os.Open(fp.file.Path())
	if err != nil {
		if os.IsNotExist(err) {
			fp.advanceMetadata(DiskMissing, 0, time.Time{})
			fp.advanceContent(DiskMissing, "")
			return DiskMissing, nil
		}
		return fp.DiskState(), err
	}

	info, statErr := handle.Stat()
	if statErr != nil {
		_ = handle.Close()
		return fp.DiskState(), statErr
	}

	checksum, hashErr := HashHexReader(handle, sha256.New())
	closeErr := handle.Close()
	if hashErr != nil {
		return fp.DiskState(), hashErr
	}
	if closeErr != nil {
		return fp.DiskState(), closeErr
	}

	fp.advanceMetadata(DiskPresent, info.Size(), info.ModTime())
	fp.advanceContent(DiskPresent, checksum)
	return DiskPresent, nil
}

func (fp *FileFingerprint) advanceMetadata(state DiskState, size int64, modTime time.Time) {
	if fp.hasCurrent {
		fp.prevDisk = fp.disk
		fp.prevSize = fp.size
		fp.prevModTime = fp.modTime
		fp.hasPrevious = true
	}

	fp.disk = state
	fp.size = size
	fp.modTime = modTime
	fp.hasCurrent = true
}

func (fp *FileFingerprint) advanceContent(state DiskState, checksum string) {
	if fp.hasContentCurrent {
		fp.prevContentDisk = fp.contentDisk
		fp.prevChecksum = fp.checksum
		fp.hasContentPrevious = true
	}

	fp.contentDisk = state
	fp.checksum = checksum
	fp.hasContentCurrent = true
}
