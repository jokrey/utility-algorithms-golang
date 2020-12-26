package wsclientable

import (
	"net/url"
)

// Default Authenticate Function, which permits all connections that specify a user=<userName> in the params
// for example: y.x.com/route?user=test.
func AuthenticateUserPermitAll() func(url.Values) (string, error) {
	return func(initialParams url.Values) (string, error) {
		// initialParams are in url: y.x.com/signaling?user=test
		userName := initialParams.Get("user")
		if len(userName) == 0 {
			return "", MissingURLFieldError{MissingFieldName: "user"}
		}

		return userName, nil
	}
}

// Default Authenticate Function, which permits all connections that specify a user=<userName> and password=<pw>
// for example: y.x.com/route?user=test&password=123456bestpw
// the combination is checked against correctCombination(userName, password), which should check a database or something
// NOTE: Use ssl. For clear(!) reasons.
// goland:noinspection GoUnusedExportedFunction
//goland:noinspection GoUnusedExportedFunction
func AuthenticateUserPermitPassword(correctCombination func(string, string) bool) func(url.Values) (string, error) {
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
