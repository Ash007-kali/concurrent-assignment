package jobs

import (
	"assignment-concurrent-system/internal/model"
	"context"
	"errors"
	"log"
	"sync"
	"time"
)

//
// --------------------
// Base Job
// --------------------
//

type BaseJobManager struct {
	mu   sync.Mutex
	jobs map[string]*BaseJob
	ttl  time.Duration
}

func NewBaseJobManager(ttl time.Duration) *BaseJobManager {
	return &BaseJobManager{
		jobs: make(map[string]*BaseJob),
		ttl:  ttl,
	}
}

func (m *BaseJobManager) GetOrCreate(ctx context.Context, companyID string) *BaseJob {
	m.mu.Lock()
	defer m.mu.Unlock()

	if job, ok := m.jobs[companyID]; ok {
		job.mu.Lock()
		completed := !job.CompletedAt.IsZero()
		expired := completed && time.Since(job.CompletedAt) > m.ttl
		job.mu.Unlock()

		if !expired {
			log.Printf("[BaseJob] cache hit company=%s", companyID)
			return job
		}

		log.Printf("[BaseJob] TTL expired, evicting company=%s", companyID)
		delete(m.jobs, companyID)
	}

	log.Printf("[BaseJob] creating new job company=%s", companyID)

	job := &BaseJob{
		CompanyID: companyID,
		CreatedAt: time.Now(),
		Done:      make(chan struct{}),
	}

	m.jobs[companyID] = job
	go m.run(job, ctx)

	return job
}

func (m *BaseJobManager) run(job *BaseJob, parentCtx context.Context) {
	log.Printf("[BaseJob] started company=%s", job.CompanyID)

	defer func() {
		job.mu.Lock()
		job.CompletedAt = time.Now()
		job.mu.Unlock()
		close(job.Done)
	}()

	ctx, cancel := context.WithTimeout(parentCtx, 20*time.Second)
	defer cancel()

	select {
	case <-time.After(10 * time.Second):
		job.mu.Lock()
		job.Result = "base-data-for-" + job.CompanyID
		job.mu.Unlock()
		log.Printf("[BaseJob] completed company=%s", job.CompanyID)

	case <-ctx.Done():
		job.mu.Lock()
		job.Err = ctx.Err()
		job.mu.Unlock()
		log.Printf("[BaseJob] failed company=%s err=%v", job.CompanyID, ctx.Err())
	}
}

//
// --------------------
// API Job
// --------------------
//

type APIJobManager struct {
	mu      sync.Mutex
	jobs    map[string]*APIJob
	ttl     time.Duration
	baseMgr *BaseJobManager
}

func NewAPIJobManager(baseMgr *BaseJobManager, ttl time.Duration) *APIJobManager {
	return &APIJobManager{
		jobs:    make(map[string]*APIJob),
		ttl:     ttl,
		baseMgr: baseMgr,
	}
}

func apiJobKey(companyID string, api model.APIType) string {
	return companyID + ":" + string(api)
}

func (m *APIJobManager) GetOrCreate(ctx context.Context, companyID string, api model.APIType) *APIJob {
	key := apiJobKey(companyID, api)

	m.mu.Lock()
	defer m.mu.Unlock()

	if job, ok := m.jobs[key]; ok {
		job.mu.Lock()
		completed := !job.CompletedAt.IsZero()
		expired := completed && time.Since(job.CompletedAt) > m.ttl
		job.mu.Unlock()

		if !expired {
			log.Printf("[APIJob] cache hit api=%s company=%s", api, companyID)
			return job
		}

		log.Printf("[APIJob] TTL expired, evicting api=%s company=%s", api, companyID)
		delete(m.jobs, key)
	}

	log.Printf("[APIJob] creating api=%s company=%s", api, companyID)

	job := &APIJob{
		CompanyID: companyID,
		APIType:   api,
		Done:      make(chan struct{}),
	}

	m.jobs[key] = job
	go m.run(job, ctx)

	return job
}

func (m *APIJobManager) run(job *APIJob, parentCtx context.Context) {
	log.Printf("[APIJob] started api=%s company=%s", job.APIType, job.CompanyID)

	defer func() {
		job.mu.Lock()
		job.CompletedAt = time.Now()
		job.mu.Unlock()
		close(job.Done)
	}()

	ctx, cancel := context.WithTimeout(parentCtx, 30*time.Second)
	defer cancel()

	baseJob := m.baseMgr.GetOrCreate(ctx, job.CompanyID)

	select {
	case <-baseJob.Done:
		baseJob.mu.Lock()
		err := baseJob.Err
		result := baseJob.Result
		baseJob.mu.Unlock()

		if err != nil {
			job.mu.Lock()
			job.Err = err
			job.mu.Unlock()
			log.Printf("[APIJob] base job failed api=%s company=%s err=%v",
				job.APIType, job.CompanyID, err)
			return
		}

		time.Sleep(10 * time.Second)

		job.mu.Lock()
		job.Result = string(job.APIType) + "-result-using-" + result
		job.mu.Unlock()

		log.Printf("[APIJob] completed api=%s company=%s",
			job.APIType, job.CompanyID)

	case <-ctx.Done():
		job.mu.Lock()
		job.Err = errors.New("timeout waiting for base job")
		job.mu.Unlock()
		log.Printf("[APIJob] timeout api=%s company=%s",
			job.APIType, job.CompanyID)
	}
}
