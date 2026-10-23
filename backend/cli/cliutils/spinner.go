package cliutils

import (
	"fmt"
	"time"
)

func ShowSpinner(text string, done chan bool) {
	frames := []string{"|", "/", "-", "\\"}
	start := time.Now()
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for i := 0; ; i++ {
		select {
		case <-done:
			fmt.Printf("\033[2K\r%s (%.1fs)\n", text, time.Since(start).Seconds())
			return
		case <-ticker.C:
			fmt.Printf("\033[2K\r%s %s (%.1fs)", frames[i%len(frames)], text, time.Since(start).Seconds())
		}
	}
}
