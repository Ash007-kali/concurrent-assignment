package jobs

import (
	"assignment-concurrent-system/internal/model"
	"context"
	"errors"
	"sync"
	"time"
)

type BaseJobManager struct {
	mu   sync.Mutex
	Jobs map[string]*BaseJob
	TTL  time.Duration
}

func (m *BaseJobManager) GetOrCreate(companyID string) *BaseJob {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check existing
	if job, ok := m.Jobs[companyID]; ok {

		// Lazy TTL eviction
		if !job.CompletedAt.IsZero() &&
			time.Since(job.CompletedAt) > m.TTL {
			delete(m.Jobs, companyID)
		} else {
			return job
		}
	}

	// Create new job
	job := &BaseJob{
		CompanyID: companyID,
		Done:      make(chan struct{}),
		CreatedAt: time.Now(),
	}
	m.Jobs[companyID] = job

	go m.runBaseJob(job)

	return job
}

func (m *BaseJobManager) runBaseJob(job *BaseJob) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// simulate heavy computation
	select {
	case <-time.After(20 * time.Second):
		job.Result = "base-data-for-" + job.CompanyID
	case <-ctx.Done():
		job.Err = ctx.Err()
	}

	job.CompletedAt = time.Now()
	close(job.Done)
}

type APIJobManager struct {
	mu      sync.Mutex
	Jobs    map[string]*APIJob
	BaseMgr *BaseJobManager
}

// utility function to create the key
func apiJobKey(companyID string, api model.APIType) string {
	return companyID + ":" + string(api)
}

func (m *APIJobManager) GetOrCreate(companyID string, api model.APIType) *APIJob {
	key := apiJobKey(companyID, api)

	m.mu.Lock()
	if job, ok := m.Jobs[key]; ok {
		m.mu.Unlock()
		return job
	}

	job := &APIJob{
		CompanyID: companyID,
		APIType:   api,
		Done:      make(chan struct{}),
	}
	m.Jobs[key] = job
	m.mu.Unlock()

	go m.runAPIJob(job)

	return job
}

func (m *APIJobManager) runAPIJob(job *APIJob) {
	defer func() {
		m.mu.Lock()
		delete(m.Jobs, apiJobKey(job.CompanyID, job.APIType))
		m.mu.Unlock()
	}()

	// wait for base job
	baseJob := m.BaseMgr.GetOrCreate(job.CompanyID)

	select {
	case <-baseJob.Done:
		if baseJob.Err != nil {
			job.Err = baseJob.Err
			close(job.Done)
			return
		}
	case <-time.After(30 * time.Second):
		job.Err = errors.New("timeout waiting for base job")
		close(job.Done)
		return
	}

	// simulate API-specific work
	time.Sleep(20 * time.Second)
	job.Result = string(job.APIType) + "-result-using-" + baseJob.Result

	close(job.Done)
}
