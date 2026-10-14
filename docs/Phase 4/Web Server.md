## Goal
in cpp their are options for more robust web server frameworks but I've elected to write my own just using boost to manage listening on ports and connections. The [provided code](https://www.boost.org/doc/libs/master/libs/beast/example/http/server/async/http_server_async.cpp) from boost shows how to create a simple async web server, but this is intended to just simply use a file directory structure to handle requests and serve content.

## Specification
- The server should be able to handle requests without blocking
- The server api should allow easy use in the following ways
	- Allow creation with no arguments
	- configure routes with a simple call like `server.add(route, function)`
	- support the following:
		- static routes `/api/users`
		- dynamic routes `api/users/{uuid}`
		- allow query params `/api/users?olderThan=20`

## Design
### Server
main class that is intended to hold all of the needed resources and then run to handle the connections

#### public
- `void add(Route route, Handler function)`
- `void run(string host, int **port**)`
