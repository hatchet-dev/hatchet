def get_metadata(token: str) -> tuple[tuple[str, str]]:
    return (("authorization", "bearer " + token),)
