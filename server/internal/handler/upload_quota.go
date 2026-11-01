package handler

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

// uploadQuotaWindow is the fixed window of the per-user upload quota.
const uploadQuotaWindow = time.Hour

// uploadQuota bounds what one account may send per hour (files and bytes), so
// that a member cannot fill the uploads volume (audit S-004). In memory, per
// instance, like middleware.RateLimiter; the zero value is ready to use.
type uploadQuota struct {
	now func() time.Time

	mu        sync.Mutex
	users     map[uuid.UUID]*uploadWindow
	lastSweep time.Time
}

type uploadWindow struct {
	start time.Time
	files int
	bytes int64
}

// allow counts one upload for user and reports whether the quota (maxFiles
// files and maxBytes bytes per window) still has room. When it does not,
// retryAfter is the time left in the current window. The size of a file is
// only known once written: callers count the file first with allow, then
// account for its bytes with add.
func (q *uploadQuota) allow(user uuid.UUID, maxFiles int, maxBytes int64) (allowed bool, retryAfter time.Duration) {
	q.mu.Lock()
	defer q.mu.Unlock()
	w := q.window(user)
	if w.files >= maxFiles || w.bytes >= maxBytes {
		return false, uploadQuotaWindow - q.clock().Sub(w.start)
	}
	w.files++
	return true, 0
}

// add accounts for the bytes of a file accepted by allow.
func (q *uploadQuota) add(user uuid.UUID, n int64) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.window(user).bytes += n
}

// window returns the current window of user, reset when the previous one has
// expired; stale entries are swept once per window. Caller holds q.mu.
func (q *uploadQuota) window(user uuid.UUID) *uploadWindow {
	now := q.clock()
	if q.users == nil {
		q.users = make(map[uuid.UUID]*uploadWindow)
	}
	if now.Sub(q.lastSweep) >= uploadQuotaWindow {
		for k, w := range q.users {
			if now.Sub(w.start) >= uploadQuotaWindow {
				delete(q.users, k)
			}
		}
		q.lastSweep = now
	}
	w := q.users[user]
	if w == nil || now.Sub(w.start) >= uploadQuotaWindow {
		w = &uploadWindow{start: now}
		q.users[user] = w
	}
	return w
}

func (q *uploadQuota) clock() time.Time {
	if q.now != nil {
		return q.now()
	}
	return time.Now()
}
