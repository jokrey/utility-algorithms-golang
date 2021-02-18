package wsclientable

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"
)

//See UrlWithParamsForRoomConnection, StartChatClientLoop
func StartChatClientLoopAs(baseurl, userID string) error {
	return StartChatClientLoop(UrlWithParamsForUserConnection(baseurl, userID), "message")
}

//See UrlWithParamsForRoomConnection, StartChatClientLoop
func StartChatClientLoopForRoom(baseurl, roomID, userID string) error {
	return StartChatClientLoop(UrlWithParamsForRoomConnection(baseurl, roomID, userID), "message")
}

//Connect to url. Listen to os.stdin for messages to send to remote
func StartChatClientLoop(url string, chatMType string) error {
	client, err := Connect(url)
	if err != nil {
		return err
	}
	log.Println("Connected to ", url)

	clientClosed := make(chan ClientCloseMessage)
	go func() {
		c, r := client.ListenLoop(MessageHandlers{
			chatMType: func(mType string, client ClientConnection, message map[string]interface{}) {
				log.Println("Received MESSAGE from(", message["from"], "): ", message["text"])
			},
		})
		clientClosed <- ClientCloseMessage{c, r}
	}()

	stdin := ReadLines()

	for {
		select {
		case newLine := <-stdin:
			log.Println("Entered: ", newLine) //Writing to Stdout
			if newLine == "exit" || newLine == "close" {
				_ = client.Close()
			} else if newLine == "help" {
				log.Println("Commands:")
				log.Println("help - prints these words")
				log.Println("exit/close - closes the client and this function returns")
				log.Println("send <to> <message> - send <message> to <to> (separated by spaces, <to> cannot contain space, <message> can)")
			} else if strings.HasPrefix(newLine, "send ") {
				newLine = newLine[5:] //cut off the command prefix
				toCutOff := strings.Index(newLine, " ")
				if toCutOff >= 0 && toCutOff < len(newLine) {
					to := newLine[:toCutOff]
					message := newLine[toCutOff+1:]
					e := client.SendTyped(chatMType, "{\"to\":\""+to+"\", \"text\":\""+message+"\"}")
					if e != nil {
						log.Println("Error sending: ", e)
					} else {
						log.Printf("Send to(%v) message %v\n", to, message)
					}
				} else {
					log.Println("Invalid args to send")
				}
			} else {
				log.Println("Unrecognised command")
			}
		case clientCloseMessage := <-clientClosed:
			if clientCloseMessage.code != 1000 {
				return fmt.Errorf("unexpected client closing: code=%v, reason=%v", clientCloseMessage.code, clientCloseMessage.text)
			} else {
				return nil
			}
		}
	}
}

func ReadLines() <-chan string {
	lines := make(chan string)
	go func() {
		defer close(lines)
		scan := bufio.NewScanner(os.Stdin)
		for scan.Scan() {
			s := scan.Text()
			lines <- s
		}
	}()
	return lines
}
func ReadOneLine() (string, error) {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		return scanner.Text(), nil
	}
	return "", scanner.Err()
}
