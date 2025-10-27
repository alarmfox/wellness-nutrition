// calendar.js - Admin Calendar Management

// ============================================================================
// CONSTANTS
// ============================================================================
const BookingType = {
    SIMPLE: 'SIMPLE',
    MASSAGE: 'MASSAGE',
    APPOINTMENT: 'APPOINTMENT',
    DISABLE: 'DISABLE'
};

// ============================================================================
// UTILITIES
// ============================================================================
function getCookie(name) {
    const value = `; ${document.cookie}`;
    const parts = value.split(`; ${name}=`);
    if (parts.length === 2) return parts.pop().split(';').shift();
    return null;
}

// ============================================================================
// STATE MANAGEMENT
// ============================================================================
const CalendarState = {
    currentDate: new Date(),
    bookings: [],
    users: [],
    instructors: [],
    selectedInstructorId: '',

    modal: {
        slotTime: null,
        slotData: null,
        bookingId: null,
        refund: false
    },

    resetModal() {
        this.modal.slotTime = null;
        this.modal.slotData = null;
        this.modal.bookingId = null;
    },

    getSlots() {
        const slotMap = new Map();

        this.bookings.forEach(booking => {
            const startTime = new Date(booking.startsAt).getTime();

            if (!slotMap.has(startTime)) {
                slotMap.set(startTime, {
                    startsAt: startTime,
                    InstructorSlots: []
                });
            }

            const slot = slotMap.get(startTime);
            let instructorSlot = slot.InstructorSlots.find(
                is => is.InstructorID === booking.instructorId
            );

            if (!instructorSlot) {
                const instructor = this.instructors.find(i => i.ID === booking.instructorId);
                instructorSlot = {
                    InstructorID: booking.instructorId,
                    InstructorName: instructor ? `${instructor.FirstName} ${instructor.LastName}`.trim() : 'Unknown',
                    State: 'AVAILABLE',
                    Disabled: false,
                    PeopleCount: 0,
                    MaxCapacity: 2
                };
                slot.InstructorSlots.push(instructorSlot);
            }

            if (booking.type === BookingType.DISABLE) {
                instructorSlot.Disabled = true;
                instructorSlot.State = 'UNAVAILABLE';
            } else if (booking.type === BookingType.MASSAGE) {
                instructorSlot.State = 'MASSAGE';
            } else if (booking.type === BookingType.APPOINTMENT) {
                instructorSlot.State = 'APPOINTMENT';
            } else if (booking.type === BookingType.SIMPLE && booking.user) {
                instructorSlot.PeopleCount++;
            }
        });

        return Array.from(slotMap.values());
    },

    getOrCreateSlotData(slotTime) {
        const slots = this.getSlots();
        const targetTime = new Date(slotTime).getTime();
        let slotData = slots.find(slot => slot.startsAt === targetTime);

        if (!slotData) {
            slotData = {
                startsAt: targetTime,
                InstructorSlots: []
            };
        }

        this.instructors.forEach(instructor => {
            const exists = slotData.InstructorSlots.find(is => is.InstructorID === instructor.ID);
            if (!exists) {
                slotData.InstructorSlots.push({
                    InstructorID: instructor.ID,
                    InstructorName: `${instructor.FirstName} ${instructor.LastName || ''}`.trim(),
                    State: 'AVAILABLE',
                    Disabled: false,
                    PeopleCount: 0,
                    MaxCapacity: 2
                });
            }
        });

        return slotData;
    }
};

// ============================================================================
// API SERVICE
// ============================================================================
const API = {
    async fetchBookings(from, to, instructorId = null) {
        let url = `/api/admin/bookings?from=${from.toISOString()}&to=${to.toISOString()}`;
        if (instructorId) {
            url += `&instructorId=${instructorId}`;
        }
        const response = await fetch(url);
        const data = await response.json();
        return Array.isArray(data) ? data : [];
    },

    async fetchUsers() {
        const response = await fetch('/api/admin/users');
        return response.json();
    },

    async fetchInstructors() {
        const response = await fetch('/api/admin/instructors');
        return response.json();
    },

    async createBooking(type, payload) {
        const csrfToken = getCookie('csrf_token');
        const response = await fetch('/api/admin/bookings', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                'X-CSRF-Token': csrfToken
            },
            body: JSON.stringify({ type, ...payload })
        });
        return response.json();
    },

    async deleteBooking(bookingId, refund = false) {
        const csrfToken = getCookie('csrf_token');
        const response = await fetch(`/api/admin/bookings/${bookingId}?refund=${encodeURIComponent(refund)}`, {
            method: 'DELETE',
            headers: {
                'X-CSRF-Token': csrfToken
            }
        });
        return response.status == 204;
    }
};

// ============================================================================
// MODAL MANAGER
// ============================================================================
const Modal = {
    element: null,

    init() {
        this.element = document.getElementById('slotModal');
    },

    show(title, content, showConfirm, onConfirm) {
        if (!this.element) return;

        const titleEl = document.getElementById('modalTitle');
        const body = this.element.querySelector('.modal-body');
        const confirmBtn = document.getElementById('confirmBtn');

        if (titleEl) titleEl.textContent = title;
        if (body) body.innerHTML = content;

        if (confirmBtn) {
            if (showConfirm && onConfirm) {
                confirmBtn.style.display = 'block';
                confirmBtn.onclick = onConfirm;
            } else {
                confirmBtn.style.display = 'none';
            }
        }

        this.element.style.display = 'block';
    },

    hide() {
        if (this.element) {
            this.element.style.display = 'none';
        }
        CalendarState.resetModal();
    },

    showDeleteBooking(bookingId, firstName, lastName, slotTime) {
        CalendarState.modal.bookingId = bookingId;
        CalendarState.modal.slotTime = slotTime;

        const modal = document.getElementById('deleteBookingModal');
        const userName = document.getElementById('deleteBookingUserName');
        const slotInfo = document.getElementById('deleteBookingSlotInfo');

        if (modal && userName && slotInfo) {
            userName.textContent = `${firstName} ${lastName}`;
            slotInfo.textContent = UI.formatSlotDateTime(slotTime);
            modal.style.display = 'block';
        }
    },

    hideDeleteBooking() {
        const modal = document.getElementById('deleteBookingModal');
        if (modal) {
            modal.style.display = 'none';
        }
        CalendarState.modal.bookingId = null;
    }
};

// ============================================================================
// OPERATION FORM BUILDER
// ============================================================================
const OperationForm = {
    build(operation) {
        const slotTime = CalendarState.modal.slotTime;
        const slotData = CalendarState.modal.slotData;

        let html = `<div style="margin-bottom: 20px;">
            <strong>Slot:</strong> ${UI.formatSlotDateTime(slotTime)}
        </div>`;

        html += this.buildInstructorSelect(operation, slotData?.InstructorSlots);

        if (operation === BookingType.SIMPLE) {
            html += this.buildUserSelect();
        }

        return html;
    },

    buildInstructorSelect(operation, instructorStates) {
        const availableInstructors = this.getAvailableInstructors(operation, instructorStates);
        const canSelectAll = operation === BookingType.DISABLE &&
                            availableInstructors.length === CalendarState.instructors.length &&
                            availableInstructors.length > 0;

        let html = `<div style="margin-bottom: 16px;">
            <label style="display: block; margin-bottom: 8px; font-weight: 500;">
                Seleziona Istruttore *
            </label>
            <select id="operationInstructorId" style="width: 100%; padding: 8px; border: 1px solid #ccc; border-radius: 4px;">`;

        if (canSelectAll) {
            html += '<option value="all">Tutti</option>';
        } else {
            html += '<option value="">-- Seleziona istruttore --</option>';
        }

        if (availableInstructors.length === 0) {
            html += '<option value="" disabled>Nessun istruttore disponibile</option>';
        } else {
            availableInstructors.forEach(instructor => {
                const lastName = instructor.LastName ? ` ${instructor.LastName}` : '';
                html += `<option value="${instructor.ID}">${instructor.FirstName}${lastName}</option>`;
            });
        }

        html += `</select>`;

        if (canSelectAll) {
            html += `<div style="margin-top: 8px; font-size: 12px; color: #666;">
                Seleziona "Tutti" per applicare a tutti gli istruttori disponibili
            </div>`;
        }

        html += '</div>';
        return html;
    },

    buildUserSelect() {
        let html = `<div style="margin-bottom: 16px;">
            <label style="display: block; margin-bottom: 8px; font-weight: 500;">
                Seleziona Utente *
            </label>
            <select id="operationUserId" style="width: 100%; padding: 8px; border: 1px solid #ccc; border-radius: 4px;">
                <option value="">-- Seleziona utente --</option>`;

        CalendarState.users.forEach(user => {
            if (user.RemainingAccesses > 0) {
                html += `<option value="${user.ID}">${user.FirstName} ${user.LastName} (${user.Email})</option>`;
            }
        });

        html += `</select></div>`;
        return html;
    },

    getAvailableInstructors(operation, instructorStates) {
        if (!instructorStates) return CalendarState.instructors;

        const exclusionRules = {
            [BookingType.SIMPLE]: ['UNAVAILABLE'],
            [BookingType.DISABLE]: ['MASSAGE', 'APPOINTMENT', 'UNAVAILABLE'],
            [BookingType.MASSAGE]: ['MASSAGE', 'APPOINTMENT', 'UNAVAILABLE'],
            [BookingType.APPOINTMENT]: ['MASSAGE', 'APPOINTMENT', 'UNAVAILABLE']
        };

        const excludeStates = exclusionRules[operation] || [];
        const occupiedIds = new Set();

        instructorStates.forEach(is => {
            const state = is.Disabled || is.State === 'UNAVAILABLE' ? 'UNAVAILABLE' :
                         is.State === 'MASSAGE' ? 'MASSAGE' :
                         is.State === 'APPOINTMENT' ? 'APPOINTMENT' : 'AVAILABLE';

            if (excludeStates.includes(state)) {
                occupiedIds.add(is.InstructorID);
            }

            if (operation === BookingType.DISABLE && is.PeopleCount > 0) {
                occupiedIds.add(is.InstructorID);
            }
        });

        return CalendarState.instructors.filter(inst => !occupiedIds.has(inst.ID));
    },

    async submit(operation) {
        const instructorIdEl = document.getElementById('operationInstructorId');
        const userIdEl = document.getElementById('operationUserId');

        const instructorId = instructorIdEl ? instructorIdEl.value : '';
        const userId = userIdEl ? userIdEl.value : '';

        if (!instructorId) {
            UI.showToast('Seleziona un istruttore', false);
            return false;
        }

        if (operation === BookingType.SIMPLE && !userId) {
            UI.showToast('Seleziona un utente', false);
            return false;
        }

        if (instructorId === 'all') {
            return await this.submitForAll(operation, userId);
        }

        return await this.submitForOne(operation, instructorId, userId);
    },

    async submitForAll(operation, userId) {
        const instructors = CalendarState.instructors;
        UI.showLoading('Elaborazione in corso...');

        let successCount = 0;
        let errorCount = 0;

        for (const instructor of instructors) {
            try {
                const payload = {
                    startsAt: CalendarState.modal.slotTime,
                    instructorId: instructor.ID
                };

                if (operation === BookingType.SIMPLE && userId) {
                    payload.userId = userId;
                }

                const data = await API.createBooking(operation, payload);

                if (data.error) {
                    errorCount++;
                } else if (data.hasBookings && operation === BookingType.DISABLE) {
                    const confirmed = confirm(
                        `${instructor.FirstName} ha ${data.bookingCount} prenotazioni. Eliminarle?`
                    );
                    if (confirmed) {
                        const confirmData = await API.createBooking(operation, {
                            ...payload,
                            confirmed: true
                        });
                        if (confirmData.error) {
                            errorCount++;
                        } else {
                            successCount++;
                        }
                    } else {
                        errorCount++;
                    }
                } else {
                    successCount++;
                }
            } catch (error) {
                errorCount++;
                console.error(error);
            }
        }

        UI.hideLoading();

        if (successCount > 0) {
            const message = errorCount === 0
                ? `Operazione completata per tutti gli istruttori (${successCount})`
                : `Completato: ${successCount} successi, ${errorCount} errori`;
            UI.showToast(message, errorCount === 0);
            return true;
        }

        UI.showToast('Operazione fallita per tutti gli istruttori', false);
        return false;
    },

    async submitForOne(operation, instructorId, userId) {
        const instructor = CalendarState.instructors.find(i => i.ID === parseInt(instructorId));
        const instructorName = instructor ? instructor.FirstName : 'istruttore';

        UI.showLoading('Elaborazione in corso...');

        try {
            const payload = {
                startsAt: CalendarState.modal.slotTime,
                instructorId: parseInt(instructorId)
            };

            if (operation === BookingType.SIMPLE && userId) {
                payload.userId = userId;
            }

            const data = await API.createBooking(operation, payload);
            UI.hideLoading();

            if (data.error) {
                UI.showToast(data.error, false);
                return false;
            }

            if (data.hasBookings && operation === BookingType.DISABLE) {
                const confirmed = confirm(
                    `Questo slot ha ${data.bookingCount} prenotazioni. Eliminarle tutte?`
                );
                if (!confirmed) return false;

                return await this.confirmOperation(operation, instructorId, userId);
            }

            UI.showToast(`Operazione completata per ${instructorName}`, true);
            return true;
        } catch (error) {
            UI.hideLoading();
            UI.showToast('Errore durante l\'operazione', false);
            console.error(error);
            return false;
        }
    },

    async confirmOperation(operation, instructorId, userId) {
        UI.showLoading('Eliminazione prenotazioni...');
        try {
            const payload = {
                startsAt: CalendarState.modal.slotTime,
                instructorId: parseInt(instructorId),
                confirmed: true
            };

            if (operation === BookingType.SIMPLE && userId) {
                payload.userId = parseInt(userId);
            }

            const data = await API.createBooking(operation, payload);
            UI.hideLoading();

            if (data.error) {
                UI.showToast(data.error, false);
                return false;
            }

            UI.showToast(`Slot disabilitato. ${data.deletedCount || 0} prenotazioni eliminate`, true);
            return true;
        } catch (error) {
            UI.hideLoading();
            UI.showToast('Errore durante l\'operazione', false);
            console.error(error);
            return false;
        }
    }
};

// ============================================================================
// SLOT OVERVIEW
// ============================================================================
const SlotOverview = {
    build() {
        const slotData = CalendarState.modal.slotData;

        let html = '<div style="margin-bottom: 20px; padding: 12px; border: 1px solid #e0e0e0; border-radius: 4px; background: #f9f9f9;">';
        html += '<div style="margin-bottom: 10px;"><strong>Stato attuale per istruttore:</strong></div>';

        if (slotData?.InstructorSlots?.length > 0) {
            slotData.InstructorSlots.forEach(is => {
                const info = this.getStateInfo(is);
                html += `<div style="display: flex; justify-content: space-between; padding: 6px 0; border-bottom: 1px solid #e0e0e0;">
                    <span style="font-weight: 500;">${is.InstructorName}</span>
                    <span style="color: ${info.color}; font-size: 13px;">${info.text} (${is.PeopleCount}/${is.MaxCapacity})</span>
                </div>`;
            });
        } else {
            html += '<div style="color: #999; font-style: italic;">Nessun istruttore configurato per questo slot</div>';
        }

        html += '</div>';
        html += this.buildActionCards();

        return html;
    },

    getStateInfo(instructorSlot) {
        if (instructorSlot.Disabled || instructorSlot.State === 'UNAVAILABLE') {
            return { text: 'Non disponibile', color: '#757575' };
        } else if (instructorSlot.State === 'MASSAGE') {
            return { text: '💆 Massaggio', color: '#ff9800' };
        } else if (instructorSlot.State === 'APPOINTMENT') {
            return { text: '📅 Appuntamento', color: '#2196f3' };
        }
        return { text: 'Disponibile', color: '#4caf50' };
    },

    buildActionCards() {
        const slotData = CalendarState.modal.slotData;
        if (!slotData?.InstructorSlots) return '';

        const disabled = slotData.InstructorSlots.filter(is => is.Disabled || is.State === 'UNAVAILABLE');
        const massage = slotData.InstructorSlots.filter(is => is.State === 'MASSAGE');
        const appointment = slotData.InstructorSlots.filter(is => is.State === 'APPOINTMENT');
        const allUnavailable = disabled.length === slotData.InstructorSlots.length;

        let html = '<div>';

        if (!allUnavailable) {
            html += `
                <div class="action-card" onclick="SlotActions.showOperation('${BookingType.SIMPLE}')">
                    <h3>📅 Crea Prenotazione</h3>
                    <p>Prenota questo slot per un utente</p>
                </div>
                <div class="action-card" onclick="SlotActions.showOperation('${BookingType.DISABLE}')">
                    <h3>🚫 Segna Non Disponibile</h3>
                    <p>Disabilita questo slot per uno o più istruttori</p>
                </div>
                <div class="action-card" onclick="SlotActions.showOperation('${BookingType.MASSAGE}')">
                    <h3>💆 Riserva per Massaggio</h3>
                    <p>Blocca questo slot per un massaggio</p>
                </div>
                <div class="action-card" onclick="SlotActions.showOperation('${BookingType.APPOINTMENT}')">
                    <h3>📅 Riserva per Appuntamento</h3>
                    <p>Blocca questo slot per un appuntamento</p>
                </div>
            `;
        }

        disabled.forEach(is => {
            html += `
                <div class="action-card" onclick="SlotActions.quickEnable(${is.InstructorID}, '${is.InstructorName}')">
                    <h3>✅ Abilita per ${is.InstructorName}</h3>
                    <p>Rendi disponibile lo slot per questo istruttore</p>
                </div>
            `;
        });

        massage.forEach(is => {
            html += `
                <div class="action-card" onclick="SlotActions.quickUnreserve(${is.InstructorID}, '${is.InstructorName}', '${BookingType.MASSAGE}')">
                    <h3>🔓 Elimina Massaggio - ${is.InstructorName}</h3>
                    <p>Rimuovi la prenotazione per massaggio</p>
                </div>
            `;
        });

        appointment.forEach(is => {
            html += `
                <div class="action-card" onclick="SlotActions.quickUnreserve(${is.InstructorID}, '${is.InstructorName}', '${BookingType.APPOINTMENT}')">
                    <h3>🔓 Elimina Appuntamento - ${is.InstructorName}</h3>
                    <p>Rimuovi la prenotazione per appuntamento</p>
                </div>
            `;
        });

        html += '</div>';
        return html;
    }
};

// ============================================================================
// SLOT ACTIONS
// ============================================================================
const SlotActions = {
    openSlot(slotTime, slotData) {
        CalendarState.modal.slotTime = slotTime;
        CalendarState.modal.slotData = slotData;
        Modal.show('Gestione Slot', SlotOverview.build(), false, null);
    },

    async showOperation(operation) {
        const titles = {
            [BookingType.SIMPLE]: 'Crea Prenotazione',
            [BookingType.DISABLE]: 'Segna Non Disponibile',
            [BookingType.MASSAGE]: 'Riserva per Massaggio',
            [BookingType.APPOINTMENT]: 'Riserva per Appuntamento'
        };

        if (operation === BookingType.SIMPLE && CalendarState.users.length === 0) {
            await DataLoader.loadUsers();
        }

        const content = OperationForm.build(operation);

        Modal.show(
            titles[operation],
            content,
            true,
            async () => {
                const success = await OperationForm.submit(operation);
                if (success) {
                    Modal.hide();
                    Calendar.load();
                }
            }
        );
    },

    findBookingToDelete(instructorId, bookingType) {
        const slotTime = CalendarState.modal.slotTime;
        const targetTime = new Date(slotTime).getTime();

        return CalendarState.bookings.find(b => {
            const bookingTime = new Date(b.startsAt).getTime();
            return bookingTime === targetTime &&
                   b.instructorId === instructorId &&
                   b.type === bookingType;
        });
    },

    async quickEnable(instructorId, instructorName) {
        if (!confirm(`Confermi di voler abilitare lo slot per ${instructorName}?`)) return;

        UI.showLoading(`Abilitazione slot per ${instructorName}...`);

        try {
            const booking = this.findBookingToDelete(instructorId, BookingType.DISABLE);

            if (!booking) {
                UI.hideLoading();
                UI.showToast('Booking non trovato', false);
                return;
            }

            const data = await API.deleteBooking(booking.id);
            UI.hideLoading();

            if (data.error) {
                UI.showToast(data.error, false);
            } else {
                UI.showToast(`Slot abilitato per ${instructorName}`, true);
                Modal.hide();
                Calendar.load();
            }
        } catch (error) {
            UI.hideLoading();
            UI.showToast('Errore durante l\'abilitazione', false);
            console.error(error);
        }
    },

    async quickUnreserve(instructorId, instructorName, bookingType) {
        if (!confirm(`Confermi di voler eliminare la prenotazione per ${instructorName}?`)) return;

        UI.showLoading(`Eliminazione in corso per ${instructorName}...`);

        try {
            const booking = this.findBookingToDelete(instructorId, bookingType);

            if (!booking) {
                UI.hideLoading();
                UI.showToast('Booking non trovato', false);
                return;
            }

            const data = await API.deleteBooking(booking.id);
            UI.hideLoading();

            if (data.error) {
                UI.showToast(data.error, false);
            } else {
                UI.showToast(`Prenotazione eliminata per ${instructorName}`, true);
                Modal.hide();
                Calendar.load();
            }
        } catch (error) {
            UI.hideLoading();
            UI.showToast('Errore durante l\'eliminazione', false);
            console.error(error);
        }
    }
};

// ============================================================================
// DATA LOADER
// ============================================================================
const DataLoader = {
    async loadBookings() {
        UI.showLoading('Caricamento calendario...');

        const from = new Date(CalendarState.currentDate);
        const to = new Date(CalendarState.currentDate);

        const day = from.getDay();
        const diff = from.getDate() - day + (day === 0 ? -6 : 1);
        from.setDate(diff);
        from.setHours(0, 0, 0, 0);
        to.setDate(from.getDate() + 6);
        to.setHours(23, 59, 59, 999);

        try {
            CalendarState.bookings = await API.fetchBookings(from, to, CalendarState.selectedInstructorId);
            Calendar.render();
            UI.hideLoading();
        } catch (error) {
            console.error('Error loading bookings:', error);
            UI.hideLoading();
            UI.showToast('Errore nel caricamento del calendario', false);
        }
    },

    async loadUsers() {
        try {
            const data = await API.fetchUsers();
            CalendarState.users = data || [];
        } catch (error) {
            console.error('Error loading users:', error);
        }
    },

    async loadInstructors() {
        try {
            const data = await API.fetchInstructors();
            CalendarState.instructors = data || [];
            this.populateInstructorFilter();
        } catch (error) {
            console.error('Error loading instructors:', error);
        }
    },

    populateInstructorFilter() {
        const select = document.getElementById('instructorFilter');
        if (!select) return;

        select.innerHTML = '<option value="">Tutti</option>';
        CalendarState.instructors.forEach(instructor => {
            const option = document.createElement('option');
            option.value = instructor.ID;
            option.textContent = `${instructor.FirstName} ${instructor.LastName}`;
            select.appendChild(option);
        });
    }
};

// ============================================================================
// CALENDAR RENDERER
// ============================================================================
const Calendar = {
    async load() {
        await DataLoader.loadBookings();
    },

    render() {
        const weekStart = new Date(CalendarState.currentDate);
        const day = weekStart.getDay();
        const diff = weekStart.getDate() - day + (day === 0 ? -6 : 1);
        weekStart.setDate(diff);

        const weekEnd = new Date(weekStart);
        weekEnd.setDate(weekStart.getDate() + 6);

        const currentMonth = document.getElementById('currentMonth');
        if (currentMonth) {
            currentMonth.textContent = `${weekStart.toLocaleDateString('it-IT', { day: 'numeric', month: 'short' })} - ${weekEnd.toLocaleDateString('it-IT', { day: 'numeric', month: 'short', year: 'numeric' })}`;
        }

        let html = '<div class="week-view">';
        html += '<div class="day-header"></div>';

        for (let i = 0; i < 6; i++) {
            const date = new Date(weekStart);
            date.setDate(weekStart.getDate() + i);
            const dayName = date.toLocaleDateString('it-IT', { weekday: 'short' });
            const dayNum = date.getDate();
            html += `<div class="day-header">${dayName} ${dayNum}</div>`;
        }

        for (let hour = 7; hour <= 21; hour++) {
            html += `<div class="time-label">${hour}:00</div>`;

            for (let i = 0; i < 6; i++) {
                const date = new Date(weekStart);
                date.setDate(weekStart.getDate() + i);
                date.setHours(hour, 0, 0, 0);

                html += this.renderTimeSlot(date);
            }
        }

        html += '</div>';

        const calendarView = document.getElementById('calendarView');
        if (calendarView) {
            calendarView.innerHTML = html;
        }
    },

    renderTimeSlot(date) {
        const targetTime = date.getTime();
        const isoTime = date.toISOString();

        // Get all bookings for this time slot
        const allBookings = CalendarState.bookings.filter(b => {
            const bookingTime = new Date(b.startsAt).getTime();
            return bookingTime === targetTime;
        });

        // Filter by selected instructor if applicable
        const filteredBookings = CalendarState.selectedInstructorId
            ? allBookings.filter(b => b.instructorId === parseInt(CalendarState.selectedInstructorId))
            : allBookings;

        let html = `<div class="time-slot" onclick="Calendar.handleSlotClick('${isoTime}')">`;

        // Render each booking
        filteredBookings.forEach(booking => {
            const instructor = CalendarState.instructors.find(i => i.ID === booking.instructorId);
            if (!instructor) return;

            const instructorName = `${instructor.FirstName} ${instructor.LastName}`.trim();

            if (booking.type === BookingType.DISABLE) {
                html += `<div class="disabled-text">🚫 Non disponibile - ${instructorName}</div>`;
            } else if (booking.type === BookingType.MASSAGE) {
                html += `<div class="reserved-text">💆 Massaggio - ${instructorName}</div>`;
            } else if (booking.type === BookingType.APPOINTMENT) {
                html += `<div class="reserved-text">📅 Appuntamento - ${instructorName}</div>`;
            } else if (booking.type === BookingType.SIMPLE && booking.user) {
                const cssClass = booking.user.subType === 'SHARED' ? 'booking shared' : 'booking';
                const displayName = `${instructorName} - ${booking.user.lastName} ${booking.user.firstName.substring(0, 3)}.`;
                const title = `${instructorName} - ${booking.user.firstName} ${booking.user.lastName}`;

                html += `<div class="${cssClass}" title="${title}" onclick="event.stopPropagation(); Calendar.handleBookingClick('${booking.id}', '${booking.user.firstName}', '${booking.user.lastName}', '${isoTime}')">
                    ${displayName}
                </div>`;
            }
        });

        html += '</div>';
        return html;
    },

    handleSlotClick(slotTime) {
        const slotData = CalendarState.getOrCreateSlotData(slotTime);
        SlotActions.openSlot(slotTime, slotData);
    },

    handleBookingClick(bookingId, firstName, lastName, slotTime) {
        Modal.showDeleteBooking(bookingId, firstName, lastName, slotTime);
    },

    previousWeek() {
        CalendarState.currentDate.setDate(CalendarState.currentDate.getDate() - 7);
        this.load();
    },

    nextWeek() {
        CalendarState.currentDate.setDate(CalendarState.currentDate.getDate() + 7);
        this.load();
    },

    today() {
        CalendarState.currentDate = new Date();
        this.load();
    }
};

// ============================================================================
// BOOKING ACTIONS
// ============================================================================
const BookingActions = {
    async delete() {
        if (!CalendarState.modal.bookingId) {
            UI.showToast('Errore: prenotazione non selezionata', false);
            return;
        }

        UI.showLoading('Eliminazione prenotazione...');
        try {
            const data = await API.deleteBooking(CalendarState.modal.bookingId, CalendarState.modal.refund == "true");
            UI.hideLoading();

            if (data.error) {
                UI.showToast(data.error, false);
            } else {
                UI.showToast('Prenotazione eliminata con successo', true);
                Modal.hideDeleteBooking();
                Calendar.load();
            }
        } catch (error) {
            UI.hideLoading();
            UI.showToast('Errore durante l\'eliminazione', false);
            console.error(error);
        }
    }
};

// ============================================================================
// GLOBAL EVENT HANDLERS
// ============================================================================
function onInstructorFilterChange() {
    const select = document.getElementById('instructorFilter');
    if (select) {
        CalendarState.selectedInstructorId = select.value;
        Calendar.load();
    }
}

function closeSlotModal() {
    Modal.hide();
}

function closeDeleteBookingModal() {
    Modal.hideDeleteBooking();
}

function confirmDeleteBooking() {
    BookingActions.delete();
}

function previousPeriod() {
    Calendar.previousWeek();
}

function nextPeriod() {
    Calendar.nextWeek();
}

function today() {
    Calendar.today();
}

function toggleRefund(elem) {
    console.log(elem)
   CalendarState.modal.refund = elem.checked;
}

window.onclick = function(event) {
    const slotModal = document.getElementById('slotModal');
    if (event.target === slotModal) {
        closeSlotModal();
    }
};

// ============================================================================
// INITIALIZATION
// ============================================================================
document.addEventListener('DOMContentLoaded', function() {
    Modal.init();

    Promise.all([
        DataLoader.loadUsers(),
        DataLoader.loadInstructors()
    ]).then(() => {
        Calendar.load();
    }).catch(error => {
        console.error('Initialization error:', error);
        UI.showToast('Errore durante l\'inizializzazione', false);
    });
});
