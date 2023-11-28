package app

import (
	"fmt"
)

const periodicTime = 15

func (app *App) runPeriodic() error {
	if err := app.SetupDRS(); err != nil {
		return fmt.Errorf("setup DRS: %w", err)
	}

	if err := app.SetupDRSQemu(); err != nil {
		return fmt.Errorf("setup DRS QEMU: %w", err)
	}

	return nil
}
