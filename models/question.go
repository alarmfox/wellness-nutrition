package models

import (
	"database/sql"
	"fmt"
)

type Question struct {
	ID       int
	Sku      string
	Index    int
	Next     int
	Previous int
	Question string
	Star1    int
	Star2    int
	Star3    int
	Star4    int
	Star5    int
}

type QuestionRepository struct {
	db *sql.DB
}

func NewQuestionRepository(db *sql.DB) *QuestionRepository {
	return &QuestionRepository{db: db}
}

// GetAll retrieves all questions ordered by index
func (r *QuestionRepository) GetAll() ([]*Question, error) {
	query := `SELECT id, sku, index, next, previous, question, star1, star2, star3, star4, star5
			  FROM questions ORDER BY index`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var questions []*Question
	for rows.Next() {
		q := &Question{}
		err := rows.Scan(&q.ID, &q.Sku, &q.Index, &q.Next, &q.Previous, &q.Question,
			&q.Star1, &q.Star2, &q.Star3, &q.Star4, &q.Star5)
		if err != nil {
			return nil, err
		}
		questions = append(questions, q)
	}

	return questions, rows.Err()
}

// GetByID retrieves a single question by ID
func (r *QuestionRepository) GetByID(id int) (*Question, error) {
	query := `SELECT id, sku, index, next, previous, question, star1, star2, star3, star4, star5
			  FROM questions WHERE id = $1`

	q := &Question{}
	err := r.db.QueryRow(query, id).Scan(&q.ID, &q.Sku, &q.Index, &q.Next, &q.Previous, &q.Question,
		&q.Star1, &q.Star2, &q.Star3, &q.Star4, &q.Star5)
	if err != nil {
		return nil, err
	}

	return q, nil
}

// Create creates a new question
func (r *QuestionRepository) Create(q *Question) error {
	query := `INSERT INTO questions (sku, index, next, previous, question, star1, star2, star3, star4, star5)
			  VALUES ($1, $2, $3, $4, $5, 0, 0, 0, 0, 0) RETURNING id`

	err := r.db.QueryRow(query, q.Sku, q.Index, q.Next, q.Previous, q.Question).Scan(&q.ID)
	if err != nil {
		return fmt.Errorf("failed to create question: %w", err)
	}

	return nil
}

// Update updates an existing question
func (r *QuestionRepository) Update(q *Question) error {
	query := `UPDATE questions
			  SET sku = $1, index = $2, next = $3, previous = $4, question = $5
			  WHERE id = $6`

	result, err := r.db.Exec(query, q.Sku, q.Index, q.Next, q.Previous, q.Question, q.ID)
	if err != nil {
		return fmt.Errorf("failed to update question: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return fmt.Errorf("question with id %d not found", q.ID)
	}

	return nil
}

// Delete deletes a question by ID
func (r *QuestionRepository) Delete(id int) error {
	query := `DELETE FROM questions WHERE id = $1`

	result, err := r.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete question: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return fmt.Errorf("question with id %d not found", id)
	}

	return nil
}

// UpdateResults updates the star ratings for a question
func (r *QuestionRepository) UpdateResults(id int, stars [5]int) error {
	query := `UPDATE questions
			  SET star1 = star1 + $1, star2 = star2 + $2, star3 = star3 + $3,
			      star4 = star4 + $4, star5 = star5 + $5
			  WHERE id = $6`

	_, err := r.db.Exec(query, stars[0], stars[1], stars[2], stars[3], stars[4], id)
	if err != nil {
		return fmt.Errorf("failed to update results: %w", err)
	}

	return nil
}

// GetResults retrieves aggregated results for all questions
func (r *QuestionRepository) GetResults() ([]*Question, error) {
	query := `SELECT id, sku, index, next, previous, question, star1, star2, star3, star4, star5
			  FROM questions ORDER BY index`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var questions []*Question
	for rows.Next() {
		q := &Question{}
		err := rows.Scan(&q.ID, &q.Sku, &q.Index, &q.Next, &q.Previous, &q.Question,
			&q.Star1, &q.Star2, &q.Star3, &q.Star4, &q.Star5)
		if err != nil {
			return nil, err
		}
		questions = append(questions, q)
	}

	return questions, rows.Err()
}
