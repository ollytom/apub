package lemmy

import (
	"encoding/json"
	"fmt"
	"net/http"
)

func (c *Client) Login(name, password string) error {
	if !c.ready {
		if err := c.init(); err != nil {
			return err
		}
	}

	params := map[string]interface{}{
		"username_or_email": name,
		"password":          password,
	}
	resp, err := c.post("/user/login", params)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("remote status %s: %w", resp.Status, decodeError(resp.Body))
	}

	var response struct {
		JWT string
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return fmt.Errorf("decode login response: %w", err)
	}
	c.authToken = response.JWT
	return nil
}

func (c *Client) Authenticated() bool {
	return c.authToken != ""
}
