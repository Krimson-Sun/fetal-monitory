package generators

import (
	"fetal-emulator/internal/config"
	"math/rand"
	"sync"
	"time"
)

type fhrGenerator struct {
	rand        *rand.Rand
	config      config.FHRConfig
	baseValue   int
	variability int
	stats       GeneratorStats
	mu          sync.RWMutex
}

func NewFHRGenerator(cfg config.FHRConfig) FHRGenerator {
	return &fhrGenerator{
		rand:        rand.New(rand.NewSource(time.Now().UnixNano())),
		config:      cfg,
		baseValue:   cfg.BaseValue,
		variability: cfg.Variability,
		stats: GeneratorStats{
			MinValueGenerated: 300, // Начальное большое значение
			MaxValueGenerated: 0,   // Начальное маленькое значение
		},
	}
}

func (g *fhrGenerator) NextValue() int {
	g.mu.Lock()
	defer g.mu.Unlock()

	variation := g.rand.Intn(g.variability*2+1) - g.variability
	value := g.baseValue + variation

	// Ограничиваем физиологическими пределами
	if value < g.config.MinValue {
		value = g.config.MinValue
	}
	if value > g.config.MaxValue {
		value = g.config.MaxValue
	}

	// Обновляем статистику
	g.updateStats(value)

	return value
}

// SetBaseValue устанавливает базовое значение пульса
func (g *fhrGenerator) SetBaseValue(value int) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if value < g.config.MinValue {
		value = g.config.MinValue
	}
	if value > g.config.MaxValue {
		value = g.config.MaxValue
	}
	g.baseValue = value
}

// GetVariability возвращает текущую вариабельность
func (g *fhrGenerator) GetVariability() int {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.variability
}

// SetVariability устанавливает вариабельность
func (g *fhrGenerator) SetVariability(variability int) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if variability < 0 {
		variability = 0
	}
	if variability > 50 { // Максимальная вариабельность
		variability = 50
	}
	g.variability = variability
}

// Validate проверяет корректность работы генератора
func (g *fhrGenerator) Validate() error {
	value := g.NextValue()
	if value < g.config.MinValue || value > g.config.MaxValue {
		return ErrInvalidFHRValue
	}
	return nil
}

// Reset сбрасывает состояние генератора
func (g *fhrGenerator) Reset() {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.baseValue = g.config.BaseValue
	g.variability = g.config.Variability
	g.stats = GeneratorStats{
		MinValueGenerated: 300,
		MaxValueGenerated: 0,
	}
}

// Seed устанавливает seed для случайного генератора
func (g *fhrGenerator) Seed(seed int64) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.rand.Seed(seed)
}

// GetStats возвращает статистику работы генератора
func (g *fhrGenerator) GetStats() GeneratorStats {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.stats
}

// updateStats обновляет статистику генератора
func (g *fhrGenerator) updateStats(value int) {
	g.stats.TotalValuesGenerated++
	g.stats.LastValue = value

	if value < g.stats.MinValueGenerated {
		g.stats.MinValueGenerated = value
	}
	if value > g.stats.MaxValueGenerated {
		g.stats.MaxValueGenerated = value
	}

	// Пересчитываем среднее значение
	totalSum := g.stats.AverageValue * float64(g.stats.TotalValuesGenerated-1)
	g.stats.AverageValue = (totalSum + float64(value)) / float64(g.stats.TotalValuesGenerated)
}
