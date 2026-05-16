# Live Presentation Script: Proving Every Feature

This is your step-by-step script for your panel presentation. Follow these exact steps to physically prove every single capability of Hrux-DB using the Microservice Frontend.

### Prerequisites
1. Start your Database: `go run cmd/http-server/main.go`
2. Start the Microservice: `npm run dev` (I just fixed this command for you!)
3. Open `http://localhost:3000` in your browser.

---

### Step 1: Prove API Key Integration (DBaaS)
* **What to say:** "Our database isn't just a local tool; it is a Cloud-Ready Database-as-a-Service. This Flash Sale frontend is powered by a completely separate Node.js microservice that connects to Hrux-DB over HTTP using a secure API Key."
* **Action:** Show them the `server.js` code where you initialize `new HruxClient(url, 'hrux_dev_key_123')`.

### Step 2: Prove Real-Time Pub/Sub
* **What to say:** "Hrux-DB is also a real-time message broker. You don't need WebSockets or Kafka."
* **Action:** Open **two browser windows side-by-side**. Click "BUY NOW" on Window 1. 
* **Result:** Instantly, Window 2's Event Stream will populate with the purchase event.

### Step 3: Prove Sets (Uniqueness & Fraud Prevention)
* **What to say:** "In a flash sale, people try to cheat by spamming the buy button. We use Hrux-DB's native `Set` data structure to prevent this."
* **Action:** Click the "BUY NOW" button a second time.
* **Result:** The UI will instantly flash red and say: `❌ You already purchased this item! (Rejected by Hrux Sets!)`. 

### Step 4: Prove True Concurrency (The 256-Way Shard)
* **What to say:** "Now let's see what happens when 1,000 people click Buy at the exact same millisecond. A standard database would row-lock and crash."
* **Action:** Open your terminal and run `node load-test.js`.
* **Result:** The terminal will show 1,000 requests processed in milliseconds. Look at the browser—the inventory will instantly plummet from 499 down to 0!

### Step 5: Prove Queues (FIFO Waitlists)
* **What to say:** "Now that the inventory is at 0, what happens to the next person who tries to buy?"
* **Action:** Click "BUY NOW" on the frontend.
* **Result:** The UI will say: `❌ Sold out! (Pushed to Hrux Queue!)`. Explain that Hrux-DB automatically pushed them to a waitlist without writing complex SQL logic.

### Step 6: Prove WAL Durability & Crash Recovery
* **What to say:** "Because this is purely in-memory, you might think a power outage destroys the inventory data. Let's pull the plug."
* **Action:** Go to your Go terminal (Terminal 1) and press **`Ctrl + C`**. The database is dead.
* **Action:** Start it again (`go run cmd/http-server/main.go`). Refresh the browser.
* **Result:** The inventory count loads as exactly `0`. Explain that Hrux-DB instantly read the `.wal` file on boot and restored every single transaction perfectly.

### Step 7: Prove TTL (Time-To-Live Expirations)
* **What to say:** "When someone successfully buys an item, we generate a digital receipt that is only valid for 5 minutes. Hrux-DB's internal sweepers handle deleting it automatically."
* **Action:** You can prove this via terminal by fetching the receipt. (The microservice creates a key called `receipt_panel_guest_...`). 
Wait 5 minutes, run a GET request, and the database will return `key not found`.
