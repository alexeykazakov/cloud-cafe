const API_BASE = '';

async function loadMenu() {
    const grid = document.getElementById('menu-grid');
    const errorBanner = document.getElementById('error-banner');

    try {
        const res = await fetch(`${API_BASE}/api/menu`);
        if (!res.ok) throw new Error(`HTTP ${res.status}`);

        const drinks = await res.json();
        errorBanner.classList.add('hidden');

        grid.innerHTML = drinks.map(drink => `
            <div class="drink-card" onclick="openOrder(${drink.id}, '${escapeHTML(drink.name)}')">
                <img src="${drink.image_url}" alt="${escapeHTML(drink.name)}" loading="lazy">
                <div class="drink-info">
                    <h3>${escapeHTML(drink.name)}</h3>
                    <p>${escapeHTML(drink.description)}</p>
                    <div class="drink-footer">
                        <span class="price">$${drink.price.toFixed(2)}</span>
                        <button class="btn-order">Order</button>
                    </div>
                </div>
            </div>
        `).join('');
    } catch (err) {
        console.error('Failed to load menu:', err);
        errorBanner.classList.remove('hidden');
        grid.innerHTML = '<div class="loading">Unable to load menu. The kitchen might be down.</div>';
    }
}

async function loadOrders() {
    const list = document.getElementById('orders-list');

    try {
        const res = await fetch(`${API_BASE}/api/orders?limit=10`);
        if (!res.ok) throw new Error(`HTTP ${res.status}`);

        const orders = await res.json();

        if (!orders || orders.length === 0) {
            list.innerHTML = '<p class="empty-state">No orders yet. Be the first! ✨</p>';
            return;
        }

        list.innerHTML = orders.map(order => `
            <div class="order-item">
                <div class="order-details">
                    <span class="order-drink">${escapeHTML(order.drink_name)}</span>
                    <span class="order-customer">for ${escapeHTML(order.customer_name)}</span>
                </div>
                <span class="order-status status-${order.status}">${capitalize(order.status)}</span>
            </div>
        `).join('');
    } catch (err) {
        console.error('Failed to load orders:', err);
    }
}

function openOrder(drinkId, drinkName) {
    document.getElementById('order-drink-id').value = drinkId;
    document.getElementById('modal-drink-name').textContent = drinkName;
    document.getElementById('customer-name').value = '';
    document.getElementById('order-modal').classList.remove('hidden');
    document.getElementById('customer-name').focus();
}

function closeModal() {
    document.getElementById('order-modal').classList.add('hidden');
}

document.getElementById('order-form').addEventListener('submit', async (e) => {
    e.preventDefault();

    const drinkId = parseInt(document.getElementById('order-drink-id').value);
    const customerName = document.getElementById('customer-name').value.trim();

    if (!customerName) return;

    try {
        const res = await fetch(`${API_BASE}/api/orders`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ drink_id: drinkId, customer_name: customerName }),
        });

        if (!res.ok) throw new Error(`HTTP ${res.status}`);

        closeModal();
        showToast('Order placed! Your drink is brewing ☕');
        loadOrders();
    } catch (err) {
        console.error('Failed to place order:', err);
        showToast('Failed to place order. Try again!');
    }
});

document.querySelector('.modal-backdrop')?.addEventListener('click', closeModal);

function showToast(message) {
    const toast = document.getElementById('toast');
    document.getElementById('toast-message').textContent = message;
    toast.classList.remove('hidden');
    setTimeout(() => toast.classList.add('hidden'), 3000);
}

function escapeHTML(str) {
    const div = document.createElement('div');
    div.textContent = str;
    return div.innerHTML;
}

function capitalize(str) {
    return str.charAt(0).toUpperCase() + str.slice(1);
}

loadMenu();
loadOrders();
