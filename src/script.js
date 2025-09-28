import '@fontsource/montserrat/400.css';
import '@fontsource/montserrat/700.css';
import { resetRecording, startRecording } from "./recording";
import { resetCharts, updateData } from './dataProcess';


// Инициализация графиков
document.addEventListener('DOMContentLoaded', function() {
  resetCharts();
  // Инициализация графика динамики
  //const trendCanvas = document.getElementById('trendChart').getContext('2d');
  // const trendChart = new Chart(trendCanvas, {
  //   type: 'line',
  //   data: {
  //     datasets: [
  //       {
  //         label: 'STV',
  //         data: [],
  //         borderColor: '#4CAF50',
  //         borderWidth: 2,
  //         pointRadius: 0,
  //         yAxisID: 'y'
  //       },
  //       {
  //         label: 'Децелерации',
  //         data: [],
  //         borderColor: '#F44336',
  //         borderWidth: 2,
  //         pointRadius: 0,
  //         yAxisID: 'y2'
  //       },
  //       {
  //         label: 'Ускорения',
  //         data: [],
  //         borderColor: '#2196F3',
  //         borderWidth: 2,
  //         pointRadius: 0,
  //         yAxisID: 'y2'
  //       }
  //     ]
  //   },
  //   options: {
  //     responsive: true,
  //     maintainAspectRatio: false,
  //     plugins: {
  //       legend: {
  //         display: false
  //       },
  //       tooltip: {
  //         enabled: false
  //       }
  //     },
  //     scales: {
  //       x: {
  //         type: 'linear',
  //         position: 'bottom',
  //         grid: {
  //           display: false
  //         },
  //         ticks: {
  //           color: '#666',
  //           font: {
  //             size: 12
  //           },
  //           callback: function(value) {
  //             const date = new Date(value - initDate);
  //             return `${date.getMinutes()}:${String(date.getSeconds()).padStart(2, '0')}`;
  //           }
  //         }
  //       },
  //       y: {
  //         type: 'linear',
  //         position: 'left',
  //         min: 0,
  //         max: 15,
  //         grid: {
  //           color: 'rgba(200, 200, 200, 0.3)',
  //           lineWidth: 1,
  //           drawBorder: false
  //         },
  //         ticks: {
  //           color: '#666',
  //           font: {
  //             size: 12
  //           }
  //         },
  //         title: {
  //           display: true,
  //           text: 'STV (мс)',
  //           color: '#666',
  //           font: {
  //             size: 13
  //           }
  //         }
  //       },
  //       y2: {
  //         //type: 'linear',
  //         position: 'right',
  //         min: 0,
  //         max: 10,
  //         grid: {
  //           display: false
  //         },
  //         ticks: {
  //           color: '#666',
  //           font: {
  //             size: 12
  //           }
  //         },
  //         title: {
  //           display: true,
  //           text: 'События/мин',
  //           color: '#666',
  //           font: {
  //             size: 13
  //           }
  //         }
  //       }
  //     }
  //   }
  // });
  
  // Запуск обновления данных
  document.getElementById('recording-btn').onclick = startRecording
  document.getElementById('reset-btn').onclick = resetRecording
  // Инициализация начальных данных
  updateData();
});