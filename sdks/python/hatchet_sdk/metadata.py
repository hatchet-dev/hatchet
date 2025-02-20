def get_metadata(token: str) -> list[tuple[str, str]]:
    return [("authorization", "bearer " + token)]
