#pragma once
#include <boost/asio/connect.hpp>
#include <boost/asio/ip/tcp.hpp>
#include <boost/beast/core.hpp>
#include <boost/beast/core/string_type.hpp>
#include <boost/beast/http.hpp>
#include <boost/beast/http/fields.hpp>
#include <boost/beast/http/message_fwd.hpp>
#include <boost/beast/http/string_body_fwd.hpp>
#include <boost/beast/version.hpp>
#include <iostream>
#include <nlohmann/json.hpp>
#include <stdexcept>
#include <string>
#include <unordered_map>

namespace beast = boost::beast;   // from <boost/beast.hpp>
namespace http = beast::http;     // from <boost/beast/http.hpp>
using tcp = boost::asio::ip::tcp; // from <boost/asio/ip/tcp.hpp>
using json = nlohmann::json;

const std::string MIME_TYPE = "application/json"; // we're only returning json
const bool DEBUG = true;
class Requests {

  template <class Body, class Allocator>
  static http::message_generator
  respond(http::request<Body, http::basic_fields<Allocator>> &&req,
          std::string why, http::status status) {
    if (DEBUG) {
      std::cout << "Debug output display\n"
                << req.target() << std::endl
                << req.method_string() << std::endl;
    }
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
    std::string msg =
        "The resource '" + std::string(req.target()) + "' was not found.";
    return respond(std::move(req), msg, http::status::not_found);
  }

  template <class Body, class Allocator>
  static http::message_generator
  not_allowed(http::request<Body, http::basic_fields<Allocator>> &&req) {
    std::string msg = "The resource '" + std::string(req.target()) +
                      "' does not support: " + req.method_string();
    return respond(std::move(req), msg, http::status::method_not_allowed);
  }

  template <class Body, class Allocator>
  static http::message_generator
  server_error(http::request<Body, http::basic_fields<Allocator>> &&req,
               std::string error) {
    std::string msg = "An error occured: '" + error + "'";
    return respond(std::move(req), msg, http::status::internal_server_error);
  }

  template <class Body, class Allocator>
  static http::message_generator
  forbidden(http::request<Body, http::basic_fields<Allocator>> &&req,
            std::string msg) {
    std::string rejection_message =
        msg == "" ? "This resource is fobidden" : msg;
    return respond(std::move(req), rejection_message, http::status::forbidden);
  }

  template <class Body, class Allocator>
  static http::message_generator
  unauthorized(http::request<Body, http::basic_fields<Allocator>> &&req,
               std::string msg) {
    std::string rejection_message =
        msg == "" ? "This resource is unauthorized" : msg;
    return respond(std::move(req), rejection_message,
                   http::status::unauthorized);
  }

  // ---------- SUCCESS CODES ------------
  template <class Body, class Allocator>
  static http::message_generator
  ok(http::request<Body, http::basic_fields<Allocator>> &&req,
     std::string msg) {
    return respond(std::move(req), msg, http::status::ok);
  }

  // --------- HELPER FUNCTIONS ----------
  template <class Body, class Allocator>
  static std::string
  grab_dynamic_query(http::request<Body, http::basic_fields<Allocator>> &req) {
    auto r = std::string(req.target());
    size_t start = r.rfind('/');
    if (start == std::string::npos) {
      throw std::invalid_argument("request target has no /'s");
    }
    std::string tail = r.substr(start + 1);
    size_t end = tail.find('?');
    if (end != std::string::npos) {
      tail = tail.substr(0, end);
    }
    return tail;
  }

  template <class Body, class Allocator>
  static std::unordered_map<std::string, std::string>
  grab_query_params(http::request<Body, http::basic_fields<Allocator>> &req) {
    auto r = std::string(req.target());
    size_t start = r.find('?');
    if (start == std::string::npos) {
      return {};
    }
    std::string tail = r.substr(start + 1);
    std::unordered_map<std::string, std::string> map = {};
    while (tail.length() > 0) {
      start = tail.find('=');
      size_t end = tail.find('&');
      if (start == std::string::npos || end == std::string::npos) {
        throw std::invalid_argument("malformatted query parameter");
      }
      std::string param = tail.substr(0, start);
      std::string value = tail.substr(start + 1, end - start);
      map.emplace(param, value);
      tail = tail.substr(end + 1);
    }
    return map;
  }

  template <class Body, class Allocator>
  static std::string
  extract_route(http::request<Body, http::basic_fields<Allocator>> &req) {
    auto r = std::string(req.target());
    size_t start = r.find('?');
    if (start == std::string::npos) {
      return r;
    }
    return r.substr(0, start);
  }
};

void fail(beast::error_code ec, char const *what) {
  std::cerr << what << ": " << ec.message() << "\n";
}
