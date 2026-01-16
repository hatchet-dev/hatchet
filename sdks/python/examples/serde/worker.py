# > Pydantic Serialization/Deserialization

import base64
import zlib
from typing import Annotated, Any

from pydantic import BaseModel, PlainSerializer, ValidationInfo, model_validator

from hatchet_sdk import Context, EmptyModel, Hatchet
from hatchet_sdk.serde import is_in_hatchet_serialization_context

hatchet = Hatchet(debug=True)


def serializor(input_str: str, info: ValidationInfo) -> str:
    if is_in_hatchet_serialization_context(info.context):
        return base64.b64encode(zlib.compress(input_str.encode("utf-8"))).decode(
            "utf-8"
        )
    return input_str


class HatchetOutput(BaseModel):
    result: Annotated[str, PlainSerializer(serializor, str)]

    @model_validator(mode="before")
    @classmethod
    def resolve_secret_large(cls, data: Any, info: ValidationInfo) -> Any:
        if is_in_hatchet_serialization_context(info.context):
            data["result"] = zlib.decompress(
                base64.b64decode(data["result"].encode("utf-8"))
            ).decode("utf-8")
        return data


class TestOutput(BaseModel):
    final_result: str


serde_workflow = hatchet.workflow(
    name="serde-example-workflow", input_validator=EmptyModel
)


@serde_workflow.task()
def generate_result(input: EmptyModel, ctx: Context) -> HatchetOutput:
    return HatchetOutput(result="my_result")


@serde_workflow.task(parents=[generate_result])
def read_result(input: EmptyModel, ctx: Context) -> TestOutput:
    return TestOutput(final_result=ctx.task_output(generate_result).result)


def main() -> None:
    worker = hatchet.worker("test-worker", workflows=[serde_workflow])
    worker.start()


# !!

if __name__ == "__main__":
    main()
