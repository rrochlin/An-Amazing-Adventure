#pragma once
#include <boost/asio/connect.hpp>
#include <boost/asio/ip/tcp.hpp>
#include <boost/beast/core.hpp>
#include <boost/beast/core/string_type.hpp>
#include <boost/beast/http.hpp>
#include <boost/beast/http/message_fwd.hpp>
#include <boost/beast/http/string_body_fwd.hpp>
#include <boost/beast/version.hpp>
#include <cstdlib>
#include <iostream>
#include <nlohmann/json.hpp>

namespace beast = boost::beast;   // from <boost/beast.hpp>
namespace http = beast::http;     // from <boost/beast/http.hpp>
namespace net = boost::asio;      // from <boost/asio.hpp>
using tcp = boost::asio::ip::tcp; // from <boost/asio/ip/tcp.hpp>
using json = nlohmann::json;

std::string MIME_TYPE = "application/json"; // we're only returning json
class Requests {

  template <class Body, class Allocator>
  static http::message_generator
  respond(http::request<Body, http::basic_fields<Allocator>> &&req,
          std::string why, http::status status) {
    http::response<http::string_body> res{status, req.version()};
    json _response;
    _response["body"] = why;
    std::string response = _response.dump();
    res.set(http::field::server, BOOST_BEAST_VERSION_STRING);
    res.set(http::field::content_type, MIME_TYPE);
    res.keep_alive(req.keep_alive());
    res.body() = response;
    res.prepare_payload();
    return res;
  }

public:
  // ------- ERROR CODES ---------
  template <class Body, class Allocator>
  static http::message_generator
  not_found(http::request<Body, http::basic_fields<Allocator>> &&req) {
    std::string msg = "The resource '" + req.target() + "' was not found.";
    return respond(req, msg, http::status::not_found);
  }

  template <class Body, class Allocator>
  static http::message_generator
  not_allowed(http::request<Body, http::basic_fields<Allocator>> &&req) {
    std::string msg =
        "The resource '" + req.target() + "' does not support: " + req.method();
    return respond(req, msg, http::status::method_not_allowed);
  }

  template <class Body, class Allocator>
  static http::message_generator
  server_error(http::request<Body, http::basic_fields<Allocator>> &&req,
               std::string error) {
    std::string msg = "An error occured: '" + error + "'";
    return respond(req, msg, http::status::internal_server_error);
  }

  template <class Body, class Allocator>
  static http::message_generator
  forbidden(http::request<Body, http::basic_fields<Allocator>> &&req,
            std::string msg) {
    std::string rejection_message =
        msg == "" ? "This resource is fobidden" : msg;
    return respond(req, rejection_message, http::status::forbidden);
  }

  template <class Body, class Allocator>
  static http::message_generator
  unauthorized(http::request<Body, http::basic_fields<Allocator>> &&req,
               std::string msg) {
    std::string rejection_message =
        msg == "" ? "This resource is unauthorized" : msg;
    return respond(req, rejection_message, http::status::unauthorized);
  }

  // ---------- SUCCESS CODES ------------
  template <class Body, class Allocator>
  static http::message_generator
  ok(http::request<Body, http::basic_fields<Allocator>> &&req,
     std::string msg) {
    return respond(req, msg, http::status::ok);
  }
};

void fail(beast::error_code ec, char const *what) {
  std::cerr << what << ": " << ec.message() << "\n";
}
