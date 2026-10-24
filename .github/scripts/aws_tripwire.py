#!/usr/bin/env python3
"""A stand-in AWS endpoint for the unit tests: it records every request and
answers with an error. Unit tests must mock AWS and Bedrock, so the test job
points the SDK here (AWS_ENDPOINT_URL) and fails if anything was recorded.

Usage: aws_tripwire.py LOG_FILE [PORT]
"""
import http.server
import json
import sys

LOG = sys.argv[1]
PORT = int(sys.argv[2]) if len(sys.argv) > 2 else 4599


class Recorder(http.server.BaseHTTPRequestHandler):
    def _record(self):
        length = int(self.headers.get("Content-Length") or 0)
        self.rfile.read(length)
        with open(LOG, "a") as f:
            f.write(json.dumps({
                "method": self.command,
                "path": self.path,
                "target": self.headers.get("X-Amz-Target"),
            }) + "\n")
        self.send_response(500)
        self.send_header("Content-Type", "application/x-amz-json-1.0")
        self.end_headers()
        self.wfile.write(b'{"__type":"InternalFailure","message":"unit tests must not call AWS"}')

    do_GET = do_POST = do_PUT = do_DELETE = do_HEAD = _record

    def log_message(self, *args):
        pass


http.server.HTTPServer(("127.0.0.1", PORT), Recorder).serve_forever()
