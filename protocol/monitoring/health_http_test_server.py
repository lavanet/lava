#!/usr/bin/env python3
"""
Simple HTTP server to receive health probe results.
Usage: python3 healthResultsServer.py [port] [output_file]
"""
import json
from http.server import BaseHTTPRequestHandler, HTTPServer
import sys
import os
from datetime import datetime

class HealthResultsHandler(BaseHTTPRequestHandler):
    def __init__(self, *args, **kwargs):
        self.output_file = kwargs.pop('output_file', 'health_results.json')
        super().__init__(*args, **kwargs)

    def do_POST(self):
        content_length = int(self.headers.get('Content-Length', 0))
        if content_length > 0:
            body = self.rfile.read(content_length)
            try:
                data = json.loads(body.decode('utf-8'))
                self.process_health_results(data)
            except json.JSONDecodeError as e:
                print(f"Error decoding JSON: {e}")
                self.send_response(400)
                self.send_header('Content-type', 'text/plain')
                self.end_headers()
                self.wfile.write(b'Invalid JSON')
                return
        
        self.send_response(200)
        self.send_header('Content-type', 'text/plain')
        self.end_headers()
        self.wfile.write(b'OK')

    def process_health_results(self, data):
        timestamp = datetime.now()
        timestamp_str = timestamp.isoformat()
        
        # Create output directory
        output_dir = "./out"
        os.makedirs(output_dir, exist_ok=True)
        
        # Generate filename with timestamp and GUID
        guid = data.get('resultsPostGUID', 'no-guid')
        provider_addr = "unknown-provider"
        if 'providerAddresses' in data and data['providerAddresses']:
            provider_addr = data['providerAddresses'][0].split('@')[-1][:8]  # First 8 chars after @
        
        filename_timestamp = timestamp.strftime("%Y%m%d_%H%M%S")
        json_filename = f"{output_dir}/health_results_{filename_timestamp}_{guid}_{provider_addr}.json"
        log_filename = f"{output_dir}/health_results_{filename_timestamp}_{guid}_{provider_addr}.log"
        
        print(f"\n{'='*60}")
        print(f"HEALTH RESULTS RECEIVED - {timestamp_str}")
        print(f"{'='*60}")
        
        # Prepare log content
        log_content = []
        log_content.append(f"HEALTH RESULTS RECEIVED - {timestamp_str}")
        log_content.append("="*60)
        
        # Print and log summary
        if 'resultsPostGUID' in data:
            msg = f"GUID: {data['resultsPostGUID']}"
            print(msg)
            log_content.append(msg)
        
        if 'providerAddresses' in data:
            msg = f"Provider Addresses: {data['providerAddresses']}"
            print(msg)
            log_content.append(msg)
        
        # Print provider data
        if 'providerData' in data and data['providerData']:
            msg = "\n📊 PROVIDER DATA:"
            print(msg)
            log_content.append(msg)
            for entity, reply_data in data['providerData'].items():
                msg = f"  {entity}: Block={reply_data.get('block', 'N/A')}, Latency={reply_data.get('latency', 'N/A')}"
                print(msg)
                log_content.append(msg)
        
        # Print unhealthy providers
        if 'unhealthyProviders' in data and data['unhealthyProviders']:
            msg = "\n❌ UNHEALTHY PROVIDERS:"
            print(msg)
            log_content.append(msg)
            for entity, error in data['unhealthyProviders'].items():
                msg = f"  {entity}: {error}"
                print(msg)
                log_content.append(msg)
        
        # Print frozen providers
        if 'frozenProviders' in data and data['frozenProviders']:
            msg = "\n🧊 FROZEN PROVIDERS:"
            print(msg)
            log_content.append(msg)
            for entity in data['frozenProviders']:
                msg = f"  {entity}"
                print(msg)
                log_content.append(msg)
        
        # Print jailed providers
        if 'jailedProviders' in data and data['jailedProviders']:
            msg = "\n🔒 JAILED PROVIDERS:"
            print(msg)
            log_content.append(msg)
            for entity in data['jailedProviders']:
                msg = f"  {entity}"
                print(msg)
                log_content.append(msg)
        
        # Print latest blocks
        if 'latestBlocks' in data and data['latestBlocks']:
            msg = "\n🔗 LATEST BLOCKS:"
            print(msg)
            log_content.append(msg)
            for spec, block in data['latestBlocks'].items():
                msg = f"  {spec}: {block}"
                print(msg)
                log_content.append(msg)
        
        # Print consumer data
        if 'consumerBlocks' in data and data['consumerBlocks']:
            msg = "\n🔌 CONSUMER BLOCKS:"
            print(msg)
            log_content.append(msg)
            for entity, block in data['consumerBlocks'].items():
                msg = f"  {entity}: {block}"
                print(msg)
                log_content.append(msg)
        
        print(f"\n{'='*60}")
        log_content.append("="*60)
        
        # Save individual JSON file
        result_with_timestamp = {
            'timestamp': timestamp_str,
            'guid': guid,
            'provider_addresses': data.get('providerAddresses', []),
            'healthResults': data
        }
        
        try:
            # Save JSON file
            with open(json_filename, 'w') as f:
                json.dump(result_with_timestamp, f, indent=2)
            
            # Save log file
            with open(log_filename, 'w') as f:
                f.write('\n'.join(log_content))
            
            print(f"✅ Results saved to:")
            print(f"   JSON: {json_filename}")
            print(f"   Log:  {log_filename}")
            
            # Also save to main results file for backwards compatibility
            try:
                with open(self.output_file, 'r') as f:
                    all_results = json.load(f)
            except (FileNotFoundError, json.JSONDecodeError):
                all_results = []
            
            if not isinstance(all_results, list):
                all_results = [all_results]
            
            all_results.append(result_with_timestamp)
            
            with open(self.output_file, 'w') as f:
                json.dump(all_results, f, indent=2)
            
        except Exception as e:
            print(f"❌ Error saving results: {e}")

    def log_message(self, format, *args):
        # Suppress default HTTP logging
        pass

def run_server(port=6510, output_file='health_results.json'):
    server_address = ('', port)
    
    def handler(*args, **kwargs):
        return HealthResultsHandler(*args, output_file=output_file, **kwargs)
    
    httpd = HTTPServer(server_address, handler)
    print(f"🚀 Health Results Server running on port {port}")
    print(f"📁 Saving results to: {output_file}")
    print(f"📡 Send POST requests to: http://localhost:{port}")
    print("Press Ctrl+C to stop\n")
    
    try:
        httpd.serve_forever()
    except KeyboardInterrupt:
        print("\n🛑 Server stopped")

if __name__ == '__main__':
    port = int(sys.argv[1]) if len(sys.argv) > 1 else 6510
    output_file = sys.argv[2] if len(sys.argv) > 2 else 'health_results.json'
    run_server(port, output_file)