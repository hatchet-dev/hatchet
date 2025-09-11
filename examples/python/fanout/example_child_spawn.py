# > Child spawn
from examples.fanout.worker import ChildInput, child_wf

# 👀 example: run this inside of a parent task to spawn a child
child_wf.run(
    ChildInput(a="b"),
)

# > Error handling
try:
    child_wf.run(
        ChildInput(a="b"),
    )
except Exception as e:
    print(f"Child workflow failed: {e}")
