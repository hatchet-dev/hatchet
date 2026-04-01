# > Agent
import httpx
from pydantic import BaseModel

from hatchet_sdk import Context, Hatchet
from hatchet_sdk.runnables.workflow import MCPProvider

hatchet = Hatchet(debug=True)


class TemperatureCoords(BaseModel):
    latitude: float
    longitude: float


class TemperatureInput(BaseModel):
    location_name: str
    coords: TemperatureCoords


class TemperatureContent(BaseModel):
    text: str


temp_workflow = hatchet.workflow(
    name="get_temperature", input_validator=TemperatureInput,
    description="Get the current temperature at a location",
)


@temp_workflow.task()
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


@hatchet.task(input_validator=TemperatureInput, description="Get the current temperature at a location",
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

# You can use a workflow
temperature_tool_claude = temp_workflow.mcp_tool(
    MCPProvider.CLAUDE
)

# Or a standalone task
temperature_tool_claude = get_temperature_standalone.mcp_tool(
    MCPProvider.CLAUDE
)

# You can use a workflow
temperature_tool_openai = temp_workflow.mcp_tool(
    MCPProvider.OPENAI,
)

# Or a standalone task
temperature_tool_openai = get_temperature_standalone.mcp_tool(
    MCPProvider.OPENAI,
)

def main() -> None:
    worker = hatchet.worker(
        "test-worker", workflows=[temp_workflow, get_temperature_standalone]
    )
    worker.start()


if __name__ == "__main__":
    main()
