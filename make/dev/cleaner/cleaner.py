#!/usr/bin/python
import SimpleHTTPServer
import SocketServer
from subprocess import call

class MyRequestHandler(SimpleHTTPServer.SimpleHTTPRequestHandler):
    def do_GET(self):
        # run gc cmd here
        print "do gc worker"
        try:
            # show all containers
            status = call(["docker", "ps", "-a"])
            print "sleep 1s..., stop cargo_registry"
            call(["sleep", "1"])
            call(["docker", "stop", "cargo_registry"])
            print "sleep 1s..., do garbage collection process"
            call(["sleep", "1"])
            call(["docker", "run", "-it", "--name", "gc", "--rm", "--volumes-from", "cargo_registry", "registry:2.5.0", "garbage-collect", "/etc/registry/config.yml"])
            print "sleep 1s..., start cargo_registry again"
            call(["sleep", "1"])
            call(["docker", "start", "cargo_registry"])
        except:
            print "gc cmd exec error"
            self.send_error(500)
            return
        print "gc cmd exec ok: ", status
        self.send_response(200)
        self.send_header('Content-Type', 'text/plain')
        self.end_headers()

        self.wfile.write("gc cmd exec ok");
        self.wfile.close();
        return

PORT = 8000

Handler = MyRequestHandler

httpd = SocketServer.TCPServer(("", PORT), Handler)

print "serving at port", PORT
httpd.serve_forever()
