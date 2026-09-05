package diagnostics

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

// The local sink is a small, bounded set of JSONL files inside the log
// directory: one active file plus four rotated generations.
const (
	logFileName     = "saveforge-diagnostics.jsonl"
	logFileMaxBytes = 2 << 20 // 2 MiB
	logFileCount    = 5       // the active file included
	recordMaxBytes  = 8 << 10 // 8 KiB
)

// rotatedName is the name of generation n, where 0 is the active file. Only
// these exact names are ever created, renamed or removed, so an unrelated file
// that happens to live in the same directory is never touched by rotation.
func rotatedName(generation int) string {
	if generation == 0 {
		return logFileName
	}
	return "saveforge-diagnostics." + string(rune('0'+generation)) + ".jsonl"
}

// writeLoop owns the local file for the lifetime of the service. It is the
// only goroutine that opens, writes, rotates or closes it, which is what keeps
// the file I/O off every caller's goroutine and out of every other lock.
func (service *Service) writeLoop() {
	defer close(service.done)

	var file *os.File
	var size int64
	// disabled latches after the first sink failure. The service keeps
	// accepting records into its memory buffer; only the file is given up, and
	// it is given up once rather than retried per record.
	disabled := false
	defer func() {
		if file != nil {
			_ = file.Close()
		}
	}()

	for record := range service.queue {
		if disabled {
			service.countDrop()
			continue
		}
		encoded, err := json.Marshal(record)
		if err != nil || len(encoded)+1 > recordMaxBytes {
			// A record the sink cannot represent is counted and skipped. It is
			// never retried and never described in a second record: a logger that
			// logs its own failures can loop.
			service.countDrop()
			continue
		}
		encoded = append(encoded, '\n')

		if file == nil {
			file, size, err = service.openLog()
			if err != nil {
				file = nil
				disabled = true
				service.failSink()
				continue
			}
		}
		if size+int64(len(encoded)) > logFileMaxBytes {
			_ = file.Close()
			file = nil
			if err := service.rotate(); err != nil {
				disabled = true
				service.failSink()
				continue
			}
		}
		if file == nil {
			file, size, err = service.openLog()
			if err != nil {
				file = nil
				disabled = true
				service.failSink()
				continue
			}
		}
		written, err := file.Write(encoded)
		if err != nil {
			_ = file.Close()
			file = nil
			disabled = true
			service.failSink()
			continue
		}
		size += int64(written)
	}
}

// openLog opens the active file for appending and reports its current size.
func (service *Service) openLog() (*os.File, int64, error) {
	if err := os.MkdirAll(service.directory, 0o700); err != nil {
		return nil, 0, err
	}
	path := filepath.Join(service.directory, logFileName)
	if info, err := os.Lstat(path); err == nil && !info.Mode().IsRegular() {
		return nil, 0, errors.New("diagnostic log is not a regular file")
	} else if err != nil && !os.IsNotExist(err) {
		return nil, 0, err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return nil, 0, err
	}
	if err := file.Chmod(0o600); err != nil {
		_ = file.Close()
		return nil, 0, err
	}
	info, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return nil, 0, err
	}
	return file, info.Size(), nil
}

// rotate shifts the generations up by one and drops the oldest. It renames
// only the names rotatedName produces; a missing generation is skipped and no
// other file in the directory is read, moved or removed.
func (service *Service) rotate() error {
	for generation := 0; generation < logFileCount; generation++ {
		info, err := os.Lstat(filepath.Join(service.directory, rotatedName(generation)))
		if err == nil && !info.Mode().IsRegular() {
			return errors.New("diagnostic generation is not a regular file")
		}
		if err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	oldest := filepath.Join(service.directory, rotatedName(logFileCount-1))
	if err := os.Remove(oldest); err != nil && !os.IsNotExist(err) {
		return err
	}
	for generation := logFileCount - 2; generation >= 0; generation-- {
		from := filepath.Join(service.directory, rotatedName(generation))
		to := filepath.Join(service.directory, rotatedName(generation+1))
		if _, err := os.Stat(from); os.IsNotExist(err) {
			continue
		}
		if err := os.Rename(from, to); err != nil {
			return err
		}
	}
	return nil
}

// failSink marks local logging unavailable and counts the lost record. The
// operation that produced it is unaffected: the in-memory buffer still holds
// the record and the caller was never told about the file at all.
func (service *Service) failSink() {
	service.mutex.Lock()
	service.sinkFailed = true
	service.dropped++
	service.mutex.Unlock()
}

func (service *Service) countDrop() {
	service.mutex.Lock()
	service.dropped++
	service.mutex.Unlock()
}
