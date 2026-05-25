package main

import (
	"embed"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"sync"

	"github.com/gorilla/websocket"
)

//go:embed all:web
var webFS embed.FS

type session struct {
	conn *websocket.Conn
	mu   sync.Mutex
}

var (
	sessions sync.Map
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}
)

func handleSession(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("token")
	if token == "" {
		http.Error(w, "missing token", http.StatusBadRequest)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("upgrade %s: %v", token, err)
		return
	}
	defer conn.Close()

	s := &session{conn: conn}
	if _, loaded := sessions.LoadOrStore(token, s); loaded {
		conn.WriteMessage(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "token already in use"))
		return
	}
	defer sessions.Delete(token)

	log.Printf("session open: %s", token)

	for {
		if _, _, err := conn.NextReader(); err != nil {
			break
		}
	}

	log.Printf("session closed: %s", token)
}

func handleUpload(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("token")
	if token == "" {
		http.Error(w, "missing token", http.StatusBadRequest)
		return
	}

	val, ok := sessions.Load(token)
	if !ok {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}
	s := val.(*session)

	s.mu.Lock()
	defer s.mu.Unlock()

	wc, err := s.conn.NextWriter(websocket.BinaryMessage)
	if err != nil {
		http.Error(w, "session unavailable", http.StatusGone)
		return
	}

	n, err := io.Copy(wc, r.Body)
	wc.Close()

	if err != nil {
		log.Printf("upload %s: copy error after %d bytes: %v", token, n, err)
		http.Error(w, "transfer error", http.StatusInternalServerError)
		return
	}

	log.Printf("upload %s: sent %d bytes", token, n)
	w.WriteHeader(http.StatusNoContent)
}

func handlePing(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte("pong"))
}

func main() {
	webContent, err := fs.Sub(webFS, "web")
	if err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()
	if os.Getenv("SERVE_UI") == "true" {
		mux.Handle("GET /", http.FileServer(http.FS(webContent)))
	}
	mux.HandleFunc("GET /ping", handlePing)
	mux.HandleFunc("GET /session/{token}", handleSession)
	mux.HandleFunc("POST /upload/{token}", handleUpload)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	addr := ":" + port
	log.Printf("relay listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("error listening and serving: %v", err)
	}
}
