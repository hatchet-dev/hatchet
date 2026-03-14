# Third-party integration example - requires: pip install cohere
# See: /guides/rag-and-indexing

import cohere

client = cohere.Client()


# > Cohere embedding usage
def embed(text: str) -> list[float]:
    r = client.embed(texts=[text], model="embed-english-v3.0", input_type="search_document")
    if isinstance(r, cohere.EmbeddingsFloatsEmbedResponse):
        return list(r.embeddings[0])
    raise TypeError(f"Expected float embeddings, got {type(r).__name__}")
# !!
