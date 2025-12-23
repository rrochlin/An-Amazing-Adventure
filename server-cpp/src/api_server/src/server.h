/*
 * configurable web server. Instances of server handle accepting web
 * requests and calling configured routes that match the call pattern.
 * Currently only supports simple patterns. Does not support query params
 * or parameterized calls
 */
#pragma once
#include <boost/asio/connect.hpp>
#include <boost/asio/ip/address.hpp>
#include <boost/asio/ip/tcp.hpp>
#include <boost/beast/core.hpp>
#include <boost/beast/http.hpp>
#include <boost/beast/version.hpp>
#include <functional>
#include <string>
#include <unordered_map>

namespace beast = boost::beast;   // from <boost/beast.hpp>
namespace http = beast::http;     // from <boost/beast/http.hpp>
namespace net = boost::asio;      // from <boost/asio.hpp>
using tcp = boost::asio::ip::tcp; // from <boost/asio/ip/tcp.hpp>

using Handler =
    std::function<http::message_generator(http::request<http::string_body> &&)>;

struct route_node {
  // currently will cull all downstream paths
  bool dynamic;
  uint8_t methods;
  std::string base;
  std::unordered_map<std::string, route_node *> children;
  std::vector<std::string> query_params;
  Handler funcs[7];

  template <class Body, class Allocator>
  Handler parse_request(http::request<Body, http::basic_fields<Allocator>> &);

  bool add(std::string, std::string, Handler);
  bool contains(std::string);

private:
  route_node *find_match(std::string key);
};

class Server {
public:
  Server(std::string host, std::string port);
  void run();
  bool addRoute(std::string, std::string, Handler);
  template <class Body, class Allocator>
  http::message_generator
  handle_request(http::request<Body, http::basic_fields<Allocator>> &&);

private:
  route_node *_routes;
  net::ip::address _host;
  unsigned short _port;
  int _threads;
};
