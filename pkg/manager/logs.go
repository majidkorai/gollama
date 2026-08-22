package manager

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"sync"
)

// instanceLogMaxBytes bounds a single instance log (~10 MB). Logs beyond it
// are rotated one level to <path>.1 (P4-T3); llama-server with --verbose
// writes per-request lines and \r progress spam that would otherwise grow
// unbounded on long-lived instances (idle_ttl: 0).
const instanceLogMaxBytes = 10 * 1024 * 1024

// TailLogFile returns the last maxBytes of path (the whole file when smaller
// than that). It drops a partial first line so callers always get whole
// lines, without ever reading the full file into memory.
func TailLogFile(path string, maxBytes int64) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		return nil, err
	}
	size := fi.Size()
	if size <= maxBytes {
		buf := make([]byte, size)
		if _, err := io.ReadFull(f, buf); err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
			return nil, err
		}
		return buf, nil
	}
	if _, err := f.Seek(-maxBytes, io.SeekEnd); err != nil {
		return nil, err
	}
	buf := make([]byte, maxBytes)
	if _, err := io.ReadFull(f, buf); err != nil && err != io.ErrUnexpectedEOF {
		return nil, err
	}
	if i := bytes.IndexByte(buf, '\n'); i >= 0 {
		buf = buf[i+1:]
	}
	return buf, nil
}

// prepareInstanceLog resets the log file for a fresh instance launch: a
// previous log larger than instanceLogMaxBytes is rotated to <path>.1 (one
// level of history, matching the old truncate-on-start behavior otherwise).
func prepareInstanceLog(path string) {
	prepareInstanceLogMax(path, instanceLogMaxBytes)
}

func prepareInstanceLogMax(path string, maxBytes int64) {
	if fi, err := os.Stat(path); err == nil && fi.Size() > maxBytes {
		os.Remove(path + ".1")
		if err := os.Rename(path, path+".1"); err != nil {
			os.Remove(path)
		}
		return
	}
	os.Remove(path)
}

// rotatingLogWriter is an io.Writer that appends to a log file and rotates
// it to <path>.1 (overwriting any existing .1) once it exceeds maxBytes.
type rotatingLogWriter struct {
	mu       sync.Mutex
	path     string
	file     *os.File
	maxBytes int64
}

func newRotatingLogWriter(path string, maxBytes int64) (*rotatingLogWriter, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}
	return &rotatingLogWriter{path: path, file: f, maxBytes: maxBytes}, nil
}

func (w *rotatingLogWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if err := w.maybeRotate(); err != nil {
		// Rotation is best-effort; keep writing to the current file.
		fmt.Fprintf(os.Stderr, "gollama: log rotation for %s failed: %v\n", w.path, err)
	}
	return w.file.Write(p)
}

func (w *rotatingLogWriter) maybeRotate() error {
	fi, err := w.file.Stat()
	if err != nil {
		return err
	}
	if fi.Size() < w.maxBytes {
		return nil
	}
	if err := w.file.Close(); err != nil {
		return err
	}
	os.Remove(w.path + ".1")
	if err := os.Rename(w.path, w.path+".1"); err != nil {
		// Rename failed — reopen the original file and keep going.
		f, openErr := os.OpenFile(w.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if openErr != nil {
			return openErr
		}
		w.file = f
		return err
	}
	f, err := os.OpenFile(w.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	w.file = f
	return nil
}
