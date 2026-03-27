# > Simple
import httpx
from pydantic import BaseModel

from hatchet_sdk import Context, Hatchet

hatchet = Hatchet(debug=True)


class TemperatureInput(BaseModel):
    latitude: float
    longitude: float

class TemperatureContent(BaseModel):
    text: str


temp_workflow = hatchet.workflow(name="get_temperature", input_validator=TemperatureInput)


@temp_workflow.task(name="get_temperature")
async def get_temperature(input: TemperatureInput, ctx: Context) -> TemperatureContent:
    async with httpx.AsyncClient() as client:
        response = await client.get(
            "https://api.open-meteo.com/v1/forecast",
            params={
                "latitude": input.latitude,
                "longitude": input.longitude,
                "current": "temperature_2m",
                "temperature_unit": "fahrenheit",
            },
        )
        data = response.json()

    return TemperatureContent(
        text=f"Temperature: {data['current']['temperature_2m']}°F"
    )


def main() -> None:
    worker = hatchet.worker("test-worker", slots=1, workflows=[temp_workflow])
    worker.start()


if __name__ == "__main__":
    main()
