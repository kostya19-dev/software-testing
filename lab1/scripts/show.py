import argparse
import errno
from functools import partial
from http.server import SimpleHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("directory", type=Path)
    parser.add_argument("--port", type=int, default=8081)
    args = parser.parse_args()
    if not 0 <= args.port <= 65535:
        parser.error("Порт должен быть от 0 до 65535")
    directory = args.directory.resolve()
    if not (directory / "index.html").is_file():
        parser.error("Нет index.html: сначала выполните make test или make report")

    handler = partial(SimpleHTTPRequestHandler, directory=str(directory))
    try:
        server = ThreadingHTTPServer(("127.0.0.1", args.port), handler)
    except OSError as error:
        if error.errno != errno.EADDRINUSE:
            raise
        print(f"Порт {args.port} занят; выбираю свободный.", flush=True)
        server = ThreadingHTTPServer(("127.0.0.1", 0), handler)

    with server:
        port = server.server_address[1]
        print(f"Откройте в браузере: http://localhost:{port}", flush=True)
        print(f"HTML-файл: {directory / 'index.html'}", flush=True)
        print("Остановка сервера: Ctrl+C", flush=True)
        try:
            server.serve_forever()
        except KeyboardInterrupt:
            print("\nСервер остановлен.")


if __name__ == "__main__":
    main()
