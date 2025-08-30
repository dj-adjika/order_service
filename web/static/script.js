console.log("Script loaded!");

function getOrder() {
    const orderId = document.getElementById('orderId').value.trim();
    if (!orderId) {
        showError('Please enter an Order ID');
        return;
    }

    console.log("Searching for order:", orderId);

    showLoading(true);
    hideError();
    hideOrderInfo();

    fetch(`/order/${orderId}`)
        .then(response => {
            if (!response.ok) {
                return response.json().then(err => {
                    throw new Error(err.error || 'Order not found');
                });
            }
            return response.json();
        })
        .then(order => {
            displayOrder(order);
            showLoading(false);
        })
        .catch(error => {
            showError(error.message);
            showLoading(false);
        });
}

function displayOrder(order) {
    document.getElementById('orderDetails').innerHTML = `
        <p><strong>Order UID:</strong> ${order.order_uid}</p>
        <p><strong>Track Number:</strong> ${order.track_number}</p>
        <p><strong>Entry:</strong> ${order.entry}</p>
        <p><strong>Locale:</strong> ${order.locale}</p>
        <p><strong>Customer ID:</strong> ${order.customer_id}</p>
        <p><strong>Delivery Service:</strong> ${order.delivery_service}</p>
        <p><strong>Date Created:</strong> ${new Date(order.date_created).toLocaleString()}</p>
    `;

    document.getElementById('deliveryInfo').innerHTML = `
        <p><strong>Name:</strong> ${order.delivery.name}</p>
        <p><strong>Phone:</strong> ${order.delivery.phone}</p>
        <p><strong>Address:</strong> ${order.delivery.city}, ${order.delivery.address}, ${order.delivery.region} ${order.delivery.zip}</p>
        <p><strong>Email:</strong> ${order.delivery.email}</p>
    `;

    document.getElementById('paymentInfo').innerHTML = `
        <p><strong>Transaction:</strong> ${order.payment.transaction}</p>
        <p><strong>Amount:</strong> $${(order.payment.amount / 100).toFixed(2)}</p>
        <p><strong>Currency:</strong> ${order.payment.currency}</p>
        <p><strong>Provider:</strong> ${order.payment.provider}</p>
        <p><strong>Bank:</strong> ${order.payment.bank}</p>
        <p><strong>Delivery Cost:</strong> $${(order.payment.delivery_cost / 100).toFixed(2)}</p>
    `;

    const itemsHtml = order.items.map(item => `
        <div class="item">
            <p><strong>Name:</strong> ${item.name}</p>
            <p><strong>Brand:</strong> ${item.brand}</p>
            <p><strong>Price:</strong> $${(item.price / 100).toFixed(2)}</p>
            <p><strong>Sale:</strong> ${item.sale}%</p>
            <p><strong>Total Price:</strong> $${(item.total_price / 100).toFixed(2)}</p>
            <p><strong>Status:</strong> ${item.status}</p>
        </div>
    `).join('');

    document.getElementById('itemsList').innerHTML = itemsHtml;

    showOrderInfo();
}

function showLoading(show) {
    document.getElementById('loading').style.display = show ? 'block' : 'none';
}

function showError(message) {
    const errorDiv = document.getElementById('error');
    errorDiv.textContent = message;
    errorDiv.style.display = 'block';
}

function hideError() {
    document.getElementById('error').style.display = 'none';
}

function showOrderInfo() {
    document.getElementById('orderInfo').style.display = 'block';
}

function hideOrderInfo() {
    document.getElementById('orderInfo').style.display = 'none';
}

document.getElementById('orderId').addEventListener('keypress', function(e) {
    if (e.key === 'Enter') {
        getOrder();
    }
});