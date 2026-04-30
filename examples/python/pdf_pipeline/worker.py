import base64
import io

from pydantic import BaseModel

from hatchet_sdk import Context, Hatchet

hatchet = Hatchet()


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


class PipelineResult(BaseModel):
    filename: str
    category: str
    summary: str
    word_count: int
    page_count: int



# > Define the DAG
pdf_pipeline = hatchet.workflow(name="PdfPipeline", input_validator=PdfInput)


# > Extract text task
@pdf_pipeline.task()
def extract_text(input: PdfInput, ctx: Context) -> ExtractOutput:
    try:
        from pypdf import PdfReader
    except ImportError:
        raise ImportError(
            "pypdf is required for this example. "
            "Install it in your Python environment before running."
        )

    decoded = base64.b64decode(input.content_base64)
    reader = PdfReader(io.BytesIO(decoded))
    text = "\n".join(page.extract_text() or "" for page in reader.pages)

    return ExtractOutput(text=text, page_count=len(reader.pages))




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




# > Format result task
@pdf_pipeline.task(parents=[extract_text, classify_document, summarize_text])
def format_result(input: PdfInput, ctx: Context) -> PipelineResult:
    extract = ctx.task_output(extract_text)
    classify = ctx.task_output(classify_document)
    summary = ctx.task_output(summarize_text)

    return PipelineResult(
        filename=input.filename,
        category=classify.category,
        summary=summary.summary,
        word_count=summary.word_count,
        page_count=extract.page_count,
    )




def main() -> None:
    worker = hatchet.worker(
        "pdf-pipeline-worker",
        workflows=[pdf_pipeline],
    )
    worker.start()


if __name__ == "__main__":
    main()
