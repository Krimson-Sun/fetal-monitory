import { Chart } from "chart.js/auto";
import zoomPlugin from 'chartjs-plugin-zoom';

// Регистрируем плагин
Chart.register(zoomPlugin);

let hrChart;
let uterineChart;


export function resetCharts(){
      let initDate = Date.now();
      if (hrChart) hrChart.destroy();
      if (uterineChart) uterineChart.destroy();
      
      const hrCanvas = document.getElementById('hrChart').getContext('2d');
      hrChart = new Chart(hrCanvas, {
        type: 'line',
        data: {
          datasets: [
            {
              label: 'ЧСС плода',
              data: [],
              borderColor: '#0096A6',
              borderWidth: 2.5,
              pointRadius: 0,
              yAxisID: 'y'
            },
            {
              label: 'Ускорения',
              data: [],
              borderColor: '#4CAF50',
              borderWidth: 2,
              pointRadius: 5,
              pointBackgroundColor: '#4CAF50',
              pointBorderWidth: 1,
              pointStyle: 'triangle',
              yAxisID: 'y',
              showLine: false
            },
            {
              label: 'Децелерации',
              data: [],
              borderColor: '#F44336',
              borderWidth: 2,
              pointRadius: 5,
              pointBackgroundColor: '#F44336',
              pointBorderWidth: 1,
              pointStyle: 'triangle',
              yAxisID: 'y',
              showLine: false
            }
          ]
        },
        options: {
          responsive: true,
          //maintainAspectRatio: false,
          elements: {
            line: {
              tension: 0.3
            }
          },
          plugins: {
            legend: {
              display: false
            },
            tooltip: {
              enabled: false
            },
            zoom:{
              pan:{
                enabled: true,
                mode: 'x'
              },
              zoom: {
                wheel: { enabled: true },
                pinch: { enabled: true },
                mode: 'x',
                limits:{x:{min:0, max:100}}
              }
            }
          },
          scales: {
            x: {
              type: 'linear',
              position: 'bottom',
              grid: {
                display: false
              },
              ticks: {
                color: '#666',
                font: {
                  size: 12
                },
                callback: function(value) {
                  const date = new Date(value - initDate);
                  return `${date.getMinutes()}:${String(date.getSeconds()).padStart(2, '0')}`;
                }
              }
            },
            y: {
              type: 'linear',
              position: 'left',
              min: 50,
              max: 200,
              grid: {
                color: 'rgba(200, 200, 200, 0.3)',
                lineWidth: 1,
                drawBorder: false
              },
              ticks: {
                color: '#666',
                font: {
                  size: 12
                }
              },
              title: {
                display: true,
                text: 'ЧСС плода (уд/мин)',
                color: '#666',
                font: {
                  size: 13
                }
              }
            }
          }
        }
      });
      
      // Инициализация графика маточных сокращений
      const uterineCanvas = document.getElementById('uterineChart').getContext('2d');
      uterineChart = new Chart(uterineCanvas, {
        type: 'line',
        data: {
          datasets: [
            {
              label: 'Маточные сокращения',
              data: [],
              borderColor: '#0096A6',
              borderWidth: 2.5,
              pointRadius: 0,
              yAxisID: 'y'
            }
          ]
        },
        options: {
          responsive: true,
          //maintainAspectRatio: false,
          elements: {
            line: {
              tension: 0.3
            }
          },
          plugins: {
            legend: {
              display: false
            },
            tooltip: {
              enabled: false
            }
          },
          scales: {
            x: {
              type: 'linear',
              position: 'bottom',
              grid: {
                display: false
              },
              ticks: {
                color: '#666',
                font: {
                  size: 12
                },
                callback: function(value) {
                  const date = new Date(value - initDate);
                  return `${date.getMinutes()}:${String(date.getSeconds()).padStart(2, '0')}`;
                }
              }
            },
            y: {
              type: 'linear',
              position: 'left',
              min: 0,
              max: 100,
              grid: {
                color: 'rgba(200, 200, 200, 0.3)',
                lineWidth: 1,
                drawBorder: false
              },
              ticks: {
                color: '#666',
                font: {
                  size: 12
                }
              },
              title: {
                display: true,
                text: 'Маточные сокращения (%)',
                color: '#666',
                font: {
                  size: 13
                }
              }
            }
          }
        }
      });
}
  
export function updateData() {
    const now = Date.now();

    // Обновление ЧСС
    const hrValue = 130 + Math.random() * 20;
    hrChart.data.datasets[0].data.push({ x: now, y: hrValue });

    // Симуляция ускорений (зеленые треугольники)
    if (Math.random() < 0.05) {
        hrChart.data.datasets[1].data.push({ 
        x: now, 
        y: hrValue + 15,
        label: 'Ускорение'
        });
    }

    // Симуляция децелераций (красные треугольники)
    if (Math.random() < 0.03) {
        const decelType = Math.random() < 0.5 ? 'R' : 'L';
        hrChart.data.datasets[2].data.push({ 
        x: now, 
        y: hrValue - 25,
        label: `Децелерация ${decelType}`
        });
    }

    // Ограничение данных для ЧСС
    const maxPoints = 600;
    if (hrChart.data.datasets[0].data.length > maxPoints) {
        hrChart.data.datasets[0].data = hrChart.data.datasets[0].data.slice(-maxPoints);
        hrChart.data.datasets[1].data = hrChart.data.datasets[1].data.slice(-maxPoints);
        hrChart.data.datasets[2].data = hrChart.data.datasets[2].data.slice(-maxPoints);
    }

    hrChart.options.scales.x.min = hrChart.data.datasets[0].data[0].x;
    hrChart.options.scales.x.max = now;

    // Обновление маточных сокращений
    const uterineValue = Math.random() * 50;
    uterineChart.data.datasets[0].data.push({ x: now, y: uterineValue });

    // Ограничение данных для маточных сокращений
    if (uterineChart.data.datasets[0].data.length > maxPoints) {
        uterineChart.data.datasets[0].data = uterineChart.data.datasets[0].data.slice(-maxPoints);
    }
    uterineChart.options.scales.x.min = uterineChart.data.datasets[0].data[0].x;
    uterineChart.options.scales.x.max = now;

    // Обновление метрик
    const stv = 7.2 + (Math.random() - 0.5) * 2;
    const ltv = 12.5 + (Math.random() - 0.5) * 3;
    const lateDecel = Math.random() * 10;
    const accelerations = Math.floor(Math.random() * 5);

    document.getElementById('stv-value').textContent = stv.toFixed(1);
    document.getElementById('ltv-value').textContent = ltv.toFixed(1);
    document.getElementById('late-decel-value').textContent = `${lateDecel.toFixed(1)}%`;
    document.getElementById('accelerations-value').textContent = accelerations;

    // Обновление статуса метрик
    updateMetricStatus('stv', stv);
    updateMetricStatus('ltv', ltv);
    updateMetricStatus('late-decel', lateDecel);
    updateMetricStatus('accelerations', accelerations);

    // Обновление прогноза
    const forecast = Math.random() * 20;
    document.getElementById('forecast-value').textContent = `${forecast.toFixed(0)}%`;

    // Обновление цвета прогноза
    if (forecast < 5) {
        document.getElementById('forecast-value').className = 'forecast-value forecast-green';
        document.getElementById('status-badge').className = 'status-badge status-green';
        document.getElementById('status-badge').textContent = 'Все в порядке';
    } else if (forecast < 15) {
        document.getElementById('forecast-value').className = 'forecast-value forecast-yellow';
        document.getElementById('status-badge').className = 'status-badge status-yellow';
        document.getElementById('status-badge').textContent = 'Повышенный риск';
    } else {
        document.getElementById('forecast-value').className = 'forecast-value forecast-red alert-pulse';
        document.getElementById('status-badge').className = 'status-badge status-red alert-pulse';
        document.getElementById('status-badge').textContent = 'Экстренное вмешательство';
    }

    // Обновление динамики показателей
    // trendChart.data.datasets[0].data.push({ x: now, y: stv });
    // trendChart.data.datasets[1].data.push({ x: now, y: lateDecel });
    // trendChart.data.datasets[2].data.push({ x: now, y: accelerations });

    // Ограничение данных для динамики
    // const trendMaxPoints = 300;
    // if (trendChart.data.datasets[0].data.length > trendMaxPoints) {
    //   trendChart.data.datasets[0].data = trendChart.data.datasets[0].data.slice(-trendMaxPoints);
    //   trendChart.data.datasets[1].data = trendChart.data.datasets[1].data.slice(-trendMaxPoints);
    //   trendChart.data.datasets[2].data = trendChart.data.datasets[2].data.slice(-trendMaxPoints);
        
    //   trendChart.options.scales.x.min = trendChart.data.datasets[0].data[0].x;
    //   trendChart.options.scales.x.max = now;
    // }

    // Обновление всех графиков
    hrChart.update('none');
    uterineChart.update('none');
    //trendChart.update('none');
    }

    // Обновление статуса метрик
function updateMetricStatus(id, value) {
    const element = document.getElementById(`${id}-status`);
    if (!element) return;

    if (id === 'stv' || id === 'ltv') {
        if (value >= 6 && value <= 9 && id === 'stv') {
        element.className = 'metric-status metric-status-green';
        } else if (value >= 5 && value <= 25 && id === 'ltv') {
        element.className = 'metric-status metric-status-green';
        } else {
        element.className = 'metric-status metric-status-red';
        }
    } else if (id === 'late-decel') {
        if (value < 5) {
        element.className = 'metric-status metric-status-green';
        } else if (value < 10) {
        element.className = 'metric-status metric-status-yellow';
        } else {
        element.className = 'metric-status metric-status-red';
        }
    } else if (id === 'accelerations') {
        if (value >= 2) {
        element.className = 'metric-status metric-status-green';
        } else {
        element.className = 'metric-status metric-status-yellow';
        }
    }
}