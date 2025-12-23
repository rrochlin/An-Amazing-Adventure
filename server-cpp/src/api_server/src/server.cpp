#include "server.h"
#include "handlers.cpp"
#include "listener.cpp"
#include "requests.cpp"
#include <boost/asio/dispatch.hpp>
#include <boost/asio/strand.hpp>
#include <boost/beast/http.hpp>
#include <boost/beast/http/message.hpp>
#include <memory>
#include <sstream>
#include <stdexcept>
#include <string>
#include <thread>

// Generally we are only going to be returning json responses from the
// server. There will be no other response types accepted/taken
// All asset requests will be served from presigned s3 url's
// so they do not need to pass through the server.
// Could also serve them from GCP?
void add_all_routes(Server *server) {
  server->addRoute("GET", "/api/test", test_handler);
  //  server->addRoute("GET /api/describe/{uuid}", cfg.HandlerDescribe)
  //  server->addRoute("POST /api/games/{uuid}", cfg.HandlerStartGame)
  //  server->addRoute("DELETE /api/games/{uuid}", cfg.HandlerDeleteGame)
  //  server->addRoute("POST /api/chat/{uuid}", cfg.HandlerChat)
  //  server->addRoute("GET /api/worldready/{uuid}", cfg.HandlerWorldReady)
  //  server->addRoute("GET /api/games", cfg.HandlerListGames)

  // server->addRoute("POST /api/login", cfg.HandlerLogin)
  // server->addRoute("POST /api/refresh", cfg.HandlerRefresh)
  // server->addRoute("POST /api/revoke", cfg.HandlerRevoke)
  // server->addRoute("PUT /api/users", cfg.HandlerUpdateUser)
  // server->addRoute("POST /api/users", cfg.HandlerUsers)
}

Server::Server(std::string host, std::string port) {
  _host = net::ip::make_address(host);
  _port = static_cast<unsigned short>(std::stoi(port));
  _threads = 4; // make this configurable later

  _routes = new route_node();

  add_all_routes(this);
}

void Server::run() {
  net::io_context ioc{_threads};

  // Create and launch a listening port
  std::make_shared<listener>(ioc, tcp::endpoint{_host, _port}, this)->run();

  // Run the I/O service on the requested number of threads
  std::vector<std::thread> v;
  v.reserve(_threads - 1);
  for (auto i = _threads - 1; i > 0; --i)
    v.emplace_back([&ioc] { ioc.run(); });
  ioc.run();
}

template <class Body, class Allocator>
http::message_generator Server::handle_request(
    http::request<Body, http::basic_fields<Allocator>> &&req) {
  Handler method = _routes->parse_request(req);

  if (!method) {
    return Requests::not_found(std::move(req));
  }

  return method(std::move(req));
}

// I only need to add these for now
bool Server::addRoute(std::string method, std::string route, Handler func) {
  std::cout << "trying to add a route\n";
  return this->_routes->add(method, route, func);
}

enum http_method {
  GET = 0x1,
  POST = 0x10,
  PUT = 0x100,
  PATCH = 0x1000,
  DELETE = 0x10000,
  HEAD = 0x1000000,
  OPTIONS = 0x10000000,
};

http_method get_method(std::string str_method) {
  if (str_method == "GET")
    return GET;
  if (str_method == "POST")
    return POST;
  if (str_method == "PUT")
    return PUT;
  if (str_method == "PATCH")
    return PATCH;
  if (str_method == "DELETE")
    return DELETE;
  if (str_method == "HEAD")
    return HEAD;
  if (str_method == "OPTIONS")
    return OPTIONS;
  throw std::invalid_argument("unsupported method: " + str_method);
}

route_node *route_node::find_match(std::string key) {
  std::stringstream ss(key);
  std::string temp;
  route_node *head = this;
  while (std::getline(ss, temp, '/')) {
    if (!head->children.contains(temp)) {
      return nullptr;
    }
    head = children.at(temp);
    if (head->dynamic) {
      break;
    }
  }
  return head;
}

// TODO pretty sure this is crashing now
template <class Body, class Allocator>
Handler route_node::parse_request(
    http::request<Body, http::basic_fields<Allocator>> &req) {
  std::cout << "parsing request\n";
  http_method method = get_method(req.method_string());
  auto head = this->find_match(req.target());
  if (!((method && head->methods) == method)) {
    throw std::invalid_argument("this method does not exist on the route");
  }
  return head->funcs[(int)std::log2((int)method)];
}

// TODO super janky needs a lot of work
bool route_node::add(std::string method, std::string route, Handler func) {
  if (DEBUG) {
    std::cout << "trying to add " << route << std::endl;
  }
  std::stringstream ss(route);
  std::string temp;
  route_node *head = this;
  std::getline(ss, temp, '/');
  while (std::getline(ss, temp, '/')) {
    std::cout << "adding " << temp << std::endl;
    if (!head->children.contains(temp)) {
      auto node = new route_node();
      node->base = temp;
      this->children.emplace(temp, node);
    }
    head = children.at(temp);
    if (head->dynamic) {
      break;
    }
  }

  auto method_c = get_method(method);
  if ((method_c && head->methods) != 0) {
    return false;
  }

  head->funcs[(int)std::log2((int)method_c)] = func;
  head->methods |= method_c;
  return true;
}
