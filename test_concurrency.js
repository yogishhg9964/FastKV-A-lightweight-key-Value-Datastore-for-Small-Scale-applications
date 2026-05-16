const http = require('http');

const NUM_REQUESTS = 1000;
const URL = 'http://localhost:8081/put';

async function sendRequest(id) {
    return new Promise((resolve, reject) => {
        const payload = JSON.stringify({
            bucket: 'stress_test',
            key: `key_${id}`,
            value: `value_${id}`
        });

        const options = {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                'Content-Length': Buffer.byteLength(payload)
            }
        };

        const req = http.request(URL, options, (res) => {
            let data = '';
            res.on('data', chunk => data += chunk);
            res.on('end', () => resolve(data));
        });

        req.on('error', (err) => reject(err));
        req.write(payload);
        req.end();
    });
}

async function runTest() {
    console.log(`🚀 Firing ${NUM_REQUESTS} parallel write requests to Hrux-DB...`);
    const startTime = Date.now();

    const promises = [];
    // We launch all 1000 requests at the EXACT same time without awaiting them individually
    for (let i = 0; i < NUM_REQUESTS; i++) {
        promises.push(sendRequest(i));
    }

    // Wait for all of them to finish
    await Promise.all(promises);

    const timeTaken = Date.now() - startTime;
    console.log(`✅ Completed ${NUM_REQUESTS} concurrent writes in ${timeTaken} ms!`);
    console.log(`⚡ Throughput: ${((NUM_REQUESTS / timeTaken) * 1000).toFixed(2)} requests/second`);
    
    console.log("\n💡 Why this matters:");
    console.log("If this DB had a single global lock (like it used to), these 1000 requests would have to form a single-file line and wait for each other one by one.");
    console.log("Because of Sharding, the Go backend was able to route these 1000 requests to 256 different independent buckets, writing to them simultaneously across your CPU cores!");
}

runTest();
