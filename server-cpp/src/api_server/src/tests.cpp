#include "requests.cpp"
#include "server.h"
#include <boost/beast/http.hpp>

http::message_generator test_handler(http::request<http::string_body> &&req) {
  std::cout << "running the test handler\n";
  return Requests::ok(std::move(req), "message received");
}

http::message_generator
test_dynamic_handler(http::request<http::string_body> &&req) {
  std::cout << "running dynamic function";
  auto dynamic_value = Requests::grab_dynamic_query(req);
  return Requests::ok(std::move(req), "recieved: " + dynamic_value);
}

http::message_generator
test_query_param_handler(http::request<http::string_body> &&req) {
  std::cout << "running query function";
  auto query_params = Requests::grab_query_params(req);
  std::stringstream ss;
  ss << "received: \n";
  for (auto i : query_params) {
    std::cout << i.first << " : " << i.second;
    ss << i.first << " : " << i.second;
  }
  ss << "\n";
  return Requests::ok(std::move(req), ss.str());
}

void add_all_routes_tests(Server *server) {
  server->addRoute(GET, "/api/test", test_handler);
  // TODO work on both of these
  server->addRoute(GET, "/api/test/dynamic/{uuid}", test_dynamic_handler);
  server->addRoute(GET, "/api/test/query_params", test_query_param_handler);
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
