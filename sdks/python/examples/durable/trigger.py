import asyncio
import logging
from typing import Optional, Dict
import click
from rich.console import Console
from rich.panel import Panel
from rich.logging import RichHandler
from rich.table import Table

from examples.durable.worker import (
    COMMAND_EVENT_KEY,
    CommandMessage,
    WorkerResponse,
    worker_workflow,
    hatchet,
)
from hatchet_sdk.clients.admin import TriggerWorkflowOptions


# Configure logging with rich handler
logging.basicConfig(
    level=logging.INFO,
    format="%(message)s",
    handlers=[RichHandler(rich_tracebacks=True)]
)
logger = logging.getLogger("trigger")

# Initialize rich console for pretty printing
console = Console()


def create_response_table(response: WorkerResponse) -> Table:
    """Create a rich table for displaying the response"""
    table = Table(show_header=True, header_style="bold magenta")
    table.add_column("Field", style="cyan")
    table.add_column("Value", style="green")
    
    # Add basic info
    table.add_row("Message", response.message)
    table.add_row("Message Hash", response.message_hash)
    
    # Add data fields
    for key, value in response.data.items():
        table.add_row(key, str(value))
    
    return table


async def listen_for_worker_events(workflow_run_id: str) -> None:
    """Listen for events from the worker and display them nicely"""
    logger.info("Starting worker event listener")
    try:
        async for event in hatchet.listener.stream(
            workflow_run_id=workflow_run_id
        ):
            logger.debug(f"Received event: {event}")
            if hasattr(event, 'payload') and event.payload:
                try:
                    # Parse the response
                    logger.debug(f"Parsing payload: {event.payload}")
                    response = WorkerResponse.model_validate_json(
                        event.payload
                    )
                    
                    # Create response table
                    table = create_response_table(response)
                    
                    # Display in a nice panel with the table
                    console.print(
                        Panel(
                            table,
                            title="🤖 Worker Response",
                            border_style="green"
                        )
                    )
                    
                    # Log verification
                    if response.message_hash:
                        logger.info(
                            f"Message verified with hash: "
                            f"{response.message_hash}"
                        )
                    
                except Exception as e:
                    logger.error(f"Failed to parse response: {e}")
                    console.print(
                        f"Error parsing response: {e}", 
                        style="red"
                    )
    except Exception as e:
        logger.error(f"Event listener error: {e}")
        raise


async def send_command(
    workflow_run_id: str, 
    command: str, 
    data: Optional[Dict] = None
) -> None:
    """
    Send a command to the worker
    
    Args:
        workflow_run_id: ID of the workflow run
        command: Command to send
        data: Optional data payload for the command
        
    Raises:
        Exception: If command sending fails
    """
    try:
        logger.info(f"Sending command: {command}")
        
        # Create command message with proper structure
        message = CommandMessage(
            command=command,
            data={
                "timestamp": "2024-03-21T10:00:00Z",
                "command_id": command.lower(),
                **(data or {})
            }
        )
        
        # Calculate message hash for verification
        msg_hash = message.calculate_hash()
        logger.debug(f"Message hash: {msg_hash}")
        
        # Send command event with proper structure
        hatchet.event.push(
            COMMAND_EVENT_KEY,
            message.model_dump()
        )
        
        # Display confirmation
        console.print(
            Panel(
                f"Command: {command}\nHash: {msg_hash}",
                title="📤 Sent Command",
                border_style="blue"
            )
        )
    except Exception as e:
        logger.error(f"Failed to send command: {e}")
        raise


@click.command()
def main() -> None:
    """Simple CLI for communicating with the worker"""
    try:
        logger.info("Starting communication workflow")
        
        # Start with an initial command
        initial_command = CommandMessage(
            command="start",
            data={"init": True}
        )
        
        # Start the workflow
        logger.debug("Starting workflow with initial command")
        ref = asyncio.run(
            worker_workflow.aio_run_no_wait(
                input=initial_command,
                options=TriggerWorkflowOptions(
                    key="communication-flow-1",
                    sticky=True,
                )
            )
        )
        
        logger.info(f"Workflow started: {ref.workflow_run_id}")
        console.print(
            Panel(
                f"ID: {ref.workflow_run_id}\n"
                f"Hash: {initial_command.calculate_hash()}",
                title="🚀 Workflow Started",
                border_style="green"
            )
        )
        
        async def cli_loop():
            # Start the event listener
            logger.debug("Starting event listener task")
            listener = asyncio.create_task(
                listen_for_worker_events(ref.workflow_run_id)
            )
            
            try:
                while True:
                    # Get command from user
                    command = click.prompt(
                        "\n📝 Enter command",
                        type=str,
                        default="ping"
                    )
                    logger.debug(f"User entered command: {command}")
                    
                    # Handle exit command
                    if command.lower() in ["exit", "quit", "stop"]:
                        logger.info("Stopping workflow")
                        await send_command(
                            ref.workflow_run_id, 
                            "stop"
                        )
                        console.print("\n👋 Goodbye!", style="cyan")
                        break
                    
                    # Send the command with timestamp
                    await send_command(
                        ref.workflow_run_id,
                        command,
                        {
                            "timestamp": "2024-03-21T10:00:00Z",
                            "command_id": command.lower()
                        }
                    )
                    
                    # Small delay to prevent flooding
                    await asyncio.sleep(0.1)
                    
            except KeyboardInterrupt:
                logger.warning("Interrupted by user")
                console.print("\n⚠️ Interrupted by user", style="yellow")
                await send_command(ref.workflow_run_id, "stop")
                console.print("\n👋 Goodbye!", style="cyan")
            
            finally:
                # Cancel the listener
                listener.cancel()
                try:
                    await listener
                except asyncio.CancelledError:
                    pass
        
        # Run the CLI loop
        asyncio.run(cli_loop())
        
    except Exception as e:
        logger.error(f"Application error: \nError: {str(e)}")
        raise


if __name__ == "__main__":
    main()
