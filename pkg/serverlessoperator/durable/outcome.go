package durable

import "encoding/json"

// phase is what the engine has committed to about the invocation so far. It only moves
// forward from phaseRunning, and the first transition wins: an eviction the engine has
// acknowledged and an error the engine has reported are both final statements about the
// invocation, and the relay reports whichever came first regardless of how the socket ends.
type phase int32

const (
	// phaseRunning: the engine has neither acknowledged an eviction nor reported an error.
	phaseRunning phase = iota
	// phaseEvictionAcked: an eviction_ack was forwarded to the endpoint. The engine has
	// recorded the eviction and re-invokes the task itself, so no terminal event is due.
	phaseEvictionAcked
	// phaseEngineErrored: an error frame was forwarded to the endpoint. The engine's error
	// (non-determinism on replay and the like) is permanent.
	phaseEngineErrored
)

// exitKind is how the relay ended.
type exitKind int

const (
	// exitDoneOutput: the endpoint sent done with an output.
	exitDoneOutput exitKind = iota + 1
	// exitDoneError: the endpoint sent done with an error and a retry decision.
	exitDoneError
	// exitDoneEvicted: the endpoint sent done with status evicted.
	exitDoneEvicted
	// exitClosedWithoutDone: the socket ended without a done frame (close frame, transport
	// error, read timeout, failed ping or write).
	exitClosedWithoutDone
	// exitUnresponsive: missedPongLimit pings went unanswered.
	exitUnresponsive
	// exitBackpressure: the endpoint fell behind the send queue's frame or byte budget.
	exitBackpressure
	// exitLinkFailure: the engine side of the relay failed.
	exitLinkFailure
	// exitProtocolViolation: the endpoint sent something the contract forbids.
	exitProtocolViolation
	// exitServerEvict: the engine superseded the invocation with a server_evict notice.
	exitServerEvict
	// exitTimeout: the endpoint's request timeout elapsed.
	exitTimeout
	// exitCancelled: the engine cancelled the task.
	exitCancelled
	// exitShutdown: the operator is stopping.
	exitShutdown
)

// exit describes one way the relay ended, before the phase is applied to it. closeCode is
// what the relay sends in its close frame (0 for none); msg and retry are the exit's own
// failure message and retry decision, used when the phase does not override them.
type exit struct {
	output    json.RawMessage
	msg       string
	kind      exitKind
	closeCode int
	retry     bool
}

// unacknowledgedEvictionMsg is reported when an endpoint claims an eviction the engine never
// acknowledged. The failure is retryable so the task gets a terminal state instead of
// waiting for an eviction the engine has no record of.
const unacknowledgedEvictionMsg = "endpoint reported an eviction the engine never acknowledged"

// resolve is the relay's outcome table: the phase the engine put the invocation in, crossed
// with the exit. engineErr is the message of the engine's error frame when ph is
// phaseEngineErrored.
//
//	exit                | running                          | eviction acked           | engine errored
//	--------------------+----------------------------------+--------------------------+-------------------------
//	done output         | completed                        | evicted (endpoint)       | failed, engine error
//	done error          | failed, done's error and retry   | failed, done's error     | failed, engine error
//	done evicted        | failed retryable, close 4005     | evicted (endpoint)       | failed, engine error
//	closed without done | failed retryable                 | evicted (endpoint)       | failed, engine error
//	unresponsive        | failed retryable, close 4008     | evicted (endpoint)       | failed, engine error
//	backpressure        | failed retryable, close 1013     | evicted (endpoint)       | failed, engine error
//	link failure        | failed retryable, close 1011     | evicted (endpoint)       | failed, engine error
//	protocol violation  | failed, exit's close code        | evicted (endpoint)       | failed, engine error
//	timeout             | failed retryable, close 4007     | evicted (endpoint)       | failed, engine error
//	server evict        | evicted (server), close 4001     | evicted (server)         | evicted (server)
//	cancelled           | cancelled, close 4003            | cancelled                | cancelled
//	shutdown            | shutdown, close 4002             | evicted (endpoint)       | failed, engine error
//
// Every failure under an engine error is non-retryable. A server_evict is the engine's later
// word on the invocation and wins over both earlier phases; an engine cancel produces no
// event in any phase because the core already sent CANCELLED. The close code is the exit's
// in every cell except done evicted while running, which is a protocol violation.
func resolve(ph phase, engineErr string, e exit) Outcome {
	switch e.kind {
	case exitServerEvict:
		return evicted(e.closeCode, EvictionSourceServer)
	case exitCancelled:
		return Outcome{Kind: KindCancelled, CloseCode: e.closeCode}
	}

	switch ph {
	case phaseEngineErrored:
		if engineErr == "" {
			engineErr = "engine reported an error for the invocation"
		}

		return failed(e.closeCode, engineErr, false)
	case phaseEvictionAcked:
		if e.kind == exitDoneError {
			return failed(e.closeCode, e.msg, e.retry)
		}

		return evicted(e.closeCode, EvictionSourceEndpoint)
	}

	switch e.kind {
	case exitDoneOutput:
		return completed(e.output)
	case exitDoneError:
		return failed(e.closeCode, e.msg, e.retry)
	case exitDoneEvicted:
		return failed(CloseForbiddenMessage, unacknowledgedEvictionMsg, true)
	case exitProtocolViolation:
		return failed(e.closeCode, e.msg, false)
	case exitShutdown:
		return Outcome{Kind: KindShutdown, CloseCode: e.closeCode}
	default:
		// closed without done, unresponsive, backpressure, link failure, timeout: the crash
		// rule, a retryable failure.
		return failed(e.closeCode, e.msg, true)
	}
}
