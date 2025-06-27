package tinybench

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/codahale/tinystat"
)

const (
	// Default sampling configuration
	DefaultSamples  = 100
	DefaultDuration = 10 * time.Millisecond
	DefaultTableFmt = "%-20s %-12s %-12s %-18s %-18s\n"
)

// Result represents a single benchmark result
type Result struct {
	Name      string    `json:"name"`
	Samples   []float64 `json:"samples"`
	Timestamp int64     `json:"timestamp"`
}

// Runner manages benchmarks and handles persistence
type Runner struct {
	filename string
	filter   string
	samples  int
	duration time.Duration
	tableFmt string
	results  map[string]Result
	showRef  bool
}

// New creates a new benchmark runner
func New(filename string) *Runner {
	return &Runner{
		filename: filename,
		samples:  DefaultSamples,
		duration: DefaultDuration,
		tableFmt: DefaultTableFmt,
		results:  make(map[string]Result),
		showRef:  true,
	}
}

// Filter sets a prefix filter for benchmark names
func (r *Runner) Filter(prefix string) *Runner {
	r.filter = prefix
	return r
}

// Samples sets the number of samples to collect per benchmark
func (r *Runner) Samples(n int) *Runner {
	r.samples = n
	return r
}

// Duration sets the duration for each sample
func (r *Runner) Duration(d time.Duration) *Runner {
	r.duration = d
	return r
}

// TableFormat sets the table format string
func (r *Runner) TableFormat(fmt string) *Runner {
	r.tableFmt = fmt
	return r
}

// ShowReference controls whether to show reference comparison column
func (r *Runner) ShowReference(show bool) *Runner {
	r.showRef = show
	return r
}

// shouldRun checks if a benchmark matches the filter
func (r *Runner) shouldRun(name string) bool {
	if r.filter == "" {
		return true
	}
	return strings.HasPrefix(name, r.filter)
}

// benchmark runs a function repeatedly and returns performance samples
func (r *Runner) benchmark(fn func()) []float64 {
	samples := make([]float64, r.samples)
	for i := range samples {
		start := time.Now()
		ops := 0
		for time.Since(start) < r.duration {
			fn()
			ops++
		}
		samples[i] = float64(ops) / time.Since(start).Seconds()
	}
	return samples
}

// loadResults loads previous results from JSON file
func (r *Runner) loadResults() map[string]Result {
	data, err := os.ReadFile(r.filename)
	if err != nil {
		return make(map[string]Result)
	}

	var results map[string]Result
	if err := json.Unmarshal(data, &results); err != nil {
		return make(map[string]Result)
	}

	return results
}

// saveResults saves current results to JSON file
func (r *Runner) saveResults() {
	data, err := json.MarshalIndent(r.results, "", "  ")
	if err != nil {
		fmt.Printf("Error marshaling results: %v\n", err)
		return
	}

	if err := os.WriteFile(r.filename, data, 0644); err != nil {
		fmt.Printf("Error writing results file: %v\n", err)
	}
}

// saveResult saves a single result incrementally
func (r *Runner) saveResult(result Result) {
	// Load current results to merge with
	current := r.loadResults()
	current[result.Name] = result

	// Save merged results
	data, err := json.MarshalIndent(current, "", "  ")
	if err != nil {
		fmt.Printf("Error marshaling results: %v\n", err)
		return
	}

	if err := os.WriteFile(r.filename, data, 0644); err != nil {
		fmt.Printf("Error writing results file: %v\n", err)
	}
}

// formatResult formats statistical comparison between samples
func (r *Runner) formatResult(ourSamples, refSamples []float64) string {
	our := tinystat.Summarize(ourSamples)
	ref := tinystat.Summarize(refSamples)
	if ref.Mean == 0 {
		if our.Mean > 0 {
			return "✅ inf"
		}
		return "~ 1.00x"
	}

	speedup := our.Mean / ref.Mean
	diff := tinystat.Compare(our, ref, 99)
	if !diff.Significant() {
		return fmt.Sprintf("~ %.2fx (p=%.3f)", speedup, diff.PValue)
	}

	if speedup > 1 {
		return fmt.Sprintf("✅ %.2fx (p=%.3f)", speedup, diff.PValue)
	}

	return fmt.Sprintf("❌ %.2fx (p=%.3f)", speedup, diff.PValue)
}

// formatDelta formats comparison between current and previous runs
func (r *Runner) formatDelta(current, previous Result) string {
	if len(previous.Samples) == 0 {
		return "new"
	}

	curr := tinystat.Summarize(current.Samples)
	prev := tinystat.Summarize(previous.Samples)
	if prev.Mean == 0 {
		if curr.Mean > 0 {
			return "✅ inf"
		}
		return "~ 1.00x"
	}

	speedup := curr.Mean / prev.Mean
	diff := tinystat.Compare(curr, prev, 99)
	if !diff.Significant() {
		return fmt.Sprintf("~ %.2fx (p=%.3f)", speedup, diff.PValue)
	}

	if speedup > 1 {
		return fmt.Sprintf("✅ %.2fx (p=%.3f)", speedup, diff.PValue)
	}

	return fmt.Sprintf("❌ %.2fx (p=%.3f)", speedup, diff.PValue)
}

// formatTime formats nanoseconds per operation
func (r *Runner) formatTime(nsPerOp float64) string {
	if nsPerOp >= 1000000 {
		return fmt.Sprintf("%.1fms", nsPerOp/1000000)
	}
	return fmt.Sprintf("%.1fns", nsPerOp)
}

// formatOps formats operations per second
func (r *Runner) formatOps(opsPerSec float64) string {
	if opsPerSec >= 1000000 {
		return fmt.Sprintf("%.1fM", opsPerSec/1000000)
	}
	if opsPerSec >= 1000 {
		return fmt.Sprintf("%.1fK", opsPerSec/1000)
	}
	return fmt.Sprintf("%.0f", opsPerSec)
}

// Start begins a benchmark session and prints the header
func (r *Runner) Start() *Runner {
	if r.showRef {
		fmt.Printf(r.tableFmt, "name", "time/op", "ops/s", "vs prev", "vs ref")
		fmt.Printf(r.tableFmt, "--------------------", "------------", "------------", "------------------", "------------------")
	} else {
		fmt.Printf("%-20s %-12s %-12s %-18s\n", "name", "time/op", "ops/s", "vs prev")
		fmt.Printf("%-20s %-12s %-12s %-18s\n", "--------------------", "------------", "------------", "------------------")
	}
	return r
}

// Run executes a benchmark with optional reference comparison
func (r *Runner) Run(name string, ourFn func(), refFn ...func()) *Runner {
	if !r.shouldRun(name) {
		return r
	}

	// Load previous results for delta comparison
	prevResults := r.loadResults()

	// Benchmark our implementation
	ourSamples := r.benchmark(ourFn)
	ourMean := tinystat.Summarize(ourSamples).Mean
	nsPerOp := 1e9 / ourMean

	// Create result
	result := Result{
		Name:      name,
		Samples:   ourSamples,
		Timestamp: time.Now().Unix(),
	}

	// Calculate delta vs previous run
	prevResult, exists := prevResults[name]
	delta := "new"
	if exists {
		delta = r.formatDelta(result, prevResult)
	}

	// Calculate vs reference if provided
	vsRef := ""
	if len(refFn) > 0 && refFn[0] != nil {
		refSamples := r.benchmark(refFn[0])
		vsRef = r.formatResult(ourSamples, refSamples)
	}

	// Format and display result
	if r.showRef {
		fmt.Printf(r.tableFmt,
			name,
			r.formatTime(nsPerOp),
			r.formatOps(ourMean),
			delta,
			vsRef)
	} else {
		fmt.Printf("%-20s %-12s %-12s %-18s\n",
			name,
			r.formatTime(nsPerOp),
			r.formatOps(ourMean),
			delta)
	}

	// Save result incrementally
	r.saveResult(result)

	return r
}

// Finish completes the benchmark session
func (r *Runner) Finish() {
	// Results are already saved incrementally, nothing more to do
}
