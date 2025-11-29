package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

func MustJson(r *http.Request) bool {
	result := false
	if strings.Contains(r.Header.Get("Accept"), "application/json") {
		result = true
	} else if strings.Contains(r.Header.Get("Content-Type"), "application/json") {
		result = true
	} else if r.URL.Query().Get("format") == "json" {
		result = true
	}
	return result
}

func HttpInput(r *http.Request) (map[string]interface{}, error) {

	bodyBytes, err := io.ReadAll(r.Body)
	r.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	defer r.Body.Close()
	if err != nil {
		return nil, err
	}

	var params = map[string]interface{}{}
	content := strings.TrimSpace(string(bodyBytes))
	if strings.HasPrefix(content, "{") && strings.HasSuffix(content, "}") {
		if err := json.Unmarshal([]byte(content), &params); err != nil {
			return nil, err
		}
	}

	if err := r.ParseForm(); err != nil {
		return nil, err
	}
	for key, arr := range r.Form {
		params[key] = arr[0]
	}

	return params, nil
}

func HttpOutput(r *http.Request, w http.ResponseWriter, args ...interface{}) {

	var (
		code int    = http.StatusOK
		msg  string = "OK"
		data interface{}
	)

	for _, arg := range args {
		switch v := arg.(type) {
		case int:
			code = v
		case string:
			msg = v
		default:
			data = interface{}(v)
		}
	}

	if !MustJson(r) {
		if code != http.StatusOK {
			http.Error(w, msg, code)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(msg))
		return
	}

	result, err := json.Marshal(map[string]interface{}{"code": code, "message": msg, "data": data})
	if err != nil {
		w.Write([]byte("Server error"))
		return
	}

	if r.URL.Query().Has("callback") {
		callback := r.URL.Query().Get("callback")
		result = []byte(fmt.Sprintf("%s(%s)", callback, string(result)))
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(result)
}
