import React from 'react'
import { useData } from 'nextra/data'
import { CodeBlock } from './CodeBlock'
import { Src } from './codeData'

interface GithubSnippetProps {
  src: Src
  target: string
}

export const GithubSnippet = ({ src, target }: GithubSnippetProps) => {
  const { contents } = useData()
  const snippet = contents.find(c => c.props.rawUrl === src.rawUrl) as Src

  return <CodeBlock source={{
    ...src,
    ...snippet,
  }} target={target} />
}
