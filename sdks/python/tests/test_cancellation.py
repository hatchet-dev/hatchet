"""Unit tests for CancellationToken and cancellation utilities."""

import asyncio
import threading
import time

import pytest

from hatchet_sdk.cancellation import CancellationToken
from hatchet_sdk.exceptions import CancellationReason, CancelledError
from hatchet_sdk.runnables.contextvars import ctx_cancellation_token
from hatchet_sdk.utils.cancellation import await_with_cancellation


class TestCancellationToken:
    """Tests for the CancellationToken class."""

    def test_initial_state(self) -> None:
        """Token should start in non-cancelled state."""
        token = CancellationToken()
        assert token.is_cancelled is False

    def test_cancel_sets_flag(self) -> None:
        """cancel() should set is_cancelled to True."""
        token = CancellationToken()
        token.cancel()
        assert token.is_cancelled is True

    def test_cancel_sets_reason(self) -> None:
        """cancel() should set the reason."""
        token = CancellationToken()
        token.cancel(CancellationReason.USER_REQUESTED)
        assert token.reason == CancellationReason.USER_REQUESTED

    def test_cancel_default_reason_is_unknown(self) -> None:
        """cancel() without explicit reason should default to UNKNOWN."""
        token = CancellationToken()
        token.cancel()
        assert token.reason == CancellationReason.UNKNOWN

    def test_reason_is_none_before_cancel(self) -> None:
        """reason should be None before cancellation."""
        token = CancellationToken()
        assert token.reason is None

    def test_cancel_idempotent(self) -> None:
        """Multiple calls to cancel() should be safe."""
        token = CancellationToken()
        token.cancel()
        token.cancel()  # Should not raise
        assert token.is_cancelled is True

    def test_cancel_idempotent_preserves_reason(self) -> None:
        """Multiple calls to cancel() should preserve the original reason."""
        token = CancellationToken()
        token.cancel(CancellationReason.USER_REQUESTED)
        token.cancel(CancellationReason.TIMEOUT)  # Second call should be ignored
        assert token.reason == CancellationReason.USER_REQUESTED

    def test_sync_wait_returns_true_when_cancelled(self) -> None:
        """wait() should return True immediately if already cancelled."""
        token = CancellationToken()
        token.cancel()
        result = token.wait(timeout=0.1)
        assert result is True

    def test_sync_wait_timeout_returns_false(self) -> None:
        """wait() should return False when timeout expires without cancellation."""
        token = CancellationToken()
        start = time.monotonic()
        result = token.wait(timeout=0.1)
        elapsed = time.monotonic() - start
        assert result is False
        assert elapsed >= 0.1

    def test_sync_wait_interrupted_by_cancel(self) -> None:
        """wait() should return True when cancelled during wait."""
        token = CancellationToken()

        def cancel_after_delay() -> None:
            time.sleep(0.1)
            token.cancel()

        thread = threading.Thread(target=cancel_after_delay)
        thread.start()

        start = time.monotonic()
        result = token.wait(timeout=1.0)
        elapsed = time.monotonic() - start

        thread.join()

        assert result is True
        assert elapsed < 0.5  # Should be much faster than timeout

    @pytest.mark.asyncio
    async def test_aio_wait_returns_when_cancelled(self) -> None:
        """aio_wait() should return when cancelled."""
        token = CancellationToken()

        async def cancel_after_delay() -> None:
            await asyncio.sleep(0.1)
            token.cancel()

        asyncio.create_task(cancel_after_delay())

        start = time.monotonic()
        await token.aio_wait()
        elapsed = time.monotonic() - start

        assert elapsed < 0.5  # Should be fast

    def test_register_child(self) -> None:
        """register_child() should add run IDs to the list."""
        token = CancellationToken()
        token.register_child("run-1")
        token.register_child("run-2")

        children = token.get_child_run_ids()
        assert children == ["run-1", "run-2"]

    def test_get_child_run_ids_returns_copy(self) -> None:
        """get_child_run_ids() should return a copy, not the internal list."""
        token = CancellationToken()
        token.register_child("run-1")

        children = token.get_child_run_ids()
        children.append("run-2")  # Modify the copy

        assert token.get_child_run_ids() == ["run-1"]  # Original unchanged

    def test_callback_invoked_on_cancel(self) -> None:
        """Callbacks should be invoked when cancel() is called."""
        token = CancellationToken()
        called = []

        def callback() -> None:
            called.append(True)

        token.add_callback(callback)
        token.cancel()

        assert called == [True]

    def test_callback_invoked_immediately_if_already_cancelled(self) -> None:
        """Callbacks added after cancellation should be invoked immediately."""
        token = CancellationToken()
        token.cancel()

        called = []

        def callback() -> None:
            called.append(True)

        token.add_callback(callback)

        assert called == [True]

    def test_multiple_callbacks(self) -> None:
        """Multiple callbacks should all be invoked."""
        token = CancellationToken()
        results: list[int] = []

        token.add_callback(lambda: results.append(1))
        token.add_callback(lambda: results.append(2))
        token.add_callback(lambda: results.append(3))

        token.cancel()

        assert results == [1, 2, 3]

    def test_repr(self) -> None:
        """__repr__ should provide useful debugging info."""
        token = CancellationToken()
        token.register_child("run-1")

        repr_str = repr(token)
        assert "cancelled=False" in repr_str
        assert "children=1" in repr_str


class TestAwaitWithCancellation:
    """Tests for the await_with_cancellation utility."""

    @pytest.mark.asyncio
    async def test_no_token_awaits_directly(self) -> None:
        """Without a token, coroutine should be awaited directly."""

        async def simple_coro() -> str:
            return "result"

        result = await await_with_cancellation(simple_coro(), None)
        assert result == "result"

    @pytest.mark.asyncio
    async def test_token_not_cancelled_returns_result(self) -> None:
        """With a non-cancelled token, should return coroutine result."""
        token = CancellationToken()

        async def simple_coro() -> str:
            await asyncio.sleep(0.01)
            return "result"

        result = await await_with_cancellation(simple_coro(), token)
        assert result == "result"

    @pytest.mark.asyncio
    async def test_already_cancelled_raises_immediately(self) -> None:
        """With an already-cancelled token, should raise immediately."""
        token = CancellationToken()
        token.cancel()

        async def simple_coro() -> str:
            await asyncio.sleep(10)  # Would block if actually awaited
            return "result"

        with pytest.raises(asyncio.CancelledError):
            await await_with_cancellation(simple_coro(), token)

    @pytest.mark.asyncio
    async def test_cancellation_during_await_raises(self) -> None:
        """Should raise CancelledError when token is cancelled during await."""
        token = CancellationToken()

        async def slow_coro() -> str:
            await asyncio.sleep(10)
            return "result"

        async def cancel_after_delay() -> None:
            await asyncio.sleep(0.1)
            token.cancel()

        asyncio.create_task(cancel_after_delay())

        start = time.monotonic()
        with pytest.raises(asyncio.CancelledError):
            await await_with_cancellation(slow_coro(), token)
        elapsed = time.monotonic() - start

        assert elapsed < 0.5  # Should be cancelled quickly

    @pytest.mark.asyncio
    async def test_cancel_callback_invoked(self) -> None:
        """Cancel callback should be invoked on cancellation."""
        token = CancellationToken()
        callback_called = []

        async def cancel_callback() -> None:
            callback_called.append(True)

        async def slow_coro() -> str:
            await asyncio.sleep(10)
            return "result"

        async def cancel_after_delay() -> None:
            await asyncio.sleep(0.1)
            token.cancel()

        asyncio.create_task(cancel_after_delay())

        with pytest.raises(asyncio.CancelledError):
            await await_with_cancellation(
                slow_coro(), token, cancel_callback=cancel_callback
            )

        assert callback_called == [True]

    @pytest.mark.asyncio
    async def test_cancel_callback_not_invoked_on_success(self) -> None:
        """Cancel callback should NOT be invoked when coroutine completes normally."""
        token = CancellationToken()
        callback_called = []

        async def cancel_callback() -> None:
            callback_called.append(True)

        async def fast_coro() -> str:
            await asyncio.sleep(0.01)
            return "result"

        result = await await_with_cancellation(
            fast_coro(), token, cancel_callback=cancel_callback
        )

        assert result == "result"
        assert callback_called == []


class TestCancellationReason:
    """Tests for the CancellationReason enum."""

    def test_all_reasons_exist(self) -> None:
        """All expected cancellation reasons should exist."""
        assert CancellationReason.USER_REQUESTED.value == "user_requested"
        assert CancellationReason.TIMEOUT.value == "timeout"
        assert CancellationReason.PARENT_CANCELLED.value == "parent_cancelled"
        assert CancellationReason.WORKFLOW_CANCELLED.value == "workflow_cancelled"
        assert CancellationReason.UNKNOWN.value == "unknown"

    def test_reasons_are_strings(self) -> None:
        """Cancellation reason values should be strings."""
        for reason in CancellationReason:
            assert isinstance(reason.value, str)


class TestCancelledErrorException:
    """Tests for the CancelledError exception."""

    def test_cancelled_error_is_base_exception(self) -> None:
        """CancelledError should be a BaseException (not Exception)."""
        err = CancelledError("test message")
        assert isinstance(err, BaseException)
        assert not isinstance(
            err, Exception
        )  # Should NOT be caught by except Exception
        assert str(err) == "test message"

    def test_cancelled_error_not_caught_by_except_exception(self) -> None:
        """CancelledError should NOT be caught by except Exception."""
        caught_by_exception = False
        caught_by_cancelled_error = False

        try:
            raise CancelledError("test")
        except Exception:
            caught_by_exception = True
        except CancelledError:
            caught_by_cancelled_error = True

        assert not caught_by_exception
        assert caught_by_cancelled_error

    def test_cancelled_error_with_reason(self) -> None:
        """CancelledError should accept and store a reason."""
        err = CancelledError("test message", reason=CancellationReason.TIMEOUT)
        assert err.reason == CancellationReason.TIMEOUT

    def test_cancelled_error_reason_defaults_to_none(self) -> None:
        """CancelledError reason should default to None."""
        err = CancelledError("test message")
        assert err.reason is None

    def test_cancelled_error_message_property(self) -> None:
        """CancelledError should have a message property."""
        err = CancelledError("test message")
        assert err.message == "test message"

    def test_cancelled_error_default_message(self) -> None:
        """CancelledError should have a default message."""
        err = CancelledError()
        assert err.message == "Operation cancelled"

    def test_can_be_raised_and_caught(self) -> None:
        """CancelledError should be raisable and catchable."""
        with pytest.raises(CancelledError) as exc_info:
            raise CancelledError("Operation cancelled")

        assert "Operation cancelled" in str(exc_info.value)

    def test_can_be_raised_with_reason(self) -> None:
        """CancelledError should be raisable with a reason."""
        with pytest.raises(CancelledError) as exc_info:
            raise CancelledError(
                "Parent was cancelled", reason=CancellationReason.PARENT_CANCELLED
            )

        assert exc_info.value.reason == CancellationReason.PARENT_CANCELLED


class TestContextVarPropagation:
    """Tests for context variable propagation of cancellation token."""

    def test_context_var_default_is_none(self) -> None:
        """ctx_cancellation_token should default to None."""
        assert ctx_cancellation_token.get() is None

    def test_context_var_can_be_set_and_retrieved(self) -> None:
        """ctx_cancellation_token should be settable and retrievable."""
        token = CancellationToken()
        ctx_cancellation_token.set(token)
        try:
            assert ctx_cancellation_token.get() is token
        finally:
            ctx_cancellation_token.set(None)

    @pytest.mark.asyncio
    async def test_context_var_propagates_in_async(self) -> None:
        """ctx_cancellation_token should propagate in async context."""
        token = CancellationToken()
        ctx_cancellation_token.set(token)

        async def check_token() -> CancellationToken | None:
            return ctx_cancellation_token.get()

        try:
            retrieved = await check_token()
            assert retrieved is token
        finally:
            ctx_cancellation_token.set(None)
