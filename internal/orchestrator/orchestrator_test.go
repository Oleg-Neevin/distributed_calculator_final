package orchestrator

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Oleg-Neevin/distributed_calculator_final/internal/auth"
	"github.com/Oleg-Neevin/distributed_calculator_final/internal/db"
	pb "github.com/Oleg-Neevin/distributed_calculator_final/proto/generated/proto"
)

func TestGetTask(t *testing.T) {
	server := &TaskServer{}

	database := db.GetInstance()

	userID, _ := database.CreateUser("testtask", "password")
	lastID, _ := database.GetLastExpressionID()
	expressionID := lastID + 1
	database.SaveExpression(expressionID, userID, "5+5", "processing", 0)

	taskID, err := database.SaveTask(expressionID, 5.0, 5.0, "+")
	if err != nil {
		t.Fatalf("Failed to create test task: %v", err)
	}

	task, err := server.GetTask(context.Background(), &pb.TaskRequest{})
	if err != nil {
		t.Fatalf("GetTask failed: %v", err)
	}

	if !task.HasTask {
		t.Error("Expected task, got none")
	}

	if int(task.Id) != taskID {
		t.Errorf("Task ID mismatch: expected %d, got %d", taskID, task.Id)
	}

	if task.Arg1 != 5.0 || task.Arg2 != 5.0 || task.Operation != "+" {
		t.Errorf("Task data mismatch: expected (5.0, 5.0, +), got (%f, %f, %s)",
			task.Arg1, task.Arg2, task.Operation)
	}
}

func TestSendTaskResult(t *testing.T) {
	server := &TaskServer{}

	database := db.GetInstance()

	userID, _ := database.CreateUser("testresult", "password")
	lastID, _ := database.GetLastExpressionID()
	expressionID := lastID + 1
	database.SaveExpression(expressionID, userID, "10+5", "processing", 0)

	taskID, _ := database.SaveTask(expressionID, 10.0, 5.0, "+")

	result := &pb.TaskResult{
		Id:     int32(taskID),
		Result: 15.0,
	}

	response, err := server.SendTaskResult(context.Background(), result)
	if err != nil {
		t.Fatalf("SendTaskResult failed: %v", err)
	}

	if !response.Success {
		t.Error("Expected success response, got failure")
	}

	resultValue, processed, err := database.GetTaskResult(taskID)
	if err != nil {
		t.Fatalf("Failed to get task result: %v", err)
	}

	if !processed {
		t.Error("Task should be marked as processed")
	}

	if resultValue != 15.0 {
		t.Errorf("Result mismatch: expected 15.0, got %f", resultValue)
	}
}

func TestHandleCalculate(t *testing.T) {
	reqBody := []byte(`{"expression": "3+4"}`)
	req := httptest.NewRequest("POST", "/api/v1/calculate", bytes.NewBuffer(reqBody))

	userID := 42
	token, _ := auth.GenerateToken(userID)
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()

	handler := http.HandlerFunc(handleCalculate)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusCreated {
		t.Errorf("Handler returned wrong status code: got %v want %v",
			status, http.StatusCreated)
	}

	var response map[string]int
	err := json.Unmarshal(rr.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if _, exists := response["id"]; !exists {
		t.Error("Response should contain expression ID")
	}

	database := db.GetInstance()
	expr, status, _, err := database.GetExpression(response["id"], userID)
	if err != nil {
		t.Fatalf("Failed to get expression: %v", err)
	}

	if expr != "3+4" {
		t.Errorf("Expression mismatch: expected 3+4, got %s", expr)
	}

	if status != "processing" {
		t.Errorf("Status mismatch: expected processing, got %s", status)
	}
}

func TestHandleExpressions(t *testing.T) {
	database := db.GetInstance()

	userID, _ := database.CreateUser("expruser", "password")

	lastID, _ := database.GetLastExpressionID()
	expressionID := lastID + 1
	database.SaveExpression(expressionID, userID, "7*8", "completed", 56.0)

	req := httptest.NewRequest("GET", "/api/v1/expressions", nil)

	token, _ := auth.GenerateToken(userID)
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()

	handler := http.HandlerFunc(handleExpressions)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	var response map[string]interface{}
	err := json.Unmarshal(rr.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	expressions, exists := response["expressions"].([]interface{})
	if !exists {
		t.Fatal("Response should contain expressions array")
	}

	if len(expressions) < 1 {
		t.Error("Expected at least one expression in response")
	}

	expr := expressions[0].(map[string]interface{})
	if expr["expression"] != "7*8" {
		t.Errorf("Expression mismatch: expected 7*8, got %s", expr["expression"])
	}

	if expr["status"] != "completed" {
		t.Errorf("Status mismatch: expected completed, got %s", expr["status"])
	}

	if expr["result"].(float64) != 56.0 {
		t.Errorf("Result mismatch: expected 56.0, got %f", expr["result"])
	}
}

func TestParseExpression(t *testing.T) {
	parser := TestExporter{
		ParseExpression: func(id int, userID int, expression string) {
		},
	}

	expression := "2+3"
	parser.ParseExpression(1, 1, expression)

}

type TestExporter struct {
	ParseExpression func(id int, userID int, expression string)
}
