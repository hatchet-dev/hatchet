from types import TracebackType

from fastapi import BackgroundTasks, FastAPI
from pydantic import BaseModel

from hatchet_sdk import Context, Hatchet


class Session:
    ## simulate async db session
    async def __aenter__(self) -> "Session":
        return self

    async def __aexit__(
        self,
        type_: type[BaseException] | None,
        value: BaseException | None,
        traceback: TracebackType | None,
    ) -> None:
        pass


class User:
    def __init__(self, id: int, email: str):
        self.id = id
        self.email = email


async def get_user(db: Session, user_id: int) -> User:
    return User(user_id, "test@example.com")


async def create_user(db: Session) -> User:
    return User(1, "test@example.com")


async def send_welcome_email(email: str) -> None:
    print(f"Sending welcome email to {email}")


app = FastAPI()
hatchet = Hatchet()


# > FastAPI Background Tasks
async def send_welcome_email_task_bg(user_id: int) -> None:
    async with Session() as db:
        user = await get_user(db, user_id)

        await send_welcome_email(user.email)


@app.post("/user")
async def post__create_user__background_tasks(
    background_tasks: BackgroundTasks,
) -> User:
    async with Session() as db:
        user = await create_user(db)

        background_tasks.add_task(send_welcome_email_task_bg, user.id)

        return user




# > Hatchet Task
class WelcomeEmailInput(BaseModel):
    user_id: int


@hatchet.task(input_validator=WelcomeEmailInput)
async def send_welcome_email_task_hatchet(
    input: WelcomeEmailInput, _ctx: Context
) -> None:
    async with Session() as db:
        user = await get_user(db, input.user_id)

        await send_welcome_email(user.email)


@app.post("/user")
async def post__create_user__hatchet() -> User:
    async with Session() as db:
        user = await create_user(db)

        await send_welcome_email_task_hatchet.aio_run_no_wait(
            WelcomeEmailInput(user_id=user.id)
        )

        return user


