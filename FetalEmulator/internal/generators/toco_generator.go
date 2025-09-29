package generators

import (
	"fetal-emulator/internal/config"
	"math/rand"
	"sync"
	"time"
)

type tocoGenerator struct {
	rand             *rand.Rand
	config           config.TOCOConfig
	lastContraction  time.Time
	inContraction    bool
	contractionStart time.Time
	contractionValue int
	mu               sync.RWMutex
}

func NewTOCOGenerator(cfg config.TOCOConfig) TOCOGenerator {
	return &tocoGenerator{
		rand:            rand.New(rand.NewSource(time.Now().UnixNano())),
		config:          cfg,
		lastContraction: time.Now(),
	}
}

func (g *tocoGenerator) NextValue() int {
	g.mu.Lock()
	defer g.mu.Unlock()

	now := time.Now()

	// Если не в схватке, проверяем не пора ли начать новую
	if !g.inContraction {
		timeSinceLast := now.Sub(g.lastContraction)
		minInterval := g.config.MinContractionInterval
		maxInterval := g.config.MaxContractionInterval

		// Случайный интервал между схватками
		if timeSinceLast > minInterval {
			probability := float64(timeSinceLast-minInterval) / float64(maxInterval-minInterval)
			if probability > 1.0 {
				probability = 1.0
			}

			if g.rand.Float64() < probability {
				g.inContraction = true
				g.contractionStart = now
				g.contractionValue = 0
			}
		}

		// Если не в схватке - возвращаем базовый уровень
		if !g.inContraction {
			return g.rand.Intn(6) // 0-5
		}
	}

	// Если в схватке - вычисляем текущее значение
	if g.inContraction {
		elapsed := now.Sub(g.contractionStart)
		contractionDuration := g.config.ContractionDuration

		// Разбиваем схватку на фазы: подъем, пик, спад
		phaseDuration := contractionDuration / 3

		if elapsed < phaseDuration {
			// Фаза подъема (0 → PeakIntensity)
			progress := float64(elapsed) / float64(phaseDuration)
			g.contractionValue = int(progress * float64(g.config.PeakIntensity))
		} else if elapsed < 2*phaseDuration {
			// Фаза пика (PeakIntensity)
			g.contractionValue = g.config.PeakIntensity
		} else if elapsed < contractionDuration {
			// Фаза спада (PeakIntensity → 0)
			progress := float64(elapsed-2*phaseDuration) / float64(phaseDuration)
			g.contractionValue = g.config.PeakIntensity - int(progress*float64(g.config.PeakIntensity))
		} else {
			// Схватка закончилась
			g.inContraction = false
			g.lastContraction = now
			g.contractionValue = 0
		}

		return g.contractionValue
	}

	return g.rand.Intn(6)
}

func (g *tocoGenerator) GetContractionStatus() bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.inContraction
}

func (g *tocoGenerator) GetContractionProgress() int {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if !g.inContraction {
		return 0
	}

	elapsed := time.Since(g.contractionStart)
	totalDuration := g.config.ContractionDuration

	if elapsed >= totalDuration {
		return 100
	}

	progress := int(float64(elapsed) / float64(totalDuration) * 100)
	if progress > 100 {
		return 100
	}
	return progress
}

func (g *tocoGenerator) ForceContraction() {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.inContraction = true
	g.contractionStart = time.Now()
	g.contractionValue = 0
}

func (g *tocoGenerator) Reset() {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.inContraction = false
	g.lastContraction = time.Now()
	g.contractionValue = 0
}

func (g *tocoGenerator) Validate() error {
	value := g.NextValue()
	if value < 0 || value > 100 {
		return ErrInvalidTOCOValue
	}
	return nil
}

func (g *tocoGenerator) Seed(seed int64) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.rand.Seed(seed)
}
