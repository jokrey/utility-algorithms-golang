package wsclientable

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

// Default Authenticate Function, which permits all connections that specify a user=<userName> in the params
// for example: y.x.com/route?user=test.
func AuthenticateUserPermitAll() func(initialParams url.Values) (string, error) {
	return func(initialParams url.Values) (string, error) {
		// initialParams are in url: y.x.com/signaling?user=test
		userName := initialParams.Get("user")
		if len(userName) == 0 {
			return "", MissingURLFieldError{MissingFieldName: "user"}
		}

		return userName, nil
	}
}

// Returns complete url for room connection (baseurl example: http://dns.com:8080/route)
// Same as http://dns.com:8080/route?user=<userID>
func UrlWithParamsForUserConnection(baseurl, userID string) string {
	if !strings.HasSuffix(baseurl, "?") {
		baseurl += "?"
	}
	return baseurl + "&user=" + userID
}

// See Connect and UrlWithParamsForRoomConnection
func ConnectAs(baseurl, userID string) (*ClientConnection, error) {
	return Connect(UrlWithParamsForUserConnection(baseurl, userID))
}

// Default Authenticate Function, which permits all connections that specify a user=<userName> and password=<pw>
// for example: y.x.com/route?user=test&password=123456bestpw
// the combination is checked against correctCombination(userName, password), which should check a database or something
// NOTE: Use ssl. For clear(!) reasons.
//goland:noinspection GoUnusedExportedFunction
func AuthenticateUserPermitPassword(correctCombination func(string, string) bool) func(initialParams url.Values) (string, error) {
	return func(initialParams url.Values) (string, error) {
		userName := initialParams.Get("user")
		password := initialParams.Get("password")

		if len(userName) == 0 {
			return "", MissingURLFieldError{MissingFieldName: "user"}
		}
		if len(password) == 0 {
			return "", MissingURLFieldError{MissingFieldName: "password"}
		}
		if correctCombination(userName, password) {
			return "", AuthenticationError{Reason: "userName password combination check failed"}
		}

		return userName, nil
	}
}

//Checks for every connection whether the user is not already connected, the room exists and the user is allowed in that room
func AuthenticateRoomUserPermitAllowed(rooms RoomControllerI) func(initialParams url.Values) (string, error) {
	return func(initialParams url.Values) (string, error) {
		roomID := initialParams.Get("room")
		if len(roomID) == 0 {
			return "", MissingURLFieldError{MissingFieldName: "room"}
		}
		userID := initialParams.Get("user")
		if len(userID) == 0 {
			return "", MissingURLFieldError{MissingFieldName: "user"}
		}

		if rooms.IsConnected(roomID, userID) {
			return "", AuthenticationError{Reason: "User(" + userID + ") already connected in room: " + roomID}
		}

		room := rooms.GetRoom(roomID)
		if room == nil {
			return "", AuthenticationError{Reason: "Could not find room: " + roomID}
		}
		if !room.IsAllowed(userID) {
			return "", AuthenticationError{Reason: "User(" + userID + ") not currently allowed in room: " + roomID}
		}

		convertedClientID := RoomIDAndUserIDToClientConnectionIDString(roomID, userID)
		return convertedClientID, nil
	}
}

// Returns complete url for room connection (baseurl example: http://dns.com:8080/route)
// Same as http://dns.com:8080/route?user=<userID>&room=<roomID>
func UrlWithParamsForRoomConnection(baseurl, roomID, userID string) string {
	if !strings.HasSuffix(baseurl, "?") {
		baseurl += "?"
	}
	return baseurl + "room=" + roomID + "&user=" + userID
}

// See Connect and UrlWithParamsForRoomConnection
func ConnectToRoom(baseurl, roomID, userID string) (*ClientConnection, error) {
	return Connect(UrlWithParamsForRoomConnection(baseurl, roomID, userID))
}

// yes, I know the following is ugly, but golang is seriously missing support for generics and this is kinda mostly ok.
func ConnectionIDStringToRoomIDAndUserID(connectionID string) (string, string, error) {
	var connectionIDJSON map[string]string
	if err := json.Unmarshal([]byte(connectionID), &connectionIDJSON); err != nil {
		return "", "", fmt.Errorf("cannot unmarshal: %v", err.Error())
	}
	return connectionIDJSON["r"], connectionIDJSON["u"], nil
}
func RoomIDAndUserIDToClientConnectionIDString(roomID, userID string) string {
	return "{\"r\":\"" + roomID + "\", \"u\":\"" + userID + "\"}"
}
