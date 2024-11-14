
const defaultUser = 'hatchet-dev'

const defaultRepos = {
  'ts': 'hatchet-typescript',
  'py': 'hatchet-python',
  'go': 'hatchet'
}

export const extToLanguage = {
  'ts': 'typescript',
  'py': 'python',
  'go': 'go'
}

const defaultBranch = 'main'

type RepoProps = {
  user?: string
  repo?: string
  branch?: string
  path: string
}

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
    githubUrl: string
    language?: string
}

export const getSnippets = (props: RepoProps[]): Promise<{ props: { ssg: { contents: Src[] } } }> => {
  return Promise.all(props.map(prop =>
    fetch(getRawUrl(prop))
      .then(res => res.text())
      .then(raw => ({
        raw,
        props: prop,
        rawUrl: getRawUrl(prop),
        githubUrl: getUIUrl(prop),
        language: extToLanguage[prop.path.split('.').pop() as keyof typeof extToLanguage]
      }))
  )).then(results => ({
    props: {
      ssg: {
        contents: results
      },

    }
  }))
}
