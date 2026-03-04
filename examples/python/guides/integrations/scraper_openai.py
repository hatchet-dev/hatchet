# Third-party integration example - requires: pip install openai
# See: /guides/web-scraping

from openai import OpenAI

client = OpenAI()


# > OpenAI web search usage
def search_and_extract(query: str) -> dict:
    response = client.responses.create(
        model="gpt-4o-mini",
        tools=[{"type": "web_search"}],
        input=query,
    )
    return {"query": query, "content": response.output_text}
