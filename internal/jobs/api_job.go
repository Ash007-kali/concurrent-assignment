package jobs

import (
	"assignment-concurrent-system/internal/model"
	"sync"
	"time"
)

type APIJob struct {
	mu          sync.Mutex
	CompanyID   string
	APIType     model.APIType
	Result      string
	Err         error
	Done        chan struct{}
	CreatedAt   time.Time
	CompletedAt time.Time
}
