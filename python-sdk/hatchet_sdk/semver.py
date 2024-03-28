def bump_minor_version(version: str) -> str:
    """
    Bumps the minor version of a semantic version string. NOTE this doesn't follow full semver,
    missing the build metadata and pre-release version.

    :param version: A semantic version string in the format major.minor.patch
    :return: A string with the minor version bumped and patch version reset to 0
    :raises ValueError: If the input is not a valid semantic version string
    """
    # if it starts with a v, remove it
    had_v = False
    if version.startswith("v"):
        version = version[1:]
        had_v = True

    parts = version.split(".")
    if len(parts) != 3:
        raise ValueError(f"Invalid semantic version: {version}")

    try:
        major, minor, _ = map(int, parts)
    except ValueError:
        raise ValueError(f"Invalid semantic version: {version}")

    new_minor = minor + 1
    new_version = f"{major}.{new_minor}.0"

    if had_v:
        new_version = "v" + new_version
    return new_version
