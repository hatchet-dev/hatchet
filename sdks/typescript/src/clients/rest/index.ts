import api from './api';
import { Api } from './generated/Api';

import { Worker as _Worker, Workflow as _Workflow } from './generated/data-contracts';
import * as APIContracts from './generated/data-contracts';

// Then, re-export them as needed
type ApiWorker = _Worker;
type ApiWorkflow = _Workflow;

// Export everything by default
export { ApiWorker, ApiWorkflow, APIContracts, Api };

export default api;
