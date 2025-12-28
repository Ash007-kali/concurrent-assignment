package requests

import (
	"assignment-concurrent-system/internal/jobs"
	"assignment-concurrent-system/internal/model"
	"net/http"
)

func NewRouter(apiMgr *jobs.APIJobManager) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc(
		"/api/financials",
		makeAPIHandler(apiMgr, model.APIFinancials),
	)

	mux.HandleFunc(
		"/api/sales",
		makeAPIHandler(apiMgr, model.APISales),
	)

	mux.HandleFunc(
		"/api/employee",
		makeAPIHandler(apiMgr, model.APIEmployee),
	)

	return mux
}
