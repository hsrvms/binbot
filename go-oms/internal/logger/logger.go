package logger

import (
	"fmt"
	"os"
	"sync"
	"time"
)

type LineRotator struct {
	mu        sync.Mutex
	filename  string
	file      *os.File
	lineCount int
	maxLines  int
}

func NewLineRotator(filename string, maxLines int) (*LineRotator, error) {
	lr := &LineRotator{
		filename: filename,
		maxLines: maxLines,
	}

	if err := lr.rotate(); err != nil {
		return nil, err
	}

	return lr, nil
}

func (lr *LineRotator) Write(p []byte) (n int, err error) {
	lr.mu.Lock()
	defer lr.mu.Unlock()

	newLines := 0
	for _, b := range p {
		if b == '\n' {
			newLines++
		}
	}

	if lr.lineCount+newLines > lr.maxLines {
		if err := lr.rotate(); err != nil {
			return 0, err
		}
	}

	n, err = lr.file.Write(p)
	lr.lineCount += newLines
	return n, err
}

func (lr *LineRotator) rotate() error {
	if lr.file != nil {
		lr.file.Close()
	}

	if _, err := os.Stat(lr.filename); err == nil {
		backupName := fmt.Sprintf("%s.%d.bak", lr.filename, time.Now().UnixNano())
		os.Rename(lr.filename, backupName)
	}

	f, err := os.OpenFile(lr.filename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
	if err != nil {
		return err
	}
	lr.file = f
	lr.lineCount = 0
	return nil
}
