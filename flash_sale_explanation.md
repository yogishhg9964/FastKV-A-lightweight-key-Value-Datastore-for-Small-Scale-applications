What is a Flash Sale Engine?
A Flash Sale is an e-commerce event where a highly desirable, limited-quantity item (like concert tickets, limited-edition sneakers, or a heavily discounted iPhone) goes on sale at a specific time.

A Flash Sale Engine is the backend software responsible for surviving the event. It is universally considered the hardest problem in database engineering.

Why? Because at the exact second the sale starts, 10,000 users will press "Buy" at the exact same millisecond.

Traditional SQL databases crash because all 10,000 requests try to lock and update the exact same inventory = 500 row on the hard drive.
Standard Redis caches struggle because they force all 10,000 requests to wait in a single-file line and process them on only one CPU core.
How this Microservice tests EVERY feature of Hrux-DB
By building an external Node.js application to run a Flash Sale, we are proving that Hrux-DB is a production-grade engine capable of solving this massive problem. Here is how the microservice maps to every feature we built, from the initial stage to now:

1. Basic Key-Value Storage (The Initial Stage)
At its core, the microservice uses Hrux-DB to store {"bucket": "inventory", "key": "iphone_15_pro", "value": 500}. Every purchase successfully uses the GET and PUT HTTP API endpoints we built on Day 1 to check the count and decrement it.

2. In-Memory Speed
Because Hrux-DB stores the active inventory entirely in RAM rather than on a hard disk, the microservice can read and update the iphone_15_pro key thousands of times per second with microsecond latency.

3. Advanced Data Structures (Sets & Queues)
We built native data structures into Hrux-DB so developers don't have to write complex logic in their apps:

Sets (Uniqueness): When a user buys a phone, their userID is added to a Hrux-DB Set called buyers. Because Sets enforce strict uniqueness, if a user's browser glitches and sends 50 "Buy" requests at once, the database inherently rejects the duplicates, preventing double-charging.
Queues (FIFO): Once the 500 items are sold out, anyone who clicks "Buy" is instantly pushed onto a Hrux-DB Queue called waitlist. This proves our First-In-First-Out data structures work perfectly.
4. Time-To-Live (TTL)
When a purchase succeeds, the microservice issues a PUT command to create a digital receipt (receipt_user123), but attaches a 300-second TTL. This proves that the database's internal background sweepers work, as it will automatically delete the receipt from memory after exactly 5 minutes without the Node.js server having to do any cleanup!

5. True Concurrency (The 256-Way Sharding)
Our load-test.js script fires 1,000 simultaneous "Buy" requests. This tests our biggest architectural upgrade. Instead of a single global lock freezing the database, Hrux-DB instantly routes the 1,000 requests across its 256 independent Shards (mutex locks), utilizing your entire multi-core CPU to process the traffic jam flawlessly.

6. Zero-Data-Loss Durability (The WAL)
Flash sales involve real money and real inventory; you cannot afford to lose data if the server loses power. Every time the inventory drops by 1, Hrux-DB writes that operation to the .wal file on disk. We can test this by pulling the plug (Ctrl+C) on the database mid-sale. When restarted, the database will read the WAL and the inventory will be perfectly accurate.

7. Background Auto-Compaction
As the load test runs, it generates thousands of operations in the WAL file. This proves our background Compaction Goroutine works! Once it hits the threshold, Hrux-DB will silently compress the WAL file into a clean snapshot in the background, without causing any lag to the live flash sale.

8. Real-Time Broadcasting (The Pub/Sub Engine)
The Flash Sale webpage shows a "Live Purchases" scrolling feed. This proves Hrux-DB is not just a database, but a message broker. Every time an inventory PUT succeeds, the Node server calls db.publish(). Hrux-DB instantly pushes that message over Server-Sent Events to all connected browsers.

9. Cloud-Ready Integration (API Keys & SDK)
The entire microservice connects to Hrux-DB using the hrux-client.js SDK and authenticates using the hrux_dev_key_123 API Key. This proves your HTTP Server Auth Middleware works, making Hrux-DB a secure, cloud-ready Database-as-a-Service (DBaaS) rather than a vulnerable local tool.

In a single load-test command, we are stressing every single piece of architecture you've built. That is why the Flash Sale is the ultimate "Big One" for your presentation!