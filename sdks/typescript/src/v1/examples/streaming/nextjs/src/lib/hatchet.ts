// Simple standalone hatchet client for Next.js demo
// This avoids all the complex Node.js dependencies and bundling issues

interface HatchetRuns {
  subscribeToStream(workflowRunId: string): AsyncIterable<string>;
}

interface HatchetClient {
  runs: HatchetRuns;
}

class SimpleHatchetClient implements HatchetClient {
  runs: HatchetRuns = {
    async* subscribeToStream(workflowRunId: string): AsyncIterable<string> {
      // Simulate the streaming workflow from the actual example
      const annaKarenina = `Happy families are all alike; every unhappy family is unhappy in its own way.

Everything was in confusion in the Oblonskys' house. The wife had discovered that the husband was carrying on an intrigue with a French girl, who had been a governess in their family, and she had announced to her husband that she could not go on living in the same house with him.`;

      // Stream in chunks like the real workflow
      const chunkSize = 10;
      for (let i = 0; i < annaKarenina.length; i += chunkSize) {
        const chunk = annaKarenina.slice(i, i + chunkSize);
        yield chunk;
        // Simulate delay like the real workflow (200ms)
        await new Promise(resolve => setTimeout(resolve, 200));
      }
    }
  };

  static init(): HatchetClient {
    return new SimpleHatchetClient();
  }
}

export const hatchet = SimpleHatchetClient.init();
