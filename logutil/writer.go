package logutil

import (
	"fmt"
	"os"
	"sync"
)

// RotatableWriter is a thread-safe writer that automatically rotates the log file
// when it exceeds a certain size.
type RotatableWriter struct {
	filename string
	maxSize  int64 // in bytes
	file     *os.File
	mu       sync.Mutex
}

func NewRotatableWriter(filename string, maxSizeKB int) (*RotatableWriter, error) {
	w := &RotatableWriter{
		filename: filename,
		maxSize:  int64(maxSizeKB) * 1024,
	}
	if err := w.open(); err != nil {
		return nil, err
	}
	return w, nil
}

func (w *RotatableWriter) open() error {
	f, err := os.OpenFile(w.filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	w.file = f
	return nil
}

func (w *RotatableWriter) Write(p []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.maxSize > 0 {
		info, err := w.file.Stat()
		if err == nil && info.Size() > w.maxSize {
			if err := w.rotate(); err != nil {
				// If rotation fails, complain to stderr but try to keep writing
				fmt.Fprintf(os.Stderr, "Log rotation failed: %v\n", err)
			}
		}
	}

	return w.file.Write(p)
}

func (w *RotatableWriter) rotate() error {
	w.file.Close()
	// Standard rotation: Rename current to .old (overwriting existing .old)
	os.Rename(w.filename, w.filename+".old")
	return w.open()
}

// Close closes the underlying file.
func (w *RotatableWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.file != nil {
		return w.file.Close()
	}
	return nil
}
