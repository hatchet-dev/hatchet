from hatchet_sdk import Hatchet

hatchet = Hatchet(debug=True)


# > Step 03 Push Event
# Push an event to trigger the workflow. Use the same key as on_events.
hatchet.event.push(
    "order:created",
    {"message": "Order #1234", "source": "webhook"},
)
# !!
