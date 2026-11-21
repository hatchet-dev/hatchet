from dataclasses import dataclass

import pytest
from pydantic import BaseModel

from hatchet_sdk import Context, Hatchet


class TemperatureCoords(BaseModel):
    latitude: float
    longitude: float


class TemperatureInput(BaseModel):
    location_name: str
    coords: TemperatureCoords


@dataclass
class TemperatureCoordsDataclass:
    latitude: float
    longitude: float


@dataclass
class TemperatureInputDataclass:
    location_name: str
    coords: TemperatureCoordsDataclass


@pytest.mark.asyncio(loop_scope="session")
async def test_mcp_tool_no_description(hatchet: Hatchet) -> None:
    # no description raises
    @hatchet.task(
        name="blahblah",
    )
    async def get_temperature(input: None, ctx: Context) -> None:
        return None

    with pytest.RaisesExc(ValueError, match="has no description"):
        get_temperature.mcp_tool("CLAUDE")


@pytest.mark.asyncio(loop_scope="session")
async def test_mcp_tool_no_input_validator(hatchet: Hatchet) -> None:
    # no input_validator raises
    @hatchet.task(name="blahblah", description="blahblah")
    async def get_temperature(input: None, ctx: Context) -> None:
        return None

    with pytest.RaisesExc(ValueError, match="has no input validator"):
        get_temperature.mcp_tool("CLAUDE")


@pytest.mark.asyncio(loop_scope="session")
async def test_mcp_tool_pydantic(hatchet: Hatchet) -> None:
    @hatchet.task(
        name="blahblah", description="blahblah", input_validator=TemperatureInput
    )
    async def get_temperature(input: TemperatureInput, ctx: Context) -> None:
        return None

    tool = get_temperature.mcp_tool("CLAUDE")
    assert isinstance(tool.input_schema, dict)
    assert "location_name" in tool.input_schema["required"]


@pytest.mark.asyncio(loop_scope="session")
async def test_mcp_tool_dataclass(hatchet: Hatchet) -> None:
    @hatchet.task(
        name="blahblah",
        description="blahblah",
        input_validator=TemperatureInputDataclass,
    )
    async def get_temperature(input: TemperatureInputDataclass, ctx: Context) -> None:
        return None

    tool = get_temperature.mcp_tool("CLAUDE")
    assert isinstance(tool.input_schema, dict)
    assert "location_name" in tool.input_schema["required"]
