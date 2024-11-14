
const defaultUser = 'hatchet-dev'

const defaultRepos = {
  'ts': 'hatchet-typescript',
  'py': 'hatchet-python',
  'go': 'hatchet'
}

const defaultBranch = 'main'

type RepoProps = {
  user?: string
  repo?: string
  branch?: string
  path: string
}


const sourceCode = `
imports

// ❓ Not OnFailure Step
// This is a comment that is not a snippet
// ‼️



// ❓ OnFailure Step
// This workflow will fail because the step will throw an error
// we define an onFailure step to handle this case
const workflow: Workflow = {
  // ... normal workflow definition
  id: 'on-failure-example',
  description: 'test',
  on: {
    event: 'user:create',
  },
  // ,
  steps: [
    {
      name: 'dag-step1',
      run: async (ctx) => {
        // 👀 this step will always throw an error
        throw new Error('Step 1 failed');
      },
    },
  ],
  // 👀 After the workflow fails, this special step will run
  onFailure: {
    name: 'on-failure-step',
    run: async (ctx) => {
      console.log('Starting On Failure Step!');
      return { onFailure: 'step' };
    },
  },
  // ...
  "extra": "cheese",
  // ,
};
// ‼️

DO NOT PRINT THIS
`;

const getRawUrl = ({ user, repo, branch, path }: RepoProps) => {
  const ext = path.split('.').pop()
  return `https://raw.githubusercontent.com/${user || defaultUser}/${repo || defaultRepos[ext]}/refs/heads/${branch || defaultBranch}/${path}`
}

const getUIUrl = ({ user, repo, branch, path }: RepoProps) => {
  const ext = path.split('.').pop()
  return `https://github.com/${user || defaultUser}/${repo || defaultRepos[ext]}/blob/${branch || defaultBranch}/${path}`
}

export type Src = {
    raw: string
    props: RepoProps
    rawUrl: string
    uiUrl: string
}

export const getSnippets = (props: RepoProps[]): Promise<{ props: { ssg: { contents: Src[] } } }> => {
  return Promise.all(props.map(prop =>
    fetch(getRawUrl(prop))
      .then(res => sourceCode)
      .then(raw => ({ raw, props: prop, rawUrl: getRawUrl(prop), uiUrl: getUIUrl(prop) }))
  )).then(results => ({
    props: {
      ssg: {
        contents: results
      }
    }
  }))
}
