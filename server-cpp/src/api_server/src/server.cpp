#include "server.h"
#include "handlers.cpp"
#include "listener.cpp"
#include "requests.cpp"
#include "tests.cpp"
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

Server::Server(std::string host, std::string port) {
  _host = net::ip::make_address(host);
  _port = static_cast<unsigned short>(std::stoi(port));
  _threads = 4; // make this configurable later

  _routes = new route_node();

  add_all_routes(this);
  add_all_routes_tests(this);
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
bool Server::addRoute(http_method method, std::string route, Handler func) {
  std::cout << "trying to add a route\n";
  return this->_routes->add(method, route, func);
}

route_node *route_node::find_match(std::string key) {
  std::cout << "searching for route: " << key << std::endl;
  std::stringstream ss(key);
  std::string temp;
  route_node *head = this;
  std::getline(ss, temp, '/');
  while (std::getline(ss, temp, '/')) {
    std::cout << "parsing: " << temp << std::endl;
    if (!head->children.contains(temp) && !head->dynamic) {
      return nullptr;
    }
    if (!head->children.contains(temp) && head->dynamic) {
      return head->children.at("");
    }
    head = head->children.at(temp);
  }
  return head;
}

// TODO pretty sure this is crashing now
template <class Body, class Allocator>
Handler route_node::parse_request(
    http::request<Body, http::basic_fields<Allocator>> &req) {
  std::cout << "parsing request\n";
  http_method method = get_method(req.method_string());
  std::cout << "got method: " << req.method_string() << std::endl;
  auto head = this->find_match(Requests::extract_route(req));
  std::cout << "found method " << method << std::endl;
  if (head == nullptr || !((method && head->methods) == method)) {
    std::cout << "error method not found\n";
    std::cout << method << std::endl << head->methods << std::endl;
    throw std::invalid_argument("this method does not exist on the route");
  }
  std::cout << "returning\n";
  return head->funcs[(int)std::log2((int)method)];
}

bool route_node::add(http_method method, std::string route, Handler func) {
  std::cout << "trying to add " << route << std::endl;
  std::stringstream ss(route);
  std::string temp;
  route_node *head = this;
  std::getline(ss, temp, '/');
  while (std::getline(ss, temp, '/')) {
    std::cout << "adding: " << temp << std::endl;
    if (temp[0] == '{') {
      if (!head->dynamic) {
        head->dynamic = true;
        head->children.emplace("", new route_node());
      }
      head = head->children.at("");
      break;
    }
    if (!head->children.contains(temp)) {
      std::cout << "creating new node\n";
      auto node = new route_node();
      node->base = temp;
      head->children.emplace(temp, node);
    }
    head = head->children.at(temp);
    if (head->dynamic) {
      break;
    }
  }

  if ((method && head->methods) != 0) {
    // can't add things twice currently
    return false;
  }

  head->funcs[(int)std::log2((int)method)] = func;
  head->methods |= method;
  return true;
}

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
