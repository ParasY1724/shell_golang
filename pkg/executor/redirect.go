package executor

import (
	"os"
)

func WithStdRedirect(filename string, appendMode bool, isStderr bool, fn func()) error {
	var file *os.File
	var err error

	flags := os.O_CREATE | os.O_WRONLY
	if appendMode {
		flags |= os.O_APPEND
	}
	
	// Default permissions 0644
	file, err = os.OpenFile(filename, flags, 0644)
	if err != nil {
		return err
	}

	var old *os.File

	if isStderr {
		old = os.Stderr
		os.Stderr = file
	} else {
		old = os.Stdout
		os.Stdout = file
	}

	defer func() {
		file.Close()
		if isStderr {
			os.Stderr = old
		} else {
			os.Stdout = old
		}
	}()

	fn()
	
	return nil
}