#include "requests.cpp"
#include "server.h"
#include <boost/beast/http.hpp>

http::message_generator test_handler(http::request<http::string_body> &&req) {
  return Requests::ok(std::move(req), "message received");
}

/*
        // game routes
        mux.HandleFunc("GET /api/describe/{uuid}", cfg.HandlerDescribe)
        mux.HandleFunc("POST /api/games/{uuid}", cfg.HandlerStartGame)
        mux.HandleFunc("DELETE /api/games/{uuid}", cfg.HandlerDeleteGame)
        mux.HandleFunc("POST /api/chat/{uuid}", cfg.HandlerChat)
        mux.HandleFunc("GET /api/worldready/{uuid}", cfg.HandlerWorldReady)
        mux.HandleFunc("GET /api/games", cfg.HandlerListGames)

        // user routes
        mux.HandleFunc("POST /api/login", cfg.HandlerLogin)
        mux.HandleFunc("POST /api/refresh", cfg.HandlerRefresh)
        mux.HandleFunc("POST /api/revoke", cfg.HandlerRevoke)
        mux.HandleFunc("PUT /api/users", cfg.HandlerUpdateUser)
        mux.HandleFunc("POST /api/users", cfg.HandlerUsers)
*/
