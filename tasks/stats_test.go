package tasks_test

import (
	"crynux_relay/tasks"
	"crynux_relay/tests"
	"testing"
	"time"
)

func TestStats(t *testing.T) {
	ch := make(chan int)

	go tasks.StartStatsTaskExecutionTimeCountWithTerminateChannel(ch)
	time.Sleep(5 * time.Minute)
	ch <- 1

	go tasks.StartStatsTaskUploadResultTimeCountWithTerminateChannel(ch)
	time.Sleep(5 * time.Minute)
	ch <- 1

	t.Cleanup(func() {
		tests.ClearDB()
	})
}
