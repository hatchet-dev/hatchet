import { HighlightFrames } from '@/next/pages/learn/pages/1.first-run/first-run.keyframes';
import { Highlights } from './highlights';
import React from 'react';
import { Snippet } from '@/next/lib/docs/snips';

export type SupportedLanguage = 'typescript' | 'python' | 'go';
export type TypeScriptPackageManager = 'npm' | 'pnpm' | 'yarn';
export type PythonPackageManager = 'poetry' | 'uv' | 'pip' | 'pipenv';
export type GoPackageManager = 'go';

export type PackageManager =
  | TypeScriptPackageManager
  | PythonPackageManager
  | GoPackageManager;

export interface Commands<C> extends Record<PackageManager, C> {}

export interface LanguageSpecificCode {
  path: string;
  repo?: string;
}

export type Extra<E> = Record<SupportedLanguage, Partial<E>>;

export interface LessonPlan<S extends string, E, C> {
  title: string;
  description: React.ReactNode;
  defaultLanguage: SupportedLanguage;
  extraDefaults: Extra<E>;
  intro: React.ReactNode;
  duration: string;
  steps: Record<S, LessonStep>;
  commands: Commands<C>;
  codeKeyFrames: Highlights<HighlightFrames, S>;
  codeBlockDefaults: {
    showLineNumbers: boolean;
  };
}

export interface LessonStep {
  title: string;
  description: () => React.ReactNode;
  content?: () => React.ReactNode;
  code?: Record<SupportedLanguage, Snippet>;
}

export type CommandMap<S extends string, T extends Record<string, string>> = {
  [K in S]?: T;
};
