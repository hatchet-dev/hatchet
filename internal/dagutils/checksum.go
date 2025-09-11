package dagutils

import (
	"github.com/hatchet-dev/hatchet/internal/datautils"
	"github.com/hatchet-dev/hatchet/internal/digest"
	"github.com/hatchet-dev/hatchet/pkg/repository"
)

func Checksum(opts *repository.CreateWorkflowVersionOpts) (string, error) {
	// ensure no cycles
	for i, job := range opts.Jobs {
		if HasCycle(job.Steps) {
			return "", &repository.JobRunHasCycleError{
				JobName: job.Name,
			}
		}

		var err error
		opts.Jobs[i].Steps, err = OrderWorkflowSteps(job.Steps)

		if err != nil {
			return "", err
		}
	}

	// compute a checksum for the workflow
	declaredValues, err := datautils.ToJSONMap(opts)

	if err != nil {
		return "", err
	}

	workflowChecksum, err := digest.DigestValues(declaredValues)

	if err != nil {
		return "", err
	}

	return workflowChecksum.String(), nil
}
