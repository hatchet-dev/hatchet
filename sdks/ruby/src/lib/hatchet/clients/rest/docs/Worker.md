# HatchetSdkRest::Worker

## Properties

| Name | Type | Description | Notes |
| ---- | ---- | ----------- | ----- |
| **metadata** | [**APIResourceMeta**](APIResourceMeta.md) |  |  |
| **name** | **String** | The name of the worker. |  |
| **type** | [**WorkerType**](WorkerType.md) |  |  |
| **last_heartbeat_at** | **Time** | The time this worker last sent a heartbeat. | [optional] |
| **last_listener_established** | **Time** | The time this worker last sent a heartbeat. | [optional] |
| **actions** | **Array&lt;String&gt;** | The actions this worker can perform. | [optional] |
| **slots** | [**Array&lt;SemaphoreSlots&gt;**](SemaphoreSlots.md) | The semaphore slot state for the worker. | [optional] |
| **recent_step_runs** | [**Array&lt;RecentStepRuns&gt;**](RecentStepRuns.md) | The recent step runs for the worker. | [optional] |
| **status** | **String** | The status of the worker. | [optional] |
| **max_runs** | **Integer** | The maximum number of runs this worker can execute concurrently. | [optional] |
| **available_runs** | **Integer** | The number of runs this worker can execute concurrently. | [optional] |
| **dispatcher_id** | **String** | the id of the assigned dispatcher, in UUID format | [optional] |
| **labels** | [**Array&lt;WorkerLabel&gt;**](WorkerLabel.md) | The current label state of the worker. | [optional] |
| **webhook_url** | **String** | The webhook URL for the worker. | [optional] |
| **webhook_id** | **String** | The webhook ID for the worker. | [optional] |
| **runtime_info** | [**WorkerRuntimeInfo**](WorkerRuntimeInfo.md) |  | [optional] |

## Example

```ruby
require 'hatchet-sdk-rest'

instance = HatchetSdkRest::Worker.new(
  metadata: null,
  name: null,
  type: null,
  last_heartbeat_at: 2022-12-13T15:06:48.888358-05:00,
  last_listener_established: 2022-12-13T15:06:48.888358-05:00,
  actions: null,
  slots: null,
  recent_step_runs: null,
  status: null,
  max_runs: null,
  available_runs: null,
  dispatcher_id: bb214807-246e-43a5-a25d-41761d1cff9e,
  labels: null,
  webhook_url: null,
  webhook_id: null,
  runtime_info: null
)
```

