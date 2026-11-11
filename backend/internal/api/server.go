// Package api exposes the local JSON/CSV API with token authentication.
package api

import (
	"crypto/subtle"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Erfangh-80/Code-analysis/backend/internal/domain"
	"github.com/Erfangh-80/Code-analysis/backend/internal/jobs"
	"github.com/Erfangh-80/Code-analysis/backend/internal/storage"
)

type Config struct {
	Token          string
	AllowedOrigins []string
}
type Server struct {
	store  *storage.Store
	jobs   *jobs.Manager
	config Config
}

func New(store *storage.Store, manager *jobs.Manager, config Config) http.Handler {
	return &Server{store: store, jobs: manager, config: config}
}
func (s *Server) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	requestID := randomRequestID()
	writer.Header().Set("X-Request-ID", requestID)
	if s.setCORS(writer, request) {
		if request.Method == http.MethodOptions {
			writer.WriteHeader(http.StatusNoContent)
			return
		}
	}
	if request.URL.Path == "/api/v1/health" {
		writeJSON(writer, http.StatusOK, map[string]any{"status": "ok", "time": time.Now().UTC()})
		return
	}
	if !strings.HasPrefix(request.URL.Path, "/api/v1/") {
		writeError(writer, http.StatusNotFound, "NOT_FOUND", "مسیر پیدا نشد.", requestID)
		return
	}
	if !s.authorized(request) {
		writeError(writer, http.StatusUnauthorized, "UNAUTHORIZED", "توکن API معتبر نیست.", requestID)
		return
	}
	status := http.StatusInternalServerError
	defer func() {
		log.Printf("request_id=%s method=%s path=%s status=%d", requestID, request.Method, request.URL.Path, status)
	}()
	status = s.route(writer, request, requestID)
}
func (s *Server) route(w http.ResponseWriter, r *http.Request, id string) int {
	parts := strings.Split(strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/"), "/"), "/")
	if len(parts) == 1 && parts[0] == "analyses" {
		if r.Method == http.MethodPost {
			return s.createAnalysis(w, r, id)
		}
		if r.Method == http.MethodGet {
			return s.listAnalyses(w, id)
		}
	}
	if len(parts) >= 2 && parts[0] == "analyses" {
		analysisID := parts[1]
		if len(parts) == 2 && r.Method == http.MethodGet {
			return s.getAnalysis(w, id, analysisID)
		}
		if len(parts) == 3 && parts[2] == "files" && r.Method == http.MethodGet {
			return s.listFiles(w, r, id, analysisID)
		}
		if len(parts) == 4 && parts[2] == "files" && r.Method == http.MethodGet {
			return s.getFile(w, id, analysisID, parts[3])
		}
		if len(parts) == 3 && parts[2] == "modules" && r.Method == http.MethodGet {
			return s.modules(w, id, analysisID)
		}
		if len(parts) == 3 && parts[2] == "hotspots" && r.Method == http.MethodGet {
			return s.hotspots(w, r, id, analysisID)
		}
		if len(parts) == 3 && parts[2] == "heatmap" && r.Method == http.MethodGet {
			return s.heatmap(w, r, id, analysisID)
		}
		if len(parts) == 3 && parts[2] == "reports" && r.Method == http.MethodGet {
			return s.report(w, r, id, analysisID)
		}
		if len(parts) == 3 && parts[2] == "evaluations" && r.Method == http.MethodPost {
			return s.createEvaluation(w, r, id, analysisID)
		}
		if len(parts) == 4 && parts[2] == "evaluations" && r.Method == http.MethodGet {
			return s.getEvaluation(w, id, analysisID, parts[3])
		}
	}
	writeError(w, http.StatusNotFound, "NOT_FOUND", "مسیر پیدا نشد.", id)
	return http.StatusNotFound
}
func (s *Server) authorized(r *http.Request) bool {
	if s.config.Token == "" {
		return false
	}
	provided := r.Header.Get("X-Analyzer-Token")
	return len(provided) == len(s.config.Token) && subtle.ConstantTimeCompare([]byte(provided), []byte(s.config.Token)) == 1
}
func (s *Server) setCORS(w http.ResponseWriter, r *http.Request) bool {
	origin := r.Header.Get("Origin")
	for _, allowed := range s.config.AllowedOrigins {
		if origin != "" && origin == allowed {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Analyzer-Token")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			return true
		}
	}
	return false
}

type createAnalysisRequest struct {
	RepositoryPath string              `json:"repository_path"`
	Ref            string              `json:"ref"`
	Languages      []string            `json:"languages"`
	History        domain.HistoryRange `json:"history"`
	Weights        *domain.Weights     `json:"weights"`
}

func (s *Server) createAnalysis(w http.ResponseWriter, r *http.Request, id string) int {
	var body createAnalysisRequest
	if err := decode(r, &body); err != nil {
		writeError(w, 400, "INVALID_JSON", "بدنه درخواست JSON معتبر نیست.", id)
		return 400
	}
	weights := domain.DefaultWeights()
	if body.Weights != nil {
		weights = *body.Weights
	}
	analysis, err := s.jobs.Submit(domain.AnalysisInput{RepositoryPath: body.RepositoryPath, Ref: body.Ref, Languages: body.Languages, History: body.History, Weights: weights})
	if err != nil {
		return jobError(w, id, err)
	}
	writeJSON(w, http.StatusAccepted, map[string]any{"analysis_id": analysis.ID, "status": analysis.Status})
	return http.StatusAccepted
}
func (s *Server) listAnalyses(w http.ResponseWriter, id string) int {
	items, err := s.store.ListAnalyses()
	if err != nil {
		return internalError(w, id)
	}
	writeJSON(w, 200, map[string]any{"items": items})
	return 200
}
func (s *Server) getAnalysis(w http.ResponseWriter, id, analysisID string) int {
	analysis, err := s.store.GetAnalysis(analysisID)
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, 404, "ANALYSIS_NOT_FOUND", "تحلیل پیدا نشد.", id)
		return 404
	}
	if err != nil {
		return internalError(w, id)
	}
	writeJSON(w, 200, analysis)
	return 200
}
func (s *Server) listFiles(w http.ResponseWriter, r *http.Request, id, analysisID string) int {
	if !s.analysisExists(w, id, analysisID) {
		return 404
	}
	limit := queryInt(r, "limit", 50, 1, 200)
	offset := queryInt(r, "offset", 0, 0, math.MaxInt)
	query := storage.FileQuery{Limit: limit, Offset: offset, Search: r.URL.Query().Get("search"), Language: r.URL.Query().Get("language"), Module: r.URL.Query().Get("module")}
	files, total, err := s.store.ListFiles(analysisID, query)
	if err != nil {
		return internalError(w, id)
	}
	bins := queryInt(r, "bins", 10, 2, 50)
	if x, ok := optionalInt(r, "heatmap_x"); ok {
		y, yes := optionalInt(r, "heatmap_y")
		if !yes {
			writeError(w, 400, "INVALID_FILTER", "heatmap_y الزامی است.", id)
			return 400
		}
		files = filterHeatmap(files, x, y, bins)
		total = len(files)
	}
	writeJSON(w, 200, map[string]any{"items": files, "total": total, "limit": limit, "offset": offset})
	return 200
}
func (s *Server) getFile(w http.ResponseWriter, id, analysisID, fileID string) int {
	file, err := s.store.GetFile(analysisID, fileID)
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, 404, "FILE_NOT_FOUND", "فایل پیدا نشد.", id)
		return 404
	}
	if err != nil {
		return internalError(w, id)
	}
	writeJSON(w, 200, file)
	return 200
}
func (s *Server) modules(w http.ResponseWriter, id, analysisID string) int {
	items, err := s.store.ListModules(analysisID)
	if err != nil {
		return internalError(w, id)
	}
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].HighestScore == nil {
			return false
		}
		if items[j].HighestScore == nil {
			return true
		}
		return *items[i].HighestScore > *items[j].HighestScore
	})
	writeJSON(w, 200, map[string]any{"items": items})
	return 200
}
func (s *Server) hotspots(w http.ResponseWriter, r *http.Request, id, analysisID string) int {
	switch r.URL.Query().Get("level") {
	case "", "file":
		files, _, err := s.store.ListFiles(analysisID, storage.FileQuery{Limit: 200})
		if err != nil {
			return internalError(w, id)
		}
		writeJSON(w, 200, map[string]any{"level": "file", "items": files})
		return 200
	case "module":
		return s.modules(w, id, analysisID)
	default:
		writeError(w, 400, "INVALID_LEVEL", "level باید file یا module باشد.", id)
		return 400
	}
}
func (s *Server) heatmap(w http.ResponseWriter, r *http.Request, id, analysisID string) int {
	bins := queryInt(r, "bins", 10, 2, 50)
	level := r.URL.Query().Get("level")
	if level == "" {
		level = "file"
	}
	files, err := s.store.AllFiles(analysisID)
	if err != nil {
		return internalError(w, id)
	}
	var values []heatValue
	if level == "file" {
		for _, file := range files {
			if file.HotspotScore != nil && file.Normalized.Churn != nil && file.Normalized.Complexity != nil {
				values = append(values, heatValue{Name: file.Path, X: *file.Normalized.Churn, Y: *file.Normalized.Complexity, Score: *file.HotspotScore})
			}
		}
	} else if level == "module" {
		values = moduleHeatValues(files)
	} else {
		writeError(w, 400, "INVALID_LEVEL", "level باید file یا module باشد.", id)
		return 400
	}
	writeJSON(w, 200, map[string]any{"level": level, "bins": bins, "x_axis": "normalized_churn", "y_axis": "normalized_complexity", "cells": buildHeatmap(values, bins)})
	return 200
}
func (s *Server) report(w http.ResponseWriter, r *http.Request, id, analysisID string) int {
	level := r.URL.Query().Get("level")
	if level == "" {
		level = "file"
	}
	format := r.URL.Query().Get("format")
	if format == "" {
		format = "json"
	}
	if level != "file" && level != "module" {
		writeError(w, 400, "INVALID_LEVEL", "level باید file یا module باشد.", id)
		return 400
	}
	if format != "json" && format != "csv" {
		writeError(w, 400, "INVALID_FORMAT", "format باید json یا csv باشد.", id)
		return 400
	}
	analysis, err := s.store.GetAnalysis(analysisID)
	if err != nil {
		return internalError(w, id)
	}
	if level == "file" {
		files, err := s.store.AllFiles(analysisID)
		if err != nil {
			return internalError(w, id)
		}
		if format == "json" {
			writeJSON(w, 200, map[string]any{"analysis": analysis, "items": files})
			return 200
		}
		return writeFileCSV(w, files)
	}
	modules, err := s.store.ListModules(analysisID)
	if err != nil {
		return internalError(w, id)
	}
	if format == "json" {
		writeJSON(w, 200, map[string]any{"analysis": analysis, "items": modules})
		return 200
	}
	return writeModuleCSV(w, modules)
}

type createEvaluationRequest struct {
	History domain.HistoryRange `json:"history"`
}

func (s *Server) createEvaluation(w http.ResponseWriter, r *http.Request, id, analysisID string) int {
	var body createEvaluationRequest
	if err := decode(r, &body); err != nil {
		writeError(w, 400, "INVALID_JSON", "بدنه درخواست JSON معتبر نیست.", id)
		return 400
	}
	evaluation, err := s.jobs.StartEvaluation(analysisID, body.History)
	if err != nil {
		return jobError(w, id, err)
	}
	writeJSON(w, 202, map[string]any{"evaluation_id": evaluation.ID, "status": evaluation.Status})
	return 202
}
func (s *Server) getEvaluation(w http.ResponseWriter, id, analysisID, evaluationID string) int {
	value, err := s.store.GetEvaluation(analysisID, evaluationID)
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, 404, "EVALUATION_NOT_FOUND", "ارزیابی پیدا نشد.", id)
		return 404
	}
	if err != nil {
		return internalError(w, id)
	}
	writeJSON(w, 200, value)
	return 200
}
func (s *Server) analysisExists(w http.ResponseWriter, id, analysisID string) bool {
	_, err := s.store.GetAnalysis(analysisID)
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, 404, "ANALYSIS_NOT_FOUND", "تحلیل پیدا نشد.", id)
		return false
	}
	return err == nil
}

func decode(r *http.Request, value any) error {
	decoder := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(value); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return fmt.Errorf("unexpected JSON data")
	}
	return nil
}
func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
func writeError(w http.ResponseWriter, status int, code, message, id string) {
	writeJSON(w, status, map[string]any{"error": map[string]string{"code": code, "message": message}, "request_id": id})
}
func internalError(w http.ResponseWriter, id string) int {
	writeError(w, 500, "INTERNAL_ERROR", "خطای داخلی رخ داد.", id)
	return 500
}
func jobError(w http.ResponseWriter, id string, err error) int {
	var value *jobs.JobError
	if errors.As(err, &value) {
		status := 400
		if value.Code == "REPO_NOT_ALLOWED" {
			status = 403
		}
		if value.Code == "QUEUE_FULL" {
			status = 429
		}
		writeError(w, status, value.Code, value.Message, id)
		return status
	}
	return internalError(w, id)
}
func queryInt(r *http.Request, name string, fallback, min, max int) int {
	value, err := strconv.Atoi(r.URL.Query().Get(name))
	if err != nil || r.URL.Query().Get(name) == "" {
		return fallback
	}
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}
func optionalInt(r *http.Request, name string) (int, bool) {
	raw := r.URL.Query().Get(name)
	if raw == "" {
		return 0, false
	}
	value, err := strconv.Atoi(raw)
	return value, err == nil
}
func randomRequestID() string { return strconv.FormatInt(time.Now().UnixNano(), 36) }
