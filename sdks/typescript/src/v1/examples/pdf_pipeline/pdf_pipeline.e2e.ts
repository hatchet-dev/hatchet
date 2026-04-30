import { makeE2EClient } from '../__e2e__/harness';
import { pdfPipeline, PdfInput } from './workflow';
import { makeSamplePdf } from './sample-pdf';

describe('pdf-pipeline-e2e', () => {
  const hatchet = makeE2EClient();

  it('processes a PDF through the full DAG pipeline', async () => {
    const pdfBytes = makeSamplePdf('Invoice from Acme Corp. Total amount due: 150 dollars.');
    const input: PdfInput = {
      filename: 'test-invoice.pdf',
      contentBase64: pdfBytes.toString('base64'),
    };

    const result = await pdfPipeline.run(input);

    expect((result as any).extract_text.pageCount).toBe(1);
    expect((result as any).extract_text.text).toContain('Invoice');

    expect((result as any).classify_document.category).toBe('invoice');

    expect((result as any).summarize_text.wordCount).toBeGreaterThan(0);
    expect((result as any).summarize_text.summary.length).toBeGreaterThan(0);

    const { keywords } = (result as any).extract_keywords;
    expect(keywords.length).toBeGreaterThan(0);
    expect(keywords).toContain('acme');
    expect(keywords).toContain('invoice');

    expect((result as any).format_result.filename).toBe('test-invoice.pdf');
    expect((result as any).format_result.category).toBe('invoice');
    expect((result as any).format_result.pageCount).toBe(1);
    expect((result as any).format_result.keywords).toEqual(keywords);
  }, 60_000);
});
