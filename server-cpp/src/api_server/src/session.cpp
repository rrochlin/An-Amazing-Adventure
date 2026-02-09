#include "requests.cpp"
#include "server.h"
#include <boost/asio/dispatch.hpp>
#include <boost/asio/strand.hpp>
#include <boost/beast/core.hpp>
#include <boost/beast/http.hpp>
#include <boost/beast/version.hpp>
#include <boost/config.hpp>
#include <memory>

namespace beast = boost::beast;   // from <boost/beast.hpp>
namespace http = beast::http;     // from <boost/beast/http.hpp>
namespace net = boost::asio;      // from <boost/asio.hpp>
using tcp = boost::asio::ip::tcp; // from <boost/asio/ip/tcp.hpp>

// Handles an HTTP server connection
class session : public std::enable_shared_from_this<session> {
   beast::tcp_stream stream_;
   beast::flat_buffer buffer_;
   http::request<http::string_body> req_;
   Server *server_;

 public:
   // Take ownership of the stream
   session(tcp::socket &&socket, Server *server)
       : stream_(std::move(socket)), server_(server) {}

   // Start the asynchronous operation
   void run() {
      // We need to be executing within a strand to perform async operations
      // on the I/O objects in this session. Although not strictly necessary
      // for single-threaded contexts, this example code is written to be
      // thread-safe by default.
      net::dispatch(
          stream_.get_executor(),
          beast::bind_front_handler(&session::do_read, shared_from_this()));
   }

   void do_read() {
      // Make the request empty before reading,
      // otherwise the operation behavior is undefined.
      req_ = {};

      // Set the timeout.
      stream_.expires_after(std::chrono::seconds(30));

      // Read a request
      http::async_read(
          stream_, buffer_, req_,
          beast::bind_front_handler(&session::on_read, shared_from_this()));
   }

   void on_read(beast::error_code ec, std::size_t bytes_transferred) {
      boost::ignore_unused(bytes_transferred);

      // This means they closed the connection
      if (ec == http::error::end_of_stream)
         return do_close();

      if (ec)
         return fail(ec, "read");

      // Send the response
      send_response(server_->handle_request(std::move(req_)));
   }

   void send_response(http::message_generator &&msg) {
      bool keep_alive = msg.keep_alive();

      // Write the response
      beast::async_write(stream_, std::move(msg),
                         beast::bind_front_handler(&session::on_write,
                                                   shared_from_this(),
                                                   keep_alive));
   }

   void on_write(bool keep_alive, beast::error_code ec,
                 std::size_t bytes_transferred) {
      boost::ignore_unused(bytes_transferred);

      if (ec)
         return fail(ec, "write");

      if (!keep_alive) {
         // This means we should close the connection, usually because
         // the response indicated the "Connection: close" semantic.
         return do_close();
      }

      // Read another request
      do_read();
   }

   void do_close() {
      // Send a TCP shutdown
      beast::error_code ec;
      stream_.socket().shutdown(tcp::socket::shutdown_send, ec);

      // At this point the connection is closed gracefully
   }
};
