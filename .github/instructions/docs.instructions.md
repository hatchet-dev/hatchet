---
applyTo: "frontend/docs/**/*.mdx"
---

# Docs Style Guide

## Audience

These docs are written for engineers. Do not assume familiarity with Hatchet or similar platforms — explain concepts from first principles or link to relevant background.

## Tone and Voice

- Address the reader as "you" (e.g. "you can configure..." not "the user can configure...").
- Use a helpful but prosaic tone — clear and direct, not overly formal or conversational.
- Avoid passive voice (e.g. "Hatchet triggers the workflow" not "the workflow is triggered").
- Always use the Oxford comma (e.g. "workflows, tasks, and runs" not "workflows, tasks and runs").
- Place commas and periods inside quotation marks (e.g. "workflows," not "workflows",).

## Content Quality

- Make sure the page is technically accurate.
- Make sure the page is written clearly.
- Make sure the page has a clear motivation.
- Make sure the most important thing about the page is clear in the first paragraph.
- Make sure the page maps to problems our customers face.
- Do NOT reference specific SDK methods or naming.
- Do NOT use writing patterns that signal AI-generated content, including:
  - Em dashes (—) in place of commas or parentheses
  - Filler openers like "In the world of..." or "It's worth noting that..."
  - Bullet-heavy structure where prose would read better
  - Vague superlatives like "seamless," "powerful," "robust," or "intuitive"
  - "Best practices" as a section header or catch-all
  - Excessive hedging like "it's important to note" or "keep in mind that"
  - Lists of exactly three items when two or four would be more natural

## Structure

- Every page should have a clear title that describes what the reader will be able to do after reading it (e.g. "Run a workflow on a schedule" not "Scheduled Workflows").
- All subheadings should use sentence case (e.g. "Running a workflow" not "Running A Workflow").
- Start with the simplest implementation and get progressively more complex, making it clear when tradeoffs need to be made.
- Make sure the page has languages for all tabs, with a to-do for missing examples.
- Python should be AIO first with callouts for sync where appropriate.
- Code examples should be complete and runnable, not pseudocode or partial snippets.
- Make sure we have a comma-separated list of keywords at the bottom of the page.

## Flagging Gaps

- Flag any assumptions the page makes about prior knowledge that isn't linked to.
- Flag where a visual or diagram could be helpful. (TODO-DOCS: Add a diagram here/})
- Flag where a video tutorial could be helpful. (TODO-DOCS: Add a video of the dashboard view \*/)

## Links

- Check all links — no dead links.
- Link to related pages where relevant.
