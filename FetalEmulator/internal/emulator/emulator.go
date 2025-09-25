package emulator

import (
	"context"
	"fetal-emulator/internal/config"
	"fetal-emulator/internal/generators"
	"fetal-emulator/internal/models"
	"fetal-emulator/internal/senders"
	"log"
	"time"
)

type Emulator struct {
	fhrGen  generators.FHRGenerator
	tocoGen generators.TOCOGenerator
	sender  senders.DataSender
	config  config.EmulatorConfig
}

func NewEmulator(
	fhrGen generators.FHRGenerator,
	tocoGen generators.TOCOGenerator,
	sender senders.DataSender,
	cfg config.EmulatorConfig,
) *Emulator {
	return &Emulator{
		fhrGen:  fhrGen,
		tocoGen: tocoGen,
		sender:  sender,
		config:  cfg,
	}
}

func (e *Emulator) Run() error {
	ctx, cancel := context.WithTimeout(context.Background(), e.config.Duration)
	defer cancel()

	ticker := time.NewTicker(e.config.SampleRate)
	defer ticker.Stop()

	log.Printf("Starting emulator for %v with rate %v", e.config.Duration, e.config.SampleRate)

	for {
		select {
		case <-ticker.C:
			data := models.DataPoint{
				Timestamp: time.Now(),
				FHR:       e.fhrGen.NextValue(),
				TOCO:      e.tocoGen.NextValue(),
			}

			if err := e.sender.Send(data); err != nil {
				log.Printf("Send error: %v", err)
			}

		case <-ctx.Done():
			log.Println("Emulator stopped")
			return nil
		}
	}
}
