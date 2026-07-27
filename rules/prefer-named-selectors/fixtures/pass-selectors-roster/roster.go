package roster

type Player struct {
	Name   string
	Active bool
	Score  int
}

func FilterActive(players []Player) []Player {
	active := make([]Player, 0, len(players))
	for _, player := range players {
		if player.Active {
			active = append(active, player)
		}
	}
	return active
}

func FindTopScorer(players []Player) Player {
	top := Player{}
	for _, player := range FilterActive(players) {
		if player.Score > top.Score {
			top = player
		}
	}
	return top
}
