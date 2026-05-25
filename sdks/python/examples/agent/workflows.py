# > Agent
from typing import Any

import httpx
from pydantic import BaseModel

from hatchet_sdk import Context, Hatchet
from hatchet_sdk.runnables.workflow import MCPProvider

hatchet = Hatchet(debug=True)
# !!


# > Models
class TemperatureCoords(BaseModel):
    latitude: float
    longitude: float


class TemperatureInput(BaseModel):
    location_name: str
    coords: TemperatureCoords


class TemperatureContent(BaseModel):
    text: str


# !!


# > Workflow definition
get_temperature_workflow = hatchet.workflow(
    name="get_temperature",
    input_validator=TemperatureInput,
    description="Get the current temperature at a location",
)


@get_temperature_workflow.task()
async def get_temperature(input: TemperatureInput, ctx: Context) -> TemperatureContent:
    async with httpx.AsyncClient() as client:
        response = await client.get(
            "https://api.open-meteo.com/v1/forecast",
            params={
                "latitude": input.coords.latitude,
                "longitude": input.coords.longitude,
                "current": "temperature_2m",
                "temperature_unit": "fahrenheit",
            },
        )
        data = response.json()

    return TemperatureContent(
        text=f"Temperature in {input.location_name}: {data['current']['temperature_2m']}°F"
    )


# !!


# > Standalone task
@hatchet.task(
    input_validator=TemperatureInput,
    description="Get the current temperature at a location",
)
async def get_temperature_standalone(
    input: TemperatureInput, ctx: Context
) -> TemperatureContent:
    async with httpx.AsyncClient() as client:
        response = await client.get(
            "https://api.open-meteo.com/v1/forecast",
            params={
                "latitude": input.coords.latitude,
                "longitude": input.coords.longitude,
                "current": "temperature_2m",
                "temperature_unit": "fahrenheit",
            },
        )
        data = response.json()

    return TemperatureContent(
        text=f"Temperature in {input.location_name}: {data['current']['temperature_2m']}°F"
    )


# !!


# > Create MCP tools
def create_temperature_workflow_tool_claude() -> Any:
    return get_temperature_workflow.mcp_tool(MCPProvider.CLAUDE)


def create_temperature_workflow_tool_openai() -> Any:
    return get_temperature_workflow.mcp_tool(MCPProvider.OPENAI)


def create_temperature_task_tool_claude() -> Any:
    return get_temperature_standalone.mcp_tool(MCPProvider.CLAUDE)


def create_temperature_task_tool_openai() -> Any:
    return get_temperature_standalone.mcp_tool(MCPProvider.OPENAI)


# !!
