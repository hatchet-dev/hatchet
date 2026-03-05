# Third-party integration example - requires: pip install openai
# See: /guides/rag-and-indexing

from openai import OpenAI

client = OpenAI()


# > OpenAI embedding usage
def embed(text: str) -> list[float]:
    r = client.embeddings.create(model="text-embedding-3-small", input=text)
    return r.data[0].embedding
