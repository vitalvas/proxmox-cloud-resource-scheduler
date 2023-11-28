package app

import (
	"fmt"
)

const periodicTime = 10

func (app *App) runPeriodic() error {
	if err := app.SetupCRS(); err != nil {
		return fmt.Errorf("setup CRS: %w", err)
	}

	if err := app.SetupCRSQemu(); err != nil {
		return fmt.Errorf("setup CRS QEMU: %w", err)
	}

	return nil
}
