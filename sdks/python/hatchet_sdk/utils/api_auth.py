def create_authorization_header(token: str) -> tuple[tuple[str, str]]:
    return (("authorization", "bearer " + token),)
