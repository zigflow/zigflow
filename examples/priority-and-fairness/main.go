/*
 * Copyright 2025 - 2026 Zigflow authors <https://github.com/zigflow/zigflow/graphs/contributors>
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	gh "github.com/mrsimonemms/golang-helpers"
	"github.com/mrsimonemms/golang-helpers/temporal"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"go.temporal.io/sdk/client"
	sdktemporal "go.temporal.io/sdk/temporal"
)

// ──────────────────────────────────────────────────────────────────────────────
// Config
// ──────────────────────────────────────────────────────────────────────────────

const (
	numGroups    = 5
	workflowType = "priority-fairness"
	taskQueue    = "zigflow"
)

// ──────────────────────────────────────────────────────────────────────────────
// Styles
// ──────────────────────────────────────────────────────────────────────────────

//nolint:misspell // colour is a local alias for lipgloss.Color.
type colour = lipgloss.Color

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colour("213"))

	groupLabelStyle = lipgloss.NewStyle().
			Width(10).
			Foreground(colour("252"))

	barStyles = [numGroups]lipgloss.Style{
		lipgloss.NewStyle().Foreground(colour("196")), // Group 1 red
		lipgloss.NewStyle().Foreground(colour("214")), // Group 2 orange
		lipgloss.NewStyle().Foreground(colour("226")), // Group 3 yellow
		lipgloss.NewStyle().Foreground(colour("82")),  // Group 4 green
		lipgloss.NewStyle().Foreground(colour("39")),  // Group 5 blue
	}

	countStyle = lipgloss.NewStyle().Foreground(colour("244"))
	doneStyle  = lipgloss.NewStyle().Foreground(colour("82")).Bold(true)
	dimStyle   = lipgloss.NewStyle().Foreground(colour("240"))
	errorStyle = lipgloss.NewStyle().Foreground(colour("196"))
)

// ──────────────────────────────────────────────────────────────────────────────
// Per-group counters (shared between goroutines and the TUI model)
// ──────────────────────────────────────────────────────────────────────────────

type groupCounters struct {
	total     int
	completed atomic.Int64
	failed    atomic.Int64
}

// ──────────────────────────────────────────────────────────────────────────────
// Bubble Tea model
// ──────────────────────────────────────────────────────────────────────────────

type (
	tickMsg time.Time
	doneMsg struct{}
)

type model struct {
	groups    [numGroups]*groupCounters
	total     int
	startedAt time.Time
	finished  bool
	quitting  bool
	doneCh    <-chan struct{}

	// bar width in characters
	barWidth int
}

func newModel(groups [numGroups]*groupCounters, total int, doneCh <-chan struct{}) *model {
	return &model{
		groups:    groups,
		total:     total,
		startedAt: time.Now(),
		barWidth:  40,
		doneCh:    doneCh,
	}
}

func (m *model) Init() tea.Cmd {
	return tea.Batch(tickCmd(), waitForDone(m.doneCh))
}

func tickCmd() tea.Cmd {
	return tea.Tick(200*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func waitForDone(ch <-chan struct{}) tea.Cmd {
	return func() tea.Msg {
		<-ch
		return doneMsg{}
	}
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		_ = msg
		if msg.String() == "q" || msg.String() == "ctrl+c" {
			m.quitting = true
			return m, tea.Quit
		}
	case tickMsg:
		return m, tickCmd()
	case doneMsg:
		m.finished = true
		return m, tea.Quit
	}
	return m, nil
}

func (m *model) View() string {
	var sb strings.Builder

	elapsed := time.Since(m.startedAt).Round(time.Second)

	sb.WriteString("\n")
	sb.WriteString(titleStyle.Render("  Priority & Fairness Demo") + "\n")
	sb.WriteString(dimStyle.Render(fmt.Sprintf("  %d workflows · %d priority groups · %s elapsed",
		m.total, numGroups, elapsed)) + "\n\n")

	totalCompleted := int64(0)

	for i, g := range m.groups {
		completed := g.completed.Load()
		failed := g.failed.Load()
		totalCompleted += completed

		frac := float64(completed) / float64(g.total)
		filled := int(frac * float64(m.barWidth))
		if filled > m.barWidth {
			filled = m.barWidth
		}

		bar := strings.Repeat("=", filled)
		empty := strings.Repeat(" ", m.barWidth-filled)

		label := groupLabelStyle.Render(fmt.Sprintf("  Group %d:", i+1))
		filledBar := barStyles[i].Render(bar)

		countInfo := countStyle.Render(fmt.Sprintf(" %d/%d", completed, g.total))
		if failed > 0 {
			countInfo += " " + errorStyle.Render(fmt.Sprintf("(%d err)", failed))
		}

		sb.WriteString(label + filledBar + dimStyle.Render(">"+empty) + countInfo + "\n")
	}

	sb.WriteString("\n")

	if m.finished {
		sb.WriteString(doneStyle.Render("  ✓ All complete!") + "\n")
	} else {
		remaining := int64(m.total) - totalCompleted
		sb.WriteString(dimStyle.Render(fmt.Sprintf("  %d remaining · press q to quit", remaining)) + "\n")
	}
	sb.WriteString("\n")

	return sb.String()
}

// ──────────────────────────────────────────────────────────────────────────────
// Workflow launching
// ──────────────────────────────────────────────────────────────────────────────

// groupPriority holds the Temporal priority settings for one customer group.
type groupPriority struct {
	priorityKey int
	fairnessKey string
	weight      float32
}

// groupPriorities defines how work is scheduled across the five groups.
//
// Group 1 (mission-critical) uses a higher priority (priorityKey=1),
// so it is scheduled ahead of all other work.
//
// Groups 2–5 share priorityKey=2, forming a common lower-priority lane.
// Within that lane, each group has its own fairnessKey and weight.
// The weights control how throughput is split between groups under load,
// with higher-tier groups receiving a larger share.
var groupPriorities = [numGroups]groupPriority{
	{priorityKey: 1, fairnessKey: "group-1-mission-critical", weight: 1},
	{priorityKey: 2, fairnessKey: "group-2-paid", weight: 8},
	{priorityKey: 2, fairnessKey: "group-3-standard", weight: 4},
	{priorityKey: 2, fairnessKey: "group-4-hobby", weight: 2},
	{priorityKey: 2, fairnessKey: "group-5-free", weight: 1},
}

func launchGroup(
	ctx context.Context,
	c client.Client,
	groupIdx int, // 0-based
	count int,
	counters *groupCounters,
	wg *sync.WaitGroup,
) {
	groupNum := groupIdx + 1 // 1–5
	p := groupPriorities[groupIdx]

	for i := range count {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			wfID := fmt.Sprintf("priority-fairness-g%d-%d-%d", groupNum, idx, time.Now().UnixNano())

			opts := client.StartWorkflowOptions{
				ID:        wfID,
				TaskQueue: taskQueue,
				Priority: sdktemporal.Priority{
					PriorityKey:    p.priorityKey,
					FairnessKey:    p.fairnessKey,
					FairnessWeight: p.weight,
				},
			}

			we, err := c.ExecuteWorkflow(ctx, opts, workflowType, map[string]any{
				"priorityKey":    p.priorityKey,
				"fairnessKey":    p.fairnessKey,
				"fairnessWeight": p.weight,
			})
			if err != nil {
				log.Error().Err(err).
					Int("group", groupNum).
					Int("index", idx).
					Msg("Failed to start workflow")
				counters.failed.Add(1)
				return
			}

			var result any
			if err := we.Get(ctx, &result); err != nil {
				log.Error().Err(err).
					Str("workflowId", we.GetID()).
					Msg("Workflow failed")
				counters.failed.Add(1)
				return
			}

			counters.completed.Add(1)
		}(i)
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// Main
// ──────────────────────────────────────────────────────────────────────────────

func exec() error {
	n := flag.Int("n", 1000, "total number of workflows to run (will be split evenly across 5 priority groups)")
	flag.Parse()

	total := max(*n, numGroups)

	perGroup := total / numGroups
	remainder := total % numGroups

	// Silence zerolog during the TUI — it would corrupt the output.
	// Errors are tracked via counters instead.
	zerolog.SetGlobalLevel(zerolog.Disabled)

	c, err := temporal.NewConnectionWithEnvvars(
		temporal.WithZerolog(&log.Logger),
	)
	if err != nil {
		return gh.FatalError{
			Cause: err,
			Msg:   "Unable to create client",
		}
	}
	defer c.Close()

	// Build per-group counters. The first `remainder` groups get one extra job
	// so that the total is exactly n.
	var groups [numGroups]*groupCounters
	for i := range numGroups {
		count := perGroup
		if i < remainder {
			count++
		}
		groups[i] = &groupCounters{total: count}
	}

	// doneCh is closed when all goroutines finish.
	doneCh := make(chan struct{})

	ctx := context.Background()
	var wg sync.WaitGroup

	// Launch all groups concurrently. Lower-priority groups will queue behind
	// higher-priority ones but fairness ensures they still make progress.
	for i, g := range groups {
		launchGroup(ctx, c, i, g.total, g, &wg)
	}

	// Close doneCh once everything finishes.
	go func() {
		wg.Wait()
		close(doneCh)
	}()

	p := tea.NewProgram(newModel(groups, total, doneCh))
	if _, err := p.Run(); err != nil {
		return gh.FatalError{
			Cause: err,
			Msg:   "TUI error",
		}
	}

	return nil
}

func main() {
	if err := exec(); err != nil {
		os.Exit(gh.HandleFatalError(err))
	}
}
