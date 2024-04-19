import base64
import os

from dotenv import load_dotenv

from hatchet_sdk import Context, Hatchet

load_dotenv()

hatchet = Hatchet(debug=True)


@hatchet.workflow(on_events=["man:create"])
class ManualTriggerWorkflow:
    @hatchet.step()
    def step1(self, context: Context):
        res = context.playground("res", "HELLO")

        # Get the directory of the current script
        script_dir = os.path.dirname(os.path.abspath(__file__))

        # Construct the path to the image file relative to the script's directory
        image_path = os.path.join(script_dir, "image.jpeg")

        # Load the image file
        with open(image_path, "rb") as image_file:
            image_data = image_file.read()

        print(len(image_data))

        # Encode the image data as base64
        base64_image = base64.b64encode(image_data).decode("utf-8")

        # Stream the base64-encoded image data
        context.put_stream(base64_image)

        context.sleep(3)
        print("executed step1")
        return {"step1": "data1 " + res}

    @hatchet.step(parents=["step1"], timeout="4s")
    def step2(self, context):
        print("started step2")
        context.sleep(1)
        print("finished step2")
        return {"step2": "data2"}


workflow = ManualTriggerWorkflow()
worker = hatchet.worker("manual-worker", max_runs=4)
worker.register_workflow(workflow)

worker.start()
