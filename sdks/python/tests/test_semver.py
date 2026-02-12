from hatchet_sdk.deprecated.deprecation import parse_semver, semver_less_than


class TestParseSemver:
    def test_standard_version_with_v_prefix(self) -> None:
        assert parse_semver("v0.78.23") == (0, 78, 23)

    def test_version_without_v_prefix(self) -> None:
        assert parse_semver("1.2.3") == (1, 2, 3)

    def test_strips_prerelease_suffix(self) -> None:
        assert parse_semver("v0.1.0-alpha.0") == (0, 1, 0)
        assert parse_semver("v10.20.30-rc.1") == (10, 20, 30)

    def test_empty_string(self) -> None:
        assert parse_semver("") == (0, 0, 0)

    def test_malformed_input(self) -> None:
        assert parse_semver("v1.2") == (0, 0, 0)
        assert parse_semver("not-a-version") == (0, 0, 0)


class TestSemverLessThan:
    def test_less_than_patch(self) -> None:
        assert semver_less_than("v0.78.22", "v0.78.23") is True

    def test_equal(self) -> None:
        assert semver_less_than("v0.78.23", "v0.78.23") is False

    def test_greater_than_patch(self) -> None:
        assert semver_less_than("v0.78.24", "v0.78.23") is False

    def test_minor_comparison(self) -> None:
        assert semver_less_than("v0.77.99", "v0.78.0") is True
        assert semver_less_than("v0.79.0", "v0.78.99") is False

    def test_major_comparison(self) -> None:
        assert semver_less_than("v0.78.23", "v1.0.0") is True
        assert semver_less_than("v1.0.0", "v0.99.99") is False

    def test_prerelease(self) -> None:
        assert semver_less_than("v0.1.0-alpha.0", "v0.78.23") is True

    def test_empty_string_as_zero(self) -> None:
        assert semver_less_than("", "v0.78.23") is True
        assert semver_less_than("v0.78.23", "") is False
