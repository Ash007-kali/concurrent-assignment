package jobs

import (
	"assignment-concurrent-system/internal/model"
	"time"
)

type APIJob struct {
	CompanyID   string
	APIType     model.APIType
	Result      string
	Err         error
	Done        chan struct{}
	CreatedAt   time.Time
	CompletedAt time.Time
}
