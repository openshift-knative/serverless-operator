package errors

// a sync progressing error captures the idea that the operand is in an incomplete state
// and needs to be reconciled.  It should be used when the sync loop should abort, but
// it implies the idea of "progressing", not "failure".  A built-in error type can be
// used and returned when a true failure is encountered.
type SyncProgressingError struct {
	message string
}

// implement the error interface
func (e *SyncProgressingError) Error() string {
	return e.message
}

// NewSyncError("Sync failed on xyz")
// A builder func, should we choose to add additional metadata on the custom err
func NewSyncError(msg string) *SyncProgressingError {
	err := &SyncProgressingError{
		message: msg,
	}
	return err
}

func IsSyncError(err error) bool {
	if err == nil {
		return false
	}
	_, ok := err.(*SyncProgressingError)
	return ok
}
