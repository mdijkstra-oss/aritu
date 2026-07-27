package profile

func DisplayName(user map[string]string) string {
	if user["nickname"] != "" {
		return user["nickname"]
	}
	return user["first_name"] + " " + user["last_name"]
}

func Initials(user map[string]string) string {
	initials := ""
	if user["first_name"] != "" {
		initials += user["first_name"][:1]
	}
	if user["last_name"] != "" {
		initials += user["last_name"][:1]
	}
	return initials
}
