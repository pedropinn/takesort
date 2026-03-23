package logmsg

// Startup and shutdown
const (
	StartupConfigFailed = "failed to load config"
	StartupBegin        = "takesort starting"
	ShutdownComplete    = "takesort shutdown complete"
)

// Orchestrator lifecycle
const (
	OrchestratorStarted              = "orchestrator started"
	OrchestratorStopping             = "orchestrator stopping"
	OrchestratorFailed               = "orchestrator failed"
	OrchestratorMediaDirInaccessible = "media directory inaccessible"
)

// Watcher lifecycle
const (
	WatcherStarted        = "watcher started"
	WatcherStopping       = "watcher stopping"
	WatcherError          = "watcher error"
	WatcherFileDetected   = "file detected"
	WatcherEventIgnored   = "event in excluded directory ignored"
	WatcherLstatFailed    = "lstat failed for event"
	WatcherSymlinkIgnored = "symlink ignored"
)

// Directory operations
const (
	DirHiddenDeleting     = "deleting hidden directory"
	DirHiddenDeleteFailed = "failed to delete hidden directory"
	DirWatchFailed        = "failed to watch subdirectory"
)

// File pipeline
const (
	FileSymlinkRejected  = "symlink rejected"
	FileValidationFailed = "file failed validation"
	FileClassified       = "file classified"
	FileUnknownDeleting  = "unknown file type, deleting"
	FileTrashDeleting    = "deleting trash file"
	FileStatFailed       = "failed to stat file"
	FileDateResolved     = "date resolved"
	FileConflictDetected = "conflict detected, redirecting to conflicts"
	FileProxyProcessing  = "processing proxy file"
	FileProxyFailed      = "proxy processing failed"
	FileMoving           = "moving file"
	FileMoveFailed       = "file move failed"
	FileProcessFailed    = "file processing failed"
)

// Debounce
const (
	DebounceStarted         = "debounce started"
	DebounceComplete        = "debounce complete"
	DebounceFailed          = "debounce failed"
	DebounceFileDisappeared = "file disappeared during debounce"
)

// Orphan processing
const (
	OrphanScanFailed          = "orphan scan failed"
	OrphanProcessingStarted   = "processing orphan files"
	OrphanDebounceDisappeared = "orphan disappeared during debounce"
	OrphanDebounceFailed      = "orphan debounce failed"
	OrphanProcessFailed       = "orphan processing failed"
)

// Cleanup
const (
	CleanupAfterOrphansFailed  = "cleanup after orphan processing failed"
	CleanupAfterFileFailed     = "cleanup after file processing failed"
	CleanupErrorsDirRemoved    = "removed empty errors directory"
	CleanupConflictsDirRemoved = "removed empty conflicts directory"
)
