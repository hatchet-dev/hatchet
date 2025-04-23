import { Label } from '@/next/components/ui/label';
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/next/components/ui/card';
import { Button } from '@/next/components/ui/button';
import { regions, DefaultMachineTypes } from './types';
import { CreateManagedWorkerRuntimeConfigRequest } from '@/lib/api/generated/cloud/data-contracts';
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
} from '@/next/components/ui/command';
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/next/components/ui/popover';
import { useState } from 'react';
import useCan from '@/next/hooks/use-can';
import { managedCompute } from '@/next/lib/can/features/managed-compute.permissions';
import {
  Tabs,
  TabsContent,
  TabsList,
  TabsTrigger,
} from '@/next/components/ui/tabs';
import { Input } from '@/next/components/ui/input';
import { Checkbox } from '@/next/components/ui/checkbox';
import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger,
} from '@/next/components/ui/accordion';

export type ScalingType = 'Autoscaling' | 'Static';
export const scalingTypes: ScalingType[] = ['Static', 'Autoscaling'];

export interface MachineConfigValue
  extends CreateManagedWorkerRuntimeConfigRequest {}

export interface MachineConfigProps {
  config: MachineConfigValue;
  setConfig: (config: MachineConfigValue) => void;
  actions?: React.ReactNode;
  type?: 'create' | 'update';
}

export function MachineConfig({
  config,
  setConfig,
  actions,
  type = 'create',
}: MachineConfigProps) {
  const selectedMachineType = DefaultMachineTypes.find(
    (type) => type.cpuKind === config.cpuKind,
  );

  const { can } = useCan();

  const [openRegion, setOpenRegion] = useState(false);
  const [openMachineType, setOpenMachineType] = useState(false);
  const [scalingType, setScalingType] = useState<ScalingType>(
    config.autoscaling ? 'Autoscaling' : 'Static',
  );

  const selectedRegion = regions.find((i) => i.value === config.regions?.[0]);

  // Get max replicas based on plan
  const getMaxReplicas = () => {
    if (!can) {
      return Infinity;
    }

    // Check maximum replicas for each plan
    for (let i = 1; i <= 20; i++) {
      const allowed = can(managedCompute.maxReplicas(i));
      if (!allowed) {
        return i - 1; // Return the last valid replica count
      }
    }
    return 20; // Default to maximum
  };

  const handleScalingTypeChange = (value: ScalingType) => {
    setScalingType(value);
    if (value === 'Static') {
      setConfig({
        ...config,
        numReplicas: 1,
        autoscaling: undefined,
      });
    } else {
      const maxReplicas = getMaxReplicas();
      setConfig({
        ...config,
        numReplicas: undefined,
        autoscaling: {
          waitDuration: '1m',
          rollingWindowDuration: '2m',
          utilizationScaleUpThreshold: 0.75,
          utilizationScaleDownThreshold: 0.25,
          increment: 1,
          scaleToZero: true,
          minAwakeReplicas: Math.min(1, maxReplicas),
          maxReplicas: maxReplicas,
          fly: {
            autoscalingKey: 'dashboard',
            currentReplicas: 1,
          },
        },
      });
    }
  };

  return (
    <Card variant={type === 'update' ? 'borderless' : 'default'}>
      <CardHeader>
        <CardTitle>Machine Configuration</CardTitle>
        <CardDescription>
          Configure the compute resources for your worker service.
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="space-y-2">
          <Label htmlFor="region">Region</Label>
          <Popover open={openRegion} onOpenChange={setOpenRegion}>
            <PopoverTrigger asChild>
              <Button
                variant="outline"
                role="combobox"
                aria-expanded={openRegion}
                className="w-full justify-between"
              >
                {selectedRegion ? (
                  <div className="flex items-center gap-2">
                    <span>{selectedRegion.emoji}</span>
                    <span>{selectedRegion.name}</span>
                  </div>
                ) : (
                  <span>Select region</span>
                )}
              </Button>
            </PopoverTrigger>
            <PopoverContent className="w-full p-0">
              <Command>
                <CommandInput placeholder="Search regions..." />
                <CommandEmpty>No regions found.</CommandEmpty>
                <CommandGroup className="max-h-[300px] overflow-auto">
                  {regions.map((region) => (
                    <CommandItem
                      key={region.value}
                      onSelect={() => {
                        setConfig({
                          ...config,
                          regions: [region.value],
                        });
                        setOpenRegion(false);
                      }}
                    >
                      <div className="flex items-center gap-2">
                        <span>{region.emoji}</span>
                        <span>{region.name}</span>
                      </div>
                    </CommandItem>
                  ))}
                </CommandGroup>
              </Command>
            </PopoverContent>
          </Popover>
        </div>
        <div className="space-y-2">
          <Label htmlFor="machineType">Machine Type</Label>
          <Popover open={openMachineType} onOpenChange={setOpenMachineType}>
            <PopoverTrigger asChild>
              <Button
                variant="outline"
                role="combobox"
                aria-expanded={openMachineType}
                className="w-full justify-between"
              >
                {selectedMachineType ? (
                  <div className="flex items-center gap-2">
                    <span>{selectedMachineType.title}</span>
                    {!can(
                      managedCompute.selectCompute(selectedMachineType),
                    ) && <span>🔒</span>}
                  </div>
                ) : (
                  <span>Select machine type</span>
                )}
              </Button>
            </PopoverTrigger>
            <PopoverContent className="w-full p-0">
              <Command>
                <CommandInput placeholder="Search machine types..." />
                <CommandEmpty>No machine types found.</CommandEmpty>
                <CommandGroup className="max-h-[300px] overflow-auto">
                  {DefaultMachineTypes.map((machineType) => (
                    <CommandItem
                      key={machineType.title}
                      onSelect={() => {
                        setConfig({
                          ...config,
                          cpuKind: machineType.cpuKind,
                          cpus: machineType.cpus,
                          memoryMb: machineType.memoryMb,
                        });
                        setOpenMachineType(false);
                      }}
                    >
                      <div className="flex items-center gap-2">
                        <span>{machineType.title}</span>
                        {!can(managedCompute.selectCompute(machineType)) && (
                          <span>🔒</span>
                        )}
                      </div>
                    </CommandItem>
                  ))}
                </CommandGroup>
              </Command>
            </PopoverContent>
          </Popover>
        </div>
        <div className="space-y-2">
          <Label>Scaling Method</Label>
          <Tabs
            defaultValue="Static"
            value={scalingType}
            onValueChange={(value) =>
              handleScalingTypeChange(value as ScalingType)
            }
          >
            <TabsList>
              {scalingTypes.map((type) => (
                <TabsTrigger value={type} key={type}>
                  {type}
                </TabsTrigger>
              ))}
            </TabsList>
            <TabsContent value="Static" className="pt-4 grid gap-4">
              <Label htmlFor="numReplicas">
                Number of replicas (max: {getMaxReplicas()})
              </Label>
              <Input
                id="numReplicas"
                type="number"
                value={config.numReplicas ?? 1}
                onChange={(e) => {
                  const value = parseInt(e.target.value);
                  const validatedValue = Math.min(
                    Math.max(1, value),
                    getMaxReplicas(),
                  );
                  setConfig({
                    ...config,
                    numReplicas: validatedValue,
                  });
                }}
                min={1}
                max={getMaxReplicas()}
              />
            </TabsContent>
            <TabsContent value="Autoscaling" className="pt-4 grid gap-4 p-1">
              <Label htmlFor="minAwakeReplicas">Min Replicas</Label>
              <Input
                id="minAwakeReplicas"
                type="number"
                value={config.autoscaling?.minAwakeReplicas ?? 1}
                onChange={(e) => {
                  const minValue = parseInt(e.target.value);
                  const maxAllowed = getMaxReplicas();
                  const validatedMin = Math.min(
                    Math.max(1, minValue),
                    maxAllowed,
                  );

                  const currentConfig = { ...config };
                  if (currentConfig.autoscaling) {
                    currentConfig.autoscaling.minAwakeReplicas = validatedMin;
                    if (currentConfig.autoscaling.fly) {
                      currentConfig.autoscaling.fly.currentReplicas =
                        validatedMin;
                    }

                    // If min replicas is greater than max replicas, update max replicas
                    if (currentConfig.autoscaling.maxReplicas < validatedMin) {
                      currentConfig.autoscaling.maxReplicas = validatedMin;
                    }
                  }
                  setConfig(currentConfig);
                }}
                min={1}
                max={getMaxReplicas()}
              />
              <Label htmlFor="maxReplicas">
                Max Replicas (max: {getMaxReplicas()})
              </Label>
              <Input
                id="maxReplicas"
                type="number"
                value={config.autoscaling?.maxReplicas || 1}
                onChange={(e) => {
                  const maxValue = parseInt(e.target.value);
                  const minReplicas = config.autoscaling?.minAwakeReplicas || 1;
                  const maxAllowed = getMaxReplicas();
                  const validatedMax = Math.min(
                    Math.max(maxValue, minReplicas),
                    maxAllowed,
                  );

                  const currentConfig = { ...config };
                  if (currentConfig.autoscaling) {
                    currentConfig.autoscaling.maxReplicas = validatedMax;
                  }
                  setConfig(currentConfig);
                }}
                min={config.autoscaling?.minAwakeReplicas || 1}
                max={getMaxReplicas()}
              />
              <div className="flex flex-row gap-4 items-center">
                <Label htmlFor="scaleToZero">
                  Scale to zero during periods of inactivity?
                </Label>
                <Checkbox
                  id="scaleToZero"
                  checked={config.autoscaling?.scaleToZero || false}
                  onCheckedChange={(checked) => {
                    const currentConfig = { ...config };
                    if (currentConfig.autoscaling) {
                      currentConfig.autoscaling.scaleToZero =
                        checked as boolean;
                    }
                    setConfig(currentConfig);
                  }}
                />
              </div>
              <Accordion type="single" collapsible>
                <AccordionItem value="advanced" className="border-none">
                  <AccordionTrigger>
                    Advanced autoscaling settings
                  </AccordionTrigger>
                  <AccordionContent className="flex flex-col gap-4 p-1">
                    <Label htmlFor="waitDuration">Wait Duration</Label>
                    <div className="text-sm text-muted-foreground">
                      How long to wait between autoscaling events. For example:
                      10s (10 seconds), 5m (5 minutes), 1h (1 hour).
                    </div>
                    <Input
                      id="waitDuration"
                      value={config.autoscaling?.waitDuration || '1m'}
                      onChange={(e) => {
                        const currentConfig = { ...config };
                        if (currentConfig.autoscaling) {
                          currentConfig.autoscaling.waitDuration =
                            e.target.value;
                        }
                        setConfig(currentConfig);
                      }}
                    />
                    <Label htmlFor="rollingWindowDuration">
                      Rolling Window Duration
                    </Label>
                    <div className="text-sm text-muted-foreground">
                      The amount of time to look at utilization metrics for
                      autoscaling. Lower values will lead to faster scale-up and
                      scale-down. Example: 2m (2 minutes), 5m (5 minutes), 1h (1
                      hour).
                    </div>
                    <Input
                      id="rollingWindowDuration"
                      value={config.autoscaling?.rollingWindowDuration || '2m'}
                      onChange={(e) => {
                        const currentConfig = { ...config };
                        if (currentConfig.autoscaling) {
                          currentConfig.autoscaling.rollingWindowDuration =
                            e.target.value;
                        }
                        setConfig(currentConfig);
                      }}
                    />
                    <Label htmlFor="utilizationScaleUpThreshold">
                      Utilization Scale Up Threshold
                    </Label>
                    <div className="text-sm text-muted-foreground">
                      A value between 0 and 1 which represents the utilization
                      threshold at which to scale up. For example, 0.75 means
                      that if the utilization is above 75%, scale up.
                    </div>
                    <Input
                      id="utilizationScaleUpThreshold"
                      type="number"
                      min={0}
                      max={1}
                      step={0.01}
                      value={
                        config.autoscaling?.utilizationScaleUpThreshold || 0.75
                      }
                      onChange={(e) => {
                        const currentConfig = { ...config };
                        if (currentConfig.autoscaling) {
                          currentConfig.autoscaling.utilizationScaleUpThreshold =
                            parseFloat(e.target.value);
                        }
                        setConfig(currentConfig);
                      }}
                    />
                    <Label htmlFor="utilizationScaleDownThreshold">
                      Utilization Scale Down Threshold
                    </Label>
                    <div className="text-sm text-muted-foreground">
                      A value between 0 and 1 which represents the utilization
                      threshold at which to scale down. For example, 0.25 means
                      that if the utilization is below 25%, scale down.
                    </div>
                    <Input
                      id="utilizationScaleDownThreshold"
                      type="number"
                      min={0}
                      max={1}
                      step={0.01}
                      value={
                        config.autoscaling?.utilizationScaleDownThreshold ||
                        0.25
                      }
                      onChange={(e) => {
                        const currentConfig = { ...config };
                        if (currentConfig.autoscaling) {
                          currentConfig.autoscaling.utilizationScaleDownThreshold =
                            parseFloat(e.target.value);
                        }
                        setConfig(currentConfig);
                      }}
                    />
                    <Label htmlFor="increment">Scaling Increment</Label>
                    <div className="text-sm text-muted-foreground">
                      The number of replicas to scale by when scaling up or
                      down.
                    </div>
                    <Input
                      id="increment"
                      type="number"
                      value={config.autoscaling?.increment || 1}
                      onChange={(e) => {
                        const currentConfig = { ...config };
                        if (currentConfig.autoscaling) {
                          currentConfig.autoscaling.increment = parseInt(
                            e.target.value,
                          );
                        }
                        setConfig(currentConfig);
                      }}
                    />
                  </AccordionContent>
                </AccordionItem>
              </Accordion>
            </TabsContent>
          </Tabs>
        </div>
      </CardContent>
      {actions}
    </Card>
  );
}
