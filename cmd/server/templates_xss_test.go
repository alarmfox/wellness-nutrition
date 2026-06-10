package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSurveyQuestionTemplatesDoNotInterpolateQuestionHTML(t *testing.T) {
	for _, path := range []string{
		"templates/survey-questions.html",
		"templates/survey-results.html",
	} {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(data), "${q.Question}") {
			t.Fatalf("%s still interpolates question text into HTML", path)
		}
	}
}

func TestIndexTemplateDoesNotInterpolateInstructorHTML(t *testing.T) {
	data, err := os.ReadFile("templates/index.html")
	if err != nil {
		t.Fatal(err)
	}
	for _, pattern := range []string{"<h1>Slot per ${instructorName}", "showSlotsForInstructor('${instructor.ID}', '${instructorName}')"} {
		if strings.Contains(string(data), pattern) {
			t.Fatalf("index template still interpolates instructor text with %q", pattern)
		}
	}
}

func TestCalendarDeleteUsesBooleanRefundState(t *testing.T) {
	data, err := os.ReadFile("static/js/calendar.js")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), `refund == "true"`) {
		t.Fatal("calendar delete still compares boolean refund state to a string")
	}
	if !strings.Contains(string(data), "refundCheckbox.checked") {
		t.Fatal("calendar delete should read the refund checkbox state")
	}
}

func TestTemplatesDoNotUseInlineCSS(t *testing.T) {
	files, err := filepath.Glob("templates/*.html")
	if err != nil {
		t.Fatal(err)
	}

	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		content := string(data)
		if strings.Contains(content, "<style") || strings.Contains(content, "style=") {
			t.Fatalf("%s contains inline CSS", file)
		}
	}
}
