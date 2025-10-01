
export async function(){
    try {
        const formData = new FormData();
        formData.append('bpm_file', bpmFile);
        formData.append('uc_file', ucFile);
        formData.append('session_id', 'session_' + Date.now());

        const response = await fetch(endpoint, {
            method: 'POST',
            body: formData
        });

        if (!response.ok) {
            const errorText = await response.text();
            throw new Error(`HTTP error! status: ${response.status}, message: ${errorText}`);
        }

        const data = await response.json();
        
        result.className = 'success';
        result.innerHTML = `
            <h3>✅ Данные успешно обработаны!</h3>
            <p><strong>Session ID:</strong> ${data.session_id}</p>
            <p><strong>Статус:</strong> ${data.status}</p>
            <p><strong>Предикт:</strong> ${data.prediction}</p>
            <p><strong>BPM точек:</strong> ${data.data?.bpm?.time_sec?.length || 0}</p>
            <p><strong>UC точек:</strong> ${data.data?.uterus?.time_sec?.length || 0}</p>
            <details>
                <summary>Подробные данные</summary>
                <pre>${JSON.stringify(data, null, 2)}</pre>
            </details>
        `;

    } catch (error) {
        console.error('Error:', error);
        result.className = 'error';
        result.innerHTML = `
            <h3>❌ Ошибка при обработке файлов</h3>
            <p><strong>Сообщение:</strong> ${error.message}</p>
            <p><strong>Проверьте:</strong></p>
            <ul>
                <li>Сервер запущен на localhost:8080</li>
                <li>Оба файла в формате CSV</li>
                <li>Файлы имеют правильную структуру (2 колонки)</li>
                <li>Файлы не пустые</li>
            </ul>
        `;
    } finally {
        sendButton.disabled = false;
    }
}