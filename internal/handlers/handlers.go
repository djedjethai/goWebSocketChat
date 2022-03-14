package handlers

import (
	"fmt"
	"log"
	"net/http"
	"sort"

	"github.com/CloudyKit/jet/v6"
	"github.com/gorilla/websocket"
)

// this is a channel who will only accept data of type WsPayload
var wsChan = make(chan WsPayload)

var clients = make(map[WebSocketConnection]string)

var views = jet.NewSet(
	jet.NewOSFileSystemLoader("./html"),
	jet.InDevelopmentMode(),
)

var upgradeConnection = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

func Home(w http.ResponseWriter, r *http.Request) {
	err := renderPage(w, "home.jet", nil)
	if err != nil {
		log.Println(err)
	}

}

// wrapper to hold the actual websocket connection
// this is predefined by Gorilla
type WebSocketConnection struct {
	*websocket.Conn
}

// WsJsonResponse defines the response sent back from websocket
type WsJsonResponse struct {
	Action         string   `json:"action"`
	Message        string   `json:"message"`
	MessageType    string   `json:"message_type"`
	ConnectedUsers []string `json:"connected_users"`
}

// WsPayload
type WsPayload struct {
	Action   string              `json:"action"`
	Username string              `json:"username"`
	Message  string              `json:"message"`
	Conn     WebSocketConnection `json:"-"`
}

func WsEndpoint(w http.ResponseWriter, r *http.Request) {
	// first upgrade the connection
	ws, err := upgradeConnection.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
	}

	log.Println("Client connected to endpoint")

	var response WsJsonResponse
	response.Message = `<em><small>Connected to server</small></em>`

	// when someonw connect to the server with the websocket connection
	// add them to the map[WebSocketConnection]
	conn := WebSocketConnection{Conn: ws}
	clients[conn] = ""

	err = ws.WriteJSON(response)
	if err != nil {
		log.Println(err)
	}

	go ListenForWs(&conn)
}

func ListenForWs(conn *WebSocketConnection) {
	// make sure that whenever this go routine stop executing
	// for whatever reason, it comes back, this following func will run
	defer func() {
		if r := recover(); r != nil {
			log.Println("Error", fmt.Sprintf("%v", r))
		}
	}()

	// we know we gonna get a payload periodically
	var payload WsPayload

	for {
		err := conn.ReadJSON(&payload)
		if err != nil {
			// do noting as there is no payload
		} else {
			payload.Conn = *conn
			wsChan <- payload
		}
	}
}

func ListenToWsChannel() {
	var response WsJsonResponse

	for {
		e := <-wsChan
		switch e.Action {
		case "username":
			// get a list of all users and send it back via broadcast
			clients[e.Conn] = e.Username
			users := getUserList()

			response.Action = "list_users"
			response.ConnectedUsers = users
		case "broadcast":
			response.Action = "broadcast"
			response.Message = fmt.Sprintf("<strong>%s</strong>: %s", e.Username, e.Message)
			// users := getUserList()
			// response.ConnectedUsers = users
			// fmt.Println("read the message: ", e.Message)
		case "left":
			response.Action = "list_users"
			delete(clients, e.Conn)
			users := getUserList()
			response.ConnectedUsers = users
		}
		// response.Action = "Got here"
		// response.Message = fmt.Sprintf("Some message and action was %s", e.Action)
		// // then need to broadcast the datas to all users
		broadcastToAll(response)
	}
}

func getUserList() []string {
	var userList []string
	for _, x := range clients {
		if x != "" {
			userList = append(userList, x)
		}
	}
	sort.Strings(userList)
	return userList
}

// broadcast datas to all user
func broadcastToAll(response WsJsonResponse) {
	for client := range clients {
		err := client.WriteJSON(response)
		if err != nil {
			log.Println("websocket err")
			_ = client.Close()
			delete(clients, client)
		}
	}
}

func renderPage(w http.ResponseWriter, tmpl string, data jet.VarMap) error {
	view, err := views.GetTemplate(tmpl)
	if err != nil {
		log.Println(err)
		return err
	}

	err = view.Execute(w, data, nil)
	if err != nil {
		log.Println(err)
		return err
	}

	return nil
}
