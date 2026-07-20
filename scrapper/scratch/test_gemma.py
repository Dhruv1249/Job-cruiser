"""
Test script to verify direct API connectivity to Gemma 4 MoE and Dense models with SSL fallback and thinking-part parsing.
"""

import json
import requests
import urllib3
urllib3.disable_warnings(urllib3.exceptions.InsecureRequestWarning)

from config import GEMINI_API_KEY, GEMMA_MOE_MODEL, GEMMA_DENSE_MODEL

def test_model(model_name: str):
    """
    Test a single Gemma 4 model endpoint.
    """
    url = f"https://generativelanguage.googleapis.com/v1beta/models/{model_name}:generateContent?key={GEMINI_API_KEY}"
    payload = {
        "contents": [
            {
                "parts": [
                    {"text": "Hello, respond with a short JSON object: {\"status\": \"ok\", \"model\": \"" + model_name + "\"}"}
                ]
            }
        ],
        "generationConfig": {
            "responseMimeType": "application/json"
        }
    }
    try:
        resp = requests.post(url, json=payload, timeout=20, verify=False)
        print(f"[{model_name}] Status code: {resp.status_code}")
        if resp.status_code == 200:
            data = resp.json()
            parts = data.get("candidates", [])[0].get("content", {}).get("parts", [])
            print(f"[{model_name}] Received {len(parts)} parts.")
            for i, p in enumerate(parts):
                print(f"  Part {i} (thought={p.get('thought')}): {p.get('text')}")
            return True
        else:
            print(f"[{model_name}] Error output: {resp.text}")
            return False
    except Exception as e:
        print(f"[{model_name}] Exception: {e}")
        return False

def main():
    """
    Test both Gemma 4 MoE and Dense models.
    """
    print("Testing Gemma 4 API connectivity with SSL fallback...")
    moe_ok = test_model(GEMMA_MOE_MODEL)
    dense_ok = test_model(GEMMA_DENSE_MODEL)
    
    if moe_ok and dense_ok:
        print("\nSUCCESS: Both Gemma 4 models are active and responding!")
    else:
        print("\nFAILURE: One or both Gemma 4 models failed.")

if __name__ == "__main__":
    main()
