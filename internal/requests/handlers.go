package requests

import (
	"assignment-concurrent-system/internal/jobs"
	"assignment-concurrent-system/internal/model"
	"net/http"
)

func handleAPIRequest(
	w http.ResponseWriter,
	r *http.Request,
	apiMgr *jobs.APIJobManager,
	apiType model.APIType,
) {
	companyID := r.URL.Query().Get("companyId")
	if companyID == "" {
		http.Error(w, "companyId is required", http.StatusBadRequest)
		return
	}

	job := apiMgr.GetOrCreate(companyID, apiType)

	select {
	case <-job.Done:
		if job.Err != nil {
			http.Error(w, job.Err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(job.Result))

	case <-r.Context().Done():
		http.Error(w, "request cancelled", http.StatusRequestTimeout)
	}
}

func makeAPIHandler(apiMgr *jobs.APIJobManager, apiType model.APIType) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		handleAPIRequest(w, r, apiMgr, apiType)
	}
}
