package service

import (
	"context"

	"ride-sharing/shared/auth"
)

// AccountBonusAdapter credits bonus points on the drivers Postgres table.
type AccountBonusAdapter struct {
	Store *auth.AccountStore
}

func (a *AccountBonusAdapter) AddBonusPoints(ctx context.Context, driverID string, points int) error {
	return a.Store.AddBonusPoints(ctx, driverID, points)
}

func (a *AccountBonusAdapter) GetBonusPoints(ctx context.Context, driverID string) (int, error) {
	acct, err := a.Store.GetByID(ctx, driverID)
	if err != nil {
		return 0, err
	}
	return acct.BonusPoints, nil
}
