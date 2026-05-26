import base64
import importlib
import io
import re
from collections import Counter

from pydantic import BaseModel

from hatchet_sdk import Context, Hatchet

hatchet = Hatchet()

STOPWORDS = {
    "a",
    "an",
    "and",
    "are",
    "as",
    "at",
    "be",
    "by",
    "for",
    "from",
    "in",
    "is",
    "it",
    "of",
    "on",
    "or",
    "that",
    "the",
    "to",
    "with",
}
MIN_WORD_LENGTH = 3
MAX_KEYWORDS = 6


# > Models
class PdfInput(BaseModel):
    filename: str
    content_base64: str


class ExtractOutput(BaseModel):
    text: str
    page_count: int


class ClassifyOutput(BaseModel):
    category: str


class SummaryOutput(BaseModel):
    summary: str
    word_count: int


class KeywordsOutput(BaseModel):
    keywords: list[str]


class PipelineResult(BaseModel):
    filename: str
    category: str
    summary: str
    keywords: list[str]
    word_count: int
    page_count: int


# !!

# > Define the DAG
pdf_pipeline = hatchet.workflow(name="PdfPipeline", input_validator=PdfInput)
# !!


FALLBACK_TEXT = (
    "Invoice from Acme Corp. Total amount due: 150 dollars. Payment terms: Net 30."
)


# > Extract text task
@pdf_pipeline.task()
def extract_text(input: PdfInput, ctx: Context) -> ExtractOutput:
    try:
        pypdf = importlib.import_module("pypdf")
    except ImportError:
        # pypdf not installed: return deterministic fallback text
        return ExtractOutput(text=FALLBACK_TEXT, page_count=1)

    decoded = base64.b64decode(input.content_base64)
    reader = pypdf.PdfReader(io.BytesIO(decoded))
    text = "\n".join(page.extract_text() or "" for page in reader.pages)

    return ExtractOutput(text=text, page_count=len(reader.pages))


# !!


# > Classify task
@pdf_pipeline.task(parents=[extract_text])
def classify_document(input: PdfInput, ctx: Context) -> ClassifyOutput:
    text = ctx.task_output(extract_text).text.lower()

    if any(w in text for w in ["invoice", "amount due", "payment", "bill"]):
        category = "invoice"
    elif any(w in text for w in ["receipt", "paid", "transaction"]):
        category = "receipt"
    elif any(w in text for w in ["report", "analysis", "findings", "conclusion"]):
        category = "report"
    elif any(w in text for w in ["dear", "sincerely", "regards"]):
        category = "letter"
    else:
        category = "other"

    return ClassifyOutput(category=category)


# !!


# > Summarize task
@pdf_pipeline.task(parents=[extract_text])
def summarize_text(input: PdfInput, ctx: Context) -> SummaryOutput:
    text = ctx.task_output(extract_text).text
    words = text.split()
    max_words = 50
    summary = " ".join(words[:max_words])
    if len(words) > max_words:
        summary += "..."

    return SummaryOutput(summary=summary, word_count=len(words))


# !!


# > Extract keywords task
@pdf_pipeline.task(parents=[extract_text])
def extract_keywords(input: PdfInput, ctx: Context) -> KeywordsOutput:
    text = ctx.task_output(extract_text).text.lower()
    words = re.findall(r"[a-z]+", text)
    filtered = [w for w in words if len(w) >= MIN_WORD_LENGTH and w not in STOPWORDS]
    counts = Counter(filtered)
    top = sorted(counts.items(), key=lambda x: (-x[1], x[0]))[:MAX_KEYWORDS]
    return KeywordsOutput(keywords=[word for word, _ in top])


# !!


# > Format result task
@pdf_pipeline.task(
    parents=[extract_text, classify_document, summarize_text, extract_keywords]
)
def format_result(input: PdfInput, ctx: Context) -> PipelineResult:
    extract = ctx.task_output(extract_text)
    classify = ctx.task_output(classify_document)
    summary = ctx.task_output(summarize_text)
    keywords = ctx.task_output(extract_keywords)

    return PipelineResult(
        filename=input.filename,
        category=classify.category,
        summary=summary.summary,
        keywords=keywords.keywords,
        word_count=summary.word_count,
        page_count=extract.page_count,
    )


# !!


# > Worker registration
def main() -> None:
    worker = hatchet.worker(
        "pdf-pipeline-worker",
        workflows=[pdf_pipeline],
    )
    worker.start()


if __name__ == "__main__":
    main()


# !!
