package roster

type Player struct {
	Name   string
	Active bool
	Score  int
}

func actives(players []Player) []Player {
	kept := make([]Player, 0, len(players))
	for _, player := range players {
		if player.Active {
			kept = append(kept, player)
		}
	}
	return kept
}

func best(players []Player) Player {
	top := Player{}
	for _, player := range actives(players) {
		if player.Score > top.Score {
			top = player
		}
	}
	return top
}
