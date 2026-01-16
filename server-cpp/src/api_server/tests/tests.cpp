#pragma once
#include "../src/requests.cpp"
#include "../src/server.h"
#include "test_utils.h"
#include <boost/beast/http.hpp>
#include <boost/beast/http/message_fwd.hpp>
#include <boost/beast/http/string_body_fwd.hpp>
http::message_generator test_handler(http::request<http::string_body> &&req) {
   std::cout << "running the test handler\n";
   return Requests::ok(std::move(req), "message received");
}

http::message_generator
test_dynamic_handler(http::request<http::string_body> &&req) {
   std::cout << "running dynamic function\n";
   auto dynamic_value = Requests::grab_dynamic_query(req);
   return Requests::ok(std::move(req), "recieved: " + dynamic_value);
}

http::message_generator
test_query_param_handler(http::request<http::string_body> &&req) {
   std::cout << "running query function with: " << req.target() << std::endl;
   auto query_params = Requests::grab_query_params(req);
   std::stringstream ss;
   ss << "received: \r";
   for (auto i : query_params) {
      std::cout << i.first << " : " << i.second << std::endl;
      ss << i.first << " : " << i.second << " || ";
   }
   return Requests::ok(std::move(req), ss.str());
}

http::message_generator
test_unauth_post(http::request<http::string_body> &&req) {
   // for the unauthenticated post route we will need to parse the request body
   // and then respond to the clinet showing we've successfully done that.
   std::cout << "running unauthenticated post test with: " << req.target()
             << std::endl;
   auto body = req.body();
   std::cout << body;
   return Requests::ok(std::move(req), "ok");
}

void add_all_routes_tests(Server *server) {
   server->addRoute(GET, "/api/test", test_handler);
   server->addRoute(GET, "/api/test/dynamic/{uuid}", test_dynamic_handler);
   server->addRoute(GET, "/api/test/query_params", test_query_param_handler);
   server->addRoute(POST, "/api/test", test_unauth_post);
}

TEST(route_get_basic) {
   route_node root;
   ASSERT_TRUE(root.add(GET, "/api/test", test_handler));

   http::request<http::string_body> req{http::verb::get, "/api/test", 11};
   Handler h = root.parse_request(req);
}

TEST(route_get_dynamic_handler) {
   route_node root;
   ASSERT_TRUE(root.add(GET, "/api/test/dynamic/{uuid}", test_dynamic_handler));

   http::request<http::string_body> req{http::verb::get,
                                        "/api/test/dynamic/{uuid}", 1928349832};
   Handler h = root.parse_request(req);
}

TEST(route_get_query_param) {
   route_node root;
   ASSERT_TRUE(
       root.add(GET, "/api/test/query_params", test_query_param_handler));

   http::request<http::string_body> req{http::verb::get,
                                        "/api/test/query_params?searching", 0};
   Handler h = root.parse_request(req);
}

TEST(route_post_public) {
   route_node root;
   ASSERT_TRUE(root.add(POST, "/api/test", test_unauth_post));
   http::request<http::string_body> req{http::verb::post, "/api/test", 928374};
}
