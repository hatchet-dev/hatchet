import importlib
import io


def make_sample_pdf(text: str) -> bytes:
    """Create a minimal single-page PDF containing the given text."""
    try:
        pypdf = importlib.import_module("pypdf")
        pypdf_generic = importlib.import_module("pypdf.generic")
    except ImportError:
        raise ImportError(
            "pypdf is required for this example. "
            "Install it in your Python environment before running."
        )

    writer = pypdf.PdfWriter()
    writer.add_blank_page(width=612, height=792)
    page = writer.pages[0]

    font = pypdf_generic.DictionaryObject()
    font[pypdf_generic.NameObject("/Type")] = pypdf_generic.NameObject("/Font")
    font[pypdf_generic.NameObject("/Subtype")] = pypdf_generic.NameObject("/Type1")
    font[pypdf_generic.NameObject("/BaseFont")] = pypdf_generic.NameObject("/Helvetica")
    font_ref = writer._add_object(font)

    resources = page.get("/Resources", pypdf_generic.DictionaryObject())
    if "/Font" not in resources:
        resources[pypdf_generic.NameObject("/Font")] = pypdf_generic.DictionaryObject()
    resources["/Font"][pypdf_generic.NameObject("/F1")] = font_ref
    page[pypdf_generic.NameObject("/Resources")] = resources

    stream = pypdf_generic.DecodedStreamObject()
    stream.set_data(f"BT /F1 12 Tf 72 720 Td ({text}) Tj ET".encode())
    page[pypdf_generic.NameObject("/Contents")] = writer._add_object(stream)

    buf = io.BytesIO()
    writer.write(buf)
    return buf.getvalue()
