(function () {
    const state = {
        currentStep: 0,
        totalSteps: Number(document.documentElement.dataset.totalSteps || '0')
    };

    function setButtonContent(button, isFinalStep) {
        button.textContent = '';

        if (isFinalStep) {
            const icon = document.createElement('span');
            icon.className = 'material-icons icon-sm';
            icon.textContent = 'check';
            button.append(icon, ' Conferma');
            return;
        }

        const icon = document.createElement('span');
        icon.className = 'material-icons icon-sm';
        icon.textContent = 'arrow_forward';
        button.append('Avanti ', icon);
    }

    function updateUI() {
        document.querySelectorAll('.form-step').forEach((step, index) => {
            step.classList.toggle('active', index === state.currentStep);
        });

        document.querySelectorAll('.progress-step').forEach((step, index) => {
            step.classList.toggle('completed', index < state.currentStep);
            step.classList.toggle('active', index === state.currentStep);
        });

        const currentStepEl = document.getElementById('currentStep');
        if (currentStepEl) {
            currentStepEl.textContent = state.currentStep + 1;
        }

        const prevBtn = document.getElementById('prevBtn');
        const nextBtn = document.getElementById('nextBtn');
        if (prevBtn) {
            prevBtn.classList.toggle('is-hidden', state.currentStep === 0);
        }
        if (nextBtn) {
            setButtonContent(nextBtn, state.currentStep === state.totalSteps - 1);
        }
    }

    function isCurrentStepAnswered() {
        const currentQuestion = document.querySelector(`.form-step[data-step="${state.currentStep}"]`);
        if (!currentQuestion) {
            return true;
        }
        return Array.from(currentQuestion.querySelectorAll('input[type="radio"]')).some(radio => radio.checked);
    }

    function checkAllAnswered() {
        return Array.from(document.querySelectorAll('.form-step')).every(step =>
            Array.from(step.querySelectorAll('input[type="radio"]')).some(radio => radio.checked)
        );
    }

    function submitSurvey() {
        if (!checkAllAnswered()) {
            alert('Per favore, rispondi a tutte le domande prima di inviare.');
            return;
        }

        const csrfToken = getCookie('csrf_token');
        if (!csrfToken) {
            alert('Sessione scaduta. Ricarica la pagina e riprova.');
            return;
        }

        document.getElementById('csrf_token').value = csrfToken;
        document.getElementById('survey').submit();
    }

    function changeStep(direction) {
        if (direction > 0 && state.currentStep < state.totalSteps - 1 && !isCurrentStepAnswered()) {
            alert('Per favore, seleziona una valutazione prima di continuare.');
            return;
        }

        if (state.currentStep === state.totalSteps - 1 && direction > 0) {
            submitSurvey();
            return;
        }

        state.currentStep = Math.max(0, Math.min(state.totalSteps - 1, state.currentStep + direction));
        updateUI();
    }

    document.addEventListener('DOMContentLoaded', () => {
        document.querySelectorAll('.question-number').forEach((elem, index) => {
            elem.textContent = `Domanda ${index + 1} di ${state.totalSteps}`;
        });

        document.getElementById('prevBtn')?.addEventListener('click', () => changeStep(-1));
        document.getElementById('nextBtn')?.addEventListener('click', () => changeStep(1));
        updateUI();
    });
})();
