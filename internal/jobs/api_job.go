package jobs

import "assignment-concurrent-system/internal/model"

type APIJob struct {
	CompanyID string
	APIType   model.APIType
	Result    string
	Err       error
	Done      chan struct{}
}
