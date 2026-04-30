import { pdfPipeline, PdfInput } from './workflow';

function makeTestPdf(text: string): Buffer {
  const stream = `BT /F1 12 Tf 72 720 Td (${text}) Tj ET`;
  const objects = [
    '1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj',
    '2 0 obj\n<< /Type /Pages /Kids [3 0 R] /Count 1 >>\nendobj',
    `3 0 obj\n<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Contents 4 0 R /Resources << /Font << /F1 5 0 R >> >> >>\nendobj`,
    `4 0 obj\n<< /Length ${stream.length} >>\nstream\n${stream}\nendstream\nendobj`,
    '5 0 obj\n<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>\nendobj',
  ];
  let body = '%PDF-1.4\n';
  const offsets: number[] = [];
  for (const obj of objects) {
    offsets.push(body.length);
    body += obj + '\n\n';
  }
  const xrefOffset = body.length;
  let xref = `xref\n0 ${objects.length + 1}\n`;
  xref += '0000000000 65535 f \n';
  for (const off of offsets) {
    xref += String(off).padStart(10, '0') + ' 00000 n \n';
  }
  body += xref;
  body += `trailer\n<< /Size ${objects.length + 1} /Root 1 0 R >>\n`;
  body += `startxref\n${xrefOffset}\n%%EOF`;
  return Buffer.from(body, 'latin1');
}

// > Trigger the pipeline
async function main() {
  const pdfBytes = makeTestPdf(
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
