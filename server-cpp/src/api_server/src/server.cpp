#include "server.h"
#include "listener.cpp"
#include "server_utils.cpp"
#include <boost/asio/dispatch.hpp>
#include <boost/asio/strand.hpp>
#include <boost/beast/http.hpp>
#include <memory>
#include <thread>

// Generally we are only going to be returning json responses from the
// server. There will be no other response types accepted/taken
// All asset requests will be served from presigned s3 url's
// so they do not need to pass through the server.
// Could also serve them from GCP?
Server::Server(std::string host, std::string port) {
  auto const _host = net::ip::make_address(host);
  auto const _port = static_cast<unsigned short>(std::stoi(port));
  auto const _threads = 4; // make this configurable later
}

void Server::run() {
  net::io_context ioc{_threads};

  // Create and launch a listening port
  auto self = shared_from_this();
  std::make_shared<listener>(ioc, tcp::endpoint{_host, _port}, self)->run();

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
  std::string uri = req.target();
  if (!_routes.contains(uri)) {
    return Requests::not_found(std::move(req));
  }
  // TODO this dictionary stores anonymous functions, that while they do have
  //  the method type stored in them, don't actually provide a way to match
  //  the method here. Since the method is only really available via the
  //  signature
  if (!_routes.find(std::string(req.target()))->second.method() ==
      req.method()) {
    return Requests::not_allowed(std::move(req));
  }
}
