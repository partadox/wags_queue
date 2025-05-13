// WAGS Queue System - Main JavaScript

// Global state
const state = {
    username: '',
    apiKey: '',
    currentPage: 'dashboard'
};

// DOM Elements
const loginForm = document.getElementById('login-form');
const loginError = document.getElementById('login-error');
const loginContainer = document.getElementById('login-container');
const mainContainer = document.getElementById('main-container');
const logoutBtn = document.getElementById('logout-btn');
const navLinks = document.querySelectorAll('.nav-link[data-page]');
const pages = document.querySelectorAll('.page-content');

// Forms
const sendMessageForm = document.getElementById('send-message-form');
const sendBulkForm = document.getElementById('send-bulk-form');
const messagesFilterBtn = document.getElementById('messages-filter-btn');
const broadcastsFilterBtn = document.getElementById('broadcasts-filter-btn');

// Initialize Bootstrap modals
const messageDetailsModal = new bootstrap.Modal(document.getElementById('message-details-modal'));
const broadcastDetailsModal = new bootstrap.Modal(document.getElementById('broadcast-details-modal'));

// Check if user is already logged in
document.addEventListener('DOMContentLoaded', () => {
    const savedUsername = localStorage.getItem('username');
    const savedApiKey = localStorage.getItem('apiKey');
    
    if (savedUsername && savedApiKey) {
        state.username = savedUsername;
        state.apiKey = savedApiKey;
        showMainApp();
        loadDashboard();
    }
});

// Login form submission
loginForm.addEventListener('submit', async (e) => {
    e.preventDefault();
    
    const username = document.getElementById('username').value;
    const key = document.getElementById('password').value;
    
    try {
        const response = await fetch('/api/auth/login', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({ username, key })
        });
        
        const data = await response.json();
        
        if (response.ok) {
            // Store username and key instead of token
            state.username = data.username;
            state.apiKey = data.key;
            
            // Store in localStorage
            localStorage.setItem('username', data.username);
            localStorage.setItem('apiKey', data.key);
            
            showMainApp();
            loadDashboard();
            loadAvailableYears(); // Load years after successful login
        } else {
            loginError.textContent = data.error || 'Login failed. Please check your credentials.';
            loginError.classList.remove('d-none');
        }
    } catch (error) {
        loginError.textContent = 'Network error. Please try again.';
        loginError.classList.remove('d-none');
    }
});

// Logout button
logoutBtn.addEventListener('click', (e) => {
    e.preventDefault();
    logout();
});

// Navigate between pages
navLinks.forEach(link => {
    link.addEventListener('click', (e) => {
        e.preventDefault();
        const page = e.target.getAttribute('data-page');
        showPage(page);
        
        // Remove active class from all nav links
        navLinks.forEach(navLink => navLink.classList.remove('active'));
        
        // Add active class to clicked link
        e.target.classList.add('active');
        
        // Load data for specific pages
        if (page === 'dashboard') {
            loadDashboard();
        } else if (page === 'messages') {
            loadMessages();
        } else if (page === 'broadcasts') {
            loadBroadcasts();
        }
    });
});

// Show page function
function showPage(pageName) {
    state.currentPage = pageName;
    
    // Hide all pages
    pages.forEach(page => page.classList.add('d-none'));
    
    // Show the selected page
    const pageToShow = document.getElementById(`${pageName}-page`);
    if (pageToShow) {
        pageToShow.classList.remove('d-none');
    }
}

// Show main app function
function showMainApp() {
    loginContainer.classList.add('d-none');
    mainContainer.classList.remove('d-none');
    showPage('dashboard');
    
    // Load available years if logged in
    if (state.apiKey) {
        loadAvailableYears();
    }
}

// Logout function
function logout() {
    state.username = '';
    state.apiKey = '';
    localStorage.removeItem('username');
    localStorage.removeItem('apiKey');
    mainContainer.classList.add('d-none');
    loginContainer.classList.remove('d-none');
    
    // Clear form fields
    document.getElementById('username').value = '';
    document.getElementById('password').value = '';
    loginError.classList.add('d-none');
}

// API Request Helper
async function apiRequest(endpoint, method = 'GET', body = null) {
    const headers = {
        'Content-Type': 'application/json',
        'X-Api-Key': state.apiKey
    };
    
    const options = {
        method,
        headers
    };
    
    if (body) {
        options.body = JSON.stringify(body);
    }
    
    try {
        const response = await fetch(`/api${endpoint}`, options);
        
        // If unauthorized, logout
        if (response.status === 401) {
            logout();
            throw new Error('Session expired. Please login again.');
        }
        
        const data = await response.json();
        
        if (!response.ok) {
            throw new Error(data.error || 'API request failed');
        }
        
        return data;
    } catch (error) {
        console.error('API Request Error:', error);
        throw error;
    }
}

// Load Dashboard Data
async function loadDashboard() {
    try {
        // Get current date for default year/month
        const now = new Date();
        const currentYear = now.getFullYear();
        const currentMonth = now.getMonth() + 1; // JavaScript months are 0-indexed
        
        // Get message counts
        const messages = await apiRequest(`/ui/messages?year=${currentYear}&month=${currentMonth}`);
        
        // Calculate statistics
        let totalMessages = messages.length;
        let sentMessages = 0;
        let failedMessages = 0;
        
        messages.forEach(msg => {
            if (msg.status === 'SENT') sentMessages++;
            if (msg.status === 'FAILED') failedMessages++;
        });
        
        // Update dashboard UI
        document.getElementById('total-messages').textContent = totalMessages;
        document.getElementById('sent-messages').textContent = sentMessages;
        document.getElementById('failed-messages').textContent = failedMessages;
        
    } catch (error) {
        console.error('Error loading dashboard:', error);
        // Could show an error toast or notification here
    }
}

// Load Messages
async function loadMessages() {
    const year = document.getElementById('messages-year').value;
    const month = document.getElementById('messages-month').value;
    
    const messagesLoading = document.getElementById('messages-loading');
    const messagesTableBody = document.getElementById('messages-table-body');
    
    try {
        messagesLoading.classList.remove('d-none');
        messagesTableBody.innerHTML = '';
        
        const messages = await apiRequest(`/ui/messages?year=${year}&month=${month}`);
        
        if (messages.length === 0) {
            messagesTableBody.innerHTML = `
                <tr>
                    <td colspan="8" class="text-center">No messages found</td>
                </tr>
            `;
        } else {
            messages.forEach(msg => {
                messagesTableBody.innerHTML += `
                    <tr>
                        <td>${msg.id}</td>
                        <td>${msg.recipient}</td>
                        <td>
                            <span class="status-badge status-${msg.status}">
                                ${msg.status}
                            </span>
                        </td>
                        <td>${msg.broadcast_message}</td>
                        <td>${msg.dt_store}</td>
                        <td>${msg.dt_queue}</td>
                        <td>${msg.dt_send || '-'}</td>
                        <td>
                            <button class="btn btn-sm btn-info view-message" data-id="${msg.id}" 
                                data-recipient="${msg.recipient}" 
                                data-status="${msg.status}" 
                                data-message="${encodeURIComponent(msg.message)}"
                                data-dt-store="${msg.dt_store}"
                                data-dt-queue="${msg.dt_queue}"
                                data-dt-send="${msg.dt_send || '-'}">
                                <i class="bi bi-eye"></i>
                            </button>
                        </td>
                    </tr>
                `;
            });
            
            // Add event listeners to view buttons
            document.querySelectorAll('.view-message').forEach(btn => {
                btn.addEventListener('click', (e) => {
                    const button = e.currentTarget;
                    const msgId = button.getAttribute('data-id');
                    const recipient = button.getAttribute('data-recipient');
                    const status = button.getAttribute('data-status');
                    const message = decodeURIComponent(button.getAttribute('data-message'));
                    const dtStore = button.getAttribute('data-dt-store');
                    const dtQueue = button.getAttribute('data-dt-queue');
                    const dtSend = button.getAttribute('data-dt-send');
                    
                    // Populate modal
                    document.getElementById('modal-recipient').textContent = recipient;
                    document.getElementById('modal-status').innerHTML = `
                        <span class="status-badge status-${status}">
                            ${status}
                        </span>
                    `;
                    document.getElementById('modal-message').textContent = message;
                    document.getElementById('modal-dt-store').textContent = dtStore;
                    document.getElementById('modal-dt-queue').textContent = dtQueue;
                    document.getElementById('modal-dt-send').textContent = dtSend;
                    
                    // Show modal
                    messageDetailsModal.show();
                });
            });
        }
    } catch (error) {
        console.error('Error loading messages:', error);
        messagesTableBody.innerHTML = `
            <tr>
                <td colspan="8" class="text-center text-danger">Error loading messages: ${error.message}</td>
            </tr>
        `;
    } finally {
        messagesLoading.classList.add('d-none');
    }
}

// Load Broadcasts
async function loadBroadcasts() {
    const year = document.getElementById('broadcasts-year').value;
    const month = document.getElementById('broadcasts-month').value;
    
    const broadcastsLoading = document.getElementById('broadcasts-loading');
    const broadcastsTableBody = document.getElementById('broadcasts-table-body');
    
    try {
        broadcastsLoading.classList.remove('d-none');
        broadcastsTableBody.innerHTML = '';
        
        const broadcasts = await apiRequest(`/ui/broadcasts?year=${year}&month=${month}`);
        
        if (broadcasts.length === 0) {
            broadcastsTableBody.innerHTML = `
                <tr>
                    <td colspan="5" class="text-center">No broadcasts found</td>
                </tr>
            `;
        } else {
            broadcasts.forEach(bulk => {
                broadcastsTableBody.innerHTML += `
                    <tr>
                        <td>${bulk.id}</td>
                        <td>
                            <span class="status-badge status-${bulk.status}">
                                ${bulk.status}
                            </span>
                        </td>
                        <td>${bulk.dt_store}</td>
                        <td>${bulk.dt_convert || '-'}</td>
                        <td>
                            <button class="btn btn-sm btn-info view-broadcast" data-id="${bulk.id}">
                                <i class="bi bi-eye"></i> View Details
                            </button>
                        </td>
                    </tr>
                `;
            });
            
            // Add event listeners to view buttons
            document.querySelectorAll('.view-broadcast').forEach(btn => {
                btn.addEventListener('click', async (e) => {
                    const bulkId = e.currentTarget.getAttribute('data-id');
                    await loadBroadcastDetails(bulkId);
                    broadcastDetailsModal.show();
                });
            });
        }
    } catch (error) {
        console.error('Error loading broadcasts:', error);
        broadcastsTableBody.innerHTML = `
            <tr>
                <td colspan="5" class="text-center text-danger">Error loading broadcasts: ${error.message}</td>
            </tr>
        `;
    } finally {
        broadcastsLoading.classList.add('d-none');
    }
}

// Load Broadcast Details
async function loadBroadcastDetails(bulkId) {
    const broadcastDetailsLoading = document.getElementById('broadcast-details-loading');
    const broadcastDetailsTableBody = document.getElementById('broadcast-details-table-body');
    
    try {
        broadcastDetailsLoading.classList.remove('d-none');
        broadcastDetailsTableBody.innerHTML = '';
        
        const messages = await apiRequest(`/ui/broadcasts/${bulkId}/details`);
        
        if (messages.length === 0) {
            broadcastDetailsTableBody.innerHTML = `
                <tr>
                    <td colspan="7" class="text-center">No messages found for this broadcast</td>
                </tr>
            `;
        } else {
            messages.forEach(msg => {
                broadcastDetailsTableBody.innerHTML += `
                    <tr>
                        <td>${msg.id}</td>
                        <td>${msg.recipient}</td>
                        <td>
                            <span class="status-badge status-${msg.status}">
                                ${msg.status}
                            </span>
                        </td>
                        <td>${msg.dt_store}</td>
                        <td>${msg.dt_queue}</td>
                        <td>${msg.dt_send || '-'}</td>
                        <td>
                            <button class="btn btn-sm btn-info view-broadcast-message" data-id="${msg.id}" 
                                data-recipient="${msg.recipient}" 
                                data-status="${msg.status}" 
                                data-message="${encodeURIComponent(msg.message)}"
                                data-dt-store="${msg.dt_store}"
                                data-dt-queue="${msg.dt_queue}"
                                data-dt-send="${msg.dt_send || '-'}">
                                <i class="bi bi-eye"></i>
                            </button>
                        </td>
                    </tr>
                `;
            });
            
            // Add event listeners to view buttons
            document.querySelectorAll('.view-broadcast-message').forEach(btn => {
                btn.addEventListener('click', (e) => {
                    const button = e.currentTarget;
                    const msgId = button.getAttribute('data-id');
                    const recipient = button.getAttribute('data-recipient');
                    const status = button.getAttribute('data-status');
                    const message = decodeURIComponent(button.getAttribute('data-message'));
                    const dtStore = button.getAttribute('data-dt-store');
                    const dtQueue = button.getAttribute('data-dt-queue');
                    const dtSend = button.getAttribute('data-dt-send');
                    
                    // Populate modal
                    document.getElementById('modal-recipient').textContent = recipient;
                    document.getElementById('modal-status').innerHTML = `
                        <span class="status-badge status-${status}">
                            ${status}
                        </span>
                    `;
                    document.getElementById('modal-message').textContent = message;
                    document.getElementById('modal-dt-store').textContent = dtStore;
                    document.getElementById('modal-dt-queue').textContent = dtQueue;
                    document.getElementById('modal-dt-send').textContent = dtSend;
                    
                    // Hide broadcast details modal and show message details modal
                    broadcastDetailsModal.hide();
                    messageDetailsModal.show();
                });
            });
        }
    } catch (error) {
        console.error('Error loading broadcast details:', error);
        broadcastDetailsTableBody.innerHTML = `
            <tr>
                <td colspan="7" class="text-center text-danger">Error loading broadcast details: ${error.message}</td>
            </tr>
        `;
    } finally {
        broadcastDetailsLoading.classList.add('d-none');
    }
}

// Send Message
sendMessageForm.addEventListener('submit', async (e) => {
    e.preventDefault();
    
    const messageSuccess = document.getElementById('send-message-success');
    const messageError = document.getElementById('send-message-error');
    
    // Hide previous alerts
    messageSuccess.classList.add('d-none');
    messageError.classList.add('d-none');
    
    const recipient = document.getElementById('recipient').value;
    const message = document.getElementById('message-text').value;
    
    try {
        const response = await apiRequest('/messages/send', 'POST', {
            recipient,
            message,
            dt_store: new Date().toISOString()
        });
        
        messageSuccess.textContent = `Message queued successfully with ID: ${response.message_id}`;
        messageSuccess.classList.remove('d-none');
        
        // Clear form
        document.getElementById('recipient').value = '';
        document.getElementById('message-text').value = '';
    } catch (error) {
        messageError.textContent = `Error sending message: ${error.message}`;
        messageError.classList.remove('d-none');
    }
});

// Send Bulk Message
sendBulkForm.addEventListener('submit', async (e) => {
    e.preventDefault();
    
    const bulkSuccess = document.getElementById('send-bulk-success');
    const bulkError = document.getElementById('send-bulk-error');
    
    // Hide previous alerts
    bulkSuccess.classList.add('d-none');
    bulkError.classList.add('d-none');
    
    const recipientsText = document.getElementById('recipients').value;
    const message = document.getElementById('bulk-message-text').value;
    
    // Parse recipients (one per line)
    const recipients = recipientsText.split('\n')
        .map(line => line.trim())
        .filter(line => line.length > 0);
    
    if (recipients.length === 0) {
        bulkError.textContent = 'Please enter at least one recipient';
        bulkError.classList.remove('d-none');
        return;
    }
    
    try {
        const response = await apiRequest('/messages/send-bulk', 'POST', {
            recipients,
            message,
            dt_store: new Date().toISOString()
        });
        
        bulkSuccess.textContent = `Bulk message queued successfully with ID: ${response.bulk_message_id}`;
        bulkSuccess.classList.remove('d-none');
        
        // Clear form
        document.getElementById('recipients').value = '';
        document.getElementById('bulk-message-text').value = '';
    } catch (error) {
        bulkError.textContent = `Error sending bulk message: ${error.message}`;
        bulkError.classList.remove('d-none');
    }
});

// Filter Buttons
messagesFilterBtn.addEventListener('click', () => {
    loadMessages();
});

broadcastsFilterBtn.addEventListener('click', () => {
    loadBroadcasts();
});

// Initialize with current date for filters
document.addEventListener('DOMContentLoaded', () => {
    const now = new Date();
    const currentYear = now.getFullYear();
    const currentMonth = now.getMonth() + 1; // JavaScript months are 0-indexed
    
    // Load available years from API
    loadAvailableYears();
    
    // Set current month in filters
    document.getElementById('messages-month').value = currentMonth;
    document.getElementById('broadcasts-month').value = currentMonth;
});

// Load Available Years
async function loadAvailableYears() {
    try {
        // Skip if not logged in
        if (!state.apiKey) return;
        
        const years = await apiRequest('/ui/years');
        
        // Update year dropdowns
        const messagesYearSelect = document.getElementById('messages-year');
        const broadcastsYearSelect = document.getElementById('broadcasts-year');
        
        // Clear existing options
        messagesYearSelect.innerHTML = '';
        broadcastsYearSelect.innerHTML = '';
        
        // Add years from API
        years.forEach(year => {
            messagesYearSelect.innerHTML += `<option value="${year}">${year}</option>`;
            broadcastsYearSelect.innerHTML += `<option value="${year}">${year}</option>`;
        });
        
        // Set current year as default if available
        const currentYear = new Date().getFullYear();
        const hasCurrentYear = years.includes(currentYear);
        
        if (hasCurrentYear) {
            messagesYearSelect.value = currentYear;
            broadcastsYearSelect.value = currentYear;
        } else if (years.length > 0) {
            // If current year not available, use first available year
            messagesYearSelect.value = years[0];
            broadcastsYearSelect.value = years[0];
        }
    } catch (error) {
        console.error('Error loading available years:', error);
        // Fallback to current year
        const currentYear = new Date().getFullYear();
        document.getElementById('messages-year').value = currentYear;
        document.getElementById('broadcasts-year').value = currentYear;
    }
}
