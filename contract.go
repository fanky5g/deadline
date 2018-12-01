package deadline

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

// Contract represents a unit entity to keep in check
type Contract struct {
	ID       string `json:"id"`
	TimeOut  int64  `json:"timeout"`
	Listener string `json:"listener"`
}

// ExecOnTimeout calls http endpoint listener after contract times out
func (c Contract) ExecOnTimeout() error {
	// get http client
	data := struct {
		ID string `json:"id"`
	}{c.ID}

	body := &bytes.Buffer{}
	err := json.NewEncoder(body).Encode(data)
	if err != nil {
		return err
	}

	client := GetClient()
	req, _ := http.NewRequest("POST", c.Listener, body)

	res, err := client.Do(req)
	if err != nil {
		return err
	}

	if res.StatusCode >= 200 && res.StatusCode < 300 {
		return nil
	}

	// try display error response
	defer res.Body.Close()
	errorResponse, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil
	}

	fmt.Println("Error!!! Listener returned: %v", string(errorResponse))
	return nil
}
