package storage

import (
	"database/sql"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	_ "modernc.org/sqlite"
)

//go:embed migrations/001_init.sql
var schemaSQL string

//go:embed migrations/002_session_context.sql
var schemaSQL002 string

//go:embed migrations/003_domain_slug.sql
var schemaSQL003 string

//go:embed migrations/004_domain_nodes.sql
var schemaSQL004 string

// Store SQLite 存储
type Store struct {
	db *sql.DB
}

// Open 打开数据库并执行迁移
func Open(dbPath string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return nil, fmt.Errorf("创建数据目录失败: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath+"?_pragma=foreign_keys(1)")
	if err != nil {
		return nil, fmt.Errorf("打开数据库失败: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("连接数据库失败: %w", err)
	}

	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, err
	}
	if err := s.EnsureDefaultUser(); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) migrate() error {
	sqlText := schemaSQL
	if sqlText == "" {
		for _, p := range []string{"migrations/001_init.sql", filepath.Join("internal", "storage", "migrations", "001_init.sql")} {
			if b, err := os.ReadFile(p); err == nil {
				sqlText = string(b)
				break
			}
		}
	}
	if sqlText == "" {
		return fmt.Errorf("找不到迁移 SQL")
	}
	if _, err := s.db.Exec(sqlText); err != nil {
		return fmt.Errorf("执行迁移失败: %w", err)
	}
	if schemaSQL002 != "" {
		if _, err := s.db.Exec(schemaSQL002); err != nil {
			if !strings.Contains(err.Error(), "duplicate column") {
				return fmt.Errorf("执行迁移 002 失败: %w", err)
			}
		}
	}
	if schemaSQL003 != "" {
		if _, err := s.db.Exec(schemaSQL003); err != nil {
			if !strings.Contains(err.Error(), "duplicate column") {
				return fmt.Errorf("执行迁移 003 失败: %w", err)
			}
		}
	}
	if schemaSQL004 != "" {
		if _, err := s.db.Exec(schemaSQL004); err != nil {
			if !strings.Contains(err.Error(), "duplicate column") {
				return fmt.Errorf("执行迁移 004 失败: %w", err)
			}
		}
	}
	return nil
}

// Close 关闭数据库
func (s *Store) Close() error {
	return s.db.Close()
}

// DB 暴露底层连接（测试用）
func (s *Store) DB() *sql.DB {
	return s.db
}

// EnsureDefaultUser 确保默认用户存在
func (s *Store) EnsureDefaultUser() error {
	_, err := s.db.Exec(
		`INSERT OR IGNORE INTO users (id, created_at) VALUES (?, ?)`,
		DefaultUserID, time.Now().UTC(),
	)
	return err
}

// CreateDomain 创建知识领域（示例树，测试用）
func (s *Store) CreateDomain(name string) (*Domain, *KnowledgeTree, error) {
	treeJSON, err := SampleTreeJSON(uuid.New().String(), name)
	if err != nil {
		return nil, nil, err
	}
	var tree KnowledgeTree
	if err := json.Unmarshal([]byte(treeJSON), &tree); err != nil {
		return nil, nil, err
	}
	return s.CreateDomainFromTree(name, "", &tree, "", DomainSourceSkillPack)
}

const (
	DomainSourceSkillPack  = "skill_pack"
	DomainSourceGenerated  = "generated"
)

// CreateDomainFromTree 从知识树创建领域；同 slug 幂等返回已有记录
func (s *Store) CreateDomainFromTree(name, slug string, tree *KnowledgeTree, nodesJSON, source string) (*Domain, *KnowledgeTree, error) {
	if slug != "" {
		if existing, existingTree, err := s.GetDomainBySlug(slug); err == nil {
			return existing, existingTree, nil
		}
	}
	if nodesJSON == "" {
		nodesJSON = "{}"
	}
	if source == "" {
		source = DomainSourceGenerated
	}
	id := uuid.New().String()
	tree.DomainID = id
	tree.DomainName = name
	treeJSON, err := json.Marshal(tree)
	if err != nil {
		return nil, nil, err
	}
	now := time.Now().UTC()
	_, err = s.db.Exec(
		`INSERT INTO domains (id, name, tree_json, slug, created_at, nodes_json, source) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		id, name, string(treeJSON), nullIfEmpty(slug), now, nodesJSON, source,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("创建领域失败: %w", err)
	}
	domain := &Domain{ID: id, Name: name, Slug: slug, TreeJSON: string(treeJSON), Source: source, CreatedAt: now}
	return domain, tree, nil
}

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// GetDomainBySlug 按 slug 获取领域
func (s *Store) GetDomainBySlug(slug string) (*Domain, *KnowledgeTree, error) {
	var id, name, treeJSON string
	var slugVal sql.NullString
	err := s.db.QueryRow(
		`SELECT id, name, tree_json, slug FROM domains WHERE slug = ?`, slug,
	).Scan(&id, &name, &treeJSON, &slugVal)
	if err == sql.ErrNoRows {
		return nil, nil, fmt.Errorf("领域不存在")
	}
	if err != nil {
		return nil, nil, err
	}
	var tree KnowledgeTree
	if err := json.Unmarshal([]byte(treeJSON), &tree); err != nil {
		return nil, nil, err
	}
	tree.DomainID = id
	tree.DomainName = name
	domain := &Domain{ID: id, Name: name, TreeJSON: treeJSON, CreatedAt: time.Time{}}
	if slugVal.Valid {
		domain.Slug = slugVal.String
	}
	return domain, &tree, nil
}

// GetDomainSlug 获取领域 slug
func (s *Store) GetDomainSlug(domainID string) (string, error) {
	var slug sql.NullString
	err := s.db.QueryRow(`SELECT slug FROM domains WHERE id = ?`, domainID).Scan(&slug)
	if err == sql.ErrNoRows {
		return "", fmt.Errorf("领域不存在")
	}
	if err != nil {
		return "", err
	}
	if slug.Valid {
		return slug.String, nil
	}
	return "", nil
}

// GetDomainTree 获取知识树
func (s *Store) GetDomainTree(domainID string) (*KnowledgeTree, error) {
	var name, treeJSON string
	err := s.db.QueryRow(
		`SELECT name, tree_json FROM domains WHERE id = ?`, domainID,
	).Scan(&name, &treeJSON)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("领域不存在")
	}
	if err != nil {
		return nil, err
	}
	var tree KnowledgeTree
	if err := json.Unmarshal([]byte(treeJSON), &tree); err != nil {
		return nil, err
	}
	tree.DomainID = domainID
	tree.DomainName = name
	return &tree, nil
}

// GetDomainNodesJSON 获取领域节点边界 JSON
func (s *Store) GetDomainNodesJSON(domainID string) (string, error) {
	var nodesJSON sql.NullString
	err := s.db.QueryRow(`SELECT nodes_json FROM domains WHERE id = ?`, domainID).Scan(&nodesJSON)
	if err == sql.ErrNoRows {
		return "", fmt.Errorf("领域不存在")
	}
	if err != nil {
		return "", err
	}
	if nodesJSON.Valid && nodesJSON.String != "" {
		return nodesJSON.String, nil
	}
	return "{}", nil
}

// ListDomainSummaries 列出全部课程及用户进度摘要
func (s *Store) ListDomainSummaries(userID string) ([]DomainSummary, error) {
	rows, err := s.db.Query(`
		SELECT id, name, slug, source, created_at, tree_json
		FROM domains
		ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []DomainSummary
	for rows.Next() {
		var id, name, treeJSON string
		var slug, source sql.NullString
		var createdAt time.Time
		if err := rows.Scan(&id, &name, &slug, &source, &createdAt, &treeJSON); err != nil {
			return nil, err
		}
		var tree KnowledgeTree
		nodeTotal := 0
		if err := json.Unmarshal([]byte(treeJSON), &tree); err == nil {
			for _, layer := range tree.Layers {
				nodeTotal += len(layer.Nodes)
			}
		}
		completed, err := s.countCompletedNodes(userID, id)
		if err != nil {
			return nil, err
		}
		item := DomainSummary{
			ID: id, Name: name, CreatedAt: createdAt,
			NodeTotal: nodeTotal, Completed: completed,
		}
		if slug.Valid {
			item.Slug = slug.String
		}
		if source.Valid {
			item.Source = source.String
		}
		list = append(list, item)
	}
	return list, rows.Err()
}

func (s *Store) countCompletedNodes(userID, domainID string) (int, error) {
	var n int
	err := s.db.QueryRow(
		`SELECT COUNT(*) FROM user_progress WHERE user_id = ? AND domain_id = ? AND status = 'completed'`,
		userID, domainID,
	).Scan(&n)
	return n, err
}

// ListProgress 获取用户学习进度
func (s *Store) ListProgress(userID, domainID string) ([]UserProgress, error) {
	query := `SELECT user_id, domain_id, node_key, layer, status, mastery, updated_at
		FROM user_progress WHERE user_id = ?`
	args := []any{userID}
	if domainID != "" {
		query += ` AND domain_id = ?`
		args = append(args, domainID)
	}
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []UserProgress
	for rows.Next() {
		var p UserProgress
		if err := rows.Scan(&p.UserID, &p.DomainID, &p.NodeKey, &p.Layer, &p.Status, &p.Mastery, &p.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, p)
	}
	return list, rows.Err()
}

// UpsertProgress 更新或插入进度
func (s *Store) UpsertProgress(p UserProgress) error {
	_, err := s.db.Exec(`
		INSERT INTO user_progress (user_id, domain_id, node_key, layer, status, mastery, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(user_id, domain_id, node_key) DO UPDATE SET
			layer=excluded.layer, status=excluded.status, mastery=excluded.mastery, updated_at=excluded.updated_at`,
		p.UserID, p.DomainID, p.NodeKey, p.Layer, p.Status, p.Mastery, time.Now().UTC(),
	)
	return err
}

// CreateSession 创建教学会话
func (s *Store) CreateSession(userID, domainID, domainSlug, nodeKey, phase string, ctx *SessionContext) (*Session, error) {
	id := uuid.New().String()
	now := time.Now().UTC()
	if phase == "" {
		phase = "explain"
	}
	ctxJSON := "{}"
	if ctx != nil {
		b, _ := json.Marshal(ctx)
		ctxJSON = string(b)
	}
	_, err := s.db.Exec(
		`INSERT INTO sessions (id, user_id, domain_id, node_key, status, created_at, phase, context_json, domain_slug)
		 VALUES (?, ?, ?, ?, 'active', ?, ?, ?, ?)`,
		id, userID, domainID, nodeKey, now, phase, ctxJSON, domainSlug,
	)
	if err != nil {
		return nil, err
	}
	return &Session{
		ID: id, UserID: userID, DomainID: domainID, DomainSlug: domainSlug,
		NodeKey: nodeKey, Status: "active", Phase: phase, ContextJSON: ctxJSON, CreatedAt: now,
	}, nil
}

// GetSession 获取会话
func (s *Store) GetSession(id string) (*Session, error) {
	var sess Session
	err := s.db.QueryRow(
		`SELECT id, user_id, domain_id, node_key, status, created_at,
		 COALESCE(phase,'explain'), COALESCE(context_json,'{}'), COALESCE(domain_slug,'')
		 FROM sessions WHERE id = ?`, id,
	).Scan(&sess.ID, &sess.UserID, &sess.DomainID, &sess.NodeKey, &sess.Status, &sess.CreatedAt,
		&sess.Phase, &sess.ContextJSON, &sess.DomainSlug)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("会话不存在")
	}
	return &sess, err
}

// UpdateSession 更新会话阶段与上下文
func (s *Store) UpdateSession(sess *Session) error {
	ctxJSON := sess.ContextJSON
	if ctxJSON == "" {
		ctxJSON = "{}"
	}
	_, err := s.db.Exec(
		`UPDATE sessions SET phase = ?, context_json = ?, status = ? WHERE id = ?`,
		sess.Phase, ctxJSON, sess.Status, sess.ID,
	)
	return err
}

// ParseSessionContext 解析 context_json
func ParseSessionContext(sess *Session) SessionContext {
	var ctx SessionContext
	if sess.ContextJSON != "" {
		_ = json.Unmarshal([]byte(sess.ContextJSON), &ctx)
	}
	if ctx.DomainSlug == "" {
		ctx.DomainSlug = sess.DomainSlug
	}
	return ctx
}

// SaveSessionContext 写回 context_json
func SaveSessionContext(sess *Session, ctx SessionContext) error {
	b, err := json.Marshal(ctx)
	if err != nil {
		return err
	}
	sess.ContextJSON = string(b)
	return nil
}

// AddMessage 添加会话消息
func (s *Store) AddMessage(sessionID, role, content string) (*SessionMessage, error) {
	res, err := s.db.Exec(
		`INSERT INTO session_messages (session_id, role, content, created_at) VALUES (?, ?, ?, ?)`,
		sessionID, role, content, time.Now().UTC(),
	)
	if err != nil {
		return nil, err
	}
	msgID, _ := res.LastInsertId()
	return &SessionMessage{
		ID:        msgID,
		SessionID: sessionID,
		Role:      role,
		Content:   content,
		CreatedAt: time.Now().UTC(),
	}, nil
}

// ListMessages 获取会话消息列表
func (s *Store) ListMessages(sessionID string) ([]SessionMessage, error) {
	rows, err := s.db.Query(
		`SELECT id, session_id, role, content, created_at FROM session_messages WHERE session_id = ? ORDER BY id`,
		sessionID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []SessionMessage
	for rows.Next() {
		var m SessionMessage
		if err := rows.Scan(&m.ID, &m.SessionID, &m.Role, &m.Content, &m.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, m)
	}
	return list, rows.Err()
}

// FindActiveSession 查找节点上未完成的活跃会话
func (s *Store) FindActiveSession(userID, domainID, nodeKey string) (*Session, error) {
	var sess Session
	err := s.db.QueryRow(
		`SELECT id, user_id, domain_id, node_key, status, created_at,
		 COALESCE(phase,'explain'), COALESCE(context_json,'{}'), COALESCE(domain_slug,'')
		 FROM sessions
		 WHERE user_id = ? AND domain_id = ? AND node_key = ?
		   AND status = 'active' AND COALESCE(phase,'explain') != 'completed'
		 ORDER BY created_at DESC LIMIT 1`,
		userID, domainID, nodeKey,
	).Scan(&sess.ID, &sess.UserID, &sess.DomainID, &sess.NodeKey, &sess.Status, &sess.CreatedAt,
		&sess.Phase, &sess.ContextJSON, &sess.DomainSlug)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &sess, nil
}

// FindLatestSession 查找节点上最近一次会话（含已完成，用于恢复聊天记录）
func (s *Store) FindLatestSession(userID, domainID, nodeKey string) (*Session, error) {
	var sess Session
	err := s.db.QueryRow(
		`SELECT id, user_id, domain_id, node_key, status, created_at,
		 COALESCE(phase,'explain'), COALESCE(context_json,'{}'), COALESCE(domain_slug,'')
		 FROM sessions
		 WHERE user_id = ? AND domain_id = ? AND node_key = ?
		 ORDER BY created_at DESC LIMIT 1`,
		userID, domainID, nodeKey,
	).Scan(&sess.ID, &sess.UserID, &sess.DomainID, &sess.NodeKey, &sess.Status, &sess.CreatedAt,
		&sess.Phase, &sess.ContextJSON, &sess.DomainSlug)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &sess, nil
}

// DeleteMessage 删除单条消息（LLM 失败回滚用）
func (s *Store) DeleteMessage(msgID int64) error {
	_, err := s.db.Exec(`DELETE FROM session_messages WHERE id = ?`, msgID)
	return err
}
