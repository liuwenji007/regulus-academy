package storage

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
)

const PlanningOpenMessage = "你好，我是行动助手。不用整理成完美句子——先把现在脑子里装的事、最近一直拖着没动的事，还有学习上卡住的地方，随便说几条就行。"

// CreatePlanningSession 创建规划会话
func (s *Store) CreatePlanningSession(userID, phase string) (*PlanningSession, error) {
	id := uuid.New().String()
	now := time.Now().UTC()
	if phase == "" {
		phase = "intake"
	}
	_, err := s.db.Exec(
		`INSERT INTO planning_sessions (id, user_id, phase, plan_json, status, created_at, updated_at)
		 VALUES (?, ?, ?, '{}', 'active', ?, ?)`,
		id, userID, phase, now, now,
	)
	if err != nil {
		return nil, err
	}
	return &PlanningSession{
		ID: id, UserID: userID, Phase: phase, PlanJSON: "{}",
		Status: "active", CreatedAt: now, UpdatedAt: now,
	}, nil
}

// GetPlanningSession 获取规划会话
func (s *Store) GetPlanningSession(id string) (*PlanningSession, error) {
	var sess PlanningSession
	var planJSON sql.NullString
	err := s.db.QueryRow(
		`SELECT id, user_id, COALESCE(phase,'intake'), COALESCE(plan_json,'{}'),
		 COALESCE(status,'active'), created_at, updated_at
		 FROM planning_sessions WHERE id = ?`, id,
	).Scan(&sess.ID, &sess.UserID, &sess.Phase, &planJSON, &sess.Status, &sess.CreatedAt, &sess.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("规划会话不存在")
	}
	if err != nil {
		return nil, err
	}
	if planJSON.Valid {
		sess.PlanJSON = planJSON.String
	} else {
		sess.PlanJSON = "{}"
	}
	return &sess, nil
}

// FindActivePlanningSession 查找用户最近一次 active 规划会话
func (s *Store) FindActivePlanningSession(userID string) (*PlanningSession, error) {
	var sess PlanningSession
	var planJSON sql.NullString
	err := s.db.QueryRow(
		`SELECT id, user_id, COALESCE(phase,'intake'), COALESCE(plan_json,'{}'),
		 COALESCE(status,'active'), created_at, updated_at
		 FROM planning_sessions
		 WHERE user_id = ? AND status = 'active'
		 ORDER BY updated_at DESC LIMIT 1`, userID,
	).Scan(&sess.ID, &sess.UserID, &sess.Phase, &planJSON, &sess.Status, &sess.CreatedAt, &sess.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if planJSON.Valid {
		sess.PlanJSON = planJSON.String
	} else {
		sess.PlanJSON = "{}"
	}
	return &sess, nil
}

// UpdatePlanningSession 更新规划会话
func (s *Store) UpdatePlanningSession(sess *PlanningSession) error {
	now := time.Now().UTC()
	planJSON := sess.PlanJSON
	if planJSON == "" {
		planJSON = "{}"
	}
	_, err := s.db.Exec(
		`UPDATE planning_sessions SET phase = ?, plan_json = ?, status = ?, updated_at = ? WHERE id = ?`,
		sess.Phase, planJSON, sess.Status, now, sess.ID,
	)
	if err != nil {
		return err
	}
	sess.UpdatedAt = now
	return nil
}

// AddPlanningMessage 添加规划消息
func (s *Store) AddPlanningMessage(sessionID, role, content string) (*PlanningMessage, error) {
	now := time.Now().UTC()
	res, err := s.db.Exec(
		`INSERT INTO planning_messages (session_id, role, content, created_at) VALUES (?, ?, ?, ?)`,
		sessionID, role, content, now,
	)
	if err != nil {
		return nil, err
	}
	msgID, _ := res.LastInsertId()
	return &PlanningMessage{
		ID: msgID, SessionID: sessionID, Role: role, Content: content, CreatedAt: now,
	}, nil
}

// ListPlanningMessages 获取规划消息列表
func (s *Store) ListPlanningMessages(sessionID string) ([]PlanningMessage, error) {
	rows, err := s.db.Query(
		`SELECT id, session_id, role, content, created_at FROM planning_messages WHERE session_id = ? ORDER BY id`,
		sessionID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []PlanningMessage
	for rows.Next() {
		var m PlanningMessage
		if err := rows.Scan(&m.ID, &m.SessionID, &m.Role, &m.Content, &m.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, m)
	}
	return list, rows.Err()
}

// ArchiveOtherPlanningSessions 将同用户其他 active 会话归档（新建时可选）
func (s *Store) ArchiveOtherPlanningSessions(userID, keepID string) error {
	_, err := s.db.Exec(
		`UPDATE planning_sessions SET status = 'archived', updated_at = ? WHERE user_id = ? AND status = 'active' AND id != ?`,
		time.Now().UTC(), userID, keepID,
	)
	return err
}
