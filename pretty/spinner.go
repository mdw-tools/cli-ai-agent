package pretty

import (
	"fmt"
	"sync"
	"time"
)

// Spinner displays a loading animation
type Spinner struct {
	mu      sync.Mutex
	active  bool
	done    chan bool
	message string
}

func NewSpinner(message string) *Spinner {
	return &Spinner{
		done:    make(chan bool),
		message: message,
	}
}

func (this *Spinner) Start() {
	this.mu.Lock()
	if this.active {
		this.mu.Unlock()
		return
	}
	this.active = true
	this.mu.Unlock()

	go func() {
		chars := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
		i := 0
		for {
			select {
			case <-this.done:
				fmt.Print("\r\033[K") // Clear the line
				return
			default:
				fmt.Printf("\r%s %s", chars[i], this.message)
				i = (i + 1) % len(chars)
				time.Sleep(80 * time.Millisecond)
			}
		}
	}()
}

func (this *Spinner) Stop() {
	this.mu.Lock()
	defer this.mu.Unlock()
	if !this.active {
		return
	}
	this.active = false
	this.done <- true
}
