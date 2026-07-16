from hatchet_sdk import Context, EmptyModel, Hatchet

hatchet = Hatchet()

# > Slot cost


@hatchet.task(slot_cost=5)
def omega(input: EmptyModel, ctx: Context) -> None:
    print("heavy work")


@hatchet.task(slot_cost=1)
def weenie(input: EmptyModel, ctx: Context) -> None:
    print("light work")




def main() -> None:
    worker = hatchet.worker("slot-cost-worker", workflows=[omega, weenie])
    worker.start()


if __name__ == "__main__":
    main()
