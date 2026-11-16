import shutil


def rm_rf(path: str) -> None:
    shutil.rmtree(path, ignore_errors=True)
