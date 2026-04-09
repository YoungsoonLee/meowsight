"""Tiny OpenAI-compatible mock server for local end-to-end testing.

Returns a fixed chat-completion response with a `usage` block so the MeowSight
proxy can record token counts and cost. No API key required.
"""
from http.server import BaseHTTPRequestHandler, HTTPServer
import json
import time


class Handler(BaseHTTPRequestHandler):
    def do_POST(self):
        length = int(self.headers.get("Content-Length", 0))
        try:
            req = json.loads(self.rfile.read(length) or b"{}")
        except json.JSONDecodeError:
            req = {}

        model = req.get("model", "gpt-4o-mini")
        body = {
            "id": "chatcmpl-mock-001",
            "object": "chat.completion",
            "created": int(time.time()),
            "model": model,
            "choices": [
                {
                    "index": 0,
                    "message": {"role": "assistant", "content": "hello from mock"},
                    "finish_reason": "stop",
                }
            ],
            "usage": {
                "prompt_tokens": 12,
                "completion_tokens": 6,
                "total_tokens": 18,
            },
        }
        data = json.dumps(body).encode()
        self.send_response(200)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(data)))
        self.end_headers()
        self.wfile.write(data)

    def log_message(self, format, *args):
        # Quieter logs
        print("mock-openai:", format % args)


if __name__ == "__main__":
    HTTPServer(("0.0.0.0", 11500), Handler).serve_forever()
