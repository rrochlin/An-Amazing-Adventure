#include "api_server/src/server.h"

int main() {

  Server server("0.0.0.0", "3000");
  server.run();
}
