// Original file: ../api-contracts/dispatcher/dispatcher.proto


export interface WorkerRegisterRequest {
  'tenantId'?: (string);
  'workerName'?: (string);
  'actions'?: (string)[];
  'services'?: (string)[];
}

export interface WorkerRegisterRequest__Output {
  'tenantId'?: (string);
  'workerName'?: (string);
  'actions'?: (string)[];
  'services'?: (string)[];
}
