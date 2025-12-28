package jobs

import (
	"assignment-concurrent-system/internal/model"
	"context"
	"sync"
	"testing"
	"time"
)

func TestAPIJob_Deduplication(t *testing.T) {
	baseMgr := NewBaseJobManager(1 * time.Minute)
	apiMgr := NewAPIJobManager(baseMgr, 5*time.Minute)

	const companyID = "123"
	const callers = 5

	var wg sync.WaitGroup
	wg.Add(callers)

	jobs := make([]*APIJob, callers)
	ctx := context.Background()

	for i := 0; i < callers; i++ {
		go func(idx int) {
			defer wg.Done()
			jobs[idx] = apiMgr.GetOrCreate(ctx, companyID, model.APIFinancials)
		}(i)
	}

	wg.Wait()

	// All callers must get the same APIJob instance
	first := jobs[0]
	for i := 1; i < callers; i++ {
		if jobs[i] != first {
			t.Fatalf("expected same APIJob instance, got different ones")
		}
	}

	// Wait for completion
	<-first.Done

	first.mu.Lock()
	defer first.mu.Unlock()

	if first.Err != nil {
		t.Fatalf("job failed: %v", first.Err)
	}

	if first.Result == "" {
		t.Fatalf("expected result, got empty")
	}
}

func TestBaseJob_SharedAcrossAPIs(t *testing.T) {
	baseMgr := NewBaseJobManager(1 * time.Minute)
	apiMgr := NewAPIJobManager(baseMgr, 5*time.Minute)

	ctx := context.Background()
	companyID := "42"

	finJob := apiMgr.GetOrCreate(ctx, companyID, model.APIFinancials)
	salesJob := apiMgr.GetOrCreate(ctx, companyID, model.APISales)

	<-finJob.Done
	<-salesJob.Done

	finJob.mu.Lock()
	salesJob.mu.Lock()

	if finJob.Err != nil || salesJob.Err != nil {
		finJob.mu.Unlock()
		salesJob.mu.Unlock()
		t.Fatalf("one of the API jobs failed")
	}

	finJob.mu.Unlock()
	salesJob.mu.Unlock()

	// Safely inspect base manager
	baseMgr.mu.Lock()
	defer baseMgr.mu.Unlock()

	if len(baseMgr.jobs) != 1 {
		t.Fatalf("expected 1 base job, got %d", len(baseMgr.jobs))
	}

	baseJob := baseMgr.jobs[companyID]
	if baseJob == nil {
		t.Fatalf("base job not found")
	}

	baseJob.mu.Lock()
	defer baseJob.mu.Unlock()

	if baseJob.Result == "" {
		t.Fatalf("base job result is empty")
	}
}
