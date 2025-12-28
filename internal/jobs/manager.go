package jobs

import (
	"assignment-concurrent-system/internal/model"
	"context"
	"errors"
	"log"
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

	if job, ok := m.Jobs[companyID]; ok {
		if !job.CompletedAt.IsZero() &&
			time.Since(job.CompletedAt) > m.TTL {

			log.Printf("[BaseJob] TTL expired, evicting company=%s", companyID)
			delete(m.Jobs, companyID)

		} else {
			log.Printf("[BaseJob] cache hit company=%s", companyID)
			return job
		}
	}

	log.Printf("[BaseJob] creating new job company=%s", companyID)

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
	log.Printf("[BaseJob] started company=%s", job.CompanyID)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	select {
	case <-time.After(20 * time.Second):
		job.Result = "base-data-for-" + job.CompanyID
		log.Printf("[BaseJob] completed company=%s", job.CompanyID)

	case <-ctx.Done():
		job.Err = ctx.Err()
		log.Printf("[BaseJob] timeout company=%s err=%v", job.CompanyID, job.Err)
	}

	job.CompletedAt = time.Now()
	close(job.Done)
}

type APIJobManager struct {
	mu      sync.Mutex
	Jobs    map[string]*APIJob
	BaseMgr *BaseJobManager
	TTL     time.Duration
}

// utility function to create the key
func apiJobKey(companyID string, api model.APIType) string {
	return companyID + ":" + string(api)
}

func (m *APIJobManager) GetOrCreate(companyID string, api model.APIType) *APIJob {
	key := apiJobKey(companyID, api)

	m.mu.Lock()
	if job, ok := m.Jobs[key]; ok {

		// Lazy TTL eviction
		if !job.CompletedAt.IsZero() &&
			time.Since(job.CompletedAt) > m.TTL {

			log.Printf("[APIJob] TTL expired, evicting api=%s company=%s", api, companyID)
			delete(m.Jobs, key)

		} else {
			log.Printf("[APIJob] cache hit api=%s company=%s", api, companyID)
			return job
		}
	}

	log.Printf("[APIJob] creating api=%s company=%s", api, companyID)

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
	log.Printf("[APIJob] started api=%s company=%s", job.APIType, job.CompanyID)

	defer func() {
		m.mu.Lock()
		delete(m.Jobs, apiJobKey(job.CompanyID, job.APIType))
		m.mu.Unlock()

		log.Printf("[APIJob] cleaned up api=%s company=%s", job.APIType, job.CompanyID)
	}()

	baseJob := m.BaseMgr.GetOrCreate(job.CompanyID)

	log.Printf("[APIJob] waiting for base job api=%s company=%s",
		job.APIType, job.CompanyID)

	select {
	case <-baseJob.Done:
		if baseJob.Err != nil {
			job.Err = baseJob.Err
			log.Printf("[APIJob] base job failed api=%s company=%s err=%v",
				job.APIType, job.CompanyID, job.Err)
			close(job.Done)
			return
		}

	case <-time.After(30 * time.Second):
		job.Err = errors.New("timeout waiting for base job")
		log.Printf("[APIJob] timeout waiting base job api=%s company=%s",
			job.APIType, job.CompanyID)
		close(job.Done)
		return
	}

	log.Printf("[APIJob] base job ready api=%s company=%s",
		job.APIType, job.CompanyID)

	time.Sleep(20 * time.Second)
	job.Result = string(job.APIType) + "-result-using-" + baseJob.Result

	log.Printf("[APIJob] completed api=%s company=%s",
		job.APIType, job.CompanyID)

	close(job.Done)
}
