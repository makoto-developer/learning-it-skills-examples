package main

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/example/linkshort/internal/service"
)

type api struct {
	svc *service.Service
}

func newRouter(svc *service.Service) http.Handler {
	handlers := &api{svc: svc}
	mux := http.NewServeMux()

	mux.HandleFunc("POST /v1/links", handlers.createLink)
	mux.HandleFunc("GET /v1/links", handlers.listLinks)
	mux.HandleFunc("GET /v1/links/{key}", handlers.getLink)
	mux.HandleFunc("GET /r/{key}", handlers.redirect)
	mux.HandleFunc("GET /healthz", alwaysOK)
	mux.HandleFunc("GET /readyz", alwaysOK)

	return mux
}

func alwaysOK(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok\n"))
}

func (a *api) createLink(w http.ResponseWriter, r *http.Request) {
	var body struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&body); err != nil {
		writeError(w, r, http.StatusBadRequest, "リクエストボディが JSON として読めません")
		return
	}

	link, err := a.svc.CreateLink(r.Context(), body.URL)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, link)
}

func (a *api) getLink(w http.ResponseWriter, r *http.Request) {
	link, err := a.svc.GetLink(r.Context(), r.PathValue("key"))
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, link)
}

func (a *api) listLinks(w http.ResponseWriter, r *http.Request) {
	size, err := strconv.Atoi(firstNonEmpty(r.URL.Query().Get("page_size"), "0"))
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "page_size は整数で指定してください")
		return
	}

	res, err := a.svc.ListLinks(r.Context(), service.ListLinksRequest{
		PageSize:  size,
		PageToken: r.URL.Query().Get("page_token"),
	})
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func (a *api) redirect(w http.ResponseWriter, r *http.Request) {
	link, err := a.svc.GetLink(r.Context(), r.PathValue("key"))
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	// 転送先を後から差し替えられるよう、恒久ではなく一時リダイレクトにする
	http.Redirect(w, r, link.URL, http.StatusFound)
}

// writeServiceError は業務エラーを HTTP のステータスに翻訳する。
// この対応表を1箇所に閉じ込めることで、service 層は HTTP を知らずに済む。
func writeServiceError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, service.ErrInvalidArgument):
		writeError(w, r, http.StatusBadRequest, err.Error())
	case errors.Is(err, service.ErrNotFound):
		writeError(w, r, http.StatusNotFound, "そのキーのリンクはありません")
	default:
		slog.ErrorContext(r.Context(), "unhandled error", "error", err, "path", r.URL.Path)
		writeError(w, r, http.StatusInternalServerError, "内部エラーが発生しました")
	}
}

func writeError(w http.ResponseWriter, r *http.Request, status int, message string) {
	slog.WarnContext(r.Context(), "request rejected", "status", status, "message", message)
	writeJSON(w, status, map[string]string{"error": message})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		slog.Error("failed to write response", "error", err)
	}
}

func firstNonEmpty(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

// statusRecorder はログに載せるためにステータスコードを覚えておく。
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (s *statusRecorder) WriteHeader(status int) {
	s.status = status
	s.ResponseWriter.WriteHeader(status)
}

// withRequestLog は全リクエストを構造化ログで1行ずつ出す。
// grep ではなくフィールド検索で絞れるので、障害時に status>=500 だけ抜ける。
func withRequestLog(next http.Handler, logger *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		recorder := &statusRecorder{ResponseWriter: w, status: http.StatusOK}

		next.ServeHTTP(recorder, r)

		logger.InfoContext(r.Context(), "http request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", recorder.status,
			"duration_ms", time.Since(started).Milliseconds(),
			"user_agent", r.UserAgent(),
		)
	})
}
