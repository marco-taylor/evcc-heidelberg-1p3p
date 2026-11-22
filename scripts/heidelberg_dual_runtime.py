#!/usr/bin/env python3
"""Generic HTTP controller fixture for isolated Docker PV/circuit tests.

The Heidelberg-specific driver and phase guards are tested over Modbus TCP in
Go. This fixture tests the application layer without bypassing sponsorship.
"""
import argparse
import json
import threading
import time
import urllib.error
import urllib.parse
import urllib.request
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer

LOCK = threading.Lock()
MODELS = [{"current": 6, "enabled": False, "phases": 3, "offline": False} for _ in range(2)]
PV = 30000
BUDGET = 18000  # Synthetic fixture value, not the user's electrical rating.


def power(model):
    return 230 * model["phases"] * model["current"] if model["enabled"] else 0


class Handler(BaseHTTPRequestHandler):
    def log_message(self, *_):
        pass

    def do_POST(self):
        self.do_GET()

    def do_GET(self):
        global PV, BUDGET
        parsed = urllib.parse.urlparse(self.path)
        path = parsed.path.strip("/").split("/")
        query = urllib.parse.parse_qs(parsed.query)
        with LOCK:
            code, result = 200, "ok"
            if path[0] == "control":
                if "pv" in query:
                    PV = float(query["pv"][0])
                if "budget" in query:
                    BUDGET = float(query["budget"][0])
                if "offline" in query:
                    MODELS[0]["offline"] = query["offline"][0] == "true"
            elif path[0] == "state":
                result = {"controllers": [dict(m, power=power(m)) for m in MODELS], "pv": PV, "budget": BUDGET}
            elif path[0] == "grid":
                result = sum(map(power, MODELS)) - PV
            elif path[0] == "pv":
                result = PV
            elif path[0] == "budget":
                result = BUDGET
            elif len(path) == 3 and path[0] == "controller":
                model = MODELS[int(path[1]) - 1]
                key = path[2]
                if model["offline"]:
                    code, result = 503, "simulated controller unavailable"
                elif key == "status":
                    result = "C" if model["enabled"] else "B"
                elif key in ("enable", "maxcurrent", "setphases"):
                    value = query["value"][0]
                    if key == "enable":
                        model["enabled"] = value.lower() == "true"
                    elif key == "maxcurrent":
                        model["current"] = float(value)
                    else:
                        model["phases"] = int(value)
                elif key == "power":
                    result = power(model)
                else:
                    result = model[key]
            else:
                code, result = 404, "unknown path"
            data = (json.dumps(result) if isinstance(result, (dict, bool)) else str(result)).encode()
        self.send_response(code)
        self.send_header("Content-Length", str(len(data)))
        self.end_headers()
        self.wfile.write(data)


def request(url, post=False):
    req = urllib.request.Request(url, method="POST" if post else "GET")
    with urllib.request.urlopen(req, timeout=3) as response:
        return response.read().decode()


def eventually(label, condition, timeout=120):
    deadline = time.monotonic() + timeout
    last = None
    while time.monotonic() < deadline:
        try:
            last = condition()
            if last:
                print("PASS:", label, flush=True)
                return
        except (OSError, ValueError, KeyError, urllib.error.URLError) as error:
            last = str(error)
        time.sleep(0.25)
    raise AssertionError(f"{label}: deadline exceeded ({last})")


def check(base):
    state = lambda: json.loads(request("http://localhost:8080/state"))
    eventually("UI and two loadpoints", lambda: len(json.loads(request(base + "/api/state"))["result"]["loadpoints"]) == 2)
    for index in (1, 2):
        request(f"{base}/api/loadpoints/{index}/mode/pv", post=True)
    eventually("both controllers charge from synthetic PV", lambda: all(m["power"] > 0 for m in state()["controllers"]))
    eventually("shared synthetic power budget", lambda: 0 < sum(m["power"] for m in state()["controllers"]) <= state()["budget"] + 1)
    request("http://localhost:8080/control?budget=15000")
    eventually("dynamic power limit reduction", lambda: 0 < sum(m["power"] for m in state()["controllers"]) <= 15001)
    request("http://localhost:8080/control?offline=true&budget=18000")
    request(f"{base}/api/loadpoints/2/mode/off", post=True)
    eventually("reachable controller stops independently", lambda: not state()["controllers"][1]["enabled"])
    request(f"{base}/api/loadpoints/2/mode/pv", post=True)
    eventually("reachable controller continues with peer unavailable", lambda: state()["controllers"][1]["power"] > 0)
    request("http://localhost:8080/control?offline=false&budget=18000")
    request(f"{base}/api/loadpoints/1/mode/off", post=True)
    eventually("controller recovery", lambda: not state()["controllers"][0]["enabled"])
    print(json.dumps(state(), sort_keys=True), flush=True)


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--check")
    args = parser.parse_args()
    if args.check:
        check(args.check)
    else:
        ThreadingHTTPServer(("0.0.0.0", 8080), Handler).serve_forever()
