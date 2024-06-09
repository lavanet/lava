from http.server import BaseHTTPRequestHandler, HTTPServer
import sys

class RequestHandler(BaseHTTPRequestHandler):
    def do_GET(self):
        self.print_request()

    def do_POST(self):
        self.print_request()

    def print_request(self):
        # Print request line
        print(f"Request: {self.command} {self.path} {self.request_version}")

        # Print headers
        for header, value in self.headers.items():
            print(f"{header}: {value}")

        # Print a blank line to separate headers from the body
        print()

        # If there's a message body, print it
        content_length = int(self.headers.get("Content-Length", 0))
        if content_length > 0:
            body = self.rfile.read(content_length)
            print(f"Body:\n{body.decode('utf-8')}")

        # Send a response back to the client
        self.send_response(200)
        self.send_header("Content-type", "text/html")
        self.end_headers()
        self.wfile.write(b"OK")

def run_server(port=8000):
    server_address = ('', port)
    httpd = HTTPServer(server_address, RequestHandler)
    print(f"Server running on port {port}")
    httpd.serve_forever()

if __name__ == '__main__':
    if len(sys.argv) > 1:
        port = int(sys.argv[1])
        run_server(port)
    else:
        run_server()