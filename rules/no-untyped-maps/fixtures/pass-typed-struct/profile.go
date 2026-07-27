package profile

import "encoding/json"

type User struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Nickname  string `json:"nickname"`
}

func ParseUser(raw []byte) (User, error) {
	var user User
	err := json.Unmarshal(raw, &user)
	return user, err
}

func DisplayName(user User) string {
	if user.Nickname != "" {
		return user.Nickname
	}
	return user.FirstName + " " + user.LastName
}
