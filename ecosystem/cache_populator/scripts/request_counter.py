import argparse
from http.server import BaseHTTPRequestHandler, HTTPServer
import time

# Global variables to keep track of requests and timing
request_count = 0
start_time = None

class RequestCounterHandler(BaseHTTPRequestHandler):
    # Suppress default server log messages
    def log_message(self, format, *args):
        return

    def _handle_request(self, request_type):
        global request_count, start_time
        
        if start_time is None:
            start_time = time.time()
        
        request_count += 1
        current_time = time.time()
        elapsed_time = current_time - start_time
        requests_per_second = request_count / elapsed_time if elapsed_time > 0 else 0
        
        # Unified single line output
        print(f"{request_type} requests: {request_count}, Rate: {requests_per_second:.2f} req/s")
        
        self.send_response(200)
        self.send_header('Content-type', 'text/html')
        self.end_headers()
        self.wfile.write(bytes(f"Number of requests: {request_count}\nRequests per second: {requests_per_second:.2f}", "utf8"))

    def do_GET(self):
        self._handle_request("GET")

    def do_POST(self):
        self._handle_request("POST")

def run(port=8080):
    server_address = ('', port)
    httpd = HTTPServer(server_address, RequestCounterHandler)
    print(f'Starting httpd server on port {port}')
    httpd.serve_forever()

if __name__ == "__main__":
    parser = argparse.ArgumentParser(description='Request Counter Server')
    parser.add_argument('--port', type=int, default=8080, help='Port to listen on')
    args = parser.parse_args()
    run(port=args.port)
