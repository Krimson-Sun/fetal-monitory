import { resetCharts, updateData } from "./dataProcess";

let wasSaved = false;

export function startRecording(){
    resetCharts();
    wasSaved = false;
    let interval = setInterval(updateData, 100);
    console.log('click!');
    setStopBtn(interval)
    hideBtn('input-btn', 'ctg-recording-button input')

    const mainContent = document.getElementById('main-content');
    mainContent.className = "main-content";
}

export function stopRecording(interval){
    clearInterval(interval);
    console.log('click!');
    setDownloadBtn();

    showBtn('reset-btn', 'ctg-recording-button reset');
}

export function resetRecording(){
    if (!wasSaved && !confirm('После сброса текущая запись будет потеряна! Все равно сбросить? ')) return 0

    resetCharts();
    const mainContent = document.getElementById('main-content');
    mainContent.className = "main-content disabled";

    hideBtn('reset-btn', 'ctg-recording-button reset');
    showBtn('input-btn', 'ctg-recording-button input');
    setPlayBtn();
}

export function saveRecording(){
    prompt('Введите название для записи')
    alert('Запись сохранена!')
    wasSaved = true;
}

function setPlayBtn(){
    const btn = document.getElementById('recording-btn');

    btn.innerHTML = `
        <span class="icon">
            <svg viewBox="0 0 24 24" fill="none">
                <path d="M3 4 A2 2 0 0 1 5.99 2.71 L20.83 11.19 A2 2 0 0 1 20.76 14.70 L5.93 22.47 A2 2 0 0 1 3 20.70 Z" fill="white" />
            </svg>
        </span>`
    btn.className = "ctg-recording-button";
    btn.title = "Начать наблюдение"
    btn.onclick = startRecording;
}
function setStopBtn(interval){
    const btn = document.getElementById('recording-btn');

    btn.innerHTML = `
      <span class="icon">
        <svg viewBox="0 0 12 12" fill="none">
        <rect x="0" y="0" width="12" height="12" rx="2" fill="white"/>
        </svg>
      </span>`
    btn.className = "ctg-recording-button recording";
    btn.title = "Остановить запись"
    btn.onclick = ()=>stopRecording(interval);
}
function setDownloadBtn(){
    const btn = document.getElementById('recording-btn');
    btn.innerHTML = `
        <span class="icon">
            <svg viewBox="0 0 32 32" fill="white">
            <path d="M27,1H2C1.448,1,1,1.448,1,2v28c0,0.552,0.448,1,1,1h28c0.552,0,1-0.448,1-1V5L27,1z M8,3h16
                v10H8V3z M29,29H3V3h4v10c0,0.552,0.448,1,1,1h16c0.552,0,1-0.448,1-1V3h1.172L29,5.829V29z M9,26h14c0.552,0,1-0.448,1-1v-7
                c0-0.552-0.448-1-1-1H9c-0.552,0-1,0.448-1,1v7C8,25.552,8.448,26,9,26z M9,18h14v7H9V18z M18,12h5V4h-5V12z M19,5h3v6h-3V5z M10,19
                h12v1H10V19z M10,21h12v1H10V21z M10,23h12v1H10V23z"/>
            </svg>
            </svg>
        </span>`
    btn.className = "ctg-recording-button";
    btn.title = "Сохранить наблюдение"
    btn.onclick = saveRecording;
}

function hideBtn(id, className){
    const btn=document.getElementById(id);
    btn.className = `${className} hidden`
    btn.disabled = true
}
function showBtn(id, className){
    const btn=document.getElementById(id);
    btn.className = className;
    btn.disabled = false
}