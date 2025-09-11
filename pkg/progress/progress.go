package progress

import (
	"fmt"
	"time"
)

// Spinner provides a simple text-based spinner for long operations
type Spinner struct {
	message string
	frames  []string
	done    chan bool
	active  bool
}

// NewSpinner creates a new spinner with the given message
func NewSpinner(message string) *Spinner {
	return &Spinner{
		message: message,
		frames:  []string{"|", "/", "-", "\\"},
		done:    make(chan bool, 1),
		active:  false,
	}
}

// Start begins the spinner animation
func (s *Spinner) Start() {
	if s.active {
		return
	}
	
	s.active = true
	go func() {
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
		
		i := 0
		for {
			select {
			case <-s.done:
				return
			case <-ticker.C:
				fmt.Printf("\r%s %s", s.frames[i%len(s.frames)], s.message)
				i++
			}
		}
	}()
}

// Stop ends the spinner animation and prints a completion message
func (s *Spinner) Stop(successMessage string) {
	if !s.active {
		return
	}
	
	s.active = false
	s.done <- true
	
	// Clear the line and print success message
	fmt.Printf("\r✅ %s\n", successMessage)
}

// StopWithError ends the spinner animation and prints an error message
func (s *Spinner) StopWithError(errorMessage string) {
	if !s.active {
		return
	}
	
	s.active = false
	s.done <- true
	
	// Clear the line and print error message
	fmt.Printf("\r❌ %s\n", errorMessage)
}

// UpdateMessage changes the spinner message while it's running
func (s *Spinner) UpdateMessage(message string) {
	s.message = message
}

// ProgressBar provides a simple text-based progress bar
type ProgressBar struct {
	total   int
	current int
	width   int
	prefix  string
}

// NewProgressBar creates a new progress bar
func NewProgressBar(total int, prefix string) *ProgressBar {
	return &ProgressBar{
		total:   total,
		current: 0,
		width:   50,
		prefix:  prefix,
	}
}

// Update updates the progress bar and displays it
func (pb *ProgressBar) Update(current int) {
	pb.current = current
	pb.display()
}

// Increment increments the progress by 1 and displays the bar
func (pb *ProgressBar) Increment() {
	pb.current++
	pb.display()
}

// Finish completes the progress bar
func (pb *ProgressBar) Finish(message string) {
	pb.current = pb.total
	pb.display()
	fmt.Printf(" ✅ %s\n", message)
}

func (pb *ProgressBar) display() {
	percent := float64(pb.current) / float64(pb.total)
	filled := int(percent * float64(pb.width))
	
	bar := ""
	for i := 0; i < pb.width; i++ {
		if i < filled {
			bar += "█"
		} else {
			bar += "░"
		}
	}
	
	fmt.Printf("\r%s [%s] %d/%d (%.1f%%)", 
		pb.prefix, bar, pb.current, pb.total, percent*100)
}