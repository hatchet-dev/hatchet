        
client = new_client()

# register a workflow
# client.admin.put_workflow(CreateWorkflowVersionOpts(
#     name="python-workflow",
#     description="This is a test workflow",
#     version="v0.4.0",
#     event_triggers=[
#         "user:create",
#     ],
#     jobs=[
#         CreateWorkflowJobOpts(
#             name="my-job",
#             timeout="60s",
#             steps=[
#                 CreateWorkflowStepOpts(
#                     readable_id="my-step",
#                     action="user:create",
#                     timeout="60s",
#                     inputs='{}'
#                 )
#             ]
#         )
#     ]
# ))

listener : ActionListenerImpl = client.dispatcher.get_action_listener(GetActionListenerRequest(
    worker_name="test-worker",
    services=["default"],
    actions=["user:create"],
))

async def process_actions():
    generator = listener.actions()

    for action in generator:
        print(action)
        # Process each action here
        # For example, you can access action attributes like action.tenant_id, action.worker_id, etc.
        pass  # Replace this with your actual processing code

# try:
#     for action in generator:
#         print(action)
#         # Process each action here
#         # For example, you can access action attributes like action.tenant_id, action.worker_id, etc.
#         pass  # Replace this with your actual processing code
# except Exception as e:
#     # Handle any exceptions that might occur
#     print(f"An error occurred: {e}")

loop = asyncio.new_event_loop()
asyncio.set_event_loop(loop)
# tasks = list()
# tasks.append(asyncio.create_task(process_actions()))

# print("sending event")

# event = {
#     "test": "test",
# }

# client.event.push(
#     "user:create",
#     event,
# )

# loop.run_until_complete(process_actions())  
# loop.close()

loop = asyncio.new_event_loop()
asyncio.set_event_loop(loop)

