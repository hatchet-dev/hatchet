package dagutils

import (
	"encoding/json"

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

	optsBytes, err := json.Marshal(opts)

	if err != nil {
		return "", err
	}

	workflowChecksum, err := digest.DigestBytes(optsBytes)

	if err != nil {
		return "", err
	}

	return workflowChecksum.String(), nil
}
