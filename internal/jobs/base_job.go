package jobs

import (
	"sync"
	"time"
)

type BaseJob struct {
	mu          sync.Mutex
	CompanyID   string
	Result      string
	Err         error
	Done        chan struct{}
	CreatedAt   time.Time
	CompletedAt time.Time
}
