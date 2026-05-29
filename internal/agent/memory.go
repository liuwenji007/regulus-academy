package agent

import (
	"github.com/regulus-academy/regulus-academy/internal/storage"
)

// PickReinforceConcept 选一个待强化概念
func PickReinforceConcept(store *storage.Store, userID, domainID string) *string {
	list, err := store.ListMistakesForReinforce(userID, domainID, 1)
	if err != nil || len(list) == 0 {
		return nil
	}
	c := list[0].Concept
	return &c
}
