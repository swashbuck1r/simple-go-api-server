package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// newTestRouter builds the same routes as setupRouter but with an explicit,
// deterministic set of todos (setupRouter seeds from viper, which we don't want
// in a unit test).
func newTestRouter(initial []Todo) *gin.Engine {
	r := gin.New()
	todoList := newTodoList(initial)
	r.GET("/todos", todoList.getTodos)
	r.POST("/todos", todoList.createTodo)
	r.PUT("/todos/:id", todoList.updateTodoStatus)
	r.DELETE("/todos/:id", todoList.deleteTodo)
	return r
}

func TestGetTodos(t *testing.T) {
	r := newTestRouter([]Todo{{ID: "1", Title: "write tests", Complete: false}})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/todos", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("GET /todos status = %d, want %d", w.Code, http.StatusOK)
	}
	var got []Todo
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("decoding body: %v", err)
	}
	if len(got) != 1 || got[0].Title != "write tests" {
		t.Fatalf("GET /todos body = %+v, want one todo titled %q", got, "write tests")
	}
}

func TestCreateTodo(t *testing.T) {
	cases := []struct {
		name     string
		body     string
		wantCode int
	}{
		{name: "valid", body: `{"id":"2","title":"ship v0","complete":false}`, wantCode: http.StatusCreated},
		{name: "malformed json", body: `{not json`, wantCode: http.StatusBadRequest},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := newTestRouter(nil)
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/todos", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			r.ServeHTTP(w, req)

			if w.Code != tc.wantCode {
				t.Fatalf("POST /todos (%s) status = %d, want %d", tc.name, w.Code, tc.wantCode)
			}
		})
	}
}

func TestUpdateTodoStatusToggles(t *testing.T) {
	todoList := newTodoList([]Todo{{ID: "1", Title: "toggle me", Complete: false}})

	// Toggle once: false -> true.
	todoList.toggleForTest(t, "1")
	if !todoList.Todos[0].Complete {
		t.Fatalf("after one toggle, Complete = false, want true")
	}
	// Toggle again: true -> false.
	todoList.toggleForTest(t, "1")
	if todoList.Todos[0].Complete {
		t.Fatalf("after two toggles, Complete = true, want false")
	}
}

func TestUpdateTodoStatusNotFound(t *testing.T) {
	r := newTestRouter([]Todo{{ID: "1", Title: "exists", Complete: false}})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/todos/does-not-exist", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("PUT unknown id status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestDeleteTodo(t *testing.T) {
	r := newTestRouter([]Todo{{ID: "1", Title: "delete me", Complete: false}})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/todos/1", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("DELETE /todos/1 status = %d, want %d", w.Code, http.StatusOK)
	}
}

// toggleForTest drives updateTodoStatus through the router for the given id.
func (todoList *TodoList) toggleForTest(t *testing.T, id string) {
	t.Helper()
	r := gin.New()
	r.PUT("/todos/:id", todoList.updateTodoStatus)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/todos/"+id, nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("toggle id %q status = %d, want %d", id, w.Code, http.StatusOK)
	}
}
