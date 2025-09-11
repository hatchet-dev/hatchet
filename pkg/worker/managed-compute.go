package worker

import (
	"context"
	"os"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/pkg/client"
	"github.com/hatchet-dev/hatchet/pkg/client/cloud/rest"
)

type ManagedCompute struct {
	ActionRegistry  *ActionRegistry
	Client          client.Client
	MaxRuns         int
	RuntimeConfigs  []rest.CreateManagedWorkerRuntimeConfigRequest
	CloudRegisterID *string
	Logger          *zerolog.Logger
}

func NewManagedCompute(actionRegistry *ActionRegistry, client client.Client, maxRuns int) *ManagedCompute {
	if maxRuns == 0 {
		maxRuns = 1
	}

	runtimeConfigs := getComputeConfigs(actionRegistry, maxRuns)
	cloudRegisterID := client.CloudRegisterID()

	mc := &ManagedCompute{
		ActionRegistry:  actionRegistry,
		Client:          client,
		MaxRuns:         maxRuns,
		RuntimeConfigs:  runtimeConfigs,
		CloudRegisterID: cloudRegisterID,
		Logger:          client.Logger(),
	}

	if len(mc.RuntimeConfigs) == 0 {
		mc.Logger.Debug().Msg("No compute configs found, skipping cloud registration and running all actions locally.")
		return mc
	}

	if mc.CloudRegisterID == nil {
		mc.Logger.Warn().Msg("Managed cloud compute plan:")
		for _, runtimeConfig := range mc.RuntimeConfigs {
			mc.Logger.Warn().Msg("    ----------------------------")
			mc.Logger.Warn().Msgf("    actions: %v", *runtimeConfig.Actions)
			mc.Logger.Warn().Msgf("    num replicas: %d", runtimeConfig.NumReplicas)
			mc.Logger.Warn().Msgf("    cpu kind: %s", runtimeConfig.CpuKind)
			mc.Logger.Warn().Msgf("    cpus: %d", runtimeConfig.Cpus)
			mc.Logger.Warn().Msgf("    memory mb: %d", runtimeConfig.MemoryMb)
			mc.Logger.Warn().Msgf("    regions: %v", runtimeConfig.Regions)
		}

		mc.Logger.Warn().Msg("NOTICE: local mode detected, skipping cloud registration and running all actions locally.")

		return mc
	}

	// Register the cloud compute plan
	mc.CloudRegister(context.Background())

	return mc
}

func getComputeConfigs(actions *ActionRegistry, maxRuns int) []rest.CreateManagedWorkerRuntimeConfigRequest {
	computeMap := make(map[string]rest.CreateManagedWorkerRuntimeConfigRequest)

	for action, details := range *actions {
		compute := details.Compute()

		if compute == nil {
			continue
		}

		key, err := compute.ComputeHash()

		if err != nil {
			panic(err)
		}

		if _, exists := computeMap[key]; !exists {
			computeMap[key] = rest.CreateManagedWorkerRuntimeConfigRequest{
				Actions:     &[]string{},
				NumReplicas: compute.NumReplicas,
				CpuKind:     string(compute.CPUKind),
				Cpus:        compute.CPUs,
				MemoryMb:    compute.MemoryMB,
				Regions:     &compute.Regions,
				Slots:       &maxRuns,
				Gpus:        compute.GPUs,
				GpuKind:     compute.GPUKind,
			}
		}

		*computeMap[key].Actions = append(*computeMap[key].Actions, action)
	}

	var configs []rest.CreateManagedWorkerRuntimeConfigRequest
	for _, config := range computeMap {
		configs = append(configs, config)
	}

	return configs
}

func (mc *ManagedCompute) CloudRegister(ctx context.Context) {
	if mc.CloudRegisterID != nil {
		mc.Logger.Info().Msg("Registering cloud compute plan with ID: " + *mc.CloudRegisterID)

		if len(mc.RuntimeConfigs) == 0 {
			mc.Logger.Warn().Msg("No actions to register, skipping cloud registration.")
			os.Exit(0)
		}

		req := rest.InfraAsCodeRequest{
			RuntimeConfigs: mc.RuntimeConfigs,
		}

		_, err := mc.Client.CloudAPI().InfraAsCodeCreateWithResponse(ctx, uuid.MustParse(*mc.CloudRegisterID), req)

		if err != nil {
			mc.Logger.Error().Err(err).Msg("Could not register cloud compute plan.")
			os.Exit(1)
		}

		os.Exit(0)
	}
}
