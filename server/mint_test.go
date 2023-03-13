package server

import (
	"math/rand"
	"testing"

	"github.com/reliablestaking/zombie-fight-club-server/metadata"
)

func TestLifeBar(t *testing.T) {
	for i := 1; i < 500; i++ {
		diff := rand.Intn(200-0) + 0
		winner, loser, _ := metadata.DetermineLifeBar(diff)
		if winner > 100 || loser > 100 || winner < 0 || loser < 0 {
			t.Errorf("More than 100 %d/%d for value %d", winner, loser, diff)
		}
	}
}
