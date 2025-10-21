# Survey Feature Documentation

## Overview

The survey feature allows users to provide satisfaction feedback through a multi-question survey. The survey is publicly accessible (no login required) and presents questions with a 1-5 star rating system. Admins can manage questions and view aggregated results.

## User Features

### Public Survey
- **URL**: `/survey`
- **Access**: Public (no login required)
- **Features**:
  - Multi-step form with progress indicator
  - 1-5 star rating for each question
  - Navigate back and forth between questions
  - Submit all responses at once
  - Thank you page after submission

### Survey Flow
1. User visits `/survey`
2. Questions are presented one at a time
3. User selects 1-5 stars for each question
4. User can navigate back to change answers
5. On final question, user clicks "Conferma" to submit
6. After submission, redirected to `/survey/thanks`
7. Thank you page auto-redirects back to survey after 10 seconds

## Admin Features

### Question Management
- **URL**: `/admin/survey/questions`
- **Access**: Admin only
- **Features**:
  - View all survey questions
  - Add new questions
  - Edit existing questions
  - Delete questions
  - Questions are automatically ordered by index

### Results Dashboard
- **URL**: `/admin/survey/results`
- **Access**: Admin only
- **Features**:
  - View aggregated results for all questions
  - See total number of responses per question
  - View distribution of 1-5 star ratings
  - Visual bar charts showing rating distribution
  - Statistics cards for each star rating

## API Endpoints

### Public Endpoints
- `POST /survey/submit` - Submit survey responses

### Admin Endpoints
- `GET /api/admin/survey/questions` - Get all questions
- `POST /api/admin/survey/questions/create` - Create a new question
- `POST /api/admin/survey/questions/update` - Update a question
- `POST /api/admin/survey/questions/delete` - Delete a question
- `GET /api/admin/survey/results` - Get aggregated results

## Database Schema

### Questions Table
```sql
CREATE TABLE questions (
    id SERIAL PRIMARY KEY,
    sku VARCHAR(255) UNIQUE NOT NULL,
    index INTEGER NOT NULL,
    next INTEGER NOT NULL,
    previous INTEGER NOT NULL,
    question TEXT NOT NULL,
    star1 INTEGER NOT NULL DEFAULT 0, 
    star2 INTEGER NOT NULL DEFAULT 0, 
    star3 INTEGER NOT NULL DEFAULT 0, 
    star4 INTEGER NOT NULL DEFAULT 0, 
    star5 INTEGER NOT NULL DEFAULT 0
);
```

**Fields:**
- `id`: Auto-incrementing primary key
- `sku`: Unique identifier for the question (e.g., "q1", "q2")
- `index`: Display order (1-based)
- `next`: Index of next question (0 if last)
- `previous`: Index of previous question (0 if first)
- `question`: The question text
- `star1-5`: Count of responses for each star rating

## Setup Instructions

### 1. Run Migrations
```bash
cd cmd/migrations
export DATABASE_URL="postgresql://user:pass@localhost:5432/dbname?sslmode=disable"
go run .
```

This creates the `questions` table.

### 2. Seed Sample Questions
```bash
cd cmd/seed
export DATABASE_URL="postgresql://user:pass@localhost:5432/dbname?sslmode=disable"
go run . -seed=survey
```

This creates 5 sample survey questions in Italian:
1. Come giudichi la qualità del servizio ricevuto?
2. Quanto sei soddisfatto/a della professionalità dello staff?
3. Le informazioni ricevute sono state chiare e utili?
4. Come valuti l'ambiente e la pulizia della struttura?
5. Raccomanderesti questo servizio ad amici o familiari?

### 3. Access the Survey
- Public survey: `http://localhost:3000/survey`
- Admin questions: `http://localhost:3000/admin/survey/questions`
- Admin results: `http://localhost:3000/admin/survey/results`

## Implementation Details

### Models (`models/question.go`)
- `Question` struct representing a survey question
- `QuestionRepository` for database operations
- Methods: GetAll, GetByID, Create, Update, Delete, UpdateResults, GetResults

### Handlers (`handlers/survey.go`)
- `SurveyHandler` for processing requests
- `SubmitSurvey`: Process survey form submissions
- `GetAllQuestions`: Return all questions (admin)
- `CreateQuestion`: Create new question (admin)
- `UpdateQuestion`: Update existing question (admin)
- `DeleteQuestion`: Delete question (admin)
- `GetResults`: Return aggregated results (admin)

### Templates
- `survey.html`: Public survey page with multi-step form
- `survey-thanks.html`: Thank you page after submission
- `survey-questions.html`: Admin page for managing questions
- `survey-results.html`: Admin page for viewing results

### Static Assets
- `css/style.css`: Survey-specific styles for star ratings and progress
- `js/index.js`: Client-side form navigation logic
- `images/logo.png`: Logo displayed on survey pages
- `images/favicon.ico`: Favicon for survey pages

## Navigation

The admin navigation has been updated to include survey links:
- From calendar/users/events pages → "Gestione Sondaggio" link
- From survey questions page → "Visualizza Risultati" link
- From survey results page → "Gestisci Domande" link

## Future Enhancements

Possible improvements:
1. Export results to CSV/Excel
2. Date range filtering for results
3. Multi-language support for questions
4. Question categories/sections
5. Required vs optional questions
6. Text feedback in addition to star ratings
7. Email notifications when responses are submitted
8. Response rate tracking
