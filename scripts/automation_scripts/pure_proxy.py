import asyncio
import aiohttp
from aiohttp import web
from functools import partial

port_url_map = {
    5555: "http://localhost:1317",  # Replace with actual target URLs
    5556: "http://localhost:26657",
}

async def proxy_handler(request, server_port):
    target_url = port_url_map.get(server_port)
    
    if not target_url:
        return web.Response(text=f"No target URL configured for port {server_port}", status=404)

    path = request.rel_url.path
    query_string = request.rel_url.query_string
    url = f"{target_url}{path}"
    if query_string:
        url += f"?{query_string}"

    print(f"Proxying request to: {url}")  # Debug print
    print(f"Request headers: {request.headers}")  # Debug print

    try:
        async with aiohttp.ClientSession() as session:
            method = request.method
            headers = {k: v for k, v in request.headers.items() if k.lower() not in ('host', 'content-length')}
            data = await request.read()

            async with session.request(method, url, headers=headers, data=data, allow_redirects=False) as resp:
                print(f"Response status: {resp.status}")  # Debug print
                print(f"Response headers: {resp.headers}")  # Debug print

                response = web.StreamResponse(status=resp.status, headers=resp.headers)
                await response.prepare(request)

                async for chunk, _ in resp.content.iter_chunks():
                    await response.write(chunk)
                    print(f"Wrote chunk of size: {len(chunk)}")  # Debug print

                await response.write_eof()
                return response

    except Exception as e:
        print(f"Error proxying request: {str(e)}")
        return web.Response(text=f"Error proxying request: {str(e)}", status=500)

def create_app(port):
    app = web.Application()
    handler = partial(proxy_handler, server_port=port)
    app.router.add_route('*', '/{path:.*}', handler)
    return app

async def run_app(app, port):
    runner = web.AppRunner(app)
    await runner.setup()
    site = web.TCPSite(runner, '0.0.0.0', port)
    await site.start()
    print(f"Server started on port {port}")
    return runner

async def main():
    runners = []
    for port in port_url_map.keys():
        app = create_app(port)
        runner = await run_app(app, port)
        runners.append(runner)

    print("Proxy server is running. Press Ctrl+C to stop.")
    try:
        await asyncio.Event().wait()
    except KeyboardInterrupt:
        print("Stopping server...")
    finally:
        for runner in runners:
            await runner.cleanup()

if __name__ == '__main__':
    asyncio.run(main())