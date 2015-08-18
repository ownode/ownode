package services

import (
	"github.com/ownode/models"
)

type ByObjectBalance []models.Object

func (s ByObjectBalance) Len() int {
    return len(s)
}

func (s ByObjectBalance) Swap(i, j int) {
    s[i], s[j] = s[j], s[i]
}

func (s ByObjectBalance) Less(i, j int) bool {
    return s[i].Balance > s[j].Balance
}
