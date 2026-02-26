/*
 * Copyright 2025 - 2026 Zigflow authors <https://github.com/mrsimonemms/zigflow/graphs/contributors>
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

package graph

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/serverlessworkflow/sdk-go/v3/model"
)

type mermaidGenerator struct{}

func (g *mermaidGenerator) Generate(wf *model.Workflow) (string, error) {
	b := &mermaidBuilder{
		seenKeys: make(map[string]int),
	}
	b.linef(0, "flowchart TD")
	b.generate(wf)
	return b.build(), nil
}

// nodeResult holds the entry and exit node IDs for a rendered task.
// For most tasks entry == exit. Tasks with split control flow (Fork, Try)
// expose different entry and exit nodes for sequential chaining.
type nodeResult struct {
	entryID string
	exitID  string
}

// crossEdge is an edge that crosses subgraph boundaries, collected during
// rendering and emitted after all subgraphs are declared.
type crossEdge struct {
	fromID string
	toID   string
	label  string
}

type mermaidBuilder struct {
	lines      []string
	crossEdges []crossEdge
	seenKeys   map[string]int // "ctx:key" → times seen, for deduplication
}

func (b *mermaidBuilder) linef(indent int, format string, args ...any) {
	prefix := strings.Repeat("    ", indent)
	b.lines = append(b.lines, prefix+fmt.Sprintf(format, args...))
}

func (b *mermaidBuilder) build() string {
	return strings.Join(b.lines, "\n") + "\n"
}

// nodeID returns a unique Mermaid-safe node ID for the given context and task key.
// Duplicate keys within the same context get a numeric suffix (_2, _3, …).
func (b *mermaidBuilder) nodeID(ctx, key string) string {
	full := ctx + ":" + key
	count := b.seenKeys[full]
	b.seenKeys[full] = count + 1
	base := sanitizeID(ctx) + "_" + sanitizeID(key)
	if count > 0 {
		return fmt.Sprintf("%s_%d", base, count+1)
	}
	return base
}

// subWFStartID returns the deterministic start-node ID for a named sub-workflow.
// Must be consistent whether called from the main flow (to build cross-edges)
// or from the sub-workflow's own subgraph render.
func subWFStartID(name string) string {
	return "wf_" + sanitizeID(name) + "__start"
}

func subWFEndID(name string) string {
	return "wf_" + sanitizeID(name) + "__end"
}

var nonAlphanumRe = regexp.MustCompile(`[^a-zA-Z0-9]`)

func sanitizeID(s string) string {
	result := nonAlphanumRe.ReplaceAllString(s, "_")
	if result == "" {
		return "n"
	}
	return result
}

// escapeLabel sanitises a string for use inside a Mermaid quoted node label.
// Double-quotes are replaced with single-quotes; long strings are truncated.
func escapeLabel(s string) string {
	s = strings.ReplaceAll(s, `"`, `'`)
	if len(s) > 50 {
		s = s[:47] + "..."
	}
	return s
}

// nodeLabel builds the standard task label: TASK_TYPE (task_name), with an
// optional conditional suffix appended before escaping.
func nodeLabel(taskType, key, cond string) string {
	return escapeLabel(taskType + " (" + key + ")" + cond)
}

// emitNode emits a single Mermaid node declaration, selecting the correct
// shape syntax from info.Shape. All task-specific strings come from NodeInfo;
// mermaid.go itself contains no task type names.
func (b *mermaidBuilder) emitNode(indent int, id string, info NodeInfo, key, cond string) {
	label := nodeLabel(info.TypeName, key, cond)
	switch info.Shape {
	case ShapeDiamond:
		b.linef(indent, `%s{"%s"}`, id, label)
	case ShapeSubroutine:
		b.linef(indent, `%s[["%s"]]`, id, label)
	default:
		b.linef(indent, `%s["%s"]`, id, label)
	}
}

// ── Workflow topology helpers ────────────────────────────────────────────────

// isAllDoTasks reports whether every item in the list is a DoTask.
func isAllDoTasks(tasks *model.TaskList) bool {
	if tasks == nil {
		return false
	}
	for _, item := range *tasks {
		if item.AsDoTask() == nil {
			return false
		}
	}
	return true
}

// splitTasks separates a task list into main-flow tasks and sub-workflow tasks,
// replicating the logic from task_builder_do.go:
//
//   - Non-DoTask items → always part of the main flow (sets hasNoDo = true)
//   - DoTask before any non-DoTask → part of the main flow (inline)
//   - DoTask after a non-DoTask → sub-workflow (registered separately by the runtime)
func splitTasks(tasks *model.TaskList) (main, subWFs []*model.TaskItem) {
	var hasNoDo bool
	for _, item := range *tasks {
		if item.AsDoTask() == nil {
			hasNoDo = true
			main = append(main, item)
		} else if hasNoDo {
			subWFs = append(subWFs, item)
		} else {
			main = append(main, item)
		}
	}
	return
}

// makeSubWFSet builds a name→bool lookup from a slice of sub-workflow items.
func makeSubWFSet(items []*model.TaskItem) map[string]bool {
	m := make(map[string]bool, len(items))
	for _, item := range items {
		m[item.Key] = true
	}
	return m
}

// sliceToTaskList wraps a []*TaskItem into a *model.TaskList.
func sliceToTaskList(items []*model.TaskItem) *model.TaskList {
	tl := model.TaskList(make([]*model.TaskItem, len(items)))
	copy(tl, items)
	return &tl
}

// ── Top-level generation ─────────────────────────────────────────────────────

func (b *mermaidBuilder) generate(wf *model.Workflow) {
	if wf.Do == nil || len(*wf.Do) == 0 {
		return
	}

	if isAllDoTasks(wf.Do) {
		// Multiple independent workflows: each top-level DoTask is its own
		// Temporal workflow. Show each as a separate subgraph.
		for _, item := range *wf.Do {
			if do := item.AsDoTask(); do != nil {
				b.renderSubgraphFlow(1, item.Key, item.Key, do.Do, nil)
			}
		}
	} else {
		mainTasks, subWFItems := splitTasks(wf.Do)
		swSet := makeSubWFSet(subWFItems)

		if len(subWFItems) > 0 {
			// Render the main flow in a named subgraph, followed by each
			// sub-workflow's subgraph. Cross-edges link switch branches etc.
			b.renderSubgraphFlow(1, wf.Document.Name, wf.Document.Name, sliceToTaskList(mainTasks), swSet)
			for _, item := range subWFItems {
				if do := item.AsDoTask(); do != nil {
					b.renderSubgraphFlow(1, item.Key, item.Key, do.Do, nil)
				}
			}
		} else {
			// Simple single workflow: flat, no outer subgraph.
			b.renderFlatFlow(1, wf.Document.Name, mainTasks, nil)
		}
	}

	// Emit cross-edges after all subgraph declarations.
	for _, e := range b.crossEdges {
		b.linef(1, `%s -->|"%s"| %s`, e.fromID, escapeLabel(e.label), e.toID)
	}
}

// ── Subgraph-wrapped rendering ───────────────────────────────────────────────

func (b *mermaidBuilder) renderSubgraphFlow(
	indent int, ctx, label string, tasks *model.TaskList, swNames map[string]bool,
) {
	safeCtx := sanitizeID(ctx)
	b.linef(indent, `subgraph wf_%s["%s"]`, safeCtx, escapeLabel(label))
	b.linef(indent+1, "direction TB")

	startID := subWFStartID(ctx)
	endID := subWFEndID(ctx)
	b.linef(indent+1, `%s([Start])`, startID)
	b.linef(indent+1, `%s([End])`, endID)

	if tasks == nil || len(*tasks) == 0 {
		b.linef(indent+1, "%s --> %s", startID, endID)
	} else {
		prevExitID := startID
		for _, item := range *tasks {
			r := b.renderTask(indent+1, ctx, item, swNames, endID)
			b.linef(indent+1, "%s --> %s", prevExitID, r.entryID)
			prevExitID = r.exitID
		}
		b.linef(indent+1, "%s --> %s", prevExitID, endID)
	}

	b.linef(indent, "end")
}

// ── Flat rendering (no subgraph wrapper) ─────────────────────────────────────

func (b *mermaidBuilder) renderFlatFlow(
	indent int, ctx string, tasks []*model.TaskItem, swNames map[string]bool,
) {
	startID := sanitizeID(ctx) + "__start"
	endID := sanitizeID(ctx) + "__end"
	b.linef(indent, `%s([Start])`, startID)
	b.linef(indent, `%s([End])`, endID)

	if len(tasks) == 0 {
		b.linef(indent, "%s --> %s", startID, endID)
		return
	}

	prevExitID := startID
	for _, item := range tasks {
		r := b.renderTask(indent, ctx, item, swNames, endID)
		b.linef(indent, "%s --> %s", prevExitID, r.entryID)
		prevExitID = r.exitID
	}
	b.linef(indent, "%s --> %s", prevExitID, endID)
}

// ── Per-task rendering ───────────────────────────────────────────────────────

func (b *mermaidBuilder) renderTask(
	indent int, ctx string, item *model.TaskItem,
	swNames map[string]bool, ctxEndID string,
) nodeResult {
	base := item.GetBase()
	condSuffix := ""
	if base.If != nil {
		condSuffix = " [?]"
	}

	switch {
	case item.AsDoTask() != nil:
		return b.renderDoInline(indent, ctx, item, swNames, ctxEndID)
	case item.AsTryTask() != nil:
		return b.renderTry(indent, ctx, item)
	case item.AsSwitchTask() != nil:
		return b.renderSwitch(indent, ctx, item, swNames, ctxEndID)
	case item.AsForkTask() != nil:
		return b.renderFork(indent, ctx, item)
	case item.AsForTask() != nil:
		return b.renderFor(indent, ctx, item)
	default:
		info, ok := nodeInfoFrom(item)
		if !ok {
			// Truly unknown task type: fall back to a plain rect with the key.
			id := b.nodeID(ctx, item.Key)
			b.linef(indent, `%s["%s"]`, id, escapeLabel(item.Key+condSuffix))
			return nodeResult{id, id}
		}
		return b.renderLeaf(indent, ctx, item, info, condSuffix)
	}
}

// renderLeaf handles all single-node tasks via NodeInfo — one code path for
// every leaf task type.
func (b *mermaidBuilder) renderLeaf(
	indent int, ctx string, item *model.TaskItem, info NodeInfo, cond string,
) nodeResult {
	id := b.nodeID(ctx, item.Key)
	b.emitNode(indent, id, info, item.Key, cond)
	return nodeResult{id, id}
}

// ── Compound task renderers ──────────────────────────────────────────────────

// renderDoInline expands a DoTask's children inline in the current flow.
func (b *mermaidBuilder) renderDoInline(
	indent int, ctx string, item *model.TaskItem,
	swNames map[string]bool, ctxEndID string,
) nodeResult {
	do := item.AsDoTask()
	if do.Do == nil || len(*do.Do) == 0 {
		id := b.nodeID(ctx, item.Key)
		b.linef(indent, `%s["%s"]`, id, escapeLabel(item.Key))
		return nodeResult{id, id}
	}

	var entryID, prevExitID string
	for _, child := range *do.Do {
		r := b.renderTask(indent, ctx, child, swNames, ctxEndID)
		if entryID == "" {
			entryID = r.entryID
		}
		if prevExitID != "" {
			b.linef(indent, "%s --> %s", prevExitID, r.entryID)
		}
		prevExitID = r.exitID
	}
	return nodeResult{entryID, prevExitID}
}

// renderSwitch renders a SwitchTask as a diamond with a labelled edge per case.
// Cases whose `then` targets a sub-workflow are recorded as cross-edges and
// emitted after all subgraphs.
func (b *mermaidBuilder) renderSwitch(
	indent int, ctx string, item *model.TaskItem,
	swNames map[string]bool, ctxEndID string,
) nodeResult {
	task := item.AsSwitchTask()
	info, _ := nodeInfoFrom(item)
	id := b.nodeID(ctx, item.Key)
	b.emitNode(indent, id, info, item.Key, "")

	for _, switchItem := range task.Switch {
		for _, switchCase := range switchItem {
			if switchCase.Then == nil {
				continue
			}
			then := switchCase.Then

			edgeLabel := "default"
			if switchCase.When != nil {
				edgeLabel = switchCase.When.Value
			}

			switch {
			case then.IsTermination():
				b.linef(indent, `%s -->|"%s"| %s`, id, escapeLabel(edgeLabel), ctxEndID)
			case !then.IsEnum():
				// A task or workflow name.
				target := then.Value
				if swNames != nil && swNames[target] {
					// Cross-subgraph edge to a sub-workflow.
					b.crossEdges = append(b.crossEdges, crossEdge{
						fromID: id,
						toID:   subWFStartID(target),
						label:  edgeLabel,
					})
				} else {
					// Intra-flow edge to another task in the same context.
					// Use the predictive base ID for the first occurrence of that key.
					targetID := sanitizeID(ctx) + "_" + sanitizeID(target)
					b.linef(indent, `%s -->|"%s"| %s`, id, escapeLabel(edgeLabel), targetID)
				}
			}
			// "continue" → sequential flow handles it; no explicit edge needed.
		}
	}
	return nodeResult{id, id}
}

// renderFork renders a ForkTask with fan-out subgraphs for each branch.
func (b *mermaidBuilder) renderFork(
	indent int, ctx string, item *model.TaskItem,
) nodeResult {
	task := item.AsForkTask()
	info, _ := nodeInfoFrom(item)
	id := b.nodeID(ctx, item.Key)
	cond := ""
	if task.Fork.Compete {
		cond = " 🏁"
	}
	b.emitNode(indent, id, info, item.Key, cond)

	// Join node: all branches reconverge here.
	joinID := sanitizeID(ctx) + "_" + sanitizeID(item.Key) + "__join"
	b.linef(indent, `%s((" "))`, joinID)

	if task.Fork.Branches != nil {
		for _, branch := range *task.Fork.Branches {
			branchCtx := ctx + "_" + branch.Key
			safeCtx := sanitizeID(branchCtx)

			b.linef(indent, `subgraph fork_%s["%s"]`, safeCtx, escapeLabel(branch.Key))
			b.linef(indent+1, "direction TB")

			branchStartID := safeCtx + "__start"
			branchEndID := safeCtx + "__end"
			b.linef(indent+1, `%s([ ])`, branchStartID)
			b.linef(indent+1, `%s([ ])`, branchEndID)

			if do := branch.AsDoTask(); do != nil && do.Do != nil && len(*do.Do) > 0 {
				prevID := branchStartID
				for _, child := range *do.Do {
					r := b.renderTask(indent+1, branchCtx, child, nil, branchEndID)
					b.linef(indent+1, "%s --> %s", prevID, r.entryID)
					prevID = r.exitID
				}
				b.linef(indent+1, "%s --> %s", prevID, branchEndID)
			} else {
				b.linef(indent+1, "%s --> %s", branchStartID, branchEndID)
			}

			b.linef(indent, "end")
			b.linef(indent, "%s --> %s", id, branchStartID)
			b.linef(indent, "%s --> %s", branchEndID, joinID)
		}
	}

	return nodeResult{id, joinID}
}

// renderFor renders a ForTask with a subgraph for the loop body and a back-edge
// to show the iteration.
func (b *mermaidBuilder) renderFor(
	indent int, ctx string, item *model.TaskItem,
) nodeResult {
	task := item.AsForTask()
	info, _ := nodeInfoFrom(item)
	id := b.nodeID(ctx, item.Key)
	b.emitNode(indent, id, info, item.Key, "")

	if task.Do != nil && len(*task.Do) > 0 {
		bodyCtx := ctx + "_" + item.Key + "_body"
		safeBody := sanitizeID(bodyCtx)

		b.linef(indent, `subgraph body_%s["%s (loop body)"]`, sanitizeID(item.Key), escapeLabel(item.Key))
		b.linef(indent+1, "direction TB")

		bodyStartID := safeBody + "__start"
		b.linef(indent+1, `%s([ ])`, bodyStartID)

		prevID := bodyStartID
		var lastExitID string
		for _, child := range *task.Do {
			r := b.renderTask(indent+1, bodyCtx, child, nil, "")
			b.linef(indent+1, "%s --> %s", prevID, r.entryID)
			prevID = r.exitID
			lastExitID = r.exitID
		}

		b.linef(indent, "end")
		b.linef(indent, "%s --> %s", id, bodyStartID)
		if lastExitID != "" {
			b.linef(indent, `%s -->|"next iteration"| %s`, lastExitID, id)
		}
	}

	return nodeResult{id, id}
}

// renderTry renders a TryTask with separate try and catch subgraphs.
// The happy-path exit is the end of the try block; the catch block is shown as
// a side branch connected via a dashed error edge.
func (b *mermaidBuilder) renderTry(
	indent int, ctx string, item *model.TaskItem,
) nodeResult {
	task := item.AsTryTask()

	tryCtx := ctx + "_" + item.Key + "_try"
	safeTry := sanitizeID(tryCtx)

	b.linef(indent, `subgraph try_%s["TRY (%s)"]`, sanitizeID(item.Key), escapeLabel(item.Key))
	b.linef(indent+1, "direction TB")

	tryStartID := safeTry + "__start"
	tryEndID := safeTry + "__end"
	b.linef(indent+1, `%s([ ])`, tryStartID)
	b.linef(indent+1, `%s([ ])`, tryEndID)

	if task.Try != nil && len(*task.Try) > 0 {
		prevID := tryStartID
		var lastExitID string
		for _, child := range *task.Try {
			r := b.renderTask(indent+1, tryCtx, child, nil, tryEndID)
			b.linef(indent+1, "%s --> %s", prevID, r.entryID)
			prevID = r.exitID
			lastExitID = r.exitID
		}
		b.linef(indent+1, "%s --> %s", lastExitID, tryEndID)
	} else {
		b.linef(indent+1, "%s --> %s", tryStartID, tryEndID)
	}

	b.linef(indent, "end")

	// Catch block (shown as a side branch for the error path).
	if task.Catch != nil && task.Catch.Do != nil && len(*task.Catch.Do) > 0 {
		catchCtx := ctx + "_" + item.Key + "_catch"
		safeCatch := sanitizeID(catchCtx)

		b.linef(indent, `subgraph catch_%s["CATCH (%s)"]`, sanitizeID(item.Key), escapeLabel(item.Key))
		b.linef(indent+1, "direction TB")

		catchStartID := safeCatch + "__start"
		catchEndID := safeCatch + "__end"
		b.linef(indent+1, `%s([ ])`, catchStartID)
		b.linef(indent+1, `%s([ ])`, catchEndID)

		prevID := catchStartID
		var lastExitID string
		for _, child := range *task.Catch.Do {
			r := b.renderTask(indent+1, catchCtx, child, nil, catchEndID)
			b.linef(indent+1, "%s --> %s", prevID, r.entryID)
			prevID = r.exitID
			lastExitID = r.exitID
		}
		b.linef(indent+1, "%s --> %s", lastExitID, catchEndID)
		b.linef(indent, "end")

		// Dashed error edge from try end to catch start.
		b.linef(indent, `%s -.->|"on error"| %s`, tryEndID, catchStartID)
	}

	// Happy-path exit is the try-block end node.
	return nodeResult{tryStartID, tryEndID}
}
