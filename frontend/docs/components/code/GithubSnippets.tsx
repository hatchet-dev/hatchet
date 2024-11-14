import React from 'react'
import { useData } from 'nextra/data'
import { CodeRenderer } from './CodeRenderer'
import { Src } from './codeData'

interface GithubSnippetProps {
  src: Src
  target: string
}

export const GithubSnippet = ({ src, target }: GithubSnippetProps) => {
  const { contents } = useData()

  const snippet = contents.find(c => c.props.rawUrl === src.rawUrl) as Src

  console.log(snippet)
  return <CodeRenderer source={{
    ...snippet,
    raw: snippet.raw
  }} target={target} />
}
