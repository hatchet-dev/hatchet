package fairpool

import (
	"context"
	"time"

	"github.com/google/uuid"
	promclient "github.com/prometheus/client_golang/prometheus"

	"github.com/hatchet-dev/hatchet/pkg/integrations/metrics/prometheus"
)

const (
	// budgetWindow bounds a budget both ways: the balance stays within plus or minus this much
	// refill at the current rate, and AwaitBudget waits at most this long in total.
	budgetWindow = 10 * time.Second

	// accountIdleTimeout is how long a key may go without a checkout before its account, and so its
	// held-seconds series, is dropped. A dropped budget is full again, as it would be by then anyway.
	accountIdleTimeout = 10 * time.Minute

	accountSweepInterval = time.Minute
)

// account is a key's held time and budget. It is guarded by the gate's mutex.
type account struct {
	inFlight    int64
	heldSeconds promclient.Counter
	lastUsed    time.Time

	// balance is the budget's token count in held-seconds; filled is when it was last refilled.
	// A zero filled means the budget has never been used and is full.
	balance float64
	filled  time.Time
}

func (g *gate) accountLocked(key string, now time.Time) *account {
	acct, ok := g.accounts[key]
	if !ok {
		acct = &account{heldSeconds: prometheus.FairpoolHeldSeconds.WithLabelValues(g.poolName, key)}
		g.accounts[key] = acct
	}

	acct.lastUsed = now

	return acct
}

// sweepLocked drops the accounts of keys that have been idle for accountIdleTimeout, at most once
// per accountSweepInterval, so the held-seconds series stay bounded by recently active tenants.
func (g *gate) sweepLocked(now time.Time) {
	if now.Sub(g.lastSweep) < accountSweepInterval {
		return
	}

	g.lastSweep = now

	for key, acct := range g.accounts {
		if acct.inFlight == 0 && now.Sub(acct.lastUsed) >= accountIdleTimeout {
			delete(g.accounts, key)
			prometheus.FairpoolHeldSeconds.DeleteLabelValues(g.poolName, key)
		}
	}
}

// refill brings the balance up to now: it adds rate held-seconds per second since the last refill
// and clamps the balance to plus or minus budgetWindow of the current rate, so a lower rate never
// keeps more than budgetWindow of debt. Without a budget (rate 0) the bucket counts as full, so a
// budget that comes back starts full.
func (a *account) refill(rate float64, now time.Time) {
	if rate <= 0 {
		a.filled = time.Time{}
		return
	}

	limit := rate * budgetWindow.Seconds()

	if a.filled.IsZero() {
		a.balance = limit
	} else {
		a.balance += rate * now.Sub(a.filled).Seconds()
	}

	a.balance = min(limit, max(-limit, a.balance))
	a.filled = now
}

// debit takes held seconds from a budget of rate held-seconds per second.
func (a *account) debit(held, rate float64, now time.Time) {
	a.refill(rate, now)

	if rate > 0 {
		a.balance = max(-rate*budgetWindow.Seconds(), a.balance-held)
	}
}

// wait is how long until a budget of rate held-seconds per second is back to zero: at most
// budgetWindow, by the clamp in refill.
func (a *account) wait(rate float64, now time.Time) time.Duration {
	a.refill(rate, now)

	if rate <= 0 || a.balance >= 0 {
		return 0
	}

	return time.Duration(-a.balance / rate * float64(time.Second))
}

// budgetWait is how long tenantID should wait now for its budget. It does not create an account: a
// tenant without one has not checked out recently, so its budget is full.
func (g *gate) budgetWait(key string, tenantID uuid.UUID, now time.Time) time.Duration {
	rate := g.budgetFor(tenantID)
	if rate <= 0 {
		return 0
	}

	g.mu.Lock()
	defer g.mu.Unlock()

	acct, ok := g.accounts[key]
	if !ok {
		return 0
	}

	return acct.wait(rate, now)
}

// awaitBudget returns at once when tenantID's balance is not negative. Otherwise it sleeps until
// the balance should be back to zero and checks it again, because releases that landed meanwhile
// may have debited it, for at most budgetWindow in total, then returns nil whatever the balance.
// It returns ctx's error if ctx ends first. It holds no lock and no slot while it sleeps.
//
// It reserves nothing: waiters that wake together and find the balance back at zero all return,
// and their checkouts debit the bucket when they are released. There is no ordering between
// waiters. A nil gate (an Ungated pool) has no budget.
func (g *gate) awaitBudget(ctx context.Context, key string, tenantID uuid.UUID) error {
	if g == nil {
		return nil
	}

	start := time.Now()
	deadline := start.Add(budgetWindow)

	wait := g.budgetWait(key, tenantID, start)
	if wait <= 0 {
		return nil
	}

	defer func() {
		prometheus.FairpoolBudgetWait.WithLabelValues(g.poolName).Observe(time.Since(start).Seconds())
	}()

	timer := time.NewTimer(min(wait, budgetWindow))
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
		}

		now := time.Now()
		left := deadline.Sub(now)

		if left <= 0 {
			return nil
		}

		wait = g.budgetWait(key, tenantID, now)
		if wait <= 0 {
			return nil
		}

		timer.Reset(min(wait, left))
	}
}
