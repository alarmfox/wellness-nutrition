(function () {
    function emptyState(iconName, text, details) {
        const wrapper = document.createElement('div');
        wrapper.className = 'empty-state';

        const icon = document.createElement('span');
        icon.className = 'material-icons';
        icon.textContent = iconName;

        const paragraph = document.createElement('p');
        paragraph.textContent = text;
        if (details) {
            paragraph.appendChild(document.createElement('br'));
            paragraph.append(details);
        }

        wrapper.append(icon, paragraph);
        return wrapper;
    }

    async function loadResults() {
        const container = document.getElementById('resultsContainer');
        try {
            const response = await fetch('/api/admin/survey/results');
            if (!response.ok) throw new Error('Failed to load results');
            renderResults(await response.json());
        } catch (error) {
            console.error('Error loading results:', error);
            container.replaceChildren(emptyState('error_outline', 'Errore nel caricamento dei risultati'));
        }
    }

    function renderResults(questions) {
        const container = document.getElementById('resultsContainer');

        if (!questions || questions.length === 0) {
            container.replaceChildren(emptyState(
                'poll',
                'Nessun risultato disponibile.',
                'Le domande del sondaggio non hanno ancora ricevuto risposte.'
            ));
            return;
        }

        questions.sort((a, b) => a.Index - b.Index);
        container.replaceChildren(...questions.map(renderResultItem));
    }

    function renderResultItem(q) {
        const total = q.Star1 + q.Star2 + q.Star3 + q.Star4 + q.Star5;
        const maxValue = Math.max(q.Star1, q.Star2, q.Star3, q.Star4, q.Star5, 1);

        const item = document.createElement('div');
        item.className = 'result-item';

        const header = document.createElement('div');
        header.className = 'question-header';

        const number = document.createElement('div');
        number.className = 'question-number';
        number.textContent = q.Index;

        const text = document.createElement('div');
        text.className = 'question-text';
        text.textContent = q.Question;
        header.append(number, text);

        const responses = document.createElement('div');
        responses.className = 'total-responses';
        const strong = document.createElement('strong');
        strong.textContent = total;
        responses.append(strong, ` ${total === 1 ? 'risposta ricevuta' : 'risposte ricevute'}`);

        const chartContainer = document.createElement('div');
        chartContainer.className = 'chart-container';
        const chart = document.createElement('div');
        chart.className = 'bar-chart';
        chart.append(
            renderBar(5, q.Star5, maxValue),
            renderBar(4, q.Star4, maxValue),
            renderBar(3, q.Star3, maxValue),
            renderBar(2, q.Star2, maxValue),
            renderBar(1, q.Star1, maxValue)
        );
        chartContainer.appendChild(chart);

        const stats = document.createElement('div');
        stats.className = 'stats-grid';
        stats.append(
            renderStatCard(1, q.Star1),
            renderStatCard(2, q.Star2),
            renderStatCard(3, q.Star3),
            renderStatCard(4, q.Star4),
            renderStatCard(5, q.Star5)
        );

        item.append(header, responses, chartContainer, stats);
        return item;
    }

    function renderBar(stars, count, maxValue) {
        const percentage = maxValue > 0 ? (count / maxValue * 100) : 0;

        const row = document.createElement('div');
        row.className = 'bar-row';

        const label = document.createElement('div');
        label.className = 'bar-label';
        label.append(...renderStars(stars, true));

        const wrapper = document.createElement('div');
        wrapper.className = 'bar-wrapper';

        const fill = document.createElement('div');
        fill.className = 'bar-fill';
        fill.style.width = `${percentage}%`;
        if (count > 0 && percentage > 15) {
            fill.textContent = count;
        }
        wrapper.appendChild(fill);

        const countElem = document.createElement('div');
        countElem.className = 'bar-count';
        countElem.textContent = count;

        row.append(label, wrapper, countElem);
        return row;
    }

    function renderStatCard(stars, count) {
        const card = document.createElement('div');
        card.className = 'stat-card';

        const rating = document.createElement('div');
        rating.className = 'star-rating';
        rating.append(...renderStars(stars));

        const countElem = document.createElement('div');
        countElem.className = 'stat-count';
        countElem.textContent = count;

        const label = document.createElement('div');
        label.className = 'stat-label';
        label.textContent = count === 1 ? 'voto' : 'voti';

        card.append(rating, countElem, label);
        return card;
    }

    function renderStars(count, small = false) {
        const stars = [];
        for (let i = 1; i <= 5; i++) {
            const star = document.createElement('span');
            star.className = `material-icons star${i > count ? ' empty' : ''}${small ? ' star-small' : ''}`;
            star.textContent = 'star';
            stars.push(star);
        }
        return stars;
    }

    document.addEventListener('DOMContentLoaded', loadResults);
})();
