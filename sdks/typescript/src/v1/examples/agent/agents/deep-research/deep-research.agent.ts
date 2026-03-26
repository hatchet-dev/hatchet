import { search } from '@/agents/deep-research/tools/search.tool';
import { planSearch, PlanSearchOutput } from '@/agents/deep-research/tools/plan-search.tool';
import { websiteToMd } from '@/agents/deep-research/tools/website-to-md.tool';
import { summarize } from '@/agents/deep-research/tools/summarize.tool';
import { judgeResults } from '@/agents/deep-research/tools/judge-results.tool';
import { extractFacts } from '@/agents/deep-research/tools/extract-facts.tool';
import { judgeFacts, JudgeFactsOutput } from '@/agents/deep-research/tools/judge-facts.tool';
import { z } from 'zod';
import { client } from './../../client';

const MessageSchema = z.object({
  message: z.string(),
});

const SourceSchema = z.object({
  url: z.string(),
  title: z.string().optional(),
  index: z.number(),
});

const ResponseSchema = z.object({
  result: z.object({
    isComplete: z.boolean(),
    reason: z.string(),
    sources: z.array(SourceSchema),
    summary: z.string().optional(),
    facts: z
      .array(
        z.object({
          text: z.string(),
          sourceIndex: z.number(),
        })
      )
      .optional(),
    iterations: z.number().optional(),
    factsJudgment: z
      .object({
        reason: z.string(),
        hasEnoughFacts: z.boolean(),
        missingAspects: z.array(z.string()),
      })
      .optional(),
    searchPlans: z.string().optional(),
  }),
});

type Source = z.infer<typeof SourceSchema>;
type Fact = {
  text: string;
  sourceIndex: number;
};

export const deepResearchAgent = client.agent({
  name: 'deep-research-agent',
  description: 'A tool that performs deep research on a given query',
  inputSchema: MessageSchema,
  outputSchema: ResponseSchema,
  executionTimeout: '15m',
  fn: async (input, ctx) => {
    ctx.logger.info(`Starting deep research agent with query: ${input.message}`);

    let iteration = 0;
    const maxIterations = 3;
    const allFacts: Fact[] = [];
    const allSources: Source[] = [];
    let missingAspects: string[] = [];
    let plan: PlanSearchOutput | undefined = undefined;
    let factsJudgment: JudgeFactsOutput | undefined = undefined;

    while (!ctx.cancelled && iteration < maxIterations) {
      iteration++;
      ctx.logger.info(`Starting iteration ${iteration}/${maxIterations}`);

      // Plan the search based on the query, existing facts, and missing aspects
      ctx.logger.info(
        `Planning search with ${allFacts.length} existing facts and ${missingAspects.length} missing aspects`
      );

      plan = await planSearch.run({
        query: input.message,
        existingFacts: allFacts.map((f) => f.text),
        missingAspects: missingAspects,
      });

      ctx.logger.info(`Search plan for iteration ${iteration}: ${plan.reasoning}. Queries:`);

      for (const query of plan.queries) {
        ctx.logger.info(`${query}`);
      }

      ctx.logger.info(`Executing ${plan.queries.length} search queries`);
      const results = await search.run(plan.queries.map((query: string) => ({ query })));

      // Flatten and deduplicate sources
      const newSources = results.flatMap((result) => result.sources);
      const uniqueSources = new Map(
        newSources.map((source, index) => [source.url, { ...source, index }])
      );

      ctx.logger.info(
        `Found ${newSources.length} new sources, ${uniqueSources.size} unique sources`
      );

      // Add new sources to all sources
      allSources.push(...Array.from(uniqueSources.values()));

      // Convert sources to markdown
      ctx.logger.info(`Converting ${uniqueSources.size} sources to markdown`);
      const mdResults = await websiteToMd.run(
        Array.from(uniqueSources.values())
          .sort((a, b) => a.index - b.index)
          .map((source) => ({
            url: source.url,
            index: source.index,
            title: source.title || '',
          }))
      );

      // Extract facts from each source
      ctx.logger.info('Extracting facts from markdown content');
      const factsResults = await extractFacts.run(
        mdResults.map((result) => ({
          source: result.markdown,
          query: input.message,
          sourceInfo: {
            url: result.url,
            title: result.title,
            index: result.index,
          },
        }))
      );

      // Add new facts to all facts
      const newFacts = factsResults.flatMap((result) => result.facts);
      allFacts.push(...newFacts);
      ctx.logger.info(`Extracted ${newFacts.length} new facts, total facts: ${allFacts.length}`);

      // Judge if we have enough facts
      ctx.logger.info('Judging if we have enough facts');
      factsJudgment = await judgeFacts.run({
        query: input.message,
        facts: allFacts.map((f) => f.text),
      });

      // Update missing aspects for next iteration
      missingAspects = factsJudgment.missingAspects;
      ctx.logger.info(`Missing aspects: ${missingAspects.join(', ')}`);

      // If we have enough facts or reached max iterations, generate final summary
      if (factsJudgment.hasEnoughFacts || iteration >= maxIterations) {
        ctx.logger.info(
          `Generating final summary (hasEnoughFacts: ${
            factsJudgment.hasEnoughFacts
          }, reachedMaxIterations: ${iteration >= maxIterations})`
        );
        break;
      }
    }

    // Always summarize and judge results after the loop
    const summarizeResult = await summarize.run({
      text: input.message,
      facts: allFacts,
      sources: allSources,
    });

    ctx.logger.info('Judging final results');
    const judgeResult = await judgeResults.run({
      query: input.message,
      result: summarizeResult.summary,
    });

    ctx.logger.info(
      `Deep research complete (isComplete: ${judgeResult.isComplete}, totalFacts: ${allFacts.length}, totalSources: ${allSources.length}, iterations: ${iteration})`
    );

    return {
      result: {
        isComplete: judgeResult.isComplete,
        reason: judgeResult.reason,
        sources: allSources,
        summary: summarizeResult.summary,
        facts: allFacts,
        iterations: iteration,
        factsJudgment: factsJudgment,
        searchPlans: plan?.reasoning,
      },
    };
  },
});
