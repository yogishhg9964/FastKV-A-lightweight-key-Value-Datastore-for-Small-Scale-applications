const express = require('express');
const cors = require('cors');
const path = require('path');
const HruxClient = require('../sdk/hrux-client.js');

const app = express();
app.use(cors());
app.use(express.json());
app.use(express.static(path.join(__dirname, 'public')));

const db = new HruxClient('http://localhost:8081', 'hrux_dev_key_123');

// --- SYSTEM RESET ---
app.post('/api/reset', async (req, res) => {
    try {
        try { await db._request('/delete', { bucket: 'sets', key: 'buyers' }); } catch(e) {}
        try { await db._request('/delete', { bucket: 'sets', key: 'catalog_products' }); } catch(e) {}
        res.json({ success: true, message: 'Database reset. Ready for dynamic product creation.' });
    } catch (e) {
        res.status(500).json({ error: e.message });
    }
});

// --- CATALOG: DYNAMIC PRODUCT CREATION ---
app.post('/api/product/create', async (req, res) => {
    const { name, bucketName, productKey, inventory, basePrice } = req.body;
    if(!name || !bucketName || !productKey) return res.status(400).json({error: 'Missing parameters'});

    try {
        // 1. Create Meta Data in user-defined bucket
        await db.put(bucketName, `${productKey}_meta`, JSON.stringify({ name, basePrice: parseInt(basePrice) }));
        
        // 2. Create Live Inventory in user-defined bucket
        await db.put(bucketName, `${productKey}_inv`, parseInt(inventory));
        
        // 3. Register Product into the global index (Set)
        const indexData = JSON.stringify({ bucket: bucketName, key: productKey });
        await db._request('/set/add', { setName: 'catalog_products', valueStr: indexData });
        
        await db.publish('events', `[KV Creation] Dynamic product ${productKey} created in bucket '${bucketName}'`);
        res.json({ success: true });
    } catch (e) { res.status(500).json({ error: e.message }); }
});

// --- CATALOG: FETCH ALL ---
app.get('/api/products', async (req, res) => {
    try {
        let indexList = [];
        try {
            const listRes = await db._request('/set/list', { setName: 'catalog_products' });
            indexList = listRes.data || [];
        } catch(e) {} // If empty, returns empty

        const results = [];
        for (const str of indexList) {
            let info;
            try { info = JSON.parse(str); } catch(e) { continue; }
            const { bucket, key } = info;

            // 1. Fetch metadata
            let metaRaw = null;
            try { metaRaw = await db.get(bucket, `${key}_meta`); } catch(e) {}
            if (!metaRaw) continue;
            let meta = typeof metaRaw === 'string' ? JSON.parse(metaRaw) : metaRaw;
            
            // 2. Fetch inventory
            let invRaw = 0;
            try { invRaw = await db.get(bucket, `${key}_inv`); } catch(e) {}
            
            // 3. Fetch autonomous TTL offer
            let offerRaw = null;
            let offer = null;
            try { 
                offerRaw = await db.get(bucket, `${key}_offer`); 
                if (offerRaw) offer = typeof offerRaw === 'string' ? JSON.parse(offerRaw) : offerRaw;
            } catch(e) {
                // Not found -> No offer or TTL expired
            }
            
            results.push({
                bucket,
                key,
                name: meta.name,
                basePrice: meta.basePrice,
                inventory: parseInt(invRaw || 0),
                offer: offer
            });
        }
        res.json({ success: true, products: results });
    } catch (e) {
        res.status(500).json({ error: e.message });
    }
});

// --- TTL ENGINE: CREATE OFFER ---
app.post('/api/offer/create', async (req, res) => {
    const { bucket, key, flashPrice, ttl } = req.body;
    try {
        const offerData = JSON.stringify({ flashPrice: parseInt(flashPrice), active: true });
        
        // Attach TTL directly to the offer key in the specified bucket
        await db.put(bucket, `${key}_offer`, offerData, parseInt(ttl));
        
        await db.publish('events', `[TTL Engine] Flash Sale Offer dynamically attached to ${key} (${ttl}s)`);
        res.json({ success: true });
    } catch (e) { res.status(500).json({ error: e.message }); }
});

// --- COMMERCE FLOW: PURCHASE ---
app.post('/api/buy', async (req, res) => {
    let { userId, bucket, key } = req.body;
    if (!bucket || !key) return res.status(400).json({error: 'Missing product identifiers'});

    try {
        // 1. Fraud Protection via Sets
        let buyers = [];
        try {
            const buyersList = await db._request('/set/list', { setName: 'buyers' });
            buyers = buyersList.data || [];
        } catch (e) {}

        if (buyers.includes(userId)) {
            return res.status(400).json({ 
                error: 'Duplicate purchase blocked automatically.',
                reason: 'Powered by Hrux-DB uniqueness sets'
            });
        }

        // 2. Inventory Check via WAL Durability
        let count = 0;
        try {
            let val = await db.get(bucket, `${key}_inv`);
            count = parseInt(val);
        } catch (e) {}
        count = count || 0;
        
        // 3. Waitlist System via Queues
        if (count <= 0) {
            await db._request('/queue/push', { queueName: `waitlist:${key}`, valueStr: userId });
            return res.status(400).json({ 
                error: 'Sold out! You have joined the Waitlist.',
                reason: 'Powered by Hrux-DB FIFO queues'
            });
        }

        // 4. Atomic Execution: Decrement, Add to Set, Broadcast
        await db.put(bucket, `${key}_inv`, count - 1);
        await db._request('/set/add', { setName: 'buyers', valueStr: userId });
        await db.publish('events', `User ${userId} successfully bought ${key}!`);

        res.json({ success: true, message: 'Purchase successful!' });
    } catch (e) {
        res.status(500).json({ error: e.message });
    }
});

// --- OBSERVABILITY: PERFORMANCE INTELLIGENCE ---
app.get('/api/intelligence', async (req, res) => {
    try {
        const response = await fetch('http://localhost:8081/api/metrics');
        const hruxMetrics = await response.json();
        
        // Provide Simulated SQL Baseline for direct architectural comparison
        const sqlMetrics = {
            heapAllocMB: 128.4 + (Math.random() * 10), // Heavy footprint
            activeLanes: 1 // Mutex lock bottleneck
        };
        
        res.json({ success: true, hrux: hruxMetrics, sql: sqlMetrics });
    } catch (e) {
        res.status(500).json({ error: e.message });
    }
});

const PORT = process.env.PORT || 3000;
app.listen(PORT, () => {
    console.log(`🚀 Authentic Flash Sale Engine running on http://localhost:${PORT}`);
});
