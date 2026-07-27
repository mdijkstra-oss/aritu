package scenario

func IsExpired(ageDays int) bool {
	return ageDays > retentionDays
}

const retentionDays = 30
