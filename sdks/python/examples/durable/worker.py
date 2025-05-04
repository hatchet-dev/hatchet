import json
import logging
import hashlib
from typing import Dict, Any, Literal, Set
from pydantic import BaseModel, Field

from hatchet_sdk import (
    DurableContext,
    Hatchet,
    UserEventCondition,
)

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s [WORKER] %(levelname)s: %(message)s',
    datefmt='%H:%M:%S'
)
logger = logging.getLogger(__name__)

# Initialize Hatchet
logger.info("Initializing Hatchet client")
hatchet = Hatchet(debug=True)

# Event keys for communication
COMMAND_EVENT_KEY = "worker:command"
WORKER_RESPONSE_KEY = "worker:response"


class CommandMessage(BaseModel):
    """Message sent from trigger to worker"""
    command: str = Field(
        ..., 
        description="Command to execute"
    )
    data: Dict[str, Any] = Field(
        default_factory=dict, 
        description="Additional command data"
    )
    type: Literal["command"] = Field(
        default="command", 
        description="Message type"
    )

    def calculate_hash(self) -> str:
        """Calculate a hash of the command and data for verification"""
        content = f"{self.command}:{json.dumps(self.data, sort_keys=True)}"
        return hashlib.sha256(content.encode()).hexdigest()[:8]


class WorkerResponse(BaseModel):
    """Message sent from worker back to trigger"""
    message: str = Field(
        ..., 
        description="Response message"
    )
    data: Dict[str, Any] = Field(
        default_factory=dict, 
        description="Additional response data"
    )
    type: Literal["response"] = Field(
        default="response", 
        description="Message type"
    )
    message_hash: str = Field(
        default="", 
        description="Hash of the original message"
    )


# Create workflow
logger.info("Creating communication workflow")
worker_workflow = hatchet.workflow(
    name="CommunicationWorkflow",
    input_validator=CommandMessage
)


@worker_workflow.durable_task()
async def communication_task(
    input: CommandMessage, 
    ctx: DurableContext
) -> None:
    """
    Single durable task that handles continuous communication
    
    Args:
        input: Initial command message
        ctx: Durable task context
    """
    logger.info(f"Starting worker with command: {input.command}")
    logger.debug(f"Initial input data: {input.data}")
    
    # Set for deduplication
    processed_hashes: Set[str] = set()
    
    while True:  # Keep running until stop command
        try:
            # Stream ready status
            logger.debug("Sending ready status")
            ready_response = WorkerResponse(
                message="Waiting for command...",
                data={"status": "ready"}
            )
            ctx.put_stream(json.dumps(ready_response.model_dump()))
            
            # Wait for next command
            logger.info("Waiting for next command...")
            event_data = await ctx.aio_wait_for(
                "command",
                UserEventCondition(event_key=COMMAND_EVENT_KEY),
            )
            
            # Parse command from event data
            try:
                # Log raw event data for debugging
                logger.debug(f"Received raw event data: {event_data}")
                
                # Handle both string and dict formats
                if isinstance(event_data, str):
                    try:
                        command_data = json.loads(event_data)
                    except json.JSONDecodeError:
                        # If it's not JSON, use as raw command
                        cmd_obj = {"command": event_data, "type": "command"}
                        command_data = cmd_obj
                else:
                    # It's already a dict
                    command_data = event_data
                
                # Create a simple CommandMessage for processing
                if "command" not in command_data and isinstance(command_data, dict):
                    # Try to find command in standard event data format
                    command_str = "ping"  # Default command
                    for key, value in command_data.items():
                        if isinstance(value, dict) and "command" in value:
                            command_str = value["command"]
                            command_data = value
                            break
                    
                    if "command" not in command_data:
                        # Just create a simple command
                        command_data = {
                            "command": command_str,
                            "data": {},
                            "type": "command"
                        }
                
                # Create command object
                logger.debug(f"Using command data: {command_data}")
                command = CommandMessage.model_validate(command_data)
                msg_hash = command.calculate_hash()
                
                # Skip if already processed (deduplication)
                if msg_hash in processed_hashes:
                    logger.debug(
                        f"Skipping duplicate command with hash: {msg_hash}"
                    )
                    continue
                
                processed_hashes.add(msg_hash)
                logger.info(
                    f"Processed command: {command.command} (hash: {msg_hash})"
                )
            except Exception as e:
                import traceback
                tb_str = traceback.format_exc()
                logger.error(
                    f"Failed to parse command: {str(e)}\n{tb_str}"
                )
                raise
            
            # Handle stop command
            if command.command.lower() == "stop":
                logger.info("Received stop command, shutting down...")
                final_response = WorkerResponse(
                    message="Stopping worker...",
                    data={"status": "stopping"},
                    message_hash=msg_hash
                )
                ctx.put_stream(json.dumps(final_response.model_dump()))
                break
            
            # Process command
            logger.info(f"Processing command: {command.command}")
            
            # Create response with command verification
            response = WorkerResponse(
                message=f"Processed command: {command.command}",
                data={
                    "received_data": command.data,
                    "timestamp": "2024-03-21T10:00:00Z",
                    "command_verified": True,
                    "original_command": command.command
                },
                message_hash=msg_hash
            )
            
            # Stream response
            logger.debug(f"Sending response with hash {msg_hash}")
            ctx.put_stream(json.dumps(response.model_dump()))
            
        except Exception as e:
            import traceback
            tb_str = traceback.format_exc()
            logger.error(
                f"Error processing command: {str(e)}\n{tb_str}"
            )
            error_response = WorkerResponse(
                message=f"Error processing command: {str(e)}",
                data={"error": str(e), "traceback": tb_str},
                message_hash="error"
            )
            ctx.put_stream(json.dumps(error_response.model_dump()))


def main() -> None:
    """Start the worker service"""
    try:
        logger.info("Starting worker service")
        worker = hatchet.worker(
            "communication-worker",
            workflows=[worker_workflow],
        )
        logger.info("Worker initialized, starting...")
        worker.start()
    except Exception as e:
        import traceback
        tb_str = traceback.format_exc()
        logger.error(f"Failed to start worker: {str(e)}\n{tb_str}")
        raise


if __name__ == "__main__":
    main()
