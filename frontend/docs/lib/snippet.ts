export type Snippet = {
    title: string;
    content: string;
    githubUrl: string;
    codePath: string;
    language: 'python' | 'typescript' | 'go' | 'ruby'
};
