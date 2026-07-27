// Plan based execution with fallback
package scheduler

import (
	"context"
	"go-ai-tools/registry"
	"time"
)

// Scheduler executes plans
type Scheduler struct {
	registry *registry.Registry
}

// Step in a plan
type Step struct {
	ToolName  string
	Arguments string
	WaitFor   []string // step IDs to wait for
}

// Plan is sequence of steps
type Plan struct {
	Steps      []Step
	Parameters map[string]any `json:"parameters"`
}

// StepResult is the outcome of a single step
type StepResult struct {
	Step   Step
	Result any
	Error  error
	Start  time.Time
	End    time.Time
}

// PlanSummary aggregates all step results
type PlanSummary struct {
	Results    []StepResult
	Successes  int
	Failures   int
	TotalSteps int
	Duration   time.Duration
	StopReason string
}

// New creates a scheduler
func New(r *registry.Registry) *Scheduler {
	return &Scheduler{registry: r}
}

// Execute runs all plan steps in order
func (s *Scheduler) Execute(ctx context.Context, plan *Plan) *PlanSummary {
	summary := &PlanSummary{TotalSteps: len(plan.Steps)}
	completed := make(map[string]bool)

	for _, step := range plan.Steps {
		// Wait for dependencies
		for _, wait := range step.WaitFor {
			if !completed[wait] {
				time.Sleep(100 * time.Millisecond)
			}
		}

		start := time.Now()
		result, err := s.registry.Call(ctx, step.ToolName, []byte(step.Arguments))
		end := time.Now()

		sr := StepResult{Step: step, Start: start, End: end}
		if err != nil {
			sr.Error = err
			summary.Failures++
		} else {
			sr.Result = result
			summary.Successes++
			completed[step.ToolName] = true
		}
	}

	summary.Duration = time.Since(time.Now())
	return summary
}

// ExecuteWithFallback runs with fallback on failure
func (s *Scheduler) ExecuteWithFallback(ctx context.Context, plan *Plan, fallbackStep func(context.Context, Step) (*StepResult, error)) *PlanSummary {
	summary := &PlanSummary{TotalSteps: len(plan.Steps)}

	for i, step := range plan.Steps {
		start := time.Now()
		result, err := s.registry.Call(ctx, step.ToolName, []byte(step.Arguments))
		end := time.Now()

		sr := StepResult{Step: step, Start: start, End: end}
		if err != nil {
			if fallbackStep != nil {
				fb, fbErr := fallbackStep(ctx, step)
				if fbErr == nil {
					sr = *fb
				}
			}
			sr.Error = err
			summary.Failures++
		} else {
			sr.Result = result
			summary.Successes++
		}
		summary.Results = append(summary.Results, sr)
		_ = i
	}

	summary.Duration = time.Since(time.Now())
	return summary
}
