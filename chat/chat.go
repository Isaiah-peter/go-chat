package chat

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"real-chat-app/utils"

	"github.com/gorilla/websocket"
)

type Chat struct {
	users    map[string]*User
	messages chan *Message
	join     chan *User
	leave    chan *User
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  512,
	WriteBufferSize: 512,
	CheckOrigin: func(r *http.Request) bool {
		log.Printf("%s %s %s %v\n", r.Method, r.Host, r.RequestURI, r.Proto)
		return r.Method == http.MethodGet
	},
}

func (c *Chat) Handler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Fatal("Error on websocket connection:", err.Error())
	}
	key := r.URL.Query()
	username := key.Get("username")
	if strings.TrimSpace(username) == "" {
		username = fmt.Sprintf("anom-%d", utils.GetRandomI64())
	}

	user := &User{
		UserName: username,
		Conn:     conn,
		Global:   c,
	}

	c.join <- user

	user.Read()
}

func (c *Chat) Run() {
	for {
		select {
		case user := <-c.join:
			c.add(user)
		case user := <-c.leave:
			c.disconnect(user)
		case message := <-c.messages:
			c.broadcast(message)
		}
	}
}

func (c *Chat) add(user *User) {
	if _, ok := c.users[user.UserName]; !ok {
		c.users[user.UserName] = user
		body := fmt.Sprintf("%s join the chat", user.UserName)
		c.broadcast(NewMessage(body, "Server"))
	}
}

func (c *Chat) disconnect(user *User) {
	if _, ok := c.users[user.UserName]; ok {
		defer user.Conn.Close()
		delete(c.users, user.UserName)
		body := fmt.Sprintf("%s left the chat", user.UserName)
		c.broadcast(NewMessage(body, "Server"))
	}
}

func (c *Chat) broadcast(message *Message) {
	log.Printf("broadcast message %v\n", message)
	for _, user := range c.users {
		user.Write(message)
	}
}

func Start(port string) {
	log.Printf("Chat listening on http://localhost:%s\n", port)

	c := &Chat{
		users:    make(map[string]*User),
		messages: make(chan *Message),
		join:     make(chan *User),
		leave:    make(chan *User),
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Welcome to go webchat"))
	})
	http.HandleFunc("/chat", c.Handler)

	go c.Run()

	log.Fatal(http.ListenAndServe(port, nil))
}
