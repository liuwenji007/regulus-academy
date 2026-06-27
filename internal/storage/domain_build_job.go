package storage

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	DomainBuildJobRunning = "running"
	DomainBuildJobDone    = "done"
	DomainBuildJobFailed  = "failed"

	DomainJobKindBuild    = "build"
	DomainJobKindExtend   = "extend"
	DomainJobKindAudit    = "audit"
	DomainJobKindOptimize = "optimize"
)

// DomainBuildJob 异步建课任务
type DomainBuildJob struct {
	ID         string
	UserID     string
	Topic      string
	Goal       string
	Force      bool
	JobKind    string
	DomainID   string
	Status     string
	Phase      string
	Message    string
	ResultJSON string
	Error      string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// CreateDomainBuildJob 创建 running 状态建课任务
func (s *Store) CreateDomainBuildJob(userID, topic, goal string, force bool) (*DomainBuildJob, error) {
	return s.CreateDomainBuildJobEx(userID, topic, goal, force, DomainJobKindBuild, "")
}

// CreateDomainBuildJobEx 创建带 job_kind 与 domain_id 的异步任务。
func (s *Store) CreateDomainBuildJobEx(userID, topic, goal string, force bool, jobKind, domainID string) (*DomainBuildJob, error) {
	userID = normalizeUserID(userID)
	topic = strings.TrimSpace(topic)
	if topic == "" {
		return nil, fmt.Errorf("主题不能为空")
	}
	if strings.TrimSpace(jobKind) == "" {
		jobKind = DomainJobKindBuild
	}
	now := time.Now().UTC()
	job := &DomainBuildJob{
		ID:       uuid.New().String(),
		UserID:   userID,
		Topic:    topic,
		Goal:     strings.TrimSpace(goal),
		Force:    force,
		JobKind:  jobKind,
		DomainID: strings.TrimSpace(domainID),
		Status:   DomainBuildJobRunning,
		Phase:    "starting",
		Message:  "任务已创建",
		CreatedAt: now,
		UpdatedAt: now,
	}
	forceInt := 0
	if force {
		forceInt = 1
	}
	_, err := s.db.Exec(
		`INSERT INTO domain_build_jobs (id, user_id, topic, goal, force_build, job_kind, domain_id, status, phase, message, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		job.ID, job.UserID, job.Topic, job.Goal, forceInt, job.JobKind, nullIfEmpty(job.DomainID),
		job.Status, job.Phase, job.Message,
		job.CreatedAt.Format(time.RFC3339), job.UpdatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return nil, err
	}
	return job, nil
}

// UpdateDomainBuildJobProgress 更新阶段与提示文案
func (s *Store) UpdateDomainBuildJobProgress(id, phase, message string) error {
	phase = strings.TrimSpace(phase)
	message = strings.TrimSpace(message)
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := s.db.Exec(
		`UPDATE domain_build_jobs SET phase = ?, message = ?, updated_at = ? WHERE id = ? AND status = ?`,
		phase, message, now, id, DomainBuildJobRunning,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// FinishDomainBuildJob 标记任务成功并写入结果 JSON
func (s *Store) FinishDomainBuildJob(id, resultJSON string) error {
	return s.FinishDomainBuildJobEx(id, resultJSON, "done", "课程已就绪")
}

// FinishDomainBuildJobEx 标记任务成功（自定义完成文案）。
func (s *Store) FinishDomainBuildJobEx(id, resultJSON, phase, message string) error {
	if strings.TrimSpace(phase) == "" {
		phase = "done"
	}
	if strings.TrimSpace(message) == "" {
		message = "任务已完成"
	}
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := s.db.Exec(
		`UPDATE domain_build_jobs SET status = ?, phase = ?, message = ?, result_json = ?, error = NULL, updated_at = ?
		 WHERE id = ? AND status = ?`,
		DomainBuildJobDone, phase, message, resultJSON, now, id, DomainBuildJobRunning,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// FailDomainBuildJob 标记任务失败
func (s *Store) FailDomainBuildJob(id, errMsg string) error {
	errMsg = strings.TrimSpace(errMsg)
	if errMsg == "" {
		errMsg = "建课失败"
	}
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := s.db.Exec(
		`UPDATE domain_build_jobs SET status = ?, phase = ?, message = ?, error = ?, updated_at = ?
		 WHERE id = ? AND status = ?`,
		DomainBuildJobFailed, "failed", errMsg, errMsg, now, id, DomainBuildJobRunning,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// GetDomainBuildJob 按 ID 读取任务（须属于该用户）
func (s *Store) GetDomainBuildJob(userID, id string) (*DomainBuildJob, error) {
	userID = normalizeUserID(userID)
	var j DomainBuildJob
	var forceInt int
	var created, updated string
	var jobKind, domainID string
	err := s.db.QueryRow(
		`SELECT id, user_id, topic, goal, force_build,
		        COALESCE(job_kind, 'build'), COALESCE(domain_id, ''),
		        status, phase, message,
		        COALESCE(result_json, ''), COALESCE(error, ''), created_at, updated_at
		 FROM domain_build_jobs WHERE id = ? AND user_id = ?`,
		id, userID,
	).Scan(
		&j.ID, &j.UserID, &j.Topic, &j.Goal, &forceInt,
		&jobKind, &domainID,
		&j.Status, &j.Phase, &j.Message,
		&j.ResultJSON, &j.Error, &created, &updated,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("建课任务不存在")
	}
	if err != nil {
		return nil, err
	}
	j.Force = forceInt != 0
	j.JobKind = strings.TrimSpace(jobKind)
	if j.JobKind == "" {
		j.JobKind = DomainJobKindBuild
	}
	j.DomainID = strings.TrimSpace(domainID)
	if t, e := time.Parse(time.RFC3339, created); e == nil {
		j.CreatedAt = t
	}
	if t, e := time.Parse(time.RFC3339, updated); e == nil {
		j.UpdatedAt = t
	}
	return &j, nil
}
