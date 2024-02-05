from .hatchet import hatchet
from hatchet_sdk import Context
from bs4 import BeautifulSoup
from openai import OpenAI
import requests

openai = OpenAI()


@hatchet.workflow(on_events=["question:create"])
class GenerateWorkflow:
    def __init__(self):
        self.my_value = "test"

    @hatchet.concurrency(max_runs=1)
    def limit_to_one_per_session(self, context: Context):
        return context.workflow_input()['session_id']

    @hatchet.step()
    def request_analytics(self, ctx: Context):
        message = ctx.workflow_input()["messages"][-1]

        sentiment_prompt = f"Describe the following request as FRUSTRATED, NEUTRAL, SATISFIED, or ANGRY: {message}"
        model = "gpt-3.5-turbo"

        sentiment_result = openai.chat.completions.create(
            model=model,
            messages=[
                {"role": "system", "content": sentiment_prompt},
                {"role": "user", "content": ctx.workflow_input()[
                    "message"]}
            ]
        )

        category_prompt = f"Categorize the following request as ISSUE, FEATURE_REQUEST, or OTHER: {message}"
        model = "gpt-3.5-turbo"

        category_result = openai.chat.completions.create(
            model=model,
            messages=[
                {"role": "system", "content": category_prompt},
                {"role": "user", "content": ctx.workflow_input()[
                    "message"]}
            ]
        )

        return {
            "n_messages": len(ctx.workflow_input()["messages"]),
            "sentiment": sentiment_result.choices[0].message,
            "category": category_result.choices[0].message,
        }

    @hatchet.step()
    def load_docs(self, context: Context):
        # use beautiful soup to parse the html content
        url = context.workflow_input()['url']
        html_content = requests.get(url)
        soup = BeautifulSoup(html_content, 'html.parser')
        element = soup.find('body')
        text_content = element.get_text(separator=' | ')

        return {
            "docs": text_content,
        }

    @hatchet.step(parents=["load_docs"])
    def reason_docs(self, ctx: Context):
        message = ctx.workflow_input()["messages"][-1]
        docs = ctx.step_output("load_docs")
        prompt = sldkfjsdf.replace(
            ctx.override('prompt',
                         "The user is asking the following question:<message> \n\n What are the most relevant sentences in the following document?\n<docs>",
                         ),
            {
                "<message>": message,
                "<docs>": docs
            }
        )

        model = ctx.override("gpt-3.5-turbo", options=[''])

        completion = openai.chat.completions.create(
            model=model,
            messages=[
                {"role": "system", "content": prompt},
                {"role": "user", "content": message}
            ]
        )

        return {
            "research": completion.choices[0].message,
        }

    @hatchet.step(parents=["reason_docs"])
    def generate_response(self, ctx: Context):
        message = ctx.workflow_input()["messages"]
        prompt = "Compose a poem that explains the concept of recursion in programming."
        model = "gpt-3.5-turbo"

        completion = openai.chat.completions.create(
            model=model,
            messages=[
                {"role": "system", "content": prompt},
                {"role": "user", "content": message}
            ]
        )

        return {
            "message": completion.choices[0].message,
        }
