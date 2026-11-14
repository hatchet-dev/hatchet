export type Snippet = {
    title: string;
    content: string;
    githubUrl: string;
    codePath: string;
    language: 'python' | 'typescript' | 'typescript_sandbox' | 'go' | 'ruby'
};
