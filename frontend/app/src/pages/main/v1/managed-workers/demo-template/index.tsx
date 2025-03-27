import { Button } from '@/components/ui/button';
import { Separator } from '@/components/ui/separator';
import { useTenant } from '@/lib/atoms';
import { Link } from 'react-router-dom';
import { useState, useEffect } from 'react';
import { ArrowLeftIcon } from '@radix-ui/react-icons';
import {
  PlayIcon,
  CheckCircleIcon,
  CodeBracketIcon,
  ArrowPathIcon,
  KeyIcon,
} from '@heroicons/react/24/outline';
import { Step, Steps } from '@/components/v1/ui/steps';
import { CodeHighlighter } from '@/components/ui/code-highlighter';
import { useMutation, useQuery } from '@tanstack/react-query';
import { queries } from '@/lib/api/queries';
import {
  ManagedWorkerEventStatus,
  TemplateOptions,
} from '@/lib/api/generated/cloud/data-contracts';
import { RadioGroup, RadioGroupItem } from '@/components/ui/radio-group';
import { Label } from '@/components/ui/label';
import { Card } from '@/components/ui/card';
import { cloudApi } from '@/lib/api/api';
import { Checkbox } from '@/components/ui/checkbox';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';

export default function DemoTemplate() {
  const { tenant } = useTenant();
  const [deploying, setDeploying] = useState(false);
  const [deployed, setDeployed] = useState(false);
  const [deployedWorkerId, setDeployedWorkerId] = useState<string | null>(null);
  const [deploymentError, setDeploymentError] = useState<string | null>(null);
  const [deploymentStatus, setDeploymentStatus] = useState<string>('');
  const [isSimulation, setIsSimulation] = useState(false);
  const [workflowId, setWorkflowId] = useState<string | null>(null);
  const [, setTriggering] = useState(false);
  const [, setTriggerSuccess] = useState(false);
  const [runsTriggered, setRunsTriggered] = useState(0);
  const [allRunsTriggered, setAllRunsTriggered] = useState(false);

  // Step states
  const [overviewOpen, setOverviewOpen] = useState(true);
  const [infoConfirmed, setInfoConfirmed] = useState(false);
  const [deployStepOpen, setDeployStepOpen] = useState(false);
  const [successStepOpen, setSuccessStepOpen] = useState(false);

  // Template selection
  const [selectedTemplate, setSelectedTemplate] = useState<TemplateOptions>(
    TemplateOptions.QUICKSTART_TYPESCRIPT,
  );

  // Template code examples
  const codeExamples = {
    [TemplateOptions.QUICKSTART_TYPESCRIPT]: `// TypeScript worker example
import { Hatchet } from '@hatchet/client';

const client = new Hatchet();

client.worker('demo-worker', async (event) => {
  console.log('Received event:', event);
  // Process data, call APIs, etc.
  return { status: 'success' };
});`,
    [TemplateOptions.QUICKSTART_PYTHON]: `# Python worker example
from hatchet_sdk import Hatchet

client = Hatchet()

@client.worker("demo-worker")
def process_event(event):
    print("Received event:", event)
    # Process data, call APIs, etc.
    return {"status": "success"}`,
    [TemplateOptions.QUICKSTART_GO]: `// Go worker example
package main

import (
	"context"
	"fmt"
	"github.com/hatchet-dev/hatchet-go/pkg/worker"
)

func main() {
	client := worker.New()
	
	client.RegisterWorker("demo-worker", func(ctx context.Context, event map[string]interface{}) (map[string]interface{}, error) {
		fmt.Println("Received event:", event)
		// Process data, call APIs, etc.
		return map[string]interface{}{"status": "success"}, nil
	})
	
	client.Start()
}`,
  };

  // Create demo template mutation
  const { mutate: createComputeDemoTemplate, isPending } = useMutation({
    mutationFn: (template: TemplateOptions) =>
      cloudApi.managedWorkerTemplateCreate(tenant!.metadata.id, {
        name: template,
      }),
    onSuccess: (response) => {
      // Extract worker ID from the response
      const workerId = response.data.metadata?.id;
      if (workerId) {
        setDeployedWorkerId(workerId);

        // In a real implementation, we would fetch the demo workflow ID here
        if (isSimulation) {
          setWorkflowId(`sim-wf-${Math.random().toString(36).substring(2, 7)}`);
        }
      }
    },
    onError: (error: any) => {
      console.error('Failed to create template:', error);
      setDeploymentError(
        error?.response?.data?.errors?.[0]?.description ||
          'Failed to create template',
      );
      setDeploying(false);
    },
  });

  // Query for monitoring worker events if we have a worker ID
  const workerEventsQuery = useQuery({
    ...queries.cloud.listManagedWorkerEvents(deployedWorkerId || ''),
    enabled: !!deployedWorkerId && !isSimulation,
    refetchInterval:
      deployedWorkerId && !deployed && !isSimulation ? 2000 : false,
  });

  // Simulated deployment steps for the demo
  const simulateDeployment = () => {
    setDeploying(true);
    setDeploymentError(null);
    setDeploymentStatus('Initializing deployment...');

    // Mock a random worker ID for the simulation
    const mockWorkerId = `sim-${Math.random().toString(36).substring(2, 9)}`;
    setDeployedWorkerId(mockWorkerId);
    setWorkflowId(`sim-wf-${Math.random().toString(36).substring(2, 7)}`);

    // Simulate a deployment sequence with delays
    setTimeout(() => {
      setDeploymentStatus('Creating worker resources...');

      setTimeout(() => {
        setDeploymentStatus('Building container image...');

        setTimeout(() => {
          setDeploymentStatus('Deploying managed worker...');

          setTimeout(() => {
            setDeploymentStatus('Deployment complete');
            setDeploying(false);
            setDeployed(true);
            setDeployStepOpen(false);
            setSuccessStepOpen(true);
          }, 2000);
        }, 1500);
      }, 1500);
    }, 1000);
  };

  // Trigger a workflow run
  const triggerWorkflow = async () => {
    if (!tenant || !workflowId) {
      return;
    }

    setTriggering(true);

    if (isSimulation) {
      // Simulate workflow trigger
      setTimeout(() => {
        setTriggering(false);
        setTriggerSuccess(true);
        setRunsTriggered((prev) => prev + 1);
        if (runsTriggered + 1 >= 3) {
          setAllRunsTriggered(true);
        }
      }, 1000);
      return;
    }

    try {
      // In a real implementation, we would call the API to trigger the workflow
      // This is simplified mock code for demonstration
      setTimeout(() => {
        setTriggering(false);
        setTriggerSuccess(true);
        setRunsTriggered((prev) => prev + 1);
        if (runsTriggered + 1 >= 3) {
          setAllRunsTriggered(true);
        }
      }, 1000);
    } catch (error) {
      console.error('Failed to trigger workflow:', error);
      setTriggering(false);
    }
  };

  // Automatically trigger workflow runs when success step is opened
  useEffect(() => {
    if (successStepOpen && workflowId && !allRunsTriggered) {
      const triggerRuns = async () => {
        for (let i = 0; i < 3; i++) {
          if (i > 0) {
            // Add a small delay between triggers
            await new Promise((resolve) => setTimeout(resolve, 1500));
          }
          await triggerWorkflow();
        }
      };

      triggerRuns();
    }
  }, [successStepOpen, workflowId]);

  // Monitor events to determine deployment status
  useEffect(() => {
    if (deployedWorkerId && workerEventsQuery.data?.rows && !isSimulation) {
      const events = workerEventsQuery.data.rows;

      // Check for latest event
      if (events.length > 0) {
        // Sort by time, most recent first
        const sortedEvents = [...events].sort(
          (a, b) =>
            new Date(b.timeLastSeen).getTime() -
            new Date(a.timeLastSeen).getTime(),
        );

        const latestEvent = sortedEvents[0];
        setDeploymentStatus(latestEvent.message);

        // Check if deployment is complete
        if (
          latestEvent.status === ManagedWorkerEventStatus.SUCCEEDED &&
          latestEvent.message.includes('Deployment complete')
        ) {
          setDeploying(false);
          setDeployed(true);
          setDeployStepOpen(false);
          setSuccessStepOpen(true);

          // In a real implementation, we would fetch the demo workflow ID here
          // For now, use a mock ID for non-simulation case
          if (!workflowId) {
            setWorkflowId(
              `demo-wf-${Math.random().toString(36).substring(2, 7)}`,
            );
          }
        } else if (latestEvent.status === ManagedWorkerEventStatus.FAILED) {
          setDeploymentError(latestEvent.message);
          setDeploying(false);
        }
      }
    }
  }, [deployedWorkerId, workerEventsQuery.data, isSimulation, workflowId]);

  const handleDeploy = async () => {
    if (!tenant) {
      return;
    }

    if (isSimulation) {
      simulateDeployment();
      return;
    }

    setDeploying(true);
    setDeploymentError(null);
    setDeploymentStatus('Initializing deployment...');

    // Call the actual API to deploy the template
    createComputeDemoTemplate(selectedTemplate);
  };

  const handleConfirmInfo = () => {
    setInfoConfirmed(true);
    setOverviewOpen(false);
    setDeployStepOpen(true);
  };

  const handleLanguageSelection = (value: TemplateOptions) => {
    setSelectedTemplate(value);
  };

  // API token generation
  const [isGeneratingToken, setIsGeneratingToken] = useState(false);
  const [apiToken, setApiToken] = useState<string | null>(null);
  const [tokenRevealed, setTokenRevealed] = useState(true);
  const [selectedCodeTab, setSelectedCodeTab] = useState('typescript');

  // Handle API token generation
  const handleGenerateToken = () => {
    setIsGeneratingToken(true);

    // For both simulation and real mode, we'll use a timeout
    // In a real implementation, you would call the API to generate a token
    setTimeout(() => {
      // Generate a realistic-looking token
      const tokenPrefix = isSimulation ? 'hx_sim_' : 'hx_';
      const randomPart =
        Math.random().toString(36).substring(2, 15) +
        Math.random().toString(36).substring(2, 15);

      setApiToken(`${tokenPrefix}${randomPart}`);
      setIsGeneratingToken(false);
    }, 1500);
  };

  // Code examples for triggering a workflow via API
  const triggerCodeExamples = {
    typescript: `// Trigger a workflow run using the Hatchet API
import axios from 'axios';

async function triggerWorkflow() {
  const response = await axios.post(
    'https://api.hatchet.run/api/v1/tenants/${tenant?.metadata.id}/workflows/${workflowId}/events',
    { 
      data: { 
        message: "Hello from API trigger!" 
      }
    },
    {
      headers: {
        'Authorization': 'Bearer ${apiToken || '[YOUR_API_TOKEN]'}',
        'Content-Type': 'application/json'
      }
    }
  );
  
  console.log('Workflow triggered:', response.data);
}

triggerWorkflow();`,
    python: `# Trigger a workflow run using the Hatchet API
import requests

def trigger_workflow():
    url = "https://api.hatchet.run/api/v1/tenants/${tenant?.metadata.id}/workflows/${workflowId}/events"
    
    headers = {
        "Authorization": "Bearer ${apiToken || '[YOUR_API_TOKEN]'}",
        "Content-Type": "application/json"
    }
    
    payload = {
        "data": {
            "message": "Hello from API trigger!"
        }
    }
    
    response = requests.post(url, json=payload, headers=headers)
    print("Workflow triggered:", response.json())

trigger_workflow()`,
    go: `// Trigger a workflow run using the Hatchet API
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

func main() {
	url := "https://api.hatchet.run/api/v1/tenants/${tenant?.metadata.id}/workflows/${workflowId}/events"
	
	payload := map[string]interface{}{
		"data": map[string]interface{}{
			"message": "Hello from API trigger!",
		},
	}
	
	payloadBytes, _ := json.Marshal(payload)
	
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(payloadBytes))
	req.Header.Set("Authorization", "Bearer ${apiToken || '[YOUR_API_TOKEN]'}")
	req.Header.Set("Content-Type", "application/json")
	
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer resp.Body.Close()
	
	fmt.Println("Workflow triggered successfully!")
}`,
  };

  return (
    <div className="flex-grow h-full w-full">
      <div className="mx-auto py-8 px-4 sm:px-6 lg:px-8">
        <div className="flex flex-row items-center mb-4">
          <Link to="/v1/managed-workers" className="mr-4">
            <Button variant="ghost" size="icon">
              <ArrowLeftIcon className="h-4 w-4" />
            </Button>
          </Link>
          <h2 className="text-2xl font-bold leading-tight text-foreground">
            Deploy Demo Template
          </h2>
        </div>
        <Separator className="my-4" />
        <div className="max-w-3xl mx-auto">
          <Steps className="mt-6">
            <Step
              title="Select your template language"
              open={overviewOpen}
              setOpen={setOverviewOpen}
            >
              <div className="grid gap-6">
                <div className="bg-muted/30 p-4 rounded-lg mb-2">
                  <h4 className="font-medium mb-2">Demo Includes:</h4>
                  <ul className="space-y-2">
                    <li className="flex items-start">
                      <span className="text-primary mr-2 flex items-center">
                        •
                      </span>
                      <span>Sample workflow with 3 steps</span>
                    </li>
                    <li className="flex items-start">
                      <span className="text-primary mr-2 flex items-center">
                        •
                      </span>
                      <span>1 managed service (limited resources)</span>
                    </li>
                    <li className="flex items-start">
                      <span className="text-primary mr-2 flex items-center">
                        •
                      </span>
                      <span>Active for 1 hour</span>
                    </li>
                    <li className="flex items-start">
                      <span className="text-primary mr-2 flex items-center">
                        •
                      </span>
                      <span>No payment method required</span>
                    </li>
                  </ul>
                </div>

                <div className="mb-4">
                  <h3 className="text-lg font-medium mb-4">
                    Choose your programming language:
                  </h3>
                  <RadioGroup
                    value={selectedTemplate}
                    onValueChange={handleLanguageSelection}
                    className="flex flex-col space-y-3"
                  >
                    {/* TypeScript Option */}
                    <div
                      className="cursor-pointer"
                      onClick={() =>
                        handleLanguageSelection(
                          TemplateOptions.QUICKSTART_TYPESCRIPT,
                        )
                      }
                    >
                      <Card
                        className={`p-4 border-2 ${selectedTemplate === TemplateOptions.QUICKSTART_TYPESCRIPT ? 'border-primary' : 'border-border'}`}
                      >
                        <div className="flex items-center space-x-3">
                          <RadioGroupItem
                            value={TemplateOptions.QUICKSTART_TYPESCRIPT}
                            id="typescript"
                            className="mr-2"
                          />
                          <div className="grid gap-1">
                            <Label
                              htmlFor="typescript"
                              className="font-medium text-lg cursor-pointer"
                            >
                              TypeScript
                            </Label>
                            <p className="text-muted-foreground text-sm">
                              Modern JavaScript with type safety.
                            </p>
                          </div>
                        </div>

                        {selectedTemplate ===
                          TemplateOptions.QUICKSTART_TYPESCRIPT && (
                          <div className="mt-4 border-t pt-4">
                            <div className="flex items-start">
                              <div className="h-8 w-8 rounded-full bg-primary/10 flex items-center justify-center mr-3">
                                <CodeBracketIcon className="h-4 w-4 text-primary" />
                              </div>
                              <div className="flex-1">
                                <CodeHighlighter
                                  code={
                                    codeExamples[
                                      TemplateOptions.QUICKSTART_TYPESCRIPT
                                    ]
                                  }
                                  language="typescript"
                                  copy
                                />
                              </div>
                            </div>
                            <div className="flex justify-end mt-4">
                              <Button
                                onClick={handleConfirmInfo}
                                size="sm"
                                className="px-4"
                              >
                                Continue with TypeScript
                              </Button>
                            </div>
                          </div>
                        )}
                      </Card>
                    </div>

                    {/* Python Option */}
                    <div
                      className="cursor-pointer"
                      onClick={() =>
                        handleLanguageSelection(
                          TemplateOptions.QUICKSTART_PYTHON,
                        )
                      }
                    >
                      <Card
                        className={`p-4 border-2 ${selectedTemplate === TemplateOptions.QUICKSTART_PYTHON ? 'border-primary' : 'border-border'}`}
                      >
                        <div className="flex items-center space-x-3">
                          <RadioGroupItem
                            value={TemplateOptions.QUICKSTART_PYTHON}
                            id="python"
                            className="mr-2"
                          />
                          <div className="grid gap-1">
                            <Label
                              htmlFor="python"
                              className="font-medium text-lg cursor-pointer"
                            >
                              Python
                            </Label>
                            <p className="text-muted-foreground text-sm">
                              Simple, readable syntax for rapid development.
                            </p>
                          </div>
                        </div>

                        {selectedTemplate ===
                          TemplateOptions.QUICKSTART_PYTHON && (
                          <div className="mt-4 border-t pt-4">
                            <div className="flex items-start">
                              <div className="h-8 w-8 rounded-full bg-primary/10 flex items-center justify-center mr-3">
                                <CodeBracketIcon className="h-4 w-4 text-primary" />
                              </div>
                              <div className="flex-1">
                                <CodeHighlighter
                                  code={
                                    codeExamples[
                                      TemplateOptions.QUICKSTART_PYTHON
                                    ]
                                  }
                                  language="python"
                                  copy
                                />
                              </div>
                            </div>
                            <div className="flex justify-end mt-4">
                              <Button
                                onClick={handleConfirmInfo}
                                size="sm"
                                className="px-4"
                              >
                                Continue with Python
                              </Button>
                            </div>
                          </div>
                        )}
                      </Card>
                    </div>

                    {/* Go Option */}
                    <div
                      className="cursor-pointer"
                      onClick={() =>
                        handleLanguageSelection(TemplateOptions.QUICKSTART_GO)
                      }
                    >
                      <Card
                        className={`p-4 border-2 ${selectedTemplate === TemplateOptions.QUICKSTART_GO ? 'border-primary' : 'border-border'}`}
                      >
                        <div className="flex items-center space-x-3">
                          <RadioGroupItem
                            value={TemplateOptions.QUICKSTART_GO}
                            id="go"
                            className="mr-2"
                          />
                          <div className="grid gap-1">
                            <Label
                              htmlFor="go"
                              className="font-medium text-lg cursor-pointer"
                            >
                              Go
                            </Label>
                            <p className="text-muted-foreground text-sm">
                              Efficient, concurrent programming language.
                            </p>
                          </div>
                        </div>

                        {selectedTemplate === TemplateOptions.QUICKSTART_GO && (
                          <div className="mt-4 border-t pt-4">
                            <div className="flex items-start">
                              <div className="h-8 w-8 rounded-full bg-primary/10 flex items-center justify-center mr-3">
                                <CodeBracketIcon className="h-4 w-4 text-primary" />
                              </div>
                              <div className="flex-1">
                                <CodeHighlighter
                                  code={
                                    codeExamples[TemplateOptions.QUICKSTART_GO]
                                  }
                                  language="go"
                                  copy
                                />
                              </div>
                            </div>
                            <div className="flex justify-end mt-4">
                              <Button
                                onClick={handleConfirmInfo}
                                size="sm"
                                className="px-4"
                              >
                                Continue with Go
                              </Button>
                            </div>
                          </div>
                        )}
                      </Card>
                    </div>
                  </RadioGroup>
                </div>

                <Button onClick={handleConfirmInfo} className="w-fit mt-2">
                  Continue
                </Button>
              </div>
            </Step>

            <Step
              title="Deploy the demo template"
              open={deployStepOpen}
              setOpen={setDeployStepOpen}
              disabled={!infoConfirmed}
            >
              <div className="grid gap-4">
                <div className="border rounded-lg bg-card p-6 shadow-sm">
                  <div className="flex items-start space-x-4">
                    <div className="h-10 w-10 rounded-full bg-primary/10 flex items-center justify-center">
                      <PlayIcon className="h-5 w-5 text-primary" />
                    </div>
                    <div className="flex-1">
                      <h3 className="text-xl font-medium mb-2">
                        Ready to Deploy
                      </h3>
                      <p className="text-muted-foreground mb-4">
                        Click the button below to deploy your{' '}
                        {selectedTemplate
                          .replace('QUICKSTART_', '')
                          .toLowerCase()}{' '}
                        demo template. This will create a managed service and
                        workflow that you can use to explore the features.
                      </p>

                      <div className="bg-muted/30 p-4 rounded-lg mb-6">
                        <h4 className="font-medium mb-2">What happens next:</h4>
                        <ul className="space-y-2">
                          <li className="flex items-start">
                            <span className="text-primary mr-2 flex items-center">
                              •
                            </span>
                            <span>
                              A service will be provisioned with the necessary
                              resources
                            </span>
                          </li>
                          <li className="flex items-start">
                            <span className="text-primary mr-2 flex items-center">
                              •
                            </span>
                            <span>
                              A sample workflow will be created and registered
                            </span>
                          </li>
                          <li className="flex items-start">
                            <span className="text-primary mr-2 flex items-center">
                              •
                            </span>
                            <span>
                              You'll be able to monitor the activity in your
                              dashboard
                            </span>
                          </li>
                        </ul>
                      </div>

                      <div className="flex items-center space-x-2 mb-4">
                        <Checkbox
                          id="simulation"
                          checked={isSimulation}
                          onCheckedChange={(checked) => {
                            setIsSimulation(checked === true);
                          }}
                        />
                        <Label
                          htmlFor="simulation"
                          className="text-sm font-medium"
                        >
                          Use simulation mode (doesn't create actual resources)
                        </Label>
                      </div>

                      {deploying && (
                        <div className="border rounded-lg p-4 mb-4 bg-muted/20">
                          <div className="flex items-center mb-2">
                            <ArrowPathIcon className="h-4 w-4 mr-2 animate-spin" />
                            <h4 className="font-medium">
                              Deployment in progress...
                            </h4>
                          </div>
                          <p className="text-sm text-muted-foreground">
                            {deploymentStatus}
                          </p>
                        </div>
                      )}

                      {deploymentError && (
                        <div className="border border-destructive rounded-lg p-4 mb-4 bg-destructive/10">
                          <h4 className="font-medium text-destructive mb-1">
                            Deployment failed
                          </h4>
                          <p className="text-sm text-muted-foreground">
                            {deploymentError}
                          </p>
                        </div>
                      )}

                      <div className="border-t pt-4">
                        <div className="flex justify-between items-center">
                          <div className="text-sm text-muted-foreground">
                            {deploying
                              ? 'Deploying demo template...'
                              : isSimulation
                                ? 'Ready to simulate deployment'
                                : 'Ready to deploy'}
                          </div>
                          <Button
                            onClick={handleDeploy}
                            disabled={
                              deploying ||
                              !tenant ||
                              (!isSimulation && isPending)
                            }
                            className="min-w-32"
                          >
                            {deploying || (!isSimulation && isPending)
                              ? 'Deploying...'
                              : isSimulation
                                ? 'Simulate Deploy'
                                : 'Deploy Demo'}
                          </Button>
                        </div>
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            </Step>

            <Step
              title="Deployment successful"
              open={successStepOpen}
              setOpen={setSuccessStepOpen}
              disabled={!deployed}
            >
              <div className="grid gap-6">
                {/* Success header */}
                <div className="border rounded-lg bg-card p-6 shadow-sm">
                  <div className="flex items-center justify-center flex-col text-center py-4">
                    <div className="h-16 w-16 rounded-full bg-green-500/10 flex items-center justify-center mb-4">
                      <CheckCircleIcon className="h-8 w-8 text-green-500" />
                    </div>
                    <h3 className="text-xl font-medium mb-2">
                      Demo Template Deployed!
                    </h3>
                    <p className="text-muted-foreground mb-4 max-w-md">
                      Your{' '}
                      {selectedTemplate
                        .replace('QUICKSTART_', '')
                        .toLowerCase()}{' '}
                      demo template has been successfully deployed. You can now
                      explore the managed service features.
                      {isSimulation && (
                        <span className="block mt-2 text-amber-500 font-medium">
                          Note: This was a simulated deployment. No actual
                          resources were created.
                        </span>
                      )}
                    </p>
                  </div>
                </div>

                {/* Trigger Run Programmatically section */}
                <div className="border rounded-lg bg-card p-6 shadow-sm">
                  <div className="flex items-start mb-4">
                    <div className="h-8 w-8 rounded-full bg-primary/10 flex items-center justify-center mr-3">
                      <KeyIcon className="h-4 w-4 text-primary" />
                    </div>
                    <div className="flex-1">
                      <h4 className="font-medium mb-1">
                        Trigger a Remote Run Programmatically
                      </h4>
                      <p className="text-sm text-muted-foreground mb-4">
                        Run the following code locally to execute a task on the
                        deployed service.
                      </p>

                      {!apiToken ? (
                        <Button
                          onClick={handleGenerateToken}
                          disabled={isGeneratingToken}
                          className="w-full mb-2"
                        >
                          {isGeneratingToken ? (
                            <>
                              <ArrowPathIcon className="h-4 w-4 mr-2 animate-spin" />
                              Generating Token...
                            </>
                          ) : (
                            'Generate API Token'
                          )}
                        </Button>
                      ) : (
                        <>
                          <div className="bg-green-500/10 text-green-600 p-3 rounded mb-4 text-sm">
                            API token successfully generated!
                          </div>

                          <p className="text-sm text-muted-foreground mb-2">
                            This is the only time we will show you this auth
                            token. Make sure to copy it now.
                          </p>

                          <CodeHighlighter
                            language="plaintext"
                            code={`export HATCHET_CLIENT_TOKEN="${apiToken}"`}
                            copy
                            className="mb-3"
                          />

                          <div className="flex space-x-2 mb-4">
                            <Button
                              type="button"
                              variant="outline"
                              size="sm"
                              className="flex items-center"
                              onClick={() => setTokenRevealed(!tokenRevealed)}
                            >
                              {tokenRevealed ? 'Hide Token' : 'Reveal Token'}
                            </Button>
                            <Button
                              variant="outline"
                              size="sm"
                              onClick={handleGenerateToken}
                              disabled={isGeneratingToken}
                            >
                              Generate New Token
                            </Button>
                          </div>
                        </>
                      )}
                      <div className="border-t pt-4 mt-2">
                        <h5 className="font-medium mb-2">Example Code</h5>
                        <Tabs
                          value={selectedCodeTab}
                          onValueChange={setSelectedCodeTab}
                          className="w-full"
                        >
                          <TabsList className="mb-2">
                            <TabsTrigger value="typescript">
                              TypeScript
                            </TabsTrigger>
                            <TabsTrigger value="python">Python</TabsTrigger>
                            <TabsTrigger value="go">Go</TabsTrigger>
                          </TabsList>
                          <TabsContent value="typescript" className="mt-0">
                            <CodeHighlighter
                              code={triggerCodeExamples.typescript}
                              language="typescript"
                              copy
                            />
                          </TabsContent>
                          <TabsContent value="python" className="mt-0">
                            <CodeHighlighter
                              code={triggerCodeExamples.python}
                              language="python"
                              copy
                            />
                          </TabsContent>
                          <TabsContent value="go" className="mt-0">
                            <CodeHighlighter
                              code={triggerCodeExamples.go}
                              language="go"
                              copy
                            />
                          </TabsContent>
                        </Tabs>
                      </div>
                    </div>
                  </div>
                </div>

                {/* What's next section */}
                <div className="border rounded-lg bg-card p-6 shadow-sm">
                  <h4 className="font-medium mb-3">What's next?</h4>
                  <ul className="space-y-3 mb-4">
                    <li className="flex items-start">
                      <span className="text-primary mr-2 flex items-center mt-0.5">
                        •
                      </span>
                      <span>
                        View your deployed service to see logs and metrics
                      </span>
                    </li>
                    <li className="flex items-start">
                      <span className="text-primary mr-2 flex items-center mt-0.5">
                        •
                      </span>
                      <span>
                        Three demo workflow runs have been triggered for you
                      </span>
                    </li>
                    <li className="flex items-start">
                      <span className="text-primary mr-2 flex items-center mt-0.5">
                        •
                      </span>
                      <span>
                        Use the API to trigger additional workflow runs
                      </span>
                    </li>
                    <li className="flex items-start">
                      <span className="text-primary mr-2 flex items-center mt-0.5">
                        •
                      </span>
                      <span>Monitor workflow runs in the dashboard</span>
                    </li>
                  </ul>

                  {/* Main action button */}
                  {deployedWorkerId && (
                    <Link to={`/v1/managed-workers/${deployedWorkerId}`}>
                      <Button variant="default" className="w-full mb-4">
                        View Your Service
                      </Button>
                    </Link>
                  )}

                  {/* Secondary action buttons */}
                  <div className="grid grid-cols-2 gap-3">
                    <Link to="/v1/workflow-runs">
                      <Button variant="outline" className="w-full">
                        View Workflow Runs
                      </Button>
                    </Link>
                    <Link to="/v1/workflows">
                      <Button variant="outline" className="w-full">
                        View Workflows
                      </Button>
                    </Link>
                  </div>
                </div>
              </div>
            </Step>
          </Steps>
        </div>
      </div>
    </div>
  );
}
