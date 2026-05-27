package runner

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/AlvaroMrJack/k-flow-spec/pkg/kapso"
)

func PollUntil(ctx context.Context, client *kapso.Client, workflowID, executionID string, timeout time.Duration, prevStep interface{}, targetStatuses ...string) (*kapso.ExecutionStatus, error) {
	deadline := time.Now().Add(timeout)
	pollInterval := 500 * time.Millisecond
	attempt := 0

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		status, err := client.GetExecution(ctx, workflowID, executionID)
		if err != nil {
			attempt++
			sleepTime := time.Duration(math.Min(float64(pollInterval)*math.Pow(1.5, float64(attempt)), float64(5*time.Second)))
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(sleepTime):
			}
			continue
		}

		if isStatusMatch(status, targetStatuses) {
			if status.Status == "waiting" && prevStep != nil {
				if stepsEqual(status.CurrentStep, prevStep) {
					attempt++
					sleepTime := time.Duration(math.Min(float64(pollInterval)*math.Pow(1.2, float64(attempt)), float64(3*time.Second)))
					select {
					case <-ctx.Done():
						return nil, ctx.Err()
					case <-time.After(sleepTime):
					}
					continue
				}
			}
			return status, nil
		}

		attempt++
		sleepTime := time.Duration(math.Min(float64(pollInterval)*math.Pow(1.2, float64(attempt)), float64(3*time.Second)))
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(sleepTime):
		}
	}

	return nil, fmt.Errorf("polling timeout after %v (last status: %v)", timeout, "unknown")
}

func isStatusMatch(status *kapso.ExecutionStatus, targetStatuses []string) bool {
	for _, target := range targetStatuses {
		if status.Status == target {
			return true
		}
	}
	return false
}

func stepsEqual(a, b interface{}) bool {
	return fmt.Sprintf("%v", a) == fmt.Sprintf("%v", b)
}
