package storage

import (
	"database/sql"
	"time"
)

// UpsertMistake 记录或更新错题
func (s *Store) UpsertMistake(userID, domainID, nodeKey, concept string) error {
	now := time.Now().UTC()
	var id int64
	err := s.db.QueryRow(
		`SELECT id FROM mistakes WHERE user_id = ? AND domain_id = ? AND node_key = ? AND concept = ?`,
		userID, domainID, nodeKey, concept,
	).Scan(&id)
	if err == sql.ErrNoRows {
		_, err = s.db.Exec(
			`INSERT INTO mistakes (user_id, domain_id, node_key, concept, wrong_count, reinforcement_count, last_wrong, created_at)
			 VALUES (?, ?, ?, ?, 1, 0, ?, ?)`,
			userID, domainID, nodeKey, concept, now, now,
		)
		return err
	}
	if err != nil {
		return err
	}
	_, err = s.db.Exec(
		`UPDATE mistakes SET wrong_count = wrong_count + 1, last_wrong = ? WHERE id = ?`,
		now, id,
	)
	return err
}

// ListMistakesForReinforce 待强化错题（reinforcement_count < 2）
func (s *Store) ListMistakesForReinforce(userID, domainID string, limit int) ([]Mistake, error) {
	if limit <= 0 {
		limit = 5
	}
	rows, err := s.db.Query(
		`SELECT id, user_id, domain_id, node_key, concept, wrong_count, reinforcement_count, last_wrong
		 FROM mistakes WHERE user_id = ? AND domain_id = ? AND reinforcement_count < 2
		 ORDER BY last_wrong ASC LIMIT ?`,
		userID, domainID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanMistakes(rows)
}

// IncrementReinforcement 答对强化题后计数+1
func (s *Store) IncrementReinforcement(userID, domainID, concept string) error {
	_, err := s.db.Exec(
		`UPDATE mistakes SET reinforcement_count = reinforcement_count + 1
		 WHERE user_id = ? AND domain_id = ? AND concept = ?`,
		userID, domainID, concept,
	)
	return err
}

func scanMistakes(rows *sql.Rows) ([]Mistake, error) {
	var list []Mistake
	for rows.Next() {
		var m Mistake
		var lastWrong sql.NullTime
		if err := rows.Scan(&m.ID, &m.UserID, &m.DomainID, &m.NodeKey, &m.Concept,
			&m.WrongCount, &m.ReinforcementCount, &lastWrong); err != nil {
			return nil, err
		}
		if lastWrong.Valid {
			t := lastWrong.Time
			m.LastWrong = &t
		}
		list = append(list, m)
	}
	return list, rows.Err()
}
