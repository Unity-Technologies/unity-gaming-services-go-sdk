package server

import (
	"errors"
	"fmt"
	"io"
	"path/filepath"

	"github.com/fsnotify/fsnotify"
)

// watchForConfigChanges watches the provided configuration file for changes.
func (s *Server) watchForConfigChanges() {
	w, _ := fsnotify.NewWatcher()
	_ = w.Add(filepath.Dir(s.cfgFile))

	s.wg.Add(1)
	s.internalEventProcessorReady <- struct{}{}
	defer s.wg.Done()

	for {
		select {
		case evt, ok := <-w.Events:
			if !ok {
				return
			}

			// Ignore events for other files.
			if evt.Name != s.cfgFile {
				continue
			}

			// We only care about when the config file has been rewritten.
			if evt.Op&fsnotify.Write != fsnotify.Write {
				continue
			}

			c, err := newConfigFromFile(s.cfgFile)
			if err != nil {
				// Multiplay truncates the file when a deallocation occurs,
				// which results in two writes. The first write will produce an
				// empty file, meaning JSON parsing will fail.
				if !errors.Is(err, io.EOF) {
					s.PushError(fmt.Errorf("error parsing new configuration: %w", err))
				}

				continue
			}

			s.setConfig(c)

		case err, ok := <-w.Errors:
			if !ok {
				return
			}

			s.PushError(fmt.Errorf("error watching config file: %w", err))

		case <-s.done:
			_ = w.Close()
			close(s.internalEventProcessorReady)

			return
		}
	}
}
