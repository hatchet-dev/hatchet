import React from 'react'
import { useData } from 'nextra/data'
import { Code } from 'nextra/components'
import { parseDocComments } from './codeParser'

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
DO NOT PRINT THIS

// ❓ Not OnFailure Step
WOWOOWOW
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

export const getSnippets = (props: RepoProps[]) => {
  return Promise.all(props.map(prop =>
    fetch(getRawUrl(prop))
      .then(res => res.text())
      .then(raw => ({ raw, props: prop }))
  )).then(results => ({
    props: {
      ssg: {
        contents: results
      }
    }
  }))
}

export const GithubSnippets = () => {
  // Get the data from SSG, and render it as a component.
  const { contents } = useData()
  const [collapsed, setCollapsed] = React.useState(true);

  return <>
    <button onClick={() => setCollapsed(!collapsed)}>
      {collapsed ? 'Show' : 'Hide'}
    </button>
    <pre>{parseDocComments(sourceCode, "OnFailure Step", collapsed)}</pre>
  </>

  // return contents.map(({ raw, props }) => <>
  //   <a href={getUIUrl(props)}>View complete code on GitHub</a>
  //   <strong>
  //     <pre>
  //       {raw}
  //     </pre>
  //   </strong>
  // </>
  // )
}
