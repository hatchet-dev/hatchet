# > Agent
import httpx
from pydantic import BaseModel

from hatchet_sdk import Context, Hatchet

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
    name="get_temperature",
    description="Get the current temperature at a location",
    input_validator=TemperatureInput,
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


def main() -> None:
    worker = hatchet.worker(
        "test-worker", workflows=[temp_workflow, get_temperature_standalone]
    )
    worker.start()


if __name__ == "__main__":
    main()
