from http.server import BaseHTTPRequestHandler, HTTPServer
import json

class Handler(BaseHTTPRequestHandler):
    def do_POST(self):
        length = int(self.headers.get("content-length", 0))
        body = self.rfile.read(length).decode("utf-8")

        print("\n--- Webhook Received ---")
        print("Path:", self.path)
        print("Headers:", dict(self.headers))
        print("Body:", body)

        self.send_response(200)
        self.end_headers()
        self.wfile.write(b'{"status":"ok"}')

HTTPServer(("0.0.0.0", 9000), Handler).serve_forever()
