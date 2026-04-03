# Effective Agent Patterns

This directory contains implementations of the effective agent patterns described in [Anthropic's Building Effective Agents](https://www.anthropic.com/engineering/building-effective-agents) blog post, adapted for the Icepick framework.

## Overview

Anthropic's research with dozens of teams has shown that the most successful agent implementations use simple, composable patterns rather than complex frameworks. These patterns can be combined and customized to fit different use cases.

## Pattern Categories

### Workflows
**Workflows** are systems where LLMs and tools are orchestrated through predefined code paths. They offer predictability and consistency for well-defined tasks.

### Agents
**Agents** are systems where LLMs dynamically direct their own processes and tool usage, maintaining control over how they accomplish tasks. They're better when flexibility and model-driven decision-making are needed.

---

## 1. Prompt Chaining

**Location**: [`1.prompt-chaining/`](./1.prompt-chaining/)

### Pattern Description
Prompt chaining decomposes a task into a sequence of steps, where each LLM call processes the output of the previous one. You can add programmatic checks (gates) on intermediate steps to ensure the process stays on track.

### When to Use
- Tasks can be easily decomposed into fixed subtasks
- Trading latency for higher accuracy by making each LLM call an easier task
- Need validation gates between steps

### Examples
- Generating marketing copy, then translating it
- Writing an outline, checking criteria, then writing the full document
- Multi-step content processing with validation

### Implementation Notes
The example demonstrates:
- **Sequential processing**: Each tool runs after the previous one completes
- **Gating logic**: Validates intermediate results before proceeding
- **Clear flow**: `oneTool` → validation → `twoTool` → `threeTool`

---

## 2. Routing

**Location**: [`2.routing/`](./2.routing/)

### Pattern Description
Routing classifies an input and directs it to a specialized followup task. This allows separation of concerns and building more specialized prompts without one input type hurting performance on others.

### When to Use
- Complex tasks with distinct categories better handled separately
- Classification can be handled accurately by LLM or traditional algorithms
- Need specialized handling for different input types

### Examples
- Customer service: routing questions, refunds, technical support
- Multi-model routing: easy questions to smaller models, hard ones to larger models
- Content classification with specialized processors

### Implementation Notes
The example shows:
- **Classification first**: Determines the type of request
- **Specialized handlers**: Different tools for sales vs support
- **Fallback handling**: Graceful degradation for unhandled cases

---

## 3. Parallelization

**Location**: [`3.parallelization/`](./3.parallelization/)

Parallelization has two key variations:

### 3.1 Sectioning
**Location**: [`3.parallelization/1.sectioning/`](./3.parallelization/1.sectioning/)

#### Pattern Description
Breaking a task into independent subtasks that run simultaneously, then aggregating results programmatically.

#### When to Use
- Independent subtasks can be parallelized for speed
- Multiple considerations need separate focused attention
- Implementing guardrails alongside main processing

#### Examples
- Guardrails: One model processes queries while another screens for inappropriate content
- Code review: Multiple aspects evaluated simultaneously
- Multi-faceted analysis requiring separate specialized attention

#### Implementation Notes
- **Parallel execution**: `Promise.all()` runs appropriateness check and main content generation simultaneously
- **Independent concerns**: Each subprocess handles a different aspect
- **Conditional logic**: Results combined based on appropriateness check

### 3.2 Voting
**Location**: [`3.parallelization/2.voting/`](./3.parallelization/2.voting/)

#### Pattern Description
Running the same or similar tasks multiple times to get diverse outputs, then using voting logic to determine the final result.

#### When to Use
- Need multiple perspectives for higher confidence
- Quality assurance through consensus
- Balancing false positives/negatives with vote thresholds

#### Examples
- Code vulnerability review with multiple specialized prompts
- Content moderation with different evaluation criteria
- Quality assessment requiring consensus

#### Implementation Notes
- **Multiple evaluators**: Safety, helpfulness, and accuracy voters run in parallel
- **Voting logic**: Majority approval (2/3) required
- **Detailed feedback**: Each voter provides reasoning for transparency

---

## 4. Evaluator-Optimizer

**Location**: [`4.evaluator-optimizer/`](./4.evaluator-optimizer/)

### Pattern Description
One LLM generates a response while another provides evaluation and feedback in a loop, iteratively improving the output.

### When to Use
- Clear evaluation criteria exist
- Iterative refinement provides measurable value
- LLM can provide useful feedback (similar to human feedback improving results)
- Quality improvements possible through iteration

### Examples
- Literary translation with nuance refinement
- Complex search requiring multiple rounds
- Content creation with quality improvement loops
- Creative writing with iterative polish

### Implementation Notes
The example demonstrates:
- **Iterative loop**: Up to 3 rounds of generation and evaluation
- **Feedback incorporation**: Previous feedback guides next generation
- **Completion criteria**: Evaluator determines when quality is sufficient
- **Fallback**: Maximum iterations prevent infinite loops

---

## Missing Patterns

These patterns from Anthropic's post are not yet implemented but could be added:

### Orchestrator-Workers
A central LLM dynamically breaks down tasks and delegates to worker LLMs. Useful for unpredictable subtasks like complex coding changes or multi-source research.

### Autonomous Agents
Systems where LLMs plan and operate independently with tool usage, maintaining control over task completion. Suitable for open-ended problems requiring many unpredictable steps.

---

## Key Principles

1. **Start Simple**: Begin with basic prompts and only add complexity when needed
2. **Measure Performance**: Use comprehensive evaluation to guide iterations
3. **Composable Patterns**: These patterns can be combined for complex use cases
4. **Tool Design**: Invest in clear tool documentation and interfaces (Agent-Computer Interface)
5. **Transparency**: Show the agent's planning and decision-making steps

## Usage Tips

1. **Pattern Selection**: Choose the simplest pattern that solves your problem
2. **Combination**: These patterns can be nested and combined
3. **Evaluation**: Always measure if complexity improvements justify the costs
4. **Iteration**: Start with one pattern and evolve based on real performance data

---

For more details on each pattern, explore the individual directories and their implementations.
