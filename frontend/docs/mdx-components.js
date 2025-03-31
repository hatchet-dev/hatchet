// MDX components for Nextra 4
import React from 'react';
import { Callout, Card, Cards, Steps, Tabs, FileTree } from 'nextra/components';

export function useMDXComponents(components) {
  return {
    ...components,
    // Adding Nextra components so they can be used in MDX files
    Callout,
    Card,
    Cards,
    Steps,
    Tabs,
    FileTree,
    // You can add your custom components here
  }
} 