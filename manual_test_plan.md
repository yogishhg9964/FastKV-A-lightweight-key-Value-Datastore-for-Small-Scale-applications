# Hrux-DB Manual Testing Guide

This guide provides step-by-step commands to manually verify every feature of your newly upgraded Hrux-DB engine. 

**Prerequisite:** Ensure your backend is running (`go run cmd/http-server/main.go`). 
Open a **new PowerShell terminal** to run these test commands. We will use Windows PowerShell's native `Invoke-RestMethod` (abbreviated as `irm`) as it formats JSON beautifully.

---

## Test Suite 1: Basic Key-Value Operations

### 1.1 Put a Key
**Goal:** Verify basic storage in memory and WAL.
```powershell
irm -Method POST -Uri "http://localhost:8081/put" -Body '{"bucket":"users", "key":"user1", "value":"Harsh"}' -ContentType "application/json"
```
**Expected Result:** `{"success": true}`

### 1.2 Get a Key
**Goal:** Verify retrieval.
```powershell
irm -Method POST -Uri "http://localhost:8081/get" -Body '{"bucket":"users", "key":"user1"}' -ContentType "application/json"
```
**Expected Result:** `{"success": true, "data": "Harsh"}`

### 1.3 Delete a Key
**Goal:** Verify deletion.
```powershell
irm -Method POST -Uri "http://localhost:8081/delete" -Body '{"bucket":"users", "key":"user1"}' -ContentType "application/json"
```
**Expected Result:** `{"success": true}` (And if you run the `get` command again, it should return success: false).

---

## Test Suite 2: Time-To-Live (TTL) Lazy Expiration

### 2.1 Set Key with 5-Second TTL
**Goal:** Verify the TTL engine works.
```powershell
irm -Method POST -Uri "http://localhost:8081/put" -Body '{"bucket":"cache", "key":"temp_data", "value":"This will self-destruct", "ttl": 5}' -ContentType "application/json"
```
**Expected Result:** `{"success": true}`

### 2.2 Verify Immediately
Run this *immediately* after the previous command:
```powershell
irm -Method POST -Uri "http://localhost:8081/get" -Body '{"bucket":"cache", "key":"temp_data"}' -ContentType "application/json"
```
**Expected Result:** `{"success": true, "data": "This will self-destruct"}`

### 2.3 Verify After 6 Seconds
Wait 6 seconds and run the exact same `get` command.
**Expected Result:** `{"success": false, "error": "key expired"}`

---

## Test Suite 3: Zero-Data-Loss WAL (Crash Test)

### 3.1 Store Permanent Data
```powershell
irm -Method POST -Uri "http://localhost:8081/put" -Body '{"bucket":"config", "key":"theme", "value":"dark_mode"}' -ContentType "application/json"
```

### 3.2 Simulate a Server Crash
1. Go to the terminal where your backend (`go run cmd/http-server/main.go`) is running.
2. Press `Ctrl + C` to aggressively kill the server.

### 3.3 Restart and Verify Recovery
1. Start the server again (`go run cmd/http-server/main.go`). You should see a log saying "WAL loaded successfully".
2. Run the `get` command to ensure the data survived the crash:
```powershell
irm -Method POST -Uri "http://localhost:8081/get" -Body '{"bucket":"config", "key":"theme"}' -ContentType "application/json"
```
**Expected Result:** `{"success": true, "data": "dark_mode"}`

---

## Test Suite 4: Non-Blocking SCAN

### 4.1 Seed Multiple Keys
Run these quickly to add data to the "inventory" bucket:
```powershell
irm -Method POST -Uri "http://localhost:8081/put" -Body '{"bucket":"inventory", "key":"item1", "value":"apple"}' -ContentType "application/json"
irm -Method POST -Uri "http://localhost:8081/put" -Body '{"bucket":"inventory", "key":"item2", "value":"banana"}' -ContentType "application/json"
irm -Method POST -Uri "http://localhost:8081/put" -Body '{"bucket":"inventory", "key":"item3", "value":"orange"}' -ContentType "application/json"
```

### 4.2 Incremental Scan (Batch of 2)
**Goal:** Verify cursor pagination works without locking the DB.
```powershell
irm -Method POST -Uri "http://localhost:8081/scan" -Body '{"bucket":"inventory", "cursor":0, "count":2}' -ContentType "application/json"
```
**Expected Result:** You will get a `success: true` with a `data` array containing 2 keys, and a `nextCursor` value (e.g., `42`).

### 4.3 Continue Scan
Take the `nextCursor` from the previous output, and put it in this command to get the remaining keys:
```powershell
irm -Method POST -Uri "http://localhost:8081/scan" -Body '{"bucket":"inventory", "cursor":<YOUR_NEXT_CURSOR_HERE>, "count":2}' -ContentType "application/json"
```
**Expected Result:** The remaining keys, and `nextCursor: 0` (meaning the scan is complete).

---

## Test Suite 5: Advanced Data Structures

### 5.1 Set (Unique Values Only)
```powershell
# Add "admin" role
irm -Method POST -Uri "http://localhost:8081/set/add" -Body '{"setName":"roles", "valueStr":"admin"}' -ContentType "application/json"
# Add "editor" role
irm -Method POST -Uri "http://localhost:8081/set/add" -Body '{"setName":"roles", "valueStr":"editor"}' -ContentType "application/json"

# List all roles
irm -Method POST -Uri "http://localhost:8081/set/list" -Body '{"setName":"roles"}' -ContentType "application/json"
```
**Expected Result:** A JSON array containing `"admin"` and `"editor"`. (If you try to add `"admin"` again, the list size will not increase, proving it is a unique Set).

### 5.2 Queue (First In, First Out)
```powershell
# Push tasks to queue
irm -Method POST -Uri "http://localhost:8081/queue/push" -Body '{"queueName":"jobs", "valueStr":"task_A"}' -ContentType "application/json"
irm -Method POST -Uri "http://localhost:8081/queue/push" -Body '{"queueName":"jobs", "valueStr":"task_B"}' -ContentType "application/json"

# Pop a task
irm -Method POST -Uri "http://localhost:8081/queue/pop" -Body '{"queueName":"jobs"}' -ContentType "application/json"
```
**Expected Result:** The pop command will return `"task_A"`, because it was pushed first.

### 5.3 Stack (Last In, First Out)
```powershell
# Push pages to stack (like a browser history back button)
irm -Method POST -Uri "http://localhost:8081/stack/push" -Body '{"stackName":"history", "valueStr":"page_1"}' -ContentType "application/json"
irm -Method POST -Uri "http://localhost:8081/stack/push" -Body '{"stackName":"history", "valueStr":"page_2"}' -ContentType "application/json"

# Pop a page
irm -Method POST -Uri "http://localhost:8081/stack/pop" -Body '{"stackName":"history"}' -ContentType "application/json"
```
**Expected Result:** The pop command will return `"page_2"`, because it was pushed last.
