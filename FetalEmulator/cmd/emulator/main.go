package main

import (
	"fetal-emulator/internal/config"
	"fetal-emulator/internal/emulator"
	"fetal-emulator/internal/generators"
	"fetal-emulator/internal/senders"
	"log"
)

func main() {
	// Загрузка конфигурации
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Ошибка загрузки конфигурации: %v", err)
	}

	// Инициализация компонентов
	fhrGen := generators.NewFHRGenerator(cfg.FHR)
	tocoGen := generators.NewTOCOGenerator(cfg.TOCO)
	sender, err := senders.NewFileSender(cfg.Output.FilePath)
	if err != nil {
		log.Fatalf("Ошибка инициализации отправителя: %v", err)
	}
	defer sender.Close()

	// Создание и запуск эмулятора
	emulator := emulator.NewEmulator(fhrGen, tocoGen, sender, cfg.Emulator)
	if err := emulator.Run(); err != nil {
		log.Fatalf("Ошибка работы эмулятора: %v", err)
	}
}
