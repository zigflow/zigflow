package tasks

import (
	"fmt"

	"github.com/serverlessworkflow/sdk-go/v3/model"
	"github.com/zigflow/zigflow/pkg/cloudevents"
	"go.temporal.io/sdk/worker"
)

func NewCallCatalogTaskBuilder(
	temporalWorker worker.Worker,
	task *model.CallFunction,
	taskName string,
	doc *model.Workflow,
	emitter *cloudevents.Events,
	taskOpts *TaskOpts,
) (TaskBuilder, error) {
	if task.Call != customCallFunctionCatalog {
		return nil, fmt.Errorf("unsupported call task '%s' for catalog builder", task.Call)
	}

	return nil, nil
}
