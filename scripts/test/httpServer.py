import json
import csv
from http.server import BaseHTTPRequestHandler, HTTPServer
import sys

payload_ret = "OK"


class RequestHandler(BaseHTTPRequestHandler):
    def __init__(self, *args, **kwargs):
        self.csv_file_name = kwargs.pop('csv_file_name', 'data.csv')
        super().__init__(*args, **kwargs)

    def do_GET(self):
        self.print_request()

    def do_POST(self):
        content_length = int(self.headers.get("Content-Length", 0))
        if content_length > 0:
            body = self.rfile.read(content_length)
            data = json.loads(body.decode('utf-8'))
            self.write_to_csv(data)
        self.send_response(200)
        self.send_header("Content-type", "text/html")
        self.end_headers()
        self.wfile.write(b"OK")

    def write_to_csv(self, data):
        with open(self.csv_file_name, 'a', newline='') as csvfile:
            fieldnames = data[0].keys()
            writer = csv.DictWriter(csvfile, fieldnames=fieldnames)
            if csvfile.tell() == 0:  # Check if file is empty to write header
                writer.writeheader()
            for entry in data:
                writer.writerow(entry)

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


def run_server(port=8000, csv_file_name='data.csv'):
    server_address = ('', port)

    def handler(*args, **kwargs):
        return RequestHandler(*args, csv_file_name=csv_file_name, **kwargs)
    httpd = HTTPServer(server_address, handler)
    print(f"Server running on port {port}, writing to {csv_file_name}")
    httpd.serve_forever()


if __name__ == '__main__':
    port = int(sys.argv[1]) if len(sys.argv) > 1 else 8000
    csv_file_name = sys.argv[2] if len(sys.argv) > 2 else 'data.csv'
    run_server(port, csv_file_name)
