# import requests
# import time
# import json
# import os
# import re
# from datetime import datetime

# # ================= CONFIG =================
# BASE_URL = "http://localhost:8080"
# EMAIL = "admin@example.com"
# LOG_FILE = "server.log"               
# CSV_FILE_PATH = "/home/riya/Desktop/cns4 1.csv"  # your large CSV
# NUM_SCANS = 1800                      # change to 20000 later
# DELAY_BETWEEN_SCANS = 0.5             # seconds — avoid overwhelming server
# # ===========================================

# DEVICE_ID = None
# ACCESS_TOKEN = None

# def wait_for_otp(log_file, timeout=60):
#     print("Waiting for OTP in server logs...")
#     start = time.time()
#     pattern = r"OTP: (\d{6})"  # matches "OTP: 123456"

#     while time.time() - start < timeout:
#         if not os.path.exists(log_file):
#             print(f"Log file not found: {log_file}. Waiting...")
#             time.sleep(2)
#             continue

#         with open(log_file, "r") as f:
#             content = f.read()
#             match = re.search(pattern, content)
#             if match:
#                 otp = match.group(1)
#                 print(f"OTP automatically detected: {otp}")
#                 return otp

#         time.sleep(2)

#     raise Exception(f"OTP not found in {log_file} within {timeout}s")

# def login():
#     global DEVICE_ID, ACCESS_TOKEN

#     print("1. Sending OTP request...")
#     resp = requests.post(f"{BASE_URL}/auth/otp/send", json={"email": EMAIL})
#     resp.raise_for_status()
#     data = resp.json()
#     DEVICE_ID = data["device_id"]
#     print(f"Device ID: {DEVICE_ID}")

#     otp = wait_for_otp(LOG_FILE)

#     print("2. Verifying OTP...")
#     resp = requests.post(
#         f"{BASE_URL}/auth/otp/verify",
#         json={"email": EMAIL, "otp": otp, "device_id": DEVICE_ID}
#     )
#     resp.raise_for_status()
#     data = resp.json()
#     ACCESS_TOKEN = data["access_token"]
#     print("Login successful!")

# def create_one_scan():
#     headers = {
#         "Content-Type": "application/json",
#         "Authorization": f"Bearer {ACCESS_TOKEN}",
#         "X-Device-ID": DEVICE_ID
#     }

#     resp = requests.post(f"{BASE_URL}/scans", headers=headers, json={
#         "vehicle_id": 1,
#         "material_type_id": 1
#     })

#     if resp.status_code != 201:
#         print(f"Create scan failed: {resp.status_code} - {resp.text}")
#         return None

#     data = resp.json()
#     scan_id = data["id"]
#     print(f"Scan created: {scan_id}")
#     return scan_id

# def upload_coords_to_scan(scan_id):
#     headers = {
#         "Authorization": f"Bearer {ACCESS_TOKEN}",
#         "X-Device-ID": DEVICE_ID
#     }

#     files = {"cns_file": open(CSV_FILE_PATH, "rb")}

#     start = time.time()
#     resp = requests.post(
#         f"{BASE_URL}/scans/{scan_id}/coordinates",
#         headers=headers,
#         files=files
#     )
#     duration = time.time() - start

#     if resp.status_code != 200:
#         print(f"Upload failed for scan {scan_id}: {resp.status_code} - {resp.text}")
#     else:
#         data = resp.json()
#         print(f"Success: scan {scan_id} — {data.get('count', 'N/A')} coords in {duration:.2f}s")

# # ================= MAIN =================
# if __name__ == "__main__":
#     print(f"Starting bulk seed at {datetime.now()}")
#     print(f"Target: {NUM_SCANS} scans, each with coords from {CSV_FILE_PATH}")

#     login()

#     success = 0
#     for i in range(1, NUM_SCANS + 1):
#         print(f"\n--- Scan {i}/{NUM_SCANS} ---")
#         scan_id = create_one_scan()
#         if scan_id:
#             upload_coords_to_scan(scan_id)
#             success += 1
#         time.sleep(DELAY_BETWEEN_SCANS)  # prevent server overload

#     print(f"\nFinished! Successfully processed {success}/{NUM_SCANS} scans")




import requests
import time
import json
import os
import re
from datetime import datetime
from concurrent.futures import ThreadPoolExecutor, as_completed
from tqdm import tqdm

# ================= CONFIG =================
BASE_URL = "http://localhost:8080"
EMAIL = "admin@example.com"
LOG_FILE = "server.log"
CSV_FILE_PATH = "/home/riya/Desktop/cns4 1.csv"  
NUM_SCANS = 200
MAX_WORKERS = 5  
DELAY_BETWEEN_BATCHES = 1  
RETRY_COUNT = 3

DEVICE_ID = None
ACCESS_TOKEN = None

def wait_for_otp(log_file, timeout=60):
    print("Waiting for OTP in server logs...")
    start = time.time()
    pattern = r"OTP: (\d{6})"  # matches "OTP: 123456"

    while time.time() - start < timeout:
        if not os.path.exists(log_file):
            print(f"Log file not found: {log_file}. Waiting...")
            time.sleep(2)
            continue

        with open(log_file, "r") as f:
            content = f.read()
            match = re.search(pattern, content)
            if match:
                otp = match.group(1)
                print(f"OTP automatically detected: {otp}")
                return otp

        time.sleep(2)

    raise Exception(f"OTP not found in {log_file} within {timeout}s")

def login():
    global DEVICE_ID, ACCESS_TOKEN

    print("1. Sending OTP request...")
    resp = requests.post(f"{BASE_URL}/auth/otp/send", json={"email": EMAIL})
    resp.raise_for_status()
    data = resp.json()
    DEVICE_ID = data["device_id"]
    print(f"Device ID: {DEVICE_ID}")

    otp = wait_for_otp(LOG_FILE)

    print("2. Verifying OTP...")
    resp = requests.post(
        f"{BASE_URL}/auth/otp/verify",
        json={"email": EMAIL, "otp": otp, "device_id": DEVICE_ID}
    )
    resp.raise_for_status()
    data = resp.json()
    ACCESS_TOKEN = data["access_token"]
    print("Login successful!")

def create_one_scan():
    headers = {
        "Content-Type": "application/json",
        "Authorization": f"Bearer {ACCESS_TOKEN}",
        "X-Device-ID": DEVICE_ID
    }

    resp = requests.post(f"{BASE_URL}/scans", headers=headers, json={
        "vehicle_id": 1,
        "material_type_id": 1
    })

    if resp.status_code != 201:
        print(f"Create scan failed: {resp.status_code} - {resp.text}")
        return None

    data = resp.json()
    scan_id = data["id"]
    return scan_id

def upload_coords_to_scan(scan_id):
    headers = {
        "Authorization": f"Bearer {ACCESS_TOKEN}",
        "X-Device-ID": DEVICE_ID
    }

    files = {"cns_file": open(CSV_FILE_PATH, "rb")}

    for attempt in range(1, RETRY_COUNT + 1):
        start = time.time()
        resp = requests.post(f"{BASE_URL}/scans/{scan_id}/coordinates", headers=headers, files=files)
        duration = time.time() - start

        if resp.status_code == 200:
            data = resp.json()
            print(f"Success: scan {scan_id} — {data.get('count', 'N/A')} coords in {duration:.2f}s")
            return True

        print(f"Attempt {attempt}: Upload failed for scan {scan_id}: {resp.status_code} - {resp.text}")
        time.sleep(DELAY_BETWEEN_BATCHES)

    return False

# ================= MAIN =================
if __name__ == "__main__":
    print(f"Starting bulk seed at {datetime.now()}")
    print(f"Target: {NUM_SCANS} scans, each with coords from {CSV_FILE_PATH}")

    login()

    print("Creating scans sequentially...")
    scan_ids = []
    for i in tqdm(range(1, NUM_SCANS + 1), desc="Creating scans"):
        scan_id = create_one_scan()
        if scan_id:
            scan_ids.append(scan_id)
        time.sleep(0.1)  # small delay

    print(f"Created {len(scan_ids)} scans. Starting parallel uploads...")

    success = 0
    with ThreadPoolExecutor(max_workers=MAX_WORKERS) as executor:
        future_to_scan = {executor.submit(upload_coords_to_scan, sid): sid for sid in scan_ids}
        for future in tqdm(as_completed(future_to_scan), total=len(scan_ids), desc="Uploading"):
            if future.result():
                success += 1
            time.sleep(DELAY_BETWEEN_BATCHES)

    print(f"\nFinished! Successfully processed {success}/{NUM_SCANS} scans")
    print(f"Ended at {datetime.now()}")