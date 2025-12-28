package jobs

import "time"

type BaseJob struct {
	CompanyID   string
	Result      string
	Err         error
	Done        chan struct{}
	CreatedAt   time.Time
	CompletedAt time.Time
}
