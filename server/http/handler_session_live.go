package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/teexue/nexakit/event"
	"github.com/teexue/nexakit/provider"
	"github.com/teexue/nexa/core/service"
	"github.com/teexue/nexakit/session"
)

type runSnapshot struct {
	Type      string             `json:"type"`
	SessionID string             `json:"session_id"`
	Agent     string             `json:"agent"`
	Messages  []provider.Message `json:"messages"`
	Metadata  map[string]string  `json:"metadata,omitempty"`
}

func (s *Server) detachRunContext() (context.Context, context.CancelFunc) {
	parent := context.Background()
	if s.shutdownCtx != nil {
		parent = s.shutdownCtx
	}
	return context.WithCancel(parent)
}

func (s *Server) publishAndStream(
	c *gin.Context,
	result *service.RunResult,
	events <-chan event.Event,
	cancel context.CancelFunc,
) {
	sessionID := result.Session.ID
	agentName := result.Config.Agent.Name
	s.logger.Info("log.agent.run_started", "session_id", sessionID, "agent", agentName)
	runStart := time.Now()
	s.health.AgentMetrics.RecordRunStart(agentName)
	s.svc.Hub.Attach(service.RunAttach{
		SessionID: sessionID,
		Agent:     agentName,
		Events:    events,
		Cancel:    cancel,
		Messages:  result.Session.GetMessages(),
		Metadata:  result.Session.GetMetadata(),
		Finish: func(completed bool) {
			result.Cleanup(s.registry)
			cancel()
			s.health.AgentMetrics.RecordRunEnd(agentName, time.Since(runStart), completed)
		},
	})
	sub := s.svc.Hub.Subscribe(sessionID, false)
	if sub == nil {
		respondError(c, http.StatusInternalServerError, "run_error", "api.error.run_error")
		return
	}
	s.streamSubscription(c, sub, sessionID, false)
}

func (s *Server) loadSessionOrRespond(c *gin.Context) (*session.Session, bool) {
	if !s.requireSessionStore(c) {
		return nil, false
	}
	sess, err := s.svc.LoadSession(c.Param("id"), identityFromGin(c).UserID)
	if err != nil {
		writeSessionLoadError(c, err)
		return nil, false
	}
	return sess, true
}

func (s *Server) handleSessionEvents(c *gin.Context) {
	sess, ok := s.loadSessionOrRespond(c)
	if !ok {
		return
	}
	sub := s.svc.Hub.Subscribe(sess.ID, true)
	if sub == nil {
		respondError(c, http.StatusNotFound, "run_not_found", "api.error.run_not_found")
		return
	}
	s.streamSubscription(c, sub, sess.ID, true)
}

func (s *Server) handleSessionAbort(c *gin.Context) {
	sess, ok := s.loadSessionOrRespond(c)
	if !ok {
		return
	}
	aborted := s.svc.Hub.Abort(sess.ID) != nil
	c.JSON(http.StatusOK, gin.H{"aborted": aborted, "session_id": sess.ID})
}

func (s *Server) requireSessionStore(c *gin.Context) bool {
	if s.store == nil {
		respondError(c, http.StatusServiceUnavailable, "session_error", "api.error.session_not_configured")
		return false
	}
	return true
}

func writeSessionLoadError(c *gin.Context, err error) {
	if errors.Is(err, session.ErrNotFound) {
		respondError(c, http.StatusNotFound, "not_found", "api.error.session_not_found")
		return
	}
	respondErrorDetails(c, errorDetails{
		Status:  http.StatusInternalServerError,
		Code:    "session_error",
		MsgKey:  "api.error.session_error",
		Details: err.Error(),
	})
}

func (s *Server) streamSubscription(
	c *gin.Context,
	sub *service.RunSub,
	sessionID string,
	replay bool,
) {
	defer sub.Close()
	flusher, ok := beginSSE(c, sessionID)
	if !ok {
		return
	}
	if replay && !writeSnapshotSSE(c, flusher, sub, sessionID) {
		return
	}
	if replay {
		for _, ev := range sub.Replay {
			if !writeSSEEvent(c, flusher, ev) {
				return
			}
		}
	}
	forwardLive(c, flusher, sub.C)
}

func beginSSE(c *gin.Context, sessionID string) (http.Flusher, bool) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Session-Id", sessionID)
	c.Header("X-Accel-Buffering", "no")
	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		respondError(c, http.StatusInternalServerError, "stream_error", "api.error.streaming_unsupported")
		return nil, false
	}
	return flusher, true
}

func writeSnapshotSSE(
	c *gin.Context,
	flusher http.Flusher,
	sub *service.RunSub,
	sessionID string,
) bool {
	frame, err := json.Marshal(runSnapshot{
		Type:      "snapshot",
		SessionID: sessionID,
		Agent:     sub.Agent,
		Messages:  sub.Messages,
		Metadata:  sub.Metadata,
	})
	if err != nil {
		return false
	}
	if _, err := fmt.Fprintf(c.Writer, "data: %s\n\n", frame); err != nil {
		return false
	}
	flusher.Flush()
	return true
}

func writeSSEEvent(c *gin.Context, flusher http.Flusher, ev event.Event) bool {
	if _, err := fmt.Fprint(c.Writer, encodeRunSSE(ev)); err != nil {
		return false
	}
	flusher.Flush()
	return true
}

func forwardLive(c *gin.Context, flusher http.Flusher, ch <-chan event.Event) {
	for {
		select {
		case <-c.Request.Context().Done():
			return
		case ev, ok := <-ch:
			if !ok {
				return
			}
			if !writeSSEEvent(c, flusher, ev) {
				return
			}
		}
	}
}
