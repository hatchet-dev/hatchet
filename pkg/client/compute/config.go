package compute

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"

	"github.com/hatchet-dev/hatchet/pkg/client/cloud/rest"
)

// Region represents a managed worker region
type Region = rest.ManagedWorkerRegion
type GPUKind = rest.CreateManagedWorkerRuntimeConfigRequestGpuKind

// Compute is the base struct for different compute configurations.
type Compute struct {
	Pool        *string  `json:"pool,omitempty" validate:"omitempty"`
	NumReplicas int      `json:"numReplicas" validate:"min=0,max=1000"`
	Regions     []Region `json:"regions,omitempty" validate:"omitempty"`
	CPUs        int      `json:"cpus" validate:"min=1,max=64"`
	CPUKind     CPUKind  `json:"computeKind" validate:"required,oneof=shared performance"`
	MemoryMB    int      `json:"memoryMb" validate:"min=256,max=65536"`
	// GPU-specific fields
	GPUKind *GPUKind `json:"gpuKind,omitempty" validate:"omitempty"`
	GPUs    *int     `json:"gpus,omitempty" validate:"omitempty,min=1,max=8"`
}

// CPUKind represents the type of compute.
type CPUKind string

const (
	ComputeKindSharedCPU      CPUKind = "shared"
	ComputeKindPerformanceCPU CPUKind = "performance"
)

// ComputeHash generates a SHA256 hash of the Compute configuration.
func (c *Compute) ComputeHash() (string, error) {
	str, err := json.Marshal(c)
	if err != nil {
		return "", err
	}
	hash := sha256.Sum256(str)
	return fmt.Sprintf("%x", hash), nil
}
