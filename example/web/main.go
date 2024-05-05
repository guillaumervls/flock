package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/gorilla/schema"
	"github.com/guillaumervls/flock/example/web/models"
	"github.com/guillaumervls/flock/example/web/views"
	"github.com/jackc/pgx/v5"
)

var (
	decoder = schema.NewDecoder()
	db      *pgx.Conn
)

func main() {
	var err error
	db, err = pgx.Connect(
		context.Background(),
		fmt.Sprintf("host=%s password=%s user=postgres", os.Getenv("DB_HOST"), os.Getenv("DB_PASSWORD")),
	)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v\n", err)
	}
	defer db.Close(context.Background())

	_, err = db.Exec(context.Background(), "CREATE TABLE IF NOT EXISTS todos (task TEXT)")
	if err != nil {
		log.Fatalf("Unable to create todos table: %v\n", err)
	}

	r := mux.NewRouter()

	r.HandleFunc("/", listTodos).Methods("GET")
	r.HandleFunc("/", createTodo).Methods("POST")

	http.Handle("/", r)

	log.Println("Server started on http://localhost:8000")
	log.Fatal(http.ListenAndServe(":8000", nil))
}

func listTodos(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query(context.Background(), "SELECT task FROM todos")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	todos := []models.Todo{}
	for rows.Next() {
		var todo models.Todo
		err = rows.Scan(&todo.Task)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		todos = append(todos, todo)
	}
	component := views.Index(todos)
	component.Render(context.Background(), w)
}

func createTodo(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var todo models.Todo
	err = decoder.Decode(&todo, r.PostForm)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	_, err = db.Exec(context.Background(), "INSERT INTO todos (task) VALUES ($1)", todo.Task)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}
