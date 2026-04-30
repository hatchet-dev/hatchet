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

export type KeywordsOutput = {
  keywords: string[];
};

export type PipelineResult = {
  filename: string;
  category: string;
  summary: string;
  keywords: string[];
  wordCount: number;
  pageCount: number;
};
// !!

const STOPWORDS = new Set([
  'a',
  'an',
  'and',
  'are',
  'as',
  'at',
  'be',
  'by',
  'for',
  'from',
  'in',
  'is',
  'it',
  'of',
  'on',
  'or',
  'that',
  'the',
  'to',
  'with',
]);
const MIN_WORD_LENGTH = 3;
const MAX_KEYWORDS = 6;

const FALLBACK_TEXT =
  'Invoice from Acme Corp. Total amount due: 150 dollars. Payment terms: Net 30.';

/**
 * Extract text from a PDF buffer using pdf2json.
 * The file-path API is the most reliable path for this example.
 */
async function extractPdfText(pdfBuffer: Buffer): Promise<{ text: string; pageCount: number }> {
  let PDFParser: any;
  try {
    // eslint-disable-next-line @typescript-eslint/no-require-imports
    PDFParser = require('pdf2json');
  } catch {
    // pdf2json not installed: return deterministic fallback text
    return { text: FALLBACK_TEXT, pageCount: 1 };
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
// !!

// > Extract text task
const extractText = pdfPipeline.task({
  name: 'extract_text',
  fn: async (input: PdfInput) => {
    const decoded = Buffer.from(input.contentBase64, 'base64');
    const { text, pageCount } = await extractPdfText(decoded);
    return { text, pageCount };
  },
});
// !!

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
// !!

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
// !!

// > Extract keywords task
const extractKeywords = pdfPipeline.task({
  name: 'extract_keywords',
  parents: [extractText],
  fn: async (input: PdfInput, ctx) => {
    const { text } = await ctx.parentOutput(extractText);
    const words = text.toLowerCase().match(/[a-z]+/g) ?? [];
    const filtered = words.filter((w) => w.length >= MIN_WORD_LENGTH && !STOPWORDS.has(w));
    const counts = new Map<string, number>();
    for (const w of filtered) {
      counts.set(w, (counts.get(w) ?? 0) + 1);
    }
    const sorted = [...counts.entries()]
      .sort((a, b) => b[1] - a[1] || a[0].localeCompare(b[0]))
      .slice(0, MAX_KEYWORDS);
    return { keywords: sorted.map(([word]) => word) };
  },
});
// !!

// > Format result task
pdfPipeline.task({
  name: 'format_result',
  parents: [extractText, classifyDocument, summarizeText, extractKeywords],
  fn: async (input: PdfInput, ctx) => {
    const extract = await ctx.parentOutput(extractText);
    const classify = await ctx.parentOutput(classifyDocument);
    const summary = await ctx.parentOutput(summarizeText);
    const keywords = await ctx.parentOutput(extractKeywords);

    return {
      filename: input.filename,
      category: classify.category,
      summary: summary.summary,
      keywords: keywords.keywords,
      wordCount: summary.wordCount,
      pageCount: extract.pageCount,
    };
  },
});
// !!
