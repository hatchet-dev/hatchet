from .hatchet import hatchet
from hatchet_sdk import Context
from bs4 import BeautifulSoup
from openai import OpenAI
import requests
import time

openai = OpenAI()


@hatchet.workflow(on_events=["trigger:create"])
class ManualTriggerWorkflow:
    @hatchet.step()
    def step1(self, context):
        messages = context.workflow_input()['request']['messages']
        print("> starting step1", messages)
        return {"status": "thinking"}

    @hatchet.step(parents=["step1"])
    def step2(self, context):
        print("starting step2")
        return {"status": "writing a response"}

    @hatchet.step(parents=["step2"], timeout='5m')
    def step3(self, context):
        messages = context.workflow_input()['request']['messages']
        prompt = "Compose a poem that explains the concept of recursion in programming."
        model = "gpt-3.5-turbo"

        completion = openai.chat.completions.create(
            model=model,
            messages=[
                {"role": "system", "content": prompt},
            ] + messages
        )

        return {
            "complete": "true",
            "status": "idle",
            "message": completion.choices[0].message.content,
        }


@hatchet.workflow(on_events=["question:create"])
class GenerateWorkflow:

    # @hatchet.concurrency(max_runs=1)
    # def limit_to_one_per_session(self, context: Context):
    #     return context.workflow_input()['session_id']

    # @hatchet.step()
    # def request_analytics(self, context: Context):
    #     message = ctx.workflow_input()["messages"][-1]

    #     sentiment_prompt = f"Describe the following request as FRUSTRATED, NEUTRAL, SATISFIED, or ANGRY: {message}"
    #     model = "gpt-3.5-turbo"

    #     sentiment_result = openai.chat.completions.create(
    #         model=model,
    #         messages=[
    #             {"role": "system", "content": sentiment_prompt},
    #             {"role": "user", "content": ctx.workflow_input()[
    #                 "message"]}
    #         ]
    #     )

    #     category_prompt = f"Categorize the following request as ISSUE, FEATURE_REQUEST, or OTHER: {message}"
    #     model = "gpt-3.5-turbo"

    #     category_result = openai.chat.completions.create(
    #         model=model,
    #         messages=[
    #             {"role": "system", "content": category_prompt},
    #             {"role": "user", "content": ctx.workflow_input()[
    #                 "message"]}
    #         ]
    #     )

    #     return {
    #         "n_messages": len(ctx.workflow_input()["messages"]),
    #         "sentiment": sentiment_result.choices[0].message,
    #         "category": category_result.choices[0].message,
    #     }

    @hatchet.step()
    def start(self, context: Context):
        return {
            "status": "reading hatchet docs",
        }

    @hatchet.step(parents=["start"])
    def load_docs(self, context: Context):
        # use beautiful soup to parse the html content
        url = context.workflow_input()['request']['url']

        html_content = requests.get(url).text
        soup = BeautifulSoup(html_content, 'html.parser')
        element = soup.find('body')
        text_content = element.get_text(separator=' | ')

        return {
            "status": "making sense of the docs",
            "docs": text_content,
        }

    @hatchet.step(parents=["load_docs"])
    def reason_docs(self, ctx: Context):
        message = ctx.workflow_input()['request']["messages"][-1]
        docs = ctx.step_output("load_docs")['docs']

        prompt = ctx.overrides(
            'reason:prompt',
            "The user is asking the following question:\
            {{message}}\
            What are the most relevant sentences in the following document?\
            {{docs}}")

        prompt = prompt.format(message=message['content'], docs=docs)

        model = "gpt-3.5-turbo"  # ctx.override("gpt-3.5-turbo", options=[''])

        completion = openai.chat.completions.create(
            model=model,
            messages=[
                {"role": "system", "content": prompt},
                message
            ]
        )

        return {
            "status": "writing a response",
            "research": completion.choices[0].message.content,
        }

    @hatchet.step(parents=["reason_docs"])
    def generate_response(self, ctx: Context):
        messages = ctx.workflow_input()['request']["messages"]
        research = ctx.step_output("reason_docs")['research']

        prompt = ctx.overrides(
            'answer:prompt',
            "You are a sales engineer for a company called Hatchet.\
            Help address the user's question. \
            Use the following context:\
            {{research}}")

        prompt = prompt.format(research=research)

        model = ctx.overrides('answer:model', "gpt-3.5-turbo")

        completion = openai.chat.completions.create(
            model=model,
            messages=[
                {"role": "system", "content": prompt},
            ] + messages
        )

        return {
            "completed": "true",
            "status": "idle",
            "message": completion.choices[0].message.content,
        }
