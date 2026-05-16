/**
 * Official JavaScript SDK for Hrux-DB
 * Provides a simple interface to connect to a Hrux-DB server.
 */
class HruxClient {
    /**
     * Initialize the Hrux-DB client
     * @param {string} url - The URL of the Hrux-DB server (e.g., http://localhost:8081)
     * @param {string} apiKey - The API Key for authentication
     */
    constructor(url, apiKey) {
        this.url = url.replace(/\/$/, ''); // Remove trailing slash if present
        this.apiKey = apiKey;
    }

    async _request(endpoint, body) {
        const response = await fetch(`${this.url}${endpoint}`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                'Authorization': `Bearer ${this.apiKey}`
            },
            body: JSON.stringify(body)
        });

        const data = await response.json();
        if (!data.success) {
            throw new Error(`Hrux-DB Error: ${data.error}`);
        }
        return data;
    }

    // --- Key-Value Operations ---

    /**
     * Store a value in a bucket
     * @param {string} bucket - The bucket name
     * @param {string} key - The key
     * @param {any} value - The value to store
     * @param {number} ttlSeconds - Optional time-to-live in seconds
     */
    async put(bucket, key, value, ttlSeconds = 0) {
        // Convert value to string format if it's an object
        const stringValue = typeof value === 'object' ? JSON.stringify(value) : String(value);
        return this._request('/put', { bucket, key, value: stringValue, ttl: ttlSeconds });
    }

    /**
     * Get a value from a bucket
     * @param {string} bucket - The bucket name
     * @param {string} key - The key
     */
    async get(bucket, key) {
        const res = await this._request('/get', { bucket, key });
        try {
            // Attempt to parse as JSON if it was stored as an object
            return JSON.parse(res.data);
        } catch (e) {
            return res.data;
        }
    }

    /**
     * Delete a key from a bucket
     */
    async delete(bucket, key) {
        return this._request('/delete', { bucket, key });
    }

    /**
     * Scan for keys in a bucket incrementally
     * @param {string} bucket - The bucket name
     * @param {number} cursor - Current cursor (start at 0)
     * @param {number} count - Number of keys to fetch
     */
    async scan(bucket, cursor = 0, count = 10) {
        return this._request('/scan', { bucket, cursor, count });
    }

    // --- Real-Time Pub/Sub ---

    /**
     * Publish a message to a channel
     * @param {string} channel - Channel name
     * @param {string} message - Message payload
     */
    async publish(channel, message) {
        return this._request('/publish', { bucket: channel, value: String(message) });
    }

    /**
     * Subscribe to a channel to receive real-time updates
     * @param {string} channel - Channel name
     * @param {Function} onMessage - Callback triggered when a message arrives
     * @returns {Function} A function to call when you want to unsubscribe
     */
    subscribe(channel, onMessage) {
        // Use standard Server-Sent Events (EventSource). Works natively in browsers.
        const eventSourceUrl = `${this.url}/subscribe?channel=${encodeURIComponent(channel)}&apikey=${encodeURIComponent(this.apiKey)}`;
        const eventSource = new EventSource(eventSourceUrl);

        eventSource.onmessage = (event) => {
            onMessage(event.data);
        };

        eventSource.onerror = (err) => {
            console.error(`Hrux-DB Subscription Error [${channel}]:`, err);
        };

        // Return a cleanup function
        return () => {
            eventSource.close();
        };
    }
}

// Export for Node.js / ES Modules
if (typeof module !== 'undefined' && module.exports) {
    module.exports = HruxClient;
} else if (typeof window !== 'undefined') {
    window.HruxClient = HruxClient;
}
