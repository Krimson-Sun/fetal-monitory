package csvreader

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
	"time"
)

type DataPoint struct {
	TimeSec float64
	Value   float64
}

func ReadCSVFile(filename string) ([]DataPoint, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open CSV file %s: %w", filename, err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV data: %w", err)
	}

	if len(records) < 2 {
		return nil, fmt.Errorf("CSV file %s has no data records", filename)
	}

	var dataPoints []DataPoint
	for i, record := range records[1:] { // Skip header
		if len(record) < 2 {
			return nil, fmt.Errorf("invalid record at line %d: expected 2 columns", i+2)
		}

		timeSec, err := strconv.ParseFloat(record[0], 64)
		if err != nil {
			return nil, fmt.Errorf("invalid time format at line %d: %w", i+2, err)
		}

		value, err := strconv.ParseFloat(record[1], 64)
		if err != nil {
			return nil, fmt.Errorf("invalid value format at line %d: %w", i+2, err)
		}

		dataPoints = append(dataPoints, DataPoint{
			TimeSec: timeSec,
			Value:   value,
		})
	}

	return dataPoints, nil
}

func StreamData(dataPoints []DataPoint, startTime time.Time, dataChan chan<- DataPoint) {
	for _, point := range dataPoints {
		// Calculate when to send this data point based on its time offset
		targetTime := startTime.Add(time.Duration(point.TimeSec * float64(time.Second)))
		sleepDuration := time.Until(targetTime)

		if sleepDuration > 0 {
			time.Sleep(sleepDuration)
		}

		dataChan <- point
	}
	close(dataChan)
}
