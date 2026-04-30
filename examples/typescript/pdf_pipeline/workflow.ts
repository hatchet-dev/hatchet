import { writeFileSync, rmSync, mkdtempSync } from 'fs';
import { join } from 'path';
import { tmpdir } from 'os';
import { hatchet } from '../hatchet-client';

// > Models
export type PdfInput = {
  filename: string;
  contentBase64: string;
};

export type ExtractOutput = {
  text: string;
  pageCount: number;
};

export type ClassifyOutput = {
  category: string;
};

export type SummaryOutput = {
  summary: string;
  wordCount: number;
};

export type PipelineResult = {
  filename: string;
  category: string;
  summary: string;
  wordCount: number;
  pageCount: number;
};

/**
 * Extract text from a PDF buffer using pdf2json.
 * The file-path API is the most reliable path for this example.
 */
async function extractPdfText(pdfBuffer: Buffer): Promise<{ text: string; pageCount: number }> {
  let PDFParser: any;
  try {
    PDFParser = require('pdf2json');
  } catch {
    throw new Error(
      'pdf2json v4 is required for this example. Install it in your project before running.'
    );
  }

  const tmpDir = mkdtempSync(join(tmpdir(), 'hatchet-pdf-'));
  const tmpFile = join(tmpDir, 'input.pdf');
  writeFileSync(tmpFile, pdfBuffer);

  try {
    const pdfData: any = await new Promise((resolve, reject) => {
      const parser = new PDFParser();
      parser.on('pdfParser_dataReady', resolve);
      parser.on('pdfParser_dataError', (err: any) => reject(new Error(err.parserError)));
      parser.loadPDF(tmpFile);
    });

    const pageCount = pdfData.Pages.length;
    const text = pdfData.Pages.map((page: any) =>
      page.Texts.map((t: any) => t.R.map((r: any) => decodeURIComponent(r.T)).join('')).join(' ')
    ).join('\n');

    return { text, pageCount };
  } finally {
    try {
        rmSync(tmpDir, { recursive: true, force: true });
    } catch {
      // best-effort cleanup
    }
  }
}

// > Define the DAG
export const pdfPipeline = hatchet.workflow<PdfInput>({
  name: 'pdf-pipeline',
});

// > Extract text task
const extractText = pdfPipeline.task({
  name: 'extract_text',
  fn: async (input: PdfInput) => {
    const decoded = Buffer.from(input.contentBase64, 'base64');
    const { text, pageCount } = await extractPdfText(decoded);
    return { text, pageCount };
  },
});

// > Classify task
const classifyDocument = pdfPipeline.task({
  name: 'classify_document',
  parents: [extractText],
  fn: async (input: PdfInput, ctx) => {
    const { text } = await ctx.parentOutput(extractText);
    const lower = text.toLowerCase();

    let category: string;
    if (['invoice', 'amount due', 'payment', 'bill'].some((w) => lower.includes(w))) {
      category = 'invoice';
    } else if (['receipt', 'paid', 'transaction'].some((w) => lower.includes(w))) {
      category = 'receipt';
    } else if (['report', 'analysis', 'findings', 'conclusion'].some((w) => lower.includes(w))) {
      category = 'report';
    } else if (['dear', 'sincerely', 'regards'].some((w) => lower.includes(w))) {
      category = 'letter';
    } else {
      category = 'other';
    }

    return { category };
  },
});

// > Summarize task
const summarizeText = pdfPipeline.task({
  name: 'summarize_text',
  parents: [extractText],
  fn: async (input: PdfInput, ctx) => {
    const { text } = await ctx.parentOutput(extractText);
    const words = text.split(/\s+/).filter(Boolean);
    const maxWords = 50;
    let summary = words.slice(0, maxWords).join(' ');
    if (words.length > maxWords) {
      summary += '...';
    }
    return { summary, wordCount: words.length };
  },
});

// > Format result task
pdfPipeline.task({
  name: 'format_result',
  parents: [extractText, classifyDocument, summarizeText],
  fn: async (input: PdfInput, ctx) => {
    const extract = await ctx.parentOutput(extractText);
    const classify = await ctx.parentOutput(classifyDocument);
    const summary = await ctx.parentOutput(summarizeText);

    return {
      filename: input.filename,
      category: classify.category,
      summary: summary.summary,
      wordCount: summary.wordCount,
      pageCount: extract.pageCount,
    };
  },
});
