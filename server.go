package main

import (
	"log"
	"net/http"
	"time"

	r "github.com/dancannon/gorethink"
	"github.com/gorilla/mux"
)

var (
	router  *mux.Router
	session *r.Session
)

func init() {
	var err error

	session, err = r.Connect(r.ConnectOpts{
		Address:  "rethinkdb-internal:28015",
		Database: "todo",
		MaxOpen:  40,
	})
	if err != nil {
		log.Fatalln(err.Error())
	}

	err = r.DBCreate("todo").Exec(session)
	if err != nil {
		log.Fatalln(err.Error())
		return
	}
	err = r.TableCreate("items").Exec(session)
	if err != nil {
		log.Fatalln(err.Error())
		return
	}
}

func NewServer(addr string) *http.Server {
	// Setup router
	router = initRouting()

	// Create and start server
	return &http.Server{
		Addr:    addr,
		Handler: router,
	}
}

func StartServer(server *http.Server) {
	log.Println("Starting server")
	err := server.ListenAndServe()
	if err != nil {
		log.Fatalln("Error: %v", err)
	}
}

func initRouting() *mux.Router {

	r := mux.NewRouter()

	r.HandleFunc("/", indexHandler)
	r.HandleFunc("/all", indexHandler)
	r.HandleFunc("/active", activeIndexHandler)
	r.HandleFunc("/completed", completedIndexHandler)
	r.HandleFunc("/new", newHandler)
	r.HandleFunc("/toggle/{id}", toggleHandler)
	r.HandleFunc("/delete/{id}", deleteHandler)
	r.HandleFunc("/clear", clearHandler)

	// Add handler for websocket server
	r.Handle("/ws/all", newChangesHandler(allChanges))
	r.Handle("/ws/active", newChangesHandler(activeChanges))
	r.Handle("/ws/completed", newChangesHandler(completedChanges))

	// Add handler for static files
	r.PathPrefix("/").Handler(http.FileServer(http.Dir("static")))

	return r
}

func newChangesHandler(fn func(chan interface{})) http.HandlerFunc {
	h := newHub()
	go h.run()

	fn(h.broadcast)

	return wsHandler(h)
}

// Handlers

func indexHandler(w http.ResponseWriter, req *http.Request) {
	items := []TodoItem{}

	// Fetch all the items from the database
	res, err := r.Table("items").OrderBy(r.Asc("Created")).Run(session)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = res.All(&items)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	renderTemplate(w, "index", map[string]interface{}{
		"Items": items,
		"Route": "all",
	})
}

func activeIndexHandler(w http.ResponseWriter, req *http.Request) {
	items := []TodoItem{}

	// Fetch all the items from the database
	query := r.Table("items").Filter(r.Row.Field("Status").Eq("active"))
	query = query.OrderBy(r.Asc("Created"))
	res, err := query.Run(session)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	err = res.All(&items)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	renderTemplate(w, "index", map[string]interface{}{
		"Items": items,
		"Route": "active",
	})
}

func completedIndexHandler(w http.ResponseWriter, req *http.Request) {
	items := []TodoItem{}

	// Fetch all the items from the database
	query := r.Table("items").Filter(r.Row.Field("Status").Eq("complete"))
	query = query.OrderBy(r.Asc("Created"))
	res, err := query.Run(session)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	err = res.All(&items)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	renderTemplate(w, "index", map[string]interface{}{
		"Items": items,
		"Route": "completed",
	})
}

func newHandler(w http.ResponseWriter, req *http.Request) {
	// Create the item
	item := NewTodoItem(req.PostFormValue("text"))
	item.Created = time.Now()

	// Insert the new item into the database
	_, err := r.Table("items").Insert(item).RunWrite(session)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, req, "/", http.StatusFound)
}

func toggleHandler(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	id := vars["id"]
	if id == "" {
		http.NotFound(w, req)
		return
	}

	// Check that the item exists
	res, err := r.Table("items").Get(id).Run(session)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if res.IsNil() {
		http.NotFound(w, req)
		return
	}

	// Toggle the item
	_, err = r.Table("items").Get(id).Update(map[string]interface{}{"Status": r.Branch(
		r.Row.Field("Status").Eq("active"),
		"complete",
		"active",
	)}).RunWrite(session)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, req, "/", http.StatusFound)
}

func deleteHandler(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	id := vars["id"]
	if id == "" {
		http.NotFound(w, req)
		return
	}

	// Check that the item exists
	res, err := r.Table("items").Get(id).Run(session)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if res.IsNil() {
		http.NotFound(w, req)
		return
	}

	// Delete the item
	_, err = r.Table("items").Get(id).Delete().RunWrite(session)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, req, "/", http.StatusFound)
}

func clearHandler(w http.ResponseWriter, req *http.Request) {
	// Delete all completed items
	_, err := r.Table("items").Filter(
		r.Row.Field("Status").Eq("complete"),
	).Delete().RunWrite(session)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, req, "/", http.StatusFound)
}
