// Bundles Monaco locally instead of letting @monaco-editor/react fetch it
// from cdn.jsdelivr.net at runtime, which breaks air-gapped/self-hosted
// deployments. Importing this module configures both the editor's worker
// environment and the loader before any <Editor /> instance mounts.
import { loader } from '@monaco-editor/react';
import * as monaco from 'monaco-editor';
import editorWorker from 'monaco-editor/esm/vs/editor/editor.worker?worker';
import jsonWorker from 'monaco-editor/esm/vs/language/json/json.worker?worker';

self.MonacoEnvironment = {
  getWorker(_workerId, label) {
    if (label === 'json') {
      return new jsonWorker();
    }
    return new editorWorker();
  },
};

loader.config({ monaco });
