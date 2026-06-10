(function () {
    const endpoint = '/api/admin/survey/questions';
    let questions = [];

    function icon(name) {
        const elem = document.createElement('span');
        elem.className = 'material-icons icon-sm';
        elem.textContent = name;
        return elem;
    }

    function showEmptyState(list) {
        list.textContent = '';
        const empty = document.createElement('p');
        empty.className = 'survey-empty';
        empty.textContent = 'Nessuna domanda presente. Aggiungi la prima domanda.';
        list.appendChild(empty);
    }

    async function loadQuestions() {
        try {
            const response = await fetch(endpoint);
            if (!response.ok) throw new Error('Failed to load questions');
            questions = await response.json();
            renderQuestions();
        } catch (error) {
            console.error('Error loading questions:', error);
            alert('Errore nel caricamento delle domande');
        }
    }

    function renderQuestions() {
        const list = document.getElementById('questionList');
        if (!questions || questions.length === 0) {
            showEmptyState(list);
            return;
        }

        questions.sort((a, b) => a.Index - b.Index);
        for (let i = 0; i < questions.length; i++) {
            questions[i].Previous = i > 0 ? questions[i - 1].Index : 0;
            questions[i].Next = i < questions.length - 1 ? questions[i + 1].Index : 0;
        }

        list.textContent = '';
        questions.forEach(q => {
            const item = document.createElement('div');
            item.className = 'question-item';

            const text = document.createElement('div');
            text.className = 'question-text';

            const index = document.createElement('span');
            index.className = 'question-index';
            index.textContent = `#${q.Index}`;

            const question = document.createElement('span');
            question.textContent = q.Question;
            text.append(index, question);

            const actions = document.createElement('div');
            actions.className = 'question-actions';

            const editButton = document.createElement('button');
            editButton.className = 'btn';
            editButton.type = 'button';
            editButton.append(icon('edit'), ' Modifica');
            editButton.addEventListener('click', () => openEditModal(q.ID));

            const deleteButton = document.createElement('button');
            deleteButton.className = 'btn btn-danger';
            deleteButton.type = 'button';
            deleteButton.append(icon('delete'), ' Elimina');
            deleteButton.addEventListener('click', () => deleteQuestion(q.ID));

            actions.append(editButton, deleteButton);
            item.append(text, actions);
            list.appendChild(item);
        });
    }

    function openCreateModal() {
        document.getElementById('modalTitle').textContent = 'Aggiungi Domanda';
        document.getElementById('questionForm').reset();
        document.getElementById('questionId').value = '';
        const maxIndex = questions.length > 0 ? Math.max(...questions.map(q => q.Index)) : 0;
        document.getElementById('questionIndex').value = maxIndex + 1;
        document.getElementById('questionModal').style.display = 'block';
    }

    function openEditModal(id) {
        const question = questions.find(q => q.ID === id);
        if (!question) return;

        document.getElementById('modalTitle').textContent = 'Modifica Domanda';
        document.getElementById('questionId').value = question.ID;
        document.getElementById('questionSku').value = question.Sku;
        document.getElementById('questionIndex').value = question.Index;
        document.getElementById('questionText').value = question.Question;
        document.getElementById('questionModal').style.display = 'block';
    }

    function closeModal() {
        document.getElementById('questionModal').style.display = 'none';
    }

    async function saveQuestion(event) {
        event.preventDefault();

        const formData = new FormData(event.target);
        const id = formData.get('id');
        const index = parseInt(formData.get('index'), 10);
        if (index < 1) {
            alert('La posizione deve essere almeno 1');
            return;
        }

        const duplicateIndex = questions.find(q => q.Index === index && (!id || q.ID !== parseInt(id, 10)));
        if (duplicateIndex) {
            alert(`La posizione ${index} è già utilizzata. Le posizioni verranno riordinate automaticamente.`);
        }

        let sku = formData.get('sku');
        if (!id || !sku) {
            sku = 'q' + Date.now();
        }

        const sortedQuestions = [...questions]
            .filter(q => !id || q.ID !== parseInt(id, 10))
            .sort((a, b) => a.Index - b.Index);

        let previous = 0;
        let next = 0;
        for (const question of sortedQuestions) {
            if (question.Index < index) {
                previous = question.Index;
            } else if (question.Index > index && next === 0) {
                next = question.Index;
            }
        }

        const data = {
            id: id ? parseInt(id, 10) : 0,
            sku,
            index,
            next,
            previous,
            question: formData.get('question')
        };

        try {
            const response = await fetch(endpoint, {
                method: id ? 'PUT' : 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'X-CSRF-Token': getCookie('csrf_token')
                },
                body: JSON.stringify(data)
            });

            if (!response.ok) throw new Error('Failed to save question');
            closeModal();
            await loadQuestions();
        } catch (error) {
            console.error('Error saving question:', error);
            alert('Errore nel salvataggio della domanda');
        }
    }

    async function deleteQuestion(id) {
        if (!confirm('Sei sicuro di voler eliminare questa domanda?')) return;

        try {
            const response = await fetch(`${endpoint}/${id}`, {
                method: 'DELETE',
                headers: {
                    'X-CSRF-Token': getCookie('csrf_token')
                }
            });
            if (!response.ok) throw new Error('Failed to delete question');
            await loadQuestions();
        } catch (error) {
            console.error('Error deleting question:', error);
            alert('Errore nell\'eliminazione della domanda');
        }
    }

    document.addEventListener('DOMContentLoaded', () => {
        document.getElementById('createQuestionBtn')?.addEventListener('click', openCreateModal);
        document.getElementById('closeQuestionModalBtn')?.addEventListener('click', closeModal);
        document.getElementById('questionForm')?.addEventListener('submit', saveQuestion);
        document.getElementById('questionModal')?.addEventListener('click', event => {
            if (event.target.id === 'questionModal') {
                closeModal();
            }
        });
        loadQuestions();
    });
})();
