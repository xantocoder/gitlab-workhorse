package healthcheck

import (
	"encoding/json"
	"net/http"
)

type status struct {
	HTTPStatus int               `json:"-"`
	Status     string            `json:"status"`
	Probes     map[string]string `json:"probes"`
}

func (s *status) Write(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(s.HTTPStatus)
	json.NewEncoder(w).Encode(s)
}

func createStatus(ok bool, probes map[string]error) status {
	status := status{
		Probes: make(map[string]string),
	}

	if ok {
		status.Status = "ok"
		status.HTTPStatus = http.StatusOK
	} else {
		status.Status = "fail"
		status.HTTPStatus = http.StatusInternalServerError
	}

	for probeURL, probeErr := range probes {
		if probeErr != nil {
			status.Probes[probeURL] = probeErr.Error()
		} else {
			status.Probes[probeURL] = "OK"
		}
	}

	return status
}
