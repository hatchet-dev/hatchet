from hatchet_sdk import Hatchet

hatchet = Hatchet(debug=True)

# for i in range(10):
hatchet.event.push("dag:create", {"test": "test"})
