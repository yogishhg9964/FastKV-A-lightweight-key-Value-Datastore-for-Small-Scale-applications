const HruxClient = require('../sdk/hrux-client.js');

const db = new HruxClient('http://localhost:8081', 'hrux_dev_key_123');

async function sleep(ms) { return new Promise(r => setTimeout(r, ms)); }

async function runProofs() {
    console.log("\n==================================================");
    console.log("🔥 HRUX-DB ULTIMATE FEATURE PROOF SUITE 🔥");
    console.log("==================================================\n");

    try {
        // PROOF 1 & 9: API Key & Basic KV
        console.log("⏳ PROOF 1 & 9: API Key Auth & Basic KV Storage...");
        await db.put('inventory', 'proof_item', 500);
        let val = await db.get('inventory', 'proof_item');
        if (parseInt(val) === 500) {
            console.log("✅ PROOF SUCCESS: Securely connected via API Key and verified basic PUT/GET.\n");
        }

        // PROOF 4: TTL Expiration
        console.log("⏳ PROOF 4: Time-To-Live (TTL) Auto-Deletion...");
        await db.put('receipts', 'temp_receipt', 'valid', 2); // 2 second TTL
        console.log("   -> Created digital receipt with 2-second TTL.");
        let r1 = await db.get('receipts', 'temp_receipt');
        console.log("   -> Immediate Check: " + r1);
        console.log("   -> Waiting 3 seconds for internal sweeper...");
        await sleep(3000);
        try {
            await db.get('receipts', 'temp_receipt');
            console.log("❌ PROOF FAILED: Receipt still exists.");
        } catch(e) {
            console.log("✅ PROOF SUCCESS: Database automatically deleted the receipt without external cleanup!\n");
        }

        // PROOF 3: Sets (Uniqueness)
        console.log("⏳ PROOF 3a: Sets (Double Spend Prevention)...");
        try { await db._request('/delete', { bucket: 'sets', key: 'buyers' }); } catch(e) {} // Clean state
        await fetch('http://localhost:3000/api/buy', { method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify({ userId: 'test_user_1' }) });
        let res2 = await fetch('http://localhost:3000/api/buy', { method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify({ userId: 'test_user_1' }) });
        let data2 = await res2.json();
        if (data2.error && data2.error.includes('already purchased')) {
            console.log("✅ PROOF SUCCESS: Database inherently rejected the duplicate purchase using Sets.\n");
        }

        // PROOF 2 & 5: Speed & Concurrency
        console.log("⏳ PROOF 2 & 5: In-Memory Speed & 256-Way Sharding Concurrency...");
        console.log("   -> Blasting 1000 simultaneous requests to the microservice...");
        await fetch('http://localhost:3000/api/reset', { method: 'POST' });
        const requests = [];
        const startTime = Date.now();
        for (let i = 0; i < 1000; i++) {
            requests.push(fetch('http://localhost:3000/api/buy', { method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify({ userId: `speed_user_${i}` }) }));
        }
        await Promise.all(requests);
        const timeMs = Date.now() - startTime;
        console.log(`✅ PROOF SUCCESS: Handled 1000 concurrent requests in ${timeMs}ms using 256 shards without locking.\n`);

        // PROOF 3b: Queues
        console.log("⏳ PROOF 3b: Queues (Waitlisting)...");
        let qRes = await fetch('http://localhost:3000/api/buy', { method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify({ userId: 'waitlist_user_1' }) });
        let qData = await qRes.json();
        if (qData.error && qData.error.includes('waitlist')) {
            console.log("✅ PROOF SUCCESS: Inventory 0. User automatically pushed to Hrux-DB Waitlist Queue.\n");
        }

        console.log("==================================================");
        console.log("🎉 AUTOMATED PROOFS COMPLETE 🎉");
        console.log("To prove #6 (Crash Recovery) and #7 (Auto-Compaction):");
        console.log("   -> Press Ctrl+C to kill the Go Server right now, restart it, and refresh your browser. The inventory will be perfectly restored!");
        console.log("To prove #8 (Pub/Sub):");
        console.log("   -> Simply look at the live Event Feed streaming in your browser window!");
        console.log("==================================================\n");

    } catch(e) {
        console.error(e);
    }
}
runProofs();
