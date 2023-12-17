package logger

import (
	"context"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"net/http"
	"net/http/httptrace"
	"strings"
	"time"
)

const APP_ID = "zeebe-app-id"
const BMPN_ID = "zeebe-bpmn-id"
const PROCESS_KEY = "zeebe-process-key"
const JOB_TYPE = "zeebe-job-type"
const INPUTS = "zeebe-worker-inputs"
const OUTPUTS = "zeebe-worker-outputs"
const RESPONSE = "zeebe-response"
const BODY = "body"
const MESSAGE = "message"

type LoggingMiddleware struct {
	handler http.Handler
}

func (l *LoggingMiddleware) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	withAppID(req)
	LogRequest(req)
	l.handler.ServeHTTP(w, req)
}

func NewMiddleware(h http.Handler) http.Handler {
	return &LoggingMiddleware{handler: h}
}

// Inject app_id field into request's context and modify original request
func withAppID(req *http.Request) {
	appId := req.Header.Get("X-APP-ID")
	if len(appId) == 0 {
		appId = uuid.NewString()
	}
	ctx := context.WithValue(req.Context(), APP_ID, appId)
	r := req.WithContext(ctx)
	*req = *r
}

// Inject ClientTrace into request's context and modify original request
func WithClientTrace(req *http.Request) {
	var start time.Time
	trace := &httptrace.ClientTrace{
		GetConn: func(hostPort string) { start = time.Now() },
		GotFirstResponseByte: func() {
			Log(req.Context(), nil).WithField("time_to_first_byte_received", time.Since(start)).Tracef("request")
		},
	}
	r := req.WithContext(httptrace.WithClientTrace(req.Context(), trace))
	*req = *r
}

func Log(ctx context.Context, err error) *log.Entry {
	entry := log.WithContext(ctx)
	if err != nil {
		entry = entry.WithField("error", err)
	}
	if appID := ctx.Value(APP_ID); appID != nil {
		entry = entry.WithField(APP_ID, appID)
	}
	if jobType := ctx.Value(JOB_TYPE); jobType != nil {
		entry = entry.WithField(JOB_TYPE, jobType)
	}
	return entry
}

func LogRequest(req *http.Request) {
	Log(req.Context(), nil).WithFields(log.Fields{
		"request":    req.RequestURI,
		"method":     req.Method,
		"remote":     req.RemoteAddr,
		"user-agent": escape(req.UserAgent()),
		"referer":    escape(req.Referer()),
	}).Debugf("request")
}

func LogResponse(res *http.Response) {
	Log(res.Request.Context(), nil).WithFields(log.Fields{
		"status":         res.Status,
		"content-length": res.ContentLength,
	}).Debugf("response")
}

func escape(s string) string {
	return strings.Replace(strings.Replace(s, "\n", "", -1), "\r", "", -1)
}
