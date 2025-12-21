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
#include <cstdlib>
#include <functional>
#include <iostream>
#include <string>
#include <unordered_map>

namespace beast = boost::beast;   // from <boost/beast.hpp>
namespace http = beast::http;     // from <boost/beast/http.hpp>
namespace net = boost::asio;      // from <boost/asio.hpp>
using tcp = boost::asio::ip::tcp; // from <boost/asio/ip/tcp.hpp>
using Handler = std::function<http::response<http::string_body>(
    const http::request<http::string_body> &)>;

class Server : std::enable_shared_from_this<Server> {
public:
  Server(std::string host, std::string port);
  ~Server();
  void run();
  bool addRoute(std::string, Handler);
  template <class Body, class Allocator>
  http::message_generator
  handle_request(http::request<Body, http::basic_fields<Allocator>> &&);

private:
  std::unordered_map<std::string, Handler> _routes;
  net::ip::address _host;
  unsigned short _port;
  int _threads;
};
