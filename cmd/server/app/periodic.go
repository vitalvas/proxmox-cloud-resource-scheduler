package app

import (
	"context"
	"fmt"
)

func (app *App) runPeriodic(_ context.Context) error {
	lock, err := app.consul.GetLock("periodic")
	if err != nil {
		return err
	}

	if _, err := lock.Lock(nil); err != nil {
		return err
	}

	defer lock.Unlock()

	if err := app.SetupDRS(); err != nil {
		return fmt.Errorf("setup DRS: %w", err)
	}

	if err := app.SetupDRSQemu(); err != nil {
		return fmt.Errorf("setup DRS QEMU: %w", err)
	}

	return nil
}
