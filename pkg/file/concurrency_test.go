package file

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestJSONLinesConcurrentAppendAndRead(t *testing.T) {
	path := filepath.Join(t.TempDir(), "events.jsonl")
	const writers = 24
	const writesPerWorker = 8

	var wg sync.WaitGroup
	errCh := make(chan error, writers*2)
	for i := range writers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range writesPerWorker {
				if err := JSONLineAppend(path, map[string]int{"worker": i, "seq": j}); err != nil {
					errCh <- err
					return
				}
			}
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			var lines []string
			if err := JSONLineRead(path, func(line string) error {
				var event map[string]int
				if err := json.Unmarshal([]byte(line), &event); err != nil {
					return fmt.Errorf("partial line %q: %w", line, err)
				}
				lines = append(lines, line)
				return nil
			}); err != nil {
				if !os.IsNotExist(err) {
					errCh <- err
				}
			}
		}()
	}

	wg.Wait()
	close(errCh)
	for err := range errCh {
		t.Fatal(err)
	}

	lines, err := JSONLineReadAll(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != writers*writesPerWorker {
		t.Fatalf("line count = %d, want %d", len(lines), writers*writesPerWorker)
	}
}
