let currentStep = 0; // 0-based index
const totalSteps = {{len .Questions}};

function updateUI() {
    // Update form steps
    document.querySelectorAll('.form-step').forEach((step, index) => {
        step.classList.toggle('active', index === currentStep);
    });

    // Update progress bar
    document.querySelectorAll('.progress-step').forEach((step, index) => {
        step.classList.remove('active', 'completed');
        if (index < currentStep) {
            step.classList.add('completed');
        } else if (index === currentStep) {
            step.classList.add('active');
        }
    });

    // Update progress text
    document.getElementById('currentStep').textContent = currentStep + 1;

    // Update buttons
    const prevBtn = document.getElementById('prevBtn');
    const nextBtn = document.getElementById('nextBtn');

    prevBtn.style.visibility = currentStep === 0 ? 'hidden' : 'visible';

    if (currentStep === totalSteps - 1) {
        nextBtn.innerHTML = '<span class="material-icons" style="font-size: 18px;">check</span> Conferma';
    } else {
        nextBtn.innerHTML = 'Avanti <span class="material-icons" style="font-size: 18px;">arrow_forward</span>';
    }
}

function changeStep(direction) {
    // Check if current question is answered before moving forward
    if (direction > 0 && currentStep < totalSteps) {
        const currentQuestion = document.querySelector(`.form-step[data-step="${currentStep}"]`);
        if (currentQuestion) {
            const radios = currentQuestion.querySelectorAll('input[type="radio"]');
            const answered = Array.from(radios).some(radio => radio.checked);

            if (!answered && currentStep < totalSteps - 1) {
                alert('Per favore, seleziona una valutazione prima di continuare.');
                return;
            }
        }
    }

    // Submit form if on last step and moving forward
    if (currentStep === totalSteps - 1 && direction > 0) {
        const form = document.getElementById('survey');
        // Check if all questions are answered
        const allAnswered = checkAllAnswered();
        if (allAnswered) {
            form.submit();
        } else {
            alert('Per favore, rispondi a tutte le domande prima di inviare.');
        }
        return;
    }

    // Change step
    currentStep += direction;
    if (currentStep < 0) currentStep = 0;
    if (currentStep >= totalSteps) currentStep = totalSteps - 1;

    updateUI();
}

function checkAllAnswered() {
    let allAnswered = true;
    document.querySelectorAll('.form-step').forEach(step => {
        const radios = step.querySelectorAll('input[type="radio"]');
        const answered = Array.from(radios).some(radio => radio.checked);
        if (!answered) allAnswered = false;
    });
    return allAnswered;
}

// Fix question numbers on page load
document.addEventListener('DOMContentLoaded', function() {
    document.querySelectorAll('.question-number').forEach((elem, index) => {
        elem.textContent = `Domanda ${index + 1} di ${totalSteps}`;
    });
    updateUI();
});

// Initialize
updateUI();
