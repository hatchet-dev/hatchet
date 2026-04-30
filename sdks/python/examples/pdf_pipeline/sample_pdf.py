def make_sample_pdf(text: str) -> bytes:
    """Create a minimal single-page PDF containing the given text.

    Uses no external dependencies. Constructs the PDF binary directly.
    """
    stream = f"BT /F1 12 Tf 72 720 Td ({text}) Tj ET"
    objects = [
        "1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj",
        "2 0 obj\n<< /Type /Pages /Kids [3 0 R] /Count 1 >>\nendobj",
        f"3 0 obj\n<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792]"
        f" /Contents 4 0 R /Resources << /Font << /F1 5 0 R >> >> >>\nendobj",
        f"4 0 obj\n<< /Length {len(stream)} >>\nstream\n{stream}\nendstream\nendobj",
        "5 0 obj\n<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>\nendobj",
    ]

    body = "%PDF-1.4\n"
    offsets: list[int] = []
    for obj in objects:
        offsets.append(len(body))
        body += obj + "\n\n"

    xref_offset = len(body)
    xref = f"xref\n0 {len(objects) + 1}\n"
    xref += "0000000000 65535 f \n"
    for off in offsets:
        xref += f"{off:010d} 00000 n \n"

    body += xref
    body += f"trailer\n<< /Size {len(objects) + 1} /Root 1 0 R >>\n"
    body += f"startxref\n{xref_offset}\n%%EOF"

    return body.encode("latin-1")
