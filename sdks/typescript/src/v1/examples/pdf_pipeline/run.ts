// > Trigger the pipeline
import { pdfPipeline, PdfInput } from './workflow';
import { makeSamplePdf } from './sample-pdf';

async function main() {
  const pdfBytes = makeSamplePdf(
    'Invoice from Acme Corp. Total amount due: 150 dollars. Payment terms: Net 30.'
  );
  const input: PdfInput = {
    filename: 'sample-invoice.pdf',
    contentBase64: pdfBytes.toString('base64'),
  };

  const result = await pdfPipeline.run(input);
  console.log('Pipeline result:', result);
}
// !!

if (require.main === module) {
  main()
    .catch(console.error)
    .finally(() => process.exit(0));
}
