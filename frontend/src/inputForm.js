import { setDataToCharts } from "./dataProcess";
import { setDelFunction, setSaveDelBtns, setSaveFunction} from "./recording";

let bpmFile = null;
let ucFile = null;
const baseUrl = 'http://localhost:8080'

const changeBpmFile = (e) => {
      if (e.target.files.length > 0 && e.target.files[0] != '') {  
        bpmFile = e.target.files[0];
          bpmFileInfo.innerHTML = `
              <strong>Файл BPM:</strong> ${bpmFile.name}<br>
              <strong>Размер:</strong> ${(bpmFile.size / 1024).toFixed(2)} KB
          `;
          bpmFileInfo.style.display = ''
          bpmSection.classList.add('active');
      } else{
        bpmFile = null
        bpmFileInfo.innerHTML = '';
        bpmFileInfo.style.display = 'none'
        bpmSection.classList.remove('active');
      }
      checkFilesReady();
  }

const changeUcFile = (e) => {
  if (e.target.files.length > 0) {
      ucFile = e.target.files[0];
      ucFileInfo.innerHTML = `
          <strong>Файл UC:</strong> ${ucFile.name}<br>
          <strong>Размер:</strong> ${(ucFile.size / 1024).toFixed(2)} KB
      `;
      ucFileInfo.style.display = ''
      ucSection.classList.add('active');
  } else{
      ucFile = null
      ucFileInfo.innerHTML = '';
      ucFileInfo.style.display = 'none'
      ucSection.classList.remove('active');
  }
  checkFilesReady();
}

// Открытие окна импорта
export function openFileImportAlert() {
  const alertElement = document.getElementById('file-import-alert');
  alertElement.style.display = '';
}

// Закрытие окна импорта
export function closeFileImportAlert() {
  const alertElement = document.getElementById('file-import-alert');
  alertElement.style.display = 'none';

  const hrFile = document.getElementById('bpmFileInput');
  const uterineFile = document.getElementById('ucFileInput');

  hrFile.value = null;
  uterineFile.value = null;
  hrFile.dispatchEvent(new Event('change', changeBpmFile))
  uterineFile.dispatchEvent(new Event('change', changeUcFile))
}

export async function sendFiles() {
    // Здесь будет обработка загрузки файлов
    const bpmFile = document.getElementById('bpmFileInput').files[0];
    const ucFile = document.getElementById('ucFileInput').files[0];
    const uploadBtn = document.getElementById('sendButton')
    
    if (!bpmFile || !ucFile) {
      alert('Пожалуйста, выберите оба файла');
      return;
    }
    if (bpmFile.type != 'text/csv' || ucFile.type != 'text/csv'){
      alert('Файлы должны быть в формате .csv')
      return;
    }

    uploadBtn.classList.add('loading');
    uploadBtn.disabled = true;
    uploadBtn.innerHTML = `
      <div class="spinner"></div>
      <span class="text">Данные обрабатываются</span>
    `
    try {
        const formData = new FormData();
        formData.append('bpm_file', bpmFile);
        formData.append('uc_file', ucFile);
        formData.append('session_id', 'session_' + Date.now());

        const response = await fetch(`${baseUrl}/upload-dual`, {
            method: 'POST',
            body: formData
        });

        if (!response.ok) {
            const errorText = await response.text();
            throw new Error(`HTTP error! status: ${response.status}, message: ${errorText}`);
        }

        const data = await response.json();
        console.log('Данные успешно обработаны:', data)
        setDelFunction(()=>
          fetch(`${baseUrl}/decision`, {
            method: 'POST',
            body:JSON.stringify({
              session_id: data['session_id'],
              save:false
            })
          })
        )
        setSaveFunction(()=>
          fetch(`${baseUrl}/decision`, {
            method: 'POST',
            body:JSON.stringify({
              session_id: data['session_id'],
              save: true
            })
          })
        )

        setDataToCharts(data)
        closeFileImportAlert()
    } catch (error) {
        alert(`
          Ошибка при обработке файлов\n
          Сообщение: ${error.message}
        `)
    } finally {
        uploadBtn.classList.remove('loading');
        uploadBtn.disabled = false;
        uploadBtn.innerHTML = `
          <span class="text">Обработать данные</span>
        `
    }
}

export function setInputAlertEventListeners(){
  const bpmFileInput = document.getElementById('bpmFileInput');
  const ucFileInput = document.getElementById('ucFileInput');
  const bpmFileInfo = document.getElementById('bpmFileInfo');
  const ucFileInfo = document.getElementById('ucFileInfo');

  document.getElementById('close-import-alert').addEventListener('click', closeFileImportAlert);
  document.getElementById('sendButton').addEventListener('click', sendFiles);

  bpmFileInput.addEventListener('change', changeBpmFile);

  // Обработчики для UC файла
  ucFileInput.addEventListener('change', changeUcFile)
  
  // Закрытие при клике за пределами
  document.getElementById('file-import-alert').addEventListener('click', function(e) {
    if (e.target === this) {
      closeFileImportAlert();
    }
  });

  // Пример вызова окна импорта
  document.getElementById('input-btn').addEventListener('click', openFileImportAlert);

  // Drag and drop functionality
  [bpmSection, ucSection].forEach((section, index) => {
      section.addEventListener('dragover', (e) => {
          e.preventDefault();
          section.classList.add('active');
      });

      section.addEventListener('dragleave', (e) => {
          e.preventDefault();
          if (!section.contains(e.relatedTarget)) {
              section.classList.remove('active');
          }
      });

      section.addEventListener('drop', (e) => {
          e.preventDefault();
          section.classList.remove('active');
          
          const files = e.dataTransfer.files;
          if (files.length > 0) {
              const file = files[0];
              if (file.name.toLowerCase().endsWith('.csv')) {
                  if (index === 0) {
                      bpmFileInput.files = files;
                      bpmFileInput.dispatchEvent(new Event('change'));
                  } else {
                      ucFileInput.files = files;
                      ucFileInput.dispatchEvent(new Event('change'));
                  }
              }
          }
      });
  });
}

function checkFilesReady() {
    if (bpmFile && ucFile) {
        sendButton.disabled = false;
    } else {
        sendButton.disabled = true;
    }
}