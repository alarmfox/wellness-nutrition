package handlers

import (
	"sync"
	"time"

	"github.com/alarmfox/wellness-nutrition/app/models"
)

const referenceCacheTTL = 2 * time.Minute

var surveyQuestionsCache = struct {
	sync.Mutex
	expiresAt time.Time
	questions []*models.Question
}{}

func getCachedSurveyQuestions(repo *models.QuestionRepository) ([]*models.Question, error) {
	now := time.Now()

	surveyQuestionsCache.Lock()
	if now.Before(surveyQuestionsCache.expiresAt) && surveyQuestionsCache.questions != nil {
		questions := cloneQuestions(surveyQuestionsCache.questions)
		surveyQuestionsCache.Unlock()
		return questions, nil
	}
	surveyQuestionsCache.Unlock()

	questions, err := repo.GetAll()
	if err != nil {
		return nil, err
	}

	surveyQuestionsCache.Lock()
	surveyQuestionsCache.questions = cloneQuestions(questions)
	surveyQuestionsCache.expiresAt = now.Add(referenceCacheTTL)
	surveyQuestionsCache.Unlock()

	return cloneQuestions(questions), nil
}

func invalidateSurveyQuestionsCache() {
	surveyQuestionsCache.Lock()
	surveyQuestionsCache.expiresAt = time.Time{}
	surveyQuestionsCache.questions = nil
	surveyQuestionsCache.Unlock()
}

func cloneQuestions(questions []*models.Question) []*models.Question {
	if questions == nil {
		return nil
	}

	cloned := make([]*models.Question, len(questions))
	for i, question := range questions {
		if question == nil {
			continue
		}
		q := *question
		cloned[i] = &q
	}
	return cloned
}

func cloneInstructors(instructors []*models.Instructor) []*models.Instructor {
	if instructors == nil {
		return nil
	}

	cloned := make([]*models.Instructor, len(instructors))
	for i, instructor := range instructors {
		if instructor == nil {
			continue
		}
		ins := *instructor
		cloned[i] = &ins
	}
	return cloned
}
