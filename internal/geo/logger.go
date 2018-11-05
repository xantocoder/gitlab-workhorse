package geo

import (
	"net/http"

	log "github.com/sirupsen/logrus"
)

func logHTTPError(w http.ResponseWriter, err error, msg string) {
	http.Error(w, msg, 500)

	log.WithFields(log.Fields{
		"msg":  msg,
		"code": 500,
		"err":  err,
	}).Error("geo.logHTTPError")
}
