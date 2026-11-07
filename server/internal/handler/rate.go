package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/shinsaber/colec/internal/middleware"
	"github.com/shinsaber/colec/internal/model"
)

// Hourly limits per account of the social writes (I-1): enough for an honest
// use, a brake on a flood. Admins are not limited, as for the uploads.
const (
	rateWindow       = time.Hour
	rateReviews      = 60
	rateReports      = 20
	rateReactions    = 300
	rateSweepEntries = 10000
)

// hourlyLimit counts the writes of each (account, action) per hour.
type hourlyLimit struct {
	now func() time.Time

	mu      sync.Mutex
	windows map[string]*rateWindowCount
}

type rateWindowCount struct {
	start time.Time
	n     int
}

func (l *hourlyLimit) clock() time.Time {
	if l.now != nil {
		return l.now()
	}
	return time.Now()
}

// allow counts one write of user for action, or tells how long to wait.
func (l *hourlyLimit) allow(user uuid.UUID, action string, max int) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.clock()
	if l.windows == nil || len(l.windows) > rateSweepEntries {
		for k, w := range l.windows {
			if now.Sub(w.start) >= rateWindow {
				delete(l.windows, k)
			}
		}
		if l.windows == nil {
			l.windows = map[string]*rateWindowCount{}
		}
	}
	key := action + ":" + user.String()
	w := l.windows[key]
	if w == nil || now.Sub(w.start) >= rateWindow {
		w = &rateWindowCount{start: now}
		l.windows[key] = w
	}
	if w.n >= max {
		return false, rateWindow - now.Sub(w.start)
	}
	w.n++
	return true, 0
}

// limited answers 429 rate_limited when the account exceeded its hourly limit.
func (h *Handler) limited(w http.ResponseWriter, r *http.Request, action string, max int) bool {
	if middleware.GetRole(r.Context()) == model.RoleAdmin {
		return false
	}
	ok, retry := h.rates.allow(middleware.GetUserID(r.Context()), action, max)
	if ok {
		return false
	}
	seconds := int(retry.Seconds()) + 1
	w.Header().Set("Retry-After", strconv.Itoa(seconds))
	respondError(w, http.StatusTooManyRequests, "rate_limited",
		fmt.Sprintf("too many %s (%d per hour), retry in %d s", action, max, seconds))
	return true
}
