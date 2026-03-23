package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/atopos31/llmio/common"
	"github.com/atopos31/llmio/models"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupHomeTestDB(t *testing.T) func() {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}

	if err := db.AutoMigrate(&models.ChatLog{}); err != nil {
		t.Fatalf("failed to migrate chat logs: %v", err)
	}

	models.DB = db

	return func() {
		models.DB = nil
	}
}

func TestModelTokenUsages_ReturnsTopModelsWithinHours(t *testing.T) {
	cleanup := setupHomeTestDB(t)
	defer cleanup()

	now := time.Now()
	logs := []models.ChatLog{
		{Name: "gpt-4.1", Usage: models.Usage{TotalTokens: 500}, Model: gorm.Model{CreatedAt: now.Add(-1 * time.Hour)}},
		{Name: "gpt-4.1", Usage: models.Usage{TotalTokens: 700}, Model: gorm.Model{CreatedAt: now.Add(-2 * time.Hour)}},
		{Name: "claude-3.7", Usage: models.Usage{TotalTokens: 900}, Model: gorm.Model{CreatedAt: now.Add(-3 * time.Hour)}},
		{Name: "gpt-4o-mini", Usage: models.Usage{TotalTokens: 600}, Model: gorm.Model{CreatedAt: now.Add(-4 * time.Hour)}},
		{Name: "deepseek-v3", Usage: models.Usage{TotalTokens: 400}, Model: gorm.Model{CreatedAt: now.Add(-5 * time.Hour)}},
		{Name: "qwen-max", Usage: models.Usage{TotalTokens: 300}, Model: gorm.Model{CreatedAt: now.Add(-6 * time.Hour)}},
		{Name: "old-model", Usage: models.Usage{TotalTokens: 5000}, Model: gorm.Model{CreatedAt: now.Add(-48 * time.Hour)}},
	}
	if err := models.DB.Create(&logs).Error; err != nil {
		t.Fatalf("failed to seed chat logs: %v", err)
	}

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Params = gin.Params{{Key: "hours", Value: "24"}}
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/metrics/model-tokens/24", nil)

	ModelTokenUsages(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}

	var response common.Response
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	payload, err := json.Marshal(response.Data)
	if err != nil {
		t.Fatalf("failed to re-encode data: %v", err)
	}

	var usages []ModelTokenUsage
	if err := json.Unmarshal(payload, &usages); err != nil {
		t.Fatalf("failed to decode usages: %v", err)
	}

	if len(usages) != 5 {
		t.Fatalf("expected top 5 models, got %d", len(usages))
	}

	if usages[0].Model != "gpt-4.1" || usages[0].Tokens != 1200 {
		t.Fatalf("expected top model gpt-4.1 with 1200 tokens, got %#v", usages[0])
	}

	for _, usage := range usages {
		if usage.Model == "old-model" {
			t.Fatalf("expected old-model to be excluded from 24h window")
		}
	}
}

func TestModelTokenUsages_RejectsInvalidHours(t *testing.T) {
	cleanup := setupHomeTestDB(t)
	defer cleanup()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Params = gin.Params{{Key: "hours", Value: "0"}}
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/metrics/model-tokens/0", nil)

	ModelTokenUsages(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200 for bad request wrapper, got %d", recorder.Code)
	}

	var response common.Response
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected bad request code, got %d", response.Code)
	}
}
